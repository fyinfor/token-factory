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

import React, { useCallback } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import { Typography, Tag } from '@douyinfe/semi-ui';
import SkeletonWrapper from '../components/SkeletonWrapper';
import {
  userIsDistributorUser,
  userIsSupplierUser,
  showInfo,
  isAdmin,
} from '../../../helpers';

/**
 * 顶栏申请入口：浅色用 #409EFF 系。
 * 注意：tailwind.config 覆盖了默认 colors，无 blue/zinc 等键，暗色必须用任意值或 semi-color-*，否则 dark: 不会产出 CSS。
 */
const APPLY_BTN_BASE =
  'flex-shrink-0 inline-flex items-center justify-center text-sm font-semibold transition-all duration-200 ease-in-out rounded-lg px-3.5 py-2 min-h-[2.25rem] border shadow-sm dark:shadow-none';
const APPLY_BTN_IDLE =
  'border-[#b3d8ff] bg-[#ecf5ff] text-[#409EFF] hover:bg-[#d9ecff] hover:border-[#409EFF] active:bg-[#c6e2ff] dark:border-[rgba(96,165,250,0.35)] dark:bg-[rgba(59,130,246,0.1)] dark:text-[rgba(147,197,253,0.92)] dark:hover:bg-[rgba(59,130,246,0.15)] dark:hover:border-[rgba(96,165,250,0.48)] dark:active:bg-[rgba(59,130,246,0.12)]';
const APPLY_BTN_ACTIVE =
  'border-[#409EFF] bg-[#d9ecff] text-[#337ecc] shadow-md dark:border-[rgba(96,165,250,0.55)] dark:bg-[rgba(59,130,246,0.18)] dark:text-[rgba(191,219,254,0.95)] dark:shadow-none dark:ring-1 dark:ring-[rgba(96,165,250,0.28)]';

const HeaderLogo = ({
  isMobile,
  isConsoleRoute,
  logo,
  logoLoaded,
  isLoading,
  systemName,
  isSelfUseMode,
  isDemoSiteMode,
  userState,
  t,
}) => {
  const location = useLocation();
  const navigate = useNavigate();
  const user = userState?.user;

  /** 未登录且已在登录页时：提示先登录，并同步 redirect 到目标申请页 */
  const handleApplyEntryClick = useCallback(
    (applyPath) => (e) => {
      if (user) return;
      if (location.pathname !== '/login') return;
      e.preventDefault();
      showInfo(t('请先登录'));
      navigate(`/login?redirect=${encodeURIComponent(applyPath)}`, {
        replace: true,
      });
    },
    [user, location.pathname, navigate, t],
  );

  if (isMobile && isConsoleRoute) {
    return null;
  }

  const distributorApplyPath = '/console/distributor/apply';
  const supplierApplyPath = '/console/supplier/apply';

  /** 管理员不展示；已登录代理仅隐藏「成为代理」（「提供算力」仍看是否已有供应商身份） */
  const hideApplyLinks = isAdmin();
  const hideDistributorApplyAsLoggedInDistributor = Boolean(
    user && userIsDistributorUser(user),
  );

  const showDistributorApply =
    !hideApplyLinks && !hideDistributorApplyAsLoggedInDistributor;
  const showSupplierApply = !hideApplyLinks && !userIsSupplierUser(user);
  const showApplyLinks = showDistributorApply || showSupplierApply;

  const distributorTo = user
    ? distributorApplyPath
    : `/login?redirect=${encodeURIComponent(distributorApplyPath)}`;
  const supplierTo = user
    ? supplierApplyPath
    : `/login?redirect=${encodeURIComponent(supplierApplyPath)}`;

  const distributorActive = location.pathname.startsWith(distributorApplyPath);
  const supplierActive = location.pathname.startsWith(supplierApplyPath);

  return (
    <div className='flex items-center gap-2 md:gap-3 flex-shrink-0'>
      <Link to='/' className='group flex items-center gap-2'>
        <div className='relative w-8 h-8 md:w-8 md:h-8'>
          <SkeletonWrapper loading={isLoading || !logoLoaded} type='image' />
          <img
            src={logo}
            alt='logo'
            className={`absolute inset-0 w-full h-full transition-all duration-200 group-hover:scale-110 rounded-full ${!isLoading && logoLoaded ? 'opacity-100' : 'opacity-0'}`}
          />
        </div>
        <div className='hidden md:flex items-center gap-2'>
          <div className='flex items-center gap-2'>
            <SkeletonWrapper
              loading={isLoading}
              type='title'
              width={120}
              height={24}
            >
              <Typography.Title
                heading={4}
                className='!text-lg !font-semibold !mb-0'
              >
                {systemName}
              </Typography.Title>
            </SkeletonWrapper>
            {(isSelfUseMode || isDemoSiteMode) && !isLoading && (
              <Tag
                color={isSelfUseMode ? 'purple' : 'blue'}
                className='text-xs px-1.5 py-0.5 rounded whitespace-nowrap shadow-sm'
                size='small'
                shape='circle'
              >
                {isSelfUseMode ? t('自用模式') : t('演示站点')}
              </Tag>
            )}
          </div>
        </div>
      </Link>
      {showApplyLinks && (
        <div className='hidden sm:flex items-center gap-2 flex-shrink-0'>
          {showDistributorApply && (
            <Link
              to={distributorTo}
              onClick={handleApplyEntryClick(distributorApplyPath)}
              className={`${APPLY_BTN_BASE} ${distributorActive ? APPLY_BTN_ACTIVE : APPLY_BTN_IDLE}`}
            >
              {t('成为代理')}
            </Link>
          )}
          {showSupplierApply && (
            <Link
              to={supplierTo}
              onClick={handleApplyEntryClick(supplierApplyPath)}
              className={`${APPLY_BTN_BASE} ${supplierActive ? APPLY_BTN_ACTIVE : APPLY_BTN_IDLE}`}
            >
              {t('提供算力')}
            </Link>
          )}
        </div>
      )}
    </div>
  );
};

export default HeaderLogo;
