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
import ModelsTable from '../../components/table/models';

const ModelPage = () => {
  const [searchParams] = useSearchParams();
  const editId = searchParams.get('id');
  const modelName = searchParams.get('name');

  useEffect(() => {
    // 如果 URL 中有 id 参数，自动打开编辑抽屉
    if (editId) {
      const timer = setTimeout(() => {
        window.dispatchEvent(new CustomEvent('openModelEdit', { detail: { id: parseInt(editId), name: modelName } }));
      }, 100);
      return () => clearTimeout(timer);
    }
  }, [editId, modelName]);

  return (
    <div className='mt-[60px] px-2'>
      <ModelsTable />
    </div>
  );
};

export default ModelPage;
