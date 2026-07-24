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
  useContext,
  useEffect,
  useCallback,
  useRef,
  useLayoutEffect,
} from 'react';
import { useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Layout, Toast, Modal } from '@douyinfe/semi-ui';

// Context
import { UserContext } from '../../context/User';
import { StatusContext } from '../../context/Status';
import { useIsMobile } from '../../hooks/common/useIsMobile';

// hooks
import { usePlaygroundState } from '../../hooks/playground/usePlaygroundState';
import { useApiRequest } from '../../hooks/playground/useApiRequest';
import { useSyncMessageAndCustomBody } from '../../hooks/playground/useSyncMessageAndCustomBody';
import { useDataLoader } from '../../hooks/playground/useDataLoader';

// Constants and utils
import {
  MESSAGE_ROLES,
  ERROR_MESSAGES,
  getDefaultMessages,
  PLAYGROUND_MEDIA_MAX_COUNT,
} from '../../constants/playground.constants';
import { appendUploadedMediaUrl } from '../../helpers/playgroundMediaInputUtils';
import {
  getLogo,
  stringToColor,
  buildMessageContent,
  createMessage,
  createLoadingAssistantMessage,
  getTextContent,
  buildApiPayload,
  encodeToBase64,
  toBoolean,
} from '../../helpers';

// Components
import {
  OptimizedSettingsPanel,
  OptimizedDebugPanel,
} from '../../components/playground/OptimizedComponents';
import {
  getMessageStorageKey,
  loadModeMessages,
  saveModeMessages,
} from '../../components/playground/messageStorage';
import ChatArea from '../../components/playground/ChatArea';
import FloatingButtons from '../../components/playground/FloatingButtons';
import { PlaygroundProvider } from '../../contexts/PlaygroundContext';

// 生成头像
const generateAvatarDataUrl = (username) => {
  if (!username) {
    return 'https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/docs-icon.png';
  }
  const firstLetter = (username[0] || '').toUpperCase();
  const bgColor = stringToColor(username);
  const svg = `
    <svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 32 32">
      <circle cx="16" cy="16" r="16" fill="${bgColor}" />
      <text x="50%" y="50%" dominant-baseline="central" text-anchor="middle" font-size="16" fill="#ffffff" font-family="sans-serif">${firstLetter}</text>
    </svg>
  `;
  return `data:image/svg+xml;base64,${encodeToBase64(svg)}`;
};

const arePlaygroundMessagesEquivalent = (nextMessages, currentMessages) => {
  if (!Array.isArray(nextMessages) || !Array.isArray(currentMessages)) {
    return false;
  }
  if (nextMessages.length !== currentMessages.length) {
    return false;
  }
  return nextMessages.every((next, index) => {
    const current = currentMessages[index];
    return (
      String(next?.id || '') === String(current?.id || '') &&
      next?.role === current?.role &&
      next?.status === current?.status &&
      Boolean(next?.editing) === Boolean(current?.editing) &&
      Boolean(next?.like) === Boolean(current?.like) &&
      Boolean(next?.dislike) === Boolean(current?.dislike) &&
      next?.content === current?.content &&
      next?.reasoningContent === current?.reasoningContent &&
      next?.isReasoningExpanded === current?.isReasoningExpanded &&
      next?.generatedImages === current?.generatedImages &&
      next?.mediaDimensions === current?.mediaDimensions &&
      next?.videoTask === current?.videoTask
    );
  });
};

const stripDialogueTransientState = (messages) =>
  (Array.isArray(messages) ? messages : []).map((message) => {
    if (!message || typeof message !== 'object') {
      return message;
    }
    const { editing, ...rest } = message;
    return rest;
  });

const loadSavedModeMessages = async (userId) => {
  const saved = await loadModeMessages(userId);
  if (saved || !userId) {
    return saved;
  }
  return loadModeMessages(null);
};

const loadLegacyModeMessages = (userId) => {
  const legacyRaw = localStorage.getItem(getMessageStorageKey(userId));
  if (legacyRaw || !userId) {
    return {
      raw: legacyRaw,
      key: getMessageStorageKey(userId),
    };
  }
  return {
    raw: localStorage.getItem(getMessageStorageKey(null)),
    key: getMessageStorageKey(null),
  };
};

const loadLegacyMessages = (userId) => {
  const legacyMessageKey = userId
    ? `playground_messages_${userId}`
    : 'playground_messages';
  const legacyMessageRaw = localStorage.getItem(legacyMessageKey);
  if (legacyMessageRaw || !userId) {
    return {
      raw: legacyMessageRaw,
      key: legacyMessageKey,
    };
  }
  return {
    raw: localStorage.getItem('playground_messages'),
    key: 'playground_messages',
  };
};

const Playground = () => {
  const { t } = useTranslation();
  const [userState] = useContext(UserContext);
  const [statusState] = useContext(StatusContext);
  const isMobile = useIsMobile();
  const styleState = { isMobile };
  const [searchParams] = useSearchParams();
  const modeMessagesRef = useRef({
    text: [],
    image: [],
    video: [],
  });
  const previousModeRef = useRef('text');
  const currentMessagesRef = useRef([]);
  const modeStoreInitializedRef = useRef(false);
  const activeVideoPollTaskIdsRef = useRef(new Set());
  const pendingPlaygroundChatScrollRef = useRef(false);
  const messageResetInFlightRef = useRef(false);

  const state = usePlaygroundState(userState?.user?.id);
  const {
    inputs,
    parameterEnabled,
    showDebugPanel,
    customRequestMode,
    customRequestBody,
    showSettings,
    models,
    modelTypes,
    supplierOptions,
    groups,
    message,
    debugData,
    activeDebugTab,
    previewPayload,
    sseSourceRef,
    chatRef,
    handleInputChange,
    handleParameterToggle,
    debouncedSaveConfig,
    handleConfigImport,
    handleConfigReset,
    setShowSettings,
    setModels,
    setModelTypes,
    setSupplierOptions,
    setGroups,
    setStatus,
    setMessage,
    setDebugData,
    setActiveDebugTab,
    setPreviewPayload,
    setShowDebugPanel,
    setCustomRequestMode,
    setCustomRequestBody,
  } = state;
  const hideMediaTabs = toBoolean(
    statusState?.status?.aliyun_guardrail_hide_playground_media_tabs,
  );

  const persistModeMessages = useCallback(() => {
    saveModeMessages(userState?.user?.id, modeMessagesRef.current);
  }, [userState?.user?.id]);

  const saveMessagesForMode = useCallback(
    (messagesToSave, mode) => {
      const targetMode =
        mode || previousModeRef.current || inputs.display_mode || 'text';
      modeMessagesRef.current[targetMode] = Array.isArray(messagesToSave)
        ? messagesToSave
        : [];
      persistModeMessages();
    },
    [inputs.display_mode, persistModeMessages],
  );

  const setMessageForMode = useCallback(
    (updater, mode) => {
      const targetMode =
        mode || previousModeRef.current || inputs.display_mode || 'text';
      const currentMode =
        previousModeRef.current || inputs.display_mode || 'text';
      if (targetMode === currentMode) {
        setMessage(updater);
        return;
      }
      const prevMessages = Array.isArray(modeMessagesRef.current[targetMode])
        ? modeMessagesRef.current[targetMode]
        : [];
      const nextMessages =
        typeof updater === 'function' ? updater(prevMessages) : updater;
      modeMessagesRef.current[targetMode] = Array.isArray(nextMessages)
        ? nextMessages
        : [];
      persistModeMessages();
    },
    [inputs.display_mode, persistModeMessages, setMessage],
  );

  const handleMediaDimensionsChange = useCallback(
    (messageId, url, dimensions) => {
      const width = Number(dimensions?.width || 0);
      const height = Number(dimensions?.height || 0);
      if (!messageId || !url || !width || !height) return;

      const currentMode =
        previousModeRef.current || inputs.display_mode || 'text';
      setMessageForMode((prevMessages) => {
        let changed = false;
        const updatedMessages = prevMessages.map((item) => {
          if (item?.id !== messageId) return item;
          const prevDimensions = item.mediaDimensions?.[url];
          if (
            Number(prevDimensions?.width || 0) === width &&
            Number(prevDimensions?.height || 0) === height
          ) {
            return item;
          }
          changed = true;
          return {
            ...item,
            mediaDimensions: {
              ...(item.mediaDimensions || {}),
              [url]: { width, height },
            },
          };
        });
        if (changed) {
          setTimeout(
            () => saveMessagesForMode(updatedMessages, currentMode),
            0,
          );
        }
        return changed ? updatedMessages : prevMessages;
      }, currentMode);
    },
    [inputs.display_mode, saveMessagesForMode, setMessageForMode],
  );

  useLayoutEffect(() => {
    if (!pendingPlaygroundChatScrollRef.current) return;
    pendingPlaygroundChatScrollRef.current = false;
    const scrollNow = () => {
      try {
        chatRef.current?.scrollToBottom?.(false);
      } catch (_) {
        // Semi Chat ref 在极少数情况下可能尚未就绪
      }
    };
    scrollNow();
    const timers = [80, 200, 450].map((ms) => setTimeout(scrollNow, ms));
    return () => timers.forEach(clearTimeout);
  }, [message]);

  useEffect(() => {
    currentMessagesRef.current = Array.isArray(message) ? message : [];
  }, [message]);

  // API 请求相关
  const { sendRequest, onStopGenerator, startVideoTaskPolling } = useApiRequest(
    setMessageForMode,
    setDebugData,
    setActiveDebugTab,
    sseSourceRef,
    saveMessagesForMode,
  );

  useEffect(() => {
    const terminalStatuses = new Set([
      'completed',
      'succeeded',
      'success',
      'failed',
      'error',
      'cancelled',
      'timeout',
    ]);
    const shouldResumeStatuses = new Set([
      'queued',
      'processing',
      'in_progress',
      'running',
      'pending',
      'submitted',
      '',
    ]);
    const candidates = (Array.isArray(message) ? message : []).filter((msg) => {
      if (msg?.role !== MESSAGE_ROLES.ASSISTANT) return false;
      const taskId = msg?.videoTask?.taskId;
      if (!taskId) return false;
      if (msg?.videoTask?.playableUrl) return false;
      const taskStatus = String(msg?.videoTask?.status || '').toLowerCase();
      if (terminalStatuses.has(taskStatus)) return false;
      return shouldResumeStatuses.has(taskStatus);
    });
    candidates.forEach((msg) => {
      const taskId = msg?.videoTask?.taskId;
      if (!taskId || activeVideoPollTaskIdsRef.current.has(taskId)) return;
      activeVideoPollTaskIdsRef.current.add(taskId);
      startVideoTaskPolling(taskId, (patch) => {
        const nextTaskStatus = String(
          patch?.videoTask?.status || '',
        ).toLowerCase();
        const nextMessageStatus = String(patch?.status || '').toLowerCase();
        const isTerminal =
          terminalStatuses.has(nextTaskStatus) || nextMessageStatus === 'error';
        if (isTerminal) {
          activeVideoPollTaskIdsRef.current.delete(taskId);
        }
        setMessageForMode((prevMessages) => {
          const updated = prevMessages.map((item) => {
            if (item?.id !== msg.id) return item;
            return {
              ...item,
              ...(patch.content !== undefined
                ? { content: patch.content }
                : {}),
              ...(patch.status ? { status: patch.status } : {}),
              ...(patch.videoTask !== undefined
                ? { videoTask: patch.videoTask }
                : {}),
            };
          });
          setTimeout(() => saveMessagesForMode(updated, 'video'), 0);
          return updated;
        }, 'video');
      });
    });
  }, [message, setMessageForMode, saveMessagesForMode, startVideoTaskPolling]);

  // 数据加载（modelTypes 参与按类型筛选，与模型广场一致在客户端过滤）
  useDataLoader(
    userState,
    inputs,
    modelTypes,
    handleInputChange,
    setModels,
    setModelTypes,
    setSupplierOptions,
    setGroups,
    setStatus,
  );

  useEffect(() => {
    const displayMode = inputs.display_mode || 'text';
    if ((displayMode === 'image' || displayMode === 'video') && inputs.stream) {
      handleInputChange('stream', false);
    }
  }, [inputs.display_mode, inputs.stream, handleInputChange]);

  useEffect(() => {
    const displayMode = inputs.display_mode || 'text';
    if (hideMediaTabs && (displayMode === 'image' || displayMode === 'video')) {
      handleInputChange('display_mode', 'text');
    }
  }, [hideMediaTabs, inputs.display_mode, handleInputChange]);

  // 恢复分模式消息（文本/图片/视频）持久化快照
  useEffect(() => {
    let cancelled = false;
    modeStoreInitializedRef.current = false;
    const restoreModeMessages = async () => {
      const defaultMsgs = getDefaultMessages(t);
      let restored;
      try {
        const userId = userState?.user?.id;
        const saved = await loadSavedModeMessages(userId);
        let parsed = saved;

        if (!parsed) {
          const legacyMode = loadLegacyModeMessages(userId);
          if (legacyMode.raw) {
            parsed = JSON.parse(legacyMode.raw);
            await saveModeMessages(userId, parsed);
            localStorage.removeItem(legacyMode.key);
          }
        }

        if (!parsed) {
          const legacyMessage = loadLegacyMessages(userId);
          if (legacyMessage.raw) {
            const legacyMessageData = JSON.parse(legacyMessage.raw);
            if (Array.isArray(legacyMessageData?.messages)) {
              parsed = {
                text: legacyMessageData.messages,
                image: defaultMsgs,
                video: defaultMsgs,
              };
              await saveModeMessages(userId, parsed);
              localStorage.removeItem(legacyMessage.key);
            }
          }
        }

        if (Array.isArray(parsed)) {
          parsed = {
            text: parsed,
            image: defaultMsgs,
            video: defaultMsgs,
          };
        } else if (Array.isArray(parsed?.messages)) {
          parsed = {
            text: parsed.messages,
            image: defaultMsgs,
            video: defaultMsgs,
          };
        }

        if (!parsed) {
          // 没有持久化数据，初始化为默认消息
          restored = {
            text: defaultMsgs,
            image: defaultMsgs,
            video: defaultMsgs,
          };
        } else {
          restored = {
            text: Array.isArray(parsed.text) ? parsed.text : defaultMsgs,
            image: Array.isArray(parsed.image) ? parsed.image : defaultMsgs,
            video: Array.isArray(parsed.video) ? parsed.video : defaultMsgs,
          };
        }
        if (cancelled) return;

        modeMessagesRef.current = restored;
        const currentMode = inputs.display_mode || 'text';
        const currentModeMessages = restored[currentMode] || [];
        if (Array.isArray(currentModeMessages)) {
          pendingPlaygroundChatScrollRef.current = true;
          setMessage(currentModeMessages);
        }
        previousModeRef.current = currentMode;
        modeStoreInitializedRef.current = true;
      } catch (err) {
        console.warn('恢复分模式消息失败:', err);
        if (cancelled) return;
        const fallbackMessages = getDefaultMessages(t);
        const currentMode = inputs.display_mode || 'text';
        restored = {
          text: fallbackMessages,
          image: fallbackMessages,
          video: fallbackMessages,
        };
        modeMessagesRef.current = restored;
        pendingPlaygroundChatScrollRef.current = true;
        setMessage(restored[currentMode] || []);
        previousModeRef.current = currentMode;
        modeStoreInitializedRef.current = true;
      }
    };
    restoreModeMessages();
    return () => {
      cancelled = true;
    };
  }, [setMessage, t, userState?.user?.id]);

  useEffect(() => {
    if (!modeStoreInitializedRef.current) return;
    const activeMessageMode =
      previousModeRef.current || inputs.display_mode || 'text';
    modeMessagesRef.current[activeMessageMode] = Array.isArray(message)
      ? message
      : [];
  }, [inputs.display_mode, message]);

  useEffect(() => {
    if (!modeStoreInitializedRef.current) return;
    const nextMode = inputs.display_mode || 'text';
    const prevMode = previousModeRef.current || nextMode;
    if (nextMode === prevMode) return;
    modeMessagesRef.current[prevMode] = currentMessagesRef.current || [];
    const nextMessages = modeMessagesRef.current[nextMode];
    pendingPlaygroundChatScrollRef.current = true;
    setMessage(Array.isArray(nextMessages) ? nextMessages : []);
    previousModeRef.current = nextMode;
  }, [inputs.display_mode, setMessage]);

  // 消息和自定义请求体同步
  const { syncMessageToCustomBody, syncCustomBodyToMessage } =
    useSyncMessageAndCustomBody(
      customRequestMode,
      customRequestBody,
      message,
      inputs,
      setCustomRequestBody,
      setMessage,
      debouncedSaveConfig,
    );

  // 角色信息
  const roleInfo = {
    user: {
      name: userState?.user?.username || 'User',
      avatar: generateAvatarDataUrl(userState?.user?.username),
    },
    assistant: {
      name: 'Assistant',
      avatar: getLogo(),
    },
    system: {
      name: 'System',
      avatar: getLogo(),
    },
  };

  // 构建预览请求体
  const constructPreviewPayload = useCallback(() => {
    try {
      // 如果是自定义请求体模式且有自定义内容，直接返回解析后的自定义请求体
      if (customRequestMode && customRequestBody && customRequestBody.trim()) {
        try {
          return JSON.parse(customRequestBody);
        } catch (parseError) {
          console.warn('自定义请求体JSON解析失败，回退到默认预览:', parseError);
        }
      }

      // 默认预览逻辑
      let messages = [...message];

      // 如果存在用户消息
      if (
        !(
          messages.length === 0 ||
          messages.every((msg) => msg.role !== MESSAGE_ROLES.USER)
        )
      ) {
        // 处理最后一个用户消息的图片
        for (let i = messages.length - 1; i >= 0; i--) {
          if (messages[i].role === MESSAGE_ROLES.USER) {
            const mode = inputs.display_mode || 'text';
            // 文本 / 图片 / 视频：媒体侧栏图片需并入用户消息（视频请求体仍由 buildApiPayload 单独组装）
            const allowMedia =
              mode === 'text' || mode === 'image' || mode === 'video';
            if (allowMedia && inputs.imageUrls) {
              const validImageUrls = inputs.imageUrls.filter(
                (url) => url.trim() !== '',
              );
              if (validImageUrls.length > 0) {
                const textContent = getTextContent(messages[i]) || '示例消息';
                const content = buildMessageContent(
                  textContent,
                  validImageUrls,
                  true,
                );
                messages[i] = { ...messages[i], content };
              }
            }
            break;
          }
        }
      }

      return buildApiPayload(messages, null, inputs, parameterEnabled);
    } catch (error) {
      console.error('构造预览请求体失败:', error);
      return null;
    }
  }, [inputs, parameterEnabled, message, customRequestMode, customRequestBody]);

  // 发送消息（副作用放在 setState 外，避免 StrictMode 重复执行 updater 导致多次请求）
  const onMessageSend = useCallback(
    (content, attachment, options = {}) => {
      const baseMessages = Array.isArray(options.baseMessages)
        ? options.baseMessages
        : null;
      const reuseUserMessage = options.reuseUserMessage;
      const loadingMessage = createLoadingAssistantMessage();
      const existingMessages = Array.isArray(baseMessages)
        ? baseMessages
        : Array.isArray(currentMessagesRef.current)
          ? currentMessagesRef.current
          : [];

      // 如果是自定义请求体模式
      if (customRequestMode && customRequestBody) {
        try {
          const mode = inputs.display_mode || 'text';
          const customPayload = JSON.parse(customRequestBody);
          const userMessage =
            reuseUserMessage || createMessage(MESSAGE_ROLES.USER, content);
          const newMessages = [
            ...existingMessages,
            userMessage,
            loadingMessage,
          ];

          setMessage(newMessages);
          sendRequest(
            customPayload,
            customPayload.stream !== false,
            mode,
            loadingMessage.id,
          );
          setTimeout(() => saveMessagesForMode(newMessages, mode), 0);
          return;
        } catch (error) {
          console.error('自定义请求体JSON解析失败:', error);
          Toast.error(ERROR_MESSAGES.JSON_PARSE_ERROR);
          return;
        }
      }

      // 默认模式
      const mode = inputs.display_mode || 'text';
      // 文本 / 图片 / 视频：发送前将媒体侧栏图片拼入用户消息多模态 content
      const allowMedia =
        mode === 'text' || mode === 'image' || mode === 'video';
      const validImageUrls = allowMedia
        ? inputs.imageUrls.filter((url) => url.trim() !== '')
        : [];
      const messageContent = buildMessageContent(
        content,
        validImageUrls,
        allowMedia,
      );
      const userMessageWithImages =
        reuseUserMessage || createMessage(MESSAGE_ROLES.USER, messageContent);
      const payloadMessages = [...existingMessages, userMessageWithImages];
      const payload = buildApiPayload(
        payloadMessages,
        null,
        inputs,
        parameterEnabled,
      );
      const isChatEndpoint = payload?.__endpoint === 'chat';
      const messagesWithLoading = [...payloadMessages, loadingMessage];

      setMessage(messagesWithLoading);
      sendRequest(
        payload,
        isChatEndpoint ? inputs.stream : false,
        mode,
        loadingMessage.id,
      );
      setTimeout(() => saveMessagesForMode(messagesWithLoading, mode), 0);
    },
    [
      customRequestMode,
      customRequestBody,
      inputs,
      parameterEnabled,
      sendRequest,
      saveMessagesForMode,
      setMessage,
    ],
  );

  const handleDialogueChatsChange = useCallback(
    (nextMessages = []) => {
      const normalizedMessages = Array.isArray(nextMessages)
        ? nextMessages
        : [];
      const currentMessages = Array.isArray(currentMessagesRef.current)
        ? currentMessagesRef.current
        : [];

      if (
        arePlaygroundMessagesEquivalent(normalizedMessages, currentMessages)
      ) {
        return;
      }

      const mode = previousModeRef.current || inputs.display_mode || 'text';
      setMessage(normalizedMessages);
      setTimeout(
        () =>
          saveMessagesForMode(
            stripDialogueTransientState(normalizedMessages),
            mode,
          ),
        0,
      );
    },
    [inputs.display_mode, saveMessagesForMode, setMessage],
  );

  const handleDialogueMessageReset = useCallback(
    (targetMessage) => {
      if (messageResetInFlightRef.current) {
        return;
      }

      const currentMessages = Array.isArray(currentMessagesRef.current)
        ? currentMessagesRef.current
        : [];
      let messageIndex = currentMessages.findIndex(
        (msg) => msg?.id === targetMessage?.id,
      );

      if (messageIndex === -1) {
        messageIndex = currentMessages.findIndex(
          (msg) => msg === targetMessage,
        );
      }

      if (messageIndex === -1) {
        return;
      }

      let userMessageToResend = null;
      let messagesToKeep = currentMessages;

      if (targetMessage.role === MESSAGE_ROLES.USER) {
        userMessageToResend = currentMessages[messageIndex];
        messagesToKeep = currentMessages.slice(0, messageIndex);
      } else {
        let userMessageIndex = messageIndex - 1;
        while (
          userMessageIndex >= 0 &&
          currentMessages[userMessageIndex]?.role !== MESSAGE_ROLES.USER
        ) {
          userMessageIndex--;
        }

        if (userMessageIndex >= 0) {
          userMessageToResend = currentMessages[userMessageIndex];
          messagesToKeep = currentMessages.slice(0, userMessageIndex);
        }
      }

      const contentToSend = getTextContent(userMessageToResend);
      const hasImageContent =
        Array.isArray(userMessageToResend?.content) &&
        userMessageToResend.content.some((item) => item?.type === 'image_url');
      if (
        !userMessageToResend ||
        ((!contentToSend || contentToSend.trim() === '') && !hasImageContent)
      ) {
        return;
      }

      messageResetInFlightRef.current = true;
      onMessageSend(contentToSend, undefined, {
        baseMessages: messagesToKeep,
        reuseUserMessage: userMessageToResend,
      });
      window.setTimeout(() => {
        messageResetInFlightRef.current = false;
      }, 500);
    },
    [onMessageSend],
  );

  // Effects

  // 同步消息和自定义请求体
  useEffect(() => {
    syncMessageToCustomBody();
  }, [message, syncMessageToCustomBody]);

  useEffect(() => {
    syncCustomBodyToMessage();
  }, [customRequestBody, syncCustomBodyToMessage]);

  // 处理URL参数
  useEffect(() => {
    if (searchParams.get('expired')) {
      Toast.warning(t('登录过期，请重新登录！'));
    }
  }, [searchParams, t]);

  // Playground 组件无需再监听窗口变化，isMobile 由 useIsMobile Hook 自动更新

  // 构建预览payload
  useEffect(() => {
    const timer = setTimeout(() => {
      const preview = constructPreviewPayload();
      setPreviewPayload(preview);
      setDebugData((prev) => ({
        ...prev,
        previewRequest: preview ? JSON.stringify(preview, null, 2) : null,
        previewTimestamp: preview ? new Date().toISOString() : null,
      }));
    }, 300);

    return () => clearTimeout(timer);
  }, [
    message,
    inputs,
    parameterEnabled,
    customRequestMode,
    customRequestBody,
    constructPreviewPayload,
    setPreviewPayload,
    setDebugData,
  ]);

  // 自动保存配置
  useEffect(() => {
    debouncedSaveConfig();
  }, [
    inputs,
    parameterEnabled,
    showDebugPanel,
    customRequestMode,
    customRequestBody,
    debouncedSaveConfig,
  ]);

  // 兜底持久化：任何消息变更（含视频轮询进度与完成态）都同步落盘，
  // 避免刷新后丢失 videoTask.playableUrl 导致播放器消失。
  useEffect(() => {
    if (!modeStoreInitializedRef.current) return;
    const timer = setTimeout(() => {
      persistModeMessages();
    }, 120);
    return () => clearTimeout(timer);
  }, [message, persistModeMessages]);

  // 清空对话的处理函数
  const handleClearMessages = useCallback(() => {
    Modal.confirm({
      title: t('确认清空当前对话？'),
      content: t('此操作将清空当前展示模式下的全部消息，且不可撤销。'),
      okText: t('确认清空'),
      cancelText: t('取消'),
      okButtonProps: { type: 'danger' },
      onOk: () => {
        const currentMode = inputs.display_mode || 'text';
        modeMessagesRef.current[currentMode] = [];
        currentMessagesRef.current = [];
        persistModeMessages();
        setMessage([]);
        // 清空对话后保存，传入空数组
        setTimeout(() => saveMessagesForMode([], currentMode), 0);
      },
    });
  }, [
    inputs.display_mode,
    saveMessagesForMode,
    setMessage,
    t,
    persistModeMessages,
  ]);

  // 处理粘贴图片
  const handlePasteImage = useCallback(
    (base64Data) => {
      const mode = inputs.display_mode || 'text';
      if (mode !== 'image' && mode !== 'video') {
        return;
      }
      const { urls, ok } = appendUploadedMediaUrl(
        inputs.imageUrls,
        base64Data,
        PLAYGROUND_MEDIA_MAX_COUNT,
      );
      if (!ok) {
        Toast.warning({
          content: t('操练场素材已达上限', '最多添加 {{count}} 个', {
            count: PLAYGROUND_MEDIA_MAX_COUNT,
          }),
          duration: 3,
        });
        return;
      }
      handleInputChange('imageUrls', urls);
    },
    [inputs.display_mode, inputs.imageUrls, handleInputChange, t],
  );

  // Playground Context 值
  const playgroundContextValue = {
    onPasteImage: handlePasteImage,
    imageUrls: inputs.imageUrls || [],
    imageEnabled: ['image', 'video'].includes(inputs.display_mode || 'text'),
  };

  return (
    <PlaygroundProvider value={playgroundContextValue}>
      <div className='h-full'>
        <Layout className='h-full bg-transparent flex flex-col md:flex-row'>
          {(showSettings || !isMobile) && (
            <Layout.Sider
              className={`
              bg-transparent border-r-0 flex-shrink-0 overflow-auto mt-[60px]
              ${
                isMobile
                  ? 'fixed top-0 left-0 right-0 bottom-0 z-[1000] w-full h-auto bg-white shadow-lg'
                  : 'relative z-[1] w-80 playground-shell-h'
              }
            `}
              width={isMobile ? '100%' : 320}
            >
              <OptimizedSettingsPanel
                inputs={inputs}
                parameterEnabled={parameterEnabled}
                models={models}
                modelTypes={modelTypes}
                supplierOptions={supplierOptions}
                groups={groups}
                styleState={styleState}
                showSettings={showSettings}
                showDebugPanel={showDebugPanel}
                customRequestMode={customRequestMode}
                customRequestBody={customRequestBody}
                onInputChange={handleInputChange}
                onParameterToggle={handleParameterToggle}
                onCloseSettings={() => setShowSettings(false)}
                onConfigImport={handleConfigImport}
                onConfigReset={handleConfigReset}
                onCustomRequestModeChange={setCustomRequestMode}
                onCustomRequestBodyChange={setCustomRequestBody}
                previewPayload={previewPayload}
                messages={message}
                userId={userState?.user?.id}
                hideMediaTabs={hideMediaTabs}
              />
            </Layout.Sider>
          )}

          <Layout.Content className='relative flex-1 overflow-hidden'>
            <div className='playground-shell-h mt-[60px] flex min-h-0 flex-col overflow-hidden lg:flex-row'>
              <div className='flex min-h-0 flex-1 flex-col'>
                <ChatArea
                  chatRef={chatRef}
                  message={message}
                  inputs={inputs}
                  styleState={styleState}
                  showDebugPanel={showDebugPanel}
                  roleInfo={roleInfo}
                  onMessageSend={onMessageSend}
                  onMessageReset={handleDialogueMessageReset}
                  onChatsChange={handleDialogueChatsChange}
                  onMediaDimensionsChange={handleMediaDimensionsChange}
                  onStopGenerator={onStopGenerator}
                  onClearMessages={handleClearMessages}
                  onToggleDebugPanel={() => setShowDebugPanel(!showDebugPanel)}
                />
              </div>

              {/* 调试面板 - 桌面端 */}
              {showDebugPanel && !isMobile && (
                <div className='w-96 flex-shrink-0 h-full'>
                  <OptimizedDebugPanel
                    debugData={debugData}
                    activeDebugTab={activeDebugTab}
                    onActiveDebugTabChange={setActiveDebugTab}
                    styleState={styleState}
                    customRequestMode={customRequestMode}
                  />
                </div>
              )}
            </div>

            {/* 调试面板 - 移动端覆盖层 */}
            {showDebugPanel && isMobile && (
              <div className='fixed top-0 left-0 right-0 bottom-0 z-[1000] bg-white overflow-auto shadow-lg'>
                <OptimizedDebugPanel
                  debugData={debugData}
                  activeDebugTab={activeDebugTab}
                  onActiveDebugTabChange={setActiveDebugTab}
                  styleState={styleState}
                  showDebugPanel={showDebugPanel}
                  onCloseDebugPanel={() => setShowDebugPanel(false)}
                  customRequestMode={customRequestMode}
                />
              </div>
            )}

            {/* 浮动按钮 */}
            <FloatingButtons
              styleState={styleState}
              showSettings={showSettings}
              showDebugPanel={showDebugPanel}
              onToggleSettings={() => setShowSettings(!showSettings)}
              onToggleDebugPanel={() => setShowDebugPanel(!showDebugPanel)}
            />
          </Layout.Content>
        </Layout>
      </div>
    </PlaygroundProvider>
  );
};

export default Playground;
