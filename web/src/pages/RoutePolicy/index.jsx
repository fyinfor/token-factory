import React from 'react';
import { useTranslation } from 'react-i18next';
import RoutePolicyCard from '../../components/settings/personal/cards/RoutePolicyCard';

const RoutePolicyPage = () => {
  const { t } = useTranslation();
  return (
    <div className='mt-[64px]'>
      <div className='px-2 md:px-8 py-4 md:py-6'>
        <RoutePolicyCard t={t} />
      </div>
    </div>
  );
};

export default RoutePolicyPage;
