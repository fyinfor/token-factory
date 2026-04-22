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

import React, { useMemo, useState, useEffect, useRef } from 'react';
import { Card, Avatar, Typography, Collapse, Tag, Button, Toast } from '@douyinfe/semi-ui';
import { IconListView, IconCopy } from '@douyinfe/semi-icons';

const { Text } = Typography;

const ModelChannelList = ({ modelData, displayPrice, currency, siteDisplayType, tokenUnit, t }) => {
  if (!modelData?.channel_list || modelData.channel_list.length === 0) {
    return null;
  }

  // 按 supplier_application_id 分组通道
  const groupedChannels = useMemo(() => {
    const groups = {};
    modelData.channel_list.forEach((channel) => {
      const supplierId = channel.supplier_application_id;
      if (!groups[supplierId]) {
        groups[supplierId] = {
          supplierId,
          supplierAlias: channel.supplier_alias || t('未知供应商'),
          channels: [],
        };
      }
      groups[supplierId].channels.push(channel);
    });
    return Object.values(groups);
  }, [modelData.channel_list, t]);

  // 生成所有面板的 keys，默认全部展开
  const allKeys = useMemo(() =>
    groupedChannels.map(group => `group-${group.supplierId}`)
  , [groupedChannels]);

  // 使用字符串形式来稳定比较
  const allKeysStr = allKeys.join(',');
  const prevKeysStr = useRef('');

  // 管理展开状态
  const [activeKey, setActiveKey] = useState(allKeys);

  // 当 allKeys 实际变化时（基于字符串比较），更新 activeKey
  useEffect(() => {
    if (allKeysStr !== prevKeysStr.current) {
      setActiveKey(allKeys);
      prevKeysStr.current = allKeysStr;
    }
  }, [allKeysStr, allKeys]);

  // 格式化通道信息
  const formatChannelInfo = (channel) => {
    const firstRow = [];
    const secondRow = [];
    
    // 计算价格的辅助函数
    const calculatePrice = (ratio) => {
      const priceUSD = ratio * 2; // 按量计费的标准计算方式
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
      
      const unitLabel = tokenUnit === 'K' ? 'K' : 'M';
      return `${symbol}${parseFloat(numericPrice.toFixed(2))} / 1${unitLabel} Tokens`;
    };
    
    // 第一行：输入价格和输出价格
    if (channel.model_ratio !== undefined && channel.model_ratio !== null) {
      firstRow.push({ 
        label: t('输入价格'), 
        value: calculatePrice(channel.model_ratio)
      });
    }
    
    if (channel.model_ratio !== undefined && channel.model_ratio !== null &&
        channel.completion_ratio !== undefined && channel.completion_ratio !== null) {
      const outputRatio = channel.model_ratio * channel.completion_ratio;
      firstRow.push({ 
        label: t('输出价格'), 
        value: calculatePrice(outputRatio)
      });
    }
    
    // 第二行：缓存读取价格和缓存创建价格
    if (channel.model_ratio !== undefined && channel.model_ratio !== null &&
        channel.cache_ratio !== undefined && channel.cache_ratio !== null) {
      const cacheRatio = channel.model_ratio * channel.cache_ratio;
      secondRow.push({ 
        label: t('缓存读取价格'), 
        value: calculatePrice(cacheRatio)
      });
    }
    
    if (channel.model_ratio !== undefined && channel.model_ratio !== null &&
        channel.create_cache_ratio !== undefined && channel.create_cache_ratio !== null) {
      const createCacheRatio = channel.model_ratio * channel.create_cache_ratio;
      secondRow.push({ 
        label: t('缓存创建价格'), 
        value: calculatePrice(createCacheRatio)
      });
    }
    
    return { firstRow, secondRow };
  };

  return (
    <Card className='!rounded-2xl shadow-sm border-0 mb-6'>
      <div className='flex items-center mb-4'>
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
      
      <Collapse activeKey={activeKey} onChange={setActiveKey}>
        {groupedChannels.map((group) => (
          <Collapse.Panel
            key={`group-${group.supplierId}`}
            itemKey={`group-${group.supplierId}`}
            header={
              <div className='flex items-center justify-between w-full pr-4'>
                <span className='font-medium'>
                  <Tag color='blue' size='small' className='ml-2'>
                    {group.supplierAlias}
                  </Tag>
                </span>
                <span className='text-sm text-gray-500'>
                  {group.channels.length} {t('个通道')}
                </span>
              </div>
            }
          >
            <div className='space-y-3'>
              {group.channels.map((channel, idx) => {
                const channelInfo = formatChannelInfo(channel);
                const channelPath = `${channel.supplier_alias}/${modelData.model_name}/${channel.channel_no}`;
                
                const handleCopy = () => {
                  navigator.clipboard.writeText(channelPath).then(() => {
                    Toast.success({ content: t('已复制通道') });
                  }).catch(() => {
                    Toast.error({ content: t('复制失败') });
                  });
                };
                
                return (
                  <div key={`${channel.channel_id}-${idx}`} className='flex gap-3 items-start'>
                    <div className='flex items-center justify-center min-w-[24px] h-[24px] rounded-full bg-blue-100 text-blue-600 text-xs font-semibold mt-3'>
                      {channel.channel_no}
                    </div>
                    <Card
                      className='!rounded-lg shadow-sm !mb-2 flex-1'
                      bodyStyle={{ padding: '12px' }}
                    >
                      <div className='flex items-center justify-between gap-4'>
                        <div className='flex flex-col gap-2 text-sm flex-1'>
                          {/* 第一行：输入和输出价格 */}
                          {channelInfo.firstRow.length > 0 && (
                            <div className='flex flex-wrap gap-4'>
                              {channelInfo.firstRow.map((item) => (
                                <div key={item.label} className='flex items-center gap-2 grow'>
                                  <span className='text-gray-600'>{item.label}:</span>
                                  <span className='font-medium text-gray-900'>{item.value}</span>
                                </div>
                              ))}
                            </div>
                          )}
                          {/* 第二行：缓存价格 */}
                          {channelInfo.secondRow.length > 0 && (
                            <div className='flex flex-wrap gap-4'>
                              {channelInfo.secondRow.map((item) => (
                                <div key={item.label} className='flex items-center gap-2 grow'>
                                  <span className='text-gray-600'>{item.label}:</span>
                                  <span className='font-medium text-gray-900'>{item.value}</span>
                                </div>
                              ))}
                            </div>
                          )}
                        </div>
                        <Button
                          icon={<IconCopy />}
                          size='small'
                          type='tertiary'
                          onClick={handleCopy}
                          title={channelPath}
                        />
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
  );
};

export default ModelChannelList;
