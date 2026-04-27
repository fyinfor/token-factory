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
import { Button, Tag, Typography, Popconfirm } from '@douyinfe/semi-ui';
import { timestamp2string } from '../../../helpers';

const { Text } = Typography;

export const getSupplierApplicationsColumns = (t, openReview) => {
  const getStatusTag = (status) => {
    const statusMap = {
      0: { color: 'orange', text: t('待审核') },
      1: { color: 'green', text: t('审核通过') },
      2: { color: 'red', text: t('审核驳回') },
    };
    const statusInfo = statusMap[status] || { color: 'grey', text: t('未知') };
    return <Tag color={statusInfo.color}>{statusInfo.text}</Tag>;
  };

  return [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 80,
    },
    {
      title: t('企业/主体名称'),
      dataIndex: 'company_name',
      width: 200,
    },
    {
      title: t('企业Logo'),
      dataIndex: 'company_logo_url',
      width: 120,
      render: (text) => text ? (
        <img
          src={text}
          alt={t('企业Logo')}
          style={{ width: 32, height: 32, borderRadius: 6, objectFit: 'cover' }}
        />
      ) : '-',
    },
    {
      title: t('统一社会信用代码'),
      dataIndex: 'credit_code',
      width: 180,
    },
    {
      title: t('供应商别名'),
      dataIndex: 'supplier_alias',
      width: 160,
      render: (text) => text || '-',
    },
    {
      title: t('法人/经营者姓名'),
      dataIndex: 'legal_representative',
      width: 150,
    },
    {
      title: t('对接人姓名'),
      dataIndex: 'contact_name',
      width: 120,
    },
    {
      title: t('对接人手机号'),
      dataIndex: 'contact_mobile',
      width: 130,
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      width: 100,
      render: (text, record) => getStatusTag(record.status),
    },
    {
      title: t('申请时间'),
      dataIndex: 'created_at',
      width: 160,
      render: (text) => timestamp2string(text),
    },
    {
      title: t('操作'),
      dataIndex: 'operate',
      width: 100,
      fixed: 'right',
      render: (text, record) => {
        const isReviewed = record.status !== 0;
        return (
          <div>
            <Button
              theme='light'
              type='primary'
              size='small'
              onClick={() => openReview(record)}
              disabled={isReviewed}
            >
              {t('审批')}
            </Button>
          </div>
        );
      },
    },
  ];
};
