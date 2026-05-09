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

import React, { useCallback, useLayoutEffect, useRef, useState } from 'react';
import { Button, Form, Row, Col, Typography, Spin, TextArea } from '@douyinfe/semi-ui';
const { Text } = Typography;
import {
  API,
  removeTrailingSlash,
  showError,
  showSuccess,
  verifyJSON,
} from '../../../helpers';
import { useTranslation } from 'react-i18next';

/**
 * 生成与后端 buildUserEpayNotifyURL(service.GetCallbackAddress()) 一致的默认异步通知地址说明。
 * @param {string} serverAddress 系统「服务器地址」
 * @param {string} customCallback 自定义回调根地址（空则用服务器地址）
 * @returns {string}
 */
function getDefaultYipayNotifyHint(serverAddress, customCallback) {
  const base = String(customCallback || serverAddress || '')
    .trim()
    .replace(/\/$/, '');
  if (!base) {
    return '';
  }
  return `${base}/api/user/epay/notify`;
}

/**
 * 同步跳转地址默认值（与后端 system_setting.ServerAddress + "/console/log" 一致）。
 * @param {string} serverAddress 系统「服务器地址」
 * @returns {string}
 */
function getDefaultYipayReturnHint(serverAddress) {
  const base = String(serverAddress || '')
    .trim()
    .replace(/\/$/, '');
  if (!base) {
    return '';
  }
  return `${base}/console/log`;
}

/**
 * 规范化在线支付通道值，兼容历史大小写或别名。
 * @param {string} provider 在线支付通道原始值
 * @returns {'epay' | 'yipay'} 规范化后的通道值
 */
function normalizeOnlinePayProvider(provider) {
  const value = (provider || '').toLowerCase();
  if (value === 'yipay' || value === 'jeepay') {
    return 'yipay';
  }
  return 'epay';
}

/**
 * 从父组件拉取的 options 构建表单模型（含 Yipay 网关地址与 JSON 字段格式化）。
 * @param {Record<string, unknown>} opts PaymentSetting 聚合后的 option
 * @returns {Record<string, unknown>} 供 Semi Form 使用的扁平值
 */
function buildFormModelFromOptions(opts) {
  const prov = normalizeOnlinePayProvider(opts.OnlinePayProvider || 'yipay');
  let yipayReq = opts.YipayRequestURL || '';
  if (prov === 'yipay' && !String(yipayReq).trim() && opts.PayAddress) {
    yipayReq = opts.PayAddress;
  }
  const model = {
    PayAddress: opts.PayAddress || '',
    OnlinePayProvider: prov,
    EpayId: opts.EpayId || '',
    EpayKey: opts.EpayKey || '',
    YipayAppSecret: String(opts.YipayAppSecret ?? ''),
    YipayMchNo: opts.YipayMchNo || '',
    YipayAppId: opts.YipayAppId || '',
    YipayNotifyUrl: opts.YipayNotifyUrl || '',
    YipayReturnUrl: opts.YipayReturnUrl || '',
    YipayChannelExtra: String(opts.YipayChannelExtra ?? ''),
    YipayRequestURL: yipayReq,
    Price: opts.Price !== undefined ? parseFloat(String(opts.Price)) : 7.3,
    MinTopUp:
      opts.MinTopUp !== undefined ? parseFloat(String(opts.MinTopUp)) : 1,
    TopupGroupRatio: opts.TopupGroupRatio || '',
    CustomCallbackAddress: opts.CustomCallbackAddress || '',
    PayMethods: opts.PayMethods || '',
    AmountOptions: opts.AmountOptions || '',
    AmountDiscount: opts.AmountDiscount || '',
  };
  try {
    if (model.AmountOptions) {
      model.AmountOptions = JSON.stringify(
        JSON.parse(String(model.AmountOptions)),
        null,
        2,
      );
    }
  } catch {
    /* keep raw */
  }
  try {
    if (model.AmountDiscount) {
      model.AmountDiscount = JSON.stringify(
        JSON.parse(String(model.AmountDiscount)),
        null,
        2,
      );
    }
  } catch {
    /* keep raw */
  }
  return model;
}

/**
 * 管理后台「易支付 / Yipay」网关配置表单。
 * @param {{ options: Record<string, unknown>, refresh?: () => void }} props
 */
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
    YipayNotifyUrl: '',
    YipayReturnUrl: '',
    YipayChannelExtra: '',
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
  /** Semi Form API 引用；在 getFormApi 回调内与应用 options 同步，避免早于 Form 挂载的 effect 拿不到实例 */
  const formApiRef = useRef(null);

  /**
   * 当前在线支付通道（规范化后）。
   * @type {'epay' | 'yipay'}
   */
  const currentProvider = normalizeOnlinePayProvider(
    inputs.OnlinePayProvider || 'yipay',
  );

  /**
   * 将服务端 options 同步到本地 state，并写入 Semi Form。
   * YipayChannelExtra 由独立 TextArea 绑定 inputs，不写入 Form（避免放在 display:none 的 Form Field 内导致无法编辑）。
   * @param {Record<string, unknown>} opts
   */
  const flushOptionsIntoForm = useCallback((opts) => {
    if (!opts) {
      return;
    }
    const currentInputs = buildFormModelFromOptions(opts);
    setInputs(currentInputs);
    setOriginInputs({ ...currentInputs });
    const api = formApiRef.current;
    if (!api) {
      return;
    }
    const { YipayChannelExtra: _channelExtraOmit, ...formRest } = currentInputs;
    void _channelExtraOmit;
    api.setValues(formRest, { isOverride: true });
    api.setValue('YipayAppSecret', currentInputs.YipayAppSecret);
  }, []);

  useLayoutEffect(() => {
    flushOptionsIntoForm(props.options);
  }, [props.options, flushOptionsIntoForm]);

  /**
   * Form 挂载即拿到 API 时立刻灌入当前 options（解决首屏 effect 早于 getFormApi 的问题）。
   * @param {{ setValues: Function, setValue: Function }} api Semi FormApi
   */
  const handleGetFormApi = useCallback(
    (api) => {
      formApiRef.current = api;
      flushOptionsIntoForm(props.options);
    },
    [props.options, flushOptionsIntoForm],
  );

  /**
   * 同步 Yipay channelExtra 到本地状态（独立于 Semi Form）。
   * @param {string} val 文本框当前值
   */
  const handleYipayChannelExtraChange = (val) => {
    setInputs((prev) => ({ ...prev, YipayChannelExtra: val }));
  };

  /**
   * 表单变更时合并状态（仅合入非 undefined 字段，避免 Semi 在卸载表单项时用 undefined 冲掉内存中的值）。
   * 从易支付切换到 Yipay 时，若请求地址为空则用原支付地址预填。
   * @param {Record<string, unknown>} formValues Semi Form 当前变更后的表单快照
   */
  const handleFormChange = (formValues) => {
    setInputs((prev) => {
      const next = { ...prev };
      Object.keys(formValues).forEach((key) => {
        const v = formValues[key];
        if (v !== undefined) {
          next[key] = v;
        }
      });
      const prevProv = normalizeOnlinePayProvider(prev.OnlinePayProvider);
      next.OnlinePayProvider = normalizeOnlinePayProvider(
        next.OnlinePayProvider || 'yipay',
      );
      const nextProv = next.OnlinePayProvider;
      if (
        prevProv !== nextProv &&
        nextProv === 'yipay' &&
        !(next.YipayRequestURL || '').trim() &&
        (next.PayAddress || '').trim()
      ) {
        next.YipayRequestURL = next.PayAddress;
      }
      return next;
    });
  };

  /**
   * 校验并批量提交支付网关相关 option 项。
   */
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
      const secretTrim = (inputs.YipayAppSecret || '').trim();
      const secretUnchanged =
        originInputs.YipayAppSecret &&
        inputs.YipayAppSecret === originInputs.YipayAppSecret;
      if (!secretTrim && !secretUnchanged) {
        showError(t('Yipay 模式下必须填写 AppSecret'));
        return;
      }
      if (!inputs.YipayRequestURL?.trim()) {
        showError(t('Yipay 模式下必须填写支付地址（网关根 URL）'));
        return;
      }
      const chExtra = (inputs.YipayChannelExtra || '').trim();
      if (chExtra && !verifyJSON(chExtra)) {
        showError(t('Yipay 渠道扩展参数须为合法 JSON（Jeepay channelExtra）'));
        return;
      }
    }

    setLoading(true);
    try {
      const options = [
        {
          key: 'OnlinePayProvider',
          value: normalizeOnlinePayProvider(
            inputs.OnlinePayProvider || 'yipay',
          ),
        },
      ];

      if (currentProvider === 'epay') {
        options.push({
          key: 'PayAddress',
          value: removeTrailingSlash(inputs.PayAddress || ''),
        });
      }
      if (currentProvider === 'yipay') {
        const gatewayBase = removeTrailingSlash(
          (inputs.YipayRequestURL || '').trim(),
        );
        options.push({ key: 'YipayRequestURL', value: gatewayBase });
        options.push({ key: 'PayAddress', value: gatewayBase });
      }

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
      if (currentProvider === 'yipay' && inputs.YipayNotifyUrl !== '') {
        options.push({
          key: 'YipayNotifyUrl',
          value: inputs.YipayNotifyUrl.trim(),
        });
      }
      if (currentProvider === 'yipay' && inputs.YipayReturnUrl !== '') {
        options.push({
          key: 'YipayReturnUrl',
          value: inputs.YipayReturnUrl.trim(),
        });
      }
      if (
        currentProvider === 'yipay' &&
        (inputs.YipayChannelExtra ?? '').trim() !==
          (originInputs.YipayChannelExtra ?? '').trim()
      ) {
        options.push({
          key: 'YipayChannelExtra',
          value: (inputs.YipayChannelExtra ?? '').trim(),
        });
      }
      if (
        currentProvider === 'yipay' &&
        inputs.YipayAppSecret !== undefined &&
        inputs.YipayAppSecret.trim() !== '' &&
        inputs.YipayAppSecret !== originInputs.YipayAppSecret
      ) {
        options.push({
          key: 'YipayAppSecret',
          value: inputs.YipayAppSecret.trim(),
        });
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
        initValues={{}}
        onValueChange={handleFormChange}
        getFormApi={handleGetFormApi}
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
            <Col
              xs={24}
              sm={24}
              md={8}
              lg={8}
              xl={8}
              style={{
                display: currentProvider === 'epay' ? 'block' : 'none',
              }}
            >
              <Form.Input
                field='PayAddress'
                label={t('支付地址')}
                placeholder={t('例如：https://yourdomain.com')}
              />
            </Col>
            <Col
              xs={24}
              sm={24}
              md={8}
              lg={8}
              xl={8}
              style={{
                display: currentProvider === 'yipay' ? 'block' : 'none',
              }}
            >
              <Form.Input
                field='YipayRequestURL'
                label={t('支付地址')}
                placeholder={t(
                  '例如：https://pay.xxx.com（与易支付网关根地址相同）',
                )}
                extraText={t(
                  '对应配置项 YipayRequestURL；保存时会同步写入 PayAddress，供服务端与易支付字段兼容。',
                )}
              />
            </Col>
            <Col
              xs={24}
              sm={24}
              md={8}
              lg={8}
              xl={8}
              style={{
                display: currentProvider === 'epay' ? 'block' : 'none',
              }}
            >
              <Form.Input
                field='EpayId'
                label={t('易支付商户号')}
                placeholder={t('例如：0001')}
              />
            </Col>
            <Col
              xs={24}
              sm={24}
              md={8}
              lg={8}
              xl={8}
              style={{
                display: currentProvider === 'epay' ? 'block' : 'none',
              }}
            >
              <Form.Input
                field='EpayKey'
                label={t('易支付商户密钥')}
                placeholder={t('敏感信息不会发送到前端显示')}
                type='password'
              />
            </Col>
          </Row>
          <Row
            gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
            style={{
              marginTop: 16,
              display: currentProvider === 'yipay' ? 'flex' : 'none',
            }}
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
                placeholder={t(
                  '用于 MD5 签名的密钥（后端脱敏回显；修改后保存才会更新）',
                )}
                autoComplete='off'
              />
            </Col>
          </Row>
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
          <Row
            gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
            style={{
              marginTop: 16,
              display: currentProvider === 'yipay' ? 'flex' : 'none',
            }}
          >
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='YipayNotifyUrl'
                label={t('Yipay 异步通知地址')}
                placeholder={t(
                  '可留空，留空则使用下方「回调地址」或服务器地址拼接默认路径',
                )}
                extraText={(() => {
                  const hint = getDefaultYipayNotifyHint(
                    props.options.ServerAddress,
                    props.options.CustomCallbackAddress,
                  );
                  return hint
                    ? `默认：${hint}（留空时与后端异步通知规则一致）`
                    : '请先配置「服务器地址」或「回调地址」以预览默认异步通知 URL';
                })()}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='YipayReturnUrl'
                label={t('Yipay 同步通知地址')}
                placeholder={t('例如：https://yourdomain.com/console/log')}
                extraText={(() => {
                  const hint = getDefaultYipayReturnHint(
                    props.options.ServerAddress,
                  );
                  return hint
                    ? `默认：${hint}（留空时使用）`
                    : '请先配置「服务器地址」以预览默认同步跳转 URL';
                })()}
              />
            </Col>
          </Row>
          <Row
            gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
            style={{
              marginTop: 8,
              display: currentProvider === 'yipay' ? 'flex' : 'none',
            }}
          >
            <Col span={24}>
              <div style={{ marginBottom: 8 }}>
                <Text strong>{t('Yipay 渠道扩展参数（channelExtra）')}</Text>
              </div>
              <TextArea
                value={inputs.YipayChannelExtra}
                onChange={handleYipayChannelExtraChange}
                placeholder={'{"payDataType":"payUrl"}'}
                rows={1}
                autoComplete='off'
                style={{
                  width: '100%',
                  fontFamily: 'monospace',
                  fontSize: 12,
                }}
              />
              <Text
                type='tertiary'
                size='small'
                style={{ marginTop: 8, display: 'block' }}
              >
                {t(
                  '可选。Jeepay channelExtra（JSON）。留空时：一般 _PC 会附带 payDataType=payUrl；PP_PC 会附带 payDataType 与 cancelUrl（与 returnUrl 同源，见 yiPay PPPcOrderRQ）。可在此覆盖 cancelUrl 等。',
                )}
              </Text>
            </Col>
          </Row>
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
