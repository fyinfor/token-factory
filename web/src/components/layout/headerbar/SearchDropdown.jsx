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

import React, { useState, useRef, useEffect } from 'react';
import { Input, Dropdown, Typography } from '@douyinfe/semi-ui';
import { IconSearch } from '@douyinfe/semi-icons';

const mockSearchData = [
  {
    month: '四月 2026',
    items: [
      { id: 1, name: 'Google: Gemma 4 31B', icon: '⬥', color: 'text-blue-500' },
      { id: 2, name: 'Qwen: Qwen3.6 Plus (free)', icon: '⬡', color: 'text-purple-500' },
      { id: 3, name: 'Z.ai: GLM 5V Turbo', icon: '⬢', color: 'text-gray-800 dark:text-white' },
      { id: 4, name: 'Arcee AI: Trinity Large Thinking', icon: '⬢', color: 'text-teal-500' },
      { id: 5, name: 'xAI: Grok 4.20 Multi-Agent', icon: '⚡', color: 'text-gray-800 dark:text-white' },
      { id: 6, name: 'xAI: Grok 4.20', icon: '⚡', color: 'text-gray-800 dark:text-white' },
    ],
  },
  {
    month: '三月 2026',
    items: [
      { id: 7, name: 'Google: Lyria 3 Pro Preview', icon: '⬥', color: 'text-blue-500' },
    ],
  },
];

const SearchDropdown = ({ isMobile }) => {
  const [searchValue, setSearchValue] = useState('');
  const [visible, setVisible] = useState(false);
  const [filteredData, setFilteredData] = useState(mockSearchData);
  const dropdownRef = useRef(null);
  const inputRef = useRef(null);

  useEffect(() => {
    if (searchValue.trim() === '') {
      setFilteredData(mockSearchData);
    } else {
      const filtered = mockSearchData
        .map((group) => ({
          ...group,
          items: group.items.filter((item) =>
            item.name.toLowerCase().includes(searchValue.toLowerCase())
          ),
        }))
        .filter((group) => group.items.length > 0);
      setFilteredData(filtered);
    }
  }, [searchValue]);

  useEffect(() => {
    const handleKeyDown = (event) => {
      if (event.key === '/') {
        event.preventDefault();
        inputRef.current?.focus();
        setVisible(true);
      }
      if (event.key === 'Escape') {
        setVisible(false);
        inputRef.current?.blur();
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => {
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, []);

  const handleItemClick = (item) => {
    console.log('Selected:', item);
    setVisible(false);
    setSearchValue('');
  };

  const renderDropdownContent = () => {
    return (
      <div className='w-80 md:w-96 max-h-96 overflow-y-auto'>
        {filteredData.length > 0 ? (
          filteredData.map((group) => (
            <div key={group.month} className='py-2'>
              <Typography.Text
                className='!px-4 !py-2 !text-xs !font-semibold !text-semi-color-text-2 dark:!text-gray-400 uppercase tracking-wider block'
              >
                {group.month}
              </Typography.Text>
              <div>
                {group.items.map((item) => (
                  <div
                    key={item.id}
                    onClick={() => handleItemClick(item)}
                    className='px-4 py-2.5 flex items-center gap-3 cursor-pointer hover:!bg-semi-color-fill-1 dark:hover:!bg-gray-700 transition-colors'
                  >
                    <span className={`text-lg ${item.color}`}>{item.icon}</span>
                    <Typography.Text className='!text-sm !font-medium !text-semi-color-text-0 dark:!text-gray-200'>
                      {item.name}
                    </Typography.Text>
                  </div>
                ))}
              </div>
            </div>
          ))
        ) : (
          <div className='px-4 py-8 text-center'>
            <Typography.Text className='!text-sm !text-semi-color-text-2 dark:!text-gray-400'>
              No results found
            </Typography.Text>
          </div>
        )}
      </div>
    );
  };

  return (
    <div className='relative' ref={dropdownRef}>
      <Dropdown
        visible={visible}
        onVisibleChange={setVisible}
        position='bottomLeft'
        trigger='custom'
        getPopupContainer={() => dropdownRef.current}
        render={
          <div className='!bg-semi-color-bg-overlay !border-semi-color-border !shadow-lg !rounded-lg dark:!bg-gray-800 dark:!border-gray-600'>
            {renderDropdownContent()}
          </div>
        }
      >
        <div className='relative'>
          <Input
            ref={inputRef}
            placeholder='Search'
            prefix={<IconSearch className='text-semi-color-text-2 dark:text-gray-400' />}
            suffix={
              <kbd className='px-1.5 py-0.5 text-xs font-semibold !text-semi-color-text-2 dark:!text-gray-400 !bg-semi-color-fill-0 dark:!bg-gray-700 border !border-semi-color-border dark:!border-gray-600 rounded'>
                /
              </kbd>
            }
            value={searchValue}
            onChange={setSearchValue}
            onFocus={() => setVisible(true)}
            className='!w-48 lg:!w-64 !h-9 !text-sm !bg-semi-color-fill-0 dark:!bg-gray-800/50 !border-semi-color-border dark:!border-gray-700 hover:!border-semi-color-primary dark:hover:!border-blue-400 focus:!border-semi-color-primary dark:focus:!border-blue-400'
            style={{ borderRadius: '6px', paddingRight: '10px' }}
          />
        </div>
      </Dropdown>
    </div>
  );
};

export default SearchDropdown;
