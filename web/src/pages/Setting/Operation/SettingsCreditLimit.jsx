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

import React, { useEffect, useState, useRef, useCallback } from 'react';
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
} from '../../../helpers';

const { Text } = Typography;

export default function SettingsCreditLimit(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    QuotaForNewUser: '',
    PreConsumedQuota: '',
    QuotaForInviter: '',
    QuotaForInvitee: '',
    AffiliateDefaultCommissionBps: '1000',
    DistributorApplyCsImageUrl: '',
    DistributorWithdrawCsImageUrl: '',
    DistributorWithdrawNotice: '',
    DistributorMinWithdrawQuota: '',
    'quota_setting.enable_free_model_pre_consume': true,
  });
  const refForm = useRef();
  const [inputsRow, setInputsRow] = useState(inputs);
  const [uploadingKey, setUploadingKey] = useState(null);
  const [showManualUrl, setShowManualUrl] = useState(false);

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
          <Form.Section text={t('额度设置')}>
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  label={t('新用户初始额度')}
                  field={'QuotaForNewUser'}
                  step={1}
                  min={0}
                  suffix={'Token'}
                  placeholder={''}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      QuotaForNewUser: String(value),
                    })
                  }
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  label={t('请求预扣费额度')}
                  field={'PreConsumedQuota'}
                  step={1}
                  min={0}
                  suffix={'Token'}
                  extraText={t('请求结束后多退少补')}
                  placeholder={''}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      PreConsumedQuota: String(value),
                    })
                  }
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  label={t('邀请新用户奖励额度')}
                  field={'QuotaForInviter'}
                  step={1}
                  min={0}
                  suffix={'Token'}
                  extraText={''}
                  placeholder={t('例如：2000')}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      QuotaForInviter: String(value),
                    })
                  }
                />
              </Col>
            </Row>
            <Row>
              <Col xs={24} sm={12} md={8} lg={8} xl={6}>
                <Form.InputNumber
                  label={t('新用户使用邀请码奖励额度')}
                  field={'QuotaForInvitee'}
                  step={1}
                  min={0}
                  suffix={'Token'}
                  extraText={''}
                  placeholder={t('例如：1000')}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      QuotaForInvitee: String(value),
                    })
                  }
                />
              </Col>
            </Row>
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8}>
                <Form.InputNumber
                  label={t('分销商线下提现最低额度（内部点数）')}
                  field='DistributorMinWithdrawQuota'
                  step={1}
                  min={0}
                  suffix={'Token'}
                  extraText={t(
                    '留空则与「单价对应额度」一致；超级管理员可限制单次线下提现下限',
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
              </Col>
            </Row>
            <Row gutter={16}>
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
                  placeholder={t('例如：工作日 1–3 个工作日到账；请确保银行卡信息与实名一致。')}
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
            <Row>
              <Col>
                <Form.Switch
                  label={t('对免费模型启用预消耗')}
                  field={'quota_setting.enable_free_model_pre_consume'}
                  extraText={t(
                    '开启后，对免费模型（倍率为0，或者价格为0）的模型也会预消耗额度',
                  )}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      'quota_setting.enable_free_model_pre_consume': value,
                    })
                  }
                />
              </Col>
            </Row>

            <Row>
              <Button size='default' onClick={onSubmit}>
                {t('保存额度设置')}
              </Button>
            </Row>
          </Form.Section>
        </Form>
      </Spin>
    </>
  );
}
