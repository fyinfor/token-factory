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
import { Modal, Typography, Checkbox } from '@douyinfe/semi-ui';
import { useTableCompactMode } from '../common/useTableCompactMode';

const DEFAULT_PAGE_SIZE = 50;

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
  const { t, i18n } = useTranslation();
  const [compactMode, setCompactMode] = useTableCompactMode('users');

  // State management
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [activePage, setActivePage] = useState(1);
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE);
  const [searching, setSearching] = useState(false);
  const [studentView, setStudentView] = useState('all');
  const [studentRewardAmount, setStudentRewardAmount] = useState(0);
  const [studentRewardLoading, setStudentRewardLoading] = useState(false);
  const [groupOptions, setGroupOptions] = useState([]);
  const [userCount, setUserCount] = useState(0);
  const [tagOptions, setTagOptions] = useState([]);
  const [selectedTag, setSelectedTag] = useState('');

  // Modal states
  const [showAddUser, setShowAddUser] = useState(false);
  const [showImportUsers, setShowImportUsers] = useState(false);
  const [showEditUser, setShowEditUser] = useState(false);
  const [editingUser, setEditingUser] = useState({
    id: undefined,
  });

  // Form initial values
  const formInitValues = {
    searchKeyword: '',
    searchRemark: '',
    searchGroup: '',
    searchTag: '',
  };

  // Form API reference
  const [formApi, setFormApi] = useState(null);

  // Get form values helper function
  const getFormValues = () => {
    const formValues = formApi ? formApi.getValues() : {};
    return {
      searchKeyword: formValues.searchKeyword || '',
      searchRemark: formValues.searchRemark || '',
      searchGroup: formValues.searchGroup || '',
      searchTag: formValues.searchTag || '',
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
  const loadUsers = async (startIdx, pageSize, tag = null) => {
    setLoading(true);
    const viewQuery =
      studentView && studentView !== 'all'
        ? `&student_view=${encodeURIComponent(studentView)}`
        : '';
    const tagFilter = tag || selectedTag;
    const tagQuery = tagFilter ? `&tag=${encodeURIComponent(tagFilter)}` : '';
    const res = await API.get(
      `/api/user/?p=${startIdx}&page_size=${pageSize}${viewQuery}${tagQuery}`,
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
    searchTag = null,
    searchRemark = null,
  ) => {
    // If no parameters passed, get values from form
    if (searchKeyword === null || searchGroup === null) {
      const formValues = getFormValues();
      searchKeyword = formValues.searchKeyword;
      searchGroup = formValues.searchGroup;
      searchTag = formValues.searchTag;
      searchRemark = formValues.searchRemark;
    }
    if (searchRemark === null) {
      searchRemark = getFormValues().searchRemark;
    }

    if (
      searchKeyword === '' &&
      searchGroup === '' &&
      !searchTag &&
      !selectedTag &&
      searchRemark === ''
    ) {
      // If keyword is blank, load files instead
      await loadUsers(startIdx, pageSize);
      return;
    }
    setSearching(true);
    const tagQuery =
      searchTag || selectedTag
        ? `&tag=${encodeURIComponent(searchTag || selectedTag)}`
        : '';
    const remarkQuery = searchRemark
      ? `&remark=${encodeURIComponent(searchRemark)}`
      : '';
    const res = await API.get(
      `/api/user/search?keyword=${encodeURIComponent(searchKeyword || '')}&group=${encodeURIComponent(searchGroup || '')}&student_view=${encodeURIComponent(studentView)}${tagQuery}${remarkQuery}&p=${startIdx}&page_size=${pageSize}`,
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
    let convertOrdinaryInvites = false;
    if (action === 'set_distributor') {
      try {
        const previewRes = await API.get(
          `/api/user/${userId}/ordinary-invite-preview`,
        );
        const preview = previewRes.data?.data;
        if (previewRes.data?.success && Number(preview?.total || 0) > 0) {
          let checked = false;
          const confirmed = await new Promise((resolve) => {
            Modal.confirm({
              title: t('设为代理'),
              content: (
                <div className='flex flex-col gap-3'>
                  <Typography.Text type='tertiary' size='small'>
                    {t(
                      '该用户有 {{total}} 位历史普通邀请，其中 {{convertible}} 位可转为代理下级。未勾选时只开通代理身份，不转换历史邀请。',
                      {
                        total: preview.total || 0,
                        convertible: preview.convertible || 0,
                      },
                    )}
                  </Typography.Text>
                  <Checkbox
                    disabled={Number(preview.convertible || 0) <= 0}
                    onChange={(event) => {
                      checked = Boolean(event.target.checked);
                    }}
                  >
                    {t('将可转换的历史邀请用户加入该代理名下')}
                  </Checkbox>
                </div>
              ),
              onOk: () => resolve({ confirmed: true, checked }),
              onCancel: () => resolve({ confirmed: false, checked: false }),
            });
          });
          if (!confirmed.confirmed) return;
          convertOrdinaryInvites = confirmed.checked;
        }
      } catch {
        showError(t('加载历史邀请信息失败'));
        return;
      }
    }
    // Trigger loading state to force table re-render
    setLoading(true);

    const res = await API.post('/api/user/manage', {
      id: userId,
      action,
      convert_ordinary_invites: convertOrdinaryInvites,
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
            real_name_verified:
              action === 'remove_real_name' ? false : u.real_name_verified,
            real_name_verified_at:
              action === 'remove_real_name' ? null : u.real_name_verified_at,
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
    const { searchKeyword, searchRemark, searchGroup, searchTag } =
      getFormValues();
    if (
      searchKeyword === '' &&
      searchRemark === '' &&
      searchGroup === '' &&
      !searchTag &&
      !selectedTag
    ) {
      loadUsers(0, pageSize).then();
    } else {
      searchUsers(
        0,
        pageSize,
        searchKeyword,
        searchGroup,
        searchTag,
        searchRemark,
      ).then();
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
    const { searchKeyword, searchRemark, searchGroup, searchTag } =
      getFormValues();
    if (
      searchKeyword === '' &&
      searchRemark === '' &&
      searchGroup === '' &&
      !searchTag &&
      !selectedTag
    ) {
      loadUsers(page, pageSize).then();
    } else {
      searchUsers(
        page,
        pageSize,
        searchKeyword,
        searchGroup,
        searchTag,
        searchRemark,
      ).then();
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
    const { searchKeyword, searchRemark, searchGroup, searchTag } =
      getFormValues();
    if (
      searchKeyword === '' &&
      searchRemark === '' &&
      searchGroup === '' &&
      !searchTag &&
      !selectedTag
    ) {
      await loadUsers(page, pageSize);
    } else {
      await searchUsers(
        page,
        pageSize,
        searchKeyword,
        searchGroup,
        searchTag,
        searchRemark,
      );
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

  // Fetch user tags
  const fetchUserTags = async () => {
    try {
      let res = await API.get(`/api/user/tags`);
      if (res === undefined) {
        return;
      }
      const { success, data } = res.data;
      if (success) {
        setTagOptions(
          (data || []).map((tag) => ({
            label: tag,
            value: tag,
          })),
        );
      }
    } catch (error) {
      showError(error.message);
    }
  };

  // Modal control functions
  const closeAddUser = () => {
    setShowAddUser(false);
  };

  const closeImportUsers = () => {
    setShowImportUsers(false);
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
    fetchUserTags().then();
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
    tagOptions,
    selectedTag,
    setSelectedTag,
    fetchUserTags,

    // Modal state
    showAddUser,
    showImportUsers,
    showEditUser,
    editingUser,
    setShowAddUser,
    setShowImportUsers,
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
    closeImportUsers,
    closeEditUser,
    getFormValues,

    // Translation
    t,
    language: i18n.language,
  };
};
