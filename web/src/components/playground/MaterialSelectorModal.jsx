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

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Button,
  Checkbox,
  Empty,
  Image,
  Modal,
  Spin,
  Tabs,
  Typography,
} from '@douyinfe/semi-ui';
import {
  IconRefresh,
} from '@douyinfe/semi-icons';
import {
  CheckCircle2,
  Clock,
  AlertCircle,
  Library,
  Image as ImageTypeIcon,
  Video as VideoTypeIcon,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { listMaterialAssets, listRealAssets } from '../../helpers/materialApi';
import { MaterialAssetType, MaterialStatus } from '../../constants';

const { Text } = Typography;

/** 稳定空数组，避免 default prop 每次渲染产生新引用触发 effect 循环 */
const EMPTY_EXISTING_ASSETS = [];

/**
 * 素材类型 -> 图标 + 文案（与素材管理页一致）
 */
const getAssetTypeMeta = (assetType, t) => {
  if (assetType === MaterialAssetType.VIDEO) {
    return {
      icon: VideoTypeIcon,
      label: t('视频'),
      color: 'var(--semi-color-warning)',
    };
  }
  return {
    icon: ImageTypeIcon,
    label: t('图片'),
    color: 'var(--semi-color-primary)',
  };
};

/**
 * 【需求3】素材状态图标映射
 */
const getStatusMeta = (status) => {
  switch (status) {
    case MaterialStatus.ACTIVE:
      return { icon: CheckCircle2, color: 'var(--semi-color-success)' };
    case MaterialStatus.FAILED:
      return { icon: AlertCircle, color: 'var(--semi-color-danger)' };
    case MaterialStatus.PENDING:
    default:
      return { icon: Clock, color: 'var(--semi-color-warning)' };
  }
};

/**
 * 【需求2/4】素材库选择弹窗组件。
 * 复用素材管理模块数据接口，支持单选/多选/全选/取消全选。
 *
 * Props:
 * - visible: boolean 弹窗显隐
 * - onClose: () => void 关闭回调
 * - onConfirm: (selectedAssets: Array) => void 确认选择回调
 * - existingAssets?: Array 当前已选素材（用于回显选中状态）
 */
const MaterialSelectorModal = ({
  visible,
  onClose,
  onConfirm,
  existingAssets = EMPTY_EXISTING_ASSETS,
}) => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();

  // 【需求1】数据同源：复用素材管理模块接口
  const [assets, setAssets] = useState([]);
  const [loading, setLoading] = useState(false);
  const [selectedIds, setSelectedIds] = useState(new Set());
  const [imgErrors, setImgErrors] = useState(new Set());
  // 素材分类标签：虚拟人像 / 真人人像
  const [activeTab, setActiveTab] = useState('virtual');

  // 加载素材列表（按标签切换接口：虚拟人像用 listMaterialAssets，真人人像用 listRealAssets）
  const loadAssets = useCallback(
    async (tab = activeTab) => {
      setLoading(true);
      try {
        const res =
          tab === 'real'
            ? await listRealAssets({ page: 1, pageSize: 100 })
            : await listMaterialAssets({ page: 1, pageSize: 100 });
        if (res?.success) {
          setAssets(res.data?.items || []);
        }
      } catch {
        // 静默失败，展示空状态
      } finally {
        setLoading(false);
      }
    },
    [activeTab],
  );

  // 弹窗打开时加载数据
  useEffect(() => {
    if (!visible) return;
    loadAssets();
    setImgErrors(new Set());
  }, [visible, loadAssets]);

  // 切换标签时重新加载
  const handleTabChange = useCallback((key) => {
    setActiveTab(key);
    setAssets([]);
    setSelectedIds(new Set());
    setImgErrors(new Set());
    loadAssets(key);
  }, [loadAssets]);

  // 弹窗打开时回显已选素材（与加载分离，避免 existingAssets 引用变化导致重复拉取/重置预览）
  useEffect(() => {
    if (!visible) return;
    const existingIdSet = new Set(
      (Array.isArray(existingAssets) ? existingAssets : [])
        .map((a) => String(a?.asset_id || ''))
        .filter(Boolean),
    );
    setSelectedIds(existingIdSet);
  }, [visible, existingAssets]);

  // 可选素材（仅 Active 状态）
  const selectableAssets = useMemo(
    () => assets.filter((a) => a.status === MaterialStatus.ACTIVE),
    [assets],
  );

  // 【需求4】全选/取消全选
  const allSelectableIds = useMemo(
    () => selectableAssets.map((a) => String(a.asset_id)),
    [selectableAssets],
  );
  const isAllSelected =
    allSelectableIds.length > 0 &&
    allSelectableIds.every((id) => selectedIds.has(id));

  // 【需求2】图片加载失败时展示占位内容，避免第三方 502/CORS 报错打断弹窗交互
  const handleImgError = useCallback((assetId) => {
    const id = String(assetId || '').trim();
    if (!id) return;
    setImgErrors((prev) => {
      if (prev.has(id)) return prev;
      const next = new Set(prev);
      next.add(id);
      return next;
    });
  }, []);

  const handleToggleSelectAll = useCallback(() => {
    if (isAllSelected) {
      // 取消全选：移除当前页所有可选素材
      setSelectedIds((prev) => {
        const next = new Set(prev);
        for (const id of allSelectableIds) {
          next.delete(id);
        }
        return next;
      });
    } else {
      // 全选：添加当前页所有可选素材
      setSelectedIds((prev) => {
        const next = new Set(prev);
        for (const id of allSelectableIds) {
          next.add(id);
        }
        return next;
      });
    }
  }, [isAllSelected, allSelectableIds]);

  // 【需求4】单选/取消选中
  const handleToggleAsset = useCallback((assetId) => {
    const id = String(assetId);
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  }, []);

  // 【需求4】确认选择后回填、关闭弹窗、清空缓存
  const handleConfirm = useCallback(() => {
    const selected = assets.filter((a) => selectedIds.has(String(a.asset_id)));
    if (onConfirm) {
      onConfirm(selected);
    }
    onClose?.();
  }, [assets, selectedIds, onConfirm, onClose]);

  // 渲染单个素材卡片
  const renderAssetCard = (asset) => {
    const assetId = String(asset.asset_id);
    const isSelected = selectedIds.has(assetId);
    const typeMeta = getAssetTypeMeta(asset.asset_type, t);
    const statusMeta = getStatusMeta(asset.status);
    const TypeIcon = typeMeta.icon;
    const StatusIcon = statusMeta.icon;
    const isActive = asset.status === MaterialStatus.ACTIVE;
    const isVideo = asset.asset_type === MaterialAssetType.VIDEO;
    const previewFailed = imgErrors.has(assetId);

    return (
      <div
        key={assetId}
        onClick={() => isActive && handleToggleAsset(assetId)}
        style={{
          position: 'relative',
          width: '100%',
          borderRadius: 8,
          border: isSelected
            ? '2px solid var(--semi-color-primary)'
            : '2px solid var(--semi-color-border)',
          overflow: 'hidden',
          cursor: isActive ? 'pointer' : 'not-allowed',
          opacity: isActive ? 1 : 0.5,
          transition: 'border-color 200ms ease, opacity 200ms ease',
          background: 'var(--semi-color-bg-0)',
        }}
      >
        {/* 预览图 */}
        <div
          style={{
            position: 'relative',
            width: '100%',
            height: 100,
            background: 'var(--semi-color-fill-1)',
            overflow: 'hidden',
          }}
        >
          {previewFailed ? (
            <div
              style={{
                width: '100%',
                height: 100,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                background: 'var(--semi-color-fill-1)',
                color: 'var(--semi-color-text-3)',
                fontSize: 12,
              }}
            >
              {t('图片加载失败')}
            </div>
          ) : isVideo ? (
            <video
              src={asset.url}
              preload='metadata'
              muted
              playsInline
              style={{ width: '100%', height: 100, objectFit: 'cover' }}
              onError={() => handleImgError(assetId)}
            />
          ) : (
            <Image
              src={asset.url}
              width='100%'
              height={100}
              style={{ objectFit: 'cover' }}
              alt={asset.name}
              preview={false}
              onError={() => handleImgError(assetId)}
            />
          )}
          {/* 左上角：图片/视频类型标签（对齐素材管理页） */}
          <div
            style={{
              position: 'absolute',
              top: 6,
              left: 6,
              zIndex: 2,
              display: 'flex',
              alignItems: 'center',
              gap: 4,
              padding: '2px 6px',
              borderRadius: 6,
              background: 'rgba(0,0,0,0.55)',
              color: '#fff',
            }}
          >
            <TypeIcon size={14} />
            <span style={{ fontSize: 12 }}>{typeMeta.label}</span>
          </div>
          {/* 选中标记 */}
          {isSelected && (
            <div
              style={{
                position: 'absolute',
                top: 6,
                right: 6,
                width: 22,
                height: 22,
                borderRadius: '50%',
                background: 'var(--semi-color-primary)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                color: '#fff',
                fontSize: 12,
                fontWeight: 'bold',
                zIndex: 2,
              }}
            >
              ✓
            </div>
          )}
          {/* 右下角：素材状态图标（可用/处理中/失败） */}
          <div
            style={{
              position: 'absolute',
              bottom: 6,
              right: 6,
              zIndex: 2,
              padding: '2px 4px',
              borderRadius: 4,
              background: 'rgba(0,0,0,0.55)',
              display: 'flex',
              alignItems: 'center',
            }}
          >
            <StatusIcon size={14} style={{ color: statusMeta.color }} />
          </div>
        </div>
        {/* 名称 */}
        <div style={{ padding: '6px 8px' }}>
          <Text
            ellipsis={{ showTooltip: true }}
            size='small'
            style={{ fontSize: 12 }}
          >
            {asset.name || t('未命名素材')}
          </Text>
        </div>
      </div>
    );
  };

  const selectedCount = selectedIds.size;

  return (
    <Modal
      title={
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <Library size={18} />
          <span>{t('从素材库选择')}</span>
        </div>
      }
      visible={visible}
      onCancel={onClose}
      width={isMobile ? 'calc(100vw - 32px)' : 680}
      footer={
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            width: '100%',
          }}
        >
          <Text type='tertiary' size='small'>
            {t('已选择 {{count}} 个素材', { count: selectedCount })}
          </Text>
          <div style={{ display: 'flex', gap: 8 }}>
            <Button onClick={onClose}>{t('取消')}</Button>
            <Button
              theme='solid'
              onClick={handleConfirm}
              disabled={selectedCount === 0}
            >
              {t('确认选择')}
            </Button>
          </div>
        </div>
      }
    >
      {/* 素材分类标签：虚拟人像 / 真人人像 */}
      <Tabs
        type='line'
        activeKey={activeTab}
        onChange={handleTabChange}
        style={{ marginBottom: 4 }}
      >
        <Tabs.TabPane tab={t('虚拟人像')} itemKey='virtual' />
        <Tabs.TabPane tab={t('真人人像')} itemKey='real' />
      </Tabs>

      {/* 工具栏：全选 + 刷新 */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          marginBottom: 12,
        }}
      >
        <Checkbox
          checked={isAllSelected}
          onChange={handleToggleSelectAll}
          disabled={selectableAssets.length === 0}
        >
          {t('全选')}
        </Checkbox>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <Typography.Text type='tertiary' size='small'>
            {t('仅支持Seedance2.0模型')}
          </Typography.Text>
          <Button
            icon={<IconRefresh />}
            size='small'
            onClick={loadAssets}
            loading={loading}
          >
            {t('刷新')}
          </Button>
        </div>
      </div>

      {/* 【需求1/2】素材列表 / 空状态 */}
      <Spin spinning={loading}>
        {assets.length === 0 && !loading ? (
          /* 【需求2】空状态兜底：引导创建素材，内置跳转链接 */
          <Empty
            style={{ padding: '32px 0' }}
            description={
              <div style={{ textAlign: 'center' }}>
                <Text type='tertiary'>
                  {t('暂无素材，请先在素材管理中创建素材')}
                </Text>
                <br />
                <Button
                  theme='solid'
                  type='primary'
                  size='small'
                  style={{ marginTop: 12 }}
                  onClick={() => {
                    onClose?.();
                    window.location.hash = '#/seedance';
                  }}
                >
                  {t('前往素材管理')}
                </Button>
              </div>
            }
          />
        ) : (
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: isMobile
                ? 'repeat(2, 1fr)'
                : 'repeat(4, 1fr)',
              gap: 10,
              maxHeight: '50vh',
              overflowY: 'auto',
              paddingRight: 4,
            }}
          >
            {assets.map((asset) => renderAssetCard(asset))}
          </div>
        )}
      </Spin>
    </Modal>
  );
};

export default MaterialSelectorModal;
