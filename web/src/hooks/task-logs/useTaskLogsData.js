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

import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { Modal } from '@douyinfe/semi-ui';
import {
  API,
  copy,
  isAdmin,
  showError,
  showSuccess,
  timestamp2string,
  getLast7DaysStartTimestamp,
  getLast7DaysEndTimestamp,
} from '../../helpers';
import { ITEMS_PER_PAGE } from '../../constants';
import { useTableCompactMode } from '../common/useTableCompactMode';

const normalizeTagList = (csv) => {
  if (!csv || typeof csv !== 'string') return [];
  return [
    ...new Set(
      csv
        .replaceAll('，', ',')
        .replaceAll('、', ',')
        .split(',')
        .map((item) => item.trim())
        .filter(Boolean),
    ),
  ];
};

const hasTag = (csv, tag) => normalizeTagList(csv).includes(tag);

export const useTaskLogsData = () => {
  const { t } = useTranslation();

  // Define column keys for selection
  const COLUMN_KEYS = {
    SUBMIT_TIME: 'submit_time',
    FINISH_TIME: 'finish_time',
    DURATION: 'duration',
    CHANNEL: 'channel',
    USERNAME: 'username',
    PLATFORM: 'platform',
    TYPE: 'type',
    MODEL: 'model',
    TASK_ID: 'task_id',
    TASK_STATUS: 'task_status',
    PROGRESS: 'progress',
    FAIL_REASON: 'fail_reason',
    RESULT_URL: 'result_url',
  };

  // Basic state
  const [logs, setLogs] = useState([]);
  const [loading, setLoading] = useState(false);
  const [activePage, setActivePage] = useState(1);
  const [logCount, setLogCount] = useState(0);
  const [pageSize, setPageSize] = useState(ITEMS_PER_PAGE);
  const [modelOptions, setModelOptions] = useState([]);

  // User and admin
  const isAdminUser = isAdmin();
  // Role-specific storage key to prevent different roles from overwriting each other
  // v2：平台列默认隐藏，与旧缓存隔离以免沿用旧默认勾选
  const STORAGE_KEY = isAdminUser
    ? 'task-logs-table-columns-admin-v2'
    : 'task-logs-table-columns-user-v2';

  // Modal state
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [modalContent, setModalContent] = useState('');

  // 新增：视频预览弹窗状态
  const [isVideoModalOpen, setIsVideoModalOpen] = useState(false);
  const [videoUrl, setVideoUrl] = useState('');

  // Audio preview modal state
  const [isAudioModalOpen, setIsAudioModalOpen] = useState(false);
  const [audioClips, setAudioClips] = useState([]);

  // User info modal state
  const [showUserInfo, setShowUserInfoModal] = useState(false);
  const [userInfoData, setUserInfoData] = useState(null);

  // Form state
  const [formApi, setFormApi] = useState(null);

  const formInitValues = {
    channel_id: '',
    task_id: '',
    request_id: '',
    model_name: '',
    video_type: '',
    status: '',
    username: '',
    dateRange: [
      timestamp2string(getLast7DaysStartTimestamp()),
      timestamp2string(getLast7DaysEndTimestamp()),
    ],
  };

  // Column visibility state
  const [visibleColumns, setVisibleColumns] = useState({});
  const [showColumnSelector, setShowColumnSelector] = useState(false);

  // Compact mode
  const [compactMode, setCompactMode] = useTableCompactMode('taskLogs');

  // Load saved column preferences from localStorage
  useEffect(() => {
    const savedColumns = localStorage.getItem(STORAGE_KEY);
    if (savedColumns) {
      try {
        const parsed = JSON.parse(savedColumns);
        const defaults = getDefaultColumnVisibility();
        const merged = { ...defaults, ...parsed };

        // For non-admin users, force-hide admin-only columns (does not touch admin settings)
        if (!isAdminUser) {
          merged[COLUMN_KEYS.CHANNEL] = false;
          merged[COLUMN_KEYS.USERNAME] = false;
        }
        setVisibleColumns(merged);
      } catch (e) {
        console.error('Failed to parse saved column preferences', e);
        initDefaultColumns();
      }
    } else {
      initDefaultColumns();
    }
  }, []);

  // Get default column visibility based on user role
  const getDefaultColumnVisibility = () => {
    return {
      [COLUMN_KEYS.SUBMIT_TIME]: true,
      [COLUMN_KEYS.FINISH_TIME]: true,
      [COLUMN_KEYS.DURATION]: true,
      [COLUMN_KEYS.CHANNEL]: isAdminUser,
      [COLUMN_KEYS.USERNAME]: isAdminUser,
      [COLUMN_KEYS.PLATFORM]: false,
      [COLUMN_KEYS.TYPE]: true,
      [COLUMN_KEYS.MODEL]: true,
      [COLUMN_KEYS.TASK_ID]: true,
      [COLUMN_KEYS.TASK_STATUS]: true,
      [COLUMN_KEYS.PROGRESS]: true,
      [COLUMN_KEYS.FAIL_REASON]: true,
      [COLUMN_KEYS.RESULT_URL]: true,
    };
  };

  // Initialize default column visibility
  const initDefaultColumns = () => {
    const defaults = getDefaultColumnVisibility();
    setVisibleColumns(defaults);
    localStorage.setItem(STORAGE_KEY, JSON.stringify(defaults));
  };

  // Handle column visibility change
  const handleColumnVisibilityChange = (columnKey, checked) => {
    const updatedColumns = { ...visibleColumns, [columnKey]: checked };
    setVisibleColumns(updatedColumns);
  };

  // Handle "Select All" checkbox
  const handleSelectAll = (checked) => {
    const allKeys = Object.keys(COLUMN_KEYS).map((key) => COLUMN_KEYS[key]);
    const updatedColumns = {};

    allKeys.forEach((key) => {
      if (
        (key === COLUMN_KEYS.CHANNEL || key === COLUMN_KEYS.USERNAME) &&
        !isAdminUser
      ) {
        updatedColumns[key] = false;
      } else {
        updatedColumns[key] = checked;
      }
    });

    setVisibleColumns(updatedColumns);
  };

  // Persist column settings to the role-specific STORAGE_KEY
  useEffect(() => {
    if (Object.keys(visibleColumns).length > 0) {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(visibleColumns));
    }
  }, [visibleColumns]);

  // Get form values helper function
  const getFormValues = () => {
    const formValues = formApi ? formApi.getValues() : {};

    // 处理时间范围
    let start_timestamp = timestamp2string(getLast7DaysStartTimestamp());
    let end_timestamp = timestamp2string(getLast7DaysEndTimestamp());

    if (
      formValues.dateRange &&
      Array.isArray(formValues.dateRange) &&
      formValues.dateRange.length === 2
    ) {
      start_timestamp = formValues.dateRange[0];
      end_timestamp = formValues.dateRange[1];
    }

    return {
      channel_id: formValues.channel_id || '',
      task_id: formValues.task_id || '',
      request_id: String(formValues.request_id || '').trim(),
      model_name: String(formValues.model_name || '').trim(),
      video_type: formValues.video_type || '',
      status: formValues.status || '',
      username: String(formValues.username || '').trim(),
      start_timestamp,
      end_timestamp,
    };
  };

  const resolveTaskModelName = (log) => {
    let props = log?.properties;
    if (typeof props === 'string' && props.trim() !== '') {
      try {
        props = JSON.parse(props);
      } catch {
        props = null;
      }
    }
    if (!props || typeof props !== 'object') {
      return '';
    }
    return String(props.origin_model_name || props.upstream_model_name || '').trim();
  };

  // Enrich logs data
  const enrichLogs = (items) => {
    return items.map((log) => ({
      ...log,
      model_name: resolveTaskModelName(log),
      timestamp2string: timestamp2string(log.created_at),
      key: '' + log.id,
    }));
  };

  // Sync page data
  const syncPageData = (payload) => {
    const items = enrichLogs(payload.items || []);
    setLogs(items);
    setLogCount(payload.total || 0);
    setActivePage(payload.page || 1);
    setPageSize(payload.page_size || pageSize);
  };

  // Load logs function
  const loadLogs = async (page = 1, size = pageSize) => {
    setLoading(true);
    const {
      channel_id,
      task_id,
      request_id,
      model_name,
      video_type,
      status,
      username,
      start_timestamp,
      end_timestamp,
    } = getFormValues();
    let localStartTimestamp = parseInt(Date.parse(start_timestamp) / 1000);
    let localEndTimestamp = parseInt(Date.parse(end_timestamp) / 1000);
    const modelQuery = encodeURIComponent(model_name);
    const requestIdQuery = encodeURIComponent(request_id);
    const videoTypeQuery = encodeURIComponent(video_type);
    const statusQuery = encodeURIComponent(status);
    const usernameQuery = encodeURIComponent(username);
    let url = isAdminUser
      ? `/api/task/?p=${page}&page_size=${size}&channel_id=${channel_id}&task_id=${task_id}&request_id=${requestIdQuery}&model_name=${modelQuery}&video_type=${videoTypeQuery}&status=${statusQuery}&username=${usernameQuery}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}`
      : `/api/task/self?p=${page}&page_size=${size}&task_id=${task_id}&request_id=${requestIdQuery}&model_name=${modelQuery}&video_type=${videoTypeQuery}&status=${statusQuery}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}`;
    const res = await API.get(url);
    const { success, message, data } = res.data;
    if (success) {
      syncPageData(data);
    } else {
      showError(message);
    }
    setLoading(false);
  };

  // Page handlers
  const handlePageChange = (page) => {
    loadLogs(page, pageSize).then();
  };

  const handlePageSizeChange = async (size) => {
    localStorage.setItem('task-page-size', size + '');
    await loadLogs(1, size);
  };

  // Refresh function
  const refresh = async () => {
    await loadLogs(1, pageSize);
  };

  // Copy text function
  const copyText = async (text) => {
    if (await copy(text)) {
      showSuccess(t('已复制：') + text);
    } else {
      Modal.error({ title: t('无法复制到剪贴板，请手动复制'), content: text });
    }
  };

  // Modal handlers
  const openContentModal = (content) => {
    setModalContent(content);
    setIsModalOpen(true);
  };

  // 新增：打开视频预览弹窗
  const openVideoModal = (url) => {
    setVideoUrl(url);
    setIsVideoModalOpen(true);
  };

  const openAudioModal = (clips) => {
    setAudioClips(clips);
    setIsAudioModalOpen(true);
  };

  // User info function
  const showUserInfoFunc = async (userId) => {
    if (!isAdminUser) {
      return;
    }
    const res = await API.get(`/api/user/${userId}`);
    const { success, message, data } = res.data;
    if (success) {
      setUserInfoData(data);
      setShowUserInfoModal(true);
    } else {
      showError(message);
    }
  };

  const loadModelOptions = async () => {
    try {
      const res = await API.get('/api/user/models?scene=playground');
      const { success, message, data } = res.data || {};
      if (success) {
        const items = Array.isArray(data) ? data : data?.items || [];
        const options = items
          .filter((item) => hasTag(item?.tags || '', '视频'))
          .map((item) => {
            const name = item?.model_name || item?.model || '';
            return {
              label: name,
              value: name,
            };
          })
          .filter((opt) => opt.value);
        setModelOptions(options);
      } else {
        showError(message);
      }
    } catch (e) {
      showError(e.message || t('加载模型列表失败'));
    }
  };

  // Initialize data
  useEffect(() => {
    const localPageSize =
      parseInt(localStorage.getItem('task-page-size')) || ITEMS_PER_PAGE;
    setPageSize(localPageSize);
    loadModelOptions();
    loadLogs(1, localPageSize).then();
  }, []);

  return {
    // Basic state
    logs,
    loading,
    activePage,
    logCount,
    pageSize,
    isAdminUser,
    modelOptions,

    // Modal state
    isModalOpen,
    setIsModalOpen,
    modalContent,

    // 新增：视频弹窗状态
    isVideoModalOpen,
    setIsVideoModalOpen,
    videoUrl,

    // Audio preview modal
    isAudioModalOpen,
    setIsAudioModalOpen,
    audioClips,

    // Form state
    formApi,
    setFormApi,
    formInitValues,
    getFormValues,

    // Column visibility
    visibleColumns,
    showColumnSelector,
    setShowColumnSelector,
    handleColumnVisibilityChange,
    handleSelectAll,
    initDefaultColumns,
    COLUMN_KEYS,

    // Compact mode
    compactMode,
    setCompactMode,

    // User info modal
    showUserInfo,
    setShowUserInfoModal,
    userInfoData,
    showUserInfoFunc,

    // Functions
    loadLogs,
    handlePageChange,
    handlePageSizeChange,
    refresh,
    copyText,
    openContentModal,
    openVideoModal,
    openAudioModal,
    enrichLogs,
    syncPageData,

    // Translation
    t,
  };
};
