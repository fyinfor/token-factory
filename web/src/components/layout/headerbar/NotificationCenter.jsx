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

import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { createPortal } from 'react-dom';
import {
  BellRing,
  Check,
  ChevronDown,
  Inbox,
  LoaderCircle,
  Megaphone,
  X,
} from 'lucide-react';
import {
  API,
  showError,
  showSuccess,
  timestamp2string,
} from '../../../helpers';
import AnnouncementList from '../AnnouncementList';

const PAGE_SIZE = 10;
const PULL_THRESHOLD = 72;

const NotificationCenter = ({
  visible,
  onClose,
  showMessages,
  messageUnreadCount,
  announcementUnreadCount,
  announcements,
  announcementUnreadKeys,
  markAnnouncementsRead,
  onMessagesMarkedRead,
  onMessageReadStateChanged,
  onRefreshAnnouncements,
  fallbackNoticeContent,
  t,
}) => {
  const [activeTab, setActiveTab] = useState('announcements');
  const [messageItems, setMessageItems] = useState([]);
  const [messageTotal, setMessageTotal] = useState(0);
  const [messagePage, setMessagePage] = useState(0);
  const [messageLoading, setMessageLoading] = useState(false);
  const [announcementLimit, setAnnouncementLimit] = useState(PAGE_SIZE);
  const [refreshing, setRefreshing] = useState(false);
  const [pullDistance, setPullDistance] = useState(0);
  const [bindActionID, setBindActionID] = useState(0);
  const [actionExitingID, setActionExitingID] = useState(0);
  const scrollRef = useRef(null);
  const closeButtonRef = useRef(null);
  const touchStartYRef = useRef(null);
  const markingMessageIDsRef = useRef(new Set());
  const hasMoreMessages = messageItems.length < messageTotal;
  const visibleAnnouncements = useMemo(
    () => announcements.slice(0, announcementLimit),
    [announcementLimit, announcements],
  );
  const hasMoreAnnouncements = announcementLimit < announcements.length;

  const markLoadedMessagesRead = useCallback(
    async (items) => {
      const unreadItems = items.filter(
        (item) =>
          item?.id &&
          !item.is_read &&
          !markingMessageIDsRef.current.has(item.id),
      );
      if (unreadItems.length === 0) {
        return;
      }
      const optimisticReadAt = Math.floor(Date.now() / 1000);
      const optimisticReadIDs = new Set(unreadItems.map((item) => item.id));
      setMessageItems((currentItems) =>
        currentItems.map((item) =>
          optimisticReadIDs.has(item.id)
            ? {
                ...item,
                is_read: true,
                read_at: item.read_at || optimisticReadAt,
              }
            : item,
        ),
      );
      onMessagesMarkedRead?.(unreadItems.length);
      unreadItems.forEach((item) => markingMessageIDsRef.current.add(item.id));
      const results = await Promise.allSettled(
        unreadItems.map((item) =>
          API.post(`/api/user/messages/${item.id}/read`, null, {
            skipErrorHandler: true,
          }),
        ),
      );
      const failedReadIDs = new Set();
      results.forEach((result, index) => {
        if (result.status === 'rejected' || !result.value?.data?.success) {
          failedReadIDs.add(unreadItems[index].id);
        }
      });
      unreadItems.forEach((item) =>
        markingMessageIDsRef.current.delete(item.id),
      );
      if (failedReadIDs.size > 0) {
        setMessageItems((currentItems) =>
          currentItems.map((item) =>
            failedReadIDs.has(item.id)
              ? { ...item, is_read: false, read_at: 0 }
              : item,
          ),
        );
      }
      await onMessageReadStateChanged?.();
    },
    [onMessageReadStateChanged, onMessagesMarkedRead],
  );

  const loadMessages = useCallback(
    async ({ page = 1, append = false } = {}) => {
      if (messageLoading) {
        return;
      }
      setMessageLoading(true);
      try {
        const res = await API.get('/api/user/messages/self', {
          params: { p: page, page_size: PAGE_SIZE, read_status: 'all' },
          skipErrorHandler: true,
        });
        const { success, message, data } = res.data || {};
        if (!success) {
          showError(message || t('加载站内消息失败'));
          return;
        }
        const nextItems = Array.isArray(data?.items) ? data.items : [];
        const total = Number(data?.total || nextItems.length || 0);
        setMessageItems((currentItems) =>
          append ? [...currentItems, ...nextItems] : nextItems,
        );
        setMessageTotal(total);
        setMessagePage(page);
        if (visible && activeTab === 'messages') {
          await markLoadedMessagesRead(nextItems);
        }
      } catch (error) {
        showError(error?.message || t('加载站内消息失败'));
      } finally {
        setMessageLoading(false);
      }
    },
    [activeTab, markLoadedMessagesRead, messageLoading, t, visible],
  );

  const handleRefresh = useCallback(async () => {
    if (refreshing) {
      return;
    }
    setRefreshing(true);
    try {
      if (activeTab === 'announcements') {
        await onRefreshAnnouncements?.();
        setAnnouncementLimit(PAGE_SIZE);
      } else {
        await loadMessages({ page: 1 });
      }
    } finally {
      setRefreshing(false);
      setPullDistance(0);
    }
  }, [activeTab, loadMessages, onRefreshAnnouncements, refreshing]);

  const handleLoadMore = useCallback(() => {
    if (activeTab === 'announcements') {
      if (hasMoreAnnouncements) {
        setAnnouncementLimit((limit) => limit + PAGE_SIZE);
      }
      return;
    }
    if (hasMoreMessages && !messageLoading) {
      loadMessages({ page: messagePage + 1, append: true });
    }
  }, [
    activeTab,
    hasMoreAnnouncements,
    hasMoreMessages,
    loadMessages,
    messageLoading,
    messagePage,
  ]);

  const handleScroll = useCallback(
    (event) => {
      const target = event.currentTarget;
      if (target.scrollHeight - target.scrollTop - target.clientHeight < 120) {
        handleLoadMore();
      }
    },
    [handleLoadMore],
  );

  const handleTouchStart = useCallback((event) => {
    if (event.currentTarget.scrollTop === 0) {
      touchStartYRef.current = event.touches[0]?.clientY ?? null;
    }
  }, []);

  const handleTouchMove = useCallback((event) => {
    if (touchStartYRef.current === null || event.currentTarget.scrollTop > 0) {
      return;
    }
    const currentY = event.touches[0]?.clientY ?? touchStartYRef.current;
    const distance = Math.max(0, currentY - touchStartYRef.current);
    if (distance > 0) {
      setPullDistance(Math.min(96, distance * 0.55));
    }
  }, []);

  const handleTouchEnd = useCallback(() => {
    touchStartYRef.current = null;
    if (pullDistance >= PULL_THRESHOLD) {
      handleRefresh();
      return;
    }
    setPullDistance(0);
  }, [handleRefresh, pullDistance]);

  const handleTabChange = useCallback((nextTab) => {
    setActiveTab(nextTab);
    setPullDistance(0);
    requestAnimationFrame(() => {
      if (scrollRef.current) {
        scrollRef.current.scrollTop = 0;
      }
    });
  }, []);

  const handleBindRequestAction = useCallback(
    async (messageItem, action) => {
      const requestID = Number(messageItem?.biz_id || 0);
      if (!requestID || bindActionID > 0) {
        return;
      }
      setBindActionID(requestID);
      try {
        const res = await API.post(
          `/api/distributor/bind_requests/${requestID}/${action}`,
          null,
          { skipErrorHandler: true },
        );
        const { success, message, data } = res.data || {};
        if (!success) {
          showError(message || t('处理绑定请求失败'));
          return;
        }
        showSuccess(message || t('绑定请求已处理'));
        const nextStatus = Number(
          data?.status || (action === 'accept' ? 2 : 3),
        );
        setActionExitingID(requestID);
        window.setTimeout(() => {
          setMessageItems((items) =>
            items.map((item) =>
              item.id === messageItem.id
                ? {
                    ...item,
                    is_read: true,
                    read_at: item.read_at || Math.floor(Date.now() / 1000),
                    bind_request_status: nextStatus,
                  }
                : item,
            ),
          );
          setActionExitingID(0);
        }, 260);
        await onMessageReadStateChanged?.();
      } catch (error) {
        showError(error?.message || t('处理绑定请求失败'));
      } finally {
        setBindActionID(0);
      }
    },
    [bindActionID, onMessageReadStateChanged, t],
  );

  useEffect(() => {
    if (!visible) {
      return undefined;
    }
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    const handleKeyDown = (event) => {
      if (event.key === 'Escape') {
        onClose();
      }
    };
    document.addEventListener('keydown', handleKeyDown);
    window.setTimeout(() => closeButtonRef.current?.focus(), 40);
    return () => {
      document.body.style.overflow = previousOverflow;
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [onClose, visible]);

  useEffect(() => {
    if (!visible) {
      return;
    }
    setActiveTab('announcements');
    setAnnouncementLimit(PAGE_SIZE);
    setMessageItems([]);
    setMessageTotal(0);
    setMessagePage(0);
  }, [visible]);

  useEffect(() => {
    if (!showMessages && activeTab === 'messages') {
      handleTabChange('announcements');
    }
  }, [activeTab, handleTabChange, showMessages]);

  useEffect(() => {
    if (visible && activeTab === 'announcements') {
      markAnnouncementsRead(visibleAnnouncements);
    }
  }, [activeTab, markAnnouncementsRead, visible, visibleAnnouncements]);

  useEffect(() => {
    if (!visible || activeTab !== 'messages') {
      return;
    }
    if (messagePage === 0) {
      loadMessages({ page: 1 });
    } else {
      markLoadedMessagesRead(messageItems);
    }
    // messageItems is intentionally handled by the load/append path.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeTab, visible]);

  const renderMessageList = () => {
    if (messageLoading && messageItems.length === 0) {
      return (
        <div className='notification-empty-state'>
          <LoaderCircle className='notification-spinner' size={26} />
          <span>{t('加载中...')}</span>
        </div>
      );
    }
    if (messageItems.length === 0) {
      return (
        <div className='notification-empty-state'>
          <Inbox size={30} />
          <span>{t('暂无站内消息')}</span>
        </div>
      );
    }
    return messageItems.map((item) => {
      const isBindRequest =
        item.biz_type === 'distributor_bind_request' &&
        item.type === 'distributor_bind_request';
      const bindStatus = Number(item.bind_request_status);
      const requestID = Number(item.biz_id || 0);
      return (
        <article className='notification-message-item' key={item.id}>
          <div className='notification-message-heading'>
            <div>
              <h3>{item.title || t('消息')}</h3>
              <time>{timestamp2string(item.created_at || 0)}</time>
            </div>
            {!item.is_read ? (
              <span
                className='notification-unread-dot'
                aria-label={t('未读')}
              />
            ) : null}
          </div>
          <p>{item.content || ''}</p>
          {isBindRequest ? (
            <div
              className={`notification-message-actions ${
                actionExitingID === requestID ? 'is-exiting' : ''
              }`}
            >
              {bindStatus === 1 ? (
                <>
                  <button
                    className='notification-action-primary'
                    disabled={bindActionID > 0}
                    onClick={() => handleBindRequestAction(item, 'accept')}
                    type='button'
                  >
                    {bindActionID === requestID ? (
                      <LoaderCircle
                        className='notification-spinner'
                        size={15}
                      />
                    ) : (
                      <Check size={15} />
                    )}
                    <span>{t('接受')}</span>
                  </button>
                  <button
                    className='notification-action-secondary'
                    disabled={bindActionID > 0}
                    onClick={() => handleBindRequestAction(item, 'reject')}
                    type='button'
                  >
                    <X size={15} />
                    <span>{t('拒绝')}</span>
                  </button>
                </>
              ) : bindStatus === 2 ? (
                <span className='notification-action-result is-accepted'>
                  <Check size={14} /> {t('已接受')}
                </span>
              ) : bindStatus === 3 ? (
                <span className='notification-action-result is-rejected'>
                  <X size={14} /> {t('已拒绝')}
                </span>
              ) : null}
            </div>
          ) : null}
        </article>
      );
    });
  };

  return createPortal(
    <div
      className='notification-center-root'
      data-open={visible ? 'true' : 'false'}
      aria-hidden={!visible}
    >
      <button
        className='notification-center-backdrop'
        onClick={onClose}
        aria-label={t('关闭')}
        tabIndex={visible ? 0 : -1}
        type='button'
      />
      <aside
        className='notification-center-panel'
        role='dialog'
        aria-modal='true'
        aria-label={t('通知')}
      >
        <header className='notification-center-header'>
          <div className='notification-center-title'>
            <span className='notification-center-title-icon'>
              <BellRing size={18} />
            </span>
            <strong>{t('通知')}</strong>
          </div>
          <button
            ref={closeButtonRef}
            className='notification-icon-button'
            onClick={onClose}
            aria-label={t('关闭')}
            tabIndex={visible ? 0 : -1}
            type='button'
          >
            <X size={19} />
          </button>
        </header>

        <nav
          className={`notification-center-tabs ${showMessages ? '' : 'is-single'}`}
          aria-label={t('通知')}
        >
          <button
            className={activeTab === 'announcements' ? 'is-active' : ''}
            onClick={() => handleTabChange('announcements')}
            type='button'
          >
            <span>{t('系统公告')}</span>
            {announcementUnreadCount > 0 ? (
              <b>
                {announcementUnreadCount > 99 ? '99+' : announcementUnreadCount}
              </b>
            ) : null}
          </button>
          {showMessages ? (
            <button
              className={activeTab === 'messages' ? 'is-active' : ''}
              onClick={() => handleTabChange('messages')}
              type='button'
            >
              <span>{t('站内消息')}</span>
              {messageUnreadCount > 0 ? (
                <b>{messageUnreadCount > 99 ? '99+' : messageUnreadCount}</b>
              ) : null}
            </button>
          ) : null}
          <span
            className='notification-tab-indicator'
            data-position={activeTab}
          />
        </nav>

        <div
          ref={scrollRef}
          className='notification-center-scroll'
          onScroll={handleScroll}
          onTouchStart={handleTouchStart}
          onTouchMove={handleTouchMove}
          onTouchEnd={handleTouchEnd}
        >
          <div
            className={`notification-pull-indicator ${
              pullDistance >= PULL_THRESHOLD ? 'is-ready' : ''
            }`}
            style={{ height: `${pullDistance}px` }}
            aria-hidden='true'
          >
            {refreshing ? (
              <LoaderCircle className='notification-spinner' size={20} />
            ) : (
              <ChevronDown size={20} />
            )}
          </div>
          <div className='notification-center-content'>
            {activeTab === 'announcements' ? (
              visibleAnnouncements.length === 0 && !fallbackNoticeContent ? (
                <div className='notification-empty-state'>
                  <Megaphone size={30} />
                  <span>{t('暂无系统公告')}</span>
                </div>
              ) : (
                <AnnouncementList
                  items={visibleAnnouncements}
                  unreadKeys={announcementUnreadKeys}
                  fallbackContent={fallbackNoticeContent}
                />
              )
            ) : (
              renderMessageList()
            )}
          </div>
          {(messageLoading && messageItems.length > 0) || refreshing ? (
            <div className='notification-loading-more'>
              <LoaderCircle className='notification-spinner' size={18} />
            </div>
          ) : null}
        </div>
      </aside>
    </div>,
    document.body,
  );
};

export default NotificationCenter;
