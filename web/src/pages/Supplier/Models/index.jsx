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
import { Button, Empty } from '@douyinfe/semi-ui';
import { IllustrationNoAccess, IllustrationNoAccessDark } from '@douyinfe/semi-illustrations';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import ModelsTable from '../../../components/table/models';
import { isSupplier } from '../../../helpers/utils';

const SupplierModelsContent = () => {
  return (
    <div className='mt-[60px] px-2'>
      <ModelsTable apiBasePath='/api/user/supplier/models' />
    </div>
  );
};

const SupplierModelsPage = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();

  if (!isSupplier()) {
    return (
      <div className='mt-[60px] px-2'>
        <div className='flex items-center justify-center' style={{ minHeight: 'calc(100vh - 360px)' }}>
          <Empty
            image={<IllustrationNoAccess style={{ width: 200, height: 200 }} />}
            darkModeImage={<IllustrationNoAccessDark style={{ width: 200, height: 200 }} />}
            layout="horizontal"
            title={t('需要供应商权限')}
            description={t('您需要先成为供应商才能访问此页面。')}
          >
            <Button
              theme='solid'
              type='primary'
              size='large'
              className='!rounded-md mt-4'
              style={{ fontWeight: 500 }}
              onClick={() => navigate('/console/supplier/apply')}
            >
              {t('前往申请')}
            </Button>
          </Empty>
        </div>
      </div>
    );
  }

  return <SupplierModelsContent />;
};

export default SupplierModelsPage;
