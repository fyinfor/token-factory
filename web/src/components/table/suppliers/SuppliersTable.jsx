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
import { Table, Typography } from '@douyinfe/semi-ui';
import { getSuppliersColumns } from './SuppliersColumnDefs';
import { useIsMobile } from '../../../hooks/common/useIsMobile';

const { Text } = Typography;

const SuppliersTable = ({
  suppliers,
  loading,
  t,
  openEdit,
  handleDeactivate,
  compactMode,
}) => {
  const isMobile = useIsMobile();
  const columns = getSuppliersColumns(t, openEdit, handleDeactivate);

  if (isMobile) {
    return (
      <div className='space-y-2'>
        {suppliers.map((supplier) => (
          <div
            key={supplier.id}
            className='border rounded p-3 bg-white dark:bg-gray-800'
          >
            <div className='space-y-2'>
              <div className='flex justify-between items-center'>
                <Text strong>ID: {supplier.id}</Text>
                {columns[1].render(supplier.status)}
              </div>
              <div>
                <Text type='secondary'>{t('用户名')}:</Text>
                <div>{supplier.applicant_username}</div>
              </div>
              <div>
                <Text type='secondary'>{t('企业/主体名称')}:</Text>
                <div>{supplier.company_name}</div>
              </div>
              <div>
                <Text type='secondary'>{t('统一社会信用代码')}:</Text>
                <div>{supplier.credit_code}</div>
              </div>
              <div>
                <Text type='secondary'>{t('法人/经营者姓名')}:</Text>
                <div>{supplier.legal_representative}</div>
              </div>
              <div>
                <Text type='secondary'>{t('对接人姓名')}:</Text>
                <div>{supplier.contact_name}</div>
              </div>
              <div>
                <Text type='secondary'>{t('对接人手机号')}:</Text>
                <div>{supplier.contact_mobile}</div>
              </div>
              <div>
                <Text type='secondary'>{t('创建时间')}:</Text>
                <div>{columns[9].render(supplier.created_at)}</div>
              </div>
              <div className='mt-2'>
                {columns[10].render(null, supplier)}
              </div>
            </div>
          </div>
        ))}
      </div>
    );
  }

  return (
    <Table
      columns={columns}
      dataSource={suppliers}
      loading={loading}
      pagination={false}
      size={compactMode ? 'small' : 'default'}
      scroll={{ x: 1400 }}
    />
  );
};

export default SuppliersTable;
