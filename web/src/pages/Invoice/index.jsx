/*
Copyright (C) 2025 QuantumNous
*/

import React from 'react';
import { Typography } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import InvoiceManagement from '../../components/settings/personal/cards/InvoiceManagement';

const { Title, Text } = Typography;

const InvoicePage = () => {
  const { t } = useTranslation();

  return (
    <div className='mt-[64px]'>
      <div className='px-2 md:px-8 py-4 md:py-6 max-w-6xl mx-auto'>
        <div className='mb-4 md:mb-6'>
          <Title heading={3}>{t('发票管理')}</Title>
          <Text type='tertiary'>{t('管理充值订单的开票申请与开票记录')}</Text>
        </div>
        <InvoiceManagement t={t} />
      </div>
    </div>
  );
};

export default InvoicePage;
