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
import { Button, Card, Empty, Spin, Tabs } from '@douyinfe/semi-ui';
import {
  IllustrationNoAccess,
  IllustrationNoAccessDark,
} from '@douyinfe/semi-illustrations';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';

import ModelSettingsVisualEditor from '../../Setting/Ratio/ModelSettingsVisualEditor';
import ModelRatioNotSetEditor from '../../Setting/Ratio/ModelRationNotSetEditor';
import UpstreamRatioSync from '../../Setting/Ratio/UpstreamRatioSync';

import { API, isSupplier, showError, toBoolean } from '../../../helpers';

/**
 * SupplierPricingSettingsContent 供应商定价设置内容区。
 */
const SupplierPricingSettingsContent = () => {
  const { t } = useTranslation();

  let [inputs, setInputs] = useState({
    ModelPrice: '',
    ModelRatio: '',
    CacheRatio: '',
    CreateCacheRatio: '',
    CompletionRatio: '',
    GroupRatio: '',
    GroupGroupRatio: '',
    ImageRatio: '',
    AudioRatio: '',
    AudioCompletionRatio: '',
    VideoRatio: '',
    VideoCompletionRatio: '',
    VideoPrice: '',
    AutoGroups: '',
    DefaultUseAutoGroup: false,
    ExposeRatioEnabled: false,
    UserUsableGroups: '',
    'group_ratio_setting.group_special_usable_group': '',
  });

  const [loading, setLoading] = useState(false);

  /**
   * 获取供应商定价配置；若无供应商资质，给出明确引导提示。
   */
  const getOptions = async () => {
    try {
      const res = await API.get('/api/option/');
      const { success, message, data } = res.data;
      if (success) {
        let newInputs = {};
        data.forEach((item) => {
          if (item.value.startsWith('{') || item.value.startsWith('[')) {
            try {
              item.value = JSON.stringify(JSON.parse(item.value), null, 2);
            } catch (e) {}
          }
          if (
            ['DefaultUseAutoGroup', 'ExposeRatioEnabled'].includes(item.key)
          ) {
            newInputs[item.key] = toBoolean(item.value);
          } else {
            newInputs[item.key] = item.value;
          }
        });
        setInputs(newInputs);
      } else {
        showError(message);
      }
    } catch (error) {
      if (error?.response?.status === 403) {
        showError(t('请先申请供应商资质'));
        return;
      }
      showError(error?.response?.data?.message || t('获取配置失败'));
    }
  };

  /**
   * 刷新当前页数据。
   */
  const onRefresh = async () => {
    try {
      setLoading(true);
      await getOptions();
    } catch (error) {
      showError('刷新失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    onRefresh();
  }, []);

  return (
    <div className='mt-[60px] px-2'>
      <Spin spinning={loading} size='large'>
        <Card style={{ marginTop: '10px' }}>
          <Tabs type='card' defaultActiveKey='visual'>
            <Tabs.TabPane tab={t('价格设置')} itemKey='visual'>
              <ModelSettingsVisualEditor options={inputs} refresh={onRefresh} />
            </Tabs.TabPane>
            <Tabs.TabPane tab={t('未设置价格模型')} itemKey='unset_models'>
              <ModelRatioNotSetEditor options={inputs} refresh={onRefresh} />
            </Tabs.TabPane>
            <Tabs.TabPane tab={t('上游倍率同步')} itemKey='upstream_sync'>
              <UpstreamRatioSync options={inputs} refresh={onRefresh} />
            </Tabs.TabPane>
          </Tabs>
        </Card>
      </Spin>
    </div>
  );
};

/**
 * PricingSettingsPage 供应商定价设置页；非供应商时展示权限引导页。
 */
const PricingSettingsPage = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();

  if (!isSupplier()) {
    return (
      <div className='mt-[60px] px-2'>
        <div
          className='flex items-center justify-center'
          style={{ minHeight: 'calc(100vh - 360px)' }}
        >
          <Empty
            image={<IllustrationNoAccess style={{ width: 200, height: 200 }} />}
            darkModeImage={
              <IllustrationNoAccessDark style={{ width: 200, height: 200 }} />
            }
            layout='horizontal'
            title={t('需要供应商权限')}
            description={t('您需要先成为供应商才能访问此页面。')}
          >
            <Button
              theme='solid'
              type='primary'
              size='large'
              className='!rounded-md mt-4'
              style={{ fontWeight: 500 }}
              onClick={() => navigate('/console/supplier/apply')}
            >
              {t('前往申请')}
            </Button>
          </Empty>
        </div>
      </div>
    );
  }

  return <SupplierPricingSettingsContent />;
};

export default PricingSettingsPage;
