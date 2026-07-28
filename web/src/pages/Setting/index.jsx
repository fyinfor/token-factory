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

import React, { useEffect, useMemo, useRef } from 'react';
import { Tabs } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { useLocation, useNavigate } from 'react-router-dom';

import ApiRateLimitSetting from '../../components/settings/ApiRateLimitSetting';
import ChatsSetting from '../../components/settings/ChatsSetting';
import DashboardSetting from '../../components/settings/DashboardSetting';
import DrawingSetting from '../../components/settings/DrawingSetting';
import ModelDeploymentSetting from '../../components/settings/ModelDeploymentSetting';
import ModelSetting from '../../components/settings/ModelSetting';
import OperationSetting from '../../components/settings/OperationSetting';
import OtherSetting from '../../components/settings/OtherSetting';
import PaymentSetting from '../../components/settings/PaymentSetting';
import PerformanceSetting from '../../components/settings/PerformanceSetting';
import RateLimitSetting from '../../components/settings/RateLimitSetting';
import RatioSetting from '../../components/settings/RatioSetting';
import SystemSetting from '../../components/settings/SystemSetting';
import {
  getSettingSelection,
  getSettingUrl,
} from '../../constants/setting.constants';

const SETTING_COMPONENTS = {
  operation: OperationSetting,
  dashboard: DashboardSetting,
  chats: ChatsSetting,
  drawing: DrawingSetting,
  payment: PaymentSetting,
  ratio: RatioSetting,
  ratelimit: RateLimitSetting,
  models: ModelSetting,
  'model-deployment': ModelDeploymentSetting,
  performance: PerformanceSetting,
  'api-rate-limit': ApiRateLimitSetting,
  system: SystemSetting,
  other: OtherSetting,
};

const scrollSettingContentToTop = (element) => {
  if (!element) return;

  const behavior = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    ? 'auto'
    : 'smooth';
  let scrollParent = element.parentElement;
  while (scrollParent) {
    const { overflowY } = window.getComputedStyle(scrollParent);
    if (
      /(auto|scroll)/.test(overflowY) &&
      scrollParent.scrollHeight > scrollParent.clientHeight
    ) {
      scrollParent.scrollTo({ top: 0, behavior });
      return;
    }
    scrollParent = scrollParent.parentElement;
  }

  window.scrollTo({ top: 0, behavior });
};

const Setting = () => {
  const { t } = useTranslation();
  const location = useLocation();
  const navigate = useNavigate();
  const selection = useMemo(
    () => getSettingSelection(location.search),
    [location.search],
  );
  const contentRef = useRef(null);
  const ActiveSetting = SETTING_COMPONENTS[selection.item.group];
  const requestedKey = `${selection.category.key}:${selection.page.key}:${selection.item.key}`;

  useEffect(() => {
    const canonicalSearch = `?category=${selection.category.key}&page=${selection.page.key}&item=${selection.item.key}`;
    if (location.search !== canonicalSearch) {
      navigate(
        { pathname: location.pathname, search: canonicalSearch },
        { replace: true },
      );
    }
  }, [location.pathname, location.search, navigate, requestedKey, selection]);

  useEffect(() => {
    scrollSettingContentToTop(contentRef.current);
  }, [requestedKey]);

  if (!ActiveSetting) {
    return null;
  }

  return (
    <div ref={contentRef} className='settings-page mt-[60px] px-2'>
      <div className='settings-page-header'>
        <div className='settings-page-heading'>
          <span className='settings-page-category'>
            {t(selection.category.label)}
          </span>
          <h1>{t(selection.page.label)}</h1>
        </div>
        <Tabs
          className='settings-page-tabs'
          type='button'
          size='small'
          collapsible
          activeKey={selection.item.key}
          tabList={selection.page.items.map((settingItem) => ({
            itemKey: settingItem.key,
            tab: t(settingItem.label),
          }))}
          onChange={(itemKey) =>
            navigate(
              getSettingUrl(
                selection.category.key,
                selection.page.key,
                itemKey,
              ),
            )
          }
        />
      </div>
      <div key={requestedKey} className='settings-page-transition is-entering'>
        <ActiveSetting activeSection={selection.item.section} />
      </div>
    </div>
  );
};

export default Setting;
