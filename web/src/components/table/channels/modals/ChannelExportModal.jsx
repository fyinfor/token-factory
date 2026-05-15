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
import {
  Button,
  Checkbox,
  Divider,
  Modal,
  Space,
  Typography,
} from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../../../helpers';

const { Text } = Typography;

/** 全部可导出字段的定义 */
const ALL_EXPORT_FIELDS = [
  { key: 'name',          label: '名称' },
  { key: 'discountRate',  label: '折扣率' },
  { key: 'routeSlug',     label: '路由后缀' },
  { key: 'quota',         label: '额度' },
  { key: 'disabled',      label: '状态（是否禁用）' },
  { key: 'supplierName',  label: '供应商' },
  { key: 'type',          label: '类型' },
  { key: 'logo',          label: '企业 Logo' },
  { key: 'providerType',  label: '供应商类型' },
  { key: 'apiKey',        label: '密钥' },
  { key: 'apiBaseUrl',    label: 'API 地址' },
  { key: 'models',        label: '模型' },
  { key: 'groups',        label: '分组' },
  { key: 'modelRedirect', label: '模型重定向' },
];

/** 默认勾选的字段（routeSlug、supplierName 默认不选） */
const DEFAULT_SELECTED_FIELDS = [
  'name', 'discountRate', 'quota', 'disabled',
  'type', 'logo', 'providerType', 'apiKey', 'apiBaseUrl',
  'models', 'groups', 'modelRedirect',
];

/**
 * ChannelExportModal 渠道导出字段选择弹窗。
 * @param {{ visible: boolean, onCancel: () => void, selectedChannels: Channel[] }} props
 */
export default function ChannelExportModal({ visible, onCancel, selectedChannels }) {
  const { t } = useTranslation();
  const [selectedFields, setSelectedFields] = useState(DEFAULT_SELECTED_FIELDS);
  const [exporting, setExporting] = useState(false);

  const allKeys = ALL_EXPORT_FIELDS.map((f) => f.key);
  const isAllSelected = allKeys.every((k) => selectedFields.includes(k));
  const isNoneSelected = selectedFields.length === 0;

  /** 全选 */
  const handleSelectAll = () => setSelectedFields([...allKeys]);

  /** 取消全选 */
  const handleDeselectAll = () => setSelectedFields([]);

  /** 恢复默认 */
  const handleRestoreDefault = () => setSelectedFields([...DEFAULT_SELECTED_FIELDS]);

  /** 切换单个字段 */
  const handleToggleField = (key) => {
    setSelectedFields((prev) =>
      prev.includes(key) ? prev.filter((k) => k !== key) : [...prev, key],
    );
  };

  /** 执行导出 */
  const handleExport = async () => {
    if (selectedChannels.length === 0) {
      showError(t('请先选择需要导出的渠道'));
      return;
    }
    if (selectedFields.length === 0) {
      showError(t('请至少选择一个导出字段'));
      return;
    }

    setExporting(true);
    try {
      const channelIds = selectedChannels.map((ch) => ch.id);
      const res = await API.post('/api/channel/export', {
        channel_ids: channelIds,
        fields: selectedFields,
      });

      if (!res?.data?.success) {
        showError(res?.data?.message || t('导出失败'));
        return;
      }

      // 生成文件名：channel-export-YYYYMMDD-HHmmss.json
      const now = new Date();
      const pad = (n) => String(n).padStart(2, '0');
      const dateStr =
        `${now.getFullYear()}${pad(now.getMonth() + 1)}${pad(now.getDate())}` +
        `-${pad(now.getHours())}${pad(now.getMinutes())}${pad(now.getSeconds())}`;
      const filename = `channel-export-${dateStr}.json`;

      // 触发文件下载
      const json = JSON.stringify(res.data.data, null, 2);
      const blob = new Blob([json], { type: 'application/json;charset=utf-8' });
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement('a');
      anchor.href = url;
      anchor.download = filename;
      document.body.appendChild(anchor);
      anchor.click();
      document.body.removeChild(anchor);
      URL.revokeObjectURL(url);

      showSuccess(t('渠道导出成功'));
      onCancel();
    } catch (err) {
      showError(err?.message || t('导出失败'));
    } finally {
      setExporting(false);
    }
  };

  return (
    <Modal
      title={t('导出字段选择')}
      visible={visible}
      onCancel={onCancel}
      width={480}
      footer={
        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
          <Button onClick={onCancel} disabled={exporting}>
            {t('取消')}
          </Button>
          <Button
            type='primary'
            loading={exporting}
            disabled={selectedFields.length === 0}
            onClick={handleExport}
          >
            {t('确认导出')}（{selectedChannels.length} {t('个渠道')}）
          </Button>
        </div>
      }
    >
      {/* 快捷操作区 */}
      <Space style={{ marginBottom: 12 }}>
        <Button size='small' theme='borderless' onClick={handleSelectAll} disabled={isAllSelected}>
          {t('全选')}
        </Button>
        <Button size='small' theme='borderless' onClick={handleDeselectAll} disabled={isNoneSelected}>
          {t('取消全选')}
        </Button>
        <Button size='small' theme='borderless' onClick={handleRestoreDefault}>
          {t('恢复默认')}
        </Button>
        <Text type='secondary' size='small'>
          {t('已选')} {selectedFields.length} / {ALL_EXPORT_FIELDS.length} {t('个字段')}
        </Text>
      </Space>

      <Divider margin='8px' />

      {/* 字段列表（两列排布） */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: '1fr 1fr',
          gap: '8px 16px',
          padding: '4px 0',
        }}
      >
        {ALL_EXPORT_FIELDS.map((field) => (
          <Checkbox
            key={field.key}
            checked={selectedFields.includes(field.key)}
            onChange={() => handleToggleField(field.key)}
            // name 字段强制勾选（导入时需要它做匹配）
            disabled={field.key === 'name'}
          >
            {t(field.label)}
            {field.key === 'name' && (
              <Text type='tertiary' size='small' style={{ marginLeft: 4 }}>
                ({t('必选')})
              </Text>
            )}
          </Checkbox>
        ))}
      </div>
    </Modal>
  );
}
