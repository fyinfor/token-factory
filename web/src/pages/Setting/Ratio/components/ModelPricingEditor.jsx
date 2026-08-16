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
import {
  Banner,
  Button,
  Card,
  Empty,
  Input,
  Modal,
  Radio,
  RadioGroup,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Tooltip,
  Typography,
} from '@douyinfe/semi-ui';
import {
  IconDelete,
  IconHelpCircle,
  IconPlus,
  IconRefresh,
  IconSave,
  IconSearch,
} from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import {
  PAGE_SIZE,
  PRICE_SUFFIX,
  buildSummaryText,
  getEffectiveBillingMode,
  hasValue,
  useModelPricingEditorState,
} from '../hooks/useModelPricingEditorState';
import {
  buildTierPriceDetails,
  CURRENCY_OPTIONS,
  emptyTierPricing,
  getCurrencySymbol,
  hasTierPricing,
  normalizeTierPricing,
  TIER_BOUNDARY_LT,
  TIER_BOUNDARY_LTE,
} from '../utils/requestTierPricing';
import TierRowsEditor from './TierRowsEditor';
import JsonCodeEditor from '../../../../components/common/ui/JsonCodeEditor';
import { VIDEO_PRICING_JSON_PLACEHOLDER } from '../utils/videoPricingJson';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import { getCurrencyConfig, showError } from '../../../../helpers';
import { formatImageResolutionDisplayLabel, formatVideoResolutionDisplayLabel } from '../../../../helpers/videoResolutionLabel';

const { Text } = Typography;
const EMPTY_CANDIDATE_MODEL_NAMES = [];

// ---- 阶梯计费摘要渲染（v2 重构：统一 tierPricing 模型）----

const renderSummary = (record, t) => {
  if (getEffectiveBillingMode(record) !== 'tiered') {
    return <span>{buildSummaryText(record, t)}</span>;
  }

  const tp = normalizeTierPricing(record.tierPricing);
  if (tp.tiers.length === 0) {
    return <span>{buildSummaryText(record, t)}</span>;
  }

  const symbol = getCurrencySymbol(tp.currency);

  return (
    <span className='inline-flex flex-wrap items-center gap-x-2 gap-y-1'>
      <span>{t('阶梯计费')}｜</span>
      <Tooltip
        position='top'
        content={
          <div style={{ maxWidth: 320 }}>
            <div className='font-medium mb-1'>{t('阶梯计费价格明细')}</div>
            {tp.tiers.map((row, idx) => {
              const prev = idx === 0 ? 0 : tp.tiers[idx - 1]?.up_to || 0;
              return (
                <div key={idx}>
                  {prev}～{row.up_to || '∞'}：
                  {t('输入')} {symbol}{Number(row.inputPrice.toFixed(2))} / {t('输出')} {symbol}{Number(row.outputPrice.toFixed(2))}
                </div>
              );
            })}
          </div>
        }
      >
        <IconHelpCircle
          size='small'
          style={{ color: 'var(--semi-color-text-2)', cursor: 'help' }}
        />
      </Tooltip>
      <span>{tp.tiers.length}{t('档')}</span>
    </span>
  );
};
const VIDEO_RESOLUTION_OPTIONS = [
  { label: '480p', value: '854x480' },
  { label: '540p', value: '960x540' },
  { label: '720p', value: '1280x720' },
  { label: '768p', value: '1366x768' },
  { label: '1080p', value: '1920x1080' },
  { label: '2K', value: '2560x1440' },
  { label: '4K', value: '3840x2160' },
];
/** Ai 绘图按张计费档位：与短边分档一致（512P / 1K / 2K / 4K），与视频档位相互独立 */
const IMAGE_RESOLUTION_OPTIONS = [
  { label: '512P', value: '512P' },
  { label: '1K', value: '1K' },
  { label: '2K', value: '2K' },
  { label: '4K', value: '4K' },
];
const VIDEO_RESOLUTION_LABEL_MAP = VIDEO_RESOLUTION_OPTIONS.reduce(
  (acc, item) => ({ ...acc, [item.value]: item.label }),
  {},
);
const DEFAULT_VIDEO_FPS = 24;
const VIDEO_RULE_CARD_STYLE = {
  padding: '10px 12px',
  marginBottom: 8,
  borderRadius: 8,
  border: '1px solid var(--semi-color-border)',
  background: 'var(--semi-color-fill-0)',
};
const VIDEO_RULE_HEADER_ROW_STYLE = {
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'space-between',
  gap: 8,
  flexWrap: 'wrap',
};
const VIDEO_RULE_INPUT_ROW_STYLE = {
  marginTop: 10,
  display: 'flex',
  flexWrap: 'wrap',
  gap: 8,
};

const getSelectableResolutionOptions = (rows, currentIndex) => {
  const usedLabels = new Set(
    (rows || [])
      .map((item, index) =>
        index === currentIndex ? '' : item?.resolution || '',
      )
      .filter(Boolean)
      .map((resolution) =>
        (formatVideoResolutionDisplayLabel(resolution) || resolution)
          .toLowerCase()
          .trim(),
      ),
  );
  const current = String(rows?.[currentIndex]?.resolution || '').trim();
  const options = VIDEO_RESOLUTION_OPTIONS.filter((item) => {
    const label = (
      formatVideoResolutionDisplayLabel(item.value) || item.value
    )
      .toLowerCase()
      .trim();
    return !usedLabels.has(label);
  });
  // 兼容 MiniMax H3 的 768P 等官方档位：当前行值与下拉 canonical 像素不一致时，仍展示同一档位。
  if (current) {
    const currentLabel = (
      formatVideoResolutionDisplayLabel(current) || current
    )
      .toLowerCase()
      .trim();
    const withoutDup = options.filter((item) => {
      if (item.value === current) {
        return true;
      }
      const itemLabel = (
        formatVideoResolutionDisplayLabel(item.value) || item.value
      )
        .toLowerCase()
        .trim();
      return itemLabel !== currentLabel;
    });
    if (!withoutDup.some((item) => item.value === current)) {
      withoutDup.unshift({
        label: formatVideoResolutionDisplayLabel(current) || current,
        value: current,
      });
    }
    return withoutDup;
  }
  return options;
};

const getSelectableImageResolutionOptions = (rows, currentIndex) => {
  const usedLabels = new Set(
    (rows || [])
      .map((item, index) =>
        index === currentIndex ? '' : item?.resolution || '',
      )
      .filter(Boolean)
      .map((resolution) =>
        (
          formatImageResolutionDisplayLabel(resolution) || resolution
        )
          .toLowerCase()
          .trim(),
      ),
  );
  const current = String(rows?.[currentIndex]?.resolution || '').trim();
  const options = IMAGE_RESOLUTION_OPTIONS.filter((item) => {
    const label = (
      formatImageResolutionDisplayLabel(item.value) || item.value
    )
      .toLowerCase()
      .trim();
    return !usedLabels.has(label);
  });
  // 兼容历史写法（如 1080p、1024x1024），保证已保存行仍可选中展示
  if (current) {
    const currentLabel = (
      formatImageResolutionDisplayLabel(current) || current
    )
      .toLowerCase()
      .trim();
    const alreadyListed = options.some(
      (item) =>
        (
          formatImageResolutionDisplayLabel(item.value) || item.value
        )
          .toLowerCase()
          .trim() === currentLabel,
    );
    if (!alreadyListed && !usedLabels.has(currentLabel)) {
      const display =
        formatImageResolutionDisplayLabel(current) || current;
      options.unshift({ label: display, value: current });
    }
  }
  return options;
};

const getBillingModeMeta = (billingMode, t) => {
  switch (billingMode) {
    case 'per-request':
      return { color: 'teal', label: t('按次计费') };
    case 'tiered':
      return { color: 'purple', label: t('阶梯计费') };
    case 'per-token':
    default:
      return { color: 'violet', label: t('按量计费') };
  }
};

const getRuleByResolution = (rows, resolution) =>
  (rows || []).find((row) => row?.resolution === resolution) || null;

const formatTokenNumber = (value) => {
  if (!Number.isFinite(value) || value <= 0) {
    return '-';
  }
  return Math.round(value).toLocaleString();
};

const formatSystemCurrencyPrice = (usdAmount, suffix = '/次') => {
  if (!Number.isFinite(usdAmount) || usdAmount <= 0) {
    return '-';
  }
  const { symbol, rate } = getCurrencyConfig();
  const converted = usdAmount * (Number.isFinite(rate) && rate > 0 ? rate : 1);
  return `${symbol}${converted.toFixed(6)}${suffix}`;
};

const formatPriceByUnit = (amount, unit, customSymbol = '¤', suffix = '') => {
  if (!Number.isFinite(amount) || amount <= 0) return '-';
  switch (unit) {
    case 'USD':
      return `$${amount.toFixed(6)}${suffix}`;
    case 'CNY':
      return `¥${amount.toFixed(6)}${suffix}`;
    case 'CUSTOM':
      return `${customSymbol}${amount.toFixed(6)}${suffix}`;
    case 'TOKENS':
      return `${amount.toFixed(6)} Token${suffix}`;
    default:
      return `${amount.toFixed(6)}${suffix}`;
  }
};

const pickDemoResolution = (selectedModel) => {
  const preferred = ['1920x1080', '1280x720', '854x480'];
  const fromRules = [
    ...(selectedModel?.videoTextToVideoRules || []),
    ...(selectedModel?.videoImageToVideoRules || []),
    ...(selectedModel?.videoUploadRules || []),
    ...(selectedModel?.videoGenerateRules || []),
  ]
    .map((row) => row?.resolution)
    .filter(Boolean);
  const options = Array.from(new Set([...fromRules, ...preferred]));
  const scoredOptions = options
    .map((resolution) => {
      const hasText = Boolean(
        getRuleByResolution(selectedModel?.videoTextToVideoRules, resolution),
      );
      const hasImage = Boolean(
        getRuleByResolution(selectedModel?.videoImageToVideoRules, resolution),
      );
      const hasUpload = Boolean(
        getRuleByResolution(selectedModel?.videoUploadRules, resolution),
      );
      const hasGenerate = Boolean(
        getRuleByResolution(selectedModel?.videoGenerateRules, resolution),
      );
      const coverage = [hasText, hasImage, hasUpload, hasGenerate].filter(
        Boolean,
      ).length;
      const preferredIndex = preferred.indexOf(resolution);
      return {
        resolution,
        hasText,
        coverage,
        preferredRank:
          preferredIndex === -1 ? Number.MAX_SAFE_INTEGER : preferredIndex,
      };
    })
    .filter((item) => item.coverage > 0)
    .sort((a, b) => {
      if (b.coverage !== a.coverage) return b.coverage - a.coverage;
      if (a.hasText !== b.hasText) return a.hasText ? -1 : 1;
      return a.preferredRank - b.preferredRank;
    });
  if (scoredOptions.length > 0) {
    return scoredOptions[0].resolution;
  }
  return preferred[0];
};

const PriceInput = ({
  label,
  value,
  placeholder,
  onChange,
  suffix = PRICE_SUFFIX,
  disabled = false,
  extraText = '',
  headerAction = null,
  hidden = false,
}) => (
  <div style={{ marginBottom: 16 }}>
    <div className='mb-1 font-medium text-gray-700 flex items-center justify-between gap-3'>
      <span>{label}</span>
      {headerAction}
    </div>
    {!hidden ? (
      <Input
        value={value}
        placeholder={placeholder}
        onChange={onChange}
        suffix={suffix}
        disabled={disabled}
      />
    ) : null}
    {extraText ? (
      <div className='mt-1 text-xs text-gray-500'>{extraText}</div>
    ) : null}
  </div>
);

export default function ModelPricingEditor({
  options,
  refresh,
  candidateModelNames = EMPTY_CANDIDATE_MODEL_NAMES,
  forceCandidateModelNames = false,
  filterMode = 'all',
  optionKeys,
  onSaveOutput,
  allowAddModel = true,
  allowDeleteModel = true,
  listDescription = '',
  emptyTitle = '',
  emptyDescription = '',
  includeAllCandidateModels = false,
  onSelectedModelChange,
}) {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const [addVisible, setAddVisible] = useState(false);
  const [batchVisible, setBatchVisible] = useState(false);
  // visibleCategories 保留用于向后兼容，统一模型下不再需要toggle
  const [visibleCategories] = useState({
    output: true,
    cache_read: true,
    cache_write: true,
  });
  const [newModelName, setNewModelName] = useState('');
  const [videoEditMode, setVideoEditMode] = useState('visual');
  const [videoJsonDraft, setVideoJsonDraft] = useState('');

  const {
    selectedModel,
    selectedModelName,
    selectedModelNames,
    setSelectedModelName,
    setSelectedModelNames,
    searchText,
    setSearchText,
    currentPage,
    setCurrentPage,
    loading,
    pendingDeleteModelNames,
    filteredModels,
    pagedData,
    selectedWarnings,
    previewRows,
    isOptionalFieldEnabled,
    handleOptionalFieldToggle,
    handleNumericFieldChange,
    handleBillingModeChange,
    handleVideoBillingModeChange,
    handleVideoPriceUnitChange,
    updateVideoRuleRow,
    addVideoRuleRow,
    removeVideoRuleRow,
    applyVideoPricingJson,
    getVideoPricingJsonText,
    handleImageGenPriceUnitChange,
    updateImageRuleRow,
    addImageRuleRow,
    removeImageRuleRow,
    updateTierPricing,
    handleTierCurrencyChange,
    clearAllTierRatios,
    handleSubmit,
    addModel,
    markModelForDelete,
    restorePendingDelete,
    applySelectedModelPricing,
  } = useModelPricingEditorState({
    options,
    refresh,
    t,
    candidateModelNames,
    strictCandidateModelNames: forceCandidateModelNames,
    filterMode,
    optionKeys,
    onSaveOutput,
    visibleCategories,
    includeAllCandidateModels,
  });

  useEffect(() => {
    onSelectedModelChange?.(selectedModelName);
  }, [onSelectedModelChange, selectedModelName]);

  const tierPriceDetails = useMemo(() => {
    if (!selectedModel?.tierPricing) return [];
    return buildTierPriceDetails(selectedModel.tierPricing, t);
  }, [selectedModel?.tierPricing, t]);

  const videoPerVideoBillingHint = useMemo(() => {
    const { type } = getCurrencyConfig();
    if (type === 'CNY') {
      return t('视频按条价计费说明（人民币展示）');
    }
    if (type === 'TOKENS') {
      return t('视频按条价计费说明（Token模式）');
    }
    if (type === 'CUSTOM') {
      return t('视频按条价计费说明（自定义货币）');
    }
    return t('视频按条价计费说明（美元等）');
  }, [t]);

  const perVideoPriceSuffix = useMemo(() => {
    const unit = ['USD', 'CNY', 'CUSTOM'].includes(
      selectedModel?.videoPriceUnit,
    )
      ? selectedModel.videoPriceUnit
      : getCurrencyConfig().type;
    let sym = getCurrencyConfig().symbol || '$';
    if (unit === 'USD') sym = '$';
    else if (unit === 'CNY') sym = '¥';
    else if (unit === 'CUSTOM') sym = getCurrencyConfig().symbol || '¤';
    if (selectedModel?.videoBillingMode === 'per-token') {
      return `${sym} /1M token`;
    }
    return sym;
  }, [selectedModel?.videoPriceUnit, selectedModel?.videoBillingMode, t]);

  const imagePerImageBillingHint = useMemo(() => {
    const { type } = getCurrencyConfig();
    if (type === 'CNY') {
      return t('图片按张价计费说明（人民币展示）');
    }
    if (type === 'TOKENS') {
      return t('图片按张价计费说明（Token模式）');
    }
    if (type === 'CUSTOM') {
      return t('图片按张价计费说明（自定义货币）');
    }
    return t('图片按张价计费说明（美元等）');
  }, [t]);

  const perImagePriceSuffix = useMemo(() => {
    const unit = ['USD', 'CNY', 'CUSTOM'].includes(
      selectedModel?.imageGenPriceUnit,
    )
      ? selectedModel.imageGenPriceUnit
      : getCurrencyConfig().type;
    if (unit === 'USD') return `${t('每张')}$`;
    if (unit === 'CNY') return `${t('每张')}¥`;
    if (unit === 'CUSTOM') {
      return `${t('每张')}${getCurrencyConfig().symbol || '¤'}`;
    }
    return `${t('每张')}${getCurrencyConfig().symbol || '$'}`;
  }, [selectedModel?.imageGenPriceUnit, t]);

  const flatPerVideoPriceSuffix = useMemo(() => {
    const unit = ['USD', 'CNY', 'CUSTOM'].includes(
      selectedModel?.videoPriceUnit,
    )
      ? selectedModel.videoPriceUnit
      : getCurrencyConfig().type;
    if (unit === 'USD') return '$/视频';
    if (unit === 'CNY') return '¥/视频';
    if (unit === 'CUSTOM') return `${getCurrencyConfig().symbol || '¤'}/视频`;
    return '$/视频';
  }, [selectedModel?.videoPriceUnit, t]);

  const columns = useMemo(
    () => [
      {
        title: t('模型名称'),
        dataIndex: 'name',
        key: 'name',
        render: (text, record) => (
          <Space>
            <Button
              theme='borderless'
              type='tertiary'
              onClick={() => setSelectedModelName(record.name)}
              style={{
                padding: 0,
                color:
                  record.name === selectedModelName
                    ? 'var(--semi-color-primary)'
                    : undefined,
              }}
            >
              {text}
            </Button>
            {pendingDeleteModelNames.includes(record.name) ? (
              <Tag color='red' shape='circle'>
                {t('待删除')}
              </Tag>
            ) : null}
            {Array.isArray(selectedModelNames) &&
            record?.name &&
            selectedModelNames.includes(record.name) ? (
              <Tag color='green' shape='circle'>
                {t('已勾选')}
              </Tag>
            ) : null}
          </Space>
        ),
      },
      {
        title: t('计费方式'),
        dataIndex: 'billingMode',
        key: 'billingMode',
        render: (_, record) => {
          const meta = getBillingModeMeta(getEffectiveBillingMode(record), t);
          return <Tag color={meta.color}>{meta.label}</Tag>;
        },
      },
      {
        title: t('价格摘要'),
        dataIndex: 'summary',
        key: 'summary',
        render: (_, record) => renderSummary(record, t),
      },
      {
        title: t('操作'),
        key: 'action',
        render: (_, record) => (
          <Space>
            {allowDeleteModel ? (
              pendingDeleteModelNames.includes(record.name) ? (
                <Button
                  size='small'
                  icon={<IconRefresh />}
                  onClick={(event) => {
                    event.stopPropagation();
                    restorePendingDelete(record.name);
                  }}
                >
                  {t('恢复')}
                </Button>
              ) : (
                <Button
                  size='small'
                  type='danger'
                  icon={<IconDelete />}
                  onClick={(event) => {
                    event.stopPropagation();
                    markModelForDelete(record.name);
                  }}
                />
              )
            ) : null}
          </Space>
        ),
      },
    ],
    [
      allowDeleteModel,
      markModelForDelete,
      pendingDeleteModelNames,
      restorePendingDelete,
      selectedModelName,
      selectedModelNames,
      setSelectedModelName,
      t,
    ],
  );

  const handleAddModel = () => {
    if (addModel(newModelName)) {
      setNewModelName('');
      setAddVisible(false);
    }
  };

  useEffect(() => {
    setVideoEditMode('visual');
    setVideoJsonDraft('');
  }, [selectedModelName]);

  const syncVideoJsonBeforeSave = () => {
    if (
      videoEditMode !== 'json' ||
      !selectedModel ||
      !isOptionalFieldEnabled(selectedModel, 'video')
    ) {
      return true;
    }
    const result = applyVideoPricingJson(videoJsonDraft);
    if (!result.ok) {
      showError(
        result.message === '不是合法的 JSON 字符串'
          ? t('不是合法的 JSON 字符串')
          : result.message,
      );
      return false;
    }
    return true;
  };

  const switchVideoToJsonMode = () => {
    if (!selectedModel) return;
    setVideoJsonDraft(getVideoPricingJsonText());
    setVideoEditMode('json');
  };

  const switchVideoToVisualMode = () => {
    if (!syncVideoJsonBeforeSave()) return;
    setVideoEditMode('visual');
  };

  const handleApplyChanges = () => {
    if (!syncVideoJsonBeforeSave()) return;
    handleSubmit();
  };

  const rowSelection = {
    selectedRowKeys: selectedModelNames,
    onChange: (selectedRowKeys) => setSelectedModelNames(selectedRowKeys),
  };

  return (
    <>
      <Space vertical align='start' style={{ width: '100%' }}>
        <Card bodyStyle={{ padding: 12 }} style={{ width: '100%' }}>
          <Space wrap>
            {allowAddModel ? (
              <Button
                icon={<IconPlus />}
                onClick={() => setAddVisible(true)}
                style={isMobile ? { width: '100%' } : undefined}
              >
                {t('添加模型')}
              </Button>
            ) : null}
            <Button
              type='primary'
              icon={<IconSave />}
              loading={loading}
              onClick={handleApplyChanges}
              style={isMobile ? { width: '100%' } : undefined}
            >
              {t('应用更改')}
            </Button>
            <Button
              disabled={!selectedModel || selectedModelNames.length === 0}
              onClick={() => setBatchVisible(true)}
              style={isMobile ? { width: '100%' } : undefined}
            >
              {t('批量应用当前模型价格')}
              {selectedModelNames.length > 0
                ? ` (${selectedModelNames.length})`
                : ''}
            </Button>
            <Input
              prefix={<IconSearch />}
              placeholder={t('搜索模型名称')}
              value={searchText}
              onChange={(value) => setSearchText(value)}
              style={{ width: isMobile ? '100%' : 220 }}
              showClear
            />
          </Space>
        </Card>
        {listDescription ? (
          <div className='text-sm text-gray-500'>{listDescription}</div>
        ) : null}
        {selectedModelNames.length > 0 ? (
          <div
            style={{
              width: '100%',
              padding: '10px 12px',
              borderRadius: 8,
              background: 'var(--semi-color-primary-light-default)',
              border: '1px solid var(--semi-color-primary)',
              color: 'var(--semi-color-primary)',
              fontWeight: 600,
            }}
          >
            {t('已勾选 {{count}} 个模型', { count: selectedModelNames.length })}
          </div>
        ) : null}

        <div
          style={{
            width: '100%',
            display: 'grid',
            gap: 16,
            gridTemplateColumns: isMobile
              ? 'minmax(0, 1fr)'
              : 'minmax(360px, 1.1fr) minmax(420px, 1fr)',
          }}
        >
          <Card
            bodyStyle={{ padding: 0 }}
            style={isMobile ? { order: 2 } : undefined}
          >
            <div style={{ overflowX: 'auto' }}>
              <Table
                columns={columns}
                dataSource={pagedData}
                rowKey='name'
                rowSelection={rowSelection}
                pagination={{
                  currentPage,
                  pageSize: PAGE_SIZE,
                  total: filteredModels.length,
                  onPageChange: (page) => setCurrentPage(page),
                  showTotal: true,
                  showSizeChanger: false,
                }}
                empty={
                  <div style={{ textAlign: 'center', padding: '20px' }}>
                    {emptyTitle || t('暂无模型')}
                  </div>
                }
                onRow={(record) => {
                  const isPendingDelete = pendingDeleteModelNames.includes(
                    record.name,
                  );
                  return {
                    style: {
                      background: isPendingDelete
                        ? 'var(--semi-color-danger-light-default)'
                        : selectedModelNames.includes(record.name)
                          ? 'var(--semi-color-success-light-default)'
                          : record.name === selectedModelName
                            ? 'var(--semi-color-primary-light-default)'
                            : undefined,
                      boxShadow: isPendingDelete
                        ? 'inset 4px 0 0 var(--semi-color-danger)'
                        : selectedModelNames.includes(record.name)
                          ? 'inset 4px 0 0 var(--semi-color-success)'
                          : record.name === selectedModelName
                            ? 'inset 4px 0 0 var(--semi-color-primary)'
                            : undefined,
                      opacity: isPendingDelete ? 0.72 : undefined,
                      transition:
                        'background 0.2s ease, box-shadow 0.2s ease, opacity 0.2s ease',
                    },
                    onClick: () => setSelectedModelName(record.name),
                  };
                }}
                scroll={isMobile ? { x: 720 } : undefined}
              />
            </div>
          </Card>

          <Card
            style={isMobile ? { order: 1 } : undefined}
            title={selectedModel ? selectedModel.name : t('模型计费编辑器')}
            headerExtraContent={
              selectedModel
                ? (() => {
                    const meta = getBillingModeMeta(
                      getEffectiveBillingMode(selectedModel),
                      t,
                    );
                    return <Tag color={meta.color}>{meta.label}</Tag>;
                  })()
                : null
            }
          >
            {!selectedModel ? (
              <Empty
                title={emptyTitle || t('暂无模型')}
                description={
                  emptyDescription || t('请先新增模型或从左侧列表选择一个模型')
                }
              />
            ) : (
              <div>
                <div className='mb-4'>
                  <div className='mb-2 font-medium text-gray-700 flex items-center gap-1'>
                    {t('计费方式')}
                    <Tooltip
                      position='top'
                      content={
                        <div style={{ maxWidth: 320 }}>
                          <div className='font-medium mb-1'>
                            {t('计费方式生效优先级')}
                          </div>
                          <div>
                            {t(
                              '系统按“按次计费 > 阶梯计费 > 按量计费”的优先级选择实际生效的计费方式。',
                            )}
                          </div>
                          <div className='mt-2 text-xs'>
                            {t(
                              '如需使用低优先级计费方式，请先清空更高优先级中已填写的价格；已配置的其它价格会保留保存，但不会生效。',
                            )}
                          </div>
                        </div>
                      }
                    >
                      <IconHelpCircle
                        style={{
                          cursor: 'help',
                          color: 'var(--semi-color-text-2)',
                        }}
                      />
                    </Tooltip>
                  </div>
                  <RadioGroup
                    type='button'
                    value={selectedModel.billingMode}
                    onChange={(event) =>
                      handleBillingModeChange(event.target.value)
                    }
                  >
                    <Radio value='per-token'>{t('按量计费')}</Radio>
                    <Radio value='tiered'>{t('阶梯计费')}</Radio>
                    <Radio value='per-request'>{t('按次计费')}</Radio>
                  </RadioGroup>
                  <div className='mt-2 text-xs text-gray-500'>
                    {t(
                      '这个界面默认按价格填写，保存时会自动换算回后端需要的倍率 JSON。',
                    )}
                  </div>
                </div>

                {selectedWarnings.length > 0 ? (
                  <Card
                    bodyStyle={{ padding: 12 }}
                    style={{
                      marginBottom: 16,
                      background: 'var(--semi-color-warning-light-default)',
                    }}
                  >
                    <div className='font-medium mb-2'>{t('当前提示')}</div>
                    {selectedWarnings.map((warning) => (
                      <div key={warning} className='text-sm text-gray-700 mb-1'>
                        {warning}
                      </div>
                    ))}
                  </Card>
                ) : null}

                {selectedModel.billingMode === 'per-token' ? (
                  <>
                    <Card
                      bodyStyle={{ padding: 16 }}
                      style={{
                        marginBottom: 16,
                        background: 'var(--semi-color-fill-0)',
                      }}
                    >
                      <div className='font-medium mb-3'>{t('基础价格')}</div>
                      <PriceInput
                        label={t('输入价格')}
                        value={selectedModel.inputPrice}
                        placeholder={t('输入 $/1M')}
                        onChange={(value) =>
                          handleNumericFieldChange('inputPrice', value)
                        }
                      />
                      {selectedModel.completionRatioLocked ? (
                        <Banner
                          type='warning'
                          bordered
                          fullMode={false}
                          closeIcon={null}
                          style={{ marginBottom: 12 }}
                          title={t('输出价格已锁定')}
                          description={t(
                            '该模型输出倍率由后端固定为 {{ratio}}。输出价格不能在这里修改。',
                            {
                              ratio: selectedModel.lockedCompletionRatio || '-',
                            },
                          )}
                        />
                      ) : null}
                      <PriceInput
                        label={t('输出价格')}
                        value={selectedModel.completionPrice}
                        placeholder={t('输入 $/1M')}
                        onChange={(value) =>
                          handleNumericFieldChange('completionPrice', value)
                        }
                        headerAction={
                          <Switch
                            size='small'
                            checked={isOptionalFieldEnabled(
                              selectedModel,
                              'completionPrice',
                            )}
                            disabled={selectedModel.completionRatioLocked}
                            onChange={(checked) =>
                              handleOptionalFieldToggle(
                                'completionPrice',
                                checked,
                              )
                            }
                          />
                        }
                        hidden={
                          !isOptionalFieldEnabled(
                            selectedModel,
                            'completionPrice',
                          )
                        }
                        disabled={
                          !hasValue(selectedModel.inputPrice) ||
                          selectedModel.completionRatioLocked
                        }
                        extraText={
                          selectedModel.completionRatioLocked
                            ? t(
                                '后端固定倍率：{{ratio}}。该字段仅展示换算后的价格。',
                                {
                                  ratio:
                                    selectedModel.lockedCompletionRatio || '-',
                                },
                              )
                            : !isOptionalFieldEnabled(
                                  selectedModel,
                                  'completionPrice',
                                )
                              ? t('当前未启用，需要时再打开即可。')
                              : ''
                        }
                      />
                      <PriceInput
                        label={t('缓存读取价格')}
                        value={selectedModel.cachePrice}
                        placeholder={t('输入 $/1M')}
                        onChange={(value) =>
                          handleNumericFieldChange('cachePrice', value)
                        }
                        headerAction={
                          <Switch
                            size='small'
                            checked={isOptionalFieldEnabled(
                              selectedModel,
                              'cachePrice',
                            )}
                            onChange={(checked) =>
                              handleOptionalFieldToggle('cachePrice', checked)
                            }
                          />
                        }
                        hidden={
                          !isOptionalFieldEnabled(selectedModel, 'cachePrice')
                        }
                        disabled={!hasValue(selectedModel.inputPrice)}
                        extraText={
                          !isOptionalFieldEnabled(selectedModel, 'cachePrice')
                            ? t('当前未启用，需要时再打开即可。')
                            : ''
                        }
                      />
                      <PriceInput
                        label={t('缓存创建价格')}
                        value={selectedModel.createCachePrice}
                        placeholder={t('输入 $/1M')}
                        onChange={(value) =>
                          handleNumericFieldChange('createCachePrice', value)
                        }
                        headerAction={
                          <Switch
                            size='small'
                            checked={isOptionalFieldEnabled(
                              selectedModel,
                              'createCachePrice',
                            )}
                            onChange={(checked) =>
                              handleOptionalFieldToggle(
                                'createCachePrice',
                                checked,
                              )
                            }
                          />
                        }
                        hidden={
                          !isOptionalFieldEnabled(
                            selectedModel,
                            'createCachePrice',
                          )
                        }
                        disabled={!hasValue(selectedModel.inputPrice)}
                        extraText={
                          !isOptionalFieldEnabled(
                            selectedModel,
                            'createCachePrice',
                          )
                            ? t('当前未启用，需要时再打开即可。')
                            : ''
                        }
                      />
                    </Card>

                    <Card
                      bodyStyle={{ padding: 16 }}
                      style={{
                        marginBottom: 16,
                        background: 'var(--semi-color-fill-0)',
                      }}
                    >
                      <div className='mb-3'>
                        <div className='font-medium'>{t('扩展价格')}</div>
                        <div className='text-xs text-gray-500 mt-1'>
                          {t('这些价格都是可选项，不填也可以。')}
                        </div>
                      </div>
                      <PriceInput
                        label={t('图片输入价格')}
                        value={selectedModel.imagePrice}
                        placeholder={t('输入 $/1M')}
                        onChange={(value) =>
                          handleNumericFieldChange('imagePrice', value)
                        }
                        headerAction={
                          <Switch
                            size='small'
                            checked={isOptionalFieldEnabled(
                              selectedModel,
                              'imagePrice',
                            )}
                            onChange={(checked) =>
                              handleOptionalFieldToggle('imagePrice', checked)
                            }
                          />
                        }
                        hidden={
                          !isOptionalFieldEnabled(selectedModel, 'imagePrice')
                        }
                        disabled={!hasValue(selectedModel.inputPrice)}
                        extraText={
                          !isOptionalFieldEnabled(selectedModel, 'imagePrice')
                            ? t('当前未启用，需要时再打开即可。')
                            : ''
                        }
                      />
                      <div style={{ marginTop: 8 }}>
                        <div className='mb-1 font-medium text-gray-700 flex items-center justify-between gap-3'>
                          <span>{t('图片生成计费')}</span>
                          <Switch
                            size='small'
                            checked={isOptionalFieldEnabled(
                              selectedModel,
                              'imageGeneration',
                            )}
                            onChange={(checked) =>
                              handleOptionalFieldToggle(
                                'imageGeneration',
                                checked,
                              )
                            }
                          />
                        </div>
                        {!isOptionalFieldEnabled(
                          selectedModel,
                          'imageGeneration',
                        ) ? (
                          <div className='mt-1 text-xs text-gray-500'>
                            {t('当前未启用，需要时再打开即可。')}
                          </div>
                        ) : (
                          <div
                            style={{
                              marginTop: 8,
                              padding: 12,
                              background: 'var(--semi-color-fill-1)',
                              borderRadius: 6,
                            }}
                          >
                            <div className='mb-2 text-xs text-gray-600'>
                              {imagePerImageBillingHint}
                            </div>
                            <div
                              style={{
                                marginBottom: 12,
                                display: 'flex',
                                justifyContent: 'flex-end',
                              }}
                            >
                              <Select
                                value={
                                  selectedModel.imageGenPriceUnit || 'USD'
                                }
                                style={{ width: 170 }}
                                onChange={(value) =>
                                  handleImageGenPriceUnitChange(String(value))
                                }
                                optionList={[
                                  { label: 'USD ($)', value: 'USD' },
                                  { label: 'CNY (¥)', value: 'CNY' },
                                  {
                                    label: `${t('自定义')} (${getCurrencyConfig().symbol || '¤'})`,
                                    value: 'CUSTOM',
                                  },
                                ]}
                              />
                            </div>

                            <div className='mb-2 font-medium text-gray-700'>
                              {t('文生图价格')}
                            </div>
                            {(selectedModel.imageTextToImageRules || []).map(
                              (row, index, arr) => (
                                <div
                                  key={`text-image-rule-${index}`}
                                  style={{
                                    ...VIDEO_RULE_CARD_STYLE,
                                    marginBottom:
                                      index < arr.length - 1 ? 10 : 0,
                                    display: 'flex',
                                    alignItems: 'center',
                                    flexWrap: 'wrap',
                                    gap: 8,
                                  }}
                                >
                                  <Select
                                    value={row.resolution}
                                    placeholder={t('选择分辨率')}
                                    filter
                                    style={{ width: 140 }}
                                    optionList={getSelectableImageResolutionOptions(
                                      selectedModel.imageTextToImageRules,
                                      index,
                                    )}
                                    onChange={(value) =>
                                      updateImageRuleRow(
                                        'text',
                                        index,
                                        'resolution',
                                        String(value || ''),
                                      )
                                    }
                                  />
                                  <Input
                                    value={row.imagePrice}
                                    placeholder={t('每张价格')}
                                    suffix={perImagePriceSuffix}
                                    style={{ width: 200 }}
                                    onChange={(value) =>
                                      updateImageRuleRow(
                                        'text',
                                        index,
                                        'imagePrice',
                                        value,
                                      )
                                    }
                                  />
                                  <Button
                                    type='danger'
                                    icon={<IconDelete />}
                                    onClick={() =>
                                      removeImageRuleRow('text', index)
                                    }
                                  />
                                </div>
                              ),
                            )}
                            <Button
                              theme='borderless'
                              icon={<IconPlus />}
                              onClick={() => addImageRuleRow('text')}
                              style={{ marginBottom: 12 }}
                            >
                              {t('新增文生图规则')}
                            </Button>

                            <div className='mb-2 font-medium text-gray-700'>
                              {t('图生图价格')}
                            </div>
                            {(selectedModel.imageImageToImageRules || []).map(
                              (row, index, arr) => (
                                <div
                                  key={`image-to-image-rule-${index}`}
                                  style={{
                                    ...VIDEO_RULE_CARD_STYLE,
                                    marginBottom:
                                      index < arr.length - 1 ? 10 : 0,
                                    display: 'flex',
                                    alignItems: 'center',
                                    flexWrap: 'wrap',
                                    gap: 8,
                                  }}
                                >
                                  <Select
                                    value={row.resolution}
                                    placeholder={t('选择分辨率')}
                                    filter
                                    style={{ width: 140 }}
                                    optionList={getSelectableImageResolutionOptions(
                                      selectedModel.imageImageToImageRules,
                                      index,
                                    )}
                                    onChange={(value) =>
                                      updateImageRuleRow(
                                        'imageToImage',
                                        index,
                                        'resolution',
                                        String(value || ''),
                                      )
                                    }
                                  />
                                  <Input
                                    value={row.imagePrice}
                                    placeholder={t('每张价格')}
                                    suffix={perImagePriceSuffix}
                                    style={{ width: 200 }}
                                    onChange={(value) =>
                                      updateImageRuleRow(
                                        'imageToImage',
                                        index,
                                        'imagePrice',
                                        value,
                                      )
                                    }
                                  />
                                  <Button
                                    type='danger'
                                    icon={<IconDelete />}
                                    onClick={() =>
                                      removeImageRuleRow('imageToImage', index)
                                    }
                                  />
                                </div>
                              ),
                            )}
                            <Button
                              theme='borderless'
                              icon={<IconPlus />}
                              onClick={() => addImageRuleRow('imageToImage')}
                              style={{ marginBottom: 8 }}
                            >
                              {t('新增图生图规则')}
                            </Button>

                            <PriceInput
                              label={t('相似分辨率阈值')}
                              value={selectedModel.imageSimilarityThreshold}
                              placeholder={t('默认 0.35')}
                              onChange={(value) =>
                                handleNumericFieldChange(
                                  'imageSimilarityThreshold',
                                  value,
                                )
                              }
                              suffix={t('比例')}
                              extraText={t(
                                '请求分辨率与预设差异在阈值内按最近档位计费；差异过大或无分辨率时使用下方兜底每张价格。',
                              )}
                            />
                            <div style={{ marginTop: 8 }}>
                              <PriceInput
                                label={t('无分辨率或差异过大时的每张价格')}
                                value={selectedModel.imageGenFixedPrice}
                                placeholder={t('输入每张图片价格')}
                                suffix={perImagePriceSuffix}
                                onChange={(value) =>
                                  handleNumericFieldChange(
                                    'imageGenFixedPrice',
                                    value,
                                  )
                                }
                                extraText={t(
                                  '当请求未带分辨率，或与已配置档位像素差距超过阈值时，按此固定每张价格计费。',
                                )}
                              />
                            </div>
                          </div>
                        )}
                      </div>
                      <PriceInput
                        label={t('音频输入价格')}
                        value={selectedModel.audioInputPrice}
                        placeholder={t('输入 $/1M')}
                        onChange={(value) =>
                          handleNumericFieldChange('audioInputPrice', value)
                        }
                        headerAction={
                          <Switch
                            size='small'
                            checked={isOptionalFieldEnabled(
                              selectedModel,
                              'audioInputPrice',
                            )}
                            onChange={(checked) =>
                              handleOptionalFieldToggle(
                                'audioInputPrice',
                                checked,
                              )
                            }
                          />
                        }
                        hidden={
                          !isOptionalFieldEnabled(
                            selectedModel,
                            'audioInputPrice',
                          )
                        }
                        disabled={!hasValue(selectedModel.inputPrice)}
                        extraText={
                          !isOptionalFieldEnabled(
                            selectedModel,
                            'audioInputPrice',
                          )
                            ? t('当前未启用，需要时再打开即可。')
                            : ''
                        }
                      />
                      <PriceInput
                        label={t('音频输出价格')}
                        value={selectedModel.audioOutputPrice}
                        placeholder={t('输入 $/1M')}
                        onChange={(value) =>
                          handleNumericFieldChange('audioOutputPrice', value)
                        }
                        headerAction={
                          <Switch
                            size='small'
                            checked={isOptionalFieldEnabled(
                              selectedModel,
                              'audioOutputPrice',
                            )}
                            disabled={
                              !isOptionalFieldEnabled(
                                selectedModel,
                                'audioInputPrice',
                              )
                            }
                            onChange={(checked) =>
                              handleOptionalFieldToggle(
                                'audioOutputPrice',
                                checked,
                              )
                            }
                          />
                        }
                        hidden={
                          !isOptionalFieldEnabled(
                            selectedModel,
                            'audioOutputPrice',
                          )
                        }
                        disabled={!hasValue(selectedModel.audioInputPrice)}
                        extraText={
                          !isOptionalFieldEnabled(
                            selectedModel,
                            'audioInputPrice',
                          )
                            ? t('请先开启并填写音频输入价格。')
                            : !isOptionalFieldEnabled(
                                  selectedModel,
                                  'audioOutputPrice',
                                )
                              ? t('当前未启用，需要时再打开即可。')
                              : ''
                        }
                      />
                      <div style={{ marginTop: 8 }}>
                        <div className='mb-1 font-medium text-gray-700 flex items-center justify-between gap-3 flex-wrap'>
                          <span className='flex items-center gap-1'>
                            {t('视频价格')}
                            <Tooltip
                              position='top'
                              content={
                                <div style={{ maxWidth: 360 }}>
                                  <div className='font-medium mb-1'>
                                    {t('视频计费说明')}
                                  </div>
                                  <div className='text-sm'>
                                    {t('视频预扣逻辑说明')}
                                  </div>
                                  <div className='mt-2 text-xs opacity-90'>
                                    {t('视频完成后结算说明')}
                                  </div>
                                </div>
                              }
                            >
                              <IconHelpCircle
                                style={{
                                  cursor: 'help',
                                  color: 'var(--semi-color-text-2)',
                                }}
                              />
                            </Tooltip>
                          </span>
                          <Space>
                            {isOptionalFieldEnabled(selectedModel, 'video') ? (
                              <>
                                <Button
                                  size='small'
                                  type={
                                    videoEditMode === 'visual'
                                      ? 'primary'
                                      : 'tertiary'
                                  }
                                  onClick={switchVideoToVisualMode}
                                >
                                  {t('可视化')}
                                </Button>
                                <Button
                                  size='small'
                                  type={
                                    videoEditMode === 'json'
                                      ? 'primary'
                                      : 'tertiary'
                                  }
                                  onClick={switchVideoToJsonMode}
                                >
                                  {t('JSON 模式')}
                                </Button>
                              </>
                            ) : null}
                            <Switch
                              size='small'
                              checked={isOptionalFieldEnabled(
                                selectedModel,
                                'video',
                              )}
                              onChange={(checked) =>
                                handleOptionalFieldToggle('video', checked)
                              }
                            />
                          </Space>
                        </div>
                        {!isOptionalFieldEnabled(selectedModel, 'video') ? (
                          <div className='mt-1 text-xs text-gray-500'>
                            {t('当前未启用，需要时再打开即可。')}
                          </div>
                        ) : (
                          <div
                            style={{
                              marginTop: 8,
                              padding: 12,
                              background: 'var(--semi-color-fill-1)',
                              borderRadius: 6,
                            }}
                          >
                            {videoEditMode === 'json' ? (
                              <JsonCodeEditor
                                value={videoJsonDraft}
                                onChange={setVideoJsonDraft}
                                minRows={22}
                                maxRows={36}
                                placeholder={VIDEO_PRICING_JSON_PLACEHOLDER}
                              />
                            ) : (
                              <>
                            <div className='mb-2 text-xs text-gray-600'>
                              {t('计费模式')}
                            </div>
                            <div
                              style={{
                                marginBottom: 12,
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'space-between',
                                gap: 8,
                              }}
                            >
                              <RadioGroup
                                type='button'
                                value={selectedModel.videoBillingMode}
                                onChange={(event) =>
                                  handleVideoBillingModeChange(
                                    event.target.value,
                                  )
                                }
                              >
                                <Radio value='per-second'>
                                  {t('按视频秒收费')}
                                </Radio>
                                <Radio value='per-item'>
                                  {t('按视频条数收费')}
                                </Radio>
                                <Radio value='per-token'>
                                  {t('按 token 收费')}
                                </Radio>
                              </RadioGroup>
                              <Select
                                value={selectedModel.videoPriceUnit || 'USD'}
                                style={{ width: 170 }}
                                onChange={(value) =>
                                  handleVideoPriceUnitChange(String(value))
                                }
                                optionList={[
                                  { label: 'USD ($)', value: 'USD' },
                                  { label: 'CNY (¥)', value: 'CNY' },
                                  {
                                    label: `${t('自定义')} (${getCurrencyConfig().symbol || '¤'})`,
                                    value: 'CUSTOM',
                                  },
                                ]}
                              />
                            </div>
                            {(selectedModel.videoBillingMode === 'per-second' ||
                              selectedModel.videoBillingMode === 'per-token') ? (
                              <>
                                <div className='mb-2 text-xs text-gray-600'>
                                  {selectedModel.videoBillingMode === 'per-token'
                                    ? t(
                                        '按上游 total_tokens ÷ 1M × 分辨率单价（/1M tokens）计费；可按文生/图生/视频生 + 分辨率配置价格。',
                                      )
                                    : t(
                                        '按真实秒数向上取整计费；可按文生/图生/视频生 + 分辨率配置每秒价格。',
                                      )}
                                </div>

                                <div className='mb-2 font-medium text-gray-700'>
                                  {t('文生视频')}
                                </div>
                                {(
                                  selectedModel.videoTextToVideoRules || []
                                ).map((row, index, arr) => (
                                  <div
                                    key={`text-rule-${index}`}
                                    style={{
                                      ...VIDEO_RULE_CARD_STYLE,
                                      marginBottom:
                                        index < arr.length - 1 ? 10 : 0,
                                      display: 'flex',
                                      alignItems: 'center',
                                      flexWrap: 'wrap',
                                      gap: 8,
                                    }}
                                  >
                                    <div
                                      style={{
                                        ...VIDEO_RULE_HEADER_ROW_STYLE,
                                        flex: '0 1 auto',
                                        justifyContent: 'flex-start',
                                      }}
                                    >
                                      <Select
                                        value={row.resolution}
                                        placeholder={t('选择分辨率')}
                                        filter
                                        style={{ width: 140 }}
                                        optionList={getSelectableResolutionOptions(
                                          selectedModel.videoTextToVideoRules,
                                          index,
                                        )}
                                        onChange={(value) =>
                                          updateVideoRuleRow(
                                            'text',
                                            index,
                                            'resolution',
                                            String(value || ''),
                                          )
                                        }
                                      />
                                      <div className='flex items-center gap-2'>
                                        <Switch
                                          size='small'
                                          checked={Boolean(
                                            row.audioPricingEnabled,
                                          )}
                                          checkedText={t('开')}
                                          uncheckedText={t('关')}
                                          onChange={(checked) =>
                                            updateVideoRuleRow(
                                              'text',
                                              index,
                                              'audioPricingEnabled',
                                              checked,
                                            )
                                          }
                                        />
                                        <Tag
                                          size='small'
                                          color={
                                            row.audioPricingEnabled
                                              ? 'blue'
                                              : 'grey'
                                          }
                                        >
                                          {row.audioPricingEnabled
                                            ? t('音轨计费')
                                            : t('统一计费')}
                                        </Tag>
                                      </div>
                                      <Button
                                        type='danger'
                                        icon={<IconDelete />}
                                        onClick={() =>
                                          removeVideoRuleRow('text', index)
                                        }
                                      />
                                    </div>
                                    <div
                                      style={{
                                        ...VIDEO_RULE_INPUT_ROW_STYLE,
                                        marginTop: 0,
                                        flex: '0 1 auto',
                                      }}
                                    >
                                      {row.audioPricingEnabled ? (
                                        <>
                                          <Input
                                            value={row.noAudioPrice}
                                            placeholder={t('无音轨价格')}
                                            suffix={perVideoPriceSuffix}
                                            style={{ width: 180 }}
                                            onChange={(value) =>
                                              updateVideoRuleRow(
                                                'text',
                                                index,
                                                'noAudioPrice',
                                                value,
                                              )
                                            }
                                          />
                                          <Input
                                            value={row.withAudioPrice}
                                            placeholder={t('有音轨价格')}
                                            suffix={perVideoPriceSuffix}
                                            style={{ width: 180 }}
                                            onChange={(value) =>
                                              updateVideoRuleRow(
                                                'text',
                                                index,
                                                'withAudioPrice',
                                                value,
                                              )
                                            }
                                          />
                                        </>
                                      ) : (
                                        <Input
                                          value={row.tokenPrice}
                                          placeholder={t('统一价格')}
                                          suffix={perVideoPriceSuffix}
                                          style={{ width: 180 }}
                                          onChange={(value) =>
                                            updateVideoRuleRow(
                                              'text',
                                              index,
                                              'tokenPrice',
                                              value,
                                            )
                                          }
                                        />
                                      )}
                                    </div>
                                  </div>
                                ))}
                                <Button
                                  theme='borderless'
                                  icon={<IconPlus />}
                                  onClick={() => addVideoRuleRow('text')}
                                  style={{ marginBottom: 8 }}
                                >
                                  {t('新增文生视频规则')}
                                </Button>

                                <div className='mb-2 font-medium text-gray-700'>
                                  {t('图生视频价格')}
                                </div>
                                {(
                                  selectedModel.videoImageToVideoRules || []
                                ).map((row, index, arr) => (
                                  <div
                                    key={`image-rule-${index}`}
                                    style={{
                                      ...VIDEO_RULE_CARD_STYLE,
                                      marginBottom:
                                        index < arr.length - 1 ? 10 : 0,
                                      display: 'flex',
                                      alignItems: 'center',
                                      flexWrap: 'wrap',
                                      gap: 8,
                                    }}
                                  >
                                    <div
                                      style={{
                                        ...VIDEO_RULE_HEADER_ROW_STYLE,
                                        flex: '0 1 auto',
                                        justifyContent: 'flex-start',
                                      }}
                                    >
                                      <Select
                                        value={row.resolution}
                                        placeholder={t('选择分辨率')}
                                        filter
                                        style={{ width: 140 }}
                                        optionList={getSelectableResolutionOptions(
                                          selectedModel.videoImageToVideoRules,
                                          index,
                                        )}
                                        onChange={(value) =>
                                          updateVideoRuleRow(
                                            'image',
                                            index,
                                            'resolution',
                                            String(value || ''),
                                          )
                                        }
                                      />
                                      <div className='flex items-center gap-2'>
                                        <Switch
                                          size='small'
                                          checked={Boolean(
                                            row.audioPricingEnabled,
                                          )}
                                          checkedText={t('开')}
                                          uncheckedText={t('关')}
                                          onChange={(checked) =>
                                            updateVideoRuleRow(
                                              'image',
                                              index,
                                              'audioPricingEnabled',
                                              checked,
                                            )
                                          }
                                        />
                                        <Tag
                                          size='small'
                                          color={
                                            row.audioPricingEnabled
                                              ? 'blue'
                                              : 'grey'
                                          }
                                        >
                                          {row.audioPricingEnabled
                                            ? t('音轨计费')
                                            : t('统一计费')}
                                        </Tag>
                                      </div>
                                      <Button
                                        type='danger'
                                        icon={<IconDelete />}
                                        onClick={() =>
                                          removeVideoRuleRow('image', index)
                                        }
                                      />
                                    </div>
                                    <div
                                      style={{
                                        ...VIDEO_RULE_INPUT_ROW_STYLE,
                                        marginTop: 0,
                                        flex: '0 1 auto',
                                      }}
                                    >
                                      {row.audioPricingEnabled ? (
                                        <>
                                          <Input
                                            value={row.noAudioPrice}
                                            placeholder={t('无音轨价格')}
                                            suffix={perVideoPriceSuffix}
                                            style={{ width: 180 }}
                                            onChange={(value) =>
                                              updateVideoRuleRow(
                                                'image',
                                                index,
                                                'noAudioPrice',
                                                value,
                                              )
                                            }
                                          />
                                          <Input
                                            value={row.withAudioPrice}
                                            placeholder={t('有音轨价格')}
                                            suffix={perVideoPriceSuffix}
                                            style={{ width: 180 }}
                                            onChange={(value) =>
                                              updateVideoRuleRow(
                                                'image',
                                                index,
                                                'withAudioPrice',
                                                value,
                                              )
                                            }
                                          />
                                        </>
                                      ) : (
                                        <Input
                                          value={row.tokenPrice}
                                          placeholder={t('统一价格')}
                                          suffix={perVideoPriceSuffix}
                                          style={{ width: 180 }}
                                          onChange={(value) =>
                                            updateVideoRuleRow(
                                              'image',
                                              index,
                                              'tokenPrice',
                                              value,
                                            )
                                          }
                                        />
                                      )}
                                    </div>
                                  </div>
                                ))}
                                <Button
                                  theme='borderless'
                                  icon={<IconPlus />}
                                  onClick={() => addVideoRuleRow('image')}
                                  style={{ marginBottom: 8 }}
                                >
                                  {t('新增图生视频规则')}
                                </Button>

                                <div className='mb-2 font-medium text-gray-700'>
                                  {t('视频生成视频价格')}
                                </div>
                                {(selectedModel.videoGenerateRules || []).map(
                                  (row, index, arr) => (
                                    <div
                                      key={`video-generate-rule-${index}`}
                                      style={{
                                        ...VIDEO_RULE_CARD_STYLE,
                                        marginBottom:
                                          index < arr.length - 1 ? 10 : 0,
                                        display: 'flex',
                                        alignItems: 'center',
                                        flexWrap: 'wrap',
                                        gap: 8,
                                      }}
                                    >
                                      <div
                                        style={{
                                          ...VIDEO_RULE_HEADER_ROW_STYLE,
                                          flex: '0 1 auto',
                                          justifyContent: 'flex-start',
                                        }}
                                      >
                                        <Select
                                          value={row.resolution}
                                          placeholder={t('选择分辨率')}
                                          filter
                                          style={{ width: 140 }}
                                          optionList={getSelectableResolutionOptions(
                                            selectedModel.videoGenerateRules,
                                            index,
                                          )}
                                          onChange={(value) =>
                                            updateVideoRuleRow(
                                              'videoGenerate',
                                              index,
                                              'resolution',
                                              String(value || ''),
                                            )
                                          }
                                        />
                                        <div className='flex items-center gap-2'>
                                          <Switch
                                            size='small'
                                            checked={Boolean(
                                              row.audioPricingEnabled,
                                            )}
                                            checkedText={t('开')}
                                            uncheckedText={t('关')}
                                            onChange={(checked) =>
                                              updateVideoRuleRow(
                                                'videoGenerate',
                                                index,
                                                'audioPricingEnabled',
                                                checked,
                                              )
                                            }
                                          />
                                          <Tag
                                            size='small'
                                            color={
                                              row.audioPricingEnabled
                                                ? 'blue'
                                                : 'grey'
                                            }
                                          >
                                            {row.audioPricingEnabled
                                              ? t('音轨计费')
                                              : t('统一计费')}
                                          </Tag>
                                        </div>
                                        <Button
                                          type='danger'
                                          icon={<IconDelete />}
                                          onClick={() =>
                                            removeVideoRuleRow(
                                              'videoGenerate',
                                              index,
                                            )
                                          }
                                        />
                                      </div>
                                      <div
                                        style={{
                                          ...VIDEO_RULE_INPUT_ROW_STYLE,
                                          marginTop: 0,
                                          flex: '0 1 auto',
                                        }}
                                      >
                                        {row.audioPricingEnabled ? (
                                          <>
                                            <Input
                                              value={row.noAudioPrice}
                                              placeholder={t('无音轨价格')}
                                              suffix={perVideoPriceSuffix}
                                              style={{ width: 180 }}
                                              onChange={(value) =>
                                                updateVideoRuleRow(
                                                  'videoGenerate',
                                                  index,
                                                  'noAudioPrice',
                                                  value,
                                                )
                                              }
                                            />
                                            <Input
                                              value={row.withAudioPrice}
                                              placeholder={t('有音轨价格')}
                                              suffix={perVideoPriceSuffix}
                                              style={{ width: 180 }}
                                              onChange={(value) =>
                                                updateVideoRuleRow(
                                                  'videoGenerate',
                                                  index,
                                                  'withAudioPrice',
                                                  value,
                                                )
                                              }
                                            />
                                          </>
                                        ) : (
                                          <Input
                                            value={row.tokenPrice}
                                            placeholder={t('统一价格')}
                                            suffix={perVideoPriceSuffix}
                                            style={{ width: 180 }}
                                            onChange={(value) =>
                                              updateVideoRuleRow(
                                                'videoGenerate',
                                                index,
                                                'tokenPrice',
                                                value,
                                              )
                                            }
                                          />
                                        )}
                                      </div>
                                    </div>
                                  ),
                                )}
                                <Button
                                  theme='borderless'
                                  icon={<IconPlus />}
                                  onClick={() =>
                                    addVideoRuleRow('videoGenerate')
                                  }
                                  style={{ marginBottom: 8 }}
                                >
                                  {t('新增生成视频规则')}
                                </Button>

                                <PriceInput
                                  label={t('相似分辨率阈值')}
                                  value={selectedModel.videoSimilarityThreshold}
                                  placeholder={t('默认 0.35')}
                                  onChange={(value) =>
                                    handleNumericFieldChange(
                                      'videoSimilarityThreshold',
                                      value,
                                    )
                                  }
                                  suffix={t('比例')}
                                  extraText={t(
                                    '上传视频与预设分辨率差异在阈值内按相似规则处理，差异过大按实际分辨率处理。',
                                  )}
                                />
                              </>
                            ) : (
                              <>
                                <div className='mb-2 text-xs text-gray-600'>
                                  {videoPerVideoBillingHint}
                                </div>
                                {[
                                  [
                                    'text',
                                    t('文生视频（多分辨率规则）'),
                                    'videoTextToVideoRules',
                                    t('新增文生视频规则'),
                                  ],
                                  [
                                    'image',
                                    t('图生视频价格'),
                                    'videoImageToVideoRules',
                                    t('新增图生视频规则'),
                                  ],
                                  [
                                    'videoGenerate',
                                    t('视频生成视频价格'),
                                    'videoGenerateRules',
                                    t('新增生成视频规则'),
                                  ],
                                ].map(([section, title, prop, addLabel]) => (
                                  <React.Fragment key={`pv-${section}`}>
                                    <div className='mb-2 font-medium text-gray-700'>
                                      {title}
                                    </div>
                                    {(selectedModel[prop] || []).map(
                                      (row, index, arr) => (
                                        <div
                                          key={`${section}-pv-row-${index}`}
                                          style={{
                                            ...VIDEO_RULE_CARD_STYLE,
                                            marginBottom:
                                              index < arr.length - 1 ? 10 : 0,
                                            display: 'flex',
                                            alignItems: 'center',
                                            flexWrap: 'wrap',
                                            gap: 8,
                                          }}
                                        >
                                          <div
                                            style={{
                                              ...VIDEO_RULE_HEADER_ROW_STYLE,
                                              flex: '0 1 auto',
                                              justifyContent: 'flex-start',
                                            }}
                                          >
                                            <Select
                                              value={row.resolution}
                                              placeholder={t('选择分辨率')}
                                              filter
                                              style={{ width: 140 }}
                                              optionList={getSelectableResolutionOptions(
                                                selectedModel[prop] || [],
                                                index,
                                              )}
                                              onChange={(value) =>
                                                updateVideoRuleRow(
                                                  section,
                                                  index,
                                                  'resolution',
                                                  String(value || ''),
                                                )
                                              }
                                            />
                                            <div className='flex items-center gap-2'>
                                              <Switch
                                                size='small'
                                                checked={Boolean(
                                                  row.audioPricingEnabled,
                                                )}
                                                checkedText={t('开')}
                                                uncheckedText={t('关')}
                                                onChange={(checked) =>
                                                  updateVideoRuleRow(
                                                    section,
                                                    index,
                                                    'audioPricingEnabled',
                                                    checked,
                                                  )
                                                }
                                              />
                                              <Tag
                                                size='small'
                                                color={
                                                  row.audioPricingEnabled
                                                    ? 'blue'
                                                    : 'grey'
                                                }
                                              >
                                                {row.audioPricingEnabled
                                                  ? t('音轨计费')
                                                  : t('统一计费')}
                                              </Tag>
                                            </div>
                                            <Button
                                              type='danger'
                                              icon={<IconDelete />}
                                              onClick={() =>
                                                removeVideoRuleRow(
                                                  section,
                                                  index,
                                                )
                                              }
                                            />
                                          </div>
                                          <div
                                            style={{
                                              ...VIDEO_RULE_INPUT_ROW_STYLE,
                                              marginTop: 0,
                                              flex: '0 1 auto',
                                            }}
                                          >
                                            {row.audioPricingEnabled ? (
                                              <>
                                                <Input
                                                  value={row.noAudioPrice}
                                                  placeholder={t('无音轨价格')}
                                                  suffix={perVideoPriceSuffix}
                                                  style={{ width: 180 }}
                                                  onChange={(value) =>
                                                    updateVideoRuleRow(
                                                      section,
                                                      index,
                                                      'noAudioPrice',
                                                      value,
                                                    )
                                                  }
                                                />
                                                <Input
                                                  value={row.withAudioPrice}
                                                  placeholder={t('有音轨价格')}
                                                  suffix={perVideoPriceSuffix}
                                                  style={{ width: 180 }}
                                                  onChange={(value) =>
                                                    updateVideoRuleRow(
                                                      section,
                                                      index,
                                                      'withAudioPrice',
                                                      value,
                                                    )
                                                  }
                                                />
                                              </>
                                            ) : (
                                              <Input
                                                value={row.videoPrice}
                                                placeholder={t('统一价格')}
                                                suffix={perVideoPriceSuffix}
                                                style={{ width: 180 }}
                                                onChange={(value) =>
                                                  updateVideoRuleRow(
                                                    section,
                                                    index,
                                                    'videoPrice',
                                                    value,
                                                  )
                                                }
                                              />
                                            )}
                                          </div>
                                        </div>
                                      ),
                                    )}
                                    <Button
                                      theme='borderless'
                                      icon={<IconPlus />}
                                      onClick={() => addVideoRuleRow(section)}
                                      style={{ marginBottom: 8 }}
                                    >
                                      {addLabel}
                                    </Button>
                                  </React.Fragment>
                                ))}
                                <div style={{ marginTop: 8 }}>
                                  <PriceInput
                                    label={t('无分辨率表时的单视频价')}
                                    value={selectedModel.videoFixedPrice}
                                    placeholder={t('输入每个视频价格')}
                                    suffix={flatPerVideoPriceSuffix}
                                    onChange={(value) =>
                                      handleNumericFieldChange(
                                        'videoFixedPrice',
                                        value,
                                      )
                                    }
                                    extraText={t(
                                      '适用于供应商按视频条数计费的场景，例如部分视频生成模型。',
                                    )}
                                  />
                                </div>
                              </>
                            )}
                              </>
                            )}
                          </div>
                        )}
                      </div>
                    </Card>
                    {isOptionalFieldEnabled(selectedModel, 'video') ? (
                    <Card
                      bodyStyle={{ padding: 16 }}
                      style={{
                        marginBottom: 16,
                        background: 'var(--semi-color-fill-0)',
                      }}
                    >
                      <div className='mb-2 font-medium text-gray-700'>
                        {t('视频超分（按秒）')}
                      </div>
                      <div className='mb-2 text-xs text-gray-600'>
                        {t(
                          '按「原分辨率 → 超分分辨率」配置每秒单价，表示从原分辨率转到该超分分辨率。任务命中渠道超分规则且超分成功时，在原视频计费上叠加：向上取整秒数 × 本表单价。未配置对应档位则不收超分费。',
                        )}
                      </div>
                      {(selectedModel.videoUpscaleRules || []).length > 0 ? (
                        <div
                          className='mb-2 text-xs text-gray-500'
                          style={{
                            display: 'grid',
                            gridTemplateColumns: '140px 140px 180px 64px',
                            gap: 8,
                            padding: '0 12px',
                          }}
                        >
                          <span>{t('超分分辨率')}</span>
                          <span>{t('原分辨率')}</span>
                          <span>{t('价格')}</span>
                          <span />
                        </div>
                      ) : null}
                      {(selectedModel.videoUpscaleRules || []).map(
                        (row, index, arr) => (
                          <div
                            key={`upscale-rule-${index}`}
                            style={{
                              ...VIDEO_RULE_CARD_STYLE,
                              marginBottom: index < arr.length - 1 ? 10 : 0,
                              display: 'grid',
                              gridTemplateColumns: '140px 140px 180px 64px',
                              alignItems: 'center',
                              gap: 8,
                            }}
                          >
                            <Select
                              value={row.resolution}
                              placeholder={t('超分分辨率')}
                              optionList={VIDEO_RESOLUTION_OPTIONS.filter((item) =>
                                ['1280x720', '1920x1080', '2560x1440', '3840x2160'].includes(
                                  item.value,
                                ),
                              )}
                              style={{ width: 140 }}
                              onChange={(value) =>
                                updateVideoRuleRow(
                                  'upscale',
                                  index,
                                  'resolution',
                                  value,
                                )
                              }
                            />
                            <Select
                              value={row.sourceResolution}
                              placeholder={t('原分辨率')}
                              optionList={VIDEO_RESOLUTION_OPTIONS}
                              style={{ width: 140 }}
                              onChange={(value) =>
                                updateVideoRuleRow(
                                  'upscale',
                                  index,
                                  'sourceResolution',
                                  value,
                                )
                              }
                            />
                            <Input
                              value={row.tokenPrice}
                              placeholder={t('每秒价格')}
                              suffix={
                                selectedModel.videoPriceUnit === 'CNY'
                                  ? '¥/秒'
                                  : selectedModel.videoPriceUnit === 'CUSTOM'
                                    ? `${getCurrencyConfig().symbol || '¤'}/秒`
                                    : '$/秒'
                              }
                              style={{ width: 180 }}
                              onChange={(value) =>
                                updateVideoRuleRow(
                                  'upscale',
                                  index,
                                  'tokenPrice',
                                  value,
                                )
                              }
                            />
                            <Button
                              theme='borderless'
                              type='danger'
                              onClick={() =>
                                removeVideoRuleRow('upscale', index)
                              }
                            >
                              {t('删除')}
                            </Button>
                          </div>
                        ),
                      )}
                      <Button
                        theme='borderless'
                        icon={<IconPlus />}
                        onClick={() => addVideoRuleRow('upscale')}
                        style={{ marginTop: 8 }}
                      >
                        {t('添加超分价格')}
                      </Button>
                    </Card>
                    ) : null}
                    <Card
                      bodyStyle={{ padding: 16 }}
                      style={{
                        marginBottom: 16,
                        background: 'var(--semi-color-fill-0)',
                      }}
                    >
                      <div className='mb-1 font-medium text-gray-700 flex items-center justify-between gap-3 flex-wrap'>
                        <span className='flex items-center gap-1'>
                          {t('ASR 语音识别（按秒计费）')}
                          <Tooltip
                            position='top'
                            content={
                              <div style={{ maxWidth: 360 }}>
                                <div className='text-sm'>
                                  {t(
                                    '适用于阿里云 ASR 语音转写模型（同步/异步渠道），按识别音频时长（秒）× 每秒单价计费。',
                                  )}
                                </div>
                              </div>
                            }
                          >
                            <IconHelpCircle
                              style={{
                                cursor: 'help',
                                color: 'var(--semi-color-text-2)',
                              }}
                            />
                          </Tooltip>
                        </span>
                        <Switch
                          size='small'
                          checked={isOptionalFieldEnabled(
                            selectedModel,
                            'asrSecondPrice',
                          )}
                          onChange={(checked) =>
                            handleOptionalFieldToggle('asrSecondPrice', checked)
                          }
                        />
                      </div>
                      {!isOptionalFieldEnabled(selectedModel, 'asrSecondPrice') ? (
                        <div className='mt-1 text-xs text-gray-500'>
                          {t('当前未启用，需要时再打开即可。')}
                        </div>
                      ) : (
                        <div
                          style={{
                            marginTop: 8,
                            padding: 12,
                            background: 'var(--semi-color-fill-1)',
                            borderRadius: 6,
                          }}
                        >
                          <PriceInput
                            label={t('每秒价格')}
                            value={selectedModel.asrSecondPrice}
                            placeholder={t('输入每秒价格（USD）')}
                            suffix='$ / 秒'
                            onChange={(value) =>
                              handleNumericFieldChange('asrSecondPrice', value)
                            }
                            extraText={t(
                              '按识别音频时长（秒）计费；同步与异步 ASR 渠道共用该单价。',
                            )}
                          />
                        </div>
                      )}
                    </Card>
                  </>
                ) : null}

                {selectedModel.billingMode === 'tiered' ? (
                  <>
                    <Space wrap>
                      {hasTierPricing(selectedModel?.tierPricing) ? (
                        <Button
                          size='small'
                          type='danger'
                          onClick={() => clearAllTierRatios()}
                        >
                          {t('清除阶梯计费')}
                        </Button>
                      ) : null}
                      <Select
                        size='small'
                        value={
                          selectedModel?.tierPricing?.boundary ||
                          TIER_BOUNDARY_LT
                        }
                        optionList={[
                          {
                            label: t('边界：不含上限 (<)'),
                            value: TIER_BOUNDARY_LT,
                          },
                          {
                            label: t('边界：含上限 (≤)'),
                            value: TIER_BOUNDARY_LTE,
                          },
                        ]}
                        style={{ minWidth: 180 }}
                        onChange={(boundary) =>
                          updateTierPricing({
                            ...(selectedModel?.tierPricing ||
                              emptyTierPricing()),
                            boundary,
                          })
                        }
                      />
                      <Select
                        size='small'
                        value={selectedModel?.tierPricing?.currency || 'USD'}
                        optionList={CURRENCY_OPTIONS.map((c) => ({
                          label: c.label,
                          value: c.key,
                        }))}
                        style={{ minWidth: 140 }}
                        onChange={(currency) =>
                          handleTierCurrencyChange(currency)
                        }
                      />
                    </Space>
                    <div className='w-full rounded-md bg-[var(--semi-color-fill-0)] p-3 text-xs mt-4'>
                      <div className='mb-2 font-medium text-[var(--semi-color-text-0)]'>
                        {t('阶梯计费价格明细')}
                      </div>
                      {tierPriceDetails.length > 0 ? (
                        <div className='grid grid-cols-1 gap-y-2'>
                          {tierPriceDetails.map((detail) => (
                            <div key={detail.key} className='flex items-start'>
                              <span className='w-20 flex-shrink-0 text-[var(--semi-color-text-1)]'>
                                {detail.label}
                              </span>
                              <span className='flex-1 font-medium text-[var(--semi-color-text-0)] flex flex-col gap-1'>
                                {detail.segments.map((segment) => (
                                  <span
                                    key={`${segment.range}-${segment.price}`}
                                  >
                                    {segment.range}：{segment.price} / 1M tokens
                                  </span>
                                ))}
                              </span>
                            </div>
                          ))}
                        </div>
                      ) : (
                        <Text type='secondary'>
                          {t('未配置阶梯计费规则，请在下方添加档位。')}
                        </Text>
                      )}
                    </div>
                    <div className='my-4 text-xs text-gray-500'>
                      {t(
                        '阶梯区间从 0 开始；最后一档固定为无限且不能删除。每个档位统一配置 4 项价格（基准货币 /1M tokens）。边界可选不含上限(<)或含上限(≤)。使用时按系统货币汇率换算；基准货币与系统货币一致时不换算。',
                      )}
                    </div>
                    <Card
                      style={{
                        width: '100%',
                        marginBottom: 8,
                        background: 'var(--semi-color-fill-0)',
                      }}
                    >
                      <TierRowsEditor
                        t={t}
                        value={selectedModel.tierPricing?.tiers || []}
                        onChange={(tiers) =>
                          updateTierPricing({
                            ...(selectedModel.tierPricing || emptyTierPricing()),
                            tiers,
                          })
                        }
                        currency={
                          selectedModel.tierPricing?.currency || 'USD'
                        }
                      />
                    </Card>
                  </>
                ) : null}

                {selectedModel.billingMode === 'per-request' ? (
                  <PriceInput
                    label={t('固定价格')}
                    value={selectedModel.fixedPrice}
                    placeholder={t('输入每次调用价格')}
                    suffix={t('$/次')}
                    onChange={(value) =>
                      handleNumericFieldChange('fixedPrice', value)
                    }
                    extraText={t('适合 MJ / 任务类等按次收费模型。')}
                  />
                ) : null}

                <Card
                  bodyStyle={{ padding: 16 }}
                  style={{ background: 'var(--semi-color-fill-0)' }}
                >
                  <div className='font-medium mb-3'>{t('保存预览')}</div>
                  <div className='text-xs text-gray-500 mb-3'>
                    {t(
                      '下面展示这个模型保存后会写入哪些后端字段，便于和原始 JSON 编辑框保持一致。',
                    )}
                  </div>
                  <Space vertical align='start' style={{ width: '100%' }}>
                    {previewRows.map((group) => (
                      <div key={group.key} style={{ width: '100%' }}>
                        <div className='font-medium'>{group.title}</div>
                        {group.description ? (
                          <div className='text-xs text-gray-500 mb-2'>
                            {group.description}
                          </div>
                        ) : null}
                        <div
                          style={{
                            display: 'grid',
                            gridTemplateColumns: 'minmax(140px, 180px) 1fr',
                            gap: 8,
                          }}
                        >
                          {group.rows.map((row) => (
                            <React.Fragment key={`${group.key}-${row.key}`}>
                              <Text strong>{row.label}</Text>
                              <Text>{row.value}</Text>
                            </React.Fragment>
                          ))}
                        </div>
                      </div>
                    ))}
                  </Space>
                </Card>
              </div>
            )}
          </Card>
        </div>
      </Space>

      {allowAddModel ? (
        <Modal
          title={t('添加模型')}
          visible={addVisible}
          onCancel={() => {
            setAddVisible(false);
            setNewModelName('');
          }}
          onOk={handleAddModel}
        >
          <Input
            value={newModelName}
            placeholder={t('输入模型名称，例如 gpt-4.1')}
            onChange={(value) => setNewModelName(value)}
          />
        </Modal>
      ) : null}

      <Modal
        title={t('批量应用当前模型价格')}
        visible={batchVisible}
        onCancel={() => setBatchVisible(false)}
        onOk={() => {
          if (applySelectedModelPricing()) {
            setBatchVisible(false);
          }
        }}
      >
        <div className='text-sm text-gray-600'>
          {selectedModel
            ? t(
                '将把当前编辑中的模型 {{name}} 的价格配置，批量应用到已勾选的 {{count}} 个模型。',
                {
                  name: selectedModel.name,
                  count: selectedModelNames.length,
                },
              )
            : t('请先选择一个作为模板的模型')}
        </div>
        {selectedModel ? (
          <div className='text-xs text-gray-500 mt-3'>
            {t(
              '适合同系列模型一起定价，例如把 gpt-5.1 的价格批量同步到 gpt-5.1-high、gpt-5.1-low 等模型。',
            )}
          </div>
        ) : null}
      </Modal>
    </>
  );
}
