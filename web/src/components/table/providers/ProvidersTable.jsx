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

import React, { useMemo } from 'react';
import { Empty, Tag, Button, Popconfirm, Space } from '@douyinfe/semi-ui';
import { IconEdit, IconDelete } from '@douyinfe/semi-icons';
import CardTable from '../../common/ui/CardTable';
import {
  IllustrationNoResult,
  IllustrationNoResultDark,
} from '@douyinfe/semi-illustrations';
import { getLobeHubIcon } from '../../../helpers/render';

const ProvidersTable = ({
  providers,
  loading,
  activePage,
  pageSize,
  providerCount,
  handlePageChange,
  handlePageSizeChange,
  setEditingProvider,
  setShowEdit,
  deleteProvider,
  t,
}) => {
  const columns = useMemo(
    () => [
      {
        title: 'ID',
        dataIndex: 'id',
        key: 'id',
        width: 70,
      },
      {
        title: t('图标'),
        dataIndex: 'icon',
        key: 'icon',
        width: 70,
        render: (text) => {
          if (!text) return '-';
          const icon = getLobeHubIcon(text, 24);
          return icon || <span className='text-xs text-gray-400'>{text}</span>;
        },
      },
      {
        title: t('供应商名称'),
        dataIndex: 'name',
        key: 'name',
        width: 180,
        render: (text) => (
          <span className='font-medium'>{text}</span>
        ),
      },
      {
        title: t('描述'),
        dataIndex: 'description',
        key: 'description',
        ellipsis: true,
        render: (text) => text || '-',
      },
      {
        title: t('状态'),
        dataIndex: 'status',
        key: 'status',
        width: 100,
        render: (status) => (
          <Tag color={status === 1 ? 'green' : 'grey'} size='small'>
            {status === 1 ? t('启用') : t('禁用')}
          </Tag>
        ),
      },
      {
        title: t('创建时间'),
        dataIndex: 'created_time',
        key: 'created_time',
        width: 180,
        render: (ts) => {
          if (!ts) return '-';
          return new Date(ts * 1000).toLocaleString();
        },
      },
      {
        title: t('操作'),
        dataIndex: 'operate',
        key: 'operate',
        width: 150,
        fixed: 'right',
        render: (_, record) => (
          <Space>
            <Button
              theme='light'
              type='tertiary'
              size='small'
              icon={<IconEdit />}
              onClick={() => {
                setEditingProvider(record);
                setShowEdit(true);
              }}
            >
              {t('编辑')}
            </Button>
            <Popconfirm
              title={t('确认删除')}
              content={t('确定要删除此供应商吗？删除后不可恢复。')}
              onConfirm={() => deleteProvider(record.id)}
              okType='danger'
            >
              <Button
                theme='light'
                type='danger'
                size='small'
                icon={<IconDelete />}
              >
                {t('删除')}
              </Button>
            </Popconfirm>
          </Space>
        ),
      },
    ],
    [t, setEditingProvider, setShowEdit, deleteProvider],
  );

  return (
    <CardTable
      columns={columns}
      dataSource={providers}
      scroll={{ x: 'max-content' }}
      pagination={{
        currentPage: activePage,
        pageSize: pageSize,
        total: providerCount,
        showSizeChanger: true,
        pageSizeOptions: [10, 20, 50, 100],
        onPageSizeChange: handlePageSizeChange,
        onPageChange: handlePageChange,
      }}
      hidePagination={true}
      loading={loading}
      rowKey='id'
      empty={
        <Empty
          image={<IllustrationNoResult style={{ width: 150, height: 150 }} />}
          darkModeImage={
            <IllustrationNoResultDark style={{ width: 150, height: 150 }} />
          }
          description={t('搜索无结果')}
          style={{ padding: 30 }}
        />
      }
      className='rounded-xl overflow-hidden'
      size='middle'
    />
  );
};

export default ProvidersTable;
