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
  Tooltip,
} from '@douyinfe/semi-ui';
import { IconSearch, IconInfoCircle } from '@douyinfe/semi-icons';
import { AlertCircle } from 'lucide-react';
import { API, showError, showSuccess, showInfo } from '../../helpers';
import { CHANNEL_SUPPLIER_TYPE_OPTIONS } from '../../constants';

const { Text } = Typography;
const { Option } = Select;

const MODEL_TABLE_PAGE_SIZE = 10;
const getRowKey = (item) => `${item.channel_id || 0}:${item.model_name || ''}`;

const markupRatesEqual = (a, b) =>
  Math.round(Number(a ?? 0) * 100) === Math.round(Number(b ?? 0) * 100);

function supplierTypeDisplay(value) {
  const v = String(value || '').trim();
  if (!v) return '—';
  const opt = CHANNEL_SUPPLIER_TYPE_OPTIONS.find((o) => o.value === v);
  return opt ? opt.label : v;
}

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
  const [filterSupplierType, setFilterSupplierType] = useState(null);
  const [page, setPage] = useState(1);
  const [baselineValues, setBaselineValues] = useState({});
  const [discountValues, setDiscountValues] = useState({});

  const loadData = useCallback(
    async (opts = {}) => {
      const { silent = false } = opts;
      if (!inviteeId || !visible) return;
      if (!silent) setLoading(true);
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
        if (!silent) setLoading(false);
      }
    },
    [inviteeId, visible, t],
  );

  useEffect(() => {
    if (visible) {
      loadData();
    } else {
      setSearchKeyword('');
      setFilterSupplierType(null);
      setPage(1);
      setBaselineValues({});
      setDiscountValues({});
    }
  }, [visible, loadData]);

  const supplierTypeFilterOptions = useMemo(() => {
    const seen = new Set();
    const ordered = [];
    CHANNEL_SUPPLIER_TYPE_OPTIONS.forEach((opt) => {
      const has = modelData.some(
        (item) => String(item.supplier_type || '').trim() === opt.value,
      );
      if (has) ordered.push({ value: opt.value, label: opt.label });
      seen.add(opt.value);
    });
    modelData.forEach((item) => {
      const v = String(item.supplier_type || '').trim();
      if (v && !seen.has(v)) {
        seen.add(v);
        ordered.push({ value: v, label: v });
      }
    });
    return ordered;
  }, [modelData]);

  const filteredData = useMemo(() => {
    let result = modelData;
    if (filterSupplierType != null && filterSupplierType !== '') {
      result = result.filter(
        (item) => String(item.supplier_type || '').trim() === filterSupplierType,
      );
    }
    const keyword = searchKeyword.toLowerCase().trim();
    if (!keyword) {
      return result;
    }
    return result.filter((item) => {
      const path = `${item.channel_path || ''} ${item.model_name || ''} ${item.channel_id || ''}`.toLowerCase();
      const raw = String(item.supplier_type || '').trim().toLowerCase();
      const disp = supplierTypeDisplay(item.supplier_type).toLowerCase();
      return (
        path.includes(keyword) ||
        raw.includes(keyword) ||
        disp.includes(keyword)
      );
    });
  }, [modelData, searchKeyword, filterSupplierType]);

  const hasActiveFilter =
    (filterSupplierType != null && filterSupplierType !== '') ||
    !!searchKeyword.trim();

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
  }, [searchKeyword, filterSupplierType, modelData.length]);

  const handleDiscountChange = (rowKey, value) => {
    const numValue = value === null || value === undefined ? 0 : Number(value);
    const clampedValue = Math.max(0, Math.min(100, numValue));

    setDiscountValues((prev) => ({
      ...prev,
      [rowKey]: clampedValue,
    }));
  };

  const persistDiscounts = async () => {
    if (modifiedCount === 0) {
      showInfo(t('暂无修改'));
      return false;
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
        await loadData({ silent: true });
        return true;
      }
      showError(res.data?.message || t('保存失败'));
      return false;
    } catch (e) {
      showError(t('保存失败'));
      return false;
    } finally {
      setSaving(false);
    }
  };

  const handleSaveOnly = async () => {
    await persistDiscounts();
  };

  const handleSaveAndClose = async () => {
    const ok = await persistDiscounts();
    if (ok) onCancel();
  };

  const handleClose = () => {
    onCancel();
  };

  const columns = [
    {
      title: t('模型 / 通道路径'),
      dataIndex: 'model_name',
      width: 380,
      render: (text, record) => (
        <div className='flex flex-col gap-1'>
          <Typography.Text strong copyable>
            {text}
          </Typography.Text>
          <Typography.Text type='secondary' size='small' copyable>
            {record.channel_path || '—'}
          </Typography.Text>
          <Typography.Text type='tertiary' size='small'>
            {supplierTypeDisplay(record.supplier_type)}
          </Typography.Text>
        </div>
      ),
    },
    {
      title: t('系统折扣率'),
      dataIndex: 'default_markup_discount_rate',
      width: 140,
      render: (value) => (
        <Text type='secondary'>{Number(value || 0).toFixed(2)}%</Text>
      ),
    },
    {
      title: t('模型折扣率'),
      dataIndex: 'current_markup_discount_rate',
      width: 200,
      render: (_, record) => {
        const rowKey = getRowKey(record);
        const currentValue =
          discountValues[rowKey] ?? record.current_markup_discount_rate;
        const baseline =
          baselineValues[rowKey] ?? record.current_markup_discount_rate;
        const isModified = !markupRatesEqual(currentValue, baseline);
        const systemRate = Number(record.default_markup_discount_rate ?? 0);
        const aboveSystem = currentValue > systemRate + 1e-6;
        const tipContent = t(
          '您当前设置的折扣率高于系统默认折扣率，会导致用户使用价格高于正常价。',
        );
        return (
          <div className='flex items-center gap-1.5 w-full min-w-0'>
            {aboveSystem ? (
              <Tooltip content={tipContent} position='top'>
                <span
                  className='inline-flex shrink-0 cursor-help text-amber-500 dark:text-amber-400'
                  role='img'
                  aria-label={tipContent}
                >
                  <AlertCircle size={18} strokeWidth={2.25} />
                </span>
              </Tooltip>
            ) : (
              <span
                className='inline-flex w-[18px] shrink-0 justify-center'
                aria-hidden
              />
            )}
            <InputNumber
              value={currentValue}
              onChange={(v) => handleDiscountChange(rowKey, v)}
              min={0}
              max={100}
              precision={2}
              suffix='%'
              className={`min-w-0 flex-1 ${
                isModified ? 'border-semi-color-primary' : ''
              }`}
            />
          </div>
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
            {t('模型折扣率设置')}
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
      width={760}
      footer={
        <div className='flex flex-wrap justify-end gap-2'>
          <Button
            type='primary'
            theme='solid'
            loading={saving}
            onClick={handleSaveOnly}
          >
            {t('保存')}
          </Button>
          <Button type='primary' loading={saving} onClick={handleSaveAndClose}>
            {t('保存并关闭')}
          </Button>
          <Button type='tertiary' disabled={saving} onClick={handleClose}>
            {t('关闭')}
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
              '说明：本功能用于为被邀请用户配置各模型的折扣率（在官方定价基础上的加价比例）。当设置的折扣率高于系统折扣率时，用户侧模型价格会高于系统默认；低于系统折扣率时则会低于系统默认（即减价）。未修改时与渠道系统折扣率一致，修改后仅对该用户生效。',
            )}
          />

          <div className='flex flex-col sm:flex-row gap-2'>
            <Input
              placeholder={t('搜索模型、通道路径或供应商类型...')}
              value={searchKeyword}
              onChange={setSearchKeyword}
              className='flex-1 min-w-0'
              prefix={<IconSearch />}
              showClear
            />
            <Select
              placeholder={t('筛选供应商类型')}
              value={filterSupplierType}
              onChange={setFilterSupplierType}
              showClear
              filter
              className='w-full sm:w-[220px] flex-shrink-0'
            >
              {supplierTypeFilterOptions.map((opt) => (
                <Option key={opt.value} value={opt.value}>
                  {opt.label}
                </Option>
              ))}
            </Select>
          </div>

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
