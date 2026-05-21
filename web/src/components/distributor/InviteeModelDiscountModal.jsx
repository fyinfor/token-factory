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

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import {
  Modal,
  Button,
  Input,
  Table,
  Typography,
  InputNumber,
  Banner,
  Spin,
  Space,
  Select,
} from '@douyinfe/semi-ui';
import { IconSearch, IconInfoCircle } from '@douyinfe/semi-icons';
import { API, showError, showSuccess } from '../../helpers';

const { Text } = Typography;
const { Option } = Select;

const MODEL_TABLE_PAGE_SIZE = 10;
const getRowKey = (item) => `${item.channel_id || 0}:${item.model_name || ''}`;

/**
 * 被邀请用户模型折扣率编辑弹框
 * 布局参考 ModelTestModal.jsx
 * @param {object} props
 * @param {boolean} props.visible - 弹框显示状态
 * @param {function} props.onCancel - 关闭弹框回调
 * @param {number} props.inviteeId - 被邀请用户ID
 * @param {string} props.inviteeLabel - 被邀请用户显示名称
 * @param {function} props.t - 翻译函数
 */
const InviteeModelDiscountModal = ({
  visible,
  onCancel,
  inviteeId,
  inviteeLabel,
  t,
}) => {
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [modelData, setModelData] = useState([]);
  const [searchKeyword, setSearchKeyword] = useState('');
  const [filterChannelId, setFilterChannelId] = useState(null);
  const [page, setPage] = useState(1);
  const [modifiedModels, setModifiedModels] = useState(new Set());
  const [discountValues, setDiscountValues] = useState({});

  // 加载模型折扣数据
  const loadData = useCallback(async () => {
    if (!inviteeId || !visible) return;
    setLoading(true);
    try {
      const res = await API.get(
        `/api/distributor/invitee-model-discounts?invitee_id=${inviteeId}`,
      );
      if (!res.data?.success) {
        showError(res.data?.message || t('加载失败'));
        return;
      }
      const items = res.data?.data?.items || [];
      setModelData(items);

      // 初始化折扣值映射
      const initialValues = {};
      items.forEach((item) => {
        initialValues[getRowKey(item)] = item.current_markup_discount_rate;
      });
      setDiscountValues(initialValues);
      setModifiedModels(new Set());
    } catch (e) {
      showError(t('加载失败'));
    } finally {
      setLoading(false);
    }
  }, [inviteeId, visible, t]);

  useEffect(() => {
    if (visible) {
      loadData();
    } else {
      // 重置状态
      setSearchKeyword('');
      setFilterChannelId(null);
      setPage(1);
      setModifiedModels(new Set());
      setDiscountValues({});
    }
  }, [visible, loadData]);

  const channelOptions = useMemo(() => {
    const byId = new Map();
    modelData.forEach((item) => {
      const id = item.channel_id;
      if (id == null || byId.has(id)) return;
      const name = String(item.channel_name || '').trim();
      byId.set(id, {
        value: id,
        label: name ? `${name} (${id})` : `#${id}`,
      });
    });
    return Array.from(byId.values()).sort((a, b) =>
      String(a.label).localeCompare(String(b.label)),
    );
  }, [modelData]);

  const filteredData = useMemo(() => {
    let result = modelData;
    if (filterChannelId != null && filterChannelId !== '') {
      result = result.filter((item) => item.channel_id === filterChannelId);
    }
    const keyword = searchKeyword.toLowerCase().trim();
    if (keyword) {
      result = result.filter((item) =>
        `${item.model_name || ''} ${item.channel_name || ''} ${item.channel_id || ''}`
          .toLowerCase()
          .includes(keyword),
      );
    }
    return result;
  }, [modelData, searchKeyword, filterChannelId]);

  const hasActiveFilter =
    (filterChannelId != null && filterChannelId !== '') ||
    !!searchKeyword.trim();

  useEffect(() => {
    setPage(1);
  }, [searchKeyword, filterChannelId, modelData.length]);

  // 处理折扣值变化
  const handleDiscountChange = (rowKey, value) => {
    const numValue = value === null || value === undefined ? 0 : Number(value);
    // 限制在 0-100 范围
    const clampedValue = Math.max(0, Math.min(100, numValue));

    setDiscountValues((prev) => ({
      ...prev,
      [rowKey]: clampedValue,
    }));

    // 标记为已修改
    setModifiedModels((prev) => new Set([...prev, rowKey]));
  };

  // 保存修改
  const handleSave = async () => {
    if (modifiedModels.size === 0) {
      onCancel();
      return;
    }

    setSaving(true);
    try {
      // 构建请求数据 - 只提交所有模型的当前值（后端会判断与默认值的差异）
      const discounts = modelData.map((item) => ({
        model_name: item.model_name,
        channel_id: item.channel_id,
        markup_discount_rate:
          discountValues[getRowKey(item)] ?? item.current_markup_discount_rate,
      }));

      const res = await API.put('/api/distributor/invitee-model-discounts', {
        invitee_id: inviteeId,
        discounts,
      });

      if (res.data?.success) {
        showSuccess(t('保存成功'));
        onCancel();
      } else {
        showError(res.data?.message || t('保存失败'));
      }
    } catch (e) {
      showError(t('保存失败'));
    } finally {
      setSaving(false);
    }
  };

  const columns = [
    {
      title: t('模型 / 渠道'),
      dataIndex: 'model_name',
      width: 360,
      render: (text, record) => (
        <div className='flex flex-col gap-1'>
          <Typography.Text strong copyable>
            {text}
          </Typography.Text>
          <Typography.Text type='secondary' size='small' copyable>
            {record.channel_name || '-'}
          </Typography.Text>
          {/* <Typography.Text type='tertiary' size='small'>
            {t('渠道 ID')}：{record.channel_id}
          </Typography.Text> */}
        </div>
      ),
    },
    {
      title: t('默认加价折扣率'),
      dataIndex: 'default_markup_discount_rate',
      width: 140,
      render: (value) => (
        <Text type='secondary'>{Number(value || 0).toFixed(2)}%</Text>
      ),
    },
    {
      title: t('加价折扣率修改'),
      dataIndex: 'current_markup_discount_rate',
      width: 160,
      render: (_, record) => {
        const rowKey = getRowKey(record);
        const currentValue =
          discountValues[rowKey] ?? record.current_markup_discount_rate;
        const isModified = modifiedModels.has(rowKey);
        return (
          <InputNumber
            value={currentValue}
            onChange={(v) => handleDiscountChange(rowKey, v)}
            min={0}
            max={100}
            precision={2}
            suffix='%'
            className={
              isModified ? 'w-full border-semi-color-primary' : 'w-full'
            }
          />
        );
      },
    },
  ];

  // 分页数据
  const pagedData = (() => {
    const start = (page - 1) * MODEL_TABLE_PAGE_SIZE;
    const end = start + MODEL_TABLE_PAGE_SIZE;
    return filteredData.slice(start, end);
  })();

  return (
    <Modal
      title={
        <div className='flex items-center gap-2'>
          <Typography.Text strong className='!text-base'>
            {t('模型加价折扣率设置')}
          </Typography.Text>
          {inviteeLabel && (
            <Typography.Text type='tertiary' size='small'>
              ({inviteeLabel})
            </Typography.Text>
          )}
        </div>
      }
      visible={visible}
      onCancel={onCancel}
      width={720}
      footer={
        <div className='flex justify-end gap-2'>
          <Button onClick={onCancel} disabled={saving}>
            {t('取消')}
          </Button>
          <Button type='primary' loading={saving} onClick={handleSave}>
            {t('保存')}
          </Button>
        </div>
      }
    >
      <Spin spinning={loading}>
        <div className='space-y-4'>
          <Banner
            type='info'
            closeIcon={null}
            icon={<IconInfoCircle />}
            className='!rounded-lg'
            description={t(
              '说明：本功能配置被邀请用户相对官方价格的加价折扣率，默认取渠道加价折扣；修改后仅对该用户生效。取值 0–100%，0 表示不加价，100 表示在官方价上加价 100%。',
            )}
          />

          <div className='flex flex-col sm:flex-row gap-2'>
            <Input
              placeholder={t('搜索模型或渠道...')}
              value={searchKeyword}
              onChange={setSearchKeyword}
              className='flex-1 min-w-0'
              prefix={<IconSearch />}
              showClear
            />
            <Select
              placeholder={t('搜索渠道')}
              value={filterChannelId}
              onChange={setFilterChannelId}
              showClear
              filter
              className='w-full sm:w-[220px] flex-shrink-0'
            >
              {channelOptions.map((opt) => (
                <Option key={opt.value} value={opt.value}>
                  {opt.label}
                </Option>
              ))}
            </Select>
          </div>

          {/* 统计信息 */}
          <Space className='text-sm'>
            <Text type='tertiary'>
              {hasActiveFilter
                ? t('显示 ${shown} / 共 ${total} 个模型渠道')
                    .replace('${shown}', String(filteredData.length))
                    .replace('${total}', String(modelData.length))
                : t('共 ${count} 个模型渠道').replace(
                    '${count}',
                    String(modelData.length),
                  )}
            </Text>
            {modifiedModels.size > 0 && (
              <Text type='primary'>
                {t('已修改 ${count} 个').replace(
                  '${count}',
                  modifiedModels.size,
                )}
              </Text>
            )}
          </Space>

          {/* 表格 */}
          <Table
            columns={columns}
            dataSource={pagedData}
            pagination={{
              currentPage: page,
              pageSize: MODEL_TABLE_PAGE_SIZE,
              total: filteredData.length,
              showSizeChanger: false,
              onPageChange: (p) => setPage(p),
            }}
            rowKey={getRowKey}
          />
        </div>
      </Spin>
    </Modal>
  );
};

export default InviteeModelDiscountModal;
