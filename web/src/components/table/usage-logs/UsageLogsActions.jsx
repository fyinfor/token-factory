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

import React, { useState } from 'react';
import { Tag, Space, Skeleton, Tooltip, Button } from '@douyinfe/semi-ui';
import { IconDownload } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { renderLogStatDisplayQuota } from '../../../helpers';
import CompactModeToggle from '../../common/ui/CompactModeToggle';
import { useMinimumLoadingTime } from '../../../hooks/common/useMinimumLoadingTime';
import { API, showError, showSuccess } from '../../../helpers';
import { getTodayStartTimestamp } from '../../../helpers/utils';
import { normalizeLanguage } from '../../../i18n/language';

const LogsActions = ({
  stat,
  loadingStat,
  showStat,
  compactMode,
  setCompactMode,
  t,
  getFormValues,
  isAdminUser,
  supplierChannelLogsView,
}) => {
  const showSkeleton = useMinimumLoadingTime(loadingStat);
  const needSkeleton = !showStat || showSkeleton;
  const [exporting, setExporting] = useState(false);
  const { i18n } = useTranslation();

  const buildLogTypeQueryParam = (logType) => {
    if (logType == null || logType === '' || logType === '0') {
      return '0';
    }
    const raw = Array.isArray(logType) ? logType : [logType];
    const types = raw
      .map((v) => parseInt(String(v), 10))
      .filter((n) => !Number.isNaN(n) && n > 0);
    if (!types.length) {
      return '0';
    }
    return [...new Set(types)].join(',');
  };

  const handleExportStatement = async () => {
    if (typeof getFormValues !== 'function') {
      showError(t('当前视图不支持导出对账单'));
      return;
    }
    setExporting(true);
    try {
      const formValues = getFormValues();
      const today = getTodayStartTimestamp();
      const startTs = formValues.start_timestamp || today - 90 * 24 * 60 * 60;
      const endTs = formValues.end_timestamp || today + 24 * 60 * 60 - 1;
      // 把当前界面语言透传给后端，让 CSV 表头/元信息跟随用户语言。
      // 归一化后只保留 supportedLanguages 内的合法 lang，否则后端兜底 zh-CN。
      const lang = normalizeLanguage(i18n.language || '');
      const params = new URLSearchParams({
        start_timestamp: String(startTs),
        end_timestamp: String(endTs),
        model_name: formValues.model_name || '',
        token_name: formValues.token_name || '',
        group: formValues.group || '',
        request_id: formValues.request_id || '',
        type: buildLogTypeQueryParam(formValues.logType),
        lang: lang || '',
      });
      let url;
      if (supplierChannelLogsView) {
        if (formValues.channel) {
          params.set('channel', String(formValues.channel));
        }
        url = `/api/user/supplier-channel-logs/export?${params.toString()}`;
      } else {
        url = `/api/log/self/export?${params.toString()}`;
      }
      const res = await API.get(url, { responseType: 'blob' });
      const blob = new Blob([res.data], { type: 'text/csv;charset=utf-8' });
      const objectUrl = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = objectUrl;
      a.download = `statement-${new Date().toISOString().replace(/[-:T.Z]/g, '').slice(0, 14)}.csv`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(objectUrl);
      showSuccess(t('对账单导出成功'));
    } catch (e) {
      const text = e?.response?.data ? await blobToText(e.response.data) : e?.message || String(e);
      showError(t('对账单导出失败') + ': ' + text);
    } finally {
      setExporting(false);
    }
  };

  const placeholder = (
    <Space>
      <Skeleton.Title style={{ width: 108, height: 21, borderRadius: 6 }} />
      <Skeleton.Title style={{ width: 65, height: 21, borderRadius: 6 }} />
      <Skeleton.Title style={{ width: 64, height: 21, borderRadius: 6 }} />
    </Space>
  );

  return (
    <div className='flex flex-col md:flex-row justify-between items-start md:items-center gap-2 w-full'>
      <Skeleton loading={needSkeleton} active placeholder={placeholder}>
        <Space>
          <Tooltip content={t('消耗额度统计说明')}>
            <Tag
              color='blue'
              style={{
                fontWeight: 500,
                boxShadow: '0 2px 8px rgba(0, 0, 0, 0.1)',
                padding: 13,
                cursor: 'help',
              }}
              className='!rounded-lg'
            >
              {t('消耗额度')}: {renderLogStatDisplayQuota(stat)}
            </Tag>
          </Tooltip>
          <Tooltip content={t('RPM 统计说明')}>
            <Tag
              color='pink'
              style={{
                fontWeight: 500,
                boxShadow: '0 2px 8px rgba(0, 0, 0, 0.1)',
                padding: 13,
                cursor: 'help',
              }}
              className='!rounded-lg'
            >
              {t('RPM')}: {stat?.rpm ?? 0}
            </Tag>
          </Tooltip>
          <Tooltip content={t('TPM 统计说明')}>
            <Tag
              color='white'
              style={{
                border: 'none',
                boxShadow: '0 2px 8px rgba(0, 0, 0, 0.1)',
                fontWeight: 500,
                padding: 13,
                cursor: 'help',
              }}
              className='!rounded-lg'
            >
              {t('TPM')}: {stat?.tpm ?? 0}
            </Tag>
          </Tooltip>
        </Space>
      </Skeleton>

      <Space>
        <Tooltip content={t('导出对账单')}>
          <Button
            icon={<IconDownload />}
            loading={exporting}
            onClick={handleExportStatement}
            theme='outline'
            type='tertiary'
          >
            {t('导出对账单')}
          </Button>
        </Tooltip>
        <CompactModeToggle
          compactMode={compactMode}
          setCompactMode={setCompactMode}
          t={t}
        />
      </Space>
    </div>
  );
};

// 把后端返回的 blob 错误体转成可读字符串。
async function blobToText(blob) {
  try {
    return await blob.text();
  } catch (_) {
    return '';
  }
}

export default LogsActions;
