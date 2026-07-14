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
import { Card, Switch, Typography } from '@douyinfe/semi-ui';
import { Activity } from 'lucide-react';
import { API, showError, showSuccess, toBoolean } from '../../../helpers';
import { useTranslation } from 'react-i18next';

const { Text, Title } = Typography;

const SettingsPerfOverview = ({ options, refresh }) => {
  const { t } = useTranslation();
  const [enabled, setEnabled] = useState(true);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const value = options['console_setting.perf_overview_enabled'];
    setEnabled(value === undefined ? true : toBoolean(value));
  }, [options]);

  const handleToggleEnabled = async (checked) => {
    setLoading(true);
    try {
      const res = await API.put('/api/option/', {
        key: 'console_setting.perf_overview_enabled',
        value: checked ? 'true' : 'false',
      });
      if (res.data.success) {
        setEnabled(checked);
        showSuccess(t('设置已保存'));
        refresh?.();
      } else {
        showError(res.data.message);
      }
    } catch (err) {
      showError(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card
      title={
        <div className='flex items-center justify-between gap-4 w-full'>
          <div className='flex items-center gap-2 min-w-0'>
            <Activity size={16} className='text-blue-500 shrink-0' />
            <div className='min-w-0'>
              <Title heading={6} style={{ margin: 0 }}>
                {t('模型运行性能概览')}
              </Title>
              <Text type='tertiary' size='small'>
                {t('控制性能概览页面及侧边栏入口是否显示')}
              </Text>
            </div>
          </div>
          <div className='flex items-center gap-2 shrink-0'>
            <Switch
              checked={enabled}
              loading={loading}
              onChange={handleToggleEnabled}
            />
            <Text>{enabled ? t('已启用') : t('已禁用')}</Text>
          </div>
        </div>
      }
      bodyStyle={{ display: 'none' }}
    />
  );
};

export default SettingsPerfOverview;
