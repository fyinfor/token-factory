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
  Avatar,
  Banner,
  Button,
  Card,
  Form,
  Modal,
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
import {
  Braces,
  FileText,
  Languages,
  RotateCcw,
  Save,
  Settings2,
  Sparkles,
  Square,
  Upload,
  X,
} from 'lucide-react';
import {
  API,
  getUserIdFromLocalStorage,
  showError,
  showSuccess,
} from '../../../../helpers';
import ApiMarkdownRenderer from '../../../common/markdown/ApiMarkdownRenderer';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import {
  isValidPolishedMarkdown,
  readAssistantContent,
  readAssistantStreamChunk,
  stripOuterMarkdownFence,
  stripReasoningContent,
} from './documentAiUtils';

const { Text, Title } = Typography;

const MAX_IMPORT_FILE_SIZE = 2 * 1024 * 1024;
const MAX_AI_DOCUMENT_LENGTH = 120000;

const normalizeAssistantMarkdown = (content) =>
  stripOuterMarkdownFence(stripReasoningContent(content));

const streamDocumentCompletion = async ({ payload, signal, onProgress }) => {
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
    const data = await response.json();
    return readAssistantContent({ data });
  }
  if (!response.body) throw new Error('Streaming response body is unavailable');

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
  return normalizeAssistantMarkdown(streamed);
};

const HTTP_TEMPLATE = `## 创建任务

\`\`\`http
POST /v1/video/generations
Content-Type: application/json
Authorization: Bearer {{api_key}}

{
  "model": "{{model}}",
  "prompt": "海边日落",
  "duration": 5
}
\`\`\`

## 查询任务

请在这里补充轮询流程、状态判断和结果处理。
`;

const getChannelDocKey = (item) =>
  `${item?.channel_id || ''}:${item?.model_name || ''}`;

const normalizeTagList = (value) =>
  String(value || '')
    .replaceAll('，', ',')
    .replaceAll('、', ',')
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean);

const isTextModel = (item) => {
  const tags = normalizeTagList(item?.tags);
  return tags.length === 0 || tags.includes('文本');
};

const parseLegacyApiDocs = (value) => {
  if (!value) return [];
  try {
    const parsed = typeof value === 'string' ? JSON.parse(value) : value;
    return Array.isArray(parsed) ? parsed : [];
  } catch (_) {
    return [];
  }
};

const legacyParamValue = (param, modelName) => {
  if (param?.example !== undefined && param?.example !== '') {
    return param.example === '<model_name>' ? '{{model}}' : param.example;
  }
  if (param?.name === 'model') return '{{model}}';
  if (param?.type === 'boolean') return false;
  if (param?.type === 'integer' || param?.type === 'number') return 0;
  if (param?.type === 'array') return [];
  if (param?.type === 'object') return {};
  return param?.name === 'model' ? modelName || '{{model}}' : '';
};

const buildLegacyObject = (params, modelName) => {
  const result = {};
  (Array.isArray(params) ? params : []).forEach((param) => {
    if (!param?.name) return;
    if (Array.isArray(param.children) && param.children.length > 0) {
      const child = buildLegacyObject(param.children, modelName);
      result[param.name] = param.type === 'array' ? [child] : child;
      return;
    }
    result[param.name] = legacyParamValue(param, modelName);
  });
  return result;
};

const legacyParamsTable = (title, params) => {
  const rows = (Array.isArray(params) ? params : []).filter(
    (param) => param?.name,
  );
  if (rows.length === 0) return '';
  return [
    `### ${title}`,
    '',
    '| 参数 | 类型 | 必填 | 说明 |',
    '| --- | --- | --- | --- |',
    ...rows.map(
      (param) =>
        `| \`${param.name}\` | ${param.type || 'string'} | ${param.required ? '是' : '否'} | ${String(param.description || '').replaceAll('|', '\\|')} |`,
    ),
  ].join('\n');
};

const legacyApiDocsToMarkdown = (raw, modelName) => {
  const docs = parseLegacyApiDocs(raw);
  return docs
    .map((api, index) => {
      const method = String(api?.method || 'POST').toUpperCase();
      const path = String(api?.path || '').replaceAll(
        '<model_name>',
        '{{model}}',
      );
      const body = buildLegacyObject(api?.body_params, modelName);
      const requestBlock =
        method === 'GET'
          ? `\`\`\`http\nGET ${path}\nAuthorization: Bearer {{api_key}}\n\`\`\``
          : `\`\`\`http\n${method} ${path}\nContent-Type: application/json\nAuthorization: Bearer {{api_key}}\n\n${JSON.stringify(body, null, 2)}\n\`\`\``;
      return [
        `## ${api?.description || `API ${index + 1}`}`,
        '',
        api?.detail || '',
        '',
        requestBlock,
        '',
        legacyParamsTable(
          method === 'GET' ? '请求参数' : 'Body 参数',
          method === 'GET' ? api?.query_params : api?.body_params,
        ),
        '',
        legacyParamsTable('返回字段', api?.response_params),
      ]
        .filter((part) => part !== '')
        .join('\n\n');
    })
    .join('\n\n---\n\n');
};

const MarkdownEditor = ({
  value,
  onChange,
  placeholder,
  minRows = 6,
  disabled = false,
  t,
}) => (
  <Card className='!rounded-lg border' bodyStyle={{ padding: 12 }}>
    <Tabs type='line'>
      <Tabs.TabPane tab={t('编辑')} itemKey='edit' className='mt-2'>
        <TextArea
          autosize={{ minRows, maxRows: Math.max(minRows, 30) }}
          value={value || ''}
          placeholder={placeholder}
          onChange={onChange}
          disabled={disabled}
        />
      </Tabs.TabPane>
      <Tabs.TabPane tab={t('预览')} itemKey='preview' className='mt-2'>
        {String(value || '').trim() ? (
          <ApiMarkdownRenderer content={value} t={t} showToc />
        ) : (
          <Text type='tertiary'>{t('暂无内容')}</Text>
        )}
      </Tabs.TabPane>
    </Tabs>
  </Card>
);

const EditModelDocsModal = ({ visible, editingModel, onClose, refresh, t }) => {
  const isMobile = useIsMobile();
  const fileInputRef = useRef(null);
  const aiAbortControllerRef = useRef(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [aiAction, setAiAction] = useState('');
  const [isDraggingFiles, setIsDraggingFiles] = useState(false);
  const [documentLanguage, setDocumentLanguage] = useState('zh');
  const [modelDetail, setModelDetail] = useState(null);
  const [channelDocs, setChannelDocs] = useState([]);
  const [selectedChannelDocKey, setSelectedChannelDocKey] = useState('');
  const [docIntroduction, setDocIntroduction] = useState('');
  const [apiDocsMarkdown, setApiDocsMarkdown] = useState('');
  const [apiDocsMarkdownEn, setApiDocsMarkdownEn] = useState('');
  const [templateModels, setTemplateModels] = useState([]);
  const [templateChannelDocs, setTemplateChannelDocs] = useState([]);
  const [textModels, setTextModels] = useState([]);
  const [selectedTextModel, setSelectedTextModel] = useState('');
  const [promptSettingsVisible, setPromptSettingsVisible] = useState(false);
  const [promptSettingsLoading, setPromptSettingsLoading] = useState(false);
  const [promptSettingsSaving, setPromptSettingsSaving] = useState(false);
  const [promptSettings, setPromptSettings] = useState({
    polish_prompt: '',
    translate_prompt: '',
    is_default: true,
  });

  const modelId = editingModel?.id;

  const selectedChannelDoc = useMemo(
    () =>
      channelDocs.find(
        (item) => getChannelDocKey(item) === selectedChannelDocKey,
      ),
    [channelDocs, selectedChannelDocKey],
  );

  const channelDocOptions = useMemo(
    () =>
      channelDocs.map((item) => ({
        value: getChannelDocKey(item),
        label: `${item.channel_name ? `${item.channel_name} (#${item.channel_id})` : `#${item.channel_id}`} / ${item.model_name}${item.channel_status === 1 ? '' : ` [${t('被禁用')}]`}`,
      })),
    [channelDocs, t],
  );

  const templateOptions = useMemo(
    () => [
      ...templateChannelDocs
        .filter(
          (item) =>
            getChannelDocKey(item) !== selectedChannelDocKey &&
            (item.doc_introduction ||
              item.api_docs ||
              item.api_docs_markdown ||
              item.api_docs_markdown_en),
        )
        .map((item) => ({
          label: `${item.model_name}（${item.channel_name || item.route_slug || item.channel_no || `#${item.channel_id}`}）`,
          value: `channel:${item.id}`,
        })),
      ...templateModels
        .filter(
          (item) =>
            item.id !== modelId && (item.doc_introduction || item.api_docs),
        )
        .map((item) => ({
          label: item.model_name,
          value: `model:${item.id}`,
        })),
    ],
    [modelId, selectedChannelDocKey, templateChannelDocs, templateModels],
  );

  const textModelOptions = useMemo(
    () =>
      textModels.map((item) => ({
        label: item.model_name,
        value: item.model_name,
      })),
    [textModels],
  );

  const applyDocItem = (item, targetModelName = item?.model_name) => {
    const rawLegacy = item?.api_docs || '';
    setDocIntroduction(item?.doc_introduction || '');
    setApiDocsMarkdown(
      item?.api_docs_markdown ||
        legacyApiDocsToMarkdown(rawLegacy, targetModelName),
    );
    setApiDocsMarkdownEn(item?.api_docs_markdown_en || '');
    setDocumentLanguage('zh');
  };

  const loadModel = async (preferredKey = '') => {
    if (!visible || !modelId) return;
    setLoading(true);
    try {
      const res = await API.get(`/api/models/${modelId}/channel_docs`);
      const { success, message, data } = res.data || {};
      if (!success) {
        showError(message || t('加载模型信息失败'));
        return;
      }
      const items = Array.isArray(data?.items) ? data.items : [];
      const first =
        items.find((item) => getChannelDocKey(item) === preferredKey) ||
        items[0];
      setModelDetail({ id: data?.model_id, model_name: data?.model_name });
      setChannelDocs(items);
      setSelectedChannelDocKey(first ? getChannelDocKey(first) : '');
      applyDocItem(first);
    } catch (_) {
      showError(t('加载模型信息失败'));
    } finally {
      setLoading(false);
    }
  };

  const loadSupportingData = async () => {
    const [modelsResult, channelTemplatesResult, textModelsResult] =
      await Promise.allSettled([
        API.get('/api/models/?p=1&page_size=1000'),
        API.get('/api/models/channel_doc_templates'),
        API.get('/api/user/models?scene=playground'),
      ]);
    if (modelsResult.status === 'fulfilled') {
      const items = modelsResult.value.data?.data?.items || [];
      setTemplateModels(Array.isArray(items) ? items : []);
    } else {
      setTemplateModels([]);
    }
    if (
      channelTemplatesResult.status === 'fulfilled' &&
      channelTemplatesResult.value.data?.success
    ) {
      const items = channelTemplatesResult.value.data?.data;
      setTemplateChannelDocs(Array.isArray(items) ? items : []);
    } else {
      setTemplateChannelDocs([]);
    }
    if (
      textModelsResult.status === 'fulfilled' &&
      textModelsResult.value.data?.success
    ) {
      const raw = textModelsResult.value.data?.data;
      const items = Array.isArray(raw) ? raw : raw?.items || [];
      const available = items
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
    } else {
      setTextModels([]);
      setSelectedTextModel('');
    }
  };

  useEffect(() => {
    if (visible) {
      loadModel();
      loadSupportingData();
      return;
    }
    setModelDetail(null);
    setChannelDocs([]);
    setSelectedChannelDocKey('');
    setDocIntroduction('');
    setApiDocsMarkdown('');
    setApiDocsMarkdownEn('');
    setTemplateChannelDocs([]);
    setPromptSettingsVisible(false);
    aiAbortControllerRef.current?.abort();
    aiAbortControllerRef.current = null;
    setAiAction('');
    setIsDraggingFiles(false);
  }, [visible, modelId]);

  const selectChannelDoc = (key) => {
    const item = channelDocs.find((doc) => getChannelDocKey(doc) === key);
    setSelectedChannelDocKey(key || '');
    applyDocItem(item);
  };

  const applyTemplate = async (templateKey) => {
    const [kind, rawId] = String(templateKey || '').split(':');
    const id = Number(rawId);
    const picked =
      kind === 'channel'
        ? templateChannelDocs.find((item) => item.id === id)
        : templateModels.find((item) => item.id === id);
    if (!picked) return;
    Modal.confirm({
      title: t('确认套用文档配置？'),
      content: t('当前编辑内容将被所选模型的文档配置覆盖。'),
      onOk: async () => {
        if (kind === 'channel') {
          applyDocItem(picked, selectedChannelDoc?.model_name);
          return;
        }
        let detail = picked;
        try {
          const res = await API.get(`/api/models/${id}`);
          if (res.data?.success) detail = res.data.data;
        } catch (_) {}
        const raw = detail?.api_docs || '';
        setDocIntroduction(detail?.doc_introduction || '');
        setApiDocsMarkdown(
          legacyApiDocsToMarkdown(raw, selectedChannelDoc?.model_name),
        );
        setApiDocsMarkdownEn('');
        setDocumentLanguage('zh');
      },
    });
  };

  const activeMarkdown =
    documentLanguage === 'en' ? apiDocsMarkdownEn : apiDocsMarkdown;
  const setActiveMarkdown =
    documentLanguage === 'en' ? setApiDocsMarkdownEn : setApiDocsMarkdown;

  const importDocuments = async (fileList) => {
    if (aiAction) return;
    const files = Array.from(fileList || []);
    if (files.length === 0) return;
    const unsupported = files.find(
      (file) => !/\.(md|markdown|txt)$/i.test(file.name),
    );
    if (unsupported) {
      showError(t('仅支持上传 Markdown 或 TXT 文件'));
      return;
    }
    const oversized = files.find((file) => file.size > MAX_IMPORT_FILE_SIZE);
    if (oversized) {
      showError(t('单个文档不能超过 2 MB'));
      return;
    }
    try {
      const contents = await Promise.all(files.map((file) => file.text()));
      const imported = contents
        .map((content, index) => {
          const value = String(content || '').trim();
          if (!value) return '';
          if (files.length === 1) return value;
          const title = files[index].name.replace(/\.(md|markdown|txt)$/i, '');
          return `# ${title}\n\n${value}`;
        })
        .filter(Boolean)
        .join('\n\n---\n\n');
      if (!imported) {
        showError(t('导入的文档没有内容'));
        return;
      }
      setActiveMarkdown((current) =>
        current.trim() ? `${current.trim()}\n\n---\n\n${imported}` : imported,
      );
      showSuccess(t('已导入 {{count}} 个文档', { count: files.length }));
    } catch (_) {
      showError(t('读取文档失败'));
    }
  };

  const handleFileInputChange = (event) => {
    const files = Array.from(event.target.files || []);
    event.target.value = '';
    importDocuments(files);
  };

  const handleDocumentDrop = (event) => {
    event.preventDefault();
    setIsDraggingFiles(false);
    importDocuments(event.dataTransfer?.files);
  };

  const insertHttpTemplate = () => {
    setActiveMarkdown((current) =>
      current.trim()
        ? `${current.trim()}\n\n---\n\n${HTTP_TEMPLATE}`
        : HTTP_TEMPLATE,
    );
  };

  const applyPromptSettingsResponse = (data) => {
    setPromptSettings({
      polish_prompt: data?.polish_prompt || '',
      translate_prompt: data?.translate_prompt || '',
      is_default: Boolean(data?.is_default),
    });
  };

  const openPromptSettings = async () => {
    setPromptSettingsVisible(true);
    setPromptSettingsLoading(true);
    try {
      const res = await API.get('/api/models/document_ai/prompts');
      if (!res.data?.success) {
        throw new Error(res.data?.message || t('加载提示词失败'));
      }
      applyPromptSettingsResponse(res.data.data);
    } catch (error) {
      setPromptSettingsVisible(false);
      showError(error.response?.data?.message || error.message);
    } finally {
      setPromptSettingsLoading(false);
    }
  };

  const savePromptSettings = async () => {
    const polishPrompt = promptSettings.polish_prompt.trim();
    const translatePrompt = promptSettings.translate_prompt.trim();
    if (!polishPrompt || !translatePrompt) {
      showError(t('润色和翻译提示词不能为空'));
      return;
    }
    setPromptSettingsSaving(true);
    try {
      const res = await API.put('/api/models/document_ai/prompts', {
        polish_prompt: polishPrompt,
        translate_prompt: translatePrompt,
      });
      if (!res.data?.success) {
        throw new Error(res.data?.message || t('保存提示词失败'));
      }
      applyPromptSettingsResponse(res.data.data);
      setPromptSettingsVisible(false);
      showSuccess(t('提示词已保存'));
    } catch (error) {
      showError(error.response?.data?.message || error.message);
    } finally {
      setPromptSettingsSaving(false);
    }
  };

  const resetPromptSettings = () => {
    Modal.confirm({
      title: t('恢复默认提示词？'),
      content: t('当前自定义提示词将被清除，并立即恢复为后端默认内容。'),
      onOk: async () => {
        setPromptSettingsSaving(true);
        try {
          const res = await API.delete('/api/models/document_ai/prompts');
          if (!res.data?.success) {
            throw new Error(res.data?.message || t('恢复默认提示词失败'));
          }
          applyPromptSettingsResponse(res.data.data);
          showSuccess(t('已恢复默认提示词'));
        } catch (error) {
          showError(error.response?.data?.message || error.message);
          throw error;
        } finally {
          setPromptSettingsSaving(false);
        }
      },
    });
  };

  const runAiAction = async (action) => {
    const content =
      action === 'translate' ? apiDocsMarkdown.trim() : activeMarkdown.trim();
    if (!content) {
      showError(t('请先填写或导入 API 文档'));
      return;
    }
    if (!selectedTextModel) {
      showError(t('暂无可用文本模型'));
      return;
    }
    if (content.length > MAX_AI_DOCUMENT_LENGTH) {
      showError(t('文档过长，请拆分后再进行 AI 处理'));
      return;
    }
    const targetOriginal =
      action === 'translate' ? apiDocsMarkdownEn : activeMarkdown;
    const updateTarget =
      action === 'translate' ? setApiDocsMarkdownEn : setActiveMarkdown;
    const controller = new AbortController();
    aiAbortControllerRef.current = controller;
    setAiAction(action);
    if (action === 'translate') setDocumentLanguage('en');
    try {
      const result = await streamDocumentCompletion({
        signal: controller.signal,
        onProgress: (streamed) => {
          updateTarget(normalizeAssistantMarkdown(streamed));
        },
        payload: {
          model: selectedTextModel,
          action,
          document: content,
        },
      });
      if (!result) {
        throw new Error(t('文本模型未返回有效内容'));
      }
      if (action === 'polish' && !isValidPolishedMarkdown(content, result)) {
        throw new Error(t('AI 返回内容不符合要求，已保留原文'));
      }
      updateTarget(result);
      showSuccess(
        action === 'translate'
          ? t('中译英处理完成')
          : t('AI 润色与模板化处理完成'),
      );
    } catch (error) {
      updateTarget(targetOriginal);
      if (error.name === 'AbortError') {
        showSuccess(
          action === 'translate'
            ? t('已停止翻译，已恢复原文')
            : t('已停止润色，已恢复原文'),
        );
        return;
      }
      showError(error.message || t('AI 处理失败'));
    } finally {
      if (aiAbortControllerRef.current === controller) {
        aiAbortControllerRef.current = null;
      }
      setAiAction('');
    }
  };

  const stopAiAction = () => {
    aiAbortControllerRef.current?.abort();
  };

  const save = async () => {
    if (!modelDetail?.id || !selectedChannelDoc) return;
    setSaving(true);
    try {
      const payload = {
        channel_id: selectedChannelDoc.channel_id,
        model_name: selectedChannelDoc.model_name,
        doc_introduction: docIntroduction || '',
        api_docs: '',
        api_docs_markdown: apiDocsMarkdown || '',
        api_docs_markdown_en: apiDocsMarkdownEn || '',
      };
      const res = await API.put(
        `/api/models/${modelDetail.id}/channel_docs`,
        payload,
      );
      const { success, message, data: savedDoc } = res.data || {};
      if (!success) {
        showError(message || t('保存失败'));
        return;
      }
      if (
        (savedDoc?.api_docs_markdown || '') !== payload.api_docs_markdown ||
        (savedDoc?.api_docs_markdown_en || '') !== payload.api_docs_markdown_en
      ) {
        throw new Error(
          t('文档未被后端保存，请重启后端并确认数据库迁移已完成'),
        );
      }
      await loadModel(selectedChannelDocKey);
      showSuccess(t('渠道文档保存成功！'));
      refresh?.();
    } catch (error) {
      showError(
        error.response?.data?.message || error.message || t('保存失败'),
      );
    } finally {
      setSaving(false);
    }
  };

  const restoreInheritedDocs = async () => {
    if (!modelDetail?.id || !selectedChannelDoc?.configured) return;
    setSaving(true);
    try {
      const params = new URLSearchParams({
        channel_id: String(selectedChannelDoc.channel_id),
        model_name: selectedChannelDoc.model_name,
      });
      const res = await API.delete(
        `/api/models/${modelDetail.id}/channel_docs?${params.toString()}`,
      );
      if (!res.data?.success) {
        showError(res.data?.message || t('恢复继承失败'));
        return;
      }
      await loadModel(selectedChannelDocKey);
      showSuccess(t('已恢复继承模型默认文档'));
      refresh?.();
    } catch (error) {
      showError(error.response?.data?.message || t('恢复继承失败'));
    } finally {
      setSaving(false);
    }
  };

  return (
    <SideSheet
      placement='right'
      visible={visible}
      width={isMobile ? '100%' : 920}
      title={
        <Space>
          <Tag color='cyan' shape='circle'>
            {t('API 文档')}
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
          <Space wrap>
            <Button
              type='tertiary'
              disabled={!selectedChannelDoc?.configured || Boolean(aiAction)}
              loading={saving}
              onClick={restoreInheritedDocs}
            >
              {t('恢复继承')}
            </Button>
            <Button
              icon={<Save size={16} />}
              loading={saving}
              disabled={!selectedChannelDoc || Boolean(aiAction)}
              onClick={save}
            >
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
        <div className='space-y-3 p-3'>
          <Card className='!rounded-lg shadow-sm border-0'>
            <div className='mb-3 flex items-center'>
              <Avatar size='small' color='cyan' className='mr-2 shadow-md'>
                <FileText size={16} />
              </Avatar>
              <div>
                <Text className='text-base font-medium'>{t('文档配置')}</Text>
                <div className='text-xs text-semi-color-text-2'>
                  {t('配置后将在模型广场的渠道 API 文档中展示')}
                </div>
              </div>
            </div>
            <Form>
              <Form.Slot label={t('渠道')}>
                <Select
                  filter
                  disabled={Boolean(aiAction)}
                  style={{ width: '100%' }}
                  value={selectedChannelDocKey || undefined}
                  placeholder={t('请选择渠道模型')}
                  emptyContent={t('暂无匹配的渠道模型')}
                  optionList={channelDocOptions}
                  onChange={selectChannelDoc}
                />
                {selectedChannelDoc ? (
                  <div className='mt-1 text-xs text-semi-color-text-2'>
                    {selectedChannelDoc.configured
                      ? t('当前使用渠道独立文档')
                      : t('当前继承模型默认文档，保存后转为渠道独立文档')}
                  </div>
                ) : null}
              </Form.Slot>
              <Form.Slot label={t('套用其他模型文档')}>
                <Select
                  filter
                  showClear
                  disabled={Boolean(aiAction)}
                  style={{ width: '100%' }}
                  placeholder={t('搜索模型名称并套用已有文档')}
                  optionList={templateOptions}
                  onChange={applyTemplate}
                />
              </Form.Slot>
              <Form.Slot label={t('模型介绍')}>
                <MarkdownEditor
                  value={docIntroduction}
                  onChange={setDocIntroduction}
                  placeholder={t('可填写 Markdown 格式的模型介绍')}
                  t={t}
                />
              </Form.Slot>
            </Form>
          </Card>

          <Card className='!rounded-lg shadow-sm border-0'>
            <div className='mb-3 flex flex-wrap items-center justify-between gap-2'>
              <div>
                <Text className='text-base font-medium'>{t('API 文档')}</Text>
                <div className='text-xs text-semi-color-text-2'>
                  {t(
                    '上传、编辑并预览 Markdown，支持多个 MD 或 TXT 文件合并导入',
                  )}
                </div>
              </div>
              <div className='flex flex-wrap items-center gap-2'>
                <input
                  ref={fileInputRef}
                  type='file'
                  accept='.md,.markdown,.txt,text/markdown,text/plain'
                  multiple
                  hidden
                  onChange={handleFileInputChange}
                />
                <Tooltip content={t('可一次选择多个 Markdown 或 TXT 文件')}>
                  <Button
                    icon={<Upload size={16} />}
                    disabled={Boolean(aiAction)}
                    onClick={() => fileInputRef.current?.click()}
                  >
                    {t('导入文档')}
                  </Button>
                </Tooltip>
                <Button
                  icon={<Braces size={16} />}
                  disabled={Boolean(aiAction)}
                  onClick={insertHttpTemplate}
                >
                  {t('插入请求模板')}
                </Button>
              </div>
            </div>

            <Banner
              type='info'
              closeIcon={null}
              className='!rounded-lg mb-3'
              description={
                <span>
                  {t('文档支持')} <code>{'{{base_url}}'}</code>、
                  <code>{'{{model}}'}</code>、<code>{'{{api_key}}'}</code>{' '}
                  {t(
                    '模板变量；首页展示时会替换为当前站点、渠道模型和所选 API Key。',
                  )}
                </span>
              }
            />

            <div className='mb-3 flex flex-wrap items-center gap-2'>
              <div
                className='flex items-center gap-2'
                role='group'
                aria-label={t('文档语言')}
              >
                <Button
                  type={documentLanguage === 'zh' ? 'primary' : 'tertiary'}
                  theme={documentLanguage === 'zh' ? 'solid' : 'light'}
                  disabled={Boolean(aiAction)}
                  onClick={() => setDocumentLanguage('zh')}
                >
                  {t('中文文档')}
                </Button>
                <Button
                  type={documentLanguage === 'en' ? 'primary' : 'tertiary'}
                  theme={documentLanguage === 'en' ? 'solid' : 'light'}
                  disabled={Boolean(aiAction)}
                  onClick={() => setDocumentLanguage('en')}
                >
                  English
                </Button>
              </div>
              <Select
                filter
                disabled={Boolean(aiAction)}
                style={{ minWidth: 220, flex: '1 1 240px' }}
                prefix={t('AI 处理模型')}
                value={selectedTextModel || undefined}
                placeholder={t('请选择模型')}
                emptyContent={t('暂无可用文本模型')}
                optionList={textModelOptions}
                onChange={setSelectedTextModel}
              />
              <Tooltip content={t('设置润色和翻译提示词')}>
                <Button
                  icon={<Settings2 size={16} />}
                  disabled={Boolean(aiAction)}
                  onClick={openPromptSettings}
                >
                  {t('提示词设置')}
                </Button>
              </Tooltip>
              <Button
                type='primary'
                theme='light'
                icon={<Sparkles size={16} />}
                loading={aiAction === 'polish'}
                disabled={!selectedTextModel || Boolean(aiAction)}
                onClick={() => runAiAction('polish')}
              >
                {t('AI 润色并模板化')}
              </Button>
              <Button
                type='tertiary'
                icon={<Languages size={16} />}
                loading={aiAction === 'translate'}
                disabled={!selectedTextModel || Boolean(aiAction)}
                onClick={() => runAiAction('translate')}
              >
                {t('中译英')}
              </Button>
              {aiAction ? (
                <Button
                  type='danger'
                  theme='light'
                  icon={<Square size={14} fill='currentColor' />}
                  onClick={stopAiAction}
                >
                  {aiAction === 'translate' ? t('停止翻译') : t('停止润色')}
                </Button>
              ) : null}
            </div>

            {aiAction ? (
              <div
                className='document-ai-running-status mb-3 flex items-center gap-3 rounded-lg border px-3 py-2'
                role='status'
                aria-live='polite'
              >
                <Sparkles
                  className='document-ai-running-icon shrink-0'
                  size={18}
                />
                <Text strong>
                  {aiAction === 'translate'
                    ? t('正在调用 {{model}} 生成英文文档', {
                        model: selectedTextModel,
                      })
                    : t('正在调用 {{model}} 润色文档', {
                        model: selectedTextModel,
                      })}
                </Text>
                <span className='document-ai-running-dots' aria-hidden='true'>
                  <i />
                  <i />
                  <i />
                </span>
                <Text className='ml-auto' type='tertiary' size='small'>
                  {t('已生成 {{count}} 个字符', {
                    count: activeMarkdown.length,
                  })}
                </Text>
              </div>
            ) : null}

            <div
              className={`relative rounded-lg border-2 border-dashed p-2 transition-colors ${
                aiAction ? 'document-ai-editor-running' : ''
              } ${
                isDraggingFiles
                  ? 'border-semi-color-primary bg-semi-color-primary-light-default'
                  : 'border-semi-color-border'
              }`}
              onDragEnter={(event) => {
                event.preventDefault();
                if (aiAction) return;
                setIsDraggingFiles(true);
              }}
              onDragOver={(event) => event.preventDefault()}
              onDragLeave={(event) => {
                if (!event.currentTarget.contains(event.relatedTarget)) {
                  setIsDraggingFiles(false);
                }
              }}
              onDrop={handleDocumentDrop}
            >
              {isDraggingFiles ? (
                <div className='pointer-events-none absolute inset-0 z-10 flex items-center justify-center rounded-lg bg-semi-color-bg-0/90'>
                  <div className='text-center'>
                    <Upload className='mx-auto mb-2' size={24} />
                    <Text strong>{t('松开鼠标导入文档')}</Text>
                  </div>
                </div>
              ) : null}
              <div className='mb-2 flex items-center justify-between gap-2 px-1'>
                <Text strong>
                  {documentLanguage === 'en' ? t('英文文档') : t('中文文档')}
                </Text>
                <Text type='tertiary' size='small'>
                  {t('可点击导入，也可将多个 MD 或 TXT 文件拖到这里')}
                </Text>
              </div>
              <MarkdownEditor
                value={activeMarkdown}
                onChange={setActiveMarkdown}
                placeholder={t('请输入或导入 Markdown API 文档')}
                minRows={18}
                disabled={Boolean(aiAction)}
                t={t}
              />
            </div>
          </Card>
        </div>
      </Spin>
      <Modal
        title={t('文档 AI 提示词设置')}
        visible={promptSettingsVisible}
        width={760}
        centered
        maskClosable={false}
        onCancel={() => setPromptSettingsVisible(false)}
        onOk={savePromptSettings}
        confirmLoading={promptSettingsSaving}
        okText={t('保存')}
        cancelText={t('取消')}
      >
        <Spin spinning={promptSettingsLoading}>
          <div className='mb-3 flex items-center justify-between gap-3'>
            <Text type='tertiary'>
              {t('提示词仅由后端读取，不会包含在浏览器发起的 AI 请求中。')}
            </Text>
            <Button
              type='tertiary'
              theme='light'
              icon={<RotateCcw size={15} />}
              loading={promptSettingsSaving}
              disabled={promptSettings.is_default}
              onClick={resetPromptSettings}
            >
              {t('恢复默认')}
            </Button>
          </div>
          <Tabs type='line'>
            <Tabs.TabPane tab={t('润色提示词')} itemKey='polish'>
              <TextArea
                className='document-ai-prompt-textarea mt-3'
                autosize={{ minRows: 16, maxRows: 24 }}
                value={promptSettings.polish_prompt}
                disabled={promptSettingsLoading || promptSettingsSaving}
                onChange={(value) =>
                  setPromptSettings((current) => ({
                    ...current,
                    polish_prompt: value,
                  }))
                }
              />
            </Tabs.TabPane>
            <Tabs.TabPane tab={t('翻译提示词')} itemKey='translate'>
              <TextArea
                className='document-ai-prompt-textarea mt-3'
                autosize={{ minRows: 16, maxRows: 24 }}
                value={promptSettings.translate_prompt}
                disabled={promptSettingsLoading || promptSettingsSaving}
                onChange={(value) =>
                  setPromptSettings((current) => ({
                    ...current,
                    translate_prompt: value,
                  }))
                }
              />
            </Tabs.TabPane>
          </Tabs>
        </Spin>
      </Modal>
    </SideSheet>
  );
};

export default EditModelDocsModal;
