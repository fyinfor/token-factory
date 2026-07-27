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
  Button,
  Card,
  Col,
  ImagePreview,
  Input,
  Popconfirm,
  Row,
  Space,
  Switch,
  Upload,
} from '@douyinfe/semi-ui';
import {
  IconArrowDown,
  IconArrowUp,
  IconDelete,
  IconPlus,
  IconUpload,
} from '@douyinfe/semi-icons';
import Text from '@douyinfe/semi-ui/lib/es/typography/text';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';

const ENABLED_KEY = 'HomeFooterCertificatesEnabled';
const CERTIFICATES_KEY = 'HomeFooterCertificates';

const emptyCertificate = () => ({
  title: '',
  image_url: '',
  link_url: '',
});

const parseCertificates = (raw) => {
  if (!raw || typeof raw !== 'string') {
    return [];
  }
  try {
    const value = JSON.parse(raw);
    if (!Array.isArray(value)) {
      return [];
    }
    return value.map((item) => ({
      title: String(item?.title || '').trim(),
      image_url: String(item?.image_url || '').trim(),
      link_url: String(item?.link_url || '').trim(),
    }));
  } catch {
    return [];
  }
};

const stringifyCertificates = (certificates) => {
  const cleaned = certificates
    .map((item) => ({
      title: String(item?.title || '').trim(),
      image_url: String(item?.image_url || '').trim(),
      link_url: String(item?.link_url || '').trim(),
    }))
    .filter((item) => item.image_url);
  return JSON.stringify(cleaned);
};

export default function HomeFooterCertificatesSetting() {
  const { t } = useTranslation();
  const [enabled, setEnabled] = useState(false);
  const [certificates, setCertificates] = useState([]);
  const [loading, setLoading] = useState(false);
  const [previewUrl, setPreviewUrl] = useState('');
  const countText = useMemo(
    () => t('{{count}} 张', { count: certificates.length }),
    [certificates.length, t],
  );

  const loadOptions = async () => {
    try {
      const res = await API.get('/api/option/');
      const { success, message, data } = res.data || {};
      if (!success) {
        showError(message || t('加载设置失败'));
        return;
      }
      const optionMap = {};
      (Array.isArray(data) ? data : []).forEach((item) => {
        optionMap[item.key] = item.value;
      });
      setEnabled(optionMap[ENABLED_KEY] === 'true');
      setCertificates(parseCertificates(optionMap[CERTIFICATES_KEY] || '[]'));
    } catch (error) {
      showError(error?.message || t('加载设置失败'));
    }
  };

  useEffect(() => {
    loadOptions();
  }, []);

  const updateCertificate = (index, key, value) => {
    setCertificates((items) =>
      items.map((item, i) => (i === index ? { ...item, [key]: value } : item)),
    );
  };

  const addCertificate = () => {
    setCertificates((items) => [...items, emptyCertificate()]);
  };

  const removeCertificate = (index) => {
    setCertificates((items) => items.filter((_, i) => i !== index));
  };

  const moveCertificate = (index, offset) => {
    setCertificates((items) => {
      const nextIndex = index + offset;
      if (nextIndex < 0 || nextIndex >= items.length) {
        return items;
      }
      const next = [...items];
      const [item] = next.splice(index, 1);
      next.splice(nextIndex, 0, item);
      return next;
    });
  };

  const uploadCertificate =
    (index) =>
    async ({ file, onSuccess, onError }) => {
      const inst = file?.fileInstance || file;
      if (!inst) {
        onError?.(new Error('no file'));
        return;
      }

      try {
        setLoading(true);
        const fd = new FormData();
        fd.append('file', inst);
        fd.append('purpose', 'homepage');
        const res = await API.post('/api/oss/upload', fd, {
          skipErrorHandler: true,
        });
        const { success, message, data } = res.data || {};
        const url = data?.url;
        if (!success || !url) {
          throw new Error(message || t('上传失败'));
        }
        updateCertificate(index, 'image_url', url);
        onSuccess?.(data);
        showSuccess(t('证书图片上传成功，请点击保存设置'));
      } catch (error) {
        onError?.(error);
        showError(
          error?.response?.data?.message || error?.message || t('上传失败'),
        );
      } finally {
        setLoading(false);
      }
    };

  const save = async () => {
    try {
      setLoading(true);
      const certificatesValue = stringifyCertificates(certificates);
      const results = await Promise.all([
        API.put('/api/option/', {
          key: ENABLED_KEY,
          value: String(enabled),
        }),
        API.put('/api/option/', {
          key: CERTIFICATES_KEY,
          value: certificatesValue,
        }),
      ]);
      const failed = results.find((res) => !res.data?.success);
      if (failed) {
        throw new Error(failed.data?.message || t('保存失败'));
      }
      setCertificates(parseCertificates(certificatesValue));
      showSuccess(t('首页底部证书展示设置已保存'));
    } catch (error) {
      showError(error?.message || t('保存失败'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card
      title={t('首页底部证书展示')}
      style={{ marginTop: 16, marginBottom: 16 }}
    >
      <Space vertical align='start' spacing='medium' style={{ width: '100%' }}>
        <Space wrap style={{ width: '100%', justifyContent: 'space-between' }}>
          <Space align='center' wrap>
            <Text>{t('启用')}</Text>
            <Switch checked={enabled} onChange={setEnabled} />
            <Text type='tertiary' size='small'>
              {t('当前 {{countText}}，底部以小图展示', { countText })}
            </Text>
          </Space>
          <Space align='center' wrap>
            <Button icon={<IconPlus />} theme='light' onClick={addCertificate}>
              {t('添加证书')}
            </Button>
            <Button type='primary' loading={loading} onClick={save}>
              {t('保存设置')}
            </Button>
          </Space>
        </Space>

        <Text type='tertiary' size='small'>
          {t(
            '每张证书包含标题、图片和可选跳转链接，点击首页底部图片会放大预览。',
          )}
        </Text>

        {certificates.length === 0 ? (
          <Button icon={<IconPlus />} theme='light' onClick={addCertificate}>
            {t('添加第一张证书')}
          </Button>
        ) : (
          <Space
            vertical
            align='start'
            spacing='medium'
            style={{ width: '100%' }}
          >
            {certificates.map((certificate, index) => (
              <div
                key={index}
                style={{
                  width: '100%',
                  border: '1px solid var(--semi-color-border)',
                  borderRadius: 8,
                  padding: 16,
                  background: 'var(--semi-color-bg-1)',
                }}
              >
                <Space
                  wrap
                  style={{
                    width: '100%',
                    justifyContent: 'space-between',
                    marginBottom: 12,
                  }}
                >
                  <Text strong>{`${t('证书')} #${index + 1}`}</Text>
                  <Space spacing='tight'>
                    <Button
                      icon={<IconArrowUp />}
                      disabled={index === 0}
                      theme='borderless'
                      onClick={() => moveCertificate(index, -1)}
                    />
                    <Button
                      icon={<IconArrowDown />}
                      disabled={index === certificates.length - 1}
                      theme='borderless'
                      onClick={() => moveCertificate(index, 1)}
                    />
                    <Popconfirm
                      title={t('确定删除这张证书吗？')}
                      position='left'
                      onConfirm={() => removeCertificate(index)}
                    >
                      <Button
                        icon={<IconDelete />}
                        type='danger'
                        theme='borderless'
                      />
                    </Popconfirm>
                  </Space>
                </Space>

                <Row gutter={16}>
                  <Col xs={24} md={6}>
                    {certificate.image_url ? (
                      <button
                        type='button'
                        onClick={() => setPreviewUrl(certificate.image_url)}
                        style={{
                          width: '100%',
                          height: 96,
                          padding: 0,
                          cursor: 'zoom-in',
                          border: '1px solid var(--semi-color-border)',
                          borderRadius: 8,
                          background: 'var(--semi-color-fill-0)',
                        }}
                      >
                        <img
                          src={certificate.image_url}
                          alt={t('证书预览')}
                          style={{
                            width: '100%',
                            height: '100%',
                            objectFit: 'contain',
                          }}
                        />
                      </button>
                    ) : (
                      <div
                        style={{
                          width: '100%',
                          height: 96,
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          borderRadius: 8,
                          border: '1px dashed var(--semi-color-border)',
                          color: 'var(--semi-color-text-2)',
                          background: 'var(--semi-color-fill-0)',
                        }}
                      >
                        {t('暂无图片')}
                      </div>
                    )}
                    <Upload
                      action=''
                      accept='image/*'
                      showUploadList={false}
                      customRequest={uploadCertificate(index)}
                    >
                      <Button
                        icon={<IconUpload />}
                        loading={loading}
                        style={{ marginTop: 8, width: '100%' }}
                      >
                        {t('上传图片')}
                      </Button>
                    </Upload>
                  </Col>
                  <Col xs={24} md={18}>
                    <Space
                      vertical
                      align='start'
                      spacing='tight'
                      style={{ width: '100%' }}
                    >
                      <Input
                        value={certificate.title}
                        placeholder={t('证书标题（如 EDI 证书、ICP 备案证书）')}
                        onChange={(value) =>
                          updateCertificate(index, 'title', value)
                        }
                      />
                      <Input
                        value={certificate.image_url}
                        placeholder={t('证书图片地址')}
                        onChange={(value) =>
                          updateCertificate(index, 'image_url', value)
                        }
                      />
                      <Input
                        value={certificate.link_url}
                        placeholder={t('跳转链接（可留空）')}
                        onChange={(value) =>
                          updateCertificate(index, 'link_url', value)
                        }
                      />
                    </Space>
                  </Col>
                </Row>
              </div>
            ))}
          </Space>
        )}
      </Space>

      <ImagePreview
        src={previewUrl || ''}
        visible={Boolean(previewUrl)}
        onVisibleChange={(visible) => {
          if (!visible) {
            setPreviewUrl('');
          }
        }}
      />
    </Card>
  );
}
