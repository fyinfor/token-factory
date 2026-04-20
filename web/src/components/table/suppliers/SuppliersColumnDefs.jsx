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
import { Button, Space, Tag } from '@douyinfe/semi-ui';
import { timestamp2string } from '../../../helpers';

export const getSuppliersColumns = (t, openEdit, handleDeactivate) => {
  return [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 80,
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      width: 100,
      render: (status) => {
        if (status === 1) {
          return <Tag color='green'>{t('已启用')}</Tag>;
        } else if (status === 3) {
          return <Tag color='red'>{t('已注销')}</Tag>;
        }
        return <Tag>{t('未知')}</Tag>;
      },
    },
    {
      title: t('用户名'),
      dataIndex: 'applicant_username',
      width: 100,
    },
    {
      title: t('企业/主体名称'),
      dataIndex: 'company_name',
      width: 200,
    },
    {
      title: t('供应商别名'),
      dataIndex: 'supplier_alias',
      width: 160,
      render: (text) => text || '-',
    },
    {
      title: t('统一社会信用代码'),
      dataIndex: 'credit_code',
      width: 180,
    },
    {
      title: t('法人/经营者姓名'),
      dataIndex: 'legal_representative',
      width: 150,
    },
    {
      title: t('企业规模'),
      dataIndex: 'company_size',
      width: 120,
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
      title: t('对接人微信/企业微信'),
      dataIndex: 'contact_wechat',
      width: 150,
    },
    {
      title: t('创建时间'),
      dataIndex: 'created_at',
      width: 160,
      render: (text) => timestamp2string(text),
    },
    {
      title: t('操作'),
      dataIndex: 'operate',
      width: 150,
      fixed: 'right',
      render: (text, record) => {
        const isDeactivated = record.status === 3;
        return (
          <Space>
            <Button
              theme='light'
              type='primary'
              size='small'
              disabled={isDeactivated}
              onClick={() => openEdit(record)}
            >
              {t('修改')}
            </Button>
            <Button
              theme='light'
              type='danger'
              size='small'
              disabled={isDeactivated}
              onClick={() => handleDeactivate(record)}
            >
              {t('注销')}
            </Button>
          </Space>
        );
      },
    },
  ];
};
