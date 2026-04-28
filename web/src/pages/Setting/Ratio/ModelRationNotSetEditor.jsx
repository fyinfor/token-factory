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
import { useNavigate } from 'react-router-dom';
import { API, isAdmin, isSupplier, showError } from '../../../helpers';
import { useTranslation } from 'react-i18next';
import { Button, Empty, Tabs } from '@douyinfe/semi-ui';
import {
  IllustrationNoAccess,
  IllustrationNoAccessDark,
} from '@douyinfe/semi-illustrations';
import ModelPricingEditor from './components/ModelPricingEditor';
import SupplierModelPricingEditor from './components/SupplierModelPricingEditor';

/**
 * ModelRatioNotSetEditor 「未设置模型」定价页：全局未设置 / 渠道未设置两个 Tab。
 * 管理员仍写 Option；供应商通过 `/api/user/supplier/pricing/*` 独立表存储。
 */
export default function ModelRatioNotSetEditor(props) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [enabledModels, setEnabledModels] = useState([]);
  const [pricingChannels, setPricingChannels] = useState([]);

  /** getAllEnabledModels 拉取当前启用模型列表，用于候选筛选。 */
  const getAllEnabledModels = async () => {
    try {
      const res = await API.get('/api/channel/models_enabled');
      const { success, message, data } = res.data;
      if (success) {
        setEnabledModels(data);
      } else {
        showError(message);
      }
    } catch (error) {
      console.error(t('获取启用模型失败:'), error);
      showError(error?.response?.data?.message || t('获取启用模型失败'));
    }
  };

  /**
   * 获取可配置定价的供应商渠道列表。
   */
  const getPricingSuppliers = async () => {
    try {
      const res = await API.get('/api/pricing');
      if (res?.data?.success) {
        setPricingChannels(res.data.channels || []);
      }
    } catch (error) {
      console.error(t('获取渠道商列表失败:'), error);
    }
  };

  useEffect(() => {
    getAllEnabledModels();
    getPricingSuppliers();
  }, []);

  const extendedOptions = {
    ...props.options,
    __pricingChannels: pricingChannels,
  };

  /**
   * saveSupplierGlobalUnsetPricingOutput 供应商「全局未设置模型」保存入口。
   */
  const saveSupplierGlobalUnsetPricingOutput = async (output) => {
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

  // 管理员或供应商均可访问；仅普通用户显示“需要供应商权限”提示。
  if (!isSupplier() && !isAdmin()) {
    return (
      <div className='py-4'>
        <div
          className='flex items-center justify-center'
          style={{ minHeight: 320 }}
        >
          <Empty
            image={<IllustrationNoAccess style={{ width: 180, height: 180 }} />}
            darkModeImage={
              <IllustrationNoAccessDark style={{ width: 180, height: 180 }} />
            }
            layout='horizontal'
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

  return (
    <Tabs type='line' defaultActiveKey='global_unset'>
      <Tabs.TabPane tab={t('全局未设置模型')} itemKey='global_unset'>
        <ModelPricingEditor
          options={props.options}
          refresh={props.refresh}
          candidateModelNames={enabledModels}
          filterMode='unset'
          allowAddModel={false}
          allowDeleteModel={false}
          showConflictFilter={false}
          listDescription={t(
            '此页面仅显示未设置价格或基础倍率的模型，设置后会自动从列表中移出',
          )}
          emptyTitle={t('没有未设置定价的模型')}
          emptyDescription={t('当前没有未设置定价的模型')}
          onSaveOutput={isSupplier() ? saveSupplierGlobalUnsetPricingOutput : undefined}
        />
      </Tabs.TabPane>
      <Tabs.TabPane tab={t('渠道未设置模型')} itemKey='supplier_unset'>
        <SupplierModelPricingEditor
          options={extendedOptions}
          refresh={props.refresh}
          candidateModelNames={enabledModels}
          filterMode='unset'
          listDescription={t('渠道未设置模型列表说明')}
          useSupplierPricingApi={isSupplier()}
        />
      </Tabs.TabPane>
    </Tabs>
  );
}
