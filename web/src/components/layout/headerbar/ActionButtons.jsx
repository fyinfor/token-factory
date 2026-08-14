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

import React, { lazy, Suspense, useState } from 'react';
import { Button, Dropdown } from '@douyinfe/semi-ui';
import {
  Bell,
  Check,
  ChevronLeft,
  ChevronRight,
  Languages,
  Monitor,
  Moon,
  MoreHorizontal,
  ScrollText,
  Search,
  Sun,
} from 'lucide-react';
import NotificationButton from './NotificationButton';
import ThemeToggle from './ThemeToggle';
import LanguageSelector from './LanguageSelector';
import SettingsQuickSearch from './SettingsQuickSearch';
import UserArea from './UserArea';
import {
  LANGUAGE_NATIVE_LABELS,
  normalizeLanguage,
  supportedLanguages,
} from '../../../i18n/language';
import { USER_ROLES } from '../../../constants/user.constants';

const ChangelogSideSheet = lazy(() => import('./ChangelogSideSheet'));

const iconButtonClass =
  '!h-8 !w-8 !rounded-full !bg-semi-color-fill-0 !p-0 !text-current hover:!bg-semi-color-fill-1 focus-visible:!ring-2 focus-visible:!ring-semi-color-primary dark:!bg-semi-color-fill-1 dark:hover:!bg-semi-color-fill-2';

const compactMenuItemClass =
  '!justify-start !px-3 !py-2 !text-left !text-sm !text-semi-color-text-0 hover:!bg-semi-color-fill-1 dark:!text-gray-200 dark:hover:!bg-gray-600';

const themeOptions = [
  { key: 'light', icon: Sun, label: '浅色模式' },
  { key: 'dark', icon: Moon, label: '深色模式' },
  { key: 'auto', icon: Monitor, label: '自动模式' },
];

const ChangelogButton = ({ onOpen, t }) => (
  <Button
    icon={<ScrollText size={18} />}
    aria-label={t('更新日志')}
    title={t('更新日志')}
    onClick={onOpen}
    theme='borderless'
    type='tertiary'
    className={iconButtonClass}
  />
);

const CompactActionMenu = ({
  unreadCount,
  onNoticeOpen,
  onSearchOpen,
  isRootUser,
  changelogEnabled,
  onChangelogOpen,
  theme,
  onThemeToggle,
  currentLang,
  onLanguageChange,
  t,
}) => {
  const [languageView, setLanguageView] = useState(false);
  const [menuVisible, setMenuVisible] = useState(false);
  const normalizedLanguage = normalizeLanguage(currentLang) || 'zh-CN';
  const currentLanguageLabel =
    LANGUAGE_NATIVE_LABELS[normalizedLanguage] || normalizedLanguage;

  const closeMenuAction = (action) => {
    action();
    setLanguageView(false);
    setMenuVisible(false);
  };

  return (
    <Dropdown
      position='bottomRight'
      trigger='click'
      visible={menuVisible}
      clickToHide={false}
      zIndex={2000}
      onVisibleChange={(visible) => {
        setMenuVisible(visible);
        if (!visible) setLanguageView(false);
      }}
      render={
        <Dropdown.Menu className='min-w-[220px] !rounded-lg !border-semi-color-border !bg-semi-color-bg-overlay !shadow-lg dark:!border-gray-600 dark:!bg-gray-700'>
          {languageView ? (
            <>
              <Dropdown.Item
                icon={<ChevronLeft size={17} />}
                onClick={() => setLanguageView(false)}
                className={compactMenuItemClass}
              >
                {t('返回')}
              </Dropdown.Item>
              <Dropdown.Divider />
              {supportedLanguages.map((languageCode) => {
                const selected = normalizedLanguage === languageCode;
                return (
                  <Dropdown.Item
                    key={languageCode}
                    onClick={() =>
                      closeMenuAction(() => onLanguageChange(languageCode))
                    }
                    className={`${compactMenuItemClass} ${
                      selected
                        ? '!bg-semi-color-primary-light-default !font-semibold'
                        : ''
                    }`}
                  >
                    <span className='flex min-w-[156px] items-center justify-between gap-3'>
                      <span>{LANGUAGE_NATIVE_LABELS[languageCode]}</span>
                      {selected ? (
                        <Check
                          size={15}
                          className='text-semi-color-primary'
                          aria-hidden='true'
                        />
                      ) : null}
                    </span>
                  </Dropdown.Item>
                );
              })}
            </>
          ) : (
            <>
              <Dropdown.Item
                icon={<Bell size={17} />}
                onClick={() => closeMenuAction(onNoticeOpen)}
                className={`${compactMenuItemClass} xl:!hidden`}
              >
                <span className='flex min-w-[156px] items-center justify-between gap-3'>
                  <span>{t('通知')}</span>
                  {unreadCount > 0 ? (
                    <span className='rounded-full bg-semi-color-danger px-1.5 py-0.5 text-[10px] font-semibold leading-none text-white'>
                      {unreadCount > 99 ? '99+' : unreadCount}
                    </span>
                  ) : null}
                </span>
              </Dropdown.Item>

              <Dropdown.Item
                icon={<Search size={17} />}
                onClick={() => closeMenuAction(onSearchOpen)}
                className={`${compactMenuItemClass} ${
                  isRootUser ? 'xl:!hidden' : ''
                }`}
              >
                {t('搜索功能')}
              </Dropdown.Item>

              <Dropdown.Item
                icon={<Languages size={17} />}
                onClick={() => setLanguageView(true)}
                className={`${compactMenuItemClass} xl:!hidden`}
              >
                <span className='flex min-w-[156px] items-center justify-between gap-3'>
                  <span>{t('语言')}</span>
                  <span className='flex min-w-0 items-center gap-1 text-xs text-semi-color-text-2 dark:text-gray-400'>
                    <span className='max-w-[92px] truncate'>
                      {currentLanguageLabel}
                    </span>
                    <ChevronRight size={14} aria-hidden='true' />
                  </span>
                </span>
              </Dropdown.Item>

              <Dropdown.Divider
                className={isRootUser ? 'xl:!hidden' : undefined}
              />
              <div className='px-3 pb-1 pt-1.5 text-xs font-medium text-semi-color-text-2 dark:text-gray-400'>
                {t('选择主题')}
              </div>
              {themeOptions.map((option) => {
                const Icon = option.icon;
                const selected = theme === option.key;

                return (
                  <Dropdown.Item
                    key={option.key}
                    icon={<Icon size={17} />}
                    onClick={() =>
                      closeMenuAction(() => onThemeToggle(option.key))
                    }
                    className={`${compactMenuItemClass} ${
                      selected
                        ? '!bg-semi-color-primary-light-default !font-semibold'
                        : ''
                    }`}
                  >
                    <span className='flex min-w-[156px] items-center justify-between gap-3'>
                      <span>{t(option.label)}</span>
                      {selected ? (
                        <Check
                          size={15}
                          className='text-semi-color-primary'
                          aria-hidden='true'
                        />
                      ) : null}
                    </span>
                  </Dropdown.Item>
                );
              })}

              {changelogEnabled ? (
                <>
                  <Dropdown.Divider />
                  <Dropdown.Item
                    icon={<ScrollText size={17} />}
                    onClick={() => closeMenuAction(onChangelogOpen)}
                    className={compactMenuItemClass}
                  >
                    {t('更新日志')}
                  </Dropdown.Item>
                </>
              ) : null}
            </>
          )}
        </Dropdown.Menu>
      }
    >
      <Button
        icon={<MoreHorizontal size={19} />}
        aria-label={t('更多')}
        title={t('更多')}
        theme='borderless'
        type='tertiary'
        className={iconButtonClass}
      />
    </Dropdown>
  );
};

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
  changelogEnabled,
  logout,
  navigate,
  t,
}) => {
  const [changelogVisible, setChangelogVisible] = useState(false);
  const [quickSearchVisible, setQuickSearchVisible] = useState(false);
  const isRootUser = Number(userState?.user?.role) >= USER_ROLES.ROOT;
  const openChangelog = () => setChangelogVisible(true);
  const openQuickSearch = () => {
    if (isRootUser) {
      setQuickSearchVisible(true);
      return;
    }
    navigate('/pricing');
  };

  return (
    <>
      <div className='flex items-center gap-1.5 sm:gap-2 xl:gap-3'>
        {/* <NewYearButton isNewYear={isNewYear} /> */}
        <SettingsQuickSearch
          userState={userState}
          visible={quickSearchVisible}
          onVisibleChange={setQuickSearchVisible}
          triggerClassName='hidden xl:block'
        />

        {changelogEnabled ? (
          <div className='hidden min-[1600px]:block'>
            <ChangelogButton onOpen={openChangelog} t={t} />
          </div>
        ) : null}

        <div className='hidden xl:block'>
          <NotificationButton
            unreadCount={unreadCount}
            bubble={notificationBubble}
            bubbleVisible={notificationBubbleVisible}
            onNoticeOpen={onNoticeOpen}
            t={t}
          />
        </div>

        <div className='hidden min-[1600px]:block'>
          <ThemeToggle theme={theme} onThemeToggle={onThemeToggle} t={t} />
        </div>

        <div className='hidden xl:block'>
          <LanguageSelector
            currentLang={currentLang}
            onLanguageChange={onLanguageChange}
          />
        </div>

        <div className='min-[1600px]:hidden'>
          <CompactActionMenu
            unreadCount={unreadCount}
            onNoticeOpen={onNoticeOpen}
            onSearchOpen={openQuickSearch}
            isRootUser={isRootUser}
            changelogEnabled={changelogEnabled}
            onChangelogOpen={openChangelog}
            theme={theme}
            onThemeToggle={onThemeToggle}
            currentLang={currentLang}
            onLanguageChange={onLanguageChange}
            t={t}
          />
        </div>

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

      {changelogVisible ? (
        <Suspense fallback={null}>
          <ChangelogSideSheet
            visible
            onClose={() => setChangelogVisible(false)}
            isMobile={isMobile}
            t={t}
          />
        </Suspense>
      ) : null}
    </>
  );
};

export default ActionButtons;
