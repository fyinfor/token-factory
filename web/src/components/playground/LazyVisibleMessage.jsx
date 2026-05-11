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

import React, { useEffect, useRef, useState } from 'react';

/** 视口外预加载边距，减少滚动时出现空白 */
const ROOT_MARGIN = '280px 0px 280px 0px';

/**
 * 操练场消息气泡：进入（或接近）可视区域后再挂载子树，减轻长对话下 DOM/媒体压力。
 * variant=media：更高占位，避免图片/视频模式切换后列表高度不足导致滚不到底部。
 */
const LazyVisibleMessage = ({
  messageId,
  className,
  variant = 'default',
  children,
}) => {
  const [visible, setVisible] = useState(false);
  const rootRef = useRef(null);

  useEffect(() => {
    const rootEl = rootRef.current;
    if (!rootEl || visible) {
      return undefined;
    }
    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) {
            setVisible(true);
            break;
          }
        }
      },
      { root: null, rootMargin: ROOT_MARGIN, threshold: 0 },
    );
    observer.observe(rootEl);
    return () => observer.disconnect();
  }, [messageId, visible]);

  const placeholderClass =
    variant === 'media'
      ? 'rounded-lg min-h-[220px] sm:min-h-[280px] w-full max-w-3xl bg-semi-color-fill-0 animate-pulse'
      : 'rounded-lg min-h-[72px] max-w-3xl bg-semi-color-fill-0 animate-pulse';

  return (
    <div ref={rootRef} className={className}>
      {visible ? (
        children
      ) : (
        <div className={placeholderClass} aria-hidden />
      )}
    </div>
  );
};

export default LazyVisibleMessage;
