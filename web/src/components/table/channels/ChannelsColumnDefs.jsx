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

import React from 'react';
import {
  Button,
  Dropdown,
  InputNumber,
  Modal,
  Space,
  SplitButtonGroup,
  Tag,
  Tooltip,
  Typography,
} from '@douyinfe/semi-ui';
import {
  timestamp2string,
  renderGroup,
  renderQuota,
  getChannelIcon,
  renderQuotaWithAmount,
  showSuccess,
  showError,
  showInfo,
} from '../../../helpers';
import {
  CHANNEL_OPTIONS,
  MODEL_FETCHABLE_CHANNEL_TYPES,
} from '../../../constants';
import { parseUpstreamUpdateMeta } from '../../../hooks/channels/upstreamUpdateUtils';
import {
  IconTreeTriangleDown,
  IconMore,
  IconAlertTriangle,
} from '@douyinfe/semi-icons';
import { FaRandom } from 'react-icons/fa';

// Render functions
const renderType = (type, record = {}, t) => {
  const channelInfo = record?.channel_info;
  let type2label = new Map();
  for (let i = 0; i < CHANNEL_OPTIONS.length; i++) {
    type2label[CHANNEL_OPTIONS[i].value] = CHANNEL_OPTIONS[i];
  }
  type2label[0] = { value: 0, label: t('未知类型'), color: 'grey' };

  let icon = getChannelIcon(type);

  if (channelInfo?.is_multi_key) {
    icon =
      channelInfo?.multi_key_mode === 'random' ? (
        <div className='flex items-center gap-1'>
          <FaRandom className='text-blue-500' />
          {icon}
        </div>
      ) : (
        <div className='flex items-center gap-1'>
          <IconTreeTriangleDown className='text-blue-500' />
          {icon}
        </div>
      );
  }

  const typeTag = (
    <Tag color={type2label[type]?.color} shape='circle' prefixIcon={icon}>
      {type2label[type]?.label}
    </Tag>
  );

  let ionetMeta = null;
  if (record?.other_info) {
    try {
      const parsed = JSON.parse(record.other_info);
      if (parsed && typeof parsed === 'object' && parsed.source === 'ionet') {
        ionetMeta = parsed;
      }
    } catch (error) {
      // ignore invalid metadata
    }
  }

  if (!ionetMeta) {
    return typeTag;
  }

  const handleNavigate = (event) => {
    event?.stopPropagation?.();
    if (!ionetMeta?.deployment_id) {
      return;
    }
    const targetUrl = `/console/deployment?deployment_id=${ionetMeta.deployment_id}`;
    window.open(targetUrl, '_blank', 'noopener');
  };

  return (
    <Space spacing={6}>
      {typeTag}
      <Tooltip
        content={
          <div className='max-w-xs'>
            <div className='text-xs text-gray-600'>
              {t('来源于 IO.NET 部署')}
            </div>
            {ionetMeta?.deployment_id && (
              <div className='text-xs text-gray-500 mt-1'>
                {t('部署 ID')}: {ionetMeta.deployment_id}
              </div>
            )}
          </div>
        }
      >
        <span>
          <Tag
            color='purple'
            type='light'
            className='cursor-pointer'
            onClick={handleNavigate}
          >
            IO.NET
          </Tag>
        </span>
      </Tooltip>
    </Space>
  );
};

const renderTagType = (t) => {
  return (
    <Tag color='light-blue' shape='circle' type='light'>
      {t('标签聚合')}
    </Tag>
  );
};

const renderStatus = (status, channelInfo = undefined, t) => {
  if (channelInfo) {
    if (channelInfo.is_multi_key) {
      let keySize = channelInfo.multi_key_size;
      let enabledKeySize = keySize;
      if (channelInfo.multi_key_status_list) {
        enabledKeySize =
          keySize - Object.keys(channelInfo.multi_key_status_list).length;
      }
      return renderMultiKeyStatus(status, keySize, enabledKeySize, t);
    }
  }
  switch (status) {
    case 1:
      return (
        <Tag color='green' shape='circle'>
          {t('已启用')}
        </Tag>
      );
    case 2:
      return (
        <Tag color='red' shape='circle'>
          {t('已禁用')}
        </Tag>
      );
    case 3:
      return (
        <Tag color='yellow' shape='circle'>
          {t('自动禁用')}
        </Tag>
      );
    default:
      return (
        <Tag color='grey' shape='circle'>
          {t('未知状态')}
        </Tag>
      );
  }
};

const renderMultiKeyStatus = (status, keySize, enabledKeySize, t) => {
  switch (status) {
    case 1:
      return (
        <Tag color='green' shape='circle'>
          {t('已启用')} {enabledKeySize}/{keySize}
        </Tag>
      );
    case 2:
      return (
        <Tag color='red' shape='circle'>
          {t('已禁用')} {enabledKeySize}/{keySize}
        </Tag>
      );
    case 3:
      return (
        <Tag color='yellow' shape='circle'>
          {t('自动禁用')} {enabledKeySize}/{keySize}
        </Tag>
      );
    default:
      return (
        <Tag color='grey' shape='circle'>
          {t('未知状态')} {enabledKeySize}/{keySize}
        </Tag>
      );
  }
};

const renderResponseTime = (responseTime, t) => {
  let time = responseTime / 1000;
  time = time.toFixed(2) + t(' 秒');
  if (responseTime === 0) {
    return (
      <Tag color='grey' shape='circle'>
        {t('未测试')}
      </Tag>
    );
  } else if (responseTime <= 1000) {
    return (
      <Tag color='green' shape='circle'>
        {time}
      </Tag>
    );
  } else if (responseTime <= 3000) {
    return (
      <Tag color='lime' shape='circle'>
        {time}
      </Tag>
    );
  } else if (responseTime <= 5000) {
    return (
      <Tag color='yellow' shape='circle'>
        {time}
      </Tag>
    );
  } else {
    return (
      <Tag color='red' shape='circle'>
        {time}
      </Tag>
    );
  }
};

const isRequestPassThroughEnabled = (record) => {
  if (!record || record.children !== undefined) {
    return false;
  }
  const settingValue = record.setting;
  if (!settingValue) {
    return false;
  }
  if (typeof settingValue === 'object') {
    return settingValue.pass_through_body_enabled === true;
  }
  if (typeof settingValue !== 'string') {
    return false;
  }
  try {
    const parsed = JSON.parse(settingValue);
    return parsed?.pass_through_body_enabled === true;
  } catch (error) {
    return false;
  }
};

const getUpstreamUpdateMeta = (record) => {
  const supported =
    !!record &&
    record.children === undefined &&
    MODEL_FETCHABLE_CHANNEL_TYPES.has(record.type);
  if (!record || record.children !== undefined) {
    return {
      supported: false,
      enabled: false,
      pendingAddModels: [],
      pendingRemoveModels: [],
    };
  }
  const parsed =
    record?.upstreamUpdateMeta && typeof record.upstreamUpdateMeta === 'object'
      ? record.upstreamUpdateMeta
      : parseUpstreamUpdateMeta(record?.settings);
  return {
    supported,
    enabled: parsed?.enabled === true,
    pendingAddModels: Array.isArray(parsed?.pendingAddModels)
      ? parsed.pendingAddModels
      : [],
    pendingRemoveModels: Array.isArray(parsed?.pendingRemoveModels)
      ? parsed.pendingRemoveModels
      : [],
  };
};

export const getChannelsColumns = ({
  t,
  COLUMN_KEYS,
  updateChannelBalance,
  setChannelBalanceManually,
  manageChannel,
  manageTag,
  submitTagEdit,
  testChannel,
  setCurrentTestChannel,
  setShowModelTestModal,
  setEditingChannel,
  setShowEdit,
  setShowEditTag,
  setEditingTag,
  copySelectedChannel,
  refresh,
  activePage,
  channels,
  checkOllamaVersion,
  setShowMultiKeyManageModal,
  setCurrentMultiKeyChannel,
  openUpstreamUpdateModal,
  detectChannelUpstreamUpdates,
  openOnboardModal,
  isAdminChannelPage,
}) => {
  return [
    {
      key: COLUMN_KEYS.ID,
      title: t('ID'),
      dataIndex: 'id',
    },
    {
      key: COLUMN_KEYS.NAME,
      title: t('名称'),
      dataIndex: 'name',
      render: (text, record, index) => {
        const passThroughEnabled = isRequestPassThroughEnabled(record);
        const upstreamUpdateMeta = getUpstreamUpdateMeta(record);
        const pendingAddCount = upstreamUpdateMeta.pendingAddModels.length;
        const pendingRemoveCount =
          upstreamUpdateMeta.pendingRemoveModels.length;
        const showUpstreamUpdateTag =
          upstreamUpdateMeta.supported &&
          upstreamUpdateMeta.enabled &&
          (pendingAddCount > 0 || pendingRemoveCount > 0);
        const nameNode =
          record.remark && record.remark.trim() !== '' ? (
            <Tooltip
              content={
                <div className='flex flex-col gap-2 max-w-xs'>
                  <div className='text-sm'>{record.remark}</div>
                  <Button
                    size='small'
                    type='primary'
                    theme='outline'
                    onClick={(e) => {
                      e.stopPropagation();
                      navigator.clipboard
                        .writeText(record.remark)
                        .then(() => {
                          showSuccess(t('复制成功'));
                        })
                        .catch(() => {
                          showError(t('复制失败'));
                        });
                    }}
                  >
                    {t('复制')}
                  </Button>
                </div>
              }
              trigger='hover'
              position='topLeft'
            >
              <span>{text}</span>
            </Tooltip>
          ) : (
            <span>{text}</span>
          );

        if (!passThroughEnabled && !showUpstreamUpdateTag) {
          return nameNode;
        }

        return (
          <Space spacing={6} align='center'>
            {nameNode}
            {passThroughEnabled && (
              <Tooltip
                content={t(
                  '该渠道已开启请求透传：参数覆写、模型重定向、渠道适配等 NewAPI 内置功能将失效，非最佳实践；如因此产生问题，请勿提交 issue 反馈。',
                )}
                trigger='hover'
                position='topLeft'
              >
                <span className='inline-flex items-center'>
                  <IconAlertTriangle
                    style={{ color: 'var(--semi-color-warning)' }}
                  />
                </span>
              </Tooltip>
            )}
            {showUpstreamUpdateTag && (
              <Space spacing={4} align='center'>
                {pendingAddCount > 0 ? (
                  <Tooltip content={t('点击处理新增模型')} position='top'>
                    <Tag
                      color='green'
                      type='light'
                      size='small'
                      shape='circle'
                      className='cursor-pointer transition-all duration-150 hover:opacity-85 hover:-translate-y-px active:scale-95'
                      onClick={(e) => {
                        e.stopPropagation();
                        openUpstreamUpdateModal(
                          record,
                          upstreamUpdateMeta.pendingAddModels,
                          upstreamUpdateMeta.pendingRemoveModels,
                          'add',
                        );
                      }}
                    >
                      +{pendingAddCount}
                    </Tag>
                  </Tooltip>
                ) : null}
                {pendingRemoveCount > 0 ? (
                  <Tooltip content={t('点击处理删除模型')} position='top'>
                    <Tag
                      color='red'
                      type='light'
                      size='small'
                      shape='circle'
                      className='cursor-pointer transition-all duration-150 hover:opacity-85 hover:-translate-y-px active:scale-95'
                      onClick={(e) => {
                        e.stopPropagation();
                        openUpstreamUpdateModal(
                          record,
                          upstreamUpdateMeta.pendingAddModels,
                          upstreamUpdateMeta.pendingRemoveModels,
                          'remove',
                        );
                      }}
                    >
                      -{pendingRemoveCount}
                    </Tag>
                  </Tooltip>
                ) : null}
              </Space>
            )}
          </Space>
        );
      },
    },
    {
      key: COLUMN_KEYS.PRICE_DISCOUNT,
      title: t('折扣率'),
      dataIndex: 'price_discount_percent',
      render: (text, record) => {
        if (record.children !== undefined) {
          return '-';
        }
        const raw = text;
        if (raw === null || raw === undefined || raw === '') {
          return '100%';
        }
        const n = Number(raw);
        if (Number.isNaN(n)) {
          return '-';
        }
        const s = n % 1 === 0 ? String(Math.trunc(n)) : n.toFixed(2);
        return `${s}%`;
      },
    },
    {
      key: COLUMN_KEYS.ROUTE_SLUG,
      title: t('路由后缀'),
      dataIndex: 'route_slug',
      render: (_text, record) => {
        if (record.children !== undefined) {
          return '-';
        }
        const slug = String(record?.route_slug ?? '').trim();
        if (!slug) {
          return (
            <Tooltip content={t('路由后缀空说明')}>
              <Typography.Text type='tertiary'>—</Typography.Text>
            </Tooltip>
          );
        }
        return (
          <Tooltip
            content={
              <div className='max-w-sm text-xs'>
                <div>{t('路由后缀说明')}</div>
                <div className='mt-2 font-mono'>
                  {'{model}/'}
                  {slug}
                </div>
              </div>
            }
          >
            <Tag color='blue' type='light' size='small' className='font-mono'>
              {slug}
            </Tag>
          </Tooltip>
        );
      },
    },
    {
      key: COLUMN_KEYS.BALANCE,
      title: isAdminChannelPage ? t('额度') : t('已用'),
      dataIndex: 'expired_time',
      render: (text, record, index) => {
        if (isAdminChannelPage && record.children === undefined) {
          return (
            <Space spacing={4}>
              <Tooltip
                content={
                  t('上游余额：') + renderQuotaWithAmount(record.balance)
                }
              >
                <Tag
                  color='light-blue'
                  type='light'
                  shape='circle'
                  className='cursor-pointer'
                  onClick={() => updateChannelBalance(record)}
                >
                  {renderQuotaWithAmount(record.balance)}
                </Tag>
              </Tooltip>
              <Tag
                color='white'
                type='ghost'
                shape='circle'
                className='cursor-pointer'
                onClick={() => {
                  let manualBalance = Number(record.balance || 0);
                  Modal.confirm({
                    title: t('手动设置额度'),
                    content: (
                      <InputNumber
                        min={0}
                        step={0.01}
                        precision={2}
                        defaultValue={manualBalance}
                        onChange={(value) => {
                          manualBalance = Number(value);
                        }}
                      />
                    ),
                    onOk: async () => {
                      await setChannelBalanceManually(record, manualBalance);
                    },
                  });
                }}
              >
                {t('手动设值')}
              </Tag>
            </Space>
          );
        }
        if (record.children === undefined) {
          return (
            <Tooltip content={t('已用额度')}>
              <Tag color='white' type='ghost' shape='circle'>
                {renderQuota(record.used_quota)}
              </Tag>
            </Tooltip>
          );
        } else {
          return (
            <Tooltip content={t('已用额度')}>
              <Tag color='white' type='ghost' shape='circle'>
                {renderQuota(record.used_quota)}
              </Tag>
            </Tooltip>
          );
        }
      },
    },
    ...(isAdminChannelPage
      ? [
          {
            key: COLUMN_KEYS.USED_REMAINING,
            title: t('已用'),
            dataIndex: 'used_remaining',
            render: (_text, record) => {
              if (record.children !== undefined) {
                return '-';
              }
              return (
                <Tooltip content={t('已用额度')}>
                  <Tag color='white' type='ghost' shape='circle'>
                    {renderQuota(record.used_quota)}
                  </Tag>
                </Tooltip>
              );
            },
          },
        ]
      : []),
    {
      key: COLUMN_KEYS.STATUS,
      title: t('状态'),
      dataIndex: 'status',
      render: (text, record, index) => {
        if (text === 3) {
          if (record.other_info === '') {
            record.other_info = '{}';
          }
          let otherInfo = JSON.parse(record.other_info);
          let reason = otherInfo['status_reason'];
          let time = otherInfo['status_time'];
          return (
            <div>
              <Tooltip
                content={
                  t('原因：') + reason + t('，时间：') + timestamp2string(time)
                }
              >
                {renderStatus(text, record.channel_info, t)}
              </Tooltip>
            </div>
          );
        } else {
          return renderStatus(text, record.channel_info, t);
        }
      },
    },
    {
      key: COLUMN_KEYS.RESPONSE_TIME,
      title: t('响应时间'),
      dataIndex: 'response_time',
      render: (text, record, index) => <div>{renderResponseTime(text, t)}</div>,
    },
    {
      key: COLUMN_KEYS.SUPPLIER,
      title: t('供应商'),
      dataIndex: 'supplier_name',
      render: (text, record) => {
        if (record.children !== undefined) {
          return '-';
        }
        if (!text || text.trim() === '') {
          return '-';
        }
        return text;
      },
    },
    {
      key: COLUMN_KEYS.PRIORITY,
      title: t('优先级'),
      dataIndex: 'priority',
      render: (text, record, index) => {
        if (record.children === undefined) {
          return (
            <div>
              <InputNumber
                style={{ width: 70 }}
                name='priority'
                onBlur={(e) => {
                  manageChannel(record.id, 'priority', record, e.target.value);
                }}
                keepFocus={true}
                innerButtons
                defaultValue={record.priority}
                min={-999}
                size='small'
              />
            </div>
          );
        } else {
          return (
            <InputNumber
              style={{ width: 70 }}
              name='priority'
              keepFocus={true}
              onBlur={(e) => {
                Modal.warning({
                  title: t('修改子渠道优先级'),
                  content:
                    t('确定要修改所有子渠道优先级为 ') +
                    e.target.value +
                    t(' 吗？'),
                  onOk: () => {
                    if (e.target.value === '') {
                      return;
                    }
                    submitTagEdit('priority', {
                      tag: record.key,
                      priority: e.target.value,
                    });
                  },
                });
              }}
              innerButtons
              defaultValue={record.priority}
              min={-999}
              size='small'
            />
          );
        }
      },
    },
    {
      key: COLUMN_KEYS.WEIGHT,
      title: t('权重'),
      dataIndex: 'weight',
      render: (text, record, index) => {
        if (record.children === undefined) {
          return (
            <div>
              <InputNumber
                style={{ width: 70 }}
                name='weight'
                onBlur={(e) => {
                  manageChannel(record.id, 'weight', record, e.target.value);
                }}
                keepFocus={true}
                innerButtons
                defaultValue={record.weight}
                min={0}
                size='small'
              />
            </div>
          );
        } else {
          return (
            <InputNumber
              style={{ width: 70 }}
              name='weight'
              keepFocus={true}
              onBlur={(e) => {
                Modal.warning({
                  title: t('修改子渠道权重'),
                  content:
                    t('确定要修改所有子渠道权重为 ') +
                    e.target.value +
                    t(' 吗？'),
                  onOk: () => {
                    if (e.target.value === '') {
                      return;
                    }
                    submitTagEdit('weight', {
                      tag: record.key,
                      weight: e.target.value,
                    });
                  },
                });
              }}
              innerButtons
              defaultValue={record.weight}
              min={-999}
              size='small'
            />
          );
        }
      },
    },
    {
      key: COLUMN_KEYS.TYPE,
      title: t('类型'),
      dataIndex: 'type',
      render: (text, record, index) => {
        if (record.children === undefined) {
          return <>{renderType(text, record, t)}</>;
        } else {
          return <>{renderTagType(t)}</>;
        }
      },
    },
    {
      key: COLUMN_KEYS.GROUP,
      title: t('分组'),
      dataIndex: 'group',
      render: (text, record, index) => (
        <div>
          <Space spacing={2}>
            {text
              ?.split(',')
              .sort((a, b) => {
                if (a === 'default') return -1;
                if (b === 'default') return 1;
                return a.localeCompare(b);
              })
              .map((item, index) => renderGroup(item))}
          </Space>
        </div>
      ),
    },
    {
      key: COLUMN_KEYS.OPERATE,
      title: '',
      dataIndex: 'operate',
      fixed: 'right',
      render: (text, record, index) => {
        if (record.children === undefined) {
          const upstreamUpdateMeta = getUpstreamUpdateMeta(record);
          const moreMenuItems = [
            {
              node: 'item',
              name: t('删除'),
              type: 'danger',
              onClick: () => {
                Modal.confirm({
                  title: t('确定是否要删除此渠道？'),
                  content: t('此修改将不可逆'),
                  onOk: () => {
                    (async () => {
                      await manageChannel(record.id, 'delete', record);
                      await refresh();
                      setTimeout(() => {
                        if (channels.length === 0 && activePage > 1) {
                          refresh(activePage - 1);
                        }
                      }, 100);
                    })();
                  },
                });
              },
            },
            {
              node: 'item',
              name: t('复制'),
              type: 'tertiary',
              onClick: () => {
                Modal.confirm({
                  title: t('确定是否要复制此渠道？'),
                  content: t('复制渠道的所有信息'),
                  onOk: () => copySelectedChannel(record),
                });
              },
            },
          ];

          if (upstreamUpdateMeta.supported) {
            moreMenuItems.push({
              node: 'item',
              name: t('仅检测上游模型更新'),
              type: 'tertiary',
              onClick: () => {
                detectChannelUpstreamUpdates(record);
              },
            });
            moreMenuItems.push({
              node: 'item',
              name: t('处理上游模型更新'),
              type: 'tertiary',
              onClick: () => {
                if (!upstreamUpdateMeta.enabled) {
                  showInfo(t('该渠道未开启上游模型更新检测'));
                  return;
                }
                if (
                  upstreamUpdateMeta.pendingAddModels.length === 0 &&
                  upstreamUpdateMeta.pendingRemoveModels.length === 0
                ) {
                  showInfo(t('该渠道暂无可处理的上游模型更新'));
                  return;
                }
                openUpstreamUpdateModal(
                  record,
                  upstreamUpdateMeta.pendingAddModels,
                  upstreamUpdateMeta.pendingRemoveModels,
                  upstreamUpdateMeta.pendingAddModels.length > 0
                    ? 'add'
                    : 'remove',
                );
              },
            });
          }

          if (record.type === 4) {
            moreMenuItems.unshift({
              node: 'item',
              name: t('测活'),
              type: 'tertiary',
              onClick: () => checkOllamaVersion(record),
            });
          }

          return (
            <Space wrap>
              <SplitButtonGroup
                className='overflow-hidden'
                aria-label={t('测试单个渠道操作项目组')}
              >
                <Button
                  size='small'
                  type='tertiary'
                  onClick={() => testChannel(record, '')}
                >
                  {t('测试')}
                </Button>
                <Button
                  size='small'
                  type='tertiary'
                  icon={<IconTreeTriangleDown />}
                  onClick={() => {
                    setCurrentTestChannel(record);
                    setShowModelTestModal(true);
                  }}
                />
              </SplitButtonGroup>

              {record.status === 1 ? (
                <Button
                  type='danger'
                  size='small'
                  onClick={() => manageChannel(record.id, 'disable', record)}
                >
                  {t('禁用')}
                </Button>
              ) : (
                <Button
                  size='small'
                  onClick={() => manageChannel(record.id, 'enable', record)}
                >
                  {t('启用')}
                </Button>
              )}

              {record.channel_info?.is_multi_key ? (
                <SplitButtonGroup aria-label={t('多密钥渠道操作项目组')}>
                  <Button
                    type='tertiary'
                    size='small'
                    onClick={() => {
                      setEditingChannel(record);
                      setShowEdit(true);
                    }}
                  >
                    {t('编辑')}
                  </Button>
                  <Dropdown
                    trigger='click'
                    position='bottomRight'
                    menu={[
                      {
                        node: 'item',
                        name: t('多密钥管理'),
                        onClick: () => {
                          setCurrentMultiKeyChannel(record);
                          setShowMultiKeyManageModal(true);
                        },
                      },
                    ]}
                  >
                    <Button
                      type='tertiary'
                      size='small'
                      icon={<IconTreeTriangleDown />}
                    />
                  </Dropdown>
                </SplitButtonGroup>
              ) : (
                <Button
                  type='tertiary'
                  size='small'
                  onClick={() => {
                    setEditingChannel(record);
                    setShowEdit(true);
                  }}
                >
                  {t('编辑')}
                </Button>
              )}

              {openOnboardModal && (
                <Button
                  type='primary'
                  theme='light'
                  size='small'
                  onClick={() => openOnboardModal(record)}
                >
                  {t('上架向导')}
                </Button>
              )}

              <Dropdown
                trigger='click'
                position='bottomRight'
                menu={moreMenuItems}
              >
                <Button icon={<IconMore />} type='tertiary' size='small' />
              </Dropdown>
            </Space>
          );
        } else {
          // 标签操作按钮
          return (
            <Space wrap>
              <Button
                type='tertiary'
                size='small'
                onClick={() => manageTag(record.key, 'enable')}
              >
                {t('启用全部')}
              </Button>
              <Button
                type='tertiary'
                size='small'
                onClick={() => manageTag(record.key, 'disable')}
              >
                {t('禁用全部')}
              </Button>
              <Button
                type='tertiary'
                size='small'
                onClick={() => {
                  setShowEditTag(true);
                  setEditingTag(record.key);
                }}
              >
                {t('编辑')}
              </Button>
            </Space>
          );
        }
      },
    },
  ];
};
