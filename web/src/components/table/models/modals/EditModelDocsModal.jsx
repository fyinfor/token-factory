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

import React, { useEffect, useMemo, useState } from 'react';
import {
  Avatar,
  Button,
  Card,
  Col,
  Form,
  Input,
  Modal,
  Row,
  Select,
  SideSheet,
  Space,
  Spin,
  Tabs,
  Tag,
  TextArea,
  Tooltip,
  Typography,
} from '@douyinfe/semi-ui';
import { IconDelete, IconHelpCircle, IconPlus } from '@douyinfe/semi-icons';
import { FileText, Save, X } from 'lucide-react';
import { API, showError, showSuccess } from '../../../../helpers';
import MarkdownRenderer from '../../../common/markdown/MarkdownRenderer';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';

const { Text, Title } = Typography;

const createId = () =>
  `${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 8)}`;

const emptyParam = () => ({
  id: createId(),
  name: '',
  type: 'string',
  example: '',
  description: '',
  required: false,
  children: [],
});

const emptyApi = () => ({
  id: createId(),
  description: '',
  path: '/v1/chat/completions',
  method: 'POST',
  detail: '',
  body_params: [],
  path_params: [],
  response_params: [],
});

const parseApiDocs = (value) => {
  if (!value) return [];
  try {
    const parsed = typeof value === 'string' ? JSON.parse(value) : value;
    return Array.isArray(parsed) ? parsed : [];
  } catch (_) {
    return [];
  }
};

const ensureParamIds = (params = []) =>
  (Array.isArray(params) ? params : []).map((param) => ({
    ...param,
    id: param.id || createId(),
    children: ensureParamIds(param.children),
  }));

const ensureApiDocIds = (docs = []) =>
  (Array.isArray(docs) ? docs : []).map((api) => ({
    ...api,
    id: api.id || createId(),
    body_params: ensureParamIds(api.body_params),
    path_params: ensureParamIds(api.path_params),
    response_params: ensureParamIds(api.response_params),
  }));

const MarkdownEditor = ({ label, value, onChange, placeholder, t }) => (
  <Form.Slot label={label}>
    <style>
      {`
        .api-docs-markdown-editor-card > .semi-card-body {
          padding-top: 0 !important;
        }
      `}
    </style>
    <Card
      className='!rounded-2xl api-docs-markdown-editor-card'
      bodyStyle={{ padding: 12 }}
    >
      <Tabs type='line'>
        <Tabs.TabPane tab={t('编辑')} itemKey='edit' className='mt-2'>
          <TextArea
            autosize={{ minRows: 5, maxRows: 14 }}
            value={value || ''}
            placeholder={placeholder}
            onChange={onChange}
            showClear
          />
        </Tabs.TabPane>
        <Tabs.TabPane tab={t('预览')} itemKey='preview' className='mt-2'>
          {String(value || '').trim() ? (
            <MarkdownRenderer content={value} />
          ) : (
            <Text type='tertiary'>{t('暂无内容')}</Text>
          )}
        </Tabs.TabPane>
      </Tabs>
    </Card>
  </Form.Slot>
);

const updateParamTree = (params, id, updater) =>
  params.map((param) => {
    if (param.id === id) {
      const next = typeof updater === 'function' ? updater(param) : updater;
      return { ...param, ...next };
    }
    return {
      ...param,
      children: updateParamTree(param.children || [], id, updater),
    };
  });

const removeParamFromTree = (params, id) =>
  params
    .filter((param) => param.id !== id)
    .map((param) => ({
      ...param,
      children: removeParamFromTree(param.children || [], id),
    }));

const addChildToParamTree = (params, parentId) =>
  params.map((param) => {
    if (param.id === parentId) {
      return {
        ...param,
        children: [...(param.children || []), emptyParam()],
      };
    }
    return {
      ...param,
      children: addChildToParamTree(param.children || [], parentId),
    };
  });

const ParamRow = ({
  param,
  depth,
  onPatch,
  onRemove,
  onAddChild,
  showRequired,
  t,
}) => {
  const canNest = param.type === 'object' || param.type === 'array';
  return (
    <div
      className='rounded-lg border p-3'
      style={{
        borderColor: 'var(--semi-color-border)',
        marginLeft: depth > 0 ? 18 : 0,
        backgroundColor:
          depth > 0 ? 'var(--semi-color-fill-0)' : 'var(--semi-color-bg-0)',
      }}
    >
      <Row gutter={8} align='middle'>
        <Col span={7}>
          <Input
            value={param.name}
            placeholder={depth > 0 ? t('字段名') : t('字段名或字段路径')}
            onChange={(name) => onPatch(param.id, { name })}
          />
        </Col>
        <Col span={5}>
          <Select
            value={param.type || 'string'}
            style={{ width: '100%' }}
            optionList={[
              'string',
              'number',
              'integer',
              'boolean',
              'object',
              'array',
              'null',
            ].map((item) => ({ label: item, value: item }))}
            onChange={(type) =>
              onPatch(param.id, (current) => ({
                type,
                children:
                  type === 'object' || type === 'array'
                    ? current.children || []
                    : [],
              }))
            }
          />
        </Col>
        <Col span={showRequired ? 7 : 10}>
          <Input
            value={param.example}
            placeholder={canNest ? t('容器本身可留空') : t('示例值')}
            suffix={
              <Tooltip
                content={
                  <div style={{ whiteSpace: 'pre-wrap' }}>
                    {t('可使用以下占位符自动替换：\n<model_name>：模型名称')}
                  </div>
                }
                showArrow
              >
                <IconHelpCircle className='mr-2' style={{ color: 'var(--semi-color-text-2)' }} />
              </Tooltip>
            }
            onChange={(example) => onPatch(param.id, { example })}
          />
        </Col>
        {showRequired ? (
          <Col span={3}>
            <Select
              value={param.required ? 1 : 0}
              style={{ width: '100%' }}
              optionList={[
                { label: t('必填'), value: 1 },
                { label: t('选填'), value: 0 },
              ]}
              onChange={(required) =>
                onPatch(param.id, { required: required === 1 })
              }
            />
          </Col>
        ) : null}
        <Col span={2}>
          <Space spacing={2}>
            {canNest ? (
              <Button
                icon={<IconPlus />}
                type='tertiary'
                theme='borderless'
                onClick={() => onAddChild(param.id)}
                title={t('添加子参数')}
              />
            ) : null}
            <Button
              icon={<IconDelete />}
              type='danger'
              theme='borderless'
              onClick={() => onRemove(param.id)}
              title={t('删除')}
            />
          </Space>
        </Col>
        <Col span={24} className='mt-2'>
          <TextArea
            autosize={{ minRows: 2, maxRows: 8 }}
            value={param.description}
            placeholder={t('参数说明，支持多行填写')}
            style={{ width: '100%' }}
            onChange={(description) => onPatch(param.id, { description })}
          />
        </Col>
      </Row>
      {canNest ? (
        <div className='mt-3 space-y-2'>
          {(param.children || []).length > 0 ? (
            (param.children || []).map((child) => (
              <ParamRow
                key={child.id}
                param={child}
                depth={depth + 1}
                onPatch={onPatch}
                onRemove={onRemove}
                onAddChild={onAddChild}
                showRequired={showRequired}
                t={t}
              />
            ))
          ) : (
            <Button
              size='small'
              type='tertiary'
              icon={<IconPlus />}
              onClick={() => onAddChild(param.id)}
            >
              {param.type === 'array' ? t('添加数组元素结构') : t('添加子参数')}
            </Button>
          )}
        </div>
      ) : null}
    </div>
  );
};

const ParamEditor = ({
  title,
  value = [],
  onChange,
  showRequired = true,
  t,
}) => {
  const params = Array.isArray(value) ? value : [];

  const patchParam = (id, patch) => {
    onChange(updateParamTree(params, id, patch));
  };

  const removeParam = (id) => {
    onChange(removeParamFromTree(params, id));
  };

  const addChildParam = (id) => {
    onChange(addChildToParamTree(params, id));
  };

  return (
    <Card
      className='!rounded-2xl mb-3'
      title={title}
      headerExtraContent={
        <Button
          size='small'
          icon={<IconPlus />}
          onClick={() => onChange([...params, emptyParam()])}
        >
          {t('添加参数')}
        </Button>
      }
      bodyStyle={{ padding: 12 }}
    >
      {params.length === 0 ? (
        <div className='text-center py-4'>
          <Text type='tertiary'>{t('暂无参数')}</Text>
        </div>
      ) : (
        <div className='space-y-2'>
          {params.map((param) => (
            <ParamRow
              key={param.id}
              param={param}
              depth={0}
              onPatch={patchParam}
              onRemove={removeParam}
              onAddChild={addChildParam}
              showRequired={showRequired}
              t={t}
            />
          ))}
        </div>
      )}
    </Card>
  );
};

const EditModelDocsModal = ({ visible, editingModel, onClose, refresh, t }) => {
  const isMobile = useIsMobile();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [modelDetail, setModelDetail] = useState(null);
  const [docIntroduction, setDocIntroduction] = useState('');
  const [apis, setApis] = useState([]);
  const [templateModels, setTemplateModels] = useState([]);

  const modelId = editingModel?.id;

  const templateOptions = useMemo(
    () =>
      templateModels
        .filter((item) => item.id !== modelId && item.api_docs)
        .map((item) => ({
          label: item.model_name,
          value: item.id,
        })),
    [templateModels, modelId],
  );

  const loadTemplateModels = async () => {
    try {
      const res = await API.get('/api/models/?p=1&page_size=1000');
      if (res.data?.success) {
        const items = res.data.data?.items || [];
        setTemplateModels(Array.isArray(items) ? items : []);
      }
    } catch (_) {
      setTemplateModels([]);
    }
  };

  const loadModel = async () => {
    if (!visible || !modelId) return;
    setLoading(true);
    try {
      const res = await API.get(`/api/models/${modelId}`);
      const { success, message, data } = res.data || {};
      if (!success) {
        showError(message || t('加载模型信息失败'));
        return;
      }
      setModelDetail(data);
      setDocIntroduction(data?.doc_introduction || '');
      setApis(ensureApiDocIds(parseApiDocs(data?.api_docs)));
    } catch (e) {
      showError(t('加载模型信息失败'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (visible) {
      loadModel();
      loadTemplateModels();
    } else {
      setModelDetail(null);
      setDocIntroduction('');
      setApis([]);
    }
  }, [visible, modelId]);

  const applyTemplate = async (id) => {
    const picked = templateModels.find((item) => item.id === id);
    if (!picked) return;
    Modal.confirm({
      title: t('确认套用文档配置？'),
      content: t('当前编辑内容将被所选模型的文档配置覆盖。'),
      onOk: async () => {
        let detail = picked;
        try {
          const res = await API.get(`/api/models/${id}`);
          if (res.data?.success) {
            detail = res.data.data;
          }
        } catch (_) {}
        setDocIntroduction(detail?.doc_introduction || '');
        setApis(ensureApiDocIds(parseApiDocs(detail?.api_docs)));
      },
    });
  };

  const patchApi = (id, patch) => {
    setApis((prev) =>
      prev.map((item) => (item.id === id ? { ...item, ...patch } : item)),
    );
  };

  const removeApi = (id) => {
    setApis((prev) => prev.filter((item) => item.id !== id));
  };

  const save = async () => {
    if (!modelDetail?.id) return;
    setSaving(true);
    try {
      const payload = {
        id: modelDetail.id,
        doc_introduction: docIntroduction || '',
        api_docs: apis.length > 0 ? JSON.stringify(apis, null, 2) : '',
      };
      const res = await API.put('/api/models/?docs_only=true', payload);
      const { success, message } = res.data || {};
      if (!success) {
        showError(message || t('保存失败'));
        return;
      }
      showSuccess(t('模型文档保存成功！'));
      refresh?.();
      onClose?.();
    } catch (e) {
      showError(e.response?.data?.message || t('保存失败'));
    } finally {
      setSaving(false);
    }
  };

  return (
    <SideSheet
      placement='right'
      visible={visible}
      width={isMobile ? '100%' : 840}
      title={
        <Space>
          <Tag color='cyan' shape='circle'>
            {t('文档')}
          </Tag>
          <Title heading={4} className='m-0'>
            {editingModel?.model_name || t('模型文档')}
          </Title>
        </Space>
      }
      bodyStyle={{ padding: 0 }}
      closeIcon={null}
      onCancel={onClose}
      footer={
        <div className='flex justify-end'>
          <Space>
            <Button icon={<Save size={16} />} loading={saving} onClick={save}>
              {t('保存')}
            </Button>
            <Button icon={<X size={16} />} type='tertiary' onClick={onClose}>
              {t('取消')}
            </Button>
          </Space>
        </div>
      }
    >
      <Spin spinning={loading}>
        <div className='p-3'>
          <Card className='!rounded-2xl shadow-sm border-0 mb-3'>
            <div className='flex items-center mb-3'>
              <Avatar size='small' color='cyan' className='mr-2 shadow-md'>
                <FileText size={16} />
              </Avatar>
              <div>
                <Text className='text-lg font-medium'>{t('文档配置')}</Text>
                <div className='text-xs text-gray-600'>
                  {t('配置后将在模型广场的通道文档中展示')}
                </div>
              </div>
            </div>
            <Form>
              <Form.Slot label={t('套用其他模型文档')}>
                <Select
                  filter
                  showClear
                  style={{ width: '100%' }}
                  placeholder={t('搜索模型名称并套用已有文档')}
                  optionList={templateOptions}
                  onChange={applyTemplate}
                />
              </Form.Slot>
              <MarkdownEditor
                label={t('模型介绍')}
                value={docIntroduction}
                onChange={setDocIntroduction}
                placeholder={t('可填写 Markdown 格式的模型介绍')}
                t={t}
              />
            </Form>
          </Card>

          <div className='flex justify-between items-center my-2'>
            <Text strong>{t('API 列表')}</Text>
            <Button
              icon={<IconPlus />}
              onClick={() => setApis([...apis, emptyApi()])}
            >
              {t('添加 API')}
            </Button>
          </div>

          {apis.length === 0 ? (
            <Card className='!rounded-2xl text-center py-6'>
              <Text type='tertiary'>{t('暂无 API 文档')}</Text>
            </Card>
          ) : (
            <div className='space-y-3'>
              {apis.map((api, index) => (
                <Card
                  key={api.id}
                  className='!rounded-2xl'
                  title={`${t('API')} #${index + 1}`}
                  headerExtraContent={
                    <Button
                      type='danger'
                      theme='borderless'
                      icon={<IconDelete />}
                      onClick={() => removeApi(api.id)}
                    />
                  }
                >
                  <Row gutter={12}>
                    <Col span={24}>
                      <TextArea
                        autosize={{ minRows: 2, maxRows: 6 }}
                        value={api.description}
                        placeholder={t('API 描述，支持多行填写')}
                        style={{ width: '100%' }}
                        onChange={(description) =>
                          patchApi(api.id, { description })
                        }
                      />
                    </Col>
                    <Col span={16} className='mt-2'>
                      <Input
                        value={api.path}
                        placeholder={t('接口路径，如 /v1/chat/completions')}
                        onChange={(path) => patchApi(api.id, { path })}
                      />
                    </Col>
                    <Col span={8} className='mt-2'>
                      <Select
                        value={api.method || 'POST'}
                        style={{ width: '100%' }}
                        optionList={[
                          { label: 'POST', value: 'POST' },
                          { label: 'GET', value: 'GET' },
                        ]}
                        onChange={(method) => patchApi(api.id, { method })}
                      />
                    </Col>
                    <Col span={24} className='mt-2'>
                      <MarkdownEditor
                        label={t('接口详情')}
                        value={api.detail}
                        onChange={(detail) => patchApi(api.id, { detail })}
                        placeholder={t('可填写 Markdown 格式的接口说明')}
                        t={t}
                      />
                    </Col>
                    <Col span={24}>
                      {(api.method || 'POST') === 'POST' ? (
                        <ParamEditor
                          title={t('Body 参数')}
                          value={api.body_params}
                          onChange={(body_params) =>
                            patchApi(api.id, { body_params })
                          }
                          t={t}
                        />
                      ) : (
                        <ParamEditor
                          title={t('Path 参数')}
                          value={api.path_params}
                          onChange={(path_params) =>
                            patchApi(api.id, { path_params })
                          }
                          t={t}
                        />
                      )}
                      <ParamEditor
                        title={t('返回数据结构')}
                        value={api.response_params}
                        onChange={(response_params) =>
                          patchApi(api.id, { response_params })
                        }
                        showRequired={false}
                        t={t}
                      />
                    </Col>
                  </Row>
                </Card>
              ))}
            </div>
          )}
        </div>
      </Spin>
    </SideSheet>
  );
};

export default EditModelDocsModal;
