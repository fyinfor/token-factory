import React, { useMemo, useState } from 'react';
import { Empty, Select } from '@douyinfe/semi-ui';
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

export default function SupplierModelPricingEditor({
  options,
  refresh,
  candidateModelNames = [],
  filterMode = 'all',
  listDescription = '',
}) {
  const { t } = useTranslation();
  const [channelId, setChannelId] = useState('all');

  const channels = useMemo(() => {
    const raw = options.__pricingChannels || [];
    if (!Array.isArray(raw)) return [];
    return raw.filter((s) => s?.channel_id).map((s) => ({
      label: s.channel_name || `#${s.channel_id}`,
      value: String(s.channel_id),
      }));
  }, [options.__pricingChannels]);

  const channelModelPrice = useMemo(() => parseJSON(options.ChannelModelPrice), [
    options.ChannelModelPrice,
  ]);
  const channelModelRatio = useMemo(() => parseJSON(options.ChannelModelRatio), [
    options.ChannelModelRatio,
  ]);
  const channelCompletionRatio = useMemo(
    () => parseJSON(options.ChannelCompletionRatio),
    [options.ChannelCompletionRatio],
  );
  const channelCacheRatio = useMemo(() => parseJSON(options.ChannelCacheRatio), [
    options.ChannelCacheRatio,
  ]);
  const channelCreateCacheRatio = useMemo(
    () => parseJSON(options.ChannelCreateCacheRatio),
    [options.ChannelCreateCacheRatio],
  );
  const channelImageRatio = useMemo(() => parseJSON(options.ChannelImageRatio), [
    options.ChannelImageRatio,
  ]);
  const channelAudioRatio = useMemo(() => parseJSON(options.ChannelAudioRatio), [
    options.ChannelAudioRatio,
  ]);
  const channelAudioCompletionRatio = useMemo(
    () => parseJSON(options.ChannelAudioCompletionRatio),
    [options.ChannelAudioCompletionRatio],
  );

  const activeChannelId = channelId === 'all' ? '' : channelId;

  const scopedOptions = useMemo(() => {
    if (!activeChannelId) return options;
    return {
      ...options,
      ModelPrice: JSON.stringify(channelModelPrice[activeChannelId] || {}, null, 2),
      ModelRatio: JSON.stringify(channelModelRatio[activeChannelId] || {}, null, 2),
      CompletionRatio: JSON.stringify(
        channelCompletionRatio[activeChannelId] || {},
        null,
        2,
      ),
      CacheRatio: JSON.stringify(channelCacheRatio[activeChannelId] || {}, null, 2),
      CreateCacheRatio: JSON.stringify(
        channelCreateCacheRatio[activeChannelId] || {},
        null,
        2,
      ),
      ImageRatio: JSON.stringify(channelImageRatio[activeChannelId] || {}, null, 2),
      AudioRatio: JSON.stringify(channelAudioRatio[activeChannelId] || {}, null, 2),
      AudioCompletionRatio: JSON.stringify(
        channelAudioCompletionRatio[activeChannelId] || {},
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
    options,
  ]);

  const handleSaveOutput = async (output) => {
    if (!activeChannelId) {
      throw new Error(t('请先选择渠道'));
    }
    const mergeChannelData = (fullMap, partialMap) => ({
      ...fullMap,
      [activeChannelId]: partialMap || {},
    });
    const requestQueue = [
      ['ChannelModelPrice', mergeChannelData(channelModelPrice, output.ModelPrice)],
      ['ChannelModelRatio', mergeChannelData(channelModelRatio, output.ModelRatio)],
      [
        'ChannelCompletionRatio',
        mergeChannelData(channelCompletionRatio, output.CompletionRatio),
      ],
      ['ChannelCacheRatio', mergeChannelData(channelCacheRatio, output.CacheRatio)],
      [
        'ChannelCreateCacheRatio',
        mergeChannelData(channelCreateCacheRatio, output.CreateCacheRatio),
      ],
      ['ChannelImageRatio', mergeChannelData(channelImageRatio, output.ImageRatio)],
      ['ChannelAudioRatio', mergeChannelData(channelAudioRatio, output.AudioRatio)],
      [
        'ChannelAudioCompletionRatio',
        mergeChannelData(channelAudioCompletionRatio, output.AudioCompletionRatio),
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
            <div className='mb-1 font-medium text-gray-700'>{t('当前渠道')}</div>
            <Select
              style={{ width: '100%', maxWidth: 420 }}
              value={channelId}
              onChange={setChannelId}
              optionList={[
                { label: t('请选择渠道'), value: 'all' },
                ...channels,
              ]}
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
              candidateModelNames={candidateModelNames}
              filterMode={filterMode}
              listDescription={listDescription}
              onSaveOutput={handleSaveOutput}
            />
          )}
        </>
      )}
    </div>
  );
}
