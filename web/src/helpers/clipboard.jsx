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

import React from 'react';
import { Modal, TextArea, Typography } from '@douyinfe/semi-ui';

/** 当前环境是否支持 Clipboard API 读取（需 HTTPS 或 localhost） */
export function isClipboardReadSupported() {
  return Boolean(
    typeof window !== 'undefined' &&
      window.isSecureContext &&
      typeof navigator?.clipboard?.readText === 'function',
  );
}

/**
 * 查询剪贴板读取权限；不支持 query 时返回 prompt（尝试读取会触发授权）
 * @returns {'granted'|'denied'|'prompt'|'unsupported'}
 */
export async function queryClipboardReadPermission() {
  if (!isClipboardReadSupported()) {
    return 'unsupported';
  }
  try {
    if (typeof navigator.permissions?.query !== 'function') {
      return 'prompt';
    }
    const status = await navigator.permissions.query({
      name: 'clipboard-read',
    });
    return status.state;
  } catch {
    return 'prompt';
  }
}

/**
 * 尝试读取剪贴板文本
 * @throws {Error} message: unsupported | denied | failed
 */
export async function readClipboardText() {
  if (!isClipboardReadSupported()) {
    throw new Error('unsupported');
  }
  const permission = await queryClipboardReadPermission();
  if (permission === 'denied') {
    throw new Error('denied');
  }
  try {
    return await navigator.clipboard.readText();
  } catch (err) {
    if (err?.name === 'NotAllowedError') {
      throw new Error('denied');
    }
    throw new Error('failed');
  }
}

/**
 * 弹出文本框供用户手动粘贴
 * @returns {Promise<string|null>} 确认后返回 trim 后文本；取消返回 null
 */
export function promptManualPaste({
  t,
  title,
  description,
  placeholder,
}) {
  return new Promise((resolve) => {
    let value = '';
    Modal.confirm({
      title,
      centered: true,
      content: (
        <div className='flex flex-col gap-2'>
          {description ? (
            <Typography.Text type='secondary' size='small'>
              {description}
            </Typography.Text>
          ) : null}
          <TextArea
            autosize={{ minRows: 4, maxRows: 12 }}
            placeholder={placeholder}
            onChange={(v) => {
              value = v;
            }}
          />
        </div>
      ),
      okText: t('确定'),
      cancelText: t('取消'),
      onOk: () => {
        const trimmed = String(value ?? '').trim();
        resolve(trimmed || null);
      },
      onCancel: () => resolve(null),
    });
  });
}

/**
 * 优先读取剪贴板；不可用或失败时打开手动粘贴框
 * @returns {Promise<string|null>}
 */
export async function readClipboardTextOrManualPaste({
  t,
  manualTitle,
  manualDescription,
  manualPlaceholder,
}) {
  const manual = () =>
    promptManualPaste({
      t,
      title: manualTitle,
      description: manualDescription,
      placeholder: manualPlaceholder,
    });

  try {
    return await readClipboardText();
  } catch {
    return manual();
  }
}
