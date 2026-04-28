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
} from '../../../../../helpers';
import PricingCardSkeleton from './PricingCardSkeleton';
import { useMinimumLoadingTime } from '../../../../../hooks/common/useMinimumLoadingTime';
import { renderLimitedItems } from '../../../../common/ui/RenderUtils';
import { useIsMobile } from '../../../../../hooks/common/useIsMobile';

const CARD_STYLES = {
  container:
    'w-12 h-12 rounded-2xl flex items-center justify-center relative shadow-md',
  icon: 'w-8 h-8 flex items-center justify-center',
  selected: 'border-blue-500 bg-blue-50',
  default: 'border-gray-200 hover:border-gray-300',
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
}) => {
  const showSkeleton = useMinimumLoadingTime(loading);
  const startIndex = (currentPage - 1) * pageSize;
  const paginatedModels = filteredModels.slice(
    startIndex,
    startIndex + pageSize,
  );
  const getModelKey = (model) => model.key ?? model.model_name ?? model.id;
  const isMobile = useIsMobile();

  const handleCheckboxChange = (model, checked) => {
    if (!setSelectedRowKeys) return;
    const modelKey = getModelKey(model);
    const newKeys = checked
      ? Array.from(new Set([...selectedRowKeys, modelKey]))
      : selectedRowKeys.filter((key) => key !== modelKey);
    setSelectedRowKeys(newKeys);
    rowSelection?.onChange?.(newKeys, null);
  };

  // 获取供应商列表
  const getSupplierIds = (model) => {
    if (!model.channel_list || model.channel_list.length === 0) {
      return [t('官方')];
    }
    const uniqueAliases = [...new Set(model.channel_list.map(ch => ch.supplier_alias).filter(Boolean))];
    return uniqueAliases.length > 0 ? uniqueAliases : [t('官方')];
  };

  // 根据 supplier_type 返回对应的 Tag 颜色
  const getSupplierTypeColor = (supplierType) => {
    switch (supplierType) {
      case '公有云':
        return 'yellow';
      case 'AIDC':
        return 'green';
      case '企业中转站':
        return 'blue';
      case '个人中转站':
        return 'purple';
      default:
        return stringToColor(supplierType);
    }
  };

  // 根据模型的 channel_list 推导可展示的供应商项
  // company_logo_url / supplier_type / supplier_alias 直接从每个 channel 上取，
  // 按 (logo + supplier_type + alias) 组合键去重。
  const getSupplierLogos = (model) => {
    if (!model?.channel_list || model.channel_list.length === 0) return [];
    const seen = new Set();
    const items = [];
    model.channel_list.forEach((ch, idx) => {
      const logo = (ch?.company_logo_url && String(ch.company_logo_url).trim()) || '';
      const supplierType = (ch?.supplier_type && String(ch.supplier_type).trim()) || '';
      const alias = (ch?.supplier_alias && String(ch.supplier_alias).trim()) || '';
      const name = ch?.channel_name || '';
      const dedupKey = `${logo}|${supplierType}|${alias}`;
      if (seen.has(dedupKey)) return;
      seen.add(dedupKey);
      items.push({
        key: ch?.channel_id ?? `${dedupKey}-${idx}`,
        logo,
        supplierType,
        alias,
        name,
      });
    });
    return items;
  };

  // 从 channel_list 计算价格信息（用于按量计费模型）
  const calculateChannelPrices = (model) => {
    if (!model.channel_list || model.channel_list.length === 0 || model.quota_type !== 0) {
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
      const numericPrice = parseFloat(rawDisplayPrice.replace(/[^0-9.]/g, '')) / unitDivisor;
      
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
      
      return { value: parseFloat(numericPrice.toFixed(2)), symbol };
    };

    // 提取所有通道的价格
    const prices = {
      input: [],
      output: [],
      cache: [],
      createCache: [],
    };

    model.channel_list.forEach(ch => {
      if (ch.model_ratio !== undefined && ch.model_ratio !== null) {
        const inputPriceUSD = ch.model_ratio * 2 * usedGroupRatio;
        prices.input.push(formatPrice(inputPriceUSD));

        if (ch.completion_ratio !== undefined && ch.completion_ratio !== null) {
          const outputPriceUSD =
            ch.model_ratio * ch.completion_ratio * 2 * usedGroupRatio;
          prices.output.push(formatPrice(outputPriceUSD));
        }

        if (ch.cache_ratio !== undefined && ch.cache_ratio !== null) {
          const cachePriceUSD =
            ch.model_ratio * ch.cache_ratio * 2 * usedGroupRatio;
          prices.cache.push(formatPrice(cachePriceUSD));
        }

        if (ch.create_cache_ratio !== undefined && ch.create_cache_ratio !== null) {
          const createCachePriceUSD =
            ch.model_ratio * ch.create_cache_ratio * 2 * usedGroupRatio;
          prices.createCache.push(formatPrice(createCachePriceUSD));
        }
      }
    });

    // 根数据价格（用同一口径计算，用于与 channel 价格比较）
    const rootPrices = {};
    if (model.model_ratio !== undefined && model.model_ratio !== null) {
      rootPrices.input = formatPrice(model.model_ratio * 2 * usedGroupRatio);
      if (model.completion_ratio !== undefined && model.completion_ratio !== null) {
        rootPrices.output = formatPrice(
          model.model_ratio * model.completion_ratio * 2 * usedGroupRatio,
        );
      }
      if (model.cache_ratio !== undefined && model.cache_ratio !== null) {
        rootPrices.cache = formatPrice(
          model.model_ratio * model.cache_ratio * 2 * usedGroupRatio,
        );
      }
      if (model.create_cache_ratio !== undefined && model.create_cache_ratio !== null) {
        rootPrices.createCache = formatPrice(
          model.model_ratio * model.create_cache_ratio * 2 * usedGroupRatio,
        );
      }
    }

    // 若根价格高于任意一个 channel 的对应价格，则返回划线原价与折扣
    const getOriginal = (rootPrice, channelPriceArray) => {
      if (!rootPrice || !channelPriceArray || channelPriceArray.length === 0) return null;
      const minChannel = Math.min(...channelPriceArray.map((p) => p.value));
      if (rootPrice.value > minChannel && rootPrice.value > 0) {
        const discount = Math.round((1 - minChannel / rootPrice.value) * 100);
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
        return { single: `${p.symbol}${p.value}`, min: null, max: null, symbol: p.symbol };
      }

      const values = priceArray.map(p => p.value);
      const uniqueValues = [...new Set(values)];
      
      if (uniqueValues.length === 1) {
        const p = priceArray[0];
        return { single: `${p.symbol}${p.value}`, min: null, max: null, symbol: p.symbol };
      }

      const min = Math.min(...values);
      const max = Math.max(...values);
      const symbol = priceArray[0].symbol;
      return { single: null, min, max, symbol };
    };

    const unitLabel = tokenUnit === 'K' ? 'K' : 'M';
    const unitSuffix = ` / 1${unitLabel} Tokens`;

    return {
      input: calculateRange(prices.input),
      output: calculateRange(prices.output),
      cache: calculateRange(prices.cache),
      createCache: calculateRange(prices.createCache),
      original: {
        input: getOriginal(rootPrices.input, prices.input),
        output: getOriginal(rootPrices.output, prices.output),
        cache: getOriginal(rootPrices.cache, prices.cache),
        createCache: getOriginal(rootPrices.createCache, prices.createCache),
      },
      unitSuffix,
    };
  };

  // 获取模型的价格项（优先使用 channel 价格）
  const getModelPriceItemsForCard = (model, priceData) => {
    const channelPrices = calculateChannelPrices(model);
    
    // 如果没有 channel 价格，使用原有逻辑
    if (!channelPrices) {
      return getModelPriceItems(priceData, t, siteDisplayType);
    }

    // 使用 channel 价格构建价格项
    const items = [];
    const { input, output, cache, createCache, original, unitSuffix } = channelPrices;

    if (input) {
      items.push({
        key: 'input',
        label: t('输入价格'),
        value: input.single || `${input.symbol}${input.min} ~ ${input.symbol}${input.max}`,
        suffix: unitSuffix,
        original: original?.input,
      });
    }

    if (output) {
      items.push({
        key: 'output',
        label: t('输出价格'),
        value: output.single || `${output.symbol}${output.min} ~ ${output.symbol}${output.max}`,
        suffix: unitSuffix,
        original: original?.output,
      });
    }

    // if (cache) {
    //   items.push({
    //     key: 'cache',
    //     label: t('缓存读取价格'),
    //     value: cache.single || `${cache.symbol}${cache.min} ~ ${cache.symbol}${cache.max}`,
    //     suffix: unitSuffix,
    //     original: original?.cache,
    //   });
    // }

    // if (createCache) {
    //   items.push({
    //     key: 'create-cache',
    //     label: t('缓存创建价格'),
    //     value: createCache.single || `${createCache.symbol}${createCache.min} ~ ${createCache.symbol}${createCache.max}`,
    //     suffix: unitSuffix,
    //     original: original?.createCache,
    //   });
    // }

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

    const avatarText = model.model_name.slice(0, 2).toUpperCase();
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
  const getModelDescription = (record) => {
    return record.description || '';
  };

  // 渲染标签
  const renderTags = (record) => {
    // 计费类型标签（左边）
    let billingTag = (
      <Tag key='billing' shape='circle' color='white' size='small'>
        -
      </Tag>
    );
    if (record.quota_type === 1) {
      billingTag = (
        <Tag key='billing' shape='circle' color='teal' size='small'>
          {t('按次计费')}
        </Tag>
      );
    } else if (record.quota_type === 0) {
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
        customTags.push(
          <Tag
            key={`custom-${idx}`}
            shape='circle'
            color={stringToColor(tg)}
            size='small'
          >
            {tg}
          </Tag>,
        );
      });
    }

    return (
      <div className='flex items-center justify-between'>
        <div className='flex items-center gap-2'>{billingTag}</div>
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
      <PricingCardSkeleton
        rowSelection={!!rowSelection}
        showRatio={showRatio}
      />
    );
  }

  if (!filteredModels || filteredModels.length === 0) {
    return (
      <div className='flex justify-center items-center py-20'>
        <Empty
          image={<IllustrationNoResult style={{ width: 150, height: 150 }} />}
          darkModeImage={
            <IllustrationNoResultDark style={{ width: 150, height: 150 }} />
          }
          description={t('搜索无结果')}
        />
      </div>
    );
  }

  return (
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

          const supplierIds = getSupplierIds(model);
          const supplierLogos = getSupplierLogos(model);

          return (
            <Card
              key={modelKey || index}
              className={`flex-1 min-w-[350px] max-w-[600px] !rounded-2xl transition-all duration-200 hover:shadow-lg border cursor-pointer ${isSelected ? CARD_STYLES.selected : CARD_STYLES.default}`}
              bodyStyle={{ height: '100%' }}
              onClick={() => openModelDetail && openModelDetail(model)}
            >
              <div className='flex flex-col h-full'>
                {/* 头部：图标 + 模型名称 + 操作按钮 */}
                <div className='flex items-start justify-between mb-3'>
                  <div className='flex items-start space-x-3 flex-1 min-w-0'>
                    {getModelIcon(model)}
                    <div className='flex-1 min-w-0'>
                      <h3 className='text-lg font-bold text-gray-900 truncate'>
                        {model.model_name}
                      </h3>
                      <div className='flex flex-col gap-1 text-xs mt-1'>
                        {getModelPriceItemsForCard(model, priceData).map((item) => (
                          <div key={item.key} className='flex items-center'>
                            <span className='w-20 flex-shrink-0'>{item.label}</span>
                            <span className='flex-1 font-bold text-black inline-flex items-center flex-wrap gap-1'>
                              {item.original && (
                                <>
                                  <span className='line-through text-gray-400 font-normal text-[10px]'>
                                    {item.original.text}
                                  </span>
                                  <Tag color='red' size='small' shape='circle'>
                                    -{item.original.discount}%
                                  </Tag>
                                </>
                              )}
                              <span>{item.value}{item.suffix}</span>
                            </span>
                          </div>
                        ))}
                        <div className='flex items-center'>
                          <span className='w-20 flex-shrink-0'>{t('供应商')}</span>
                          <div className='flex-1 flex items-center flex-wrap gap-1'>
                            {supplierLogos.length === 0 ? (
                              <span className='font-bold text-black'>{supplierIds.join(', ')}</span>
                            ) : (
                              supplierLogos.map((s) => (
                                <div
                                  key={s.key}
                                  className='h-7 rounded-md flex items-center gap-1 overflow-hidden'
                                  style={{
                                    backgroundColor:
                                      'var(--semi-color-fill-0)',
                                    paddingRight: s.supplierType ? 4 : 0,
                                  }}
                                >
                                  {s.logo ? (
                                    <img
                                      src={s.logo}
                                      alt={s.alias || s.name || ''}
                                      className='w-7 h-7 object-contain rounded-md'
                                    />
                                  ) : (
                                    <span
                                      className='h-6 px-2 flex items-center text-xs font-medium'
                                      style={{
                                        color: 'var(--semi-color-text-1)',
                                      }}
                                    >
                                      {s.alias || s.name || t('官方')}
                                    </span>
                                  )}
                                  {s.supplierType && (
                                    <Tag
                                      size='small'
                                      shape='circle'
                                      color={getSupplierTypeColor(s.supplierType)}
                                    >
                                      {s.supplierType}
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
                <div className='flex-1 mb-4'>
                  <p
                    className='text-xs line-clamp-2 leading-relaxed'
                    style={{ color: 'var(--semi-color-text-2)' }}
                  >
                    {getModelDescription(model)}
                  </p>
                </div>

                {/* 底部区域 */}
                <div className='mt-auto'>
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
                            ? priceData?.inputRatio ?? model.model_ratio
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
  );
};

export default PricingCardView;
