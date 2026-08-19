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

import React, { useCallback, useEffect, useState } from 'react';
import {
  Button,
  Card,
  Input,
  Space,
  Spin,
  Switch,
  Typography,
  Upload,
} from '@douyinfe/semi-ui';
import { ExternalLink, FileCode2, Upload as UploadIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../../helpers';

export const COMPUTE_PAGE_STATUS_CHANGED_EVENT = 'compute-page-status-changed';

const { Text, Title } = Typography;

const EMPTY_CONFIG = {
  enabled: false,
  allow_javascript: false,
  allow_popups: false,
  content_url: '',
  redirect_to_url: false,
  has_html: false,
  has_url: false,
  has_content: false,
  file_name: '',
  updated_at: 0,
};

function isHTMLFile(file) {
  const name = String(file?.name || '').toLowerCase();
  return name.endsWith('.html') || name.endsWith('.htm');
}

export default function SettingsComputePage() {
  const { t } = useTranslation();
  const [config, setConfig] = useState(EMPTY_CONFIG);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [contentURL, setContentURL] = useState('');

  const loadConfig = useCallback(async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/compute-page/admin/');
      const { success, message, data } = res.data || {};
      if (!success) {
        showError(message || t('加载算力页面配置失败'));
        return;
      }
      const nextConfig = { ...EMPTY_CONFIG, ...data };
      setConfig(nextConfig);
      setContentURL(nextConfig.content_url);
    } catch (error) {
      showError(error?.response?.data?.message || t('加载算力页面配置失败'));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    loadConfig();
  }, [loadConfig]);

  const updateEnabled = async (enabled) => {
    setSaving(true);
    try {
      const res = await API.put('/api/compute-page/admin/enabled', {
        enabled,
      });
      const { success, message, data } = res.data || {};
      if (!success) {
        showError(message || t('保存失败，请重试'));
        return;
      }
      setConfig({ ...EMPTY_CONFIG, ...data });
      window.dispatchEvent(new Event(COMPUTE_PAGE_STATUS_CHANGED_EVENT));
      showSuccess(enabled ? t('算力页面已开启') : t('算力页面已关闭'));
    } catch (error) {
      showError(error?.response?.data?.message || t('保存失败，请重试'));
    } finally {
      setSaving(false);
    }
  };

  const updateJavaScript = async (allowJavaScript) => {
    setSaving(true);
    try {
      const res = await API.put('/api/compute-page/admin/javascript', {
        allow_javascript: allowJavaScript,
      });
      const { success, message, data } = res.data || {};
      if (!success) {
        showError(message || t('保存失败，请重试'));
        return;
      }
      setConfig({ ...EMPTY_CONFIG, ...data });
      showSuccess(t('保存成功'));
    } catch (error) {
      showError(error?.response?.data?.message || t('保存失败，请重试'));
    } finally {
      setSaving(false);
    }
  };

  const updatePopups = async (allowPopups) => {
    setSaving(true);
    try {
      const res = await API.put('/api/compute-page/admin/popups', {
        allow_popups: allowPopups,
      });
      const { success, message, data } = res.data || {};
      if (!success) {
        showError(message || t('保存失败，请重试'));
        return;
      }
      setConfig({ ...EMPTY_CONFIG, ...data });
      showSuccess(t('保存成功'));
    } catch (error) {
      showError(error?.response?.data?.message || t('保存失败，请重试'));
    } finally {
      setSaving(false);
    }
  };

  const updateContentURL = async () => {
    setSaving(true);
    try {
      const res = await API.put('/api/compute-page/admin/url', {
        content_url: contentURL,
      });
      const { success, message, data } = res.data || {};
      if (!success) {
        showError(message || t('保存失败，请重试'));
        return;
      }
      const nextConfig = { ...EMPTY_CONFIG, ...data };
      setConfig(nextConfig);
      setContentURL(nextConfig.content_url);
      window.dispatchEvent(new Event(COMPUTE_PAGE_STATUS_CHANGED_EVENT));
      showSuccess(t('保存成功'));
    } catch (error) {
      showError(error?.response?.data?.message || t('保存失败，请重试'));
    } finally {
      setSaving(false);
    }
  };

  const updateRedirectToURL = async (redirectToURL) => {
    setSaving(true);
    try {
      const res = await API.put('/api/compute-page/admin/redirect', {
        redirect_to_url: redirectToURL,
      });
      const { success, message, data } = res.data || {};
      if (!success) {
        showError(message || t('保存失败，请重试'));
        return;
      }
      setConfig({ ...EMPTY_CONFIG, ...data });
      window.dispatchEvent(new Event(COMPUTE_PAGE_STATUS_CHANGED_EVENT));
      showSuccess(t('保存成功'));
    } catch (error) {
      showError(error?.response?.data?.message || t('保存失败，请重试'));
    } finally {
      setSaving(false);
    }
  };

  const uploadHTML = async ({ file, onSuccess, onError, onProgress }) => {
    const fileInstance = file?.fileInstance || file;
    if (!isHTMLFile(fileInstance)) {
      const error = new Error('invalid html');
      showError(t('仅支持 HTML 文件'));
      onError?.(error);
      return;
    }
    if (fileInstance.size > 20 * 1024 * 1024) {
      const error = new Error('html too large');
      showError(t('HTML 文件不能超过 20 MB'));
      onError?.(error);
      return;
    }

    const form = new FormData();
    form.append('file', fileInstance);
    setUploading(true);
    try {
      const res = await API.post('/api/compute-page/admin/content', form, {
        skipErrorHandler: true,
        onUploadProgress: (event) => {
          onProgress?.({
            loaded: event.loaded,
            total: event.total || event.loaded || 1,
          });
        },
      });
      const { success, message, data } = res.data || {};
      if (!success) {
        const error = new Error(message || 'upload failed');
        showError(message || t('上传失败'));
        onError?.(error);
        return;
      }
      setConfig({ ...EMPTY_CONFIG, ...data });
      onSuccess?.(data);
      showSuccess(t('HTML 文件已上传'));
    } catch (error) {
      showError(error?.response?.data?.message || t('上传失败'));
      onError?.(error);
    } finally {
      setUploading(false);
    }
  };

  return (
    <Card style={{ marginTop: 16 }}>
      <Spin spinning={loading}>
        <div className='flex flex-col gap-5'>
          <div className='flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between'>
            <div>
              <Title heading={5} style={{ margin: 0 }}>
                {t('算力页面')}
              </Title>
              <Text type='tertiary'>
                {t('开启后将在顶部导航首页后显示算力入口')}
              </Text>
            </div>
            <div className='flex items-center gap-3'>
              <Text>{config.enabled ? t('已开启') : t('已关闭')}</Text>
              <Switch
                checked={config.enabled}
                disabled={loading || saving || !config.has_content}
                loading={saving}
                checkedText='｜'
                uncheckedText='〇'
                onChange={updateEnabled}
              />
            </div>
          </div>

          <div className='rounded-md border border-solid border-semi-color-border bg-semi-color-fill-0 p-4'>
            <div className='flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between'>
              <div className='flex min-w-0 flex-1 flex-col gap-1'>
                <Text strong>{t('网页地址')}</Text>
                <Text type='tertiary' size='small'>
                  {t(
                    '填写 HTTP 或 HTTPS 网址后优先显示该网站；清空后将使用已上传的 HTML 文件',
                  )}
                </Text>
              </div>
              <div className='flex w-full shrink-0 gap-2 sm:w-[440px]'>
                <Input
                  value={contentURL}
                  placeholder='https://example.com'
                  showClear
                  disabled={loading || saving || uploading}
                  onChange={setContentURL}
                  onEnterPress={updateContentURL}
                />
                <Button
                  theme='solid'
                  loading={saving}
                  disabled={loading || saving || uploading}
                  onClick={updateContentURL}
                >
                  {t('保存网址')}
                </Button>
              </div>
            </div>
          </div>

          <div className='rounded-md border border-solid border-semi-color-border bg-semi-color-fill-0 p-4'>
            <div className='flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between'>
              <div className='flex min-w-0 flex-col gap-1'>
                <Text strong>{t('点击算力入口跳转网址')}</Text>
                <Text type='tertiary' size='small'>
                  {t(
                    '开启后，点击顶部算力入口会在当前页面直接跳转到填写的网址',
                  )}
                </Text>
              </div>
              <div className='flex shrink-0 items-center gap-3'>
                <Text>
                  {config.redirect_to_url ? t('已开启') : t('已关闭')}
                </Text>
                <Switch
                  checked={config.redirect_to_url}
                  disabled={loading || saving || !config.has_url}
                  loading={saving}
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={updateRedirectToURL}
                />
              </div>
            </div>
          </div>

          <div className='rounded-md border border-solid border-semi-color-border bg-semi-color-fill-0 p-4'>
            <div className='flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between'>
              <div className='flex min-w-0 items-center gap-3'>
                <FileCode2 size={22} aria-hidden='true' />
                <div className='min-w-0'>
                  <Text strong className='!block truncate'>
                    {config.file_name || t('尚未上传 HTML 文件')}
                  </Text>
                  <Text type='tertiary' size='small'>
                    {t('请上传 UTF-8 编码的自包含 HTML 文件，最大 20 MB')}
                  </Text>
                </div>
              </div>
              <Space wrap>
                <Upload
                  action=''
                  accept='.html,.htm,text/html'
                  customRequest={uploadHTML}
                  showUploadList={false}
                  disabled={uploading || saving}
                >
                  <Button
                    icon={<UploadIcon size={16} />}
                    loading={uploading}
                    disabled={uploading || saving}
                  >
                    {config.has_html ? t('替换 HTML') : t('上传 HTML')}
                  </Button>
                </Upload>
                {config.has_content ? (
                  <Button
                    icon={<ExternalLink size={16} />}
                    disabled={!config.enabled}
                    onClick={() =>
                      window.open('/compute', '_blank', 'noopener,noreferrer')
                    }
                  >
                    {t('预览页面')}
                  </Button>
                ) : null}
              </Space>
            </div>
          </div>

          <div className='rounded-md border border-solid border-semi-color-border bg-semi-color-fill-0 p-4'>
            <div className='flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between'>
              <div className='flex min-w-0 flex-col gap-1'>
                <Text strong>{t('允许新窗口跳转')}</Text>
                <Text type='warning' size='small'>
                  {t(
                    '开启后，页面可通过链接或脚本打开新窗口，目标页面不会继承算力页沙箱限制',
                  )}
                </Text>
              </div>
              <div className='flex shrink-0 items-center gap-3'>
                <Text>{config.allow_popups ? t('已开启') : t('已关闭')}</Text>
                <Switch
                  checked={config.allow_popups}
                  disabled={loading || saving}
                  loading={saving}
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={updatePopups}
                />
              </div>
            </div>
          </div>

          <div className='rounded-md border border-solid border-semi-color-border bg-semi-color-fill-0 p-4'>
            <div className='flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between'>
              <div className='flex min-w-0 flex-col gap-1'>
                <Text strong>{t('允许执行 JavaScript')}</Text>
                <Text type='warning' size='small'>
                  {t(
                    '开启后，上传的 HTML 可执行脚本、访问当前站点上下文并发起网络请求，请仅上传完全可信的内容',
                  )}
                </Text>
              </div>
              <div className='flex shrink-0 items-center gap-3'>
                <Text>
                  {config.allow_javascript ? t('已开启') : t('已关闭')}
                </Text>
                <Switch
                  checked={config.allow_javascript}
                  disabled={loading || saving}
                  loading={saving}
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={updateJavaScript}
                />
              </div>
            </div>
          </div>

          {!config.has_content ? (
            <Text type='warning'>
              {t('填写网址或上传 HTML 文件后才能开启算力页面')}
            </Text>
          ) : null}
        </div>
      </Spin>
    </Card>
  );
}
