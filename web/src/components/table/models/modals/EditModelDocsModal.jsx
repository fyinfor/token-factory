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
  Save,
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
    API.getUri({ url: '/api/playground/chat/completions' }),
    {
      method: 'POST',
      credentials: 'include',
      signal,
      headers: {
        Accept: 'text/event-stream',
        'Content-Type': 'application/json',
        'New-API-User': String(getUserIdFromLocalStorage()),
      },
      body: JSON.stringify({ ...payload, stream: true }),
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

const POLISH_SYSTEM_PROMPT = `你是 API 文档技术编辑。请把原始 Markdown 重构成面向开发任务、可直接复制使用的发布版文档。目标是去除无效重复，同时保留足够的场景、示例和完整调用流程；不要只做近义词替换，也不要为了缩短篇幅删除实用内容。

你必须直接以“# ”开始输出最终 Markdown。禁止输出思考过程、分析、草稿、检查清单、<think>、<analysis>、前言或第二份文档。

按开发者完成任务的顺序组织：
1. 一个 H1 标题、适用模型、BaseURL、鉴权方式和简短接口概览。接口概览应让读者立即看见主要方法与路径。
2. “快速开始”：提供最小可运行请求、请求响应，以及取得最终结果所需的下一步。
3. 快速开始的同一请求使用一个 :::code-group，依次给出 cURL、JavaScript、Python、Java 四种可运行示例。四个标签必须使用相同 URL、请求字段和业务语义；只转换客户端写法，不改变 API 行为。
4. 如果原文是异步 API，增加“异步任务流程”：完整说明创建、取得任务 ID、查询、成功和失败处理。语言示例只需完成创建和一次状态查询，不得自行添加轮询间隔、重试次数或超时数字。原文存在 completed 或 failed 响应正文时必须原样保留其字段结构。
5. “请求参数”：按核心参数、媒体输入、生成控制等业务含义分组；每个字段只完整解释一次。字段的别名或不同传入位置写在同一行。
6. “常用场景”：先用简表说明差异，再为具有独立输入组合或调用目的的主要场景提供可复制示例。场景表中列出的主要模式必须能在后文找到对应示例或明确的差异 JSON。
7. “响应、错误与限制”：保留完整成功结果、状态字段、失败处理、能力限制和原文已有的错误信息。
8. 素材库、回调、尾帧等独有能力放在最后，并保留至少一个可直接套用的示例。

编辑规则：
- 删除重复章节、重复请求头、重复参数表，以及请求结构完全相同且只更换提示词文本的示例。不得删除独有字段组合、限制、响应结构或必要业务步骤。
- 原文超过 8000 字符时，最终正文控制在原文的 50%–80%。以信息覆盖为先，绝不能比原文更长。
- 长文通常保留 4–6 个完整场景请求，不含快速开始的多语言版本、响应示例和状态查询。若原文的独立场景不足则不要凑数；若场景超过 6 个，将结构相近的次要场景改成差异 JSON，不得直接消失。
- 参数表只能使用“字段 / 类型 / 必填 / 说明”四列或更少。传入位置、默认值、枚举和限制写入“说明”，不得增加第五列。
- 状态枚举不混入参数表。状态不超过 8 个时使用“状态值 / 含义 / 下一步”三列表格，状态值使用行内代码；“下一步”只写原文明确给出的查询、取值或错误处理动作，不得自行给出简化提示词、减少素材等建议。
- 只允许一个 H1；主要章节用 H2，子章节用 H3；不保留机械编号，不手写目录，不过度使用分隔线。
- 一个 :::code-group 最多四个标签，只用于快速开始中同一请求的 cURL、JavaScript、Python、Java 写法。不要把不同业务场景塞进同一个标签组。其他场景沿用原文的示例语言或使用 JSON 差异片段。
- 多语言代码必须使用各语言标准 HTTP 客户端并保持请求体字段一致。异步接口不得把单次请求机械翻译成虚假的完整轮询流程。
- 当前站点写成 {{base_url}}，模型写成 {{model}}，API Key 写成 {{api_key}}；不要替换第三方素材地址或业务回调地址。
- 保持原文语言与事实。严禁补充原文没有的轮询间隔、默认值、支持范围、错误码或最佳实践；不确定时宁可省略，不写“待补充”。
- 保证所有 Markdown 代码围栏和 :::code-group 成对闭合。

输出前只在内部检查，不要展示检查过程：唯一 H1、无思考内容、无重复整篇、表格不超过四列、模板变量正确、篇幅达标、场景表与示例对应、原文中的完整结果没有丢失。`;

const TRANSLATE_SYSTEM_PROMPT = `Translate the supplied Chinese API documentation into clear technical English.

Rules:
1. Output only the complete translated Markdown. Do not wrap the whole result in another code fence.
2. Translate prose, headings, table labels, and comments intended for readers.
3. Preserve URLs, endpoint paths, HTTP methods, JSON keys, field names, identifiers, shell commands, code syntax, and the placeholders {{base_url}}, {{model}}, and {{api_key}} exactly.
4. Preserve all Markdown structure and fenced-code language tags.
5. Do not omit steps, especially asynchronous task creation, polling, failure handling, and result download flows.
6. Preserve :::code-group containers and fenced-code labels such as \`\`\`javascript [JavaScript].
7. Do not invent or correct API behavior unless the source explicitly says so.`;

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
          temperature: action === 'translate' ? 0.1 : 0.2,
          messages: [
            {
              role: 'system',
              content:
                action === 'translate'
                  ? TRANSLATE_SYSTEM_PROMPT
                  : POLISH_SYSTEM_PROMPT,
            },
            {
              role: 'user',
              content:
                action === 'translate'
                  ? content
                  : `请对下面的原始 API 文档进行明显的结构重构。不要只替换近义词；重点降低参数和示例的重复度，并让首次调用流程出现在最前面。\n\n--- 原始文档开始 ---\n\n${content}\n\n--- 原始文档结束 ---`,
            },
          ],
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
    </SideSheet>
  );
};

export default EditModelDocsModal;
