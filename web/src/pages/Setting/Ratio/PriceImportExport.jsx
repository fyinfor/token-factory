import React, { useRef, useState } from 'react';
import {
  Button,
  Modal,
  Space,
  Spin,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import {
  IconDownload,
  IconUpload,
} from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../../helpers';

const { Text, Title } = Typography;

/**
 * PriceImportExport 价格导入/导出操作区。
 * @param {{ refresh: () => void }} props
 */
export default function PriceImportExport({ refresh }) {
  const { t } = useTranslation();
  const fileInputRef = useRef(null);

  const [exporting, setExporting] = useState(false);
  const [importing, setImporting] = useState(false);
  const [resultVisible, setResultVisible] = useState(false);
  const [importResult, setImportResult] = useState(null);

  // ─── 导出 ──────────────────────────────────────────────────────────────────

  const handleExport = async () => {
    setExporting(true);
    try {
      const res = await API.get('/api/admin/price/export');
      if (!res?.data?.success) {
        showError(res?.data?.message || t('导出失败'));
        return;
      }
      const json = JSON.stringify(res.data.data, null, 2);
      const blob = new Blob([json], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement('a');
      anchor.href = url;
      anchor.download = `price_export_${new Date().toISOString().slice(0, 10)}.json`;
      document.body.appendChild(anchor);
      anchor.click();
      document.body.removeChild(anchor);
      URL.revokeObjectURL(url);
      showSuccess(t('价格导出成功'));
    } catch (err) {
      showError(err?.message || t('导出失败'));
    } finally {
      setExporting(false);
    }
  };

  // ─── 导入 ──────────────────────────────────────────────────────────────────

  const handleImportClick = () => {
    fileInputRef.current?.click();
  };

  const handleFileChange = async (e) => {
    const file = e.target.files?.[0];
    if (!file) return;
    // 重置 input，允许重复选择同一文件
    e.target.value = '';

    if (!file.name.endsWith('.json') && file.type !== 'application/json') {
      showError(t('请选择 JSON 格式的导出文件'));
      return;
    }

    let parsed;
    try {
      const text = await file.text();
      parsed = JSON.parse(text);
    } catch {
      showError(t('文件解析失败，请确认是合法的 JSON 导出文件'));
      return;
    }

    // 基础结构校验
    if (typeof parsed !== 'object' || parsed === null) {
      showError(t('文件格式错误：根节点必须为 JSON 对象'));
      return;
    }
    if (!('global_prices' in parsed) && !('channels' in parsed)) {
      showError(t('文件格式错误：缺少 global_prices 或 channels 字段'));
      return;
    }

    setImporting(true);
    try {
      const res = await API.post('/api/admin/price/import', parsed);
      if (!res?.data?.success) {
        showError(res?.data?.message || t('导入失败'));
        return;
      }
      setImportResult(res.data.data);
      setResultVisible(true);
      await refresh();
    } catch (err) {
      showError(err?.message || t('导入失败'));
    } finally {
      setImporting(false);
    }
  };

  // ─── 导入结果弹窗 ──────────────────────────────────────────────────────────

  const renderImportResult = () => {
    if (!importResult) return null;
    const {
      global_updated = 0,
      global_added = 0,
      channel_stats = [],
      skipped_channels = [],
    } = importResult;

    const totalChannelAdded = channel_stats.reduce((s, c) => s + (c.added || 0), 0);
    const totalChannelUpdated = channel_stats.reduce((s, c) => s + (c.updated || 0), 0);

    return (
      <div style={{ lineHeight: 1.8 }}>
        <Title heading={6} style={{ marginBottom: 8 }}>
          {t('全局价格')}
        </Title>
        <div style={{ marginBottom: 12 }}>
          <Tag color='green' style={{ marginRight: 8 }}>
            {t('新增')} {global_added}
          </Tag>
          <Tag color='blue'>
            {t('更新')} {global_updated}
          </Tag>
        </div>

        {channel_stats.length > 0 && (
          <>
            <Title heading={6} style={{ marginBottom: 8 }}>
              {t('渠道价格')}
            </Title>
            <div style={{ marginBottom: 8 }}>
              <Tag color='green' style={{ marginRight: 8 }}>
                {t('新增')} {totalChannelAdded}
              </Tag>
              <Tag color='blue'>
                {t('更新')} {totalChannelUpdated}
              </Tag>
            </div>
            <div style={{ maxHeight: 200, overflowY: 'auto', marginBottom: 12 }}>
              {channel_stats.map((s) => (
                <div key={s.channel_name} style={{ fontSize: 12, color: 'var(--semi-color-text-2)' }}>
                  {s.channel_name}：{t('新增')} {s.added}，{t('更新')} {s.updated}
                </div>
              ))}
            </div>
          </>
        )}

        {skipped_channels.length > 0 && (
          <>
            <Title heading={6} style={{ marginBottom: 8 }}>
              {t('跳过的渠道')}（{t('名称不匹配或渠道不存在')}）
            </Title>
            <div style={{ maxHeight: 120, overflowY: 'auto' }}>
              {skipped_channels.map((name) => (
                <Tag
                  key={name}
                  color='grey'
                  style={{ marginRight: 4, marginBottom: 4, fontSize: 12 }}
                >
                  {name}
                </Tag>
              ))}
            </div>
          </>
        )}
      </div>
    );
  };

  return (
    <>
      <Space>
        <Button
          icon={<IconDownload />}
          loading={exporting}
          onClick={handleExport}
          theme='light'
        >
          {t('导出价格')}
        </Button>
        <Spin spinning={importing}>
          <Button
            icon={<IconUpload />}
            loading={importing}
            onClick={handleImportClick}
            theme='light'
          >
            {t('导入价格')}
          </Button>
        </Spin>
      </Space>

      {/* 隐藏的文件选择器 */}
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
