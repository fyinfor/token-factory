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
import { Button, Card, Input, Space, Switch, Table, Typography } from '@douyinfe/semi-ui';
import { IconDelete, IconPlus } from '@douyinfe/semi-icons';
import {
  TIER_CATEGORIES,
  ensureFinalInfinityTierRows,
  normalizeTierRule,
  priceToRatio,
  ratioToPrice,
} from '../utils/requestTierPricing';

const { Text } = Typography;

const TierRowsEditor = ({ t, value, onChange, exchangeRate = 1, visibleCategories, onVisibleCategoriesChange }) => {
  const rule = normalizeTierRule(value);

  const updateRows = (key, rows) => {
    const updated = { ...rule, [key]: ensureFinalInfinityTierRows(rows) };
    // 过滤掉开关未打开的类别（除了 input，input 始终保留）
    const filtered = { ...updated };
    if (!visibleCategories.output) delete filtered.output;
    if (!visibleCategories.cache_read) delete filtered.cache_read;
    if (!visibleCategories.cache_write) delete filtered.cache_write;
    onChange(filtered);
  };
  const insertTopRow = (key, rows) => {
    const firstFinite = rows.find((row) => Number(row.up_to) > 0);
    const upTo = firstFinite
      ? Math.max(1, Math.floor(Number(firstFinite.up_to) / 2))
      : 128000;
    const next = rows.length
      ? [{ up_to: upTo, ratio: 1 }, ...rows]
      : [{ up_to: 0, ratio: 1 }];
    updateRows(key, next);
  };

  return (
    <Space vertical align='start' style={{ width: '100%' }}>
      <style>{`
        .tier-table .semi-table-thead th,
        .tier-table .semi-table-tbody td {
          background-color: var(--semi-color-fill-0) !important;
        }
      `}</style>
      <div className='flex items-center justify-between w-full mb-2'>
        <Text type='tertiary' size='small'>
          {t('当前汇率')}: {exchangeRate.toFixed(2)}
        </Text>
      </div>
      {TIER_CATEGORIES.map(({ key, label }) => {
        const rows = ensureFinalInfinityTierRows(rule[key] || []);
        // 输入始终显示内容，其他通过开关控制内容显示
        const isContentVisible = key === 'input' || visibleCategories[key];
        return (
          <Card
            key={key}
            title={
              key === 'input' ? (
                <span>{t(label)}</span>
              ) : (
                <div className='flex items-center justify-between w-full'>
                  <span>{t(label)}</span>
                  <Switch
                    size='small'
                    checked={visibleCategories[key]}
                    onChange={(checked) =>
                      onVisibleCategoriesChange({ ...visibleCategories, [key]: checked })
                    }
                  />
                </div>
              )
            }
            bodyStyle={isContentVisible ? {} : { padding: '0 !important' }}
            style={{ width: '100%', background: 'var(--semi-color-fill-0)' }}
          >
            {isContentVisible ? (
              <>
                <Table
                  size='small'
                  pagination={false}
                  dataSource={rows.map((row, index) => ({ ...row, _idx: index }))}
                  rowKey='_idx'
                  className='tier-table'
                  columns={[
                    {
                      title: t('区间 token'),
                      render: (_, row) => {
                        const previous =
                          row._idx === 0 ? 0 : rows[row._idx - 1]?.up_to || 0;
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
                            row._idx === rows.length - 1
                              ? '∞'
                              : String(row.up_to ?? '')
                          }
                          placeholder={t('最后一档固定无限')}
                          disabled={row._idx === rows.length - 1}
                          onChange={(v) => {
                            const next = [...rows];
                            next[row._idx] = { ...next[row._idx], up_to: v };
                            updateRows(key, next);
                          }}
                        />
                      ),
                    },
                    {
                      title: t('价格 (USD)'),
                      dataIndex: 'ratio',
                      render: (_, row) => {
                        const previous =
                          row._idx === 0 ? 0 : rows[row._idx - 1]?.up_to || 0;
                        // 最后一行（up_to === 0）使用上一档的 token 数量
                        const tokenCount = row.up_to === 0
                          ? (row._idx === 0 ? 1 : previous)
                          : row.up_to - previous;

                        // 价格模式：显示价格，保存时转换为倍率
                        const priceValue = row.ratio
                          ? ratioToPrice(row.ratio, tokenCount, exchangeRate)
                          : '';
                        return (
                          <Input
                            value={String(priceValue)}
                            placeholder='0.001'
                            onChange={(v) => {
                              const next = [...rows];
                              const priceUSD = parseFloat(v);
                              const newRatio = priceToRatio(
                                priceUSD,
                                tokenCount,
                                exchangeRate,
                              );
                              next[row._idx] = { ...next[row._idx], ratio: newRatio };
                              updateRows(key, next);
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
                          disabled={row._idx === rows.length - 1}
                          onClick={() =>
                            updateRows(
                              key,
                              rows.filter((_, idx) => idx !== row._idx),
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
                  onClick={() => insertTopRow(key, rows)}
                >
                  {t('添加档位')}
                </Button>
              </>
            ) : null}
          </Card>
        );
      })}
    </Space>
  );
};

export default TierRowsEditor;
