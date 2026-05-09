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
import { API, isSupplier } from '../../../helpers';

export default function ModelSettingsVisualEditor(props) {
  const { t } = useTranslation();
  const [pricingSuppliers, setPricingSuppliers] = useState([]);

  /**
   * saveSupplierGlobalPricingOutput 供应商全局定价保存到后端表，不写全局 Option。
   */
  const saveSupplierGlobalPricingOutput = async (output) => {
    const res = await API.put('/api/user/supplier/pricing/global', {
      ModelPrice: output.ModelPrice || {},
      ModelRatio: output.ModelRatio || {},
      CompletionRatio: output.CompletionRatio || {},
      CacheRatio: output.CacheRatio || {},
      CreateCacheRatio: output.CreateCacheRatio || {},
      ImageRatio: output.ImageRatio || {},
      AudioRatio: output.AudioRatio || {},
      AudioCompletionRatio: output.AudioCompletionRatio || {},
    });
    if (!res?.data?.success) {
      throw new Error(res?.data?.message || t('保存失败'));
    }
  };

  useEffect(() => {
    const loadSuppliers = async () => {
      try {
        if (isSupplier()) {
          const channels = [];
          let page = 1;
          let total = 0;
          do {
            const res = await API.get(
              `/api/user/supplier/channels?p=${page}&page_size=1000`,
            );
            if (!res?.data?.success) break;
            const items = res.data.data?.items || [];
            total = res.data.data?.total || items.length;
            channels.push(
              ...items.map((item) => ({
                channel_id: item.id,
                channel_name: item.name,
                channel_no: item.channel_no,
              })),
            );
            page += 1;
          } while (channels.length < total);
          setPricingSuppliers(channels);
        } else {
          const res = await API.get('/api/channel/');
          if (res?.data?.success) {
            setPricingSuppliers(
              (res.data.data?.items || []).map((item) => ({
                channel_id: item.id,
                channel_name: item.name,
                channel_no: item.channel_no,
              })) || [],
            );
          }
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
        <ModelPricingEditor
          options={props.options}
          refresh={props.refresh}
          onSaveOutput={
            isSupplier() ? saveSupplierGlobalPricingOutput : undefined
          }
        />
      </Tabs.TabPane>
      <Tabs.TabPane tab={t('渠道模型定价')} itemKey='supplier'>
        <SupplierModelPricingEditor
          options={extendedOptions}
          refresh={props.refresh}
          candidateModelNames={props.candidateModelNames}
          useSupplierPricingApi={isSupplier()}
        />
      </Tabs.TabPane>
    </Tabs>
  );
}
