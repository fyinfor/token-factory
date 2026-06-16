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
import { IconSearch, IconInfoCircle, IconDownload } from '@douyinfe/semi-icons';
import { API, showError, showSuccess, showInfo } from '../../helpers';
import { CHANNEL_SUPPLIER_TYPE_OPTIONS } from '../../constants';

const { Text } = Typography;
const { Option } = Select;

const MODEL_TABLE_PAGE_SIZE = 10;
const getRowKey = (item) => `${item.channel_id || 0}:${item.model_name || ''}`;

const markupRatesEqual = (a, b) =>
  Math.round(Number(a ?? 0) * 100) === Math.round(Number(b ?? 0) * 100);

const clampMarkupRate = (value, max = 100) => {
  const n = Number(value ?? 0);
  if (!Number.isFinite(n)) return 0;
  return Math.max(0, Math.min(max, n));
};

const formatMarkupRate = (value) => {
  const n = Number(value ?? 0);
  if (!Number.isFinite(n)) return '0.0%';
  return `${n.toFixed(1)}%`;
};

const getCostDiscountPercent = (record) => {
  const n = Number(record?.channel_price_discount_percent ?? 100);
  if (!Number.isFinite(n)) return 100;
  return Math.max(0, n);
};

const maxAgentMarkupRate = (record) =>
  Math.max(0, 100 - getCostDiscountPercent(record));

const calcSaleRatePercent = (record, markupRate) => {
  const costDiscountPercent = getCostDiscountPercent(record);
  const markup = Number(markupRate ?? 0);
  if (!Number.isFinite(markup)) return costDiscountPercent;
  return Math.max(0, costDiscountPercent + Math.max(0, markup));
};

const FormulaHeader = ({ title, hint }) => (
  <Tooltip content={hint} position='top'>
    <span className='cursor-help'>{title}</span>
  </Tooltip>
);

const FormulaCell = ({ cost, markup, result, markupType = 'secondary' }) => (
  <div className='flex min-w-0 flex-col gap-1 font-mono tabular-nums'>
    <div className='whitespace-nowrap'>
      <Text type='tertiary'>{formatMarkupRate(cost)}</Text>
      <Text type='tertiary'> + </Text>
      <Text type={markupType}>{formatMarkupRate(markup)}</Text>
    </div>
    <div className='whitespace-nowrap'>
      <Text type='tertiary'>= </Text>
      <Text strong type={result > 100 ? 'warning' : 'success'}>
        {formatMarkupRate(result)}
      </Text>
    </div>
  </div>
);

const formatExportTimestamp = () => {
  const d = new Date();
  const pad = (n) => String(n).padStart(2, '0');
  return `${d.getFullYear()}${pad(d.getMonth() + 1)}${pad(d.getDate())}-${pad(
    d.getHours(),
  )}${pad(d.getMinutes())}${pad(d.getSeconds())}`;
};

const downloadBlob = (blob, filename) => {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  URL.revokeObjectURL(url);
};

const getDownloadFilename = (disposition, fallback) => {
  const text = String(disposition || '');
  const utf8Match = text.match(/filename\*=UTF-8''([^;]+)/i);
  if (utf8Match?.[1]) {
    try {
      return decodeURIComponent(utf8Match[1]);
    } catch {
      return fallback;
    }
  }
  const asciiMatch = text.match(/filename="?([^";]+)"?/i);
  return asciiMatch?.[1] || fallback;
};

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
  const [exporting, setExporting] = useState(false);
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
          const maxRate = maxAgentMarkupRate(item);
          initialValues[getRowKey(item)] = clampMarkupRate(
            item.current_markup_discount_rate,
            maxRate,
          );
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
        (item) =>
          String(item.supplier_type || '').trim() === filterSupplierType,
      );
    }
    const keyword = searchKeyword.toLowerCase().trim();
    if (!keyword) {
      return result;
    }
    return result.filter((item) => {
      const path =
        `${item.channel_path || ''} ${item.model_name || ''} ${item.channel_id || ''}`.toLowerCase();
      const raw = String(item.supplier_type || '')
        .trim()
        .toLowerCase();
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

  const handleDiscountChange = (rowKey, value, maxRate) => {
    setDiscountValues((prev) => ({
      ...prev,
      [rowKey]: clampMarkupRate(value, maxRate),
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
        markup_discount_rate: clampMarkupRate(
          discountValues[getRowKey(item)] ?? item.current_markup_discount_rate,
          maxAgentMarkupRate(item),
        ),
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

  const handleExportPriceTable = useCallback(() => {
    if (!filteredData.length) {
      showError(t('\u8bf7\u9009\u62e9\u8981\u5bfc\u51fa\u7684\u6570\u636e'));
      return;
    }

    const fallbackName = `agent-model-discount-prices-${formatExportTimestamp()}.xlsx`;
    setExporting(true);
    API.get('/api/distributor/invitee-model-discounts/export', {
      responseType: 'blob',
      disableDuplicate: true,
      params: {
        invitee_id: inviteeId,
        q: searchKeyword.trim(),
        supplier_type: filterSupplierType || '',
      },
    })
      .then(async (res) => {
        const contentType =
          res.headers?.['content-type'] || res.data?.type || '';
        if (contentType.includes('application/json')) {
          const text = await res.data.text();
          try {
            const payload = JSON.parse(text);
            showError(payload?.message || t('\u5bfc\u51fa\u5931\u8d25'));
          } catch {
            showError(text || t('\u5bfc\u51fa\u5931\u8d25'));
          }
          return;
        }
        const blob = new Blob([res.data], {
          type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
        });
        downloadBlob(
          blob,
          getDownloadFilename(
            res.headers?.['content-disposition'],
            fallbackName,
          ),
        );
        showSuccess(t('\u4ef7\u683c\u5bfc\u51fa\u6210\u529f'));
      })
      .catch(() => {
        showError(t('\u5bfc\u51fa\u5931\u8d25'));
      })
      .finally(() => {
        setExporting(false);
      });
  }, [filterSupplierType, filteredData.length, inviteeId, searchKeyword, t]);

  const columns = [
    {
      title: t('模型 / 通道路径'),
      dataIndex: 'model_name',
      width: 320,
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
      title: (
        <FormulaHeader
          title={t('成本折扣')}
          hint={t('渠道成本价占官方价的比例，来自渠道成本折扣配置。')}
        />
      ),
      dataIndex: 'channel_price_discount_percent',
      width: 120,
      render: (_, record) => (
        <div className='font-mono tabular-nums'>
          <Text type='secondary'>
            {formatMarkupRate(getCostDiscountPercent(record))}
          </Text>
        </div>
      ),
    },
    {
      title: (
        <FormulaHeader
          title={t('平台加价折扣比例')}
          hint={t('平台默认加价比例，默认作为代理加价比例初始值。')}
        />
      ),
      dataIndex: 'default_markup_discount_rate',
      width: 145,
      render: (value) => (
        <Text type='secondary'>{formatMarkupRate(value)}</Text>
      ),
    },
    {
      title: (
        <FormulaHeader
          title={t('平台售价比例')}
          hint={t('成本折扣 + 平台加价折扣比例 = 平台售价比例。')}
        />
      ),
      dataIndex: 'platform_sale_rate',
      width: 150,
      render: (_, record) => {
        const cost = getCostDiscountPercent(record);
        const markup = Number(record.default_markup_discount_rate || 0);
        const result = calcSaleRatePercent(record, markup);
        return <FormulaCell cost={cost} markup={markup} result={result} />;
      },
    },
    {
      title: (
        <FormulaHeader
          title={t('代理加价比例')}
          hint={t('代理为该下级设置的加价比例，上限为 100% - 成本折扣。')}
        />
      ),
      dataIndex: 'current_markup_discount_rate',
      width: 240,
      render: (_, record) => {
        const rowKey = getRowKey(record);
        const maxRate = maxAgentMarkupRate(record);
        const currentValue =
          discountValues[rowKey] ??
          clampMarkupRate(record.current_markup_discount_rate, maxRate);
        const baseline =
          baselineValues[rowKey] ??
          clampMarkupRate(record.current_markup_discount_rate, maxRate);
        const isModified = !markupRatesEqual(currentValue, baseline);
        return (
          <div className='w-full min-w-0'>
            <InputNumber
              value={currentValue}
              onChange={(v) => handleDiscountChange(rowKey, v, maxRate)}
              min={0}
              max={maxRate}
              precision={1}
              suffix='%'
              aria-label={t('代理加价比例')}
              disabled={maxRate <= 0}
              className={`w-full ${
                isModified ? 'border-semi-color-primary' : ''
              }`}
            />
            <Text type='tertiary' size='small' className='mt-1 !block'>
              {t('可输入范围')}：0.0% - {formatMarkupRate(maxRate)}
            </Text>
            <Text type='tertiary' size='small' className='!block'>
              {t('上限 = 100% - 成本折扣')}
            </Text>
          </div>
        );
      },
    },
    {
      title: (
        <FormulaHeader
          title={t('修改后售价比例')}
          hint={t('成本折扣 + 当前代理加价比例 = 下级用户最终售价比例。')}
        />
      ),
      dataIndex: 'preview_discount_percent',
      width: 150,
      render: (_, record) => {
        const rowKey = getRowKey(record);
        const maxRate = maxAgentMarkupRate(record);
        const currentValue =
          discountValues[rowKey] ??
          clampMarkupRate(record.current_markup_discount_rate, maxRate);
        const cost = getCostDiscountPercent(record);
        const saleRate = calcSaleRatePercent(record, currentValue);
        return (
          <FormulaCell
            cost={cost}
            markup={currentValue}
            result={saleRate}
            markupType='primary'
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
      width={1180}
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
              '说明：本功能用于为被邀请用户配置各模型的代理加价比例。成本折扣 + 平台加价折扣比例 = 平台售价比例；成本折扣 + 代理加价比例 = 修改后售价比例。代理加价比例上限为 100% - 成本折扣，避免超过官方价。',
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
            <Button
              type='tertiary'
              icon={<IconDownload />}
              loading={exporting}
              disabled={
                loading || saving || exporting || filteredData.length === 0
              }
              onClick={handleExportPriceTable}
              className='w-full sm:w-auto flex-shrink-0'
            >
              {t('导出')} Excel
            </Button>
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

          <style>
            {`
              .invitee-model-discount-table .semi-table-thead > .semi-table-row > .semi-table-row-head:nth-child(n + 2):nth-child(-n + 4),
              .invitee-model-discount-table .semi-table-tbody > .semi-table-row > .semi-table-row-cell:nth-child(n + 2):nth-child(-n + 4) {
                background: color-mix(in srgb, var(--semi-color-info) 4%, var(--semi-color-bg-2)) !important;
              }

              .invitee-model-discount-table .semi-table-thead > .semi-table-row > .semi-table-row-head:nth-child(n + 5):nth-child(-n + 6),
              .invitee-model-discount-table .semi-table-tbody > .semi-table-row > .semi-table-row-cell:nth-child(n + 5):nth-child(-n + 6) {
                background: color-mix(in srgb, var(--semi-color-success) 5%, var(--semi-color-bg-2)) !important;
              }

              .invitee-model-discount-table .semi-table-tbody > .semi-table-row:hover > .semi-table-row-cell:nth-child(n + 2):nth-child(-n + 4) {
                background: color-mix(in srgb, var(--semi-color-info) 8%, var(--semi-color-fill-0)) !important;
              }

              .invitee-model-discount-table .semi-table-tbody > .semi-table-row:hover > .semi-table-row-cell:nth-child(n + 5):nth-child(-n + 6) {
                background: color-mix(in srgb, var(--semi-color-success) 9%, var(--semi-color-fill-0)) !important;
              }

              .invitee-model-discount-table,
              .invitee-model-discount-table .semi-table-container,
              .invitee-model-discount-table .semi-table-header,
              .invitee-model-discount-table .semi-table-body,
              .invitee-model-discount-table table {
                width: 100% !important;
              }

              .invitee-model-discount-table .semi-table-thead > .semi-table-row > .semi-table-row-head:nth-child(2),
              .invitee-model-discount-table .semi-table-tbody > .semi-table-row > .semi-table-row-cell:nth-child(2),
              .invitee-model-discount-table .semi-table-thead > .semi-table-row > .semi-table-row-head:nth-child(5),
              .invitee-model-discount-table .semi-table-tbody > .semi-table-row > .semi-table-row-cell:nth-child(5) {
                box-shadow: inset 1px 0 0 var(--semi-color-border);
              }
            `}
          </style>

          <Table
            className='invitee-model-discount-table'
            columns={columns}
            dataSource={pagedData}
            scroll={{ x: 1125 }}
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
