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

import React, { useMemo, useState, useEffect, useRef, useContext } from 'react';
import {
  Card,
  Avatar,
  Typography,
  Collapse,
  Tag,
  Button,
  Toast,
  Tooltip,
} from '@douyinfe/semi-ui';
import { IconCopy, IconListView } from '@douyinfe/semi-icons';
import {
  copy,
  stringToColor,
  isVideoPricingModel,
} from '../../../../../helpers';
import {
  getUsedGroupContext,
  pickChannelScopedModelFloat,
  userCanViewHomeCostPrice,
} from '../../../../../helpers/utils';
import { UserContext } from '../../../../../context/User';
import ApiDocsSidePanel from './ApiDocsSidePanel';
import ModelTokenList from './ModelTokenList';
import VideoFlatClipHintTable from '../../components/VideoFlatClipHintTable';
import ImagePerImageHintTable from '../../components/ImagePerImageHintTable';
import PrecisePriceText, {
  formatCurrencyAmount,
  formatPreciseCurrencyValue,
  toDisplayCurrencyValue,
} from '../../components/PrecisePriceText';
import {
  pickVideoFlatClipHintForChannel,
  hasVideoFlatClipTierTable,
} from '../../constants/videoFlatClipLaneI18n';
import {
  pickImagePerImageHintForChannel,
  hasImagePerImageTierTable,
} from '../../constants/imagePerImageHintI18n';

import { renderModelTestResultSummary } from '../../../../../helpers/modelStability';
import {
  computeChannelCostRates,
  formatBillingUsdDisplay,
} from '../../../../../helpers/billingFormula';

const { Text } = Typography;

const ROUTE_PREVIEW_LIMIT = 6;

const StepTitle = ({ label, title, desc, icon }) => (
  <div className='flex items-start gap-3 mb-4'>
    <div
      className='flex items-center justify-center gap-1.5 shrink-0 rounded-full font-semibold text-xs px-3'
      style={{
        height: 30,
        width: 84,
        color: 'var(--semi-color-bg-0)',
        backgroundColor: 'var(--semi-color-primary)',
        boxShadow: '0 6px 14px rgba(var(--semi-blue-5), 0.24)',
      }}
    >
      {icon ? <span className='inline-flex items-center'>{icon}</span> : null}
      {label}
    </div>
    <div className='min-w-0'>
      <Text className='text-lg font-medium'>{title}</Text>
      <div className='text-xs text-gray-600 mt-0.5'>{desc}</div>
    </div>
  </div>
);

const copyText = async (text, t, successText = '已复制') => {
  if (await copy(text)) {
    Toast.success({ content: t(successText) });
  } else {
    Toast.error({ content: t('复制失败') });
  }
};

const hasRatioValue = (value) =>
  value !== undefined &&
  value !== null &&
  value !== '' &&
  Number.isFinite(Number(value));

const getSupplierTypeColor = (supplierType) => {
  switch (supplierType) {
    case '公有云':
      return 'green';
    case 'AIDC':
      return 'light-green';
    case '企业中转站':
      return 'lime';
    case '个人中转站':
      return 'yellow';
    default:
      return stringToColor(supplierType);
  }
};

const getChannelRouteModelName = (modelData, channel) => {
  const modelName = modelData?.model_name || '';
  if (channel?.route_slug) {
    return `${modelName}/${channel.route_slug}`;
  }
  return `${channel?.supplier_alias || ''}/${modelName}/${channel?.channel_no || ''}`;
};

const copyModelName = (modelName, t) => {
  copyText(modelName, t, `模型${modelName}复制成功`);
};

const getStabilityLevel = (row) => {
  if (!row) return 0;
  if (row.display_stability_grade > 0) {
    return Math.max(1, Math.min(5, Number(row.display_stability_grade)));
  }
  const latency = Number(row.display_response_time_ms || 0);
  if (!row.last_test_success) return 1;
  if (latency > 0 && latency <= 1000) return 5;
  if (latency > 0 && latency <= 3000) return 4;
  if (latency > 0 && latency <= 5000) return 3;
  return latency > 0 ? 2 : 3;
};

const StabilityBattery = ({ row, t }) => {
  const level = getStabilityLevel(row);
  return (
    <div className='flex items-center gap-2'>
      <Tooltip content={t('稳定性')}>
        <div
          className='flex items-end gap-0.5 rounded-md px-1.5 py-1 border'
          style={{
            borderColor: 'rgba(34, 197, 94, 0.35)',
            backgroundColor: 'rgba(34, 197, 94, 0.08)',
          }}
        >
          {[1, 2, 3, 4, 5].map((idx) => (
            <span
              key={idx}
              className='block rounded-sm transition-all duration-300'
              style={{
                width: 4,
                height: 5 + idx * 2,
                backgroundColor:
                  idx <= level ? 'rgb(34, 197, 94)' : 'rgba(34, 197, 94, 0.18)',
              }}
            />
          ))}
        </div>
      </Tooltip>
      <Text type='tertiary' size='small'>
        {level > 0 ? t('稳定') : t('未测')}
      </Text>
    </div>
  );
};

/** 成本折扣展示：100% - 成本折扣率% = 优惠幅度，如 25% 成本折扣显示 成本折扣：-75% */
const formatCostDiscountDisplay = (priceDiscountPercent, t) => {
  const costPercent = Number(priceDiscountPercent);
  if (!Number.isFinite(costPercent)) {
    return null;
  }
  const savingsPercent = Math.round(100 - costPercent);
  if (savingsPercent <= 0) {
    return { text: `${t('成本折扣')}：${t('0折扣')}`, hasDiscount: false };
  }
  return { text: `${t('成本折扣')}：-${savingsPercent}%`, hasDiscount: true };
};

const PriceComparisonList = ({ items, t, blurPricing = false }) => {
  if (!items || items.length === 0) {
    return null;
  }

  const rowUnitLabels = [
    ...new Set(items.map((item) => item.priceUnitLabel).filter(Boolean)),
  ];
  const sharedTokenUnitLabel =
    rowUnitLabels.length === 1 &&
    items.every((item) => item.priceUnitLabel === rowUnitLabels[0]);
  const priceHeaderUnit = sharedTokenUnitLabel ? ` /${rowUnitLabels[0]}` : '';
  const renderPriceValue = (value, item, exactKey = 'valueExact') => (
    <>
      <PrecisePriceText exact={item?.[exactKey]}>{value}</PrecisePriceText>
      {!sharedTokenUnitLabel && item.priceUnitLabel ? (
        <span className='ml-1 font-normal text-[10px] text-semi-color-text-2'>
          /{item.priceUnitLabel}
        </span>
      ) : null}
    </>
  );

  return (
    <div
      className='rounded-2xl overflow-hidden'
      style={{
        backgroundColor: 'var(--semi-color-bg-1)',
        border: '1px solid var(--semi-color-border)',
        boxShadow: '0 12px 26px rgba(15, 23, 42, 0.06)',
        backdropFilter: 'saturate(180%) blur(18px)',
      }}
    >
      <div
        className='grid items-center gap-2 mx-2 mt-2 px-2 py-1.5 text-[11px] font-semibold rounded-full'
        style={{
          gridTemplateColumns: '96px minmax(0, 1fr) minmax(0, 1fr) 52px',
          backgroundColor: 'var(--semi-color-fill-0)',
          color: 'var(--semi-color-text-2)',
        }}
      >
        <span>{t('价格项')}</span>
        <span>
          {t('平台价')}
          {priceHeaderUnit}
        </span>
        <span>
          {t('官方价')}
          {priceHeaderUnit}
        </span>
        <span className='text-right'>{t('折扣')}</span>
      </div>
      <div
        style={
          blurPricing
            ? {
                filter: 'blur(8px)',
                userSelect: 'none',
                pointerEvents: 'none',
              }
            : undefined
        }
      >
        {items.map((item, idx) => (
          <div
            key={item.key || item.label}
            className='grid items-center gap-2 mx-2 px-2 py-2.5 text-sm'
            style={{
              gridTemplateColumns: '96px minmax(0, 1fr) minmax(0, 1fr) 52px',
              borderTop:
                idx === 0 ? 'none' : '1px solid var(--semi-color-border)',
            }}
          >
            <div className='min-w-0 flex items-center gap-1.5'>
              <span
                className='inline-flex items-center justify-center rounded-full shrink-0'
                style={{
                  width: 20,
                  height: 20,
                  color: 'rgb(0, 122, 255)',
                  backgroundColor: 'rgba(0, 122, 255, 0.12)',
                }}
              >
                <span
                  className='rounded-full'
                  style={{
                    width: 7,
                    height: 7,
                    backgroundColor: 'rgb(0, 122, 255)',
                  }}
                />
              </span>
              <span
                className='font-semibold text-semi-color-text-0 truncate'
                title={item.label}
              >
                {item.label}
              </span>
            </div>
            <div
              className='min-w-0 font-bold truncate'
              style={{ color: 'var(--semi-color-primary)' }}
              title={item.valueTitle || item.value}
            >
              {renderPriceValue(item.value, item)}
            </div>
            <div
              className={`min-w-0 text-xs truncate ${
                item.hasDiscount ? 'line-through' : ''
              }`}
              style={{
                color: item.hasDiscount
                  ? 'var(--semi-color-text-2)'
                  : 'var(--semi-color-text-1)',
              }}
              title={
                item.officialTitle ||
                item.official ||
                item.original ||
                undefined
              }
            >
              {item.official || item.original
                ? renderPriceValue(
                    item.official || item.original,
                    item,
                    item.official ? 'officialExact' : 'originalExact',
                  )
                : '—'}
            </div>
            <div className='flex justify-end'>
              {item.discount != null ? (
                <span
                  className='inline-flex items-center justify-center text-[11px] font-semibold rounded-full'
                  style={{
                    minWidth: 42,
                    height: 22,
                    padding: '0 7px',
                    color: item.hasDiscount
                      ? '#dc2626'
                      : 'var(--semi-color-text-2)',
                    backgroundColor: item.hasDiscount
                      ? 'rgba(255, 59, 48, 0.11)'
                      : 'rgba(142, 142, 147, 0.12)',
                  }}
                >
                  {item.hasDiscount ? `-${item.discount}%` : '0%'}
                </span>
              ) : (
                <span className='text-xs text-gray-400'>—</span>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

const ModelChannelList = ({
  modelData,
  channelMtrMap = {},
  displayPrice,
  currency,
  siteDisplayType,
  tokenUnit,
  t,
  selectedGroup,
  groupRatio,
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
}) => {
  const [userState] = useContext(UserContext);
  const [docsVisible, setDocsVisible] = useState(false);
  const [docsModelName, setDocsModelName] = useState('');
  const [routeListExpanded, setRouteListExpanded] = useState(false);
  const channelList = modelData?.channel_list || [];
  const isLoggedIn = Boolean(userState?.user);
  const canViewCostPrice = useMemo(
    () => userCanViewHomeCostPrice(userState?.user),
    [userState?.user],
  );
  const showCostPricePanel = showCostPrice && canViewCostPrice;

  const { usedGroupRatio } = useMemo(
    () =>
      getUsedGroupContext(modelData, selectedGroup ?? 'all', groupRatio || {}),
    [modelData, selectedGroup, groupRatio],
  );

  const routeChannels = useMemo(
    () =>
      channelList.map((channel, idx) => ({
        channel,
        idx,
        routeModelName: getChannelRouteModelName(modelData, channel),
        badge: channel.route_slug || channel.channel_no || String(idx + 1),
      })),
    [channelList, modelData],
  );

  // 按 supplier_application_id 分组通道
  const groupedChannels = useMemo(() => {
    const groups = {};
    channelList.forEach((channel) => {
      const supplierId = channel.supplier_application_id;
      if (!groups[supplierId]) {
        groups[supplierId] = {
          supplierId,
          supplierAlias:
            (channel?.supplier_alias &&
              String(channel.supplier_alias).trim()) ||
            '',
          companyLogoUrl:
            (channel?.company_logo_url &&
              String(channel.company_logo_url).trim()) ||
            '',
          supplierType:
            (channel?.supplier_type && String(channel.supplier_type).trim()) ||
            '',
          channels: [],
        };
      }
      groups[supplierId].channels.push(channel);
    });
    return Object.values(groups);
  }, [channelList, t]);

  const groupKeys = useMemo(
    () => groupedChannels.map((group) => `group-${group.supplierId}`),
    [groupedChannels],
  );
  const defaultActiveKeys = useMemo(() => groupKeys.slice(0, 1), [groupKeys]);

  // 使用字符串形式来稳定比较
  const groupKeysStr = groupKeys.join(',');
  const prevKeysStr = useRef('');

  // 管理展开状态
  const [activeKey, setActiveKey] = useState(defaultActiveKeys);
  const activeKeys = Array.isArray(activeKey)
    ? activeKey
    : activeKey
      ? [activeKey]
      : [];
  const activeKeySet = useMemo(() => new Set(activeKeys), [activeKeys]);
  const displayedRouteChannels = routeListExpanded
    ? routeChannels
    : routeChannels.slice(0, ROUTE_PREVIEW_LIMIT);
  const hiddenRouteCount = Math.max(
    0,
    routeChannels.length - ROUTE_PREVIEW_LIMIT,
  );
  const hasHiddenRoutes = hiddenRouteCount > 0;

  const openApiDocs = (channelModelName) => {
    setDocsModelName(channelModelName || modelData?.model_name || '');
    setDocsVisible(true);
  };

  const handleCollapseChange = (nextKey) => {
    setActiveKey(Array.isArray(nextKey) ? nextKey : nextKey ? [nextKey] : []);
  };

  // 当分组实际变化时（基于字符串比较），重置为首组展开，避免一次性渲染全部渠道卡
  useEffect(() => {
    if (groupKeysStr !== prevKeysStr.current) {
      setActiveKey(defaultActiveKeys);
      setRouteListExpanded(false);
      prevKeysStr.current = groupKeysStr;
    }
  }, [groupKeysStr, defaultActiveKeys]);

  // 格式化通道信息（新计费公式：含分组倍率、成本折扣、加价折扣）
  const formatChannelInfo = (channel) => {
    // 判断计费类型：优先使用 channel.quota_type，否则使用 modelData.quota_type
    const quotaType =
      channel.quota_type !== undefined
        ? channel.quota_type
        : modelData?.quota_type;
    const isPerToken = quotaType === 0 || quotaType === 3; // 0=按量计费, 1=按次计费, 3=阶梯计费（按量）

    // ============================================================
    // 新计费公式参数：ch.model_ratio / ch.model_price 为原始渠道倍率（后端不再预乘成本折扣）
    //   成本折扣率 = price_discount_percent / 100
    //   加价倍率   = markup_discount_rate / 100
    //
    //   输入   = (ch.model_ratio × costDisc + globalMr × markupRate) × 2 × groupRatio
    //   输出   = (ch.model_ratio × cr × costDisc + globalMr × globalCR × markupRate) × 2 × groupRatio
    //   缓存读 = (ch.model_ratio × cacheRatio × costDisc + globalMr × globalCacheR × markupRate) × 2 × groupRatio
    //   缓存写 = (ch.model_ratio × createCacheRatio × costDisc + globalMr × globalCreateCacheR × markupRate) × 2 × groupRatio
    //   固定价 = (ch.model_price × costDisc + globalMp × markupRate) × groupRatio
    // ============================================================
    const costDisc =
      (channel.price_discount_percent != null
        ? channel.price_discount_percent
        : 100) / 100;
    const markupRate = (channel.markup_discount_rate || 0) / 100;
    const globalMr = modelData?.model_ratio || 0;
    const globalMp = modelData?.model_price || 0;
    const globalCR = modelData?.completion_ratio || 0;
    const globalCacheR =
      modelData?.cache_ratio != null ? Number(modelData.cache_ratio) : 0;
    const globalCreateCacheR =
      modelData?.create_cache_ratio != null
        ? Number(modelData.create_cache_ratio)
        : 0;
    const hideTextTokenPrices = isVideoPricingModel(modelData);

    // 计算价格，返回 { display, value }
    const calculatePrice = (
      nominalRatio,
      isFixedPrice = false,
      applyGroupRatio = true,
    ) => {
      let priceUSD;
      const ratio = applyGroupRatio ? usedGroupRatio : 1;
      if (isFixedPrice) {
        // 按次计费：直接使用价格
        priceUSD = nominalRatio * ratio;
      } else {
        // 按量计费：有效倍率 × 2 × 分组倍率
        priceUSD = nominalRatio * 2 * ratio;
      }
      const value = toDisplayCurrencyValue(priceUSD, { tokenUnit });
      const display = formatCurrencyAmount(value);
      const exactDisplay = formatPreciseCurrencyValue(value);
      if (isFixedPrice) {
        return {
          display: `${display} / ${t('次')}`,
          fullDisplay: `${exactDisplay} / ${t('次')}`,
          exactDisplay,
          unitLabel: null,
          value,
          rawUsd: priceUSD,
        };
      } else {
        const unitLabel = tokenUnit === 'K' ? 'K' : 'M';
        return {
          display,
          fullDisplay: `${exactDisplay} / 1${unitLabel} Tokens`,
          exactDisplay,
          unitLabel,
          value,
          rawUsd: priceUSD,
        };
      }
    };

    // 构造单条价格项，若全局价格高于有效通道价格则附带原价与折扣
    const makeItem = (label, channelValue, rootValue, isFixedPrice = false) => {
      if (!hasRatioValue(channelValue)) return null;
      const current = calculatePrice(Number(channelValue), isFixedPrice);
      let official = null;
      let discount = null;
      if (hasRatioValue(rootValue) && Number(rootValue) > 0) {
        const root = calculatePrice(Number(rootValue), isFixedPrice, false);
        official = root.display;
        const channelOriginal = calculatePrice(
          Number(channelValue),
          isFixedPrice,
          false,
        );
        if (root.rawUsd >= channelOriginal.rawUsd && root.rawUsd > 0) {
          discount = Math.round(
            (1 - channelOriginal.rawUsd / root.rawUsd) * 100,
          );
        }
      }
      return {
        label,
        value: current.display,
        valueTitle: current.exactDisplay,
        valueExact: current.exactDisplay,
        original: official,
        originalExact:
          official && hasRatioValue(rootValue)
            ? calculatePrice(Number(rootValue), isFixedPrice, false)
                .exactDisplay
            : undefined,
        officialTitle:
          official && hasRatioValue(rootValue)
            ? calculatePrice(Number(rootValue), isFixedPrice, false)
                .exactDisplay
            : undefined,
        officialExact:
          official && hasRatioValue(rootValue)
            ? calculatePrice(Number(rootValue), isFixedPrice, false)
                .exactDisplay
            : undefined,
        priceUnitLabel: current.unitLabel,
        discount,
        hasDiscount: discount > 0,
      };
    };

    const items = [];

    // 按次计费
    if (isPerToken === false) {
      // 固定价：ch.model_price × costDisc + globalMp × markupRate
      const effModelPrice = hasRatioValue(channel.model_price)
        ? Number(channel.model_price) * costDisc + globalMp * markupRate
        : null;
      items.push(
        makeItem(t('模型价格'), effModelPrice, modelData?.model_price, true),
      );
    }
    // 按量计费
    else {
      if (!hideTextTokenPrices) {
        // 输入：ch.model_ratio × costDisc + globalMr × markupRate
        const effInputRate = hasRatioValue(channel.model_ratio)
          ? Number(channel.model_ratio) * costDisc + globalMr * markupRate
          : null;
        items.push(
          makeItem(t('输入价格'), effInputRate, modelData?.model_ratio, false),
        );

        // 输出价格：仅当全局模型配置了 completion_ratio 时才展示
        if (
          hasRatioValue(channel.model_ratio) &&
          hasRatioValue(channel.completion_ratio) &&
          hasRatioValue(modelData?.completion_ratio)
        ) {
          const effOut =
            Number(channel.model_ratio) *
              Number(channel.completion_ratio) *
              costDisc +
            globalMr * globalCR * markupRate;
          const rootOut = hasRatioValue(modelData?.model_ratio)
            ? Number(modelData.model_ratio) * Number(modelData.completion_ratio)
            : null;
          items.push(makeItem(t('输出价格'), effOut, rootOut, false));
        }

        // 缓存读取价格：仅当全局模型配置了 cache_ratio 时才展示
        if (
          hasRatioValue(channel.model_ratio) &&
          hasRatioValue(channel.cache_ratio) &&
          hasRatioValue(modelData?.cache_ratio)
        ) {
          const effCacheRate =
            Number(channel.model_ratio) *
              Number(channel.cache_ratio) *
              costDisc +
            globalMr * globalCacheR * markupRate;
          const rootC = hasRatioValue(modelData?.model_ratio)
            ? Number(modelData.model_ratio) * Number(modelData.cache_ratio)
            : null;
          items.push(makeItem(t('缓存读取价格'), effCacheRate, rootC, false));
        }

        // 缓存创建价格：仅当全局模型配置了 create_cache_ratio 时才展示
        if (
          hasRatioValue(channel.model_ratio) &&
          hasRatioValue(channel.create_cache_ratio) &&
          hasRatioValue(modelData?.create_cache_ratio)
        ) {
          const effCreateCacheRate =
            Number(channel.model_ratio) *
              Number(channel.create_cache_ratio) *
              costDisc +
            globalMr * globalCreateCacheR * markupRate;
          const rootCC = hasRatioValue(modelData?.model_ratio)
            ? Number(modelData.model_ratio) *
              Number(modelData.create_cache_ratio)
            : null;
          items.push(
            makeItem(t('缓存创建价格'), effCreateCacheRate, rootCC, false),
          );
        }
      }

      const chVideoRatio = pickChannelScopedModelFloat(
        channelVideoRatioMap,
        channel.channel_id,
        modelData?.model_name,
      );
      const chVideoCompletionRatio = pickChannelScopedModelFloat(
        channelVideoCompletionRatioMap,
        channel.channel_id,
        modelData?.model_name,
      );
      const modelHasVideoRatio = hasRatioValue(modelData?.video_ratio);
      const modelHasVideoCompletion = hasRatioValue(
        modelData?.video_completion_ratio,
      );
      const showVideoToken =
        modelHasVideoRatio ||
        modelHasVideoCompletion ||
        chVideoRatio != null ||
        chVideoCompletionRatio != null;
      if (hasRatioValue(channel.model_ratio) && showVideoToken) {
        const effInputRate =
          Number(channel.model_ratio) * costDisc + globalMr * markupRate;
        const effVideoRatio =
          chVideoRatio != null
            ? chVideoRatio
            : modelHasVideoRatio
              ? Number(modelData.video_ratio)
              : 1;
        const effVideoCompletionRatio =
          chVideoCompletionRatio != null
            ? chVideoCompletionRatio
            : modelHasVideoCompletion
              ? Number(modelData.video_completion_ratio)
              : 1;
        const effVideo = effInputRate * effVideoRatio * effVideoCompletionRatio;
        const rootVideo = hasRatioValue(modelData?.model_ratio)
          ? Number(modelData.model_ratio) *
            (modelHasVideoRatio ? Number(modelData.video_ratio) : 1) *
            (modelHasVideoCompletion
              ? Number(modelData.video_completion_ratio)
              : 1)
          : null;
        items.push(makeItem(t('视频（倍率计价）'), effVideo, rootVideo, false));
      }
    }

    const chVideoPrice = pickChannelScopedModelFloat(
      channelVideoPriceMap,
      channel.channel_id,
      modelData?.model_name,
    );
    const rootVideoPrice = hasRatioValue(modelData?.video_price)
      ? Number(modelData.video_price)
      : null;
    const currentVideoPrice =
      chVideoPrice != null ? chVideoPrice : rootVideoPrice;
    const vHint = pickVideoFlatClipHintForChannel(modelData, channel);
    const showVideoFlatTable = hasVideoFlatClipTierTable(vHint);
    if (
      !showVideoFlatTable &&
      currentVideoPrice != null &&
      currentVideoPrice > 0
    ) {
      items.push(
        makeItem(
          t('视频按条（固定价）'),
          Number(currentVideoPrice),
          rootVideoPrice,
          true,
        ),
      );
    }
    return items.filter(Boolean);
  };

  /** 格式化单通道成本价（渠道/全局原价 × 成本折扣率，不含加价与分组倍率） */
  const formatChannelCostInfo = (channel) => {
    const quotaType =
      channel.quota_type !== undefined
        ? channel.quota_type
        : modelData?.quota_type;
    const vHint = pickVideoFlatClipHintForChannel(modelData, channel);
    const showVideoFlatTable = hasVideoFlatClipTierTable(vHint);
    const iHint = pickImagePerImageHintForChannel(modelData, channel);
    const showImagePerImageTable = hasImagePerImageTierTable(iHint);
    const modelHasVideoFlatPrice =
      modelData?.video_price != null &&
      modelData?.video_price !== undefined &&
      Number.isFinite(Number(modelData.video_price)) &&
      Number(modelData.video_price) > 0;
    const hideTextTokenPrices = isVideoPricingModel(modelData);

    const costItems = computeChannelCostRates({
      channelId: channel.channel_id,
      modelName: modelData?.model_name,
      optionModelRatio: channel.option_model_ratio,
      optionCompletionRatio: channel.option_completion_ratio,
      optionCacheRatio: channel.option_cache_ratio,
      optionCreateCacheRatio: channel.option_create_cache_ratio,
      optionModelPrice: channel.option_model_price,
      optionImageRatio: channel.option_image_ratio,
      optionImagePrice: channel.option_image_price,
      optionAudioRatio: channel.option_audio_ratio,
      optionAudioCompletionRatio: channel.option_audio_completion_ratio,
      optionVideoRatio: channel.option_video_ratio,
      optionVideoCompletionRatio: channel.option_video_completion_ratio,
      optionVideoPrice: channel.option_video_price,
      channelModelRatioMap,
      channelCompletionRatioMap,
      channelCacheRatioMap,
      channelCreateCacheRatioMap,
      channelModelPriceMap,
      channelImageRatioMap,
      channelImagePriceMap,
      channelAudioRatioMap,
      channelAudioCompletionRatioMap,
      channelVideoRatioMap,
      channelVideoCompletionRatioMap,
      channelVideoPriceMap,
      priceDiscountPercent:
        channel.price_discount_percent != null
          ? channel.price_discount_percent
          : 100,
      globalModelRatio: modelData?.model_ratio,
      globalModelPrice: modelData?.model_price,
      globalCompletionRatio: modelData?.completion_ratio,
      globalCacheRatio: modelData?.cache_ratio,
      globalCreateCacheRatio: modelData?.create_cache_ratio,
      globalImageRatio: modelData?.image_ratio,
      globalImagePrice: modelData?.image_price,
      globalAudioRatio: modelData?.audio_ratio,
      globalAudioCompletionRatio: modelData?.audio_completion_ratio,
      globalVideoRatio: modelData?.video_ratio,
      globalVideoCompletionRatio: modelData?.video_completion_ratio,
      globalVideoPrice: modelData?.video_price,
      skipImageTokenPricing: showImagePerImageTable,
      skipImageFlatSimple: showImagePerImageTable,
      skipVideoTokenPricing: showVideoFlatTable || modelHasVideoFlatPrice,
      skipVideoFlatSimple: showVideoFlatTable,
      quotaType,
    }).filter((item) => {
      if (!hideTextTokenPrices) return true;
      return (
        item.key === 'video_input' ||
        item.key === 'video_output' ||
        item.key === 'video_flat' ||
        item.key === 'model_price'
      );
    });
    const formatCostDisplay = (displayUsdPerM, isFixedPrice, fixedUnitKey) => {
      const priceUSD = displayUsdPerM;
      const value = formatBillingUsdDisplay(priceUSD, { tokenUnit });
      const preciseValue = formatPreciseCurrencyValue(
        toDisplayCurrencyValue(priceUSD, { tokenUnit }),
      );
      if (isFixedPrice) {
        const unit = fixedUnitKey === '张' ? t('张') : t('次');
        return {
          value: `${value} / ${unit}`,
          exact: preciseValue,
        };
      }
      const unitLabel = tokenUnit === 'K' ? 'K' : 'M';
      return {
        value: `${value} / 1${unitLabel} Tokens`,
        exact: preciseValue,
      };
    };
    return {
      items: costItems.map((item) => {
        const price = formatCostDisplay(
          item.displayUsdPerM,
          item.isFixedPrice,
          item.fixedUnitKey,
        );
        return {
          label: t(item.labelKey),
          value: price.value,
          exact: price.exact,
        };
      }),
      videoHint: showVideoFlatTable ? vHint : null,
      imageHint: showImagePerImageTable ? iHint : null,
    };
  };

  if (channelList.length === 0) {
    return null;
  }

  return (
    <>
      <Card className='!rounded-2xl shadow-sm border-0 mb-6'>
        <StepTitle
          label={t('第二步')}
          title={t('选择通道路由模型名')}
          desc={t('复制带渠道路由的模型名，可将请求固定到指定渠道')}
          icon={<IconListView size={16} />}
        />
        <div className='space-y-2'>
          {displayedRouteChannels.map(
            ({ channel, idx, routeModelName, badge }) => {
              const row = channelMtrMap[String(channel.channel_id)];
              return (
                <div
                  key={`route-${channel.channel_id}-${idx}`}
                  className='flex items-center gap-2 rounded-lg px-3 py-2 overflow-hidden'
                  style={{
                    backgroundColor: 'var(--semi-color-fill-0)',
                  }}
                >
                  <Tag
                    size='small'
                    shape='circle'
                    color='blue'
                    type='light'
                    className='shrink-0'
                  >
                    {badge}
                  </Tag>
                  <div className='min-w-0 flex-1 flex items-center gap-2'>
                    {channel.supplier_type ? (
                      <Tag
                        size='small'
                        shape='circle'
                        color={getSupplierTypeColor(channel.supplier_type)}
                        className='shrink-0'
                      >
                        {channel.supplier_type}
                      </Tag>
                    ) : null}
                    <div className='shrink-0'>
                      <StabilityBattery row={row} t={t} />
                    </div>
                    <Text
                      className='font-mono text-sm min-w-0'
                      ellipsis={{ showTooltip: true }}
                    >
                      {routeModelName}
                    </Text>
                  </div>
                  <Tooltip content={t('复制模型名字')}>
                    <Button
                      type='primary'
                      theme='light'
                      size='small'
                      icon={<IconCopy />}
                      onClick={() => copyModelName(routeModelName, t)}
                      aria-label={t('复制模型名字')}
                    >
                      {t('复制')}
                    </Button>
                  </Tooltip>
                </div>
              );
            },
          )}
          {hasHiddenRoutes ? (
            <Button
              theme='light'
              type='tertiary'
              size='small'
              className='w-full'
              onClick={() => setRouteListExpanded((v) => !v)}
            >
              {routeListExpanded ? t('收起') : t('展开全部')}（
              {hiddenRouteCount}）
            </Button>
          ) : null}
        </div>
      </Card>

      <ModelTokenList
        visible={isLoggedIn}
        t={t}
        stepLabel={t('第三步')}
        title={t('复制API Key')}
        description={t('复制可用于调用上述 API 端点的 API Key')}
      />

      <Card className='!rounded-2xl shadow-sm border-0 mb-3'>
        <div className='flex items-center justify-between gap-3 mb-4'>
          <div className='flex items-center min-w-0'>
            <Avatar size='small' color='indigo' className='mr-2 shadow-md'>
              <IconListView size={16} />
            </Avatar>
            <div className='min-w-0'>
              <Text className='text-lg font-medium'>{t('渠道信息与价格')}</Text>
              <div className='text-xs text-gray-600'>
                {t('按渠道展示稳定性、路由和当前价格')}
              </div>
            </div>
          </div>
          <Tag shape='circle' color='blue' type='light'>
            {channelList.length} {t('个通道')}
          </Tag>
        </div>

        <Collapse activeKey={activeKeys} onChange={handleCollapseChange}>
          {groupedChannels.map((group) => {
            const panelKey = `group-${group.supplierId}`;
            const panelActive = activeKeySet.has(panelKey);
            return (
              <Collapse.Panel
                key={panelKey}
                itemKey={panelKey}
                header={
                  <div className='flex items-center justify-between w-full'>
                    {group.companyLogoUrl || group.supplierType ? (
                      <span
                        className='h-7 rounded-md flex items-center overflow-hidden ml-2'
                        style={{ backgroundColor: 'var(--semi-color-fill-0)' }}
                      >
                        {group.companyLogoUrl ? (
                          <img
                            src={group.companyLogoUrl}
                            alt={group.supplierAlias || ''}
                            className='w-7 h-7 object-contain rounded-md'
                          />
                        ) : null}
                        {group.supplierType && (
                          <Tag
                            size='small'
                            shape='circle'
                            color={getSupplierTypeColor(group.supplierType)}
                            className='mx-1'
                          >
                            {group.supplierType}
                          </Tag>
                        )}
                      </span>
                    ) : null}
                    <span className='text-sm text-gray-500'>
                      {group.channels.length} {t('个通道')}
                    </span>
                  </div>
                }
              >
                {panelActive ? (
                  <div className='space-y-3'>
                    {group.channels.map((channel, idx) => {
                      const channelItems = formatChannelInfo(channel);
                      const row = channelMtrMap[String(channel.channel_id)];
                      const vHint = pickVideoFlatClipHintForChannel(
                        modelData,
                        channel,
                      );
                      const showVideoFlatTable =
                        hasVideoFlatClipTierTable(vHint);
                      const iHint = pickImagePerImageHintForChannel(
                        modelData,
                        channel,
                      );
                      const showImagePerImageTable =
                        hasImagePerImageTierTable(iHint);
                      const channelPath = getChannelRouteModelName(
                        modelData,
                        channel,
                      );
                      const channelBadge =
                        channel.route_slug || channel.channel_no || String(idx);

                      const handleCopy = () => {
                        copyModelName(channelPath, t);
                      };

                      return (
                        <div key={`${channel.channel_id}-${idx}`}>
                          <Card
                            className='!rounded-xl shadow-sm !mb-2 flex-1'
                            bodyStyle={{ padding: '12px' }}
                          >
                            <div className='flex flex-col gap-3 text-sm'>
                              <div className='flex items-start justify-between gap-2'>
                                <div className='min-w-0 flex-1'>
                                  <div className='flex items-center gap-2 min-w-0'>
                                    <Tag
                                      shape='circle'
                                      color='blue'
                                      type='light'
                                      size='small'
                                      className='shrink-0'
                                    >
                                      {channelBadge}
                                    </Tag>
                                    <Text
                                      strong
                                      ellipsis={{ showTooltip: true }}
                                    >
                                      {channelPath}
                                    </Text>
                                  </div>
                                  <div className='flex flex-wrap gap-2 items-center mt-1'>
                                    <StabilityBattery row={row} t={t} />
                                    {renderModelTestResultSummary(row, t)}
                                  </div>
                                </div>
                                <div className='flex flex-wrap gap-2 items-center shrink-0 ml-1'>
                                  <Tooltip content={t('复制模型名字')}>
                                    <Button
                                      type='primary'
                                      theme='light'
                                      size='small'
                                      icon={<IconCopy />}
                                      onClick={handleCopy}
                                      title={channelPath}
                                      aria-label={t('复制模型名字')}
                                    >
                                      {t('复制')}
                                    </Button>
                                  </Tooltip>
                                  <Tooltip content={t('查看 API 文档')}>
                                    <Button
                                      theme='light'
                                      type='warning'
                                      size='small'
                                      onClick={() => openApiDocs(channelPath)}
                                    >
                                      {t('文档')}
                                    </Button>
                                  </Tooltip>
                                </div>
                              </div>
                              <PriceComparisonList
                                items={channelItems}
                                t={t}
                                blurPricing={blurPricing}
                              />
                              {showVideoFlatTable ? (
                                <VideoFlatClipHintTable
                                  hint={vHint}
                                  usedGroupRatio={usedGroupRatio}
                                  displayPrice={displayPrice}
                                  t={t}
                                  blurPricing={blurPricing}
                                />
                              ) : null}
                              {showImagePerImageTable ? (
                                <ImagePerImageHintTable
                                  hint={iHint}
                                  usedGroupRatio={usedGroupRatio}
                                  displayPrice={displayPrice}
                                  t={t}
                                  blurPricing={blurPricing}
                                />
                              ) : null}
                            </div>
                          </Card>
                        </div>
                      );
                    })}
                  </div>
                ) : null}
              </Collapse.Panel>
            );
          })}
        </Collapse>
      </Card>
      {showCostPricePanel ? (
        <Card className='!rounded-2xl shadow-sm border-0 mb-3'>
          <div className='flex items-center mb-3'>
            <Avatar size='small' color='orange' className='mr-2 shadow-md'>
              <span className='font-semibold text-sm leading-none'>$</span>
            </Avatar>
            <Text className='text-lg font-medium'>{t('成本价')}</Text>
          </div>
          <div className='space-y-3'>
            {channelList.map((channel, idx) => {
              const costInfo = formatChannelCostInfo(channel);
              const costItems = costInfo.items || [];
              const hasCostContent =
                costItems.length > 0 ||
                costInfo.videoHint ||
                costInfo.imageHint;
              if (!hasCostContent) {
                return null;
              }
              return (
                <div
                  key={`cost-${channel.channel_id}-${idx}`}
                  className='rounded-lg border border-semi-color-border px-3 py-2'
                >
                  {channel.price_discount_percent != null
                    ? (() => {
                        const discountDisplay = formatCostDiscountDisplay(
                          channel.price_discount_percent,
                          t,
                        );
                        if (!discountDisplay) {
                          return null;
                        }
                        return (
                          <div className='mb-2'>
                            <span
                              className='inline-flex items-center justify-center text-[11px] font-semibold rounded-full'
                              style={{
                                minWidth: 42,
                                height: 22,
                                padding: '0 7px',
                                color: discountDisplay.hasDiscount
                                  ? '#dc2626'
                                  : 'var(--semi-color-text-2)',
                                backgroundColor: discountDisplay.hasDiscount
                                  ? 'rgba(255, 59, 48, 0.11)'
                                  : 'rgba(142, 142, 147, 0.12)',
                              }}
                            >
                              {discountDisplay.text}
                            </span>
                          </div>
                        );
                      })()
                    : null}
                  <div className='flex flex-col gap-1 text-sm'>
                    {costItems.map((item) => (
                      <div
                        key={item.label}
                        className='flex items-center gap-2 flex-wrap'
                      >
                        <span className='text-semi-color-text-1'>
                          {item.label}:
                        </span>
                        <span className='font-medium text-semi-color-text-0'>
                          <PrecisePriceText exact={item.exact}>
                            {item.value}
                          </PrecisePriceText>
                        </span>
                      </div>
                    ))}
                    {costInfo.videoHint ? (
                      <VideoFlatClipHintTable
                        hint={costInfo.videoHint}
                        usedGroupRatio={1}
                        displayPrice={formatBillingUsdDisplay}
                        t={t}
                        blurPricing={blurPricing}
                        isCostPrice
                      />
                    ) : null}
                    {costInfo.imageHint ? (
                      <ImagePerImageHintTable
                        hint={costInfo.imageHint}
                        usedGroupRatio={1}
                        displayPrice={formatBillingUsdDisplay}
                        t={t}
                        blurPricing={blurPricing}
                        isCostPrice
                      />
                    ) : null}
                  </div>
                </div>
              );
            })}
          </div>
        </Card>
      ) : null}
      <ApiDocsSidePanel
        visible={docsVisible}
        onClose={() => {
          setDocsVisible(false);
          setDocsModelName('');
        }}
        modelName={docsModelName || modelData?.model_name}
        docIntroduction={modelData?.doc_introduction}
        apiDocs={modelData?.api_docs}
        t={t}
      />
    </>
  );
};

export default ModelChannelList;
