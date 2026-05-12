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

import React, { useState, useEffect } from 'react';
import { Button, Card, Input, Space, Switch, Table, Typography } from '@douyinfe/semi-ui';
import { IconDelete, IconPlus } from '@douyinfe/semi-icons';
import {
  ensureFinalInfinityTierSegments,
  normalizeTierSegments,
  priceToRatio,
  ratioToPrice,
} from '../utils/requestTierPricing';

const { Text } = Typography;

const TierRowsEditor = ({ t, value, onChange, exchangeRate = 1, tierType = 'input' }) => {
  const tier = normalizeTierSegments(value);
  const [editingValue, setEditingValue] = useState(null);
  const [editingKey, setEditingKey] = useState(null);

  // 当 value 变化时，清除编辑状态
  useEffect(() => {
    setEditingValue(null);
    setEditingKey(null);
  }, [value]);

  const updateSegments = (segments) => {
    const updated = ensureFinalInfinityTierSegments(segments);
    onChange({ segments: updated });
  };

  const insertTopRow = (segments) => {
    const firstFinite = segments.find((row) => Number(row.up_to) > 0);
    const upTo = firstFinite
      ? Math.max(1, Math.floor(Number(firstFinite.up_to) / 2))
      : 128000;
    // 添加新档位时不自动设置价格，保持为空（ratio = 0）
    const next = segments.length
      ? [{ up_to: upTo, ratio: 0 }, ...segments]
      : [{ up_to: 0, ratio: 0 }];
    updateSegments(next);
  };

  // 验证价格递增：每一档价格不能大于下一档
  const validatePriceIncrease = (segments) => {
    for (let i = 0; i < segments.length - 1; i++) {
      const current = segments[i];
      const next = segments[i + 1];
      if (current.ratio > 0 && next.ratio > 0 && current.ratio > next.ratio) {
        return false;
      }
    }
    return true;
  };

  return (
    <Space vertical align='start' style={{ width: '100%' }}>
      <style>{`
        .tier-table .semi-table-thead th,
        .tier-table .semi-table-tbody td {
          background-color: var(--semi-color-fill-0) !important;
        }
      `}</style>
      <Table
        size='small'
        pagination={false}
        dataSource={tier.segments.map((row, index) => ({ ...row, _idx: index }))}
        rowKey='_idx'
        className='tier-table'
        columns={[
          {
            title: t('区间 token'),
            render: (_, row) => {
              const previous =
                row._idx === 0 ? 0 : tier.segments[row._idx - 1]?.up_to || 0;
              const current = row.up_to || '∞';
              return `${previous}～${current}`;
            },
          },
          {
            title: t('上限 token'),
            dataIndex: 'up_to',
            render: (_, row) => (
              <Input
                value={
                  row._idx === tier.segments.length - 1
                    ? '∞'
                    : String(row.up_to ?? '')
                }
                placeholder={t('最后一档固定无限')}
                disabled={row._idx === tier.segments.length - 1}
                onChange={(v) => {
                  const next = [...tier.segments];
                  next[row._idx] = { ...next[row._idx], up_to: v };
                  updateSegments(next);
                }}
              />
            ),
          },
          {
            title: t('价格 (USD/1M token)'),
            dataIndex: 'ratio',
            render: (_, row) => {
              const previous =
                row._idx === 0 ? 0 : tier.segments[row._idx - 1]?.up_to || 0;
              // 价格显示基于固定的 1M token 数量，不随档位变化
              const fixedTokenCount = 1000000;

              // 价格模式：显示价格，保存时转换为倍率
              const currentInputKey = `${tierType}_${row._idx}`;
              const priceValue = row.ratio
                ? ratioToPrice(row.ratio * 2, fixedTokenCount, exchangeRate)
                : '';
              const displayValue = editingKey === currentInputKey ? editingValue : String(priceValue);
              return (
                <Input
                  value={displayValue}
                  onFocus={() => {
                    setEditingKey(currentInputKey);
                    setEditingValue(String(priceValue));
                  }}
                  onChange={(v) => {
                    setEditingValue(v);
                  }}
                  onBlur={(e) => {
                    const v = e.target.value;
                    if (!v) {
                      setEditingKey(null);
                      setEditingValue(null);
                      return;
                    }
                    const priceUSD = parseFloat(v);
                    const newRatio = priceToRatio(
                      priceUSD,
                      fixedTokenCount,
                      exchangeRate,
                    ) / 2;
                    const next = [...tier.segments];
                    next[row._idx] = { ...next[row._idx], ratio: newRatio };

                    // 验证价格递增
                    if (!validatePriceIncrease(next)) {
                      // 如果价格不递增，不保存并提示用户
                      setEditingKey(null);
                      setEditingValue(null);
                      return;
                    }

                    updateSegments(next);
                    setEditingKey(null);
                    setEditingValue(null);
                  }}
                />
              );
            },
          },
          {
            title: t('操作'),
            render: (_, row) => (
              <Button
                type='danger'
                size='small'
                icon={<IconDelete />}
                disabled={row._idx === tier.segments.length - 1}
                onClick={() =>
                  updateSegments(
                    tier.segments.filter((_, idx) => idx !== row._idx),
                  )
                }
              />
            ),
          },
        ]}
      />
      <Button
        className='mt-2'
        size='small'
        icon={<IconPlus />}
        onClick={() => insertTopRow(tier.segments)}
      >
        {t('添加档位')}
      </Button>
    </Space>
  );
};

export default TierRowsEditor;
