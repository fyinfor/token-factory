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
import { Button, Form, Space } from '@douyinfe/semi-ui';
import { IconPlus, IconSearch } from '@douyinfe/semi-icons';
import CardPro from '../../common/ui/CardPro';
import ProvidersTable from './ProvidersTable';
import EditVendorModal from '../models/modals/EditVendorModal';
import { useProvidersData } from '../../../hooks/providers/useProvidersData';
import { useIsMobile } from '../../../hooks/common/useIsMobile';
import { createCardProPagination } from '../../../helpers/utils';

const ProvidersPage = () => {
  const providersData = useProvidersData();
  const isMobile = useIsMobile();

  const {
    showEdit,
    editingProvider,
    closeEdit,
    refresh,
    setEditingProvider,
    setShowEdit,
    formInitValues,
    setFormApi,
    searchProviders,
    loading,
    searching,
    t,
  } = providersData;

  return (
    <>
      <EditVendorModal
        visible={showEdit}
        handleClose={closeEdit}
        editingVendor={showEdit ? editingProvider : { id: undefined }}
        refresh={refresh}
      />

      <CardPro
        type='type1'
        actionsArea={
          <div className='flex flex-col md:flex-row justify-between items-center gap-2 w-full'>
            <div className='flex gap-2 order-2 md:order-1'>
              <Button
                theme='solid'
                type='primary'
                size='default'
                icon={<IconPlus />}
                onClick={() => {
                  setEditingProvider({ id: undefined });
                  setShowEdit(true);
                }}
              >
                {t('新增供应商')}
              </Button>
            </div>

            <div className='w-full md:w-auto order-1 md:order-2'>
              <Form
                layout='horizontal'
                initValues={formInitValues}
                getFormApi={setFormApi}
                onSubmit={searchProviders}
              >
                <Space>
                  <Form.Input
                    field='searchKeyword'
                    placeholder={t('搜索供应商名称或描述')}
                    showClear
                    style={{ width: isMobile ? '100%' : 240 }}
                    onEnterPress={searchProviders}
                  />
                  <Button
                    icon={<IconSearch />}
                    htmlType='submit'
                    loading={loading || searching}
                    theme='light'
                  >
                    {t('搜索')}
                  </Button>
                </Space>
              </Form>
            </div>
          </div>
        }
        paginationArea={createCardProPagination({
          currentPage: providersData.activePage,
          pageSize: providersData.pageSize,
          total: providersData.providerCount,
          onPageChange: providersData.handlePageChange,
          onPageSizeChange: providersData.handlePageSizeChange,
          isMobile: isMobile,
          t: providersData.t,
        })}
        t={providersData.t}
      >
        <ProvidersTable {...providersData} />
      </CardPro>
    </>
  );
};

export default ProvidersPage;
