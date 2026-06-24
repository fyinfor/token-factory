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
import {
  Card,
  Select,
  Typography,
  Button,
  Switch,
  RadioGroup,
  Input,
  Slider,
} from '@douyinfe/semi-ui';
import { Sparkles, Users, ToggleLeft, X, Settings } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import {
  buildPlaygroundImageSizeOptions,
  buildPlaygroundVideoResolutionOptions,
  formatVideoResolutionDisplayLabel,
  getPlaygroundVideoSizeForTier,
  renderGroupOption,
  selectFilter,
} from '../../helpers';
import { PLAYGROUND_VIDEO_DURATION_OPTIONS } from '../../constants/playground.constants';
import { CHANNEL_TYPES_WITH_GENERATE_AUDIO } from '../../constants/channel.constants';
import ParameterControl from './ParameterControl';

/**
 * 操练场分组下拉：与全局 renderGroupOption 一致，但不展示倍率角标。
 * @param {Record<string, unknown>} item Semi Select 传入的选项渲染参数
 */
const renderPlaygroundGroupOption = (item) =>
  renderGroupOption({ ...item, ratio: undefined });
import ImageUrlInput from './ImageUrlInput';
import VideoUrlInput from './VideoUrlInput';
import ConfigManager from './ConfigManager';
import CustomRequestEditor from './CustomRequestEditor';

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
  const isImageMode = displayMode === 'image';
  const isVideoMode = displayMode === 'video';
  const mediaModeEnabled = isImageMode || isVideoMode;
  const imageSizeOptions = buildPlaygroundImageSizeOptions(
    inputs.selected_image_pricing_tiers,
  );
  const selectedImageSize = imageSizeOptions.some(
    (option) => option.value === inputs.image_size,
  )
    ? inputs.image_size
    : imageSizeOptions[0]?.value || '1280x720';
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
  );
  const videoMediaHint = isVideoMode
    ? t(
        '操练场视频素材提示',
        '图片地址：第 1 张为首帧，2 张为首尾帧，更多张时最后一张为尾帧。视频地址：填写则作为源视频参与生成。未填写的字段不会加入请求。',
      )
    : '';
  const selectedChannelType = Number(inputs.selected_channel_type ?? 0);
  const showGenerateAudioSwitch =
    isVideoMode &&
    CHANNEL_TYPES_WITH_GENERATE_AUDIO.has(selectedChannelType);
  const applyVideoResolutionPreset = (preset) => {
    onInputChange('video_resolution_preset', preset);
  };

  return (
    <Card
      className='h-full flex flex-col'
      bordered={false}
      bodyStyle={{
        padding: styleState.isMobile ? '16px' : '24px',
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
      }}
    >
      {/* 标题区域 - 与调试面板保持一致 */}
      <div className='flex items-center justify-between mb-6 flex-shrink-0'>
        <div className='flex items-center'>
          <div className='w-10 h-10 rounded-full bg-gradient-to-r from-purple-500 to-pink-500 flex items-center justify-center mr-3'>
            <Settings size={20} className='text-white' />
          </div>
          <Typography.Title heading={5} className='mb-0'>
            {t('模型配置')}
          </Typography.Title>
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

      {/* 展示模式（置顶，左右切换） */}
      <div className='mb-4 flex-shrink-0'>
        <Typography.Text strong className='text-sm mb-2 block'>
          {t('展示模式')}
        </Typography.Text>
        <RadioGroup
          type='button'
          buttonSize='large'
          value={displayMode}
          options={[
            { label: t('文本'), value: 'text' },
            { label: t('图片'), value: 'image' },
            { label: t('视频'), value: 'video' },
          ]}
          onChange={(e) => {
            const nextMode = e?.target?.value || 'text';
            onInputChange('display_mode', nextMode);
            // 切换模式时重置类型筛选，避免残留筛选导致“明明有模型却不显示”
            onInputChange('model_type', '');
          }}
          disabled={customRequestMode}
        />
      </div>

      {/* 移动端配置管理 */}
      {styleState.isMobile && (
        <div className='mb-4 flex-shrink-0'>
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

      <div className='space-y-6 overflow-y-auto flex-1 pr-2 model-settings-scroll'>
        {/* 自定义请求体编辑器 */}
        <CustomRequestEditor
          customRequestMode={customRequestMode}
          customRequestBody={customRequestBody}
          onCustomRequestModeChange={onCustomRequestModeChange}
          onCustomRequestBodyChange={onCustomRequestBodyChange}
          defaultPayload={previewPayload}
        />

        {/* 分组选择（UI 隐藏，保留数据加载与配置逻辑） */}
        <div className={`hidden ${customRequestMode ? 'opacity-50' : ''}`}>
          <div className='flex items-center gap-2 mb-2'>
            <Users size={16} className='text-gray-500' />
            <Typography.Text strong className='text-sm'>
              {t('分组')}
            </Typography.Text>
            {customRequestMode && (
              <Typography.Text className='text-xs text-orange-600'>
                ({t('已在自定义模式中忽略')})
              </Typography.Text>
            )}
          </div>
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
            className='!rounded-lg'
            disabled={customRequestMode}
          />
        </div>

        {/* 模型选择 */}
        <div className={customRequestMode ? 'opacity-50' : ''}>
          <div className='flex items-center gap-2 mb-2'>
            <Sparkles size={16} className='text-gray-500' />
            <Typography.Text strong className='text-sm'>
              {t('模型类型')}
            </Typography.Text>
            {customRequestMode && (
              <Typography.Text className='text-xs text-orange-600'>
                ({t('已在自定义模式中忽略')})
              </Typography.Text>
            )}
          </div>
          {/* model_type 为 0 表示未关联类型，与空字符串「全部」不同，不能用 value={x || ''} */}
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
            className='!rounded-lg mb-3'
            disabled={customRequestMode}
          />

          <div className='flex items-center gap-2 mb-2'>
            <Sparkles size={16} className='text-gray-500' />
            <Typography.Text strong className='text-sm'>
              {t('模型')}
            </Typography.Text>
            {customRequestMode && (
              <Typography.Text className='text-xs text-orange-600'>
                ({t('已在自定义模式中忽略')})
              </Typography.Text>
            )}
          </div>
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

          <div className='flex items-center gap-2 mb-2 mt-3'>
            <Sparkles size={16} className='text-gray-500' />
            <Typography.Text strong className='text-sm'>
              {t('渠道商')}
            </Typography.Text>
            {customRequestMode && (
              <Typography.Text className='text-xs text-orange-600'>
                ({t('已在自定义模式中忽略')})
              </Typography.Text>
            )}
          </div>
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
        </div>

        {/* 素材 URL：视频模式分图片地址 / 视频地址 */}
        {mediaModeEnabled && (
          <div className={customRequestMode ? 'opacity-50' : ''}>
            <ImageUrlInput
              imageUrls={inputs.imageUrls}
              imageEnabled={true}
              onImageUrlsChange={(urls) => onInputChange('imageUrls', urls)}
              onImageEnabledChange={() => {}}
              allowToggle={false}
              disabled={customRequestMode}
            />
            {isVideoMode && (
              <VideoUrlInput
                videoUrls={inputs.videoUrls || ['']}
                videoEnabled={true}
                onVideoUrlsChange={(urls) => onInputChange('videoUrls', urls)}
                onVideoEnabledChange={() => {}}
                allowToggle={false}
                disabled={customRequestMode}
              />
            )}
            <Typography.Text className='text-xs mt-1 block text-gray-500'>
              {isVideoMode
                ? videoMediaHint
                : t('图片模式支持图片 URL 作为素材')}
            </Typography.Text>
          </div>
        )}

        {/* 模式参数区 */}
        {displayMode === 'text' && (
          <div className={customRequestMode ? 'opacity-50' : ''}>
            <ParameterControl
              inputs={inputs}
              parameterEnabled={parameterEnabled}
              onInputChange={onInputChange}
              onParameterToggle={onParameterToggle}
              disabled={customRequestMode}
            />
          </div>
        )}

        {isImageMode && (
          <div className={customRequestMode ? 'opacity-50' : ''}>
            <Typography.Text strong className='text-sm mb-2 block'>
              {t('图片参数')}
            </Typography.Text>
            <Typography.Text className='text-xs text-gray-500 mb-2 block'>
              {t('用于文生图/图生图的核心参数')}
            </Typography.Text>
            <Typography.Text className='text-xs text-orange-600 mb-2 block'>
              {t('计费以实际图片质量为准')}
            </Typography.Text>
            <div className='space-y-3'>
              <Typography.Text strong className='text-sm block'>
                {t('分辨率')}
              </Typography.Text>
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
              <Typography.Text strong className='text-sm block'>
                {t('生成图片数量')}
              </Typography.Text>
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
            </div>
          </div>
        )}

        {isVideoMode && (
          <div className={customRequestMode ? 'opacity-50' : ''}>
            <Typography.Text strong className='text-sm mb-2 block'>
              {t('视频参数')}
            </Typography.Text>
            <Typography.Text className='text-xs text-gray-500 mb-2 block'>
              {t('用于视频生成的核心参数（时长、分辨率）')}
            </Typography.Text>
            <Typography.Text className='text-xs text-orange-600 mb-2 block'>
              {t('计费以实际视频质量为准')}
            </Typography.Text>
            <div className='space-y-3'>
              <Typography.Text strong className='text-sm block'>
                {t('视频时长（秒）')}
              </Typography.Text>
              <Select
                optionList={PLAYGROUND_VIDEO_DURATION_OPTIONS}
                value={Math.max(
                  3,
                  Math.min(30, Number(inputs.video_duration) || 5),
                )}
                onChange={(value) =>
                  onInputChange('video_duration', Number(value) || 5)
                }
                disabled={customRequestMode}
                style={{ width: '100%' }}
              />
              <Typography.Text strong className='text-sm block'>
                {t('分辨率')}
              </Typography.Text>
              <Select
                placeholder={t('分辨率预设')}
                optionList={videoResolutionOptions}
                value={selectedVideoResolution}
                onChange={(value) => applyVideoResolutionPreset(value)}
                disabled={customRequestMode}
                style={{ width: '100%' }}
              />
              <Typography.Text strong className='text-sm block'>
                {t('\u753b\u9762\u65b9\u5411')}
              </Typography.Text>
              <RadioGroup
                type='button'
                buttonSize='middle'
                value={inputs.video_orientation || 'landscape'}
                options={[
                  { label: t('\u6a2a\u5c4f'), value: 'landscape' },
                  { label: t('\u7ad6\u5c4f'), value: 'portrait' },
                ]}
                onChange={(e) =>
                  onInputChange(
                    'video_orientation',
                    e?.target?.value || 'landscape',
                  )
                }
                disabled={customRequestMode}
              />
              <Typography.Text className='text-xs text-gray-500 block'>
                {selectedVideoSize.size}
              </Typography.Text>
              {showGenerateAudioSwitch && (
                <div className='flex items-center justify-between pt-1'>
                  <Typography.Text strong className='text-sm'>
                    {t('生成音频')}
                  </Typography.Text>
                  <Switch
                    checked={inputs.generate_audio !== false}
                    onChange={(checked) =>
                      onInputChange('generate_audio', checked)
                    }
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
              <Typography.Text strong className='text-sm block'>
                {t('运动强度 motion')}
              </Typography.Text>
              <div className='px-1'>
                <Slider
                  min={0}
                  max={1}
                  step={0.1}
                  marks={{
                    0: '0.0',
                    0.5: '0.5',
                    1: '1.0',
                  }}
                  value={Number(inputs.video_motion ?? 0.4)}
                  onChange={(value) =>
                    onInputChange('video_motion', Number(value ?? 0.4))
                  }
                  disabled={customRequestMode}
                  tipFormatter={(value) => `${Number(value).toFixed(1)}`}
                />
              </div>
              <Typography.Text strong className='text-sm block'>
                {t('生成数量 n')}
              </Typography.Text>
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
            </div>
          </div>
        )}

        {/* 流式输出开关 */}
        <div className={customRequestMode ? 'opacity-50' : ''}>
          <div className='flex items-center justify-between'>
            <div className='flex items-center gap-2'>
              <ToggleLeft size={16} className='text-gray-500' />
              <Typography.Text strong className='text-sm'>
                {t('流式输出')}
              </Typography.Text>
              {customRequestMode && (
                <Typography.Text className='text-xs text-orange-600'>
                  ({t('已在自定义模式中忽略')})
                </Typography.Text>
              )}
            </div>
            <Switch
              checked={inputs.stream}
              onChange={(checked) => onInputChange('stream', checked)}
              checkedText={t('开')}
              uncheckedText={t('关')}
              size='small'
              disabled={customRequestMode || isImageMode || isVideoMode}
            />
          </div>
          {(isImageMode || isVideoMode) && (
            <Typography.Text className='text-xs text-orange-600'>
              {t('图片/视频模式不支持流式输出，已自动关闭')}
            </Typography.Text>
          )}
        </div>
      </div>

      {/* 桌面端的配置管理放在底部 */}
      {!styleState.isMobile && (
        <div className='flex-shrink-0 pt-3'>
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
