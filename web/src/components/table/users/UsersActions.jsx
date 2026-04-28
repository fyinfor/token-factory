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
import {
  Button,
  InputNumber,
  Modal,
  Select,
  Space,
  Typography,
} from '@douyinfe/semi-ui';
import { useState } from 'react';
import { API, showError } from '../../../helpers';

const UsersActions = ({
  setShowAddUser,
  t,
  studentView,
  studentRewardAmount,
  studentRewardLoading,
  setStudentRewardAmount,
  saveStudentRewardAmount,
  assignStudent,
}) => {
  const [assignVisible, setAssignVisible] = useState(false);
  const [searchingUser, setSearchingUser] = useState(false);
  const [candidateUsers, setCandidateUsers] = useState([]);
  const [selectedUserId, setSelectedUserId] = useState('');

  // Add new user
  const handleAddUser = () => {
    setShowAddUser(true);
  };

  const openAssignModal = () => {
    setCandidateUsers([]);
    setSelectedUserId('');
    setAssignVisible(true);
    loadAllUsersForSelect();
  };

  const loadAllUsersForSelect = async () => {
    setSearchingUser(true);
    try {
      const all = [];
      let page = 0;
      let total = 0;
      do {
        const res = await API.get(`/api/user/?p=${page}&page_size=100`);
        const { success, message, data } = res.data;
        if (!success) {
          showError(message || t('加载用户失败'));
          return;
        }
        const items = data?.items || [];
        total = data?.total || 0;
        all.push(...items);
        page += 1;
      } while (all.length < total);
      setCandidateUsers(all);
    } catch (error) {
      showError(t('加载用户失败'));
    } finally {
      setSearchingUser(false);
    }
  };

  const handleSelectSearch = async (keyword) => {
    const kw = String(keyword || '').trim();
    if (!kw) {
      loadAllUsersForSelect();
      return;
    }
    setSearchingUser(true);
    try {
      const res = await API.get(
        `/api/user/search?keyword=${encodeURIComponent(kw)}&p=0&page_size=50`,
      );
      const { success, message, data } = res.data;
      if (!success) {
        showError(message || t('搜索用户失败'));
        return;
      }
      setCandidateUsers(data?.items || []);
    } catch (error) {
      showError(t('搜索用户失败'));
    } finally {
      setSearchingUser(false);
    }
  };

  const handleAssignConfirm = async () => {
    const ok = await assignStudent(selectedUserId);
    if (ok) {
      setAssignVisible(false);
    }
  };

  return (
    <>
      <div className='flex gap-2 w-full md:w-auto order-2 md:order-1 items-center'>
        {studentView === 'all' && (
          <Button
            className='w-full md:w-auto'
            onClick={handleAddUser}
            size='small'
          >
            {t('添加用户')}
          </Button>
        )}

        {studentView === 'students' && (
          <>
            <Space spacing={6} align='center'>
              <Typography.Text type='tertiary' size='small'>
                {t('赠送金额')}
              </Typography.Text>
              <InputNumber
                size='small'
                min={0}
                precision={2}
                suffix='USD'
                value={studentRewardAmount}
                onChange={(val) => setStudentRewardAmount(Number(val) || 0)}
              />
              <Button
                size='small'
                loading={studentRewardLoading}
                onClick={() => saveStudentRewardAmount(studentRewardAmount)}
              >
                {t('保存额度')}
              </Button>
            </Space>
            <Button
              className='w-full md:w-auto'
              onClick={openAssignModal}
              size='small'
            >
              {t('添加学员')}
            </Button>
          </>
        )}
      </div>

      <Modal
        visible={assignVisible}
        title={t('添加学员')}
        onCancel={() => setAssignVisible(false)}
        onOk={handleAssignConfirm}
        okText={t('确认')}
        cancelText={t('取消')}
        confirmLoading={searchingUser}
      >
        <div className='space-y-3'>
          <div>
            <Typography.Text size='small' type='tertiary'>
              {t('选择用户')}
            </Typography.Text>
            <Select
              style={{ width: '100%', marginTop: 6 }}
              placeholder={t('下拉选择全部用户，支持输入手机号/用户名搜索')}
              value={selectedUserId}
              filter
              loading={searchingUser}
              searchPosition='dropdown'
              onSearch={handleSelectSearch}
              optionList={candidateUsers.map((u) => ({
                label: `${u.username} (${u.phone || '—'}) #${u.id}`,
                value: u.id,
              }))}
              onChange={(val) => setSelectedUserId(val)}
            />
          </div>
        </div>
      </Modal>
    </>
  );
};

export default UsersActions;
