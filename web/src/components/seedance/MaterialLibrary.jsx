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

import React, { useEffect, useMemo, useState } from 'react';
import {
  Banner,
  Button,
  Card,
  Checkbox,
  Empty,
  Image,
  Modal,
  Spin,
  Tabs,
  Tag,
  Tooltip,
  Typography,
  Upload,
} from '@douyinfe/semi-ui';
import { IconCopy, IconUpload, IconHelpCircle } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, copy, showError, showSuccess, showWarning } from '../../helpers';

const { Title, Text } = Typography;

const MaterialLibrary = () => {
  const { t, i18n } = useTranslation();
  const isEn = (i18n.language || '').toLowerCase().startsWith('en');

  const [config, setConfig] = useState({
    enabled: false,
    ready: false,
    max_image_size_mb: 10,
    agreement_zh: '',
    agreement_en: '',
    agreement_detail_zh: '',
    agreement_detail_en: '',
  });
  const [assets, setAssets] = useState([]);
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [agreed, setAgreed] = useState(false);
  const [detailVisible, setDetailVisible] = useState(false);

  const maxSizeMB = config.max_image_size_mb || 10;

  const loadConfig = async () => {
    try {
      const res = await API.get('/api/material/config');
      const { success, data } = res.data;
      if (success && data) setConfig(data);
    } catch (e) {
      // 静默：配置加载失败时使用默认值
    }
  };

  const loadAssets = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/material/assets', {
        params: { p: 1, size: 100 },
      });
      const { success, message, data } = res.data;
      if (success) {
        setAssets(data?.items || []);
      } else {
        showError(message);
      }
    } catch (e) {
      showError(t('加载素材列表失败'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadConfig();
    loadAssets();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

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

  const beforeUpload = ({ file }) => {
    const inst = file?.fileInstance || file;
    if (inst && inst.size > maxSizeMB * 1024 * 1024) {
      showError(t('图片超过大小限制（最大 {{n}}MB）', { n: maxSizeMB }));
      return false;
    }
    return true;
  };

  const customRequest = async ({ file, onSuccess, onError }) => {
    if (!agreed) {
      showWarning(t('请先阅读并勾选同意虚拟人像合规协议'));
      onError && onError({ message: 'not agreed' });
      return;
    }
    const inst = file?.fileInstance || file;
    if (inst && inst.size > maxSizeMB * 1024 * 1024) {
      showError(t('图片超过大小限制（最大 {{n}}MB）', { n: maxSizeMB }));
      onError && onError({ message: 'too large' });
      return;
    }
    const fd = new FormData();
    fd.append('file', inst);
    fd.append('agreed', 'true');
    setUploading(true);
    try {
      const res = await API.post('/api/material/upload', fd, {
        skipErrorHandler: true,
      });
      const { success, message, data } = res.data || {};
      if (!success) {
        showError(message || t('上传失败'));
        onError && onError({ message });
        return;
      }
      showSuccess(t('上传成功'));
      onSuccess && onSuccess(data);
      await loadAssets();
    } catch (e) {
      showError(t('上传失败，请重试'));
      onError && onError(e);
    } finally {
      setUploading(false);
    }
  };

  const handleCopy = async (asset) => {
    const ok = await copy(asset.asset_uri);
    if (ok) {
      showSuccess(t('已复制资源地址，可替换图片资源地址'));
    } else {
      showError(t('复制失败，请手动复制：') + asset.asset_uri);
    }
  };

  const renderStatusTag = (status) => {
    if (status === 'Active') return <Tag color='green'>{t('可用')}</Tag>;
    if (status === 'Failed') return <Tag color='red'>{t('失败')}</Tag>;
    return <Tag color='amber'>{t('处理中')}</Tag>;
  };

  const uploadDisabled = !agreed || !config.ready || uploading;

  return (
    <Card style={{ minHeight: '70vh' }}>
      {/* 标题 */}
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
        <Text type='tertiary'>{t('共 {{n}} 张素材', { n: assets.length })}</Text>
      </div>

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

          <Spin spinning={loading}>
            <div
              style={{
                display: 'flex',
                flexWrap: 'wrap',
                gap: 16,
              }}
            >
              {/* 上传卡片 */}
              <div
                style={{
                  width: 160,
                  height: 200,
                  border: '1px dashed var(--semi-color-border)',
                  borderRadius: 8,
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  justifyContent: 'center',
                  gap: 12,
                  background: 'var(--semi-color-fill-0)',
                }}
              >
                <Upload
                  action=''
                  accept='image/*'
                  showUploadList={false}
                  disabled={uploadDisabled}
                  beforeUpload={beforeUpload}
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
                <Text type='tertiary' size='small'>
                  {t('请上传<{{n}}M的图片', { n: maxSizeMB })}
                </Text>
              </div>

              {/* 素材列表 */}
              {assets.map((asset) => (
                <div
                  key={asset.id}
                  style={{
                    width: 160,
                    height: 200,
                    border: '1px solid var(--semi-color-border)',
                    borderRadius: 8,
                    overflow: 'hidden',
                    position: 'relative',
                    display: 'flex',
                    flexDirection: 'column',
                  }}
                >
                  <div
                    style={{
                      flex: 1,
                      background: 'var(--semi-color-fill-1)',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      overflow: 'hidden',
                    }}
                  >
                    <Image
                      src={asset.url}
                      width='100%'
                      height={150}
                      style={{ objectFit: 'cover' }}
                      alt={asset.name}
                    />
                  </div>
                  <div
                    style={{
                      padding: '6px 8px',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'space-between',
                      gap: 4,
                    }}
                  >
                    {renderStatusTag(asset.status)}
                    <Tooltip content={asset.asset_uri}>
                      <Button
                        size='small'
                        icon={<IconCopy />}
                        onClick={() => handleCopy(asset)}
                      >
                        {t('复制地址')}
                      </Button>
                    </Tooltip>
                  </div>
                </div>
              ))}
            </div>

            {assets.length === 0 && !loading && (
              <Empty
                style={{ marginTop: 24 }}
                description={t('暂无素材，点击「本地上传」添加虚拟人像')}
              />
            )}
          </Spin>
        </Tabs.TabPane>
      </Tabs>

      {/* 协议详情弹窗 */}
      <Modal
        title={t('虚拟人像合规承诺函')}
        visible={detailVisible}
        onCancel={() => setDetailVisible(false)}
        footer={
          <Button theme='solid' onClick={() => setDetailVisible(false)}>
            {t('我已知晓')}
          </Button>
        }
      >
        <div style={{ whiteSpace: 'pre-wrap', lineHeight: 1.8 }}>
          {agreementDetail || t('暂无协议详情')}
        </div>
      </Modal>
    </Card>
  );
};

export default MaterialLibrary;
