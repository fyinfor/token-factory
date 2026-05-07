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

import React, { useEffect, useRef, useState } from 'react';
import {
  Banner,
  Button,
  Col,
  Form,
  Popconfirm,
  Row,
  Spin,
  Table,
  Tag,
  Tooltip,
  Typography,
} from '@douyinfe/semi-ui';
import {
  API,
  compareObjects,
  showError,
  showSuccess,
  showWarning,
} from '../../../helpers';
import { useTranslation } from 'react-i18next';

const { Text } = Typography;

function formatBlacklistReason(reason, t) {
  const reasonText = (reason || '').trim();
  const reasonMap = {
    'api-rate-limit:GA': t('API 全局限流触发'),
    'api-rate-limit:CT': t('关键接口限流触发'),
    'api-rate-limit:GW': t('Web 全局限流触发'),
    'user-rate-limit': t('用户接口限流触发'),
    'model-total-rate-limit': t('模型请求总次数限流触发'),
    'model-success-rate-limit': t('模型请求成功次数限流触发'),
  };
  return reasonMap[reasonText] || reasonText || '-';
}

export default function SettingsApiRateLimit(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    GlobalApiRateLimitEnable: true,
    GlobalApiRateLimitNum: 180,
    GlobalApiRateLimitDuration: 180,
    CriticalRateLimitEnable: true,
    CriticalRateLimitNum: 20,
    CriticalRateLimitDuration: 1200,
    RateLimitUserWhitelist: '[]',
  });
  const [inputsRow, setInputsRow] = useState(inputs);
  const [blacklistLoading, setBlacklistLoading] = useState(false);
  const [blacklistUsers, setBlacklistUsers] = useState([]);
  const refForm = useRef();

  function handleFieldChange(fieldName) {
    return (value) => {
      setInputs((prev) => ({ ...prev, [fieldName]: value }));
    };
  }

  function onSubmit() {
    try {
      const parsed = JSON.parse(inputs.RateLimitUserWhitelist || '[]');
      if (!Array.isArray(parsed)) {
        return showError(t('白名单必须是用户ID数组，例如 [1,2,3]'));
      }
      if (parsed.some((v) => !Number.isInteger(v) || v <= 0)) {
        return showError(t('白名单中的用户ID必须为正整数'));
      }
    } catch {
      return showError(t('白名单不是合法的 JSON 数组'));
    }

    const updateArray = compareObjects(inputs, inputsRow);
    if (!updateArray.length) return showWarning(t('你似乎并没有修改什么'));

    const requestQueue = updateArray.map((item) => {
      let value = '';
      if (typeof inputs[item.key] === 'boolean') {
        value = String(inputs[item.key]);
      } else if (item.key === 'RateLimitUserWhitelist') {
        value = String(inputs[item.key]).trim();
      } else {
        value = String(Number(inputs[item.key]));
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
        for (let i = 0; i < res.length; i++) {
          if (!res[i].data.success) {
            return showError(res[i].data.message);
          }
        }
        showSuccess(t('保存成功，限流配置已即时生效'));
        props.refresh();
        fetchBlacklistUsers();
      })
      .catch(() => {
        showError(t('保存失败，请重试'));
      })
      .finally(() => {
        setLoading(false);
      });
  }

  async function fetchBlacklistUsers() {
    setBlacklistLoading(true);
    try {
      const res = await API.get('/api/option/rate_limit_blacklist_users');
      if (res.data.success) {
        setBlacklistUsers(res.data.data || []);
      }
    } catch (error) {
      showError(t('获取临时黑名单失败'));
    } finally {
      setBlacklistLoading(false);
    }
  }

  async function removeBlacklistUser(userId) {
    try {
      const res = await API.delete('/api/option/rate_limit_blacklist_users', {
        data: { user_id: userId },
      });
      if (!res.data.success) {
        return showError(res.data.message);
      }
      showSuccess(t('已移除该用户的临时黑名单'));
      fetchBlacklistUsers();
    } catch (error) {
      showError(t('移除失败，请重试'));
    }
  }

  useEffect(() => {
    const currentInputs = {};
    for (const key in props.options) {
      if (Object.prototype.hasOwnProperty.call(inputs, key)) {
        currentInputs[key] = props.options[key];
      }
    }
    const nextInputs = { ...inputs, ...currentInputs };
    setInputs(nextInputs);
    setInputsRow(structuredClone(nextInputs));
    if (refForm.current) {
      refForm.current.setValues(nextInputs);
    }
    fetchBlacklistUsers();
  }, [props.options]);

  const blacklistColumns = [
    {
      title: t('用户ID'),
      dataIndex: 'user_id',
      render: (value) => <Text>{value}</Text>,
    },
    {
      title: t('剩余时间'),
      dataIndex: 'ttl_seconds',
      render: (value) => <Tag color='orange'>{value}s</Tag>,
    },
    {
      title: t('触发原因'),
      dataIndex: 'reason',
      render: (value) => (
        <Text type='tertiary'>{formatBlacklistReason(value, t)}</Text>
      ),
    },
    {
      title: t('操作'),
      dataIndex: 'action',
      render: (_, record) => (
        <Popconfirm
          title={t('确认移除该用户临时黑名单？')}
          onConfirm={() => removeBlacklistUser(record.user_id)}
        >
          <Button type='danger' size='small'>
            {t('移除')}
          </Button>
        </Popconfirm>
      ),
    },
  ];

  return (
    <Spin spinning={loading}>
      <Form
        values={inputs}
        getFormApi={(formAPI) => (refForm.current = formAPI)}
        style={{ marginBottom: 15 }}
      >
        <Form.Section text={t('接口限流设置')}>
          <Banner
            type='info'
            description={t(
              '本页面用于配置已登录 API 的全局限流与关键接口限流。保存后立即生效（无需重启）。管理员默认在白名单内，不受限流影响。',
            )}
            style={{ marginBottom: 16 }}
          />

          <Row gutter={16}>
            <Col xs={24} sm={12} md={8}>
              <Form.Switch
                field={'GlobalApiRateLimitEnable'}
                label={
                  <span
                    style={{
                      display: 'inline-flex',
                      alignItems: 'center',
                      gap: 6,
                    }}
                  >
                    {t('启用 API 全局限流')}
                    <Tooltip
                      content={t(
                        '控制已登录用户调用 /api 时的整体访问频率。开启后，会按用户ID在指定时间窗口内限制请求次数。',
                      )}
                    >
                      <span
                        style={{
                          display: 'inline-flex',
                          width: 16,
                          height: 16,
                          borderRadius: '50%',
                          border: '1px solid var(--semi-color-border)',
                          alignItems: 'center',
                          justifyContent: 'center',
                          fontSize: 12,
                          cursor: 'help',
                          userSelect: 'none',
                        }}
                      >
                        ?
                      </span>
                    </Tooltip>
                  </span>
                }
                size='default'
                checkedText='｜'
                uncheckedText='〇'
                onChange={handleFieldChange('GlobalApiRateLimitEnable')}
              />
            </Col>
            <Col xs={24} sm={12} md={8}>
              <Form.InputNumber
                field={'GlobalApiRateLimitNum'}
                label={t('API 限制次数')}
                min={1}
                max={100000000}
                suffix={t('次')}
                onChange={handleFieldChange('GlobalApiRateLimitNum')}
                disabled={!inputs.GlobalApiRateLimitEnable}
              />
            </Col>
            <Col xs={24} sm={12} md={8}>
              <Form.InputNumber
                field={'GlobalApiRateLimitDuration'}
                label={t('API 时间窗口')}
                min={1}
                max={86400}
                suffix={t('秒')}
                onChange={handleFieldChange('GlobalApiRateLimitDuration')}
                disabled={!inputs.GlobalApiRateLimitEnable}
              />
            </Col>
          </Row>

          <Row gutter={16}>
            <Col xs={24} sm={12} md={8}>
              <Form.Switch
                field={'CriticalRateLimitEnable'}
                label={
                  <span
                    style={{
                      display: 'inline-flex',
                      alignItems: 'center',
                      gap: 6,
                    }}
                  >
                    {t('启用关键接口限流')}
                    <Tooltip
                      content={t(
                        '用于保护登录、注册、支付、验证码等敏感接口。开启后，这类接口会使用更严格的访问频率限制。',
                      )}
                    >
                      <span
                        style={{
                          display: 'inline-flex',
                          width: 16,
                          height: 16,
                          borderRadius: '50%',
                          border: '1px solid var(--semi-color-border)',
                          alignItems: 'center',
                          justifyContent: 'center',
                          fontSize: 12,
                          cursor: 'help',
                          userSelect: 'none',
                        }}
                      >
                        ?
                      </span>
                    </Tooltip>
                  </span>
                }
                size='default'
                checkedText='｜'
                uncheckedText='〇'
                onChange={handleFieldChange('CriticalRateLimitEnable')}
              />
            </Col>
            <Col xs={24} sm={12} md={8}>
              <Form.InputNumber
                field={'CriticalRateLimitNum'}
                label={t('关键接口限制次数')}
                min={1}
                max={100000000}
                suffix={t('次')}
                onChange={handleFieldChange('CriticalRateLimitNum')}
                disabled={!inputs.CriticalRateLimitEnable}
              />
            </Col>
            <Col xs={24} sm={12} md={8}>
              <Form.InputNumber
                field={'CriticalRateLimitDuration'}
                label={t('关键接口时间窗口')}
                min={1}
                max={86400}
                suffix={t('秒')}
                onChange={handleFieldChange('CriticalRateLimitDuration')}
                disabled={!inputs.CriticalRateLimitEnable}
              />
            </Col>
          </Row>

          <Row>
            <Col span={24}>
              <Form.TextArea
                field={'RateLimitUserWhitelist'}
                label={t('限流白名单（用户ID）')}
                placeholder={'[1, 2, 3]'}
                autosize={{ minRows: 4, maxRows: 12 }}
                extraText={t(
                  '填写 JSON 数组。名单内用户将跳过用户维度限流；管理员默认放行，无需手动填写。',
                )}
                onChange={handleFieldChange('RateLimitUserWhitelist')}
              />
            </Col>
          </Row>

          <Row>
            <Button size='default' onClick={onSubmit}>
              {t('保存接口限流设置')}
            </Button>
          </Row>
        </Form.Section>

        <Form.Section text={t('临时黑名单')}>
          <Banner
            type='warning'
            description={t(
              '已触发限流的用户，属于临时状态，会随 TTL 自动过期。可手动移除立即恢复。',
            )}
            style={{ marginBottom: 12 }}
          />
          <div style={{ marginBottom: 12 }}>
            <Button onClick={fetchBlacklistUsers} loading={blacklistLoading}>
              {t('刷新临时黑名单')}
            </Button>
          </div>
          <Table
            columns={blacklistColumns}
            dataSource={blacklistUsers}
            rowKey={'user_id'}
            pagination={false}
            size='small'
            empty={<Text type='tertiary'>{t('当前没有临时黑名单用户')}</Text>}
          />
        </Form.Section>
      </Form>
    </Spin>
  );
}
