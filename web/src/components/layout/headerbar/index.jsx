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

import React, { useCallback, useEffect, useRef, useState } from 'react';
import { useHeaderBar } from '../../../hooks/common/useHeaderBar';
import { useNavigation } from '../../../hooks/common/useNavigation';
import { useUserMessageUnreadCount } from '../../../hooks/common/useUserMessageUnreadCount';
import {
  OPEN_NOTIFICATION_CENTER_EVENT,
  useNotifications,
} from '../../../hooks/common/useNotifications';
import { API, setStatusData } from '../../../helpers';
import NotificationCenter from './NotificationCenter';
import MobileMenuButton from './MobileMenuButton';
import HeaderLogo from './HeaderLogo';
import MobileSiteNavDropdown from './MobileSiteNavDropdown';
import Navigation from './Navigation';
import ActionButtons from './ActionButtons';
import SettingsQuickSearch from './SettingsQuickSearch';

const NOTIFICATION_POLL_INTERVAL_MS = 2 * 60 * 1000;
const BUBBLE_MERGE_DELAY_MS = 450;
const BUBBLE_VISIBLE_DURATION_MS = 8000;

const HeaderBar = ({ onMobileMenuToggle, drawerOpen }) => {
  const {
    userState,
    statusState,
    statusDispatch,
    isMobile,
    collapsed,
    logoLoaded,
    currentLang,
    isLoading,
    systemName,
    logo,
    isNewYear,
    isSelfUseMode,
    docsNav,
    isDemoSiteMode,
    isConsoleRoute,
    theme,
    headerNavModules,
    pricingRequireAuth,
    logout,
    handleLanguageChange,
    handleThemeToggle,
    handleMobileMenuToggle,
    navigate,
    t,
  } = useHeaderBar({ onMobileMenuToggle, drawerOpen });

  const [notificationCenterVisible, setNotificationCenterVisible] =
    useState(false);
  const [fallbackNoticeContent, setFallbackNoticeContent] = useState('');
  const [notificationBubble, setNotificationBubble] = useState(null);
  const [notificationBubbleVisible, setNotificationBubbleVisible] =
    useState(false);
  const notificationCountsRef = useRef({
    initialized: false,
    messages: 0,
    announcements: 0,
  });
  const pendingIncomingRef = useRef({ messages: 0, announcements: 0 });
  const bubbleMergeTimerRef = useRef(null);
  const bubbleHideTimerRef = useRef(null);
  const bubbleRemoveTimerRef = useRef(null);
  const {
    unreadCount: messageUnreadCount,
    initialized: messageCountInitialized,
    refreshUnreadCount,
    reduceUnreadCount,
  } = useUserMessageUnreadCount(userState?.user);
  const {
    announcements,
    unreadCount: announcementUnreadCount,
    unreadKeys: announcementUnreadKeys,
    markAnnouncementsRead,
  } = useNotifications(statusState);

  const applyNotificationStatus = useCallback(
    (statusRes) => {
      if (!statusRes?.data?.success) {
        return;
      }
      statusDispatch({ type: 'set', payload: statusRes.data.data });
      setStatusData(statusRes.data.data);
    },
    [statusDispatch],
  );

  const refreshAnnouncementStatus = useCallback(async () => {
    const statusRes = await API.get('/api/status', {
      skipErrorHandler: true,
    });
    applyNotificationStatus(statusRes);
  }, [applyNotificationStatus]);

  const refreshAnnouncementData = useCallback(async () => {
    const [statusRes, noticeRes] = await Promise.all([
      API.get('/api/status', { skipErrorHandler: true }),
      API.get('/api/notice', { skipErrorHandler: true }),
    ]);
    applyNotificationStatus(statusRes);
    if (noticeRes?.data?.success) {
      setFallbackNoticeContent(noticeRes.data.data || '');
    }
  }, [applyNotificationStatus]);

  const hideNotificationBubble = useCallback(() => {
    window.clearTimeout(bubbleHideTimerRef.current);
    window.clearTimeout(bubbleRemoveTimerRef.current);
    setNotificationBubbleVisible(false);
    bubbleRemoveTimerRef.current = window.setTimeout(() => {
      setNotificationBubble(null);
    }, 380);
  }, []);

  const showNotificationBubble = useCallback((bubble) => {
    window.clearTimeout(bubbleHideTimerRef.current);
    window.clearTimeout(bubbleRemoveTimerRef.current);
    setNotificationBubble(bubble);
    setNotificationBubbleVisible(false);
    requestAnimationFrame(() => {
      requestAnimationFrame(() => setNotificationBubbleVisible(true));
    });
    bubbleHideTimerRef.current = window.setTimeout(() => {
      setNotificationBubbleVisible(false);
      bubbleRemoveTimerRef.current = window.setTimeout(() => {
        setNotificationBubble(null);
      }, 380);
    }, BUBBLE_VISIBLE_DURATION_MS);
  }, []);

  const buildNotificationBubble = useCallback(
    ({ messages, announcements: newAnnouncements }) => {
      if (messages > 0 && newAnnouncements > 0) {
        return {
          title: t('有新的动态'),
          message: t(
            '新公告和站内消息都到了，共 {{count}} 条未查看，记得来看看。',
            { count: messageUnreadCount + announcementUnreadCount },
          ),
        };
      }
      if (newAnnouncements > 0) {
        return announcementUnreadCount === 1
          ? {
              title: t('新公告提醒'),
              message: t('有一则新公告，请您抽空了解一下。'),
            }
          : {
              title: t('新公告提醒'),
              message: t('您有 {{count}} 则新公告未查看，请及时查看。', {
                count: announcementUnreadCount,
              }),
            };
      }
      return messageUnreadCount === 1
        ? {
            title: t('新消息到了'),
            message: t('您有 1 条新消息未查看，记得抽空看看。'),
          }
        : {
            title: t('温馨提醒'),
            message: t('您有 {{count}} 条消息未查看，记得及时看看。', {
              count: messageUnreadCount,
            }),
          };
    },
    [announcementUnreadCount, messageUnreadCount, t],
  );

  const handleNotificationCenterOpen = useCallback(() => {
    pendingIncomingRef.current = { messages: 0, announcements: 0 };
    window.clearTimeout(bubbleMergeTimerRef.current);
    hideNotificationBubble();
    setNotificationCenterVisible(true);
    refreshAnnouncementData().catch(() => {});
  }, [hideNotificationBubble, refreshAnnouncementData]);

  const handleNotificationCenterClose = useCallback(async () => {
    setNotificationCenterVisible(false);
    await refreshUnreadCount();
  }, [refreshUnreadCount]);

  useEffect(() => {
    window.addEventListener(
      OPEN_NOTIFICATION_CENTER_EVENT,
      handleNotificationCenterOpen,
    );
    return () =>
      window.removeEventListener(
        OPEN_NOTIFICATION_CENTER_EVENT,
        handleNotificationCenterOpen,
      );
  }, [handleNotificationCenterOpen]);

  useEffect(() => {
    if (!userState?.user?.id) {
      notificationCountsRef.current = {
        initialized: false,
        messages: 0,
        announcements: 0,
      };
      pendingIncomingRef.current = { messages: 0, announcements: 0 };
      hideNotificationBubble();
      return undefined;
    }
    const timer = window.setInterval(() => {
      refreshAnnouncementStatus().catch(() => {});
    }, NOTIFICATION_POLL_INTERVAL_MS);
    return () => window.clearInterval(timer);
  }, [hideNotificationBubble, refreshAnnouncementStatus, userState?.user?.id]);

  useEffect(() => {
    const announcementCountInitialized = Boolean(statusState?.status);
    if (!messageCountInitialized || !announcementCountInitialized) {
      return;
    }
    const previous = notificationCountsRef.current;
    if (!previous.initialized) {
      notificationCountsRef.current = {
        initialized: true,
        messages: messageUnreadCount,
        announcements: announcementUnreadCount,
      };
      return;
    }
    const messageIncrease = Math.max(0, messageUnreadCount - previous.messages);
    const announcementIncrease = Math.max(
      0,
      announcementUnreadCount - previous.announcements,
    );
    notificationCountsRef.current = {
      initialized: true,
      messages: messageUnreadCount,
      announcements: announcementUnreadCount,
    };
    if (
      notificationCenterVisible ||
      (messageIncrease === 0 && announcementIncrease === 0)
    ) {
      return;
    }
    pendingIncomingRef.current.messages += messageIncrease;
    pendingIncomingRef.current.announcements += announcementIncrease;
    window.clearTimeout(bubbleMergeTimerRef.current);
    bubbleMergeTimerRef.current = window.setTimeout(() => {
      const incoming = pendingIncomingRef.current;
      pendingIncomingRef.current = { messages: 0, announcements: 0 };
      showNotificationBubble(buildNotificationBubble(incoming));
    }, BUBBLE_MERGE_DELAY_MS);
  }, [
    announcementUnreadCount,
    buildNotificationBubble,
    messageCountInitialized,
    messageUnreadCount,
    notificationCenterVisible,
    showNotificationBubble,
    statusState?.status,
  ]);

  useEffect(
    () => () => {
      window.clearTimeout(bubbleMergeTimerRef.current);
      window.clearTimeout(bubbleHideTimerRef.current);
      window.clearTimeout(bubbleRemoveTimerRef.current);
    },
    [],
  );

  const { mainNavLinks } = useNavigation(t, docsNav, headerNavModules);

  return (
    <header className='text-semi-color-text-0 sticky top-0 z-50 bg-[rgba(255,255,255,0.92)] backdrop-blur-xl transition-colors duration-300 dark:bg-[rgba(24,24,27,0.75)]'>
      <NotificationCenter
        visible={notificationCenterVisible}
        onClose={handleNotificationCenterClose}
        showMessages={Boolean(userState?.user?.id)}
        messageUnreadCount={messageUnreadCount}
        announcementUnreadCount={announcementUnreadCount}
        announcements={announcements}
        announcementUnreadKeys={announcementUnreadKeys}
        markAnnouncementsRead={markAnnouncementsRead}
        onMessagesMarkedRead={reduceUnreadCount}
        onMessageReadStateChanged={refreshUnreadCount}
        onRefreshAnnouncements={refreshAnnouncementData}
        fallbackNoticeContent={fallbackNoticeContent}
        t={t}
      />

      <div className='w-full px-4 md:px-6'>
        <div className='flex items-center justify-between h-14 gap-2'>
          <div className='flex items-center gap-2 md:gap-4 flex-1 min-w-0 md:flex-initial'>
            <MobileMenuButton
              isConsoleRoute={isConsoleRoute}
              isMobile={isMobile}
              drawerOpen={drawerOpen}
              collapsed={collapsed}
              onToggle={handleMobileMenuToggle}
              t={t}
            />

            <HeaderLogo
              isMobile={isMobile}
              isConsoleRoute={isConsoleRoute}
              logo={logo}
              logoLoaded={logoLoaded}
              isLoading={isLoading}
              systemName={systemName}
              isSelfUseMode={isSelfUseMode}
              isDemoSiteMode={isDemoSiteMode}
              userState={userState}
              t={t}
            />

            <div className='min-w-0 flex-1 overflow-hidden md:hidden'>
              <MobileSiteNavDropdown
                mainNavLinks={mainNavLinks}
                pricingRequireAuth={pricingRequireAuth}
                userState={userState}
                isLoading={isLoading}
                t={t}
              />
            </div>

            {/* {!isMobile && <SearchDropdown isMobile={isMobile} />} */}
          </div>

          <div className='flex flex-shrink-0 items-center gap-2 md:gap-6'>
            <Navigation
              mainNavLinks={mainNavLinks}
              isMobile={isMobile}
              isLoading={isLoading}
              userState={userState}
              pricingRequireAuth={pricingRequireAuth}
            />

            <SettingsQuickSearch userState={userState} />

            <ActionButtons
              isNewYear={isNewYear}
              unreadCount={messageUnreadCount + announcementUnreadCount}
              notificationBubble={notificationBubble}
              notificationBubbleVisible={notificationBubbleVisible}
              onNoticeOpen={handleNotificationCenterOpen}
              theme={theme}
              onThemeToggle={handleThemeToggle}
              currentLang={currentLang}
              onLanguageChange={handleLanguageChange}
              userState={userState}
              isLoading={isLoading}
              isMobile={isMobile}
              isSelfUseMode={isSelfUseMode}
              logout={logout}
              navigate={navigate}
              t={t}
            />
          </div>
        </div>
      </div>
    </header>
  );
};

export default HeaderBar;
