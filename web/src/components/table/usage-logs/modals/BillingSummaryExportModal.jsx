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
import {
  Banner,
  Button,
  DatePicker,
  Modal,
  Radio,
  RadioGroup,
  Space,
  Typography,
} from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../../../helpers';
import { normalizeLanguage } from '../../../../i18n/language';

const { Text } = Typography;

const toDate = (seconds) => {
  const value = Number(seconds);
  return Number.isFinite(value) && value > 0 ? new Date(value * 1000) : null;
};

const toTimestamp = (value) => {
  if (!value) return 0;
  const date = value instanceof Date ? value : new Date(value);
  const timestamp = date.getTime();
  return Number.isFinite(timestamp) ? Math.floor(timestamp / 1000) : 0;
};

async function blobToText(blob) {
  try {
    return await blob.text();
  } catch (_) {
    return '';
  }
}

const BillingSummaryExportModal = ({ visible, onCancel, filters, t }) => {
  const { i18n } = useTranslation();
  const [dateRange, setDateRange] = useState([]);
  const [granularity, setGranularity] = useState('period');
  const [exporting, setExporting] = useState(false);

  useEffect(() => {
    if (!visible) return;
    const start = toDate(filters?.start_timestamp);
    const end = toDate(filters?.end_timestamp);
    setDateRange(start && end ? [start, end] : []);
  }, [visible, filters]);

  const handleExport = async () => {
    if (!Array.isArray(dateRange) || dateRange.length !== 2) {
      showError(t('请选择完整的导出时间范围'));
      return;
    }
    const startTimestamp = toTimestamp(dateRange[0]);
    const endTimestamp = toTimestamp(dateRange[1]);
    if (!startTimestamp || !endTimestamp || endTimestamp < startTimestamp) {
      showError(t('导出时间范围无效'));
      return;
    }

    setExporting(true);
    try {
      const params = new URLSearchParams({
        start_timestamp: String(startTimestamp),
        end_timestamp: String(endTimestamp),
        granularity,
        username: filters.username || '',
        channel: filters.channel ? String(filters.channel) : '',
        model_name: filters.model_name || '',
        token_name: filters.token_name || '',
        group: filters.group || '',
        request_id: filters.request_id || '',
        lang: normalizeLanguage(i18n.language || '') || '',
      });
      const response = await API.get(
        `/api/log/billing-summary/export?${params.toString()}`,
        { responseType: 'blob' },
      );
      const blob = new Blob([response.data], {
        type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
      });
      const objectUrl = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = objectUrl;
      link.download = `billing-summary-${new Date()
        .toISOString()
        .replace(/[-:T.Z]/g, '')
        .slice(0, 14)}.xlsx`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(objectUrl);
      showSuccess(t('计费汇总导出成功'));
      onCancel();
    } catch (error) {
      const message = error?.response?.data
        ? await blobToText(error.response.data)
        : error?.message || String(error);
      showError(`${t('计费汇总导出失败')}: ${message}`);
    } finally {
      setExporting(false);
    }
  };

  return (
    <Modal
      title={t('导出计费汇总')}
      visible={visible}
      onCancel={onCancel}
      width={620}
      footer={
        <Space>
          <Button onClick={onCancel}>{t('取消')}</Button>
          <Button
            theme='solid'
            type='primary'
            loading={exporting}
            onClick={handleExport}
          >
            {t('导出')}
          </Button>
        </Space>
      }
    >
      <Space vertical align='start' spacing='loose' className='w-full'>
        <Banner
          type='info'
          description={t(
            '工作簿按文本、视频、图片、音频拆分；视频按秒、图片按张等项目会显示真实计费单位。',
          )}
        />
        <div className='w-full'>
          <Text strong>{t('导出时间范围')}</Text>
          <DatePicker
            className='w-full mt-2'
            type='dateTimeRange'
            value={dateRange}
            onChange={(value) => setDateRange(value || [])}
            placeholder={[t('开始时间'), t('结束时间')]}
          />
        </div>
        <div className='w-full'>
          <Text strong>{t('汇总粒度')}</Text>
          <RadioGroup
            className='mt-2'
            value={granularity}
            onChange={(event) => setGranularity(event.target.value)}
          >
            <Radio value='period'>{t('整个时间范围')}</Radio>
            <Radio value='day'>{t('按日')}</Radio>
            <Radio value='month'>{t('按月')}</Radio>
          </RadioGroup>
        </div>
      </Space>
    </Modal>
  );
};

export default BillingSummaryExportModal;
