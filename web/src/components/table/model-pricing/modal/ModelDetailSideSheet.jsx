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
import { Button, SideSheet, Tabs, Typography } from '@douyinfe/semi-ui';
import { IconClose } from '@douyinfe/semi-icons';

import { API } from '../../../../helpers';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import ModelHeader from './components/ModelHeader';
import ApiDocsSidePanel from './components/ApiDocsSidePanel';
import ModelBasicInfo from './components/ModelBasicInfo';
import ModelChannelWorkspace from './components/ModelChannelWorkspace';

const { Text } = Typography;

const ModelDetailSideSheet = ({
  visible,
  onClose,
  modelData,
  groupRatio,
  currency,
  siteDisplayType,
  tokenUnit,
  displayPrice,
  vendorsMap,
  endpointMap,
  t,
  selectedGroup,
  blurPricing = false,
  showCostPrice = false,
  channelModelRatioMap = {},
  channelModelPriceMap = {},
  channelCompletionRatioMap = {},
  channelCacheRatioMap = {},
  channelCreateCacheRatioMap = {},
  channelImageRatioMap = {},
  channelImagePriceMap = {},
  channelAudioRatioMap = {},
  channelAudioCompletionRatioMap = {},
  channelVideoRatioMap = {},
  channelVideoCompletionRatioMap = {},
  channelVideoPriceMap = {},
  perfMetricsMap = {},
}) => {
  const isMobile = useIsMobile();
  const [activeSection, setActiveSection] = useState('general');
  const [channelMtrMap, setChannelMtrMap] = useState({});

  useEffect(() => {
    if (!visible) {
      setActiveSection('general');
    }
  }, [visible, modelData?.model_name]);

  useEffect(() => {
    if (
      !visible ||
      !modelData?.model_name ||
      !modelData?.channel_list?.length
    ) {
      setChannelMtrMap({});
      return;
    }

    const ids = modelData.channel_list
      .map((channel) => channel.channel_id)
      .filter((id) => id != null && id !== '');
    if (ids.length === 0) {
      setChannelMtrMap({});
      return;
    }

    let cancelled = false;
    const params = new URLSearchParams();
    params.set('model_name', modelData.model_name);
    params.set('channel_ids', ids.join(','));

    (async () => {
      try {
        const res = await API.get(
          `/api/channel/model-test-results?${params.toString()}`,
        );
        const { success, data } = res.data;
        if (!success || cancelled) return;

        const metrics = {};
        (data || []).forEach((row) => {
          metrics[String(row.channel_id)] = row;
        });
        if (!cancelled) {
          setChannelMtrMap(metrics);
        }
      } catch (error) {
        if (!cancelled) {
          setChannelMtrMap({});
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [visible, modelData?.model_name, modelData?.channel_list]);

  const channelProps = {
    modelData,
    channelMtrMap,
    displayPrice,
    currency,
    siteDisplayType,
    tokenUnit,
    t,
    selectedGroup,
    groupRatio,
    blurPricing,
    showCostPrice,
    channelModelRatioMap,
    channelModelPriceMap,
    channelCompletionRatioMap,
    channelCacheRatioMap,
    channelCreateCacheRatioMap,
    channelImageRatioMap,
    channelImagePriceMap,
    channelAudioRatioMap,
    channelAudioCompletionRatioMap,
    channelVideoRatioMap,
    channelVideoCompletionRatioMap,
    channelVideoPriceMap,
  };

  return (
    <SideSheet
      className='model-detail-sheet'
      placement='right'
      title={
        <ModelHeader modelData={modelData} vendorsMap={vendorsMap} t={t} />
      }
      bodyStyle={{
        padding: 0,
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
      }}
      visible={visible}
      width={isMobile ? '100%' : 800}
      closeIcon={
        <Button
          className='semi-button-tertiary semi-button-size-small semi-button-borderless'
          type='tertiary'
          theme='borderless'
          icon={<IconClose />}
          onClick={onClose}
        />
      }
      onCancel={onClose}
    >
      {!modelData ? (
        <div className='flex flex-1 items-center justify-center py-10'>
          <Text type='secondary'>{t('加载中...')}</Text>
        </div>
      ) : (
        <Tabs
          activeKey={activeSection}
          onChange={setActiveSection}
          type='button'
          className='model-detail-tabs flex min-h-0 flex-1 flex-col'
          contentStyle={{ minHeight: 0, flex: 1, overflow: 'hidden' }}
        >
          <Tabs.TabPane tab={t('通用与渠道')} itemKey='general'>
            <ModelChannelWorkspace
              {...channelProps}
              endpointMap={endpointMap}
              perfSummary={perfMetricsMap[modelData.model_name]}
            />
          </Tabs.TabPane>
          <Tabs.TabPane tab={t('基本信息')} itemKey='basic'>
            <div className='model-basic-content h-full overflow-y-auto p-3'>
              <ModelBasicInfo
                modelData={modelData}
                vendorsMap={vendorsMap}
                t={t}
              />
            </div>
          </Tabs.TabPane>
          <Tabs.TabPane tab={t('文档')} itemKey='docs'>
            <ApiDocsSidePanel
              visible
              embedded
              onClose={() => setActiveSection('general')}
              modelName={modelData.model_name}
              docIntroduction={modelData.doc_introduction}
              apiDocs={modelData.api_docs}
              t={t}
            />
          </Tabs.TabPane>
        </Tabs>
      )}
    </SideSheet>
  );
};

export default ModelDetailSideSheet;
