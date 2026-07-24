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

import React, { useState } from 'react';
import { Button } from '@douyinfe/semi-ui';
import { ScrollText } from 'lucide-react';
import NotificationButton from './NotificationButton';
import ThemeToggle from './ThemeToggle';
import LanguageSelector from './LanguageSelector';
import UserArea from './UserArea';
import ChangelogSideSheet from './ChangelogSideSheet';

const ActionButtons = ({
  isNewYear,
  unreadCount,
  notificationBubble,
  notificationBubbleVisible,
  onNoticeOpen,
  theme,
  onThemeToggle,
  currentLang,
  onLanguageChange,
  userState,
  isLoading,
  isMobile,
  isSelfUseMode,
  logout,
  navigate,
  t,
}) => {
  const shouldShowNoticeButton = Boolean(userState?.user?.id);
  const [changelogVisible, setChangelogVisible] = useState(false);

  return (
    <div className='flex items-center gap-2 md:gap-3'>
      {/* <NewYearButton isNewYear={isNewYear} /> */}
      <Button
        icon={<ScrollText size={18} />}
        aria-label={t('更新日志')}
        onClick={() => setChangelogVisible(true)}
        theme='borderless'
        type='tertiary'
        className='!p-1.5 !text-current focus:!bg-semi-color-fill-1 dark:focus:!bg-gray-700 !rounded-full !bg-semi-color-fill-0 dark:!bg-semi-color-fill-1 hover:!bg-semi-color-fill-1 dark:hover:!bg-semi-color-fill-2'
      />
      <ChangelogSideSheet
        visible={changelogVisible}
        onClose={() => setChangelogVisible(false)}
        isMobile={isMobile}
        t={t}
      />
      {shouldShowNoticeButton && (
        <NotificationButton
          unreadCount={unreadCount}
          bubble={notificationBubble}
          bubbleVisible={notificationBubbleVisible}
          onNoticeOpen={onNoticeOpen}
          t={t}
        />
      )}

      <ThemeToggle theme={theme} onThemeToggle={onThemeToggle} t={t} />

      <LanguageSelector
        currentLang={currentLang}
        onLanguageChange={onLanguageChange}
      />

      <UserArea
        userState={userState}
        isLoading={isLoading}
        isMobile={isMobile}
        isSelfUseMode={isSelfUseMode}
        logout={logout}
        navigate={navigate}
        t={t}
      />
    </div>
  );
};

export default ActionButtons;
