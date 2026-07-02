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

/**
 * 素材管理页面状态管理 Hook。
 * 负责：配置加载、素材列表加载、本地/在线上传、删除等业务逻辑与异常处理，
 * 与 UI 渲染解耦，便于复用与测试。
 */
import { useCallback, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  getMaterialConfig,
  listMaterialAssets,
  getMaterialAsset,
  uploadMaterialFile,
  uploadMaterialByURL,
  deleteMaterialAsset,
  showError,
  showSuccess,
} from '../../helpers';
import { detectAssetTypeByName, MaterialStatus } from '../../constants';

const PENDING_POLL_INTERVAL_MS = 5000;

// 素材库默认配置（接口未就绪时的兜底值）。
const DEFAULT_CONFIG = {
  enabled: false,
  ready: false,
  max_image_size_mb: 10,
  agreement_zh: '',
  agreement_en: '',
  agreement_detail_zh: '',
  agreement_detail_en: '',
};

/** 将上传/详情接口返回的素材合并进列表（置顶，按 asset_id 去重）。 */
const mergeAssetIntoList = (items, asset) => {
  if (!asset?.asset_id) return items;
  const rest = items.filter((a) => a.asset_id !== asset.asset_id);
  return [asset, ...rest];
};

export const useMaterialLibrary = () => {
  const { t } = useTranslation();

  const [config, setConfig] = useState(DEFAULT_CONFIG);
  const [assets, setAssets] = useState([]);
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(null);
  const [deletingId, setDeletingId] = useState(null);
  // 已勾选合规协议标记（上传前置校验）。
  const [agreed, setAgreed] = useState(false);

  const assetsRef = useRef(assets);
  assetsRef.current = assets;

  const maxSizeMB = config.max_image_size_mb || 10;

  // 加载前端配置：失败时静默回退默认值，不打断页面。
  const loadConfig = useCallback(async () => {
    try {
      const { success, data } = await getMaterialConfig();
      if (success && data) setConfig(data);
    } catch (e) {
      // 配置加载失败使用默认值，不弹错避免干扰。
    }
  }, []);

  // 加载素材列表：仅查本地库，不阻塞于上游同步。
  const loadAssets = useCallback(
    async ({ silent = false } = {}) => {
      if (!silent) setLoading(true);
      try {
        const { success, message, data } = await listMaterialAssets({
          page: 1,
          pageSize: 100,
        });
        if (success) {
          setAssets(data?.items || []);
        } else if (!silent) {
          showError(message || t('加载素材列表失败'));
        }
      } catch (e) {
        if (!silent) showError(t('加载素材列表失败'));
      } finally {
        if (!silent) setLoading(false);
      }
    },
    [t],
  );

  useEffect(() => {
    loadConfig();
    loadAssets();
  }, [loadConfig, loadAssets]);

  // 后台轻量轮询：仅对 Pending 素材调用详情接口，避免整表刷新。
  useEffect(() => {
    const pollPendingAssets = async () => {
      const pending = assetsRef.current.filter(
        (a) => a.status === MaterialStatus.PENDING,
      );
      if (pending.length === 0) return;

      const results = await Promise.all(
        pending.map(async (a) => {
          const { success, data } = await getMaterialAsset(a.asset_id);
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
    };

    const timer = setInterval(pollPendingAssets, PENDING_POLL_INTERVAL_MS);
    return () => clearInterval(timer);
  }, []);

  /**
   * 本地文件上传。
   * @param {File} fileInstance 原始文件实例
   * @returns {Promise<boolean>} 是否上传成功
   */
  const handleUploadFile = useCallback(
    async (fileInstance) => {
      // 上传前置校验：协议、文件、类型、大小。
      if (!agreed) {
        showError(t('请先阅读并勾选同意虚拟人像合规协议'));
        return false;
      }
      if (!fileInstance) {
        showError(t('请选择要上传的文件'));
        return false;
      }
      // 格式拦截：仅允许图片 / 视频。
      const type = detectAssetTypeByName(fileInstance.name);
      if (!type) {
        showError(t('仅支持上传图片或视频文件'));
        return false;
      }
      // 大小拦截。
      if (fileInstance.size > maxSizeMB * 1024 * 1024) {
        showError(t('文件超过大小限制（最大 {{n}}MB）', { n: maxSizeMB }));
        return false;
      }

      setUploading(true);
      setUploadProgress(0);
      try {
        const { success, message, data } = await uploadMaterialFile(
          fileInstance,
          agreed,
          {
            onUploadProgress: (ev) => {
              const total = ev.total || ev.loaded || 1;
              const raw = Math.round((ev.loaded * 100) / total);
              // 文件传输最多展示到 85%，剩余留给服务端 CreateAsset + GetAsset。
              setUploadProgress(Math.min(85, raw));
            },
          },
        );
        if (!success) {
          showError(message || t('上传失败'));
          return false;
        }
        // 服务端已轮询上游，直接结束进度并更新列表，不再前端二次轮询。
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
    [agreed, maxSizeMB, t],
  );

  /**
   * 在线资源链接上传。
   * @param {{url:string,name?:string}} payload
   * @returns {Promise<boolean>} 是否上传成功
   */
  const handleUploadByURL = useCallback(
    async ({ url, name }) => {
      if (!agreed) {
        showError(t('请先阅读并勾选同意虚拟人像合规协议'));
        return false;
      }
      const trimmed = (url || '').trim();
      // 链接格式拦截。
      if (!/^https?:\/\/.+/i.test(trimmed)) {
        showError(t('请输入合法的在线资源链接（http/https）'));
        return false;
      }
      // 按扩展名预判类型（后端会以 GetAsset 返回的 AssetType 为准）。
      const assetType = detectAssetTypeByName(trimmed);

      setUploading(true);
      setUploadProgress(10);
      try {
        const { success, message, data } = await uploadMaterialByURL({
          url: trimmed,
          name,
          assetType,
          agreed,
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
    [agreed, t],
  );

  /**
   * 删除素材（乐观更新列表，后台静默同步）。
   * @param {object} asset 素材对象
   * @returns {Promise<boolean>} 是否删除成功
   */
  const handleDelete = useCallback(
    async (asset) => {
      const assetId = asset?.asset_id;
      if (!assetId) return false;
      setDeletingId(assetId);
      try {
        const { success, message } = await deleteMaterialAsset(assetId);
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

  return {
    // 状态
    config,
    assets,
    loading,
    uploading,
    uploadProgress,
    deletingId,
    agreed,
    maxSizeMB,
    // 操作
    setAgreed,
    loadAssets,
    handleUploadFile,
    handleUploadByURL,
    handleDelete,
  };
};

export default useMaterialLibrary;
