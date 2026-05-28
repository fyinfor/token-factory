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
import { Tag } from '@douyinfe/semi-ui';
import SelectableButtonGroup from '../../../common/ui/SelectableButtonGroup';
import { stringToColor } from '../../../../helpers';

const getSupplierTypeColor = (supplierType) => {
  switch (supplierType) {
    case '公有云':
      return 'green';
    case 'AIDC':
      return 'light-green';
    case '企业中转站':
      return 'lime';
    case '个人中转站':
      return 'yellow';
    default:
      return stringToColor(supplierType);
  }
};

/**
 * 供应商类型筛选组件
 * @param {string|'all'} filterProviderType 当前值
 * @param {Function} setFilterProviderType setter
 * @param {Array} models 所有模型（用于提取供应商类型）
 * @param {Array} countModels 当前过滤后模型（用于计数，排除本维度筛选）
 * @param {boolean} loading 是否加载中
 * @param {Function} t i18n
 * @param {string} layout 布局模式
 */
const PricingProviderType = ({
  filterProviderType,
  setFilterProviderType,
  models = [],
  countModels = [],
  loading = false,
  t,
  layout,
}) => {
  const supplierTypes = React.useMemo(() => {
    const seen = new Set();
    models.forEach((model) => {
      if (!model.channel_list) return;
      model.channel_list.forEach((ch) => {
        const supplierType =
          (ch?.supplier_type && String(ch.supplier_type).trim()) || '';
        if (supplierType) seen.add(supplierType);
      });
    });
    return Array.from(seen).sort();
  }, [models]);

  const getCount = React.useCallback(
    (supplierType) => {
      if (supplierType === 'all') return countModels.length;
      return countModels.filter(
        (model) =>
          model.channel_list &&
          model.channel_list.some(
            (ch) => (ch?.supplier_type || '') === supplierType,
          ),
      ).length;
    },
    [countModels],
  );

  const items = React.useMemo(() => {
    const result = [
      {
        value: 'all',
        label: t('全部类型'),
        tagCount: getCount('all'),
      },
    ];

    supplierTypes.forEach((supplierType) => {
      result.push({
        value: supplierType,
        label: '',
        icon: (
          <Tag
            size='small'
            shape='circle'
            color={getSupplierTypeColor(supplierType)}
          >
            {supplierType}
          </Tag>
        ),
        tagCount: getCount(supplierType),
      });
    });

    return result;
  }, [supplierTypes, getCount, t]);

  if (supplierTypes.length === 0) return null;

  return (
    <SelectableButtonGroup
      title={t('供应商类型')}
      items={items}
      activeValue={filterProviderType}
      onChange={setFilterProviderType}
      loading={loading}
      variant='blue'
      t={t}
      layout={layout}
    />
  );
};

export default PricingProviderType;
