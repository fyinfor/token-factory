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

import React, { useEffect, useState, useRef } from 'react';
import { Button, Form, Row, Col, Typography, Spin } from '@douyinfe/semi-ui';
const { Text } = Typography;
import {
  API,
  removeTrailingSlash,
  showError,
  showSuccess,
  verifyJSON,
} from '../../../helpers';
import { useTranslation } from 'react-i18next';

export default function SettingsPaymentGateway(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    PayAddress: '',
    OnlinePayProvider: 'yipay',
    EpayId: '',
    EpayKey: '',
    YipayAppSecret: '',
    YipayMchNo: '',
    YipayAppId: '',
    YipayWayCode: '',
    YipayNotifyUrl: '',
    YipayReturnUrl: '',
    YipayRequestURL: '',
    Price: 7.3,
    MinTopUp: 1,
    TopupGroupRatio: '',
    CustomCallbackAddress: '',
    PayMethods: '',
    AmountOptions: '',
    AmountDiscount: '',
  });
  const [originInputs, setOriginInputs] = useState({});
  const formApiRef = useRef(null);

  /**
   * 规范化在线支付通道值，兼容历史大小写或别名。
   * @param {string} provider 在线支付通道原始值
   * @returns {string} 规范化后的通道值（epay / yipay）
   */
  const normalizeOnlinePayProvider = (provider) => {
    const value = (provider || '').toLowerCase();
    if (value === 'yipay' || value === 'jeepay') {
      return 'yipay';
    }
    return 'epay';
  };

  /**
   * 当前在线支付通道（规范化后）。
   * @type {'epay' | 'yipay'}
   */
  const currentProvider = normalizeOnlinePayProvider(
    inputs.OnlinePayProvider || 'yipay',
  );

  useEffect(() => {
    if (props.options && formApiRef.current) {
      const currentInputs = {
        PayAddress: props.options.PayAddress || '',
        OnlinePayProvider: normalizeOnlinePayProvider(
          props.options.OnlinePayProvider || 'yipay',
        ),
        EpayId: props.options.EpayId || '',
        EpayKey: props.options.EpayKey || '',
        YipayAppSecret: props.options.YipayAppSecret || '',
        YipayMchNo: props.options.YipayMchNo || '',
        YipayAppId: props.options.YipayAppId || '',
        YipayWayCode: props.options.YipayWayCode || '',
        YipayNotifyUrl: props.options.YipayNotifyUrl || '',
        YipayReturnUrl: props.options.YipayReturnUrl || '',
        YipayRequestURL: props.options.YipayRequestURL || '',
        Price:
          props.options.Price !== undefined
            ? parseFloat(props.options.Price)
            : 7.3,
        MinTopUp:
          props.options.MinTopUp !== undefined
            ? parseFloat(props.options.MinTopUp)
            : 1,
        TopupGroupRatio: props.options.TopupGroupRatio || '',
        CustomCallbackAddress: props.options.CustomCallbackAddress || '',
        PayMethods: props.options.PayMethods || '',
        AmountOptions: props.options.AmountOptions || '',
        AmountDiscount: props.options.AmountDiscount || '',
      };

      // 美化 JSON 展示
      try {
        if (currentInputs.AmountOptions) {
          currentInputs.AmountOptions = JSON.stringify(
            JSON.parse(currentInputs.AmountOptions),
            null,
            2,
          );
        }
      } catch {}
      try {
        if (currentInputs.AmountDiscount) {
          currentInputs.AmountDiscount = JSON.stringify(
            JSON.parse(currentInputs.AmountDiscount),
            null,
            2,
          );
        }
      } catch {}

      setInputs(currentInputs);
      setOriginInputs({ ...currentInputs });
      formApiRef.current.setValues(currentInputs);
    }
  }, [props.options]);

  const handleFormChange = (values) => {
    setInputs((prev) => {
      const next = { ...prev, ...values };
      next.OnlinePayProvider = normalizeOnlinePayProvider(
        next.OnlinePayProvider || 'yipay',
      );
      return next;
    });
  };

  const submitPayAddress = async () => {
    if (props.options.ServerAddress === '') {
      showError(t('请先填写服务器地址'));
      return;
    }

    if (originInputs['TopupGroupRatio'] !== inputs.TopupGroupRatio) {
      if (!verifyJSON(inputs.TopupGroupRatio)) {
        showError(t('充值分组倍率不是合法的 JSON 字符串'));
        return;
      }
    }

    if (originInputs['PayMethods'] !== inputs.PayMethods) {
      if (!verifyJSON(inputs.PayMethods)) {
        showError(t('充值方式设置不是合法的 JSON 字符串'));
        return;
      }
    }

    if (
      originInputs['AmountOptions'] !== inputs.AmountOptions &&
      inputs.AmountOptions.trim() !== ''
    ) {
      if (!verifyJSON(inputs.AmountOptions)) {
        showError(t('自定义充值数量选项不是合法的 JSON 数组'));
        return;
      }
    }

    if (
      originInputs['AmountDiscount'] !== inputs.AmountDiscount &&
      inputs.AmountDiscount.trim() !== ''
    ) {
      if (!verifyJSON(inputs.AmountDiscount)) {
        showError(t('充值金额折扣配置不是合法的 JSON 对象'));
        return;
      }
    }

    if (currentProvider === 'yipay') {
      if (!inputs.YipayMchNo?.trim()) {
        showError(t('Yipay 模式下必须填写商户号'));
        return;
      }
      if (!inputs.YipayAppId?.trim()) {
        showError(t('Yipay 模式下必须填写应用ID（AppId）'));
        return;
      }
      if (!inputs.YipayAppSecret?.trim()) {
        showError(t('Yipay 模式下必须填写 AppSecret'));
        return;
      }
      if (!inputs.YipayWayCode?.trim()) {
        showError(t('Yipay 模式下必须填写支付方式（WayCode）'));
        return;
      }
      if (!inputs.YipayRequestURL?.trim() && !inputs.PayAddress?.trim()) {
        showError(t('Yipay 模式下必须填写请求URL或支付地址'));
        return;
      }
    }

    setLoading(true);
    try {
      const options = [
        { key: 'PayAddress', value: removeTrailingSlash(inputs.PayAddress) },
        {
          key: 'OnlinePayProvider',
          value: normalizeOnlinePayProvider(inputs.OnlinePayProvider || 'yipay'),
        },
      ];

      if (currentProvider === 'epay' && inputs.EpayId !== '') {
        options.push({ key: 'EpayId', value: inputs.EpayId });
      }
      if (currentProvider === 'yipay' && inputs.YipayMchNo !== '') {
        options.push({ key: 'YipayMchNo', value: inputs.YipayMchNo.trim() });
      }
      if (
        currentProvider === 'epay' &&
        inputs.EpayKey !== undefined &&
        inputs.EpayKey !== ''
      ) {
        options.push({ key: 'EpayKey', value: inputs.EpayKey });
      }
      if (currentProvider === 'yipay' && inputs.YipayAppId !== '') {
        options.push({ key: 'YipayAppId', value: inputs.YipayAppId.trim() });
      }
      if (currentProvider === 'yipay' && inputs.YipayWayCode !== '') {
        options.push({ key: 'YipayWayCode', value: inputs.YipayWayCode.trim() });
      }
      if (currentProvider === 'yipay' && inputs.YipayRequestURL !== '') {
        options.push({
          key: 'YipayRequestURL',
          value: removeTrailingSlash(inputs.YipayRequestURL.trim()),
        });
      }
      if (currentProvider === 'yipay' && inputs.YipayNotifyUrl !== '') {
        options.push({ key: 'YipayNotifyUrl', value: inputs.YipayNotifyUrl.trim() });
      }
      if (currentProvider === 'yipay' && inputs.YipayReturnUrl !== '') {
        options.push({ key: 'YipayReturnUrl', value: inputs.YipayReturnUrl.trim() });
      }
      if (
        currentProvider === 'yipay' &&
        inputs.YipayAppSecret !== undefined &&
        inputs.YipayAppSecret !== ''
      ) {
        options.push({ key: 'YipayAppSecret', value: inputs.YipayAppSecret.trim() });
      }
      if (inputs.Price !== '') {
        options.push({ key: 'Price', value: inputs.Price.toString() });
      }
      if (inputs.MinTopUp !== '') {
        options.push({ key: 'MinTopUp', value: inputs.MinTopUp.toString() });
      }
      if (inputs.CustomCallbackAddress !== '') {
        options.push({
          key: 'CustomCallbackAddress',
          value: inputs.CustomCallbackAddress,
        });
      }
      if (originInputs['TopupGroupRatio'] !== inputs.TopupGroupRatio) {
        options.push({ key: 'TopupGroupRatio', value: inputs.TopupGroupRatio });
      }
      if (originInputs['PayMethods'] !== inputs.PayMethods) {
        options.push({ key: 'PayMethods', value: inputs.PayMethods });
      }
      if (originInputs['AmountOptions'] !== inputs.AmountOptions) {
        options.push({
          key: 'payment_setting.amount_options',
          value: inputs.AmountOptions,
        });
      }
      if (originInputs['AmountDiscount'] !== inputs.AmountDiscount) {
        options.push({
          key: 'payment_setting.amount_discount',
          value: inputs.AmountDiscount,
        });
      }

      // 发送请求
      const requestQueue = options.map((opt) =>
        API.put('/api/option/', {
          key: opt.key,
          value: opt.value,
        }),
      );

      const results = await Promise.all(requestQueue);

      // 检查所有请求是否成功
      const errorResults = results.filter((res) => !res.data.success);
      if (errorResults.length > 0) {
        errorResults.forEach((res) => {
          showError(res.data.message);
        });
      } else {
        showSuccess(t('更新成功'));
        // 更新本地存储的原始值
        setOriginInputs({ ...inputs });
        props.refresh && props.refresh();
      }
    } catch (error) {
      showError(t('更新失败'));
    }
    setLoading(false);
  };

  return (
    <Spin spinning={loading}>
      <Form
        initValues={inputs}
        onValueChange={handleFormChange}
        getFormApi={(api) => (formApiRef.current = api)}
      >
        <Form.Section text={t('支付设置')}>
          <Text>
            {t(
              '（当前支持易支付/Yipay 配置，默认使用上方服务器地址作为回调地址！）',
            )}
          </Text>
          <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Select
                field='OnlinePayProvider'
                label={t('在线支付通道')}
                optionList={[
                  { label: t('易支付'), value: 'epay' },
                  { label: t('Yipay'), value: 'yipay' },
                ]}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='PayAddress'
                label={t('支付地址')}
                placeholder={t('例如：https://yourdomain.com')}
              />
            </Col>
            {currentProvider === 'epay' && (
              <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='EpayId'
                label={t('易支付商户号')}
                placeholder={t('例如：0001')}
              />
              </Col>
            )}
            {currentProvider === 'epay' && (
              <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='EpayKey'
                label={t('易支付商户密钥')}
                placeholder={t('敏感信息不会发送到前端显示')}
                type='password'
              />
              </Col>
            )}
          </Row>
          {currentProvider === 'yipay' && (
            <Row
              gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
              style={{ marginTop: 16 }}
            >
              <Col xs={24} sm={24} md={8} lg={8} xl={8}>
                <Form.Input
                  field='YipayMchNo'
                  label={t('Yipay 商户号（可选覆盖）')}
                  placeholder={t('例如：M1691xxxx')}
                />
              </Col>
              <Col xs={24} sm={24} md={8} lg={8} xl={8}>
                <Form.Input
                  field='YipayAppId'
                  label={t('Yipay 应用ID')}
                  placeholder={t('例如：62b2f8f6xxxxxx')}
                />
              </Col>
              <Col xs={24} sm={24} md={8} lg={8} xl={8}>
                <Form.Input
                  field='YipayAppSecret'
                  label={t('Yipay AppSecret')}
                  placeholder={t('用于 MD5 签名的密钥')}
                  type='password'
                />
              </Col>
            </Row>
          )}
          {currentProvider === 'yipay' && (
            <Row
              gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
              style={{ marginTop: 16 }}
            >
              <Col xs={24} sm={24} md={8} lg={8} xl={8}>
                <Form.Input
                  field='YipayWayCode'
                  label={t('Yipay 支付方式')}
                  placeholder={t('例如：WX_JSAPI / ALIPAY_WAP')}
                />
              </Col>
            </Row>
          )}
          <Row
            gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
            style={{ marginTop: 16 }}
          >
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='CustomCallbackAddress'
                label={t('回调地址')}
                placeholder={t('例如：https://yourdomain.com')}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.InputNumber
                field='Price'
                precision={2}
                label={t('充值价格（x元/美金）')}
                placeholder={t('例如：7，就是7元/美金')}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.InputNumber
                field='MinTopUp'
                label={t('最低充值美元数量')}
                placeholder={t('例如：2，就是最低充值2$')}
              />
            </Col>
          </Row>
          {currentProvider === 'yipay' && (
            <Row
              gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
              style={{ marginTop: 16 }}
            >
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='YipayRequestURL'
                label={t('Yipay 请求URL')}
                placeholder={t('例如：https://pay.xxx.com')}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='YipayNotifyUrl'
                label={t('Yipay 异步通知地址')}
                placeholder={t('例如：https://yourdomain.com/api/user/epay/notify')}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='YipayReturnUrl'
                label={t('Yipay 同步通知地址')}
                placeholder={t('例如：https://yourdomain.com/console/log')}
              />
            </Col>
          </Row>
          )}
          <Form.TextArea
            field='TopupGroupRatio'
            label={t('充值分组倍率')}
            placeholder={t('为一个 JSON 文本，键为组名称，值为倍率')}
            autosize
          />
          <Form.TextArea
            field='PayMethods'
            label={t('充值方式设置')}
            placeholder={t('为一个 JSON 文本')}
            autosize
          />

          <Row
            gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
            style={{ marginTop: 16 }}
          >
            <Col span={24}>
              <Form.TextArea
                field='AmountOptions'
                label={t('自定义充值数量选项')}
                placeholder={t(
                  '为一个 JSON 数组，例如：[10, 20, 50, 100, 200, 500]',
                )}
                autosize
                extraText={t(
                  '设置用户可选择的充值数量选项，例如：[10, 20, 50, 100, 200, 500]',
                )}
              />
            </Col>
          </Row>

          <Row
            gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
            style={{ marginTop: 16 }}
          >
            <Col span={24}>
              <Form.TextArea
                field='AmountDiscount'
                label={t('充值金额折扣配置')}
                placeholder={t(
                  '为一个 JSON 对象，例如：{"100": 0.95, "200": 0.9, "500": 0.85}',
                )}
                autosize
                extraText={t(
                  '设置不同充值金额对应的折扣，键为充值金额，值为折扣率，例如：{"100": 0.95, "200": 0.9, "500": 0.85}',
                )}
              />
            </Col>
          </Row>

          <Button onClick={submitPayAddress}>{t('更新支付设置')}</Button>
        </Form.Section>
      </Form>
    </Spin>
  );
}
