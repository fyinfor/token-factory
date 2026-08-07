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
import { Card, Spin } from '@douyinfe/semi-ui';

import GroupRatioSettings from '../../pages/Setting/Ratio/GroupRatioSettings';
import ModelRatioSettings from '../../pages/Setting/Ratio/ModelRatioSettings';
import ModelSettingsVisualEditor from '../../pages/Setting/Ratio/ModelSettingsVisualEditor';
import ModelRatioNotSetEditor from '../../pages/Setting/Ratio/ModelRationNotSetEditor';
import UpstreamRatioSync from '../../pages/Setting/Ratio/UpstreamRatioSync';
import UserModelPricingSettings from '../../pages/Setting/Ratio/UserModelPricingSettings';

import { API, showError, toBoolean } from '../../helpers';

const RatioSetting = ({ activeSection = 'visual' }) => {
  let [inputs, setInputs] = useState({
    ModelPrice: '',
    ModelRatio: '',
    CacheRatio: '',
    CreateCacheRatio: '',
    CompletionRatio: '',
    GroupRatio: '',
    GroupGroupRatio: '',
    GroupModelPrice: '',
    GroupModelRatio: '',
    ChannelModelPrice: '',
    ChannelModelRatio: '',
    ChannelCompletionRatio: '',
    ChannelCacheRatio: '',
    SupplierModelPrice: '',
    SupplierModelRatio: '',
    ImageRatio: '',
    AudioRatio: '',
    AudioCompletionRatio: '',
    VideoRatio: '',
    VideoCompletionRatio: '',
    VideoPrice: '',
    ModelRequestTierPricing: '',
    ChannelModelRequestTierPricing: '',
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
            // 如果后端返回的不是合法 JSON，直接展示
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
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <Spin spinning={loading} size='large'>
      <Card style={{ marginTop: '10px' }}>
        {activeSection === 'model' && (
          <ModelRatioSettings options={inputs} refresh={onRefresh} />
        )}
        {activeSection === 'group' && (
          <GroupRatioSettings options={inputs} refresh={onRefresh} />
        )}
        {activeSection === 'visual' && (
          <ModelSettingsVisualEditor options={inputs} refresh={onRefresh} />
        )}
        {activeSection === 'unset_models' && (
          <ModelRatioNotSetEditor options={inputs} refresh={onRefresh} />
        )}
        {activeSection === 'user_pricing' && <UserModelPricingSettings />}
        {activeSection === 'request_tier_templates' && (
          <RequestTierPricingTemplateSettings
            options={inputs}
            refresh={onRefresh}
          />
        )}
        {activeSection === 'upstream_sync' && (
          <UpstreamRatioSync options={inputs} refresh={onRefresh} />
        )}
      </Card>
    </Spin>
  );
};

export default RatioSetting;
