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
  Card,
  Avatar,
  Typography,
  Badge,
  Button,
  Tooltip,
  Toast,
  Modal,
  Tag,
} from '@douyinfe/semi-ui';
import { IconCopy, IconHelpCircle, IconLink } from '@douyinfe/semi-icons';
import { API, copy } from '../../../../../helpers';
import {
  fetchTokenKey as fetchTokenKeyById,
  getServerAddress,
} from '../../../../../helpers/token';

const { Text } = Typography;

const StepTitle = ({ label, title, desc, icon, extra }) => (
  <div className='flex items-start gap-3 mb-4'>
    <div
      className='flex items-center justify-center gap-1.5 shrink-0 rounded-full font-semibold text-xs px-3'
      style={{
        height: 30,
        width: 84,
        color: 'var(--semi-color-bg-0)',
        backgroundColor: 'var(--semi-color-primary)',
        boxShadow: '0 6px 14px rgba(var(--semi-blue-5), 0.24)',
      }}
    >
      {icon ? <span className='inline-flex items-center'>{icon}</span> : null}
      {label}
    </div>
    <div className='min-w-0'>
      <div className='flex items-center gap-2 flex-wrap'>
        <Text className='text-lg font-medium'>{title}</Text>
        {extra}
      </div>
      {desc ? <div className='text-xs text-gray-600 mt-0.5'>{desc}</div> : null}
    </div>
  </div>
);

const ModalStepTitle = ({ label, title }) => (
  <div className='flex items-center gap-2 mb-2'>
    <span
      className='inline-flex items-center justify-center rounded-full text-xs font-semibold px-2 py-0.5'
      style={{
        width: 54,
        color: 'var(--semi-color-primary)',
        backgroundColor: 'var(--semi-color-primary-light-default)',
      }}
    >
      {label}
    </span>
    <Text strong>{title}</Text>
  </div>
);

const getCurrentSiteOrigin = () => {
  if (typeof window !== 'undefined' && window.location?.origin) {
    return window.location.origin.replace(/\/+$/, '');
  }
  return '';
};

const getSiteOrigin = () => {
  const currentOrigin = getCurrentSiteOrigin();
  if (currentOrigin) return currentOrigin;
  return String(getServerAddress() || '')
    .replace(/\/v1\/?$/i, '')
    .replace(/\/+$/, '');
};

const hasTag = (tags, target) =>
  String(tags || '')
    .split(',')
    .map((tag) => tag.trim())
    .filter(Boolean)
    .includes(target);

const isTextModel = (modelData) => {
  const tags = modelData?.tags || '';
  if (!tags || !String(tags).trim()) return true;
  return hasTag(tags, '文本');
};

const getChannelRouteModelName = (modelData, channel) => {
  const modelName = modelData?.model_name || '';
  if (channel?.route_slug) {
    return `${modelName}/${channel.route_slug}`;
  }
  return `${channel?.supplier_alias || ''}/${modelName}/${channel?.channel_no || ''}`;
};

const AddressUsageNote = ({ title, children }) => (
  <div
    className='mt-2 rounded-lg px-3 py-2 text-xs leading-5'
    style={{
      color: 'var(--semi-color-text-1)',
      backgroundColor: 'var(--semi-color-primary-light-default)',
    }}
  >
    <div className='font-medium mb-0.5'>{title}</div>
    <div>{children}</div>
  </div>
);

const TOOL_LINKS = {
  OpenClaw: 'https://openclaw.ai/',
  WorkBuddy: 'https://www.codebuddy.cn/work/',
  'OpenAI SDK': 'https://platform.openai.com/docs/libraries',
  'Cherry Studio': 'https://www.cherry-ai.com/',
  Chatbox: 'https://chatboxai.app/',
  LobeChat: 'https://www.lobechat.co/',
  Dify: 'https://dify.ai/',
  Hermes: 'https://hermes-agent.ai/',
  Postman: 'https://www.postman.com/',
  Apifox: 'https://apifox.com/',
  curl: 'https://curl.se/',
  'OpenAI Videos API / Sora':
    'https://platform.openai.com/docs/guides/video-generation',
};

const ToolLink = ({ name }) => {
  const href = TOOL_LINKS[name];
  if (!href) return name;
  return (
    <a
      href={href}
      target='_blank'
      rel='noreferrer'
      className='underline underline-offset-2'
      style={{ color: 'var(--semi-color-primary)' }}
    >
      {name}
    </a>
  );
};

const ToolList = ({ names }) => (
  <>
    {names.map((name, idx) => (
      <React.Fragment key={name}>
        {idx > 0 ? '、' : null}
        <ToolLink name={name} />
      </React.Fragment>
    ))}
  </>
);

const getEndpointText = (type, path) =>
  `${type || ''} ${path || ''}`.toLowerCase();

const isVideoEndpoint = (type, path) => {
  const endpointText = getEndpointText(type, path);
  return endpointText.includes('video') || endpointText.includes('视频');
};

const ModelEndpoints = ({ modelData, endpointMap = {}, t }) => {
  const [workBuddyVisible, setWorkBuddyVisible] = useState(false);
  const [tokens, setTokens] = useState([]);
  const [resolvedTokenKeys, setResolvedTokenKeys] = useState({});
  const [loadingTokenKeys, setLoadingTokenKeys] = useState({});
  const keyRequestsRef = useRef({});
  const showWorkBuddy = isTextModel(modelData);

  const siteOrigin = useMemo(() => getSiteOrigin(), []);
  const configuredBaseUrl = useMemo(() => `${siteOrigin}/v1`, [siteOrigin]);
  const workBuddyEndpoint = `${siteOrigin}/v1/chat/completions`;
  const endpointCount = Array.isArray(modelData?.supported_endpoint_types)
    ? modelData.supported_endpoint_types.length
    : 0;
  const addressCount = 1 + endpointCount;
  const routeModelNames = useMemo(() => {
    const channelList = Array.isArray(modelData?.channel_list)
      ? modelData.channel_list
      : [];
    if (channelList.length === 0) {
      return modelData?.model_name ? [modelData.model_name] : [];
    }
    return channelList.map((channel) =>
      getChannelRouteModelName(modelData, channel),
    );
  }, [modelData]);

  useEffect(() => {
    if (!showWorkBuddy || !workBuddyVisible) {
      return;
    }
    let cancelled = false;
    (async () => {
      try {
        const res = await API.get('/api/token/?p=1&size=10', {
          skipErrorHandler: true,
        });
        const { success, data } = res.data || {};
        if (!success || cancelled) return;
        const items = Array.isArray(data) ? data : data?.items || [];
        setTokens(items.filter((token) => token.status === 1));
      } catch (e) {
        if (!cancelled) setTokens([]);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [showWorkBuddy, workBuddyVisible]);

  const getApiEndpointLink = (path) => {
    try {
      if (!path) return '';
      const normalizedPath = path.startsWith('/') ? path : `/${path}`;
      return `${siteOrigin}${normalizedPath}`;
    } catch (e) {
      return path;
    }
  };

  const copyEndpoint = async (path) => {
    const endpoint = getApiEndpointLink(path);
    if (await copy(endpoint)) {
      Toast.success({ content: t('已复制API端点') });
    } else {
      Toast.error({ content: t('复制失败') });
    }
  };

  const copyBaseUrl = async () => {
    if (await copy(configuredBaseUrl)) {
      Toast.success({ content: t('已复制BaseURL') });
    } else {
      Toast.error({ content: t('复制失败') });
    }
  };

  const copyText = async (text, successText = '已复制') => {
    if (await copy(text)) {
      Toast.success({ content: t(successText) });
    } else {
      Toast.error({ content: t('复制失败') });
    }
  };

  const copyModelName = async (modelName) => {
    await copyText(modelName, `模型${modelName}复制成功`);
  };

  const fetchTokenKey = async (token) => {
    const tokenId = token?.id;
    if (!tokenId) {
      throw new Error(t('令牌不存在'));
    }
    if (resolvedTokenKeys[tokenId]) {
      return resolvedTokenKeys[tokenId];
    }
    if (keyRequestsRef.current[tokenId]) {
      return keyRequestsRef.current[tokenId];
    }
    const request = (async () => {
      setLoadingTokenKeys((prev) => ({ ...prev, [tokenId]: true }));
      try {
        const fullKey = await fetchTokenKeyById(tokenId);
        const apiKey = `sk-${fullKey}`;
        setResolvedTokenKeys((prev) => ({ ...prev, [tokenId]: apiKey }));
        return apiKey;
      } finally {
        delete keyRequestsRef.current[tokenId];
        setLoadingTokenKeys((prev) => {
          const next = { ...prev };
          delete next[tokenId];
          return next;
        });
      }
    })();
    keyRequestsRef.current[tokenId] = request;
    return request;
  };

  const copyApiKey = async (token) => {
    try {
      const apiKey = await fetchTokenKey(token);
      await copyText(apiKey, '已复制API Key');
    } catch (e) {
      Toast.error({ content: e?.message || t('获取令牌密钥失败') });
    }
  };

  const getEndpointHelpLines = (type, path, method) => {
    const endpointText = getEndpointText(type, path);

    if (
      endpointText.includes('chat') ||
      endpointText.includes('completion') ||
      endpointText.includes('对话') ||
      endpointText.includes('聊天')
    ) {
      return [
        t(
          '聊天补全端点，OpenAI 兼容生态中最常用的接口，通常用于 OpenAI SDK、Cherry Studio、Chatbox、LobeChat、Dify、工作流编排工具和自建聊天应用。',
        ),
        t(
          '请求一般使用 POST，并在请求体里传入 model、messages、temperature、stream 等参数；支持流式输出时可用于打字机效果、长文本持续返回和实时助手场景。',
        ),
        t(
          '多数只要求填写 BaseURL 的工具会自动补全 /v1/chat/completions；如果工具要求填写完整接口地址，就复制这里显示的完整 URL。',
        ),
      ];
    }

    if (endpointText.includes('response')) {
      return [
        t(
          'Responses 端点是 OpenAI 新版统一响应接口，适合需要文本生成、多模态输入、工具调用或更统一响应结构的客户端和自建应用。',
        ),
        t(
          '如果你的工具明确支持 Responses API，可以优先选择这个端点；老版 OpenAI 兼容工具通常仍使用聊天补全端点。',
        ),
        t(
          '请求通常使用 POST，模型名称仍需填写通道路由模型名，API Key 使用本侧栏第三步复制的密钥。',
        ),
      ];
    }

    if (
      endpointText.includes('image') ||
      endpointText.includes('图像') ||
      endpointText.includes('图片')
    ) {
      return [
        t(
          '图像端点通常用于文生图、图生图、图片编辑或图片分析类能力，适合绘图工具、自动化工作流和自建视觉应用接入。',
        ),
        t(
          '不同模型支持的参数可能不同，例如 prompt、image、size、quality、response_format 等，请以模型文档或调用示例为准。',
        ),
        t(
          '如果客户端只支持 OpenAI 兼容图像接口，通常需要填写完整端点地址，并确认模型名称与通道路由模型名一致。',
        ),
      ];
    }

    if (
      endpointText.includes('audio') ||
      endpointText.includes('speech') ||
      endpointText.includes('transcription') ||
      endpointText.includes('音频') ||
      endpointText.includes('语音')
    ) {
      return [
        t(
          '音频端点通常用于语音转文字、文字转语音、翻译或音频理解，适合客服质检、会议纪要、语音助手和媒体处理工作流。',
        ),
        t(
          '这类接口常需要上传音频文件或指定 voice、input、format 等参数，部分工具会要求使用 multipart/form-data。',
        ),
        t(
          '接入前建议确认客户端是否支持对应音频接口格式；如果只支持普通聊天接口，请选择聊天补全端点。',
        ),
      ];
    }

    if (isVideoEndpoint(type, path)) {
      return [
        t(
          '视频端点通常用于视频生成、视频理解、分镜或任务型视频处理，适合创作工具、自动化生产流程和异步任务场景。',
        ),
        t(
          '视频能力往往不是一次请求立即返回最终结果，通常需要先创建视频任务，再轮询任务状态，最后下载或读取生成的视频内容。',
        ),
        t(
          '更适合 Hermes、自建脚本、自建前端/后端、HTTP 工作流节点、Postman、Apifox、curl，或明确支持 OpenAI Videos API / Sora 视频端点的工具。',
        ),
      ];
    }

    if (
      endpointText.includes('embedding') ||
      endpointText.includes('向量') ||
      endpointText.includes('嵌入')
    ) {
      return [
        t(
          '向量端点用于把文本转换为 embedding，常用于知识库检索、RAG、相似度搜索、聚类、推荐和语义匹配。',
        ),
        t(
          '接入时通常传入 input 和 model，返回的向量会写入向量数据库或检索系统，例如 Milvus、Qdrant、pgvector、Elasticsearch 等。',
        ),
        t(
          '如果你的工具是知识库或工作流工具，请在 embedding 模型配置里填写这个完整端点地址和对应 API Key。',
        ),
      ];
    }

    return [
      t(
        '该端点表示模型支持的一类具体 API 能力，完整请求地址由 BaseURL 加端点路径组成，适合需要手动填写接口地址的客户端、工作流工具或自建应用。',
      ),
      t(
        '调用时通常需要同时配置 API Key 和模型名称；模型名称建议使用第二步中的通道路由模型名，以便请求固定到指定渠道。',
      ),
      t(
        '如果工具只提供 BaseURL 输入框，通常填写 BaseURL 即可；如果工具提供接口地址或 Endpoint 输入框，请复制这里的完整 URL。',
      ),
      `${t('请求方法')}：${method || 'POST'}`,
    ];
  };

  const renderEndpointHelp = (type, path, method) => (
    <div className='max-w-[340px] text-xs leading-5'>
      <div className='font-medium mb-1'>{t('端点说明')}</div>
      {getEndpointHelpLines(type, path, method).map((line, idx) => (
        <div key={`${type}-help-${idx}`} className='mb-1 last:mb-0'>
          {line}
        </div>
      ))}
    </div>
  );

  const getEndpointUsageDescription = (type, path) => {
    if (isVideoEndpoint(type, path)) {
      return (
        <>
          {t(
            '视频端点用于创建视频生成任务，常见流程是提交 prompt、model、尺寸或时长等参数后轮询任务结果。适合',
          )}
          <ToolList
            names={[
              'Hermes',
              'Postman',
              'Apifox',
              'curl',
              'OpenAI Videos API / Sora',
            ]}
          />
          {t('等工具，也适合支持自定义 HTTP 请求的工作流工具或自建应用。')}
        </>
      );
    }

    const endpointText = getEndpointText(type, path);
    if (
      endpointText.includes('chat') ||
      endpointText.includes('completion') ||
      endpointText.includes('对话') ||
      endpointText.includes('聊天')
    ) {
      return (
        <>
          {t('聊天补全端点适合')}
          <ToolList
            names={[
              'OpenClaw',
              'WorkBuddy',
              'OpenAI SDK',
              'Cherry Studio',
              'Chatbox',
              'LobeChat',
              'Dify',
            ]}
          />
          {t(
            '等工具、工作流编排工具或自建服务。请配合第二步模型名字和第三步 API Key 一起使用。',
          )}
        </>
      );
    }

    return (
      <>
        {t('用于需要填写完整接口地址的工具和应用，例如')}
        <ToolList
          names={[
            'OpenClaw',
            'OpenAI SDK',
            'Cherry Studio',
            'Chatbox',
            'LobeChat',
            'Dify',
          ]}
        />
        {t(
          '等工具、HTTP 工作流工具或自建服务。请配合第二步模型名字和第三步 API Key 一起使用；若工具仅支持聊天补全，请优先选择 /v1/chat/completions。',
        )}
      </>
    );
  };

  const renderAPIEndpoints = () => {
    if (!modelData) return null;

    const mapping = endpointMap;
    const types = modelData.supported_endpoint_types || [];

    return types.map((type) => {
      const info = mapping[type] || {};
      let path = info.path || '';
      // 如果路径中包含 {model} 占位符，替换为真实模型名称
      if (path.includes('{model}')) {
        const modelName = modelData.model_name || modelData.modelName || '';
        path = path.replaceAll('{model}', modelName);
      }
      const method = info.method || 'POST';
      const endpointLink = path ? getApiEndpointLink(path) : '';
      return (
        <div
          key={type}
          className='rounded-xl border px-3 py-2 transition-all duration-200 hover:shadow-sm'
          style={{
            borderColor: 'var(--semi-color-border)',
            backgroundColor: 'var(--semi-color-bg-0)',
          }}
        >
          <div className='flex items-center justify-between gap-2 mb-1'>
            <span className='flex items-center min-w-0'>
              <Badge dot type='success' className='mr-2' />
              <Text strong ellipsis={{ showTooltip: true }}>
                {type}
              </Text>
              <Tooltip content={renderEndpointHelp(type, path, method)}>
                <IconHelpCircle
                  size={14}
                  className='ml-1 shrink-0 cursor-help'
                  style={{ color: 'var(--semi-color-text-2)' }}
                />
              </Tooltip>
            </span>
            {path && (
              <span className='text-gray-500 text-xs shrink-0'>{method}</span>
            )}
          </div>
          <div className='flex items-center gap-2'>
            <Text className='text-sm text-gray-500 break-all font-mono flex-1'>
              {endpointLink || t('暂无端点路径')}
            </Text>
            {path && (
              <Tooltip content={endpointLink}>
                <Button
                  size='small'
                  type='primary'
                  theme='light'
                  icon={<IconCopy />}
                  onClick={() => copyEndpoint(path)}
                  aria-label={t('复制API端点')}
                >
                  {t('复制')}
                </Button>
              </Tooltip>
            )}
          </div>
          <AddressUsageNote title={t('用途说明')}>
            {getEndpointUsageDescription(type, path)}
          </AddressUsageNote>
        </div>
      );
    });
  };

  return (
    <Card className='!rounded-2xl shadow-sm border-0 mb-6'>
      <StepTitle
        label={t('第一步')}
        title={t('接口地址')}
        desc={t('按工具要求复制 BaseURL 或完整 API 端点即可。')}
        icon={<IconLink size={14} />}
        extra={
          <Tag size='small' shape='circle' color='blue' type='light'>
            {t('{{count}}选1复制使用', { count: addressCount })}
          </Tag>
        }
      />
      {/* {showWorkBuddy ? (
        <div
          className='relative inline-flex mb-3'
          style={{
            filter: 'drop-shadow(0 10px 18px rgba(245, 158, 11, 0.22))',
          }}
        >
          <Button
            theme='solid'
            type='warning'
            className='!rounded-full !font-semibold'
            style={{
              color: '#fff',
              background:
                'linear-gradient(135deg, #f59e0b 0%, #ef4444 52%, #ec4899 100%)',
              border: '0',
              paddingLeft: 10,
              paddingRight: 12,
            }}
            onClick={() => setWorkBuddyVisible(true)}
          >
            <span className='inline-flex items-center gap-1.5 leading-none'>
              <img
                src='/workBuddy.svg'
                alt=''
                className='w-7 h-7 rounded-full bg-white/90 p-0.5'
              />
              <span>{t('WorkBuddy 快捷接入')}</span>
            </span>
          </Button>
        </div>
      ) : null} */}
      <div className='rounded-xl bg-gray-50'>
        <div className='space-y-2'>
          <div
            className='rounded-xl border px-3 py-2 transition-all duration-200 hover:shadow-sm'
            style={{
              borderColor: 'var(--semi-color-border)',
              backgroundColor: 'var(--semi-color-bg-0)',
            }}
          >
            <div className='flex items-center justify-between gap-2 mb-1'>
              <span className='flex items-center min-w-0'>
                <Badge dot type='primary' className='mr-2' />
                <Text strong>{t('BaseURL')}</Text>
              </span>
              <span className='text-gray-500 text-xs shrink-0'>
                {t('基础地址')}
              </span>
            </div>
            <div className='flex items-center gap-2'>
              <Text className='text-sm text-gray-500 break-all font-mono flex-1'>
                {configuredBaseUrl}
              </Text>
              <Tooltip content={configuredBaseUrl}>
                <Button
                  size='small'
                  type='primary'
                  theme='light'
                  icon={<IconCopy />}
                  onClick={copyBaseUrl}
                  aria-label={t('复制BaseURL')}
                >
                  {t('复制')}
                </Button>
              </Tooltip>
            </div>
            <AddressUsageNote title={t('用途说明')}>
              {t(
                'BaseURL 是 OpenAI 兼容客户端的基础地址，路径固定为 /v1。通常用于',
              )}
              <ToolList
                names={[
                  'OpenClaw',
                  'WorkBuddy',
                  'OpenAI SDK',
                  'Cherry Studio',
                  'Chatbox',
                  'LobeChat',
                  'Dify',
                ]}
              />
              {t('等工具、工作流工具或自建应用的服务地址配置。')}
            </AddressUsageNote>
          </div>
          {renderAPIEndpoints()}
        </div>
      </div>
      <Modal
        title={
          <div className='flex items-center gap-2 relative'>
            <span>{t('WorkBuddy 快捷接入')}</span>
          </div>
        }
        visible={workBuddyVisible}
        onCancel={() => setWorkBuddyVisible(false)}
        footer={null}
        centered
      >
        <div className='space-y-3 pb-2'>
          <div className='rounded-xl border p-3 border-semi-color-border'>
            <ModalStepTitle label={t('第一步')} title={t('接口地址')} />
            <div className='flex items-center gap-2'>
              <Text className='font-mono text-sm break-all flex-1'>
                {workBuddyEndpoint}
              </Text>
              <Button
                size='small'
                type='primary'
                theme='light'
                icon={<IconCopy />}
                onClick={() => copyText(workBuddyEndpoint, '已复制API端点')}
                aria-label={t('复制API端点')}
              >
                {t('复制')}
              </Button>
            </div>
          </div>
          <div className='rounded-xl border p-3 border-semi-color-border'>
            <ModalStepTitle label={t('第二步')} title={t('API Key')} />
            {tokens.length > 0 ? (
              <div className='space-y-2'>
                {tokens.map((token) => (
                  <div
                    key={token.id}
                    className='flex items-center gap-2 rounded-lg px-3 py-2'
                    style={{ backgroundColor: 'var(--semi-color-fill-0)' }}
                  >
                    <Text
                      strong
                      ellipsis={{ showTooltip: true }}
                      className='flex-1'
                    >
                      {token.name || `${t('令牌')} #${token.id}`}
                    </Text>
                    <Button
                      size='small'
                      type='primary'
                      theme='light'
                      icon={<IconCopy />}
                      loading={!!loadingTokenKeys[token.id]}
                      onClick={() => copyApiKey(token)}
                    >
                      {t('复制API Key')}
                    </Button>
                  </div>
                ))}
              </div>
            ) : (
              <Text type='secondary'>{t('暂无令牌')}</Text>
            )}
          </div>
          <div className='rounded-xl border p-3 mb-2 border-semi-color-border'>
            <ModalStepTitle label={t('第三步')} title={t('模型名称')} />
            <div className='space-y-2'>
              {routeModelNames.map((modelName) => (
                <div
                  key={modelName}
                  className='flex items-center gap-2 rounded-lg px-3 py-2'
                  style={{ backgroundColor: 'var(--semi-color-fill-0)' }}
                >
                  <Text
                    className='font-mono text-sm flex-1'
                    ellipsis={{ showTooltip: true }}
                  >
                    {modelName}
                  </Text>
                  <Button
                    size='small'
                    type='primary'
                    theme='light'
                    icon={<IconCopy />}
                    onClick={() => copyModelName(modelName)}
                    aria-label={t('复制模型名字')}
                  >
                    {t('复制')}
                  </Button>
                </div>
              ))}
            </div>
          </div>
          <div
            className='rounded-xl px-3 py-2 text-xs leading-5'
            style={{
              color: 'var(--semi-color-text-2)',
              backgroundColor: 'var(--semi-color-fill-0)',
            }}
          >
            {t(
              '说明：如果 WorkBuddy 添加自定义模型时选择了「高级配置」中的「自定义协议」，开启后将直接使用填写的接口地址，不再自动补全 /chat/completions 路径；这种情况下接口地址填写到 /v1 即可，例如：{{url}}。',
              { url: configuredBaseUrl },
            )}
          </div>
        </div>
      </Modal>
    </Card>
  );
};

export default ModelEndpoints;
