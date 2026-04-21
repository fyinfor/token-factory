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

import React, { useCallback, useState } from 'react';
import { useHeaderBar } from '../../../hooks/common/useHeaderBar';
import { useNavigation } from '../../../hooks/common/useNavigation';
import { useUserMessageUnreadCount } from '../../../hooks/common/useUserMessageUnreadCount';
import UserMessageModal from './UserMessageModal';
import MobileMenuButton from './MobileMenuButton';
import HeaderLogo from './HeaderLogo';
import Navigation from './Navigation';
import ActionButtons from './ActionButtons';

const HeaderBar = ({ onMobileMenuToggle, drawerOpen }) => {
  const {
    userState,
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

  const [messageModalVisible, setMessageModalVisible] = useState(false);
  const { unreadCount: messageUnreadCount, refreshUnreadCount } =
    useUserMessageUnreadCount(userState?.user);
  // handleMessageModalOpen 打开站内消息弹窗。
  const handleMessageModalOpen = useCallback(() => {
    setMessageModalVisible(true);
  }, []);
  // handleMessageModalClose 关闭站内消息弹窗并刷新未读计数。
  const handleMessageModalClose = useCallback(async () => {
    setMessageModalVisible(false);
    await refreshUnreadCount();
  }, [refreshUnreadCount]);

  const { mainNavLinks } = useNavigation(t, docsNav, headerNavModules);

  return (
    <header className='text-semi-color-text-0 sticky top-0 z-50 transition-colors duration-300 bg-white/75 dark:bg-zinc-900/75 backdrop-blur-lg border-b border-[#f5f5f5]'>
      <UserMessageModal
        visible={messageModalVisible}
        onClose={handleMessageModalClose}
        onReadStateChanged={refreshUnreadCount}
        isMobile={isMobile}
        t={t}
      />

      <div className='w-full px-4 md:px-6'>
        <div className='flex items-center justify-between h-14'>
          <div className='flex items-center gap-3 md:gap-4 flex-shrink-0'>
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

            {/* {!isMobile && <SearchDropdown isMobile={isMobile} />} */}
          </div>

          <div className='flex items-center gap-4 md:gap-6'>
            <Navigation
              mainNavLinks={mainNavLinks}
              isMobile={isMobile}
              isLoading={isLoading}
              userState={userState}
              pricingRequireAuth={pricingRequireAuth}
            />

            <ActionButtons
              isNewYear={isNewYear}
              unreadCount={messageUnreadCount}
              onNoticeOpen={handleMessageModalOpen}
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
