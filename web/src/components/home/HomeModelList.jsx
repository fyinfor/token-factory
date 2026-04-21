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
import { Input, ImagePreview, Button } from '@douyinfe/semi-ui';
import { IconSearch } from '@douyinfe/semi-icons';
import PricingVendors from '../table/model-pricing/filter/PricingVendors';
import PricingQuotaTypes from '../table/model-pricing/filter/PricingQuotaTypes';
import PricingTags from '../table/model-pricing/filter/PricingTags';
import PricingEndpointTypes from '../table/model-pricing/filter/PricingEndpointTypes';
import PricingCardView from '../table/model-pricing/view/card/PricingCardView';
import ModelDetailSideSheet from '../table/model-pricing/modal/ModelDetailSideSheet';
import { useModelPricingData } from '../../hooks/model-pricing/useModelPricingData';
import { usePricingFilterCounts } from '../../hooks/model-pricing/usePricingFilterCounts';

const HomeModelList = () => {
  const pricingData = useModelPricingData();
  const { quotaTypeModels, endpointTypeModels, vendorModels, tagModels } =
    usePricingFilterCounts({
      models: pricingData.models,
      filterGroup: pricingData.filterGroup,
      filterQuotaType: pricingData.filterQuotaType,
      filterEndpointType: pricingData.filterEndpointType,
      filterVendor: pricingData.filterVendor,
      filterTag: pricingData.filterTag,
      searchValue: pricingData.searchValue,
    });

  React.useEffect(() => {
    pricingData.setPageSize(40);
  }, []);

  const handleResetFilters = () => {
    pricingData.setSearchValue('');
    pricingData.setFilterVendor([]);
    pricingData.setFilterQuotaType([]);
    pricingData.setFilterTag([]);
    pricingData.setFilterEndpointType([]);
    pricingData.setCurrentPage(1);
  };

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
        .home-model-layout {
          display: flex;
          gap: 1.5rem;
          width: 100%;
        }
        .home-model-sidebar {
          position: sticky;
          top: 60px;
          align-self: flex-start;
          width: 280px;
          min-width: 280px;
          max-width: 280px;
          max-height: calc(100vh - 60px);
          flex-shrink: 0;
          display: flex;
          flex-direction: column;
        }
        .home-sidebar-header {
          position: sticky;
          top: 0;
          z-index: 11;
          padding: 1rem 1rem 0.5rem 1rem;
        }
        .home-sidebar-filters {
          flex: 1;
          overflow-y: auto;
          scrollbar-width: none;
          padding: 0 1rem 1rem 1rem;
        }
        .home-sidebar-filters::-webkit-scrollbar {
          display: none;
        }
        .home-model-content {
          flex: 1;
          min-width: 0;
        }
      `}</style>
      <div className='home-model-layout'>
        <div className='home-model-sidebar'>
          <div className='home-sidebar-header'>
            <div className='flex items-center justify-between mb-4'>
              <div className='text-lg font-semibold text-gray-800'>
                {pricingData.t('筛选')}
              </div>
              <Button
                theme='outline'
                type='tertiary'
                onClick={handleResetFilters}
                className='text-gray-500 hover:text-gray-700'
              >
                {pricingData.t('重置')}
              </Button>
            </div>
          </div>
          
          <div className='home-sidebar-filters'>
            <PricingVendors
            filterVendor={pricingData.filterVendor}
            setFilterVendor={pricingData.setFilterVendor}
            models={vendorModels}
            allModels={pricingData.models}
            loading={pricingData.loading}
            t={pricingData.t}
          />

          <PricingQuotaTypes
            filterQuotaType={pricingData.filterQuotaType}
            setFilterQuotaType={pricingData.setFilterQuotaType}
            models={quotaTypeModels}
            loading={pricingData.loading}
            t={pricingData.t}
          />

          <PricingTags
            filterTag={pricingData.filterTag}
            setFilterTag={pricingData.setFilterTag}
            models={tagModels}
            allModels={pricingData.models}
            loading={pricingData.loading}
            t={pricingData.t}
          />

          <PricingEndpointTypes
            filterEndpointType={pricingData.filterEndpointType}
            setFilterEndpointType={pricingData.setFilterEndpointType}
            models={endpointTypeModels}
            allModels={pricingData.models}
            loading={pricingData.loading}
            t={pricingData.t}
          />
          </div>
        </div>

        <div className='home-model-content px-4'>
          <div className='w-full sticky top-[75px] z-index-[10] my-4 bg-[var(--semi-color-bg-0)] rounded-xl'>
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
              groupModelPrice={pricingData.groupModelPrice}
              groupModelRatio={pricingData.groupModelRatio}
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
        </div>
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
        groupModelPrice={pricingData.groupModelPrice}
        groupModelRatio={pricingData.groupModelRatio}
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
  );
};

export default HomeModelList;
