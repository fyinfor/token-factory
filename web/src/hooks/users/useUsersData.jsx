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

const DEFAULT_QUOTA_PER_UNIT = 500 * 1000;
const getQuotaPerUnit = () => {
  const n = parseFloat(localStorage.getItem('quota_per_unit') || '');
  return Number.isFinite(n) && n > 0 ? n : DEFAULT_QUOTA_PER_UNIT;
};
const quotaToUsd = (quota) => {
  const q = Number(quota);
  if (!Number.isFinite(q) || q <= 0) return 0;
  return q / getQuotaPerUnit();
};
const usdToQuota = (usd) => {
  const u = Number(usd);
  if (!Number.isFinite(u) || u <= 0) return 0;
  return Math.round(u * getQuotaPerUnit());
};

export const useUsersData = () => {
  const { t } = useTranslation();
  const [compactMode, setCompactMode] = useTableCompactMode('users');

  // State management
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [activePage, setActivePage] = useState(1);
  const [pageSize, setPageSize] = useState(ITEMS_PER_PAGE);
  const [searching, setSearching] = useState(false);
  const [studentView, setStudentView] = useState('all');
  const [studentRewardAmount, setStudentRewardAmount] = useState(0);
  const [studentRewardLoading, setStudentRewardLoading] = useState(false);
  const [groupOptions, setGroupOptions] = useState([]);
  const [userCount, setUserCount] = useState(0);

  // Modal states
  const [showAddUser, setShowAddUser] = useState(false);
  const [showEditUser, setShowEditUser] = useState(false);
  const [editingUser, setEditingUser] = useState({
    id: undefined,
  });

  // Form initial values
  const formInitValues = {
    searchKeyword: '',
    searchGroup: '',
  };

  // Form API reference
  const [formApi, setFormApi] = useState(null);

  // Get form values helper function
  const getFormValues = () => {
    const formValues = formApi ? formApi.getValues() : {};
    return {
      searchKeyword: formValues.searchKeyword || '',
      searchGroup: formValues.searchGroup || '',
    };
  };

  // Set user format with key field
  const setUserFormat = (users) => {
    for (let i = 0; i < users.length; i++) {
      users[i].key = users[i].id;
    }
    setUsers(users);
  };

  // Load users data
  const loadUsers = async (startIdx, pageSize) => {
    setLoading(true);
    const viewQuery =
      studentView && studentView !== 'all'
        ? `&student_view=${encodeURIComponent(studentView)}`
        : '';
    const res = await API.get(
      `/api/user/?p=${startIdx}&page_size=${pageSize}${viewQuery}`,
    );
    const { success, message, data } = res.data;
    if (success) {
      const newPageData = data.items;
      setActivePage(data.page);
      setUserCount(data.total);
      setUserFormat(newPageData);
    } else {
      showError(message);
    }
    setLoading(false);
  };

  // Search users with keyword and group
  const searchUsers = async (
    startIdx,
    pageSize,
    searchKeyword = null,
    searchGroup = null,
  ) => {
    // If no parameters passed, get values from form
    if (searchKeyword === null || searchGroup === null) {
      const formValues = getFormValues();
      searchKeyword = formValues.searchKeyword;
      searchGroup = formValues.searchGroup;
    }

    if (searchKeyword === '' && searchGroup === '') {
      // If keyword is blank, load files instead
      await loadUsers(startIdx, pageSize);
      return;
    }
    setSearching(true);
    const res = await API.get(
      `/api/user/search?keyword=${searchKeyword}&group=${searchGroup}&student_view=${encodeURIComponent(studentView)}&p=${startIdx}&page_size=${pageSize}`,
    );
    const { success, message, data } = res.data;
    if (success) {
      const newPageData = data.items;
      setActivePage(data.page);
      setUserCount(data.total);
      setUserFormat(newPageData);
    } else {
      showError(message);
    }
    setSearching(false);
  };

  // Manage user operations (promote, demote, enable, disable, delete)
  const manageUser = async (userId, action, record) => {
    // Trigger loading state to force table re-render
    setLoading(true);

    const res = await API.post('/api/user/manage', {
      id: userId,
      action,
    });

    const { success, message } = res.data;
    if (success) {
      showSuccess(t('操作成功完成！'));
      const user = res.data.data;

      // Create a new array and new object to ensure React detects changes
      const newUsers = users.map((u) => {
        if (u.id === userId) {
          if (action === 'delete') {
            return { ...u, DeletedAt: new Date() };
          }
          return {
            ...u,
            status: user.status,
            role: user.role,
            is_distributor: user.is_distributor,
            is_student: user.is_student,
            student_status: user.student_status,
          };
        }
        return u;
      });

      setUsers(newUsers);
    } else {
      showError(message);
    }

    setLoading(false);
  };

  const assignStudent = async (userId, rewardAmount) => {
    const uid = parseInt(String(userId), 10);
    if (!Number.isFinite(uid) || uid <= 0) {
      showError(t('请输入正确的用户ID'));
      return false;
    }
    const finalRewardAmount =
      rewardAmount === undefined || rewardAmount === null || rewardAmount === ''
        ? studentRewardAmount
        : Number(rewardAmount) || 0;
    const rewardQuota = Math.max(0, usdToQuota(finalRewardAmount));
    setLoading(true);
    try {
      const res = await API.post('/api/user/manage', {
        id: uid,
        action: 'set_student',
        reward_quota: rewardQuota,
      });
      const { success, message } = res.data;
      if (!success) {
        showError(message || t('操作失败，请重试'));
        return false;
      }
      showSuccess(t('已指定用户为学员'));
      await refresh(1);
      return true;
    } catch (error) {
      showError(t('操作失败，请重试'));
      return false;
    } finally {
      setLoading(false);
    }
  };

  const loadStudentRewardAmount = async () => {
    setStudentRewardLoading(true);
    try {
      const res = await API.get('/api/option/');
      const { success, data, message } = res.data;
      if (!success) {
        showError(message || t('加载赠送额度失败'));
        return;
      }
      const option = (data || []).find(
        (item) => item.key === 'StudentApprovalRewardQuota',
      );
      const quotaVal = parseInt(String(option?.value ?? '0'), 10);
      if (Number.isFinite(quotaVal) && quotaVal >= 0) {
        setStudentRewardAmount(quotaToUsd(quotaVal));
      } else {
        setStudentRewardAmount(0);
      }
    } catch (error) {
      showError(t('加载赠送额度失败'));
    } finally {
      setStudentRewardLoading(false);
    }
  };

  const saveStudentRewardAmount = async (rewardAmount) => {
    const rewardQuota = Math.max(0, usdToQuota(Number(rewardAmount) || 0));
    setStudentRewardLoading(true);
    try {
      const res = await API.put('/api/option/', {
        key: 'StudentApprovalRewardQuota',
        value: String(rewardQuota),
      });
      const { success, message } = res.data;
      if (!success) {
        showError(message || t('保存赠送额度失败'));
        return false;
      }
      setStudentRewardAmount(Number(rewardAmount) || 0);
      showSuccess(t('赠送额度已保存'));
      return true;
    } catch (error) {
      showError(t('保存赠送额度失败'));
      return false;
    } finally {
      setStudentRewardLoading(false);
    }
  };

  const handleStudentViewChange = (nextView) => {
    setStudentView(nextView);
    setActivePage(1);
    const { searchKeyword, searchGroup } = getFormValues();
    if (searchKeyword === '' && searchGroup === '') {
      loadUsers(0, pageSize).then();
    } else {
      searchUsers(0, pageSize, searchKeyword, searchGroup).then();
    }
  };

  const resetUserPasskey = async (user) => {
    if (!user) {
      return;
    }
    try {
      const res = await API.delete(`/api/user/${user.id}/reset_passkey`);
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('Passkey 已重置'));
      } else {
        showError(message || t('操作失败，请重试'));
      }
    } catch (error) {
      showError(t('操作失败，请重试'));
    }
  };

  const resetUserTwoFA = async (user) => {
    if (!user) {
      return;
    }
    try {
      const res = await API.delete(`/api/user/${user.id}/2fa`);
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('二步验证已重置'));
      } else {
        showError(message || t('操作失败，请重试'));
      }
    } catch (error) {
      showError(t('操作失败，请重试'));
    }
  };

  // Handle page change
  const handlePageChange = (page) => {
    setActivePage(page);
    const { searchKeyword, searchGroup } = getFormValues();
    if (searchKeyword === '' && searchGroup === '') {
      loadUsers(page, pageSize).then();
    } else {
      searchUsers(page, pageSize, searchKeyword, searchGroup).then();
    }
  };

  // Handle page size change
  const handlePageSizeChange = async (size) => {
    localStorage.setItem('page-size', size + '');
    setPageSize(size);
    setActivePage(1);
    loadUsers(activePage, size)
      .then()
      .catch((reason) => {
        showError(reason);
      });
  };

  // Handle table row styling for disabled/deleted users
  const handleRow = (record, index) => {
    if (record.DeletedAt !== null || record.status !== 1) {
      return {
        style: {
          background: 'var(--semi-color-disabled-border)',
        },
      };
    } else {
      return {};
    }
  };

  // Refresh data
  const refresh = async (page = activePage) => {
    const { searchKeyword, searchGroup } = getFormValues();
    if (searchKeyword === '' && searchGroup === '') {
      await loadUsers(page, pageSize);
    } else {
      await searchUsers(page, pageSize, searchKeyword, searchGroup);
    }
  };

  // Fetch groups data
  const fetchGroups = async () => {
    try {
      let res = await API.get(`/api/group/`);
      if (res === undefined) {
        return;
      }
      setGroupOptions(
        res.data.data.map((group) => ({
          label: group,
          value: group,
        })),
      );
    } catch (error) {
      showError(error.message);
    }
  };

  // Modal control functions
  const closeAddUser = () => {
    setShowAddUser(false);
  };

  const closeEditUser = () => {
    setShowEditUser(false);
    setEditingUser({
      id: undefined,
    });
  };

  // Initialize data on component mount
  useEffect(() => {
    loadUsers(0, pageSize)
      .then()
      .catch((reason) => {
        showError(reason);
      });
    fetchGroups().then();
  }, [studentView]);

  useEffect(() => {
    if (studentView === 'students') {
      loadStudentRewardAmount().then();
    }
  }, [studentView]);

  return {
    // Data state
    users,
    loading,
    activePage,
    pageSize,
    userCount,
    searching,
    studentView,
    studentRewardAmount,
    studentRewardLoading,
    groupOptions,

    // Modal state
    showAddUser,
    showEditUser,
    editingUser,
    setShowAddUser,
    setShowEditUser,
    setEditingUser,

    // Form state
    formInitValues,
    formApi,
    setFormApi,

    // UI state
    compactMode,
    setCompactMode,
    setStudentView: handleStudentViewChange,

    // Actions
    loadUsers,
    searchUsers,
    manageUser,
    assignStudent,
    setStudentRewardAmount,
    saveStudentRewardAmount,
    resetUserPasskey,
    resetUserTwoFA,
    handlePageChange,
    handlePageSizeChange,
    handleRow,
    refresh,
    closeAddUser,
    closeEditUser,
    getFormValues,

    // Translation
    t,
  };
};
