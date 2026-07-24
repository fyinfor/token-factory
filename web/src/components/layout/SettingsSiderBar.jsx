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

import React, { useEffect, useState } from 'react';
import { Button, Divider, Nav } from '@douyinfe/semi-ui';
import {
  ArrowLeft,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Link, useLocation } from 'react-router-dom';

import {
  SETTING_CATEGORIES,
  getSettingSelection,
  getSettingUrl,
} from '../../constants/setting.constants';
import { useSidebarCollapsed } from '../../hooks/common/useSidebarCollapsed';

const SettingsSiderBar = ({ onNavigate = () => {} }) => {
  const { t } = useTranslation();
  const location = useLocation();
  const [collapsed, toggleCollapsed] = useSidebarCollapsed();
  const { category, page } = getSettingSelection(location.search);
  const [openedKeys, setOpenedKeys] = useState([category.key]);
  const selectedKey = `setting:${category.key}:${page.key}`;

  useEffect(() => {
    setOpenedKeys((current) =>
      current.includes(category.key) ? current : [...current, category.key],
    );
  }, [category.key]);

  useEffect(() => {
    document.body.classList.add('settings-sidebar-active');
    return () => document.body.classList.remove('settings-sidebar-active');
  }, []);

  useEffect(() => {
    if (collapsed) {
      document.body.classList.add('sidebar-collapsed');
    } else {
      document.body.classList.remove('sidebar-collapsed');
    }
  }, [collapsed]);

  return (
    <div
      className='sidebar-container'
      style={{ width: 'var(--sidebar-current-width)' }}
    >
      <Nav
        className='sidebar-nav settings-sidebar-nav'
        defaultIsCollapsed={collapsed}
        isCollapsed={collapsed}
        onCollapseChange={toggleCollapsed}
        selectedKeys={[selectedKey]}
        openKeys={openedKeys}
        onOpenChange={(data) => setOpenedKeys(data.openKeys)}
        itemStyle='sidebar-nav-item'
        hoverStyle='sidebar-nav-item:hover'
        selectedStyle='sidebar-nav-item-selected'
        renderWrapper={({ itemElement, props }) => {
          if (props.itemKey === 'back-console') {
            return (
              <Link to='/console' onClick={onNavigate} className='no-underline'>
                {itemElement}
              </Link>
            );
          }

          if (!props.itemKey.startsWith('setting:')) {
            return itemElement;
          }

          const [, categoryKey, pageKey] = props.itemKey.split(':');
          return (
            <Link
              to={getSettingUrl(categoryKey, pageKey)}
              onClick={onNavigate}
              className='no-underline'
            >
              {itemElement}
            </Link>
          );
        }}
      >
        <Nav.Item
          itemKey='back-console'
          text={
            <span className='truncate font-medium text-sm'>
              {t('返回')} {t('控制台')}
            </span>
          }
          icon={
            <div className='sidebar-icon-container flex-shrink-0'>
              <ArrowLeft size={16} strokeWidth={2} />
            </div>
          }
        />

        <Divider className='sidebar-divider' />

        {SETTING_CATEGORIES.map((settingCategory) => {
          const CategoryIcon = settingCategory.icon;
          const categorySelected = settingCategory.key === category.key;

          return (
            <Nav.Sub
              key={settingCategory.key}
              itemKey={settingCategory.key}
              expandIcon={
                <span className='settings-sidebar-expand-icon'>
                  {openedKeys.includes(settingCategory.key) ? (
                    <ChevronDown size={16} />
                  ) : (
                    <ChevronRight size={16} />
                  )}
                </span>
              }
              text={
                <span
                  className='truncate font-medium text-sm'
                  style={{
                    color: categorySelected
                      ? 'var(--semi-color-primary)'
                      : 'inherit',
                  }}
                >
                  {t(settingCategory.label)}
                </span>
              }
              icon={
                <div className='sidebar-icon-container flex-shrink-0'>
                  <CategoryIcon
                    size={16}
                    strokeWidth={2}
                    color={
                      categorySelected ? 'var(--semi-color-primary)' : undefined
                    }
                  />
                </div>
              }
            >
              {settingCategory.pages.map((settingPage) => (
                <Nav.Item
                  key={settingPage.key}
                  itemKey={`setting:${settingCategory.key}:${settingPage.key}`}
                  text={
                    <span className='truncate text-sm'>
                      {t(settingPage.label)}
                    </span>
                  }
                />
              ))}
            </Nav.Sub>
          );
        })}
      </Nav>

      <div className='sidebar-collapse-button'>
        <Button
          theme='outline'
          type='tertiary'
          size='small'
          icon={
            <ChevronLeft
              size={16}
              strokeWidth={2.5}
              color='var(--semi-color-text-2)'
              style={{
                transform: collapsed ? 'rotate(180deg)' : 'rotate(0deg)',
              }}
            />
          }
          onClick={toggleCollapsed}
          icononly={collapsed ? true : undefined}
          style={
            collapsed
              ? { width: 36, height: 24, padding: 0 }
              : { padding: '4px 12px', width: '100%' }
          }
        >
          {!collapsed ? t('收起侧边栏') : null}
        </Button>
      </div>
    </div>
  );
};

export default SettingsSiderBar;
