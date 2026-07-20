/**
 * 真人分组管理页面状态管理 Hook。
 * 负责：真人认证会话创建/轮询、真人分组列表、真人素材列表/上传/删除等业务逻辑，
 * 与 UI 渲染解耦，便于复用与测试。
 *
 * 认证流程：
 *   1. startSession() 调用后端 CreateVisualValidateSession，展示二维码/H5链接。
 *   2. 前端每 3s 轮询 pollVisualResult，后端用 BytedToken 查询上游 GetVisualValidateResult。
 *   3. 认证成功后获取真人专属 GroupId，刷新分组列表。
 *   4. H5 链接 5 分钟有效，前端倒计时到期自动停止轮询。
 */
import { useCallback, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  createVisualSession,
  pollVisualResult,
  listRealGroups,
  deleteRealGroup,
  updateRealGroup,
  listRealAssets,
  uploadRealMaterialFile,
  uploadRealMaterialByURL,
  deleteRealMaterial,
  updateRealMaterial,
  getRealMaterial,
  showError,
  showSuccess,
} from '../../helpers';
import { detectAssetTypeByName, isUnsupportedMaterialAudioName, MaterialStatus } from '../../constants';

// 轮询间隔（3 秒）与最大轮询时长（5 分钟），与 H5 链接有效期对齐。
const POLL_INTERVAL_MS = 3000;
const POLL_MAX_DURATION_MS = 5 * 60 * 1000;

// 会话状态枚举（与后端 model.VisualSessionStatus 对齐）。
const SESSION_STATUS = {
  PENDING: 'pending',
  SUCCESS: 'success',
  FAILED: 'failed',
  EXPIRED: 'expired',
};

/** 将上传/详情接口返回的素材合并进列表（置顶，按 asset_id 去重）。 */
const mergeAssetIntoList = (items, asset) => {
  if (!asset?.asset_id) return items;
  const rest = items.filter((a) => a.asset_id !== asset.asset_id);
  return [asset, ...rest];
};

export const useRealPerson = () => {
  const { t } = useTranslation();

  // 真人分组列表。
  const [groups, setGroups] = useState([]);
  const [groupsLoading, setGroupsLoading] = useState(false);

  // 当前选中的分组（null 表示展示分组列表视图）。
  const [selectedGroup, setSelectedGroup] = useState(null);

  // 选中分组的素材列表。
  const [assets, setAssets] = useState([]);
  const [assetsLoading, setAssetsLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(null);
  const [deletingId, setDeletingId] = useState(null);
  const [agreed, setAgreed] = useState(false);

  // 认证会话状态。
  const [session, setSession] = useState(null); // { session_id, h5_link, qr_code, expires_at }
  const [sessionStatus, setSessionStatus] = useState(''); // pending/success/failed/expired
  const [sessionMessage, setSessionMessage] = useState('');
  const [sessionModalOpen, setSessionModalOpen] = useState(false);
  const [sessionCreating, setSessionCreating] = useState(false);

  // 轮询定时器与截止时间引用。
  const pollTimerRef = useRef(null);
  const pollDeadlineRef = useRef(0);

  // 加载真人分组列表。
  const loadGroups = useCallback(async () => {
    setGroupsLoading(true);
    try {
      const { success, message, data } = await listRealGroups();
      if (success) {
        setGroups(data?.items || []);
      } else if (message) {
        showError(message);
      }
    } catch (e) {
      showError(t('加载真人分组列表失败'));
    } finally {
      setGroupsLoading(false);
    }
  }, [t]);

  // 加载选中分组的素材列表。
  const loadAssets = useCallback(
    async (groupId, { silent = false } = {}) => {
      if (!groupId) {
        setAssets([]);
        return;
      }
      if (!silent) setAssetsLoading(true);
      try {
        const { success, message, data } = await listRealAssets({
          groupId,
          page: 1,
          pageSize: 100,
        });
        if (success) {
          setAssets(data?.items || []);
        } else if (!silent && message) {
          showError(message);
        }
      } catch (e) {
        if (!silent) showError(t('加载真人素材列表失败'));
      } finally {
        if (!silent) setAssetsLoading(false);
      }
    },
    [t],
  );

  useEffect(() => {
    loadGroups();
  }, [loadGroups]);

  // 停止轮询定时器。
  const stopPolling = useCallback(() => {
    if (pollTimerRef.current) {
      clearInterval(pollTimerRef.current);
      pollTimerRef.current = null;
    }
  }, []);

  // 单次轮询。
  const pollOnce = useCallback(
    async (sessionId) => {
      try {
        const { success, data } = await pollVisualResult(sessionId);
        if (!success || !data) return;
        const status = data.status;
        setSessionStatus(status);
        if (status === SESSION_STATUS.SUCCESS) {
          stopPolling();
          setSessionMessage('');
          showSuccess(t('真人认证成功'));
          // 刷新分组列表并自动进入新分组。
          await loadGroups();
          if (data.group_id) {
            // 认证成功后不自动选组，让用户手动点击进入。
          }
          setTimeout(() => setSessionModalOpen(false), 1500);
        } else if (status === SESSION_STATUS.FAILED || status === SESSION_STATUS.EXPIRED) {
          stopPolling();
          if (data.message) setSessionMessage(data.message);
        } else {
          // pending: 认证进行中，不展示上游内部错误（如 C500999），
          // 清除可能残留的提示信息。
          setSessionMessage('');
        }
      } catch (e) {
        // 网络异常静默忽略，下次轮询继续。
      }
    },
    [t, stopPolling, loadGroups],
  );

  // 启动轮询。
  const startPolling = useCallback(
    (sessionId) => {
      stopPolling();
      pollDeadlineRef.current = Date.now() + POLL_MAX_DURATION_MS;
      pollTimerRef.current = setInterval(() => {
        if (Date.now() >= pollDeadlineRef.current) {
          stopPolling();
          setSessionStatus(SESSION_STATUS.EXPIRED);
          setSessionMessage(t('认证会话已过期，请重新发起认证'));
          return;
        }
        pollOnce(sessionId);
      }, POLL_INTERVAL_MS);
    },
    [stopPolling, pollOnce],
  );

  // 组件卸载时清理定时器。
  useEffect(() => {
    return () => stopPolling();
  }, [stopPolling]);

  // 创建真人认证会话。
  const startSession = useCallback(async () => {
    setSessionCreating(true);
    setSessionStatus(SESSION_STATUS.PENDING);
    setSessionMessage('');
    try {
      const { success, message, data } = await createVisualSession();
      if (!success || !data) {
        showError(message || t('创建认证会话失败'));
        return;
      }
      setSession(data);
      setSessionStatus(data.status || SESSION_STATUS.PENDING);
      setSessionModalOpen(true);
      // 启动轮询。
      if (data.session_id) {
        startPolling(data.session_id);
      }
    } catch (e) {
      showError(t('创建认证会话失败'));
    } finally {
      setSessionCreating(false);
    }
  }, [t, startPolling]);

  // 关闭认证模态框（同时停止轮询）。
  const closeSessionModal = useCallback(() => {
    stopPolling();
    setSessionModalOpen(false);
  }, [stopPolling]);

  // 进入分组详情。
  const openGroup = useCallback(
    (group) => {
      setSelectedGroup(group);
      if (group?.group_id) {
        loadAssets(group.group_id);
      }
    },
    [loadAssets],
  );

  // 返回分组列表视图。
  const backToGroups = useCallback(() => {
    setSelectedGroup(null);
    setAssets([]);
  }, []);

  // 本地文件上传真人素材。
  const handleUploadFile = useCallback(
    async (fileInstance) => {
      if (!selectedGroup?.group_id) {
        showError(t('请先选择真人分组'));
        return false;
      }
      if (!agreed) {
        showError(t('请先阅读并勾选同意真人素材合规协议'));
        return false;
      }
      if (!fileInstance) {
        showError(t('请选择要上传的文件'));
        return false;
      }
      if (isUnsupportedMaterialAudioName(fileInstance.name)) {
        showError(t('音频素材仅支持 mp3/wav 格式'));
        return false;
      }
      const type = detectAssetTypeByName(fileInstance.name);
      if (!type) {
        showError(t('仅支持上传图片、视频或音频（mp3/wav）文件'));
        return false;
      }
      setUploading(true);
      setUploadProgress(0);
      try {
        const { success, message, data } = await uploadRealMaterialFile(
          fileInstance,
          agreed,
          selectedGroup.group_id,
          {
            onUploadProgress: (ev) => {
              const total = ev.total || ev.loaded || 1;
              const raw = Math.round((ev.loaded * 100) / total);
              setUploadProgress(Math.min(85, raw));
            },
          },
        );
        if (!success) {
          showError(message || t('上传失败'));
          return false;
        }
        setUploadProgress(100);
        showSuccess(
          data?.status === MaterialStatus.PENDING
            ? t('上传成功，素材仍在审核中')
            : t('上传成功'),
        );
        if (data) {
          setAssets((prev) => mergeAssetIntoList(prev, data));
        }
        return true;
      } catch (e) {
        showError(t('上传失败，请重试'));
        return false;
      } finally {
        setUploading(false);
        setUploadProgress(null);
      }
    },
    [selectedGroup, agreed, t],
  );

  // 在线链接上传真人素材。
  const handleUploadByURL = useCallback(
    async ({ url, name }) => {
      if (!selectedGroup?.group_id) {
        showError(t('请先选择真人分组'));
        return false;
      }
      if (!agreed) {
        showError(t('请先阅读并勾选同意真人素材合规协议'));
        return false;
      }
      const trimmed = (url || '').trim();
      if (!/^https?:\/\/.+/i.test(trimmed)) {
        showError(t('请输入合法的在线资源链接（http/https）'));
        return false;
      }
      if (isUnsupportedMaterialAudioName(trimmed)) {
        showError(t('音频素材仅支持 mp3/wav 格式'));
        return false;
      }
      const assetType = detectAssetTypeByName(trimmed);
      setUploading(true);
      setUploadProgress(10);
      try {
        const { success, message, data } = await uploadRealMaterialByURL({
          url: trimmed,
          name,
          assetType,
          agreed,
          groupId: selectedGroup.group_id,
        });
        if (!success) {
          showError(message || t('上传失败'));
          return false;
        }
        setUploadProgress(100);
        showSuccess(
          data?.status === MaterialStatus.PENDING
            ? t('上传成功，素材仍在审核中')
            : t('上传成功'),
        );
        if (data) {
          setAssets((prev) => mergeAssetIntoList(prev, data));
        }
        return true;
      } catch (e) {
        showError(t('上传失败，请重试'));
        return false;
      } finally {
        setUploading(false);
        setUploadProgress(null);
      }
    },
    [selectedGroup, agreed, t],
  );

  // 真人素材改名。
  const handleRename = useCallback(
    async (asset, name) => {
      const assetId = asset?.asset_id;
      const nextName = String(name || '').trim();
      if (!assetId || !nextName) {
        showError(t('素材名称不能为空'));
        return false;
      }
      try {
        const { success, message, data } = await updateRealMaterial(
          assetId,
          nextName,
        );
        if (!success) {
          showError(message || t('改名失败'));
          return false;
        }
        setAssets((prev) =>
          prev.map((a) =>
            a.asset_id === assetId
              ? { ...a, ...(data || {}), name: data?.name || nextName }
              : a,
          ),
        );
        showSuccess(t('改名成功'));
        return true;
      } catch (e) {
        showError(t('改名失败，请重试'));
        return false;
      }
    },
    [t],
  );

  // 删除单个真人素材。
  const handleDelete = useCallback(
    async (asset) => {
      const assetId = asset?.asset_id;
      if (!assetId) return false;
      setDeletingId(assetId);
      try {
        const { success, message } = await deleteRealMaterial(assetId);
        if (!success) {
          showError(message || t('删除失败'));
          return false;
        }
        setAssets((prev) => prev.filter((a) => a.asset_id !== assetId));
        showSuccess(t('删除成功'));
        return true;
      } catch (e) {
        showError(t('删除失败，请重试'));
        return false;
      } finally {
        setDeletingId(null);
      }
    },
    [t],
  );

  // 批量删除真人素材。
  const handleBatchDelete = useCallback(
    async (assetIds) => {
      if (!Array.isArray(assetIds) || assetIds.length === 0) return false;
      let okCount = 0;
      for (const id of assetIds) {
        try {
          const { success } = await deleteRealMaterial(id);
          if (success) okCount++;
        } catch (e) {
          // 继续删除其余素材。
        }
      }
      if (okCount > 0) {
        setAssets((prev) => prev.filter((a) => !assetIds.includes(a.asset_id)));
        showSuccess(t('已删除 {{n}} 个素材', { n: okCount }));
      }
      return okCount === assetIds.length;
    },
    [t],
  );

  // 删除真人分组。
  const handleDeleteGroup = useCallback(
    async (group) => {
      const groupId = group?.group_id;
      if (!groupId) return false;
      try {
        const { success, message } = await deleteRealGroup(groupId);
        if (!success) {
          showError(message || t('删除分组失败'));
          return false;
        }
        setGroups((prev) => prev.filter((g) => g.group_id !== groupId));
        // 如果当前正在查看该分组，返回列表视图。
        if (selectedGroup?.group_id === groupId) {
          backToGroups();
        }
        showSuccess(t('分组删除成功'));
        return true;
      } catch (e) {
        showError(t('删除分组失败，请重试'));
        return false;
      }
    },
    [t, selectedGroup, backToGroups],
  );

  // 更新真人分组名称和描述。
  const handleUpdateGroup = useCallback(
    async (group, { group_name, description }) => {
      const groupId = group?.group_id;
      if (!groupId) return false;
      try {
        const { success, message, data } = await updateRealGroup(groupId, { group_name, description });
        if (!success) {
          showError(message || t('更新分组失败'));
          return false;
        }
        const updated = {
          ...group,
          group_name: data?.group_name ?? group_name,
          description: data?.description ?? description,
        };
        setGroups((prev) => prev.map((g) => (g.group_id === groupId ? { ...g, ...updated } : g)));
        if (selectedGroup?.group_id === groupId) {
          setSelectedGroup((prev) => (prev ? { ...prev, ...updated } : prev));
        }
        showSuccess(t('分组更新成功'));
        return true;
      } catch (e) {
        showError(t('更新分组失败，请重试'));
        return false;
      }
    },
    [t, selectedGroup],
  );

  // 后台轮询 Pending 素材状态刷新。
  const assetsRef = useRef(assets);
  assetsRef.current = assets;
  useEffect(() => {
    if (!selectedGroup?.group_id) return;
    const timer = setInterval(async () => {
      const pending = assetsRef.current.filter(
        (a) => a.status === MaterialStatus.PENDING,
      );
      if (pending.length === 0) return;
      const results = await Promise.all(
        pending.map(async (a) => {
          const { success, data } = await getRealMaterial(a.asset_id);
          return success && data ? data : null;
        }),
      );
      const updates = new Map(
        results.filter(Boolean).map((item) => [item.asset_id, item]),
      );
      if (updates.size === 0) return;
      setAssets((prev) =>
        prev.map((a) =>
          updates.has(a.asset_id) ? { ...a, ...updates.get(a.asset_id) } : a,
        ),
      );
    }, 5000);
    return () => clearInterval(timer);
  }, [selectedGroup]);

  return {
    // 分组列表
    groups,
    groupsLoading,
    loadGroups,
    selectedGroup,
    openGroup,
    backToGroups,
    handleDeleteGroup,
    handleUpdateGroup,
    // 素材列表
    assets,
    assetsLoading,
    uploading,
    uploadProgress,
    deletingId,
    agreed,
    setAgreed,
    loadAssets,
    handleUploadFile,
    handleUploadByURL,
    handleRename,
    handleDelete,
    handleBatchDelete,
    // 认证会话
    session,
    sessionStatus,
    sessionMessage,
    sessionModalOpen,
    sessionCreating,
    setSessionModalOpen,
    startSession,
    closeSessionModal,
  };
};

export default useRealPerson;
