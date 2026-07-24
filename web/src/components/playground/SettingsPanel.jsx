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

import React, { useEffect, useMemo, useRef, useState } from 'react';
import {
  Button,
  Card,
  Input,
  RadioGroup,
  Select,
  Slider,
  Switch,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import {
  Image as ImageIcon,
  Settings,
  Sliders,
  Sparkles,
  ToggleLeft,
  Wand2,
  X,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import {
  buildPlaygroundImageSizeOptions,
  buildPlaygroundVideoResolutionOptions,
  formatPlaygroundPixelSizeLabel,
  formatVideoResolutionDisplayLabel,
  getPlaygroundImageSizeForTier,
  getPlaygroundVideoSizeForTier,
  renderGroupOption,
  selectFilter,
} from '../../helpers';
import {
  PLAYGROUND_ASPECT_RATIO_OPTIONS,
  PLAYGROUND_VIDEO_DURATION_OPTIONS,
} from '../../constants/playground.constants';
import ConfigManager from './ConfigManager';
import CustomRequestEditor from './CustomRequestEditor';
import ImageUrlInput from './ImageUrlInput';
import MaterialLibraryButton from './MaterialLibraryButton';
import ParameterControl from './ParameterControl';
import VideoUrlInput from './VideoUrlInput';
import AudioUrlInput from './AudioUrlInput';

const SECTION_KEYS = {
  BASIC: 'basic',
  PARAMS: 'params',
  MEDIA: 'media',
  ADVANCED: 'advanced',
};

const renderPlaygroundGroupOption = (item) =>
  renderGroupOption({ ...item, ratio: undefined });

// 切换文本/图片/视频时默认回到模型分栏；自定义请求模式进入高级分栏。
const getDefaultSection = (displayMode, customRequestMode) => {
  if (customRequestMode) {
    return SECTION_KEYS.ADVANCED;
  }
  return SECTION_KEYS.BASIC;
};

const buildSectionTabs = (t) => [
  {
    key: SECTION_KEYS.BASIC,
    label: t('模型'),
    icon: <Sparkles size={14} />,
  },
  {
    key: SECTION_KEYS.PARAMS,
    label: t('参数'),
    icon: <Sliders size={14} />,
  },
  {
    key: SECTION_KEYS.MEDIA,
    label: t('媒体'),
    icon: <ImageIcon size={14} />,
  },
  {
    key: SECTION_KEYS.ADVANCED,
    label: t('高级'),
    icon: <Wand2 size={14} />,
  },
];

const Surface = ({ children, className = '', tone = 'default' }) => {
  const toneClassName =
    tone === 'soft'
      ? 'bg-[var(--semi-color-fill-0)]'
      : 'bg-[var(--semi-color-bg-0)]';

  return (
    <div
      className={`rounded-lg ${toneClassName} p-3 shadow-[0_1px_3px_rgba(15,23,42,0.06)] ${className}`}
    >
      {children}
    </div>
  );
};

const Field = ({ label, children, className = '' }) => (
  <div className={className}>
    <Typography.Text className='mb-1 block text-xs font-medium text-[var(--semi-color-text-2)]'>
      {label}
    </Typography.Text>
    {children}
  </div>
);

const SectionTabButton = ({ active, icon, label, onClick }) => (
  <button
    type='button'
    onClick={onClick}
    className={`playground-section-tab flex min-h-12 flex-col items-center justify-center gap-1 rounded-lg border px-1.5 py-2 text-xs font-medium transition-all ${
      active
        ? 'border-[var(--semi-color-primary)] bg-[var(--semi-color-primary-light-default)] text-[var(--semi-color-primary)] shadow-[0_3px_12px_rgba(37,99,235,0.14)]'
        : 'border-[var(--semi-color-border)] bg-[var(--semi-color-bg-0)] text-[var(--semi-color-text-1)] hover:bg-[var(--semi-color-fill-0)]'
    }`}
  >
    {icon}
    <span className='whitespace-nowrap leading-4'>{label}</span>
  </button>
);

const DisplayModeTab = ({ active, label, disabled, onClick }) => (
  <button
    type='button'
    disabled={disabled}
    onClick={onClick}
    className={`playground-display-mode-tab ${active ? 'is-active' : ''}`}
  >
    {label}
  </button>
);

const ResolutionBadge = ({ label, title }) => (
  <div className='mt-3 flex items-center justify-between rounded-lg bg-[var(--semi-color-fill-0)] px-3 py-2'>
    <Typography.Text className='text-xs text-[var(--semi-color-text-2)]'>
      {title}
    </Typography.Text>
    <Tag size='small' color='blue' shape='round'>
      {label}
    </Tag>
  </div>
);

const ASPECT_RATIO_ICON_MAX = 18;

/** 在固定 18px 框内按比例计算预览矩形尺寸（竖屏按高度约束，横屏按宽度约束） */
const getAspectRatioIconSize = (ratio) => {
  const box = ASPECT_RATIO_ICON_MAX;
  switch (ratio) {
    case '16:9':
      return { width: box, height: Math.round((box * 9) / 16) };
    case '4:3':
      return { width: box, height: Math.round((box * 3) / 4) };
    case '1:1':
      return { width: box, height: box };
    case '3:4':
      return { width: Math.round((box * 3) / 4), height: box };
    case '9:16':
      return { width: Math.round((box * 9) / 16), height: box };
    case '21:9':
      return { width: box, height: Math.round((box * 9) / 21) };
    case 'auto':
    default:
      return { width: box, height: Math.round((box * 3) / 4) };
  }
};

const AspectRatioIcon = ({ ratio }) => {
  const size = getAspectRatioIconSize(ratio);
  return (
    <span className='playground-aspect-ratio-icon'>
      <span style={{ width: size.width, height: size.height }} />
    </span>
  );
};

const buildAspectRatioOptions = (t, includeAuto = true) =>
  PLAYGROUND_ASPECT_RATIO_OPTIONS.filter(
    (option) => includeAuto || option.value !== 'auto',
  ).map((option) => ({
    value: option.value,
    label: (
      <span className='playground-aspect-ratio-option'>
        <AspectRatioIcon ratio={option.value} />
        <span>{option.value === 'auto' ? t('Auto') : option.label}</span>
      </span>
    ),
  }));

const EmptyState = ({ icon, text }) => (
  <div className='flex min-h-[220px] flex-col items-center justify-center rounded-lg bg-[var(--semi-color-fill-0)] px-4 text-center'>
    <span className='text-[var(--semi-color-text-2)]'>{icon}</span>
    <Typography.Text className='mt-2 text-sm text-[var(--semi-color-text-2)]'>
      {text}
    </Typography.Text>
  </div>
);

const SettingsPanel = ({
  inputs,
  parameterEnabled,
  models,
  modelTypes,
  supplierOptions,
  groups,
  styleState,
  showDebugPanel,
  customRequestMode,
  customRequestBody,
  onInputChange,
  onParameterToggle,
  onCloseSettings,
  onConfigImport,
  onConfigReset,
  onCustomRequestModeChange,
  onCustomRequestBodyChange,
  previewPayload,
  messages,
  userId,
  hideMediaTabs = false,
}) => {
  const { t } = useTranslation();

  const currentConfig = {
    inputs,
    parameterEnabled,
    showDebugPanel,
    customRequestMode,
    customRequestBody,
  };

  const displayMode = inputs.display_mode || 'text';
  const isTextMode = displayMode === 'text';
  const isImageMode = displayMode === 'image';
  const isVideoMode = displayMode === 'video';
  const mediaDisabledClassName = customRequestMode
    ? 'opacity-50 pointer-events-none'
    : '';

  const imageSizeOptions = buildPlaygroundImageSizeOptions(
    inputs.selected_image_pricing_tiers,
  );
  const selectedImageSize = imageSizeOptions.some(
    (option) => option.value === inputs.image_size,
  )
    ? inputs.image_size
    : imageSizeOptions[0]?.value || '1280x720';
  const selectedImagePixelSize = getPlaygroundImageSizeForTier(
    selectedImageSize,
    inputs.image_ratio || 'auto',
  );
  const selectedImageResolutionLabel = formatPlaygroundPixelSizeLabel(
    selectedImagePixelSize.size,
  );

  const videoResolutionOptions = buildPlaygroundVideoResolutionOptions(
    inputs.selected_video_pricing_tiers,
  );
  const selectedVideoResolution = videoResolutionOptions.some(
    (option) => option.value === inputs.video_resolution_preset,
  )
    ? inputs.video_resolution_preset
    : videoResolutionOptions[0]?.value || '720p';
  const selectedVideoSize = getPlaygroundVideoSizeForTier(
    selectedVideoResolution,
    inputs.video_orientation || 'landscape',
    inputs.video_ratio || '',
  );
  const selectedVideoResolutionLabel =
    selectedVideoSize?.size || selectedVideoResolution;

  const showGenerateAudioSwitch = isVideoMode;
  const preferredSection = getDefaultSection(displayMode, customRequestMode);
  const [activeSection, setActiveSection] = useState(preferredSection);
  const previousDisplayModeRef = useRef(displayMode);

  useEffect(() => {
    if (customRequestMode) {
      setActiveSection(SECTION_KEYS.ADVANCED);
      previousDisplayModeRef.current = displayMode;
      return;
    }
    if (previousDisplayModeRef.current !== displayMode) {
      setActiveSection(SECTION_KEYS.BASIC);
      previousDisplayModeRef.current = displayMode;
    }
  }, [customRequestMode, displayMode]);

  const sectionTabs = useMemo(() => buildSectionTabs(t), [t]);
  const aspectRatioOptions = useMemo(() => buildAspectRatioOptions(t), [t]);
  const displayModeTabs = useMemo(
    () =>
      [
        { label: t('文本'), value: 'text' },
        { label: t('图片'), value: 'image' },
        { label: t('视频'), value: 'video' },
      ].filter((mode) => !hideMediaTabs || mode.value === 'text'),
    [hideMediaTabs, t],
  );

  const applyVideoResolutionPreset = (preset) => {
    onInputChange('video_resolution_preset', preset);
  };

  const applyVideoRatio = (ratio) => {
    const nextRatio = ratio || 'auto';
    if (nextRatio === 'auto') {
      const nextOrientation = inputs.video_orientation || 'landscape';
      const nextSize = getPlaygroundVideoSizeForTier(
        selectedVideoResolution,
        nextOrientation,
        'auto',
      );
      onInputChange('video_ratio', 'auto');
      onInputChange('video_width', nextSize.width);
      onInputChange('video_height', nextSize.height);
      return;
    }
    const nextOrientation =
      nextRatio === '3:4' || nextRatio === '9:16' ? 'portrait' : 'landscape';
    const nextSize = getPlaygroundVideoSizeForTier(
      selectedVideoResolution,
      nextOrientation,
      nextRatio,
    );
    onInputChange('video_ratio', nextRatio);
    onInputChange('video_orientation', nextOrientation);
    onInputChange('video_width', nextSize.width);
    onInputChange('video_height', nextSize.height);
  };

  const renderImageMaterial = () => (
    <Surface className={mediaDisabledClassName}>
      <ImageUrlInput
        imageUrls={inputs.imageUrls || ['']}
        imageEnabled={true}
        onImageUrlsChange={(urls) => onInputChange('imageUrls', urls)}
        onImageEnabledChange={() => {}}
        allowToggle={false}
        disabled={customRequestMode}
      />
    </Surface>
  );

  const renderVideoMaterial = () => (
    <Surface className={mediaDisabledClassName}>
      <div className='playground-media-stack'>
        <MaterialLibraryButton
          disabled={customRequestMode}
          imageUrls={inputs.imageUrls || ['']}
          onImageUrlsChange={(urls) => onInputChange('imageUrls', urls)}
          videoUrls={inputs.videoUrls || ['']}
          onVideoUrlsChange={(urls) => onInputChange('videoUrls', urls)}
          audioUrls={inputs.audioUrls || ['']}
          onAudioUrlsChange={(urls) => onInputChange('audioUrls', urls)}
        />
        <ImageUrlInput
          imageUrls={inputs.imageUrls || ['']}
          imageEnabled={true}
          onImageUrlsChange={(urls) => onInputChange('imageUrls', urls)}
          onImageEnabledChange={() => {}}
          allowToggle={false}
          disabled={customRequestMode}
        />
        <VideoUrlInput
          videoUrls={inputs.videoUrls || ['']}
          videoEnabled={true}
          onVideoUrlsChange={(urls) => onInputChange('videoUrls', urls)}
          onVideoEnabledChange={() => {}}
          allowToggle={false}
          disabled={customRequestMode}
        />
        {/* Seedance 2.0：参考音频链接输入 + 本地上传，写入 metadata.audio_urls */}
        <AudioUrlInput
          audioUrls={inputs.audioUrls || ['']}
          audioEnabled={true}
          onAudioUrlsChange={(urls) => onInputChange('audioUrls', urls)}
          onAudioEnabledChange={() => {}}
          allowToggle={false}
          disabled={customRequestMode}
        />
      </div>
    </Surface>
  );

  const renderImageParams = () => (
    <Surface className={mediaDisabledClassName}>
      <div className='grid grid-cols-2 gap-3'>
        <Field label={t('分辨率')} className='col-span-2'>
          <Select
            placeholder={t('图片尺寸')}
            optionList={imageSizeOptions}
            value={selectedImageSize}
            onChange={(value) => onInputChange('image_size', value)}
            disabled={customRequestMode}
            style={{ width: '100%' }}
            renderSelectedItem={(option) => (
              <span>
                {option?.label ??
                  formatVideoResolutionDisplayLabel(selectedImageSize) ??
                  selectedImageSize}
              </span>
            )}
          />
        </Field>
        <Field label={t('比例')} className='col-span-2'>
          <RadioGroup
            type='button'
            buttonSize='small'
            value={inputs.image_ratio || 'auto'}
            options={aspectRatioOptions}
            onChange={(e) =>
              onInputChange('image_ratio', e?.target?.value || 'auto')
            }
            disabled={customRequestMode}
            className='playground-aspect-ratio-group'
          />
        </Field>
        <Field label={t('生成数量')} className='col-span-2'>
          <Input
            type='number'
            min={1}
            max={4}
            placeholder={t('生成数量 n')}
            value={inputs.image_n}
            onChange={(value) =>
              onInputChange(
                'image_n',
                Math.max(1, Math.min(4, Number(value) || 1)),
              )
            }
            disabled={customRequestMode}
          />
        </Field>
      </div>
      <ResolutionBadge
        label={selectedImageResolutionLabel}
        title={t('当前分辨率')}
      />
    </Surface>
  );

  const renderVideoParams = () => (
    <Surface className={mediaDisabledClassName}>
      <div className='grid grid-cols-2 gap-3'>
        <Field label={t('视频时长（秒）')} className='col-span-2'>
          <Select
            optionList={PLAYGROUND_VIDEO_DURATION_OPTIONS}
            value={Math.max(
              4,
              Math.min(15, Number(inputs.video_duration) || 5),
            )}
            onChange={(value) =>
              onInputChange('video_duration', Number(value) || 5)
            }
            disabled={customRequestMode}
            style={{ width: '100%' }}
          />
        </Field>
        <Field label={t('分辨率')}>
          <Select
            placeholder={t('分辨率预设')}
            optionList={videoResolutionOptions}
            value={selectedVideoResolution}
            onChange={(value) => applyVideoResolutionPreset(value)}
            disabled={customRequestMode}
            style={{ width: '100%' }}
          />
        </Field>
        <Field label={t('比例')} className='col-span-2'>
          <RadioGroup
            type='button'
            buttonSize='small'
            value={inputs.video_ratio || '16:9'}
            options={aspectRatioOptions}
            onChange={(e) => applyVideoRatio(e?.target?.value)}
            disabled={customRequestMode}
            className='playground-aspect-ratio-group'
          />
        </Field>
        {showGenerateAudioSwitch && (
          <div className='col-span-2 flex items-center justify-between rounded-lg bg-[var(--semi-color-fill-0)] px-3 py-2'>
            <Typography.Text strong className='text-sm'>
              {t('生成音频')}
            </Typography.Text>
            <Switch
              checked={inputs.generate_audio !== false}
              onChange={(checked) => onInputChange('generate_audio', checked)}
              checkedText={t('开')}
              uncheckedText={t('关')}
              size='small'
              disabled={customRequestMode}
            />
          </div>
        )}
        <div className='hidden'>
          <Input
            type='number'
            min={320}
            max={4096}
            placeholder={t('宽度 width')}
            value={inputs.video_width}
            onChange={(value) =>
              onInputChange('video_width', Number(value) || 1280)
            }
            disabled={customRequestMode}
          />
          <Input
            type='number'
            min={320}
            max={4096}
            placeholder={t('高度 height')}
            value={inputs.video_height}
            onChange={(value) =>
              onInputChange('video_height', Number(value) || 720)
            }
            disabled={customRequestMode}
          />
        </div>
        <Field label={t('运动强度 motion')} className='col-span-2'>
          <div className='px-1'>
            <Slider
              min={0}
              max={1}
              step={0.1}
              marks={{ 0: '0.0', 0.5: '0.5', 1: '1.0' }}
              value={Number(inputs.video_motion ?? 0.4)}
              onChange={(value) =>
                onInputChange('video_motion', Number(value ?? 0.4))
              }
              disabled={customRequestMode}
              tipFormatter={(value) => `${Number(value).toFixed(1)}`}
            />
          </div>
        </Field>
        <Field label={t('生成数量 n')} className='col-span-2'>
          <Select
            optionList={[
              { label: '1', value: 1 },
              { label: '2', value: 2 },
              { label: '3', value: 3 },
            ]}
            value={Math.max(1, Math.min(3, Number(inputs.video_n) || 1))}
            onChange={(value) =>
              onInputChange(
                'video_n',
                Math.max(1, Math.min(3, Number(value) || 1)),
              )
            }
            disabled={customRequestMode}
            style={{ width: '100%' }}
          />
        </Field>
      </div>
      <ResolutionBadge
        label={selectedVideoResolutionLabel}
        title={t('当前分辨率')}
      />
    </Surface>
  );

  // Basic section: media modes intentionally hide output options.
  const renderBasicSection = () => (
    <div className={customRequestMode ? 'opacity-50 pointer-events-none' : ''}>
      <Surface>
        <div className='space-y-3'>
          <Field label={t('模型类型')}>
            <Select
              placeholder={t('请选择模型类型')}
              name='model_type'
              selection
              filter={selectFilter}
              autoClearSearchValue={false}
              onChange={(value) =>
                onInputChange(
                  'model_type',
                  value === undefined || value === null ? '' : value,
                )
              }
              value={
                inputs.model_type === undefined || inputs.model_type === null
                  ? ''
                  : inputs.model_type
              }
              autoComplete='new-password'
              optionList={modelTypes}
              style={{ width: '100%' }}
              dropdownStyle={{ width: '100%', maxWidth: '100%' }}
              className='!rounded-lg'
              disabled={customRequestMode}
            />
          </Field>
          <Field label={t('模型')}>
            <Select
              placeholder={t('请选择模型')}
              name='model'
              required
              selection
              filter={selectFilter}
              autoClearSearchValue={false}
              onChange={(value) => onInputChange('model', value)}
              value={inputs.model}
              autoComplete='new-password'
              optionList={models}
              style={{ width: '100%' }}
              dropdownStyle={{ width: '100%', maxWidth: '100%' }}
              className='!rounded-lg'
              disabled={customRequestMode}
            />
          </Field>
          <Field label={t('渠道商')}>
            <Select
              placeholder={t('请选择渠道商')}
              name='selected_route_slug'
              selection
              filter={selectFilter}
              autoClearSearchValue={false}
              onChange={(value) =>
                onInputChange(
                  'selected_route_slug',
                  value === undefined || value === null ? '' : value,
                )
              }
              value={
                inputs.selected_route_slug === undefined ||
                inputs.selected_route_slug === null
                  ? ''
                  : inputs.selected_route_slug
              }
              autoComplete='new-password'
              optionList={supplierOptions}
              style={{ width: '100%' }}
              dropdownStyle={{ width: '100%', maxWidth: '100%' }}
              className='!rounded-lg'
              disabled={customRequestMode}
            />
          </Field>
        </div>
      </Surface>

      <div className='hidden'>
        <Select
          placeholder={t('请选择分组')}
          name='group'
          required
          selection
          filter={selectFilter}
          autoClearSearchValue={false}
          onChange={(value) => onInputChange('group', value)}
          value={inputs.group}
          autoComplete='new-password'
          optionList={groups}
          renderOptionItem={renderPlaygroundGroupOption}
          style={{ width: '100%' }}
          dropdownStyle={{ width: '100%', maxWidth: '100%' }}
          disabled={customRequestMode}
        />
      </div>

      {isTextMode && (
        <Surface tone='soft' className='mt-3'>
          <div className='flex items-center justify-between gap-3'>
            <div className='flex min-w-0 items-center gap-2'>
              <span className='flex h-7 w-7 items-center justify-center rounded-lg bg-[var(--semi-color-primary-light-default)] text-[var(--semi-color-primary)]'>
                <ToggleLeft size={14} />
              </span>
              <Typography.Text strong className='truncate text-sm'>
                {t('流式输出')}
              </Typography.Text>
            </div>
            <Switch
              checked={inputs.stream}
              onChange={(checked) => onInputChange('stream', checked)}
              checkedText={t('开')}
              uncheckedText={t('关')}
              size='small'
              disabled={customRequestMode}
            />
          </div>
        </Surface>
      )}
    </div>
  );

  // Params section: text keeps model parameters; image/video receive migrated media params.
  const renderParamsSection = () => {
    if (isImageMode) {
      return renderImageParams();
    }
    if (isVideoMode) {
      return renderVideoParams();
    }
    return (
      <div
        className={customRequestMode ? 'opacity-50 pointer-events-none' : ''}
      >
        <ParameterControl
          inputs={inputs}
          parameterEnabled={parameterEnabled}
          onInputChange={onInputChange}
          onParameterToggle={onParameterToggle}
          disabled={customRequestMode}
        />
      </div>
    );
  };

  // Media section: text/image provide image upload; video provides image + video upload only.
  const renderMediaSection = () => {
    if (isVideoMode) {
      return renderVideoMaterial();
    }
    if (isImageMode || isTextMode) {
      return renderImageMaterial();
    }
    return (
      <EmptyState
        icon={<ImageIcon size={30} />}
        text={t('请选择文本、图片或视频模式')}
      />
    );
  };

  const renderAdvancedSection = () => (
    <Surface>
      <CustomRequestEditor
        customRequestMode={customRequestMode}
        customRequestBody={customRequestBody}
        onCustomRequestModeChange={onCustomRequestModeChange}
        onCustomRequestBodyChange={onCustomRequestBodyChange}
        defaultPayload={previewPayload}
      />
    </Surface>
  );

  const renderActiveSection = () => {
    if (activeSection === SECTION_KEYS.PARAMS) {
      return renderParamsSection();
    }
    if (activeSection === SECTION_KEYS.MEDIA) {
      return renderMediaSection();
    }
    if (activeSection === SECTION_KEYS.ADVANCED) {
      return renderAdvancedSection();
    }
    return renderBasicSection();
  };

  return (
    <Card
      className='playground-settings-panel h-full flex flex-col overflow-hidden !rounded-none'
      bordered={false}
      bodyStyle={{
        padding: 0,
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        background: 'var(--semi-color-bg-1)',
      }}
    >
      <div className='flex-shrink-0 bg-[var(--semi-color-bg-0)] px-4 pb-3 pt-4 shadow-[0_1px_0_var(--semi-color-border)]'>
        <div className='mb-3 flex items-center justify-between gap-3'>
          <div className='flex min-w-0 items-center gap-2'>
            <span className='flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-[var(--semi-color-primary-light-default)] text-[var(--semi-color-primary)]'>
              <Settings size={16} />
            </span>
            <div className='min-w-0'>
              <Typography.Title heading={6} className='!mb-0 truncate'>
                {t('模型配置')}
              </Typography.Title>
              <Typography.Text className='block truncate text-xs text-[var(--semi-color-text-2)]'>
                {inputs.model || t('请选择模型')}
              </Typography.Text>
            </div>
          </div>
          {styleState.isMobile && onCloseSettings && (
            <Button
              icon={<X size={16} />}
              onClick={onCloseSettings}
              theme='borderless'
              type='tertiary'
              size='small'
              className='!rounded-lg'
            />
          )}
        </div>

        <div className='playground-display-mode-tabs'>
          {displayModeTabs.map((mode) => (
            <DisplayModeTab
              key={mode.value}
              label={mode.label}
              active={displayMode === mode.value}
              onClick={() => onInputChange('display_mode', mode.value)}
            />
          ))}
        </div>
      </div>

      {styleState.isMobile && (
        <div className='bg-[var(--semi-color-bg-0)] px-4 py-2 shadow-[0_1px_0_var(--semi-color-border)]'>
          <ConfigManager
            currentConfig={currentConfig}
            onConfigImport={onConfigImport}
            onConfigReset={onConfigReset}
            styleState={{ ...styleState, isMobile: false }}
            messages={messages}
            userId={userId}
          />
        </div>
      )}

      <div className='flex-shrink-0 bg-[var(--semi-color-bg-0)] px-4 py-3 shadow-[0_1px_0_var(--semi-color-border)]'>
        <div className='grid grid-cols-4 gap-1.5'>
          {sectionTabs.map((item) => (
            <SectionTabButton
              key={item.key}
              active={activeSection === item.key}
              icon={item.icon}
              label={item.label}
              onClick={() => setActiveSection(item.key)}
            />
          ))}
        </div>
      </div>

      <div className='model-settings-scroll min-h-0 flex-1 overflow-y-auto px-4 py-3'>
        {renderActiveSection()}
      </div>

      {!styleState.isMobile && (
        <div className='flex-shrink-0 bg-[var(--semi-color-bg-0)] px-4 py-3 shadow-[0_-1px_0_var(--semi-color-border)]'>
          <ConfigManager
            currentConfig={currentConfig}
            onConfigImport={onConfigImport}
            onConfigReset={onConfigReset}
            styleState={styleState}
            messages={messages}
            userId={userId}
          />
        </div>
      )}
    </Card>
  );
};

export default SettingsPanel;
