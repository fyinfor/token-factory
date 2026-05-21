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
} from '@douyinfe/semi-ui';
import { IconSearch, IconInfoCircle } from '@douyinfe/semi-icons';
import { API, showError, showSuccess } from '../../helpers';

const { Text } = Typography;

const MODEL_TABLE_PAGE_SIZE = 10;
const getRowKey = (item) => `${item.channel_id || 0}:${item.model_name || ''}`;

const markupRatesEqual = (a, b) =>
  Math.round(Number(a ?? 0) * 100) === Math.round(Number(b ?? 0) * 100);

/**
 * 被邀请用户模型加价折扣率编辑弹框
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
  const [page, setPage] = useState(1);
  const [baselineValues, setBaselineValues] = useState({});
  const [discountValues, setDiscountValues] = useState({});

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

      const initialValues = {};
      items.forEach((item) => {
        initialValues[getRowKey(item)] = item.current_markup_discount_rate;
      });
      setBaselineValues(initialValues);
      setDiscountValues(initialValues);
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
      setSearchKeyword('');
      setPage(1);
      setBaselineValues({});
      setDiscountValues({});
    }
  }, [visible, loadData]);

  const filteredData = useMemo(() => {
    const keyword = searchKeyword.toLowerCase().trim();
    if (!keyword) {
      return modelData;
    }
    return modelData.filter((item) =>
      `${item.channel_path || ''} ${item.model_name || ''} ${item.channel_id || ''}`
        .toLowerCase()
        .includes(keyword),
    );
  }, [modelData, searchKeyword]);

  const hasActiveFilter = !!searchKeyword.trim();

  const modifiedCount = useMemo(() => {
    if (!modelData.length) return 0;
    return modelData.reduce((count, item) => {
      const rowKey = getRowKey(item);
      const base = baselineValues[rowKey];
      const cur = discountValues[rowKey] ?? base;
      return markupRatesEqual(cur, base) ? count : count + 1;
    }, 0);
  }, [modelData, baselineValues, discountValues]);

  useEffect(() => {
    setPage(1);
  }, [searchKeyword, modelData.length]);

  const handleDiscountChange = (rowKey, value) => {
    const numValue = value === null || value === undefined ? 0 : Number(value);
    const clampedValue = Math.max(0, Math.min(100, numValue));

    setDiscountValues((prev) => ({
      ...prev,
      [rowKey]: clampedValue,
    }));
  };

  const handleSave = async () => {
    if (modifiedCount === 0) {
      onCancel();
      return;
    }

    setSaving(true);
    try {
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
      title: t('通道路径'),
      dataIndex: 'channel_path',
      width: 360,
      render: (text) => (
        <Typography.Text strong copyable>
          {text || '—'}
        </Typography.Text>
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
        const isModified = !markupRatesEqual(
          currentValue,
          baselineValues[rowKey] ?? record.current_markup_discount_rate,
        );
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

          <Input
            placeholder={t('搜索通道路径...')}
            value={searchKeyword}
            onChange={setSearchKeyword}
            className='!w-full'
            prefix={<IconSearch />}
            showClear
          />

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
            {modifiedCount > 0 && (
              <Text type='primary'>
                {t('已修改 ${count} 个').replace(
                  '${count}',
                  String(modifiedCount),
                )}
              </Text>
            )}
          </Space>

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
