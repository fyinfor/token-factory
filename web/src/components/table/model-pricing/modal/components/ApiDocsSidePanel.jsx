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

import React, { useMemo } from 'react';
import {
  SideSheet,
  Card,
  Avatar,
  Typography,
  Button,
  Tag,
  Table,
  Space,
  Toast,
} from '@douyinfe/semi-ui';
import { IconCode, IconCopy, IconClose } from '@douyinfe/semi-icons';
import { useIsMobile } from '../../../../../hooks/common/useIsMobile';
import { getServerAddress } from '../../../../../helpers/token';

const { Text, Title } = Typography;

const normalizeApiBaseUrl = (baseUrl) =>
  String(baseUrl || '').replace(
    /^https:\/\/demo\.tokenfactoryopen\.com/i,
    'https://tokenfactoryopen.com',
  );

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

const Section = ({ title, description, children }) => (
  <Card className='!rounded-2xl shadow-sm border-0 mb-3'>
    <div className='mb-3'>
      <Text className='text-base font-medium'>{title}</Text>
      {description && (
        <div className='text-xs text-gray-500 mt-0.5'>{description}</div>
      )}
    </div>
    {children}
  </Card>
);

const getEndpointDocConfig = (endpointPath, modelName, t) => {
  switch (endpointPath) {
    case '/v1/messages':
      return {
        title: t('Messages 接口请求示例'),
        description: t('Claude 兼容的 Messages 接口'),
        body: {
          model: modelName || '',
          max_tokens: 1024,
          messages: [
            {
              role: 'user',
              content: 'Hello!',
            },
          ],
        },
        nonStreamResponse: {
          id: 'msg_xxxx',
          type: 'message',
          role: 'assistant',
          model: modelName || '',
          content: [
            {
              type: 'text',
              text: 'Hello! How can I assist you today?',
            },
          ],
          stop_reason: 'end_turn',
        },
        params: [
          {
            key: 'model',
            name: 'model',
            type: 'string',
            required: true,
            desc: t('要使用的模型 ID，需为当前通道支持的模型名称。'),
          },
          {
            key: 'max_tokens',
            name: 'max_tokens',
            type: 'integer',
            required: true,
            desc: t('生成内容的最大 token 数。'),
          },
          {
            key: 'messages',
            name: 'messages',
            type: 'array',
            required: true,
            desc: t('对话消息列表，每条消息包含 role 和 content 字段。'),
          },
        ],
      };
    case '/v1/images/generations':
      return {
        title: t('Images Generations 接口请求示例'),
        description: t('OpenAI 兼容的图片生成接口'),
        body: {
          model: modelName || '',
          prompt: 'A cute cat in watercolor style',
          size: '1024x1024',
          n: 1,
        },
        nonStreamResponse: {
          created: 1741569952,
          data: [
            {
              url: 'https://example.com/image.png',
            },
          ],
        },
        params: [
          {
            key: 'model',
            name: 'model',
            type: 'string',
            required: true,
            desc: t('要使用的模型 ID，需为当前通道支持的模型名称。'),
          },
          {
            key: 'prompt',
            name: 'prompt',
            type: 'string',
            required: true,
            desc: t('用于生成图片的文本描述。'),
          },
          {
            key: 'size',
            name: 'size',
            type: 'string',
            required: false,
            desc: t('生成图片尺寸，例如 1024x1024。'),
          },
        ],
      };
    case '/v1/videos/generations':
    case '/v1/videos':
      return {
        title: t('Videos 接口请求示例'),
        description: t('视频生成接口'),
        body: {
          model: modelName || '',
          prompt: 'A cinematic shot of a futuristic city',
        },
        nonStreamResponse: {
          id: 'video_xxxx',
          object: 'video.generation',
          status: 'queued',
          model: modelName || '',
        },
        params: [
          {
            key: 'model',
            name: 'model',
            type: 'string',
            required: true,
            desc: t('要使用的模型 ID，需为当前通道支持的模型名称。'),
          },
          {
            key: 'prompt',
            name: 'prompt',
            type: 'string',
            required: true,
            desc: t('用于生成视频的文本描述。'),
          },
        ],
      };
    case '/v1/chat/completions':
    default:
      return {
        title: t('Chat Completions 接口请求示例'),
        description: t('OpenAI 兼容的 Chat Completions 接口'),
        body: {
          model: modelName || '',
          stream: true,
          messages: [
            {
              role: 'system',
              content: 'You are a helpful assistant.',
            },
            {
              role: 'user',
              content: 'Hello!',
            },
          ],
        },
        nonStreamResponse: {
          id: 'xxxx',
          object: 'chat.completion',
          created: 1741569952,
          model: modelName || '',
          choices: [
            {
              index: 0,
              message: {
                role: 'assistant',
                content: 'Hello! How can I assist you today?',
                refusal: null,
                annotations: [],
              },
              finish_reason: 'stop',
            },
          ],
        },
        streamResponse: {
          id: 'xxxx',
          object: 'chat.completion',
          created: 1741569952,
          model: modelName || '',
          choices: [
            {
              index: 0,
              delta: {
                role: 'assistant',
                content: 'Hello! How can I assist you today?',
                refusal: null,
                annotations: [],
              },
              finish_reason: 'stop',
            },
          ],
        },
        params: [
          {
            key: 'model',
            name: 'model',
            type: 'string',
            required: true,
            desc: t('要使用的模型 ID，需为当前通道支持的模型名称。'),
          },
          {
            key: 'stream',
            name: 'stream',
            type: 'boolean',
            required: false,
            desc: t(
              '是否启用流式响应。为 true 时通过 SSE 持续返回增量内容；为 false 时一次性返回完整结果。',
            ),
          },
          {
            key: 'messages',
            name: 'messages',
            type: 'array',
            required: true,
            desc: t(
              '对话消息列表，每条消息包含 role（system/user/assistant）和 content 字段。',
            ),
          },
        ],
      };
  }
};

const ApiDocsSidePanel = ({
  visible,
  onClose,
  modelName,
  endpointPath = '/v1/chat/completions',
  t,
}) => {
  const isMobile = useIsMobile();
  const docConfig = useMemo(
    () => getEndpointDocConfig(endpointPath, modelName, t),
    [endpointPath, modelName, t],
  );

  const serverAddress = useMemo(() => {
    try {
      return normalizeApiBaseUrl(getServerAddress()).replace(/\/$/, '');
    } catch (e) {
      return '';
    }
  }, [visible]);

  const endpoint = `${serverAddress}${endpointPath}`;

  const headersJson = useMemo(
    () =>
      JSON.stringify(
        {
          'Content-Type': 'application/json',
          Authorization: 'Bearer apikey',
        },
        null,
        2,
      ),
    [],
  );

  const requestBodyJson = useMemo(
    () => JSON.stringify(docConfig.body, null, 2),
    [docConfig.body],
  );

  const nonStreamResponseJson = useMemo(
    () => JSON.stringify(docConfig.nonStreamResponse, null, 2),
    [docConfig.nonStreamResponse],
  );

  const streamResponseJson = useMemo(
    () =>
      JSON.stringify(
        docConfig.streamResponse || docConfig.nonStreamResponse,
        null,
        2,
      ),
    [docConfig.streamResponse, docConfig.nonStreamResponse],
  );

  const paramColumns = [
    {
      title: t('参数'),
      dataIndex: 'name',
      width: 120,
      render: (val) => (
        <Tag color='blue' size='small' shape='circle'>
          {val}
        </Tag>
      ),
    },
    {
      title: t('类型'),
      dataIndex: 'type',
      width: 100,
      render: (val) => (
        <Text type='tertiary' size='small'>
          {val}
        </Text>
      ),
    },
    {
      title: t('必填'),
      dataIndex: 'required',
      width: 80,
      render: (val) =>
        val ? (
          <Tag color='red' size='small' shape='circle'>
            {t('是')}
          </Tag>
        ) : (
          <Tag color='grey' size='small' shape='circle'>
            {t('否')}
          </Tag>
        ),
    },
    {
      title: t('说明'),
      dataIndex: 'desc',
      render: (val) => <Text size='small'>{val}</Text>,
    },
  ];

  const paramData = docConfig.params;

  const curlExample = useMemo(
    () =>
      [
        `curl -X POST "${endpoint}" \\`,
        `  -H "Content-Type: application/json" \\`,
        `  -H "Authorization: Bearer apikey" \\`,
        `  -d '${requestBodyJson.replace(/'/g, `'\\''`)}'`,
      ].join('\n'),
    [endpoint, requestBodyJson],
  );

  const content = (
    <div className='p-2 space-y-3'>
      <Card className='!rounded-2xl shadow-sm border-0 mb-3'>
        <div className='flex items-center mb-3'>
          <Avatar size='small' color='cyan' className='mr-2 shadow-md'>
            <IconCode size={16} />
          </Avatar>
          <div className='flex-1'>
            <Text className='text-lg font-medium'>{t('在线 API 文档')}</Text>
            <div className='text-xs text-gray-600'>
              {docConfig.title}
              {modelName && (
                <>
                  <span className='mx-1'>·</span>
                  <Tag color='blue' size='small' shape='circle'>
                    {modelName}
                  </Tag>
                </>
              )}
            </div>
          </div>
        </div>
      </Card>

      <Section title={t('接口地址')} description={docConfig.description}>
        <Space className='mb-2'>
          <Tag color='green' size='small' shape='circle'>
            POST
          </Tag>
        </Space>
        <CodeBlock content={endpoint} language='text' t={t} />
      </Section>

      <Section
        title={t('请求头 Headers')}
        description={t('请将 apikey 替换为您实际的令牌密钥')}
      >
        <CodeBlock content={headersJson} language='json' t={t} />
      </Section>

      <Section title={t('请求参数 (JSON)')}>
        <CodeBlock content={requestBodyJson} language='json' t={t} />
      </Section>

      <Section title={t('参数说明')}>
        <Table
          columns={paramColumns}
          dataSource={paramData}
          pagination={false}
          size='small'
        />
      </Section>

      <Section title={t('cURL 示例')}>
        <CodeBlock content={curlExample} language='bash' t={t} />
      </Section>

      <Section
        title={t('非流式响应 (JSON)')}
        description={t('当 stream=false 时一次性返回完整结果')}
      >
        <CodeBlock content={nonStreamResponseJson} language='json' t={t} />
      </Section>

      {docConfig.streamResponse && (
        <Section
          title={t('流式响应 (JSON)')}
          description={t(
            '当 stream=true 时，通过 SSE 持续返回多个 data: 事件，每个 chunk 结构如下',
          )}
        >
          <CodeBlock content={streamResponseJson} language='json' t={t} />
        </Section>
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
          </Space>
        }
        visible={visible}
        onCancel={onClose}
        width='100%'
        bodyStyle={{ padding: 0 }}
        closeIcon={
          <Button
            className='semi-button-tertiary semi-button-size-small semi-button-borderless'
            type='button'
            icon={<IconClose />}
            onClick={onClose}
          />
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
        width: 600,
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
