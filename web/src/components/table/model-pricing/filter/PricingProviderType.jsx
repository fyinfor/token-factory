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
import SelectableButtonGroup from '../../../common/ui/SelectableButtonGroup';

/**
 * 供应商类型筛选组件
 * @param {string|'all'} filterProviderType 当前值
 * @param {Function} setFilterProviderType setter
 * @param {number} totalCount 总数
 * @param {boolean} loading 是否加载中
 * @param {Function} t i18n
 * @param {string} layout 布局模式
 */
const PricingProviderType = ({
  filterProviderType,
  setFilterProviderType,
  totalCount = 0,
  loading = false,
  t,
  layout,
}) => {
  const items = React.useMemo(() => {
    return [
      {
        value: 'all',
        label: t('全部'),
        tagCount: totalCount,
      },
      {
        value: 'official',
        label: t('官方'),
        tagCount: totalCount,
      },
    ];
  }, [totalCount, t]);

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
