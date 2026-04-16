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

import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';
import { ITEMS_PER_PAGE } from '../../constants';
import { useTableCompactMode } from '../common/useTableCompactMode';

export const useSuppliersData = () => {
  const { t } = useTranslation();
  const [compactMode, setCompactMode] = useTableCompactMode('suppliers');

  const [suppliers, setSuppliers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [activePage, setActivePage] = useState(1);
  const [pageSize, setPageSize] = useState(ITEMS_PER_PAGE);
  const [searching, setSearching] = useState(false);
  const [supplierCount, setSupplierCount] = useState(0);

  const [showEditModal, setShowEditModal] = useState(false);
  const [editingSupplier, setEditingSupplier] = useState(null);

  const formInitValues = {
    company_name: '',
    status: '',
  };

  const [formApi, setFormApi] = useState(null);

  const getFormValues = () => {
    const formValues = formApi ? formApi.getValues() : {};
    return {
      company_name: formValues.company_name || '',
      status: formValues.status || '',
    };
  };

  const setSupplierFormat = (suppliers) => {
    for (let i = 0; i < suppliers.length; i++) {
      suppliers[i].key = suppliers[i].id;
    }
    setSuppliers(suppliers);
  };

  const loadSuppliers = async (startIdx, pageSize, companyName = '', status = '') => {
    setLoading(true);
    let url = `/api/user/supplier/list?p=${startIdx}&page_size=${pageSize}`;
    if (companyName !== '') {
      url += `&company_name=${encodeURIComponent(companyName)}`;
    }
    if (status !== '') {
      url += `&status=${status}`;
    }
    
    try {
      const res = await API.get(url);
      const { success, message, data } = res.data;
      if (success) {
        const newPageData = data.items || [];
        setActivePage(data.page || startIdx);
        setSupplierCount(data.total || 0);
        setSupplierFormat(newPageData);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.response?.data?.message || t('加载失败'));
    }
    setLoading(false);
  };

  const searchSuppliers = async (
    startIdx,
    pageSize,
    companyName = null,
    status = null,
  ) => {
    if (companyName === null || status === null) {
      const formValues = getFormValues();
      companyName = companyName === null ? formValues.company_name : companyName;
      status = status === null ? formValues.status : status;
    }

    if (companyName === '' && status === '') {
      await loadSuppliers(startIdx, pageSize);
      return;
    }

    setSearching(true);
    await loadSuppliers(startIdx, pageSize, companyName, status);
    setSearching(false);
  };

  const openEdit = (supplier = null) => {
    setEditingSupplier(supplier);
    setShowEditModal(true);
  };

  const closeEdit = () => {
    setShowEditModal(false);
    setEditingSupplier(null);
  };

  const refresh = async () => {
    const formValues = getFormValues();
    await searchSuppliers(activePage, pageSize, formValues.company_name, formValues.status);
  };

  const handlePageChange = (page) => {
    setActivePage(page);
    const formValues = getFormValues();
    searchSuppliers(page, pageSize, formValues.company_name, formValues.status);
  };

  const handlePageSizeChange = (size) => {
    setPageSize(size);
    setActivePage(1);
    const formValues = getFormValues();
    searchSuppliers(1, size, formValues.company_name, formValues.status);
  };

  useEffect(() => {
    loadSuppliers(activePage, pageSize);
  }, []);

  return {
    suppliers,
    loading,
    activePage,
    pageSize,
    searching,
    supplierCount,
    showEditModal,
    editingSupplier,
    formInitValues,
    compactMode,
    t,
    setFormApi,
    loadSuppliers,
    searchSuppliers,
    openEdit,
    closeEdit,
    refresh,
    handlePageChange,
    handlePageSizeChange,
    setCompactMode,
  };
};
