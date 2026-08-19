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

import React, { useEffect, useMemo, useRef, useState } from 'react';
import {
  Button,
  Space,
  Table,
  Form,
  Typography,
  Empty,
  Divider,
  Input,
  Modal,
  Select,
  Tag,
  Switch,
  TextArea,
  Tooltip,
} from '@douyinfe/semi-ui';
import {
  IllustrationNoResult,
  IllustrationNoResultDark,
} from '@douyinfe/semi-illustrations';
import {
  Plus,
  Edit,
  Trash2,
  Save,
  Bell,
  Maximize2,
  Languages,
  Square,
} from 'lucide-react';
import {
  API,
  getUserIdFromLocalStorage,
  showError,
  showSuccess,
  getRelativeTime,
  formatDateTimeString,
} from '../../../helpers';
import { useTranslation } from 'react-i18next';
import {
  readAssistantContent,
  readAssistantStreamChunk,
  stripOuterMarkdownFence,
  stripReasoningContent,
} from '../../../components/table/models/modals/documentAiUtils';

const { Text } = Typography;

const normalizeTranslation = (content) =>
  stripOuterMarkdownFence(stripReasoningContent(content));

const isTextModel = (item) => {
  const tags = String(item?.tags || '')
    .replaceAll('，', ',')
    .replaceAll('、', ',')
    .split(',')
    .map((tag) => tag.trim())
    .filter(Boolean);
  return tags.length === 0 || tags.includes('文本');
};

const streamAnnouncementTranslation = async ({
  payload,
  signal,
  onProgress,
}) => {
  const response = await fetch(
    API.getUri({ url: '/api/models/document_ai/generate' }),
    {
      method: 'POST',
      credentials: 'include',
      signal,
      headers: {
        Accept: 'text/event-stream',
        'Content-Type': 'application/json',
        'New-API-User': String(getUserIdFromLocalStorage()),
      },
      body: JSON.stringify(payload),
    },
  );
  if (!response.ok) {
    const rawError = await response.text();
    try {
      const parsed = JSON.parse(rawError);
      throw new Error(
        parsed?.error?.message || parsed?.message || response.statusText,
      );
    } catch (error) {
      if (error instanceof SyntaxError) {
        throw new Error(rawError || response.statusText);
      }
      throw error;
    }
  }
  if (
    !String(response.headers.get('content-type')).includes('text/event-stream')
  ) {
    return readAssistantContent({ data: await response.json() });
  }
  if (!response.body) {
    throw new Error('Streaming response body is unavailable');
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';
  let streamed = '';
  const consumeEvent = (event) => {
    const data = event
      .split(/\r?\n/)
      .filter((line) => line.startsWith('data:'))
      .map((line) => line.slice(5).trimStart())
      .join('\n')
      .trim();
    if (!data || data === '[DONE]') return;
    let parsed;
    try {
      parsed = JSON.parse(data);
    } catch (_) {
      return;
    }
    if (parsed?.error) {
      throw new Error(parsed.error.message || 'AI stream failed');
    }
    const delta = readAssistantStreamChunk(parsed);
    if (!delta) return;
    streamed += delta;
    onProgress?.(streamed);
  };

  while (true) {
    const { done, value } = await reader.read();
    buffer += decoder.decode(value || new Uint8Array(), { stream: !done });
    const events = buffer.split(/\r?\n\r?\n/);
    buffer = events.pop() || '';
    events.forEach(consumeEvent);
    if (done) break;
  }
  if (buffer.trim()) consumeEvent(buffer);
  return normalizeTranslation(streamed);
};

const SettingsAnnouncements = ({ options, refresh }) => {
  const { t } = useTranslation();

  const [announcementsList, setAnnouncementsList] = useState([]);
  const [showAnnouncementModal, setShowAnnouncementModal] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [showContentModal, setShowContentModal] = useState(false);
  const [deletingAnnouncement, setDeletingAnnouncement] = useState(null);
  const [editingAnnouncement, setEditingAnnouncement] = useState(null);
  const [modalLoading, setModalLoading] = useState(false);
  const [loading, setLoading] = useState(false);
  const [hasChanges, setHasChanges] = useState(false);
  const [announcementForm, setAnnouncementForm] = useState({
    content: '',
    content_en: '',
    publishDate: new Date(),
    type: 'default',
    extra: '',
    extra_en: '',
  });
  const [announcementLanguage, setAnnouncementLanguage] = useState('zh');
  const [textModels, setTextModels] = useState([]);
  const [selectedTextModel, setSelectedTextModel] = useState('');
  const [translationLoading, setTranslationLoading] = useState(false);
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [selectedRowKeys, setSelectedRowKeys] = useState([]);

  // 面板启用状态
  const [panelEnabled, setPanelEnabled] = useState(true);

  const translationAbortControllerRef = useRef(null);

  const textModelOptions = useMemo(
    () =>
      textModels.map((item) => ({
        label: item.model_name,
        value: item.model_name,
      })),
    [textModels],
  );

  const typeOptions = [
    { value: 'default', label: t('默认') },
    { value: 'ongoing', label: t('进行中') },
    { value: 'success', label: t('成功') },
    { value: 'warning', label: t('警告') },
    { value: 'error', label: t('错误') },
  ];

  const getTypeColor = (type) => {
    const colorMap = {
      default: 'grey',
      ongoing: 'blue',
      success: 'green',
      warning: 'orange',
      error: 'red',
    };
    return colorMap[type] || 'grey';
  };

  const columns = [
    {
      title: t('内容'),
      dataIndex: 'content',
      key: 'content',
      render: (text) => (
        <Tooltip content={text} position='topLeft' showArrow>
          <div
            style={{
              maxWidth: '300px',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            }}
          >
            {text}
          </div>
        </Tooltip>
      ),
    },
    {
      title: t('发布时间'),
      dataIndex: 'publishDate',
      key: 'publishDate',
      width: 180,
      render: (publishDate) => (
        <div>
          <div style={{ fontWeight: 'bold' }}>
            {getRelativeTime(publishDate)}
          </div>
          <div
            style={{
              fontSize: '12px',
              color: 'var(--semi-color-text-2)',
              marginTop: '2px',
            }}
          >
            {publishDate ? formatDateTimeString(new Date(publishDate)) : '-'}
          </div>
        </div>
      ),
    },
    {
      title: t('类型'),
      dataIndex: 'type',
      key: 'type',
      width: 100,
      render: (type) => (
        <Tag color={getTypeColor(type)} shape='circle'>
          {typeOptions.find((opt) => opt.value === type)?.label || type}
        </Tag>
      ),
    },
    {
      title: t('说明'),
      dataIndex: 'extra',
      key: 'extra',
      render: (text) => (
        <Tooltip content={text || '-'} showArrow>
          <div
            style={{
              maxWidth: '200px',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
              color: 'var(--semi-color-text-2)',
            }}
          >
            {text || '-'}
          </div>
        </Tooltip>
      ),
    },
    {
      title: t('操作'),
      key: 'action',
      fixed: 'right',
      width: 150,
      render: (text, record) => (
        <Space>
          <Button
            icon={<Edit size={14} />}
            theme='light'
            type='tertiary'
            size='small'
            onClick={() => handleEditAnnouncement(record)}
          >
            {t('编辑')}
          </Button>
          <Button
            icon={<Trash2 size={14} />}
            type='danger'
            theme='light'
            size='small'
            onClick={() => handleDeleteAnnouncement(record)}
          >
            {t('删除')}
          </Button>
        </Space>
      ),
    },
  ];

  const updateOption = async (key, value) => {
    const res = await API.put('/api/option/', {
      key,
      value,
    });
    const { success, message } = res.data;
    if (success) {
      showSuccess('系统公告已更新');
      if (refresh) refresh();
    } else {
      showError(message);
    }
  };

  const submitAnnouncements = async () => {
    try {
      setLoading(true);
      const announcementsJson = JSON.stringify(announcementsList);
      await updateOption('console_setting.announcements', announcementsJson);
      setHasChanges(false);
    } catch (error) {
      console.error('系统公告更新失败', error);
      showError('系统公告更新失败');
    } finally {
      setLoading(false);
    }
  };

  const handleAddAnnouncement = () => {
    setEditingAnnouncement(null);
    setAnnouncementForm({
      content: '',
      content_en: '',
      publishDate: new Date(),
      type: 'default',
      extra: '',
      extra_en: '',
    });
    setAnnouncementLanguage('zh');
    setShowAnnouncementModal(true);
  };

  const handleEditAnnouncement = (announcement) => {
    setEditingAnnouncement(announcement);
    setAnnouncementForm({
      content: announcement.content,
      content_en: announcement.content_en || '',
      publishDate: announcement.publishDate
        ? new Date(announcement.publishDate)
        : new Date(),
      type: announcement.type || 'default',
      extra: announcement.extra || '',
      extra_en: announcement.extra_en || '',
    });
    setAnnouncementLanguage('zh');
    setShowAnnouncementModal(true);
  };

  const handleDeleteAnnouncement = (announcement) => {
    setDeletingAnnouncement(announcement);
    setShowDeleteModal(true);
  };

  const confirmDeleteAnnouncement = () => {
    if (deletingAnnouncement) {
      const newList = announcementsList.filter(
        (item) => item.id !== deletingAnnouncement.id,
      );
      setAnnouncementsList(newList);
      setHasChanges(true);
      showSuccess('公告已删除，请及时点击“保存设置”进行保存');
    }
    setShowDeleteModal(false);
    setDeletingAnnouncement(null);
  };

  const handleSaveAnnouncement = async () => {
    if (!announcementForm.content || !announcementForm.publishDate) {
      showError('请填写完整的公告信息');
      return;
    }

    try {
      setModalLoading(true);

      // 将publishDate转换为ISO字符串保存
      const formData = {
        ...announcementForm,
        publishDate: announcementForm.publishDate.toISOString(),
      };

      let newList;
      if (editingAnnouncement) {
        newList = announcementsList.map((item) =>
          item.id === editingAnnouncement.id ? { ...item, ...formData } : item,
        );
      } else {
        const newId =
          Math.max(...announcementsList.map((item) => item.id), 0) + 1;
        const newAnnouncement = {
          id: newId,
          ...formData,
        };
        newList = [...announcementsList, newAnnouncement];
      }

      setAnnouncementsList(newList);
      setHasChanges(true);
      setShowAnnouncementModal(false);
      showSuccess(
        editingAnnouncement
          ? '公告已更新，请及时点击“保存设置”进行保存'
          : '公告已添加，请及时点击“保存设置”进行保存',
      );
    } catch (error) {
      showError('操作失败: ' + error.message);
    } finally {
      setModalLoading(false);
    }
  };

  const loadTextModels = async () => {
    try {
      const res = await API.get('/api/user/models?scene=playground');
      if (!res.data?.success) {
        throw new Error(res.data?.message || t('加载模型信息失败'));
      }
      const raw = res.data?.data;
      const available = (Array.isArray(raw) ? raw : raw?.items || [])
        .map((item) =>
          typeof item === 'string' ? { model_name: item, tags: '' } : item,
        )
        .filter((item) => item?.model_name && isTextModel(item));
      setTextModels(available);
      setSelectedTextModel((current) =>
        available.some((item) => item.model_name === current)
          ? current
          : available[0]?.model_name || '',
      );
    } catch (error) {
      setTextModels([]);
      setSelectedTextModel('');
      showError(error.message || t('加载模型信息失败'));
    }
  };

  useEffect(() => {
    if (showAnnouncementModal) {
      loadTextModels();
      return;
    }
    translationAbortControllerRef.current?.abort();
    translationAbortControllerRef.current = null;
    setTranslationLoading(false);
  }, [showAnnouncementModal]);

  const runAnnouncementTranslation = async () => {
    const source = announcementForm.content.trim();
    if (!source) {
      showError(t('请先填写公告内容'));
      return;
    }
    if (!selectedTextModel) {
      showError(t('暂无可用文本模型'));
      return;
    }

    const original = announcementForm.content_en;
    const controller = new AbortController();
    translationAbortControllerRef.current = controller;
    setTranslationLoading(true);
    setAnnouncementLanguage('en');
    try {
      const result = await streamAnnouncementTranslation({
        signal: controller.signal,
        payload: {
          model: selectedTextModel,
          section: 'announcement',
          action: 'translate',
          document: source,
        },
        onProgress: (streamed) =>
          setAnnouncementForm((current) => ({
            ...current,
            content_en: normalizeTranslation(streamed),
          })),
      });
      if (!result) {
        throw new Error(t('文本模型未返回有效内容'));
      }
      setAnnouncementForm((current) => ({
        ...current,
        content_en: result,
      }));
      showSuccess(t('公告中译英完成'));
    } catch (error) {
      setAnnouncementForm((current) => ({
        ...current,
        content_en: original,
      }));
      if (error.name === 'AbortError') {
        showSuccess(t('已停止翻译，已恢复原文'));
      } else {
        showError(error.message || t('AI 处理失败'));
      }
    } finally {
      if (translationAbortControllerRef.current === controller) {
        translationAbortControllerRef.current = null;
      }
      setTranslationLoading(false);
    }
  };

  const stopAnnouncementTranslation = () => {
    translationAbortControllerRef.current?.abort();
  };

  const parseAnnouncements = (announcementsStr) => {
    if (!announcementsStr) {
      setAnnouncementsList([]);
      return;
    }

    try {
      const parsed = JSON.parse(announcementsStr);
      const list = Array.isArray(parsed) ? parsed : [];
      // 确保每个项目都有id
      const listWithIds = list.map((item, index) => ({
        ...item,
        id: item.id || index + 1,
      }));
      setAnnouncementsList(listWithIds);
    } catch (error) {
      console.error('解析系统公告失败:', error);
      setAnnouncementsList([]);
    }
  };

  useEffect(() => {
    const annStr =
      options['console_setting.announcements'] ?? options.Announcements;
    if (annStr !== undefined) {
      parseAnnouncements(annStr);
    }
  }, [options['console_setting.announcements'], options.Announcements]);

  useEffect(() => {
    const enabledStr = options['console_setting.announcements_enabled'];
    setPanelEnabled(
      enabledStr === undefined
        ? true
        : enabledStr === 'true' || enabledStr === true,
    );
  }, [options['console_setting.announcements_enabled']]);

  const handleToggleEnabled = async (checked) => {
    const newValue = checked ? 'true' : 'false';
    try {
      const res = await API.put('/api/option/', {
        key: 'console_setting.announcements_enabled',
        value: newValue,
      });
      if (res.data.success) {
        setPanelEnabled(checked);
        showSuccess(t('设置已保存'));
        refresh?.();
      } else {
        showError(res.data.message);
      }
    } catch (err) {
      showError(err.message);
    }
  };

  const handleBatchDelete = () => {
    if (selectedRowKeys.length === 0) {
      showError('请先选择要删除的系统公告');
      return;
    }

    const newList = announcementsList.filter(
      (item) => !selectedRowKeys.includes(item.id),
    );
    setAnnouncementsList(newList);
    setSelectedRowKeys([]);
    setHasChanges(true);
    showSuccess(
      `已删除 ${selectedRowKeys.length} 个系统公告，请及时点击“保存设置”进行保存`,
    );
  };

  const renderHeader = () => (
    <div className='flex flex-col w-full'>
      <div className='mb-2'>
        <div className='flex items-center text-blue-500'>
          <Bell size={16} className='mr-2' />
          <Text>
            {t(
              '系统公告管理，可以发布系统通知和重要消息（最多100个，前端显示最新20条）',
            )}
          </Text>
        </div>
      </div>

      <Divider margin='12px' />

      <div className='flex flex-col md:flex-row justify-between items-center gap-4 w-full'>
        <div className='flex gap-2 w-full md:w-auto order-2 md:order-1'>
          <Button
            theme='light'
            type='primary'
            icon={<Plus size={14} />}
            className='w-full md:w-auto'
            onClick={handleAddAnnouncement}
          >
            {t('添加公告')}
          </Button>
          <Button
            icon={<Trash2 size={14} />}
            type='danger'
            theme='light'
            onClick={handleBatchDelete}
            disabled={selectedRowKeys.length === 0}
            className='w-full md:w-auto'
          >
            {t('批量删除')}{' '}
            {selectedRowKeys.length > 0 && `(${selectedRowKeys.length})`}
          </Button>
          <Button
            icon={<Save size={14} />}
            onClick={submitAnnouncements}
            loading={loading}
            disabled={!hasChanges}
            type='secondary'
            className='w-full md:w-auto'
          >
            {t('保存设置')}
          </Button>
        </div>

        {/* 启用开关 */}
        <div className='order-1 md:order-2 flex items-center gap-2'>
          <Switch checked={panelEnabled} onChange={handleToggleEnabled} />
          <Text>{panelEnabled ? t('已启用') : t('已禁用')}</Text>
        </div>
      </div>
    </div>
  );

  // 计算当前页显示的数据（按发布时间倒序排序，最新优先显示）
  const getCurrentPageData = () => {
    const sortedList = [...announcementsList].sort((a, b) => {
      const dateA = new Date(a.publishDate).getTime();
      const dateB = new Date(b.publishDate).getTime();
      return dateB - dateA; // 倒序，最新的排在前面
    });

    const startIndex = (currentPage - 1) * pageSize;
    const endIndex = startIndex + pageSize;
    return sortedList.slice(startIndex, endIndex);
  };

  const rowSelection = {
    selectedRowKeys,
    onChange: (selectedRowKeys, selectedRows) => {
      setSelectedRowKeys(selectedRowKeys);
    },
    onSelect: (record, selected, selectedRows) => {
      console.log(`选择行: ${selected}`, record);
    },
    onSelectAll: (selected, selectedRows) => {
      console.log(`全选: ${selected}`, selectedRows);
    },
    getCheckboxProps: (record) => ({
      disabled: false,
      name: record.id,
    }),
  };

  return (
    <>
      <Form.Section text={renderHeader()}>
        <Table
          columns={columns}
          dataSource={getCurrentPageData()}
          rowSelection={rowSelection}
          rowKey='id'
          scroll={{ x: 'max-content' }}
          pagination={{
            currentPage: currentPage,
            pageSize: pageSize,
            total: announcementsList.length,
            showSizeChanger: true,
            showQuickJumper: true,
            pageSizeOptions: ['5', '10', '20', '50'],
            onChange: (page, size) => {
              setCurrentPage(page);
              setPageSize(size);
            },
            onShowSizeChange: (current, size) => {
              setCurrentPage(1);
              setPageSize(size);
            },
          }}
          size='middle'
          loading={loading}
          empty={
            <Empty
              image={
                <IllustrationNoResult style={{ width: 150, height: 150 }} />
              }
              darkModeImage={
                <IllustrationNoResultDark style={{ width: 150, height: 150 }} />
              }
              description={t('暂无系统公告')}
              style={{ padding: 30 }}
            />
          }
          className='overflow-hidden'
        />
      </Form.Section>

      <Modal
        title={editingAnnouncement ? t('编辑公告') : t('添加公告')}
        visible={showAnnouncementModal}
        onOk={handleSaveAnnouncement}
        onCancel={() => {
          translationAbortControllerRef.current?.abort();
          setShowAnnouncementModal(false);
        }}
        okText={t('保存')}
        cancelText={t('取消')}
        confirmLoading={modalLoading}
      >
        <Form
          layout='vertical'
          initValues={announcementForm}
          key={editingAnnouncement ? editingAnnouncement.id : 'new'}
        >
          <Form.Slot label={t('公告语言')}>
            <div className='flex flex-wrap items-center gap-2'>
              <Button
                type={announcementLanguage === 'zh' ? 'primary' : 'tertiary'}
                theme={announcementLanguage === 'zh' ? 'solid' : 'light'}
                disabled={translationLoading || modalLoading}
                onClick={() => setAnnouncementLanguage('zh')}
              >
                {t('中文公告')}
              </Button>
              <Button
                type={announcementLanguage === 'en' ? 'primary' : 'tertiary'}
                theme={announcementLanguage === 'en' ? 'solid' : 'light'}
                disabled={translationLoading || modalLoading}
                onClick={() => setAnnouncementLanguage('en')}
              >
                {t('英文公告')}
              </Button>
            </div>
          </Form.Slot>
          <Form.Slot label={t('AI 翻译')}>
            <div className='flex flex-wrap items-center gap-2'>
              <Select
                filter
                style={{ minWidth: 220, flex: '1 1 220px' }}
                value={selectedTextModel || undefined}
                placeholder={t('选择用于处理文档的文本模型')}
                emptyContent={t('暂无可用文本模型')}
                optionList={textModelOptions}
                disabled={translationLoading || modalLoading}
                onChange={setSelectedTextModel}
              />
              <Button
                type='tertiary'
                icon={<Languages size={16} />}
                loading={translationLoading}
                disabled={
                  !selectedTextModel || translationLoading || modalLoading
                }
                onClick={runAnnouncementTranslation}
              >
                {t('中译英')}
              </Button>
              {translationLoading ? (
                <Button
                  type='danger'
                  theme='light'
                  icon={<Square size={14} fill='currentColor' />}
                  onClick={stopAnnouncementTranslation}
                >
                  {t('停止翻译')}
                </Button>
              ) : null}
            </div>
          </Form.Slot>
          <Form.Slot
            label={
              announcementLanguage === 'en' ? t('公告英文内容') : t('公告内容')
            }
          >
            <TextArea
              key={`announcement-content-${announcementLanguage}`}
              value={
                announcementLanguage === 'en'
                  ? announcementForm.content_en
                  : announcementForm.content
              }
              placeholder={
                announcementLanguage === 'en'
                  ? t('请输入公告英文内容（支持 Markdown/HTML）')
                  : t('请输入公告内容（支持 Markdown/HTML）')
              }
              maxCount={1500}
              rows={8}
              style={{ width: '100%' }}
              disabled={translationLoading || modalLoading}
              onChange={(value) =>
                setAnnouncementForm((current) => ({
                  ...current,
                  [announcementLanguage === 'en' ? 'content_en' : 'content']:
                    value,
                }))
              }
            />
          </Form.Slot>
          <Button
            theme='light'
            type='tertiary'
            size='small'
            icon={<Maximize2 size={14} />}
            style={{ marginBottom: 16 }}
            onClick={() => setShowContentModal(true)}
            disabled={translationLoading || modalLoading}
          >
            {t('放大编辑')}
          </Button>
          <Form.DatePicker
            field='publishDate'
            label={t('发布日期')}
            type='dateTime'
            rules={[{ required: true, message: t('请选择发布日期') }]}
            onChange={(value) =>
              setAnnouncementForm({ ...announcementForm, publishDate: value })
            }
          />
          <Form.Select
            field='type'
            label={t('公告类型')}
            optionList={typeOptions}
            onChange={(value) =>
              setAnnouncementForm({ ...announcementForm, type: value })
            }
          />
          <Form.Slot label={t('说明信息')}>
            <Input
              key={`announcement-extra-${announcementLanguage}`}
              placeholder={
                announcementLanguage === 'en'
                  ? t('可选，公告的英文补充说明')
                  : t('可选，公告的补充说明')
              }
              value={
                announcementLanguage === 'en'
                  ? announcementForm.extra_en
                  : announcementForm.extra
              }
              disabled={translationLoading || modalLoading}
              onChange={(value) =>
                setAnnouncementForm((current) => ({
                  ...current,
                  [announcementLanguage === 'en' ? 'extra_en' : 'extra']: value,
                }))
              }
            />
          </Form.Slot>
        </Form>
      </Modal>

      <Modal
        title={t('确认删除')}
        visible={showDeleteModal}
        onOk={confirmDeleteAnnouncement}
        onCancel={() => {
          setShowDeleteModal(false);
          setDeletingAnnouncement(null);
        }}
        okText={t('确认删除')}
        cancelText={t('取消')}
        type='warning'
        okButtonProps={{
          type: 'danger',
          theme: 'solid',
        }}
      >
        <Text>{t('确定要删除此公告吗？')}</Text>
      </Modal>

      {/* 公告内容放大编辑 Modal */}
      <Modal
        title={
          announcementLanguage === 'en'
            ? t('编辑公告英文内容')
            : t('编辑公告内容')
        }
        visible={showContentModal}
        onOk={() => setShowContentModal(false)}
        onCancel={() => setShowContentModal(false)}
        okText={t('确定')}
        cancelText={t('取消')}
        width={800}
      >
        <TextArea
          value={
            announcementLanguage === 'en'
              ? announcementForm.content_en
              : announcementForm.content
          }
          placeholder={
            announcementLanguage === 'en'
              ? t('请输入公告英文内容（支持 Markdown/HTML）')
              : t('请输入公告内容（支持 Markdown/HTML）')
          }
          maxCount={1500}
          rows={15}
          style={{ width: '100%' }}
          disabled={translationLoading || modalLoading}
          onChange={(value) =>
            setAnnouncementForm((current) => ({
              ...current,
              [announcementLanguage === 'en' ? 'content_en' : 'content']: value,
            }))
          }
        />
      </Modal>
    </>
  );
};

export default SettingsAnnouncements;
