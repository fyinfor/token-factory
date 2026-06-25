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

import React, { useRef, useEffect, useState } from 'react';
import {
  Typography,
  TextArea,
  Button,
  Progress,
  Modal,
} from '@douyinfe/semi-ui';
import MarkdownRenderer from '../common/markdown/MarkdownRenderer';
import ThinkingContent from './ThinkingContent';
import PlaygroundGeneratedImageGallery from './PlaygroundGeneratedImageGallery';
import {
  resolveMessageGeneratedImages,
  stripGeneratedImageMarkdown,
} from '../../helpers/playgroundImageUtils';
import { Loader2, Check, X } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import {
  PLAYGROUND_MEDIA_MAX_WIDTH,
  PLAYGROUND_MEDIA_MAX_WIDTH_PX,
  PLAYGROUND_MEDIA_MAX_HEIGHT,
} from '../../constants/playground.constants';
import { usePlaygroundMediaMaxHeightPx } from '../../hooks/playground/usePlaygroundMediaMaxHeight';
import { listMaterialAssets } from '../../helpers/materialApi';
import {
  buildAssetMap,
  isAssetUri,
  resolveAssetUrisInArray,
} from '../../helpers/materialAssetUtils';

const PLAYGROUND_MARKDOWN_MEDIA_CLASS =
  '[&_img]:max-w-[min(100%,780px)] [&_img]:max-h-[60vh] [&_img]:w-auto [&_img]:h-auto [&_img]:object-contain [&_img]:rounded-lg [&_img]:mx-auto [&_img]:cursor-zoom-in [&_video]:max-w-[min(100%,780px)] [&_video]:max-h-[60vh] [&_video]:w-auto [&_video]:h-auto [&_video]:object-contain [&_video]:rounded-lg [&_video]:mx-auto';

const getConstrainedMediaSize = (
  dimensions,
  maxWidth = PLAYGROUND_MEDIA_MAX_WIDTH_PX,
  maxHeight,
) => {
  const width = Number(dimensions?.width || 0);
  const height = Number(dimensions?.height || 0);
  if (!width || !height) return {};
  const ratio = Math.min(maxWidth / width, maxHeight / height, 1);
  return {
    width: Math.round(width * ratio),
    height: Math.round(height * ratio),
  };
};

const ANIMATION_DURATION_MS = 200;

const AnimatedRevealBox = ({
  children,
  ready = true,
  className,
  style,
  onRevealProgress,
  fluid = false,
  delay = 0,
  ...restProps
}) => {
  const contentRef = useRef(null);
  const [size, setSize] = useState({ width: 0, height: 0 });
  const [expanded, setExpanded] = useState(false);
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    if (!ready) {
      setExpanded(false);
      setVisible(false);
      setSize({ width: 0, height: 0 });
      return undefined;
    }

    const measure = () => {
      const element = contentRef.current;
      if (!element) return;
      setSize({
        width: fluid
          ? element.parentElement?.clientWidth || element.scrollWidth
          : Math.ceil(element.scrollWidth),
        height: Math.ceil(element.scrollHeight),
      });
    };

    const startTimer = window.setTimeout(() => {
      measure();
      const frameId = requestAnimationFrame(() => {
        measure();
        setExpanded(true);
        const visibleTimer = window.setTimeout(
          () => setVisible(true),
          ANIMATION_DURATION_MS,
        );
        const progressTimer = window.setInterval(
          () => onRevealProgress?.(),
          16,
        );
        const stopProgressTimer = window.setTimeout(
          () => window.clearInterval(progressTimer),
          ANIMATION_DURATION_MS + 40,
        );
      });
    }, delay);

    return () => {
      window.clearTimeout(startTimer);
    };
  }, [ready, onRevealProgress, fluid, delay]);

  return (
    <div
      className={className}
      {...restProps}
      style={{
        width: fluid ? '100%' : expanded ? `${size.width}px` : 0,
        height: fluid ? undefined : expanded ? `${size.height}px` : 0,
        maxWidth: '100%',
        overflow: fluid ? 'visible' : 'hidden',
        opacity: expanded ? 1 : 0,
        transition: fluid
          ? 'opacity 200ms ease'
          : 'width 200ms ease, height 200ms ease, opacity 200ms ease',
        ...style,
      }}
    >
      <div
        ref={contentRef}
        style={{
          width: 'max-content',
          ...(fluid ? { width: '100%' } : {}),
          maxWidth: '100%',
          opacity: fluid ? 1 : visible ? 1 : 0,
          transition: fluid ? 'none' : 'opacity 200ms ease',
        }}
      >
        {children}
      </div>
    </div>
  );
};

const MessageContent = ({
  message,
  className,
  styleState,
  onToggleReasoningExpansion,
  isEditing = false,
  onEditSave,
  onEditCancel,
  editValue,
  onEditValueChange,
  onMediaDimensionsChange,
  onRevealProgress,
}) => {
  const { t } = useTranslation();
  const mediaMaxHeightPx = usePlaygroundMediaMaxHeightPx();
  const previousContentLengthRef = useRef(0);
  const lastContentRef = useRef('');
  const [previewImageUrl, setPreviewImageUrl] = useState('');
  const [loadedMediaUrls, setLoadedMediaUrls] = useState(() => new Set());
  const [loadedMediaDimensions, setLoadedMediaDimensions] = useState({});

  const isThinkingStatus =
    message.status === 'loading' || message.status === 'incomplete';
  const isVideoGeneratingHint =
    message.role === 'assistant' &&
    typeof message.content === 'string' &&
    message.content.trim() === '视频生成中，请稍后…';

  const videoTaskStatus = String(
    message?.videoTask?.status || '',
  ).toLowerCase();
  const videoTaskProgress = Number(message?.videoTask?.progress || 0);
  const shouldShowVideoTaskProgress =
    message.role === 'assistant' &&
    !!message.videoTask &&
    ![
      'completed',
      'succeeded',
      'success',
      'failed',
      'error',
      'cancelled',
    ].includes(videoTaskStatus) &&
    // queued 且 0% 时不展示进度条，避免“假进度”误导
    !(videoTaskStatus === 'queued' && videoTaskProgress <= 0);

  useEffect(() => {
    if (!isThinkingStatus) {
      previousContentLengthRef.current = 0;
      lastContentRef.current = '';
    }
  }, [isThinkingStatus]);

  if (message.status === 'error') {
    let errorText;

    if (Array.isArray(message.content)) {
      const textContent = message.content.find((item) => item.type === 'text');
      errorText =
        textContent && textContent.text && typeof textContent.text === 'string'
          ? textContent.text
          : t('请求发生错误');
    } else if (typeof message.content === 'string') {
      errorText = message.content;
    } else {
      errorText = t('请求发生错误');
    }

    return (
      <div className={`${className}`}>
        <Typography.Text className='text-white'>{errorText}</Typography.Text>
      </div>
    );
  }

  let currentExtractedThinkingContent = null;
  let currentDisplayableFinalContent = '';
  let thinkingSource = null;

  const getTextContent = (content) => {
    if (Array.isArray(content)) {
      const textItem = content.find((item) => item.type === 'text');
      return textItem && textItem.text && typeof textItem.text === 'string'
        ? textItem.text
        : '';
    } else if (typeof content === 'string') {
      return content;
    }
    return '';
  };

  currentDisplayableFinalContent = getTextContent(message.content);

  if (message.role === 'assistant') {
    let baseContentForDisplay = getTextContent(message.content);
    let combinedThinkingContent = '';

    if (message.reasoningContent) {
      combinedThinkingContent = message.reasoningContent;
      thinkingSource = 'reasoningContent';
    }

    if (baseContentForDisplay.includes('<think>')) {
      const thinkTagRegex = /<think>([\s\S]*?)<\/think>/g;
      let match;
      let thoughtsFromPairedTags = [];
      let replyParts = [];
      let lastIndex = 0;

      while ((match = thinkTagRegex.exec(baseContentForDisplay)) !== null) {
        replyParts.push(
          baseContentForDisplay.substring(lastIndex, match.index),
        );
        thoughtsFromPairedTags.push(match[1]);
        lastIndex = match.index + match[0].length;
      }
      replyParts.push(baseContentForDisplay.substring(lastIndex));

      if (thoughtsFromPairedTags.length > 0) {
        const pairedThoughtsStr = thoughtsFromPairedTags.join('\n\n---\n\n');
        if (combinedThinkingContent) {
          combinedThinkingContent += '\n\n---\n\n' + pairedThoughtsStr;
        } else {
          combinedThinkingContent = pairedThoughtsStr;
        }
        thinkingSource = thinkingSource
          ? thinkingSource + ' & <think> tags'
          : '<think> tags';
      }

      baseContentForDisplay = replyParts.join('');
    }

    if (isThinkingStatus) {
      const lastOpenThinkIndex = baseContentForDisplay.lastIndexOf('<think>');
      if (lastOpenThinkIndex !== -1) {
        const fragmentAfterLastOpen =
          baseContentForDisplay.substring(lastOpenThinkIndex);
        if (!fragmentAfterLastOpen.includes('</think>')) {
          const unclosedThought = fragmentAfterLastOpen
            .substring('<think>'.length)
            .trim();
          if (unclosedThought) {
            if (combinedThinkingContent) {
              combinedThinkingContent += '\n\n---\n\n' + unclosedThought;
            } else {
              combinedThinkingContent = unclosedThought;
            }
            thinkingSource = thinkingSource
              ? thinkingSource + ' + streaming <think>'
              : 'streaming <think>';
          }
          baseContentForDisplay = baseContentForDisplay.substring(
            0,
            lastOpenThinkIndex,
          );
        }
      }
    }

    currentExtractedThinkingContent = combinedThinkingContent || null;
    currentDisplayableFinalContent = baseContentForDisplay
      .replace(/<\/?think>/g, '')
      .trim();
  }

  const finalExtractedThinkingContent = currentExtractedThinkingContent;
  const rawGeneratedImages = resolveMessageGeneratedImages(message);

  // 【需求6】预览图片地址 asset:// 协议转换
  const [assetMap, setAssetMap] = React.useState(null);
  React.useEffect(() => {
    // 仅当存在 asset:// 地址时加载素材映射
    const hasAssetUri = rawGeneratedImages.some(
      (src) => typeof src === 'string' && isAssetUri(src),
    );
    if (!hasAssetUri || assetMap !== null) return;
    listMaterialAssets({ page: 1, pageSize: 100 })
      .then((res) => {
        if (res?.success && Array.isArray(res.data?.items)) {
          setAssetMap(buildAssetMap(res.data.items));
        }
      })
      .catch(() => {});
  }, [rawGeneratedImages, assetMap]);
  const generatedImages = assetMap
    ? resolveAssetUrisInArray(rawGeneratedImages, assetMap)
    : rawGeneratedImages;
  const mediaDimensions = message.mediaDimensions || {};
  const videoPlayableUrl = message?.videoTask?.playableUrl;
  const mergedMediaDimensions = {
    ...mediaDimensions,
    ...loadedMediaDimensions,
  };
  const constrainedVideoSize = getConstrainedMediaSize(
    mergedMediaDimensions[videoPlayableUrl],
    PLAYGROUND_MEDIA_MAX_WIDTH_PX,
    mediaMaxHeightPx,
  );
  const finalDisplayableFinalContent =
    generatedImages.length > 0
      ? stripGeneratedImageMarkdown(currentDisplayableFinalContent)
      : currentDisplayableFinalContent;

  if (
    message.role === 'assistant' &&
    isThinkingStatus &&
    !finalExtractedThinkingContent &&
    (!finalDisplayableFinalContent ||
      finalDisplayableFinalContent.trim() === '')
  ) {
    return (
      <div
        className={`${className} flex items-center gap-2 sm:gap-4 bg-gradient-to-r from-purple-50 to-indigo-50`}
      >
        <div className='w-5 h-5 rounded-full bg-gradient-to-br from-purple-500 to-indigo-600 flex items-center justify-center shadow-lg'>
          <Loader2
            className='animate-spin text-white'
            size={styleState.isMobile ? 16 : 20}
          />
        </div>
      </div>
    );
  }

  return (
    <>
      <div className={className}>
        {message.role === 'system' && (
          <div className='mb-2 sm:mb-4'>
            <div
              className='flex items-center gap-2 p-2 sm:p-3 bg-gradient-to-r from-amber-50 to-orange-50 rounded-lg'
              style={{ border: '1px solid var(--semi-color-border)' }}
            >
              <div className='w-4 h-4 sm:w-5 sm:h-5 rounded-full bg-gradient-to-br from-amber-500 to-orange-600 flex items-center justify-center shadow-sm'>
                <Typography.Text className='text-white text-xs font-bold'>
                  S
                </Typography.Text>
              </div>
              <Typography.Text className='text-amber-700 text-xs sm:text-sm font-medium'>
                {t('系统消息')}
              </Typography.Text>
            </div>
          </div>
        )}

        {message.role === 'assistant' && (
          <ThinkingContent
            message={message}
            finalExtractedThinkingContent={finalExtractedThinkingContent}
            thinkingSource={thinkingSource}
            styleState={styleState}
            onToggleReasoningExpansion={onToggleReasoningExpansion}
          />
        )}

        {shouldShowVideoTaskProgress && (
          <div className='mb-3 p-3 rounded-lg bg-slate-50 border border-slate-200'>
            <div className='flex items-center justify-between mb-2'>
              <Typography.Text strong>{t('视频生成进度')}</Typography.Text>
              <Typography.Text type='tertiary'>
                {videoTaskStatus || 'queued'}
              </Typography.Text>
            </div>
            <Progress
              percent={Math.max(0, Math.min(100, videoTaskProgress))}
              showInfo
              size='small'
            />
          </div>
        )}

        {message.role === 'assistant' && videoPlayableUrl && (
          <div
            className='relative rounded-lg overflow-hidden'
            style={{
              width: loadedMediaUrls.has(videoPlayableUrl)
                ? constrainedVideoSize.width
                  ? `${constrainedVideoSize.width}px`
                  : PLAYGROUND_MEDIA_MAX_WIDTH
                : 0,
              height: loadedMediaUrls.has(videoPlayableUrl)
                ? constrainedVideoSize.height
                  ? `${constrainedVideoSize.height}px`
                  : 'auto'
                : 0,
              maxWidth: PLAYGROUND_MEDIA_MAX_WIDTH,
              overflow: 'hidden',
              marginBottom: loadedMediaUrls.has(videoPlayableUrl)
                ? '0.75rem'
                : 0,
              transition:
                'width 200ms ease, height 200ms ease, margin-bottom 200ms ease, opacity 200ms ease',
              transitionDelay: loadedMediaUrls.has(videoPlayableUrl)
                ? '0ms, 0ms, 0ms, 200ms'
                : '0ms',
              opacity: loadedMediaUrls.has(videoPlayableUrl) ? 1 : 0,
            }}
          >
            <video
              controls
              src={videoPlayableUrl}
              className='block rounded-lg border bg-gray-50'
              style={{
                width: constrainedVideoSize.width
                  ? `${constrainedVideoSize.width}px`
                  : '100%',
                height: constrainedVideoSize.height
                  ? `${constrainedVideoSize.height}px`
                  : 'auto',
                maxWidth: '100%',
                maxHeight: PLAYGROUND_MEDIA_MAX_HEIGHT,
                objectFit: 'contain',
                opacity: loadedMediaUrls.has(videoPlayableUrl) ? 1 : 0,
                transition: 'opacity 200ms ease',
                transitionDelay: loadedMediaUrls.has(videoPlayableUrl)
                  ? '200ms'
                  : '0ms',
              }}
              onLoadedMetadata={(event) => {
                const nextDimensions = {
                  width: event.currentTarget.videoWidth,
                  height: event.currentTarget.videoHeight,
                };
                setLoadedMediaDimensions((prev) => ({
                  ...prev,
                  [videoPlayableUrl]: nextDimensions,
                }));
                setLoadedMediaUrls((prev) =>
                  new Set(prev).add(videoPlayableUrl),
                );
                const progressTimer = window.setInterval(
                  () => onRevealProgress?.(),
                  16,
                );
                window.setTimeout(
                  () => window.clearInterval(progressTimer),
                  ANIMATION_DURATION_MS + 60,
                );
                onMediaDimensionsChange?.(
                  message.id,
                  videoPlayableUrl,
                  nextDimensions,
                );
              }}
            />
          </div>
        )}

        {message.role === 'assistant' && generatedImages.length > 0 && (
          <PlaygroundGeneratedImageGallery
            images={generatedImages}
            dimensions={mergedMediaDimensions}
            onMediaLoad={(src, dimensions) =>
              onMediaDimensionsChange?.(message.id, src, dimensions)
            }
            onRevealProgress={onRevealProgress}
            onPreview={setPreviewImageUrl}
            maxWidth={PLAYGROUND_MEDIA_MAX_WIDTH}
            maxHeight={PLAYGROUND_MEDIA_MAX_HEIGHT}
            maxHeightPx={mediaMaxHeightPx}
          />
        )}

        {isEditing ? (
          <div className='space-y-3'>
            <TextArea
              value={editValue}
              onChange={(value) => onEditValueChange(value)}
              placeholder={t('请输入消息内容...')}
              autosize={{ minRows: 3, maxRows: 12 }}
              style={{
                resize: 'vertical',
                fontSize: styleState.isMobile ? '14px' : '15px',
                lineHeight: '1.6',
              }}
              className='!border-blue-200 focus:!border-blue-400 !bg-blue-50/50'
            />
            <div className='flex items-center gap-2 w-full'>
              <Button
                size='small'
                type='danger'
                theme='light'
                icon={<X size={14} />}
                onClick={onEditCancel}
                className='flex-1'
              >
                {t('取消')}
              </Button>
              <Button
                size='small'
                type='warning'
                theme='solid'
                icon={<Check size={14} />}
                onClick={onEditSave}
                disabled={!editValue || editValue.trim() === ''}
                className='flex-1'
              >
                {t('保存')}
              </Button>
            </div>
          </div>
        ) : (
          (() => {
            if (isVideoGeneratingHint) {
              return (
                <div className='mb-1 p-3 rounded-lg bg-sky-50 border border-sky-200'>
                  <div className='flex items-center gap-2'>
                    <Loader2 size={16} className='text-sky-600 animate-spin' />
                    <Typography.Text className='text-sky-700 font-medium'>
                      {t('视频生成中，请稍后')}
                    </Typography.Text>
                    <span className='inline-flex items-center gap-1 ml-1'>
                      <span className='w-1.5 h-1.5 rounded-full bg-sky-500 animate-bounce [animation-delay:0ms]' />
                      <span className='w-1.5 h-1.5 rounded-full bg-sky-500 animate-bounce [animation-delay:150ms]' />
                      <span className='w-1.5 h-1.5 rounded-full bg-sky-500 animate-bounce [animation-delay:300ms]' />
                    </span>
                  </div>
                </div>
              );
            }

            if (Array.isArray(message.content)) {
              const textContent = message.content.find(
                (item) => item.type === 'text',
              );
              const imageContents = message.content.filter(
                (item) => item.type === 'image_url',
              );
              const hasLoadedImageContents = imageContents.some((imgItem) =>
                loadedMediaUrls.has(imgItem.image_url.url),
              );

              return (
                <div>
                  {imageContents.length > 0 && (
                    <div
                      style={{
                        display: 'flex',
                        flexDirection: 'column',
                        gap: hasLoadedImageContents ? '0.5rem' : 0,
                        marginBottom: hasLoadedImageContents ? '0.75rem' : 0,
                        overflow: 'hidden',
                        transition: 'gap 200ms ease, margin-bottom 200ms ease',
                      }}
                    >
                      {imageContents.map((imgItem, index) => {
                        const imageUrl = imgItem.image_url.url;
                        const constrainedImageSize = getConstrainedMediaSize(
                          mergedMediaDimensions[imageUrl],
                          PLAYGROUND_MEDIA_MAX_WIDTH_PX,
                          mediaMaxHeightPx,
                        );
                        return (
                          <div
                            key={index}
                            className='relative'
                            style={{
                              width: constrainedImageSize.width
                                ? loadedMediaUrls.has(imageUrl)
                                  ? `${constrainedImageSize.width}px`
                                  : 0
                                : 0,
                              height: constrainedImageSize.height
                                ? loadedMediaUrls.has(imageUrl)
                                  ? `${constrainedImageSize.height}px`
                                  : 0
                                : 0,
                              maxWidth: PLAYGROUND_MEDIA_MAX_WIDTH,
                              overflow: 'hidden',
                              opacity: loadedMediaUrls.has(imageUrl) ? 1 : 0,
                              transition:
                                'width 200ms ease, height 200ms ease, opacity 200ms ease',
                              transitionDelay: loadedMediaUrls.has(imageUrl)
                                ? '0ms, 0ms, 200ms'
                                : '0ms',
                            }}
                          >
                            <img
                              src={imageUrl}
                              alt={`用户上传的图片 ${index + 1}`}
                              className='block rounded-lg shadow-sm border cursor-zoom-in object-contain bg-gray-50'
                              style={{
                                width: constrainedImageSize.width
                                  ? `${constrainedImageSize.width}px`
                                  : '100%',
                                height: constrainedImageSize.height
                                  ? `${constrainedImageSize.height}px`
                                  : 'auto',
                                maxWidth: '100%',
                                maxHeight: PLAYGROUND_MEDIA_MAX_HEIGHT,
                                objectFit: 'contain',
                                opacity: loadedMediaUrls.has(imageUrl) ? 1 : 0,
                                transition: 'opacity 200ms ease',
                                transitionDelay: loadedMediaUrls.has(imageUrl)
                                  ? '200ms'
                                  : '0ms',
                              }}
                              onLoad={(event) => {
                                const nextDimensions = {
                                  width: event.currentTarget.naturalWidth,
                                  height: event.currentTarget.naturalHeight,
                                };
                                setLoadedMediaDimensions((prev) => ({
                                  ...prev,
                                  [imageUrl]: nextDimensions,
                                }));
                                setLoadedMediaUrls((prev) =>
                                  new Set(prev).add(imageUrl),
                                );
                                const progressTimer = window.setInterval(
                                  () => onRevealProgress?.(),
                                  16,
                                );
                                window.setTimeout(
                                  () => window.clearInterval(progressTimer),
                                  ANIMATION_DURATION_MS + 60,
                                );
                                onMediaDimensionsChange?.(
                                  message.id,
                                  imageUrl,
                                  nextDimensions,
                                );
                              }}
                              onClick={() => setPreviewImageUrl(imageUrl)}
                              onError={(e) => {
                                e.target.style.display = 'none';
                                e.target.nextSibling.style.display = 'block';
                              }}
                            />
                            <div
                              className='text-red-500 text-sm p-2 bg-red-50 rounded-lg border border-red-200'
                              style={{ display: 'none' }}
                            >
                              图片加载失败: {imageUrl}
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  )}

                  {textContent &&
                    textContent.text &&
                    typeof textContent.text === 'string' &&
                    textContent.text.trim() !== '' && (
                      <AnimatedRevealBox
                        ready
                        fluid
                        onRevealProgress={onRevealProgress}
                        className={`prose prose-xs sm:prose-sm prose-gray max-w-none overflow-x-auto text-xs sm:text-sm ${PLAYGROUND_MARKDOWN_MEDIA_CLASS} ${message.role === 'user' ? 'user-message' : ''}`}
                      >
                        <MarkdownRenderer
                          content={textContent.text}
                          className={
                            message.role === 'user' ? 'user-message' : ''
                          }
                          animated={false}
                          previousContentLength={0}
                        />
                      </AnimatedRevealBox>
                    )}
                </div>
              );
            }

            if (typeof message.content === 'string') {
              if (message.role === 'assistant') {
                if (
                  generatedImages.length > 0 &&
                  (!finalDisplayableFinalContent ||
                    finalDisplayableFinalContent.trim() === '')
                ) {
                  return null;
                }
                if (
                  finalDisplayableFinalContent &&
                  finalDisplayableFinalContent.trim() !== ''
                ) {
                  // 获取上一次的内容长度
                  let prevLength = 0;
                  if (isThinkingStatus && lastContentRef.current) {
                    // 只有当前内容包含上一次内容时，才使用上一次的长度
                    if (
                      finalDisplayableFinalContent.startsWith(
                        lastContentRef.current,
                      )
                    ) {
                      prevLength = lastContentRef.current.length;
                    }
                  }

                  // 更新最后内容的引用
                  if (isThinkingStatus) {
                    lastContentRef.current = finalDisplayableFinalContent;
                  }

                  return (
                    <AnimatedRevealBox
                      ready
                      fluid
                      onRevealProgress={onRevealProgress}
                      className={`prose prose-xs sm:prose-sm prose-gray max-w-none overflow-x-auto text-xs sm:text-sm ${PLAYGROUND_MARKDOWN_MEDIA_CLASS}`}
                      onClick={(e) => {
                        const target = e.target;
                        if (
                          target &&
                          target.tagName === 'IMG' &&
                          typeof target.src === 'string' &&
                          target.src
                        ) {
                          setPreviewImageUrl(target.src);
                        }
                      }}
                    >
                      <MarkdownRenderer
                        content={finalDisplayableFinalContent}
                        className=''
                        animated={isThinkingStatus}
                        previousContentLength={prevLength}
                      />
                    </AnimatedRevealBox>
                  );
                }
              } else {
                return (
                  <AnimatedRevealBox
                    ready
                    fluid
                    onRevealProgress={onRevealProgress}
                    className={`prose prose-xs sm:prose-sm prose-gray max-w-none overflow-x-auto text-xs sm:text-sm ${PLAYGROUND_MARKDOWN_MEDIA_CLASS} ${message.role === 'user' ? 'user-message' : ''}`}
                  >
                    <MarkdownRenderer
                      content={message.content}
                      className={message.role === 'user' ? 'user-message' : ''}
                      animated={false}
                      previousContentLength={0}
                    />
                  </AnimatedRevealBox>
                );
              }
            }

            return null;
          })()
        )}
      </div>
      <Modal
        title={t('图片预览')}
        visible={!!previewImageUrl}
        footer={null}
        onCancel={() => setPreviewImageUrl('')}
        width={880}
        centered
        bodyStyle={{ padding: 12 }}
      >
        {previewImageUrl ? (
          <img
            src={previewImageUrl}
            alt={t('图片预览')}
            className='w-full h-auto rounded-lg'
            style={{ maxHeight: '75vh', objectFit: 'contain' }}
          />
        ) : null}
      </Modal>
    </>
  );
};

export default MessageContent;
