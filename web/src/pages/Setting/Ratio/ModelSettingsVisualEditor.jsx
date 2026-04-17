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

import React, { useEffect, useState } from 'react';
import { Tabs } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import ModelPricingEditor from './components/ModelPricingEditor';
import SupplierModelPricingEditor from './components/SupplierModelPricingEditor';
import { API } from '../../../helpers';

export default function ModelSettingsVisualEditor(props) {
  const { t } = useTranslation();
  const [pricingSuppliers, setPricingSuppliers] = useState([]);

  useEffect(() => {
    const loadSuppliers = async () => {
      try {
        const res = await API.get('/api/pricing');
        if (res?.data?.success) {
          setPricingSuppliers(res.data.channels || []);
        }
      } catch (error) {
        console.error('failed to load suppliers:', error);
      }
    };
    loadSuppliers();
  }, []);

  const extendedOptions = {
    ...props.options,
    __pricingChannels: pricingSuppliers,
  };

  return (
    <Tabs type='line' defaultActiveKey='global'>
      <Tabs.TabPane tab={t('全局模型定价')} itemKey='global'>
        <ModelPricingEditor options={props.options} refresh={props.refresh} />
      </Tabs.TabPane>
      <Tabs.TabPane tab={t('渠道模型定价')} itemKey='supplier'>
        <SupplierModelPricingEditor
          options={extendedOptions}
          refresh={props.refresh}
          candidateModelNames={props.candidateModelNames}
        />
      </Tabs.TabPane>
    </Tabs>
  );
}
