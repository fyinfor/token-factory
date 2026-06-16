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
import { Button } from '@douyinfe/semi-ui';
import { IconDownload } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../../../helpers';

/**
 * ModelExportButton 模型数据一键导出按钮组件。
 * 点击后直接调用后端 API 导出，下载 JSON 文件。
 * 导出固定字段：模型类型（name, description, icon）、模型数据（model_name, name_rule, icon, description, doc_introduction, api_docs, tags, vendor, endpoints, sync_official, status）
 */
export default function ModelExportButton() {
  const { t } = useTranslation();
  const [exporting, setExporting] = useState(false);

  /** 执行导出 */
  const handleExport = async () => {
    setExporting(true);
    try {
      const res = await API.get('/api/models/export');
      if (!res?.data?.success) {
        showError(res?.data?.message || t('导出失败'));
        return;
      }

      const data = res.data.data;

      // 生成文件名：model-export-YYYYMMDD-HHmmss.json
      const now = new Date();
      const pad = (n) => String(n).padStart(2, '0');
      const dateStr =
        `${now.getFullYear()}${pad(now.getMonth() + 1)}${pad(now.getDate())}` +
        `-${pad(now.getHours())}${pad(now.getMinutes())}${pad(now.getSeconds())}`;
      const filename = `model-export-${dateStr}.json`;

      // 触发文件下载
      const json = JSON.stringify(data, null, 2);
      const blob = new Blob([json], { type: 'application/json;charset=utf-8' });
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement('a');
      anchor.href = url;
      anchor.download = filename;
      document.body.appendChild(anchor);
      anchor.click();
      document.body.removeChild(anchor);
      URL.revokeObjectURL(url);

      const vendorCount = data?.vendors?.length || 0;
      const modelCount = data?.models?.length || 0;
      showSuccess(
        t('导出成功：{{vendorCount}} 个模型类型，{{modelCount}} 个模型数据', {
          vendorCount,
          modelCount,
        }),
      );
    } catch (err) {
      showError(err?.message || t('导出失败'));
    } finally {
      setExporting(false);
    }
  };

  return (
    <Button
      icon={<IconDownload />}
      loading={exporting}
      onClick={handleExport}
      theme='light'
      size='small'
    >
      {t('导出')}
    </Button>
  );
}
