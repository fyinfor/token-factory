import React, { useEffect, useState } from 'react';
import {
  Button,
  Input,
  InputNumber,
  Radio,
  RadioGroup,
  Select,
  Slider,
  Space,
  Typography,
  Upload,
} from '@douyinfe/semi-ui';
import { IconDelete, IconUpload } from '@douyinfe/semi-icons';
import { API, showError, showSuccess, showWarning } from '../../../helpers';
import { useTranslation } from 'react-i18next';
import UserIDsInput, {
  parseUserIDs,
} from '../../../components/settings/UserIDsInput';

const KEYS = {
  policy: 'ImageWatermarkPolicy',
  users: 'ImageWatermarkUserIDs',
  type: 'ImageWatermarkType',
  text: 'ImageWatermarkText',
  logo: 'ImageWatermarkLogoURL',
  position: 'ImageWatermarkPosition',
  scale: 'ImageWatermarkScalePercent',
  margin: 'ImageWatermarkMarginPercent',
  opacity: 'ImageWatermarkOpacity',
  failure: 'ImageWatermarkFailureMode',
};

const defaults = {
  [KEYS.policy]: 'off',
  [KEYS.users]: '',
  [KEYS.type]: 'text',
  [KEYS.text]: 'TokenFactory',
  [KEYS.logo]: '',
  [KEYS.position]: 'bottom-right',
  [KEYS.scale]: '12',
  [KEYS.margin]: '3',
  [KEYS.opacity]: '0.65',
  [KEYS.failure]: 'block',
};

const snapshotFromOptions = (options) =>
  Object.fromEntries(
    Object.entries(defaults).map(([key, fallback]) => [
      key,
      options[key] === undefined || options[key] === null
        ? fallback
        : String(options[key]),
    ]),
  );

export default function SettingsImageWatermark({ options = {}, refresh }) {
  const { t } = useTranslation();
  const [values, setValues] = useState(() => snapshotFromOptions(options));
  const [saved, setSaved] = useState(() => snapshotFromOptions(options));
  const [userIDs, setUserIDs] = useState(() =>
    String(options[KEYS.users] || ''),
  );
  const [savedUserIDs, setSavedUserIDs] = useState(() =>
    String(options[KEYS.users] || ''),
  );
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);

  useEffect(() => {
    const next = snapshotFromOptions(options);
    const nextUsers = String(options[KEYS.users] || '');
    setValues(next);
    setSaved(next);
    setUserIDs(nextUsers);
    setSavedUserIDs(nextUsers);
  }, [options]);

  const setValue = (key, value) =>
    setValues((current) => ({ ...current, [key]: String(value ?? '') }));

  const uploadLogo = async ({ file, onSuccess, onError }) => {
    const instance = file.fileInstance || file;
    if (!instance) {
      onError(new Error('no file'));
      return;
    }
    setUploading(true);
    const form = new FormData();
    form.append('file', instance);
    form.append('purpose', 'watermark');
    try {
      const res = await API.post('/api/oss/upload', form, {
        skipErrorHandler: true,
      });
      const url = res.data?.data?.url;
      if (!res.data?.success || !url) {
        throw new Error(res.data?.message || t('上传失败'));
      }
      setValue(KEYS.logo, url);
      onSuccess(res.data.data);
      showSuccess(t('上传成功'));
    } catch (error) {
      showError(
        error?.response?.data?.message || error?.message || t('上传失败'),
      );
      onError(error);
    } finally {
      setUploading(false);
    }
  };

  const save = async () => {
    const normalizedUsers = parseUserIDs(userIDs).join(',');
    const normalizedSavedUsers = parseUserIDs(savedUserIDs).join(',');
    if (values[KEYS.policy] === 'users' && !normalizedUsers) {
      showWarning(t('请至少选择一名用户'));
      return;
    }
    if (values[KEYS.type] === 'logo' && !values[KEYS.logo].trim()) {
      showWarning(t('请先上传水印 Logo'));
      return;
    }
    if (values[KEYS.type] === 'text' && !values[KEYS.text].trim()) {
      showWarning(t('请输入水印文字'));
      return;
    }
    const changes = Object.entries(values).filter(
      ([key, value]) =>
        key !== KEYS.users && String(saved[key]) !== String(value),
    );
    if (normalizedUsers !== normalizedSavedUsers) {
      changes.push([KEYS.users, normalizedUsers]);
    }
    if (!changes.length) {
      showWarning(t('你似乎并没有修改什么'));
      return;
    }
    setLoading(true);
    try {
      changes.sort(([left], [right]) => {
        const disabling = values[KEYS.policy] === 'off';
        const leavingUsers =
          saved[KEYS.policy] === 'users' && values[KEYS.policy] !== 'users';
        const targetType = values[KEYS.type];
        const priority = (key) => {
          if (key === KEYS.policy)
            return disabling || leavingUsers ? -100 : 100;
          if (targetType === 'logo') {
            if (key === KEYS.logo) return -20;
            if (key === KEYS.type) return 20;
            if (key === KEYS.text) return 30;
          } else {
            if (key === KEYS.text) return -20;
            if (key === KEYS.type) return 20;
            if (key === KEYS.logo) return 30;
          }
          return 0;
        };
        return priority(left) - priority(right);
      });
      for (const [key, value] of changes) {
        const response = await API.put('/api/option/', {
          key,
          value: String(value),
        });
        if (response?.data?.success === false) {
          throw new Error(response.data?.message || 'save failed');
        }
      }
      setSaved({ ...values, [KEYS.users]: normalizedUsers });
      setSavedUserIDs(normalizedUsers);
      showSuccess(t('保存成功'));
      refresh?.();
    } catch (error) {
      showError(
        error?.response?.data?.message ||
          error?.message ||
          t('保存失败，请重试'),
      );
    } finally {
      setLoading(false);
    }
  };

  const positionOptions = [
    { value: 'top-left', label: t('左上') },
    { value: 'top-right', label: t('右上') },
    { value: 'bottom-left', label: t('左下') },
    { value: 'bottom-right', label: t('右下') },
    { value: 'center', label: t('居中') },
  ];

  return (
    <>
      <div
        style={{
          marginTop: 24,
          borderTop: '1px solid var(--semi-color-border)',
          paddingTop: 20,
        }}
      >
        <Typography.Title heading={5} style={{ marginBottom: 4 }}>
          {t('图片输出水印')}
        </Typography.Title>
        <Typography.Text type='tertiary'>
          {t(
            '平台会在图片返回前合成水印并重新托管，适用于所有标准图片生成渠道。',
          )}
        </Typography.Text>
        <div style={{ marginTop: 6 }}>
          <Typography.Text type='warning'>
            {t(
              'URL 图片会重新托管，请先在系统设置中启用本地存储或阿里云 OSS；Base64 图片不依赖托管。',
            )}
          </Typography.Text>
        </div>
        <div style={{ marginTop: 2 }}>
          <Typography.Text type='tertiary'>
            {t('处理后的图片统一输出为 PNG，暂不支持 GIF。')}
          </Typography.Text>
        </div>

        <div style={{ marginTop: 16 }}>
          <RadioGroup
            value={values[KEYS.policy]}
            onChange={(event) => setValue(KEYS.policy, event.target.value)}
            disabled={loading}
          >
            <Radio value='off'>{t('关闭')}</Radio>
            <Radio value='all'>{t('所有用户')}</Radio>
            <Radio value='users'>{t('指定用户')}</Radio>
          </RadioGroup>
        </div>

        {values[KEYS.policy] === 'users' && (
          <div style={{ marginTop: 12 }}>
            <UserIDsInput
              value={userIDs}
              onChange={setUserIDs}
              label={t('图片水印适用用户 ID')}
              extraText={t(
                '仅名单内用户会添加图片水印，多个用户 ID 使用逗号分隔',
              )}
              disabled={loading}
            />
          </div>
        )}

        {values[KEYS.policy] !== 'off' && (
          <div style={{ marginTop: 20, maxWidth: 760 }}>
            <Typography.Text strong>{t('水印内容')}</Typography.Text>
            <div style={{ marginTop: 8 }}>
              <RadioGroup
                value={values[KEYS.type]}
                onChange={(event) => setValue(KEYS.type, event.target.value)}
              >
                <Radio value='text'>{t('文字水印')}</Radio>
                <Radio value='logo'>{t('Logo 图片')}</Radio>
              </RadioGroup>
            </div>

            {values[KEYS.type] === 'text' ? (
              <div style={{ marginTop: 12, maxWidth: 440 }}>
                <Input
                  value={values[KEYS.text]}
                  onChange={(value) => setValue(KEYS.text, value)}
                  placeholder={t('水印文字')}
                  maxLength={100}
                />
                <Typography.Text type='tertiary' size='small'>
                  {t(
                    '文字水印当前仅支持英文、数字和常用符号；中文请上传 Logo 图片。',
                  )}
                </Typography.Text>
              </div>
            ) : (
              <Space align='center' wrap style={{ marginTop: 12 }}>
                {values[KEYS.logo] && (
                  <img
                    src={values[KEYS.logo]}
                    alt=''
                    style={{
                      width: 120,
                      height: 64,
                      objectFit: 'contain',
                      border: '1px solid var(--semi-color-border)',
                      background: 'var(--semi-color-fill-0)',
                    }}
                  />
                )}
                <Upload
                  action=''
                  accept='image/png,image/webp,image/jpeg'
                  showUploadList={false}
                  customRequest={uploadLogo}
                >
                  <Button icon={<IconUpload />} loading={uploading}>
                    {values[KEYS.logo] ? t('替换 Logo') : t('上传 Logo')}
                  </Button>
                </Upload>
                {values[KEYS.logo] && (
                  <Button
                    icon={<IconDelete />}
                    type='danger'
                    theme='light'
                    onClick={() => setValue(KEYS.logo, '')}
                  />
                )}
              </Space>
            )}

            <div
              style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(3, minmax(0, 1fr))',
                gap: 12,
                alignItems: 'end',
                marginTop: 20,
                maxWidth: 620,
              }}
            >
              <div style={{ minWidth: 0 }}>
                <Typography.Text type='tertiary' style={{ display: 'block' }}>
                  {t('位置')}
                </Typography.Text>
                <Select
                  value={values[KEYS.position]}
                  optionList={positionOptions}
                  onChange={(value) => setValue(KEYS.position, value)}
                  style={{ width: '100%', marginTop: 6 }}
                />
              </div>
              <div style={{ minWidth: 0 }}>
                <Typography.Text type='tertiary' style={{ display: 'block' }}>
                  {t('相对尺寸 (%)')}
                </Typography.Text>
                <InputNumber
                  value={Number(values[KEYS.scale])}
                  min={3}
                  max={40}
                  step={1}
                  onChange={(value) => setValue(KEYS.scale, value)}
                  style={{ width: '100%', marginTop: 6 }}
                />
              </div>
              <div style={{ minWidth: 0 }}>
                <Typography.Text type='tertiary' style={{ display: 'block' }}>
                  {t('边距 (%)')}
                </Typography.Text>
                <InputNumber
                  value={Number(values[KEYS.margin])}
                  min={0}
                  max={20}
                  step={1}
                  onChange={(value) => setValue(KEYS.margin, value)}
                  style={{ width: '100%', marginTop: 6 }}
                />
              </div>
            </div>

            <div style={{ marginTop: 18, maxWidth: 440 }}>
              <Typography.Text type='tertiary'>
                {t('透明度')} {Math.round(Number(values[KEYS.opacity]) * 100)}%
              </Typography.Text>
              <Slider
                value={Number(values[KEYS.opacity])}
                min={0.05}
                max={1}
                step={0.05}
                onChange={(value) => setValue(KEYS.opacity, value)}
              />
            </div>

            <div style={{ marginTop: 18 }}>
              <Typography.Text strong>{t('处理失败时')}</Typography.Text>
              <div style={{ marginTop: 8 }}>
                <RadioGroup
                  value={values[KEYS.failure]}
                  onChange={(event) =>
                    setValue(KEYS.failure, event.target.value)
                  }
                >
                  <Radio value='block'>{t('阻断返回')}</Radio>
                  <Radio value='passthrough'>{t('返回原图')}</Radio>
                </RadioGroup>
              </div>
            </div>
          </div>
        )}

        <div style={{ marginTop: 20 }}>
          <Button type='primary' loading={loading} onClick={save}>
            {t('保存图片水印设置')}
          </Button>
        </div>
      </div>
    </>
  );
}
