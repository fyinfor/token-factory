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

import React, { useMemo, useCallback } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { Button, Dropdown } from '@douyinfe/semi-ui';
import { ChevronDown } from 'lucide-react';
import SkeletonWrapper from '../components/SkeletonWrapper';
import {
  isAdmin,
  userIsDistributorUser,
  userIsSupplierUser,
} from '../../../helpers';

/** 主站入口顺序（与桌面顶栏一致） */
const PRIMARY_NAV_KEYS = ['home', 'pricing', 'docs', 'about'];

const menuClass =
  '!bg-semi-color-bg-overlay !border-semi-color-border !shadow-lg !rounded-lg dark:!bg-gray-700 dark:!border-gray-600';

const distributorApplyPath = '/console/distributor/apply';
const supplierApplyPath = '/console/supplier/apply';

const MobileSiteNavDropdown = ({
  mainNavLinks,
  pricingRequireAuth,
  userState,
  isLoading,
  t,
}) => {
  const location = useLocation();
  const navigate = useNavigate();
  const user = userState?.user;

  const byNavKey = useMemo(
    () => Object.fromEntries(mainNavLinks.map((l) => [l.itemKey, l])),
    [mainNavLinks],
  );

  const primaryAndConsoleLinks = useMemo(() => {
    const primary = PRIMARY_NAV_KEYS.map((k) => byNavKey[k]).filter(Boolean);
    const consoleLink = byNavKey.console;
    if (consoleLink) {
      return [...primary, consoleLink];
    }
    return primary;
  }, [byNavKey]);

  const applyEntries = useMemo(() => {
    const hideApplyLinks = isAdmin();
    const hideDistributorApplyAsLoggedInDistributor = Boolean(
      user && userIsDistributorUser(user),
    );
    const showDistributorApply =
      !hideApplyLinks && !hideDistributorApplyAsLoggedInDistributor;
    const showSupplierApply = !hideApplyLinks && !userIsSupplierUser(user);

    const entries = [];
    if (showDistributorApply) {
      entries.push({
        itemKey: 'distributor-apply',
        text: t('成为代理'),
        redirectPath: distributorApplyPath,
      });
    }
    if (showSupplierApply) {
      entries.push({
        itemKey: 'supplier-apply',
        text: t('提供算力'),
        redirectPath: supplierApplyPath,
      });
    }
    return entries;
  }, [user, t]);

  const currentPageLabel = useMemo(() => {
    const p = location.pathname;
    if (p === '/') return t('首页');
    if (p === '/pricing' || p.startsWith('/pricing/')) return t('模型广场');
    if (p === '/about' || p.startsWith('/about/')) return t('关于');
    if (/\/[a-z]{2}\/docs\b/i.test(p) || p.includes('/docs')) {
      return t('文档');
    }
    if (p.startsWith(distributorApplyPath)) return t('成为代理');
    if (p.startsWith(supplierApplyPath)) return t('提供算力');
    if (p.startsWith('/console')) return t('控制台');
    if (p === '/login') return t('登录');
    return t('页面导航');
  }, [location.pathname, t]);

  const isActivePath = useCallback(
    (linkPath) => {
      if (!linkPath) return false;
      const currentPath = location.pathname;
      if (linkPath === '/') {
        return currentPath === '/';
      }
      return currentPath.startsWith(linkPath);
    },
    [location.pathname],
  );

  const resolveInternalTarget = useCallback(
    (link) => {
      let targetPath = link.to;
      if (link.itemKey === 'console' && !user) {
        targetPath = '/login';
      }
      if (link.itemKey === 'pricing' && pricingRequireAuth && !user) {
        targetPath = '/login';
      }
      return targetPath;
    },
    [pricingRequireAuth, user],
  );

  const handleSelectMain = useCallback(
    (link) => {
      const run = () => {
        if (link.isExternal && link.externalLink) {
          const openInNewTab = link.openInNewTab !== false;
          if (openInNewTab) {
            window.open(link.externalLink, '_blank', 'noopener,noreferrer');
          } else {
            window.location.assign(link.externalLink);
          }
          return;
        }
        navigate(resolveInternalTarget(link));
      };
      window.setTimeout(run, 10);
    },
    [navigate, resolveInternalTarget],
  );

  const handleSelectApply = useCallback(
    (entry) => {
      window.setTimeout(() => {
        if (user) {
          navigate(entry.redirectPath);
          return;
        }
        navigate(`/login?redirect=${encodeURIComponent(entry.redirectPath)}`);
      }, 10);
    },
    [navigate, user],
  );

  if (primaryAndConsoleLinks.length === 0 && applyEntries.length === 0) {
    return null;
  }

  const itemClass = (active) =>
    `!px-3 !py-2 !text-sm !text-semi-color-text-0 dark:!text-gray-200 ${
      active
        ? '!bg-semi-color-primary-light-default dark:!bg-blue-600 !font-semibold'
        : 'hover:!bg-semi-color-fill-1 dark:hover:!bg-gray-600'
    }`;

  return (
    <div className='min-w-0 max-w-full'>
      <SkeletonWrapper
        loading={isLoading}
        type='button'
        width={140}
        height={32}
        isMobile
      >
        <Dropdown
          position='bottomLeft'
          trigger='click'
          zIndex={1200}
          getPopupContainer={() => document.body}
          render={
            <Dropdown.Menu className={menuClass}>
              {primaryAndConsoleLinks.map((link) => {
                const active = !link.isExternal && isActivePath(link.to);
                return (
                  <Dropdown.Item
                    key={link.itemKey}
                    onMouseDown={(e) => {
                      e.stopPropagation();
                    }}
                    onClick={() => handleSelectMain(link)}
                    className={itemClass(active)}
                  >
                    {link.text}
                  </Dropdown.Item>
                );
              })}
              {applyEntries.length > 0 && primaryAndConsoleLinks.length > 0 && (
                <Dropdown.Divider />
              )}
              {applyEntries.map((entry) => {
                const active = isActivePath(entry.redirectPath);
                return (
                  <Dropdown.Item
                    key={entry.itemKey}
                    onMouseDown={(e) => {
                      e.stopPropagation();
                    }}
                    onClick={() => handleSelectApply(entry)}
                    className={itemClass(active)}
                  >
                    {entry.text}
                  </Dropdown.Item>
                );
              })}
            </Dropdown.Menu>
          }
        >
          <Button
            theme='borderless'
            type='tertiary'
            disabled={isLoading}
            onMouseDown={(e) => {
              e.stopPropagation();
            }}
            className='!pl-1 !pr-1.5 !py-1.5 !h-auto !w-full !max-w-full !min-w-0 !justify-between !text-semi-color-text-0 dark:!text-gray-200 hover:!bg-semi-color-fill-1 dark:hover:!bg-gray-700 !rounded-lg !font-medium !text-sm'
            aria-haspopup='menu'
            aria-label={t('页面导航')}
          >
            <span className='min-w-0 flex-1 truncate text-left font-medium'>
              {currentPageLabel}
            </span>
            <ChevronDown
              size={14}
              strokeWidth={2.5}
              className='ml-0.5 flex-shrink-0 opacity-70'
            />
          </Button>
        </Dropdown>
      </SkeletonWrapper>
    </div>
  );
};

export default MobileSiteNavDropdown;
