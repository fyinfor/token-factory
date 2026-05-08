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

import { useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { SSE } from 'sse.js';
import {
  API_ENDPOINTS,
  MESSAGE_STATUS,
  DEBUG_TABS,
} from '../../constants/playground.constants';
import {
  getUserIdFromLocalStorage,
  handleApiError,
  processThinkTags,
  processIncompleteThinkTags,
} from '../../helpers';

export const useApiRequest = (
  setMessage,
  setDebugData,
  setActiveDebugTab,
  sseSourceRef,
  saveMessages,
) => {
  const getVideoTaskPayload = useCallback((obj) => {
    if (!obj || typeof obj !== 'object') return {};
    // 兼容 {code, data:{...}} 与扁平结构
    if (
      obj.data &&
      typeof obj.data === 'object' &&
      !Array.isArray(obj.data) &&
      (obj.data.task_id || obj.data.id || obj.data.status || obj.data.url)
    ) {
      return obj.data;
    }
    return obj;
  }, []);

  const extractVideoPlayableURL = useCallback((obj) => {
    if (!obj || typeof obj !== 'object') return '';
    const directKeys = ['url', 'video_url', 'content_url', 'output_url'];
    for (const key of directKeys) {
      const value = obj[key];
      if (typeof value === 'string' && value.trim()) {
        return value.trim();
      }
    }
    if (Array.isArray(obj.data)) {
      for (const item of obj.data) {
        const url = extractVideoPlayableURL(item);
        if (url) return url;
      }
    }
    if (obj.video && typeof obj.video === 'object') {
      const url = extractVideoPlayableURL(obj.video);
      if (url) return url;
    }
    if (obj.result && typeof obj.result === 'object') {
      const url = extractVideoPlayableURL(obj.result);
      if (url) return url;
    }
    if (obj.data && typeof obj.data === 'object' && !Array.isArray(obj.data)) {
      const url = extractVideoPlayableURL(obj.data);
      if (url) return url;
    }
    return '';
  }, []);

  const extractImageURLs = useCallback((obj) => {
    if (!obj || typeof obj !== 'object') return [];
    const raw = Array.isArray(obj?.data) ? obj.data : [];
    return raw
      .map((item) => {
        if (!item || typeof item !== 'object') return '';
        if (typeof item.url === 'string') return item.url.trim();
        return '';
      })
      .filter(Boolean);
  }, []);

  const extractImageResultURL = useCallback((obj) => {
    if (!obj || typeof obj !== 'object') return '';
    const direct = ['url', 'image_url', 'output_url', 'result_url'];
    for (const key of direct) {
      const v = obj[key];
      if (typeof v === 'string' && v.trim()) return v.trim();
    }
    if (Array.isArray(obj.data)) {
      for (const item of obj.data) {
        const url = extractImageResultURL(item);
        if (url) return url;
      }
    }
    if (obj.data && typeof obj.data === 'object') {
      const url = extractImageResultURL(obj.data);
      if (url) return url;
    }
    return '';
  }, []);

  const pollImageTaskUntilReady = useCallback(
    async (taskId, updateMessage) => {
      if (!taskId) return;
      const maxAttempts = 120;
      for (let i = 0; i < maxAttempts; i++) {
        await new Promise((resolve) => setTimeout(resolve, 2000));
        let data = {};
        try {
          const response = await fetch(
            `${API_ENDPOINTS.IMAGE_GENERATIONS_FETCH_PREFIX}/${taskId}`,
            {
              method: 'GET',
              credentials: 'include',
              headers: {
                'Content-Type': 'application/json',
                'New-Api-User': getUserIdFromLocalStorage(),
              },
            },
          );
          if (!response.ok) {
            return;
          }
          const text = await response.text();
          data = text ? JSON.parse(text) : {};
        } catch {
          continue;
        }
        const taskData = data?.data && typeof data.data === 'object' ? data.data : data;
        const status = String(taskData?.status || '').toLowerCase();
        const progress = Number(taskData?.progress || 0);
        const url = extractImageResultURL(data);
        if (url) {
          updateMessage({
            content: `![generated-image-1](${url})`,
            status: MESSAGE_STATUS.COMPLETE,
          });
          return;
        }
        if (['completed', 'succeeded', 'success'].includes(status)) {
          updateMessage({
            content: `图片生成完成。\n\n${JSON.stringify(taskData, null, 2)}`,
            status: MESSAGE_STATUS.COMPLETE,
          });
          return;
        }
        if (['failed', 'error', 'cancelled'].includes(status)) {
          updateMessage({
            content: `图片生成失败（status=${status}）。\n\n${JSON.stringify(taskData, null, 2)}`,
            status: MESSAGE_STATUS.ERROR,
          });
          return;
        }
        updateMessage({
          content: `图片生成中，请稍后… (${Math.max(0, Math.min(100, progress))}%)`,
        });
      }
    },
    [extractImageResultURL],
  );

  const pollVideoTaskUntilReady = useCallback(
    async (taskId, updateMessage) => {
      if (!taskId) return;
      const maxAttempts = 120; // 最长约4分钟（2秒轮询）
      let consecutiveNetworkFailures = 0;
      for (let i = 0; i < maxAttempts; i++) {
        await new Promise((resolve) => setTimeout(resolve, 2000));
        let data = {};
        try {
          const response = await fetch(
            `${API_ENDPOINTS.VIDEO_GENERATIONS}/${taskId}`,
            {
              method: 'GET',
              credentials: 'include',
              headers: {
                'Content-Type': 'application/json',
                'New-Api-User': getUserIdFromLocalStorage(),
              },
            },
          );
          if (!response.ok) {
            const errorBody = await response.text();
            updateMessage({
              content: `视频轮询失败（HTTP ${response.status}），已停止自动轮询。\n\n${errorBody || ''}`.trim(),
              status: MESSAGE_STATUS.ERROR,
              videoTask: {
                taskId,
                status: 'error',
                progress: 0,
              },
            });
            return;
          }
          const text = await response.text();
          try {
            data = text ? JSON.parse(text) : {};
          } catch {
            updateMessage({
              content: `视频轮询失败（返回非 JSON），已停止自动轮询。\n\n${text?.slice?.(0, 300) || ''}`.trim(),
              status: MESSAGE_STATUS.ERROR,
              videoTask: {
                taskId,
                status: 'error',
                progress: 0,
              },
            });
            return;
          }
          consecutiveNetworkFailures = 0;
        } catch (err) {
          consecutiveNetworkFailures += 1;
          if (consecutiveNetworkFailures >= 3) {
            updateMessage({
              content: `视频轮询失败（网络异常 ${consecutiveNetworkFailures} 次），已停止自动轮询。\n\n${err?.message || ''}`.trim(),
              status: MESSAGE_STATUS.ERROR,
              videoTask: {
                taskId,
                status: 'error',
                progress: 0,
              },
            });
            return;
          }
          updateMessage({
            content: '视频生成中，请稍后…',
            videoTask: {
              taskId,
              status: 'queued',
              progress: 0,
            },
          });
          continue;
        }
        const taskPayload = getVideoTaskPayload(data);
        const status = String(taskPayload?.status || '').toLowerCase();
        const progress = Number(taskPayload?.progress || 0);
        const playable = extractVideoPlayableURL(data);
        console.debug('[playground-video] poll tick', {
          taskId,
          status: status || 'queued',
          progress,
          hasPlayable: !!playable,
        });
        updateMessage({
          content: '视频生成中，请稍后…',
          videoTask: {
            taskId,
            status: status || 'queued',
            progress,
          },
        });
        if (playable) {
          updateMessage(
            {
              content: '视频已生成完成。',
              status: MESSAGE_STATUS.COMPLETE,
              videoTask: {
                taskId,
                status: 'completed',
                progress: 100,
                playableUrl: playable,
              },
            },
          );
          return;
        }
        if (['succeeded', 'success', 'completed'].includes(status)) {
          updateMessage({
            content: `视频已生成完成。\n\n任务状态：${status}\n\n${JSON.stringify(taskPayload, null, 2)}`,
            status: MESSAGE_STATUS.COMPLETE,
            videoTask: {
              taskId,
              status: 'completed',
              progress: 100,
            },
          });
          return;
        }
        if (['failed', 'error', 'cancelled'].includes(status)) {
          updateMessage(
            {
              content: `视频生成失败（status=${status}）。\n\n${JSON.stringify(data, null, 2)}`,
              status: MESSAGE_STATUS.ERROR,
              videoTask: {
                taskId,
                status,
                progress,
              },
            },
          );
          return;
        }
      }
      updateMessage({
        content: '视频生成超时，请稍后在任务日志中查看。',
        status: MESSAGE_STATUS.ERROR,
        videoTask: {
          taskId,
          status: 'timeout',
          progress: 0,
        },
      });
    },
    [extractVideoPlayableURL, getVideoTaskPayload],
  );
  const resolveEndpoint = useCallback((payload) => {
    switch (payload?.__endpoint) {
      case 'image':
        return API_ENDPOINTS.IMAGE_GENERATIONS;
      case 'video':
        return API_ENDPOINTS.VIDEO_GENERATIONS;
      default:
        return API_ENDPOINTS.CHAT_COMPLETIONS;
    }
  }, []);

  const stripLocalFields = useCallback((payload) => {
    if (!payload || typeof payload !== 'object') return payload;
    const { __endpoint, ...rest } = payload;
    return rest;
  }, []);

  const { t } = useTranslation();

  // 处理消息自动关闭逻辑的公共函数
  const applyAutoCollapseLogic = useCallback(
    (message, isThinkingComplete = true) => {
      const shouldAutoCollapse =
        isThinkingComplete && !message.hasAutoCollapsed;
      return {
        isThinkingComplete,
        hasAutoCollapsed: shouldAutoCollapse || message.hasAutoCollapsed,
        isReasoningExpanded: shouldAutoCollapse
          ? false
          : message.isReasoningExpanded,
      };
    },
    [],
  );

  // 流式消息更新
  const streamMessageUpdate = useCallback(
    (textChunk, type) => {
      setMessage((prevMessage) => {
        const lastMessage = prevMessage[prevMessage.length - 1];
        if (!lastMessage) return prevMessage;
        if (lastMessage.role !== 'assistant') return prevMessage;
        if (lastMessage.status === MESSAGE_STATUS.ERROR) {
          return prevMessage;
        }

        if (
          lastMessage.status === MESSAGE_STATUS.LOADING ||
          lastMessage.status === MESSAGE_STATUS.INCOMPLETE
        ) {
          let newMessage = { ...lastMessage };

          if (type === 'reasoning') {
            newMessage = {
              ...newMessage,
              reasoningContent:
                (lastMessage.reasoningContent || '') + textChunk,
              status: MESSAGE_STATUS.INCOMPLETE,
              isThinkingComplete: false,
            };
          } else if (type === 'content') {
            const shouldCollapseReasoning =
              !lastMessage.content && lastMessage.reasoningContent;
            const newContent = (lastMessage.content || '') + textChunk;

            let shouldCollapseFromThinkTag = false;
            let thinkingCompleteFromTags = lastMessage.isThinkingComplete;

            if (
              lastMessage.isReasoningExpanded &&
              newContent.includes('</think>')
            ) {
              const thinkMatches = newContent.match(/<think>/g);
              const thinkCloseMatches = newContent.match(/<\/think>/g);
              if (
                thinkMatches &&
                thinkCloseMatches &&
                thinkCloseMatches.length >= thinkMatches.length
              ) {
                shouldCollapseFromThinkTag = true;
                thinkingCompleteFromTags = true; // think标签闭合也标记思考完成
              }
            }

            // 如果开始接收content内容，且之前有reasoning内容，或者think标签已闭合，则标记思考完成
            const isThinkingComplete =
              (lastMessage.reasoningContent &&
                !lastMessage.isThinkingComplete) ||
              thinkingCompleteFromTags;

            const autoCollapseState = applyAutoCollapseLogic(
              lastMessage,
              isThinkingComplete,
            );

            newMessage = {
              ...newMessage,
              content: newContent,
              status: MESSAGE_STATUS.INCOMPLETE,
              ...autoCollapseState,
            };
          }

          return [...prevMessage.slice(0, -1), newMessage];
        }

        return prevMessage;
      });
    },
    [setMessage, applyAutoCollapseLogic],
  );

  // 完成消息
  const completeMessage = useCallback(
    (status = MESSAGE_STATUS.COMPLETE) => {
      setMessage((prevMessage) => {
        const lastMessage = prevMessage[prevMessage.length - 1];
        if (
          lastMessage.status === MESSAGE_STATUS.COMPLETE ||
          lastMessage.status === MESSAGE_STATUS.ERROR
        ) {
          return prevMessage;
        }

        const autoCollapseState = applyAutoCollapseLogic(lastMessage, true);

        const updatedMessages = [
          ...prevMessage.slice(0, -1),
          {
            ...lastMessage,
            status: status,
            ...autoCollapseState,
          },
        ];

        // 在消息完成时保存，传入更新后的消息列表
        if (
          status === MESSAGE_STATUS.COMPLETE ||
          status === MESSAGE_STATUS.ERROR
        ) {
          setTimeout(() => saveMessages(updatedMessages), 0);
        }

        return updatedMessages;
      });
    },
    [setMessage, applyAutoCollapseLogic, saveMessages],
  );

  // 非流式请求
  const handleNonStreamRequest = useCallback(
    async (payload) => {
      const endpoint = resolveEndpoint(payload);
      const requestBody = stripLocalFields(payload);
      setDebugData((prev) => ({
        ...prev,
        request: requestBody,
        timestamp: new Date().toISOString(),
        response: null,
        sseMessages: null, // 非流式请求清除 SSE 消息
        isStreaming: false,
      }));
      setActiveDebugTab(DEBUG_TABS.REQUEST);

      try {
        const response = await fetch(endpoint, {
          method: 'POST',
          credentials: 'include',
          headers: {
            'Content-Type': 'application/json',
            'New-Api-User': getUserIdFromLocalStorage(),
          },
          body: JSON.stringify(requestBody),
        });

        if (!response.ok) {
          let errorBody = '';
          try {
            errorBody = await response.text();
          } catch (e) {
            errorBody = '无法读取错误响应体';
          }

          const errorInfo = handleApiError(
            new Error(
              `HTTP error! status: ${response.status}, body: ${errorBody}`,
            ),
            response,
          );

          setDebugData((prev) => ({
            ...prev,
            response: JSON.stringify(errorInfo, null, 2),
          }));
          setActiveDebugTab(DEBUG_TABS.RESPONSE);

          throw new Error(
            `HTTP error! status: ${response.status}, body: ${errorBody}`,
          );
        }

        const rawText = await response.text();
        let data;
        try {
          data = rawText ? JSON.parse(rawText) : {};
        } catch (parseError) {
          const snippet = rawText.slice(0, 120);
          throw new Error(
            `接口未返回 JSON（endpoint=${endpoint}），响应片段: ${snippet}`,
          );
        }

        setDebugData((prev) => ({
          ...prev,
          response: JSON.stringify(data, null, 2),
        }));
        setActiveDebugTab(DEBUG_TABS.RESPONSE);

        if (data.choices?.[0]) {
          const choice = data.choices[0];
          let content = choice.message?.content || '';
          let reasoningContent =
            choice.message?.reasoning_content ||
            choice.message?.reasoning ||
            '';

          const processed = processThinkTags(content, reasoningContent);

          setMessage((prevMessage) => {
            const newMessages = [...prevMessage];
            const lastMessage = newMessages[newMessages.length - 1];
            if (lastMessage?.status === MESSAGE_STATUS.LOADING) {
              const autoCollapseState = applyAutoCollapseLogic(
                lastMessage,
                true,
              );

              newMessages[newMessages.length - 1] = {
                ...lastMessage,
                content: processed.content,
                reasoningContent: processed.reasoningContent,
                status: MESSAGE_STATUS.COMPLETE,
                ...autoCollapseState,
              };
            }
            return newMessages;
          });
        } else if (payload?.__endpoint === 'image') {
          const imageUrls = extractImageURLs(data);
          const taskData = data?.data && typeof data.data === 'object' ? data.data : data;
          const imageTaskId =
            taskData?.task_id || taskData?.id || data?.task_id || data?.id;
          const imageStatus = String(taskData?.status || data?.status || '').toLowerCase();
          const shouldPollImage =
            !!imageTaskId && ['queued', 'processing', 'in_progress', 'running'].includes(imageStatus);
          setMessage((prevMessage) => {
            const newMessages = [...prevMessage];
            const lastMessage = newMessages[newMessages.length - 1];
            if (lastMessage?.status === MESSAGE_STATUS.LOADING) {
              const autoCollapseState = applyAutoCollapseLogic(lastMessage, true);
              const imageMarkdown = imageUrls
                .map((url, index) => `![generated-image-${index + 1}](${url})`)
                .join('\n\n');
              newMessages[newMessages.length - 1] = {
                ...lastMessage,
                content:
                  imageMarkdown ||
                  (shouldPollImage
                    ? '图片生成中，请稍后…'
                    : `图片生成完成。\n\n${JSON.stringify(data, null, 2)}`),
                status: MESSAGE_STATUS.COMPLETE,
                ...autoCollapseState,
              };
            }
            return newMessages;
          });
          if (shouldPollImage) {
            pollImageTaskUntilReady(imageTaskId, (patch) => {
              setMessage((prevMessage) => {
                const newMessages = [...prevMessage];
                const lastMessage = newMessages[newMessages.length - 1];
                if (!lastMessage || lastMessage.role !== 'assistant') return prevMessage;
                newMessages[newMessages.length - 1] = {
                  ...lastMessage,
                  ...(patch.content !== undefined ? { content: patch.content } : {}),
                  ...(patch.status ? { status: patch.status } : {}),
                };
                return newMessages;
              });
            });
          }
        } else {
          // 图片/视频等非 chat 返回体，统一展示摘要，避免停留在 loading
          const taskPayload = getVideoTaskPayload(data);
          const taskId = taskPayload?.task_id || taskPayload?.id || data?.task_id || data?.id;
          const shouldPollVideo = payload?.__endpoint === 'video' && !!taskId;
          setMessage((prevMessage) => {
            const newMessages = [...prevMessage];
            const lastMessage = newMessages[newMessages.length - 1];
            if (lastMessage?.status === MESSAGE_STATUS.LOADING) {
              const autoCollapseState = applyAutoCollapseLogic(lastMessage, true);
              newMessages[newMessages.length - 1] = {
                ...lastMessage,
                content:
                  payload?.__endpoint === 'video'
                    ? '视频生成中，请稍后…'
                    : JSON.stringify(data, null, 2),
                status: MESSAGE_STATUS.COMPLETE,
                videoTask:
                  payload?.__endpoint === 'video' && taskId
                    ? {
                        taskId,
                        status: String(taskPayload?.status || data?.status || 'queued').toLowerCase(),
                        progress: Number(taskPayload?.progress || data?.progress || 0),
                      }
                    : undefined,
                ...autoCollapseState,
              };
            }
            return newMessages;
          });
          if (shouldPollVideo) {
            console.debug('[playground-video] poll start', { taskId });
            pollVideoTaskUntilReady(taskId, (patch) => {
              setMessage((prevMessage) => {
                const newMessages = [...prevMessage];
                const lastMessage = newMessages[newMessages.length - 1];
                if (!lastMessage || lastMessage.role !== 'assistant') return prevMessage;
                newMessages[newMessages.length - 1] = {
                  ...lastMessage,
                  ...(patch.content !== undefined
                    ? { content: patch.content }
                    : {}),
                  ...(patch.status ? { status: patch.status } : {}),
                  ...(patch.videoTask !== undefined
                    ? { videoTask: patch.videoTask }
                    : {}),
                };
                return newMessages;
              });
            });
          }
        }
      } catch (error) {
        console.error('Non-stream request error:', error);

        const errorInfo = handleApiError(error);
        setDebugData((prev) => ({
          ...prev,
          response: JSON.stringify(errorInfo, null, 2),
        }));
        setActiveDebugTab(DEBUG_TABS.RESPONSE);

        setMessage((prevMessage) => {
          const newMessages = [...prevMessage];
          const lastMessage = newMessages[newMessages.length - 1];
          if (lastMessage?.status === MESSAGE_STATUS.LOADING) {
            const autoCollapseState = applyAutoCollapseLogic(lastMessage, true);

            newMessages[newMessages.length - 1] = {
              ...lastMessage,
              content: t('请求发生错误: ') + error.message,
              status: MESSAGE_STATUS.ERROR,
              ...autoCollapseState,
            };
          }
          return newMessages;
        });
      }
    },
    [
      setDebugData,
      setActiveDebugTab,
      setMessage,
      t,
      applyAutoCollapseLogic,
      resolveEndpoint,
      stripLocalFields,
      pollVideoTaskUntilReady,
      getVideoTaskPayload,
      extractImageURLs,
      pollImageTaskUntilReady,
    ],
  );

  // SSE请求
  const handleSSE = useCallback(
    (payload) => {
      const endpoint = resolveEndpoint(payload);
      const requestBody = stripLocalFields(payload);
      setDebugData((prev) => ({
        ...prev,
        request: requestBody,
        timestamp: new Date().toISOString(),
        response: null,
        sseMessages: [], // 新增：存储 SSE 消息数组
        isStreaming: true, // 新增：标记流式状态
      }));
      setActiveDebugTab(DEBUG_TABS.REQUEST);

      const source = new SSE(endpoint, {
        headers: {
          'Content-Type': 'application/json',
          'New-Api-User': getUserIdFromLocalStorage(),
        },
        method: 'POST',
        payload: JSON.stringify(requestBody),
      });

      sseSourceRef.current = source;

      let responseData = '';
      let hasReceivedFirstResponse = false;
      let isStreamComplete = false; // 添加标志位跟踪流是否正常完成

      source.addEventListener('message', (e) => {
        if (e.data === '[DONE]') {
          isStreamComplete = true; // 标记流正常完成
          source.close();
          sseSourceRef.current = null;
          setDebugData((prev) => ({
            ...prev,
            response: responseData,
            sseMessages: [...(prev.sseMessages || []), '[DONE]'], // 添加 DONE 标记
            isStreaming: false,
          }));
          completeMessage();
          return;
        }

        try {
          const payload = JSON.parse(e.data);
          responseData += e.data + '\n';

          if (!hasReceivedFirstResponse) {
            setActiveDebugTab(DEBUG_TABS.RESPONSE);
            hasReceivedFirstResponse = true;
          }

          // 新增：将 SSE 消息添加到数组
          setDebugData((prev) => ({
            ...prev,
            sseMessages: [...(prev.sseMessages || []), e.data],
          }));

          const delta = payload.choices?.[0]?.delta;
          if (delta) {
            if (delta.reasoning_content) {
              streamMessageUpdate(delta.reasoning_content, 'reasoning');
            }
            if (delta.reasoning) {
              streamMessageUpdate(delta.reasoning, 'reasoning');
            }
            if (delta.content) {
              streamMessageUpdate(delta.content, 'content');
            }
          }
        } catch (error) {
          console.error('Failed to parse SSE message:', error);
          const errorInfo = `解析错误: ${error.message}`;

          setDebugData((prev) => ({
            ...prev,
            response: responseData + `\n\nError: ${errorInfo}`,
            sseMessages: [...(prev.sseMessages || []), e.data], // 即使解析失败也保存原始数据
            isStreaming: false,
          }));
          setActiveDebugTab(DEBUG_TABS.RESPONSE);

          streamMessageUpdate(t('解析响应数据时发生错误'), 'content');
          completeMessage(MESSAGE_STATUS.ERROR);
        }
      });

      source.addEventListener('error', (e) => {
        // 只有在流没有正常完成且连接状态异常时才处理错误
        if (!isStreamComplete && source.readyState !== 2) {
          console.error('SSE Error:', e);
          const errorMessage = e.data || t('请求发生错误');

          const errorInfo = handleApiError(new Error(errorMessage));
          errorInfo.readyState = source.readyState;

          setDebugData((prev) => ({
            ...prev,
            response:
              responseData +
              '\n\nSSE Error:\n' +
              JSON.stringify(errorInfo, null, 2),
          }));
          setActiveDebugTab(DEBUG_TABS.RESPONSE);

          streamMessageUpdate(errorMessage, 'content');
          completeMessage(MESSAGE_STATUS.ERROR);
          sseSourceRef.current = null;
          source.close();
        }
      });

      source.addEventListener('readystatechange', (e) => {
        // 检查 HTTP 状态错误，但避免与正常关闭重复处理
        if (
          e.readyState >= 2 &&
          source.status !== undefined &&
          source.status !== 200 &&
          !isStreamComplete
        ) {
          const errorInfo = handleApiError(new Error('HTTP状态错误'));
          errorInfo.status = source.status;
          errorInfo.readyState = source.readyState;

          setDebugData((prev) => ({
            ...prev,
            response:
              responseData +
              '\n\nHTTP Error:\n' +
              JSON.stringify(errorInfo, null, 2),
          }));
          setActiveDebugTab(DEBUG_TABS.RESPONSE);

          source.close();
          streamMessageUpdate(t('连接已断开'), 'content');
          completeMessage(MESSAGE_STATUS.ERROR);
        }
      });

      try {
        source.stream();
      } catch (error) {
        console.error('Failed to start SSE stream:', error);
        const errorInfo = handleApiError(error);

        setDebugData((prev) => ({
          ...prev,
          response: 'Stream启动失败:\n' + JSON.stringify(errorInfo, null, 2),
        }));
        setActiveDebugTab(DEBUG_TABS.RESPONSE);

        streamMessageUpdate(t('建立连接时发生错误'), 'content');
        completeMessage(MESSAGE_STATUS.ERROR);
      }
    },
    [
      setDebugData,
      setActiveDebugTab,
      streamMessageUpdate,
      completeMessage,
      t,
      applyAutoCollapseLogic,
      resolveEndpoint,
      stripLocalFields,
    ],
  );

  // 停止生成
  const onStopGenerator = useCallback(() => {
    // 如果仍有活动的 SSE 连接，首先关闭
    if (sseSourceRef.current) {
      sseSourceRef.current.close();
      sseSourceRef.current = null;
    }

    // 无论是否存在 SSE 连接，都尝试处理最后一条正在生成的消息
    setMessage((prevMessage) => {
      if (prevMessage.length === 0) return prevMessage;
      const lastMessage = prevMessage[prevMessage.length - 1];

      if (
        lastMessage.status === MESSAGE_STATUS.LOADING ||
        lastMessage.status === MESSAGE_STATUS.INCOMPLETE
      ) {
        const processed = processIncompleteThinkTags(
          lastMessage.content || '',
          lastMessage.reasoningContent || '',
        );

        const autoCollapseState = applyAutoCollapseLogic(lastMessage, true);

        const updatedMessages = [
          ...prevMessage.slice(0, -1),
          {
            ...lastMessage,
            status: MESSAGE_STATUS.COMPLETE,
            reasoningContent: processed.reasoningContent || null,
            content: processed.content,
            ...autoCollapseState,
          },
        ];

        // 停止生成时也保存，传入更新后的消息列表
        setTimeout(() => saveMessages(updatedMessages), 0);

        return updatedMessages;
      }
      return prevMessage;
    });
  }, [setMessage, applyAutoCollapseLogic, saveMessages]);

  // 发送请求
  const sendRequest = useCallback(
    (payload, isStream) => {
      if (isStream) {
        handleSSE(payload);
      } else {
        handleNonStreamRequest(payload);
      }
    },
    [handleSSE, handleNonStreamRequest],
  );

  const startVideoTaskPolling = useCallback(
    (taskId, updateMessage) => {
      pollVideoTaskUntilReady(taskId, updateMessage);
    },
    [pollVideoTaskUntilReady],
  );

  return {
    sendRequest,
    onStopGenerator,
    startVideoTaskPolling,
    streamMessageUpdate,
    completeMessage,
  };
};
