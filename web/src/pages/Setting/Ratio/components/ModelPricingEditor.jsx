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

import React, { useMemo, useState } from 'react';
import {
  Banner,
  Button,
  Card,
  Checkbox,
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
  IconSave,
  IconSearch,
} from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import {
  PAGE_SIZE,
  PRICE_SUFFIX,
  buildSummaryText,
  hasValue,
  useModelPricingEditorState,
} from '../hooks/useModelPricingEditorState';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import { getCurrencyConfig } from '../../../../helpers';

const { Text } = Typography;
const EMPTY_CANDIDATE_MODEL_NAMES = [];
const VIDEO_RESOLUTION_OPTIONS = [
  { label: '480p (854x480)', value: '854x480' },
  { label: '540p (960x540)', value: '960x540' },
  { label: '720p (1280x720)', value: '1280x720' },
  { label: '1080p (1920x1080)', value: '1920x1080' },
  { label: '2K (2560x1440)', value: '2560x1440' },
  { label: '4K (3840x2160)', value: '3840x2160' },
];
const VIDEO_RESOLUTION_LABEL_MAP = VIDEO_RESOLUTION_OPTIONS.reduce(
  (acc, item) => ({ ...acc, [item.value]: item.label }),
  {},
);
const DEFAULT_VIDEO_FPS = 24;

const getSelectableResolutionOptions = (rows, currentIndex) => {
  const used = new Set(
    (rows || [])
      .map((item, index) => (index === currentIndex ? '' : item?.resolution || ''))
      .filter(Boolean),
  );
  return VIDEO_RESOLUTION_OPTIONS.filter((item) => !used.has(item.value));
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
        preferredRank: preferredIndex === -1 ? Number.MAX_SAFE_INTEGER : preferredIndex,
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
  showConflictFilter = true,
  listDescription = '',
  emptyTitle = '',
  emptyDescription = '',
}) {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const [addVisible, setAddVisible] = useState(false);
  const [batchVisible, setBatchVisible] = useState(false);
  const [newModelName, setNewModelName] = useState('');

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
    conflictOnly,
    setConflictOnly,
    filteredModels,
    pagedData,
    selectedWarnings,
    previewRows,
    isOptionalFieldEnabled,
    handleOptionalFieldToggle,
    handleNumericFieldChange,
    handleBillingModeChange,
    handleVideoBillingModeChange,
    updateVideoRuleRow,
    addVideoRuleRow,
    removeVideoRuleRow,
    handleSubmit,
    addModel,
    deleteModel,
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
  });

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
    const { type, symbol } = getCurrencyConfig();
    if (type === 'TOKENS') {
      return t('标价（每条约）');
    }
    if (type === 'USD') {
      return t('$/条');
    }
    return `${symbol}${t('按条后缀')}`;
  }, [t]);

  const flatPerVideoPriceSuffix = useMemo(() => {
    const { type, symbol } = getCurrencyConfig();
    if (type === 'TOKENS' || type === 'USD') {
      return t('$/视频');
    }
    return `${symbol}${t('每视频')}`;
  }, [t]);

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
            {selectedModelNames.includes(record.name) ? (
              <Tag color='green' shape='circle'>
                {t('已勾选')}
              </Tag>
            ) : null}
            {record.hasConflict ? (
              <Tag color='red' shape='circle'>
                {t('矛盾')}
              </Tag>
            ) : null}
          </Space>
        ),
      },
      {
        title: t('计费方式'),
        dataIndex: 'billingMode',
        key: 'billingMode',
        render: (_, record) => (
          <Tag color={record.billingMode === 'per-request' ? 'teal' : 'violet'}>
            {record.billingMode === 'per-request'
              ? t('按次计费')
              : t('按量计费')}
          </Tag>
        ),
      },
      {
        title: t('价格摘要'),
        dataIndex: 'summary',
        key: 'summary',
        render: (_, record) => buildSummaryText(record, t),
      },
      {
        title: t('操作'),
        key: 'action',
        render: (_, record) => (
          <Space>
            {allowDeleteModel ? (
              <Button
                size='small'
                type='danger'
                icon={<IconDelete />}
                onClick={() => deleteModel(record.name)}
              />
            ) : null}
          </Space>
        ),
      },
    ],
    [
      allowDeleteModel,
      deleteModel,
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

  const rowSelection = {
    selectedRowKeys: selectedModelNames,
    onChange: (selectedRowKeys) => setSelectedModelNames(selectedRowKeys),
  };

  return (
    <>
      <Space vertical align='start' style={{ width: '100%' }}>
        <Space wrap className='mt-2'>
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
            onClick={handleSubmit}
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
          {showConflictFilter ? (
            <Checkbox
              checked={conflictOnly}
              onChange={(event) => setConflictOnly(event.target.checked)}
            >
              {t('仅显示矛盾倍率')}
            </Checkbox>
          ) : null}
        </Space>

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
                onRow={(record) => ({
                  style: {
                    background: selectedModelNames.includes(record.name)
                      ? 'var(--semi-color-success-light-default)'
                      : record.name === selectedModelName
                        ? 'var(--semi-color-primary-light-default)'
                        : undefined,
                    boxShadow: selectedModelNames.includes(record.name)
                      ? 'inset 4px 0 0 var(--semi-color-success)'
                      : record.name === selectedModelName
                        ? 'inset 4px 0 0 var(--semi-color-primary)'
                        : undefined,
                    transition: 'background 0.2s ease, box-shadow 0.2s ease',
                  },
                  onClick: () => setSelectedModelName(record.name),
                })}
                scroll={isMobile ? { x: 720 } : undefined}
              />
            </div>
          </Card>

          <Card
            style={isMobile ? { order: 1 } : undefined}
            title={selectedModel ? selectedModel.name : t('模型计费编辑器')}
            headerExtraContent={
              selectedModel ? (
                <Tag color='blue'>
                  {selectedModel.billingMode === 'per-request'
                    ? t('按次计费')
                    : t('按量计费')}
                </Tag>
              ) : null
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
                  <div className='mb-2 font-medium text-gray-700'>
                    {t('计费方式')}
                  </div>
                  <RadioGroup
                    type='button'
                    value={selectedModel.billingMode}
                    onChange={(event) =>
                      handleBillingModeChange(event.target.value)
                    }
                  >
                    <Radio value='per-token'>{t('按量计费')}</Radio>
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
                ) : (
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
                        <div className='mb-1 font-medium text-gray-700 flex items-center justify-between gap-3'>
                          <span className='flex items-center gap-1'>
                            {t('视频价格')}
                            <Tooltip
                              position='top'
                              content={
                                <div style={{ maxWidth: 320 }}>
                                  <div className='font-medium mb-1'>
                                    {t('视频 token 估算公式')}
                                  </div>
                                  <div>
                                    {t(
                                      '(输入视频时长 + 输出视频时长) × 输出视频宽 × 输出视频高 × 输出帧率 / 1024',
                                    )}
                                  </div>
                                  <div className='mt-2 text-xs'>
                                    {t(
                                      '上述 token 用量均为估算值；如供应商按视频条数计费，可切换到“按视频”模式。',
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
                              'video',
                            )}
                            onChange={(checked) =>
                              handleOptionalFieldToggle('video', checked)
                            }
                          />
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
                            <div className='mb-2 text-xs text-gray-600'>
                              {t('计费模式')}
                            </div>
                            <RadioGroup
                              type='button'
                              value={selectedModel.videoBillingMode}
                              onChange={(event) =>
                                handleVideoBillingModeChange(event.target.value)
                              }
                              style={{ marginBottom: 12 }}
                            >
                              <Radio value='per-token'>
                                {t('按 token 计费')}
                              </Radio>
                              <Radio value='per-video'>{t('按视频计费')}</Radio>
                            </RadioGroup>
                            {selectedModel.videoBillingMode === 'per-token' ? (
                              <>
                                <div className='mb-2 text-xs text-gray-600'>
                                  {t(
                                    '文生视频：宽×高×帧率×时长秒/压缩系数；图生视频：宽×高/压缩系数；视频生视频同文生视频。',
                                  )}
                                </div>

                                <div className='mb-3 font-medium text-gray-700'>
                                  {t('文生视频（多分辨率规则）')}
                                </div>
                                {(selectedModel.videoTextToVideoRules || []).map(
                                  (row, index) => (
                                    <div
                                      key={`text-rule-${index}`}
                                      style={{
                                        display: 'grid',
                                        gridTemplateColumns:
                                          'minmax(120px,1fr) minmax(120px,1fr) minmax(120px,1fr) auto',
                                        gap: 8,
                                        marginBottom: 8,
                                      }}
                                    >
                                      <Select
                                        value={row.resolution}
                                        placeholder={t('选择分辨率')}
                                        filter
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
                                      <Input
                                        value={row.tokenPrice}
                                        placeholder={t('token价格')}
                                        suffix={t('$/1M')}
                                        style={{ maxWidth: 150 }}
                                        onChange={(value) =>
                                          updateVideoRuleRow(
                                            'text',
                                            index,
                                            'tokenPrice',
                                            value,
                                          )
                                        }
                                      />
                                      <Input
                                        value={row.pixelCompression}
                                        placeholder={t('压缩系数')}
                                        style={{ maxWidth: 120 }}
                                        onChange={(value) =>
                                          updateVideoRuleRow(
                                            'text',
                                            index,
                                            'pixelCompression',
                                            value,
                                          )
                                        }
                                      />
                                      <Button
                                        type='danger'
                                        icon={<IconDelete />}
                                        onClick={() =>
                                          removeVideoRuleRow('text', index)
                                        }
                                      />
                                    </div>
                                  ),
                                )}
                                <Button
                                  theme='borderless'
                                  icon={<IconPlus />}
                                  onClick={() => addVideoRuleRow('text')}
                                  style={{ marginBottom: 12 }}
                                >
                                  {t('新增文生视频规则')}
                                </Button>

                                <div className='mb-3 font-medium text-gray-700'>
                                  {t('图生视频价格')}
                                </div>
                                {(selectedModel.videoImageToVideoRules || []).map(
                                  (row, index) => (
                                    <div
                                      key={`image-rule-${index}`}
                                      style={{
                                        display: 'grid',
                                        gridTemplateColumns:
                                          'minmax(120px,1fr) minmax(120px,1fr) minmax(120px,1fr) auto',
                                        gap: 8,
                                        marginBottom: 8,
                                      }}
                                    >
                                      <Select
                                        value={row.resolution}
                                        placeholder={t('选择分辨率')}
                                        filter
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
                                      <Input
                                        value={row.tokenPrice}
                                        placeholder={t('token价格')}
                                        suffix={t('$/1M')}
                                        style={{ maxWidth: 150 }}
                                        onChange={(value) =>
                                          updateVideoRuleRow(
                                            'image',
                                            index,
                                            'tokenPrice',
                                            value,
                                          )
                                        }
                                      />
                                      <Input
                                        value={row.pixelCompression}
                                        placeholder={t('压缩系数')}
                                        style={{ maxWidth: 120 }}
                                        onChange={(value) =>
                                          updateVideoRuleRow(
                                            'image',
                                            index,
                                            'pixelCompression',
                                            value,
                                          )
                                        }
                                      />
                                      <Button
                                        type='danger'
                                        icon={<IconDelete />}
                                        onClick={() =>
                                          removeVideoRuleRow('image', index)
                                        }
                                      />
                                    </div>
                                  ),
                                )}
                                <Button
                                  theme='borderless'
                                  icon={<IconPlus />}
                                  onClick={() => addVideoRuleRow('image')}
                                  style={{ marginBottom: 12 }}
                                >
                                  {t('新增图生视频规则')}
                                </Button>

                                <div className='mb-3 font-medium text-gray-700'>
                                  {t('视频生视频价格 - 上传视频价格')}
                                </div>
                                {(selectedModel.videoUploadRules || []).map((row, index) => (
                                  <div
                                    key={`video-upload-rule-${index}`}
                                    style={{
                                      display: 'grid',
                                      gridTemplateColumns:
                                        'minmax(120px,1fr) minmax(120px,1fr) minmax(120px,1fr) auto',
                                      gap: 8,
                                      marginBottom: 8,
                                    }}
                                  >
                                    <Select
                                      value={row.resolution}
                                      placeholder={t('选择分辨率')}
                                      filter
                                      optionList={getSelectableResolutionOptions(
                                        selectedModel.videoUploadRules,
                                        index,
                                      )}
                                      onChange={(value) =>
                                        updateVideoRuleRow(
                                          'videoUpload',
                                          index,
                                          'resolution',
                                          String(value || ''),
                                        )
                                      }
                                    />
                                    <Input
                                      value={row.tokenPrice}
                                      placeholder={t('价格')}
                                      suffix={t('$/1M')}
                                      style={{ maxWidth: 150 }}
                                      onChange={(value) =>
                                        updateVideoRuleRow(
                                          'videoUpload',
                                          index,
                                          'tokenPrice',
                                          value,
                                        )
                                      }
                                    />
                                    <Input
                                      value={row.pixelCompression}
                                      placeholder={t('压缩系数')}
                                      style={{ maxWidth: 120 }}
                                      onChange={(value) =>
                                        updateVideoRuleRow(
                                          'videoUpload',
                                          index,
                                          'pixelCompression',
                                          value,
                                        )
                                      }
                                    />
                                    <Button
                                      type='danger'
                                      icon={<IconDelete />}
                                      onClick={() =>
                                        removeVideoRuleRow('videoUpload', index)
                                      }
                                    />
                                  </div>
                                ))}
                                <Button
                                  theme='borderless'
                                  icon={<IconPlus />}
                                  onClick={() => addVideoRuleRow('videoUpload')}
                                  style={{ marginBottom: 12 }}
                                >
                                  {t('新增上传视频规则')}
                                </Button>

                                <div className='mb-3 font-medium text-gray-700'>
                                  {t('视频生视频价格 - 生成视频价格')}
                                </div>
                                {(selectedModel.videoGenerateRules || []).map((row, index) => (
                                  <div
                                    key={`video-generate-rule-${index}`}
                                    style={{
                                      display: 'grid',
                                      gridTemplateColumns:
                                        'minmax(120px,1fr) minmax(120px,1fr) minmax(120px,1fr) auto',
                                      gap: 8,
                                      marginBottom: 8,
                                    }}
                                  >
                                    <Select
                                      value={row.resolution}
                                      placeholder={t('选择分辨率')}
                                      filter
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
                                    <Input
                                      value={row.tokenPrice}
                                      placeholder={t('价格')}
                                      suffix={t('$/1M')}
                                      style={{ maxWidth: 150 }}
                                      onChange={(value) =>
                                        updateVideoRuleRow(
                                          'videoGenerate',
                                          index,
                                          'tokenPrice',
                                          value,
                                        )
                                      }
                                    />
                                    <Input
                                      value={row.pixelCompression}
                                      placeholder={t('压缩系数')}
                                      style={{ maxWidth: 120 }}
                                      onChange={(value) =>
                                        updateVideoRuleRow(
                                          'videoGenerate',
                                          index,
                                          'pixelCompression',
                                          value,
                                        )
                                      }
                                    />
                                    <Button
                                      type='danger'
                                      icon={<IconDelete />}
                                      onClick={() =>
                                        removeVideoRuleRow('videoGenerate', index)
                                      }
                                    />
                                  </div>
                                ))}
                                <Button
                                  theme='borderless'
                                  icon={<IconPlus />}
                                  onClick={() => addVideoRuleRow('videoGenerate')}
                                  style={{ marginBottom: 12 }}
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
                                <Card
                                  bodyStyle={{ padding: 12 }}
                                  style={{
                                    marginTop: 8,
                                    background: 'var(--semi-color-fill-0)',
                                  }}
                                >
                                  {(() => {
                                    const rulesFromAllSections = [
                                      ...(selectedModel.videoTextToVideoRules || []),
                                      ...(selectedModel.videoImageToVideoRules || []),
                                      ...(selectedModel.videoUploadRules || []),
                                      ...(selectedModel.videoGenerateRules || []),
                                    ];
                                    const configuredResolutions = Array.from(
                                      new Set(
                                        rulesFromAllSections
                                          .map((row) => row?.resolution)
                                          .filter(Boolean),
                                      ),
                                    );
                                    const optionOrder = VIDEO_RESOLUTION_OPTIONS.map(
                                      (item) => item.value,
                                    );
                                    const demoResolutions =
                                      configuredResolutions.length > 0
                                        ? [...configuredResolutions].sort(
                                            (a, b) =>
                                              (optionOrder.indexOf(a) === -1
                                                ? Number.MAX_SAFE_INTEGER
                                                : optionOrder.indexOf(a)) -
                                              (optionOrder.indexOf(b) === -1
                                                ? Number.MAX_SAFE_INTEGER
                                                : optionOrder.indexOf(b)),
                                          )
                                        : [pickDemoResolution(selectedModel)];
                                    const calcPerSecondSummary = (
                                      rule,
                                      width,
                                      height,
                                    ) => {
                                      if (!rule?.tokenPrice || !rule?.pixelCompression) {
                                        return { tokens: '-', price: '-' };
                                      }
                                      const tokenPrice = Number(rule.tokenPrice);
                                      const compression = Number(rule.pixelCompression);
                                      if (
                                        !Number.isFinite(tokenPrice) ||
                                        !Number.isFinite(compression) ||
                                        tokenPrice <= 0 ||
                                        compression <= 0
                                      ) {
                                        return { tokens: '-', price: '-' };
                                      }
                                      const tokensPerSecond =
                                        (width * height * DEFAULT_VIDEO_FPS) /
                                        compression;
                                      const pricePerSecond =
                                        (tokensPerSecond * tokenPrice) / 1000000;
                                      return {
                                        tokens: formatTokenNumber(tokensPerSecond),
                                        price: formatSystemCurrencyPrice(pricePerSecond),
                                      };
                                    };
                                    const calcImageSummary = (
                                      rule,
                                      width,
                                      height,
                                    ) => {
                                      const tokenPrice = Number(rule?.tokenPrice);
                                      const compression = Number(
                                        rule?.pixelCompression,
                                      );
                                      if (
                                        !Number.isFinite(tokenPrice) ||
                                        !Number.isFinite(compression) ||
                                        tokenPrice <= 0 ||
                                        compression <= 0
                                      ) {
                                        return { tokens: '-', price: '-' };
                                      }
                                      const tokens = (width * height) / compression;
                                      const price = (tokens * tokenPrice) / 1000000;
                                      return {
                                        tokens: formatTokenNumber(tokens),
                                        price: formatSystemCurrencyPrice(price),
                                      };
                                    };
                                    const renderPriceRow = (
                                      title,
                                      tokenLabel,
                                      tokenValue,
                                      priceLabel,
                                      priceValue,
                                    ) => (
                                      <div
                                        style={{
                                          marginTop: 8,
                                          padding: '8px 10px',
                                          borderRadius: 6,
                                          border:
                                            '1px solid var(--semi-color-border)',
                                          background:
                                            'var(--semi-color-fill-1)',
                                        }}
                                      >
                                        <div
                                          style={{
                                            fontWeight: 600,
                                            marginBottom: 6,
                                          }}
                                        >
                                          {title}
                                        </div>
                                        <div
                                          style={{
                                            display: 'flex',
                                            flexWrap: 'wrap',
                                            gap: 6,
                                          }}
                                        >
                                          <Tag color='blue' size='small'>
                                            {tokenLabel} {tokenValue}
                                          </Tag>
                                          <Tag color='green' size='small'>
                                            {priceLabel} {priceValue}
                                          </Tag>
                                        </div>
                                      </div>
                                    );
                                    return (
                                      <div className='text-xs text-gray-600'>
                                        {demoResolutions.map((resolution) => {
                                          const [w, h] = resolution
                                            .split('x')
                                            .map((v) => Number(v));
                                          const textRule = getRuleByResolution(
                                            selectedModel.videoTextToVideoRules,
                                            resolution,
                                          );
                                          const rawVideoUploadRule = getRuleByResolution(
                                            selectedModel.videoUploadRules,
                                            resolution,
                                          );
                                          const rawVideoGenerateRule = getRuleByResolution(
                                            selectedModel.videoGenerateRules,
                                            resolution,
                                          );
                                          const rawImageRule = getRuleByResolution(
                                            selectedModel.videoImageToVideoRules,
                                            resolution,
                                          );
                                          const videoUploadRule =
                                            rawVideoUploadRule || textRule;
                                          const videoGenerateRule =
                                            rawVideoGenerateRule || textRule;
                                          const imageRule = rawImageRule || textRule;
                                          const textPricingSummary =
                                            calcPerSecondSummary(textRule, w, h);
                                          const videoUploadPricingSummary =
                                            calcPerSecondSummary(videoUploadRule, w, h);
                                          const videoGeneratePricingSummary =
                                            calcPerSecondSummary(
                                              videoGenerateRule,
                                              w,
                                              h,
                                            );
                                          const imagePricingSummary = calcImageSummary(
                                            imageRule,
                                            w,
                                            h,
                                          );
                                          const finalImagePricingSummary =
                                            rawImageRule || !textRule
                                              ? imagePricingSummary
                                              : calcPerSecondSummary(textRule, w, h);
                                          return (
                                            <div
                                              key={`pricing-demo-${resolution}`}
                                              style={{ marginBottom: 10 }}
                                            >
                                              <div className='font-medium mb-1'>
                                                {t(
                                                  '{{resolution}} 价格示例（按当前分辨率价格表，约值）',
                                                  {
                                                    resolution:
                                                      VIDEO_RESOLUTION_LABEL_MAP[
                                                        resolution
                                                      ] || resolution,
                                                  },
                                                )}
                                              </div>
                                              {renderPriceRow(
                                                `${t('文生视频')} (${VIDEO_RESOLUTION_LABEL_MAP[resolution] || resolution})`,
                                                t('token 数'),
                                                textPricingSummary.tokens,
                                                t('价格'),
                                                textPricingSummary.price,
                                              )}
                                              {renderPriceRow(
                                                `${t('视频生视频-上传')} (${VIDEO_RESOLUTION_LABEL_MAP[resolution] || resolution})`,
                                                t('token 数'),
                                                videoUploadPricingSummary.tokens,
                                                t('价格'),
                                                videoUploadPricingSummary.price,
                                              )}
                                              {renderPriceRow(
                                                `${t('视频生视频-生成')} (${VIDEO_RESOLUTION_LABEL_MAP[resolution] || resolution})`,
                                                t('token 数'),
                                                videoGeneratePricingSummary.tokens,
                                                t('价格'),
                                                videoGeneratePricingSummary.price,
                                              )}
                                              {renderPriceRow(
                                                `${t('图生视频')} (${VIDEO_RESOLUTION_LABEL_MAP[resolution] || resolution})`,
                                                t('token 数'),
                                                finalImagePricingSummary.tokens,
                                                t('价格'),
                                                finalImagePricingSummary.price,
                                              )}
                                            </div>
                                          );
                                        })}
                                        <div style={{ marginTop: 4 }}>
                                          {t(
                                            '文/视频生视频示例按 24fps 与 1 秒计算：宽×高×帧率×时长/压缩系数。',
                                          )}
                                        </div>
                                        <div>
                                          {t(
                                            '图生视频示例按单张图片计算：宽×高/压缩系数 得到 token 数量，再按图片 token 价格换算。',
                                          )}
                                        </div>
                                      </div>
                                    );
                                  })()}
                                </Card>
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
                                      'videoUpload',
                                      t('视频生视频-上传'),
                                      'videoUploadRules',
                                      t('新增上传视频规则'),
                                  ],
                                  [
                                      'videoGenerate',
                                      t('视频生视频-生成'),
                                      'videoGenerateRules',
                                      t('新增生成视频规则'),
                                  ],
                                ].map(([section, title, prop, addLabel]) => (
                                  <React.Fragment key={`pv-${section}`}>
                                    <div className='mb-3 font-medium text-gray-700'>
                                      {title}
                                    </div>
                                    {(
                                      selectedModel[prop] || []
                                    ).map((row, index) => (
                                      <div
                                        key={`${section}-pv-row-${index}`}
                                        style={{
                                          display: 'grid',
                                          gridTemplateColumns:
                                            'minmax(120px,1fr) minmax(160px,1fr) auto',
                                          gap: 8,
                                          marginBottom: 8,
                                        }}
                                      >
                                        <Select
                                          value={row.resolution}
                                          placeholder={t('选择分辨率')}
                                          filter
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
                                        <Input
                                          value={row.videoPrice}
                                          placeholder={t('每条成片价格')}
                                          suffix={perVideoPriceSuffix}
                                          style={{ maxWidth: 180 }}
                                          onChange={(value) =>
                                            updateVideoRuleRow(
                                              section,
                                              index,
                                              'videoPrice',
                                              value,
                                            )
                                          }
                                        />
                                        <Button
                                          type='danger'
                                          icon={<IconDelete />}
                                          onClick={() =>
                                            removeVideoRuleRow(section, index)
                                          }
                                        />
                                      </div>
                                    ))}
                                    <Button
                                      theme='borderless'
                                      icon={<IconPlus />}
                                      onClick={() => addVideoRuleRow(section)}
                                      style={{ marginBottom: 12 }}
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
                          </div>
                        )}
                      </div>
                    </Card>
                  </>
                )}

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
                  <div
                    style={{
                      display: 'grid',
                      gridTemplateColumns: 'minmax(140px, 180px) 1fr',
                      gap: 8,
                    }}
                  >
                    {previewRows.map((row) => (
                      <React.Fragment key={row.key}>
                        <Text strong>{row.label}</Text>
                        <Text>{row.value}</Text>
                      </React.Fragment>
                    ))}
                  </div>
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
