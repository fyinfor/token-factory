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

import React, { useState, useEffect } from 'react';
import { SideSheet, Typography, Button } from '@douyinfe/semi-ui';
import { IconClose } from '@douyinfe/semi-icons';

import { API } from '../../../../helpers';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import ModelHeader from './components/ModelHeader';
import ModelBasicInfo from './components/ModelBasicInfo';
import ModelEndpoints from './components/ModelEndpoints';
import ModelPricingTable from './components/ModelPricingTable';
import ModelChannelList from './components/ModelChannelList';

const { Text } = Typography;

const ModelDetailSideSheet = ({
  visible,
  onClose,
  modelData,
  groupRatio,
  groupModelPrice,
  groupModelRatio,
  currency,
  siteDisplayType,
  tokenUnit,
  displayPrice,
  showRatio,
  usableGroup,
  vendorsMap,
  endpointMap,
  autoGroups,
  t,
  selectedGroup,
}) => {
  const isMobile = useIsMobile();
  /**
   * channel_id -> 单测/运营展示 DTO（打开详情时按需拉取，不并入 /pricing）
   */
  const [channelMtrMap, setChannelMtrMap] = useState({});

  useEffect(() => {
    if (!visible || !modelData?.model_name || !modelData?.channel_list?.length) {
      setChannelMtrMap({});
      return;
    }
    const ids = modelData.channel_list
      .map((c) => c.channel_id)
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
        if (!success || cancelled) {
          return;
        }
        const m = {};
        (data || []).forEach((row) => {
          m[String(row.channel_id)] = row;
        });
        if (!cancelled) {
          setChannelMtrMap(m);
        }
      } catch (e) {
        if (!cancelled) {
          setChannelMtrMap({});
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [visible, modelData?.model_name, modelData?.channel_list]);

  return (
    <SideSheet
      placement='right'
      title={
        <ModelHeader modelData={modelData} vendorsMap={vendorsMap} t={t} />
      }
      bodyStyle={{
        padding: '0',
        display: 'flex',
        flexDirection: 'column',
        borderBottom: '1px solid var(--semi-color-border)',
      }}
      visible={visible}
      width={isMobile ? '100%' : 600}
      closeIcon={
        <Button
          className='semi-button-tertiary semi-button-size-small semi-button-borderless'
          type='button'
          icon={<IconClose />}
          onClick={onClose}
        />
      }
      onCancel={onClose}
    >
      <div className='p-2'>
        {!modelData && (
          <div className='flex justify-center items-center py-10'>
            <Text type='secondary'>{t('加载中...')}</Text>
          </div>
        )}
        {modelData && (
          <>
            <ModelBasicInfo
              modelData={modelData}
              vendorsMap={vendorsMap}
              t={t}
            />
            <ModelEndpoints
              modelData={modelData}
              endpointMap={endpointMap}
              t={t}
            />
            <ModelChannelList
              modelData={modelData}
              channelMtrMap={channelMtrMap}
              displayPrice={displayPrice}
              currency={currency}
              siteDisplayType={siteDisplayType}
              tokenUnit={tokenUnit}
              t={t}
              selectedGroup={selectedGroup}
              groupRatio={groupRatio}
            />
            <ModelPricingTable
              modelData={modelData}
              groupRatio={groupRatio}
              groupModelPrice={groupModelPrice}
              groupModelRatio={groupModelRatio}
              currency={currency}
              siteDisplayType={siteDisplayType}
              tokenUnit={tokenUnit}
              displayPrice={displayPrice}
              showRatio={showRatio}
              usableGroup={usableGroup}
              autoGroups={autoGroups}
              t={t}
            />
          </>
        )}
      </div>
    </SideSheet>
  );
};

export default ModelDetailSideSheet;
