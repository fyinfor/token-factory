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

import React, { useEffect } from 'react';
import { useSearchParams } from 'react-router-dom';
import ChannelsTable from '../../components/table/channels';

const File = () => {
  const [searchParams] = useSearchParams();
  const editId = searchParams.get('editId');

  useEffect(() => {
    // 如果 URL 中有 editId 参数，自动打开编辑抽屉
    if (editId) {
      // 等待 ChannelsTable 加载完成
      const timer = setTimeout(() => {
        // 触发自定义事件来通知 ChannelsTable 打开编辑
        window.dispatchEvent(new CustomEvent('openChannelEdit', { detail: { id: parseInt(editId) } }));
      }, 100);
      return () => clearTimeout(timer);
    }
  }, [editId]);

  return (
    <div className='mt-[60px] px-2'>
      <ChannelsTable />
    </div>
  );
};

export default File;
