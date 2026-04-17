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
import { Card, Spin, Tabs } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';

import ModelSettingsVisualEditor from '../../Setting/Ratio/ModelSettingsVisualEditor';
import ModelRatioNotSetEditor from '../../Setting/Ratio/ModelRationNotSetEditor';
import UpstreamRatioSync from '../../Setting/Ratio/UpstreamRatioSync';

import { API, showError, toBoolean } from '../../../helpers';

const PricingSettingsPage = () => {
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
    AutoGroups: '',
    DefaultUseAutoGroup: false,
    ExposeRatioEnabled: false,
    UserUsableGroups: '',
    'group_ratio_setting.group_special_usable_group': '',
  });

  const [loading, setLoading] = useState(false);

  const getOptions = async () => {
    const res = await API.get('/api/option/');
    const { success, message, data } = res.data;
    if (success) {
      let newInputs = {};
      data.forEach((item) => {
        if (item.value.startsWith('{') || item.value.startsWith('[')) {
          try {
            item.value = JSON.stringify(JSON.parse(item.value), null, 2);
          } catch (e) {
          }
        }
        if (['DefaultUseAutoGroup', 'ExposeRatioEnabled'].includes(item.key)) {
          newInputs[item.key] = toBoolean(item.value);
        } else {
          newInputs[item.key] = item.value;
        }
      });
      setInputs(newInputs);
    } else {
      showError(message);
    }
  };

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

export default PricingSettingsPage;
