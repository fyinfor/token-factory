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
import {
  Banner,
  Button,
  Form,
  Row,
  Col,
  Typography,
  Spin,
} from '@douyinfe/semi-ui';
const { Text } = Typography;
import {
  API,
  removeTrailingSlash,
  showError,
  showSuccess,
} from '../../../helpers';
import { useTranslation } from 'react-i18next';

export default function SettingsPaymentGatewayAntom(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    AntomClientId: '',
    AntomMerchantPrivateKey: '',
    AntomPublicKey: '',
    AntomGatewayURL: 'https://open-sea-global.alipay.com',
    AntomPayCurrency: 'CNY',
    AntomSettlementCurrency: '',
    AntomPaymentMethods: '',
    AntomMinTopUp: 1,
  });
  const [originInputs, setOriginInputs] = useState({});
  const formApiRef = useRef(null);

  useEffect(() => {
    if (props.options && formApiRef.current) {
      const currentInputs = {
        AntomClientId: props.options.AntomClientId || '',
        AntomMerchantPrivateKey: props.options.AntomMerchantPrivateKey || '',
        AntomPublicKey: props.options.AntomPublicKey || '',
        AntomGatewayURL:
          props.options.AntomGatewayURL ||
          'https://open-sea-global.alipay.com',
        AntomPayCurrency: props.options.AntomPayCurrency || 'CNY',
        AntomSettlementCurrency: props.options.AntomSettlementCurrency || '',
        AntomPaymentMethods: props.options.AntomPaymentMethods || '',
        AntomMinTopUp:
          props.options.AntomMinTopUp !== undefined
            ? parseFloat(props.options.AntomMinTopUp)
            : 1,
      };
      setInputs(currentInputs);
      setOriginInputs({ ...currentInputs });
      formApiRef.current.setValues(currentInputs);
    }
  }, [props.options]);

  const handleFormChange = (values) => {
    setInputs(values);
  };

  const submitAntomSetting = async () => {
    setLoading(true);
    try {
      const options = [];
      if (inputs.AntomClientId) {
        options.push({ key: 'AntomClientId', value: inputs.AntomClientId });
      }
      const originPrivate = originInputs.AntomMerchantPrivateKey || '';
      const originPublic = originInputs.AntomPublicKey || '';
      if (
        inputs.AntomMerchantPrivateKey &&
        inputs.AntomMerchantPrivateKey !== originPrivate
      ) {
        options.push({
          key: 'AntomMerchantPrivateKey',
          value: inputs.AntomMerchantPrivateKey,
        });
      }
      if (inputs.AntomPublicKey && inputs.AntomPublicKey !== originPublic) {
        options.push({ key: 'AntomPublicKey', value: inputs.AntomPublicKey });
      }
      options.push({
        key: 'AntomGatewayURL',
        value: inputs.AntomGatewayURL || '',
      });
      options.push({
        key: 'AntomPayCurrency',
        value: inputs.AntomPayCurrency || 'CNY',
      });
      options.push({
        key: 'AntomSettlementCurrency',
        value: inputs.AntomSettlementCurrency || '',
      });
      options.push({
        key: 'AntomPaymentMethods',
        value: inputs.AntomPaymentMethods || '',
      });
      if (
        inputs.AntomMinTopUp !== undefined &&
        inputs.AntomMinTopUp !== null
      ) {
        options.push({
          key: 'AntomMinTopUp',
          value: inputs.AntomMinTopUp.toString(),
        });
      }

      const results = await Promise.all(
        options.map((opt) =>
          API.put('/api/option/', {
            key: opt.key,
            value: opt.value,
          }),
        ),
      );
      const errorResults = results.filter((res) => !res.data.success);
      if (errorResults.length > 0) {
        errorResults.forEach((res) => {
          showError(res.data.message);
        });
      } else {
        showSuccess(t('更新成功'));
        setOriginInputs({ ...inputs });
        props.refresh?.();
      }
    } catch (error) {
      showError(t('更新失败'));
    }
    setLoading(false);
  };

  const platformOrigin = (() => {
    const fromSetting = removeTrailingSlash(
      props.options.ServerAddress || '',
    );
    const isOpenSite =
      /^https?:\/\/(www\.)?tokenfactoryopen\.com$/i.test(fromSetting);
    if (fromSetting && !isOpenSite) {
      return fromSetting;
    }
    if (typeof window !== 'undefined' && window.location?.origin) {
      return window.location.origin.replace(/\/$/, '');
    }
    return fromSetting || t('网站地址');
  })();
  const notifyUrl = `${platformOrigin}/api/antom/notify`;

  return (
    <Spin spinning={loading}>
      <Form
        initValues={inputs}
        onValueChange={handleFormChange}
        getFormApi={(api) => (formApiRef.current = api)}
      >
        <Form.Section text={t('Antom 收银台')}>
          <Text>
            密钥与支付方式请在
            <a
              href='https://dashboard.alipay.com/global-payments/home'
              target='_blank'
              rel='noreferrer'
            >
              Antom Dashboard
            </a>
            配置。一期使用 Hosted Checkout，用户跳转 Antom 收银台选择支付宝 /
            支付宝HK。
          </Text>
          <Banner
            type='info'
            description={`${t('开发者 → 通知地址 → 只配')} alipay.ams.payments.payNotify：${notifyUrl}`}
          />
          <Banner
            type='warning'
            description={t(
              '地址为本平台域名 + /api/antom/notify（取自基础设置的服务器地址；若仍是 tokenfactoryopen.com 则用当前站点）。私钥与公钥只回显首尾，中间省略；不改动保存不会覆盖。',
            )}
          />
          <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='AntomClientId'
                label={t('Client ID')}
                placeholder='SANDBOX_xxx 或生产 Client-Id'
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='AntomGatewayURL'
                label={t('API 网关')}
                placeholder='https://open-sea-global.alipay.com'
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Select
                field='AntomPayCurrency'
                label={t('支付币种')}
                optionList={[
                  { label: 'CNY', value: 'CNY' },
                  { label: 'USD', value: 'USD' },
                ]}
              />
            </Col>
          </Row>
          <Row
            gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
            style={{ marginTop: 16 }}
          >
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='AntomSettlementCurrency'
                label={t('结算币种（可空）')}
                placeholder={t('默认与支付币种相同')}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='AntomPaymentMethods'
                label={t('收银台支付方式')}
                placeholder={t('留空则按已开通方式展示；不要只填 ALIPAY_CN,ALIPAY_HK')}
                extraText={t('AlipayHK 仅 HKD 订单会出现。Visa 在 Dashboard 变为已开通后，留空即可出现在收银台。卡可写 CARD。')}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.InputNumber
                field='AntomMinTopUp'
                label={t('最低充值数量')}
                placeholder='1'
              />
            </Col>
          </Row>
          <Row
            gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
            style={{ marginTop: 16 }}
          >
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.TextArea
                field='AntomMerchantPrivateKey'
                label={t('商户私钥')}
                placeholder={t('已保存则显示开头...结尾；修改请粘贴完整密钥')}
                autosize={{ minRows: 4, maxRows: 8 }}
              />
            </Col>
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.TextArea
                field='AntomPublicKey'
                label={t('Antom 公钥')}
                placeholder={t('已保存则显示开头...结尾；修改请粘贴完整密钥')}
                autosize={{ minRows: 4, maxRows: 8 }}
              />
            </Col>
          </Row>
          <Button onClick={submitAntomSetting}>{t('更新 Antom 设置')}</Button>
        </Form.Section>
      </Form>
    </Spin>
  );
}
