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
import { Bell, BellRing, ChevronRight } from 'lucide-react';

const NotificationButton = ({
  unreadCount,
  bubble,
  bubbleVisible,
  onNoticeOpen,
  t,
}) => (
  <div className='header-notification-wrap'>
    <button
      className='header-notification-button'
      aria-label={t('通知')}
      onClick={onNoticeOpen}
      title={t('通知')}
      type='button'
    >
      <Bell size={18} />
      {unreadCount > 0 ? (
        <span className='header-notification-count'>
          {unreadCount > 99 ? '99+' : unreadCount}
        </span>
      ) : null}
    </button>
    {bubble ? (
      <button
        className='header-notification-preview'
        data-visible={bubbleVisible ? 'true' : 'false'}
        aria-hidden={!bubbleVisible}
        onClick={onNoticeOpen}
        tabIndex={bubbleVisible ? 0 : -1}
        type='button'
      >
        <span className='notification-preview-pointer' aria-hidden='true' />
        <span className='notification-preview-icon' aria-hidden='true'>
          <BellRing size={16} />
        </span>
        <span className='notification-preview-copy'>
          <strong>{bubble.title}</strong>
          <span>{bubble.message}</span>
        </span>
        <ChevronRight
          className='notification-preview-arrow'
          size={16}
          aria-hidden='true'
        />
      </button>
    ) : null}
  </div>
);

export default NotificationButton;
