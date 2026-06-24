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

import React, { useMemo, useState } from 'react';
import {
  Banner,
  Button,
  Card,
  Checkbox,
  Empty,
  Image,
  Input,
  Modal,
  Popconfirm,
  Progress,
  Space,
  Spin,
  Tabs,
  Tooltip,
  Typography,
  Upload,
} from '@douyinfe/semi-ui';
import {
  IconCopy,
  IconUpload,
  IconLink,
  IconDelete,
  IconRefresh,
  IconHelpCircle,
} from '@douyinfe/semi-icons';
import {
  Image as ImageTypeIcon,
  Video as VideoTypeIcon,
  CheckCircle2,
  Clock,
  AlertCircle,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { copy, showSuccess, showError } from '../../helpers';
import { MaterialAssetType, MaterialStatus } from '../../constants';
import { useMaterialLibrary } from '../../hooks/seedance/useMaterialLibrary';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { useSmoothUploadProgress } from '../distributor/useSmoothUploadProgress';

const { Title, Text } = Typography;

/* ========================= 工具方法（纯展示映射） ========================= */

// 素材类型 -> 图标 + 文案映射（替代纯文字标签）。
const getAssetTypeMeta = (assetType, t) => {
  if (assetType === MaterialAssetType.VIDEO) {
    return { icon: VideoTypeIcon, label: t('视频'), color: 'var(--semi-color-warning)' };
  }
  // 默认按图片处理（音频已在后端拦截，理论上不会出现在列表）。
  return { icon: ImageTypeIcon, label: t('图片'), color: 'var(--semi-color-primary)' };
};

// 素材状态 -> 图标 + 颜色映射（仅渲染图标，不展示状态文字）。
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

/* ========================= 页面组件 ========================= */

const MaterialLibrary = () => {
  const { t, i18n } = useTranslation();
  const isEn = (i18n.language || '').toLowerCase().startsWith('en');

  // 业务状态与操作统一收敛到 hook，组件仅负责渲染与交互。
  const {
    config,
    assets,
    loading,
    uploading,
    uploadProgress,
    deletingId,
    agreed,
    maxSizeMB,
    setAgreed,
    loadAssets,
    handleUploadFile,
    handleUploadByURL,
    handleDelete,
  } = useMaterialLibrary();
  const isMobile = useIsMobile();

  // 协议详情弹窗 / 在线链接上传弹窗的本地 UI 状态。
  const [detailVisible, setDetailVisible] = useState(false);
  const [urlModalVisible, setUrlModalVisible] = useState(false);
  const [urlInput, setUrlInput] = useState('');
  const [urlName, setUrlName] = useState('');

  const agreementText = isEn ? config.agreement_en : config.agreement_zh;
  const agreementDetail = isEn
    ? config.agreement_detail_en
    : config.agreement_detail_zh;

  // 将协议文案中的《...》渲染为可点击查看详情的链接。
  const agreementNode = useMemo(() => {
    const text = agreementText || '';
    const start = text.indexOf('《');
    const end = text.indexOf('》');
    if (start >= 0 && end > start) {
      return (
        <span>
          {text.slice(0, start)}
          <Text
            link
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
              setDetailVisible(true);
            }}
          >
            {text.slice(start, end + 1)}
          </Text>
          {text.slice(end + 1)}
        </span>
      );
    }
    return <span>{text}</span>;
  }, [agreementText]);

  const uploadDisabled = !agreed || !config.ready || uploading;
  const uploadDisplayPct = useSmoothUploadProgress(uploadProgress);

  // 【改动】每用户唯一固定素材分组：从已有列表数据推导素材库 ID，供页面级展示（不新增接口请求）。
  const libraryId = useMemo(() => {
    const found = assets.find((a) => a.group_id);
    return found?.group_id || '';
  }, [assets]);

  /* --------------------------- 交互事件 --------------------------- */

  // Semi Upload 自定义请求：交由 hook 统一处理上传与异常。
  const customRequest = async ({ file, onSuccess, onError }) => {
    const inst = file?.fileInstance || file;
    const ok = await handleUploadFile(inst);
    if (ok) {
      onSuccess && onSuccess({});
    } else {
      onError && onError({ message: 'upload failed' });
    }
  };

  // 在线链接上传提交。
  const submitUrlUpload = async () => {
    const ok = await handleUploadByURL({ url: urlInput, name: urlName });
    if (ok) {
      setUrlModalVisible(false);
      setUrlInput('');
      setUrlName('');
    }
  };

  // 复制 asset:// 资源地址（用于替换图片资源地址）。
  const handleCopyURI = async (asset) => {
    const ok = await copy(asset.asset_uri);
    if (ok) {
      showSuccess(t('已复制资源地址，可替换图片资源地址'));
    } else {
      showError(t('复制失败，请手动复制：') + asset.asset_uri);
    }
  };

  // 【改动】复制页面级素材库 ID（原卡片内分组 ID 已迁移至标题下方）。
  const handleCopyLibraryId = async () => {
    if (!libraryId) {
      showError(t('暂无素材库 ID'));
      return;
    }
    const ok = await copy(libraryId);
    if (ok) {
      showSuccess(t('已复制素材库 ID'));
    } else {
      showError(t('复制失败，请手动复制：') + libraryId);
    }
  };

  /* --------------------------- 渲染：单张素材卡片 --------------------------- */

  const renderAssetCard = (asset) => {
    const typeMeta = getAssetTypeMeta(asset.asset_type, t);
    const statusMeta = getStatusMeta(asset.status);
    const TypeIcon = typeMeta.icon;
    const StatusIcon = statusMeta.icon;
    const isVideo = asset.asset_type === MaterialAssetType.VIDEO;

    return (
      <Card
        key={asset.id}
        className='material-asset-card'
        style={{ width: 200 }}
        bodyStyle={{ padding: 0 }}
        bordered
      >
        {/* 预览区 */}
        <div
          style={{
            position: 'relative',
            height: 150,
            background: 'var(--semi-color-fill-1)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            overflow: 'hidden',
          }}
        >
          {/* 类型图标徽标（替代纯文字标签） */}
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

          {isVideo ? (
            <video
              src={asset.url}
              controls
              preload='metadata'
              style={{ width: '100%', height: '100%', objectFit: 'cover' }}
            />
          ) : (
            <Image
              src={asset.url}
              width='100%'
              height={150}
              style={{ objectFit: 'cover' }}
              alt={asset.name}
            />
          )}
        </div>

        {/* 信息区 */}
        <div style={{ padding: '8px 10px' }}>
          {/* 名称 + 状态图标（仅图标，无状态文字） */}
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              gap: 6,
              marginBottom: 8,
            }}
          >
            <Tooltip content={asset.name}>
              <Text
                ellipsis={{ showTooltip: false }}
                style={{ maxWidth: 140, fontSize: 13 }}
              >
                {asset.name || t('未命名素材')}
              </Text>
            </Tooltip>
            <StatusIcon size={16} style={{ color: statusMeta.color, flexShrink: 0 }} />
          </div>

          {/* 操作区：复制资源地址 + 删除 */}
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              gap: 6,
            }}
          >
            <Tooltip content={asset.asset_uri}>
              <Button
                size='small'
                icon={<IconCopy />}
                onClick={() => handleCopyURI(asset)}
              >
                {t('复制地址')}
              </Button>
            </Tooltip>

            {/* 删除二次确认 */}
            <Popconfirm
              title={t('确认删除该素材？')}
              content={t('删除后将同时移除云端资产，且不可恢复。')}
              okType='danger'
              okText={t('删除')}
              cancelText={t('取消')}
              onConfirm={() => handleDelete(asset)}
            >
              <Button
                size='small'
                type='danger'
                theme='borderless'
                icon={<IconDelete />}
                loading={deletingId === asset.id}
                aria-label={t('删除')}
              />
            </Popconfirm>
          </div>
        </div>
      </Card>
    );
  };

  /* --------------------------- 渲染：主体 --------------------------- */

  return (
    <Card style={{ minHeight: '70vh' }}>
      {/* 标题栏 */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          marginBottom: 12,
        }}
      >
        <Title heading={4} style={{ margin: 0 }}>
          {t('Seedance2.0合规素材库')}
        </Title>
        <Tooltip
          content={t(
            '上传已授权的虚拟人像素材，获取 asset:// 资源地址后可用于视频生成。',
          )}
        >
          <IconHelpCircle style={{ color: 'var(--semi-color-text-2)' }} />
        </Tooltip>
        <div style={{ flex: 1 }} />
        <Text type='tertiary'>{t('共 {{n}} 个素材', { n: assets.length })}</Text>
        <Tooltip content={t('刷新列表')}>
          <Button
            theme='borderless'
            type='tertiary'
            icon={<IconRefresh />}
            loading={loading}
            onClick={loadAssets}
          />
        </Tooltip>
      </div>

      {/* 【改动】素材库 ID：页面级展示（每用户唯一固定分组，不再在素材卡片内重复展示） */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 6,
          marginBottom: 12,
        }}
      >
        <Text type='tertiary' size='small' style={{ flexShrink: 0 }}>
          {t('素材库ID')}:
        </Text>
        <Tooltip content={libraryId || t('暂无')}>
          <Text
            type='tertiary'
            size='small'
            ellipsis={{ showTooltip: false }}
            style={{ maxWidth: 360 }}
          >
            {libraryId || t('暂无')}
          </Text>
        </Tooltip>
        <Tooltip content={t('复制素材库 ID')}>
          <Button
            size='small'
            theme='borderless'
            type='tertiary'
            icon={<IconCopy size='small' />}
            disabled={!libraryId}
            onClick={handleCopyLibraryId}
          />
        </Tooltip>
      </div>

      {/* 状态提示 */}
      {!config.enabled && (
        <Banner
          type='warning'
          description={t('素材库功能未启用，请联系管理员在系统设置中开启。')}
          style={{ marginBottom: 12 }}
        />
      )}
      {config.enabled && !config.ready && (
        <Banner
          type='warning'
          description={t('素材库 API 基础地址未配置，请联系管理员。')}
          style={{ marginBottom: 12 }}
        />
      )}

      <Tabs type='line' defaultActiveKey='portrait'>
        <Tabs.TabPane tab={t('虚拟人像')} itemKey='portrait'>
          {/* 协议勾选 */}
          <div style={{ margin: '12px 0' }}>
            <Checkbox
              checked={agreed}
              onChange={(e) => setAgreed(e.target.checked)}
            >
              {agreementNode}
            </Checkbox>
          </div>

          {/* 上传入口：本地文件 / 在线链接 */}
          <Space style={{ marginBottom: 16 }}>
            <Upload
              action=''
              accept='image/*,video/*'
              showUploadList={false}
              disabled={uploadDisabled}
              customRequest={customRequest}
            >
              <Button
                icon={<IconUpload />}
                theme='solid'
                loading={uploading}
                disabled={uploadDisabled}
              >
                {t('本地上传')}
              </Button>
            </Upload>

            <Button
              icon={<IconLink />}
              disabled={uploadDisabled}
              onClick={() => setUrlModalVisible(true)}
            >
              {t('链接上传')}
            </Button>

            <Text type='tertiary' size='small'>
              {t('支持图片/视频，单个文件不超过 {{n}}MB', { n: maxSizeMB })}
            </Text>
          </Space>

          {uploadProgress != null && (
            <Progress
              percent={uploadDisplayPct ?? uploadProgress}
              showInfo
              style={{ marginBottom: 16, maxWidth: 480 }}
            />
          )}

          {/* 素材列表 */}
          <Spin spinning={loading}>
            <div
              style={{
                display: 'flex',
                flexWrap: 'wrap',
                gap: 16,
              }}
            >
              {assets.map((asset) => renderAssetCard(asset))}
            </div>

            {assets.length === 0 && !loading && (
              <Empty
                style={{ marginTop: 24 }}
                description={t('暂无素材，点击「本地上传」或「链接上传」添加')}
              />
            )}
          </Spin>
        </Tabs.TabPane>
      </Tabs>

      {/* 在线链接上传弹窗 */}
      <Modal
        title={t('通过在线链接上传')}
        visible={urlModalVisible}
        onCancel={() => setUrlModalVisible(false)}
        onOk={submitUrlUpload}
        okText={t('上传')}
        cancelText={t('取消')}
        confirmLoading={uploading}
      >
        <div style={{ marginBottom: 12 }}>
          <Text>{t('资源链接')}</Text>
          <Input
            value={urlInput}
            onChange={setUrlInput}
            placeholder='https://example.com/portrait.png'
            style={{ marginTop: 4 }}
          />
        </div>
        <div>
          <Text>{t('素材名称（可选）')}</Text>
          <Input
            value={urlName}
            onChange={setUrlName}
            placeholder={t('不填则自动取链接文件名')}
            style={{ marginTop: 4 }}
          />
        </div>
        <Banner
          type='info'
          closeIcon={null}
          description={t('仅支持图片或视频的公开可访问链接（http/https）。')}
          style={{ marginTop: 12 }}
        />
      </Modal>

      {/* 协议详情弹窗 */}
      <Modal
        title={t('虚拟人像合规承诺函')}
        visible={detailVisible}
        onCancel={() => setDetailVisible(false)}
        width={isMobile ? 'calc(100vw - 32px)' : 720}
        footer={
          <Button theme='solid' onClick={() => setDetailVisible(false)}>
            {t('我已知晓')}
          </Button>
        }
      >
        <div
          className='agreement-detail-scroll max-h-[60vh] overflow-y-auto overscroll-y-contain pr-1'
          style={{ whiteSpace: 'pre-wrap', lineHeight: 1.8 }}
        >
          {agreementDetail || t('暂无协议详情')}
        </div>
      </Modal>
    </Card>
  );
};

export default MaterialLibrary;
