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
  Tag,
  TextArea,
  Tooltip,
  Typography,
  Upload,
} from '@douyinfe/semi-ui';
import {
  IconCopy,
  IconUpload,
  IconLink,
  IconDelete,
  IconEdit,
  IconRefresh,
  IconPlus,
  IconArrowLeft,
} from '@douyinfe/semi-icons';
import {
  Image as ImageTypeIcon,
  Video as VideoTypeIcon,
  Music as AudioTypeIcon,
  CheckCircle2,
  Clock,
  AlertCircle,
  ShieldCheck,
  ChevronRight,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { copy, showSuccess, showError, getMaterialConfig } from '../../helpers';
import {
  MaterialAssetType,
  MaterialStatus,
  MATERIAL_UPLOAD_ACCEPT,
} from '../../constants';
import {
  isDefaultPersonDescription,
  isDefaultPersonName,
  useRealPerson,
} from '../../hooks/realperson/useRealPerson';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { useSmoothUploadProgress } from '../distributor/useSmoothUploadProgress';

const { Title, Text } = Typography;

const UserRoundIcon = ({ size = 18, style }) => (
  <svg
    width={size}
    height={size}
    viewBox='0 0 24 24'
    fill='none'
    stroke='currentColor'
    strokeWidth='2'
    strokeLinecap='round'
    strokeLinejoin='round'
    style={style}
    aria-hidden
  >
    <path d='M20 21a8 8 0 0 0-16 0' />
    <circle cx='12' cy='7' r='4' />
  </svg>
);

/* ========================= 工具方法（纯展示映射） ========================= */

const getAssetTypeMeta = (assetType, t) => {
  if (assetType === MaterialAssetType.VIDEO) {
    return {
      icon: VideoTypeIcon,
      label: t('视频'),
      color: 'var(--semi-color-warning)',
    };
  }
  if (assetType === MaterialAssetType.AUDIO) {
    return { icon: AudioTypeIcon, label: t('音频'), color: 'var(--semi-color-tertiary)' };
  }
  return { icon: ImageTypeIcon, label: t('图片'), color: 'var(--semi-color-primary)' };
};

const getStatusMeta = (status, t) => {
  switch (status) {
    case MaterialStatus.ACTIVE:
      return {
        icon: CheckCircle2,
        color: 'var(--semi-color-success)',
        label: t('素材可用'),
      };
    case MaterialStatus.FAILED:
      return {
        icon: AlertCircle,
        color: 'var(--semi-color-danger)',
        label: t('创建/处理失败'),
      };
    case MaterialStatus.PENDING:
    default:
      return {
        icon: Clock,
        color: 'var(--semi-color-warning)',
        label: t('处理中'),
      };
  }
};

// 会话状态 -> 展示文案与颜色。
const getSessionStatusMeta = (status, t) => {
  switch (status) {
    case 'success':
      return { text: t('认证成功'), color: 'green' };
    case 'failed':
      return { text: t('人脸核验失败'), color: 'red' };
    case 'expired':
      return { text: t('会话已过期'), color: 'orange' };
    case 'pending':
    default:
      return { text: t('认证中…'), color: 'blue' };
  }
};

/* ========================= 页面组件 ========================= */

const RealPersonLibrary = ({ embedded = false }) => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();

  const {
    // 分组列表
    groups,
    groupsLoading,
    loadGroups,
    selectedGroup,
    openGroup,
    backToGroups,
    handleDeleteGroup,
    handleUpdateGroup,
    namePromptGroup,
    clearNamePrompt,
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
  } = useRealPerson();

  // 配置（用于判断功能是否启用）。
  const [config, setConfig] = useState({
    enabled: false,
    ready: false,
    max_image_size_mb: 10,
  });
  React.useEffect(() => {
    getMaterialConfig()
      .then((res) => {
        if (res?.success && res?.data) setConfig(res.data);
      })
      .catch(() => {});
  }, []);
  const maxSizeMB = config.max_image_size_mb || 10;

  // 本地 UI 状态。
  const [urlModalVisible, setUrlModalVisible] = useState(false);
  const [urlInput, setUrlInput] = useState('');
  const [urlName, setUrlName] = useState('');
  const [assetTypeFilter, setAssetTypeFilter] = useState('all'); // all / Image / Video
  const [selectedAssetIds, setSelectedAssetIds] = useState(new Set());
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [editTarget, setEditTarget] = useState(null); // 当前编辑的人物对象
  const [editForm, setEditForm] = useState({ group_name: '', description: '' });
  const [editSaving, setEditSaving] = useState(false);
  // 素材改名弹窗（虚拟/真人共用交互，此处仅真人素材）
  const [renameVisible, setRenameVisible] = useState(false);
  const [renameTarget, setRenameTarget] = useState(null);
  const [renameName, setRenameName] = useState('');
  const [renameSaving, setRenameSaving] = useState(false);
  // 认证成功后的人物命名引导
  const [namePromptVisible, setNamePromptVisible] = useState(false);
  const [namePromptValue, setNamePromptValue] = useState('');
  const [namePromptSaving, setNamePromptSaving] = useState(false);

  React.useEffect(() => {
    if (!namePromptGroup) {
      setNamePromptVisible(false);
      return;
    }
    setNamePromptValue(
      isDefaultPersonName(
        namePromptGroup.group_name,
        namePromptGroup.group_id,
      )
        ? ''
        : namePromptGroup.group_name || '',
    );
    setNamePromptVisible(true);
  }, [namePromptGroup]);

  const uploadDisabled =
    !agreed || !config.ready || uploading || !selectedGroup;
  const uploadDisplayPct = useSmoothUploadProgress(uploadProgress);

  // 按素材类型过滤（须在组件顶层调用 Hook，不可放在 renderGroupDetail 等条件渲染函数内）。
  const filteredAssets = useMemo(() => {
    if (!selectedGroup) return [];
    if (assetTypeFilter === 'all') return assets;
    return assets.filter((a) => a.asset_type === assetTypeFilter);
  }, [selectedGroup, assets, assetTypeFilter]);

  /* --------------------------- 交互事件 --------------------------- */

  const customRequest = async ({ file, onSuccess, onError }) => {
    const inst = file?.fileInstance || file;
    const ok = await handleUploadFile(inst);
    if (ok) {
      onSuccess && onSuccess({});
    } else {
      onError && onError({ message: 'upload failed' });
    }
  };

  const submitUrlUpload = async () => {
    const ok = await handleUploadByURL({ url: urlInput, name: urlName });
    if (ok) {
      setUrlModalVisible(false);
      setUrlInput('');
      setUrlName('');
    }
  };

  const handleCopyURI = async (asset) => {
    const ok = await copy(asset.asset_uri);
    if (ok) {
      showSuccess(t('已复制资源地址，可替换图片资源地址'));
    } else {
      showError(t('复制失败，请手动复制：') + asset.asset_uri);
    }
  };

  const handleBatchDeleteConfirm = async () => {
    const ids = Array.from(selectedAssetIds);
    if (ids.length === 0) return;
    const ok = await handleBatchDelete(ids);
    if (ok) setSelectedAssetIds(new Set());
  };

  const toggleSelectAsset = (assetId) => {
    const id = String(assetId);
    setSelectedAssetIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const handleOpenEdit = (group) => {
    setEditTarget(group);
    setEditForm({
      group_name: group?.group_name || '',
      description: group?.description || '',
    });
    setEditModalVisible(true);
  };

  const handleEditSubmit = async () => {
    const name = editForm.group_name.trim();
    if (!name) {
      showError(t('人物名称不能为空'));
      return;
    }
    setEditSaving(true);
    const ok = await handleUpdateGroup(editTarget, {
      group_name: name,
      description: editForm.description.trim(),
    });
    setEditSaving(false);
    if (ok) {
      setEditModalVisible(false);
      setEditTarget(null);
    }
  };

  const submitNamePrompt = async () => {
    if (!namePromptGroup) return;
    const name = namePromptValue.trim();
    if (!name) {
      showError(t('请输入人物称呼，例如「张三」'));
      return;
    }
    setNamePromptSaving(true);
    const ok = await handleUpdateGroup(namePromptGroup, {
      group_name: name,
      description: isDefaultPersonDescription(namePromptGroup.description)
        ? t('已核验人物的专属素材空间')
        : namePromptGroup.description || '',
    });
    setNamePromptSaving(false);
    if (ok) {
      setNamePromptVisible(false);
      clearNamePrompt();
    }
  };

  const skipNamePrompt = () => {
    setNamePromptVisible(false);
    clearNamePrompt();
  };

  const openRenameModal = (asset) => {
    setRenameTarget(asset);
    setRenameName(asset?.name || '');
    setRenameVisible(true);
  };

  const submitRename = async () => {
    if (!renameTarget) return;
    setRenameSaving(true);
    const ok = await handleRename(renameTarget, renameName);
    setRenameSaving(false);
    if (ok) {
      setRenameVisible(false);
      setRenameTarget(null);
      setRenameName('');
    }
  };

  /* --------------------------- 渲染：单张素材卡片 --------------------------- */

  const renderAssetCard = (asset) => {
    const typeMeta = getAssetTypeMeta(asset.asset_type, t);
    const statusMeta = getStatusMeta(asset.status, t);
    const TypeIcon = typeMeta.icon;
    const StatusIcon = statusMeta.icon;
    const isVideo = asset.asset_type === MaterialAssetType.VIDEO;
    const isAudio = asset.asset_type === MaterialAssetType.AUDIO;
    const isSelected = selectedAssetIds.has(String(asset.asset_id));

    return (
      <Card
        key={asset.asset_id}
        className='realperson-asset-card'
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
          {/* 类型徽标 */}
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

          {/* 选中勾选框 */}
          <div
            style={{
              position: 'absolute',
              top: 6,
              right: 6,
              zIndex: 2,
            }}
          >
            <Checkbox
              checked={isSelected}
              onChange={() => toggleSelectAsset(asset.asset_id)}
            />
          </div>

          {isVideo ? (
            <video
              src={asset.url}
              controls
              preload='metadata'
              style={{ width: '100%', height: '100%', objectFit: 'cover' }}
            />
          ) : isAudio ? (
            <div
              style={{
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                gap: 8,
                padding: 12,
                width: '100%',
              }}
            >
              <AudioTypeIcon size={36} style={{ color: 'var(--semi-color-tertiary)' }} />
              <audio src={asset.url} controls preload='metadata' style={{ width: '100%' }} />
            </div>
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
                style={{ maxWidth: 110, fontSize: 13 }}
              >
                {asset.name || t('未命名素材')}
              </Text>
            </Tooltip>
            <div style={{ display: 'flex', alignItems: 'center', gap: 2, flexShrink: 0 }}>
              <Tooltip content={t('改名')}>
                <Button
                  size='small'
                  theme='borderless'
                  type='tertiary'
                  icon={<IconEdit />}
                  onClick={() => openRenameModal(asset)}
                  aria-label={t('改名')}
                />
              </Tooltip>
              <Tooltip content={statusMeta.label}>
                <StatusIcon size={16} style={{ color: statusMeta.color }} />
              </Tooltip>
            </div>
          </div>

          {/* 操作区：展示资源地址 + 复制 + 删除 */}
          <div
            style={{
              display: 'flex',
              alignItems: 'flex-start',
              justifyContent: 'space-between',
              gap: 6,
            }}
          >
            <div
              style={{
                flex: 1,
                minWidth: 0,
                display: 'flex',
                alignItems: 'center',
                gap: 2,
              }}
            >
              <Tooltip content={asset.asset_uri}>
                <Text
                  size='small'
                  ellipsis={{ showTooltip: false }}
                  style={{ flex: 1, fontFamily: 'monospace', fontSize: 11 }}
                >
                  {asset.asset_uri}
                </Text>
              </Tooltip>
              <Tooltip content={t('复制地址')}>
                <Button
                  size='small'
                  theme='borderless'
                  type='tertiary'
                  icon={<IconCopy />}
                  onClick={() => handleCopyURI(asset)}
                  aria-label={t('复制地址')}
                />
              </Tooltip>
            </div>

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
                loading={deletingId === asset.asset_id}
                aria-label={t('删除')}
              />
            </Popconfirm>
          </div>
        </div>
      </Card>
    );
  };

  /* --------------------------- 渲染：认证模态框 --------------------------- */

  const renderSessionModal = () => {
    const statusMeta = getSessionStatusMeta(sessionStatus, t);
    const qrCode = session?.qr_code;
    const h5Link = session?.h5_link;
    return (
      <Modal
        title={t('核验新人物')}
        visible={sessionModalOpen}
        onCancel={closeSessionModal}
        footer={
          <Button theme='solid' onClick={closeSessionModal}>
            {sessionStatus === 'success' ? t('去上传素材') : t('关闭')}
          </Button>
        }
        width={isMobile ? 'calc(100vw - 32px)' : 480}
        closable
        maskClosable={false}
      >
        {/* 二维码 */}
        {qrCode && (
          <div
            style={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              gap: 12,
              marginBottom: 16,
            }}
          >
            <img
              src={qrCode}
              alt={t('认证二维码')}
              style={{ width: 220, height: 220, objectFit: 'contain' }}
            />
            <Tag color={statusMeta.color} size='large'>
              {statusMeta.text}
            </Tag>
            {sessionMessage && sessionStatus !== 'success' && (
              <Text type='tertiary' size='small'>
                {sessionMessage}
              </Text>
            )}
          </div>
        )}

        {/* H5 链接 */}
        {h5Link && (
          <div style={{ marginBottom: 16 }}>
            <Text type='secondary' size='small'>
              {t('或使用 H5 链接认证：')}
            </Text>
            <div style={{ display: 'flex', gap: 8, marginTop: 4 }}>
              <Input value={h5Link} readOnly style={{ flex: 1 }} />
              <Button
                icon={<IconCopy />}
                onClick={async () => {
                  const ok = await copy(h5Link);
                  if (ok) showSuccess(t('已复制链接'));
                  else showError(t('复制失败'));
                }}
              >
                {t('复制')}
              </Button>
            </div>
          </div>
        )}

        {/* 状态提示 */}
        <Banner
          type={
            sessionStatus === 'success'
              ? 'success'
              : sessionStatus === 'expired' || sessionStatus === 'failed'
                ? 'danger'
                : 'info'
          }
          closeIcon={null}
              description={
                sessionStatus === 'success'
                  ? t('核验成功，已为该人物创建专属素材空间，即将进入上传。')
                  : sessionStatus === 'expired'
                    ? t('认证会话已过期，请重新发起认证。')
                    : sessionStatus === 'failed'
                      ? t('人脸核验失败，请重新发起认证。')
                      : t(
                          '请用手机扫码完成实名与活体核验。成功后会新建一位「已核验人物」；仅换人才需要再做一次，同一人物不必重复认证。',
                        )
              }
          style={{ marginBottom: 12 }}
        />

        {/* 过期时重新认证 */}
        {(sessionStatus === 'expired' || sessionStatus === 'failed') && (
          <Button
            theme='solid'
            icon={<IconRefresh />}
            onClick={() => startSession()}
            loading={sessionCreating}
            block
          >
            {t('重新核验')}
          </Button>
        )}
      </Modal>
    );
  };

  const renderEditModal = () => {
    return (
      <Modal
        title={t('编辑人物')}
        visible={editModalVisible}
        onOk={handleEditSubmit}
        onCancel={() => {
          setEditModalVisible(false);
          setEditTarget(null);
        }}
        okText={t('保存')}
        cancelText={t('取消')}
        confirmLoading={editSaving}
        maskClosable={false}
      >
        <div style={{ marginBottom: 12 }}>
          <Text strong>{t('人物名称')}</Text>
          <Input
            value={editForm.group_name}
            onChange={(value) =>
              setEditForm((f) => ({ ...f, group_name: value }))
            }
            maxLength={64}
            showClear
            placeholder={t('例如：张三、主演 A')}
            style={{ marginTop: 4 }}
          />
        </div>
        <div>
          <Text strong>{t('备注（可选）')}</Text>
          <TextArea
            value={editForm.description}
            onChange={(value) =>
              setEditForm((f) => ({ ...f, description: value }))
            }
            maxLength={256}
            showClear
            placeholder={t('可填写用途说明，便于区分不同人物')}
            autosize
            style={{ marginTop: 4 }}
          />
        </div>
      </Modal>
    );
  };

  const renderNamePromptModal = () => {
    return (
      <Modal
        title={t('给这位人物起个称呼')}
        visible={namePromptVisible}
        onOk={submitNamePrompt}
        onCancel={skipNamePrompt}
        okText={t('保存并继续')}
        cancelText={t('稍后命名')}
        confirmLoading={namePromptSaving}
        maskClosable={false}
      >
        <Banner
          type='info'
          closeIcon={null}
          description={t(
            '该称呼仅用于本地管理，帮助区分不同已核验人物；同一人物的素材请放在同一空间下。',
          )}
          style={{ marginBottom: 12 }}
        />
        <Text strong>{t('人物名称')}</Text>
        <Input
          value={namePromptValue}
          onChange={setNamePromptValue}
          maxLength={64}
          showClear
          placeholder={t('例如：张三、主演 A')}
          style={{ marginTop: 4 }}
          onEnterPress={submitNamePrompt}
        />
      </Modal>
    );
  };

  /* --------------------------- 渲染：分组详情视图 --------------------------- */

  const renderGroupDetail = () => {
    const personLabel =
      selectedGroup?.group_name || t('未命名人物');
    return (
      <>
        {/* 返回 + 人物信息 */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 8,
            marginBottom: 12,
            flexWrap: 'wrap',
          }}
        >
          <Button
            theme='borderless'
            type='tertiary'
            icon={<IconArrowLeft />}
            onClick={backToGroups}
          >
            {t('返回人物列表')}
          </Button>
          <Tag
            color='green'
            size='large'
            prefixIcon={<ShieldCheck size={14} />}
          >
            {t('已核验人物')}
          </Tag>
          <Text strong>{personLabel}</Text>
          <Button
            size='small'
            theme='borderless'
            type='tertiary'
            icon={<IconEdit />}
            onClick={() => handleOpenEdit(selectedGroup)}
          >
            {t('改名')}
          </Button>
        </div>

        <Banner
          type='info'
          closeIcon={null}
          description={t(
            '当前空间仅存放该核验人物本人的素材，可继续上传更多图片/视频/音频。若要管理另一位真人，请返回列表后「认证新人物」，不要混传到此空间。',
          )}
          style={{ marginBottom: 12 }}
        />

        {/* 协议勾选 */}
        <div style={{ margin: '8px 0' }}>
          <Checkbox
            checked={agreed}
            onChange={(e) => setAgreed(e.target.checked)}
          >
            {t(
              '我已知晓所上传的真人素材均已获得本人授权，并同意真人素材合规要求。',
            )}
          </Checkbox>
        </div>

        {/* 上传入口 */}
        <Space style={{ marginBottom: 12 }}>
          <Upload
            action=''
            accept={MATERIAL_UPLOAD_ACCEPT}
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
            {t('支持图片/视频/音频（mp3/wav），单个文件不超过 {{n}}MB', { n: maxSizeMB })}
          </Text>
        </Space>

        {/* 类型过滤 + 批量操作 */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            marginBottom: 12,
          }}
        >
          <Tabs
            type='button'
            size='small'
            activeKey={assetTypeFilter}
            onChange={setAssetTypeFilter}
          >
            <Tabs.TabPane tab={t('全部')} itemKey='all' />
            <Tabs.TabPane tab={t('图片')} itemKey={MaterialAssetType.IMAGE} />
            <Tabs.TabPane tab={t('视频')} itemKey={MaterialAssetType.VIDEO} />
            <Tabs.TabPane tab={t('音频')} itemKey={MaterialAssetType.AUDIO} />
          </Tabs>
          <Space>
            {selectedAssetIds.size > 0 && (
              <Popconfirm
                title={t('确认批量删除选中的 {{n}} 个素材？', {
                  n: selectedAssetIds.size,
                })}
                okType='danger'
                okText={t('删除')}
                cancelText={t('取消')}
                onConfirm={handleBatchDeleteConfirm}
              >
                <Button type='danger' theme='borderless' icon={<IconDelete />}>
                  {t('批量删除')}
                </Button>
              </Popconfirm>
            )}
            <Tooltip content={t('刷新列表')}>
              <Button
                theme='borderless'
                type='tertiary'
                icon={<IconRefresh />}
                loading={assetsLoading}
                onClick={() =>
                  selectedGroup && loadAssets(selectedGroup.group_id)
                }
              />
            </Tooltip>
          </Space>
        </div>

        {uploadProgress != null && !urlModalVisible && (
          <Progress
            percent={uploadDisplayPct ?? uploadProgress}
            showInfo
            style={{ marginBottom: 16, maxWidth: 480 }}
          />
        )}

        {/* 素材列表 */}
        <Spin spinning={assetsLoading}>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
            {filteredAssets.map((asset) => renderAssetCard(asset))}
          </div>
          {filteredAssets.length === 0 && !assetsLoading && (
            <Empty
              style={{ marginTop: 24 }}
              description={t(
                '该人物暂无素材，点击「本地上传」或「链接上传」添加',
              )}
            />
          )}
        </Spin>

        {/* 在线链接上传弹窗 */}
        <Modal
          title={t('通过在线链接上传')}
          visible={urlModalVisible}
          onCancel={() => !uploading && setUrlModalVisible(false)}
          onOk={submitUrlUpload}
          okText={t('上传')}
          cancelText={t('取消')}
          confirmLoading={uploading}
          closable={!uploading}
          maskClosable={!uploading}
        >
          {uploadProgress != null && (
            <Progress
              percent={uploadDisplayPct ?? uploadProgress}
              showInfo
              style={{ marginBottom: 16 }}
            />
          )}
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
            description={t('仅支持图片、视频或音频（mp3/wav）的公开可访问链接（http/https）。')}
            style={{ marginTop: 12 }}
          />
        </Modal>
      </>
    );
  };

  /* --------------------------- 渲染：人物列表视图 --------------------------- */

  const renderGroupList = () => {
    return (
      <>
        <Banner
          type='info'
          closeIcon={null}
          description={t(
            '按「已核验人物」管理：一位人物 = 一次实名+活体核验 = 一个素材空间。同一人物可反复上传素材，无需重新认证；换人（另一张脸）请点「认证新人物」。',
          )}
          style={{ marginBottom: 16 }}
        />
        <div
          className='realperson-library-toolbar'
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            flexWrap: 'wrap',
            gap: 12,
            marginBottom: 16,
          }}
        >
          <Space wrap>
            <Tag color='green' size='large' prefixIcon={<ShieldCheck size={14} />}>
              {t('已核验人物')}
            </Tag>
            <Tooltip
              content={t(
                '仅在需要另一位真人（另一张脸）时使用。同一人物请进入其素材空间继续上传。',
              )}
            >
              <Button
                theme='solid'
                icon={<IconPlus />}
                loading={sessionCreating}
                onClick={() => startSession()}
              >
                {t('认证新人物')}
              </Button>
            </Tooltip>
          </Space>
          <Tooltip content={t('刷新列表')}>
            <Button
              theme='borderless'
              type='tertiary'
              icon={<IconRefresh />}
              loading={groupsLoading}
              onClick={loadGroups}
            />
          </Tooltip>
        </div>

        <Spin spinning={groupsLoading}>
          {groups.length === 0 && !groupsLoading ? (
            <Empty
              style={{ marginTop: 24 }}
              description={t('暂无已核验人物，请先完成真人实名与活体核验')}
            />
          ) : (
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
              {groups.map((group) => {
                const displayName =
                  group.group_name?.trim() || t('未命名人物');
                const showDefaultTip = isDefaultPersonName(
                  group.group_name,
                  group.group_id,
                );
                const assetCount =
                  typeof group.asset_count === 'number'
                    ? group.asset_count
                    : null;
                const descriptionVisible =
                  group.description &&
                  !isDefaultPersonDescription(group.description);
                return (
                  <Tooltip
                    key={group.group_id}
                    content={t('点击进入该人物的素材空间')}
                  >
                    <Card
                      style={{
                        width: 280,
                        cursor: 'pointer',
                        transition: 'box-shadow 0.2s ease, transform 0.2s ease',
                        overflow: 'hidden',
                      }}
                      bordered
                      hoverable
                      onClick={() => openGroup(group)}
                      bodyStyle={{ padding: 0 }}
                    >
                      <div
                        style={{
                          height: 120,
                          background: 'var(--semi-color-fill-1)',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          overflow: 'hidden',
                          position: 'relative',
                        }}
                      >
                        {group.cover_url ? (
                          <Image
                            src={group.cover_url}
                            width='100%'
                            height={120}
                            style={{ objectFit: 'cover' }}
                            alt={displayName}
                          />
                        ) : (
                          <div
                            style={{
                              display: 'flex',
                              flexDirection: 'column',
                              alignItems: 'center',
                              gap: 6,
                              color: 'var(--semi-color-tertiary)',
                            }}
                          >
                            <UserRoundIcon size={36} />
                            <Text type='tertiary' size='small'>
                              {assetCount === 0
                                ? t('已认证，尚未上传素材')
                                : t('暂无封面')}
                            </Text>
                          </div>
                        )}
                      </div>

                      <div style={{ padding: '14px 16px' }}>
                        <div
                          style={{
                            display: 'flex',
                            alignItems: 'flex-start',
                            justifyContent: 'space-between',
                            gap: 8,
                            marginBottom: 8,
                          }}
                        >
                          <div
                            style={{
                              display: 'flex',
                              alignItems: 'center',
                              gap: 8,
                              minWidth: 0,
                              flex: 1,
                            }}
                          >
                            <ShieldCheck
                              size={18}
                              style={{
                                color: 'var(--semi-color-success)',
                                flexShrink: 0,
                              }}
                            />
                            <Text
                              strong
                              ellipsis={{ showTooltip: true }}
                              style={{ flex: 1 }}
                            >
                              {displayName}
                            </Text>
                          </div>
                          <div
                            style={{
                              display: 'flex',
                              alignItems: 'center',
                              gap: 2,
                              flexShrink: 0,
                            }}
                          >
                            <Tooltip content={t('编辑人物')}>
                              <Button
                                size='small'
                                theme='borderless'
                                type='tertiary'
                                icon={<IconEdit />}
                                onClick={(e) => {
                                  e.stopPropagation();
                                  handleOpenEdit(group);
                                }}
                                aria-label={t('编辑人物')}
                              />
                            </Tooltip>
                            <Popconfirm
                              title={t('确认删除该已核验人物？')}
                              content={t(
                                '删除后将移除云端人物空间，本地素材记录保留。',
                              )}
                              okType='danger'
                              okText={t('删除')}
                              cancelText={t('取消')}
                              onConfirm={(e) => {
                                e?.stopPropagation?.();
                                handleDeleteGroup(group);
                              }}
                            >
                              <Button
                                size='small'
                                type='danger'
                                theme='borderless'
                                icon={<IconDelete />}
                                onClick={(e) => e.stopPropagation()}
                                aria-label={t('删除')}
                              />
                            </Popconfirm>
                          </div>
                        </div>

                        <div
                          style={{
                            display: 'flex',
                            alignItems: 'center',
                            gap: 8,
                            flexWrap: 'wrap',
                            marginBottom: 8,
                          }}
                        >
                          {assetCount != null && (
                            <Tag size='small' color='white'>
                              {t('{{n}} 个素材', { n: assetCount })}
                            </Tag>
                          )}
                          {showDefaultTip && (
                            <Tag size='small' color='orange'>
                              {t('建议改名')}
                            </Tag>
                          )}
                        </div>

                        {descriptionVisible ? (
                          <Text
                            size='small'
                            type='tertiary'
                            ellipsis={{ showTooltip: true }}
                            style={{ display: 'block', lineHeight: 1.5 }}
                          >
                            {group.description}
                          </Text>
                        ) : null}
                      </div>

                      <div
                        style={{
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'space-between',
                          padding: '8px 16px',
                          borderTop: '1px solid var(--semi-color-border)',
                          background: 'var(--semi-color-fill-0)',
                          color: 'var(--semi-color-primary)',
                        }}
                      >
                        <Text
                          size='small'
                          strong
                          style={{ color: 'var(--semi-color-primary)' }}
                        >
                          {t('进入素材空间')}
                        </Text>
                        <ChevronRight size={16} strokeWidth={2.25} aria-hidden />
                      </div>
                    </Card>
                  </Tooltip>
                );
              })}
            </div>
          )}
        </Spin>
      </>
    );
  };

  /* --------------------------- 渲染：主体 --------------------------- */

  // 核心内容（人物列表/详情 + 认证/命名模态框），嵌入与独立模式共用。
  const coreContent = (
    <>
      {selectedGroup ? renderGroupDetail() : renderGroupList()}

      {renderSessionModal()}

      {renderEditModal()}

      {renderNamePromptModal()}

      {/* 真人素材改名 */}
      <Modal
        title={t('素材改名')}
        visible={renameVisible}
        onCancel={() => !renameSaving && setRenameVisible(false)}
        onOk={submitRename}
        okText={t('保存')}
        cancelText={t('取消')}
        confirmLoading={renameSaving}
        closable={!renameSaving}
        maskClosable={!renameSaving}
      >
        <Text>{t('素材名称')}</Text>
        <Input
          value={renameName}
          onChange={setRenameName}
          maxLength={128}
          placeholder={t('请输入素材名称')}
          style={{ marginTop: 4 }}
        />
      </Modal>
    </>
  );

  if (embedded) {
    // 嵌入模式：外层 Card / 标题栏 / 配置 Banner 由父组件（MaterialLibrary）提供。
    return coreContent;
  }

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
          {t('已核验人物管理')}
        </Title>
        <Tag color='blue' size='small'>
          {t('真人人像')}
        </Tag>
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

      {coreContent}
    </Card>
  );
};

export default RealPersonLibrary;
