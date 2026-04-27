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
import { getSupplierApplicationsColumns } from './SupplierApplicationsColumnDefs';
import { useIsMobile } from '../../../hooks/common/useIsMobile';

const { Text } = Typography;

const SupplierApplicationsTable = ({
  applications,
  loading,
  t,
  openReview,
  compactMode,
}) => {
  const isMobile = useIsMobile();
  const columns = getSupplierApplicationsColumns(t, openReview);
  const statusColumn = columns.find((col) => col.dataIndex === 'status');
  const createdAtColumn = columns.find((col) => col.dataIndex === 'created_at');
  const operateColumn = columns.find((col) => col.dataIndex === 'operate');

  if (isMobile) {
    return (
      <div className='space-y-2'>
        {applications.map((app) => (
          <div
            key={app.id}
            className='border rounded p-3 bg-white dark:bg-gray-800'
          >
            <div className='space-y-2'>
              <div className='flex justify-between items-center'>
                <Text strong>ID: {app.id}</Text>
                {statusColumn?.render ? statusColumn.render(app.status, app) : null}
              </div>
              <div>
                <Text type='secondary'>{t('企业/主体名称')}:</Text>
                <div>{app.company_name}</div>
              </div>
              <div>
                <Text type='secondary'>{t('统一社会信用代码')}:</Text>
                <div>{app.credit_code}</div>
              </div>
              <div>
                <Text type='secondary'>{t('法人/经营者姓名')}:</Text>
                <div>{app.legal_representative}</div>
              </div>
              <div>
                <Text type='secondary'>{t('对接人姓名')}:</Text>
                <div>{app.contact_name}</div>
              </div>
              <div>
                <Text type='secondary'>{t('对接人手机号')}:</Text>
                <div>{app.contact_mobile}</div>
              </div>
              <div>
                <Text type='secondary'>{t('申请时间')}:</Text>
                <div>{createdAtColumn?.render ? createdAtColumn.render(app.created_at) : '-'}</div>
              </div>
              <div className='mt-2'>
                {operateColumn?.render ? operateColumn.render(null, app) : null}
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
      dataSource={applications}
      loading={loading}
      pagination={false}
      size={compactMode ? 'small' : 'default'}
      scroll={{ x: 1200 }}
    />
  );
};

export default SupplierApplicationsTable;
