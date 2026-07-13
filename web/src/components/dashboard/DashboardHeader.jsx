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
import { RefreshCw, Search } from 'lucide-react';

const DashboardHeader = ({
  getGreeting,
  greetingVisible,
  showSearchModal,
  refresh,
  loading,
}) => {
  return (
    <div className='dashboard-glass-header flex items-center justify-between gap-4'>
      <h2
        className='dashboard-glass-header__title text-2xl font-semibold transition-opacity duration-1000 ease-in-out'
        style={{ opacity: greetingVisible ? 1 : 0 }}
      >
        {getGreeting}
      </h2>
      <div className='dashboard-glass-header__actions flex items-center gap-2'>
        <Button
          theme='borderless'
          type='tertiary'
          aria-label='Search dashboard data'
          icon={<Search size={17} />}
          onClick={showSearchModal}
          className='dashboard-glass-header__action dashboard-glass-header__action--search'
        />
        <Button
          theme='borderless'
          type='tertiary'
          aria-label='Refresh dashboard data'
          icon={<RefreshCw size={17} />}
          onClick={refresh}
          loading={loading}
          className='dashboard-glass-header__action dashboard-glass-header__action--refresh'
        />
      </div>
    </div>
  );
};

export default DashboardHeader;
