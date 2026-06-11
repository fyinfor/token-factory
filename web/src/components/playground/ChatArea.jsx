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

import React, { useCallback, useMemo, useState } from 'react';
import {
  AIChatDialogue,
  AIChatInput,
  Button,
  Card,
  Toast,
  Typography,
} from '@douyinfe/semi-ui';
import { MessageSquare, Eye, EyeOff, MessageSquarePlus } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { usePlayground } from '../../contexts/PlaygroundContext';
import {
  PLAYGROUND_MEDIA_MAX_HEIGHT,
  PLAYGROUND_MEDIA_MAX_WIDTH,
  PLAYGROUND_MEDIA_MAX_WIDTH_PX,
} from '../../constants/playground.constants';
import { usePlaygroundMediaMaxHeightPx } from '../../hooks/playground/usePlaygroundMediaMaxHeight';

const getConstrainedMediaSize = (
  dimensions,
  maxWidth = PLAYGROUND_MEDIA_MAX_WIDTH_PX,
  maxHeight,
) => {
  const width = Number(dimensions?.width || 0);
  const height = Number(dimensions?.height || 0);
  if (!width || !height) return {};
  const resolvedMaxHeight = Number(maxHeight || 640);
  const ratio = Math.min(maxWidth / width, resolvedMaxHeight / height, 1);
  return {
    width: Math.round(width * ratio),
    height: Math.round(height * ratio),
  };
};

const getVideoUrl = (item) =>
  item?.video_url || item?.file_url || item?.url || item?.src || '';

const getInputText = (messageContent) => {
  if (typeof messageContent === 'string') {
    return messageContent;
  }

  if (!messageContent || typeof messageContent !== 'object') {
    return '';
  }

  const inputContents = Array.isArray(messageContent.inputContents)
    ? messageContent.inputContents
    : [];

  return inputContents
    .map((item) => {
      if (typeof item?.text === 'string') {
        return item.text;
      }
      if (typeof item?.content === 'string') {
        return item.content;
      }
      return '';
    })
    .join('')
    .trim();
};

const toDialogueStatus = (status) => {
  switch (status) {
    case 'loading':
      return 'in_progress';
    case 'error':
      return 'failed';
    case 'complete':
      return 'completed';
    default:
      return status;
  }
};

const restorePlaygroundMessage = (message) => {
  if (!message || typeof message !== 'object') {
    return message;
  }

  const restored = { ...message };
  if (Object.prototype.hasOwnProperty.call(restored, 'playgroundContent')) {
    restored.content = restored.playgroundContent;
    delete restored.playgroundContent;
  }
  if (Object.prototype.hasOwnProperty.call(restored, 'playgroundStatus')) {
    restored.status = restored.playgroundStatus;
    delete restored.playgroundStatus;
  }
  return restored;
};

const normalizeDialogueContent = (message) => {
  const { content, role, videoTask } = message || {};
  const normalizedContent = [];

  if (Array.isArray(content)) {
    content.forEach((item) => {
      if (item?.type === 'text') {
        normalizedContent.push({
          type: 'input_text',
          text: item.text || '',
        });
      } else if (item?.type === 'image_url' && item.image_url?.url) {
        normalizedContent.push({
          type: 'input_image',
          image_url: item.image_url.url,
          detail: item.image_url?.detail || 'auto',
        });
      }
    });
  } else if (typeof content === 'string' && content.trim()) {
    normalizedContent.push({
      type: 'input_text',
      text: content,
    });
  }

  if (videoTask?.playableUrl) {
    normalizedContent.push({
      type: 'playground_video',
      video_url: videoTask.playableUrl,
      filename: 'generated-video.mp4',
    });
  }

  if (normalizedContent.length === 0) {
    return content;
  }

  return [
    {
      type: 'message',
      role,
      content: normalizedContent,
    },
  ];
};

const ChatArea = ({
  chatRef,
  message,
  inputs,
  styleState,
  showDebugPanel,
  roleInfo,
  onMessageSend,
  onMessageReset,
  onChatsChange,
  onMediaDimensionsChange,
  onStopGenerator,
  onClearMessages,
  onToggleDebugPanel,
}) => {
  const { t } = useTranslation();
  const { onPasteImage, imageEnabled } = usePlayground();
  const mediaMaxHeightPx = usePlaygroundMediaMaxHeightPx();
  const [inputFocused, setInputFocused] = useState(false);

  const safeChats = useMemo(() => {
    const list = Array.isArray(message) ? message : [];
    const seen = new Set();
    return list.map((item, index) => {
      const baseId =
        item?.id !== undefined && item?.id !== null && String(item.id) !== ''
          ? String(item.id)
          : `chat-${index}`;
      let nextId = baseId;
      if (seen.has(nextId)) {
        nextId = `${baseId}-${index}`;
      }
      seen.add(nextId);
      const dialogueStatus = toDialogueStatus(item?.status);
      const dialogueContent = normalizeDialogueContent(item);
      return {
        ...item,
        id: nextId,
        ...(dialogueStatus !== item?.status
          ? {
              status: dialogueStatus,
              playgroundStatus: item?.status,
            }
          : {}),
        ...(dialogueContent !== item?.content
          ? {
              content: dialogueContent,
              playgroundContent: item?.content,
            }
          : {}),
      };
    });
  }, [message]);

  const generating = useMemo(
    () =>
      safeChats.some((item) =>
        ['loading', 'incomplete', 'in_progress'].includes(
          item?.playgroundStatus || item?.status,
        ),
      ),
    [safeChats],
  );

  const handleInputMessageSend = useCallback(
    (messageContent) => {
      const content = getInputText(messageContent);
      if (!content) {
        return;
      }
      onMessageSend(content, messageContent?.attachments);
    },
    [onMessageSend],
  );

  const handlePaste = useCallback(
    (event) => {
      const items = event.clipboardData?.items;
      if (!items) return;

      for (let i = 0; i < items.length; i++) {
        const item = items[i];
        if (!item.type?.includes('image')) continue;

        event.preventDefault();
        const file = item.getAsFile();
        if (!file) return;

        if (!imageEnabled) {
          Toast.warning({
            content: t('请先在设置中启用图片功能'),
            duration: 3,
          });
          return;
        }

        const reader = new FileReader();
        reader.onload = (readerEvent) => {
          const base64 = readerEvent.target.result;
          if (onPasteImage) {
            onPasteImage(base64);
            Toast.success({
              content: t('图片已添加'),
              duration: 2,
            });
          } else {
            Toast.error({
              content: t('无法添加图片'),
              duration: 2,
            });
          }
        };
        reader.onerror = () => {
          Toast.error({
            content: t('粘贴图片失败'),
            duration: 2,
          });
        };
        reader.readAsDataURL(file);
        return;
      }
    },
    [imageEnabled, onPasteImage, t],
  );

  const dialogueRenderConfig = useMemo(
    () => ({
      renderDialogueAction: ({ className, defaultActionsObj }) => {
        const actions = [
          ['copy', defaultActionsObj?.copyNode],
          ['reset', defaultActionsObj?.resetNode],
          ['more', defaultActionsObj?.moreNode],
        ]
          .filter(([, node]) => Boolean(node))
          .map(([key, node]) =>
            React.isValidElement(node) ? (
              React.cloneElement(node, { key })
            ) : (
              <React.Fragment key={key}>{node}</React.Fragment>
            ),
          );

        return <div className={className}>{actions}</div>;
      },
      renderDialogueTitle: () => null,
    }),
    [],
  );

  const renderDialogueContentItem = useMemo(
    () => ({
      playground_video: (item, currentMessage) => {
        const videoUrl = getVideoUrl(item);
        if (!videoUrl) return null;

        const dimensions = currentMessage?.mediaDimensions?.[videoUrl];
        const constrainedVideoSize = getConstrainedMediaSize(
          dimensions,
          PLAYGROUND_MEDIA_MAX_WIDTH_PX,
          mediaMaxHeightPx,
        );

        return (
          <div className='playground-ai-video'>
            <video
              controls
              preload='metadata'
              src={videoUrl}
              className='playground-ai-video-player'
              style={{
                width: constrainedVideoSize.width
                  ? `${constrainedVideoSize.width}px`
                  : '100%',
                height: constrainedVideoSize.height
                  ? `${constrainedVideoSize.height}px`
                  : 'auto',
                maxWidth: PLAYGROUND_MEDIA_MAX_WIDTH,
                maxHeight: PLAYGROUND_MEDIA_MAX_HEIGHT,
              }}
              onLoadedMetadata={(event) => {
                const nextDimensions = {
                  width: event.currentTarget.videoWidth,
                  height: event.currentTarget.videoHeight,
                };
                onMediaDimensionsChange?.(
                  currentMessage?.id,
                  videoUrl,
                  nextDimensions,
                );
              }}
            />
          </div>
        );
      },
    }),
    [mediaMaxHeightPx, onMediaDimensionsChange],
  );

  const handleChatsChange = useCallback(
    (chats) => {
      const restoredChats = Array.isArray(chats)
        ? chats.map(restorePlaygroundMessage)
        : [];
      onChatsChange?.(restoredChats);
    },
    [onChatsChange],
  );

  return (
    <Card
      className='h-full'
      bordered={false}
      bodyStyle={{
        padding: 0,
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
      }}
    >
      {/* 聊天头部 */}
      {styleState.isMobile ? (
        <div className='pt-4'></div>
      ) : (
        <div className='px-6 py-4 bg-gradient-to-r from-purple-500 to-blue-500 rounded-t-2xl'>
          <div className='flex items-center justify-between'>
            <div className='flex items-center gap-3'>
              <div className='w-10 h-10 rounded-full bg-white/20 backdrop-blur flex items-center justify-center'>
                <MessageSquare size={20} className='text-white' />
              </div>
              <div>
                <Typography.Title heading={5} className='!text-white mb-0'>
                  {t('AI 对话')}
                </Typography.Title>
                <Typography.Text className='!text-white/80 text-sm hidden sm:inline'>
                  {inputs.model || t('选择模型开始对话')}
                </Typography.Text>
              </div>
            </div>
            <div className='flex items-center gap-2'>
              <Button
                icon={<MessageSquarePlus size={14} className='text-white' />}
                onClick={onClearMessages}
                theme='solid'
                size='small'
                className='!rounded-lg !border !border-blue-800 !bg-blue-600 !text-white !min-w-0 !font-medium !px-3 !shadow-none dark:!shadow-[0_0_18px_rgba(59,130,246,0.65),inset_0_1px_0_rgba(255,255,255,0.14)] hover:!bg-blue-500 hover:!border-blue-600 hover:!shadow-none dark:hover:!shadow-[0_0_24px_rgba(96,165,250,0.85)] active:!bg-blue-700 transition-[box-shadow,background-color,border-color] duration-200 [&_.semi-button-content]:!text-white [&_.semi-icon]:!text-white'
              >
                {t('新对话')}
              </Button>
              <Button
                icon={showDebugPanel ? <EyeOff size={14} /> : <Eye size={14} />}
                onClick={onToggleDebugPanel}
                theme='borderless'
                type='primary'
                size='small'
                className='!rounded-lg !text-white/80 hover:!text-white hover:!bg-white/10'
              >
                {showDebugPanel ? t('隐藏调试') : t('显示调试')}
              </Button>
            </div>
          </div>
        </div>
      )}

      {/* 聊天内容区域 */}
      <div className='flex min-h-0 flex-1 flex-col overflow-hidden'>
        <AIChatDialogue
          ref={chatRef}
          align='leftRight'
          mode='bubble'
          roleConfig={roleInfo}
          dialogueRenderConfig={dialogueRenderConfig}
          renderDialogueContentItem={renderDialogueContentItem}
          chats={safeChats}
          onChatsChange={handleChatsChange}
          onMessageReset={onMessageReset}
          className='playground-ai-dialogue'
          style={{
            flex: 1,
            minHeight: 0,
            maxWidth: '100%',
            overflow: 'hidden',
          }}
        />
        <div className='relative flex-shrink-0 p-2 sm:p-4'>
          <div
            className={`pointer-events-none absolute left-2 right-2 sm:left-4 sm:right-4 transition-all duration-300 ease-out ${
              inputFocused ? 'top-[-30px] opacity-100' : 'top-0 opacity-0'
            }`}
          >
            <div
              className='inline-block rounded-xl sm:rounded-2xl px-3 py-2 text-xs sm:text-sm text-gray-600 dark:text-gray-300 shadow-md'
              style={{
                border: '1px solid var(--semi-color-border)',
                backgroundColor: 'var(--semi-color-bg-1)',
              }}
            >
              {t('使用操练场会产生扣费，请确认模型与参数后再发送。')}
            </div>
          </div>
          <AIChatInput
            className='playground-ai-input'
            style={{
              width: '100%',
            }}
            generating={generating}
            showUploadButton={false}
            showUploadFile={false}
            showReference={false}
            sendHotKey='enter'
            clearContentOnGenerating
            onPaste={handlePaste}
            onFocus={() => setInputFocused(true)}
            onBlur={() => setInputFocused(false)}
            onMessageSend={handleInputMessageSend}
            onStopGenerate={onStopGenerator}
            placeholder={t('请输入您的问题...')}
          />
        </div>
      </div>
    </Card>
  );
};

export default ChatArea;
