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
import { renderGroupOption, selectFilter } from '../../helpers';
import ParameterControl from './ParameterControl';
import ImageUrlInput from './ImageUrlInput';
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
  const selectedModelTags = status?.selectedModelTags || [];
  const isFixed24FPS =
    isVideoMode &&
    selectedModelTags.some((tag) =>
      ['固定24帧', '24fps固定', '24帧固定'].includes(String(tag || '').trim()),
    );
  const applyVideoResolutionPreset = (preset) => {
    const [w, h] = String(preset || '1280x720')
      .split('x')
      .map((n) => Number(n));
    onInputChange('video_resolution_preset', preset);
    if (Number.isFinite(w) && Number.isFinite(h)) {
      onInputChange('video_width', w);
      onInputChange('video_height', h);
    }
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

        {/* 分组选择 */}
        <div className={customRequestMode ? 'opacity-50' : ''}>
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
            renderOptionItem={renderGroupOption}
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
            name='specific_channel_id'
            selection
            filter={selectFilter}
            autoClearSearchValue={false}
            onChange={(value) =>
              onInputChange(
                'specific_channel_id',
                value === undefined || value === null ? '' : value,
              )
            }
            value={
              inputs.specific_channel_id === undefined ||
              inputs.specific_channel_id === null
                ? ''
                : inputs.specific_channel_id
            }
            autoComplete='new-password'
            optionList={supplierOptions}
            style={{ width: '100%' }}
            dropdownStyle={{ width: '100%', maxWidth: '100%' }}
            className='!rounded-lg'
            disabled={customRequestMode}
          />

        </div>

        {/* 素材URL输入：图片模式仅图片；视频模式支持图片/视频 */}
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
            <Typography.Text className='text-xs text-gray-500 mt-1 block'>
              {isVideoMode
                ? t('视频模式支持图片或视频 URL 作为素材')
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
            <div className='space-y-3'>
              <Select
                placeholder={t('图片尺寸')}
                optionList={[
                  { label: '512x512', value: '512x512' },
                  { label: '768x768', value: '768x768' },
                  { label: '1024x1024', value: '1024x1024' },
                  { label: '1536x1024', value: '1536x1024' },
                  { label: '1024x1536', value: '1024x1536' },
                ]}
                value={inputs.image_size}
                onChange={(value) => onInputChange('image_size', value)}
                disabled={customRequestMode}
                style={{ width: '100%' }}
              />
              <Input
                type='number'
                min={1}
                max={4}
                placeholder={t('生成数量 n')}
                value={inputs.image_n}
                onChange={(value) =>
                  onInputChange('image_n', Math.max(1, Math.min(4, Number(value) || 1)))
                }
                disabled={customRequestMode}
              />
              <Select
                placeholder={t('质量 quality')}
                optionList={[
                  { label: 'standard', value: 'standard' },
                  { label: 'hd', value: 'hd' },
                ]}
                value={inputs.image_quality || 'standard'}
                onChange={(value) =>
                  onInputChange('image_quality', value || 'standard')
                }
                disabled={customRequestMode}
                style={{ width: '100%' }}
              />
              <Select
                placeholder={t('返回格式 response_format')}
                optionList={[
                  { label: 'url', value: 'url' },
                  { label: 'b64_json', value: 'b64_json' },
                ]}
                value={inputs.image_response_format || 'url'}
                onChange={(value) =>
                  onInputChange('image_response_format', value || 'url')
                }
                disabled={customRequestMode}
                style={{ width: '100%' }}
              />
              <Select
                placeholder={t('风格 style')}
                optionList={[
                  { label: 'vivid', value: 'vivid' },
                  { label: 'natural', value: 'natural' },
                ]}
                value={inputs.image_style || 'vivid'}
                onChange={(value) => onInputChange('image_style', value || 'vivid')}
                disabled={customRequestMode}
                style={{ width: '100%' }}
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
              {t('用于视频生成的核心参数（时长、分辨率、帧率）')}
            </Typography.Text>
            <div className='space-y-3'>
              <Typography.Text strong className='text-sm block'>
                {t('视频时长（秒）')}
              </Typography.Text>
              <Select
                optionList={[
                  { label: '5s', value: 5 },
                  { label: '10s', value: 10 },
                  { label: '15s', value: 15 },
                  { label: '20s', value: 20 },
                  { label: '30s', value: 30 },
                  { label: '45s', value: 45 },
                  { label: '60s', value: 60 },
                ]}
                value={inputs.video_duration}
                onChange={(value) => onInputChange('video_duration', Number(value) || 5)}
                disabled={customRequestMode}
                style={{ width: '100%' }}
              />
              <Typography.Text strong className='text-sm block'>
                {t('分辨率')}
              </Typography.Text>
              <Select
                placeholder={t('分辨率预设')}
                optionList={[
                  { label: '960x540', value: '960x540' },
                  { label: '1280x720', value: '1280x720' },
                  { label: '1920x1080', value: '1920x1080' },
                ]}
                value={inputs.video_resolution_preset}
                onChange={(value) => applyVideoResolutionPreset(value)}
                disabled={customRequestMode}
                style={{ width: '100%' }}
              />
              <div className='grid grid-cols-2 gap-2'>
                <Input
                  type='number'
                  min={320}
                  max={4096}
                  placeholder={t('宽度 width')}
                  value={inputs.video_width}
                  onChange={(value) => onInputChange('video_width', Number(value) || 1280)}
                  disabled={customRequestMode}
                />
                <Input
                  type='number'
                  min={320}
                  max={4096}
                  placeholder={t('高度 height')}
                  value={inputs.video_height}
                  onChange={(value) => onInputChange('video_height', Number(value) || 720)}
                  disabled={customRequestMode}
                />
              </div>
              <Typography.Text strong className='text-sm block'>
                {t('帧率')}
              </Typography.Text>
              <Select
                placeholder={t('帧率 fps')}
                optionList={
                  isFixed24FPS
                    ? [{ label: '24 fps (固定)', value: 24 }]
                    : [
                        { label: '12 fps', value: 12 },
                        { label: '24 fps', value: 24 },
                        { label: '30 fps', value: 30 },
                      ]
                }
                value={isFixed24FPS ? 24 : inputs.video_fps}
                onChange={(value) => onInputChange('video_fps', Number(value) || 24)}
                disabled={customRequestMode || isFixed24FPS}
                style={{ width: '100%' }}
              />
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
                  onInputChange('video_n', Math.max(1, Math.min(3, Number(value) || 1)))
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
