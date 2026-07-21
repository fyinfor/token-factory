import React, { useEffect, useRef, useState } from 'react';
import {
  Button,
  Col,
  Form,
  InputNumber,
  Row,
  Spin,
  Typography,
} from '@douyinfe/semi-ui';
import {
  API,
  compareObjects,
  getCurrencyConfig,
  showError,
  showSuccess,
  showWarning,
} from '../../../helpers';
import {
  displayAmountToQuota,
  getQuotaPerUnit,
  quotaToDisplayAmount,
} from '../../../helpers/quota';
import { useTranslation } from 'react-i18next';

const resolveQuotaPerUnit = (values) => {
  const configured = Number(values?.QuotaPerUnit);
  return Number.isFinite(configured) && configured > 0
    ? configured
    : getQuotaPerUnit();
};

const defaultInputs = {
  AliyunRealNameVerificationEnabled: false,
  AliyunRealNameVerificationAccessKeyID: '',
  AliyunRealNameVerificationAccessKeySecret: '',
  AliyunRealNameVerificationRegionID: 'cn-shanghai',
  AliyunRealNameVerificationProductCode: 'ID_PRO',
  AliyunRealNameVerificationSceneID: '',
  AliyunRealNameVerificationModel: 'SILENT_LIVENESS',
  AliyunRealNameVerificationCallbackURL: '',
  AliyunRealNameVerificationReturnURL: '',
  AliyunRealNameVerificationRewardEnabled: false,
  AliyunRealNameVerificationRewardAmount: 0,
  AliyunRealNameVerificationRequiredForTopUp: false,
};

export default function SettingsAliyunRealNameVerification({
  options,
  refresh,
}) {
  const { t } = useTranslation();
  const [inputs, setInputs] = useState(defaultInputs);
  const [inputsRow, setInputsRow] = useState(defaultInputs);
  const [loading, setLoading] = useState(false);
  const [rewardQuota, setRewardQuota] = useState(0);
  const formRef = useRef();

  useEffect(() => {
    const nextInputs = { ...defaultInputs, ...options };
    setInputs(nextInputs);
    setInputsRow(nextInputs);
    const quotaPerUnit = resolveQuotaPerUnit(nextInputs);
    setRewardQuota(
      Math.max(
        0,
        Math.round(
          Number(nextInputs.AliyunRealNameVerificationRewardAmount || 0) *
            quotaPerUnit,
        ),
      ),
    );
    formRef.current?.formApi?.setValues(nextInputs);
  }, [options]);

  const updateInput = (key, value) =>
    setInputs((current) => ({ ...current, [key]: value }));

  const onSubmit = async () => {
    const normalizedInputs = {
      ...inputs,
      AliyunRealNameVerificationAccessKeyID:
        inputs.AliyunRealNameVerificationAccessKeyID?.trim() || '',
      AliyunRealNameVerificationAccessKeySecret:
        inputs.AliyunRealNameVerificationAccessKeySecret?.trim() || '',
      AliyunRealNameVerificationRegionID:
        inputs.AliyunRealNameVerificationRegionID?.trim() || '',
      AliyunRealNameVerificationProductCode:
        inputs.AliyunRealNameVerificationProductCode?.trim() || '',
      AliyunRealNameVerificationSceneID:
        inputs.AliyunRealNameVerificationSceneID?.trim() || '',
      AliyunRealNameVerificationModel:
        inputs.AliyunRealNameVerificationModel?.trim() || '',
      AliyunRealNameVerificationCallbackURL:
        inputs.AliyunRealNameVerificationCallbackURL?.trim() || '',
      AliyunRealNameVerificationReturnURL:
        inputs.AliyunRealNameVerificationReturnURL?.trim() || '',
      AliyunRealNameVerificationRewardAmount: Math.max(
        0,
        Number(inputs.AliyunRealNameVerificationRewardAmount) || 0,
      ),
    };
    if (
      normalizedInputs.AliyunRealNameVerificationAccessKeyID &&
      normalizedInputs.AliyunRealNameVerificationAccessKeyID ===
        normalizedInputs.AliyunRealNameVerificationAccessKeySecret
    ) {
      showError(
        t('AccessKey ID \u4e0d\u80fd\u4e0e AccessKey Secret \u76f8\u540c'),
      );
      return;
    }
    if (!normalizedInputs.AliyunRealNameVerificationSceneID) {
      showError(t('\u8bf7\u586b\u5199\u8ba4\u8bc1\u573a\u666f ID'));
      return;
    }
    if (!normalizedInputs.AliyunRealNameVerificationModel) {
      showError(t('\u8bf7\u586b\u5199\u6d3b\u4f53\u68c0\u6d4b\u7c7b\u578b'));
      return;
    }
    const changes = compareObjects(normalizedInputs, inputsRow);
    if (!changes.length) {
      showWarning(
        t(
          '\u4f60\u4f3c\u4e4e\u5e76\u6ca1\u6709\u4fee\u6539\u4efb\u4f55\u8bbe\u7f6e',
        ),
      );
      return;
    }
    setLoading(true);
    try {
      const responses = await Promise.all(
        changes.map((item) =>
          API.put('/api/option/', {
            key: item.key,
            value:
              typeof normalizedInputs[item.key] === 'boolean'
                ? String(normalizedInputs[item.key])
                : normalizedInputs[item.key],
          }),
        ),
      );
      const failedResponse = responses.find(
        (response) => !response?.data?.success,
      );
      if (failedResponse) {
        showError(failedResponse.data.message);
        return;
      }
      setInputs(normalizedInputs);
      setInputsRow(normalizedInputs);
      showSuccess(t('\u4fdd\u5b58\u6210\u529f'));
      refresh();
    } catch {
      showError(t('\u4fdd\u5b58\u5931\u8d25'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Spin spinning={loading}>
      <Form ref={formRef} initValues={inputs}>
        <Form.Section text={t('\u963f\u91cc\u4e91\u5b9e\u540d\u8ba4\u8bc1')}>
          <Row gutter={16}>
            <Col xs={24} md={8}>
              <Form.Switch
                field='AliyunRealNameVerificationEnabled'
                label={t('\u542f\u7528\u5b9e\u540d\u8ba4\u8bc1')}
                onChange={(value) =>
                  updateInput('AliyunRealNameVerificationEnabled', value)
                }
              />
            </Col>
          </Row>
          <Row gutter={16}>
            <Col xs={24} md={8}>
              <Form.Input
                field='AliyunRealNameVerificationAccessKeyID'
                label='AccessKey ID'
                onChange={(value) =>
                  updateInput('AliyunRealNameVerificationAccessKeyID', value)
                }
              />
            </Col>
            <Col xs={24} md={8}>
              <Form.Input
                field='AliyunRealNameVerificationAccessKeySecret'
                mode='password'
                label='AccessKey Secret'
                onChange={(value) =>
                  updateInput(
                    'AliyunRealNameVerificationAccessKeySecret',
                    value,
                  )
                }
              />
            </Col>
            <Col xs={24} md={8}>
              <Form.Input
                field='AliyunRealNameVerificationRegionID'
                label={t('\u5730\u57df')}
                extraText={t('\u9ed8\u8ba4 cn-shanghai')}
                onChange={(value) =>
                  updateInput('AliyunRealNameVerificationRegionID', value)
                }
              />
            </Col>
          </Row>
          <Row gutter={16}>
            <Col xs={24} md={8}>
              <Form.Input
                field='AliyunRealNameVerificationProductCode'
                label={t('\u4ea7\u54c1\u4ee3\u7801')}
                extraText={t(
                  '\u9ed8\u8ba4 ID_PRO\uff0c\u8bf7\u4e0e\u963f\u91cc\u4e91\u5df2\u5f00\u901a\u7684\u91d1\u878d\u7ea7\u5b9e\u4eba\u8ba4\u8bc1\u4ea7\u54c1\u4fdd\u6301\u4e00\u81f4\u3002',
                )}
                onChange={(value) =>
                  updateInput('AliyunRealNameVerificationProductCode', value)
                }
              />
            </Col>
            <Col xs={24} md={8}>
              <Form.Input
                field='AliyunRealNameVerificationSceneID'
                label={t('\u8ba4\u8bc1\u573a\u666f ID')}
                extraText={t(
                  '\u5fc5\u586b\u3002\u4ece\u963f\u91cc\u4e91\u5b9e\u4eba\u8ba4\u8bc1\u63a7\u5236\u53f0\u521b\u5efa\u8ba4\u8bc1\u573a\u666f\u540e\u83b7\u53d6\u3002',
                )}
                placeholder='1000000006'
                onChange={(value) =>
                  updateInput('AliyunRealNameVerificationSceneID', value)
                }
              />
            </Col>
            <Col xs={24} md={8}>
              <Form.Input
                field='AliyunRealNameVerificationModel'
                label={t('\u6d3b\u4f53\u68c0\u6d4b\u7c7b\u578b')}
                extraText={t(
                  '\u9ed8\u8ba4 SILENT_LIVENESS\uff0c\u8bf7\u4e0e\u963f\u91cc\u4e91\u8ba4\u8bc1\u573a\u666f\u652f\u6301\u7684\u7c7b\u578b\u4fdd\u6301\u4e00\u81f4\u3002',
                )}
                onChange={(value) =>
                  updateInput('AliyunRealNameVerificationModel', value)
                }
              />
            </Col>
          </Row>
          <Row gutter={16}>
            <Col xs={24} md={12}>
              <Form.Input
                field='AliyunRealNameVerificationCallbackURL'
                label={t('\u963f\u91cc\u4e91\u56de\u8c03\u5730\u5740')}
                extraText={t(
                  '\u53ef\u9009\u3002\u4f20\u7ed9\u963f\u91cc\u4e91 CallbackUrl\uff1b\u7cfb\u7edf\u4f1a\u81ea\u52a8\u8ffd\u52a0 token \u53c2\u6570\uff0c\u5efa\u8bae\u586b\u5199\u53ef\u516c\u7f51\u8bbf\u95ee\u7684 HTTPS \u63a5\u53e3\u5730\u5740\u3002',
                )}
                placeholder='https://example.com/api/aliyun/real-name/callback'
                onChange={(value) =>
                  updateInput('AliyunRealNameVerificationCallbackURL', value)
                }
              />
            </Col>
            <Col xs={24} md={12}>
              <Form.Input
                field='AliyunRealNameVerificationReturnURL'
                label={t('\u8ba4\u8bc1\u5b8c\u6210\u8df3\u8f6c\u5730\u5740')}
                extraText={t(
                  '\u53ef\u9009\u3002\u4f20\u7ed9\u963f\u91cc\u4e91 ReturnUrl\uff1b\u7cfb\u7edf\u4f1a\u81ea\u52a8\u8ffd\u52a0 token \u53c2\u6570\uff0c\u7559\u7a7a\u5219\u8df3\u8f6c\u5230\u672c\u7ad9\u5b9e\u540d\u8ba4\u8bc1\u7ed3\u679c\u9875\u3002',
                )}
                placeholder='https://example.com/real-name/result'
                onChange={(value) =>
                  updateInput('AliyunRealNameVerificationReturnURL', value)
                }
              />
            </Col>
          </Row>
          <Row gutter={16}>
            <Col xs={24} md={8}>
              <Form.Switch
                field='AliyunRealNameVerificationRewardEnabled'
                label={t('\u5b9e\u540d\u540e\u5956\u52b1\u4f59\u989d')}
                onChange={(value) =>
                  updateInput('AliyunRealNameVerificationRewardEnabled', value)
                }
              />
            </Col>
            <Col xs={24} md={8}>
              <div className='flex flex-col gap-2'>
                <Typography.Text>
                  {t('\u5956\u52b1\u4f59\u989d')}
                </Typography.Text>
                <InputNumber
                  className='w-full'
                  min={0}
                  precision={2}
                  prefix={getCurrencyConfig().symbol}
                  value={Number(
                    quotaToDisplayAmount(
                      rewardQuota,
                      resolveQuotaPerUnit(inputs),
                    ).toFixed(2),
                  )}
                  onChange={(value) => {
                    const quota = Math.max(
                      0,
                      displayAmountToQuota(
                        Number(value) || 0,
                        resolveQuotaPerUnit(inputs),
                      ),
                    );
                    setRewardQuota(quota);
                    updateInput(
                      'AliyunRealNameVerificationRewardAmount',
                      quota / resolveQuotaPerUnit(inputs),
                    );
                  }}
                />
                <Typography.Text type='tertiary' size='small'>
                  {t(
                    '\u4e0e\u7528\u6237\u7ba1\u7406\u6dfb\u52a0\u989d\u5ea6\u7684\u91d1\u989d\u6362\u7b97\u65b9\u5f0f\u4e00\u81f4\u3002',
                  )}
                </Typography.Text>
                <Typography.Text>
                  {t('\u5956\u52b1\u989d\u5ea6')}
                </Typography.Text>
                <InputNumber
                  className='w-full'
                  min={0}
                  value={rewardQuota}
                  onChange={(value) => {
                    const quota = Math.max(0, Number(value) || 0);
                    setRewardQuota(quota);
                    updateInput(
                      'AliyunRealNameVerificationRewardAmount',
                      quota / resolveQuotaPerUnit(inputs),
                    );
                  }}
                />
              </div>
            </Col>
            <Col xs={24} md={8}>
              <Form.Switch
                field='AliyunRealNameVerificationRequiredForTopUp'
                label={t(
                  '\u5145\u503c\u524d\u5f3a\u5236\u5b9e\u540d\u8ba4\u8bc1',
                )}
                onChange={(value) =>
                  updateInput(
                    'AliyunRealNameVerificationRequiredForTopUp',
                    value,
                  )
                }
              />
            </Col>
          </Row>
          <Row>
            <Button onClick={onSubmit}>
              {t(
                '\u4fdd\u5b58\u963f\u91cc\u4e91\u5b9e\u540d\u8ba4\u8bc1\u8bbe\u7f6e',
              )}
            </Button>
          </Row>
        </Form.Section>
      </Form>
    </Spin>
  );
}
