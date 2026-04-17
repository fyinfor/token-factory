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

import React, {
  useEffect,
  useState,
  useRef,
  useCallback,
  useMemo,
} from 'react';
import {
  Button,
  Col,
  Form,
  Row,
  Spin,
  Upload,
  Typography,
  Space,
  InputNumber,
} from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import {
  compareObjects,
  API,
  showError,
  showSuccess,
  showWarning,
  quotaToDisplayInputAmount,
  displayInputAmountToQuota,
} from '../../../helpers';

const { Text } = Typography;

export default function SettingsDistributor(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    AffiliateDefaultCommissionBps: '1000',
    DistributorApplyCsImageUrl: '',
    DistributorWithdrawCsImageUrl: '',
    DistributorWithdrawNotice: '',
    DistributorMinWithdrawQuota: '',
  });
  const refForm = useRef();
  const [inputsRow, setInputsRow] = useState(inputs);
  const [uploadingKey, setUploadingKey] = useState(null);
  const [showManualUrl, setShowManualUrl] = useState(false);

  /** 与分销中心、余额展示一致：TOKENS 用整数点；否则用标价金额 */
  const isQuotaTokensMode = useMemo(() => {
    const fromOpt = props.options?.['general_setting.quota_display_type'];
    if (typeof fromOpt === 'string' && fromOpt) {
      return fromOpt === 'TOKENS';
    }
    if (typeof window === 'undefined') return false;
    return (localStorage.getItem('quota_display_type') || 'USD') === 'TOKENS';
  }, [props.options]);

  const minWithdrawDisplayValue = useMemo(() => {
    const s = String(inputs.DistributorMinWithdrawQuota ?? '').trim();
    if (s === '') return undefined;
    const n = parseInt(s, 10);
    if (!Number.isFinite(n) || n <= 0) return undefined;
    return quotaToDisplayInputAmount(n);
  }, [inputs.DistributorMinWithdrawQuota]);

  const persistOptionValue = useCallback(
    async (key, value) => {
      setLoading(true);
      try {
        const res = await API.put('/api/option/', {
          key,
          value: value ?? '',
        });
        if (!res.data.success) {
          showError(res.data.message);
          return;
        }
        setInputs((prev) => ({ ...prev, [key]: value }));
        setInputsRow((prev) => ({ ...prev, [key]: value }));
        refForm.current?.setValues({ [key]: value });
        showSuccess(t('已保存'));
        props.refresh?.();
      } catch {
        showError(t('保存失败'));
      } finally {
        setLoading(false);
      }
    },
    [props, t],
  );

  const handleDistributorImageUpload = useCallback(
    (optionKey) => async ({ file, onSuccess, onError }) => {
      const inst = file.fileInstance || file;
      if (!inst) {
        onError(new Error('no file'));
        return;
      }
      setUploadingKey(optionKey);
      const fd = new FormData();
      fd.append('file', inst);
      try {
        const res = await API.post('/api/oss/upload', fd, {
          skipErrorHandler: true,
        });
        const { success, message, data } = res.data || {};
        const url = data?.url;
        if (!success || !url) {
          showError(message || t('上传失败'));
          onError(new Error(message));
          return;
        }
        await persistOptionValue(optionKey, url);
        onSuccess(data);
      } catch (e) {
        const msg =
          e?.response?.data?.message ||
          e?.message ||
          t('上传失败，请确认已启用 OSS 并完成配置');
        showError(msg);
        onError(e);
      } finally {
        setUploadingKey(null);
      }
    },
    [persistOptionValue, t],
  );

  function onSubmit() {
    const updateArray = compareObjects(inputs, inputsRow);
    if (!updateArray.length) return showWarning(t('你似乎并没有修改什么'));
    const requestQueue = updateArray.map((item) => {
      let value = '';
      if (typeof inputs[item.key] === 'boolean') {
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
        if (requestQueue.length === 1) {
          if (res.includes(undefined)) return;
        } else if (requestQueue.length > 1) {
          if (res.includes(undefined))
            return showError(t('部分保存失败，请重试'));
        }
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

  useEffect(() => {
    const currentInputs = {};
    for (let key in props.options) {
      if (Object.keys(inputs).includes(key)) {
        currentInputs[key] = props.options[key];
      }
    }
    setInputs(currentInputs);
    setInputsRow(structuredClone(currentInputs));
    refForm.current.setValues(currentInputs);
  }, [props.options]);

  return (
    <>
      <Spin spinning={loading}>
        <Form
          values={inputs}
          getFormApi={(formAPI) => (refForm.current = formAPI)}
          style={{ marginBottom: 15 }}
        >
          <Form.Section text={t('分销商设置')}>
            <Typography.Text
              type='tertiary'
              style={{ marginBottom: 16, display: 'block' }}
            >
              {t(
                '配置默认分销比例、申请页与提现页客服图、提现说明及最低提现额度等。',
              )}
            </Typography.Text>
            <Row gutter={16}>
              <Col xs={24} sm={12} md={10}>
                {isQuotaTokensMode ? (
                  <Form.InputNumber
                    label={t('分销商最低提现额度')}
                    field='DistributorMinWithdrawQuota'
                    step={1}
                    min={0}
                    extraText={t(
                      '与「待使用收益」同一数字。有填写则单次提现不得低于该额度；留空则不在此单独设限，仅按系统默认最低提现执行。',
                    )}
                    placeholder={t('留空为默认')}
                    onChange={(value) =>
                      setInputs({
                        ...inputs,
                        DistributorMinWithdrawQuota:
                          value === null || value === undefined
                            ? ''
                            : String(value),
                      })
                    }
                  />
                ) : (
                  <div>
                    <Text strong className='block mb-2'>
                      {t('分销商最低提现金额')}
                    </Text>
                    <InputNumber
                      value={minWithdrawDisplayValue}
                      onNumberChange={(v) => {
                        if (
                          v === null ||
                          v === undefined ||
                          (typeof v === 'number' && Number.isNaN(v))
                        ) {
                          setInputs({
                            ...inputs,
                            DistributorMinWithdrawQuota: '',
                          });
                          return;
                        }
                        const q = displayInputAmountToQuota(Number(v));
                        setInputs({
                          ...inputs,
                          DistributorMinWithdrawQuota: String(
                            Math.max(0, Math.round(q)),
                          ),
                        });
                      }}
                      min={0}
                      precision={2}
                      step={0.01}
                      style={{ width: '100%' }}
                      placeholder={t('留空为默认')}
                    />
                    <Text
                      type='tertiary'
                      size='small'
                      className='block mt-2'
                    >
                      {t(
                        '与「待使用收益」同一套标价。有填写则单次提现不得低于对应余额；留空则不在此单独设限，仅按系统默认最低提现（与单价对应额度换算）执行。',
                      )}
                    </Text>
                  </div>
                )}
              </Col>
            </Row>
            <Row gutter={16} className='mt-8'>
              <Col xs={24} sm={12} md={8}>
                <div className='mb-4'>
                  <Text strong className='block mb-2'>
                    {t('默认分销比例')}
                  </Text>
                  <InputNumber
                    value={Number(inputs.AffiliateDefaultCommissionBps || 0) / 100}
                    onNumberChange={(n) =>
                      setInputs({
                        ...inputs,
                        AffiliateDefaultCommissionBps:
                          n === null ||
                          n === undefined ||
                          (typeof n === 'number' && Number.isNaN(n))
                            ? '0'
                            : String(Math.round(Number(n) * 100)),
                      })
                    }
                    step={0.01}
                    min={0}
                    max={100}
                    suffix='%'
                    style={{ width: '100%' }}
                  />
                  <Text type='tertiary' size='small' className='block mt-2'>
                    {t(
                      '填写 0～100 之间的百分比，例如 10 表示 10%；被邀请用户充值时按此比例给邀请人分成',
                    )}
                  </Text>
                </div>
              </Col>
            </Row>
            <Row gutter={16}>
              <Col xs={24} md={12}>
                <Text strong className='block mb-2'>
                  {t('分销商申请页右侧客服图片')}
                </Text>
                <Text type='tertiary' size='small' className='block mb-3'>
                  {t('上传后自动保存；需先在系统设置中配置并启用阿里云 OSS')}
                </Text>
                {inputs.DistributorApplyCsImageUrl ? (
                  <Space align='start' spacing='loose' wrap>
                    <img
                      src={inputs.DistributorApplyCsImageUrl}
                      alt=''
                      className='max-h-40 max-w-full rounded-lg border border-[var(--semi-color-border)] object-contain bg-[var(--semi-color-fill-0)]'
                    />
                    <Space vertical align='start' spacing='tight'>
                      <Upload
                        accept='image/*'
                        showUploadList={false}
                        customRequest={handleDistributorImageUpload(
                          'DistributorApplyCsImageUrl',
                        )}
                      >
                        <Button
                          loading={
                            uploadingKey === 'DistributorApplyCsImageUrl'
                          }
                        >
                          {t('替换图片')}
                        </Button>
                      </Upload>
                      <Button
                        type='danger'
                        theme='light'
                        onClick={() =>
                          persistOptionValue('DistributorApplyCsImageUrl', '')
                        }
                      >
                        {t('清除')}
                      </Button>
                    </Space>
                  </Space>
                ) : (
                  <Upload
                    accept='image/*'
                    showUploadList={false}
                    customRequest={handleDistributorImageUpload(
                      'DistributorApplyCsImageUrl',
                    )}
                  >
                    <Button
                      loading={uploadingKey === 'DistributorApplyCsImageUrl'}
                    >
                      {t('上传图片')}
                    </Button>
                  </Upload>
                )}
              </Col>
              <Col xs={24} md={12}>
                <Text strong className='block mb-2'>
                  {t('分销中心提现联系客服图片')}
                </Text>
                <Text type='tertiary' size='small' className='block mb-3'>
                  {t('上传后自动保存；需先在系统设置中配置并启用阿里云 OSS')}
                </Text>
                {inputs.DistributorWithdrawCsImageUrl ? (
                  <Space align='start' spacing='loose' wrap>
                    <img
                      src={inputs.DistributorWithdrawCsImageUrl}
                      alt=''
                      className='max-h-40 max-w-full rounded-lg border border-[var(--semi-color-border)] object-contain bg-[var(--semi-color-fill-0)]'
                    />
                    <Space vertical align='start' spacing='tight'>
                      <Upload
                        accept='image/*'
                        showUploadList={false}
                        customRequest={handleDistributorImageUpload(
                          'DistributorWithdrawCsImageUrl',
                        )}
                      >
                        <Button
                          loading={
                            uploadingKey === 'DistributorWithdrawCsImageUrl'
                          }
                        >
                          {t('替换图片')}
                        </Button>
                      </Upload>
                      <Button
                        type='danger'
                        theme='light'
                        onClick={() =>
                          persistOptionValue(
                            'DistributorWithdrawCsImageUrl',
                            '',
                          )
                        }
                      >
                        {t('清除')}
                      </Button>
                    </Space>
                  </Space>
                ) : (
                  <Upload
                    accept='image/*'
                    showUploadList={false}
                    customRequest={handleDistributorImageUpload(
                      'DistributorWithdrawCsImageUrl',
                    )}
                  >
                    <Button
                      loading={
                        uploadingKey === 'DistributorWithdrawCsImageUrl'
                      }
                    >
                      {t('上传图片')}
                    </Button>
                  </Upload>
                )}
              </Col>
            </Row>
            <Row gutter={16}>
              <Col span={24}>
                <Form.TextArea
                  field='DistributorWithdrawNotice'
                  label={t('分销中心线下提现说明（用户可见）')}
                  extraText={t(
                    '展示在分销商提现弹窗内；留空则不显示。可换行，建议填写到账周期、手续费或所需材料等。',
                  )}
                  placeholder={t(
                    '例如：工作日 1–3 个工作日到账；请确保银行卡信息与实名一致。',
                  )}
                  autosize={{ minRows: 4, maxRows: 12 }}
                  showClear
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      DistributorWithdrawNotice: value ?? '',
                    })
                  }
                />
              </Col>
            </Row>
            <Row>
              <Col span={24}>
                <Button
                  type='tertiary'
                  size='small'
                  onClick={() => setShowManualUrl((v) => !v)}
                >
                  {showManualUrl
                    ? t('收起手动填写')
                    : t('手动填写图片地址（高级）')}
                </Button>
              </Col>
            </Row>
            {showManualUrl ? (
              <Row gutter={16}>
                <Col xs={24} md={12}>
                  <Form.Input
                    label={t('分销商申请页右侧客服图片 URL')}
                    field='DistributorApplyCsImageUrl'
                    placeholder='https://'
                    onChange={(value) =>
                      setInputs({
                        ...inputs,
                        DistributorApplyCsImageUrl: value,
                      })
                    }
                  />
                </Col>
                <Col xs={24} md={12}>
                  <Form.Input
                    label={t('分销中心提现联系客服图片 URL')}
                    field='DistributorWithdrawCsImageUrl'
                    placeholder='https://'
                    onChange={(value) =>
                      setInputs({
                        ...inputs,
                        DistributorWithdrawCsImageUrl: value,
                      })
                    }
                  />
                </Col>
              </Row>
            ) : null}
            <Row className='mt-8'>
              <Button size='default' onClick={onSubmit}>
                {t('保存分销商设置')}
              </Button>
            </Row>
          </Form.Section>
        </Form>
      </Spin>
    </>
  );
}
