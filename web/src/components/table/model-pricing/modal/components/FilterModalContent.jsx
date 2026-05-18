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
import PricingDisplaySettings from '../../filter/PricingDisplaySettings';
import PricingGroups from '../../filter/PricingGroups';
import PricingQuotaTypes from '../../filter/PricingQuotaTypes';
import PricingEndpointTypes from '../../filter/PricingEndpointTypes';
import PricingVendors from '../../filter/PricingVendors';
import PricingTags from '../../filter/PricingTags';
import PricingSuppliers from '../../../../../components/home/PricingSuppliers';
import { usePricingFilterCounts } from '../../../../../hooks/model-pricing/usePricingFilterCounts';

const FilterModalContent = ({ sidebarProps, t }) => {
  const {
    showWithRecharge,
    setShowWithRecharge,
    currency,
    setCurrency,
    siteDisplayType,
    handleChange,
    setActiveKey,
    showRatio,
    setShowRatio,
    viewMode,
    setViewMode,
    filterGroup,
    setFilterGroup,
    filterQuotaType,
    setFilterQuotaType,
    filterEndpointType,
    setFilterEndpointType,
    filterVendor,
    setFilterVendor,
    filterTag,
    setFilterTag,
    filterSupplier,
    setFilterSupplier,
    tokenUnit,
    setTokenUnit,
    loading,
    ...categoryProps
  } = sidebarProps;

  const supplierCountModels = React.useMemo(() => {
    let result = categoryProps.models || [];
    if (filterVendor !== 'all') {
      result = filterVendor === 'unknown'
        ? result.filter((m) => !m.vendor_name)
        : result.filter((m) => m.vendor_name === filterVendor);
    }
    if (filterTag !== 'all') {
      const tagLower = filterTag.toLowerCase();
      result = result.filter((m) => {
        if (!m.tags) return false;
        return m.tags.toLowerCase().split(/[,;|]+/).map((t) => t.trim()).includes(tagLower);
      });
    }
    if (sidebarProps.searchValue?.length > 0) {
      const term = sidebarProps.searchValue.toLowerCase();
      result = result.filter(
        (m) =>
          (m.model_name && m.model_name.toLowerCase().includes(term)) ||
          (m.description && m.description.toLowerCase().includes(term)) ||
          (m.tags && m.tags.toLowerCase().includes(term)) ||
          (m.vendor_name && m.vendor_name.toLowerCase().includes(term)),
      );
    }
    return result;
  }, [categoryProps.models, filterVendor, filterTag, sidebarProps.searchValue]);

  const {
    quotaTypeModels,
    endpointTypeModels,
    vendorModels,
    tagModels,
    groupCountModels,
  } = usePricingFilterCounts({
    models: categoryProps.models,
    filterGroup,
    filterQuotaType,
    filterEndpointType,
    filterVendor,
    filterTag,
    searchValue: sidebarProps.searchValue,
  });

  return (
    <>
      <PricingDisplaySettings
        showWithRecharge={showWithRecharge}
        setShowWithRecharge={setShowWithRecharge}
        currency={currency}
        setCurrency={setCurrency}
        siteDisplayType={siteDisplayType}
        showRatio={showRatio}
        setShowRatio={setShowRatio}
        viewMode={viewMode}
        setViewMode={setViewMode}
        tokenUnit={tokenUnit}
        setTokenUnit={setTokenUnit}
        loading={loading}
        t={t}
      />

      <PricingVendors
        filterVendor={filterVendor}
        setFilterVendor={setFilterVendor}
        models={vendorModels}
        allModels={categoryProps.models}
        loading={loading}
        t={t}
      />

      <PricingGroups
        filterGroup={filterGroup}
        setFilterGroup={setFilterGroup}
        usableGroup={categoryProps.usableGroup}
        groupRatio={categoryProps.groupRatio}
        models={groupCountModels}
        loading={loading}
        t={t}
      />

      <PricingQuotaTypes
        filterQuotaType={filterQuotaType}
        setFilterQuotaType={setFilterQuotaType}
        models={quotaTypeModels}
        loading={loading}
        t={t}
      />

      <PricingTags
        filterTag={filterTag}
        setFilterTag={setFilterTag}
        models={tagModels}
        allModels={categoryProps.models}
        loading={loading}
        t={t}
      />

      <PricingSuppliers
        filterSupplier={filterSupplier}
        setFilterSupplier={setFilterSupplier}
        models={categoryProps.models}
        countModels={supplierCountModels}
        loading={loading}
        t={t}
      />

      <PricingEndpointTypes
        filterEndpointType={filterEndpointType}
        setFilterEndpointType={setFilterEndpointType}
        models={endpointTypeModels}
        allModels={categoryProps.models}
        loading={loading}
        t={t}
      />
    </>
  );
};

export default FilterModalContent;
