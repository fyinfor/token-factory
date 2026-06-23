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

import React, { useEffect, useMemo, useState, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import {
  API,
  downloadTextAsFile,
  showError,
  showSuccess,
  renderQuota,
  renderQuotaWithPrompt,
  getCurrencyConfig,
  getQuotaPerUnit,
  quotaToDisplayInputAmount,
} from '../../../../helpers';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import {
  Button,
  Modal,
  SideSheet,
  Space,
  Spin,
  Typography,
  Card,
  Tag,
  Form,
  Avatar,
  Row,
  Col,
} from '@douyinfe/semi-ui';
import {
  IconCreditCard,
  IconSave,
  IconClose,
  IconGift,
} from '@douyinfe/semi-icons';

const { Text, Title } = Typography;

const EditRedemptionModal = (props) => {
  const { t } = useTranslation();
  const isEdit = props.editingRedemption.id !== undefined;
  const [loading, setLoading] = useState(isEdit);
  const isMobile = useIsMobile();
  const formApiRef = useRef(null);

  const isTokensMode = useMemo(() => {
    if (typeof window === 'undefined') return false;
    return (localStorage.getItem('quota_display_type') || 'USD') === 'TOKENS';
  }, [props.visiable]);

  const quotaPresetOptions = useMemo(() => {
    const qpu = getQuotaPerUnit() || 500000;
    const usdPresets = [1, 10, 50, 100, 500, 1000];
    return usdPresets.map((usd) => {
      const quota = Math.round(usd * qpu);
      if (isTokensMode) {
        return { value: quota, label: `${usd}$` };
      }
      return {
        value: quotaToDisplayInputAmount(quota),
        label: renderQuota(quota),
      };
    });
  }, [isTokensMode, props.visiable]);

  const quotaToFormAmount = (quota) => {
    const amount = quotaToDisplayInputAmount(quota);
    if (isTokensMode) {
      return amount;
    }
    return Number(Number(amount).toFixed(2));
  };

  const displayAmountToRedemptionQuota = (amount) => {
    const val = Number(amount || 0);
    if (!Number.isFinite(val) || val <= 0) return 0;
    if (isTokensMode) return Math.ceil(val);

    const quotaPerUnit = getQuotaPerUnit() || 500000;
    const { type, rate } = getCurrencyConfig();
    const displayRate = type === 'USD' ? 1 : rate || 1;
    const rawQuota = (val / displayRate) * quotaPerUnit;
    return Math.max(0, Math.ceil(rawQuota - 1e-8));
  };

  const getInitValues = () => ({
    name: '',
    quota: 100000,
    quota_amount: quotaToFormAmount(100000),
    count: 1,
    expired_time: null,
  });

  const handleCancel = () => {
    props.handleClose();
  };

  const loadRedemption = async () => {
    setLoading(true);
    let res = await API.get(`/api/redemption/${props.editingRedemption.id}`);
    const { success, message, data } = res.data;
    if (success) {
      if (data.expired_time === 0) {
        data.expired_time = null;
      } else {
        data.expired_time = new Date(data.expired_time * 1000);
      }
      data.quota_amount = quotaToFormAmount(data.quota || 0);
      formApiRef.current?.setValues({ ...getInitValues(), ...data });
    } else {
      showError(message);
    }
    setLoading(false);
  };

  useEffect(() => {
    if (formApiRef.current) {
      if (isEdit) {
        loadRedemption();
      } else {
        formApiRef.current.setValues(getInitValues());
      }
    }
  }, [props.editingRedemption.id]);

  const submit = async (values) => {
    let name = values.name;
    const resolvedQuota = isTokensMode
      ? parseInt(values.quota, 10) || 0
      : displayAmountToRedemptionQuota(values.quota_amount);
    if (!isEdit && (!name || name === '')) {
      name = renderQuota(resolvedQuota);
    }
    setLoading(true);
    let localInputs = { ...values };
    localInputs.count = parseInt(localInputs.count) || 0;
    localInputs.quota = resolvedQuota;
    delete localInputs.quota_amount;
    localInputs.name = name;
    if (!localInputs.expired_time) {
      localInputs.expired_time = 0;
    } else {
      localInputs.expired_time = Math.floor(
        localInputs.expired_time.getTime() / 1000,
      );
    }
    let res;
    if (isEdit) {
      res = await API.put(`/api/redemption/`, {
        ...localInputs,
        id: parseInt(props.editingRedemption.id),
      });
    } else {
      res = await API.post(`/api/redemption/`, {
        ...localInputs,
      });
    }
    const { success, message, data } = res.data;
    if (success) {
      if (isEdit) {
        showSuccess(t('兑换码更新成功！'));
        props.refresh();
        props.handleClose();
      } else {
        showSuccess(t('兑换码创建成功！'));
        props.refresh();
        formApiRef.current?.setValues(getInitValues());
        props.handleClose();
      }
    } else {
      showError(message);
    }
    if (!isEdit && data) {
      let text = '';
      for (let i = 0; i < data.length; i++) {
        text += data[i] + '\n';
      }
      Modal.confirm({
        title: t('兑换码创建成功'),
        content: (
          <div>
            <p>{t('兑换码创建成功，是否下载兑换码？')}</p>
            <p>{t('兑换码将以文本文件的形式下载，文件名为兑换码的名称。')}</p>
          </div>
        ),
        onOk: () => {
          downloadTextAsFile(text, `${localInputs.name}.txt`);
        },
      });
    }
    setLoading(false);
  };

  return (
    <>
      <SideSheet
        placement={isEdit ? 'right' : 'left'}
        title={
          <Space>
            {isEdit ? (
              <Tag color='blue' shape='circle'>
                {t('更新')}
              </Tag>
            ) : (
              <Tag color='green' shape='circle'>
                {t('新建')}
              </Tag>
            )}
            <Title heading={4} className='m-0'>
              {isEdit ? t('更新兑换码信息') : t('创建新的兑换码')}
            </Title>
          </Space>
        }
        bodyStyle={{ padding: '0' }}
        visible={props.visiable}
        width={isMobile ? '100%' : 600}
        footer={
          <div className='flex justify-end bg-white'>
            <Space>
              <Button
                theme='solid'
                onClick={() => formApiRef.current?.submitForm()}
                icon={<IconSave />}
                loading={loading}
              >
                {t('提交')}
              </Button>
              <Button
                theme='light'
                type='primary'
                onClick={handleCancel}
                icon={<IconClose />}
              >
                {t('取消')}
              </Button>
            </Space>
          </div>
        }
        closeIcon={null}
        onCancel={() => handleCancel()}
      >
        <Spin spinning={loading}>
          <Form
            initValues={getInitValues()}
            getFormApi={(api) => (formApiRef.current = api)}
            onSubmit={submit}
          >
            {({ values }) => (
              <div className='p-2'>
                <Card className='!rounded-2xl shadow-sm border-0 mb-6'>
                  {/* Header: Basic Info */}
                  <div className='flex items-center mb-2'>
                    <Avatar
                      size='small'
                      color='blue'
                      className='mr-2 shadow-md'
                    >
                      <IconGift size={16} />
                    </Avatar>
                    <div>
                      <Text className='text-lg font-medium'>
                        {t('基本信息')}
                      </Text>
                      <div className='text-xs text-gray-600'>
                        {t('设置兑换码的基本信息')}
                      </div>
                    </div>
                  </div>

                  <Row gutter={12}>
                    <Col span={24}>
                      <Form.Input
                        field='name'
                        label={t('名称')}
                        placeholder={t('请输入名称')}
                        style={{ width: '100%' }}
                        rules={
                          !isEdit
                            ? []
                            : [{ required: true, message: t('请输入名称') }]
                        }
                        showClear
                      />
                    </Col>
                    <Col span={24}>
                      <Form.DatePicker
                        field='expired_time'
                        label={t('过期时间')}
                        type='dateTime'
                        placeholder={t('选择过期时间（可选，留空为永久）')}
                        style={{ width: '100%' }}
                        showClear
                      />
                    </Col>
                  </Row>
                </Card>

                <Card className='!rounded-2xl shadow-sm border-0'>
                  {/* Header: Quota Settings */}
                  <div className='flex items-center mb-2'>
                    <Avatar
                      size='small'
                      color='green'
                      className='mr-2 shadow-md'
                    >
                      <IconCreditCard size={16} />
                    </Avatar>
                    <div>
                      <Text className='text-lg font-medium'>
                        {t('额度设置')}
                      </Text>
                      <div className='text-xs text-gray-600'>
                        {t('设置兑换码的额度和数量')}
                      </div>
                    </div>
                  </div>

                  <Row gutter={12}>
                    <Col span={12}>
                      {isTokensMode ? (
                        <Form.AutoComplete
                          field='quota'
                          label={t('额度')}
                          placeholder={t('请输入额度')}
                          style={{ width: '100%' }}
                          type='number'
                          inputProps={{ min: 0, step: 1 }}
                          rules={[
                            { required: true, message: t('请输入额度') },
                            {
                              validator: (rule, v) => {
                                const num = parseInt(v, 10);
                                return num > 0
                                  ? Promise.resolve()
                                  : Promise.reject(t('额度必须大于0'));
                              },
                            },
                          ]}
                          extraText={renderQuotaWithPrompt(
                            Number(values.quota) || 0,
                          )}
                          data={quotaPresetOptions}
                          showClear
                        />
                      ) : (
                        <Form.AutoComplete
                          field='quota_amount'
                          label={`${t('额度')} (${getCurrencyConfig().symbol})`}
                          placeholder={t('请输入额度')}
                          style={{ width: '100%' }}
                          type='number'
                          inputProps={{ min: 0, step: 0.01 }}
                          rules={[
                            { required: true, message: t('请输入额度') },
                            {
                              validator: (rule, v) => {
                                const num = Number(v);
                                return num > 0
                                  ? Promise.resolve()
                                  : Promise.reject(t('额度必须大于0'));
                              },
                            },
                          ]}
                          extraText={t('提交时按当前汇率换算为额度保存')}
                          data={quotaPresetOptions}
                          showClear
                        />
                      )}
                    </Col>
                    {!isEdit && (
                      <Col span={12}>
                        <Form.InputNumber
                          field='count'
                          label={t('生成数量')}
                          min={1}
                          rules={[
                            { required: true, message: t('请输入生成数量') },
                            {
                              validator: (rule, v) => {
                                const num = parseInt(v, 10);
                                return num > 0
                                  ? Promise.resolve()
                                  : Promise.reject(t('生成数量必须大于0'));
                              },
                            },
                          ]}
                          style={{ width: '100%' }}
                          showClear
                        />
                      </Col>
                    )}
                  </Row>
                </Card>
              </div>
            )}
          </Form>
        </Spin>
      </SideSheet>
    </>
  );
};

export default EditRedemptionModal;
