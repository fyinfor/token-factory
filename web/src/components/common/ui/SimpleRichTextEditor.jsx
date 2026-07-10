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

const FONT_OPTIONS = [
  { label: '默认字体', value: '' },
  { label: '系统字体', value: 'system-ui' },
  { label: '微软雅黑', value: '"Microsoft YaHei"' },
  { label: '苹方', value: '"PingFang SC"' },
  { label: 'Arial', value: 'Arial' },
  { label: 'Georgia', value: 'Georgia' },
  { label: '宋体', value: 'SimSun' },
  { label: '等宽', value: 'monospace' },
];

const SIZE_OPTIONS = [
  { label: '默认字号', value: '' },
  ...[
    '12px',
    '14px',
    '16px',
    '18px',
    '20px',
    '24px',
    '28px',
    '32px',
    '36px',
    '42px',
    '48px',
    '56px',
  ].map((value) => ({ label: value, value })),
];

let typographyRegistered = false;

function registerQuillTypography() {
  if (typographyRegistered) return;

  const FontStyle = Quill.import('attributors/style/font');
  FontStyle.whitelist = FONT_OPTIONS.map((item) =>
    item.value.replace(/["']/g, ''),
  ).filter(Boolean);
  Quill.register(FontStyle, true);

  const SizeStyle = Quill.import('attributors/style/size');
  SizeStyle.whitelist = SIZE_OPTIONS.map((item) => item.value).filter(Boolean);
  Quill.register(SizeStyle, true);

  typographyRegistered = true;
}

function normalizeEmptyHtml(html) {
  const text = String(html ?? '').trim();
  if (!text || text === '<p><br></p>' || text === '<p></p>') {
    return '';
  }
  return text;
}

function createToolbarSelect(className, options) {
  const select = document.createElement('select');
  select.className = className;
  options.forEach((item) => {
    const option = document.createElement('option');
    option.value = item.value;
    option.textContent = item.label;
    select.appendChild(option);
  });
  return select;
}

function createToolbarButton(className) {
  const button = document.createElement('button');
  button.type = 'button';
  button.className = className;
  return button;
}

function createToolbarElement() {
  const toolbar = document.createElement('div');
  toolbar.appendChild(
    createToolbarSelect('ql-header', [
      { label: '标题 1', value: '1' },
      { label: '标题 2', value: '2' },
      { label: '标题 3', value: '3' },
      { label: '正文', value: '' },
    ]),
  );
  toolbar.appendChild(createToolbarSelect('ql-font', FONT_OPTIONS));
  toolbar.appendChild(createToolbarSelect('ql-size', SIZE_OPTIONS));
  toolbar.appendChild(createToolbarButton('ql-bold'));
  toolbar.appendChild(createToolbarButton('ql-italic'));
  toolbar.appendChild(createToolbarButton('ql-underline'));
  toolbar.appendChild(createToolbarButton('ql-strike'));
  toolbar.appendChild(createToolbarSelect('ql-color', []));
  toolbar.appendChild(createToolbarSelect('ql-background', []));
  toolbar.appendChild(createToolbarButton('ql-link'));
  toolbar.appendChild(createToolbarButton('ql-clean'));
  return toolbar;
}

export default function SimpleRichTextEditor({
  value,
  onChange,
  disabled,
  placeholder,
  minHeight = 120,
}) {
  const wrapRef = useRef(null);
  const toolbarHostRef = useRef(null);
  const quillRef = useRef(null);
  const valueRef = useRef(value);
  const onChangeRef = useRef(onChange);
  const syncingRef = useRef(false);
  const initialDisabledRef = useRef(Boolean(disabled));
  const placeholderRef = useRef(placeholder);
  const minHeightRef = useRef(minHeight);

  valueRef.current = value;
  onChangeRef.current = onChange;

  useEffect(() => {
    const el = wrapRef.current;
    const toolbarHost = toolbarHostRef.current;
    if (!el || !toolbarHost) return undefined;

    registerQuillTypography();
    toolbarHost.innerHTML = '';
    const toolbar = createToolbarElement();
    toolbarHost.appendChild(toolbar);

    const quill = new Quill(el, {
      theme: 'snow',
      readOnly: initialDisabledRef.current,
      placeholder: placeholderRef.current,
      modules: {
        toolbar,
      },
      formats: [
        'header',
        'font',
        'size',
        'bold',
        'italic',
        'underline',
        'strike',
        'color',
        'background',
        'link',
      ],
    });
    quillRef.current = quill;

    const root = quill.root;
    root.style.minHeight = `${minHeightRef.current}px`;
    const originalFocus = root.focus.bind(root);
    root.focus = function focusWithoutPageScroll(options) {
      const opts =
        options != null && typeof options === 'object'
          ? { ...options, preventScroll: true }
          : { preventScroll: true };
      try {
        return originalFocus(opts);
      } catch {
        return originalFocus.call(root, options);
      }
    };

    const initial = normalizeEmptyHtml(valueRef.current);
    if (initial) {
      syncingRef.current = true;
      quill.clipboard.dangerouslyPasteHTML(initial);
      syncingRef.current = false;
      quill.blur();
    }

    const emitChange = () => {
      if (syncingRef.current) return;
      onChangeRef.current?.(normalizeEmptyHtml(quill.root.innerHTML));
    };
    quill.on('text-change', emitChange);

    return () => {
      quill.off('text-change', emitChange);
      root.focus = originalFocus;
      quillRef.current = null;
      el.removeAttribute('class');
      el.removeAttribute('data-gramm');
      el.innerHTML = '';
      toolbarHost.innerHTML = '';
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
    const current = normalizeEmptyHtml(quill.root.innerHTML);
    if (next === current) return;

    syncingRef.current = true;
    if (next) {
      quill.clipboard.dangerouslyPasteHTML(next);
    } else {
      quill.setText('');
    }
    syncingRef.current = false;
    quill.blur();
  }, [value]);

  return (
    <div
      className='simple-rich-text-editor w-full overflow-hidden rounded-md bg-[var(--semi-color-bg-0)] [&_.ql-container.ql-snow]:border-[var(--semi-color-border)] [&_.ql-editor]:text-[var(--semi-color-text-0)] [&_.ql-picker.ql-font]:!w-[132px] [&_.ql-picker.ql-size]:!w-[86px] [&_.ql-toolbar.ql-snow]:border-[var(--semi-color-border)]'
      style={{ minHeight: minHeight + 44 }}
    >
      <div ref={toolbarHostRef} />
      <div ref={wrapRef} />
    </div>
  );
}
