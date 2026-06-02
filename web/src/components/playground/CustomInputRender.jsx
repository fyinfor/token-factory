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

import React, { useRef, useEffect, useCallback, useState } from 'react';
import { Toast } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { usePlayground } from '../../contexts/PlaygroundContext';

const CustomInputRender = (props) => {
  const { t } = useTranslation();
  const { onPasteImage, imageEnabled } = usePlayground();
  const { detailProps } = props;
  const { inputNode, sendNode, onClick } = detailProps;
  const containerRef = useRef(null);
  const [isInputFocused, setIsInputFocused] = useState(false);

  const handlePaste = useCallback(
    async (e) => {
      const items = e.clipboardData?.items;
      if (!items) return;

      for (let i = 0; i < items.length; i++) {
        const item = items[i];

        if (item.type.indexOf('image') !== -1) {
          e.preventDefault();
          const file = item.getAsFile();

          if (file) {
            try {
              if (!imageEnabled) {
                Toast.warning({
                  content: t('请先在设置中启用图片功能'),
                  duration: 3,
                });
                return;
              }

              const reader = new FileReader();
              reader.onload = (event) => {
                const base64 = event.target.result;

                if (onPasteImage) {
                  onPasteImage(base64);
                  Toast.success({
                    content: t('图片已添加'),
                    duration: 2,
                  });
                } else {
                  Toast.error({
                    content: t('无法添加图片'),
                    duration: 2,
                  });
                }
              };
              reader.onerror = () => {
                console.error('Failed to read image file:', reader.error);
                Toast.error({
                  content: t('粘贴图片失败'),
                  duration: 2,
                });
              };
              reader.readAsDataURL(file);
            } catch (error) {
              console.error('Failed to paste image:', error);
              Toast.error({
                content: t('粘贴图片失败'),
                duration: 2,
              });
            }
          }
          break;
        }
      }
    },
    [onPasteImage, imageEnabled, t],
  );

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    container.addEventListener('paste', handlePaste);
    return () => {
      container.removeEventListener('paste', handlePaste);
    };
  }, [handlePaste]);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const handleFocusIn = () => {
      setIsInputFocused(true);
    };

    const handleFocusOut = (event) => {
      if (!container.contains(event.relatedTarget)) {
        setIsInputFocused(false);
      }
    };

    container.addEventListener('focusin', handleFocusIn);
    container.addEventListener('focusout', handleFocusOut);
    return () => {
      container.removeEventListener('focusin', handleFocusIn);
      container.removeEventListener('focusout', handleFocusOut);
    };
  }, []);

  // 发送按钮
  const styledSendNode = React.cloneElement(sendNode, {
    className: `!rounded-full !bg-purple-500 hover:!bg-purple-600 flex-shrink-0 transition-all ${sendNode.props.className || ''}`,
    style: {
      ...sendNode.props.style,
      width: '32px',
      height: '32px',
      minWidth: '32px',
      padding: 0,
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
    },
  });

  return (
    <div className='p-2 sm:p-4 relative' ref={containerRef}>
      <div
        className={`transition-all duration-300 ease-out absolute ${
          isInputFocused ? 'top-[-30px] opacity-100 mb-2' : 'top-0 opacity-0 mb-0'
        }`}
      >
        <div
          className='rounded-xl sm:rounded-2xl px-3 py-2 text-xs sm:text-sm text-gray-600 dark:text-gray-300 shadow-md'
          style={{ border: '1px solid var(--semi-color-border)', backgroundColor: 'var(--semi-color-bg-1)' }}
        >
          {t('使用操练场会产生扣费，请确认模型与参数后再发送。')}
        </div>
      </div>
      <div
        className='flex items-center gap-2 sm:gap-3 p-2 bg-gray-50 dark:bg-gray-800 rounded-xl sm:rounded-2xl shadow-sm hover:shadow-md transition-shadow'
        style={{ border: '1px solid var(--semi-color-border)' }}
        onClick={onClick}
        title={t('支持 Ctrl+V 粘贴图片')}
      >
        <div className='flex-1'>{inputNode}</div>
        {/* 发送按钮 - 右边 */}
        {styledSendNode}
      </div>
    </div>
  );
};

export default CustomInputRender;
