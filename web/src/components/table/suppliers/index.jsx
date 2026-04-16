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

import React, { useState } from 'react';
import { Button } from '@douyinfe/semi-ui';
import { IconPlus } from '@douyinfe/semi-icons';
import CardPro from '../../common/ui/CardPro';
import SuppliersTable from './SuppliersTable';
import SuppliersFilters from './SuppliersFilters';
import SuppliersDescription from './SuppliersDescription';
import SupplierEditModal from './modals/SupplierEditModal';
import DeactivateSupplierModal from './modals/DeactivateSupplierModal';
import { useSuppliersData } from '../../../hooks/suppliers/useSuppliersData';
import { useIsMobile } from '../../../hooks/common/useIsMobile';
import { createCardProPagination } from '../../../helpers/utils';

const SuppliersPage = () => {
  const suppliersData = useSuppliersData();
  const isMobile = useIsMobile();
  const [showDeactivateModal, setShowDeactivateModal] = useState(false);
  const [deactivatingSupplier, setDeactivatingSupplier] = useState(null);

  const {
    showEditModal,
    editingSupplier,
    closeEdit,
    openEdit,
    refresh,
    formInitValues,
    setFormApi,
    searchSuppliers,
    loadSuppliers,
    activePage,
    pageSize,
    loading,
    searching,
    compactMode,
    setCompactMode,
    t,
  } = suppliersData;

  const handleDeactivate = (supplier) => {
    setDeactivatingSupplier(supplier);
    setShowDeactivateModal(true);
  };

  const closeDeactivate = () => {
    setShowDeactivateModal(false);
    setDeactivatingSupplier(null);
  };

  return (
    <>
      <SupplierEditModal
        visible={showEditModal}
        supplier={editingSupplier}
        handleClose={closeEdit}
        onSuccess={refresh}
      />

      <DeactivateSupplierModal
        visible={showDeactivateModal}
        supplier={deactivatingSupplier}
        handleClose={closeDeactivate}
        onSuccess={refresh}
      />

      <CardPro
        type='type1'
        descriptionArea={
          <SuppliersDescription
            compactMode={compactMode}
            setCompactMode={setCompactMode}
            t={t}
          />
        }
        actionsArea={
          <div style={{ display: 'flex', gap: '8px', alignItems: 'flex-end' }}>
            <SuppliersFilters
              formInitValues={formInitValues}
              setFormApi={setFormApi}
              searchSuppliers={searchSuppliers}
              loadSuppliers={loadSuppliers}
              activePage={activePage}
              pageSize={pageSize}
              loading={loading}
              searching={searching}
              t={t}
            />
            <Button
              type='primary'
              icon={<IconPlus />}
              onClick={() => openEdit(null)}
            >
              {t('新增')}
            </Button>
          </div>
        }
        paginationArea={createCardProPagination({
          currentPage: suppliersData.activePage,
          pageSize: suppliersData.pageSize,
          total: suppliersData.supplierCount,
          onPageChange: suppliersData.handlePageChange,
          onPageSizeChange: suppliersData.handlePageSizeChange,
          isMobile: isMobile,
          t: suppliersData.t,
        })}
        t={suppliersData.t}
      >
        <SuppliersTable
          suppliers={suppliersData.suppliers}
          loading={suppliersData.loading}
          t={suppliersData.t}
          openEdit={openEdit}
          handleDeactivate={handleDeactivate}
          compactMode={compactMode}
        />
      </CardPro>
    </>
  );
};

export default SuppliersPage;
