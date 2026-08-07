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

import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { Modal, Tag } from '@douyinfe/semi-ui';
import { formatVideoResolutionDisplayLabel, resolveVideoBillingResolutionLabel } from '../../helpers/videoResolutionLabel';
import {
  API,
  getLast7DaysStartTimestamp,
  getLast7DaysEndTimestamp,
  isAdmin,
  isSupplier,
  showError,
  showSuccess,
  timestamp2string,
  toUnixTimestamp,
  renderQuota,
  renderNumber,
  getQuotaPerUnit,
  getCurrencyConfig,
  getLogOther,
  copy,
  renderClaudeLogContent,
  renderLogContent,
  renderConsumeBillingProcess,
  trimDecimalsInLogDetailText,
  ceilToFixedDecimals,
  formatCeilFixedDecimals,
  formatASRSecondsDisplay,
  formatASRUserPerSecondPrice,
} from '../../helpers';
import { ITEMS_PER_PAGE } from '../../constants';
import { useTableCompactMode } from '../common/useTableCompactMode';
import ParamOverrideEntry from '../../components/table/usage-logs/components/ParamOverrideEntry';

/** 将表单 logType 规范为有效类型 id 数组（空数组表示全部类型）。 */
const normalizeLogTypes = (logType) => {
  if (logType == null || logType === '' || logType === '0') {
    return [];
  }
  const raw = Array.isArray(logType) ? logType : [logType];
  const types = raw
    .map((v) => parseInt(String(v), 10))
    .filter((n) => !Number.isNaN(n) && n > 0);
  return [...new Set(types)];
};

/** 构建 API type 查询参数（空为 0，多选为逗号分隔）。 */
const buildLogTypeQueryParam = (logType) => {
  const types = normalizeLogTypes(logType);
  if (!types.length) {
    return '0';
  }
  return types.join(',');
};

export const useLogsData = () => {
  const { t } = useTranslation();

  // Define column keys for selection
  const COLUMN_KEYS = {
    TIME: 'time',
    CHANNEL: 'channel',
    USERNAME: 'username',
    TOKEN: 'token',
    GROUP: 'group',
    TYPE: 'type',
    MODEL: 'model',
    USE_TIME: 'use_time',
    PROMPT: 'prompt',
    COMPLETION: 'completion',
    COST: 'cost',
    RETRY: 'retry',
    IP: 'ip',
    DETAILS: 'details',
  };

  // Basic state
  const [logs, setLogs] = useState([]);
  const [expandData, setExpandData] = useState({});
  const [showStat, setShowStat] = useState(false);
  const [loading, setLoading] = useState(false);
  const [loadingStat, setLoadingStat] = useState(false);
  const [activePage, setActivePage] = useState(1);
  const [logCount, setLogCount] = useState(0);
  const [pageSize, setPageSize] = useState(ITEMS_PER_PAGE);
  const [logTypes, setLogTypes] = useState([]);
  const [modelOptions, setModelOptions] = useState([]);

  // User and admin
  const isAdminUser = isAdmin();
  /** 供应商查看使用日志时与数据看板一致：按自有 channel_id 统计，不限 user_id。 */
  const supplierChannelLogsView = isSupplier() && !isAdminUser;
  // Role-specific storage key to prevent different roles from overwriting each other
  const STORAGE_KEY = isAdminUser
    ? 'logs-table-columns-admin'
    : 'logs-table-columns-user';
  const BILLING_DISPLAY_MODE_STORAGE_KEY = isAdminUser
    ? 'logs-billing-display-mode-admin'
    : 'logs-billing-display-mode-user';

  const hasVideoPerSecondDetail = (other) =>
    other?.billing_mode === 'video_per_second' &&
    Number(other?.video_seconds || 0) > 0 &&
    Number(other?.video_price_per_second || 0) > 0 &&
    Number(other?.video_quota_per_unit || 0) > 0;

  const hasVideoPerTokenDetail = (other) =>
    other?.billing_mode === 'video_token_output' &&
    Number(other?.video_total_tokens || other?.video_output_tokens || 0) > 0 &&
    Number(other?.video_token_unit_price || 0) > 0 &&
    Number(other?.video_quota_per_unit || 0) > 0;

  const hasVideoPerVideoDetail = (other) =>
    other?.billing_mode === 'video_per_video';

  /** 规格展示：优先 video_resolution；用户输入分辨率原样展示，否则归一化推断值。 */
  const getVideoSpecResolutionLabel = (other, specWidth, specHeight) => {
    const fromInput = other?.video_resolution_from_input === true;
    const upstream = String(other?.video_resolution || '').trim();
    if (upstream) {
      return resolveVideoBillingResolutionLabel(upstream, fromInput);
    }
    if (specWidth > 0 && specHeight > 0) {
      const fromPixels = formatVideoResolutionDisplayLabel(
        `${specWidth}x${specHeight}`,
      );
      return fromPixels || `${specWidth}×${specHeight}`;
    }
    return '';
  };

  const isZeroBilledVideoNoChargeLog = (record, other) => {
    if (record?.type !== 2) {
      return false;
    }

    const quota = Number(record?.quota ?? 0);
    const promptTokens = Number(record?.prompt_tokens ?? 0);
    const completionTokens = Number(record?.completion_tokens ?? 0);
    if (quota !== 0 || promptTokens !== 0 || completionTokens !== 0) {
      return false;
    }

    if (hasVideoPerSecondDetail(other) || hasVideoPerTokenDetail(other) || hasVideoPerVideoDetail(other)) {
      return false;
    }

    const text = [
      record?.content,
      record?.model_name,
      other?.request_path,
      other?.billing_mode,
    ]
      .filter(Boolean)
      .join(' ')
      .toLowerCase();

    return (
      text.includes('视频') ||
      text.includes('video') ||
      text.includes('/videos')
    );
  };

  /**
   * 异步视频任务当前所处扣费阶段（同一记录只展示其一）：
   * - 已结算：仅「实际扣费」
   * - 仅预扣：仅「预扣费」
   */
  const getVideoBillingPhase = (other, quota = 0) => {
    const finalRaw = other?.video_final_quota ?? other?.actual_quota;
    if (finalRaw !== undefined && finalRaw !== null) {
      const final = Number(finalRaw);
      if (Number.isFinite(final) && final > 0) {
        return { phase: 'actual', label: t('实际扣费'), amount: final };
      }
    }
    const isVideoTask =
      other?.billing_mode === 'video_per_second' ||
      other?.billing_mode === 'video_token_output' ||
      other?.billing_mode === 'video_per_video' ||
      Boolean(other?.task_id) ||
      String(other?.request_path || '').includes('/videos');
    if (!isVideoTask) {
      return null;
    }
    const pre = Number(
      other?.video_billed_quota ??
        other?.video_pre_consumed_quota ??
        quota ??
        0,
    );
    if (Number.isFinite(pre) && pre > 0) {
      return { phase: 'pre', label: t('预扣费'), amount: pre };
    }
    return null;
  };

  /** 结算展示用 quota：与「实际扣费/预扣费」同一口径，避免浮点公式与整数 round 不一致 */
  const getVideoSettlementQuota = (other, quota = 0) => {
    const phase = getVideoBillingPhase(other, quota);
    if (phase) {
      return phase.amount;
    }
    const billed = Number(other?.video_billed_quota || quota || 0);
    return Number.isFinite(billed) && billed > 0 ? billed : 0;
  };

  const formatVideoDisplayNumber = (value, digits = 6) => {
    const numberValue = Number(value || 0);
    if (!Number.isFinite(numberValue)) {
      return '0';
    }
    // 进一法最多 6 位，去掉末尾 0（与花费列一致）
    return formatCeilFixedDecimals(numberValue, digits);
  };

  const getVideoQuotaPerUnit = (other) => {
    const logQuotaPerUnit = Number(other?.video_quota_per_unit || 0);
    if (Number.isFinite(logQuotaPerUnit) && logQuotaPerUnit > 0) {
      return logQuotaPerUnit;
    }
    const currentQuotaPerUnit = Number(getQuotaPerUnit() || 0);
    return Number.isFinite(currentQuotaPerUnit) && currentQuotaPerUnit > 0
      ? currentQuotaPerUnit
      : 500000;
  };

  const formatVideoUsdAmount = (other, usdAmount, digits = 6) => {
    const numberValue = Number(usdAmount || 0);
    if (!Number.isFinite(numberValue)) {
      return '$0';
    }

    const { symbol, rate, type } = getCurrencyConfig();
    if (type === 'TOKENS') {
      return `${formatVideoDisplayNumber(
        numberValue * getVideoQuotaPerUnit(other),
        2,
      )} tokens`;
    }

    return `${symbol}${formatVideoDisplayNumber(
      numberValue * (rate || 1),
      digits,
    )}`;
  };

  const renderVideoQuota = (other, quota, digits = 6) => {
    const q = Number(quota || 0);
    if (!Number.isFinite(q)) {
      return renderQuota(0, digits);
    }

    const { symbol, rate, type } = getCurrencyConfig();
    if (type === 'TOKENS') {
      return formatVideoDisplayNumber(q, 2);
    }

    const displayValue = (q / getVideoQuotaPerUnit(other)) * (rate || 1);
    // 与花费列一致：6 位进一法，展示去尾零
    const fixedResult = ceilToFixedDecimals(displayValue, digits);
    if (fixedResult === 0 && q > 0 && displayValue > 0) {
      return `${symbol}${formatCeilFixedDecimals(
        Math.pow(10, -digits),
        digits,
      )}`;
    }
    return `${symbol}${formatCeilFixedDecimals(fixedResult, digits)}`;
  };

  const buildVideoCostDisplayItems = (
    other,
    billedQuota,
    tagValue,
    inlineTags,
    quotaDigits = 2,
  ) => {
    const phase = getVideoBillingPhase(other, billedQuota);
    if (!phase) {
      return [
        [
          t('花费'),
          inlineTags(
            tagValue(renderVideoQuota(other, billedQuota, quotaDigits), 'red', 'cost'),
          ),
        ],
      ];
    }
    return [
      [
        phase.label,
        inlineTags(
          tagValue(
            renderVideoQuota(other, phase.amount, quotaDigits),
            phase.phase === 'actual' ? 'red' : 'grey',
            'cost',
          ),
        ),
      ],
    ];
  };

  const renderVideoPerSecondBillingDetail = (log, other, quota) => {
    const seconds = Number(other?.video_seconds || 0);
    const pricePerSecond = Number(other?.video_price_per_second || 0);
    const groupRatio = Number(other?.group_ratio || 1);
    const channelDiscount = Number(other?.channel_price_discount ?? 100);
    const billedQuota = Number(other?.video_billed_quota || quota || 0);
    const width = Number(other?.video_width || 0);
    const height = Number(other?.video_height || 0);
    const ruleWidth = Number(other?.video_rule_width || 0);
    const ruleHeight = Number(other?.video_rule_height || 0);
    const hasAudio = other?.video_has_audio === true;
    const unifiedAudio = other?.video_unified_audio_price === true;
    const audioText = hasAudio ? t('有音频') : t('无音频');
    const priceLabel = unifiedAudio
      ? t('每秒价')
      : hasAudio
        ? t('有音轨价')
        : t('无音轨价');
    const modelName = log?.model_name || '-';
    const upstreamModelName = other?.upstream_model_name || '';
    const resolution = other?.video_resolution || '';
    const ratioLabel =
      other?.video_ratio_label || other?.video_aspect_ratio || '';
    const specWidth = ruleWidth || width;
    const specHeight = ruleHeight || height;
    const specResolutionLabel = getVideoSpecResolutionLabel(
      other,
      specWidth,
      specHeight,
    );
    const effectivePerSecond = Number(
      other?.effective_video_price_per_second || 0,
    );
    const calculatedPricePerSecond =
      effectivePerSecond > 0
        ? effectivePerSecond * groupRatio
        : pricePerSecond * groupRatio * (channelDiscount / 100);
    // 结算算式右侧必须由左侧运算元推出（秒 × 折扣后单价），不得直接贴 actual_quota，
    // 否则会出现「4×折扣价=原价总额」的假等式；实际扣费仍走 settlementQuota。
    const formulaTotalUsd =
      seconds > 0 && calculatedPricePerSecond > 0
        ? seconds * calculatedPricePerSecond
        : 0;
    const settlementQuota = getVideoSettlementQuota(other, billedQuota);
    const tagValue = (value, color = 'blue', key = String(value)) => (
      <Tag key={key} color={color} size='small'>
        {value}
      </Tag>
    );
    const inlineTags = (...nodes) => (
      <span className='flex flex-wrap items-center gap-1'>{nodes}</span>
    );
    const modelValue = inlineTags(
      tagValue(modelName, 'blue', 'model'),
      isAdminUser && upstreamModelName
        ? tagValue(
            t('实际运行：{{upstreamModel}}', {
              upstreamModel: upstreamModelName,
            }),
            'purple',
            'upstream-model',
          )
        : null,
    );
    const specValue = inlineTags(
      tagValue(specResolutionLabel, 'cyan', 'spec-resolution'),
      ratioLabel ? tagValue(ratioLabel, 'purple', 'spec-ratio') : null,
      tagValue(audioText, hasAudio ? 'green' : 'grey', 'spec-audio'),
      tagValue(t('{{seconds}} 秒', { seconds }), 'orange', 'spec-seconds'),
    );
    const calculatedPriceValue = inlineTags(
      tagValue(
        `${formatVideoUsdAmount(other, calculatedPricePerSecond)} / ${t('秒')}`,
        'green',
        'calculated-price',
      ),
      tagValue(priceLabel, 'grey', 'price-label'),
    );
    const calculationValue = inlineTags(
      tagValue(t('{{seconds}} 秒', { seconds }), 'orange', 'calc-seconds'),
      <span key='multiply-1' className='mx-1 text-gray-400'>
        ×
      </span>,
      tagValue(
        `${formatVideoUsdAmount(other, calculatedPricePerSecond)} / ${t('秒')}`,
        'green',
        'calc-price',
      ),
      <span key='equals' className='mx-1 text-gray-400'>
        =
      </span>,
      tagValue(
        formulaTotalUsd > 0
          ? formatVideoUsdAmount(other, formulaTotalUsd)
          : renderVideoQuota(other, settlementQuota, 6),
        'red',
        'calc-total',
      ),
    );
    const modeValue = inlineTags(
      tagValue(t('分辨率阶梯计费'), 'blue', 'billing-mode'),
      other?.video_capped_to_max_tier === true
        ? tagValue(t('最高档封顶'), 'orange', 'max-tier-cap')
        : null,
    );
    const actualVideoValue = inlineTags(
      tagValue(`${width}×${height}`, 'cyan', 'actual-resolution'),
      tagValue(audioText, hasAudio ? 'green' : 'grey', 'actual-audio'),
    );
    const costItems = buildVideoCostDisplayItems(
      other,
      billedQuota,
      tagValue,
      inlineTags,
    );

    const items = [
      [t('模型'), modelValue],
      [t('规格'), specValue],
      [t('计费单价'), calculatedPriceValue],
      [t('结算计算'), calculationValue],
      ...costItems,
      [t('计费模式'), modeValue],
    ];
    if (
      width > 0 &&
      height > 0 &&
      (width !== specWidth || height !== specHeight)
    ) {
      items.splice(2, 0, [t('实际视频'), actualVideoValue]);
    }

    return (
      <div className='max-w-[720px] space-y-1.5 text-sm leading-6'>
        {items.map(([label, value]) => (
          <div
            key={label}
            className='grid grid-cols-[88px_minmax(0,1fr)] gap-3'
          >
            <span className='text-gray-500'>{label}</span>
            <span className='break-words text-gray-900'>{value}</span>
          </div>
        ))}
      </div>
    );
  };

  const renderVideoPerSecondBillingBrief = (other, quota) => {
    const billedQuota = Number(other?.video_billed_quota || quota || 0);
    const seconds = Number(other?.video_seconds || 0);
    const resolution = other?.video_resolution || '-';
    const phase = getVideoBillingPhase(other, quota);

    return (
      <div className='flex flex-wrap items-center gap-2'>
        <span className='rounded-full bg-blue-600 px-2 py-0.5 text-xs font-medium text-white'>
          {t('分辨率阶梯计费')}
        </span>
        <span className='text-sm text-gray-700'>
          {phase
            ? t('{{seconds}}秒 · {{resolution}} · {{label}} {{cost}}', {
                seconds,
                resolution,
                label: phase.label,
                cost: renderVideoQuota(other, phase.amount),
              })
            : t('{{seconds}}秒 · {{resolution}} · {{cost}}', {
                seconds,
                resolution,
                cost: renderVideoQuota(other, billedQuota),
              })}
        </span>
      </div>
    );
  };

  const renderASRBillingBrief = (other) => {
    const seconds = Number(other?.audio_seconds || 0);
    const { symbol, price } = formatASRUserPerSecondPrice(other);
    return (
      <div className='flex flex-wrap items-center gap-2'>
        <span className='rounded-full bg-teal-600 px-2 py-0.5 text-xs font-medium text-white'>
          {t('语音识别按秒计费')}
        </span>
        <span className='text-sm text-gray-700'>
          {t('{{seconds}}秒 · {{symbol}}{{price}}/秒', {
            seconds: formatASRSecondsDisplay(seconds),
            symbol,
            price,
          })}
        </span>
      </div>
    );
  };

  const renderVideoPerTokenBillingBrief = (other, quota) => {
    const billedQuota = Number(other?.video_billed_quota || quota || 0);
    const totalTokens = Number(
      other?.video_total_tokens || other?.video_output_tokens || 0,
    );
    const resolution = other?.video_resolution || '-';
    const phase = getVideoBillingPhase(other, quota);

    return (
      <div className='flex flex-wrap items-center gap-2'>
        <span className='rounded-full bg-blue-600 px-2 py-0.5 text-xs font-medium text-white'>
          {t('视频按 token 计费')}
        </span>
        <span className='text-sm text-gray-700'>
          {phase
            ? t('{{tokens}} tokens · {{resolution}} · {{label}} {{cost}}', {
                tokens: totalTokens,
                resolution,
                label: phase.label,
                cost: renderVideoQuota(other, phase.amount, 6),
              })
            : t('{{tokens}} tokens · {{resolution}} · {{cost}}', {
                tokens: totalTokens,
                resolution,
                cost: renderVideoQuota(other, billedQuota, 6),
              })}
        </span>
      </div>
    );
  };

  const renderVideoPerTokenBillingDetail = (log, other, quota) => {
    const totalTokens = Number(
      other?.video_total_tokens || other?.video_output_tokens || 0,
    );
    const pricePerMillion = Number(other?.video_token_unit_price || 0);
    const groupRatio = Number(other?.group_ratio || 1);
    const channelDiscount = Number(other?.channel_price_discount ?? 100);
    const billedQuota = Number(other?.video_billed_quota || quota || 0);
    const width = Number(other?.video_width || 0);
    const height = Number(other?.video_height || 0);
    const ruleWidth = Number(other?.video_rule_width || 0);
    const ruleHeight = Number(other?.video_rule_height || 0);
    const hasAudio = other?.video_has_audio === true;
    const audioText = hasAudio ? t('有音频') : t('无音频');
    const modelName = log?.model_name || '-';
    const upstreamModelName = other?.upstream_model_name || '';
    const resolution = other?.video_resolution || '';
    const ratioLabel =
      other?.video_ratio_label || other?.video_aspect_ratio || '';
    const specWidth = ruleWidth || width;
    const specHeight = ruleHeight || height;
    const effectivePerMillion = Number(
      other?.effective_video_token_unit_price || 0,
    );
    const calculatedPricePerMillion =
      effectivePerMillion > 0
        ? effectivePerMillion * groupRatio
        : pricePerMillion * groupRatio * (channelDiscount / 100);
    const settlementQuota = getVideoSettlementQuota(other, billedQuota);
    const unitPriceText = `${formatVideoUsdAmount(other, calculatedPricePerMillion)}/ 1M tokens`;
    const tagValue = (value, color = 'blue', key = String(value)) => (
      <Tag key={key} color={color} size='small'>
        {value}
      </Tag>
    );
    const inlineTags = (...nodes) => (
      <span className='flex flex-wrap items-center gap-1'>{nodes}</span>
    );
    const modelValue = inlineTags(
      tagValue(modelName, 'blue', 'model'),
      isAdminUser && upstreamModelName
        ? tagValue(
            t('实际运行：{{upstreamModel}}', {
              upstreamModel: upstreamModelName,
            }),
            'purple',
            'upstream-model',
          )
        : null,
    );
    const specValue = inlineTags(
      tagValue(
        getVideoSpecResolutionLabel(other, specWidth, specHeight) ||
          `${specWidth}×${specHeight}`,
        'cyan',
        'spec-resolution',
      ),
      ratioLabel
        ? tagValue(ratioLabel, 'purple', 'spec-ratio')
        : null,
      tagValue(audioText, hasAudio ? 'green' : 'grey', 'spec-audio'),
      tagValue(t('{{count}} tokens', { count: totalTokens }), 'orange', 'spec-tokens'),
    );
    const calculatedPriceValue = inlineTags(
      tagValue(unitPriceText, 'green', 'calculated-price'),
    );
    const calculationValue = inlineTags(
      tagValue(`${totalTokens} tokens`, 'orange', 'calc-tokens'),
      <span key='multiply-1' className='mx-1 text-gray-400'>
        ×
      </span>,
      tagValue(unitPriceText, 'green', 'calc-price'),
      <span key='equals' className='mx-1 text-gray-400'>
        =
      </span>,
      tagValue(
        renderVideoQuota(other, settlementQuota, 6),
        'red',
        'calc-total',
      ),
    );
    const modeValue = inlineTags(
      tagValue(t('视频按 token 计费'), 'blue', 'billing-mode'),
      other?.video_capped_to_max_tier === true
        ? tagValue(t('最高档封顶'), 'orange', 'max-tier-cap')
        : null,
    );
    const actualVideoValue = inlineTags(
      tagValue(`${width}×${height}`, 'cyan', 'actual-resolution'),
      tagValue(audioText, hasAudio ? 'green' : 'grey', 'actual-audio'),
    );
    const costItems = buildVideoCostDisplayItems(
      other,
      billedQuota,
      tagValue,
      inlineTags,
      6,
    );

    const items = [
      [t('模型'), modelValue],
      [t('规格'), specValue],
      [t('计费单价'), calculatedPriceValue],
      [t('结算计算'), calculationValue],
      ...costItems,
      [t('计费模式'), modeValue],
    ];
    if (
      width > 0 &&
      height > 0 &&
      (width !== specWidth || height !== specHeight)
    ) {
      items.splice(2, 0, [t('实际视频'), actualVideoValue]);
    }

    return (
      <div className='max-w-[720px] space-y-1.5 text-sm leading-6'>
        {items.map(([label, value]) => (
          <div
            key={label}
            className='grid grid-cols-[88px_minmax(0,1fr)] gap-3'
          >
            <span className='text-gray-500'>{label}</span>
            <span className='break-words text-gray-900'>{value}</span>
          </div>
        ))}
      </div>
    );
  };

  const renderVideoPerVideoBillingDetail = (log, other, quota) => {
    const count = Number(other?.video_count || 1);
    const billedQuota = Number(other?.video_billed_quota || quota || 0);
    const quotaPerUnit = getVideoQuotaPerUnit(other);
    const pricePerVideo =
      Number(other?.video_price_per_video || 0) ||
      (quotaPerUnit > 0 && count > 0 ? billedQuota / quotaPerUnit / count : 0);
    const width = Number(other?.video_width || 0);
    const height = Number(other?.video_height || 0);
    const ruleWidth = Number(other?.video_rule_width || 0);
    const ruleHeight = Number(other?.video_rule_height || 0);
    const resolution = other?.video_resolution || '';
    const hasAudio = other?.video_has_audio === true;
    const seconds = Number(other?.video_seconds || 0);
    const modelName = log?.model_name || '-';
    const upstreamModelName = other?.upstream_model_name || '';
    const audioText = hasAudio ? t('有音频') : t('无音频');
    const priceLabel = hasAudio ? t('有音轨价') : t('无音轨价');
    const totalPrice = count * pricePerVideo;
    const tagValue = (value, color = 'blue', key = String(value)) => (
      <Tag key={key} color={color} size='small'>
        {value}
      </Tag>
    );
    const inlineTags = (...nodes) => (
      <span className='flex flex-wrap items-center gap-1'>{nodes}</span>
    );
    const modelValue = inlineTags(
      tagValue(modelName, 'blue', 'model'),
      isAdminUser && upstreamModelName
        ? tagValue(
            t('实际运行：{{upstreamModel}}', {
              upstreamModel: upstreamModelName,
            }),
            'purple',
            'upstream-model',
          )
        : null,
    );
    const specTags = [];
    const specResolutionLabel = getVideoSpecResolutionLabel(
      other,
      ruleWidth || width,
      ruleHeight || height,
    );
    if (specResolutionLabel || resolution) {
      specTags.push(
        tagValue(
          specResolutionLabel || resolution,
          'cyan',
          'matched-resolution',
        ),
      );
    }
    if (seconds > 0) {
      specTags.push(
        tagValue(t('{{seconds}} 秒', { seconds }), 'orange', 'seconds'),
      );
    }
    specTags.push(tagValue(audioText, hasAudio ? 'green' : 'grey', 'audio'));
    const specificationValue = inlineTags(...specTags);
    const countValue = inlineTags(
      tagValue(t('{{count}} 条', { count }), 'orange', 'count'),
    );
    const priceValue = inlineTags(
      tagValue(
        `${formatVideoUsdAmount(other, pricePerVideo)} / ${t('条')}`,
        'green',
        'price',
      ),
      tagValue(priceLabel, 'grey', 'price-label'),
    );
    const calculationValue = inlineTags(
      tagValue(t('{{count}} 条', { count }), 'orange', 'calc-count'),
      <span key='multiply' className='mx-1 text-gray-400'>
        ×
      </span>,
      tagValue(
        `${formatVideoUsdAmount(other, pricePerVideo)} / ${t('条')}`,
        'green',
        'calc-price',
      ),
      <span key='equals' className='mx-1 text-gray-400'>
        =
      </span>,
      tagValue(formatVideoUsdAmount(other, totalPrice), 'red', 'calc-total'),
    );
    const costItems = buildVideoCostDisplayItems(
      other,
      billedQuota,
      tagValue,
      inlineTags,
    );

    const items = [
      [t('模型'), modelValue],
      [t('规格'), specificationValue],
      [t('视频数量'), countValue],
      [t('计费单价'), priceValue],
      [t('结算计算'), calculationValue],
      ...costItems,
      [
        t('计费模式'),
        inlineTags(tagValue(t('按视频数量计费'), 'blue', 'billing-mode')),
      ],
    ];

    return (
      <div className='max-w-[720px] space-y-1.5 text-sm leading-6'>
        {items.map(([label, value]) => (
          <div
            key={label}
            className='grid grid-cols-[88px_minmax(0,1fr)] gap-3'
          >
            <span className='text-gray-500'>{label}</span>
            <span className='break-words text-gray-900'>{value}</span>
          </div>
        ))}
      </div>
    );
  };

  const renderVideoPerVideoBillingBrief = (other, quota) => {
    const billedQuota = Number(other?.video_billed_quota || quota || 0);
    const count = Number(other?.video_count || 1);
    const phase = getVideoBillingPhase(other, quota);

    return (
      <div className='flex flex-wrap items-center gap-2'>
        <span className='rounded-full bg-blue-600 px-2 py-0.5 text-xs font-medium text-white'>
          {t('按视频数量计费')}
        </span>
        <span className='text-sm text-gray-700'>
          {phase
            ? t('{{count}}条 · {{label}} {{cost}}', {
                count,
                label: phase.label,
                cost: renderVideoQuota(other, phase.amount),
              })
            : t('{{count}}条 · {{cost}}', {
                count,
                cost: renderVideoQuota(other, billedQuota),
              })}
        </span>
      </div>
    );
  };

  // Statistics state
  const [stat, setStat] = useState({
    quota: 0,
    rpm: 0,
    tpm: 0,
  });

  // Form state
  const [formApi, setFormApi] = useState(null);
  const formInitValues = {
    username: '',
    token_name: '',
    model_name: '',
    channel: '',
    group: '',
    request_id: '',
    dateRange: [
      timestamp2string(getLast7DaysStartTimestamp()),
      timestamp2string(getLast7DaysEndTimestamp()),
    ],
    logType: [],
  };

  const applyRoleColumnVisibilityGuards = (columns) => {
    const next = { ...columns };
    if (!isAdminUser) {
      next[COLUMN_KEYS.USERNAME] = false;
      next[COLUMN_KEYS.RETRY] = false;
      // 普通用户主表固定展示路由后缀列（如 u75），不可关闭。
      next[COLUMN_KEYS.CHANNEL] = true;
    }
    return next;
  };

  // Get default column visibility based on user role
  const getDefaultColumnVisibility = () => {
    return applyRoleColumnVisibilityGuards({
      [COLUMN_KEYS.TIME]: true,
      [COLUMN_KEYS.CHANNEL]: true,
      [COLUMN_KEYS.USERNAME]: isAdminUser || supplierChannelLogsView,
      [COLUMN_KEYS.TOKEN]: true,
      [COLUMN_KEYS.GROUP]: true,
      [COLUMN_KEYS.TYPE]: true,
      [COLUMN_KEYS.MODEL]: true,
      [COLUMN_KEYS.USE_TIME]: true,
      [COLUMN_KEYS.PROMPT]: true,
      [COLUMN_KEYS.COMPLETION]: true,
      [COLUMN_KEYS.COST]: true,
      [COLUMN_KEYS.RETRY]: isAdminUser,
      [COLUMN_KEYS.IP]: true,
      [COLUMN_KEYS.DETAILS]: true,
    });
  };

  const getInitialVisibleColumns = () => {
    const defaults = getDefaultColumnVisibility();
    const savedColumns = localStorage.getItem(STORAGE_KEY);

    if (!savedColumns) {
      return defaults;
    }

    try {
      const parsed = JSON.parse(savedColumns);
      return applyRoleColumnVisibilityGuards({ ...defaults, ...parsed });
    } catch (e) {
      console.error('Failed to parse saved column preferences', e);
      return defaults;
    }
  };

  const getInitialBillingDisplayMode = () => {
    const savedMode = localStorage.getItem(BILLING_DISPLAY_MODE_STORAGE_KEY);
    if (savedMode === 'price' || savedMode === 'ratio') {
      return savedMode;
    }
    return localStorage.getItem('quota_display_type') === 'TOKENS'
      ? 'ratio'
      : 'price';
  };

  // Column visibility state
  const [visibleColumns, setVisibleColumns] = useState(
    getInitialVisibleColumns,
  );
  const [showColumnSelector, setShowColumnSelector] = useState(false);
  const [billingDisplayMode, setBillingDisplayMode] = useState(
    getInitialBillingDisplayMode,
  );

  // Compact mode
  const [compactMode, setCompactMode] = useTableCompactMode('logs');

  // User info modal state
  const [showUserInfo, setShowUserInfoModal] = useState(false);
  const [userInfoData, setUserInfoData] = useState(null);

  // Channel affinity usage cache stats modal state (admin only)
  const [
    showChannelAffinityUsageCacheModal,
    setShowChannelAffinityUsageCacheModal,
  ] = useState(false);
  const [channelAffinityUsageCacheTarget, setChannelAffinityUsageCacheTarget] =
    useState(null);
  const [showParamOverrideModal, setShowParamOverrideModal] = useState(false);
  const [paramOverrideTarget, setParamOverrideTarget] = useState(null);
  /** 使用日志（错误类型）行内「错误详情」弹窗 */
  const [errorLogDetail, setErrorLogDetail] = useState({
    visible: false,
    record: null,
  });

  // Initialize default column visibility
  const initDefaultColumns = () => {
    const defaults = getDefaultColumnVisibility();
    setVisibleColumns(defaults);
    localStorage.setItem(STORAGE_KEY, JSON.stringify(defaults));
  };

  // Handle column visibility change
  const handleColumnVisibilityChange = (columnKey, checked) => {
    if (!isAdminUser && columnKey === COLUMN_KEYS.CHANNEL) {
      return;
    }
    const updatedColumns = applyRoleColumnVisibilityGuards({
      ...visibleColumns,
      [columnKey]: checked,
    });
    setVisibleColumns(updatedColumns);
  };

  // Handle "Select All" checkbox
  const handleSelectAll = (checked) => {
    const allKeys = Object.keys(COLUMN_KEYS).map((key) => COLUMN_KEYS[key]);
    const updatedColumns = {};

    allKeys.forEach((key) => {
      if (
        (key === COLUMN_KEYS.USERNAME || key === COLUMN_KEYS.RETRY) &&
        !isAdminUser
      ) {
        updatedColumns[key] = false;
      } else {
        updatedColumns[key] = checked;
      }
    });

    setVisibleColumns(applyRoleColumnVisibilityGuards(updatedColumns));
  };

  // Persist column settings to the role-specific STORAGE_KEY
  useEffect(() => {
    if (Object.keys(visibleColumns).length > 0) {
      localStorage.setItem(
        STORAGE_KEY,
        JSON.stringify(applyRoleColumnVisibilityGuards(visibleColumns)),
      );
    }
  }, [visibleColumns]);

  useEffect(() => {
    localStorage.setItem(BILLING_DISPLAY_MODE_STORAGE_KEY, billingDisplayMode);
  }, [BILLING_DISPLAY_MODE_STORAGE_KEY, billingDisplayMode]);

  const getFormValues = () => {
    const formValues = formApi ? formApi.getValues() : {};

    let start_timestamp = getLast7DaysStartTimestamp();
    let end_timestamp = getLast7DaysEndTimestamp();

    if (
      formValues.dateRange &&
      Array.isArray(formValues.dateRange) &&
      formValues.dateRange.length === 2
    ) {
      const startUnix = toUnixTimestamp(formValues.dateRange[0]);
      const endUnix = toUnixTimestamp(formValues.dateRange[1]);
      if (startUnix > 0) {
        start_timestamp = startUnix;
      }
      if (endUnix > 0) {
        end_timestamp = endUnix;
      }
    }

    return {
      username: formValues.username || '',
      token_name: formValues.token_name || '',
      model_name: formValues.model_name || '',
      start_timestamp,
      end_timestamp,
      channel: formValues.channel || '',
      group: formValues.group || '',
      request_id: formValues.request_id || '',
      logType: normalizeLogTypes(formValues.logType),
    };
  };

  // Statistics functions
  const getLogSelfStat = async () => {
    const {
      token_name,
      model_name,
      start_timestamp,
      end_timestamp,
      group,
      logType: formLogType,
    } = getFormValues();
    const currentLogTypeParam = buildLogTypeQueryParam(
      formLogType !== undefined ? formLogType : logTypes,
    );
    const localStartTimestamp = start_timestamp;
    const localEndTimestamp = end_timestamp;
    let url;
    if (supplierChannelLogsView) {
      url = `/api/user/supplier-channel-logs/stat?type=${currentLogTypeParam}&token_name=${token_name}&model_name=${model_name}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&group=${group}`;
    } else {
      url = `/api/log/self/stat?type=${currentLogTypeParam}&token_name=${token_name}&model_name=${model_name}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&group=${group}`;
    }
    url = encodeURI(url);
    let res = await API.get(url);
    const { success, message, data } = res.data;
    if (success) {
      setStat(data);
    } else {
      showError(message);
    }
  };

  const getLogStat = async () => {
    const {
      username,
      token_name,
      model_name,
      start_timestamp,
      end_timestamp,
      channel,
      group,
      logType: formLogType,
    } = getFormValues();
    const currentLogTypeParam = buildLogTypeQueryParam(
      formLogType !== undefined ? formLogType : logTypes,
    );
    const localStartTimestamp = start_timestamp;
    const localEndTimestamp = end_timestamp;
    let url = `/api/log/stat?type=${currentLogTypeParam}&username=${username}&token_name=${token_name}&model_name=${model_name}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&channel=${channel}&group=${group}`;
    url = encodeURI(url);
    let res = await API.get(url);
    const { success, message, data } = res.data;
    if (success) {
      setStat(data);
    } else {
      showError(message);
    }
  };

  const handleEyeClick = async () => {
    if (loadingStat) {
      return;
    }
    setLoadingStat(true);
    try {
      if (isAdminUser) {
        await getLogStat();
      } else {
        await getLogSelfStat();
      }
      setShowStat(true);
    } catch (err) {
      showError(err?.message || t('请求失败'));
      setShowStat(true);
    } finally {
      setLoadingStat(false);
    }
  };

  // User info function
  const showUserInfoFunc = async (userId) => {
    if (!isAdminUser) {
      return;
    }
    const res = await API.get(`/api/user/${userId}`);
    const { success, message, data } = res.data;
    if (success) {
      setUserInfoData(data);
      setShowUserInfoModal(true);
    } else {
      showError(message);
    }
  };

  const openChannelAffinityUsageCacheModal = (affinity) => {
    const a = affinity || {};
    setChannelAffinityUsageCacheTarget({
      rule_name: a.rule_name || a.reason || '',
      using_group: a.using_group || '',
      key_hint: a.key_hint || '',
      key_fp: a.key_fp || '',
    });
    setShowChannelAffinityUsageCacheModal(true);
  };

  const openParamOverrideModal = (log, other) => {
    const lines = Array.isArray(other?.po) ? other.po.filter(Boolean) : [];
    if (lines.length === 0) {
      return;
    }
    setParamOverrideTarget({
      lines,
      modelName: log?.model_name || '',
      requestId: log?.request_id || '',
      requestPath: other?.request_path || '',
    });
    setShowParamOverrideModal(true);
  };

  /**
   * 打开「错误类型」日志的完整错误内容弹窗。
   * @param {object} record 表格行数据
   */
  const openErrorLogDetail = (record) => {
    if (!record) {
      return;
    }
    setErrorLogDetail({ visible: true, record });
  };

  /**
   * 关闭错误详情弹窗。
   */
  const closeErrorLogDetail = () => {
    setErrorLogDetail({ visible: false, record: null });
  };

  /**
   * 同一异步视频任务的预扣日志与差额结算日志合并为一条展示。
   * 预扣在提交时写入；结算在成片完成后写入（other 含 actual_quota / pre_consumed_quota）。
   */
  const mergeVideoTaskBillingLogs = (rawLogs) => {
    const byTaskId = new Map();
    for (const log of rawLogs) {
      const other = getLogOther(log.other);
      const taskId = other?.task_id;
      if (!taskId) {
        continue;
      }
      const actualQuota = Number(other?.actual_quota);
      const isSettlement =
        Number.isFinite(actualQuota) &&
        actualQuota > 0 &&
        other?.pre_consumed_quota !== undefined &&
        other?.pre_consumed_quota !== null;
      const isPreCharge =
        !isSettlement &&
        (other?.billing_mode === 'video_per_second' ||
          other?.billing_mode === 'video_token_output' ||
          other?.billing_mode === 'video_per_video' ||
          String(other?.request_path || '').includes('/videos'));
      if (!isSettlement && !isPreCharge) {
        continue;
      }
      if (!byTaskId.has(taskId)) {
        byTaskId.set(taskId, { pre: null, settle: null });
      }
      const entry = byTaskId.get(taskId);
      if (isSettlement) {
        entry.settle = log;
      } else if (isPreCharge) {
        entry.pre = log;
      }
    }

    const hideIds = new Set();
    const patches = new Map();
    for (const { pre, settle } of byTaskId.values()) {
      if (!pre || !settle) {
        continue;
      }
      hideIds.add(pre.id);
      const preOther = getLogOther(pre.other);
      const settleOther = getLogOther(settle.other);
      const actualQuota = Number(settleOther.actual_quota);
      const preConsumed = Number(
        settleOther.pre_consumed_quota ?? pre.quota ?? 0,
      );
      const mergedOther = {
        ...settleOther,
        request_path: settleOther.request_path || preOther.request_path,
        video_billed_quota: actualQuota,
        video_pre_consumed_quota: preConsumed,
        video_final_quota: actualQuota,
      };
      patches.set(settle.id, {
        type: 2,
        quota: actualQuota,
        request_id: settle.request_id || pre.request_id,
        other:
          typeof settle.other === 'string'
            ? JSON.stringify(mergedOther)
            : mergedOther,
      });
    }

    return rawLogs
      .filter((log) => !hideIds.has(log.id))
      .map((log) => {
        const patch = patches.get(log.id);
        return patch ? { ...log, ...patch } : log;
      });
  };

  // Format logs data
  const setLogsFormat = (rawLogs) => {
    const logs = mergeVideoTaskBillingLogs(rawLogs);

    const requestConversionDisplayValue = (conversionChain) => {
      const chain = Array.isArray(conversionChain)
        ? conversionChain.filter(Boolean)
        : [];
      if (chain.length <= 1) {
        return t('原生格式');
      }
      return `${chain.join(' -> ')}`;
    };

    const taskFinalQuotaMap = {};
    for (let i = 0; i < logs.length; i++) {
      const other = getLogOther(logs[i].other);
      const taskId = other?.task_id;
      const actualQuota = Number(other?.actual_quota);
      if (
        taskId &&
        Number.isFinite(actualQuota) &&
        actualQuota > 0 &&
        (!taskFinalQuotaMap[taskId] || actualQuota > taskFinalQuotaMap[taskId])
      ) {
        taskFinalQuotaMap[taskId] = actualQuota;
      }
    }

    let expandDatesLocal = {};
    for (let i = 0; i < logs.length; i++) {
      logs[i].timestamp2string = timestamp2string(logs[i].created_at);
      logs[i].key = logs[i].id;
      let other = getLogOther(logs[i].other);
      const isDeltaRefundLog =
        logs[i].type === 6 && other?.billing_phase === 'delta_refund';
      const shouldShowConsumeBillingDetail =
        logs[i].type === 2 || isDeltaRefundLog;
      const billingPhase = getVideoBillingPhase(other, logs[i]?.quota || 0);
      const actualQuota = Number(other?.actual_quota);
      const displayQuota = billingPhase
        ? billingPhase.amount
        : Number.isFinite(actualQuota) && actualQuota > 0
          ? actualQuota
          : Number(other?.video_billed_quota) > 0
            ? Number(other.video_billed_quota)
            : other?.task_id && taskFinalQuotaMap[other.task_id]
              ? taskFinalQuotaMap[other.task_id]
              : logs[i]?.quota || 0;
      logs[i].quota = displayQuota;
      if (billingPhase) {
        logs[i].video_billing_phase_label = billingPhase.label;
      }
      const aggregatedQuota = displayQuota;
      const channelPriceDiscountLogPct =
        other?.channel_price_discount_percent ?? 100;
      const videoPerSecondBillingDetail = hasVideoPerSecondDetail(other)
        ? renderVideoPerSecondBillingDetail(logs[i], other, aggregatedQuota)
        : null;
      const videoPerTokenBillingDetail = hasVideoPerTokenDetail(other)
        ? renderVideoPerTokenBillingDetail(logs[i], other, aggregatedQuota)
        : null;
      const videoPerVideoBillingDetail = hasVideoPerVideoDetail(other)
        ? renderVideoPerVideoBillingDetail(logs[i], other, aggregatedQuota)
        : null;
      const videoBillingDetail =
        videoPerSecondBillingDetail ||
        videoPerTokenBillingDetail ||
        videoPerVideoBillingDetail;
      // ASR 语音识别按秒计费日志（other.asr=true，含 audio_seconds）
      const asrBillingDetail =
        other?.asr === true && Number(other?.audio_seconds || 0) > 0;
      const zeroBilledVideoNoChargeText = isZeroBilledVideoNoChargeLog(
        logs[i],
        other,
      )
        ? t('视频调用失败，本次不产生计费')
        : null;
      let expandDataLocal = [];

      const routeSlug = String(logs[i].route_slug || '').trim();
      if (
        routeSlug &&
        (logs[i].type === 0 ||
          logs[i].type === 2 ||
          logs[i].type === 5 ||
          logs[i].type === 6)
      ) {
        expandDataLocal.push({
          key: t('路由后缀'),
          value: routeSlug,
        });
      }
      if (
        isAdminUser &&
        logs[i].channel != null &&
        String(logs[i].channel) !== '' &&
        (logs[i].type === 0 || logs[i].type === 2 || logs[i].type === 6)
      ) {
        expandDataLocal.push({
          key: t('渠道'),
          value: String(logs[i].channel),
        });
      }
      if (logs[i].request_id) {
        expandDataLocal.push({
          key: t('Request ID'),
          value: logs[i].request_id,
        });
      }
      // ASR 异步任务：展示 task_id（type=6 退款日志走下方专属分支，避免重复）
      if (other?.task_id && logs[i].type !== 6) {
        expandDataLocal.push({
          key: t('任务ID'),
          value: other.task_id,
        });
      }
      if (asrBillingDetail) {
        expandDataLocal.push({
          key: t('音频时长'),
          value: `${formatASRSecondsDisplay(other.audio_seconds)} ${t('秒')}`,
        });
      }
      if (logs[i].type === 5) {
        const errContent = logs[i].content;
        if (errContent != null && String(errContent).trim() !== '') {
          expandDataLocal.push({
            key: t('错误详情'),
            value: (
              <div
                style={{
                  maxWidth: 800,
                  whiteSpace: 'pre-wrap',
                  wordBreak: 'break-word',
                  lineHeight: 1.6,
                }}
              >
                {String(errContent)}
              </div>
            ),
          });
        }
      }
      if (other?.ws || other?.audio) {
        expandDataLocal.push({
          key: t('语音输入'),
          value: other.audio_input,
        });
        expandDataLocal.push({
          key: t('语音输出'),
          value: other.audio_output,
        });
        expandDataLocal.push({
          key: t('文字输入'),
          value: other.text_input,
        });
        expandDataLocal.push({
          key: t('文字输出'),
          value: other.text_output,
        });
      }
      if (other?.cache_tokens > 0) {
        expandDataLocal.push({
          key: t('缓存 Tokens'),
          value: other.cache_tokens,
        });
      }
      if (other?.cache_creation_tokens > 0) {
        expandDataLocal.push({
          key: t('缓存创建 Tokens'),
          value: other.cache_creation_tokens,
        });
      }
      if (shouldShowConsumeBillingDetail && !other?.request_tier_pricing) {
        expandDataLocal.push({
          key: t('日志详情'),
          value: zeroBilledVideoNoChargeText
            ? zeroBilledVideoNoChargeText
            : videoPerSecondBillingDetail
              ? renderVideoPerSecondBillingBrief(other, aggregatedQuota)
              : videoPerTokenBillingDetail
                ? renderVideoPerTokenBillingBrief(other, aggregatedQuota)
                : videoPerVideoBillingDetail
                ? renderVideoPerVideoBillingBrief(other, aggregatedQuota)
                : asrBillingDetail
                  ? renderASRBillingBrief(other)
                  : other?.claude
                  ? renderClaudeLogContent(
                      other?.model_ratio,
                      other?.completion_ratio,
                      other?.model_price,
                      other?.group_ratio,
                      other?.user_group_ratio,
                      other?.cache_ratio || 1.0,
                      other?.cache_creation_ratio || 1.0,
                      other?.cache_creation_tokens_5m || 0,
                      other?.cache_creation_ratio_5m ||
                        other?.cache_creation_ratio ||
                        1.0,
                      other?.cache_creation_tokens_1h || 0,
                      other?.cache_creation_ratio_1h ||
                        other?.cache_creation_ratio ||
                        1.0,
                      billingDisplayMode,
                      true,
                      channelPriceDiscountLogPct,
                      other,
                    )
                  : renderLogContent(
                      other?.model_ratio,
                      other?.completion_ratio,
                      other?.model_price,
                      other?.group_ratio,
                      other?.user_group_ratio,
                      other?.cache_ratio || 1.0,
                      false,
                      1.0,
                      other?.web_search || false,
                      other?.web_search_call_count || 0,
                      other?.file_search || false,
                      other?.file_search_call_count || 0,
                      billingDisplayMode,
                      true,
                      other?.video_ratio || 0,
                      other?.video_completion_ratio || 1.0,
                      other?.video_output_tokens || 0,
                      other?.video_input_text_tokens || 0,
                      other?.billing_mode || '',
                      aggregatedQuota,
                      channelPriceDiscountLogPct,
                      other,
                    ),
        });
        if (
          logs[i]?.content &&
          !videoBillingDetail &&
          other?.billing_mode !== 'video_per_video'
        ) {
          expandDataLocal.push({
            key: t('其他详情'),
            value: trimDecimalsInLogDetailText(logs[i].content),
          });
        }
        if (isAdminUser && other?.reject_reason) {
          expandDataLocal.push({
            key: t('拦截原因'),
            value: other.reject_reason,
          });
        }
      }
      if (shouldShowConsumeBillingDetail) {
        let modelMapped =
          other?.is_model_mapped &&
          other?.upstream_model_name &&
          other?.upstream_model_name !== '';
        if (modelMapped && !videoBillingDetail) {
          expandDataLocal.push({
            key: t('请求并计费模型'),
            value: logs[i].model_name,
          });
          if (isAdminUser) {
            expandDataLocal.push({
              key: t('实际模型'),
              value: other.upstream_model_name,
            });
          }
        }

        const isViolationFeeLog =
          other?.violation_fee === true ||
          Boolean(other?.violation_fee_code) ||
          Boolean(other?.violation_fee_marker);

        let content = '';
        if (!isViolationFeeLog) {
          content =
            zeroBilledVideoNoChargeText ||
            (videoBillingDetail
              ? videoBillingDetail
              : renderConsumeBillingProcess({
                  record: logs[i],
                  other,
                  billingDisplayMode,
                  channelPriceDiscountPercent: channelPriceDiscountLogPct,
                  t,
                }));
          const isPerCallBilling =
            Number.isFinite(Number(other?.model_price)) &&
            Number(other?.model_price) > 0;
          // 按次 / 按张：固定单价用标签展示，键名为「计费详情」；其余仍为「计费过程」
          const billingDetailKey =
            other?.billing_mode === 'image_per_image' || isPerCallBilling
              ? t('计费详情')
              : t('计费过程');
          expandDataLocal.push({
            key: billingDetailKey,
            value: content,
          });
        }
        if (other?.reasoning_effort) {
          expandDataLocal.push({
            key: t('Reasoning Effort'),
            value: other.reasoning_effort,
          });
        }
      }
      if (logs[i].type === 6) {
        if (other?.task_id) {
          expandDataLocal.push({
            key: t('任务ID'),
            value: other.task_id,
          });
        }
        if (other?.reason) {
          expandDataLocal.push({
            key: t('失败原因'),
            value: (
              <div
                style={{
                  maxWidth: 600,
                  whiteSpace: 'normal',
                  wordBreak: 'break-word',
                  lineHeight: 1.6,
                }}
              >
                {other.reason}
              </div>
            ),
          });
        }
      }
      if (other?.request_path) {
        expandDataLocal.push({
          key: t('请求路径'),
          value: other.request_path,
        });
      }
      if (isAdminUser && other?.stream_status) {
        const ss = other.stream_status;
        const isOk = ss.status === 'ok';
        const statusLabel = isOk ? '✓ ' + t('正常') : '✗ ' + t('异常');
        let streamValue =
          statusLabel + ' (' + (ss.end_reason || 'unknown') + ')';
        if (ss.error_count > 0) {
          streamValue += ` [${t('软错误')}: ${ss.error_count}]`;
        }
        if (ss.end_error) {
          streamValue += ` - ${ss.end_error}`;
        }
        expandDataLocal.push({
          key: t('流状态'),
          value: streamValue,
        });
        if (Array.isArray(ss.errors) && ss.errors.length > 0) {
          expandDataLocal.push({
            key: t('流错误详情'),
            value: (
              <div
                style={{
                  maxWidth: 600,
                  whiteSpace: 'pre-line',
                  wordBreak: 'break-word',
                  lineHeight: 1.6,
                }}
              >
                {ss.errors.join('\n')}
              </div>
            ),
          });
        }
      }
      if (Array.isArray(other?.po) && other.po.length > 0) {
        expandDataLocal.push({
          key: t('参数覆盖'),
          value: (
            <ParamOverrideEntry
              count={other.po.length}
              t={t}
              onOpen={(event) => {
                event.stopPropagation();
                openParamOverrideModal(logs[i], other);
              }}
            />
          ),
        });
      }
      if (other?.billing_source === 'subscription') {
        const planId = other?.subscription_plan_id;
        const planTitle = other?.subscription_plan_title || '';
        const subscriptionId = other?.subscription_id;
        const unit = t('额度');
        const pre = other?.subscription_pre_consumed ?? 0;
        const postDelta = other?.subscription_post_delta ?? 0;
        const finalConsumed = other?.subscription_consumed ?? pre + postDelta;
        const remain = other?.subscription_remain;
        const total = other?.subscription_total;
        // Use multiple Description items to avoid an overlong single line.
        if (planId) {
          expandDataLocal.push({
            key: t('订阅套餐'),
            value: `#${planId} ${planTitle}`.trim(),
          });
        }
        if (subscriptionId) {
          expandDataLocal.push({
            key: t('订阅实例'),
            value: `#${subscriptionId}`,
          });
        }
        const settlementLines = [
          `${t('预扣')}：${pre} ${unit}`,
          `${t('结算差额')}：${postDelta > 0 ? '+' : ''}${postDelta} ${unit}`,
          `${t('最终抵扣')}：${finalConsumed} ${unit}`,
        ]
          .filter(Boolean)
          .join('\n');
        expandDataLocal.push({
          key: t('订阅结算'),
          value: (
            <div style={{ whiteSpace: 'pre-line' }}>{settlementLines}</div>
          ),
        });
        if (remain !== undefined && total !== undefined) {
          expandDataLocal.push({
            key: t('订阅剩余'),
            value: `${remain}/${total} ${unit}`,
          });
        }
        expandDataLocal.push({
          key: t('订阅说明'),
          value: t(
            'token 会按倍率换算成“额度/次数”，请求结束后再做差额结算（补扣/返还）。',
          ),
        });
      }
      if (isAdminUser && logs[i].type !== 6) {
        expandDataLocal.push({
          key: t('请求转换'),
          value: requestConversionDisplayValue(other?.request_conversion),
        });
      }
      if (isAdminUser && logs[i].type !== 6) {
        let localCountMode = '';
        if (other?.billing_mode === 'video_token') {
          // Video task channels billed via duration*W*H*fps/1024 token estimate;
          // fully computed locally from the request body, never reads upstream usage.
          localCountMode = t('视频本地按 token 计费');
        } else if (other?.billing_mode === 'video_per_second') {
          localCountMode = t('分辨率阶梯计费');
        } else if (other?.billing_mode === 'video_token_output') {
          localCountMode = t('视频按 token 计费');
        } else if (other?.billing_mode === 'video_per_video') {
          localCountMode = t('按视频数量计费');
        } else if (other?.billing_mode === 'image_per_image') {
          localCountMode = t('按张计费');
        } else if (other?.asr === true) {
          localCountMode = t('语音识别按秒计费');
        } else if (
          Number.isFinite(Number(other?.model_price)) &&
          Number(other?.model_price) > 0
        ) {
          localCountMode = t('按次计费');
        } else if (
          other?.image_usd_per_image > 0 ||
          (other?.use_price &&
            String(other?.request_path || '').includes('/images/'))
        ) {
          localCountMode = t('按张计费');
        } else if (other?.admin_info?.local_count_tokens) {
          localCountMode = t('本地计费');
        } else {
          localCountMode = t('上游返回');
        }
        expandDataLocal.push({
          key: t('计费模式'),
          value: localCountMode,
        });
      }
      expandDatesLocal[logs[i].key] = expandDataLocal;
    }

    setExpandData(expandDatesLocal);
    setLogs(logs);
  };

  // Load logs function
  const loadLogs = async (startIdx, pageSize, customLogTypes = null) => {
    setLoading(true);

    let url = '';
    const {
      username,
      token_name,
      model_name,
      start_timestamp,
      end_timestamp,
      channel,
      group,
      request_id,
      logType: formLogType,
    } = getFormValues();

    const currentLogTypeParam = buildLogTypeQueryParam(
      customLogTypes !== null
        ? customLogTypes
        : formLogType !== undefined
          ? formLogType
          : logTypes,
    );

    const localStartTimestamp = start_timestamp;
    const localEndTimestamp = end_timestamp;
    if (isAdminUser) {
      url = `/api/log/?p=${startIdx}&page_size=${pageSize}&type=${currentLogTypeParam}&username=${username}&token_name=${token_name}&model_name=${model_name}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&channel=${channel}&group=${group}&request_id=${request_id}`;
    } else if (supplierChannelLogsView) {
      url = `/api/user/supplier-channel-logs?p=${startIdx}&page_size=${pageSize}&type=${currentLogTypeParam}&token_name=${token_name}&model_name=${model_name}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&group=${group}&request_id=${request_id}`;
    } else {
      url = `/api/log/self/?p=${startIdx}&page_size=${pageSize}&type=${currentLogTypeParam}&token_name=${token_name}&model_name=${model_name}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&group=${group}&request_id=${request_id}`;
    }
    url = encodeURI(url);
    const res = await API.get(url);
    const { success, message, data } = res.data;
    if (success) {
      const newPageData = data.items;
      setActivePage(data.page);
      setPageSize(data.page_size);
      setLogCount(data.total);

      setLogsFormat(newPageData);
    } else {
      showError(message);
    }
    setLoading(false);
  };

  // Page handlers
  const handlePageChange = (page) => {
    setActivePage(page);
    loadLogs(page, pageSize).then((r) => {});
  };

  const handlePageSizeChange = async (size) => {
    localStorage.setItem('page-size', size + '');
    setPageSize(size);
    setActivePage(1);
    loadLogs(activePage, size)
      .then()
      .catch((reason) => {
        showError(reason);
      });
  };

  // Refresh function
  const refresh = async () => {
    setActivePage(1);
    handleEyeClick();
    await loadLogs(1, pageSize);
  };

  // Copy text function
  const copyText = async (e, text) => {
    e.stopPropagation();
    if (await copy(text)) {
      showSuccess('已复制：' + text);
    } else {
      Modal.error({ title: t('无法复制到剪贴板，请手动复制'), content: text });
    }
  };

  const loadModelOptions = async () => {
    try {
      const res = await API.get('/api/user/models');
      const { success, message, data } = res.data || {};
      if (success) {
        const options = (data || []).map((model) => ({
          label: model,
          value: model,
        }));
        setModelOptions(options);
      } else {
        showError(message);
      }
    } catch (e) {
      showError(e.message || t('加载模型列表失败'));
    }
  };

  // Initialize data
  useEffect(() => {
    const localPageSize =
      parseInt(localStorage.getItem('page-size')) || ITEMS_PER_PAGE;
    setPageSize(localPageSize);
    loadModelOptions();
    loadLogs(activePage, localPageSize)
      .then()
      .catch((reason) => {
        showError(reason);
      });
  }, []);

  // Initialize statistics when formApi is available
  useEffect(() => {
    if (formApi) {
      handleEyeClick();
    }
  }, [formApi]);

  // Check if any record has expandable content
  const hasExpandableRows = () => {
    return logs.some(
      (log) => expandData[log.key] && expandData[log.key].length > 0,
    );
  };

  return {
    // Basic state
    logs,
    expandData,
    showStat,
    loading,
    loadingStat,
    activePage,
    logCount,
    pageSize,
    logTypes,
    modelOptions,
    stat,
    isAdminUser,
    supplierChannelLogsView,

    // Form state
    formApi,
    setFormApi,
    formInitValues,
    getFormValues,

    // Column visibility
    visibleColumns,
    showColumnSelector,
    setShowColumnSelector,
    billingDisplayMode,
    setBillingDisplayMode,
    handleColumnVisibilityChange,
    handleSelectAll,
    initDefaultColumns,
    COLUMN_KEYS,

    // Compact mode
    compactMode,
    setCompactMode,

    // User info modal
    showUserInfo,
    setShowUserInfoModal,
    userInfoData,
    showUserInfoFunc,

    // Channel affinity usage cache stats modal
    showChannelAffinityUsageCacheModal,
    setShowChannelAffinityUsageCacheModal,
    channelAffinityUsageCacheTarget,
    openChannelAffinityUsageCacheModal,
    showParamOverrideModal,
    setShowParamOverrideModal,
    paramOverrideTarget,
    errorLogDetail,
    openErrorLogDetail,
    closeErrorLogDetail,

    // Functions
    loadLogs,
    handlePageChange,
    handlePageSizeChange,
    refresh,
    copyText,
    handleEyeClick,
    setLogsFormat,
    hasExpandableRows,
    setLogTypes,
    openParamOverrideModal,

    // Translation
    t,
  };
};
