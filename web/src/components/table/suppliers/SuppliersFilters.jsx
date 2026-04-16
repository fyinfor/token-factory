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
import { Form, Button } from '@douyinfe/semi-ui';

const SuppliersFilters = ({
  formInitValues,
  setFormApi,
  searchSuppliers,
  loadSuppliers,
  activePage,
  pageSize,
  loading,
  searching,
  t,
}) => {
  return (
    <Form
      layout='horizontal'
      initValues={formInitValues}
      getFormApi={(api) => setFormApi(api)}
      style={{ display: 'flex', gap: '8px', alignItems: 'flex-end' }}
    >
      <Form.Input
        field='company_name'
        label={t('企业/主体名称')}
        placeholder={t('请输入企业/主体名称')}
        showClear
        style={{ width: 200 }}
      />
      <Form.Select
        field='status'
        label={t('状态')}
        placeholder={t('请选择状态')}
        style={{ width: 150 }}
        showClear
        optionList={[
          { label: t('已启用'), value: 1 },
          { label: t('已注销'), value: 3 },
        ]}
      />
      <Button
        type='primary'
        htmlType='submit'
        onClick={() => {
          searchSuppliers(activePage, pageSize);
        }}
        loading={loading || searching}
      >
        {t('查询')}
      </Button>
      <Button
        onClick={() => {
          loadSuppliers(1, pageSize);
        }}
        loading={loading || searching}
      >
        {t('重置')}
      </Button>
    </Form>
  );
};

export default SuppliersFilters;
