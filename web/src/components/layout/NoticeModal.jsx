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

import React, { useCallback, useContext, useEffect, useState } from 'react';
import { createPortal } from 'react-dom';
import { CalendarCheck, LoaderCircle, Megaphone, X } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { API, showError } from '../../helpers';
import { StatusContext } from '../../context/Status';
import { useNotifications } from '../../hooks/common/useNotifications';
import AnnouncementList from './AnnouncementList';

const NoticeModal = ({ visible, onClose }) => {
  const { t } = useTranslation();
  const [statusState] = useContext(StatusContext);
  const [noticeContent, setNoticeContent] = useState('');
  const [loading, setLoading] = useState(false);
  const { announcements, unreadKeys, markAnnouncementsRead } =
    useNotifications(statusState);
  const visibleAnnouncements = announcements.slice(0, 20);

  const loadLegacyNotice = useCallback(async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/notice', { skipErrorHandler: true });
      const { success, message, data } = res.data || {};
      if (!success) {
        showError(message || t('加载中...'));
        return;
      }
      setNoticeContent(data || '');
    } catch (error) {
      showError(error?.message || t('加载中...'));
    } finally {
      setLoading(false);
    }
  }, [t]);

  const handleCloseToday = useCallback(() => {
    localStorage.setItem('notice_close_date', new Date().toDateString());
    onClose();
  }, [onClose]);

  useEffect(() => {
    if (!visible) {
      return undefined;
    }
    if (visibleAnnouncements.length > 0) {
      markAnnouncementsRead(visibleAnnouncements);
    } else {
      loadLegacyNotice();
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
          {loading ? (
            <div className='notification-empty-state'>
              <LoaderCircle className='notification-spinner' size={26} />
              <span>{t('加载中...')}</span>
            </div>
          ) : visibleAnnouncements.length === 0 && !noticeContent ? (
            <div className='notification-empty-state'>
              <Megaphone size={30} />
              <span>{t('暂无系统公告')}</span>
            </div>
          ) : (
            <AnnouncementList
              items={visibleAnnouncements}
              unreadKeys={unreadKeys}
              fallbackContent={noticeContent}
            />
          )}
        </div>

        <footer className='home-notice-footer'>
          <button
            className='home-notice-secondary-button'
            onClick={handleCloseToday}
            tabIndex={visible ? 0 : -1}
            type='button'
          >
            <CalendarCheck size={16} />
            <span>{t('今日关闭')}</span>
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
