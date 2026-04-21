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

import React, { useMemo } from 'react';
import { Card, Avatar, Typography, Collapse, Tag } from '@douyinfe/semi-ui';
import { IconListView } from '@douyinfe/semi-icons';

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

  // 格式化通道信息
  const formatChannelInfo = (channel) => {
    const items = [];
    
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
    
    // 计算并显示输入价格（基于 model_ratio）
    if (channel.model_ratio !== undefined && channel.model_ratio !== null) {
      items.push({ 
        label: t('输入价格'), 
        value: calculatePrice(channel.model_ratio)
      });
    }
    
    // 计算并显示输出价格（基于 model_ratio * completion_ratio）
    if (channel.model_ratio !== undefined && channel.model_ratio !== null &&
        channel.completion_ratio !== undefined && channel.completion_ratio !== null) {
      const outputRatio = channel.model_ratio * channel.completion_ratio;
      items.push({ 
        label: t('输出价格'), 
        value: calculatePrice(outputRatio)
      });
    }
    
    return items;
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
      
      <Collapse accordion defaultActiveKey={groupedChannels.length > 0 ? `group-${groupedChannels[0].supplierId}` : undefined}>
        {groupedChannels.map((group) => (
          <Collapse.Panel
            key={`group-${group.supplierId}`}
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
                return (
                  <div key={`${channel.channel_id}-${idx}`} className='flex gap-3 items-start'>
                    <div className='flex items-center justify-center min-w-[24px] h-[24px] rounded-full bg-blue-100 text-blue-600 text-xs font-semibold mt-3'>
                      {channel.channel_no}
                    </div>
                    <Card
                      className='!rounded-lg shadow-sm !mb-2 flex-1'
                      bodyStyle={{ padding: '12px' }}
                    >
                      <div className='flex flex-wrap gap-4 text-sm'>
                        {channelInfo.map((item) => (
                          <div key={item.label} className='flex items-center gap-2 grow'>
                            <span className='text-gray-600'>{item.label}:</span>
                            <span className='font-medium text-gray-900'>{item.value}</span>
                          </div>
                        ))}
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
