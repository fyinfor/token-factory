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
import { Button } from '@douyinfe/semi-ui';
import { Wallet, Plus } from 'lucide-react';
import { renderQuota } from '../../../helpers/display';

const WalletButton = ({ userState, isMobile, navigate, t }) => {
  if (!userState?.user) {
    return null;
  }

  const quota = userState.user.quota;

  const handleBalanceClick = () => {
    navigate('/console/topup');
  };

  const handleTopUpClick = (e) => {
    e.stopPropagation();
    navigate('/console/topup');
  };

  return (
    <div className='flex items-center gap-1'>
      <Button
        icon={<Wallet size={16} />}
        onClick={handleBalanceClick}
        theme='borderless'
        type='tertiary'
        className='!px-2 !py-1.5 !text-current focus:!bg-semi-color-fill-1 dark:focus:!bg-gray-700 !rounded-full !bg-semi-color-fill-0 dark:!bg-semi-color-fill-1 hover:!bg-semi-color-fill-1 dark:hover:!bg-semi-color-fill-2 !flex !items-center !gap-1.5'
      >
        {!isMobile && (
          <span className='text-sm font-medium truncate max-w-[6rem]'>
            {renderQuota(quota)}
          </span>
        )}
      </Button>
      {!isMobile && (
        <Button
          icon={<Plus size={14} />}
          onClick={handleTopUpClick}
          theme='borderless'
          type='tertiary'
          size='small'
          className='!px-1.5 !py-1 !text-current focus:!bg-semi-color-fill-1 dark:focus:!bg-gray-700 !rounded-full !bg-semi-color-fill-0 dark:!bg-semi-color-fill-1 hover:!bg-semi-color-fill-1 dark:hover:!bg-semi-color-fill-2 !text-xs'
        >
          <span className='text-xs font-medium'>{t('充值')}</span>
        </Button>
      )}
    </div>
  );
};

export default WalletButton;
