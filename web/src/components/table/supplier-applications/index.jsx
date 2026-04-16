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
import CardPro from '../../common/ui/CardPro';
import SupplierApplicationsTable from './SupplierApplicationsTable';
import SupplierApplicationsFilters from './SupplierApplicationsFilters';
import SupplierApplicationsDescription from './SupplierApplicationsDescription';
import ReviewApplicationModal from './modals/ReviewApplicationModal';
import { useSupplierApplicationsData } from '../../../hooks/supplier-applications/useSupplierApplicationsData';
import { useIsMobile } from '../../../hooks/common/useIsMobile';
import { createCardProPagination } from '../../../helpers/utils';

const SupplierApplicationsPage = () => {
  const applicationsData = useSupplierApplicationsData();
  const isMobile = useIsMobile();

  const {
    showReview,
    reviewingApplication,
    closeReview,
    handleReview,
    refresh,
    formInitValues,
    setFormApi,
    searchApplications,
    loadApplications,
    activePage,
    pageSize,
    loading,
    searching,
    compactMode,
    setCompactMode,
    t,
  } = applicationsData;

  return (
    <>
      <ReviewApplicationModal
        visible={showReview}
        application={reviewingApplication}
        handleClose={closeReview}
        handleReview={handleReview}
      />

      <CardPro
        type='type1'
        descriptionArea={
          <SupplierApplicationsDescription
            compactMode={compactMode}
            setCompactMode={setCompactMode}
            t={t}
          />
        }
        actionsArea={
          <SupplierApplicationsFilters
            formInitValues={formInitValues}
            setFormApi={setFormApi}
            searchApplications={searchApplications}
            loadApplications={loadApplications}
            activePage={activePage}
            pageSize={pageSize}
            loading={loading}
            searching={searching}
            t={t}
          />
        }
        paginationArea={createCardProPagination({
          currentPage: applicationsData.activePage,
          pageSize: applicationsData.pageSize,
          total: applicationsData.applicationCount,
          onPageChange: applicationsData.handlePageChange,
          onPageSizeChange: applicationsData.handlePageSizeChange,
          isMobile: isMobile,
          t: applicationsData.t,
        })}
        t={applicationsData.t}
      >
        <SupplierApplicationsTable {...applicationsData} />
      </CardPro>
    </>
  );
};

export default SupplierApplicationsPage;
