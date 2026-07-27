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

import React, { useContext, useEffect } from 'react';
import { createPortal } from 'react-dom';
import { BellRing, Megaphone, X } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { StatusContext } from '../../context/Status';
import { useNotifications } from '../../hooks/common/useNotifications';
import AnnouncementList from './AnnouncementList';

const NoticeModal = ({
  visible,
  onClose,
  onViewMore,
  fallbackContent = '',
}) => {
  const { t } = useTranslation();
  const [statusState] = useContext(StatusContext);
  const { announcements, unreadKeys, markAnnouncementsRead } =
    useNotifications(statusState);
  const visibleAnnouncements = announcements.slice(0, 1);

  useEffect(() => {
    if (!visible) {
      return undefined;
    }
    if (visibleAnnouncements.length > 0) {
      markAnnouncementsRead(visibleAnnouncements);
    }
    const handleKeyDown = (event) => {
      if (event.key === 'Escape') {
        onClose();
      }
    };
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
    // The visible announcement batch is fixed for the lifetime of this modal.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [visible]);

  return createPortal(
    <div
      className='home-notice-root'
      data-open={visible ? 'true' : 'false'}
      aria-hidden={!visible}
    >
      <button
        className='home-notice-backdrop'
        aria-label={t('关闭')}
        onClick={onClose}
        tabIndex={visible ? 0 : -1}
        type='button'
      />
      <section
        className='home-notice-dialog'
        role='dialog'
        aria-modal='true'
        aria-label={t('系统公告')}
      >
        <header className='home-notice-header'>
          <div className='home-notice-heading'>
            <span className='home-notice-heading-icon'>
              <Megaphone size={20} />
            </span>
            <div>
              <h2>{t('系统公告')}</h2>
              <p>{t('通知')}</p>
            </div>
          </div>
          <button
            className='notification-icon-button'
            aria-label={t('关闭')}
            onClick={onClose}
            tabIndex={visible ? 0 : -1}
            type='button'
          >
            <X size={19} />
          </button>
        </header>

        <div className='home-notice-scroll'>
          {visibleAnnouncements.length === 0 && !fallbackContent ? (
            <div className='notification-empty-state'>
              <Megaphone size={30} />
              <span>{t('暂无系统公告')}</span>
            </div>
          ) : (
            <AnnouncementList
              items={visibleAnnouncements}
              unreadKeys={unreadKeys}
              fallbackContent={fallbackContent}
            />
          )}
        </div>

        <footer className='home-notice-footer'>
          <button
            className='home-notice-secondary-button'
            onClick={onViewMore}
            tabIndex={visible ? 0 : -1}
            type='button'
          >
            <BellRing size={16} />
            <span>{t('查看更多')}</span>
          </button>
          <button
            className='home-notice-primary-button'
            onClick={onClose}
            tabIndex={visible ? 0 : -1}
            type='button'
          >
            {t('关闭公告')}
          </button>
        </footer>
      </section>
    </div>,
    document.body,
  );
};

export default NoticeModal;
