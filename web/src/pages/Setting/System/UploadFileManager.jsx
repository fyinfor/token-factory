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

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Button,
  DatePicker,
  ImagePreview,
  Input,
  Modal,
  Radio,
  RadioGroup,
  Select,
  Space,
  Table,
  Tag,
  Tooltip,
  Typography,
} from '@douyinfe/semi-ui';
import {
  CalendarClock,
  Copy,
  ExternalLink,
  Eye,
  File,
  FileAudio,
  FileImage,
  FileText,
  FileVideo,
  RefreshCw,
  Search,
  Trash2,
} from 'lucide-react';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import { API, copy, showError, showSuccess } from '../../../helpers';

const { Text, Title } = Typography;

const PURPOSE_OPTIONS = [
  { value: 'all', label: '全部用途' },
  { value: 'homepage', label: '首页' },
  { value: 'icons', label: '图标' },
  { value: 'supplier', label: '供应商' },
  { value: 'distributor', label: '分销商' },
  { value: 'channel', label: '渠道' },
  { value: 'general', label: '通用' },
  { value: 'playground', label: 'Playground' },
  { value: 'legacy', label: '旧文件' },
];

const LIFECYCLE_OPTIONS = [
  { value: 'all', label: '全部生命周期' },
  { value: 'permanent', label: '永久文件' },
  { value: 'temporary', label: '临时文件' },
];

const panelStyle = {
  border: '1px solid var(--semi-color-border)',
  borderRadius: 8,
  padding: 16,
  background: 'var(--semi-color-bg-0)',
};

function previewKind(file) {
  const mime = String(file?.mime_type || '').toLowerCase();
  const url = String(file?.url || '')
    .toLowerCase()
    .split(/[?#]/)[0];
  if (
    mime.startsWith('image/') ||
    /\.(png|jpe?g|gif|webp|bmp|svg)$/.test(url)
  ) {
    return 'image';
  }
  if (mime.startsWith('video/') || /\.(mp4|webm|mov|m4v|avi)$/.test(url)) {
    return 'video';
  }
  if (mime.startsWith('audio/') || /\.(mp3|wav|ogg|m4a|aac|flac)$/.test(url)) {
    return 'audio';
  }
  if (mime === 'application/pdf' || url.endsWith('.pdf')) return 'pdf';
  return 'file';
}

function formatBytes(value) {
  const bytes = Number(value || 0);
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const unitIndex = Math.min(
    Math.floor(Math.log(bytes) / Math.log(1024)),
    units.length - 1,
  );
  const amount = bytes / 1024 ** unitIndex;
  return `${amount >= 10 || unitIndex === 0 ? amount.toFixed(0) : amount.toFixed(1)} ${units[unitIndex]}`;
}

function purposeLabel(value, t) {
  return t(
    PURPOSE_OPTIONS.find((item) => item.value === value)?.label ||
      value ||
      '通用',
  );
}

function FileTypeIcon({ kind }) {
  const props = { size: 19, strokeWidth: 1.8 };
  if (kind === 'image') return <FileImage {...props} />;
  if (kind === 'video') return <FileVideo {...props} />;
  if (kind === 'audio') return <FileAudio {...props} />;
  if (kind === 'pdf') return <FileText {...props} />;
  return <File {...props} />;
}

export default function UploadFileManager() {
  const { t } = useTranslation();
  const [files, setFiles] = useState([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [keywordInput, setKeywordInput] = useState('');
  const [keyword, setKeyword] = useState('');
  const [purpose, setPurpose] = useState('all');
  const [lifecycle, setLifecycle] = useState('all');
  const [previewFile, setPreviewFile] = useState(null);
  const [selectedRowKeys, setSelectedRowKeys] = useState([]);
  const [deletingId, setDeletingId] = useState(null);
  const [batchDeleting, setBatchDeleting] = useState(false);
  const [expirationTarget, setExpirationTarget] = useState(null);
  const [expirationMode, setExpirationMode] = useState('scheduled');
  const [expirationAt, setExpirationAt] = useState(null);
  const [savingExpiration, setSavingExpiration] = useState(false);

  const loadFiles = useCallback(async () => {
    setLoading(true);
    try {
      const response = await API.get('/api/oss/files', {
        params: {
          p: page,
          page_size: pageSize,
          keyword,
          purpose: purpose === 'all' ? '' : purpose,
          lifecycle: lifecycle === 'all' ? '' : lifecycle,
        },
        disableDuplicate: true,
      });
      const { success, message, data } = response.data;
      if (!success) {
        showError(t(message || '读取文件列表失败'));
        return;
      }
      setFiles(data?.items || []);
      setTotal(Number(data?.total || 0));
      setSelectedRowKeys([]);
    } catch (error) {
      showError(error.response?.data?.message || t('读取文件列表失败'));
    } finally {
      setLoading(false);
    }
  }, [keyword, lifecycle, page, pageSize, purpose, t]);

  useEffect(() => {
    loadFiles();
  }, [loadFiles]);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      setPage(1);
      setKeyword(keywordInput.trim());
    }, 300);
    return () => window.clearTimeout(timer);
  }, [keywordInput]);

  const syncExistingFiles = async () => {
    setSyncing(true);
    try {
      const response = await API.post('/api/oss/files/sync');
      const { success, message, data } = response.data;
      if (!success) {
        showError(t(message || '同步已有文件失败'));
        return;
      }
      showSuccess(t('已同步 {{count}} 个文件', { count: data?.synced || 0 }));
      setPage(1);
      await loadFiles();
    } catch (error) {
      showError(error.response?.data?.message || t('同步已有文件失败'));
    } finally {
      setSyncing(false);
    }
  };

  const copyURL = useCallback(
    async (url) => {
      if (await copy(url)) {
        showSuccess(t('链接已复制'));
      } else {
        showError(t('复制失败'));
      }
    },
    [t],
  );

  const openURL = useCallback((url) => {
    window.open(url, '_blank', 'noopener,noreferrer');
  }, []);

  const showPreview = useCallback((file) => {
    setPreviewFile(file);
  }, []);

  const openExpirationEditor = useCallback((file) => {
    const expiresAt = Number(file?.expires_at || 0);
    setExpirationTarget(file);
    setExpirationMode(expiresAt > 0 ? 'scheduled' : 'permanent');
    setExpirationAt(
      new Date(expiresAt > 0 ? expiresAt * 1000 : Date.now() + 86400000),
    );
  }, []);

  const deleteFile = useCallback(
    async (file) => {
      if (!file || deletingId !== null) return;
      setDeletingId(file.id);
      try {
        const response = await API.delete(`/api/oss/files/${file.id}`);
        const { success, message } = response.data;
        if (!success) {
          showError(t(message || '删除文件失败'));
          return;
        }
        showSuccess(t('文件已删除'));
        if (files.length === 1 && page > 1) {
          setPage(page - 1);
        } else {
          await loadFiles();
        }
      } catch (error) {
        showError(error.response?.data?.message || t('删除文件失败'));
      } finally {
        setDeletingId(null);
      }
    },
    [deletingId, files.length, loadFiles, page, t],
  );

  const batchDeleteFiles = async () => {
    if (selectedRowKeys.length === 0 || batchDeleting) return;
    setBatchDeleting(true);
    try {
      const response = await API.post('/api/oss/files/batch-delete', {
        ids: selectedRowKeys,
      });
      const { success, message, data } = response.data;
      if (!success) {
        showError(t(message || '批量删除失败'));
        return;
      }
      const deletedCount = data?.deleted_ids?.length || 0;
      const failureCount = data?.failures?.length || 0;
      if (deletedCount > 0) {
        showSuccess(t('已删除 {{count}} 个文件', { count: deletedCount }));
      }
      if (failureCount > 0) {
        showError(t('{{count}} 个文件删除失败', { count: failureCount }));
      }
      setSelectedRowKeys([]);
      if (deletedCount === files.length && failureCount === 0 && page > 1) {
        setPage(page - 1);
      } else {
        await loadFiles();
      }
    } catch (error) {
      showError(error.response?.data?.message || t('批量删除失败'));
    } finally {
      setBatchDeleting(false);
    }
  };

  const saveExpiration = async () => {
    if (!expirationTarget) return;
    let expiresAt = 0;
    if (expirationMode === 'scheduled') {
      const selectedTime = dayjs(expirationAt);
      if (!expirationAt || !selectedTime.isValid()) {
        showError(t('请选择过期时间'));
        return;
      }
      expiresAt = selectedTime.unix();
      if (expiresAt <= dayjs().unix()) {
        showError(t('过期时间不能早于当前时间！'));
        return;
      }
    }
    setSavingExpiration(true);
    try {
      const response = await API.put(
        `/api/oss/files/${expirationTarget.id}/expiration`,
        { expires_at: expiresAt },
      );
      const { success, message } = response.data;
      if (!success) {
        showError(t(message || '设置过期时间失败'));
        return;
      }
      showSuccess(t('过期时间已更新'));
      setExpirationTarget(null);
      await loadFiles();
    } catch (error) {
      showError(error.response?.data?.message || t('设置过期时间失败'));
    } finally {
      setSavingExpiration(false);
    }
  };

  const columns = useMemo(
    () => [
      {
        title: t('文件'),
        dataIndex: 'original_name',
        width: 310,
        render: (_, file) => {
          const kind = previewKind(file);
          return (
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 10,
                minWidth: 0,
              }}
            >
              <button
                type='button'
                aria-label={t('预览')}
                onClick={() => showPreview(file)}
                style={{
                  width: 44,
                  height: 44,
                  minWidth: 44,
                  border: '1px solid var(--semi-color-border)',
                  borderRadius: 6,
                  display: 'grid',
                  placeItems: 'center',
                  overflow: 'hidden',
                  padding: 0,
                  color: 'var(--semi-color-text-2)',
                  background: 'var(--semi-color-fill-0)',
                  cursor: 'pointer',
                }}
              >
                {kind === 'image' ? (
                  <img
                    src={file.url}
                    alt=''
                    width={44}
                    height={44}
                    loading='lazy'
                    decoding='async'
                    fetchPriority='low'
                    draggable={false}
                    style={{
                      width: '100%',
                      height: '100%',
                      objectFit: 'cover',
                    }}
                  />
                ) : (
                  <FileTypeIcon kind={kind} />
                )}
              </button>
              <div style={{ minWidth: 0 }}>
                <Tooltip content={file.original_name || file.url}>
                  <Text
                    strong
                    ellipsis={{ showTooltip: false }}
                    style={{ display: 'block', maxWidth: 235 }}
                  >
                    {file.original_name || t('未命名文件')}
                  </Text>
                </Tooltip>
                <Text
                  type='tertiary'
                  size='small'
                  ellipsis={{ showTooltip: true }}
                  style={{ display: 'block', maxWidth: 235 }}
                >
                  {file.mime_type || t('未知类型')}
                </Text>
              </div>
            </div>
          );
        },
      },
      {
        title: t('用途'),
        dataIndex: 'purpose',
        width: 110,
        render: (value) => (
          <Tag
            color={
              value === 'legacy'
                ? 'grey'
                : value === 'playground'
                  ? 'orange'
                  : 'blue'
            }
          >
            {purposeLabel(value, t)}
          </Tag>
        ),
      },
      {
        title: t('生命周期'),
        dataIndex: 'expires_at',
        width: 165,
        render: (value) =>
          Number(value || 0) === 0 ? (
            <Tag color='green'>{t('永久')}</Tag>
          ) : (
            <div>
              <Tag color='orange'>{t('临时')}</Tag>
              <Text
                type='tertiary'
                size='small'
                style={{ display: 'block', marginTop: 4 }}
              >
                {dayjs.unix(Number(value)).format('YYYY-MM-DD HH:mm')}
              </Text>
            </div>
          ),
      },
      {
        title: t('大小'),
        dataIndex: 'size',
        width: 90,
        render: formatBytes,
      },
      {
        title: t('上传时间'),
        dataIndex: 'created_at',
        width: 155,
        render: (value) =>
          value ? dayjs.unix(Number(value)).format('YYYY-MM-DD HH:mm') : '-',
      },
      {
        title: t('操作'),
        width: 216,
        render: (_, file) => (
          <Space spacing={4}>
            <Tooltip content={t('预览')}>
              <Button
                aria-label={t('预览')}
                icon={<Eye size={16} />}
                size='small'
                onClick={() => showPreview(file)}
              />
            </Tooltip>
            <Tooltip content={t('设置过期时间')}>
              <Button
                aria-label={t('设置过期时间')}
                icon={<CalendarClock size={16} />}
                size='small'
                onClick={() => openExpirationEditor(file)}
              />
            </Tooltip>
            <Tooltip content={t('复制链接')}>
              <Button
                aria-label={t('复制链接')}
                icon={<Copy size={16} />}
                size='small'
                onClick={() => copyURL(file.url)}
              />
            </Tooltip>
            <Tooltip content={t('在新窗口打开')}>
              <Button
                aria-label={t('在新窗口打开')}
                icon={<ExternalLink size={16} />}
                size='small'
                onClick={() => openURL(file.url)}
              />
            </Tooltip>
            <Tooltip content={t('删除')}>
              <Button
                aria-label={t('删除')}
                icon={<Trash2 size={16} />}
                size='small'
                type='danger'
                loading={deletingId === file.id}
                disabled={
                  batchDeleting ||
                  (deletingId !== null && deletingId !== file.id)
                }
                onClick={() => deleteFile(file)}
              />
            </Tooltip>
          </Space>
        ),
      },
    ],
    [
      copyURL,
      deleteFile,
      deletingId,
      batchDeleting,
      openExpirationEditor,
      openURL,
      showPreview,
      t,
    ],
  );

  const kind = previewKind(previewFile);
  const scheduledExpirationValid =
    expirationMode === 'permanent' ||
    (expirationAt && dayjs(expirationAt).isAfter(dayjs()));

  return (
    <div style={panelStyle}>
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          gap: 12,
          flexWrap: 'wrap',
          marginBottom: 14,
        }}
      >
        <div>
          <Title heading={6} style={{ margin: 0 }}>
            {t('文件管理')}
          </Title>
          <Text type='tertiary' size='small'>
            {t('共 {{count}} 个文件', { count: total })}
          </Text>
        </div>
        <Space>
          <Button
            type='danger'
            icon={<Trash2 size={16} />}
            disabled={selectedRowKeys.length === 0 || deletingId !== null}
            loading={batchDeleting}
            onClick={batchDeleteFiles}
          >
            {t('批量删除')}
            {selectedRowKeys.length > 0 ? ` (${selectedRowKeys.length})` : null}
          </Button>
          <Button
            icon={<RefreshCw size={16} />}
            loading={syncing}
            onClick={syncExistingFiles}
          >
            {t('同步已有文件')}
          </Button>
        </Space>
      </div>

      <div
        style={{
          display: 'flex',
          gap: 8,
          flexWrap: 'wrap',
          marginBottom: 14,
        }}
      >
        <Input
          prefix={<Search size={16} />}
          value={keywordInput}
          onChange={setKeywordInput}
          placeholder={t('搜索文件名或链接')}
          showClear
          style={{ width: 250 }}
        />
        <Select
          value={purpose}
          optionList={PURPOSE_OPTIONS.map((item) => ({
            ...item,
            label: t(item.label),
          }))}
          onChange={(value) => {
            setPage(1);
            setPurpose(value);
          }}
          style={{ width: 150 }}
        />
        <Select
          value={lifecycle}
          optionList={LIFECYCLE_OPTIONS.map((item) => ({
            ...item,
            label: t(item.label),
          }))}
          onChange={(value) => {
            setPage(1);
            setLifecycle(value);
          }}
          style={{ width: 150 }}
        />
      </div>

      <Table
        columns={columns}
        dataSource={files}
        loading={loading}
        rowKey='id'
        style={{ width: '100%' }}
        rowSelection={{
          selectedRowKeys,
          onChange: setSelectedRowKeys,
        }}
        empty={t('暂无文件，请同步已有文件或上传新文件')}
        pagination={{
          currentPage: page,
          pageSize,
          total,
          showSizeChanger: true,
          pageSizeOpts: [10, 20, 50, 100],
          onPageChange: setPage,
          onPageSizeChange: (size) => {
            setPage(1);
            setPageSize(size);
          },
        }}
      />

      <ImagePreview
        src={kind === 'image' ? previewFile?.url || '' : ''}
        visible={Boolean(previewFile && kind === 'image')}
        onVisibleChange={(visible) => {
          if (!visible) setPreviewFile(null);
        }}
      />

      <Modal
        title={previewFile?.original_name || t('文件预览')}
        visible={Boolean(previewFile && kind !== 'image')}
        onCancel={() => setPreviewFile(null)}
        footer={
          <Button
            type='primary'
            icon={<ExternalLink size={16} />}
            onClick={() => openURL(previewFile?.url)}
          >
            {t('在新窗口打开')}
          </Button>
        }
        width={kind === 'pdf' ? 920 : 760}
        bodyStyle={{ padding: 16 }}
      >
        <div
          style={{
            minHeight: 180,
            maxHeight: '68vh',
            display: 'grid',
            placeItems: 'center',
            overflow: 'auto',
            background: 'var(--semi-color-fill-0)',
            borderRadius: 6,
          }}
        >
          {previewFile && kind === 'video' ? (
            <video
              src={previewFile.url}
              controls
              style={{ width: '100%', maxHeight: '66vh' }}
            />
          ) : null}
          {previewFile && kind === 'audio' ? (
            <audio
              src={previewFile.url}
              controls
              style={{ width: 'min(560px, 90%)' }}
            />
          ) : null}
          {previewFile && kind === 'pdf' ? (
            <iframe
              src={previewFile.url}
              title={previewFile.original_name || t('PDF 预览')}
              style={{ width: '100%', height: '66vh', border: 0 }}
            />
          ) : null}
          {previewFile && kind === 'file' ? (
            <div style={{ textAlign: 'center', padding: 32 }}>
              <File size={34} strokeWidth={1.5} />
              <div style={{ marginTop: 10 }}>
                <Text type='tertiary'>{t('此文件类型不支持内嵌预览')}</Text>
              </div>
            </div>
          ) : null}
        </div>
      </Modal>

      <Modal
        title={t('设置过期时间')}
        visible={Boolean(expirationTarget)}
        onCancel={() => setExpirationTarget(null)}
        onOk={saveExpiration}
        okText={t('保存')}
        cancelText={t('取消')}
        confirmLoading={savingExpiration}
        okButtonProps={{
          disabled: !scheduledExpirationValid,
        }}
      >
        <Text strong style={{ display: 'block', marginBottom: 16 }}>
          {expirationTarget?.original_name || t('未命名文件')}
        </Text>
        <RadioGroup
          value={expirationMode}
          onChange={(event) => setExpirationMode(event.target.value)}
          style={{ marginBottom: 16 }}
        >
          <Radio value='permanent'>{t('永久')}</Radio>
          <Radio value='scheduled'>{t('指定时间')}</Radio>
        </RadioGroup>
        <DatePicker
          type='dateTime'
          value={expirationAt}
          onChange={setExpirationAt}
          disabled={expirationMode === 'permanent'}
          placeholder={t('请选择过期时间')}
          showClear={false}
          style={{ width: '100%' }}
        />
      </Modal>
    </div>
  );
}
