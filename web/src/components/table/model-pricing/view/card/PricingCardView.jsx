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
import { Copy } from 'lucide-react';
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
  compareVideoResolutionAsc,
  isVideoPricingModel,
  hasNumericValue,
  getModelTagLabel,
  getSupplierTypeLabel,
  getModelDescription,
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
import PricingCardSkeleton from './PricingCardSkeleton';
import ModelPerfCardSection from '../../components/ModelPerfCardSection';
import { useMinimumLoadingTime } from '../../../../../hooks/common/useMinimumLoadingTime';
import { renderLimitedItems } from '../../../../common/ui/RenderUtils';
import { useIsMobile } from '../../../../../hooks/common/useIsMobile';
const CARD_STYLES = {
  container:
    'w-12 h-12 rounded-xl flex items-center justify-center relative shadow-sm border border-semi-color-border bg-white',
  icon: 'w-8 h-8 flex items-center justify-center',
  selected: 'border-blue-500 bg-blue-50 shadow-md',
  default: 'border-gray-200 hover:border-blue-200 hover:shadow-md',
};

const escapeRegExp = (value) => value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');

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

const formatVideoTierSpec = (row, t, billingMode) => {
  const res = formatCompactVideoResolution(row?.resolution, t);
  const lane = String(row?.lane || '');
  if (billingMode === 'per_second' || lane.includes('per_second')) {
    return `${res}/${t('秒')}`;
  }
  if (billingMode === 'per_token' || lane.includes('per_token')) {
    return `${res}/M token`;
  }
  return `${res}/${t('个')}`;
};

const formatVideoTierDisplayPrice = (
  usd,
  usedGroupRatio,
  displayPrice,
) => displayPrice(Number(usd || 0) * usedGroupRatio);

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

const VIDEO_CARD_TIER_PREVIEW_LIMIT = 2;

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
      spec: formatVideoTierSpec(row, t, billingMode),
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

/**
 * 渲染折扣单元格
 * 折扣视觉效果：Tag 胶囊标签样式，颜色统一 #E74C3C
 */
const renderDiscountCell = (discount) => {
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
        -{discount}%
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
    ? (firstRow.fromToken === 0 && firstRow.upTo > 0
        ? `< ${formatTierBound(firstRow.upTo)}`
        : firstRow.range)
    : '';

  // 边界隐藏逻辑：所有行都没有有效折扣（discount <= 0 或 null）时，隐藏官方价和折扣列
  const hideOfficialCols = displayRows.every(
    ({ cell }) => cell.discount == null || cell.discount <= 0
  );

  return (
    <div
      className='w-full min-w-0 overflow-hidden rounded-lg border'
      style={PRICING_TABLE_WRAPPER_STYLE}
    >
      <table
        className='w-full border-collapse'
        style={{ fontSize: 11 }}
      >
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
                    {renderDiscountCell(cell.discount)}
                  </td>
                )}
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
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

  return (
    <div
      className='w-full min-w-0 overflow-hidden rounded-lg border'
      style={PRICING_TABLE_WRAPPER_STYLE}
    >
      <table
        className='w-full border-collapse'
        style={{ fontSize: 11 }}
      >
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
                  {row.hasOriginal
                    ? renderOfficialCell(row.officialValue, row.discount > 0)
                    : <span style={{ color: 'var(--semi-color-text-3)' }}>-</span>
                  }
                </td>
              )}
              {!hideOfficialCols && (
                <td className={TABLE_CELL_CLASS.tdDiscount}>
                  {renderDiscountCell(row.discount)}
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
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

  return (
    <div
      className='w-full min-w-0 overflow-hidden rounded-lg border'
      style={PRICING_TABLE_WRAPPER_STYLE}
    >
      <table
        className='w-full border-collapse'
        style={{ fontSize: 11 }}
      >
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
                {hasOriginal
                  ? renderOfficialCell(officialValue, discount > 0)
                  : <span style={{ color: 'var(--semi-color-text-3)' }}>-</span>
                }
              </td>
            )}
            {!hideOfficialCols && (
              <td className={TABLE_CELL_CLASS.tdDiscount}>
                {renderDiscountCell(discount)}
              </td>
            )}
          </tr>
        </tbody>
      </table>
    </div>
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
const VideoPricingTable = ({
  videoTierRows,
  videoBillingMode,
  t,
}) => {
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
    (row) => row.discount == null || row.discount <= 0
  );

  return (
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
            <tr style={{ backgroundColor: getVideoTierGroupStyle(group.family).backgroundColor }}>
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
                      backgroundColor: getVideoTierGroupStyle(group.family).borderColor,
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
                  style={{ color: getVideoTierGroupStyle(group.family).color }}
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
                    {renderDiscountCell(row.discount)}
                  </td>
                )}
              </tr>
            ))}
          </tbody>
        </table>
      ))}
    </div>
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
  searchValue = '',
  channelVideoRatio = {},
  channelVideoCompletionRatio = {},
  channelVideoPrice = {},
  perfMetricsMap = {},
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
      const rawDisplayPrice = displayPrice(priceUSD);
      const unitDivisor = tokenUnit === 'K' ? 1000 : 1;
      const numericPrice =
        parseFloat(rawDisplayPrice.replace(/[^0-9.]/g, '')) / unitDivisor;

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
        value: parseFloat(numericPrice.toFixed(2)),
        rawUsd: Number(priceUSD) || 0,
        symbol,
      };
    };

    const modelHasVideoFlatPrice = hasNumericValue(model.video_price);
    const hideTextTokenPrices = isVideoPricingModel(model);

    // 提取所有通道的价格（与 relay 一致：ch.model_ratio 已含渠道折扣；再乘分组倍率）
    const prices = {
      input: [],
      output: [],
      cache: [],
      createCache: [],
      fixed: [],
      videoFlat: [],
    };
    const originalPrices = {
      input: [],
      output: [],
      cache: [],
      createCache: [],
      fixed: [],
      videoFlat: [],
    };

    model.channel_list.forEach((ch) => {
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
      },
      videoFlat: calculateRange(prices.videoFlat),
      unitSuffix,
      fixedSuffix,
      videoFlatSuffix: ` / ${t('条')}`,
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
        if (videoItems.length > 0) return videoItems;
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
      const skipFlatInput =
        isTierBilling || !!tokenTierInfo?.hasModelTier;
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
      const tierCategoryOrder = ['input', 'output', 'cache_read', 'cache_write'];
      const perCategoryRows = {};
      const activeCategories = [];
      for (const cat of tierCategoryOrder) {
        const { globalSegments, channelSegments, bandSegments } =
          resolveTierSegmentSources({
            model,
            channel: tokenTierInfo.channel,
            cat,
          });
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
    getModelDescription(record, i18n.language);

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
      const tagArr = record.tags.split(',').filter(Boolean);
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

    return (
      <div className='flex items-center justify-between'>
        <div className='flex items-center gap-2'>
          {billingTag}
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
        <div className='flex flex-wrap gap-4'>
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
                                              backgroundColor: 'rgba(231, 76, 60, 0.11)',
                                              border: 'none',
                                            }}
                                          >
                                            -{item.original.discount}%
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
                                        {getSupplierTypeLabel(s.supplierType, t)}
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

export default PricingCardView;
