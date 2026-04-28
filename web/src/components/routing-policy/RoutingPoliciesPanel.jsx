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

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Banner,
  Button,
  Card,
  Popconfirm,
  Space,
  Table,
  Tag,
  Tooltip,
  Typography,
} from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';
import RoutingPolicyEditModal from './RoutingPolicyEditModal';
import RoutingPolicyDryRunModal from './RoutingPolicyDryRunModal';

const { Text, Title } = Typography;

// 与后端 model.RoutingStrategy* 常量保持一致；如有改动需同步两端。
const STRATEGY_LABEL = {
  price: '低价优先',
  latency: '低时延优先',
  throughput: '高吞吐优先',
  balanced: '均衡',
  custom: '自定义',
};

const STRATEGY_TONE = {
  price: 'green',
  latency: 'cyan',
  throughput: 'blue',
  balanced: 'grey',
  custom: 'orange',
};

const FALLBACK_LABEL = {
  '': '不兜底',
  price: '价格兜底',
  latency: '时延兜底',
  any: '任意兜底',
};

// describeTargets 把候选池条目压成「便于在表格里一眼看完」的简短文本，
// 避免一行被冗长的 channel/model 列表撑爆。条目 >5 时截断并显示 +N。
function describeTargets(targets) {
  if (!Array.isArray(targets) || targets.length === 0) return '不限';
  const labels = targets.map((t) => {
    if (t.target_type === 'channel') return `渠道#${t.channel_id}`;
    if (t.target_type === 'model') return `模型:${t.model_name}`;
    if (t.target_type === 'channel_model')
      return `渠道#${t.channel_id}/${t.model_name}`;
    return JSON.stringify(t);
  });
  if (labels.length <= 5) return labels.join('，');
  return `${labels.slice(0, 5).join('，')} +${labels.length - 5}`;
}

const RoutingPoliciesPanel = () => {
  const { t } = useTranslation();
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(false);
  const [pageInfo, setPageInfo] = useState({ page: 1, page_size: 20, total: 0 });
  const [editingPolicy, setEditingPolicy] = useState(null);
  const [editVisible, setEditVisible] = useState(false);
  const [dryRunSeed, setDryRunSeed] = useState(null);
  const [dryRunVisible, setDryRunVisible] = useState(false);

  const refresh = useCallback(
    async (page = pageInfo.page, pageSize = pageInfo.page_size) => {
      setLoading(true);
      try {
        const res = await API.get(
          `/api/user/routing/policies?p=${page}&page_size=${pageSize}`,
        );
        const { success, message, data } = res.data;
        if (!success) {
          showError(message);
          return;
        }
        setItems(data?.items || []);
        setPageInfo({
          page: data?.page || page,
          page_size: data?.page_size || pageSize,
          total: data?.total || 0,
        });
      } catch (err) {
        showError(err?.message || String(err));
      } finally {
        setLoading(false);
      }
    },
    [pageInfo.page, pageInfo.page_size],
  );

  useEffect(() => {
    refresh(1, pageInfo.page_size);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const openCreate = () => {
    setEditingPolicy(null);
    setEditVisible(true);
  };

  const openEdit = async (policy) => {
    // 列表接口不返回 targets，详情接口返回；编辑前必须先拉详情，避免 update 时
    // 误以为 targets 为空而清空候选池。
    try {
      const res = await API.get(`/api/user/routing/policies/${policy.id}`);
      const { success, message, data } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      setEditingPolicy(data || policy);
      setEditVisible(true);
    } catch (err) {
      showError(err?.message || String(err));
    }
  };

  const handleDelete = async (policy) => {
    try {
      const res = await API.delete(`/api/user/routing/policies/${policy.id}`);
      const { success, message } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      showSuccess(t('已删除'));
      refresh(pageInfo.page, pageInfo.page_size);
    } catch (err) {
      showError(err?.message || String(err));
    }
  };

  const handleSetDefault = async (policy) => {
    try {
      const res = await API.post(
        `/api/user/routing/policies/${policy.id}/default`,
      );
      const { success, message } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      showSuccess(t('已设为默认策略'));
      refresh(pageInfo.page, pageInfo.page_size);
    } catch (err) {
      showError(err?.message || String(err));
    }
  };

  const handleClearDefault = async () => {
    try {
      const res = await API.delete(`/api/user/routing/policies/default`);
      const { success, message } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      showSuccess(t('已清除默认策略'));
      refresh(pageInfo.page, pageInfo.page_size);
    } catch (err) {
      showError(err?.message || String(err));
    }
  };

  const handleDryRun = (policy) => {
    setDryRunSeed(policy);
    setDryRunVisible(true);
  };

  const columns = useMemo(
    () => [
      {
        title: t('策略名'),
        dataIndex: 'name',
        render: (name, record) => (
          <Space>
            <Text strong>{name}</Text>
            {record.is_default && (
              <Tag color='violet' size='small'>
                {t('默认')}
              </Tag>
            )}
            {record.status === 0 && (
              <Tag color='grey' size='small'>
                {t('已禁用')}
              </Tag>
            )}
          </Space>
        ),
      },
      {
        title: t('策略类型'),
        dataIndex: 'strategy',
        render: (s) => (
          <Tag color={STRATEGY_TONE[s] || 'grey'}>
            {STRATEGY_LABEL[s] || s}
          </Tag>
        ),
      },
      {
        title: t('Fallback'),
        dataIndex: 'fallback_strategy',
        render: (f, record) => {
          const label = FALLBACK_LABEL[f ?? ''] || (f || '-');
          const allow = record.allow_fallbacks
            ? t('允许 router 回退')
            : t('禁用 router 回退');
          return (
            <Tooltip content={allow}>
              <Tag color={record.allow_fallbacks ? 'amber' : 'grey'}>
                {label}
              </Tag>
            </Tooltip>
          );
        },
      },
      {
        title: t('阈值'),
        render: (_, record) => {
          const parts = [];
          if (record.max_price > 0)
            parts.push(`max_price ≤ ${record.max_price}`);
          if (record.max_latency_ms > 0)
            parts.push(`max_latency ≤ ${record.max_latency_ms}ms`);
          if (record.min_throughput_tps > 0)
            parts.push(`min_throughput ≥ ${record.min_throughput_tps}tps`);
          return parts.length ? parts.join(' / ') : t('未设置');
        },
      },
      {
        title: t('候选池'),
        dataIndex: 'targets',
        render: (targets) => (
          <Text type='tertiary' size='small'>
            {describeTargets(targets)}
          </Text>
        ),
      },
      {
        title: t('操作'),
        width: 240,
        render: (_, record) => (
          <Space>
            <Button size='small' onClick={() => openEdit(record)}>
              {t('编辑')}
            </Button>
            <Button size='small' onClick={() => handleDryRun(record)}>
              {t('演练')}
            </Button>
            {!record.is_default && record.status === 1 && (
              <Button
                size='small'
                theme='light'
                type='primary'
                onClick={() => handleSetDefault(record)}
              >
                {t('设为默认')}
              </Button>
            )}
            <Popconfirm
              title={t('确认删除该策略？')}
              onConfirm={() => handleDelete(record)}
            >
              <Button size='small' type='danger'>
                {t('删除')}
              </Button>
            </Popconfirm>
          </Space>
        ),
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [t],
  );

  const hasDefault = items.some((p) => p.is_default);

  return (
    <Card className='!rounded-2xl shadow-sm border-0'>
      <div className='flex items-center justify-between mb-4'>
        <div>
          <Title heading={4} className='m-0'>
            {t('路由偏好')}
          </Title>
          <Text type='tertiary' size='small'>
            {t(
              '为模型调用配置策略与候选池，支持低价 / 低时延 / 高吞吐 / 均衡 / 自定义；可附带 fallback 兜底保障可用性。',
            )}
          </Text>
        </div>
        <Space>
          <Button onClick={() => refresh(pageInfo.page, pageInfo.page_size)}>
            {t('刷新')}
          </Button>
          {hasDefault && (
            <Popconfirm
              title={t('清除当前默认策略？')}
              onConfirm={handleClearDefault}
            >
              <Button>{t('清除默认')}</Button>
            </Popconfirm>
          )}
          <Button
            theme='solid'
            type='primary'
            onClick={() => {
              setDryRunSeed(null);
              setDryRunVisible(true);
            }}
          >
            {t('演练新策略')}
          </Button>
          <Button theme='solid' type='primary' onClick={openCreate}>
            {t('新建策略')}
          </Button>
        </Space>
      </div>

      <Banner
        type='info'
        description={t(
          '默认策略会在该用户调用模型时自动生效；若客户端在请求体里显式带了 OpenRouter 风格的 provider JSON，则以请求侧为准（候选池约束依旧保留）。',
        )}
        closeIcon={null}
        className='mb-4'
      />

      <Table
        columns={columns}
        dataSource={items}
        rowKey='id'
        loading={loading}
        pagination={{
          currentPage: pageInfo.page,
          pageSize: pageInfo.page_size,
          total: pageInfo.total,
          showSizeChanger: false,
          onPageChange: (page) => refresh(page, pageInfo.page_size),
        }}
      />

      <RoutingPolicyEditModal
        visible={editVisible}
        editingPolicy={editingPolicy}
        onClose={() => setEditVisible(false)}
        onSaved={() => {
          setEditVisible(false);
          refresh(pageInfo.page, pageInfo.page_size);
        }}
      />

      <RoutingPolicyDryRunModal
        visible={dryRunVisible}
        seedPolicy={dryRunSeed}
        onClose={() => setDryRunVisible(false)}
      />
    </Card>
  );
};

export default RoutingPoliciesPanel;
