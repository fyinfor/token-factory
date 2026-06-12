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

import React, { useState, useEffect, useCallback } from 'react';
import { Button, Input, Space, Table } from '@douyinfe/semi-ui';
import { IconDelete, IconPlus } from '@douyinfe/semi-icons';
import {
  normalizeTierRows,
  getCurrencySymbol,
} from '../utils/requestTierPricing';

// ============================================================
// TierRowsEditor — 阶梯计费档位编辑器（v2 重构版）
// ============================================================
// 统一表格编辑所有档位：每行 = 输入Token区间 + 4项价格 + 删除
// props:
//   t          — i18n 翻译函数
//   value      — TierPricing 对象或 tiers 数组
//   onChange   — 回调 (nextTiers: TierRow[])
//   currency   — 'USD' | 'CNY' | 'CUSTOM'，控制价格列符号
// ============================================================

const PRICE_COLUMNS = [
  { key: 'inputPrice', labelKey: '输入价格' },
  { key: 'outputPrice', labelKey: '输出价格' },
  { key: 'cacheReadPrice', labelKey: '缓存读取价格' },
  { key: 'cacheWritePrice', labelKey: '缓存写入价格' },
];

const TierRowsEditor = ({ t, value, onChange, currency = 'USD' }) => {
  // 规范化输入：接受 TierPricing 对象或纯 tiers 数组
  const tiersFromValue = Array.isArray(value) ? value : (value?.tiers ?? []);
  const [tiers, setTiers] = useState(() => normalizeTierRows(tiersFromValue));

  // 编辑状态：记录当前聚焦的单元格 (rowIndex, fieldKey)，用于控制 Input 的值
  const [editingCell, setEditingCell] = useState(null);
  const [editingText, setEditingText] = useState('');

  const currencySymbol = getCurrencySymbol(currency);

  // 外部 value 变化时同步（不覆盖正在编辑的单元格）
  useEffect(() => {
    const incoming = Array.isArray(value) ? value : (value?.tiers ?? []);
    const normalized = normalizeTierRows(incoming);
    if (editingCell === null) {
      setTiers(normalized);
    }
  }, [value, editingCell]);

  // 通知父组件
  const emitChange = useCallback(
    (nextTiers) => {
      const normalized = normalizeTierRows(nextTiers);
      setTiers(normalized);
      onChange(normalized);
    },
    [onChange],
  );

  // ---- 行操作 ----

  /** 添加新档位（插入到列表开头之前） */
  const addTier = () => {
    // 找到第一个有限 up_to，新档位取一半或默认 32000
    const firstFinite = tiers.find((row) => Number(row.up_to) > 0);
    const upTo = firstFinite
      ? Math.max(1, Math.floor(Number(firstFinite.up_to) / 2))
      : 32000;
    const newRow = {
      up_to: upTo,
      inputPrice: 0,
      outputPrice: 0,
      cacheReadPrice: 0,
      cacheWritePrice: 0,
    };
    const next = tiers.length > 0 ? [newRow, ...tiers] : [{ ...newRow, up_to: 0 }];
    emitChange(next);
  };

  /** 删除档位（最后一档无限不可删） */
  const deleteTier = (index) => {
    if (index === tiers.length - 1 && tiers[index].up_to === 0) return; // 最后一档无限不可删
    const next = tiers.filter((_, i) => i !== index);
    emitChange(next);
  };

  // ---- 价格单元格编辑 ----

  const beginEdit = (index, fieldKey) => {
    const currentValue = tiers[index]?.[fieldKey] ?? 0;
    setEditingCell({ index, fieldKey });
    setEditingText(currentValue > 0 ? String(currentValue) : '');
  };

  const commitEdit = () => {
    if (!editingCell) return;
    const { index, fieldKey } = editingCell;
    const next = tiers.map((row, i) => {
      if (i !== index) return row;
      const val = editingText === '' ? 0 : parseFloat(editingText);
      return {
        ...row,
        [fieldKey]: Number.isFinite(val) ? Math.max(0, val) : row[fieldKey],
      };
    });
    setEditingCell(null);
    setEditingText('');
    emitChange(next);
  };

  const cancelEdit = () => {
    setEditingCell(null);
    setEditingText('');
  };

  // ---- up_to 编辑 ----

  const updateUpTo = (index, rawValue) => {
    const next = tiers.map((row, i) => {
      if (i !== index) return row;
      const val = rawValue === '' || rawValue === '∞' ? 0 : parseInt(rawValue, 10);
      return {
        ...row,
        up_to: Number.isFinite(val) && val >= 0 ? val : row.up_to,
      };
    });
    emitChange(next);
  };

  // ---- 渲染 ----

  const getRangeText = (index) => {
    const prev = index === 0 ? 0 : tiers[index - 1]?.up_to || 0;
    const curr = tiers[index]?.up_to || '∞';
    return `${prev}～${curr}`;
  };

  const dataSource = tiers.map((row, i) => ({ ...row, _idx: i }));

  return (
    <Space vertical align='start' style={{ width: '100%' }}>
      <style>{`
        .tier-table .semi-table-thead th,
        .tier-table .semi-table-tbody td {
          background-color: var(--semi-color-fill-0) !important;
        }
        .tier-table .semi-table-tbody td {
          vertical-align: middle;
        }
      `}</style>
      <Table
        size='small'
        pagination={false}
        dataSource={dataSource}
        rowKey='_idx'
        className='tier-table'
        columns={[
          // 区间列（只读展示）
          {
            title: t('输入Token区间'),
            width: 160,
            render: (_, row) => getRangeText(row._idx),
          },
          // 区间上限编辑列
          {
            title: t('区间上限'),
            width: 140,
            render: (_, row) => {
              const isLast = row._idx === tiers.length - 1;
              return (
                <Input
                  size='small'
                  value={isLast ? '∞' : String(row.up_to ?? '')}
                  placeholder={t('区间上限')}
                  disabled={isLast}
                  style={{ width: 100 }}
                  onChange={(v) => updateUpTo(row._idx, v)}
                />
              );
            },
          },
          // 4 列价格输入
          ...PRICE_COLUMNS.map(({ key, labelKey }) => ({
            title: `${t(labelKey)} (${currencySymbol}/1M)`,
            width: 150,
            render: (_, row) => {
              const isEditing =
                editingCell?.index === row._idx && editingCell?.fieldKey === key;
              const displayValue = isEditing
                ? editingText
                : row[key] > 0
                  ? String(Number(row[key].toFixed(6)))
                  : '';
              return (
                <Input
                  size='small'
                  value={displayValue}
                  placeholder='0'
                  style={{ width: 120 }}
                  onFocus={() => beginEdit(row._idx, key)}
                  onChange={(v) => {
                    // 只允许数字和小数点
                    if (/^(\d+\.?\d*|\.\d*)?$/.test(v)) {
                      setEditingText(v);
                    }
                  }}
                  onBlur={() => commitEdit()}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') commitEdit();
                    if (e.key === 'Escape') cancelEdit();
                  }}
                />
              );
            },
          })),
          // 操作列
          {
            title: t('操作'),
            width: 80,
            render: (_, row) => (
              <Button
                type='danger'
                size='small'
                icon={<IconDelete />}
                disabled={row._idx === tiers.length - 1 && row.up_to === 0}
                onClick={() => deleteTier(row._idx)}
              />
            ),
          },
        ]}
      />
      <Button
        className='mt-2'
        size='small'
        icon={<IconPlus />}
        onClick={addTier}
      >
        {t('添加档位')}
      </Button>
    </Space>
  );
};

export default TierRowsEditor;
