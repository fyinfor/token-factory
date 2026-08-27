import React, { useEffect, useState } from 'react';
import { Button, Radio, RadioGroup, Typography } from '@douyinfe/semi-ui';
import { API, showSuccess, showWarning } from '../../../helpers';
import { useTranslation } from 'react-i18next';
import UserIDsInput, {
  parseUserIDs,
} from '../../../components/settings/UserIDsInput';

const POLICY_KEY = 'VideoWatermarkPolicy';
const USERS_KEY = 'VideoWatermarkUserIDs';

export default function SettingsVideoWatermark({ options = {}, refresh }) {
  const { t } = useTranslation();
  const [policy, setPolicy] = useState(options[POLICY_KEY] || 'off');
  const [userIDs, setUserIDs] = useState(String(options[USERS_KEY] || ''));
  const [savedPolicy, setSavedPolicy] = useState(options[POLICY_KEY] || 'off');
  const [savedUserIDs, setSavedUserIDs] = useState(
    String(options[USERS_KEY] || ''),
  );
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const nextPolicy = options[POLICY_KEY] || 'off';
    const nextUsers = String(options[USERS_KEY] || '');
    setPolicy(nextPolicy);
    setSavedPolicy(nextPolicy);
    setUserIDs(nextUsers);
    setSavedUserIDs(nextUsers);
  }, [options]);

  const save = async () => {
    const normalizedUsers = parseUserIDs(userIDs).join(',');
    const normalizedSavedUsers = parseUserIDs(savedUserIDs).join(',');
    if (policy === 'users' && !normalizedUsers) {
      showWarning(t('请至少选择一名用户'));
      return;
    }
    if (policy === savedPolicy && normalizedUsers === normalizedSavedUsers) {
      showWarning(t('你似乎并没有修改什么'));
      return;
    }
    setLoading(true);
    try {
      const requests = [];
      if (policy !== savedPolicy)
        requests.push(
          API.put('/api/option/', { key: POLICY_KEY, value: policy }),
        );
      if (normalizedUsers !== normalizedSavedUsers)
        requests.push(
          API.put('/api/option/', { key: USERS_KEY, value: normalizedUsers }),
        );
      const responses = await Promise.all(requests);
      if (responses.some((res) => res?.data?.success === false))
        throw new Error('save failed');
      setSavedPolicy(policy);
      setSavedUserIDs(normalizedUsers);
      showSuccess(t('保存成功'));
      refresh?.();
    } catch {
      showError(t('保存失败，请重试'));
    } finally {
      setLoading(false);
    }
  };

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
          {t('视频水印策略')}
        </Typography.Title>
        <Typography.Text type='tertiary'>
          {t('对支持水印参数的视频适配器强制添加 AIGC 水印。')}
        </Typography.Text>
        <div style={{ marginTop: 16 }}>
          <RadioGroup
            value={policy}
            onChange={(e) => setPolicy(e.target.value)}
            disabled={loading}
          >
            <Radio value='off'>{t('不强制')}</Radio>
            <Radio value='all'>{t('所有用户')}</Radio>
            <Radio value='users'>{t('指定用户')}</Radio>
          </RadioGroup>
        </div>
        {policy === 'users' && (
          <div style={{ marginTop: 12 }}>
            <UserIDsInput
              value={userIDs}
              onChange={setUserIDs}
              label={t('视频水印适用用户 ID')}
              extraText={t(
                '仅名单内用户会被强制开启视频水印，多个用户 ID 使用逗号分隔',
              )}
              disabled={loading}
            />
          </div>
        )}
        <div style={{ marginTop: 16 }}>
          <Button type='primary' loading={loading} onClick={save}>
            {t('保存视频水印设置')}
          </Button>
        </div>
      </div>
    </>
  );
}
