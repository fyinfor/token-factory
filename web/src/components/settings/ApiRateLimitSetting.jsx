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
import SettingsApiRateLimit from '../../pages/Setting/RateLimit/SettingsApiRateLimit';
import { API, showError, toBoolean } from '../../helpers';

const ApiRateLimitSetting = () => {
  const [inputs, setInputs] = useState({
    GlobalApiRateLimitEnable: true,
    GlobalApiRateLimitNum: 180,
    GlobalApiRateLimitDuration: 180,
    CriticalRateLimitEnable: true,
    CriticalRateLimitNum: 20,
    CriticalRateLimitDuration: 1200,
    RateLimitUserWhitelist: '[]',
  });
  const [loading, setLoading] = useState(false);

  const getOptions = async () => {
    const res = await API.get('/api/option/');
    const { success, message, data } = res.data;
    if (!success) {
      showError(message);
      return;
    }

    const nextInputs = {};
    data.forEach((item) => {
      if (!Object.prototype.hasOwnProperty.call(inputs, item.key)) {
        return;
      }
      if (item.key.endsWith('Enable')) {
        nextInputs[item.key] = toBoolean(item.value);
      } else if (item.key === 'RateLimitUserWhitelist') {
        try {
          nextInputs[item.key] = JSON.stringify(JSON.parse(item.value || '[]'), null, 2);
        } catch {
          nextInputs[item.key] = '[]';
        }
      } else {
        nextInputs[item.key] = Number(item.value);
      }
    });
    setInputs((prev) => ({ ...prev, ...nextInputs }));
  };

  async function onRefresh() {
    try {
      setLoading(true);
      await getOptions();
    } catch (error) {
      showError('刷新失败');
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    onRefresh();
  }, []);

  return (
    <Spin spinning={loading} size='large'>
      <Card style={{ marginTop: '10px' }}>
        <SettingsApiRateLimit options={inputs} refresh={onRefresh} />
      </Card>
    </Spin>
  );
};

export default ApiRateLimitSetting;
