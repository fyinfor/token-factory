/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import React, { useEffect, useMemo, useRef, useState } from 'react';
import {
  Banner,
  Button,
  Col,
  Form,
  Row,
  Select,
  Spin,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import {
  API,
  compareObjects,
  showError,
  showSuccess,
  showWarning,
  toBoolean,
} from '../../../helpers';

const { Text } = Typography;

const UPLOAD_MODE_DISABLED = 'disabled';

const UPLOAD_MODE_OPTIONS = [
  { label: '关闭文件上传', value: UPLOAD_MODE_DISABLED },
  { label: '本地存储', value: 'local' },
  { label: '阿里云 OSS', value: 'oss' },
];

const BASE_INPUTS = {
  'oss_setting.enabled': false,
  'oss_setting.storage_type': 'local',
  'oss_setting.endpoint': '',
  'oss_setting.bucket': '',
  'oss_setting.access_key_id': '',
  'oss_setting.access_key_secret': '',
  'oss_setting.public_base_url': '',
  'oss_setting.object_key_prefix': 'uploads/',
  'oss_setting.max_file_size_mb': 20,
  'oss_setting.local_max_file_size_mb': 20,
  'oss_setting.oss_max_file_size_mb': 20,
  'oss_setting.local_storage_path': '.',
  'oss_setting.local_url_prefix': '',
  'oss_setting.local_object_key_prefix': '',
};

const APPLICATION_KEYS = ['oss_setting.enabled', 'oss_setting.storage_type'];

const LOCAL_CONFIG_KEYS = [
  'oss_setting.local_storage_path',
  'oss_setting.local_url_prefix',
  'oss_setting.local_object_key_prefix',
  'oss_setting.local_max_file_size_mb',
];

const OSS_CONFIG_KEYS = [
  'oss_setting.endpoint',
  'oss_setting.bucket',
  'oss_setting.access_key_id',
  'oss_setting.access_key_secret',
  'oss_setting.public_base_url',
  'oss_setting.object_key_prefix',
  'oss_setting.oss_max_file_size_mb',
];

const panelStyle = {
  border: '1px solid var(--semi-color-border)',
  borderRadius: 8,
  padding: 16,
  background: 'var(--semi-color-bg-0)',
};

const testPanelStyle = {
  border: '1px solid var(--semi-color-border)',
  borderRadius: 8,
  padding: 12,
  background: 'var(--semi-color-fill-0)',
  marginTop: 16,
};

function trimTrailingSlash(value) {
  return String(value || '').trim().replace(/\/+$/, '');
}

function trimWrappingSlash(value) {
  return String(value || '').trim().replace(/^\/+|\/+$/g, '');
}

function withTrailingSlash(value) {
  const trimmed = String(value || '').trim();
  if (!trimmed) return '';
  return trimmed.endsWith('/') ? trimmed : `${trimmed}/`;
}

function normalizeLocalFolderPrefix(value) {
  const raw = String(value || '').trim();
  if (!raw) return { value: '' };
  if (raw.includes('\\') || raw.includes(':')) {
    return { error: '本地文件夹前缀只能使用 / 分隔相对路径' };
  }
  const trimmed = raw.replace(/^\/+|\/+$/g, '');
  if (!trimmed) return { value: '' };
  if (trimmed.startsWith('../') || trimmed === '..' || trimmed.startsWith('/')) {
    return { error: '本地文件夹前缀不能包含上级目录或绝对路径' };
  }
  const parts = trimmed.split('/').filter(Boolean);
  for (const part of parts) {
    if (part === '.' || part === '..') {
      return { error: '本地文件夹前缀不能包含 . 或 ..' };
    }
    if (!/^[A-Za-z0-9._-]+$/.test(part)) {
      return { error: '本地文件夹前缀只能包含字母、数字、点、下划线和短横线' };
    }
  }
  return { value: parts.join('/') };
}

function getDefaultLocalAccessPrefix(options) {
  const serverAddress = trimTrailingSlash(options?.ServerAddress);
  const browserOrigin =
    typeof window !== 'undefined'
      ? trimTrailingSlash(window.location.origin)
      : '';
  const base = serverAddress || browserOrigin;
  return base ? `${base}/api` : '/api';
}

function isLikelyImageUrl(url, mimeHint) {
  if (mimeHint && mimeHint.startsWith('image/')) {
    return true;
  }
  return /\.(png|jpe?g|gif|webp|bmp|svg)(\?|$)/i.test(url || '');
}

function isBlank(value) {
  return String(value ?? '').trim() === '';
}

function storageLabel(type, t) {
  return type === 'oss' ? t('阿里云 OSS') : t('本地存储');
}

function requiredLabel(label) {
  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', whiteSpace: 'nowrap' }}>
      <span style={{ color: 'var(--semi-color-danger)', marginRight: 2 }}>
        *
      </span>
      {label}
    </span>
  );
}

function flattenOssSettingValues(values) {
  if (!values?.oss_setting || typeof values.oss_setting !== 'object') {
    return values || {};
  }
  const flatValues = { ...values };
  Object.entries(values.oss_setting).forEach(([key, value]) => {
    flatValues[`oss_setting.${key}`] = value;
  });
  delete flatValues.oss_setting;
  return flatValues;
}

function formFieldName(key) {
  return `['${key}']`;
}

function pickKeys(source, keys) {
  return keys.reduce((next, key) => {
    next[key] = source[key];
    return next;
  }, {});
}

export default function SettingsOss(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState(BASE_INPUTS);
  const [inputsRow, setInputsRow] = useState(BASE_INPUTS);
  const [testLoading, setTestLoading] = useState(false);
  const [testUrl, setTestUrl] = useState('');
  const [testMime, setTestMime] = useState('');
  const refForm = useRef();
  const fileInputRef = useRef(null);

  const selectedStorageType = inputs['oss_setting.storage_type'] || 'local';
  const appliedStorageType = inputsRow['oss_setting.storage_type'] || 'local';
  const selectedUploadMode = inputs['oss_setting.enabled']
    ? selectedStorageType
    : UPLOAD_MODE_DISABLED;
  const appliedUploadMode = inputsRow['oss_setting.enabled']
    ? appliedStorageType
    : UPLOAD_MODE_DISABLED;
  const selectedStorageLabel = storageLabel(selectedStorageType, t);
  const appliedStorageLabel = storageLabel(appliedStorageType, t);
  const appliedUploadModeLabel =
    appliedUploadMode === UPLOAD_MODE_DISABLED
      ? t('未启用')
      : appliedStorageLabel;
  const applicationDirty = compareObjects(inputs, inputsRow).some((item) =>
    APPLICATION_KEYS.includes(item.key),
  );
  const configKeys =
    selectedStorageType === 'oss' ? OSS_CONFIG_KEYS : LOCAL_CONFIG_KEYS;
  const configDirty = compareObjects(inputs, inputsRow).some((item) =>
    configKeys.includes(item.key),
  );

  const defaultLocalAccessPrefix = useMemo(
    () => getDefaultLocalAccessPrefix(props.options),
    [props.options],
  );

  const localPreviewUrl = useMemo(() => {
    const base = trimTrailingSlash(inputs['oss_setting.local_url_prefix']);
    const normalizedPrefix = normalizeLocalFolderPrefix(
      inputs['oss_setting.local_object_key_prefix'],
    );
    const prefix = trimWrappingSlash(normalizedPrefix.value || '');
    const objectPath = ['uploads', prefix, '2026/06/09/file.png']
      .filter(Boolean)
      .join('/');
    return `${base || defaultLocalAccessPrefix}/${objectPath}`;
  }, [
    defaultLocalAccessPrefix,
    inputs['oss_setting.local_url_prefix'],
    inputs['oss_setting.local_object_key_prefix'],
  ]);

  const ossPreviewUrl = useMemo(() => {
    const prefix = trimWrappingSlash(
      inputs['oss_setting.object_key_prefix'] || 'uploads/',
    );
    const publicBase = trimTrailingSlash(inputs['oss_setting.public_base_url']);
    if (publicBase) {
      return `${publicBase}/${prefix}/2026/06/09/file.png`;
    }
    const endpoint = String(inputs['oss_setting.endpoint'] || '').trim();
    const bucket = String(inputs['oss_setting.bucket'] || '').trim();
    if (endpoint && bucket) {
      return `https://${bucket}.${endpoint}/${prefix}/2026/06/09/file.png`;
    }
    return `${t('OSS 访问域名')}/${prefix}/2026/06/09/file.png`;
  }, [
    inputs['oss_setting.bucket'],
    inputs['oss_setting.endpoint'],
    inputs['oss_setting.object_key_prefix'],
    inputs['oss_setting.public_base_url'],
    t,
  ]);

  function setFields(fields) {
    setInputs((prev) => {
      const nextInputs = { ...prev, ...fields };
      syncFormValues(nextInputs);
      return nextInputs;
    });
  }

  function syncFormValues(nextInputs) {
    if (refForm.current) {
      const formValues = pickKeys(nextInputs, [
        ...LOCAL_CONFIG_KEYS,
        ...OSS_CONFIG_KEYS,
      ]);
      refForm.current.setValues(formValues);
      Object.entries(formValues).forEach(([key, value]) => {
        refForm.current.setValue(formFieldName(key), value);
      });
    }
  }

  function setField(key, value) {
    setFields({ [key]: value });
  }

  function buildUploadModeFields(value) {
    if (value === UPLOAD_MODE_DISABLED) {
      return { 'oss_setting.enabled': false };
    }
    const fields = {
      'oss_setting.enabled': true,
      'oss_setting.storage_type': value,
    };
    if (value === 'local') {
      if (!Number(inputs['oss_setting.local_max_file_size_mb'])) {
        fields['oss_setting.local_max_file_size_mb'] = 20;
      }
    }
    if (value === 'oss' && !Number(inputs['oss_setting.oss_max_file_size_mb'])) {
      fields['oss_setting.oss_max_file_size_mb'] = 20;
    }
    return fields;
  }

  function getFormSnapshot(overrides = {}) {
    const formValues = flattenOssSettingValues(
      refForm.current?.getValues?.() || {},
    );
    return {
      ...inputs,
      ...formValues,
      'oss_setting.enabled': inputs['oss_setting.enabled'],
      'oss_setting.storage_type': inputs['oss_setting.storage_type'],
      ...overrides,
    };
  }

  function validateConfigForStorage(storageType, snapshot = getFormSnapshot()) {
    if (storageType === 'oss') {
      const requiredFields = [
        ['oss_setting.endpoint', t('Endpoint')],
        ['oss_setting.bucket', t('Bucket 名称')],
        ['oss_setting.access_key_id', t('AccessKey ID')],
        ['oss_setting.access_key_secret', t('AccessKey Secret')],
      ];
      const missing = requiredFields.find(([key]) => isBlank(snapshot[key]));
      if (missing) {
        showError(t('请填写{{field}}', { field: missing[1] }));
        return false;
      }
      if (!Number(snapshot['oss_setting.oss_max_file_size_mb'])) {
        showError(t('请填写单文件大小上限'));
        return false;
      }
      return true;
    }

    if (!Number(snapshot['oss_setting.local_max_file_size_mb'])) {
      showError(t('请填写单文件大小上限'));
      return false;
    }
    const localPrefix = normalizeLocalFolderPrefix(
      snapshot['oss_setting.local_object_key_prefix'],
    );
    if (localPrefix.error) {
      showError(t(localPrefix.error));
      return false;
    }
    return true;
  }

  function validateCurrentConfig(snapshot = getFormSnapshot()) {
    return validateConfigForStorage(selectedStorageType, snapshot);
  }

  function validateApplicationSave() {
    const snapshot = getFormSnapshot();
    if (selectedUploadMode === UPLOAD_MODE_DISABLED) {
      return true;
    }
    if (!validateCurrentConfig(snapshot)) {
      return false;
    }
    if (configDirty) {
      showWarning(
        t('请先保存{{type}}配置，再使用它作为存储', {
          type: selectedStorageLabel,
        }),
      );
      return false;
    }
    return true;
  }

  function hasUnsavedConfig(snapshot = getFormSnapshot()) {
    return hasUnsavedConfigForStorage(selectedStorageType, snapshot);
  }

  function hasUnsavedConfigForStorage(storageType, snapshot = getFormSnapshot()) {
    const targetConfigKeys =
      storageType === 'oss' ? OSS_CONFIG_KEYS : LOCAL_CONFIG_KEYS;
    return compareObjects(snapshot, inputsRow).some((item) =>
      targetConfigKeys.includes(item.key),
    );
  }

  function normalizeSnapshotForSave(snapshot, keys) {
    const next = { ...snapshot };
    if (keys.includes('oss_setting.local_object_key_prefix')) {
      const localPrefix = normalizeLocalFolderPrefix(
        next['oss_setting.local_object_key_prefix'],
      );
      if (!localPrefix.error) {
        next['oss_setting.local_object_key_prefix'] = localPrefix.value;
      }
    }
    return next;
  }

  function uploadModeStorageType(value) {
    return value === UPLOAD_MODE_DISABLED
      ? inputs['oss_setting.storage_type'] || 'local'
      : value;
  }

  function normalizeLoadedInputs(next) {
    if (!next['oss_setting.storage_type']) {
      next['oss_setting.storage_type'] = 'local';
    }
    if (!next['oss_setting.local_storage_path']) {
      next['oss_setting.local_storage_path'] = '.';
    }
    if (next['oss_setting.local_object_key_prefix']) {
      const localPrefix = normalizeLocalFolderPrefix(
        next['oss_setting.local_object_key_prefix'],
      );
      if (!localPrefix.error) {
        next['oss_setting.local_object_key_prefix'] = localPrefix.value;
      }
    }
    if (!next['oss_setting.object_key_prefix']) {
      next['oss_setting.object_key_prefix'] = 'uploads/';
    }
    if (!next['oss_setting.local_max_file_size_mb']) {
      next['oss_setting.local_max_file_size_mb'] =
        next['oss_setting.max_file_size_mb'] || 20;
    }
    if (!next['oss_setting.oss_max_file_size_mb']) {
      next['oss_setting.oss_max_file_size_mb'] =
        next['oss_setting.max_file_size_mb'] || 20;
    }
    return next;
  }

  async function saveKeys(
    keys,
    { successMessage, silent = false, overrides = {} } = {},
  ) {
    const sourceInputs = normalizeSnapshotForSave(
      getFormSnapshot(overrides),
      keys,
    );
    const changes = compareObjects(sourceInputs, inputsRow).filter((item) =>
      keys.includes(item.key),
    );
    if (!changes.length) {
      if (!silent) showWarning(t('当前配置没有修改'));
      return true;
    }
    setLoading(true);
    try {
      const res = await Promise.all(
        changes.map((item) => {
          const rawValue = sourceInputs[item.key];
          const value =
            typeof rawValue === 'boolean' || typeof rawValue === 'number'
              ? String(rawValue)
              : rawValue;
          return API.put('/api/option/', {
            key: item.key,
            value,
          });
        }),
      );
      const failed = res.find((item) => item?.data?.success === false);
      if (failed) {
        showError(failed?.data?.message || t('保存失败，请重试'));
        return false;
      }
      setInputsRow((prev) => {
        const next = { ...prev };
        changes.forEach((item) => {
          next[item.key] = sourceInputs[item.key];
        });
        return next;
      });
      setInputs((prev) => ({ ...prev, ...sourceInputs }));
      if (!silent) {
        showSuccess(successMessage || t('保存成功'));
      }
      if (props.refresh) {
        await props.refresh();
      }
      return true;
    } catch {
      showError(t('保存失败，请重试'));
      return false;
    } finally {
      setLoading(false);
    }
  }

  async function saveApplication() {
    if (!validateApplicationSave()) return false;
    return saveKeys(APPLICATION_KEYS, {
      successMessage: inputs['oss_setting.enabled']
        ? t('已使用{{type}}作为存储', { type: selectedStorageLabel })
        : t('已关闭文件上传'),
    });
  }

  async function applyUploadMode(value) {
    const fields = buildUploadModeFields(value);
    setFields(fields);

    if (value !== UPLOAD_MODE_DISABLED) {
      const storageType = uploadModeStorageType(value);
      const snapshot = getFormSnapshot(fields);
      if (!validateConfigForStorage(storageType, snapshot)) {
        return false;
      }
      if (hasUnsavedConfigForStorage(storageType, snapshot)) {
        showWarning(
          t('请先保存{{type}}配置并应用', {
            type: storageLabel(storageType, t),
          }),
        );
        return false;
      }
    }

    return saveKeys(APPLICATION_KEYS, {
      overrides: fields,
      successMessage:
        value === UPLOAD_MODE_DISABLED
          ? t('已关闭文件上传')
          : t('已使用{{type}}作为存储', {
              type: storageLabel(value, t),
            }),
    });
  }

  async function saveCurrentConfig({ silent = false } = {}) {
    if (!validateCurrentConfig()) return false;
    return saveKeys(configKeys, {
      successMessage: t('已保存{{type}}配置', {
        type: selectedStorageLabel,
      }),
      silent,
    });
  }

  async function saveCurrentConfigAndApply() {
    const storageType = selectedStorageType;
    const fields = {
      'oss_setting.enabled': true,
      'oss_setting.storage_type': storageType,
    };
    const snapshot = getFormSnapshot(fields);
    if (!validateConfigForStorage(storageType, snapshot)) return false;

    const savedConfig = await saveKeys(configKeys, {
      silent: true,
      overrides: fields,
    });
    if (!savedConfig) return false;

    const applied = await saveKeys(APPLICATION_KEYS, {
      overrides: fields,
      silent: true,
    });
    if (!applied) return false;

    setFields(fields);
    showSuccess(
      t('已保存{{type}}配置并应用', {
        type: selectedStorageLabel,
      }),
    );
    return true;
  }

  async function uploadTestFile(file) {
    if (!file) return;
    setTestLoading(true);
    setTestUrl('');
    setTestMime('');
    try {
      const snapshot = getFormSnapshot();
      if (!validateCurrentConfig(snapshot)) return;
      if (selectedUploadMode === UPLOAD_MODE_DISABLED) {
        showWarning(t('请先在上方下拉框选择本地存储或阿里云 OSS 并应用'));
        return;
      }
      if (hasUnsavedConfig(snapshot)) {
        showWarning(
          t('请先保存{{type}}配置，再测试上传', {
            type: selectedStorageLabel,
          }),
        );
        return;
      }
      if (applicationDirty || selectedStorageType !== appliedStorageType) {
        showWarning(
          t('请先点击“使用{{type}}作为存储”，再测试上传', {
            type: selectedStorageLabel,
          }),
        );
        return;
      }
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
        t('上传失败，请确认已保存存储配置且已启用');
      showError(msg);
    } finally {
      setTestLoading(false);
    }
  }

  useEffect(() => {
    setInputs((prev) => {
      const next = { ...prev };
      for (const k of Object.keys(next)) {
        if (!k.startsWith('oss_setting.')) continue;
        if (props.options[k] === undefined) continue;
        const v = props.options[k];
        if (k === 'oss_setting.enabled') {
          next[k] = toBoolean(v);
        } else if (
          k === 'oss_setting.max_file_size_mb' ||
          k === 'oss_setting.local_max_file_size_mb' ||
          k === 'oss_setting.oss_max_file_size_mb'
        ) {
          const n = parseInt(String(v), 10);
          next[k] = Number.isFinite(n) ? n : 20;
        } else {
          next[k] = v;
        }
      }
      const normalized = normalizeLoadedInputs(next);
      setInputsRow(structuredClone(normalized));
      syncFormValues(normalized);
      return normalized;
    });
  }, [props.options]);

  useEffect(() => {
    syncFormValues(inputs);
  }, [selectedUploadMode, selectedStorageType]);

  const renderTestPanel = () => (
    <div style={testPanelStyle}>
      <div style={{ marginBottom: 8 }}>
        <Text strong>{t('测试上传与预览')}</Text>
        <Text type='tertiary' size='small' style={{ marginLeft: 8 }}>
          {t('测试使用已保存并应用的当前存储方式，不会自动保存配置')}
        </Text>
      </div>
      <Row type='flex' align='middle' gutter={12} style={{ flexWrap: 'wrap' }}>
        <Col>
          <input
            ref={fileInputRef}
            type='file'
            style={{ display: 'none' }}
            disabled={testLoading}
            onChange={(e) => {
              const file = e.target.files?.[0];
              if (file) uploadTestFile(file);
              e.target.value = '';
            }}
          />
          <Button
            loading={testLoading}
            disabled={testLoading}
            onClick={() => fileInputRef.current?.click()}
          >
            {testLoading ? t('上传中…') : t('测试上传')}
          </Button>
        </Col>
        <Col>
          <Text type='tertiary' size='small'>
            {selectedStorageType === 'oss'
              ? t('将使用阿里云 OSS 配置测试')
              : t('将使用本地存储配置测试')}
          </Text>
        </Col>
      </Row>
      {testUrl ? (
        <div style={{ marginTop: 12 }}>
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
            <img
              src={testUrl}
              alt='upload-test'
              style={{
                maxWidth: '100%',
                maxHeight: 240,
                objectFit: 'contain',
                border: '1px solid var(--semi-color-border)',
                borderRadius: 4,
              }}
            />
          ) : (
            <Text type='tertiary' size='small'>
              {t('非图片类型，请通过上方链接在新标签页中打开验证。')}
            </Text>
          )}
        </div>
      ) : null}
    </div>
  );

  const renderLocalConfig = () => (
    <div style={panelStyle}>
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          gap: 12,
          marginBottom: 14,
          flexWrap: 'wrap',
        }}
      >
        <div>
          <Text strong style={{ fontSize: 16 }}>
            {t('本地存储配置')}
          </Text>
          <Text type='tertiary' size='small' style={{ marginLeft: 8 }}>
            {t('配置会独立保存，切换到 OSS 不会清空')}
          </Text>
        </div>
        <Tag color='green' shape='circle'>
          {selectedStorageType === appliedStorageType
            ? t('当前应用中')
            : t('正在编辑')}
        </Tag>
      </div>
      <Banner
        type='info'
        closeIcon={null}
        description={t(
          '本地文件固定写入程序目录 uploads/ 下，并按「本地文件夹前缀 + 年/月/日 + 文件名」分层。',
        )}
        style={{ marginBottom: 16 }}
      />
      <Row gutter={16}>
        <Col xs={24} sm={12}>
          <Form.Input
            field="['oss_setting.local_url_prefix']"
            label={t('访问前缀')}
            placeholder={defaultLocalAccessPrefix}
            onChange={(v) => setField('oss_setting.local_url_prefix', v)}
          />
          <Text type='tertiary' size='small'>
            {t('不填时使用系统默认 URL')}: {defaultLocalAccessPrefix}
          </Text>
        </Col>
        <Col xs={24} sm={12}>
          <Form.Input
            field="['oss_setting.local_object_key_prefix']"
            label={t('本地文件夹前缀')}
            onChange={(v) => {
              const localPrefix = normalizeLocalFolderPrefix(v);
              setField(
                'oss_setting.local_object_key_prefix',
                localPrefix.error ? v : localPrefix.value,
              );
            }}
          />
          <Text type='tertiary' size='small'>
            {t('可选，表示系统 uploads 目录下的子文件夹；例如填写 a 会写入 uploads/a。前后的 / 会自动移除。')}
          </Text>
        </Col>
        <Col xs={24} sm={12}>
          <Form.InputNumber
            field="['oss_setting.local_max_file_size_mb']"
            label={requiredLabel(t('单文件大小上限（MB）'))}
            min={1}
            max={1024}
            onChange={(v) => setField('oss_setting.local_max_file_size_mb', v)}
          />
        </Col>
        <Col span={24}>
          <Text type='tertiary' size='small'>
            {t('生成示例')}: {localPreviewUrl}
          </Text>
        </Col>
      </Row>
      {renderTestPanel()}
      <Row gutter={8} style={{ marginTop: 16 }}>
        <Col>
          <Button
            type='primary'
            onClick={saveCurrentConfigAndApply}
          >
            {t('保存本地存储配置并应用')}
          </Button>
        </Col>
      </Row>
    </div>
  );

  const renderOssConfig = () => (
    <div style={panelStyle}>
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          gap: 12,
          marginBottom: 14,
          flexWrap: 'wrap',
        }}
      >
        <div>
          <Text strong style={{ fontSize: 16 }}>
            {t('阿里云 OSS 配置')}
          </Text>
          <Text type='tertiary' size='small' style={{ marginLeft: 8 }}>
            {t('配置会独立保存，切换到本地存储不影响')}
          </Text>
        </div>
        <Tag color='blue' shape='circle'>
          {selectedStorageType === appliedStorageType
            ? t('当前应用中')
            : t('正在编辑')}
        </Tag>
      </div>
      <Row gutter={16}>
        <Col xs={24} sm={12}>
          <Form.Input
            field="['oss_setting.endpoint']"
            label={requiredLabel(t('Endpoint'))}
            placeholder='oss-cn-guangzhou.aliyuncs.com'
            onChange={(v) => setField('oss_setting.endpoint', v)}
          />
          <Text type='tertiary' size='small'>
            {t('不含 https://，与阿里云控制台 Bucket 概览中的外网 Endpoint 一致。')}
          </Text>
        </Col>
        <Col xs={24} sm={12}>
          <Form.Input
            field="['oss_setting.bucket']"
            label={requiredLabel(t('Bucket 名称'))}
            onChange={(v) => setField('oss_setting.bucket', v)}
          />
        </Col>
      </Row>
      <Row gutter={16} style={{ marginTop: 8 }}>
        <Col xs={24} sm={12}>
          <Form.Input
            field="['oss_setting.access_key_id']"
            label={requiredLabel(t('AccessKey ID'))}
            onChange={(v) => setField('oss_setting.access_key_id', v)}
          />
        </Col>
        <Col xs={24} sm={12}>
          <Form.Input
            field="['oss_setting.access_key_secret']"
            label={requiredLabel(t('AccessKey Secret'))}
            type='password'
            onChange={(v) => setField('oss_setting.access_key_secret', v)}
          />
        </Col>
      </Row>
      <Row gutter={16} style={{ marginTop: 8 }}>
        <Col span={24}>
          <Form.Input
            field="['oss_setting.public_base_url']"
            label={t('对外访问基址（可选）')}
            placeholder='https://cdn.example.com'
            onChange={(v) => setField('oss_setting.public_base_url', v)}
          />
          <Text type='tertiary' size='small'>
            {t('留空则使用 https://{bucket}.{endpoint}/；绑定 CDN 时填 CDN 根地址。')}
          </Text>
        </Col>
      </Row>
      <Row gutter={16} style={{ marginTop: 8 }}>
        <Col xs={24} sm={12}>
          <Form.Input
            field="['oss_setting.object_key_prefix']"
            label={t('对象键前缀')}
            placeholder='uploads/'
            onChange={(v) =>
              setField('oss_setting.object_key_prefix', withTrailingSlash(v))
            }
          />
          <Text type='tertiary' size='small'>
            {t('类似 OSS 中的文件夹，例如 uploads/ 或 images/。')}
          </Text>
        </Col>
        <Col xs={24} sm={12}>
          <Form.InputNumber
            field="['oss_setting.oss_max_file_size_mb']"
            label={requiredLabel(t('单文件大小上限（MB）'))}
            min={1}
            max={1024}
            onChange={(v) => setField('oss_setting.oss_max_file_size_mb', v)}
          />
        </Col>
      </Row>
      <Row gutter={16} style={{ marginTop: 8 }}>
        <Col span={24}>
          <Text type='tertiary' size='small'>
            {t('生成示例')}: {ossPreviewUrl}
          </Text>
        </Col>
      </Row>
      {renderTestPanel()}
      <Row gutter={8} style={{ marginTop: 16 }}>
        <Col>
          <Button
            type='primary'
            onClick={saveCurrentConfigAndApply}
          >
            {t('保存 OSS 配置并应用')}
          </Button>
        </Col>
      </Row>
    </div>
  );

  const renderDisabledConfig = () => (
    <div style={panelStyle}>
      <Text strong style={{ fontSize: 16 }}>
        {t('文件上传已关闭')}
      </Text>
      <Banner
        type='info'
        closeIcon={null}
        description={t('保存后前端仍走统一上传接口，但后端会拒绝文件上传请求。')}
        style={{ marginTop: 14 }}
      />
    </div>
  );

  const renderSelectedConfig = () => {
    if (selectedUploadMode === UPLOAD_MODE_DISABLED) {
      return renderDisabledConfig();
    }
    return selectedStorageType === 'oss' ? renderOssConfig() : renderLocalConfig();
  };

  return (
    <Spin spinning={loading}>
      <Form
        values={inputs}
        getFormApi={(formAPI) => (refForm.current = formAPI)}
        style={{ marginBottom: 15 }}
      >
        <div style={{ ...panelStyle, marginBottom: 16 }}>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              gap: 12,
              marginBottom: 12,
              flexWrap: 'wrap',
            }}
          >
            <div>
              <Text strong style={{ fontSize: 16 }}>
                {t('文件上传')}
              </Text>
              <Tag
                color={appliedUploadMode === UPLOAD_MODE_DISABLED ? 'grey' : 'green'}
                shape='circle'
                style={{ marginLeft: 8 }}
              >
                {appliedUploadMode === UPLOAD_MODE_DISABLED ? t('未启用') : t('已启用')}
              </Tag>
              <Tag
                color={
                  appliedUploadMode === UPLOAD_MODE_DISABLED
                    ? 'grey'
                    : appliedStorageType === 'oss'
                      ? 'blue'
                      : 'green'
                }
                shape='circle'
                style={{ marginLeft: 8 }}
              >
                {t('当前使用')}: {appliedUploadModeLabel}
              </Tag>
              {applicationDirty ? (
                <Tag color='orange' shape='circle' style={{ marginLeft: 8 }}>
                  {t('应用方式未保存')}
                </Tag>
              ) : null}
            </div>
            <Text type='tertiary' size='small'>
              {t('选择文件上传方式后会立即应用；缺少必填配置时会提示先补全。')}
            </Text>
          </div>
          <Row gutter={16} type='flex' align='bottom'>
            <Col xs={24}>
              <div style={{ marginBottom: 4 }}>
                <Text strong>{t('文件上传方式')}</Text>
              </div>
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 8,
                  flexWrap: 'wrap',
                }}
              >
                <Select
                  value={selectedUploadMode}
                  optionList={UPLOAD_MODE_OPTIONS.map((item) => ({
                    ...item,
                    label: t(item.label),
                  }))}
                  style={{ width: 220 }}
                  onChange={async (value) => {
                    setTestUrl('');
                    setTestMime('');
                    await applyUploadMode(value);
                  }}
                />
              </div>
            </Col>
          </Row>
        </div>

        {renderSelectedConfig()}
      </Form>
    </Spin>
  );
}
