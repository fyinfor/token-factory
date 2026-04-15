/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import React, { useEffect, useState, useRef } from 'react';
import {
  Button,
  Col,
  Form,
  Row,
  Spin,
  Typography,
  Banner,
} from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import {
  compareObjects,
  API,
  showError,
  showSuccess,
  showWarning,
  toBoolean,
} from '../../../helpers';

const { Text } = Typography;

function isLikelyImageUrl(url, mimeHint) {
  if (mimeHint && mimeHint.startsWith('image/')) {
    return true;
  }
  return /\.(png|jpe?g|gif|webp|bmp|svg)(\?|$)/i.test(url || '');
}

export default function SettingsOss(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    'oss_setting.enabled': false,
    'oss_setting.endpoint': '',
    'oss_setting.bucket': '',
    'oss_setting.access_key_id': '',
    'oss_setting.access_key_secret': '',
    'oss_setting.public_base_url': '',
    'oss_setting.object_key_prefix': 'uploads/',
    'oss_setting.max_file_size_mb': 20,
  });
  const refForm = useRef();
  const [inputsRow, setInputsRow] = useState(inputs);

  const [testLoading, setTestLoading] = useState(false);
  const [testUrl, setTestUrl] = useState('');
  const [testMime, setTestMime] = useState('');
  const fileInputRef = useRef(null);

  function onSubmit() {
    const updateArray = compareObjects(inputs, inputsRow);
    if (!updateArray.length) return showWarning(t('你似乎并没有修改什么'));
    const requestQueue = updateArray.map((item) => {
      let value = '';
      if (typeof inputs[item.key] === 'boolean') {
        value = String(inputs[item.key]);
      } else if (typeof inputs[item.key] === 'number') {
        value = String(inputs[item.key]);
      } else {
        value = inputs[item.key];
      }
      return API.put('/api/option/', {
        key: item.key,
        value,
      });
    });
    setLoading(true);
    Promise.all(requestQueue)
      .then((res) => {
        if (res.includes(undefined)) return showError(t('部分保存失败，请重试'));
        showSuccess(t('保存成功'));
        props.refresh();
      })
      .catch(() => {
        showError(t('保存失败，请重试'));
      })
      .finally(() => {
        setLoading(false);
      });
  }

  async function uploadTestFile(file) {
    if (!file) {
      return;
    }
    setTestLoading(true);
    setTestUrl('');
    setTestMime('');
    try {
      const fd = new FormData();
      fd.append('file', file);
      const res = await API.post('/api/oss/upload', fd, {
        skipErrorHandler: true,
      });
      const { success, message, data } = res.data || {};
      if (!success) {
        showError(message || t('上传失败'));
        return;
      }
      const url = data?.url;
      if (!url) {
        showError(t('响应中无 url 字段'));
        return;
      }
      setTestUrl(url);
      setTestMime(file.type || '');
      showSuccess(t('测试上传成功'));
    } catch (e) {
      const msg =
        e?.response?.data?.message ||
        e?.message ||
        t('上传失败，请确认已保存 OSS 配置且已启用');
      showError(msg);
    } finally {
      setTestLoading(false);
    }
  }

  useEffect(() => {
    setInputs((prev) => {
      const next = { ...prev };
      for (const k of Object.keys(next)) {
        if (!k.startsWith('oss_setting.')) {
          continue;
        }
        if (props.options[k] === undefined) {
          continue;
        }
        const v = props.options[k];
        if (k === 'oss_setting.enabled') {
          next[k] = toBoolean(v);
        } else if (k === 'oss_setting.max_file_size_mb') {
          const n = parseInt(String(v), 10);
          next[k] = Number.isFinite(n) ? n : 20;
        } else {
          next[k] = v;
        }
      }
      setInputsRow(structuredClone(next));
      if (refForm.current) {
        refForm.current.setValues(next);
      }
      return next;
    });
  }, [props.options]);

  return (
    <Spin spinning={loading}>
      <Banner
        type='info'
        closeIcon={null}
        description={t(
          '配置后，已登录用户可调用 POST /api/oss/upload（multipart 字段 file）上传文件。请先保存配置，再在下方选择文件自动测试上传；返回的 url 需 Bucket/对象可读或 CDN 可访问才能预览。',
        )}
        style={{ marginBottom: 16 }}
      />
      <Form
        values={inputs}
        getFormApi={(formAPI) => (refForm.current = formAPI)}
        style={{ marginBottom: 15 }}
      >
        <Form.Section text={t('阿里云 OSS')}>
          <Row gutter={16}>
            <Col span={24}>
              <Form.Switch
                field={'oss_setting.enabled'}
                label={t('启用 OSS 上传')}
                checkedText='｜'
                uncheckedText='〇'
                onChange={(value) =>
                  setInputs({ ...inputs, 'oss_setting.enabled': value })
                }
              />
            </Col>
            <Col xs={24} sm={12}>
              <Form.Input
                field={'oss_setting.endpoint'}
                label={t('Endpoint')}
                placeholder='oss-cn-guangzhou.aliyuncs.com'
                onChange={(v) =>
                  setInputs({ ...inputs, 'oss_setting.endpoint': v })
                }
              />
              <Text type='tertiary' size='small'>
                {t('不含 https://，与阿里云控制台 Bucket 概览中的外网 Endpoint 一致')}
              </Text>
            </Col>
            <Col xs={24} sm={12}>
              <Form.Input
                field={'oss_setting.bucket'}
                label={t('Bucket 名称')}
                onChange={(v) =>
                  setInputs({ ...inputs, 'oss_setting.bucket': v })
                }
              />
            </Col>
            <Col xs={24} sm={12}>
              <Form.Input
                field={'oss_setting.access_key_id'}
                label={t('AccessKey ID')}
                onChange={(v) =>
                  setInputs({ ...inputs, 'oss_setting.access_key_id': v })
                }
              />
            </Col>
            <Col xs={24} sm={12}>
              <Form.Input
                field={'oss_setting.access_key_secret'}
                label={t('AccessKey Secret')}
                type='password'
                onChange={(v) =>
                  setInputs({ ...inputs, 'oss_setting.access_key_secret': v })
                }
              />
            </Col>
            <Col span={24}>
              <Form.Input
                field={'oss_setting.public_base_url'}
                label={t('对外访问基址（可选）')}
                placeholder='https://cdn.example.com'
                onChange={(v) =>
                  setInputs({ ...inputs, 'oss_setting.public_base_url': v })
                }
              />
              <Text type='tertiary' size='small' style={{ display: 'block' }}>
                {t('留空则使用 https://{bucket}.{endpoint}/ 形式；绑定 CDN 时填 CDN 根地址')}
              </Text>
            </Col>
            <Col xs={24} sm={12}>
              <Form.Input
                field={'oss_setting.object_key_prefix'}
                label={t('对象键前缀')}
                onChange={(v) =>
                  setInputs({ ...inputs, 'oss_setting.object_key_prefix': v })
                }
              />
            </Col>
            <Col xs={24} sm={12}>
              <Form.InputNumber
                field={'oss_setting.max_file_size_mb'}
                label={t('单文件大小上限（MB）')}
                min={1}
                max={1024}
                onChange={(v) =>
                  setInputs({ ...inputs, 'oss_setting.max_file_size_mb': v })
                }
              />
            </Col>
          </Row>
          <Row style={{ marginTop: 8 }} gutter={8}>
            <Col>
              <Button type='primary' onClick={onSubmit}>
                {t('保存 OSS 设置')}
              </Button>
            </Col>
          </Row>

          <Form.Section text={t('测试上传与预览')} style={{ marginTop: 24 }}>
            <Text type='tertiary' style={{ display: 'block', marginBottom: 8 }}>
              {t(
                '使用当前已保存到服务端的 OSS 配置；选择文件后将自动上传并显示访问地址（未保存的修改不会参与上传）。',
              )}
            </Text>
            <Row type='flex' align='middle' gutter={12} style={{ flexWrap: 'wrap' }}>
              <Col>
                <input
                  ref={fileInputRef}
                  type='file'
                  style={{ display: 'none' }}
                  disabled={testLoading}
                  onChange={(e) => {
                    const file = e.target.files?.[0];
                    if (file) {
                      uploadTestFile(file);
                    }
                    e.target.value = '';
                  }}
                />
                <Button
                  loading={testLoading}
                  disabled={testLoading}
                  onClick={() => fileInputRef.current?.click()}
                >
                  {testLoading ? t('上传中…') : t('选择测试文件')}
                </Button>
              </Col>
              {testLoading ? (
                <Col>
                  <Text type='tertiary' size='small'>
                    {t('正在上传到 OSS…')}
                  </Text>
                </Col>
              ) : null}
            </Row>
            {testUrl ? (
              <div style={{ marginTop: 16 }}>
                <Text strong>{t('文件访问 URL')}</Text>
                <div
                  style={{
                    wordBreak: 'break-all',
                    marginTop: 4,
                    marginBottom: 12,
                    fontSize: 13,
                  }}
                >
                  <a href={testUrl} target='_blank' rel='noopener noreferrer'>
                    {testUrl}
                  </a>
                </div>
                {isLikelyImageUrl(testUrl, testMime) ? (
                  <div>
                    <Text strong style={{ display: 'block', marginBottom: 8 }}>
                      {t('预览')}
                    </Text>
                    <img
                      src={testUrl}
                      alt='oss-test'
                      style={{
                        maxWidth: '100%',
                        maxHeight: 280,
                        objectFit: 'contain',
                        border: '1px solid var(--semi-color-border)',
                        borderRadius: 4,
                      }}
                    />
                  </div>
                ) : (
                  <Text type='tertiary' size='small'>
                    {t('非图片类型，请通过上方链接在新标签页中打开验证。')}
                  </Text>
                )}
              </div>
            ) : null}
          </Form.Section>
        </Form.Section>
      </Form>
    </Spin>
  );
}
