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
import { Input, ImagePreview } from '@douyinfe/semi-ui';
import { IconSearch } from '@douyinfe/semi-icons';
import PricingProviderType from '../table/model-pricing/filter/PricingProviderType';
import PricingVendors from '../table/model-pricing/filter/PricingVendors';
import PricingQuotaTypes from '../table/model-pricing/filter/PricingQuotaTypes';
import PricingTags from '../table/model-pricing/filter/PricingTags';
import PricingCardView from '../table/model-pricing/view/card/PricingCardView';
import ModelDetailSideSheet from '../table/model-pricing/modal/ModelDetailSideSheet';
import { useModelPricingData } from '../../hooks/model-pricing/useModelPricingData';
import { usePricingFilterCounts } from '../../hooks/model-pricing/usePricingFilterCounts';

const HomeModelList = () => {
  const pricingData = useModelPricingData();
  const { quotaTypeModels, vendorModels, tagModels } = usePricingFilterCounts({
    models: pricingData.models,
    filterGroup: pricingData.filterGroup,
    filterQuotaType: pricingData.filterQuotaType,
    filterEndpointType: pricingData.filterEndpointType,
    filterVendor: pricingData.filterVendor,
    filterTag: pricingData.filterTag,
    searchValue: pricingData.searchValue,
  });

  const [filterProviderType, setFilterProviderType] = React.useState('all');

  React.useEffect(() => {
    pricingData.setPageSize(40);
  }, []);

  return (
    <div className='w-full home-model-list'>
      <style>{`
        .home-model-card-wrapper .grid {
          grid-template-columns: repeat(2, minmax(0, 1fr)) !important;
          gap: 0.75rem !important;
        }
        @media (min-width: 768px) {
          .home-model-card-wrapper .grid {
            grid-template-columns: repeat(3, minmax(0, 1fr)) !important;
          }
        }
        @media (min-width: 1280px) {
          .home-model-card-wrapper .grid {
            grid-template-columns: repeat(4, minmax(0, 1fr)) !important;
          }
        }
      `}</style>
      <div className='max-w-7xl mx-auto'>
        <div className='mb-6 max-w-[800px] mx-auto'>
          <Input
            prefix={<IconSearch />}
            placeholder={pricingData.t('模糊搜索模型名称')}
            value={pricingData.searchValue}
            onCompositionStart={pricingData.handleCompositionStart}
            onCompositionEnd={pricingData.handleCompositionEnd}
            onChange={pricingData.handleChange}
            showClear
            size='large'
          />
        </div>

        <div className='mb-4'>
          <PricingProviderType
            filterProviderType={filterProviderType}
            setFilterProviderType={setFilterProviderType}
            totalCount={pricingData.filteredModels.length}
            loading={pricingData.loading}
            t={pricingData.t}
            layout='inline'
          />

          <PricingVendors
            filterVendor={pricingData.filterVendor}
            setFilterVendor={pricingData.setFilterVendor}
            models={vendorModels}
            allModels={pricingData.models}
            loading={pricingData.loading}
            t={pricingData.t}
            layout='inline'
          />

          <PricingQuotaTypes
            filterQuotaType={pricingData.filterQuotaType}
            setFilterQuotaType={pricingData.setFilterQuotaType}
            models={quotaTypeModels}
            loading={pricingData.loading}
            t={pricingData.t}
            layout='inline'
          />

          <PricingTags
            filterTag={pricingData.filterTag}
            setFilterTag={pricingData.setFilterTag}
            models={tagModels}
            allModels={pricingData.models}
            loading={pricingData.loading}
            t={pricingData.t}
            layout='inline'
          />
        </div>

        <div className='home-model-card-wrapper'>
          <PricingCardView
            filteredModels={pricingData.filteredModels}
            loading={pricingData.loading}
            rowSelection={null}
            pageSize={pricingData.pageSize}
            setPageSize={pricingData.setPageSize}
            currentPage={pricingData.currentPage}
            setCurrentPage={pricingData.setCurrentPage}
            selectedGroup={pricingData.selectedGroup}
            groupRatio={pricingData.groupRatio}
            copyText={pricingData.copyText}
            setModalImageUrl={pricingData.setModalImageUrl}
            setIsModalOpenurl={pricingData.setIsModalOpenurl}
            currency={pricingData.currency}
            siteDisplayType={pricingData.siteDisplayType}
            tokenUnit={pricingData.tokenUnit}
            displayPrice={pricingData.displayPrice}
            showRatio={false}
            t={pricingData.t}
            selectedRowKeys={[]}
            setSelectedRowKeys={() => {}}
            openModelDetail={pricingData.openModelDetail}
            showSizeChanger={false}
          />
        </div>

        <ImagePreview
          src={pricingData.modalImageUrl}
          visible={pricingData.isModalOpenurl}
          onVisibleChange={(visible) => pricingData.setIsModalOpenurl(visible)}
        />

        <ModelDetailSideSheet
          visible={pricingData.showModelDetail}
          onClose={pricingData.closeModelDetail}
          modelData={pricingData.selectedModel}
          groupRatio={pricingData.groupRatio}
          usableGroup={pricingData.usableGroup}
          currency={pricingData.currency}
          siteDisplayType={pricingData.siteDisplayType}
          tokenUnit={pricingData.tokenUnit}
          displayPrice={pricingData.displayPrice}
          showRatio={false}
          vendorsMap={pricingData.vendorsMap}
          endpointMap={pricingData.endpointMap}
          autoGroups={pricingData.autoGroups}
          t={pricingData.t}
        />
      </div>
    </div>
  );
};

export default HomeModelList;
