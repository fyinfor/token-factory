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
import { useIsMobile } from '../../hooks/common/useIsMobile';

// hooks
import { usePlaygroundState } from '../../hooks/playground/usePlaygroundState';
import { useMessageActions } from '../../hooks/playground/useMessageActions';
import { useApiRequest } from '../../hooks/playground/useApiRequest';
import { useSyncMessageAndCustomBody } from '../../hooks/playground/useSyncMessageAndCustomBody';
import { useMessageEdit } from '../../hooks/playground/useMessageEdit';
import { useDataLoader } from '../../hooks/playground/useDataLoader';

// Constants and utils
import {
  MESSAGE_ROLES,
  ERROR_MESSAGES,
} from '../../constants/playground.constants';
import {
  getLogo,
  stringToColor,
  buildMessageContent,
  createMessage,
  createLoadingAssistantMessage,
  getTextContent,
  buildApiPayload,
  encodeToBase64,
} from '../../helpers';

// Components
import {
  OptimizedSettingsPanel,
  OptimizedDebugPanel,
  OptimizedMessageContent,
  OptimizedMessageActions,
} from '../../components/playground/OptimizedComponents';
import ChatArea from '../../components/playground/ChatArea';
import LazyVisibleMessage from '../../components/playground/LazyVisibleMessage';
import FloatingButtons from '../../components/playground/FloatingButtons';
import { PlaygroundProvider } from '../../contexts/PlaygroundContext';

// 生成头像
const generateAvatarDataUrl = (username) => {
  if (!username) {
    return 'https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/docs-icon.png';
  }
  const firstLetter = username[0].toUpperCase();
  const bgColor = stringToColor(username);
  const svg = `
    <svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 32 32">
      <circle cx="16" cy="16" r="16" fill="${bgColor}" />
      <text x="50%" y="50%" dominant-baseline="central" text-anchor="middle" font-size="16" fill="#ffffff" font-family="sans-serif">${firstLetter}</text>
    </svg>
  `;
  return `data:image/svg+xml;base64,${encodeToBase64(svg)}`;
};

const Playground = () => {
  const { t } = useTranslation();
  const [userState] = useContext(UserContext);
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
  const getModeStorageKey = useCallback(
    () => `playground_mode_messages_${userState?.user?.id || 'guest'}`,
    [userState?.user?.id],
  );

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
    saveMessagesImmediately,
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
    setMessage,
    setDebugData,
    setActiveDebugTab,
    sseSourceRef,
    saveMessagesImmediately,
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
        const nextTaskStatus = String(patch?.videoTask?.status || '').toLowerCase();
        const nextMessageStatus = String(patch?.status || '').toLowerCase();
        const isTerminal =
          terminalStatuses.has(nextTaskStatus) ||
          nextMessageStatus === 'complete' ||
          nextMessageStatus === 'error';
        if (isTerminal) {
          activeVideoPollTaskIdsRef.current.delete(taskId);
        }
        setMessage((prevMessages) => {
          const updated = prevMessages.map((item) => {
            if (item?.id !== msg.id) return item;
            return {
              ...item,
              ...(patch.content !== undefined ? { content: patch.content } : {}),
              ...(patch.status ? { status: patch.status } : {}),
              ...(patch.videoTask !== undefined
                ? { videoTask: patch.videoTask }
                : {}),
            };
          });
          setTimeout(() => saveMessagesImmediately(updated), 0);
          return updated;
        });
      });
    });
  }, [message, setMessage, saveMessagesImmediately, startVideoTaskPolling]);

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
    if (modeStoreInitializedRef.current) return;
    const currentMode = inputs.display_mode || 'text';
    modeMessagesRef.current[currentMode] = Array.isArray(message) ? message : [];
    previousModeRef.current = currentMode;
    modeStoreInitializedRef.current = true;
  }, [inputs.display_mode, message]);

  // 恢复分模式消息（文本/图片/视频）持久化快照
  useEffect(() => {
    try {
      const raw = localStorage.getItem(getModeStorageKey());
      if (!raw) return;
      const parsed = JSON.parse(raw);
      if (!parsed || typeof parsed !== 'object') return;
      const restored = {
        text: Array.isArray(parsed.text) ? parsed.text : [],
        image: Array.isArray(parsed.image) ? parsed.image : [],
        video: Array.isArray(parsed.video) ? parsed.video : [],
      };
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
    }
  }, [getModeStorageKey, inputs.display_mode, setMessage]);

  useEffect(() => {
    const currentMode = inputs.display_mode || 'text';
    modeMessagesRef.current[currentMode] = Array.isArray(message) ? message : [];
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

  // 消息编辑
  const {
    editingMessageId,
    editValue,
    setEditValue,
    handleMessageEdit,
    handleEditSave,
    handleEditCancel,
  } = useMessageEdit(
    setMessage,
    inputs,
    parameterEnabled,
    sendRequest,
    saveMessagesImmediately,
  );

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

  // 消息操作
  const messageActions = useMessageActions(
    message,
    setMessage,
    onMessageSend,
    saveMessagesImmediately,
  );

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
            const allowMedia = mode === 'image' || mode === 'video';
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

  // 发送消息
  function onMessageSend(content, attachment) {
    console.log('attachment: ', attachment);

    // 创建用户消息和加载消息
    const userMessage = createMessage(MESSAGE_ROLES.USER, content);
    const loadingMessage = createLoadingAssistantMessage();

    // 如果是自定义请求体模式
    if (customRequestMode && customRequestBody) {
      try {
        const customPayload = JSON.parse(customRequestBody);

        setMessage((prevMessage) => {
          const newMessages = [...prevMessage, userMessage, loadingMessage];

          // 发送自定义请求体
          sendRequest(customPayload, customPayload.stream !== false);

          // 发送消息后保存，传入新消息列表
          setTimeout(() => saveMessagesImmediately(newMessages), 0);

          return newMessages;
        });
        return;
      } catch (error) {
        console.error('自定义请求体JSON解析失败:', error);
        Toast.error(ERROR_MESSAGES.JSON_PARSE_ERROR);
        return;
      }
    }

    // 默认模式
    const mode = inputs.display_mode || 'text';
    const allowMedia = mode === 'image' || mode === 'video';
    const validImageUrls = allowMedia
      ? inputs.imageUrls.filter((url) => url.trim() !== '')
      : [];
    const messageContent = buildMessageContent(
      content,
      validImageUrls,
      allowMedia,
    );
    const userMessageWithImages = createMessage(
      MESSAGE_ROLES.USER,
      messageContent,
    );

    setMessage((prevMessage) => {
      const newMessages = [...prevMessage, userMessageWithImages];

      const payload = buildApiPayload(
        newMessages,
        null,
        inputs,
        parameterEnabled,
      );
      const isChatEndpoint = payload?.__endpoint === 'chat';
      sendRequest(payload, isChatEndpoint ? inputs.stream : false);

      // 发送消息后保存，传入新消息列表（包含用户消息和加载消息）
      const messagesWithLoading = [...newMessages, loadingMessage];
      setTimeout(() => saveMessagesImmediately(messagesWithLoading), 0);

      return messagesWithLoading;
    });
  }

  // 切换推理展开状态
  const toggleReasoningExpansion = useCallback(
    (messageId) => {
      setMessage((prevMessages) =>
        prevMessages.map((msg) =>
          msg.id === messageId && msg.role === MESSAGE_ROLES.ASSISTANT
            ? { ...msg, isReasoningExpanded: !msg.isReasoningExpanded }
            : msg,
        ),
      );
    },
    [setMessage],
  );

  // 渲染函数
  const renderCustomChatContent = useCallback(
    ({ message: msg, className }) => {
      const isCurrentlyEditing = editingMessageId === msg.id;
      const displayMode = inputs.display_mode || 'text';
      const isMediaMode = displayMode === 'image' || displayMode === 'video';
      const isLastInThread =
        Array.isArray(message) &&
        message.length > 0 &&
        msg?.id === message[message.length - 1]?.id;
      const skipLazy =
        msg?.status === 'loading' ||
        msg?.status === 'incomplete' ||
        msg?.status === 'error' ||
        (!isMediaMode && isLastInThread);

      const body = (
        <OptimizedMessageContent
          message={msg}
          className={skipLazy ? className : undefined}
          styleState={styleState}
          onToggleReasoningExpansion={toggleReasoningExpansion}
          isEditing={isCurrentlyEditing}
          onEditSave={handleEditSave}
          onEditCancel={handleEditCancel}
          editValue={editValue}
          onEditValueChange={setEditValue}
        />
      );

      if (skipLazy) {
        return body;
      }

      return (
        <LazyVisibleMessage
          messageId={msg.id}
          className={className}
          variant={isMediaMode ? 'media' : 'default'}
        >
          {body}
        </LazyVisibleMessage>
      );
    },
    [
      styleState,
      editingMessageId,
      editValue,
      handleEditSave,
      handleEditCancel,
      setEditValue,
      toggleReasoningExpansion,
      message,
      inputs.display_mode,
    ],
  );

  const renderChatBoxAction = useCallback(
    (props) => {
      const { message: currentMessage } = props;
      const isAnyMessageGenerating = message.some(
        (msg) => msg.status === 'loading' || msg.status === 'incomplete',
      );
      const isCurrentlyEditing = editingMessageId === currentMessage.id;

      return (
        <OptimizedMessageActions
          message={currentMessage}
          styleState={styleState}
          onMessageReset={messageActions.handleMessageReset}
          onMessageCopy={messageActions.handleMessageCopy}
          onMessageDelete={messageActions.handleMessageDelete}
          onRoleToggle={messageActions.handleRoleToggle}
          onMessageEdit={handleMessageEdit}
          isAnyMessageGenerating={isAnyMessageGenerating}
          isEditing={isCurrentlyEditing}
        />
      );
    },
    [messageActions, styleState, message, editingMessageId, handleMessageEdit],
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
    const timer = setTimeout(() => {
      saveMessagesImmediately(message);
      try {
        localStorage.setItem(
          getModeStorageKey(),
          JSON.stringify(modeMessagesRef.current),
        );
      } catch (err) {
        console.warn('保存分模式消息失败:', err);
      }
    }, 120);
    return () => clearTimeout(timer);
  }, [message, saveMessagesImmediately, getModeStorageKey]);

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
        try {
          localStorage.setItem(
            getModeStorageKey(),
            JSON.stringify(modeMessagesRef.current),
          );
        } catch (err) {
          console.warn('保存分模式消息失败:', err);
        }
        setMessage([]);
        // 清空对话后保存，传入空数组
        setTimeout(() => saveMessagesImmediately([]), 0);
      },
    });
  }, [inputs.display_mode, saveMessagesImmediately, setMessage, t, getModeStorageKey]);

  // 处理粘贴图片
  const handlePasteImage = useCallback(
    (base64Data) => {
      const mode = inputs.display_mode || 'text';
      if (mode !== 'image' && mode !== 'video') {
        return;
      }
      // 添加图片到 imageUrls 数组
      const newUrls = [...(inputs.imageUrls || []), base64Data];
      handleInputChange('imageUrls', newUrls);
    },
    [inputs.display_mode, inputs.imageUrls, handleInputChange],
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
                  : 'relative z-[1] w-80 h-[calc(100vh-66px)]'
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
              />
            </Layout.Sider>
          )}

          <Layout.Content className='relative flex-1 overflow-hidden'>
            <div className='overflow-hidden flex flex-col lg:flex-row h-[calc(100vh-66px)] mt-[60px]'>
              <div className='flex-1 flex flex-col'>
                <ChatArea
                  chatRef={chatRef}
                  message={message}
                  inputs={inputs}
                  styleState={styleState}
                  showDebugPanel={showDebugPanel}
                  roleInfo={roleInfo}
                  onMessageSend={onMessageSend}
                  onMessageCopy={messageActions.handleMessageCopy}
                  onMessageReset={messageActions.handleMessageReset}
                  onMessageDelete={messageActions.handleMessageDelete}
                  onStopGenerator={onStopGenerator}
                  onClearMessages={handleClearMessages}
                  onToggleDebugPanel={() => setShowDebugPanel(!showDebugPanel)}
                  renderCustomChatContent={renderCustomChatContent}
                  renderChatBoxAction={renderChatBoxAction}
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
