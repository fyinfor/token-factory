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

import React from 'react';
import { useTranslation } from 'react-i18next';
import {
  Card,
  Tag,
  Tooltip,
  Checkbox,
  Empty,
  Pagination,
  Button,
  Avatar,
} from '@douyinfe/semi-ui';
import { IconHelpCircle } from '@douyinfe/semi-icons';
import {
  ArrowRight,
  ChevronRight,
  Copy,
  Database,
  HardDriveDownload,
  HardDriveUpload,
  Image as ImageIcon,
  Type,
  Video,
} from 'lucide-react';
import {
  IllustrationNoResult,
  IllustrationNoResultDark,
} from '@douyinfe/semi-illustrations';
import {
  stringToColor,
  calculateModelPrice,
  getModelPriceItems,
  getLobeHubIcon,
  getUsedGroupContext,
  pickChannelScopedModelFloat,
  computeChannelBillingRates,
  formatVideoResolutionDisplayLabel,
  formatImageResolutionDisplayLabel,
  compareVideoResolutionAsc,
  isVideoPricingModel,
  isASRPricingModel,
  hasNumericValue,
  getModelTagLabel,
  getSupplierTypeLabel,
  getModelCardDescription,
} from '../../../../../helpers';
import {
  TIER_CATEGORY_STYLES,
  detectTokenTierPricing,
  resolveTierSegmentSources,
  buildTokenTierPreviewItems,
  formatTierBound,
} from './tierUtils';
import {
  PRICING_TABLE_WRAPPER_STYLE,
  PRICING_TABLE_HEAD_BG,
  PRICING_TABLE_HEAD_CELL_STYLE,
  PRICING_TABLE_ROW_BORDER,
  PRICING_TABLE_BODY_BG,
  DISCOUNT_MUTED_STYLE,
  TABLE_CELL_CLASS,
  getFlatPricingColumns,
  getFixedPricingColumns,
  getVideoPricingColumns,
  getTierPricingColumns,
} from './pricingTableStyles';
import { VIDEO_FLAT_LANE_I18N_KEY } from '../../constants/videoFlatClipLaneI18n';
import {
  IMAGE_PER_IMAGE_LANE_I18N_KEY,
  laneToImagePerImageFamily,
} from '../../constants/imagePerImageHintI18n';
import { isTopHotModel } from '../../utils/modelHeat';
import { formatPriceRatioFromDiscount } from '../../utils/discount';
import {
  MODEL_CARD_PRICE_MAX_DECIMALS,
  truncateModelPriceValue,
} from '../../utils/priceDisplay';
import PricingCardSkeleton from './PricingCardSkeleton';
import ModelPerfCardSection from '../../components/ModelPerfCardSection';
import './homeModelCard.css';
import { useMinimumLoadingTime } from '../../../../../hooks/common/useMinimumLoadingTime';
import { renderLimitedItems } from '../../../../common/ui/RenderUtils';
import { useIsMobile } from '../../../../../hooks/common/useIsMobile';
import { getModelChannelRouteSuffixes } from '../../utils/channelRoute';
const CARD_STYLES = {
  container:
    'w-12 h-12 rounded-xl flex items-center justify-center relative shadow-sm border border-semi-color-border bg-white',
  icon: 'w-8 h-8 flex items-center justify-center',
  selected: 'border-blue-500 bg-blue-50 shadow-md',
  default: 'border-gray-200 hover:border-blue-200 hover:shadow-md',
};

const escapeRegExp = (value) => value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');

const HOME_MODEL_DESCRIPTION_LINE_HEIGHT = 18;

const AdaptiveModelDescription = ({ children, style }) => {
  const slotRef = React.useRef(null);
  const [lineCount, setLineCount] = React.useState(0);

  React.useLayoutEffect(() => {
    const slot = slotRef.current;
    if (!slot) return undefined;

    const updateLineCount = () => {
      const availableHeight = slot.getBoundingClientRect().height;
      const nextLineCount = Math.max(
        0,
        Math.floor(availableHeight / HOME_MODEL_DESCRIPTION_LINE_HEIGHT),
      );
      setLineCount((current) =>
        current === nextLineCount ? current : nextLineCount,
      );
    };

    updateLineCount();
    if (typeof ResizeObserver === 'undefined') {
      setLineCount(1);
      return undefined;
    }

    const observer = new ResizeObserver(updateLineCount);
    observer.observe(slot);
    return () => observer.disconnect();
  }, []);

  return (
    <div ref={slotRef} className='home-model-description-slot' style={style}>
      {lineCount > 0 ? (
        <p
          className='home-model-description m-0'
          style={{
            maxHeight: lineCount * HOME_MODEL_DESCRIPTION_LINE_HEIGHT,
            WebkitLineClamp: lineCount,
          }}
        >
          {children}
        </p>
      ) : null}
    </div>
  );
};

const getAudioLabel = (row, t) => {
  if (row?.has_audio === true) return t('有音轨');
  if (row?.has_audio === false) return t('无音轨');
  return t('统一');
};

const getVideoLaneMeta = (lane, t) => {
  const normalized = String(lane || '');
  if (normalized.includes('image_to_video')) {
    return {
      label: t('图生视频'),
      color: 'amber',
    };
  }
  if (
    normalized.includes('video_to_video') ||
    normalized.includes('video_to_video_input') ||
    normalized.includes('video_to_video_output')
  ) {
    return {
      label: t('视频生视频'),
      color: 'teal',
    };
  }
  return {
    label: t('文生视频'),
    color: 'blue',
  };
};

const getVideoLaneFamily = (lane) => {
  const normalized = String(lane || '');
  if (normalized.includes('image_to_video')) return 'image_to_video';
  if (
    normalized.includes('video_to_video') ||
    normalized.includes('video_to_video_input') ||
    normalized.includes('video_to_video_output')
  ) {
    return 'video_to_video';
  }
  return 'text_to_video';
};

const VIDEO_LANE_FAMILY_ORDER = {
  text_to_video: 0,
  image_to_video: 1,
  video_to_video: 2,
};

const formatCompactVideoResolution = (resolution, t) => {
  const label =
    formatVideoResolutionDisplayLabel(resolution) || resolution || t('默认');
  return String(label);
};

const formatVideoTierSpec = (row, t) =>
  formatCompactVideoResolution(row?.resolution, t);

const formatVideoTierDisplayPrice = (usd, usedGroupRatio, displayPrice) =>
  displayPrice(Number(usd || 0) * usedGroupRatio);

const getVideoTierDiscount = (currentDisplayUsd, officialUsd) => {
  const current = Number(currentDisplayUsd || 0);
  const official = Number(officialUsd || 0);
  if (
    !Number.isFinite(current) ||
    !Number.isFinite(official) ||
    official <= 0
  ) {
    return null;
  }
  return official > current ? Math.round((1 - current / official) * 100) : 0;
};

const VIDEO_CARD_TIER_PREVIEW_LIMIT = 3;
const IMAGE_CARD_TIER_PREVIEW_LIMIT = 3;

const buildVideoTierPreviewItems = (hint, usedGroupRatio, displayPrice, t) => {
  const rows = Array.isArray(hint?.tiers) ? hint.tiers : [];
  const billingMode = String(hint?.billing_mode || '');
  const maxItems = VIDEO_CARD_TIER_PREVIEW_LIMIT;
  const seen = new Set();
  const normalizedItems = [];
  for (const row of rows) {
    const resolution =
      formatVideoResolutionDisplayLabel(row?.resolution) ||
      row?.resolution ||
      t('默认');
    const laneKey = VIDEO_FLAT_LANE_I18N_KEY[row?.lane];
    const lane = laneKey ? t(laneKey) : row?.lane || t('视频');
    const laneMeta = getVideoLaneMeta(row?.lane, t);
    const family = getVideoLaneFamily(row?.lane);
    const currentUsd = Number(row?.usd_after_channel_discount || 0);
    if (!Number.isFinite(currentUsd) || currentUsd <= 0) continue;
    const officialUsd = Number(row?.usd_official || 0);
    const platformUsd = currentUsd * usedGroupRatio;
    const key = `${lane}|${resolution}|${row?.has_audio ?? 'all'}|${currentUsd}|${officialUsd}`;
    if (seen.has(key)) continue;
    seen.add(key);
    normalizedItems.push({
      key,
      family,
      resolution: row?.resolution,
      label: laneMeta.label,
      labelColor: laneMeta.color,
      spec: formatVideoTierSpec(row, t),
      platformPrice: formatVideoTierDisplayPrice(
        currentUsd,
        usedGroupRatio,
        displayPrice,
      ),
      officialPrice:
        Number.isFinite(officialUsd) && officialUsd > 0
          ? displayPrice(officialUsd)
          : '-',
      discount: getVideoTierDiscount(platformUsd, officialUsd),
      audioLabel: getAudioLabel(row, t),
      title: lane,
    });
  }

  normalizedItems.sort((a, b) => {
    const familyOrder =
      (VIDEO_LANE_FAMILY_ORDER[a.family] ?? 99) -
      (VIDEO_LANE_FAMILY_ORDER[b.family] ?? 99);
    if (familyOrder !== 0) return familyOrder;
    return compareVideoResolutionAsc(a.resolution, b.resolution);
  });

  const items = normalizedItems.slice(0, maxItems);
  return { items };
};

const buildImageTierPreviewItems = (hint, usedGroupRatio, displayPrice, t) => {
  const rows = Array.isArray(hint?.tiers) ? hint.tiers : [];
  const seen = new Set();
  const normalizedItems = [];

  for (const row of rows) {
    const currentUsd = Number(row?.usd_after_channel_discount || 0);
    if (!Number.isFinite(currentUsd) || currentUsd <= 0) continue;

    const family = laneToImagePerImageFamily(row?.lane);
    const resolution =
      formatImageResolutionDisplayLabel(row?.resolution) ||
      row?.resolution ||
      t('默认');
    const key = `${family}|${resolution}`;
    if (seen.has(key)) continue;
    seen.add(key);

    const officialUsd = Number(row?.usd_official || 0);
    const platformUsd = currentUsd * usedGroupRatio;
    const laneKey = IMAGE_PER_IMAGE_LANE_I18N_KEY[row?.lane];
    normalizedItems.push({
      key,
      family,
      resolution: row?.resolution,
      label: t(laneKey || '文生图'),
      spec: resolution,
      platformPrice: displayPrice(platformUsd),
      officialPrice:
        Number.isFinite(officialUsd) && officialUsd > 0
          ? displayPrice(officialUsd)
          : '-',
      discount: getVideoTierDiscount(platformUsd, officialUsd),
      title: `${t(laneKey || '文生图')} ${resolution}`,
    });
  }

  normalizedItems.sort((a, b) => {
    const familyOrder =
      (a.family === 'image_to_image' ? 1 : 0) -
      (b.family === 'image_to_image' ? 1 : 0);
    if (familyOrder !== 0) return familyOrder;
    return compareVideoResolutionAsc(a.resolution, b.resolution);
  });

  // Show both image modes when available, then fill the remaining preview slot.
  const preview = [];
  const remaining = [];
  for (const family of ['text_to_image', 'image_to_image']) {
    const familyRows = normalizedItems.filter((item) => item.family === family);
    if (familyRows.length > 0) preview.push(familyRows[0]);
    remaining.push(...familyRows.slice(1));
  }

  return {
    items: [...preview, ...remaining].slice(0, IMAGE_CARD_TIER_PREVIEW_LIMIT),
  };
};

const groupVideoTierPreviewRows = (rows) => {
  const groups = [];
  for (const row of rows || []) {
    const last = groups[groups.length - 1];
    if (last && last.family === row.family) {
      last.rows.push(row);
      continue;
    }
    groups.push({
      family: row.family,
      label: row.label,
      rows: [row],
    });
  }
  return groups;
};

const getVideoTierGroupStyle = (family) => {
  switch (family) {
    case 'image_to_video':
      return {
        backgroundColor: 'rgba(var(--semi-amber-0), .55)',
        rowBackgroundColor: 'rgba(var(--semi-amber-0), .22)',
        color: 'var(--semi-amber-7)',
        borderColor: 'var(--semi-amber-4)',
      };
    case 'video_to_video':
      return {
        backgroundColor: 'rgba(var(--semi-teal-0), .55)',
        rowBackgroundColor: 'rgba(var(--semi-teal-0), .22)',
        color: 'var(--semi-teal-7)',
        borderColor: 'var(--semi-teal-4)',
      };
    default:
      return {
        backgroundColor: 'rgba(var(--semi-blue-0), .55)',
        rowBackgroundColor: 'rgba(var(--semi-blue-0), .22)',
        color: 'var(--semi-blue-7)',
        borderColor: 'var(--semi-blue-4)',
      };
  }
};

const VideoPipelineLabel = ({ family, title }) => {
  const SourceIcon =
    family === 'image_to_video'
      ? ImageIcon
      : family === 'video_to_video'
        ? Video
        : Type;

  return (
    <Tooltip content={title} position='right' showArrow>
      <span
        className='inline-flex min-w-0 items-center gap-1'
        role='img'
        aria-label={title}
      >
        <SourceIcon size={14} strokeWidth={2} />
        <ArrowRight size={12} strokeWidth={2} />
        <Video size={14} strokeWidth={2} />
      </span>
    </Tooltip>
  );
};

const ImagePipelineLabel = ({ family, title }) => {
  const SourceIcon = family === 'image_to_image' ? ImageIcon : Type;

  return (
    <Tooltip content={title} position='right' showArrow>
      <span
        className='inline-flex min-w-0 items-center gap-1'
        role='img'
        aria-label={title}
      >
        <SourceIcon size={14} strokeWidth={2} />
        <ArrowRight size={12} strokeWidth={2} />
        <ImageIcon size={14} strokeWidth={2} />
      </span>
    </Tooltip>
  );
};

const TOKEN_PRICE_LABELS = {
  cache: { Icon: Database, titleKey: '缓存读取价格' },
  'cache-ratio': { Icon: Database, titleKey: '缓存读取倍率' },
  cache_read: { Icon: HardDriveDownload, titleKey: '缓存读取' },
  cache_write: { Icon: HardDriveUpload, titleKey: '缓存写入' },
  'create-cache-ratio': { Icon: HardDriveUpload, titleKey: '缓存创建倍率' },
};

const TokenPriceLabel = ({ category, t, fallback }) => {
  const meta = TOKEN_PRICE_LABELS[category];
  if (!meta) return fallback;

  const { Icon, titleKey } = meta;
  const title = t(titleKey);
  return (
    <Tooltip content={title} position='right' showArrow>
      <span className='inline-flex' role='img' aria-label={title}>
        <Icon size={15} strokeWidth={2} />
      </span>
    </Tooltip>
  );
};

/**
 * 渲染折扣单元格
 * 折扣视觉效果：Tag 胶囊标签样式，颜色统一 #E74C3C
 */
const renderDiscountCell = (discount, t) => {
  if (discount != null && discount > 0) {
    return (
      <Tag
        size='small'
        shape='circle'
        style={{
          fontSize: 11,
          fontWeight: 700,
          color: '#E74C3C',
          backgroundColor: 'rgba(231, 76, 60, 0.11)',
          border: 'none',
        }}
      >
        {formatPriceRatioFromDiscount(discount, t)}
      </Tag>
    );
  }
  return <span style={DISCOUNT_MUTED_STYLE}>-</span>;
};

/**
 * 渲染官方价单元格
 * 当有折扣时，官方价添加删除线样式
 */
const renderOfficialCell = (officialPrice, hasDiscount) => {
  return (
    <span
      style={{
        color: 'var(--semi-color-text-2)',
        textDecoration: hasDiscount ? 'line-through' : 'none',
      }}
    >
      {officialPrice}
    </span>
  );
};

const normalizePriceUnit = (suffix) => {
  const text = String(suffix || '')
    .replace(/\s+/g, ' ')
    .trim();
  if (!text) return '/M';
  const lower = text.toLowerCase();
  if (lower.includes('token')) {
    if (text.includes('K')) return '/K';
    return '/M';
  }
  if (text.includes('秒')) return '/秒';
  if (text.includes('条')) return '/条';
  if (text.includes('次')) return '/次';
  return text.replace(/\s/g, '').replace(/^\/1/, '/');
};

const getDivPricingColumns = (t, unit = '/M') => ({
  label: t('价格项'),
  platform: `${t('平台价')}${unit}`,
  official: `${t('官方价')}${unit}`,
  discount: t('折扣'),
});

const PRICE_GRID_TEMPLATE = 'repeat(4, minmax(0, 1fr))';
const DIV_PRICING_TABLE_STYLE = {
  borderColor: 'var(--model-price-glass-border, rgba(255, 255, 255, 0.42))',
  background: 'var(--model-price-glass-bg, rgba(255, 255, 255, 0.36))',
  backdropFilter: 'blur(14px) saturate(155%)',
  WebkitBackdropFilter: 'blur(14px) saturate(155%)',
  boxShadow:
    'inset 0 1px 0 var(--model-price-glass-highlight, rgba(255, 255, 255, 0.64))',
};
const DIV_PRICING_HEAD_STYLE = {
  background: 'var(--model-price-glass-head-bg, rgba(255, 255, 255, 0.3))',
  borderBottom:
    '1px solid var(--model-price-glass-line, rgba(148, 163, 184, 0.16))',
  fontSize: 11,
  color: 'var(--model-price-glass-head-text, var(--semi-color-text-2))',
};
const DIV_PRICING_ROW_STYLE = {
  background: 'var(--model-price-glass-row-bg, rgba(255, 255, 255, 0.16))',
  fontSize: 11,
  color: 'var(--model-price-glass-text, var(--semi-color-text-0))',
};
const DIV_PRICING_ROW_BORDER =
  '1px solid var(--model-price-glass-line, rgba(148, 163, 184, 0.14))';

const DivPricingTable = ({ columns, rows, t }) => {
  if (!Array.isArray(rows) || rows.length === 0) return null;

  return (
    <div
      className='w-full min-w-0 overflow-hidden rounded-lg border'
      style={DIV_PRICING_TABLE_STYLE}
    >
      <div
        className='grid items-center'
        style={{
          ...DIV_PRICING_HEAD_STYLE,
          gridTemplateColumns: PRICE_GRID_TEMPLATE,
        }}
      >
        <div className='min-w-0 px-2 py-1 font-semibold whitespace-nowrap overflow-hidden text-ellipsis'>
          {columns.label}
        </div>
        <div className='min-w-0 px-2 py-1 text-right font-medium whitespace-nowrap overflow-hidden text-ellipsis'>
          {columns.platform}
        </div>
        <div className='min-w-0 px-2 py-1 text-right font-medium whitespace-nowrap overflow-hidden text-ellipsis'>
          {columns.official}
        </div>
        <div className='min-w-0 px-2 py-1 text-center font-medium whitespace-nowrap overflow-hidden text-ellipsis'>
          {columns.discount}
        </div>
      </div>

      {rows.map((row, index) => (
        <div
          key={row.key}
          className='grid items-center'
          style={{
            ...DIV_PRICING_ROW_STYLE,
            gridTemplateColumns: PRICE_GRID_TEMPLATE,
            borderBottom:
              index === rows.length - 1 ? 'none' : DIV_PRICING_ROW_BORDER,
          }}
        >
          <div
            className='min-w-0 px-2 py-1.5 font-semibold whitespace-nowrap overflow-hidden text-ellipsis'
            style={{
              color:
                row.color ||
                'var(--model-price-glass-text, var(--semi-color-text-0))',
            }}
          >
            {row.labelNode || row.label}
          </div>
          <div
            className='min-w-0 px-2 py-1.5 text-right font-bold whitespace-nowrap overflow-hidden text-ellipsis'
            style={{
              color:
                'var(--model-price-glass-price, var(--semi-color-primary))',
            }}
            title={row.platformValue}
          >
            {row.platformValue || '-'}
          </div>
          <div className='min-w-0 px-2 py-1.5 text-right font-medium whitespace-nowrap overflow-hidden text-ellipsis'>
            {row.hasOriginal ? (
              renderOfficialCell(row.officialValue, row.discount > 0)
            ) : (
              <span style={{ color: 'var(--semi-color-text-3)' }}>-</span>
            )}
          </div>
          <div className='min-w-0 px-2 py-1.5 text-center whitespace-nowrap overflow-hidden text-ellipsis'>
            {renderDiscountCell(row.discount, t)}
          </div>
        </div>
      ))}
    </div>
  );
};

/**
 * TokenTierTable — 阶梯计费表格
 *
 * 统一四列表格：区间 | 平台价 / M | 官方价 / M | 折扣
 * 边界逻辑：当所有行均无有效折扣数据（discount <= 0 或 null）时，隐藏官方价和折扣列
 * 折扣视觉：加粗、放大、高饱和度鲜艳色
 */
const TokenTierTable = ({ items, t }) => {
  if (!items || !Array.isArray(items.rows) || items.rows.length === 0) {
    return null;
  }

  const boundary = items.boundary || 'lt';

  const tierColumns = getTierPricingColumns(t);
  const firstRow = items.rows[0];
  const inputCell = firstRow.cells?.input;
  const outputCell = firstRow.cells?.output;

  const displayRows = [];
  if (inputCell && inputCell.platformPriceUsd > 0) {
    displayRows.push({ key: 'input', label: t('输入价格'), cell: inputCell });
  }
  if (outputCell && outputCell.platformPriceUsd > 0) {
    displayRows.push({ key: 'output', label: t('输出价格'), cell: outputCell });
  }
  if (displayRows.length === 0) return null;

  const rangeLabel = firstRow.range
    ? firstRow.fromToken === 0 && firstRow.upTo > 0
      ? boundary === 'lte'
        ? `≤ ${formatTierBound(firstRow.upTo)}`
        : `< ${formatTierBound(firstRow.upTo)}`
      : firstRow.range
    : '';

  // 边界隐藏逻辑：所有行都没有有效折扣（discount <= 0 或 null）时，隐藏官方价和折扣列
  const hideOfficialCols = displayRows.every(
    ({ cell }) => cell.discount == null || cell.discount <= 0,
  );

  const tokenTierDivTable = (
    <DivPricingTable
      columns={getDivPricingColumns(t, '/M')}
      t={t}
      rows={displayRows.map(({ key, label, cell }) => {
        const catStyle =
          TIER_CATEGORY_STYLES[key] || TIER_CATEGORY_STYLES.input;
        return {
          key,
          label: rangeLabel ? `${label}·${rangeLabel}` : label,
          labelNode: key === 'input' ? t('输入') : t('输出'),
          platformValue: cell.platformPrice,
          officialValue: cell.officialPrice,
          discount: cell.discount,
          hasOriginal: cell.officialPriceUsd > 0,
          color: catStyle.textColor,
        };
      })}
    />
  );

  return (
    tokenTierDivTable || (
      <div
        className='w-full min-w-0 overflow-hidden rounded-lg border'
        style={PRICING_TABLE_WRAPPER_STYLE}
      >
        <table className='w-full border-collapse' style={{ fontSize: 11 }}>
          <thead>
            <tr style={{ backgroundColor: PRICING_TABLE_HEAD_BG }}>
              <th
                className={TABLE_CELL_CLASS.thLeft}
                style={PRICING_TABLE_HEAD_CELL_STYLE}
              >
                {rangeLabel}
              </th>
              <th
                className={TABLE_CELL_CLASS.thCenter}
                style={PRICING_TABLE_HEAD_CELL_STYLE}
              >
                {tierColumns.platform}
              </th>
              {/* 边界隐藏逻辑：hideOfficialCols 为 true 时隐藏官方价和折扣列 */}
              {!hideOfficialCols && (
                <th
                  className={TABLE_CELL_CLASS.thCenter}
                  style={PRICING_TABLE_HEAD_CELL_STYLE}
                >
                  {tierColumns.official}
                </th>
              )}
              {!hideOfficialCols && (
                <th
                  className={TABLE_CELL_CLASS.thCenter}
                  style={PRICING_TABLE_HEAD_CELL_STYLE}
                >
                  {tierColumns.discount}
                </th>
              )}
            </tr>
          </thead>
          <tbody>
            {displayRows.map(({ key, label, cell }) => {
              const catStyle =
                TIER_CATEGORY_STYLES[key] || TIER_CATEGORY_STYLES.input;
              const showStrike =
                cell.officialPriceUsd > 0 &&
                cell.officialPriceUsd > cell.platformPriceUsd;
              return (
                <tr
                  key={key}
                  style={{
                    backgroundColor: PRICING_TABLE_BODY_BG,
                    borderBottom: PRICING_TABLE_ROW_BORDER,
                  }}
                >
                  <td
                    className={TABLE_CELL_CLASS.tdLabel}
                    style={{ color: catStyle.textColor }}
                  >
                    {label}
                  </td>
                  <td
                    className={TABLE_CELL_CLASS.tdPlatform}
                    style={{ color: 'var(--semi-color-primary)' }}
                  >
                    {cell.platformPrice}
                  </td>
                  {/* 边界隐藏逻辑：hideOfficialCols 为 true 时隐藏官方价和折扣列 */}
                  {!hideOfficialCols && (
                    <td className={TABLE_CELL_CLASS.tdOfficial}>
                      {renderOfficialCell(cell.officialPrice, showStrike)}
                    </td>
                  )}
                  {!hideOfficialCols && (
                    <td className={TABLE_CELL_CLASS.tdDiscount}>
                      {renderDiscountCell(cell.discount, t)}
                    </td>
                  )}
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    )
  );
};

/**
 * FlatPricingTable — 按量计费表格
 *
 * 统一四列表格：价格项 | 平台价 / M | 官方价 / M | 折扣
 * 仅展示输入价格、输出价格两项数据
 * 边界逻辑：当所有行均无有效的官方价/折扣数据时（平台价 ≥ 官方价），
 *   隐藏官方价和折扣列，仅保留价格项和平台价
 * 折扣视觉：加粗、放大、高饱和度鲜艳色
 */
const FlatPricingTable = ({ items, unitSuffix, t }) => {
  const flatColumns = getFlatPricingColumns(t);
  // 收集输入/输出行数据，用于统一表格渲染
  const rows = [];
  for (const item of items) {
    if (item.key !== 'input' && item.key !== 'output') continue;
    // 从 item 中提取平台价、官方价、折扣信息
    const platformValue = item.value || '-';
    const officialValue = item.original?.text || '-';
    const discount = item.original?.discount ?? null;

    rows.push({
      key: item.key,
      label: item.label,
      platformValue,
      officialValue,
      discount,
      hasOriginal: !!item.original,
    });
  }

  if (rows.length === 0) return null;

  // 边界隐藏逻辑：当所有行都没有 original 数据时（说明无有效官方价/折扣），隐藏官方价和折扣列
  // original 存在的条件是 rootValue > minChannel 且 discount > 0
  // 因此 original 不存在意味着：平台价 ≥ 官方价 或 无官方价数据，即折扣 ≤ 0
  const hideOfficialCols = rows.every((r) => !r.hasOriginal);

  const flatDivTable = (
    <DivPricingTable
      columns={getDivPricingColumns(t, normalizePriceUnit(unitSuffix))}
      t={t}
      rows={rows.map((row) => ({
        key: row.key,
        label: row.label,
        labelNode: row.key === 'input' ? t('输入') : t('输出'),
        platformValue: row.platformValue,
        officialValue: row.officialValue,
        discount: row.discount,
        hasOriginal: row.hasOriginal,
        color:
          row.key === 'input' ? 'var(--semi-blue-7)' : 'var(--semi-violet-7)',
      }))}
    />
  );

  return (
    flatDivTable || (
      <div
        className='w-full min-w-0 overflow-hidden rounded-lg border'
        style={PRICING_TABLE_WRAPPER_STYLE}
      >
        <table className='w-full border-collapse' style={{ fontSize: 11 }}>
          <thead>
            <tr style={{ backgroundColor: PRICING_TABLE_HEAD_BG }}>
              <th
                className={TABLE_CELL_CLASS.thLeft}
                style={PRICING_TABLE_HEAD_CELL_STYLE}
              >
                {flatColumns.label}
              </th>
              <th
                className={TABLE_CELL_CLASS.thCenter}
                style={PRICING_TABLE_HEAD_CELL_STYLE}
              >
                {flatColumns.platform}
              </th>
              {/* 边界隐藏逻辑：hideOfficialCols 为 true 时隐藏官方价和折扣列 */}
              {!hideOfficialCols && (
                <th
                  className={TABLE_CELL_CLASS.thCenter}
                  style={PRICING_TABLE_HEAD_CELL_STYLE}
                >
                  {flatColumns.official}
                </th>
              )}
              {!hideOfficialCols && (
                <th
                  className={TABLE_CELL_CLASS.thCenter}
                  style={PRICING_TABLE_HEAD_CELL_STYLE}
                >
                  {flatColumns.discount}
                </th>
              )}
            </tr>
          </thead>
          <tbody>
            {rows.map((row) => (
              <tr
                key={row.key}
                style={{
                  backgroundColor: PRICING_TABLE_BODY_BG,
                  borderBottom: PRICING_TABLE_ROW_BORDER,
                }}
              >
                <td
                  className={TABLE_CELL_CLASS.tdLabel}
                  style={{
                    color:
                      row.key === 'input'
                        ? 'var(--semi-blue-7)'
                        : 'var(--semi-violet-7)',
                  }}
                >
                  {row.label}
                </td>
                <td
                  className={TABLE_CELL_CLASS.tdPlatform}
                  style={{ color: 'var(--semi-color-primary)' }}
                >
                  {row.platformValue}
                </td>
                {!hideOfficialCols && (
                  <td className={TABLE_CELL_CLASS.tdOfficial}>
                    {row.hasOriginal ? (
                      renderOfficialCell(row.officialValue, row.discount > 0)
                    ) : (
                      <span style={{ color: 'var(--semi-color-text-3)' }}>
                        -
                      </span>
                    )}
                  </td>
                )}
                {!hideOfficialCols && (
                  <td className={TABLE_CELL_CLASS.tdDiscount}>
                    {renderDiscountCell(row.discount, t)}
                  </td>
                )}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    )
  );
};

/**
 * FixedPricingTable — 按次计费表格
 *
 * 统一四列表格：价格项 | 平台价/次 | 官方价/次 | 折扣
 * 仅展示模型价格一项数据
 * 边界逻辑：当无有效的官方价/折扣数据时，隐藏官方价和折扣列
 * 折扣视觉：加粗、放大、颜色统一 #E74C3C
 */
const FixedPricingTable = ({ row, t }) => {
  if (!row) return null;

  const fixedColumns = getFixedPricingColumns(t);
  const platformValue = row.value || '-';
  const officialValue = row.original?.text || '-';
  const discount = row.original?.discount ?? null;
  const hasOriginal = !!row.original;

  // 边界隐藏逻辑：无 original 数据时隐藏官方价和折扣列
  const hideOfficialCols = !hasOriginal;

  const fixedDivTable = (
    <DivPricingTable
      columns={getDivPricingColumns(t, normalizePriceUnit(row.suffix))}
      t={t}
      rows={[
        {
          key: 'fixed',
          label: row.label,
          platformValue,
          officialValue,
          discount,
          hasOriginal,
          color: 'var(--semi-color-teal-7)',
        },
      ]}
    />
  );

  return (
    fixedDivTable || (
      <div
        className='w-full min-w-0 overflow-hidden rounded-lg border'
        style={PRICING_TABLE_WRAPPER_STYLE}
      >
        <table className='w-full border-collapse' style={{ fontSize: 11 }}>
          <thead>
            <tr style={{ backgroundColor: PRICING_TABLE_HEAD_BG }}>
              <th
                className={TABLE_CELL_CLASS.thLeft}
                style={PRICING_TABLE_HEAD_CELL_STYLE}
              >
                {fixedColumns.label}
              </th>
              <th
                className={TABLE_CELL_CLASS.thCenter}
                style={PRICING_TABLE_HEAD_CELL_STYLE}
              >
                {fixedColumns.platform}
              </th>
              {/* 边界隐藏逻辑：hideOfficialCols 为 true 时隐藏官方价和折扣列 */}
              {!hideOfficialCols && (
                <th
                  className={TABLE_CELL_CLASS.thCenter}
                  style={PRICING_TABLE_HEAD_CELL_STYLE}
                >
                  {fixedColumns.official}
                </th>
              )}
              {!hideOfficialCols && (
                <th
                  className={TABLE_CELL_CLASS.thCenter}
                  style={PRICING_TABLE_HEAD_CELL_STYLE}
                >
                  {fixedColumns.discount}
                </th>
              )}
            </tr>
          </thead>
          <tbody>
            <tr
              style={{
                backgroundColor: PRICING_TABLE_BODY_BG,
                borderBottom: PRICING_TABLE_ROW_BORDER,
              }}
            >
              <td
                className={TABLE_CELL_CLASS.tdLabel}
                style={{ color: 'var(--semi-color-teal-7)' }}
              >
                {row.label}
              </td>
              <td
                className={TABLE_CELL_CLASS.tdPlatform}
                style={{ color: 'var(--semi-color-primary)' }}
              >
                {platformValue}
              </td>
              {!hideOfficialCols && (
                <td className={TABLE_CELL_CLASS.tdOfficial}>
                  {hasOriginal ? (
                    renderOfficialCell(officialValue, discount > 0)
                  ) : (
                    <span style={{ color: 'var(--semi-color-text-3)' }}>-</span>
                  )}
                </td>
              )}
              {!hideOfficialCols && (
                <td className={TABLE_CELL_CLASS.tdDiscount}>
                  {renderDiscountCell(discount, t)}
                </td>
              )}
            </tr>
          </tbody>
        </table>
      </div>
    )
  );
};

/**
 * VideoPricingTable — 视频类型计费表格
 *
 * 每个视频分组（文生视频/图生视频/视频生视频）有独立的表头行，
 * 分组名称作为首列标题，后跟 平台价 | 官方价 | 折扣
 * 数据行按分辨率/时长维度填写
 * 边界逻辑：当所有行均无有效折扣数据（平台价 ≥ 官方价 或 无官方价）时，隐藏官方价和折扣列
 * 折扣视觉：加粗、放大、高饱和度鲜艳色
 */
const VideoPricingTable = ({ videoTierRows, videoBillingMode, t }) => {
  if (!videoTierRows || videoTierRows.length === 0) return null;

  const videoColumns = getVideoPricingColumns(t);
  const unitSuffix =
    videoBillingMode === 'per_second'
      ? ` / ${t('秒')}`
      : videoBillingMode === 'per_token'
        ? ' / M token'
        : ` / ${t('条')}`;
  const groups = groupVideoTierPreviewRows(videoTierRows);

  // 边界隐藏逻辑：所有行都没有有效折扣（discount <= 0 或 null）时，隐藏官方价和折扣列
  // discount > 0 表示平台价 < 官方价（有折扣），不应隐藏
  // discount <= 0 或 null 表示平台价 ≥ 官方价 或 无数据，应隐藏
  const allRowsNoDiscount = videoTierRows.every(
    (row) => row.discount == null || row.discount <= 0,
  );

  const videoDivTable = (
    <DivPricingTable
      columns={getDivPricingColumns(t, normalizePriceUnit(unitSuffix))}
      t={t}
      rows={groups.flatMap((group) =>
        group.rows.map((row) => {
          const groupStyle = getVideoTierGroupStyle(group.family);
          return {
            key: row.key,
            label: row.spec ? `${group.label}·${row.spec}` : group.label,
            labelNode: (
              <VideoPipelineLabel
                family={row.family}
                title={`${row.title} ${row.spec}`}
              />
            ),
            platformValue: row.platformPrice,
            officialValue: row.officialPrice,
            discount: row.discount,
            hasOriginal: row.officialPrice && row.officialPrice !== '-',
            color: groupStyle.color,
            title: `${row.title} ${row.spec} ${row.audioLabel}`,
          };
        }),
      )}
    />
  );

  return (
    videoDivTable || (
      <div
        className='w-full min-w-0 overflow-hidden rounded-lg border'
        style={PRICING_TABLE_WRAPPER_STYLE}
      >
        {groups.map((group, groupIdx) => (
          <table
            key={group.family}
            className='w-full border-collapse'
            style={{ fontSize: 11 }}
          >
            {/* 分组表头行：分组名称作为首列标题，后跟 平台价 | 官方价 | 折扣 */}
            <thead>
              <tr
                style={{
                  backgroundColor: getVideoTierGroupStyle(group.family)
                    .backgroundColor,
                }}
              >
                <th
                  className={TABLE_CELL_CLASS.thLeft}
                  style={{
                    borderBottom: PRICING_TABLE_ROW_BORDER,
                    color: getVideoTierGroupStyle(group.family).color,
                  }}
                >
                  <div className='flex items-center gap-1'>
                    <span
                      className='h-3 w-0.5 rounded-full'
                      style={{
                        backgroundColor: getVideoTierGroupStyle(group.family)
                          .borderColor,
                      }}
                    />
                    <span className='font-semibold'>{group.label}</span>
                  </div>
                </th>
                <th
                  className={TABLE_CELL_CLASS.thCenter}
                  style={{
                    borderBottom: PRICING_TABLE_ROW_BORDER,
                    color: 'var(--semi-color-text-2)',
                  }}
                >
                  {videoColumns.platform}
                  {unitSuffix}
                </th>
                {/* 边界隐藏逻辑：allRowsNoDiscount 为 true 时隐藏官方价和折扣列 */}
                {!allRowsNoDiscount && (
                  <th
                    className={TABLE_CELL_CLASS.thCenter}
                    style={{
                      borderBottom: PRICING_TABLE_ROW_BORDER,
                      color: 'var(--semi-color-text-2)',
                    }}
                  >
                    {videoColumns.official}
                    {unitSuffix}
                  </th>
                )}
                {!allRowsNoDiscount && (
                  <th
                    className={TABLE_CELL_CLASS.thCenter}
                    style={{
                      borderBottom: PRICING_TABLE_ROW_BORDER,
                      color: 'var(--semi-color-text-2)',
                    }}
                  >
                    {videoColumns.discount}
                  </th>
                )}
              </tr>
            </thead>
            <tbody>
              {group.rows.map((row) => (
                <tr
                  key={row.key}
                  style={{
                    backgroundColor: PRICING_TABLE_BODY_BG,
                    borderBottom: PRICING_TABLE_ROW_BORDER,
                  }}
                >
                  <td
                    className={TABLE_CELL_CLASS.tdLabel}
                    style={{
                      color: getVideoTierGroupStyle(group.family).color,
                    }}
                    title={`${row.title} · ${row.audioLabel}`}
                  >
                    {row.spec}
                  </td>
                  <td
                    className={TABLE_CELL_CLASS.tdPlatform}
                    style={{ color: 'var(--semi-color-primary)' }}
                  >
                    {row.platformPrice}
                  </td>
                  {/* 边界隐藏逻辑：allRowsNoDiscount 为 true 时隐藏官方价和折扣列 */}
                  {!allRowsNoDiscount && (
                    <td className={TABLE_CELL_CLASS.tdOfficial}>
                      {renderOfficialCell(row.officialPrice, row.discount > 0)}
                    </td>
                  )}
                  {!allRowsNoDiscount && (
                    <td className={TABLE_CELL_CLASS.tdDiscount}>
                      {renderDiscountCell(row.discount, t)}
                    </td>
                  )}
                </tr>
              ))}
            </tbody>
          </table>
        ))}
      </div>
    )
  );
};

const ImagePricingTable = ({ imageTierRows, t }) => {
  if (!imageTierRows || imageTierRows.length === 0) return null;

  return (
    <DivPricingTable
      columns={getDivPricingColumns(t, `/${t('张')}`)}
      t={t}
      rows={imageTierRows.map((row) => ({
        key: row.key,
        label: `${row.label}·${row.spec}`,
        labelNode: <ImagePipelineLabel family={row.family} title={row.title} />,
        platformValue: row.platformPrice,
        officialValue: row.officialPrice,
        discount: row.discount,
        hasOriginal: row.officialPrice !== '-',
        color:
          row.family === 'image_to_image'
            ? 'var(--semi-green-7)'
            : 'var(--semi-blue-7)',
        title: row.title,
      }))}
    />
  );
};

const PricingCardView = ({
  filteredModels,
  loading,
  rowSelection,
  pageSize,
  setPageSize,
  currentPage,
  setCurrentPage,
  selectedGroup,
  groupRatio,
  groupModelPrice,
  groupModelRatio,
  copyText,
  setModalImageUrl,
  setIsModalOpenurl,
  currency,
  siteDisplayType,
  tokenUnit,
  displayPrice,
  showRatio,
  t,
  selectedRowKeys = [],
  setSelectedRowKeys,
  openModelDetail,
  showSizeChanger = true,
  blurPricing = false,
  homeCardMode = false,
  showModelDescription = false,
  searchValue = '',
  channelVideoRatio = {},
  channelVideoCompletionRatio = {},
  channelVideoPrice = {},
  perfMetricsMap = {},
  hotChannelScoreMap = new Map(),
  filterSupplier = 'all',
  filterSupplierType = 'all',
}) => {
  const { i18n } = useTranslation();
  const showSkeleton = useMinimumLoadingTime(loading);
  const startIndex = (currentPage - 1) * pageSize;
  const paginatedModels = filteredModels.slice(
    startIndex,
    startIndex + pageSize,
  );
  const getModelKey = (model) => model.key ?? model.model_name ?? model.id;
  const isMobile = useIsMobile();
  const normalizedSearchValue = String(searchValue || '').trim();
  const homeBottomTagLimit = isMobile ? 2 : 3;

  const renderHighlightedText = (value) => {
    const text = value == null ? '' : String(value);
    if (!normalizedSearchValue) return text;
    const regex = new RegExp(`(${escapeRegExp(normalizedSearchValue)})`, 'ig');
    return text.split(regex).map((part, idx) =>
      part.toLowerCase() === normalizedSearchValue.toLowerCase() ? (
        <span
          key={idx}
          style={{
            color: '#ef4444',
            fontWeight: 700,
            backgroundColor: 'rgba(239, 68, 68, 0.12)',
            borderRadius: 4,
          }}
        >
          {part}
        </span>
      ) : (
        part
      ),
    );
  };

  const renderPriceItem = (item) => (
    <React.Fragment key={item.key}>
      {item.flatTableRows ? (
        <FlatPricingTable
          items={item.flatTableRows}
          unitSuffix={item.unitSuffix}
          t={t}
        />
      ) : item.fixedTableRow ? (
        <FixedPricingTable row={item.fixedTableRow} t={t} />
      ) : item.videoTierRows ? (
        <VideoPricingTable
          videoTierRows={item.videoTierRows}
          videoBillingMode={item.videoBillingMode}
          t={t}
        />
      ) : item.imageTierRows ? (
        <ImagePricingTable imageTierRows={item.imageTierRows} t={t} />
      ) : item.tokenTierMerged ? (
        <TokenTierTable items={item.tokenTierMerged} t={t} />
      ) : !item.valueNode && item.value ? (
        <DivPricingTable
          columns={getDivPricingColumns(t, normalizePriceUnit(item.suffix))}
          t={t}
          rows={[
            {
              key: item.key,
              label: item.label,
              labelNode:
                item.key === 'input' || item.key === 'input-ratio'
                  ? t('输入')
                  : item.key === 'completion' || item.key === 'completion-ratio'
                    ? t('输出')
                    : undefined,
              platformValue: item.value,
              officialValue: item.original?.text,
              discount: item.original?.discount,
              hasOriginal: !!item.original,
              color: item.labelColor
                ? `var(--semi-${item.labelColor}-7)`
                : undefined,
            },
          ]}
        />
      ) : (
        <div className='flex items-center'>
          <span className='w-20 flex-shrink-0'>
            {item.labelColor ? (
              <Tag
                color={item.labelColor}
                size='small'
                shape='circle'
                type='light'
                className='max-w-full'
              >
                {item.label}
              </Tag>
            ) : (
              item.label
            )}
          </span>
          <span className='flex-1 font-bold text-black dark:text-gray-100 inline-flex items-center flex-wrap gap-1'>
            {item.valueNode ? (
              item.valueNode
            ) : item.original ? (
              <>
                <span className='line-through text-gray-400 font-normal text-[10px]'>
                  <span style={{ color: 'var(--semi-color-primary)' }}>
                    官方
                  </span>{' '}
                  {item.original.text}
                </span>
                <Tag
                  size='small'
                  shape='circle'
                  style={{
                    fontSize: 11,
                    fontWeight: 700,
                    color: '#E74C3C',
                    backgroundColor: 'rgba(231, 76, 60, 0.11)',
                    border: 'none',
                  }}
                >
                  {formatPriceRatioFromDiscount(item.original.discount, t)}
                </Tag>
                <span>
                  <span style={{ color: 'var(--semi-color-warning)' }}>
                    我们
                  </span>{' '}
                  {item.value}
                  {item.suffix}
                </span>
              </>
            ) : (
              <span
                className={
                  item.labelColor
                    ? 'inline-flex min-w-0 flex-wrap items-baseline gap-1'
                    : undefined
                }
                title={item.title}
              >
                <span>{item.value}</span>
                {item.suffix ? (
                  <span className='font-normal text-[10px] text-semi-color-text-2'>
                    {item.suffix}
                  </span>
                ) : null}
              </span>
            )}
          </span>
        </div>
      )}
    </React.Fragment>
  );

  const handleCheckboxChange = (model, checked) => {
    if (!setSelectedRowKeys) return;
    const modelKey = getModelKey(model);
    const newKeys = checked
      ? Array.from(new Set([...selectedRowKeys, modelKey]))
      : selectedRowKeys.filter((key) => key !== modelKey);
    setSelectedRowKeys(newKeys);
    rowSelection?.onChange?.(newKeys, null);
  };

  // 根据 supplier_type 返回对应的 Tag 颜色
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

  // 根据模型的 channel_list 推导可展示的供应商项。
  // 无 logo 时不展示 supplier_alias，仅保留供应商类型标签。
  const getSupplierLogos = (model) => {
    if (!model?.channel_list || model.channel_list.length === 0) return [];
    const seen = new Set();
    const items = [];
    model.channel_list.forEach((ch, idx) => {
      const logo =
        (ch?.company_logo_url && String(ch.company_logo_url).trim()) || '';
      const supplierType =
        (ch?.supplier_type && String(ch.supplier_type).trim()) || '';
      const alias =
        (ch?.supplier_alias && String(ch.supplier_alias).trim()) || '';
      const name = ch?.channel_name || '';
      if (!logo && !supplierType) return;
      const displayAlias = logo ? alias : '';
      const dedupKey = `${logo}|${supplierType}|${displayAlias}`;
      if (seen.has(dedupKey)) return;
      seen.add(dedupKey);
      items.push({
        key: ch?.channel_id ?? `${dedupKey}-${idx}`,
        logo,
        supplierType,
        alias: displayAlias,
        name,
      });
    });
    return items;
  };

  const collectDiscountsFromPriceItem = (item) => {
    const discounts = [];
    if (item?.original?.discount > 0) discounts.push(item.original.discount);
    if (item?.fixedTableRow?.original?.discount > 0) {
      discounts.push(item.fixedTableRow.original.discount);
    }
    if (Array.isArray(item?.flatTableRows)) {
      item.flatTableRows.forEach((row) => {
        if (row?.original?.discount > 0) discounts.push(row.original.discount);
      });
    }
    if (Array.isArray(item?.videoTierRows)) {
      item.videoTierRows.forEach((row) => {
        if (row?.discount > 0) discounts.push(row.discount);
      });
    }
    if (item?.tokenTierMerged?.rows) {
      item.tokenTierMerged.rows.forEach((row) => {
        Object.values(row?.cells || {}).forEach((cell) => {
          if (cell?.discount > 0) discounts.push(cell.discount);
        });
      });
    }
    return discounts;
  };

  const getBestDiscount = (priceItems) => {
    const discounts = priceItems.flatMap(collectDiscountsFromPriceItem);
    if (discounts.length === 0) return null;
    return Math.max(...discounts);
  };

  const formatDiscountBadge = (discount) => {
    if (!(discount > 0)) return '';
    return formatPriceRatioFromDiscount(discount, t);
  };

  const getPrimarySupplierType = (model) => {
    const channelList = Array.isArray(model?.channel_list)
      ? model.channel_list
      : Array.isArray(model?.ChannelList)
        ? model.ChannelList
        : [];
    const supplierType = channelList
      .map((ch) => String(ch?.supplier_type || '').trim())
      .find(Boolean);
    return supplierType || '';
  };

  const getHomeTagItems = (record) => {
    const channelQuotaType =
      record.channel_list && record.channel_list.length > 0
        ? record.channel_list[0].quota_type
        : record.quota_type;

    const items = [];
    const pushItem = (item) => {
      const key = String(item?.key || item?.text || '')
        .trim()
        .toLowerCase();
      if (!key || items.some((existing) => existing.key === key)) return;
      items.push({ ...item, key });
    };

    if (channelQuotaType === 1) {
      pushItem({ text: t('按次计费'), color: 'teal' });
    } else if (channelQuotaType === 3) {
      pushItem({ text: t('阶梯计费'), color: 'orange' });
    } else if (channelQuotaType === 0) {
      pushItem({ text: t('按量计费'), color: 'violet' });
    }

    if (record.tags) {
      record.tags
        .split(',')
        .map((tag) => tag.trim())
        .filter(Boolean)
        .filter((tag) => tag !== '热门' && tag.toLowerCase() !== 'hot')
        .forEach((tag) => {
          pushItem({
            key: `tag-${tag}`,
            text: getModelTagLabel(tag, t),
            color: stringToColor(tag),
          });
        });
    }

    return items;
  };

  const renderHomeTag = (item) => (
    <Tag
      key={item.key}
      shape='circle'
      color={item.color || 'white'}
      size='small'
      className='home-model-extra-tag max-w-[86px] shrink-0'
    >
      <span className='block min-w-0 truncate'>
        {renderHighlightedText(item.text)}
      </span>
    </Tag>
  );

  const renderHomePopoverTag = (item) => (
    <Tag
      key={`popover-${item.key}`}
      shape='circle'
      color={item.color || 'white'}
      size='small'
      className='home-model-tag-popover-item'
    >
      {renderHighlightedText(item.text)}
    </Tag>
  );

  const renderHomeBottomTags = (record) => {
    const tagItems = getHomeTagItems(record);
    const visibleItems = tagItems.slice(0, homeBottomTagLimit);
    const hiddenItems = tagItems.slice(homeBottomTagLimit);

    return (
      <div className='home-model-bottom-tags'>
        {visibleItems.map(renderHomeTag)}
        {hiddenItems.length > 0 ? (
          <Tooltip
            content={
              <div className='home-model-tag-popover'>
                {tagItems.map(renderHomePopoverTag)}
              </div>
            }
            position='top'
            showArrow
          >
            <Tag
              shape='circle'
              color='white'
              size='small'
              className='home-model-more-tag shrink-0'
            >
              +{hiddenItems.length}
            </Tag>
          </Tooltip>
        ) : null}
      </div>
    );
  };

  const calculateChannelPrices = (model, opts = {}) => {
    const { skipSimpleVideoFlat = false, skipSimpleFixed = false } = opts;
    if (!model.channel_list || model.channel_list.length === 0) {
      return null;
    }

    const { usedGroupRatio } = getUsedGroupContext(
      model,
      selectedGroup,
      groupRatio,
    );

    // 辅助函数：格式化价格
    const formatPrice = (priceUSD) => {
      const unitDivisor = tokenUnit === 'K' ? 1000 : 1;
      const rawDisplayPrice = displayPrice(priceUSD / unitDivisor);
      const numericPrice = truncateModelPriceValue(
        parseFloat(rawDisplayPrice.replace(/[^0-9.]/g, '')),
        MODEL_CARD_PRICE_MAX_DECIMALS,
      );

      let symbol = '$';
      if (currency === 'CNY') {
        symbol = '¥';
      } else if (currency === 'CUSTOM') {
        try {
          const statusStr = localStorage.getItem('status');
          if (statusStr) {
            const s = JSON.parse(statusStr);
            symbol = s?.custom_currency_symbol || '¤';
          }
        } catch (e) {
          symbol = '¤';
        }
      }

      return {
        value: numericPrice,
        rawUsd: Number(priceUSD) || 0,
        symbol,
      };
    };

    const modelHasVideoFlatPrice = hasNumericValue(model.video_price);
    const modelHasASRPrice = isASRPricingModel(model);
    const hideTextTokenPrices = isVideoPricingModel(model) || modelHasASRPrice;

    // 提取所有通道的价格（与 relay 一致：ch.model_ratio 已含渠道折扣；再乘分组倍率）
    const prices = {
      input: [],
      output: [],
      cache: [],
      createCache: [],
      fixed: [],
      videoFlat: [],
      asrPrice: [],
    };
    const originalPrices = {
      input: [],
      output: [],
      cache: [],
      createCache: [],
      fixed: [],
      videoFlat: [],
      asrPrice: [],
    };

    const displayChannels = homeCardMode
      ? model.channel_list.slice(0, 1)
      : model.channel_list;

    displayChannels.forEach((ch) => {
      const cid = ch.channel_id;
      const mname = model.model_name;

      // ============================================================
      // 新计费公式参数：
      //   ch.model_ratio / ch.model_price 为原始渠道倍率（后端不再预乘成本折扣）
      //   成本折扣率 = price_discount_percent / 100
      //   加价倍率   = markup_discount_rate / 100
      //
      //   输入   = (ch.model_ratio × costDisc + globalMr × markupRate) × 2 × groupRatio
      //   输出   = (ch.model_ratio × cr × costDisc + globalMr × globalCR × markupRate) × 2 × groupRatio
      //   缓存读 = (ch.model_ratio × cacheRatio × costDisc + globalMr × globalCacheR × markupRate) × 2 × groupRatio
      //   缓存写 = (ch.model_ratio × createCacheRatio × costDisc + globalMr × globalCreateCacheR × markupRate) × 2 × groupRatio
      //   固定价 = (ch.model_price × costDisc + globalMp × markupRate) × groupRatio
      // ============================================================
      const priceDiscountPercent =
        ch.price_discount_percent != null ? ch.price_discount_percent : 100;
      const markupDiscountPercent = ch.markup_discount_rate || 0;
      const globalMr = model.model_ratio || 0;
      const globalMp = model.model_price || 0;

      const channelVideoFlatUsd = pickChannelScopedModelFloat(
        channelVideoPrice,
        cid,
        mname,
      );
      const flatUsd =
        channelVideoFlatUsd != null
          ? channelVideoFlatUsd
          : modelHasVideoFlatPrice
            ? Number(model.video_price)
            : null;
      if (!skipSimpleVideoFlat && flatUsd != null && flatUsd > 0) {
        prices.videoFlat.push(formatPrice(flatUsd * usedGroupRatio));
        originalPrices.videoFlat.push(formatPrice(flatUsd));
      }

      // ASR 语音识别：
      //   官方价 = 全局 asr_price（美元/秒）
      //   平台价 = (渠道价（空则全局价）× 成本折扣率 + 全局价 × 加价折扣率) × 分组倍率
      //   其中 price_discount_percent 已含经营成本率，与 EffectiveModelPrice 一致
      if (modelHasASRPrice) {
        const globalAsrUsd = Number(model.asr_price);
        if (globalAsrUsd > 0) {
          const channelAsrUsd = globalAsrUsd; // 暂无渠道级 ASR 价，空则回退全局
          const costDisc = priceDiscountPercent / 100;
          const markupRate = markupDiscountPercent / 100;
          const platformAsrUsd =
            channelAsrUsd * costDisc + globalAsrUsd * markupRate;
          prices.asrPrice.push(formatPrice(platformAsrUsd * usedGroupRatio));
          originalPrices.asrPrice.push(formatPrice(platformAsrUsd));
        }
      }

      // 全局子倍率（用于加价侧）
      const globalCR = model.completion_ratio || 0;
      const globalCacheR =
        model.cache_ratio != null ? Number(model.cache_ratio) : 0;
      const globalCreateCacheR =
        model.create_cache_ratio != null ? Number(model.create_cache_ratio) : 0;

      const billingRates = computeChannelBillingRates({
        channelModelRatio: ch.model_ratio,
        channelCompletionRatio: ch.completion_ratio,
        channelCacheRatio: ch.cache_ratio,
        channelCreateCacheRatio: ch.create_cache_ratio,
        channelModelPrice: ch.model_price,
        priceDiscountPercent,
        markupDiscountPercent,
        globalModelRatio: globalMr,
        globalModelPrice: globalMp,
        globalCompletionRatio: globalCR,
        globalCacheRatio: globalCacheR,
        globalCreateCacheRatio: globalCreateCacheR,
      });

      // 按量计费
      if (model.quota_type === 0) {
        if (ch.model_ratio !== undefined && ch.model_ratio !== null) {
          if (!hideTextTokenPrices) {
            prices.input.push(
              formatPrice(billingRates.inputRatioPrice * usedGroupRatio),
            );
            originalPrices.input.push(
              formatPrice(billingRates.inputRatioPrice),
            );
          }

          // 输出价格：仅当全局模型配置了 completion_ratio 时才展示
          if (
            !hideTextTokenPrices &&
            ch.completion_ratio !== undefined &&
            ch.completion_ratio !== null &&
            model.completion_ratio != null
          ) {
            prices.output.push(
              formatPrice(billingRates.completionRatioPrice * usedGroupRatio),
            );
            originalPrices.output.push(
              formatPrice(billingRates.completionRatioPrice),
            );
          }

          // 缓存读取价格：仅当全局模型配置了 cache_ratio 时才展示
          if (
            !hideTextTokenPrices &&
            ch.cache_ratio !== undefined &&
            ch.cache_ratio !== null &&
            model.cache_ratio != null
          ) {
            prices.cache.push(
              formatPrice(billingRates.cacheRatioPrice * usedGroupRatio),
            );
            originalPrices.cache.push(
              formatPrice(billingRates.cacheRatioPrice),
            );
          }

          // 缓存创建价格：仅当全局模型配置了 create_cache_ratio 时才展示
          if (
            !hideTextTokenPrices &&
            ch.create_cache_ratio !== undefined &&
            ch.create_cache_ratio !== null &&
            model.create_cache_ratio != null
          ) {
            prices.createCache.push(
              formatPrice(
                billingRates.cacheCreationRatioPrice * usedGroupRatio,
              ),
            );
            originalPrices.createCache.push(
              formatPrice(billingRates.cacheCreationRatioPrice),
            );
          }
        }
      }
      // 按次计费
      else if (model.quota_type === 1 || ch.quota_type === 1) {
        if (
          !skipSimpleFixed &&
          ch.model_price !== undefined &&
          ch.model_price !== null
        ) {
          prices.fixed.push(
            formatPrice(billingRates.effModelPrice * usedGroupRatio),
          );
          originalPrices.fixed.push(formatPrice(billingRates.effModelPrice));
        }
      }
    });

    // 根数据价格（用同一口径计算，用于与 channel 价格比较）
    const rootPrices = {};
    if (model.quota_type === 0 && !hideTextTokenPrices) {
      if (model.model_ratio !== undefined && model.model_ratio !== null) {
        rootPrices.input = formatPrice(model.model_ratio * 2);
        if (
          model.completion_ratio !== undefined &&
          model.completion_ratio !== null
        ) {
          rootPrices.output = formatPrice(
            model.model_ratio * model.completion_ratio * 2,
          );
        }
        if (model.cache_ratio !== undefined && model.cache_ratio !== null) {
          rootPrices.cache = formatPrice(
            model.model_ratio * model.cache_ratio * 2,
          );
        }
        if (
          model.create_cache_ratio !== undefined &&
          model.create_cache_ratio !== null
        ) {
          rootPrices.createCache = formatPrice(
            model.model_ratio * model.create_cache_ratio * 2,
          );
        }
      }
    } else if (model.quota_type === 1) {
      if (
        !skipSimpleFixed &&
        model.model_price !== undefined &&
        model.model_price !== null
      ) {
        rootPrices.fixed = formatPrice(model.model_price);
      }
    }
    if (modelHasVideoFlatPrice && !skipSimpleVideoFlat) {
      rootPrices.videoFlat = formatPrice(Number(model.video_price));
    }
    if (modelHasASRPrice) {
      rootPrices.asrPrice = formatPrice(Number(model.asr_price));
    }

    // 若根价格高于任意一个 channel 的对应价格，则返回划线原价与折扣
    const getOriginal = (rootPrice, channelPriceArray) => {
      if (!rootPrice || !channelPriceArray || channelPriceArray.length === 0)
        return null;
      const rootValue = rootPrice.rawUsd ?? rootPrice.value;
      const minChannel = Math.min(
        ...channelPriceArray.map((p) => p.rawUsd ?? p.value),
      );
      if (rootValue > minChannel && rootValue > 0) {
        const discount = Math.round((1 - minChannel / rootValue) * 100);
        return {
          text: `${rootPrice.symbol}${rootPrice.value}`,
          discount,
        };
      }
      return null;
    };

    // 计算范围
    const calculateRange = (priceArray) => {
      if (priceArray.length === 0) return null;
      if (priceArray.length === 1) {
        const p = priceArray[0];
        return {
          single: `${p.symbol}${p.value}`,
          min: null,
          max: null,
          symbol: p.symbol,
        };
      }

      const values = priceArray.map((p) => p.value);
      const uniqueValues = [...new Set(values)];

      if (uniqueValues.length === 1) {
        const p = priceArray[0];
        return {
          single: `${p.symbol}${p.value}`,
          min: null,
          max: null,
          symbol: p.symbol,
        };
      }

      const min = Math.min(...values);
      const max = Math.max(...values);
      const symbol = priceArray[0].symbol;
      return { single: null, min, max, symbol };
    };

    const unitLabel = tokenUnit === 'K' ? 'K' : 'M';
    const unitSuffix = ` / 1${unitLabel} Tokens`;
    const fixedSuffix = ` / ${t('次')}`;

    return {
      input: calculateRange(prices.input),
      output: calculateRange(prices.output),
      cache: calculateRange(prices.cache),
      createCache: calculateRange(prices.createCache),
      fixed: calculateRange(prices.fixed),
      original: {
        input: getOriginal(rootPrices.input, originalPrices.input),
        output: getOriginal(rootPrices.output, originalPrices.output),
        cache: getOriginal(rootPrices.cache, originalPrices.cache),
        createCache: getOriginal(
          rootPrices.createCache,
          originalPrices.createCache,
        ),
        fixed: getOriginal(rootPrices.fixed, originalPrices.fixed),
        videoFlat: getOriginal(rootPrices.videoFlat, originalPrices.videoFlat),
        asrPrice: getOriginal(rootPrices.asrPrice, originalPrices.asrPrice),
      },
      videoFlat: calculateRange(prices.videoFlat),
      asrPrice: calculateRange(prices.asrPrice),
      unitSuffix,
      fixedSuffix,
      videoFlatSuffix: ` / ${t('条')}`,
      asrPriceSuffix: ` / ${t('秒')}`,
      quotaType:
        model.channel_list?.[0]?.quota_type != null
          ? model.channel_list[0].quota_type
          : model.quota_type,
    };
  };

  const buildVideoPriceCardItems = ({
    model,
    hint,
    videoFlat,
    videoFlatSuffix,
    original,
    useTieredVideoFlat,
  }) => {
    const videoBillingMode = String(hint?.billing_mode || '');
    const out = [];

    if (useTieredVideoFlat) {
      const { usedGroupRatio } = getUsedGroupContext(
        model,
        selectedGroup,
        groupRatio,
      );
      const videoPreview = buildVideoTierPreviewItems(
        hint,
        usedGroupRatio,
        displayPrice,
        t,
      );
      if (videoPreview.items.length > 0) {
        out.push({
          key: 'video-tier-table',
          videoTierRows: videoPreview.items,
          videoBillingMode,
        });
      }
    } else if (videoFlat) {
      const flatSuffix =
        videoBillingMode === 'per_second'
          ? ` / ${t('秒')}`
          : videoBillingMode === 'per_token'
            ? ' / M token'
            : videoFlatSuffix || ` / ${t('条')}`;
      out.push({
        key: 'video-flat',
        label:
          videoBillingMode === 'per_token'
            ? t('视频按 token 计费')
            : videoBillingMode === 'per_second'
              ? t('视频按秒计费')
              : t('视频按条（固定价）'),
        value:
          videoFlat.single ||
          `${videoFlat.symbol}${videoFlat.min} ~ ${videoFlat.symbol}${videoFlat.max}`,
        suffix: flatSuffix,
        original: original?.videoFlat,
      });
    }
    return out;
  };

  // 获取模型的价格项（优先使用 channel 价格）
  const getModelPriceItemsForCard = (model, priceData) => {
    const hint = model.video_flat_clip_hint;
    const useTieredVideoFlat =
      hint &&
      Number(hint.tier_count) > 0 &&
      Number(hint.min_usd_after_channel_discount) > 0;
    const isVideoModel = isVideoPricingModel(model);
    const imageHint = model.image_per_image_hint;
    const useTieredImagePerImage =
      imageHint &&
      Number(imageHint.tier_count) > 0 &&
      Number(imageHint.min_usd_after_channel_discount) > 0;

    if (homeCardMode && useTieredImagePerImage) {
      const { usedGroupRatio } = getUsedGroupContext(
        model,
        selectedGroup,
        groupRatio,
      );
      const imagePreview = buildImageTierPreviewItems(
        imageHint,
        usedGroupRatio,
        displayPrice,
        t,
      );
      if (imagePreview.items.length > 0) {
        return [
          {
            key: 'image-tier-table',
            imageTierRows: imagePreview.items,
          },
        ];
      }
    }

    const channelPrices = calculateChannelPrices(model, {
      skipSimpleVideoFlat: useTieredVideoFlat,
      skipSimpleFixed: useTieredImagePerImage,
    });

    if (!channelPrices) {
      if (isVideoModel) {
        const videoItems = buildVideoPriceCardItems({
          model,
          hint,
          videoFlat: null,
          videoFlatSuffix: null,
          original: null,
          useTieredVideoFlat,
        });
        // Video pricing must never fall back to generic token input/output rows.
        return videoItems;
      }
      return getModelPriceItems(priceData, t, siteDisplayType);
    }

    const items = [];
    const {
      input,
      output,
      cache,
      createCache,
      fixed,
      original,
      unitSuffix,
      fixedSuffix,
      videoFlat,
      videoFlatSuffix,
      asrPrice,
      asrPriceSuffix,
      quotaType,
    } = channelPrices;

    const videoItems = buildVideoPriceCardItems({
      model,
      hint,
      videoFlat,
      videoFlatSuffix,
      original,
      useTieredVideoFlat,
    });
    if (isVideoModel) {
      items.push(...videoItems);
    }

    const tokenTierInfo = detectTokenTierPricing(model);

    if (quotaType === 1 && fixed) {
      items.push({
        key: 'fixed-pricing-table',
        fixedTableRow: {
          label: t('模型价格'),
          value:
            fixed.single ||
            `${fixed.symbol}${fixed.min} ~ ${fixed.symbol}${fixed.max}`,
          suffix: fixedSuffix,
          original: original?.fixed,
        },
      });
    } else {
      const isTierBilling = quotaType === 3;
      const skipFlatInput = isTierBilling || !!tokenTierInfo?.hasModelTier;
      const skipFlatOutput =
        isTierBilling || !!tokenTierInfo?.hasCompletionTier;

      const flatTableRows = [];
      if (input && !skipFlatInput) {
        flatTableRows.push({
          key: 'input',
          label: t('输入价格'),
          value:
            input.single ||
            `${input.symbol}${input.min} ~ ${input.symbol}${input.max}`,
          suffix: unitSuffix,
          original: original?.input,
        });
      }

      if (output && !skipFlatOutput) {
        flatTableRows.push({
          key: 'output',
          label: t('输出价格'),
          value:
            output.single ||
            `${output.symbol}${output.min} ~ ${output.symbol}${output.max}`,
          suffix: unitSuffix,
          original: original?.output,
        });
      }

      if (flatTableRows.length > 0) {
        items.push({
          key: 'flat-pricing-table',
          flatTableRows,
          unitSuffix,
        });
      }
    }

    if (!isVideoModel) {
      items.push(...videoItems);
    }

    // ASR 语音识别：按秒单价（美元/秒）单独成表
    if (isASRPricingModel(model) && asrPrice) {
      items.push({
        key: 'asr-pricing-table',
        fixedTableRow: {
          label: t('语音识别'),
          value:
            asrPrice.single ||
            `${asrPrice.symbol}${asrPrice.min} ~ ${asrPrice.symbol}${asrPrice.max}`,
          suffix: asrPriceSuffix || ` / ${t('秒')}`,
          original: original?.asrPrice,
        },
      });
    }

    if (useTieredImagePerImage) {
      const { usedGroupRatio } = getUsedGroupContext(
        model,
        selectedGroup,
        groupRatio,
      );
      const usd =
        Number(imageHint.min_usd_after_channel_discount) * usedGroupRatio;
      items.push({
        key: 'image-per-image-tiered',
        label: t('按张'),
        valueNode: (
          <span className='font-bold text-black'>
            {t('最低价')}
            {displayPrice(usd)}
            {t('/张起')}
          </span>
        ),
      });
    }

    // 阶梯计费：以阶梯表展示（quota_type=3；兼容旧数据 quota_type=0 + 阶梯配置）
    // 同一模型下，多个阶梯类别（输入/输出/缓存读取/缓存写入）合并为单张表，
    // 行按"输入 Token 区间"对齐，列为各阶梯类别。
    if (tokenTierInfo && (quotaType === 0 || quotaType === 3)) {
      const { usedGroupRatio } = getUsedGroupContext(
        model,
        selectedGroup,
        groupRatio,
      );
      const tierCategoryOrder = [
        'input',
        'output',
        'cache_read',
        'cache_write',
      ];
      const perCategoryRows = {};
      const activeCategories = [];
      for (const cat of tierCategoryOrder) {
        const segmentSources = resolveTierSegmentSources({
          model,
          channel: tokenTierInfo.channel,
          cat,
        });
        const { globalSegments, channelSegments, bandSegments } =
          segmentSources;
        if (bandSegments.length === 0) continue;
        const rows = buildTokenTierPreviewItems(
          bandSegments,
          globalSegments,
          channelSegments,
          tokenTierInfo.channel,
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

      if (activeCategories.length > 0) {
        // 以"输入类别"的区间为行基准；
        // 输入未配置阶梯时回退到"输出类别"的区间。
        const baseCat = perCategoryRows.input
          ? 'input'
          : perCategoryRows.output
            ? 'output'
            : activeCategories[0];
        const baseRows = perCategoryRows[baseCat];

        const mergedRows = baseRows.map((baseRow, idx) => {
          const cells = {};
          for (const cat of activeCategories) {
            const catRows = perCategoryRows[cat];
            // 优先按 upTo 对齐；若对齐不上则回退到同 index
            let cellRow =
              catRows.find(
                (r) =>
                  Number(r.upTo) === Number(baseRow.upTo) &&
                  Number(r.fromToken) === Number(baseRow.fromToken),
              ) || catRows[idx];
            if (cellRow) {
              cells[cat] = {
                platformPrice: cellRow.platformPrice,
                platformPriceUsd: cellRow.platformPriceUsd,
                officialPrice: cellRow.officialPrice,
                officialPriceUsd: cellRow.officialPriceUsd,
                discount: cellRow.discount,
              };
            }
          }
          return {
            key: `tier-row-${baseRow.key}`,
            range: baseRow.range,
            fromToken: baseRow.fromToken,
            upTo: baseRow.upTo,
            cells,
          };
        });

        items.push({
          key: 'token-tier-table',
          tokenTierMerged: {
            columns: activeCategories.map((cat) => ({ key: cat })),
            rows: mergedRows,
            boundary: tokenTierInfo.boundary || 'lt',
          },
        });
      }
    }

    return items;
  };

  // 获取模型图标
  const getModelIcon = (model) => {
    if (!model || !model.model_name) {
      return (
        <div className={CARD_STYLES.container}>
          <Avatar size='large'>?</Avatar>
        </div>
      );
    }
    // 1) 优先使用模型自定义图标
    if (model.icon) {
      return (
        <div className={CARD_STYLES.container}>
          <div className={CARD_STYLES.icon}>
            {getLobeHubIcon(model.icon, 32)}
          </div>
        </div>
      );
    }
    // 2) 退化为供应商图标
    if (model.vendor_icon) {
      return (
        <div className={CARD_STYLES.container}>
          <div className={CARD_STYLES.icon}>
            {getLobeHubIcon(model.vendor_icon, 32)}
          </div>
        </div>
      );
    }

    // 如果没有供应商图标，使用模型名称生成头像

    const avatarText =
      (model.model_name || '').slice(0, 2).toUpperCase() || 'AI';
    return (
      <div className={CARD_STYLES.container}>
        <Avatar
          size='large'
          style={{
            width: 48,
            height: 48,
            borderRadius: 16,
            fontSize: 16,
            fontWeight: 'bold',
          }}
        >
          {avatarText}
        </Avatar>
      </div>
    );
  };

  // 获取模型描述
  const resolveModelDescription = (record) =>
    getModelCardDescription(record, i18n.language);

  const renderHomeModelCard = ({
    model,
    index,
    modelKey,
    isSelected,
    priceData,
  }) => {
    const priceItems = getModelPriceItemsForCard(model, priceData);
    const discountBadge = formatDiscountBadge(getBestDiscount(priceItems));
    const supplierType = getPrimarySupplierType(model);
    const supplierTypeLabel = supplierType
      ? getSupplierTypeLabel(supplierType, t)
      : '';
    const supplierSuffix = getModelChannelRouteSuffixes(model)[0] || '';
    const modelDescription = showModelDescription
      ? resolveModelDescription(model)
      : '';
    const routeMetaTitle = [supplierTypeLabel, supplierSuffix]
      .filter(Boolean)
      .join(' · ');
    const isHomeHot = isTopHotModel(model, hotChannelScoreMap, {
      filterSupplier,
      filterSupplierType,
    });
    const timePricingChannels = Array.isArray(model?.channel_list)
      ? model.channel_list.filter(
          (channel) => channel?.time_pricing?.has_schedules === true,
        )
      : [];
    const hasTimePricing = timePricingChannels.length > 0;
    const activeTimePricing = timePricingChannels.find(
      (channel) => channel?.time_pricing?.active === true,
    )?.time_pricing;
    const timePricingTitle = activeTimePricing
      ? `${t('分时计费')} · ${activeTimePricing.active_schedule_name || ''} · ${t('当前生效')}`
      : `${t('分时计费')} · ${t('渠道常规价')} · ${t('当前生效')}`;
    const pricingBlurStyle = blurPricing
      ? {
          filter: 'blur(6px)',
          userSelect: 'none',
          pointerEvents: 'none',
        }
      : undefined;

    const openDetail = (event) => {
      event?.stopPropagation?.();
      if (!blurPricing && openModelDetail) {
        openModelDetail(model);
      }
    };

    return (
      <Card
        key={modelKey || index}
        className={`home-model-card !rounded-[10px] transition-all duration-200 border ${blurPricing ? '' : 'cursor-pointer'} ${isSelected ? CARD_STYLES.selected : CARD_STYLES.default}`}
        bodyStyle={{ height: '100%', padding: 16 }}
        onClick={openDetail}
      >
        <div className='home-model-card-body flex h-full min-w-0 flex-col'>
          <div className='grid min-w-0 grid-cols-[48px_minmax(0,1fr)_auto] items-start gap-3'>
            {getModelIcon(model)}
            <div className='min-w-0'>
              <h3
                className='home-model-card-title m-0 truncate text-lg font-bold leading-tight'
                title={model.model_name}
              >
                {renderHighlightedText(model.model_name)}
              </h3>

              {(isHomeHot ||
                routeMetaTitle ||
                discountBadge ||
                hasTimePricing) && (
                <div className='home-model-title-meta'>
                  {routeMetaTitle ? (
                    <button
                      type='button'
                      className='home-model-route-chip'
                      disabled={!supplierSuffix}
                      title={
                        supplierSuffix
                          ? `${routeMetaTitle} | copy ${supplierSuffix}`
                          : routeMetaTitle
                      }
                      onClick={(event) => {
                        event.stopPropagation();
                        if (supplierSuffix) {
                          copyText?.(supplierSuffix);
                        }
                      }}
                    >
                      {supplierTypeLabel ? (
                        <span className='home-model-route-chip-supplier'>
                          {supplierTypeLabel}
                        </span>
                      ) : null}
                      {supplierTypeLabel && supplierSuffix ? (
                        <span className='home-model-route-chip-dot'>.</span>
                      ) : null}
                      {supplierSuffix ? (
                        <span className='home-model-route-chip-suffix'>
                          {renderHighlightedText(supplierSuffix)}
                        </span>
                      ) : null}
                    </button>
                  ) : null}
                  {discountBadge ? (
                    <Tag
                      shape='circle'
                      size='small'
                      className='home-model-discount-tag'
                    >
                      {discountBadge}
                    </Tag>
                  ) : null}
                  {hasTimePricing ? (
                    <span
                      className='home-model-time-pricing-pill'
                      title={timePricingTitle}
                    >
                      <span className='home-model-time-pricing-pill-text'>
                        {t('分时计费')}
                      </span>
                    </span>
                  ) : null}
                  {isHomeHot ? (
                    <span className='home-model-hot-pill' title={t('热门')}>
                      <span className='home-model-hot-pill-text'>
                        {t('热门')}
                      </span>
                    </span>
                  ) : null}
                </div>
              )}
            </div>

            <div className='flex items-center gap-2'>
              <Button
                size='small'
                theme='outline'
                type='tertiary'
                icon={<Copy size={12} />}
                aria-label='Copy model name'
                onClick={(event) => {
                  event.stopPropagation();
                  copyText?.(model.model_name);
                }}
              />

              {rowSelection && (
                <Checkbox
                  checked={isSelected}
                  onChange={(event) => {
                    event.stopPropagation();
                    handleCheckboxChange(model, event.target.checked);
                  }}
                />
              )}
            </div>
          </div>

          <div
            className='home-model-price-block mt-3 flex min-w-0 flex-col gap-2'
            style={pricingBlurStyle}
          >
            {priceItems.length > 0 ? (
              priceItems.map(renderPriceItem)
            ) : (
              <span className='text-xs text-semi-color-text-2'>-</span>
            )}
          </div>

          {modelDescription ? (
            <AdaptiveModelDescription style={pricingBlurStyle}>
              {renderHighlightedText(modelDescription)}
            </AdaptiveModelDescription>
          ) : null}

          <div
            className='mt-auto flex min-w-0 items-end justify-between gap-2 pt-3'
            style={pricingBlurStyle}
          >
            {renderHomeBottomTags(model)}
            <Button
              size='small'
              theme='outline'
              type='tertiary'
              className='home-model-detail-btn shrink-0'
              disabled={blurPricing}
              onClick={openDetail}
            >
              <span className='inline-flex items-center gap-1'>
                {t('\u8be6\u60c5')}
                <ChevronRight
                  size={14}
                  strokeWidth={2.25}
                  className='home-model-detail-arrow'
                  aria-hidden='true'
                />
              </span>
            </Button>
          </div>
        </div>
      </Card>
    );
  };

  // 渲染标签
  const renderTags = (record) => {
    // 计费类型标签（左边）- 使用 channel_list[0].quota_type
    const channelQuotaType =
      record.channel_list && record.channel_list.length > 0
        ? record.channel_list[0].quota_type
        : record.quota_type;

    let billingTag = (
      <Tag key='billing' shape='circle' color='white' size='small'>
        -
      </Tag>
    );
    if (channelQuotaType === 1) {
      billingTag = (
        <Tag key='billing' shape='circle' color='teal' size='small'>
          {t('按次计费')}
        </Tag>
      );
    } else if (channelQuotaType === 3) {
      // 阶梯计费：quota_type === 3
      billingTag = (
        <Tag key='tier' shape='circle' color='orange' size='small'>
          {t('阶梯计费')}
        </Tag>
      );
    } else if (channelQuotaType === 0) {
      billingTag = (
        <Tag key='billing' shape='circle' color='violet' size='small'>
          {t('按量计费')}
        </Tag>
      );
    }

    // 自定义标签（右边）
    const customTags = [];
    if (record.tags) {
      const tagArr = record.tags
        .split(',')
        .map((tag) => tag.trim())
        .filter(Boolean)
        .filter((tag) => tag !== '热门' && tag.toLowerCase() !== 'hot');
      tagArr.forEach((tg, idx) => {
        const tagText = getModelTagLabel(tg, t);
        customTags.push(
          <Tag
            key={`custom-${idx}`}
            shape='circle'
            color={stringToColor(tg)}
            size='small'
          >
            {renderHighlightedText(tagText)}
          </Tag>,
        );
      });
    }

    const channelSuffixTags = getModelChannelRouteSuffixes(record).map(
      (suffix, idx) => (
        <Tag
          key={`channel-${idx}`}
          shape='circle'
          color='blue'
          type='light'
          size='small'
        >
          {renderHighlightedText(suffix)}
        </Tag>
      ),
    );

    return (
      <div className='flex items-center justify-between'>
        <div className='flex items-center gap-2'>
          {billingTag}
          {channelSuffixTags}
        </div>
        <div className='flex items-center gap-1'>
          {customTags.length > 0 &&
            renderLimitedItems({
              items: customTags.map((tag, idx) => ({
                key: `custom-${idx}`,
                element: tag,
              })),
              renderItem: (item, idx) => item.element,
              maxDisplay: 3,
            })}
        </div>
      </div>
    );
  };

  // 显示骨架屏
  if (showSkeleton) {
    return (
      <>
        <PricingCardSkeleton
          rowSelection={!!rowSelection}
          showRatio={showRatio}
        />
      </>
    );
  }

  if (!filteredModels || filteredModels.length === 0) {
    return (
      <>
        <div className='flex justify-center items-center py-20'>
          <Empty
            image={<IllustrationNoResult style={{ width: 150, height: 150 }} />}
            darkModeImage={
              <IllustrationNoResultDark style={{ width: 150, height: 150 }} />
            }
            description={t('搜索无结果')}
          />
        </div>
      </>
    );
  }

  return (
    <>
      <div className='px-2 pt-2'>
        <div
          className={
            homeCardMode ? 'home-model-cards-grid' : 'flex flex-wrap gap-4'
          }
        >
          {paginatedModels.map((model, index) => {
            const modelKey = getModelKey(model);
            const isSelected = selectedRowKeys.includes(modelKey);

            const priceData = calculateModelPrice({
              record: model,
              selectedGroup,
              groupRatio,
              groupModelPrice,
              groupModelRatio,
              tokenUnit,
              displayPrice,
              currency,
              quotaDisplayType: siteDisplayType,
            });

            if (homeCardMode) {
              return renderHomeModelCard({
                model,
                index,
                modelKey,
                isSelected,
                priceData,
              });
            }

            const supplierLogos = getSupplierLogos(model);
            const hasChannelList =
              Array.isArray(model.channel_list) &&
              model.channel_list.length > 0;
            return (
              <Card
                key={modelKey || index}
                className={`flex-1 min-w-[350px] max-w-[600px] !rounded-2xl transition-all duration-200 hover:shadow-lg border ${blurPricing ? '' : 'cursor-pointer'} ${isSelected ? CARD_STYLES.selected : CARD_STYLES.default}`}
                bodyStyle={{ height: '100%' }}
                onClick={() =>
                  !blurPricing && openModelDetail && openModelDetail(model)
                }
              >
                <div className='flex flex-col h-full'>
                  {/* 头部：图标 + 模型名称 + 操作按钮 */}
                  <div className='flex items-start justify-between mb-3'>
                    <div className='flex items-start space-x-3 flex-1 min-w-0'>
                      {getModelIcon(model)}
                      <div className='flex-1 min-w-0'>
                        <div className='flex items-start justify-between gap-2'>
                          <h3 className='text-lg font-bold text-gray-900 truncate flex-1'>
                            {renderHighlightedText(model.model_name)}
                          </h3>
                        </div>
                        <div
                          className='flex flex-col gap-1 text-xs mt-1'
                          style={
                            blurPricing
                              ? {
                                  filter: 'blur(6px)',
                                  userSelect: 'none',
                                  pointerEvents: 'none',
                                }
                              : undefined
                          }
                        >
                          {getModelPriceItemsForCard(model, priceData).map(
                            (item) => (
                              <React.Fragment key={item.key}>
                                {/* 按量计费表格：输入/输出价格统一为标准四列表格 */}
                                {item.flatTableRows ? (
                                  <FlatPricingTable
                                    items={item.flatTableRows}
                                    unitSuffix={item.unitSuffix}
                                    t={t}
                                  />
                                ) : /* 按次计费表格 */
                                item.fixedTableRow ? (
                                  <FixedPricingTable
                                    row={item.fixedTableRow}
                                    t={t}
                                  />
                                ) : /* 视频类型计费表格 */
                                item.videoTierRows ? (
                                  <VideoPricingTable
                                    videoTierRows={item.videoTierRows}
                                    videoBillingMode={item.videoBillingMode}
                                    t={t}
                                  />
                                ) : /* 阶梯计费表格 */
                                item.tokenTierMerged ? (
                                  <TokenTierTable
                                    items={item.tokenTierMerged}
                                    t={t}
                                  />
                                ) : (
                                  /* 其他非表格价格项（视频倍率计价、按张计费等） */
                                  <div className='flex items-center'>
                                    <span className='w-20 flex-shrink-0'>
                                      {item.labelColor ? (
                                        <Tag
                                          color={item.labelColor}
                                          size='small'
                                          shape='circle'
                                          type='light'
                                          className='max-w-full'
                                        >
                                          {item.label}
                                        </Tag>
                                      ) : (
                                        item.label
                                      )}
                                    </span>
                                    <span className='flex-1 font-bold text-black inline-flex items-center flex-wrap gap-1'>
                                      {item.valueNode ? (
                                        item.valueNode
                                      ) : item.original ? (
                                        <>
                                          <span className='line-through text-gray-400 font-normal text-[10px]'>
                                            <span
                                              style={{
                                                color:
                                                  'var(--semi-color-primary)',
                                              }}
                                            >
                                              官方
                                            </span>{' '}
                                            {item.original.text}
                                          </span>
                                          <Tag
                                            size='small'
                                            shape='circle'
                                            style={{
                                              fontSize: 11,
                                              fontWeight: 700,
                                              color: '#E74C3C',
                                              backgroundColor:
                                                'rgba(231, 76, 60, 0.11)',
                                              border: 'none',
                                            }}
                                          >
                                            {formatPriceRatioFromDiscount(
                                              item.original.discount,
                                              t,
                                            )}
                                          </Tag>
                                          <span>
                                            <span
                                              style={{
                                                color:
                                                  'var(--semi-color-warning)',
                                              }}
                                            >
                                              我们
                                            </span>{' '}
                                            {item.value}
                                            {item.suffix}
                                          </span>
                                        </>
                                      ) : (
                                        <span
                                          className={
                                            item.labelColor
                                              ? 'inline-flex min-w-0 flex-wrap items-baseline gap-1'
                                              : undefined
                                          }
                                          title={item.title}
                                        >
                                          <span>{item.value}</span>
                                          {item.suffix ? (
                                            <span className='font-normal text-[10px] text-semi-color-text-2'>
                                              {item.suffix}
                                            </span>
                                          ) : null}
                                        </span>
                                      )}
                                    </span>
                                  </div>
                                )}
                              </React.Fragment>
                            ),
                          )}
                          <div className='flex items-center'>
                            <span className='w-20 flex-shrink-0'>
                              {t('供应商')}
                            </span>
                            <div className='flex-1 flex items-center flex-wrap gap-1'>
                              {supplierLogos.length === 0 ? (
                                hasChannelList ? null : (
                                  <span className='font-bold text-black'>
                                    {t('官方')}
                                  </span>
                                )
                              ) : (
                                supplierLogos.map((s) => (
                                  <div
                                    key={s.key}
                                    className='h-7 rounded-md flex items-center overflow-hidden'
                                    style={{
                                      backgroundColor:
                                        'var(--semi-color-fill-0)',
                                    }}
                                  >
                                    {s.logo ? (
                                      <img
                                        src={s.logo}
                                        alt={s.alias || s.name || ''}
                                        className='w-7 h-7 object-contain rounded-md'
                                      />
                                    ) : null}
                                    {s.supplierType && (
                                      <Tag
                                        size='small'
                                        shape='circle'
                                        color={getSupplierTypeColor(
                                          s.supplierType,
                                        )}
                                        className='mx-1'
                                      >
                                        {getSupplierTypeLabel(
                                          s.supplierType,
                                          t,
                                        )}
                                      </Tag>
                                    )}
                                  </div>
                                ))
                              )}
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>

                    <div className='flex items-center space-x-2 ml-3'>
                      {/* 复制按钮 */}
                      <Button
                        size='small'
                        theme='outline'
                        type='tertiary'
                        icon={<Copy size={12} />}
                        onClick={(e) => {
                          e.stopPropagation();
                          copyText(model.model_name);
                        }}
                      />

                      {/* 选择框 */}
                      {rowSelection && (
                        <Checkbox
                          checked={isSelected}
                          onChange={(e) => {
                            e.stopPropagation();
                            handleCheckboxChange(model, e.target.checked);
                          }}
                        />
                      )}
                    </div>
                  </div>

                  {/* 模型描述 - 占据剩余空间 */}
                  <div
                    className='flex-1 mb-4'
                    style={
                      blurPricing
                        ? {
                            filter: 'blur(6px)',
                            userSelect: 'none',
                            pointerEvents: 'none',
                          }
                        : undefined
                    }
                  >
                    <p
                      className='text-xs line-clamp-2 leading-relaxed'
                      style={{ color: 'var(--semi-color-text-2)' }}
                    >
                      {renderHighlightedText(resolveModelDescription(model))}
                    </p>
                  </div>

                  {/* 底部区域 */}
                  <div
                    className='mt-auto'
                    style={
                      blurPricing
                        ? {
                            filter: 'blur(6px)',
                            userSelect: 'none',
                            pointerEvents: 'none',
                          }
                        : undefined
                    }
                  >
                    {/* 运行性能 */}
                    <ModelPerfCardSection
                      perf={perfMetricsMap[model.model_name]}
                      t={t}
                    />

                    {/* 标签区域 */}
                    {renderTags(model)}

                    {/* 倍率信息（可选） */}
                    {showRatio && (
                      <div className='pt-3'>
                        <div className='flex items-center space-x-1 mb-2'>
                          <span className='text-xs font-medium text-gray-700'>
                            {t('倍率信息')}
                          </span>
                          <Tooltip
                            content={t('倍率是为了方便换算不同价格的模型')}
                          >
                            <IconHelpCircle
                              className='text-blue-500 cursor-pointer'
                              size='small'
                              onClick={(e) => {
                                e.stopPropagation();
                                setModalImageUrl('/ratio.png');
                                setIsModalOpenurl(true);
                              }}
                            />
                          </Tooltip>
                        </div>
                        <div className='grid grid-cols-3 gap-2 text-xs text-gray-600'>
                          <div>
                            {t('模型')}:{' '}
                            {model.quota_type === 0
                              ? (priceData?.inputRatio ?? model.model_ratio)
                              : t('无')}
                          </div>
                          <div>
                            {t('输出')}:{' '}
                            {model.quota_type === 0
                              ? parseFloat(model.completion_ratio.toFixed(2))
                              : t('无')}
                          </div>
                          <div>
                            {t('分组')}: {priceData?.usedGroupRatio ?? '-'}
                          </div>
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              </Card>
            );
          })}
        </div>

        {/* 分页 */}
        {filteredModels.length > 0 && (
          <div className='flex justify-center mt-6 py-4 border-t pricing-pagination-divider'>
            <Pagination
              currentPage={currentPage}
              pageSize={pageSize}
              total={filteredModels.length}
              showSizeChanger={showSizeChanger}
              pageSizeOptions={[10, 20, 50, 100]}
              size={isMobile ? 'small' : 'default'}
              showQuickJumper={isMobile}
              onPageChange={(page) => setCurrentPage(page)}
              onPageSizeChange={(size) => {
                setPageSize(size);
                setCurrentPage(1);
              }}
            />
          </div>
        )}
      </div>
    </>
  );
};

export default React.memo(PricingCardView);
