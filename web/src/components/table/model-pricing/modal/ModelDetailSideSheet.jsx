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

import React, { useCallback, useEffect, useState } from 'react';
import {
  Button,
  SideSheet,
  Skeleton,
  Tabs,
  Typography,
} from '@douyinfe/semi-ui';
import { IconClose } from '@douyinfe/semi-icons';

import { API } from '../../../../helpers';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import ModelHeader from './components/ModelHeader';
import ModelBasicInfo from './components/ModelBasicInfo';
import ModelChannelWorkspace from './components/ModelChannelWorkspace';
import { MODEL_PRICE_MAX_DECIMALS } from '../utils/priceDisplay';

const { Text } = Typography;

const ModelDetailContentSkeleton = () => (
  <div
    className='model-channel-workspace grid h-full min-h-0 grid-cols-1 grid-rows-[auto_minmax(0,1fr)] overflow-hidden md:grid-cols-[230px_minmax(0,1fr)] md:grid-rows-1'
    aria-hidden='true'
  >
    <aside className='channel-selector-pane max-h-64 min-h-0 border-b p-3 md:h-full md:max-h-none md:border-b-0 md:border-r'>
      <Skeleton
        loading
        active
        placeholder={
          <div className='space-y-3'>
            {[0, 1, 2].map((item) => (
              <Skeleton.Title
                key={item}
                style={{ width: '100%', height: 92, borderRadius: 8 }}
              />
            ))}
          </div>
        }
      />
    </aside>
    <section className='channel-detail-content h-full min-h-0 overflow-hidden p-3'>
      <Skeleton
        loading
        active
        placeholder={
          <div className='space-y-4'>
            <Skeleton.Title style={{ width: '38%', height: 22 }} />
            <Skeleton.Paragraph rows={3} />
            <Skeleton.Title style={{ width: '52%', height: 22 }} />
            <Skeleton.Paragraph rows={6} />
          </div>
        }
      />
    </section>
  </div>
);

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
  hotChannelScoreMap = new Map(),
}) => {
  const isMobile = useIsMobile();
  const [activeSection, setActiveSection] = useState('general');
  const [channelMtrMap, setChannelMtrMap] = useState({});
  const [contentReady, setContentReady] = useState(false);
  const sideSheetDisplayPrice = useCallback(
    (usdPrice) =>
      displayPrice(usdPrice, { precision: MODEL_PRICE_MAX_DECIMALS }),
    [displayPrice],
  );

  useEffect(() => {
    if (!visible) {
      setActiveSection('general');
      setContentReady(false);
    }
  }, [visible, modelData?.model_name]);

  useEffect(() => {
    if (
      !contentReady ||
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
  }, [contentReady, modelData?.model_name, modelData?.channel_list]);

  const channelProps = {
    modelData,
    channelMtrMap,
    displayPrice: sideSheetDisplayPrice,
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
    hotChannelScoreMap,
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
      afterVisibleChange={setContentReady}
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
            {contentReady ? (
              <ModelChannelWorkspace
                {...channelProps}
                endpointMap={endpointMap}
              />
            ) : (
              <ModelDetailContentSkeleton />
            )}
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
        </Tabs>
      )}
    </SideSheet>
  );
};

export default ModelDetailSideSheet;
