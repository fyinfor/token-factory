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
  Table,
  Modal,
  Input,
  InputNumber,
  Space,
} from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from '../../../helpers';
import { useTranslation } from 'react-i18next';

const { Text } = Typography;

export default function SettingsPaymentGatewayUcoin(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    UcoinEnabled: false,
    UcoinBaseUrl: '',
    UcoinMerchantId: '',
    UcoinApiKey: '',
    UcoinMinTopUp: 1,
    UcoinNotifyUrl: '',
  });
  const formApiRef = useRef(null);

  // 币种对列表（主币种 + 子币种编号）
  const [coinPairs, setCoinPairs] = useState([]);
  const [pairModalVisible, setPairModalVisible] = useState(false);
  const [editingPairIndex, setEditingPairIndex] = useState(-1);
  const [pairForm, setPairForm] = useState({
    name: '',
    mainCoinType: '',
    coinType: '',
  });

  useEffect(() => {
    if (props.options && formApiRef.current) {
      const currentInputs = {
        UcoinEnabled:
          props.options.UcoinEnabled === 'true' ||
          props.options.UcoinEnabled === true,
        UcoinBaseUrl: props.options.UcoinBaseUrl || '',
        UcoinMerchantId: props.options.UcoinMerchantId || '',
        UcoinApiKey: props.options.UcoinApiKey || '',
        UcoinMinTopUp: parseInt(props.options.UcoinMinTopUp) || 1,
        UcoinNotifyUrl: props.options.UcoinNotifyUrl || '',
      };
      setInputs(currentInputs);
      formApiRef.current.setValues(currentInputs);

      try {
        const raw = props.options.UcoinCoinPairs;
        if (raw) {
          const parsed = JSON.parse(raw);
          if (Array.isArray(parsed)) {
            setCoinPairs(
              parsed.map((item) => ({
                ...item,
                mainCoinType:
                  item.mainCoinType !== undefined && item.mainCoinType !== null
                    ? item.mainCoinType
                    : '',
                coinType:
                  item.coinType !== undefined && item.coinType !== null
                    ? String(item.coinType)
                    : '',
              })),
            );
          }
        }
      } catch {
        setCoinPairs([]);
      }
    }
  }, [props.options]);

  const handleFormChange = (values) => {
    setInputs(values);
  };

  const submitUcoinSetting = async () => {
    setLoading(true);
    try {
      const options = [];

      options.push({
        key: 'UcoinEnabled',
        value: inputs.UcoinEnabled ? 'true' : 'false',
      });
      options.push({
        key: 'UcoinBaseUrl',
        value: (inputs.UcoinBaseUrl || '').trim(),
      });
      options.push({
        key: 'UcoinMerchantId',
        value: (inputs.UcoinMerchantId || '').trim(),
      });
      // ApiKey 为脱敏回显字段：仅在用户改动（与回显的脱敏值不同）时提交，否则保持原值
      if (
        inputs.UcoinApiKey &&
        inputs.UcoinApiKey.trim() !== (props.options?.UcoinApiKey || '')
      ) {
        options.push({ key: 'UcoinApiKey', value: inputs.UcoinApiKey.trim() });
      }
      options.push({
        key: 'UcoinMinTopUp',
        value: String(inputs.UcoinMinTopUp || 1),
      });
      options.push({
        key: 'UcoinNotifyUrl',
        value: (inputs.UcoinNotifyUrl || '').trim(),
      });
      options.push({
        key: 'UcoinCoinPairs',
        value: JSON.stringify(coinPairs),
      });

      const requestQueue = options.map((opt) =>
        API.put('/api/option/', { key: opt.key, value: opt.value }),
      );
      const results = await Promise.all(requestQueue);
      const errorResults = results.filter((res) => !res.data.success);
      if (errorResults.length > 0) {
        errorResults.forEach((res) => showError(res.data.message));
      } else {
        showSuccess(t('更新成功'));
        props.refresh?.();
      }
    } catch (error) {
      showError(t('更新失败'));
    }
    setLoading(false);
  };

  const openAddPairModal = () => {
    setEditingPairIndex(-1);
    setPairForm({ name: '', mainCoinType: '', coinType: '' });
    setPairModalVisible(true);
  };

  const openEditPairModal = (record, index) => {
    setEditingPairIndex(index);
    setPairForm({
      name: record.name || '',
      mainCoinType:
        record.mainCoinType !== undefined ? String(record.mainCoinType) : '',
      coinType: record.coinType !== undefined ? String(record.coinType) : '',
    });
    setPairModalVisible(true);
  };

  const handlePairModalOk = () => {
    const mainCoinType = parseInt(pairForm.mainCoinType, 10);
    const coinType = (pairForm.coinType || '').trim();
    if (!Number.isFinite(mainCoinType)) {
      showError(t('主币种编号不能为空且必须为数字'));
      return;
    }
    if (!coinType) {
      showError(t('子币种编号不能为空'));
      return;
    }
    const newPair = {
      name: (pairForm.name || '').trim(),
      mainCoinType,
      coinType,
    };
    if (editingPairIndex === -1) {
      setCoinPairs([...coinPairs, newPair]);
    } else {
      const updated = [...coinPairs];
      updated[editingPairIndex] = newPair;
      setCoinPairs(updated);
    }
    setPairModalVisible(false);
  };

  const handleDeletePair = (index) => {
    setCoinPairs(coinPairs.filter((_, i) => i !== index));
  };

  const pairColumns = [
    {
      title: t('展示名称'),
      dataIndex: 'name',
      render: (text) => text || <Text type='tertiary'>—</Text>,
    },
    {
      title: t('主币种编号'),
      dataIndex: 'mainCoinType',
    },
    {
      title: t('子币种编号'),
      dataIndex: 'coinType',
    },
    {
      title: t('操作'),
      key: 'action',
      render: (_, record, index) => (
        <Space>
          <Button size='small' onClick={() => openEditPairModal(record, index)}>
            {t('编辑')}
          </Button>
          <Button
            size='small'
            type='danger'
            onClick={() => handleDeletePair(index)}
          >
            {t('删除')}
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <Spin spinning={loading}>
      <Form
        initValues={inputs}
        onValueChange={handleFormChange}
        getFormApi={(api) => (formApiRef.current = api)}
      >
        <Form.Section text={t('U币支付设置')}>
          <Text>
            {t(
              'U币支付（虚拟币充值）：用户充值时调用接口生成收款地址，用户向该地址转账后由回调入账。',
            )}
            <br />
          </Text>
          <Banner
            type='info'
            description={t(
              '请填写服务商提供的 BaseUrl、商户 ID 与签名密钥（Apikey），并配置至少一组可用币种。',
            )}
          />

          <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Switch
                field='UcoinEnabled'
                label={t('启用 U币支付')}
                size='default'
                checkedText='｜'
                uncheckedText='〇'
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.InputNumber
                field='UcoinMinTopUp'
                label={t('最低充值数量')}
                placeholder='1'
                min={1}
                step={1}
                extraText={t('U币充值的最低数量，默认 1')}
              />
            </Col>
          </Row>

          <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.Input
                field='UcoinBaseUrl'
                label='BaseUrl'
                placeholder='https://api-go.example.com'
                extraText={t('接口根地址，不含 /api/generateAddress 等路径')}
              />
            </Col>
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.Input
                field='UcoinMerchantId'
                label={t('商户 ID (merchantId)')}
                placeholder={t('服务商分配的商户 ID')}
              />
            </Col>
          </Row>

          <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.Input
                field='UcoinApiKey'
                label='Apikey'
                placeholder={t('签名密钥，留空表示不修改')}
                extraText={t(
                  '已保存的密钥以脱敏形式回显；如需修改请直接输入新值，留空或不改动则保持原值',
                )}
              />
            </Col>
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.Input
                field='UcoinNotifyUrl'
                label={t('回调通知地址')}
                placeholder={t('例如 https://example.com/api/user/ubcoin/notify')}
                extraText={t('留空则自动使用 服务器地址 + /api/user/ubcoin/notify')}
              />
            </Col>
          </Row>
        </Form.Section>
      </Form>

      {/* 币种对配置区块 */}
      <div style={{ marginTop: 24 }}>
        <Typography.Title heading={6} style={{ marginBottom: 8 }}>
          {t('可用币种')}
        </Typography.Title>
        <Text type='secondary'>
          {t(
            '配置 U币充值可用的主币种/子币种编号，可添加多组，保存后在充值页面展示给用户。',
          )}
        </Text>
        <div style={{ marginTop: 12, marginBottom: 12 }}>
          <Button onClick={openAddPairModal}>{t('新增币种')}</Button>
        </div>
        <Table
          columns={pairColumns}
          dataSource={coinPairs}
          rowKey={(record, index) => index}
          pagination={false}
          size='small'
          empty={<Text type='tertiary'>{t('暂无币种，点击上方按钮新增')}</Text>}
        />
        <Button onClick={submitUcoinSetting} style={{ marginTop: 16 }}>
          {t('更新 U币支付设置')}
        </Button>
      </div>

      {/* 新增/编辑币种弹窗 */}
      <Modal
        title={editingPairIndex === -1 ? t('新增币种') : t('编辑币种')}
        visible={pairModalVisible}
        onOk={handlePairModalOk}
        onCancel={() => setPairModalVisible(false)}
        okText={t('确定')}
        cancelText={t('取消')}
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          <div>
            <div style={{ marginBottom: 4 }}>
              <Text strong>{t('展示名称')}</Text>
            </div>
            <Input
              value={pairForm.name}
              onChange={(val) => setPairForm({ ...pairForm, name: val })}
              placeholder={t('例如：USDT-TRC20')}
            />
            <Text type='tertiary' size='small'>
              {t('用户在充值页面看到的币种名称，可空')}
            </Text>
          </div>
          <div>
            <div style={{ marginBottom: 4 }}>
              <Text strong>{t('主币种编号 (mainCoinType)')}</Text>
              <span style={{ color: 'var(--semi-color-danger)', marginLeft: 4 }}>
                *
              </span>
            </div>
            <InputNumber
              style={{ width: '100%' }}
              value={pairForm.mainCoinType}
              onChange={(val) =>
                setPairForm({ ...pairForm, mainCoinType: val })
              }
              placeholder='648126'
            />
          </div>
          <div>
            <div style={{ marginBottom: 4 }}>
              <Text strong>{t('子币种编号 (coinType)')}</Text>
              <span style={{ color: 'var(--semi-color-danger)', marginLeft: 4 }}>
                *
              </span>
            </div>
            <Input
              value={pairForm.coinType}
              onChange={(val) => setPairForm({ ...pairForm, coinType: val })}
              placeholder={t('例如：TRC20 合约地址或子币种编号')}
            />
            <Text type='tertiary' size='small'>
              {t('支持字符串，可为链上地址或编号')}
            </Text>
          </div>
        </div>
      </Modal>
    </Spin>
  );
}
