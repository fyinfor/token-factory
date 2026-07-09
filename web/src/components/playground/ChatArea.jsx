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

import React, {
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import {
  AIChatDialogue,
  AIChatInput,
  Button,
  Card,
  Image,
  Toast,
  Typography,
} from '@douyinfe/semi-ui';
import { IconRefresh, IconUploadError } from '@douyinfe/semi-icons';
import { MessageSquare, Eye, EyeOff, MessageSquarePlus } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import StableDialogueMarkdown from './StableDialogueMarkdown';
import { usePlayground } from '../../contexts/PlaygroundContext';
import { listMaterialAssets } from '../../helpers/materialApi';
import {
  buildAssetMap,
  isAssetUri,
  resolveAssetUriToUrl,
} from '../../helpers/materialAssetUtils';
import { processIncompleteThinkTags } from '../../helpers';
import {
  resolveMessageGeneratedImages,
  stripGeneratedImageMarkdown,
} from '../../helpers/playgroundImageUtils';
import {
  PLAYGROUND_MEDIA_MAX_HEIGHT,
  PLAYGROUND_MEDIA_MAX_WIDTH,
  PLAYGROUND_MEDIA_MAX_WIDTH_PX,
  PLAYGROUND_DIALOGUE_REFERENCE_IMAGE_MAX_WIDTH,
  PLAYGROUND_DIALOGUE_REFERENCE_IMAGE_MAX_HEIGHT,
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

const AUTO_SCROLL_BOTTOM_GAP = 48;
const AUTO_SCROLL_RESUME_BOTTOM_GAP = 8;
const AUTO_SCROLL_RENDER_SETTLE_MS = 80;
const TOUCH_SCROLL_DIRECTION_THRESHOLD = 4;

/** 操练场 Markdown 图片：避免 Semi 默认 50% 宽，并保留点击预览 */
const PlaygroundDialogueMarkdownImage = ({
  src,
  alt,
  style,
  className,
  imgStyle,
  ...rest
}) => (
  <Image
    {...rest}
    src={src}
    alt={alt || ''}
    preview
    fallback={<IconUploadError />}
    className={
      className
        ? `${className} playground-dialogue-markdown-img`
        : 'playground-dialogue-markdown-img'
    }
    style={{
      display: 'inline-block',
      width: 'auto',
      height: 'auto',
      maxWidth: PLAYGROUND_MEDIA_MAX_WIDTH,
      maxHeight: PLAYGROUND_MEDIA_MAX_HEIGHT,
      ...style,
    }}
    imgStyle={{
      display: 'block',
      width: 'auto',
      height: 'auto',
      maxWidth: PLAYGROUND_MEDIA_MAX_WIDTH,
      maxHeight: PLAYGROUND_MEDIA_MAX_HEIGHT,
      objectFit: 'contain',
      borderRadius: 8,
      ...imgStyle,
    }}
  />
);

const resolvePlaygroundMediaUrl = (url, assetMap) => {
  const rawUrl = String(url || '').trim();
  if (!rawUrl) return '';
  return assetMap ? resolveAssetUriToUrl(rawUrl, assetMap) : rawUrl;
};

const createPlaygroundMarkdownRenderProps = (assetMap) => ({
  components: {
    img: ({ src, ...rest }) => (
      <PlaygroundDialogueMarkdownImage
        {...rest}
        src={resolvePlaygroundMediaUrl(src, assetMap)}
      />
    ),
  },
});

const messageHasAssetUri = (message) => {
  const rawContent = message?.playgroundContent ?? message?.content;
  if (Array.isArray(rawContent)) {
    return rawContent.some(
      (item) => item?.type === 'image_url' && isAssetUri(item?.image_url?.url),
    );
  }
  if (typeof rawContent === 'string') {
    return isAssetUri(rawContent);
  }
  return false;
};

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

const appendTextContent = (normalizedContent, text, reasoningContentRef) => {
  const processed = processIncompleteThinkTags(
    text || '',
    reasoningContentRef.value,
  );
  reasoningContentRef.value = processed.reasoningContent || '';

  if (processed.content?.trim()) {
    normalizedContent.push({
      type: 'input_text',
      text: processed.content,
    });
  }
};

const normalizeDialogueContent = (message, assetMap) => {
  const { role, videoTask } = message || {};
  const rawContent = message?.playgroundContent ?? message?.content;
  const generatedImages = resolveMessageGeneratedImages(message);
  const normalizedContent = [];
  const reasoningContentRef = {
    value: message?.reasoningContent || '',
  };

  if (Array.isArray(rawContent)) {
    rawContent.forEach((item) => {
      if (item?.type === 'text') {
        appendTextContent(
          normalizedContent,
          item.text || '',
          reasoningContentRef,
        );
      } else if (item?.type === 'image_url' && item.image_url?.url) {
        normalizedContent.push({
          type: 'input_image',
          image_url: resolvePlaygroundMediaUrl(item.image_url.url, assetMap),
          detail: item.image_url?.detail || 'auto',
        });
      }
    });
  } else if (typeof rawContent === 'string' && rawContent.trim()) {
    const displayText = stripGeneratedImageMarkdown(rawContent);
    if (displayText.trim()) {
      appendTextContent(normalizedContent, displayText, reasoningContentRef);
    }
  }

  // 图片生成结果（含 b64_json 解码后的 data URL）在对话区以图片气泡展示
  for (const src of generatedImages) {
    if (!src) continue;
    normalizedContent.push({
      type: 'input_image',
      image_url: resolvePlaygroundMediaUrl(src, assetMap),
      detail: 'auto',
    });
  }

  if (videoTask?.playableUrl) {
    normalizedContent.push({
      type: 'playground_video',
      video_url: resolvePlaygroundMediaUrl(videoTask.playableUrl, assetMap),
      filename: 'generated-video.mp4',
    });
  }

  const reasoningContent = reasoningContentRef.value?.trim();
  const normalizedMessages = [];
  if (reasoningContent) {
    normalizedMessages.push({
      type: 'reasoning',
      content: [{ text: reasoningContent }],
      status: message?.isThinkingComplete
        ? 'completed'
        : toDialogueStatus(message?.status),
    });
  }

  if (normalizedContent.length > 0) {
    normalizedMessages.push({
      type: 'message',
      role,
      content: normalizedContent,
    });
  }

  if (normalizedMessages.length === 0) {
    return rawContent;
  }

  return normalizedMessages;
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
  const [assetMap, setAssetMap] = useState(null);
  const dialogueHostRef = useRef(null);
  const autoStickToBottomRef = useRef(true);
  const dialogueWheelScrollRef = useRef(false);
  const previousChatCountRef = useRef(0);
  const autoScrollFrameRef = useRef(null);
  const autoScrollTimerRef = useRef(null);
  const lastScrollTopRef = useRef(0);
  const touchStartYRef = useRef(null);

  const getDialogueListElement = useCallback(
    () =>
      dialogueHostRef.current?.querySelector('.semi-ai-chat-dialogue-list') ||
      null,
    [],
  );

  const isDialogueAtBottom = useCallback(
    (element, gap = AUTO_SCROLL_BOTTOM_GAP) => {
      if (!element) return true;
      return (
        element.scrollHeight - element.scrollTop - element.clientHeight <= gap
      );
    },
    [],
  );

  const cancelScheduledAutoScroll = useCallback(() => {
    if (autoScrollFrameRef.current !== null) {
      window.cancelAnimationFrame(autoScrollFrameRef.current);
      autoScrollFrameRef.current = null;
    }
    if (autoScrollTimerRef.current !== null) {
      window.clearTimeout(autoScrollTimerRef.current);
      autoScrollTimerRef.current = null;
    }
  }, []);

  const syncDialogueWheelScroll = useCallback(
    (wheelScroll) => {
      if (dialogueWheelScrollRef.current === wheelScroll) return;
      dialogueWheelScrollRef.current = wheelScroll;
      chatRef.current?.adapter?.setWheelScroll?.(wheelScroll);
    },
    [chatRef],
  );

  const pauseAutoStickToBottom = useCallback(() => {
    if (!autoStickToBottomRef.current) return;
    autoStickToBottomRef.current = false;
    cancelScheduledAutoScroll();
    syncDialogueWheelScroll(true);
  }, [cancelScheduledAutoScroll, syncDialogueWheelScroll]);

  const scrollDialogueToBottom = useCallback(() => {
    if (!autoStickToBottomRef.current) return;

    syncDialogueWheelScroll(false);
    chatRef.current?.scrollToBottom?.(false);

    const element = getDialogueListElement();
    if (element) {
      element.scrollTop = element.scrollHeight;
      lastScrollTopRef.current = element.scrollTop;
    }
  }, [chatRef, getDialogueListElement, syncDialogueWheelScroll]);

  const scheduleAutoScrollToBottom = useCallback(() => {
    if (!autoStickToBottomRef.current) return;

    cancelScheduledAutoScroll();
    autoScrollFrameRef.current = window.requestAnimationFrame(() => {
      autoScrollFrameRef.current = null;
      scrollDialogueToBottom();
    });
    autoScrollTimerRef.current = window.setTimeout(() => {
      autoScrollTimerRef.current = null;
      scrollDialogueToBottom();
    }, AUTO_SCROLL_RENDER_SETTLE_MS);
  }, [cancelScheduledAutoScroll, scrollDialogueToBottom]);

  const needsAssetMap = useMemo(() => {
    const list = Array.isArray(message) ? message : [];
    return list.some((item) => messageHasAssetUri(item));
  }, [message]);

  useEffect(() => {
    if (!needsAssetMap) return;
    let cancelled = false;
    listMaterialAssets({ page: 1, pageSize: 100 })
      .then((res) => {
        if (cancelled || !res?.success || !Array.isArray(res.data?.items)) {
          return;
        }
        setAssetMap(buildAssetMap(res.data.items));
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, [needsAssetMap]);

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
      const dialogueContent = normalizeDialogueContent(item, assetMap);
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
  }, [assetMap, message]);

  const playgroundMarkdownRenderProps = useMemo(
    () => createPlaygroundMarkdownRenderProps(assetMap),
    [assetMap],
  );

  const generating = useMemo(
    () =>
      safeChats.some((item) =>
        ['loading', 'incomplete', 'in_progress'].includes(
          item?.playgroundStatus || item?.status,
        ),
      ),
    [safeChats],
  );

  useEffect(() => {
    const element = getDialogueListElement();
    if (!element) return undefined;

    const initialAtBottom = isDialogueAtBottom(element);
    autoStickToBottomRef.current = initialAtBottom;
    lastScrollTopRef.current = element.scrollTop;
    syncDialogueWheelScroll(!initialAtBottom);

    const handleScroll = () => {
      const previousScrollTop = lastScrollTopRef.current;
      const currentScrollTop = element.scrollTop;
      const scrollingDown = currentScrollTop >= previousScrollTop;
      lastScrollTopRef.current = currentScrollTop;

      const atBottom = isDialogueAtBottom(
        element,
        AUTO_SCROLL_RESUME_BOTTOM_GAP,
      );
      if (atBottom && (autoStickToBottomRef.current || scrollingDown)) {
        if (!autoStickToBottomRef.current) {
          autoStickToBottomRef.current = true;
          syncDialogueWheelScroll(false);
          scheduleAutoScrollToBottom();
        }
        return;
      }

      pauseAutoStickToBottom();
    };

    const handleWheel = (event) => {
      if (event.deltaY < 0) {
        pauseAutoStickToBottom();
      }
    };

    const handleTouchStart = (event) => {
      touchStartYRef.current = event.touches?.[0]?.clientY ?? null;
    };

    const handleTouchMove = (event) => {
      const previousY = touchStartYRef.current;
      const currentY = event.touches?.[0]?.clientY ?? null;
      if (previousY !== null && currentY !== null) {
        const deltaY = currentY - previousY;
        if (deltaY > TOUCH_SCROLL_DIRECTION_THRESHOLD) {
          pauseAutoStickToBottom();
        }
      }
      touchStartYRef.current = currentY;
    };

    element.addEventListener('scroll', handleScroll, { passive: true });
    element.addEventListener('wheel', handleWheel, { passive: true });
    element.addEventListener('touchstart', handleTouchStart, { passive: true });
    element.addEventListener('touchmove', handleTouchMove, { passive: true });
    return () => {
      element.removeEventListener('scroll', handleScroll);
      element.removeEventListener('wheel', handleWheel);
      element.removeEventListener('touchstart', handleTouchStart);
      element.removeEventListener('touchmove', handleTouchMove);
    };
  }, [
    getDialogueListElement,
    isDialogueAtBottom,
    pauseAutoStickToBottom,
    scheduleAutoScrollToBottom,
    syncDialogueWheelScroll,
  ]);

  useEffect(
    () => () => {
      cancelScheduledAutoScroll();
    },
    [cancelScheduledAutoScroll],
  );

  useLayoutEffect(() => {
    const chatCount = safeChats.length;
    if (chatCount > previousChatCountRef.current) {
      autoStickToBottomRef.current = true;
      syncDialogueWheelScroll(false);
    }
    previousChatCountRef.current = chatCount;

    if (generating || autoStickToBottomRef.current) {
      scheduleAutoScrollToBottom();
    }
  }, [
    generating,
    safeChats,
    scheduleAutoScrollToBottom,
    syncDialogueWheelScroll,
  ]);

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
      renderDialogueAction: ({
        className,
        defaultActionsObj,
        message: chat,
      }) => {
        const chatStatus = chat?.playgroundStatus || chat?.status;
        const isMessageLoading = [
          'loading',
          'incomplete',
          'in_progress',
        ].includes(chatStatus);
        const shouldDisableReset = generating || isMessageLoading;
        const resetNode = (
          <Button
            key='reset'
            theme='borderless'
            type='tertiary'
            icon={<IconRefresh />}
            disabled={shouldDisableReset}
            onClick={(event) => {
              event.preventDefault();
              event.stopPropagation();
              if (!shouldDisableReset) {
                onMessageReset?.(chat);
              }
            }}
            aria-label={t('重试')}
          />
        );
        const actions = [
          ['copy', defaultActionsObj?.copyNode],
          ['reset', resetNode],
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
    [generating, onMessageReset, t],
  );

  const renderDialogueContentItem = useMemo(
    () => ({
      input_text: (item, currentMessage) => {
        const text = typeof item?.text === 'string' ? item.text : '';
        if (!text.trim()) return null;
        const isUserMessage = currentMessage?.role === 'user';
        const messageStatus =
          currentMessage?.playgroundStatus || currentMessage?.status;
        const isStreaming = ['loading', 'incomplete', 'in_progress'].includes(
          messageStatus,
        );
        return (
          <div
            className={`semi-ai-chat-dialogue-content semi-ai-chat-dialogue-content-bubble ${
              isUserMessage ? 'semi-ai-chat-dialogue-content-user' : ''
            }`}
            style={{ marginTop: 0 }}
          >
            <StableDialogueMarkdown
              raw={text}
              components={playgroundMarkdownRenderProps.components}
              escapeHtml={isUserMessage}
              streaming={isStreaming}
              onContentRendered={scheduleAutoScrollToBottom}
            />
          </div>
        );
      },
      input_image: (item, currentMessage) => {
        const imageUrl = resolvePlaygroundMediaUrl(item?.image_url, assetMap);
        if (!imageUrl) return null;
        const isUserReference = currentMessage?.role === 'user';
        const imageMaxWidth = isUserReference
          ? PLAYGROUND_DIALOGUE_REFERENCE_IMAGE_MAX_WIDTH
          : PLAYGROUND_MEDIA_MAX_WIDTH;
        const imageMaxHeight = isUserReference
          ? PLAYGROUND_DIALOGUE_REFERENCE_IMAGE_MAX_HEIGHT
          : PLAYGROUND_MEDIA_MAX_HEIGHT;
        return (
          <PlaygroundDialogueMarkdownImage
            src={imageUrl}
            alt=''
            className='playground-dialogue-user-image'
            style={{ maxWidth: imageMaxWidth, maxHeight: imageMaxHeight }}
            imgStyle={{ maxWidth: imageMaxWidth, maxHeight: imageMaxHeight }}
          />
        );
      },
      playground_video: (item, currentMessage) => {
        const videoUrl = resolvePlaygroundMediaUrl(getVideoUrl(item), assetMap);
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
    [
      assetMap,
      mediaMaxHeightPx,
      onMediaDimensionsChange,
      playgroundMarkdownRenderProps.components,
      scheduleAutoScrollToBottom,
    ],
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
      <div
        ref={dialogueHostRef}
        className='flex min-h-0 flex-1 flex-col overflow-hidden'
      >
        <AIChatDialogue
          ref={chatRef}
          align='leftRight'
          mode='bubble'
          roleConfig={roleInfo}
          dialogueRenderConfig={dialogueRenderConfig}
          markdownRenderProps={playgroundMarkdownRenderProps}
          renderDialogueContentItem={renderDialogueContentItem}
          chats={safeChats}
          onChatsChange={handleChatsChange}
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
