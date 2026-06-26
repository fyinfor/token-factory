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
import { API, showError, showSuccess } from '../../../helpers';
import { useTranslation } from 'react-i18next';

const { Text } = Typography;

const DEFAULT_GATEWAY_URL = 'https://open-sea-global.alipay.com';

export default function SettingsPaymentGatewayAntom(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    AntomEnabled: false,
    AntomSandbox: false,
    AntomClientId: '',
    AntomPublicKey: '',
    AntomPrivateKey: '',
    AntomSandboxClientId: '',
    AntomSandboxPublicKey: '',
    AntomSandboxPrivateKey: '',
    AntomGatewayUrl: DEFAULT_GATEWAY_URL,
    AntomNotifyUrl: '',
    AntomReturnUrl: '',
    AntomCurrency: 'USD',
    AntomSettlementCurrency: 'USD',
    AntomUnitPrice: 1.0,
    AntomMinTopUp: 1,
  });
  const formApiRef = useRef(null);

  useEffect(() => {
    if (props.options && formApiRef.current) {
      const currentInputs = {
        AntomEnabled:
          props.options.AntomEnabled === 'true' ||
          props.options.AntomEnabled === true,
        AntomSandbox: props.options.AntomSandbox === 'true',
        AntomClientId: props.options.AntomClientId || '',
        AntomPublicKey: props.options.AntomPublicKey || '',
        AntomPrivateKey: props.options.AntomPrivateKey || '',
        AntomSandboxClientId: props.options.AntomSandboxClientId || '',
        AntomSandboxPublicKey: props.options.AntomSandboxPublicKey || '',
        AntomSandboxPrivateKey: props.options.AntomSandboxPrivateKey || '',
        AntomGatewayUrl: props.options.AntomGatewayUrl || DEFAULT_GATEWAY_URL,
        AntomNotifyUrl: props.options.AntomNotifyUrl || '',
        AntomReturnUrl: props.options.AntomReturnUrl || '',
        AntomCurrency: props.options.AntomCurrency || 'USD',
        AntomSettlementCurrency:
          props.options.AntomSettlementCurrency || 'USD',
        AntomUnitPrice: parseFloat(props.options.AntomUnitPrice) || 1.0,
        AntomMinTopUp: parseInt(props.options.AntomMinTopUp) || 1,
      };
      setInputs(currentInputs);
      formApiRef.current.setValues(currentInputs);
    }
  }, [props.options]);

  const handleFormChange = (values) => {
    setInputs(values);
  };

  const submitAntomSetting = async () => {
    setLoading(true);
    try {
      const options = [
        {
          key: 'AntomEnabled',
          value: inputs.AntomEnabled ? 'true' : 'false',
        },
        {
          key: 'AntomSandbox',
          value: inputs.AntomSandbox ? 'true' : 'false',
        },
        { key: 'AntomClientId', value: inputs.AntomClientId || '' },
        { key: 'AntomPublicKey', value: inputs.AntomPublicKey || '' },
        { key: 'AntomSandboxClientId', value: inputs.AntomSandboxClientId || '' },
        {
          key: 'AntomSandboxPublicKey',
          value: inputs.AntomSandboxPublicKey || '',
        },
        {
          key: 'AntomGatewayUrl',
          value: inputs.AntomGatewayUrl || DEFAULT_GATEWAY_URL,
        },
        { key: 'AntomNotifyUrl', value: inputs.AntomNotifyUrl || '' },
        { key: 'AntomReturnUrl', value: inputs.AntomReturnUrl || '' },
        { key: 'AntomCurrency', value: inputs.AntomCurrency || 'USD' },
        {
          key: 'AntomSettlementCurrency',
          value: inputs.AntomSettlementCurrency || 'USD',
        },
        {
          key: 'AntomUnitPrice',
          value: String(inputs.AntomUnitPrice || 1.0),
        },
        {
          key: 'AntomMinTopUp',
          value: String(inputs.AntomMinTopUp || 1),
        },
      ];

      if (inputs.AntomPrivateKey && inputs.AntomPrivateKey !== '') {
        options.push({ key: 'AntomPrivateKey', value: inputs.AntomPrivateKey });
      }

      if (
        inputs.AntomSandboxPrivateKey &&
        inputs.AntomSandboxPrivateKey !== ''
      ) {
        options.push({
          key: 'AntomSandboxPrivateKey',
          value: inputs.AntomSandboxPrivateKey,
        });
      }

      const requestQueue = options.map((opt) =>
        API.put('/api/option/', {
          key: opt.key,
          value: opt.value,
        }),
      );

      const results = await Promise.all(requestQueue);
      const errorResults = results.filter((res) => !res.data.success);
      if (errorResults.length > 0) {
        errorResults.forEach((res) => {
          showError(res.data.message);
        });
      } else {
        showSuccess(t('更新成功'));
        props.refresh?.();
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
        <Form.Section text={t('Antom 设置')}>
          <Text>
            {t('Antom 是蚂蚁国际全球支付平台，支持 Payment Element 等集成方式。')}
            <a
              href='https://docs.antom.com/ac/cashierpay_zh-cn/quick_start?platform=Web&client=HTML&server=Go&integration_type=Payment+Element'
              target='_blank'
              rel='noreferrer'
            >
              {t('Antom 快速开始文档')}
            </a>
            <br />
          </Text>
          <Banner
            type='info'
            description={t(
              '请在 Antom Dashboard 的 Developer > Quick start > Integration resources 获取 Client ID、Antom 公钥，并在 Key configuration 生成商户私钥。',
            )}
          />

          <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Switch
                field='AntomEnabled'
                label={t('启用 Antom')}
                size='default'
                checkedText='｜'
                uncheckedText='〇'
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Switch
                field='AntomSandbox'
                label={t('沙盒模式')}
                size='default'
                checkedText='｜'
                uncheckedText='〇'
                extraText={t('启用后将使用 Antom 沙盒环境凭证')}
              />
            </Col>
          </Row>

          <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.Input
                field='AntomClientId'
                label={t('Client ID (生产)')}
                placeholder={t('生产环境 Client ID')}
              />
            </Col>
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.Input
                field='AntomSandboxClientId'
                label={t('Client ID (沙盒)')}
                placeholder={t('沙盒环境 Client ID')}
              />
            </Col>
          </Row>

          <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.TextArea
                field='AntomPublicKey'
                label={t('Antom 公钥 (生产)')}
                placeholder={t('用于验签的 Antom 公钥')}
                autosize={{ minRows: 3, maxRows: 6 }}
              />
            </Col>
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.TextArea
                field='AntomSandboxPublicKey'
                label={t('Antom 公钥 (沙盒)')}
                placeholder={t('沙盒环境 Antom 公钥')}
                autosize={{ minRows: 3, maxRows: 6 }}
              />
            </Col>
          </Row>

          <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.TextArea
                field='AntomPrivateKey'
                label={t('商户私钥 (生产)')}
                placeholder={t('用于签名的商户 RSA 私钥')}
                type='password'
                autosize={{ minRows: 3, maxRows: 6 }}
              />
            </Col>
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.TextArea
                field='AntomSandboxPrivateKey'
                label={t('商户私钥 (沙盒)')}
                placeholder={t('沙盒环境商户 RSA 私钥')}
                type='password'
                autosize={{ minRows: 3, maxRows: 6 }}
              />
            </Col>
          </Row>

          <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.Input
                field='AntomGatewayUrl'
                label={t('网关地址')}
                placeholder={DEFAULT_GATEWAY_URL}
                extraText={t(
                  '亚洲区默认 https://open-sea-global.alipay.com，可按 Antom Dashboard 提供的域名修改',
                )}
              />
            </Col>
          </Row>

          <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input field='AntomCurrency' label={t('支付货币')} />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='AntomSettlementCurrency'
                label={t('结算货币')}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.InputNumber
                field='AntomUnitPrice'
                label={t('单价')}
                placeholder='1.0'
                min={0}
                step={0.1}
                extraText={t('每个充值单位对应的支付货币金额，默认 1.0')}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.InputNumber
                field='AntomMinTopUp'
                label={t('最低充值数量')}
                placeholder='1'
                min={1}
                step={1}
                extraText={t('Antom 充值的最低数量，默认 1')}
              />
            </Col>
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.Input
                field='AntomNotifyUrl'
                label={t('支付回调通知地址')}
                placeholder={t('例如 https://example.com/api/antom/webhook')}
                extraText={t(
                  '留空则自动使用 服务器地址 + /api/antom/webhook',
                )}
              />
            </Col>
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.Input
                field='AntomReturnUrl'
                label={t('支付返回地址')}
                placeholder={t('例如 https://example.com/console/topup')}
                extraText={t(
                  '支付完成后用户跳转的页面，留空则自动使用 服务器地址 + /console/topup',
                )}
              />
            </Col>
          </Row>

          <Button onClick={submitAntomSetting} style={{ marginTop: 16 }}>
            {t('更新 Antom 设置')}
          </Button>
        </Form.Section>
      </Form>
    </Spin>
  );
}
