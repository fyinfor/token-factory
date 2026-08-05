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
import { Button, Tag, Tooltip, Typography } from '@douyinfe/semi-ui';
import { IconArticle } from '@douyinfe/semi-icons';

import {
  computeChannelBillingRates,
  costDiscountMultiplier,
  formatVideoResolutionDisplayLabel,
  getSupplierTypeLabel,
  getUsedGroupContext,
  isASRPricingModel,
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
import ApiDocsSidePanel from './ApiDocsSidePanel';
import { getChannelHeatKey } from '../../utils/modelHeat';
import {
  calculatePriceDiscountPercent,
  formatPriceRatioFromDiscount,
  getBestPriceDiscountPercent,
} from '../../utils/discount';
import { getChannelRouteModelName } from '../../utils/channelRoute';

const { Text } = Typography;

const getChannelKey = (channel, index) =>
  String(
    channel?.channel_id ??
      `${channel?.supplier_application_id || 'supplier'}-${index}`,
  );

const getChannelPriceScore = (channel, modelData) => {
  const costDiscount = Number(channel?.price_discount_percent ?? 100) / 100;
  const markupRate = Number(channel?.markup_discount_rate || 0) / 100;

  // ASR：按秒单价优先（渠道价空则全局价）
  if (isASRPricingModel(modelData)) {
    const globalAsr = Number(modelData?.asr_price);
    if (!Number.isFinite(globalAsr) || globalAsr <= 0) {
      return Number.POSITIVE_INFINITY;
    }
    const channelAsr = globalAsr; // 暂无渠道级 ASR 价，空则回退全局
    const effective = channelAsr * costDiscount + globalAsr * markupRate;
    return Number.isFinite(effective) ? effective : Number.POSITIVE_INFINITY;
  }

  const quotaType = channel?.quota_type ?? modelData?.quota_type;
  const channelBase = Number(
    quotaType === 1 ? channel?.model_price : channel?.model_ratio,
  );
  const globalBase = Number(
    quotaType === 1 ? modelData?.model_price : modelData?.model_ratio,
  );
  if (!Number.isFinite(channelBase)) return Number.POSITIVE_INFINITY;

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

const formatDiscountLabel = (discountPercent, t) =>
  discountPercent > 0 ? formatPriceRatioFromDiscount(discountPercent, t) : '';

const getTierBestDiscountPercent = (hint, usedGroupRatio = 1) =>
  getBestPriceDiscountPercent(
    (Array.isArray(hint?.tiers) ? hint.tiers : []).map((row) =>
      calculatePriceDiscountPercent(
        Number(row?.usd_after_channel_discount || 0) * usedGroupRatio,
        row?.usd_official,
      ),
    ),
  );

const buildChannelPriceSummary = (rows, extraDiscounts = []) => ({
  rows,
  bestDiscount: getBestPriceDiscountPercent([
    ...rows.map((row) => row.discount),
    ...extraDiscounts,
  ]),
});

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
    const current =
      (effective * costDisc + globalPrice * markupRate) * usedGroupRatio;
    return {
      current: Number.isFinite(current) && current > 0 ? current : null,
      official:
        Number.isFinite(globalPrice) && globalPrice > 0 ? globalPrice : null,
    };
  };

  return {
    input: resolveUsd('input'),
    output: resolveUsd('output'),
  };
};

const getChannelPriceSummary = ({
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

  // ASR 按秒计费优先：语音识别 ¥x/秒（不以输入/输出 token 价展示）
  if (isASRPricingModel(modelData)) {
    const globalAsrUsd = Number(modelData?.asr_price);
    if (Number.isFinite(globalAsrUsd) && globalAsrUsd > 0) {
      const costDisc = costDiscountMultiplier(
        channel?.price_discount_percent != null
          ? channel.price_discount_percent
          : 100,
      );
      const markupRate = markupRateFromPercent(
        channel?.markup_discount_rate || 0,
      );
      const channelAsrUsd = globalAsrUsd; // 暂无渠道级 ASR 价，空则回退全局
      const effectiveAsrUsd =
        channelAsrUsd * costDisc + globalAsrUsd * markupRate;
      const platformAsrUsd = effectiveAsrUsd * usedGroupRatio;
      return buildChannelPriceSummary([
        {
          key: 'asr',
          label: t('语音识别'),
          value: `${formatUsd(platformAsrUsd)}${t('/秒')}`,
          discount: calculatePriceDiscountPercent(
            effectiveAsrUsd,
            globalAsrUsd,
          ),
        },
      ]);
    }
  }

  const videoHint = channel?.video_flat_clip_hint;
  const videoPrice = Number(videoHint?.min_usd_after_channel_discount);
  if (Number.isFinite(videoPrice) && videoPrice > 0) {
    const resolution =
      formatVideoResolutionDisplayLabel(videoHint?.resolution) ||
      videoHint?.resolution ||
      t('视频');
    const unit =
      videoHint?.billing_mode === 'per_second' ? t('/秒起') : t('/条起');
    return buildChannelPriceSummary([
      {
        key: 'video',
        label: resolution,
        value: `${formatUsd(videoPrice * usedGroupRatio)}${unit}`,
        discount: getTierBestDiscountPercent(videoHint, usedGroupRatio),
      },
    ]);
  }

  const imageHint = channel?.image_per_image_hint;
  const imagePrice = Number(imageHint?.min_usd_after_channel_discount);
  if (Number.isFinite(imagePrice) && imagePrice > 0) {
    return buildChannelPriceSummary([
      {
        key: 'image',
        label: t('图片'),
        value: `${formatUsd(imagePrice * usedGroupRatio)}${t('/张起')}`,
        discount: getTierBestDiscountPercent(imageHint, usedGroupRatio),
      },
    ]);
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
      if (tierPrices.input?.current != null) {
        rows.push({
          key: 'input',
          label: t('输入'),
          value: `${formatUsd(tierPrices.input.current, true)}${unit}`,
          discount: calculatePriceDiscountPercent(
            tierPrices.input.current,
            tierPrices.input.official,
          ),
        });
      }
      if (tierPrices.output?.current != null) {
        rows.push({
          key: 'output',
          label: t('输出'),
          value: `${formatUsd(tierPrices.output.current, true)}${unit}`,
          discount: calculatePriceDiscountPercent(
            tierPrices.output.current,
            tierPrices.output.official,
          ),
        });
      }
      if (rows.length > 0) return buildChannelPriceSummary(rows);
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
    if (!(billingRates.effModelPrice >= 0)) {
      return buildChannelPriceSummary([]);
    }
    return buildChannelPriceSummary([
      {
        key: 'fixed',
        label: t('价格'),
        value: `${formatUsd(billingRates.effModelPrice * usedGroupRatio)}${t('/次起')}`,
        discount: calculatePriceDiscountPercent(
          billingRates.effModelPrice,
          modelData?.model_price,
        ),
      },
    ]);
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
      discount: calculatePriceDiscountPercent(
        billingRates.inputRatioPrice,
        Number(modelData?.model_ratio) * 2,
      ),
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
      discount: calculatePriceDiscountPercent(
        billingRates.completionRatioPrice,
        Number(modelData?.model_ratio) *
          Number(modelData?.completion_ratio) *
          2,
      ),
    });
  }
  const extraDiscounts = [];
  if (
    modelData?.cache_ratio != null &&
    channel?.cache_ratio != null &&
    Number.isFinite(billingRates.cacheRatioPrice)
  ) {
    extraDiscounts.push(
      calculatePriceDiscountPercent(
        billingRates.cacheRatioPrice,
        Number(modelData?.model_ratio) * Number(modelData?.cache_ratio) * 2,
      ),
    );
  }
  if (
    modelData?.create_cache_ratio != null &&
    channel?.create_cache_ratio != null &&
    Number.isFinite(billingRates.cacheCreationRatioPrice)
  ) {
    extraDiscounts.push(
      calculatePriceDiscountPercent(
        billingRates.cacheCreationRatioPrice,
        Number(modelData?.model_ratio) *
          Number(modelData?.create_cache_ratio) *
          2,
      ),
    );
  }
  return buildChannelPriceSummary(rows, extraDiscounts);
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
  const [docsVisible, setDocsVisible] = useState(false);
  const [docsMounted, setDocsMounted] = useState(false);

  useEffect(() => {
    setSelectedChannelKey(lowestChannelKey);
    setChannelPerfMap({});
    setLoadedChannelPerf({});
    setVisiblePerfSummary(perfSummary);
    setDocsVisible(false);
    setDocsMounted(false);
  }, [lowestChannelKey, modelData?.model_name, perfSummary]);

  useEffect(() => {
    if (docsVisible || !docsMounted) return undefined;
    const timeoutId = window.setTimeout(() => setDocsMounted(false), 300);
    return () => window.clearTimeout(timeoutId);
  }, [docsMounted, docsVisible]);

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
    const channelKey = String(selectedChannelId);
    if (!selectedChannelId || loadedChannelPerf[channelKey]) return;

    let cancelled = false;
    fetchPerfMetrics(modelData.model_name, 24, '', selectedChannelId)
      .then(perfQueryResultToSummary)
      .then((summary) => {
        if (cancelled) return;
        if (summary) {
          setChannelPerfMap((current) => ({
            ...current,
            [channelKey]: summary,
          }));
        }
        setLoadedChannelPerf((current) => ({
          ...current,
          [channelKey]: true,
        }));
      })
      .catch(() => {
        if (cancelled) return;
        setLoadedChannelPerf((current) => ({
          ...current,
          [channelKey]: true,
        }));
      });

    return () => {
      cancelled = true;
    };
  }, [loadedChannelPerf, modelData.model_name, selectedChannelId]);

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
            const isHotChannel = hotChannelScoreMap.has(
              getChannelHeatKey(modelData, channel),
            );
            const priceSummary = getChannelPriceSummary({
              channel,
              modelData,
              selectedGroup: props.selectedGroup,
              groupRatio: props.groupRatio,
              displayPrice: props.displayPrice,
              tokenUnit: props.tokenUnit,
              t,
            });
            const priceRows = priceSummary.rows;
            const discountLabel = formatDiscountLabel(
              priceSummary.bestDiscount,
              t,
            );
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
        <div className='model-api-docs-card mb-3 rounded-lg border p-3'>
          <div className='flex min-w-0 items-center justify-between gap-3'>
            <div className='flex min-w-0 items-center gap-2'>
              <span className='model-api-docs-card-icon' aria-hidden='true'>
                <IconArticle size={16} />
              </span>
              <Text strong className='text-sm'>
                {t('API 文档')}
              </Text>
            </div>
            <Tooltip content={t('查看 API 文档')}>
              <Button
                type='primary'
                theme='light'
                size='small'
                onClick={() => {
                  setDocsMounted(true);
                  setDocsVisible(true);
                }}
                aria-label={t('查看 API 文档')}
              >
                {t('查看')}
              </Button>
            </Tooltip>
          </div>
        </div>
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
        {docsMounted ? (
          <ApiDocsSidePanel
            visible={docsVisible}
            onClose={() => setDocsVisible(false)}
            modelName={getChannelRouteModelName(
              modelData,
              selectedEntry.channel,
            )}
            docIntroduction={selectedEntry.channel.doc_introduction || ''}
            apiDocs={selectedEntry.channel.api_docs || ''}
            apiDocsMarkdown={selectedEntry.channel.api_docs_markdown || ''}
            apiDocsMarkdownEn={selectedEntry.channel.api_docs_markdown_en || ''}
            useDefaultDocs={selectedEntry.channel.doc_configured !== true}
            t={t}
          />
        ) : null}
      </section>
    </div>
  );
};

export default ModelChannelWorkspace;
