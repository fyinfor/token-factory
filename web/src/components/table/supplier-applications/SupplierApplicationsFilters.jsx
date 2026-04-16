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
import { Form, Select, Button } from '@douyinfe/semi-ui';

const SupplierApplicationsFilters = ({
  formInitValues,
  setFormApi,
  searchApplications,
  loadApplications,
  activePage,
  pageSize,
  loading,
  searching,
  t,
}) => {
  const statusOptions = [
    { label: t('全部'), value: 'all' },
    { label: t('待审核'), value: '0' },
    { label: t('审核通过'), value: '1' },
    { label: t('审核驳回'), value: '2' },
  ];

  return (
    <Form
      layout='horizontal'
      initValues={formInitValues}
      getFormApi={(api) => setFormApi(api)}
      style={{ display: 'flex', gap: '8px', alignItems: 'flex-end' }}
    >
      <Form.Select
        field='status'
        label={t('状态')}
        placeholder={t('选择状态')}
        optionList={statusOptions}
        showClear
        style={{ width: 150 }}
      />
      <Button
        type='primary'
        htmlType='submit'
        onClick={() => {
          searchApplications(activePage, pageSize);
        }}
        loading={loading || searching}
      >
        {t('查询')}
      </Button>
      <Button
        onClick={() => {
          loadApplications(1, pageSize);
        }}
        loading={loading || searching}
      >
        {t('重置')}
      </Button>
    </Form>
  );
};

export default SupplierApplicationsFilters;
