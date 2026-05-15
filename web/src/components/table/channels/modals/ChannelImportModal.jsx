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

import React, { useRef, useState } from 'react';
import {
  Button,
  Modal,
  Space,
  Spin,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { IconUpload } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../../../helpers';

const { Text, Title } = Typography;

/**
 * ChannelImportModal 渠道导入入口 + 结果展示。
 * @param {{ refresh: () => void }} props
 */
export default function ChannelImportModal({ refresh }) {
  const { t } = useTranslation();
  const fileInputRef = useRef(null);

  const [importing, setImporting] = useState(false);
  const [resultVisible, setResultVisible] = useState(false);
  const [importResult, setImportResult] = useState(null);

  /** 触发文件选择器 */
  const handleImportClick = () => {
    fileInputRef.current?.click();
  };

  /** 处理文件选择 */
  const handleFileChange = async (e) => {
    const file = e.target.files?.[0];
    if (!file) return;
    // 重置 input，允许重复选择同一文件
    e.target.value = '';

    // 文件格式校验
    if (!file.name.endsWith('.json') && file.type !== 'application/json') {
      showError(t('请选择 JSON 格式的导出文件'));
      return;
    }

    // 解析 JSON
    let parsed;
    try {
      const text = await file.text();
      parsed = JSON.parse(text);
    } catch {
      showError(t('文件解析失败，请确认是合法的 JSON 导出文件'));
      return;
    }

    // 基础结构校验
    if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
      showError(t('文件格式错误：根节点必须为 JSON 对象'));
      return;
    }
    if (!Array.isArray(parsed.channels)) {
      showError(t('文件格式错误：缺少 channels 数组字段'));
      return;
    }
    if (parsed.channels.length === 0) {
      showError(t('文件中未包含任何渠道数据'));
      return;
    }

    setImporting(true);
    try {
      const res = await API.post('/api/channel/import', parsed);
      if (!res?.data?.success) {
        showError(res?.data?.message || t('导入失败'));
        return;
      }
      setImportResult(res.data.data);
      setResultVisible(true);
      // 导入成功后自动刷新列表
      await refresh?.();
      if (res.data.data?.failed === 0) {
        showSuccess(t('渠道导入成功'));
      }
    } catch (err) {
      showError(err?.message || t('导入失败'));
    } finally {
      setImporting(false);
    }
  };

  /** 渲染导入结果弹窗内容 */
  const renderImportResult = () => {
    if (!importResult) return null;
    const { added = 0, updated = 0, failed = 0, failures = [] } = importResult;

    return (
      <div style={{ lineHeight: 1.8 }}>
        {/* 统计数字 */}
        <div style={{ marginBottom: 16, display: 'flex', gap: 8, flexWrap: 'wrap' }}>
          <Tag color='green' style={{ padding: '4px 12px' }}>
            {t('新增')} {added}
          </Tag>
          <Tag color='blue' style={{ padding: '4px 12px' }}>
            {t('更新')} {updated}
          </Tag>
          {failed > 0 && (
            <Tag color='red' style={{ padding: '4px 12px' }}>
              {t('失败')} {failed}
            </Tag>
          )}
        </div>

        {/* 失败详情列表 */}
        {failures.length > 0 && (
          <>
            <Title heading={6} style={{ marginBottom: 8 }}>
              {t('失败详情')}
            </Title>
            <div
              style={{
                maxHeight: 240,
                overflowY: 'auto',
                border: '1px solid var(--semi-color-border)',
                borderRadius: 6,
                padding: '8px 12px',
              }}
            >
              {failures.map((f, idx) => (
                <div
                  key={idx}
                  style={{
                    padding: '4px 0',
                    borderBottom:
                      idx < failures.length - 1
                        ? '1px solid var(--semi-color-border)'
                        : 'none',
                  }}
                >
                  <Text strong style={{ marginRight: 8 }}>
                    {f.name}
                  </Text>
                  <Text type='danger' size='small'>
                    {f.reason}
                  </Text>
                </div>
              ))}
            </div>
          </>
        )}

        {failed === 0 && (
          <Text type='success'>{t('全部渠道导入成功，无失败记录')}</Text>
        )}
      </div>
    );
  };

  return (
    <>
      <Spin spinning={importing}>
        <Button
          icon={<IconUpload />}
          loading={importing}
          onClick={handleImportClick}
          theme='light'
          size='small'
        >
          {t('导入')}
        </Button>
      </Spin>

      {/* 隐藏的文件选择器，仅接受 .json 文件 */}
      <input
        ref={fileInputRef}
        type='file'
        accept='.json,application/json'
        style={{ display: 'none' }}
        onChange={handleFileChange}
      />

      {/* 导入结果弹窗 */}
      <Modal
        title={t('导入结果')}
        visible={resultVisible}
        onOk={() => setResultVisible(false)}
        onCancel={() => setResultVisible(false)}
        cancelButtonProps={{ style: { display: 'none' } }}
        okText={t('确定')}
        width={480}
      >
        {renderImportResult()}
      </Modal>
    </>
  );
}
