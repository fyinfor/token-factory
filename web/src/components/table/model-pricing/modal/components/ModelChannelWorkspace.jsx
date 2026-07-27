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
import React, { useEffect, useMemo, useState } from 'react';
import { Tag, Tooltip, Typography } from '@douyinfe/semi-ui';

import {
  computeChannelBillingRates,
  costDiscountMultiplier,
  formatVideoResolutionDisplayLabel,
  getSupplierTypeLabel,
  getUsedGroupContext,
  markupRateFromPercent,
} from '../../../../../helpers';
import {
  fetchPerfMetrics,
  perfQueryResultToSummary,
} from '../../../../../helpers/perfMetrics';
import {
  convertTierPriceToUSD,
  getCurrencyRatesFromStatus,
  normalizeCurrency,
} from '../../../../../pages/Setting/Ratio/utils/requestTierPricing';
import {
  findTierPriceAtBand,
  getRequestTierPricing,
} from '../../view/card/tierUtils';
import ModelChannelList from './ModelChannelList';
import ModelEndpoints from './ModelEndpoints';
import ModelPerfPanel from './ModelPerfPanel';
import { getChannelHeatKey } from '../../utils/modelHeat';
import { formatPriceRatioFromDiscount } from '../../utils/discount';

const { Text } = Typography;

const getChannelKey = (channel, index) =>
  String(
    channel?.channel_id ??
      `${channel?.supplier_application_id || 'supplier'}-${index}`,
  );

const getChannelPriceScore = (channel, modelData) => {
  const quotaType = channel?.quota_type ?? modelData?.quota_type;
  const channelBase = Number(
    quotaType === 1 ? channel?.model_price : channel?.model_ratio,
  );
  const globalBase = Number(
    quotaType === 1 ? modelData?.model_price : modelData?.model_ratio,
  );
  if (!Number.isFinite(channelBase)) return Number.POSITIVE_INFINITY;

  const costDiscount = Number(channel?.price_discount_percent ?? 100) / 100;
  const markupRate = Number(channel?.markup_discount_rate || 0) / 100;
  const effective =
    channelBase * costDiscount +
    (Number.isFinite(globalBase) ? globalBase * markupRate : 0);
  return Number.isFinite(effective) ? effective : Number.POSITIVE_INFINITY;
};

const getStabilityLevel = (row) => {
  if (!row) return 0;
  if (Number(row.display_stability_grade) > 0) {
    return Math.max(1, Math.min(5, Number(row.display_stability_grade)));
  }
  const latency = Number(row.display_response_time_ms || 0);
  if (!row.last_test_success) return 1;
  if (latency > 0 && latency <= 1000) return 5;
  if (latency > 0 && latency <= 3000) return 4;
  if (latency > 0 && latency <= 5000) return 3;
  return latency > 0 ? 2 : 0;
};

const ChannelStabilitySignal = ({ row, t }) => {
  const level = getStabilityLevel(row);
  return (
    <Tooltip content={level > 0 ? `${t('稳定性')} ${level * 20}%` : t('未测')}>
      <span className='flex items-center gap-1'>
        {Array.from({ length: 5 }, (_, index) => (
          <span
            key={index}
            className='block h-2 w-4 rounded-sm'
            style={{
              backgroundColor:
                index < level
                  ? 'var(--semi-color-success)'
                  : 'var(--semi-color-fill-2)',
            }}
          />
        ))}
      </span>
    </Tooltip>
  );
};

const ChannelHotPill = ({ t }) => (
  <span className='channel-hot-pill' title={t('热门')}>
    <span className='channel-hot-pill-text'>{t('热门')}</span>
  </span>
);

const ChannelDiscountPill = ({ discountLabel }) => (
  <Tag size='small' shape='circle' className='channel-discount-tag'>
    {discountLabel}
  </Tag>
);

const ChannelStatusBadge = ({ isHotChannel, discountLabel, t }) => {
  if (isHotChannel && discountLabel) {
    return (
      <span
        className='channel-badge-carousel'
        title={`${t('热门')} · ${discountLabel}`}
        aria-label={`${t('热门')} · ${discountLabel}`}
      >
        <span className='channel-badge-carousel-track' aria-hidden='true'>
          <span className='channel-badge-carousel-item'>
            <ChannelHotPill t={t} />
          </span>
          <span className='channel-badge-carousel-item'>
            <ChannelDiscountPill discountLabel={discountLabel} />
          </span>
          <span className='channel-badge-carousel-item'>
            <ChannelHotPill t={t} />
          </span>
        </span>
      </span>
    );
  }

  if (isHotChannel) return <ChannelHotPill t={t} />;
  if (discountLabel) {
    return <ChannelDiscountPill discountLabel={discountLabel} />;
  }
  return null;
};

const getChannelLatencyMeta = (row, t) => {
  const latency = Number(row?.display_response_time_ms || 0);
  if (!(latency > 0)) {
    return {
      text: t('未测'),
      color: 'var(--semi-color-text-2)',
      backgroundColor: 'var(--semi-color-fill-1)',
    };
  }

  const text =
    latency < 1000
      ? `${Math.round(latency)}ms`
      : `${(latency / 1000).toFixed(latency >= 10000 ? 1 : 2)}s`;
  if (row?.last_test_success === false || latency > 5000) {
    return {
      text,
      color: 'var(--semi-color-danger)',
      backgroundColor: 'var(--semi-color-danger-light-default)',
    };
  }
  if (latency <= 1000) {
    return {
      text,
      color: 'var(--semi-color-success)',
      backgroundColor: 'var(--semi-color-success-light-default)',
    };
  }
  if (latency <= 3000) {
    return {
      text,
      color: '#65a30d',
      backgroundColor: 'rgba(132, 204, 22, 0.14)',
    };
  }
  return {
    text,
    color: '#ca8a04',
    backgroundColor: 'rgba(234, 179, 8, 0.16)',
  };
};

const formatDiscountLabel = (channel, modelData, t) => {
  const officialBase = Number(
    channel?.quota_type === 1 || modelData?.quota_type === 1
      ? modelData?.model_price
      : modelData?.model_ratio,
  );
  const channelPrice = getChannelPriceScore(channel, modelData);
  if (
    !Number.isFinite(officialBase) ||
    officialBase <= 0 ||
    !Number.isFinite(channelPrice) ||
    channelPrice >= officialBase
  ) {
    return '';
  }
  const discount = Math.round((1 - channelPrice / officialBase) * 100);
  return formatPriceRatioFromDiscount(discount, t);
};

/** 阶梯计费：取第一档输入/输出平台价（USD /1M，已含渠道折扣与分组倍率） */
const getFirstTierTokenPricesUsd = (channel, modelData, usedGroupRatio) => {
  const globalRule = getRequestTierPricing(modelData);
  const channelRule = getRequestTierPricing(channel);
  const bandRule = channelRule || globalRule;
  if (!bandRule?.tiers?.length) return null;

  const currencyRates = getCurrencyRatesFromStatus();
  const globalCurrency = normalizeCurrency(globalRule?.currency);
  const channelCurrency = normalizeCurrency(
    channelRule?.currency || globalRule?.currency,
  );
  const costDisc = costDiscountMultiplier(
    channel?.price_discount_percent != null
      ? channel.price_discount_percent
      : 100,
  );
  const markupRate = markupRateFromPercent(channel?.markup_discount_rate || 0);
  const globalTiers = globalRule?.tiers || [];
  const channelTiers = channelRule?.tiers || [];

  const resolveUsd = (priceKey) => {
    const globalRaw = findTierPriceAtBand(globalTiers, 0, priceKey, 'lt') ?? 0;
    const channelRaw =
      channelTiers.length > 0
        ? findTierPriceAtBand(channelTiers, 0, priceKey, 'lt')
        : null;
    const globalPrice = convertTierPriceToUSD(
      globalRaw,
      globalCurrency,
      currencyRates,
    );
    const channelPrice =
      channelRaw != null
        ? convertTierPriceToUSD(channelRaw, channelCurrency, currencyRates)
        : null;
    const effective =
      channelTiers.length > 0 && channelPrice != null
        ? channelPrice
        : globalPrice;
    const usd =
      (effective * costDisc + globalPrice * markupRate) * usedGroupRatio;
    return Number.isFinite(usd) && usd > 0 ? usd : null;
  };

  return {
    input: resolveUsd('input'),
    output: resolveUsd('output'),
  };
};

const getChannelPriceRows = ({
  channel,
  modelData,
  selectedGroup,
  groupRatio,
  displayPrice,
  tokenUnit,
  t,
}) => {
  const { usedGroupRatio } = getUsedGroupContext(
    modelData,
    selectedGroup ?? 'all',
    groupRatio || {},
  );
  const formatUsd = (usd, perToken = false) => {
    const normalized = perToken && tokenUnit === 'K' ? usd / 1000 : usd;
    return displayPrice(normalized);
  };
  const videoHint = channel?.video_flat_clip_hint;
  const videoPrice = Number(videoHint?.min_usd_after_channel_discount);
  if (Number.isFinite(videoPrice) && videoPrice > 0) {
    const resolution =
      formatVideoResolutionDisplayLabel(videoHint?.resolution) ||
      videoHint?.resolution ||
      t('视频');
    const unit =
      videoHint?.billing_mode === 'per_second' ? t('/秒起') : t('/条起');
    return [
      {
        key: 'video',
        label: resolution,
        value: `${formatUsd(videoPrice * usedGroupRatio)}${unit}`,
      },
    ];
  }

  const imageHint = channel?.image_per_image_hint;
  const imagePrice = Number(imageHint?.min_usd_after_channel_discount);
  if (Number.isFinite(imagePrice) && imagePrice > 0) {
    return [
      {
        key: 'image',
        label: t('图片'),
        value: `${formatUsd(imagePrice * usedGroupRatio)}${t('/张起')}`,
      },
    ];
  }

  const isTiered =
    channel?.quota_type === 3 ||
    modelData?.quota_type === 3 ||
    !!getRequestTierPricing(channel) ||
    !!getRequestTierPricing(modelData);
  if (isTiered) {
    const tierPrices = getFirstTierTokenPricesUsd(
      channel,
      modelData,
      usedGroupRatio,
    );
    if (tierPrices) {
      const unit = tokenUnit === 'K' ? '/K' : '/M';
      const rows = [];
      if (tierPrices.input != null) {
        rows.push({
          key: 'input',
          label: t('输入'),
          value: `${formatUsd(tierPrices.input, true)}${unit}`,
        });
      }
      if (tierPrices.output != null) {
        rows.push({
          key: 'output',
          label: t('输出'),
          value: `${formatUsd(tierPrices.output, true)}${unit}`,
        });
      }
      if (rows.length > 0) return rows;
    }
  }

  const isFixed = channel?.quota_type === 1 || modelData?.quota_type === 1;
  const billingRates = computeChannelBillingRates({
    channelModelRatio: channel?.model_ratio,
    channelCompletionRatio: channel?.completion_ratio,
    channelCacheRatio: channel?.cache_ratio,
    channelCreateCacheRatio: channel?.create_cache_ratio,
    channelModelPrice: channel?.model_price,
    priceDiscountPercent: channel?.price_discount_percent ?? 100,
    markupDiscountPercent: channel?.markup_discount_rate || 0,
    globalModelRatio: modelData?.model_ratio,
    globalModelPrice: modelData?.model_price,
    globalCompletionRatio: modelData?.completion_ratio,
    globalCacheRatio: modelData?.cache_ratio,
    globalCreateCacheRatio: modelData?.create_cache_ratio,
  });
  if (isFixed) {
    if (!(billingRates.effModelPrice >= 0)) return [];
    return [
      {
        key: 'fixed',
        label: t('价格'),
        value: `${formatUsd(billingRates.effModelPrice * usedGroupRatio)}${t('/次起')}`,
      },
    ];
  }

  const unit = tokenUnit === 'K' ? '/K' : '/M';
  const rows = [];
  if (Number.isFinite(billingRates.inputRatioPrice)) {
    rows.push({
      key: 'input',
      label: t('输入'),
      value: `${formatUsd(
        billingRates.inputRatioPrice * usedGroupRatio,
        true,
      )}${unit}`,
    });
  }
  if (
    modelData?.completion_ratio != null &&
    channel?.completion_ratio != null &&
    Number.isFinite(billingRates.completionRatioPrice)
  ) {
    rows.push({
      key: 'output',
      label: t('输出'),
      value: `${formatUsd(
        billingRates.completionRatioPrice * usedGroupRatio,
        true,
      )}${unit}`,
    });
  }
  return rows;
};

const ModelChannelWorkspace = ({
  modelData,
  channelMtrMap = {},
  endpointMap = {},
  perfSummary = null,
  hotChannelScoreMap = new Map(),
  t,
  ...props
}) => {
  const channelList = Array.isArray(modelData?.channel_list)
    ? modelData.channel_list
    : [];
  const lowestChannelKey = useMemo(() => {
    if (channelList.length === 0) return '';
    let bestIndex = 0;
    let bestScore = getChannelPriceScore(channelList[0], modelData);
    channelList.forEach((channel, index) => {
      const score = getChannelPriceScore(channel, modelData);
      if (score < bestScore) {
        bestIndex = index;
        bestScore = score;
      }
    });
    return getChannelKey(channelList[bestIndex], bestIndex);
  }, [channelList, modelData]);
  const [selectedChannelKey, setSelectedChannelKey] =
    useState(lowestChannelKey);
  const [channelPerfMap, setChannelPerfMap] = useState({});
  const [loadedChannelPerf, setLoadedChannelPerf] = useState({});
  const [visiblePerfSummary, setVisiblePerfSummary] = useState(perfSummary);

  useEffect(() => {
    setSelectedChannelKey(lowestChannelKey);
    setChannelPerfMap({});
    setLoadedChannelPerf({});
    setVisiblePerfSummary(perfSummary);
  }, [lowestChannelKey, modelData?.model_name, perfSummary]);

  const selectedEntry = useMemo(() => {
    const index = channelList.findIndex(
      (channel, channelIndex) =>
        getChannelKey(channel, channelIndex) === selectedChannelKey,
    );
    const selectedIndex = index >= 0 ? index : 0;
    return {
      channel: channelList[selectedIndex],
      index: selectedIndex,
    };
  }, [channelList, selectedChannelKey]);

  const selectedChannelId = Number(selectedEntry.channel?.channel_id || 0);

  useEffect(() => {
    const channelKey = String(selectedChannelId);
    if (!selectedChannelId || !loadedChannelPerf[channelKey]) return;
    setVisiblePerfSummary(channelPerfMap[channelKey] || null);
  }, [channelPerfMap, loadedChannelPerf, selectedChannelId]);

  useEffect(() => {
    const channelIds = channelList
      .map((channel) => Number(channel?.channel_id || 0))
      .filter(Boolean);
    const missingChannelIds = channelIds.filter(
      (channelId) => !loadedChannelPerf[String(channelId)],
    );
    if (missingChannelIds.length === 0) return;

    let cancelled = false;
    Promise.allSettled(
      missingChannelIds.map(async (channelId) => ({
        channelId,
        summary: perfQueryResultToSummary(
          await fetchPerfMetrics(modelData.model_name, 24, '', channelId),
        ),
      })),
    ).then((results) => {
      if (cancelled) return;
      const summaries = {};
      const loaded = {};
      results.forEach((result, index) => {
        const fallbackChannelId = missingChannelIds[index];
        if (result.status === 'fulfilled') {
          const { channelId, summary } = result.value;
          if (summary) summaries[String(channelId)] = summary;
          loaded[String(channelId)] = true;
          return;
        }
        loaded[String(fallbackChannelId)] = true;
      });
      if (Object.keys(summaries).length > 0) {
        setChannelPerfMap((current) => ({ ...current, ...summaries }));
      }
      setLoadedChannelPerf((current) => ({ ...current, ...loaded }));
    });

    return () => {
      cancelled = true;
    };
  }, [channelList, loadedChannelPerf, modelData.model_name]);

  if (!selectedEntry.channel) return null;

  const selectedModelData = {
    ...modelData,
    channel_list: [selectedEntry.channel],
  };
  const selectedMetrics = {
    [String(selectedEntry.channel.channel_id)]:
      channelMtrMap[String(selectedEntry.channel.channel_id)],
  };

  return (
    <div className='model-channel-workspace grid h-full min-h-0 grid-cols-1 grid-rows-[auto_minmax(0,1fr)] overflow-hidden md:grid-cols-[230px_minmax(0,1fr)] md:grid-rows-1'>
      <aside className='channel-selector-pane max-h-64 min-h-0 overflow-y-auto overscroll-contain border-b p-3 md:h-full md:max-h-none md:border-b-0 md:border-r'>
        <div className='space-y-2'>
          {channelList.map((channel, index) => {
            const key = getChannelKey(channel, index);
            const active = key === selectedChannelKey;
            const row = channelMtrMap[String(channel.channel_id)];
            const latencyMeta = getChannelLatencyMeta(row, t);
            const routeLabel =
              channel.route_slug || channel.channel_no || modelData.model_name;
            const discountLabel = formatDiscountLabel(channel, modelData, t);
            const isHotChannel = hotChannelScoreMap.has(
              getChannelHeatKey(modelData, channel),
            );
            const priceRows = getChannelPriceRows({
              channel,
              modelData,
              selectedGroup: props.selectedGroup,
              groupRatio: props.groupRatio,
              displayPrice: props.displayPrice,
              tokenUnit: props.tokenUnit,
              t,
            });
            return (
              <button
                key={key}
                type='button'
                className='channel-selector-card w-full rounded-lg border p-3 text-left'
                aria-pressed={active}
                data-active={active ? 'true' : 'false'}
                onClick={() => setSelectedChannelKey(key)}
              >
                <span
                  className='channel-selector-active-outline'
                  aria-hidden='true'
                />
                <div className='flex min-w-0 items-center justify-between gap-2'>
                  <span className='channel-route-chip' title={routeLabel}>
                    {channel.supplier_type ? (
                      <span className='channel-route-chip-supplier'>
                        {getSupplierTypeLabel(channel.supplier_type, t)}
                      </span>
                    ) : null}
                    {channel.supplier_type && routeLabel ? (
                      <span className='channel-route-chip-dot'>·</span>
                    ) : null}
                    <span className='channel-route-chip-suffix'>
                      {routeLabel}
                    </span>
                  </span>
                  <div className='flex shrink-0 items-center'>
                    <ChannelStatusBadge
                      isHotChannel={isHotChannel}
                      discountLabel={discountLabel}
                      t={t}
                    />
                  </div>
                </div>
                <div className='channel-selector-divider my-3 border-t border-dashed' />
                <div className='space-y-1.5'>
                  {priceRows.length > 0 ? (
                    priceRows.map((price) => (
                      <div
                        key={price.key}
                        className='flex items-baseline justify-between gap-2'
                      >
                        <Text type='secondary' size='small'>
                          {price.label}
                        </Text>
                        <Text className='channel-price-value'>
                          {price.value}
                        </Text>
                      </div>
                    ))
                  ) : (
                    <Text type='tertiary' size='small'>
                      {t('价格待配置')}
                    </Text>
                  )}
                </div>
                <div className='mt-3 flex items-center justify-between gap-2'>
                  <ChannelStabilitySignal row={row} t={t} />
                  <span
                    className='rounded-full px-2 py-0.5 font-mono text-xs font-semibold'
                    style={{
                      color: latencyMeta.color,
                      backgroundColor: latencyMeta.backgroundColor,
                    }}
                  >
                    {latencyMeta.text}
                  </span>
                </div>
              </button>
            );
          })}
        </div>
      </aside>
      <section className='channel-detail-content h-full min-h-0 overflow-y-auto overscroll-contain p-3'>
        <ModelPerfPanel
          modelName={modelData.model_name}
          perfSummary={visiblePerfSummary}
          t={t}
          flat
        />
        <ModelEndpoints
          modelData={selectedModelData}
          endpointMap={endpointMap}
          t={t}
          flat
        />
        <ModelChannelList
          {...props}
          modelData={selectedModelData}
          channelMtrMap={selectedMetrics}
          t={t}
          flatDetails
        />
      </section>
    </div>
  );
};

export default ModelChannelWorkspace;
