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
along with this program. If not, see <https://www.gnu.org/licenses/gpl-3.0.html>.
*/

import React, { useRef, useState } from 'react';
import {
  Button,
  Input,
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

/** TokenFactoryOpen (type=60) 渠道类型常量 */
const CHANNEL_TYPE_TOKEN_FACTORY_OPEN = 60;

/**
 * ChannelImportModal 渠道导入入口 + 结果展示。
 * 支持建站模式：当导入数据中包含 type=60 且 apiKey 为空的渠道时，
 * 需要用户填写关联密钥（作为渠道 apiKey）。
 * @param {{ refresh: () => void }} props
 */
export default function ChannelImportModal({ refresh }) {
  const { t } = useTranslation();
  const fileInputRef = useRef(null);

  const [importing, setImporting] = useState(false);
  const [resultVisible, setResultVisible] = useState(false);
  const [importResult, setImportResult] = useState(null);

  // 建站模式相关状态
  const [parsedData, setParsedData] = useState(null);
  const [siteBuilderApiKey, setSiteBuilderApiKey] = useState('');
  const [needsSiteBuilderKey, setNeedsSiteBuilderKey] = useState(false);
  const [confirmVisible, setConfirmVisible] = useState(false);

  /** 检测导入数据是否包含需要建站密钥的渠道（type=60 且 apiKey 为空） */
  const checkNeedsSiteBuilderKey = (data) => {
    if (!Array.isArray(data?.channels)) return false;
    return data.channels.some(
      (ch) =>
        ch.type === CHANNEL_TYPE_TOKEN_FACTORY_OPEN &&
        (!ch.apiKey || String(ch.apiKey).trim() === '')
    );
  };

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

    // 检测是否包含建站渠道（type=60 且 apiKey 为空）
    if (checkNeedsSiteBuilderKey(parsed)) {
      setParsedData(parsed);
      setNeedsSiteBuilderKey(true);
      setConfirmVisible(true);
      return;
    }

    // 普通导入：直接提交
    await doImport(parsed);
  };

  /** 执行导入请求 */
  const doImport = async (data, siteBuilderKey = '') => {
    setImporting(true);
    try {
      const payload = { ...data };
      if (siteBuilderKey && siteBuilderKey.trim()) {
        payload.site_builder_api_key = siteBuilderKey.trim();
      }
      const res = await API.post('/api/channel/import', payload);
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
      // 清理建站相关状态
      setParsedData(null);
      setNeedsSiteBuilderKey(false);
      setConfirmVisible(false);
      setSiteBuilderApiKey('');
    }
  };

  /** 确认导入（建站模式，用户已填写密钥） */
  const handleConfirmImport = () => {
    if (needsSiteBuilderKey && !siteBuilderApiKey.trim()) {
      showError(t('请输入关联密钥'));
      return;
    }
    setConfirmVisible(false);
    doImport(parsedData, siteBuilderApiKey);
  };

  /** 取消建站模式确认 */
  const handleCancelConfirm = () => {
    setConfirmVisible(false);
    setParsedData(null);
    setNeedsSiteBuilderKey(false);
    setSiteBuilderApiKey('');
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

  /** 计算建站渠道数量（type=60 且 apiKey 为空） */
  const siteBuilderChannelCount = parsedData?.channels?.filter(
    (ch) =>
      ch.type === CHANNEL_TYPE_TOKEN_FACTORY_OPEN &&
      (!ch.apiKey || String(ch.apiKey).trim() === '')
  ).length || 0;

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

      {/* 建站密钥输入弹窗 */}
      <Modal
        title={t('建站渠道导入 - 设置密钥')}
        visible={confirmVisible}
        onOk={handleConfirmImport}
        onCancel={handleCancelConfirm}
        okText={t('确认导入')}
        cancelText={t('取消')}
        width={520}
        okButtonProps={{
          disabled: needsSiteBuilderKey && !siteBuilderApiKey.trim(),
        }}
      >
        <div style={{ lineHeight: 1.8 }}>
          <Text>
            {t('检测到导入数据中包含 {{count}} 个建站渠道（TokenFactoryOpen 类型），请填写关联密钥后导入。', {
              count: siteBuilderChannelCount,
            })}
          </Text>
          <div style={{ marginTop: 16, marginBottom: 8 }}>
            <Text strong>{t('关联密钥')}</Text>
          </div>
          <Input
            value={siteBuilderApiKey}
            onChange={setSiteBuilderApiKey}
            placeholder={t('请输入渠道密钥')}
            showClear
            style={{ width: '100%' }}
          />
          <div style={{ marginTop: 8 }}>
            <Text type='quaternary' size='small'>
              {t('此密钥将作为所有建站渠道的 API Key，用于访问上游平台。请在目标平台的令牌管理页面创建获取。')}
            </Text>
          </div>
        </div>
      </Modal>

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
