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
import { Link } from 'react-router-dom';
import { Avatar, Button, Dropdown, Typography } from '@douyinfe/semi-ui';
import { ChevronDown } from 'lucide-react';
import {
  IconExit,
  IconUserSetting,
  IconCreditCard,
  IconKey,
} from '@douyinfe/semi-icons';
import { renderQuota, stringToColor } from '../../../helpers/display';
import SkeletonWrapper from '../components/SkeletonWrapper';

const UserArea = ({
  userState,
  isLoading,
  isMobile,
  isSelfUseMode,
  logout,
  navigate,
  t,
}) => {
  if (isLoading) {
    return (
      <SkeletonWrapper
        loading={true}
        type='userArea'
        width={50}
        isMobile={isMobile}
      />
    );
  }

  if (userState.user) {
    const user = userState.user;
    const displayName = user.display_name || user.username;
    const quota = user.quota ?? 0;
    const goTopUp = () => {
      navigate('/console/topup');
    };

    return (
      <div className='relative'>
        <Dropdown
          position='bottomRight'
          /* 挂到 body + 高层级，避免被模型页「排序」等 Semi 浮层（约 1030–1070）遮挡 */
          zIndex={2000}
          render={
            <Dropdown.Menu className='!bg-semi-color-bg-overlay !border-semi-color-border !shadow-lg !rounded-lg dark:!bg-gray-700 dark:!border-gray-600'>
              <div className='min-w-[230px]'>
                <div className='flex items-center justify-between gap-3 rounded-md px-2.5 py-2.5'>
                  <div className='min-w-0 flex-1'>
                    <button
                      type='button'
                      onClick={goTopUp}
                      className='block max-w-full text-left'
                    >
                      <span className='block text-xs leading-4 text-semi-color-text-2 dark:text-gray-400'>
                        {t('账户余额')}
                      </span>
                      <span className='block max-w-[128px] truncate text-base font-semibold leading-6 text-semi-color-primary'>
                        {renderQuota(quota)}
                      </span>
                    </button>
                  </div>
                  <Button
                    size='small'
                    type='primary'
                    theme='solid'
                    icon={<IconCreditCard size='small' />}
                    onClick={goTopUp}
                    className='!px-2.5'
                  >
                    {t('充值')}
                  </Button>
                </div>
              </div>
              <Dropdown.Item
                onClick={() => {
                  navigate('/console/personal');
                }}
                className='!px-3 !py-1.5 !text-sm !text-semi-color-text-0 hover:!bg-semi-color-fill-1 dark:!text-gray-200 dark:hover:!bg-blue-500 dark:hover:!text-white'
              >
                <div className='flex items-center gap-2'>
                  <IconUserSetting
                    size='small'
                    className='text-gray-500 dark:text-gray-400'
                  />
                  <span>{t('个人设置')}</span>
                </div>
              </Dropdown.Item>
              <Dropdown.Item
                onClick={() => {
                  navigate('/console/token');
                }}
                className='!px-3 !py-1.5 !text-sm !text-semi-color-text-0 hover:!bg-semi-color-fill-1 dark:!text-gray-200 dark:hover:!bg-blue-500 dark:hover:!text-white'
              >
                <div className='flex items-center gap-2'>
                  <IconKey
                    size='small'
                    className='text-gray-500 dark:text-gray-400'
                  />
                  <span>{t('令牌管理')}</span>
                </div>
              </Dropdown.Item>
              <Dropdown.Item
                onClick={() => {
                  navigate('/console/topup');
                }}
                className='!px-3 !py-1.5 !text-sm !text-semi-color-text-0 hover:!bg-semi-color-fill-1 dark:!text-gray-200 dark:hover:!bg-blue-500 dark:hover:!text-white'
              >
                <div className='flex items-center gap-2'>
                  <IconCreditCard
                    size='small'
                    className='text-gray-500 dark:text-gray-400'
                  />
                  <span>{t('钱包管理')}</span>
                </div>
              </Dropdown.Item>
              <Dropdown.Item
                onClick={logout}
                className='!px-3 !py-1.5 !text-sm !text-semi-color-text-0 hover:!bg-semi-color-fill-1 dark:!text-gray-200 dark:hover:!bg-red-500 dark:hover:!text-white'
              >
                <div className='flex items-center gap-2'>
                  <IconExit
                    size='small'
                    className='text-gray-500 dark:text-gray-400'
                  />
                  <span>{t('退出')}</span>
                </div>
              </Dropdown.Item>
            </Dropdown.Menu>
          }
        >
          <Button
            theme='borderless'
            type='tertiary'
            className='flex items-center gap-1.5 !px-1.5 !py-1 !rounded-full hover:!bg-semi-color-fill-1 dark:hover:!bg-gray-700 !bg-semi-color-fill-0 dark:!bg-semi-color-fill-1 dark:hover:!bg-semi-color-fill-2'
          >
            <Avatar
              size='extra-small'
              color={stringToColor(user.username)}
              className='mr-1'
            >
              {user.username?.[0]?.toUpperCase() || ''}
            </Avatar>
            <span className='hidden md:inline'>
              <span className='flex h-8 max-w-[7rem] flex-col items-start justify-center gap-[1px] leading-none mr-1'>
                <Typography.Text
                  ellipsis={{ showTooltip: true }}
                  className='!text-xs !leading-4 !font-medium !text-semi-color-text-1 dark:!text-gray-300'
                >
                  {displayName}
                </Typography.Text>
                <Typography.Text
                  ellipsis={{ showTooltip: true }}
                  className='!text-[11px] !leading-3 !font-semibold !text-semi-color-primary'
                >
                  {renderQuota(quota)}
                </Typography.Text>
              </span>
            </span>
            <ChevronDown
              size={14}
              className='text-xs text-semi-color-text-2 dark:text-gray-400'
            />
          </Button>
        </Dropdown>
      </div>
    );
  } else {
    const showRegisterButton = !isSelfUseMode;

    const commonSizingAndLayoutClass =
      'flex items-center justify-center !py-[10px] !px-1.5 !rounded-full';

    const loginButtonSpecificStyling =
      '!bg-semi-color-fill-0 dark:!bg-semi-color-fill-1 hover:!bg-semi-color-fill-1 dark:hover:!bg-gray-700 transition-colors';
    const loginButtonClasses = `${commonSizingAndLayoutClass} ${loginButtonSpecificStyling}`;

    const registerButtonClasses = `${commonSizingAndLayoutClass}`;

    const loginButtonTextSpanClass =
      '!text-xs !text-semi-color-text-1 dark:!text-gray-300 !p-1.5';
    const registerButtonTextSpanClass = '!text-xs !text-white !p-1.5';

    return (
      <div className='flex items-center gap-2'>
        <Link to='/login' className='flex'>
          <Button
            theme='borderless'
            type='tertiary'
            className={loginButtonClasses}
          >
            <span className={loginButtonTextSpanClass}>{t('登录')}</span>
          </Button>
        </Link>
        {showRegisterButton && (
          <div className='hidden md:block'>
            <Link to='/register' className='flex'>
              <Button
                theme='solid'
                type='primary'
                className={registerButtonClasses}
              >
                <span className={registerButtonTextSpanClass}>{t('注册')}</span>
              </Button>
            </Link>
          </div>
        )}
      </div>
    );
  }
};

export default UserArea;
