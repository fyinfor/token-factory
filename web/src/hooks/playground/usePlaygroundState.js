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

import { useState, useCallback, useRef, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import {
  getDefaultMessages,
  DEFAULT_CONFIG,
  DEBUG_TABS,
  MESSAGE_ROLES,
  MESSAGE_STATUS,
} from '../../constants/playground.constants';
import {
  loadConfig,
  saveConfig,
  loadMessages,
  saveMessages,
} from '../../components/playground/configStorage';
import { processIncompleteThinkTags } from '../../helpers';

/**
 * 从 localStorage 恢复的会话若末尾仍为「生成中」状态，刷新后无法继续 SSE，会一直转圈。
 * 在读出持久化数据时收口为已完成，并可选写回存储。
 */
const needsPersistedTailNormalize = (messages) => {
  if (!Array.isArray(messages) || messages.length === 0) return false;
  const last = messages[messages.length - 1];
  if (!last || last.role !== MESSAGE_ROLES.ASSISTANT) return false;
  return (
    last.status === MESSAGE_STATUS.LOADING ||
    last.status === MESSAGE_STATUS.INCOMPLETE
  );
};

const normalizePersistedPlaygroundMessages = (messages) => {
  if (!Array.isArray(messages) || messages.length === 0) return messages;
  if (!needsPersistedTailNormalize(messages)) return messages;
  const lastMsg = messages[messages.length - 1];
  const processed = processIncompleteThinkTags(
    lastMsg.content || '',
    lastMsg.reasoningContent || '',
  );
  return [
    ...messages.slice(0, -1),
    {
      ...lastMsg,
      status: MESSAGE_STATUS.COMPLETE,
      content: processed.content,
      reasoningContent: processed.reasoningContent || null,
      isThinkingComplete: true,
    },
  ];
};

const ensureUniqueMessageIds = (messages) => {
  if (!Array.isArray(messages)) return [];
  const seen = new Set();
  return messages.map((msg, index) => {
    const baseId =
      msg?.id !== undefined && msg?.id !== null && String(msg.id) !== ''
        ? String(msg.id)
        : `msg-${Date.now()}-${index}`;
    let nextId = baseId;
    if (seen.has(nextId)) {
      nextId = `${baseId}-${Date.now()}-${index}`;
    }
    seen.add(nextId);
    return { ...msg, id: nextId };
  });
};

export const usePlaygroundState = (userId) => {
  const { t } = useTranslation();

  // 使用惰性初始化，确保只在组件首次挂载时加载配置和消息
  const [savedConfig] = useState(() => loadConfig());
  const [initialMessages] = useState(() => {
    const loaded = loadMessages(userId);
    // 检查是否是旧的中文默认消息，如果是则清除
    if (
      loaded &&
      loaded.length === 2 &&
      loaded[0].id === '2' &&
      loaded[1].id === '3'
    ) {
      const hasOldChinese =
        loaded[0].content === '你好' ||
        loaded[1].content === '你好，请问有什么可以帮助您的吗？' ||
        loaded[1].content === '你好！很高兴见到你。有什么我可以帮助你的吗？';

      if (hasOldChinese) {
        // 清除旧的默认消息
        localStorage.removeItem('playground_messages');
        return null;
      }
    }
    if (!loaded) return null;
    const normalized = normalizePersistedPlaygroundMessages(loaded);
    if (needsPersistedTailNormalize(loaded)) {
      setTimeout(() => saveMessages(normalized, userId), 0);
    }
    return normalized;
  });

  // 基础配置状态
  const [inputs, setInputs] = useState(
    savedConfig.inputs || DEFAULT_CONFIG.inputs,
  );
  const [parameterEnabled, setParameterEnabled] = useState(
    savedConfig.parameterEnabled || DEFAULT_CONFIG.parameterEnabled,
  );
  const [showDebugPanel, setShowDebugPanel] = useState(
    savedConfig.showDebugPanel || DEFAULT_CONFIG.showDebugPanel,
  );
  const [customRequestMode, setCustomRequestMode] = useState(
    savedConfig.customRequestMode || DEFAULT_CONFIG.customRequestMode,
  );
  const [customRequestBody, setCustomRequestBody] = useState(
    savedConfig.customRequestBody || DEFAULT_CONFIG.customRequestBody,
  );

  // UI状态
  const [showSettings, setShowSettings] = useState(false);
  const [models, setModels] = useState([]);
  const [modelTypes, setModelTypes] = useState([]);
  const [supplierOptions, setSupplierOptions] = useState([]);
  const [groups, setGroups] = useState([]);
  const [status, setStatus] = useState({});

  // 消息相关状态 - 使用加载的消息或默认消息初始化
  const [message, rawSetMessage] = useState(() =>
    ensureUniqueMessageIds(initialMessages || getDefaultMessages(t)),
  );
  const setMessage = useCallback((updater) => {
    rawSetMessage((prev) => {
      const next = typeof updater === 'function' ? updater(prev) : updater;
      return ensureUniqueMessageIds(next);
    });
  }, []);

  // 当语言改变时，如果是默认消息则更新
  useEffect(() => {
    // 只在没有保存的消息时才更新默认消息
    if (!initialMessages) {
      setMessage(getDefaultMessages(t));
    }
  }, [t, initialMessages]); // 当语言改变时

  // 当用户ID变化时，重新加载该用户的消息
  useEffect(() => {
    const loaded = loadMessages(userId);
    if (loaded) {
      const normalized = normalizePersistedPlaygroundMessages(loaded);
      if (needsPersistedTailNormalize(loaded)) {
        saveMessages(normalized, userId);
      }
      setMessage(normalized);
    } else {
      setMessage(getDefaultMessages(t));
    }
  }, [userId, t]);

  // 调试状态
  const [debugData, setDebugData] = useState({
    request: null,
    response: null,
    timestamp: null,
    previewRequest: null,
    previewTimestamp: null,
  });
  const [activeDebugTab, setActiveDebugTab] = useState(DEBUG_TABS.PREVIEW);
  const [previewPayload, setPreviewPayload] = useState(null);

  // 编辑状态
  const [editingMessageId, setEditingMessageId] = useState(null);
  const [editValue, setEditValue] = useState('');

  // Refs
  const sseSourceRef = useRef(null);
  const chatRef = useRef(null);
  const saveConfigTimeoutRef = useRef(null);
  const saveMessagesTimeoutRef = useRef(null);

  // 配置更新函数
  const handleInputChange = useCallback((name, value) => {
    setInputs((prev) => ({ ...prev, [name]: value }));
  }, []);

  const handleParameterToggle = useCallback((paramName) => {
    setParameterEnabled((prev) => ({
      ...prev,
      [paramName]: !prev[paramName],
    }));
  }, []);

  // 消息保存函数 - 改为立即保存，可以接受参数
  const saveMessagesImmediately = useCallback(
    (messagesToSave) => {
      // 如果提供了参数，使用参数；否则使用当前状态
      saveMessages(ensureUniqueMessageIds(messagesToSave || message), userId);
    },
    [message, userId],
  );

  // 配置保存
  const debouncedSaveConfig = useCallback(() => {
    if (saveConfigTimeoutRef.current) {
      clearTimeout(saveConfigTimeoutRef.current);
    }

    saveConfigTimeoutRef.current = setTimeout(() => {
      const configToSave = {
        inputs,
        parameterEnabled,
        showDebugPanel,
        customRequestMode,
        customRequestBody,
      };
      saveConfig(configToSave);
    }, 1000);
  }, [
    inputs,
    parameterEnabled,
    showDebugPanel,
    customRequestMode,
    customRequestBody,
  ]);

  // 配置导入/重置
  const handleConfigImport = useCallback((importedConfig) => {
    if (importedConfig.inputs) {
      setInputs((prev) => ({ ...prev, ...importedConfig.inputs }));
    }
    if (importedConfig.parameterEnabled) {
      setParameterEnabled((prev) => ({
        ...prev,
        ...importedConfig.parameterEnabled,
      }));
    }
    if (typeof importedConfig.showDebugPanel === 'boolean') {
      setShowDebugPanel(importedConfig.showDebugPanel);
    }
    if (importedConfig.customRequestMode) {
      setCustomRequestMode(importedConfig.customRequestMode);
    }
    if (importedConfig.customRequestBody) {
      setCustomRequestBody(importedConfig.customRequestBody);
    }
    // 如果导入的配置包含消息，也恢复消息
    if (importedConfig.messages && Array.isArray(importedConfig.messages)) {
      setMessage(ensureUniqueMessageIds(importedConfig.messages));
    }
  }, []);

  const handleConfigReset = useCallback((options = {}) => {
    const { resetMessages = false } = options;

    setInputs(DEFAULT_CONFIG.inputs);
    setParameterEnabled(DEFAULT_CONFIG.parameterEnabled);
    setShowDebugPanel(DEFAULT_CONFIG.showDebugPanel);
    setCustomRequestMode(DEFAULT_CONFIG.customRequestMode);
    setCustomRequestBody(DEFAULT_CONFIG.customRequestBody);

    // 只有在明确指定时才重置消息
    if (resetMessages) {
      setMessage([]);
      setTimeout(() => {
        setMessage(getDefaultMessages(t));
      }, 0);
    }
  }, []);

  // 清理定时器
  useEffect(() => {
    return () => {
      if (saveConfigTimeoutRef.current) {
        clearTimeout(saveConfigTimeoutRef.current);
      }
    };
  }, []);

  return {
    // 配置状态
    inputs,
    parameterEnabled,
    showDebugPanel,
    customRequestMode,
    customRequestBody,

    // UI状态
    showSettings,
    models,
    modelTypes,
    supplierOptions,
    groups,
    status,

    // 消息状态
    message,

    // 调试状态
    debugData,
    activeDebugTab,
    previewPayload,

    // 编辑状态
    editingMessageId,
    editValue,

    // Refs
    sseSourceRef,
    chatRef,
    saveConfigTimeoutRef,

    // 更新函数
    setInputs,
    setParameterEnabled,
    setShowDebugPanel,
    setCustomRequestMode,
    setCustomRequestBody,
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
    setEditingMessageId,
    setEditValue,

    // 处理函数
    handleInputChange,
    handleParameterToggle,
    debouncedSaveConfig,
    saveMessagesImmediately,
    handleConfigImport,
    handleConfigReset,
  };
};
