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

import React, { useContext, useEffect, useMemo, useRef, useState } from 'react';
import {
  Avatar,
  Button,
  Card,
  Collapse,
  Select,
  SideSheet,
  Space,
  Table,
  Tabs,
  Tag,
  Tooltip,
  Toast,
  Typography,
} from '@douyinfe/semi-ui';
import {
  IconClose,
  IconCode,
  IconCopy,
  IconHelpCircle,
} from '@douyinfe/semi-icons';
import { API } from '../../../../../helpers';
import { StatusContext } from '../../../../../context/Status';
import { fetchTokenKey, getServerAddress } from '../../../../../helpers/token';
import { useIsMobile } from '../../../../../hooks/common/useIsMobile';
import MarkdownRenderer from '../../../../common/markdown/MarkdownRenderer';

const { Text, Title } = Typography;

const normalizeApiBaseUrl = (baseUrl) =>
  String(baseUrl || '').replace(
    /^https:\/\/demo\.tokenfactoryopen\.com/i,
    'https://tokenfactoryopen.com',
  );

const parseApiDocs = (value) => {
  if (!value) return [];
  try {
    const parsed = typeof value === 'string' ? JSON.parse(value) : value;
    return Array.isArray(parsed) ? parsed : [];
  } catch (_) {
    return [];
  }
};

const getDefaultApiDocs = () => [
  {
    id: 'default-chat-completions',
    description: 'Chat Completions API',
    path: '/v1/chat/completions',
    method: 'POST',
    detail: '',
    body_params: [
      {
        id: 'default-model',
        name: 'model',
        type: 'string',
        required: true,
        example: '<model_name>',
        description: '要调用的模型名称。',
      },
      {
        id: 'default-messages',
        name: 'messages',
        type: 'array',
        required: true,
        example: '',
        description:
          '组成对话的消息列表。每条消息通常包含 role 和 content 字段。',
        children: [
          {
            id: 'default-messages-item',
            name: '',
            type: 'object',
            required: true,
            example: '',
            description: '单条对话消息。',
            children: [
              {
                id: 'default-message-role',
                name: 'role',
                type: 'string',
                required: true,
                example: 'user',
                description: '消息角色，例如 system、user 或 assistant。',
              },
              {
                id: 'default-message-content',
                name: 'content',
                type: 'string',
                required: true,
                example: 'Hello!',
                description: '消息内容。',
              },
            ],
          },
        ],
      },
      {
        id: 'default-stream',
        name: 'stream',
        type: 'boolean',
        required: false,
        example: 'false',
        description: '是否使用流式响应。',
      },
    ],
    path_params: [],
    response_params: [
      {
        id: 'default-response-id',
        name: 'id',
        type: 'string',
        example: 'chatcmpl-123',
        description: '响应 ID。',
      },
      {
        id: 'default-response-object',
        name: 'object',
        type: 'string',
        example: 'chat.completion',
        description: '对象类型。',
      },
      {
        id: 'default-response-created',
        name: 'created',
        type: 'integer',
        example: '1677652288',
        description: '响应创建时间戳。',
      },
      {
        id: 'default-response-model',
        name: 'model',
        type: 'string',
        example: '<model_name>',
        description: '本次响应使用的模型名称。',
      },
      {
        id: 'default-response-choices',
        name: 'choices',
        type: 'array',
        example: '',
        description: '模型生成结果列表。',
        children: [
          {
            id: 'default-response-choice-item',
            name: '',
            type: 'object',
            example: '',
            description: '单个生成结果。',
            children: [
              {
                id: 'default-response-choice-index',
                name: 'index',
                type: 'integer',
                example: '0',
                description: '结果索引。',
              },
              {
                id: 'default-response-choice-message',
                name: 'message',
                type: 'object',
                example: '',
                description: '模型返回的消息。',
                children: [
                  {
                    id: 'default-response-message-role',
                    name: 'role',
                    type: 'string',
                    example: 'assistant',
                    description: '消息角色。',
                  },
                  {
                    id: 'default-response-message-content',
                    name: 'content',
                    type: 'string',
                    example: 'Hello! How can I help you today?',
                    description: '消息内容。',
                  },
                ],
              },
              {
                id: 'default-response-choice-finish',
                name: 'finish_reason',
                type: 'string',
                example: 'stop',
                description: '生成结束原因。',
              },
            ],
          },
        ],
      },
      {
        id: 'default-response-usage',
        name: 'usage',
        type: 'object',
        example: '',
        description: 'Token 使用量统计。',
        children: [
          {
            id: 'default-response-prompt-tokens',
            name: 'prompt_tokens',
            type: 'integer',
            example: '9',
            description: '输入 Token 数。',
          },
          {
            id: 'default-response-completion-tokens',
            name: 'completion_tokens',
            type: 'integer',
            example: '12',
            description: '输出 Token 数。',
          },
          {
            id: 'default-response-total-tokens',
            name: 'total_tokens',
            type: 'integer',
            example: '21',
            description: '总 Token 数。',
          },
        ],
      },
    ],
  },
];

const replaceModelName = (value, modelName) =>
  typeof value === 'string'
    ? value.replaceAll('<model_name>', modelName || '')
    : value;

const convertExampleValue = (value, type, modelName) => {
  const replaced = replaceModelName(value, modelName);
  if (replaced === undefined || replaced === null || replaced === '') {
    if (type === 'number') return 0.1;
    if (type === 'integer') return 1;
    if (type === 'boolean') return true;
    if (type === 'array') return [];
    if (type === 'object') return {};
    if (type === 'null') return null;
    return '';
  }
  if (type === 'number' || type === 'integer') {
    const num = Number(replaced);
    return Number.isFinite(num) ? num : replaced;
  }
  if (type === 'boolean') return replaced === true || replaced === 'true';
  if (type === 'array' || type === 'object') {
    try {
      return JSON.parse(replaced);
    } catch (_) {
      return type === 'array' ? [replaced] : {};
    }
  }
  if (type === 'null') return null;
  return replaced;
};

const hasChildren = (param) =>
  Array.isArray(param?.children) && param.children.length > 0;

const buildParamValue = (param, modelName) => {
  if (param?.type === 'object' && hasChildren(param)) {
    return buildJsonExample(param.children, modelName);
  }
  if (param?.type === 'array') {
    if (!hasChildren(param)) {
      return convertExampleValue(param.example, param.type, modelName);
    }
    if (param.children.length === 1 && !param.children[0].name) {
      return [buildParamValue(param.children[0], modelName)];
    }
    return [buildJsonExample(param.children, modelName)];
  }
  return convertExampleValue(param?.example, param?.type, modelName);
};

const setNestedValue = (target, path, value) => {
  const parts = String(path || '')
    .split('.')
    .map((item) => item.trim())
    .filter(Boolean);
  let cursor = target;
  parts.forEach((part, index) => {
    const key = /^\d+$/.test(part) ? Number(part) : part;
    const isLast = index === parts.length - 1;
    if (isLast) {
      cursor[key] = value;
      return;
    }
    if (cursor[key] === undefined) {
      cursor[key] = /^\d+$/.test(parts[index + 1]) ? [] : {};
    }
    cursor = cursor[key];
  });
};

const buildJsonExample = (params = [], modelName) => {
  const root = {};
  params.forEach((param) => {
    if (!param.name) return;
    const value = buildParamValue(param, modelName);
    if (hasChildren(param)) {
      root[param.name] = value;
      return;
    }
    setNestedValue(root, param.name, value);
  });
  return root;
};

const getParamRowKey = (param, path) => param.id || path || param.name;

const buildParamTreeData = (params = [], parentPath = '') =>
  (Array.isArray(params) ? params : []).map((param, index) => {
    const name = param.name || '';
    const path = parentPath
      ? name
        ? `${parentPath}.${name}`
        : parentPath
      : name || `${index}`;
    const children = hasChildren(param)
      ? buildParamTreeData(
          param.children,
          param.type === 'array' ? `${path}[]` : path,
        )
      : undefined;
    return {
      ...param,
      key: getParamRowKey(param, path),
      displayName: name,
      fullPath: path,
      children,
    };
  });

const collectParamRowKeys = (rows = []) =>
  rows.flatMap((row) => [
    row.key,
    ...(Array.isArray(row.children) ? collectParamRowKeys(row.children) : []),
  ]);

const flattenLeafParams = (params = [], parentPath = '') => {
  const rows = [];
  (Array.isArray(params) ? params : []).forEach((param, index) => {
    const name = param.name || '';
    const path = parentPath
      ? name
        ? `${parentPath}.${name}`
        : parentPath
      : name || `${index}`;
    if (!hasChildren(param)) {
      rows.push({ ...param, name: path });
      return;
    }
    rows.push(
      ...flattenLeafParams(
        param.children,
        param.type === 'array' ? `${path}[]` : path,
      ),
    );
  });
  return rows;
};

const buildQueryUrl = (endpoint, params = [], modelName) => {
  const url = new URL(endpoint, window.location.origin);
  flattenLeafParams(params).forEach((param) => {
    if (!param.name) return;
    url.searchParams.set(
      param.name,
      String(convertExampleValue(param.example, param.type, modelName)),
    );
  });
  return url.toString();
};

const CodeBlock = ({ content, language = 'json', t }) => {
  const handleCopy = () => {
    navigator.clipboard
      .writeText(content)
      .then(() => Toast.success({ content: t('已复制') }))
      .catch(() => Toast.error({ content: t('复制失败') }));
  };
  return (
    <div className='relative rounded-lg overflow-hidden border border-gray-100'>
      <div className='absolute top-1 right-1 z-10'>
        <Button
          icon={<IconCopy />}
          size='small'
          type='tertiary'
          theme='borderless'
          onClick={handleCopy}
          title={t('复制')}
        />
      </div>
      <pre
        className='m-0 p-3 text-xs overflow-x-auto'
        style={{
          backgroundColor: 'var(--semi-color-fill-0)',
          color: 'var(--semi-color-text-0)',
          fontFamily:
            'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
          lineHeight: 1.6,
          whiteSpace: 'pre',
        }}
      >
        <code className={`language-${language}`}>{content}</code>
      </pre>
    </div>
  );
};

const buildExamples = ({ method, endpoint, requestJson, token }) => {
  const auth = `Bearer ${token || 'apikey'}`;
  const body = requestJson || '{}';
  const isPost = method === 'POST';
  return {
    curl: isPost
      ? [
          `curl -X POST "${endpoint}" \\`,
          `  -H "Content-Type: application/json" \\`,
          `  -H "Authorization: ${auth}" \\`,
          `  -d '${body.replace(/'/g, `'\\''`)}'`,
        ].join('\n')
      : [`curl -X GET "${endpoint}" \\`, `  -H "Authorization: ${auth}"`].join(
          '\n',
        ),
    javascript: isPost
      ? `const response = await fetch('${endpoint}', {\n  method: 'POST',\n  headers: {\n    'Content-Type': 'application/json',\n    Authorization: '${auth}',\n  },\n  body: JSON.stringify(${body}),\n});\nconst data = await response.json();`
      : `const response = await fetch('${endpoint}', {\n  headers: { Authorization: '${auth}' },\n});\nconst data = await response.json();`,
    go: isPost
      ? `payload := strings.NewReader(\`${body}\`)\nreq, _ := http.NewRequest("POST", "${endpoint}", payload)\nreq.Header.Set("Content-Type", "application/json")\nreq.Header.Set("Authorization", "${auth}")\nresp, err := http.DefaultClient.Do(req)`
      : `req, _ := http.NewRequest("GET", "${endpoint}", nil)\nreq.Header.Set("Authorization", "${auth}")\nresp, err := http.DefaultClient.Do(req)`,
    python: isPost
      ? `import requests\n\nresponse = requests.post(\n    "${endpoint}",\n    headers={"Content-Type": "application/json", "Authorization": "${auth}"},\n    json=${body},\n)\nprint(response.json())`
      : `import requests\n\nresponse = requests.get(\n    "${endpoint}",\n    headers={"Authorization": "${auth}"},\n)\nprint(response.json())`,
    java: isPost
      ? `HttpRequest request = HttpRequest.newBuilder()\n    .uri(URI.create("${endpoint}"))\n    .header("Content-Type", "application/json")\n    .header("Authorization", "${auth}")\n    .POST(HttpRequest.BodyPublishers.ofString("""${body}"""))\n    .build();`
      : `HttpRequest request = HttpRequest.newBuilder()\n    .uri(URI.create("${endpoint}"))\n    .header("Authorization", "${auth}")\n    .GET()\n    .build();`,
    csharp: isPost
      ? `using var client = new HttpClient();\nclient.DefaultRequestHeaders.Add("Authorization", "${auth}");\nvar content = new StringContent("""${body}""", Encoding.UTF8, "application/json");\nvar response = await client.PostAsync("${endpoint}", content);`
      : `using var client = new HttpClient();\nclient.DefaultRequestHeaders.Add("Authorization", "${auth}");\nvar response = await client.GetAsync("${endpoint}");`,
  };
};

const ApiDocsSidePanel = ({
  visible,
  onClose,
  modelName,
  docIntroduction = '',
  apiDocs = '',
  t,
}) => {
  const isMobile = useIsMobile();
  const [statusState] = useContext(StatusContext);
  const [tokens, setTokens] = useState([]);
  const [selectedTokenId, setSelectedTokenId] = useState();
  const [resolvedTokenKeys, setResolvedTokenKeys] = useState({});
  const tokenRequestsRef = useRef({});
  const configuredDocs = useMemo(() => parseApiDocs(apiDocs), [apiDocs]);
  const localModelDefaultDocsEnabled = useMemo(() => {
    const value = localStorage.getItem('model_default_docs_enabled');
    if (value === null || value === undefined) return undefined;
    return value !== 'false';
  }, [visible]);
  const modelDefaultDocsEnabled =
    localModelDefaultDocsEnabled ??
    (statusState?.status?.model_default_docs_enabled !== false &&
      statusState?.status?.model_default_docs_enabled !== 'false');
  const docs = useMemo(() => {
    const hasCustomDocs =
      String(docIntroduction || '').trim() || configuredDocs.length > 0;
    if (!hasCustomDocs && modelDefaultDocsEnabled) return getDefaultApiDocs();
    return configuredDocs;
  }, [configuredDocs, docIntroduction, modelDefaultDocsEnabled]);

  const serverAddress = useMemo(() => {
    try {
      return normalizeApiBaseUrl(getServerAddress()).replace(/\/$/, '');
    } catch (e) {
      return '';
    }
  }, [visible]);

  useEffect(() => {
    if (!visible) return;
    let cancelled = false;
    (async () => {
      try {
        const res = await API.get('/api/token/?p=1&size=100', {
          skipErrorHandler: true,
        });
        if (!res.data?.success || cancelled) return;
        const items = Array.isArray(res.data.data)
          ? res.data.data
          : res.data.data?.items || [];
        const active = items.filter((item) => item.status === 1);
        setTokens(active);
        if (active.length > 0)
          setSelectedTokenId((prev) => prev || active[0].id);
      } catch (_) {
        if (!cancelled) setTokens([]);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [visible]);

  useEffect(() => {
    if (!selectedTokenId || resolvedTokenKeys[selectedTokenId]) return;
    if (tokenRequestsRef.current[selectedTokenId]) return;
    tokenRequestsRef.current[selectedTokenId] = fetchTokenKey(selectedTokenId)
      .then((key) => {
        setResolvedTokenKeys((prev) => ({
          ...prev,
          [selectedTokenId]: `sk-${key}`,
        }));
      })
      .catch(() => {})
      .finally(() => {
        delete tokenRequestsRef.current[selectedTokenId];
      });
  }, [selectedTokenId, resolvedTokenKeys]);

  const selectedToken = selectedTokenId
    ? resolvedTokenKeys[selectedTokenId] || 'apikey'
    : 'apikey';

  const getParamColumns = (showRequired = true) => {
    const columns = [
      {
        title: t('参数'),
        dataIndex: 'displayName',
        width: 180,
        render: (val, record) => (
          <Tag color='blue' size='small' shape='circle'>
            {val || record.fullPath}
          </Tag>
        ),
      },
      { title: t('类型'), dataIndex: 'type', width: 60 },
    ];
    if (showRequired) {
      columns.push({
        title: t('必填'),
        dataIndex: 'required',
        width: 50,
        render: (val) => (
          <Tag color={val ? 'red' : 'grey'} size='small' shape='circle'>
            {val ? t('是') : t('否')}
          </Tag>
        ),
      });
    }
    columns.push({
      title: t('示例值'),
      dataIndex: 'example',
      width: 140,
      render: (val) => (
        <Text size='small'>{replaceModelName(val, modelName)}</Text>
      ),
    });
    columns.push({
      title: t('说明'),
      dataIndex: 'description',
      width: 60,
      render: (val) =>
        String(val || '').trim() ? (
          <Tooltip
            content={
              <div style={{ maxWidth: 360, whiteSpace: 'pre-wrap' }}>{val}</div>
            }
            trigger='hover'
            showArrow
          >
            <Button
              size='small'
              type='tertiary'
              theme='borderless'
              icon={<IconHelpCircle />}
              aria-label={t('说明')}
            />
          </Tooltip>
        ) : null,
    });
    return columns;
  };

  const renderParamTable = (params, showRequired = true) => {
    const dataSource = buildParamTreeData(params);
    return (
      <Table
        columns={getParamColumns(showRequired)}
        dataSource={dataSource}
        pagination={false}
        size='small'
        rowKey='key'
        defaultExpandAllRows
        defaultExpandedRowKeys={collectParamRowKeys(dataSource)}
      />
    );
  };

  const renderApiPanel = (api, index) => {
    const method = String(api.method || 'POST').toUpperCase();
    const isPost = method === 'POST';
    const endpointPath = replaceModelName(api.path || '', modelName);
    const endpoint = `${serverAddress}${endpointPath}`;
    const requestParams = isPost
      ? api.body_params || []
      : api.path_params || [];
    const responseParams = api.response_params || [];
    const requestJson = isPost
      ? JSON.stringify(buildJsonExample(requestParams, modelName), null, 2)
      : '';
    const requestUrl = isPost
      ? endpoint
      : buildQueryUrl(endpoint, requestParams, modelName);
    const headers = isPost
      ? {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${selectedToken}`,
        }
      : { Authorization: `Bearer ${selectedToken}` };
    const examples = buildExamples({
      method,
      endpoint: requestUrl,
      requestJson,
      token: selectedToken,
    });

    return (
      <Collapse.Panel
        key={api.id || index}
        itemKey={api.id || index}
        header={
          <div className='flex items-center gap-2 min-w-0'>
            <Tag color={isPost ? 'green' : 'blue'} size='small' shape='circle'>
              {method}
            </Tag>
            <Text strong ellipsis={{ showTooltip: true }}>
              {api.description || endpointPath || `${t('API')} #${index + 1}`}
            </Text>
            <Text type='tertiary' ellipsis={{ showTooltip: true }}>
              {endpointPath}
            </Text>
          </div>
        }
      >
        {api.detail ? (
          <Card
            className='!rounded-2xl shadow-sm border-0 mb-3'
            title={t('接口详情')}
          >
            <MarkdownRenderer content={api.detail} />
          </Card>
        ) : null}
        <Card
          className='!rounded-2xl shadow-sm border-0 mb-3'
          title={isPost ? t('Body 参数说明') : t('Path 参数说明')}
        >
          {renderParamTable(requestParams, true)}
        </Card>
        <Card
          className='!rounded-2xl shadow-sm border-0 mb-3'
          title={t('请求头')}
          headerExtraContent={
            tokens.length > 0 ? (
              <Select
                size='small'
                value={selectedTokenId}
                style={{ width: 180 }}
                optionList={tokens.map((token) => ({
                  label: token.name || `${t('令牌')} #${token.id}`,
                  value: token.id,
                }))}
                onChange={setSelectedTokenId}
              />
            ) : null
          }
        >
          <CodeBlock content={JSON.stringify(headers, null, 2)} t={t} />
        </Card>
        <Card
          className='!rounded-2xl shadow-sm border-0 mb-3'
          title={isPost ? t('请求参数示例') : t('请求链接示例')}
        >
          <CodeBlock
            content={isPost ? requestJson : requestUrl}
            language={isPost ? 'json' : 'text'}
            t={t}
          />
        </Card>
        <Card
          className='!rounded-2xl shadow-sm border-0 mb-3 api-docs-request-examples-card'
          title={t('请求示例')}
        >
          <Tabs type='line'>
            {[
              ['cURL', 'curl', 'bash'],
              ['JavaScript', 'javascript', 'javascript'],
              ['Go', 'go', 'go'],
              ['Python', 'python', 'python'],
              ['Java', 'java', 'java'],
              ['C#', 'csharp', 'csharp'],
            ].map(([label, key, lang]) => (
              <Tabs.TabPane
                tab={label}
                itemKey={key}
                key={key}
                className='mt-2'
              >
                <CodeBlock content={examples[key]} language={lang} t={t} />
              </Tabs.TabPane>
            ))}
          </Tabs>
        </Card>
        <Card
          className='!rounded-2xl shadow-sm border-0 mb-3'
          title={t('返回数据结构说明')}
        >
          {renderParamTable(responseParams, false)}
        </Card>
        <Card
          className='!rounded-2xl shadow-sm border-0 mb-3'
          title={t('返回数据示例')}
        >
          <CodeBlock
            content={JSON.stringify(
              buildJsonExample(responseParams, modelName),
              null,
              2,
            )}
            language='json'
            t={t}
          />
        </Card>
      </Collapse.Panel>
    );
  };

  const content = (
    <div className='p-2 space-y-3'>
      <style>
        {`
          .api-docs-list-card > .semi-card-body {
            padding-left: 0 !important;
            padding-right: 0 !important;
          }
          .api-docs-request-examples-card > .semi-card-body {
            padding-top: 0 !important;
          }
        `}
      </style>
      {docIntroduction ? (
        <Card className='!rounded-2xl shadow-sm border-0' title={t('模型介绍')}>
          <MarkdownRenderer content={docIntroduction} />
        </Card>
      ) : null}
      {docs.length > 0 ? (
        <Card
          className='!rounded-2xl shadow-sm border-0 api-docs-list-card'
          title={t('API 列表')}
        >
          <Collapse>{docs.map(renderApiPanel)}</Collapse>
        </Card>
      ) : (
        <Card
          className='!rounded-2xl shadow-sm border-0 text-center py-8'
          title={t('API 列表')}
        >
          <Text type='tertiary'>{t('暂无模型文档')}</Text>
        </Card>
      )}
    </div>
  );

  if (isMobile) {
    return (
      <SideSheet
        placement='right'
        title={
          <Space>
            <Tag color='cyan' shape='circle'>
              {t('API')}
            </Tag>
            <Title heading={5} className='m-0'>
              {t('在线 API 文档')}
            </Title>
            <div className='text-xs text-gray-600 truncate'>{modelName}</div>
          </Space>
        }
        visible={visible}
        onCancel={onClose}
        width='100%'
        bodyStyle={{ padding: 0 }}
        closeIcon={
          <Button type='tertiary' icon={<IconClose />} onClick={onClose} />
        }
      >
        {content}
      </SideSheet>
    );
  }

  if (!visible) return null;
  return (
    <div
      className='fixed top-0 h-full overflow-y-auto z-[999] semi-sidesheet-inner'
      style={{
        width: 640,
        right: 600,
        backgroundColor: 'var(--semi-color-bg-0)',
        borderRight: '1px solid var(--semi-color-border)',
        animation: 'slideInLeft 0.3s ease-out',
      }}
    >
      <div className='semi-sidesheet-header'>
        <div className='semi-sidesheet-title'>
          <Space>
            <Tag color='cyan' shape='circle'>
              {t('API')}
            </Tag>
            <Title heading={4} className='m-0'>
              {t('在线 API 文档')}
            </Title>
            <div className='text-xs text-gray-600 truncate'>{modelName}</div>
          </Space>
        </div>
        <Button
          className='semi-sidesheet-close'
          type='tertiary'
          theme='borderless'
          icon={<IconClose />}
          size='small'
          onClick={onClose}
        />
      </div>
      <div className='semi-sidesheet-body' style={{ padding: 0 }}>
        {content}
      </div>
    </div>
  );
};

export default ApiDocsSidePanel;
