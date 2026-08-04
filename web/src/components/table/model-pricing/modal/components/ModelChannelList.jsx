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
  getSupplierTypeLabel,
} from '../../../../../helpers';
import {
  getUsedGroupContext,
  pickChannelScopedModelFloat,
  userCanViewHomeCostPrice,
} from '../../../../../helpers/utils';
import {
  resolveTierSegmentSources,
  buildTokenTierPreviewItems,
  formatTierBound,
} from '../../view/card/tierUtils';
import { UserContext } from '../../../../../context/User';
import ApiDocsSidePanel from './ApiDocsSidePanel';
import ModelTokenList from './ModelTokenList';
import VideoFlatClipHintTable from '../../components/VideoFlatClipHintTable';
import ImagePerImageHintTable from '../../components/ImagePerImageHintTable';
import PrecisePriceText, {
  formatCurrencyAmount,
  formatPreciseCurrencyValue,
  formatPreciseUsdPrice,
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
import { computeChannelCostRates } from '../../../../../helpers/billingFormula';
import { formatPriceRatioFromDiscount } from '../../utils/discount';
import { getChannelRouteModelName } from '../../utils/channelRoute';

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

const copyText = async (text, t, successText = '已复制', successOptions) => {
  if (await copy(text)) {
    Toast.success({ content: t(successText, successOptions) });
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

const copyModelName = (modelName, t) => {
  copyText(modelName, t, '模型{{modelName}}复制成功', { modelName });
};

const ModelNameCopyOption = ({ description, modelName, t }) => (
  <div>
    <div className='flex min-w-0 items-center gap-2 rounded-lg bg-semi-color-fill-0 px-3 py-2.5'>
      <Text
        className='min-w-0 flex-1 font-mono text-sm'
        ellipsis={{ showTooltip: true }}
      >
        {modelName}
      </Text>
      <Tooltip content={t('复制模型名字')}>
        <Button
          type='primary'
          theme='light'
          size='small'
          icon={<IconCopy />}
          onClick={() => copyModelName(modelName, t)}
          aria-label={t('复制模型名字')}
        >
          {t('复制')}
        </Button>
      </Tooltip>
    </div>
    <Text type='secondary' size='small' className='mt-1.5 block px-1 leading-5'>
      {description}
    </Text>
  </div>
);

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

/** 成本优惠按当前语言格式化；无优惠或超过官方价时显示占位符。 */
const formatCostDiscountDisplay = (priceDiscountPercent, t) => {
  const costPercent = Number(priceDiscountPercent);
  if (!Number.isFinite(costPercent)) {
    return null;
  }
  const savingsPercent = 100 - costPercent;
  if (savingsPercent <= 0) {
    return { text: `${t('成本折扣')}：-`, hasDiscount: false };
  }
  return {
    text: `${t('成本折扣')}：${formatPriceRatioFromDiscount(savingsPercent, t)}`,
    hasDiscount: true,
  };
};
// 阶梯计费表格：输入 TOKEN 区间 类型 输入 / M 输出 / M 缓存读取 / M 缓存写入 / M
const TokenTierDetailTable = ({
  model,
  channel,
  usedGroupRatio,
  displayPrice,
  t,
}) => {
  if (!channel) return null;
  const tierCategoryOrder = ['input', 'output', 'cache_read', 'cache_write'];
  const perCategoryRows = {};
  const activeCategories = [];
  let tierBoundary = 'lt';
  for (const cat of tierCategoryOrder) {
    const segmentSources = resolveTierSegmentSources({
      model,
      channel,
      cat,
    });
    if (segmentSources.boundary) {
      tierBoundary = segmentSources.boundary;
    }
    const { globalSegments, channelSegments, bandSegments } = segmentSources;
    if (bandSegments.length === 0) continue;
    const rows = buildTokenTierPreviewItems(
      bandSegments,
      globalSegments,
      channelSegments,
      channel,
      cat,
      usedGroupRatio,
      displayPrice,
      t,
      segmentSources,
    );
    if (rows.length > 0) {
      perCategoryRows[cat] = rows;
      activeCategories.push(cat);
    }
  }

  if (activeCategories.length === 0) return null;

  const catLabelMap = {
    input: t('输入'),
    output: t('输出'),
    cache_read: t('缓存读取'),
    cache_write: t('缓存写入'),
  };

  const displayCols = activeCategories.filter((c) =>
    ['input', 'output', 'cache_read', 'cache_write'].includes(c),
  );
  if (displayCols.length === 0) return null;

  const colHeaders = displayCols.map((c) => catLabelMap[c]);

  const baseCat = perCategoryRows.input
    ? 'input'
    : perCategoryRows.output
      ? 'output'
      : activeCategories[0];
  const baseRows = perCategoryRows[baseCat];

  const tierRanges = baseRows.map((baseRow, idx) => {
    const rowData = {};
    for (const cat of displayCols) {
      const catRows = perCategoryRows[cat];
      const cellRow =
        catRows.find(
          (r) =>
            Number(r.upTo) === Number(baseRow.upTo) &&
            Number(r.fromToken) === Number(baseRow.fromToken),
        ) || catRows[idx];
      if (cellRow) {
        rowData[cat] = {
          platformPrice: cellRow.platformPrice,
          platformPriceUsd: cellRow.platformPriceUsd,
          officialPrice: cellRow.officialPrice,
          officialPriceUsd: cellRow.officialPriceUsd,
          discount: cellRow.discount,
        };
      }
    }
    return {
      range: baseRow.range,
      fromToken: baseRow.fromToken,
      upTo: baseRow.upTo,
      cells: rowData,
    };
  });

  const cellStyle = {
    padding: '6px 10px',
    fontSize: 12,
    borderBottom: '1px solid var(--semi-color-border)',
    textAlign: 'center',
  };

  const headerStyle = {
    ...cellStyle,
    backgroundColor: 'var(--semi-color-fill-0)',
    color: 'var(--semi-color-text-2)',
    fontWeight: 600,
    fontSize: 11,
  };

  const typeStyleMap = {
    official: {
      label: t('官方'),
      color: 'var(--semi-color-text-2)',
      bgColor: 'var(--semi-color-fill-1)',
    },
    platform: {
      label: t('平台'),
      color: 'var(--semi-color-primary)',
      bgColor: 'transparent',
    },
    discount: {
      label: t('折扣'),
      color: 'var(--semi-color-danger)',
      bgColor: 'rgba(var(--semi-red-0), 0.15)',
    },
  };

  return (
    <div
      className='w-full min-w-0 overflow-hidden rounded-lg border'
      style={{
        borderColor: 'var(--semi-color-border)',
        marginTop: 8,
      }}
    >
      <table className='w-full border-collapse' style={{ fontSize: 12 }}>
        <thead>
          <tr>
            <th style={{ ...headerStyle, textAlign: 'left', width: '28%' }}>
              {t('输入 TOKEN 区间')}
            </th>
            <th style={{ ...headerStyle, width: '12%' }}>{t('类型')}</th>
            {displayCols.map((cat, i) => (
              <th
                key={cat}
                style={{ ...headerStyle, width: `${60 / displayCols.length}%` }}
              >
                {colHeaders[i]} / M
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {tierRanges.map((range, rangeIdx) => {
            const rowTypes = ['official', 'platform', 'discount'];
            return rowTypes.map((type, typeIdx) => {
              const ts = typeStyleMap[type];
              const showRange = typeIdx === 0;
              const rangeRowSpan = 3;
              return (
                <tr key={`${rangeIdx}-${type}`}>
                  {showRange && (
                    <td
                      rowSpan={rangeRowSpan}
                      style={{
                        ...cellStyle,
                        textAlign: 'left',
                        fontWeight: 600,
                        color: 'var(--semi-color-text-0)',
                        verticalAlign: 'middle',
                      }}
                    >
                      {range.fromToken === 0 && range.upTo > 0
                        ? tierBoundary === 'lte'
                          ? `≤ ${formatTierBound(range.upTo)}`
                          : `< ${formatTierBound(range.upTo)}`
                        : range.range}
                    </td>
                  )}
                  <td
                    style={{
                      ...cellStyle,
                      fontWeight: 600,
                      color: ts.color,
                      backgroundColor: ts.bgColor,
                    }}
                  >
                    {ts.label}
                  </td>
                  {displayCols.map((cat) => {
                    const cell = range.cells[cat];
                    if (!cell) {
                      return (
                        <td key={cat} style={cellStyle}>
                          —
                        </td>
                      );
                    }
                    if (type === 'official') {
                      const showStrike =
                        cell.officialPriceUsd > 0 &&
                        cell.officialPriceUsd > cell.platformPriceUsd;
                      return (
                        <td
                          key={cat}
                          style={{
                            ...cellStyle,
                            color: 'var(--semi-color-text-2)',
                            textDecoration: showStrike
                              ? 'line-through'
                              : 'none',
                          }}
                        >
                          {cell.officialPrice}
                        </td>
                      );
                    }
                    if (type === 'platform') {
                      return (
                        <td
                          key={cat}
                          style={{
                            ...cellStyle,
                            color: 'var(--semi-color-primary)',
                            fontWeight: 700,
                          }}
                        >
                          {cell.platformPrice}
                        </td>
                      );
                    }
                    return (
                      <td
                        key={cat}
                        style={{ ...cellStyle, backgroundColor: ts.bgColor }}
                      >
                        {cell.discount != null && cell.discount > 0 ? (
                          <Tag
                            size='small'
                            shape='circle'
                            style={{
                              fontSize: 12,
                              fontWeight: 700,
                              color: '#E74C3C',
                              backgroundColor: 'rgba(231, 76, 60, 0.11)',
                              border: 'none',
                            }}
                          >
                            {formatPriceRatioFromDiscount(cell.discount, t)}
                          </Tag>
                        ) : (
                          <span style={{ color: 'var(--semi-color-text-3)' }}>
                            -
                          </span>
                        )}
                      </td>
                    );
                  })}
                </tr>
              );
            });
          })}
        </tbody>
      </table>
    </div>
  );
};

const PriceComparisonList = ({
  items,
  t,
  blurPricing = false,
  isCostPrice = false,
}) => {
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
    <div className='channel-price-glass-table overflow-hidden rounded-lg'>
      <div
        className='channel-price-glass-header grid items-center gap-2 px-3 py-2 text-[11px] font-semibold'
        style={{
          gridTemplateColumns: '96px minmax(0, 1fr) minmax(0, 1fr) 72px',
          color: 'var(--semi-color-text-2)',
        }}
      >
        <span>{t('价格项')}</span>
        <span>
          {isCostPrice ? t('成本价') : t('平台价')}
          {priceHeaderUnit}
        </span>
        <span>
          {t('官方价')}
          {priceHeaderUnit}
        </span>
        <span className='text-right'>
          {isCostPrice ? t('成本折扣') : t('折扣')}
        </span>
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
        {items.map((item) => (
          <div
            key={item.key || item.label}
            className='channel-price-glass-row grid items-center gap-2 px-3 py-2 text-sm'
            style={{
              gridTemplateColumns: '96px minmax(0, 1fr) minmax(0, 1fr) 72px',
            }}
          >
            <div
              className='min-w-0 truncate font-semibold text-semi-color-text-0'
              title={item.label}
            >
              {item.label}
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
                      ? '#E74C3C'
                      : 'var(--semi-color-text-2)',
                    backgroundColor: item.hasDiscount
                      ? 'rgba(231, 76, 60, 0.11)'
                      : 'rgba(142, 142, 147, 0.12)',
                  }}
                >
                  {item.hasDiscount
                    ? formatPriceRatioFromDiscount(item.discount, t)
                    : '-'}
                </span>
              ) : (
                <span className='text-xs text-gray-400'>-</span>
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
  mode = 'details',
  compactDetails = false,
  flatDetails = false,
}) => {
  const [userState] = useContext(UserContext);
  const [docsVisible, setDocsVisible] = useState(false);
  const [docsMounted, setDocsMounted] = useState(false);
  const [docsModelName, setDocsModelName] = useState('');
  const [docsChannel, setDocsChannel] = useState(null);
  const [routeListExpanded, setRouteListExpanded] = useState(false);
  const channelList = modelData?.channel_list || [];
  const defaultModelName = modelData?.model_name || modelData?.modelName || '';
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

  const openApiDocs = (channelModelName, channel) => {
    setDocsModelName(channelModelName || modelData?.model_name || '');
    setDocsChannel(channel || null);
    setDocsMounted(true);
    setDocsVisible(true);
  };

  useEffect(() => {
    if (docsVisible || !docsMounted) return undefined;
    const timeoutId = window.setTimeout(() => setDocsMounted(false), 300);
    return () => window.clearTimeout(timeoutId);
  }, [docsMounted, docsVisible]);

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
      const videoBillingMode = String(vHint?.billing_mode || '');
      const videoFlatLabel =
        videoBillingMode === 'per_token'
          ? t('视频按 token 计费')
          : videoBillingMode === 'per_second'
            ? t('视频按秒计费')
            : t('视频按条（固定价）');
      items.push(
        makeItem(
          videoFlatLabel,
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
      skipVideoTokenPricing: true,
      skipVideoFlatSimple: showVideoFlatTable,
      quotaType,
    }).filter((item) => {
      if (!hideTextTokenPrices) return true;
      return item.key === 'video_flat' || item.key === 'model_price';
    });
    const getOfficialCostUsd = (key) => {
      const modelRatio = Number(modelData?.model_ratio || 0);
      const completionRatio = Number(modelData?.completion_ratio || 0);
      const cacheRatio = Number(modelData?.cache_ratio || 0);
      const createCacheRatio = Number(modelData?.create_cache_ratio || 0);
      const imageRatio = Number(modelData?.image_ratio || 0);
      const audioRatio = Number(modelData?.audio_ratio || 0);
      const audioCompletionRatio = Number(
        modelData?.audio_completion_ratio || 0,
      );
      const videoRatio = Number(modelData?.video_ratio || 0);
      const videoCompletionRatio = Number(
        modelData?.video_completion_ratio || 0,
      );
      const officialByKey = {
        model_price: Number(modelData?.model_price || 0),
        input: modelRatio * 2,
        output: modelRatio * completionRatio * 2,
        cache_read: modelRatio * cacheRatio * 2,
        cache_create: modelRatio * createCacheRatio * 2,
        image: modelRatio * imageRatio * 2,
        image_flat: Number(modelData?.image_price || 0),
        audio_input: modelRatio * audioRatio * 2,
        audio_output: modelRatio * audioRatio * audioCompletionRatio * 2,
        video_input: modelRatio * videoRatio * 2,
        video_output: modelRatio * videoRatio * videoCompletionRatio * 2,
        video_flat: Number(modelData?.video_price || 0),
      };
      const officialUsd = officialByKey[key];
      return Number.isFinite(officialUsd) && officialUsd > 0
        ? officialUsd
        : null;
    };
    const formatCostValue = (usd, isFixedPrice, fixedUnitKey) => {
      const value = formatPreciseUsdPrice(usd, { tokenUnit });
      const exact = formatPreciseCurrencyValue(
        toDisplayCurrencyValue(usd, { tokenUnit }),
      );
      return {
        value,
        exact,
        unitLabel: isFixedPrice
          ? fixedUnitKey === '张'
            ? t('张')
            : t('次')
          : tokenUnit === 'K'
            ? 'K'
            : 'M',
      };
    };
    return {
      items: costItems.map((item) => {
        const platform = formatCostValue(
          item.displayUsdPerM,
          item.isFixedPrice,
          item.fixedUnitKey,
        );
        const officialUsd = getOfficialCostUsd(item.key);
        const official = officialUsd
          ? formatCostValue(officialUsd, item.isFixedPrice, item.fixedUnitKey)
          : null;
        const costPercent = Number(channel.price_discount_percent ?? 100);
        const discount =
          Number.isFinite(costPercent) && costPercent >= 0 && costPercent < 100
            ? 100 - costPercent
            : null;
        return {
          key: item.key,
          label: t(item.labelKey),
          value: platform.value,
          valueTitle: platform.exact,
          valueExact: platform.exact,
          official: official?.value,
          officialTitle: official?.exact,
          officialExact: official?.exact,
          priceUnitLabel: platform.unitLabel,
          discount,
          hasDiscount: discount > 0,
        };
      }),
      videoHint: showVideoFlatTable ? vHint : null,
      imageHint: showImagePerImageTable ? iHint : null,
    };
  };

  if (channelList.length === 0) {
    return null;
  }

  if (flatDetails && channelList.length === 1) {
    const channel = channelList[0];
    const channelItems = formatChannelInfo(channel);
    const vHint = pickVideoFlatClipHintForChannel(modelData, channel);
    const showVideoFlatTable = hasVideoFlatClipTierTable(vHint);
    const iHint = pickImagePerImageHintForChannel(modelData, channel);
    const showImagePerImageTable = hasImagePerImageTierTable(iHint);
    const channelPath = getChannelRouteModelName(modelData, channel);
    const channelQuotaType =
      channel.quota_type !== undefined
        ? channel.quota_type
        : modelData?.quota_type;
    const isTierBilling = channelQuotaType === 3;
    const costInfo = formatChannelCostInfo(channel);
    const costItems = costInfo.items || [];
    const hasCostContent =
      costItems.length > 0 || costInfo.videoHint || costInfo.imageHint;

    return (
      <>
        <section className='mb-6 border-b border-semi-color-border pb-6'>
          <StepTitle
            label={t('第二步')}
            title={t('选择模型名')}
            desc={t('根据路由策略选择模型名。')}
            icon={<IconListView size={16} />}
          />
          <div className='space-y-2'>
            <ModelNameCopyOption
              description={t(
                '通用模型名：优先选择低价可用渠道；请求失败后自动切换至其他可用渠道。',
              )}
              modelName={defaultModelName}
              t={t}
            />
            <ModelNameCopyOption
              description={t(
                '指定渠道模型名：优先使用对应渠道；请求失败后自动切换至其他可用渠道。',
              )}
              modelName={channelPath}
              t={t}
            />
          </div>
        </section>

        <ModelTokenList
          visible={isLoggedIn}
          showLoginPrompt
          t={t}
          stepLabel={t('第三步')}
          title={t('复制API Key')}
          description={t('复制可用于调用上述 API 端点的 API Key')}
          flat
        />

        <section className='mb-6 border-b border-semi-color-border pb-6'>
          <div className='mb-4 flex min-w-0 items-start justify-between gap-3'>
            <div className='min-w-0'>
              <Text className='text-lg font-medium'>{t('价格信息')}</Text>
            </div>
            <div className='flex shrink-0 items-center gap-2'>
              <Tooltip content={t('复制模型名字')}>
                <Button
                  type='primary'
                  theme='light'
                  size='small'
                  icon={<IconCopy />}
                  onClick={() => copyModelName(channelPath, t)}
                  title={channelPath}
                  aria-label={t('复制模型名字')}
                >
                  {t('复制')}
                </Button>
              </Tooltip>
            </div>
          </div>
          {showVideoFlatTable ? (
            <VideoFlatClipHintTable
              hint={vHint}
              usedGroupRatio={usedGroupRatio}
              displayPrice={displayPrice}
              t={t}
              blurPricing={blurPricing}
            />
          ) : null}
          {isTierBilling ? (
            <TokenTierDetailTable
              model={modelData}
              channel={channel}
              usedGroupRatio={usedGroupRatio}
              displayPrice={displayPrice}
              t={t}
            />
          ) : (
            <PriceComparisonList
              items={channelItems}
              t={t}
              blurPricing={blurPricing}
            />
          )}
          {showImagePerImageTable ? (
            <ImagePerImageHintTable
              hint={iHint}
              usedGroupRatio={usedGroupRatio}
              displayPrice={displayPrice}
              t={t}
              blurPricing={blurPricing}
            />
          ) : null}
        </section>

        {showCostPricePanel && hasCostContent ? (
          <section className='mb-3'>
            <div className='mb-3 flex min-w-0 items-start justify-between gap-3'>
              <div className='min-w-0'>
                <Text className='text-lg font-medium'>{t('成本价')}</Text>
              </div>
              {channel.price_discount_percent != null
                ? (() => {
                    const discountDisplay = formatCostDiscountDisplay(
                      channel.price_discount_percent,
                      t,
                    );
                    if (!discountDisplay) return null;
                    return (
                      <span
                        className='inline-flex h-[22px] min-w-[42px] items-center justify-center rounded-full px-2 text-[11px] font-semibold'
                        style={{
                          color: discountDisplay.hasDiscount
                            ? '#E74C3C'
                            : 'var(--semi-color-text-2)',
                          backgroundColor: discountDisplay.hasDiscount
                            ? 'rgba(231, 76, 60, 0.11)'
                            : 'rgba(142, 142, 147, 0.12)',
                        }}
                      >
                        {discountDisplay.text}
                      </span>
                    );
                  })()
                : null}
            </div>
            <div className='flex flex-col gap-2 text-sm'>
              {costItems.length > 0 ? (
                <PriceComparisonList
                  items={costItems}
                  t={t}
                  blurPricing={blurPricing}
                  isCostPrice
                />
              ) : null}
              {costInfo.videoHint ? (
                <VideoFlatClipHintTable
                  hint={costInfo.videoHint}
                  usedGroupRatio={1}
                  displayPrice={formatPreciseUsdPrice}
                  t={t}
                  blurPricing={blurPricing}
                  isCostPrice
                  priceDiscountPercent={channel.price_discount_percent}
                  markupDiscountRate={channel.markup_discount_rate}
                />
              ) : null}
              {costInfo.imageHint ? (
                <ImagePerImageHintTable
                  hint={costInfo.imageHint}
                  usedGroupRatio={1}
                  displayPrice={formatPreciseUsdPrice}
                  t={t}
                  blurPricing={blurPricing}
                  isCostPrice
                  priceDiscountPercent={channel.price_discount_percent}
                  markupDiscountRate={channel.markup_discount_rate}
                />
              ) : null}
            </div>
          </section>
        ) : null}
      </>
    );
  }

  if (mode === 'general') {
    return (
      <>
        <Card className='!rounded-2xl shadow-sm border-0 mb-6'>
          <StepTitle
            label={t('第二步')}
            title={t('选择模型名')}
            desc={t('根据路由策略选择模型名。')}
            icon={<IconListView size={16} />}
          />
          <div className='space-y-3'>
            <ModelNameCopyOption
              description={t(
                '通用模型名：优先选择低价可用渠道；请求失败后自动切换至其他可用渠道。',
              )}
              modelName={defaultModelName}
              t={t}
            />
          </div>
          <div className='mt-2 space-y-2'>
            {displayedRouteChannels.map(
              ({ channel, idx, routeModelName, badge }) => {
                const row = channelMtrMap[String(channel.channel_id)];
                return (
                  <div
                    key={`route-${channel.channel_id}-${idx}`}
                    className='flex items-center gap-2 rounded-lg px-3 py-2 overflow-hidden'
                    style={{ backgroundColor: 'var(--semi-color-fill-0)' }}
                  >
                    <Tag size='small' shape='circle' color='blue' type='light'>
                      {badge}
                    </Tag>
                    <div className='min-w-0 flex-1 flex items-center gap-2'>
                      <StabilityBattery row={row} t={t} />
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
                onClick={() => setRouteListExpanded((value) => !value)}
              >
                {routeListExpanded ? t('收起') : t('展开全部')}（
                {hiddenRouteCount}）
              </Button>
            ) : null}
          </div>
          <Text type='secondary' size='small' className='mt-2 block leading-5'>
            {t(
              '指定渠道模型名：优先使用对应渠道；请求失败后自动切换至其他可用渠道。',
            )}
          </Text>
        </Card>
        <ModelTokenList
          visible={isLoggedIn}
          showLoginPrompt
          t={t}
          stepLabel={t('第三步')}
          title={t('复制API Key')}
          description={t('复制可用于调用上述 API 端点的 API Key')}
          flat={flatDetails}
        />
      </>
    );
  }
  return (
    <>
      {!compactDetails ? (
        <Card className='!rounded-2xl shadow-sm border-0 mb-6'>
          <StepTitle
            label={t('第二步')}
            title={t('选择模型名')}
            desc={t('根据路由策略选择模型名。')}
            icon={<IconListView size={16} />}
          />
          <div className='space-y-3'>
            <ModelNameCopyOption
              description={t(
                '通用模型名：优先选择低价可用渠道；请求失败后自动切换至其他可用渠道。',
              )}
              modelName={defaultModelName}
              t={t}
            />
          </div>
          <div className='mt-2 space-y-2'>
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
                          {getSupplierTypeLabel(channel.supplier_type, t)}
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
          <Text type='secondary' size='small' className='mt-2 block leading-5'>
            {t(
              '指定渠道模型名：优先使用对应渠道；请求失败后自动切换至其他可用渠道。',
            )}
          </Text>
        </Card>
      ) : null}

      {!compactDetails ? (
        <ModelTokenList
          visible={isLoggedIn}
          showLoginPrompt
          t={t}
          stepLabel={t('第三步')}
          title={t('复制API Key')}
          description={t('复制可用于调用上述 API 端点的 API Key')}
        />
      ) : null}

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
                            {getSupplierTypeLabel(group.supplierType, t)}
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

                      const channelQuotaType =
                        channel.quota_type !== undefined
                          ? channel.quota_type
                          : modelData?.quota_type;
                      const isTierBilling = channelQuotaType === 3;

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
                                      onClick={() =>
                                        openApiDocs(channelPath, channel)
                                      }
                                    >
                                      {t('API 文档')}
                                    </Button>
                                  </Tooltip>
                                </div>
                              </div>
                              {showVideoFlatTable ? (
                                <VideoFlatClipHintTable
                                  hint={vHint}
                                  usedGroupRatio={usedGroupRatio}
                                  displayPrice={displayPrice}
                                  t={t}
                                  blurPricing={blurPricing}
                                />
                              ) : null}
                              {isTierBilling ? (
                                <TokenTierDetailTable
                                  model={modelData}
                                  channel={channel}
                                  usedGroupRatio={usedGroupRatio}
                                  displayPrice={displayPrice}
                                  t={t}
                                />
                              ) : (
                                <PriceComparisonList
                                  items={channelItems}
                                  t={t}
                                  blurPricing={blurPricing}
                                />
                              )}
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
                                  ? '#E74C3C'
                                  : 'var(--semi-color-text-2)',
                                backgroundColor: discountDisplay.hasDiscount
                                  ? 'rgba(231, 76, 60, 0.11)'
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
                        displayPrice={formatPreciseUsdPrice}
                        t={t}
                        blurPricing={blurPricing}
                        isCostPrice
                        priceDiscountPercent={channel.price_discount_percent}
                        markupDiscountRate={channel.markup_discount_rate}
                      />
                    ) : null}
                    {costInfo.imageHint ? (
                      <ImagePerImageHintTable
                        hint={costInfo.imageHint}
                        usedGroupRatio={1}
                        displayPrice={formatPreciseUsdPrice}
                        t={t}
                        blurPricing={blurPricing}
                        isCostPrice
                        priceDiscountPercent={channel.price_discount_percent}
                        markupDiscountRate={channel.markup_discount_rate}
                      />
                    ) : null}
                  </div>
                </div>
              );
            })}
          </div>
        </Card>
      ) : null}
      {docsMounted ? (
        <ApiDocsSidePanel
          visible={docsVisible}
          onClose={() => {
            setDocsVisible(false);
            setDocsModelName('');
            setDocsChannel(null);
          }}
          modelName={docsModelName || modelData?.model_name}
          docIntroduction={docsChannel?.doc_introduction || ''}
          apiDocs={docsChannel?.api_docs || ''}
          apiDocsMarkdown={docsChannel?.api_docs_markdown || ''}
          apiDocsMarkdownEn={docsChannel?.api_docs_markdown_en || ''}
          useDefaultDocs={docsChannel?.doc_configured !== true}
          t={t}
        />
      ) : null}
    </>
  );
};

export default ModelChannelList;
