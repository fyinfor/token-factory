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
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Empty, Select, Tag } from '@douyinfe/semi-ui';
import { API, showError } from '../../../../helpers';
import { useTranslation } from 'react-i18next';
import ModelPricingEditor from './ModelPricingEditor';

const parseJSON = (text) => {
  if (!text || !text.trim()) return {};
  try {
    const parsed = JSON.parse(text);
    return parsed && typeof parsed === 'object' ? parsed : {};
  } catch {
    return {};
  }
};

const getChannelSortId = (channel) => {
  const id = Number(channel?.channel_id);
  return Number.isFinite(id) ? id : 0;
};

export default function SupplierModelPricingEditor({
  options,
  refresh,
  candidateModelNames = [],
  filterMode = 'all',
  listDescription = '',
  /** 为 true 时使用表存储接口，不写全局 ChannelModel* Option */
  useSupplierPricingApi = false,
}) {
  const { t } = useTranslation();
  const [channelId, setChannelId] = useState('all');
  const [channelModelNamesMap, setChannelModelNamesMap] = useState({});
  /** 供应商渠道定价 GET /api/user/supplier/pricing/channel/:id 返回的 maps */
  const [supplierChannelMaps, setSupplierChannelMaps] = useState(null);

  const channels = useMemo(() => {
    const raw = options.__pricingChannels || [];
    if (!Array.isArray(raw)) return [];
    return [...raw]
      .filter((s) => s?.channel_id)
      .sort((a, b) => getChannelSortId(b) - getChannelSortId(a))
      .map((s) => {
        const name = s.channel_name || `#${s.channel_id}`;
        return {
          label: name,
          value: String(s.channel_id),
          channelName: name,
          channelNo: String(s.channel_no || '').trim(),
          channelTag: String(s.tag || '').trim(),
          routeSlug: String(s.route_slug || '').trim(),
          status: Number(s.status || 0),
        };
      });
  }, [options.__pricingChannels]);

  const channelOptionMap = useMemo(() => {
    return channels.reduce((acc, channel) => {
      acc[channel.value] = channel;
      return acc;
    }, {});
  }, [channels]);

  const renderChannelStatusTag = (status) => {
    const enabled = Number(status) === 1;
    return (
      <Tag color={enabled ? 'green' : 'red'} size='small' shape='circle'>
        {enabled ? t('启用') : t('禁用')}
      </Tag>
    );
  };

  const renderChannelOption = (channel) => {
    if (!channel || channel.value === 'all') {
      return channel?.label || '';
    }
    const routeSlug = channel.routeSlug || t('未设置');
    return (
      <div
        className='flex min-w-0 items-center justify-between gap-3'
        style={{ width: '100%', minWidth: 0 }}
      >
        <span
          className='supplier-channel-pricing-option-name min-w-0 flex-1 truncate'
          title={channel.channelName}
          style={{ minWidth: 0 }}
        >
          {channel.channelName}
        </span>
        <span className='flex shrink-0 items-center gap-1.5'>
          <Tag color={channel.routeSlug ? 'blue' : 'grey'} size='small'>
            {channel.routeSlug ? `/${routeSlug}` : routeSlug}
          </Tag>
          {channel.channelTag && (
            <Tag color='cyan' size='small'>
              {channel.channelTag}
            </Tag>
          )}
          {renderChannelStatusTag(channel.status)}
        </span>
      </div>
    );
  };

  const renderChannelOptionItem = (optionProps) => {
    const { value, className, style, onMouseEnter, onClick } = optionProps;
    const channel =
      String(value || '') === 'all'
        ? { label: t('请选择渠道'), value: 'all' }
        : optionProps;
    return (
      <div
        className={`${className || ''} supplier-channel-pricing-option`}
        style={{ ...style, padding: 0 }}
        onMouseEnter={onMouseEnter}
        onClick={onClick}
        role='option'
        aria-selected={optionProps.selected ? 'true' : 'false'}
      >
        <div
          className='supplier-channel-pricing-option-content flex items-center'
          style={{
            minHeight: 36,
            padding: '8px 12px',
            boxSizing: 'border-box',
            width: '100%',
          }}
        >
          {renderChannelOption(channel)}
        </div>
      </div>
    );
  };

  const filterChannelOption = (inputValue, option) => {
    const keyword = String(inputValue || '')
      .trim()
      .toLowerCase();
    if (!keyword) return true;
    const value = String(option?.value || '');
    if (value === 'all') {
      return t('请选择渠道').toLowerCase().includes(keyword);
    }
    const channel = channelOptionMap[value];
    if (!channel) return false;
    return [
      channel.channelName,
      channel.channelNo,
      channel.channelTag,
      channel.routeSlug,
      channel.value,
    ]
      .filter(Boolean)
      .join(' ')
      .toLowerCase()
      .includes(keyword);
  };

  const channelModelPrice = useMemo(
    () => parseJSON(options.ChannelModelPrice),
    [options.ChannelModelPrice],
  );
  const channelModelRatio = useMemo(
    () => parseJSON(options.ChannelModelRatio),
    [options.ChannelModelRatio],
  );
  const channelCompletionRatio = useMemo(
    () => parseJSON(options.ChannelCompletionRatio),
    [options.ChannelCompletionRatio],
  );
  const channelCacheRatio = useMemo(
    () => parseJSON(options.ChannelCacheRatio),
    [options.ChannelCacheRatio],
  );
  const channelCreateCacheRatio = useMemo(
    () => parseJSON(options.ChannelCreateCacheRatio),
    [options.ChannelCreateCacheRatio],
  );
  const channelImageRatio = useMemo(
    () => parseJSON(options.ChannelImageRatio),
    [options.ChannelImageRatio],
  );
  const channelAudioRatio = useMemo(
    () => parseJSON(options.ChannelAudioRatio),
    [options.ChannelAudioRatio],
  );
  const channelAudioCompletionRatio = useMemo(
    () => parseJSON(options.ChannelAudioCompletionRatio),
    [options.ChannelAudioCompletionRatio],
  );
  const channelVideoRatio = useMemo(
    () => parseJSON(options.ChannelVideoRatio),
    [options.ChannelVideoRatio],
  );
  const channelVideoCompletionRatio = useMemo(
    () => parseJSON(options.ChannelVideoCompletionRatio),
    [options.ChannelVideoCompletionRatio],
  );
  const channelVideoPrice = useMemo(
    () => parseJSON(options.ChannelVideoPrice),
    [options.ChannelVideoPrice],
  );
  const channelVideoPricingRules = useMemo(
    () => parseJSON(options.ChannelVideoPricingRules),
    [options.ChannelVideoPricingRules],
  );
  const channelImagePrice = useMemo(
    () => parseJSON(options.ChannelImagePrice),
    [options.ChannelImagePrice],
  );
  const channelImagePricingRules = useMemo(
    () => parseJSON(options.ChannelImagePricingRules),
    [options.ChannelImagePricingRules],
  );
  const channelModelRequestTierPricing = useMemo(
    () => parseJSON(options.ChannelModelRequestTierPricing),
    [options.ChannelModelRequestTierPricing],
  );

  const activeChannelId = channelId === 'all' ? '' : channelId;

  const handleChannelChange = useCallback((value) => {
    setChannelId(String(value || 'all'));
  }, []);

  // parseChannelModelNamesFromChannelData 解析渠道详情中的模型列表（逗号分隔）。
  const parseChannelModelNamesFromChannelData = (channelData) => {
    const raw = String(channelData?.models || '').trim();
    if (!raw) return [];
    return Array.from(
      new Set(
        raw
          .split(',')
          .map((name) => name.trim())
          .filter(Boolean),
      ),
    );
  };

  // loadChannelModelNames 加载指定渠道下配置的模型名，用于渠道模型定价页筛选。
  const loadChannelModelNames = async (targetChannelId) => {
    if (!targetChannelId) return;
    if (channelModelNamesMap[targetChannelId]) return;
    try {
      const res = await API.get(`/api/channel/${targetChannelId}`, {
        skipErrorHandler: true,
      });
      if (!res?.data?.success) {
        showError(res?.data?.message || t('获取渠道模型列表失败'));
        return;
      }
      const names = parseChannelModelNamesFromChannelData(res.data.data);
      setChannelModelNamesMap((prev) => ({
        ...prev,
        [targetChannelId]: names,
      }));
    } catch (error) {
      showError(error?.message || t('获取渠道模型列表失败'));
    }
  };

  // loadSupplierChannelPricingMaps 加载供应商渠道维度定价（表存储）。
  const loadSupplierChannelPricingMaps = useCallback(
    async (targetChannelId) => {
      try {
        const res = await API.get(
          `/api/user/supplier/pricing/channel/${targetChannelId}`,
          {
            skipErrorHandler: true,
          },
        );
        if (!res?.data?.success) {
          showError(res?.data?.message || t('获取渠道定价失败'));
          setSupplierChannelMaps({});
          return;
        }
        setSupplierChannelMaps(res.data.data || {});
      } catch (error) {
        showError(error?.message || t('获取渠道定价失败'));
        setSupplierChannelMaps({});
      }
    },
    [t],
  );

  useEffect(() => {
    if (!activeChannelId) return;
    loadChannelModelNames(activeChannelId);
  }, [activeChannelId]);

  useEffect(() => {
    if (!activeChannelId || !useSupplierPricingApi) {
      setSupplierChannelMaps(null);
      return;
    }
    loadSupplierChannelPricingMaps(activeChannelId);
  }, [activeChannelId, useSupplierPricingApi, loadSupplierChannelPricingMaps]);

  const channelScopedCandidateModelNames = useMemo(() => {
    if (!activeChannelId) return candidateModelNames;
    return channelModelNamesMap[activeChannelId] || [];
  }, [activeChannelId, candidateModelNames, channelModelNamesMap]);

  const scopedOptions = useMemo(() => {
    if (!activeChannelId) return options;
    if (useSupplierPricingApi) {
      const m = supplierChannelMaps || {};
      return {
        ...options,
        ModelPrice: JSON.stringify(m.ModelPrice || {}, null, 2),
        ModelRatio: JSON.stringify(m.ModelRatio || {}, null, 2),
        CompletionRatio: JSON.stringify(m.CompletionRatio || {}, null, 2),
        CacheRatio: JSON.stringify(m.CacheRatio || {}, null, 2),
        CreateCacheRatio: JSON.stringify(m.CreateCacheRatio || {}, null, 2),
        ImageRatio: JSON.stringify(m.ImageRatio || {}, null, 2),
        AudioRatio: JSON.stringify(m.AudioRatio || {}, null, 2),
        AudioCompletionRatio: JSON.stringify(
          m.AudioCompletionRatio || {},
          null,
          2,
        ),
        // 供应商表存储当前尚未承载视频维度价格，视频配置继续走渠道 Option 映射。
        VideoRatio: JSON.stringify(
          channelVideoRatio[activeChannelId] || {},
          null,
          2,
        ),
        VideoCompletionRatio: JSON.stringify(
          channelVideoCompletionRatio[activeChannelId] || {},
          null,
          2,
        ),
        VideoPrice: JSON.stringify(
          channelVideoPrice[activeChannelId] || {},
          null,
          2,
        ),
        VideoPricingRules: JSON.stringify(
          channelVideoPricingRules[activeChannelId] || {},
          null,
          2,
        ),
        ImagePrice: JSON.stringify(
          channelImagePrice[activeChannelId] || {},
          null,
          2,
        ),
        ImagePricingRules: JSON.stringify(
          channelImagePricingRules[activeChannelId] || {},
          null,
          2,
        ),
        ModelRequestTierPricing: JSON.stringify(
          m.ModelRequestTierPricing ||
            channelModelRequestTierPricing[activeChannelId] ||
            {},
          null,
          2,
        ),
      };
    }
    return {
      ...options,
      ModelPrice: JSON.stringify(
        channelModelPrice[activeChannelId] || {},
        null,
        2,
      ),
      ModelRatio: JSON.stringify(
        channelModelRatio[activeChannelId] || {},
        null,
        2,
      ),
      CompletionRatio: JSON.stringify(
        channelCompletionRatio[activeChannelId] || {},
        null,
        2,
      ),
      CacheRatio: JSON.stringify(
        channelCacheRatio[activeChannelId] || {},
        null,
        2,
      ),
      CreateCacheRatio: JSON.stringify(
        channelCreateCacheRatio[activeChannelId] || {},
        null,
        2,
      ),
      ImageRatio: JSON.stringify(
        channelImageRatio[activeChannelId] || {},
        null,
        2,
      ),
      AudioRatio: JSON.stringify(
        channelAudioRatio[activeChannelId] || {},
        null,
        2,
      ),
      AudioCompletionRatio: JSON.stringify(
        channelAudioCompletionRatio[activeChannelId] || {},
        null,
        2,
      ),
      VideoRatio: JSON.stringify(
        channelVideoRatio[activeChannelId] || {},
        null,
        2,
      ),
      VideoCompletionRatio: JSON.stringify(
        channelVideoCompletionRatio[activeChannelId] || {},
        null,
        2,
      ),
      VideoPrice: JSON.stringify(
        channelVideoPrice[activeChannelId] || {},
        null,
        2,
      ),
      VideoPricingRules: JSON.stringify(
        channelVideoPricingRules[activeChannelId] || {},
        null,
        2,
      ),
      ImagePrice: JSON.stringify(
        channelImagePrice[activeChannelId] || {},
        null,
        2,
      ),
      ImagePricingRules: JSON.stringify(
        channelImagePricingRules[activeChannelId] || {},
        null,
        2,
      ),
      ModelRequestTierPricing: JSON.stringify(
        channelModelRequestTierPricing[activeChannelId] || {},
        null,
        2,
      ),
    };
  }, [
    activeChannelId,
    channelAudioCompletionRatio,
    channelAudioRatio,
    channelCacheRatio,
    channelCompletionRatio,
    channelCreateCacheRatio,
    channelImageRatio,
    channelModelPrice,
    channelModelRatio,
    channelModelRequestTierPricing,
    channelVideoCompletionRatio,
    channelVideoPrice,
    channelVideoPricingRules,
    channelImagePrice,
    channelImagePricingRules,
    channelVideoRatio,
    options,
    supplierChannelMaps,
    useSupplierPricingApi,
  ]);

  const handleSaveOutput = async (output) => {
    if (!activeChannelId) {
      throw new Error(t('请先选择渠道'));
    }
    if (useSupplierPricingApi) {
      const res = await API.put(
        `/api/user/supplier/pricing/channel/${activeChannelId}`,
        {
          ModelPrice: output.ModelPrice || {},
          ModelRatio: output.ModelRatio || {},
          CompletionRatio: output.CompletionRatio || {},
          CacheRatio: output.CacheRatio || {},
          CreateCacheRatio: output.CreateCacheRatio || {},
          ImageRatio: output.ImageRatio || {},
          AudioRatio: output.AudioRatio || {},
          AudioCompletionRatio: output.AudioCompletionRatio || {},
          ModelRequestTierPricing: output.ModelRequestTierPricing || {},
        },
      );
      if (!res?.data?.success) {
        throw new Error(res?.data?.message || t('保存失败'));
      }
      // 视频维度价格暂走渠道 Option：确保“视频价格开关/按条或按秒规则”可保存并回显。
      const mergeChannelData = (fullMap, partialMap) => {
        const currentChannelMap =
          fullMap &&
          typeof fullMap === 'object' &&
          fullMap[activeChannelId] &&
          typeof fullMap[activeChannelId] === 'object'
            ? fullMap[activeChannelId]
            : {};
        const nextChannelMap = { ...currentChannelMap };
        channelScopedCandidateModelNames.forEach((modelName) => {
          delete nextChannelMap[modelName];
        });
        return {
          ...fullMap,
          [activeChannelId]: {
            ...nextChannelMap,
            ...(partialMap || {}),
          },
        };
      };
      const optionRequests = [
        [
          'ChannelVideoRatio',
          mergeChannelData(channelVideoRatio, output.VideoRatio),
        ],
        [
          'ChannelVideoCompletionRatio',
          mergeChannelData(
            channelVideoCompletionRatio,
            output.VideoCompletionRatio,
          ),
        ],
        [
          'ChannelVideoPrice',
          mergeChannelData(channelVideoPrice, output.VideoPrice),
        ],
        [
          'ChannelVideoPricingRules',
          mergeChannelData(channelVideoPricingRules, output.VideoPricingRules),
        ],
        [
          'ChannelImagePrice',
          mergeChannelData(channelImagePrice, output.ImagePrice),
        ],
        [
          'ChannelImagePricingRules',
          mergeChannelData(channelImagePricingRules, output.ImagePricingRules),
        ],
        [
          'ChannelModelRequestTierPricing',
          mergeChannelData(
            channelModelRequestTierPricing,
            output.ModelRequestTierPricing,
          ),
        ],
      ].map(([key, value]) =>
        API.put('/api/option/', { key, value: JSON.stringify(value, null, 2) }),
      );
      const optionResults = await Promise.all(optionRequests);
      for (const item of optionResults) {
        if (!item?.data?.success) {
          throw new Error(item?.data?.message || t('保存失败'));
        }
      }
      await loadSupplierChannelPricingMaps(activeChannelId);
      await refresh();
      return;
    }
    const mergeChannelData = (fullMap, partialMap) => {
      const currentChannelMap =
        fullMap &&
        typeof fullMap === 'object' &&
        fullMap[activeChannelId] &&
        typeof fullMap[activeChannelId] === 'object'
          ? fullMap[activeChannelId]
          : {};
      const nextChannelMap = { ...currentChannelMap };
      channelScopedCandidateModelNames.forEach((modelName) => {
        delete nextChannelMap[modelName];
      });
      return {
        ...fullMap,
        [activeChannelId]: {
          ...nextChannelMap,
          ...(partialMap || {}),
        },
      };
    };
    const requestQueue = [
      [
        'ChannelModelPrice',
        mergeChannelData(channelModelPrice, output.ModelPrice),
      ],
      [
        'ChannelModelRatio',
        mergeChannelData(channelModelRatio, output.ModelRatio),
      ],
      [
        'ChannelCompletionRatio',
        mergeChannelData(channelCompletionRatio, output.CompletionRatio),
      ],
      [
        'ChannelCacheRatio',
        mergeChannelData(channelCacheRatio, output.CacheRatio),
      ],
      [
        'ChannelCreateCacheRatio',
        mergeChannelData(channelCreateCacheRatio, output.CreateCacheRatio),
      ],
      [
        'ChannelImageRatio',
        mergeChannelData(channelImageRatio, output.ImageRatio),
      ],
      [
        'ChannelAudioRatio',
        mergeChannelData(channelAudioRatio, output.AudioRatio),
      ],
      [
        'ChannelAudioCompletionRatio',
        mergeChannelData(
          channelAudioCompletionRatio,
          output.AudioCompletionRatio,
        ),
      ],
      [
        'ChannelVideoRatio',
        mergeChannelData(channelVideoRatio, output.VideoRatio),
      ],
      [
        'ChannelVideoCompletionRatio',
        mergeChannelData(
          channelVideoCompletionRatio,
          output.VideoCompletionRatio,
        ),
      ],
      [
        'ChannelVideoPrice',
        mergeChannelData(channelVideoPrice, output.VideoPrice),
      ],
      [
        'ChannelVideoPricingRules',
        mergeChannelData(channelVideoPricingRules, output.VideoPricingRules),
      ],
      [
        'ChannelImagePrice',
        mergeChannelData(channelImagePrice, output.ImagePrice),
      ],
      [
        'ChannelImagePricingRules',
        mergeChannelData(channelImagePricingRules, output.ImagePricingRules),
      ],
      [
        'ChannelModelRequestTierPricing',
        mergeChannelData(
          channelModelRequestTierPricing,
          output.ModelRequestTierPricing,
        ),
      ],
    ].map(([key, value]) =>
      API.put('/api/option/', { key, value: JSON.stringify(value, null, 2) }),
    );
    const results = await Promise.all(requestQueue);
    for (const res of results) {
      if (!res?.data?.success) {
        throw new Error(res?.data?.message || t('保存失败'));
      }
    }
  };

  return (
    <div style={{ width: '100%' }}>
      {!channels.length ? (
        <Empty
          title={t('暂无渠道')}
          description={t('请先在渠道管理创建渠道，再配置渠道模型定价')}
        />
      ) : (
        <>
          <div className='text-sm text-gray-500' style={{ marginBottom: 16 }}>
            {t('渠道模型定价说明')}
          </div>

          <div style={{ marginBottom: 16 }}>
            <div className='mb-1 font-medium text-gray-700'>
              {t('当前渠道')}
            </div>
            <Select
              style={{ width: '100%', maxWidth: 380 }}
              value={channelId}
              onChange={handleChannelChange}
              filter={filterChannelOption}
              searchPosition='dropdown'
              dropdownClassName='supplier-channel-pricing-select-dropdown'
              dropdownStyle={{ minWidth: 380, maxWidth: 420, maxHeight: 360 }}
              optionList={[
                { label: t('请选择渠道'), value: 'all' },
                ...channels,
              ]}
              renderOptionItem={renderChannelOptionItem}
              renderSelectedItem={(optionNode) => {
                const selected = channelOptionMap[optionNode?.value];
                if (!selected) return optionNode?.label || '';
                return renderChannelOption(selected);
              }}
            />
            <div className='mt-1 text-xs text-gray-500'>
              {t('选择渠道后编辑的定价仅作用于该渠道')}
            </div>
          </div>

          {!activeChannelId ? (
            <Empty
              title={t('请选择渠道')}
              description={t('选择后即可编辑该渠道的模型输入/固定价格')}
            />
          ) : (
            <ModelPricingEditor
              options={scopedOptions}
              refresh={refresh}
              candidateModelNames={channelScopedCandidateModelNames}
              forceCandidateModelNames
              filterMode={filterMode}
              listDescription={listDescription}
              onSaveOutput={handleSaveOutput}
              optionKeys={{
                ModelRequestTierPricing: 'ModelRequestTierPricing',
              }}
            />
          )}
        </>
      )}
    </div>
  );
}
