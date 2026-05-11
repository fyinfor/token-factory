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
  Avatar,
  Typography,
  Badge,
  Button,
  Tooltip,
  Toast,
} from '@douyinfe/semi-ui';
import { IconCopy, IconLink } from '@douyinfe/semi-icons';
import { copy } from '../../../../../helpers';
import { getServerAddress } from '../../../../../helpers/token';

const { Text } = Typography;

const normalizeApiBaseUrl = (baseUrl) =>
  String(baseUrl || '').replace(
    /^https:\/\/demo\.tokenfactoryopen\.com/i,
    'https://tokenfactoryopen.com',
  );

const ModelEndpoints = ({ modelData, endpointMap = {}, t }) => {
  const getApiEndpointLink = (path) => {
    try {
      return `${normalizeApiBaseUrl(getServerAddress()).replace(/\/+$/, '')}${path}`;
    } catch (e) {
      return path;
    }
  };

  const copyEndpoint = async (path) => {
    const endpoint = getApiEndpointLink(path);
    if (await copy(endpoint)) {
      Toast.success({ content: t('已复制API端点') });
    } else {
      Toast.error({ content: t('复制失败') });
    }
  };

  const renderAPIEndpoints = () => {
    if (!modelData) return null;

    const mapping = endpointMap;
    const types = modelData.supported_endpoint_types || [];

    return types.map((type) => {
      const info = mapping[type] || {};
      let path = info.path || '';
      // 如果路径中包含 {model} 占位符，替换为真实模型名称
      if (path.includes('{model}')) {
        const modelName = modelData.model_name || modelData.modelName || '';
        path = path.replaceAll('{model}', modelName);
      }
      const method = info.method || 'POST';
      return (
        <div
          key={type}
          className='flex justify-between border-b border-dashed last:border-0 py-2 last:pb-0'
          style={{ borderColor: 'var(--semi-color-border)' }}
        >
          <span className='flex items-center pr-5 min-w-0 flex-1'>
            <Badge dot type='success' className='mr-2' />
            {type}
            {path && '：'}
            {path && (
              <span className='text-gray-500 md:ml-1 break-all'>{path}</span>
            )}
            {path && (
              <Tooltip content={getApiEndpointLink(path)}>
                <Button
                  size='small'
                  type='tertiary'
                  className='ml-2'
                  onClick={() => copyEndpoint(path)}
                >
                  {t('复制')}
                </Button>
              </Tooltip>
            )}
          </span>
          {path && (
            <span className='text-gray-500 text-xs md:ml-1'>{method}</span>
          )}
        </div>
      );
    });
  };

  return (
    <Card className='!rounded-2xl shadow-sm border-0 mb-6'>
      <div className='flex items-center mb-4'>
        <Avatar size='small' color='purple' className='mr-2 shadow-md'>
          <IconLink size={16} />
        </Avatar>
        <div>
          <Text className='text-lg font-medium'>{t('API端点')}</Text>
          <div className='text-xs text-gray-600'>
            {t('模型支持的接口端点信息')}
          </div>
        </div>
      </div>
      {renderAPIEndpoints()}
    </Card>
  );
};

export default ModelEndpoints;
