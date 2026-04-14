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

import { useState, useEffect, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';

export const useProvidersData = () => {
  const { t } = useTranslation();

  const [providers, setProviders] = useState([]);
  const [loading, setLoading] = useState(false);
  const [searching, setSearching] = useState(false);
  const [activePage, setActivePage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [providerCount, setProviderCount] = useState(0);

  // Edit state
  const [showEdit, setShowEdit] = useState(false);
  const [editingProvider, setEditingProvider] = useState({ id: undefined });

  // Search form
  const formInitValues = { searchKeyword: '' };
  const formApiRef = useRef(null);
  const setFormApi = (api) => {
    formApiRef.current = api;
  };
  const getFormValues = () => formApiRef.current?.getValues() || {};

  // Load providers
  const loadProviders = async (page = 1, size = pageSize) => {
    setLoading(true);
    try {
      const res = await API.get(`/api/vendors/?p=${page}&page_size=${size}`);
      const { success, message, data } = res.data;
      if (success) {
        const items = data.items || data || [];
        setProviders(Array.isArray(items) ? items : []);
        setProviderCount(data.total || items.length);
        setActivePage(data.page || page);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(t('加载供应商列表失败'));
    }
    setLoading(false);
  };

  // Search providers
  const searchProviders = async () => {
    const { searchKeyword = '' } = getFormValues();

    if (searchKeyword === '') {
      await loadProviders(1, pageSize);
      return;
    }

    setSearching(true);
    try {
      const res = await API.get(
        `/api/vendors/search?keyword=${encodeURIComponent(searchKeyword)}&p=1&page_size=${pageSize}`,
      );
      const { success, message, data } = res.data;
      if (success) {
        const items = data.items || data || [];
        setProviders(Array.isArray(items) ? items : []);
        setProviderCount(data.total || items.length);
        setActivePage(data.page || 1);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(t('搜索供应商失败'));
    }
    setSearching(false);
  };

  // Delete provider
  const deleteProvider = async (id) => {
    try {
      const res = await API.delete(`/api/vendors/${id}`);
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('供应商删除成功'));
        await refresh();
      } else {
        showError(message);
      }
    } catch (error) {
      showError(t('删除供应商失败'));
    }
  };

  // Refresh
  const refresh = async () => {
    await loadProviders(activePage, pageSize);
  };

  // Close edit
  const closeEdit = () => {
    setShowEdit(false);
    setEditingProvider({ id: undefined });
  };

  // Page change
  const handlePageChange = (page) => {
    setActivePage(page);
    loadProviders(page, pageSize);
  };

  // Page size change
  const handlePageSizeChange = async (size) => {
    setPageSize(size);
    setActivePage(1);
    await loadProviders(1, size);
  };

  // Initial load
  useEffect(() => {
    loadProviders();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return {
    providers,
    loading,
    searching,
    activePage,
    pageSize,
    providerCount,

    // Edit
    showEdit,
    setShowEdit,
    editingProvider,
    setEditingProvider,
    closeEdit,

    // Search
    formInitValues,
    setFormApi,
    searchProviders,

    // Actions
    deleteProvider,
    refresh,

    // Pagination
    handlePageChange,
    handlePageSizeChange,

    // Translation
    t,
  };
};
