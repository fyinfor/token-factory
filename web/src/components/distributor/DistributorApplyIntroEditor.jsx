/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useEffect, useRef } from 'react';
import Quill from 'quill';
import 'quill/dist/quill.snow.css';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';

function normalizeEmptyHtml(html) {
  const s = String(html ?? '').trim();
  if (s === '' || s === '<p><br></p>' || s === '<p></p>') return '';
  return html ?? '';
}

/**
 * 代理申请页说明：富文本（原生 Quill，避免 react-quill 在部分环境下不渲染）
 */
export default function DistributorApplyIntroEditor({
  value,
  onChange,
  disabled,
}) {
  const { t } = useTranslation();
  const wrapRef = useRef(null);
  const quillRef = useRef(null);
  const onChangeRef = useRef(onChange);
  const valueRef = useRef(value);
  const syncingFromPropRef = useRef(false);

  onChangeRef.current = onChange;
  valueRef.current = value;

  useEffect(() => {
    const el = wrapRef.current;
    if (!el) return undefined;

    const quill = new Quill(el, {
      theme: 'snow',
      readOnly: Boolean(disabled),
      modules: {
        toolbar: {
          container: [
            [{ header: [1, 2, 3, false] }],
            ['bold', 'italic', 'underline', 'strike'],
            [{ list: 'ordered' }, { list: 'bullet' }],
            ['blockquote', 'link', 'image'],
            ['clean'],
          ],
          handlers: {
            image: function handleImage() {
              const q = this.quill;
              const input = document.createElement('input');
              input.setAttribute('type', 'file');
              input.setAttribute('accept', 'image/*');
              input.click();
              input.onchange = async () => {
                const file = input.files?.[0];
                if (!file) return;
                const fd = new FormData();
                fd.append('file', file);
                try {
                  const res = await API.post('/api/oss/upload', fd, {
                    skipErrorHandler: true,
                  });
                  const { success, message, data } = res.data || {};
                  const url = data?.url;
                  if (!success || !url) {
                    showError(message || t('上传失败'));
                    return;
                  }
                  const range = q.getSelection(true);
                  const index = range ? range.index : Math.max(0, q.getLength() - 1);
                  q.insertEmbed(index, 'image', url);
                  q.setSelection(index + 1, 0);
                  showSuccess(t('已插入图片'));
                } catch (e) {
                  showError(
                    e?.response?.data?.message ||
                      t('上传失败，请确认已启用 OSS 并完成配置'),
                  );
                }
              };
            },
          },
        },
      },
    });

    quillRef.current = quill;

    // 程序化 setSelection 会调用 root.focus()。浏览器默认会把被聚焦的 contenteditable
    // 滚进视口，与 Quill 的 SILENT/scrollIntoView 无关；在系统设置-运营等长页面上会整页跳转到本区块
    const root = quill.root;
    const origRootFocus = root.focus.bind(root);
    root.focus = function quillRootFocusWithoutPageScroll(options) {
      const opts =
        options != null && typeof options === 'object'
          ? { ...options, preventScroll: true }
          : { preventScroll: true };
      try {
        return origRootFocus(opts);
      } catch {
        return origRootFocus.call(root, options);
      }
    };

    const initial = normalizeEmptyHtml(valueRef.current);
    if (initial) {
      syncingFromPropRef.current = true;
      quill.clipboard.dangerouslyPasteHTML(initial);
      syncingFromPropRef.current = false;
    }

    // 再兜底：移除焦点并恢复视口，防止仍有异步时序在下一帧把页面顶下去
    const scrollX = window.scrollX;
    const scrollY = window.scrollY;
    const restoreUnwantedScroll = () => {
      try {
        quill.blur();
      } catch {
        // ignore
      }
      if (window.scrollX !== scrollX || window.scrollY !== scrollY) {
        window.scrollTo({ left: scrollX, top: scrollY, behavior: 'auto' });
      }
    };
    const blurT0 = window.setTimeout(restoreUnwantedScroll, 0);
    const blurT1 = window.setTimeout(restoreUnwantedScroll, 50);
    const blurT2 = window.setTimeout(restoreUnwantedScroll, 200);

    const emitChange = () => {
      if (syncingFromPropRef.current) return;
      const html = quill.root.innerHTML;
      onChangeRef.current(normalizeEmptyHtml(html));
    };

    quill.on('text-change', emitChange);

    return () => {
      window.clearTimeout(blurT0);
      window.clearTimeout(blurT1);
      window.clearTimeout(blurT2);
      root.focus = origRootFocus;
      quill.off('text-change', emitChange);
      quillRef.current = null;
      // Snow 主题会把 .ql-toolbar 插在 quill.container 之前；只清空 el 会留下工具栏，Strict Mode / 重挂载会出现多个工具栏
      let prev = el.previousElementSibling;
      while (prev && prev.classList.contains('ql-toolbar')) {
        const node = prev;
        prev = prev.previousElementSibling;
        node.remove();
      }
      el.removeAttribute('class');
      el.removeAttribute('data-gramm');
      el.innerHTML = '';
    };
  }, []);

  useEffect(() => {
    const quill = quillRef.current;
    if (!quill) return;
    quill.enable(!disabled);
  }, [disabled]);

  useEffect(() => {
    const quill = quillRef.current;
    if (!quill) return;
    const next = normalizeEmptyHtml(value);
    const cur = normalizeEmptyHtml(quill.root.innerHTML);
    if (next === cur) return;
    const scrollX = window.scrollX;
    const scrollY = window.scrollY;
    syncingFromPropRef.current = true;
    if (!next) {
      quill.setText('');
    } else {
      quill.clipboard.dangerouslyPasteHTML(next);
    }
    syncingFromPropRef.current = false;
    // 与初始化相同：拉取配置后粘贴 HTML 会再次 focus，需避免整页被滚到本组件
    const restore = () => {
      try {
        quill.blur();
      } catch {
        // ignore
      }
      if (window.scrollX !== scrollX || window.scrollY !== scrollY) {
        window.scrollTo({ left: scrollX, top: scrollY, behavior: 'auto' });
      }
    };
    const syncT0 = window.setTimeout(restore, 0);
    const syncT1 = window.setTimeout(restore, 50);
    return () => {
      window.clearTimeout(syncT0);
      window.clearTimeout(syncT1);
    };
  }, [value]);

  return (
    <div
      className='distributor-apply-intro-editor w-full max-w-4xl rounded-md overflow-hidden bg-[var(--semi-color-bg-0)] [&_.ql-toolbar.ql-snow]:border-[var(--semi-color-border)] [&_.ql-container.ql-snow]:border-[var(--semi-color-border)] [&_.ql-container]:!min-h-[360px] [&_.ql-editor]:!min-h-[300px]'
      style={{ minHeight: 420 }}
    >
      <div ref={wrapRef} />
    </div>
  );
}
