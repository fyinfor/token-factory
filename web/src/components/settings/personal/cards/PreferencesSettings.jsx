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

import React, { useState, useEffect, useContext } from 'react';
import {
  Avatar,
  Button,
  Card,
  Input,
  Select,
  Typography,
} from '@douyinfe/semi-ui';
import { Languages, UserRound } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { API, showSuccess, showError } from '../../../../helpers';
import { UserContext } from '../../../../context/User';
import { normalizeLanguage } from '../../../../i18n/language';

// Language options with native names
const languageOptions = [
  { value: 'zh-CN', label: '简体中文' },
  { value: 'zh-TW', label: '繁體中文' },
  { value: 'en', label: 'English' },
  { value: 'fr', label: 'Français' },
  { value: 'ru', label: 'Русский' },
  { value: 'ja', label: '日本語' },
  { value: 'vi', label: 'Tiếng Việt' },
  { value: 'id', label: 'Bahasa Indonesia' },
  { value: 'ms', label: 'Bahasa Melayu' },
  { value: 'th', label: 'ไทย' },
  { value: 'sw', label: 'Kiswahili' },
];

const PreferencesSettings = ({ t }) => {
  const { i18n } = useTranslation();
  const [userState, userDispatch] = useContext(UserContext);
  const [currentLanguage, setCurrentLanguage] = useState(
    normalizeLanguage(i18n.language) || 'zh-CN',
  );
  const [languageLoading, setLanguageLoading] = useState(false);
  const [displayName, setDisplayName] = useState(
    userState?.user?.display_name || '',
  );
  const [displayNameSaving, setDisplayNameSaving] = useState(false);

  // Load saved language preference from user settings
  useEffect(() => {
    if (userState?.user?.setting) {
      try {
        const settings = JSON.parse(userState.user.setting);
        if (settings.language) {
          const lang = normalizeLanguage(settings.language);
          setCurrentLanguage(lang);
          // Sync i18n with saved preference
          if (i18n.language !== lang) {
            i18n.changeLanguage(lang);
          }
        }
      } catch (e) {
        // Ignore parse errors
      }
    }
  }, [userState?.user?.setting, i18n]);

  useEffect(() => {
    setDisplayName(userState?.user?.display_name || '');
  }, [userState?.user?.display_name]);

  const syncLocalUser = (patch) => {
    const nextUser = {
      ...(userState?.user || {}),
      ...patch,
    };
    userDispatch({
      type: 'login',
      payload: nextUser,
    });
    localStorage.setItem('user', JSON.stringify(nextUser));
    return nextUser;
  };

  const handleLanguagePreferenceChange = async (lang) => {
    if (lang === currentLanguage) return;

    setLanguageLoading(true);
    const previousLang = currentLanguage;

    try {
      // Update language immediately for responsive UX
      setCurrentLanguage(lang);
      i18n.changeLanguage(lang);
      localStorage.setItem('i18nextLng', lang);

      // Save to backend
      const res = await API.put('/api/user/self', {
        language: lang,
      });

      if (res.data.success) {
        showSuccess(t('语言偏好已保存'));
        // Keep backend preference, context state, and local cache aligned.
        let settings = {};
        if (userState?.user?.setting) {
          try {
            settings = JSON.parse(userState.user.setting) || {};
          } catch (e) {
            settings = {};
          }
        }
        settings.language = lang;
        syncLocalUser({ setting: JSON.stringify(settings) });
      } else {
        showError(res.data.message || t('保存失败'));
        // Revert on error
        setCurrentLanguage(previousLang);
        i18n.changeLanguage(previousLang);
        localStorage.setItem('i18nextLng', previousLang);
      }
    } catch (error) {
      showError(t('保存失败，请重试'));
      // Revert on error
      setCurrentLanguage(previousLang);
      i18n.changeLanguage(previousLang);
      localStorage.setItem('i18nextLng', previousLang);
    } finally {
      setLanguageLoading(false);
    }
  };

  const handleDisplayNameSave = async () => {
    const nextDisplayName = displayName.trim();
    const previousDisplayName = userState?.user?.display_name || '';
    if (nextDisplayName === previousDisplayName) return;
    if (!nextDisplayName) {
      showError(t('请输入显示名称'));
      return;
    }
    if (nextDisplayName.length > 20) {
      showError(t('显示名称不能超过 20 个字符'));
      return;
    }

    setDisplayNameSaving(true);
    try {
      const res = await API.put('/api/user/self', {
        display_name: nextDisplayName,
      });
      if (res.data.success) {
        setDisplayName(nextDisplayName);
        syncLocalUser({ display_name: nextDisplayName });
        showSuccess(t('显示名称已保存'));
      } else {
        showError(res.data.message || t('保存失败'));
      }
    } catch (error) {
      showError(t('保存失败，请重试'));
    } finally {
      setDisplayNameSaving(false);
    }
  };

  return (
    <Card className='!rounded-2xl shadow-sm border-0'>
      {/* Card Header */}
      <div className='flex items-center mb-4'>
        <Avatar size='small' color='violet' className='mr-3 shadow-md'>
          <Languages size={16} />
        </Avatar>
        <div>
          <Typography.Text className='text-lg font-medium'>
            {t('偏好设置')}
          </Typography.Text>
          <div className='text-xs text-gray-600 dark:text-gray-400'>
            {t('界面语言和其他个人偏好')}
          </div>
        </div>
      </div>
      {/* Display Name Setting Card */}
      <Card className='!rounded-xl border dark:border-gray-700'>
        <div className='flex flex-col sm:flex-row items-start sm:items-center sm:justify-between gap-4'>
          <div className='flex items-start w-full sm:w-auto'>
            <div className='w-12 h-12 rounded-full bg-blue-50 dark:bg-blue-900/30 flex items-center justify-center mr-4 flex-shrink-0'>
              <UserRound
                size={20}
                className='text-blue-600 dark:text-blue-400'
              />
            </div>
            <div>
              <Typography.Title heading={6} className='mb-1'>
                {t('显示名称')}
              </Typography.Title>
              <Typography.Text type='tertiary' className='text-sm'>
                {t('设置在界面中展示的个人名称，不影响登录用户名')}
              </Typography.Text>
            </div>
          </div>
          <div className='flex w-full sm:w-auto gap-2'>
            <Input
              value={displayName}
              onChange={setDisplayName}
              maxLength={20}
              showClear
              placeholder={t('请输入显示名称')}
              style={{ width: 220 }}
              disabled={displayNameSaving}
              onEnterPress={handleDisplayNameSave}
            />
            <Button
              type='primary'
              loading={displayNameSaving}
              disabled={
                displayName.trim() === (userState?.user?.display_name || '')
              }
              onClick={handleDisplayNameSave}
            >
              {t('保存')}
            </Button>
          </div>
        </div>
      </Card>
      <div className='h-4'></div>
      {/* Language Setting Card */}
      <Card className='!rounded-xl border dark:border-gray-700'>
        <div className='flex flex-col sm:flex-row items-start sm:items-center sm:justify-between gap-4'>
          <div className='flex items-start w-full sm:w-auto'>
            <div className='w-12 h-12 rounded-full bg-violet-50 dark:bg-violet-900/30 flex items-center justify-center mr-4 flex-shrink-0'>
              <Languages
                size={20}
                className='text-violet-600 dark:text-violet-400'
              />
            </div>
            <div>
              <Typography.Title heading={6} className='mb-1'>
                {t('语言偏好')}
              </Typography.Title>
              <Typography.Text type='tertiary' className='text-sm'>
                {t('选择您的首选界面语言，设置将自动保存并同步到所有设备')}
              </Typography.Text>
            </div>
          </div>
          <Select
            value={currentLanguage}
            onChange={handleLanguagePreferenceChange}
            style={{ width: 180 }}
            loading={languageLoading}
            optionList={languageOptions.map((opt) => ({
              value: opt.value,
              label: opt.label,
            }))}
          />
        </div>
      </Card>

      {/* Additional info */}
      <div className='mt-4 text-xs text-gray-500 dark:text-gray-400'>
        <Typography.Text type='tertiary'>
          {t(
            '提示：语言偏好会同步到您登录的所有设备，并影响API返回的错误消息语言。',
          )}
        </Typography.Text>
      </div>
    </Card>
  );
};

export default PreferencesSettings;
