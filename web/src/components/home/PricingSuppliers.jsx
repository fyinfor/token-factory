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
import SelectableButtonGroup from '../common/ui/SelectableButtonGroup';
import { stringToColor } from '../../helpers';

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

const buildSupplierIcon = (logo, supplierType) => {
  if (!logo && !supplierType) return null;
  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 3 }}>
      {logo && (
        <img
          src={logo}
          alt=''
          style={{ width: 16, height: 16, objectFit: 'contain', borderRadius: 4, flexShrink: 0 }}
        />
      )}
      {supplierType && (
        <Tag size='small' shape='circle' color={getSupplierTypeColor(supplierType)}>
          {supplierType}
        </Tag>
      )}
    </span>
  );
};

/**
 * 首页供应商渠道筛选组件
 *
 * @param {string|'all'} filterSupplier 当前选中的 supplier_alias
 * @param {Function} setFilterSupplier setter
 * @param {Array} models 所有模型（用于提取供应商列表）
 * @param {Array} countModels 当前过滤后模型（用于计数，排除本维度筛选）
 * @param {boolean} loading
 * @param {Function} t i18n
 */
const PricingSuppliers = ({
  filterSupplier,
  setFilterSupplier,
  models = [],
  countModels = [],
  loading = false,
  t,
}) => {
  // 从所有模型的 channel_list 中提取去重的供应商信息
  const allSuppliers = React.useMemo(() => {
    const seen = new Map();
    models.forEach((model) => {
      if (!model.channel_list) return;
      model.channel_list.forEach((ch) => {
        const alias = (ch?.supplier_alias && String(ch.supplier_alias).trim()) || '';
        if (!alias || seen.has(alias)) return;
        const logo = (ch?.company_logo_url && String(ch.company_logo_url).trim()) || '';
        const supplierType = (ch?.supplier_type && String(ch.supplier_type).trim()) || '';
        seen.set(alias, { alias, logo, supplierType });
      });
    });
    return Array.from(seen.values()).sort((a, b) => a.alias.localeCompare(b.alias));
  }, [models]);

  // 计算每个供应商在 countModels 中的模型数量
  const getCount = React.useCallback(
    (alias) => {
      if (alias === 'all') return countModels.length;
      return countModels.filter(
        (model) =>
          model.channel_list &&
          model.channel_list.some((ch) => (ch?.supplier_alias || '') === alias),
      ).length;
    },
    [countModels],
  );

  const items = React.useMemo(() => {
    const result = [
      { value: 'all', label: t('全部供应商'), tagCount: getCount('all') },
    ];
    allSuppliers.forEach(({ alias, logo, supplierType }) => {
      const icon = buildSupplierIcon(logo, supplierType);
      result.push({
        value: alias,
        label: icon ? '' : alias,
        icon,
        tagCount: getCount(alias),
      });
    });
    return result;
  }, [allSuppliers, getCount, t]);

  if (allSuppliers.length === 0) return null;

  return (
    <SelectableButtonGroup
      title={t('供应商渠道')}
      items={items}
      activeValue={filterSupplier}
      onChange={setFilterSupplier}
      loading={loading}
      variant='amber'
      t={t}
    />
  );
};

export default PricingSuppliers;
