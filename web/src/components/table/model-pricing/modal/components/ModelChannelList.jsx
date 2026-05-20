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

import React, { useMemo, useState, useEffect, useRef, useContext } from 'react';
import {
  Card,
  Avatar,
  Typography,
  Collapse,
  Tag,
  Button,
  Toast,
  Tooltip,
} from '@douyinfe/semi-ui';
import { IconListView } from '@douyinfe/semi-icons';
import { copy, stringToColor } from '../../../../../helpers';
import { getUsedGroupContext } from '../../../../../helpers/utils';
import { UserContext } from '../../../../../context/User';
import ApiDocsSidePanel from './ApiDocsSidePanel';
import ModelTokenList from './ModelTokenList';
import VideoFlatClipHintTable from '../../components/VideoFlatClipHintTable';
import ImagePerImageHintTable from '../../components/ImagePerImageHintTable';
import {
  pickVideoFlatClipHintForChannel,
  hasVideoFlatClipTierTable,
} from '../../constants/videoFlatClipLaneI18n';
import {
  pickImagePerImageHintForChannel,
  hasImagePerImageTierTable,
} from '../../constants/imagePerImageHintI18n';

import { renderModelTestResultSummary } from '../../../../../helpers/modelStability';

const { Text } = Typography;

const copyText = async (text, t, successText = '已复制') => {
  if (await copy(text)) {
    Toast.success({ content: t(successText) });
  } else {
    Toast.error({ content: t('复制失败') });
  }
};

const hasRatioValue = (value) =>
  value !== undefined &&
  value !== null &&
  value !== '' &&
  Number.isFinite(Number(value));

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

const ModelChannelList = ({
  modelData,
  channelMtrMap = {},
  displayPrice,
  currency,
  siteDisplayType,
  tokenUnit,
  t,
  selectedGroup,
  groupRatio,
  blurPricing = false,
}) => {
  const [userState] = useContext(UserContext);
  const [docsVisible, setDocsVisible] = useState(false);
  const [docsModelName, setDocsModelName] = useState('');
  const channelList = modelData?.channel_list || [];
  const isLoggedIn = Boolean(userState?.user);

  const { usedGroupRatio } = useMemo(
    () =>
      getUsedGroupContext(modelData, selectedGroup ?? 'all', groupRatio || {}),
    [modelData, selectedGroup, groupRatio],
  );

  // 按 supplier_application_id 分组通道
  const groupedChannels = useMemo(() => {
    const groups = {};
    channelList.forEach((channel) => {
      const supplierId = channel.supplier_application_id;
      if (!groups[supplierId]) {
        groups[supplierId] = {
          supplierId,
          supplierAlias:
            (channel?.supplier_alias &&
              String(channel.supplier_alias).trim()) ||
            '',
          companyLogoUrl:
            (channel?.company_logo_url &&
              String(channel.company_logo_url).trim()) ||
            '',
          supplierType:
            (channel?.supplier_type && String(channel.supplier_type).trim()) ||
            '',
          channels: [],
        };
      }
      groups[supplierId].channels.push(channel);
    });
    return Object.values(groups);
  }, [channelList, t]);

  // 生成所有面板的 keys，默认全部展开
  const allKeys = useMemo(
    () => groupedChannels.map((group) => `group-${group.supplierId}`),
    [groupedChannels],
  );

  // 使用字符串形式来稳定比较
  const allKeysStr = allKeys.join(',');
  const prevKeysStr = useRef('');

  // 管理展开状态
  const [activeKey, setActiveKey] = useState(allKeys);

  const openApiDocs = (channelModelName) => {
    setDocsModelName(channelModelName || modelData?.model_name || '');
    setDocsVisible(true);
  };

  // 当 allKeys 实际变化时（基于字符串比较），更新 activeKey
  useEffect(() => {
    if (allKeysStr !== prevKeysStr.current) {
      setActiveKey(allKeys);
      prevKeysStr.current = allKeysStr;
    }
  }, [allKeysStr, allKeys]);

  // 格式化通道信息（新计费公式：含分组倍率、成本折扣、加价折扣）
  const formatChannelInfo = (channel) => {
    // 判断计费类型：优先使用 channel.quota_type，否则使用 modelData.quota_type
    const quotaType =
      channel.quota_type !== undefined
        ? channel.quota_type
        : modelData?.quota_type;
    const isPerToken = quotaType === 0; // 0=按量计费, 1=按次计费

    // ============================================================
    // 新计费公式参数：ch.model_ratio / ch.model_price 为原始渠道倍率（后端不再预乘成本折扣）
    //   成本折扣率 = price_discount_percent / 100
    //   加价倍率   = markup_discount_rate / 100
    //
    //   输入   = (ch.model_ratio × costDisc + globalMr × markupRate) × 2 × groupRatio
    //   输出   = (ch.model_ratio × cr × costDisc + globalMr × globalCR × markupRate) × 2 × groupRatio
    //   缓存读 = (ch.model_ratio × cacheRatio × costDisc + globalMr × globalCacheR × markupRate) × 2 × groupRatio
    //   缓存写 = (ch.model_ratio × createCacheRatio × costDisc + globalMr × globalCreateCacheR × markupRate) × 2 × groupRatio
    //   固定价 = (ch.model_price × costDisc + globalMp × markupRate) × groupRatio
    // ============================================================
    const costDisc = (channel.price_discount_percent != null ? channel.price_discount_percent : 100) / 100;
    const markupRate = (channel.markup_discount_rate || 0) / 100;
    const globalMr = modelData?.model_ratio || 0;
    const globalMp = modelData?.model_price || 0;
    const globalCR = modelData?.completion_ratio || 0;
    const globalCacheR =
      modelData?.cache_ratio != null ? Number(modelData.cache_ratio) : 0;
    const globalCreateCacheR =
      modelData?.create_cache_ratio != null
        ? Number(modelData.create_cache_ratio)
        : 0;

    // 计算价格，返回 { display, value }
    const calculatePrice = (
      nominalRatio,
      isFixedPrice = false,
      applyGroupRatio = true,
    ) => {
      let priceUSD;
      const ratio = applyGroupRatio ? usedGroupRatio : 1;
      if (isFixedPrice) {
        // 按次计费：直接使用价格
        priceUSD = nominalRatio * ratio;
      } else {
        // 按量计费：有效倍率 × 2 × 分组倍率
        priceUSD = nominalRatio * 2 * ratio;
      }
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

      const value = parseFloat(numericPrice.toFixed(2));
      if (isFixedPrice) {
        return {
          display: `${symbol}${value} / ${t('次')}`,
          value,
        };
      } else {
        const unitLabel = tokenUnit === 'K' ? 'K' : 'M';
        return {
          display: `${symbol}${value} / 1${unitLabel} Tokens`,
          value,
        };
      }
    };

    // 构造单条价格项，若全局价格高于有效通道价格则附带原价与折扣
    const makeItem = (label, channelValue, rootValue, isFixedPrice = false) => {
      if (!hasRatioValue(channelValue)) return null;
      const current = calculatePrice(Number(channelValue), isFixedPrice);
      let original = null;
      let discount = 0;
      if (
        hasRatioValue(rootValue) &&
        Number(rootValue) > Number(channelValue)
      ) {
        const root = calculatePrice(Number(rootValue), isFixedPrice, false);
        const channelOriginal = calculatePrice(
          Number(channelValue),
          isFixedPrice,
          false,
        );
        if (root.value > channelOriginal.value && root.value > 0) {
          discount = Math.round((1 - channelOriginal.value / root.value) * 100);
          original = root.display;
        }
      }
      return { label, value: current.display, original, discount };
    };

    const items = [];

    // 按次计费
    if (isPerToken === false) {
      // 固定价：ch.model_price × costDisc + globalMp × markupRate
      const effModelPrice = hasRatioValue(channel.model_price)
        ? Number(channel.model_price) * costDisc + globalMp * markupRate
        : null;
      items.push(makeItem(t('模型价格'), effModelPrice, modelData?.model_price, true));
    }
    // 按量计费
    else {
      // 输入：ch.model_ratio × costDisc + globalMr × markupRate
      const effInputRate = hasRatioValue(channel.model_ratio)
        ? Number(channel.model_ratio) * costDisc + globalMr * markupRate
        : null;
      items.push(makeItem(t('输入价格'), effInputRate, modelData?.model_ratio, false));

      // 输出价格：仅当全局模型配置了 completion_ratio 时才展示
      if (
        hasRatioValue(channel.model_ratio) &&
        hasRatioValue(channel.completion_ratio) &&
        hasRatioValue(modelData?.completion_ratio)
      ) {
        const effOut =
          Number(channel.model_ratio) * Number(channel.completion_ratio) * costDisc +
          globalMr * globalCR * markupRate;
        const rootOut = hasRatioValue(modelData?.model_ratio)
          ? Number(modelData.model_ratio) * Number(modelData.completion_ratio)
          : null;
        items.push(makeItem(t('输出价格'), effOut, rootOut, false));
      }

      // 缓存读取价格：仅当全局模型配置了 cache_ratio 时才展示
      if (
        hasRatioValue(channel.model_ratio) &&
        hasRatioValue(channel.cache_ratio) &&
        hasRatioValue(modelData?.cache_ratio)
      ) {
        const effCacheRate =
          Number(channel.model_ratio) * Number(channel.cache_ratio) * costDisc +
          globalMr * globalCacheR * markupRate;
        const rootC = hasRatioValue(modelData?.model_ratio)
          ? Number(modelData.model_ratio) * Number(modelData.cache_ratio)
          : null;
        items.push(makeItem(t('缓存读取价格'), effCacheRate, rootC, false));
      }

      // 缓存创建价格：仅当全局模型配置了 create_cache_ratio 时才展示
      if (
        hasRatioValue(channel.model_ratio) &&
        hasRatioValue(channel.create_cache_ratio) &&
        hasRatioValue(modelData?.create_cache_ratio)
      ) {
        const effCreateCacheRate =
          Number(channel.model_ratio) * Number(channel.create_cache_ratio) * costDisc +
          globalMr * globalCreateCacheR * markupRate;
        const rootCC = hasRatioValue(modelData?.model_ratio)
          ? Number(modelData.model_ratio) * Number(modelData.create_cache_ratio)
          : null;
        items.push(makeItem(t('缓存创建价格'), effCreateCacheRate, rootCC, false));
      }
    }
    return items.filter(Boolean);
  };

  if (channelList.length === 0) {
    return null;
  }

  return (
    <>
      <Card className='!rounded-2xl shadow-sm border-0 mb-3'>
        <div className='flex items-center mb-4'>
          <div className='flex items-center min-w-0'>
            <Avatar size='small' color='purple' className='mr-2 shadow-md'>
              <IconListView size={16} />
            </Avatar>
            <div>
              <Text className='text-lg font-medium'>{t('通道列表')}</Text>
              <div className='text-xs text-gray-600'>
                {t('模型在各个通道的配置信息')}
              </div>
            </div>
          </div>
        </div>

        <Collapse activeKey={activeKey} onChange={setActiveKey}>
          {groupedChannels.map((group) => (
            <Collapse.Panel
              key={`group-${group.supplierId}`}
              itemKey={`group-${group.supplierId}`}
              header={
                <div className='flex items-center justify-between w-full'>
                  {group.companyLogoUrl || group.supplierType ? (
                    <span
                      className='h-7 rounded-md flex items-center overflow-hidden ml-2'
                      style={{backgroundColor: 'var(--semi-color-fill-0)'}}
                    >
                      {group.companyLogoUrl ? (
                        <img
                          src={group.companyLogoUrl}
                          alt={group.supplierAlias || ''}
                          className='w-7 h-7 object-contain rounded-md'
                        />
                      ) : null}
                      {group.supplierType && (
                        <Tag
                          size='small'
                          shape='circle'
                          color={getSupplierTypeColor(group.supplierType)}
                          className='mx-1'
                        >
                          {group.supplierType}
                        </Tag>
                      )}
                    </span>
                  ) : null}
                  <span className='text-sm text-gray-500'>
                    {group.channels.length} {t('个通道')}
                  </span>
                </div>
              }
            >
              <div className='space-y-3'>
                {group.channels.map((channel, idx) => {
                  const channelItems = formatChannelInfo(channel);
                  const vHint = pickVideoFlatClipHintForChannel(
                    modelData,
                    channel,
                  );
                  const showVideoFlatTable = hasVideoFlatClipTierTable(vHint);
                  const iHint = pickImagePerImageHintForChannel(
                    modelData,
                    channel,
                  );
                  const showImagePerImageTable =
                    hasImagePerImageTierTable(iHint);
                  const channelPath = channel.route_slug
                    ? `${modelData.model_name}/${channel.route_slug}`
                    : `${channel.supplier_alias}/${modelData.model_name}/${channel.channel_no}`;
                  const channelBadge =
                    channel.route_slug || channel.channel_no || String(idx);

                  const handleCopy = () => {
                    copyText(channelPath, t, '已复制通道');
                  };

                  return (
                    <div
                      key={`${channel.channel_id}-${idx}`}
                      className='flex gap-3 items-start'
                    >
                      <div className='flex items-center justify-center min-w-[24px] h-[24px] rounded-full bg-blue-100 text-blue-600 text-xs font-semibold mt-1 shrink-0'>
                        {channelBadge}
                      </div>
                      <Card
                        className='!rounded-lg shadow-sm !mb-2 flex-1'
                        bodyStyle={{ padding: '10px' }}
                      >
                        <div className='flex flex-col gap-1 text-sm'>
                          <div className='flex items-start justify-between gap-2'>
                            <div className='flex flex-wrap gap-2 items-center min-w-0 flex-1'>
                              <Text type='tertiary' size='small'>
                                {t('单测/稳定性')}
                              </Text>
                              {renderModelTestResultSummary(
                                channelMtrMap[String(channel.channel_id)],
                                t,
                              )}
                            </div>
                            <div className='flex flex-wrap gap-2 items-center shrink-0 ml-1'>
                              <Tooltip content={t('复制通道路径')}>
                                <Button
                                  theme='solid'
                                  type='primary'
                                  size='small'
                                  onClick={handleCopy}
                                  title={channelPath}
                                >
                                  {t('复制')}
                                </Button>
                              </Tooltip>
                              <Tooltip content={t('查看 API 文档')}>
                                <Button
                                  theme='light'
                                  type='warning'
                                  size='small'
                                  onClick={() => openApiDocs(channelPath)}
                                >
                                  {t('文档')}
                                </Button>
                              </Tooltip>
                            </div>
                          </div>
                          <div className='h-px bg-gray-100' />
                          {channelItems.map((item) => (
                            <div
                              key={item.label}
                              className='flex items-center gap-2 flex-wrap'
                            >
                              <span className='text-gray-600'>
                                {item.label}:
                              </span>
                              {item.original ? (
                                <>
                                  <span className='text-gray-400 line-through text-xs'>
                                    <span
                                      style={{
                                        color: 'var(--semi-color-primary)',
                                      }}
                                    >
                                      官方
                                    </span>{' '}
                                    {item.original}
                                  </span>
                                  <Tag color='red' size='small' shape='circle'>
                                    -{item.discount}%
                                  </Tag>
                                  <span className='font-medium text-gray-900'>
                                    <span
                                      style={{
                                        color: 'var(--semi-color-warning)',
                                      }}
                                    >
                                      我们
                                    </span>{' '}
                                    {item.value}
                                  </span>
                                </>
                              ) : (
                                <span className='font-medium text-gray-900'>
                                  {item.value}
                                </span>
                              )}
                            </div>
                          ))}
                          {showVideoFlatTable ? (
                            <VideoFlatClipHintTable
                              hint={vHint}
                              usedGroupRatio={usedGroupRatio}
                              displayPrice={displayPrice}
                              t={t}
                              blurPricing={blurPricing}
                            />
                          ) : null}
                          {showImagePerImageTable ? (
                            <ImagePerImageHintTable
                              hint={iHint}
                              usedGroupRatio={usedGroupRatio}
                              displayPrice={displayPrice}
                              t={t}
                              blurPricing={blurPricing}
                            />
                          ) : null}
                        </div>
                      </Card>
                    </div>
                  );
                })}
              </div>
            </Collapse.Panel>
          ))}
        </Collapse>
      </Card>
      <ModelTokenList visible={isLoggedIn} t={t} />
      <ApiDocsSidePanel
        visible={docsVisible}
        onClose={() => {
          setDocsVisible(false);
          setDocsModelName('');
        }}
        modelName={docsModelName || modelData?.model_name}
        docIntroduction={modelData?.doc_introduction}
        apiDocs={modelData?.api_docs}
        t={t}
      />
    </>
  );
};

export default ModelChannelList;
