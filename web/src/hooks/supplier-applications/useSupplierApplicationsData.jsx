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

export const useSupplierApplicationsData = () => {
  const { t } = useTranslation();
  const [compactMode, setCompactMode] = useTableCompactMode('supplier-applications');

  const [applications, setApplications] = useState([]);
  const [loading, setLoading] = useState(true);
  const [activePage, setActivePage] = useState(1);
  const [pageSize, setPageSize] = useState(ITEMS_PER_PAGE);
  const [searching, setSearching] = useState(false);
  const [applicationCount, setApplicationCount] = useState(0);

  const [showReview, setShowReview] = useState(false);
  const [reviewingApplication, setReviewingApplication] = useState(null);

  const [statusFilter, setStatusFilter] = useState('all');

  const formInitValues = {
    searchKeyword: '',
    status: '',
  };

  const [formApi, setFormApi] = useState(null);

  const getFormValues = () => {
    const formValues = formApi ? formApi.getValues() : {};
    return {
      searchKeyword: formValues.searchKeyword || '',
      status: formValues.status || '',
    };
  };

  const setApplicationFormat = (applications) => {
    for (let i = 0; i < applications.length; i++) {
      applications[i].key = applications[i].id;
    }
    setApplications(applications);
  };

  const loadApplications = async (startIdx, pageSize, status = '') => {
    setLoading(true);
    let url = `/api/user/supplier/application?p=${startIdx}&page_size=${pageSize}`;
    if (status !== '' && status !== 'all') {
      url += `&status=${status}`;
    }
    
    try {
      const res = await API.get(url);
      const { success, message, data } = res.data;
      if (success) {
        const newPageData = data.items || [];
        setActivePage(data.page || startIdx);
        setApplicationCount(data.total || 0);
        setApplicationFormat(newPageData);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.response?.data?.message || t('加载失败'));
    }
    setLoading(false);
  };

  const searchApplications = async (
    startIdx,
    pageSize,
    searchKeyword = null,
    status = null,
  ) => {
    if (searchKeyword === null || status === null) {
      const formValues = getFormValues();
      searchKeyword = formValues.searchKeyword;
      status = formValues.status;
    }

    if (searchKeyword === '' && (status === '' || status === 'all')) {
      await loadApplications(startIdx, pageSize);
      return;
    }

    setSearching(true);
    let url = `/api/user/supplier/application?p=${startIdx}&page_size=${pageSize}`;
    if (status !== '' && status !== 'all') {
      url += `&status=${status}`;
    }
    
    try {
      const res = await API.get(url);
      const { success, message, data } = res.data;
      if (success) {
        const newPageData = data.items || [];
        setActivePage(data.page || startIdx);
        setApplicationCount(data.total || 0);
        setApplicationFormat(newPageData);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.response?.data?.message || t('搜索失败'));
    }
    setSearching(false);
  };

  const handleReview = async (id, reviewData) => {
    try {
      const res = await API.post(`/api/user/supplier/application/${id}/review`, reviewData);
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('审批成功'));
        setShowReview(false);
        setReviewingApplication(null);
        refresh();
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.response?.data?.message || t('审批失败'));
    }
  };

  const openReview = (application) => {
    setReviewingApplication(application);
    setShowReview(true);
  };

  const closeReview = () => {
    setShowReview(false);
    setReviewingApplication(null);
  };

  const refresh = async () => {
    const formValues = getFormValues();
    await searchApplications(activePage, pageSize, formValues.searchKeyword, formValues.status);
  };

  const handlePageChange = (page) => {
    setActivePage(page);
    const formValues = getFormValues();
    searchApplications(page, pageSize, formValues.searchKeyword, formValues.status);
  };

  const handlePageSizeChange = (size) => {
    setPageSize(size);
    setActivePage(1);
    const formValues = getFormValues();
    searchApplications(1, size, formValues.searchKeyword, formValues.status);
  };

  const handleStatusFilterChange = (status) => {
    setStatusFilter(status);
    setActivePage(1);
    searchApplications(1, pageSize, '', status);
  };

  useEffect(() => {
    loadApplications(activePage, pageSize);
  }, []);

  return {
    applications,
    loading,
    activePage,
    pageSize,
    searching,
    applicationCount,
    showReview,
    reviewingApplication,
    statusFilter,
    formInitValues,
    compactMode,
    t,
    setFormApi,
    loadApplications,
    searchApplications,
    handleReview,
    openReview,
    closeReview,
    refresh,
    handlePageChange,
    handlePageSizeChange,
    handleStatusFilterChange,
    setCompactMode,
  };
};
