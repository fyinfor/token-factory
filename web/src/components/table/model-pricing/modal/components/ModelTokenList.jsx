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

import React, { useEffect, useRef, useState } from 'react';
import {
  Avatar,
  Button,
  Card,
  Tag,
  Toast,
  Typography,
} from '@douyinfe/semi-ui';
import { IconKey } from '@douyinfe/semi-icons';
import { API, copy } from '../../../../../helpers';
import {
  encodeChannelConnectionString,
  fetchTokenKey as fetchTokenKeyById,
  getServerAddress,
} from '../../../../../helpers/token';
import {
  renderQuotaUsage,
  renderTokenKey,
} from '../../../tokens/TokensColumnDefs';
import { useNavigate } from 'react-router-dom';

const { Text } = Typography;

const ModelTokenList = ({ visible, t }) => {
  const [tokens, setTokens] = useState([]);
  const [tokenCount, setTokenCount] = useState(0);
  const [showKeys, setShowKeys] = useState({});
  const [resolvedTokenKeys, setResolvedTokenKeys] = useState({});
  const [loadingTokenKeys, setLoadingTokenKeys] = useState({});
  const keyRequestsRef = useRef({});
  const navigate = useNavigate();

  useEffect(() => {
    if (!visible) {
      setTokens([]);
      setTokenCount(0);
      setShowKeys({});
      return;
    }
    let cancelled = false;
    (async () => {
      try {
        const res = await API.get('/api/token/?p=1&size=10', {
          skipErrorHandler: true,
        });
        const { success, data } = res.data || {};
        if (!success || cancelled) {
          return;
        }
        const items = Array.isArray(data) ? data : data?.items || [];
        setTokens(items);
        setTokenCount(Array.isArray(data) ? items.length : data?.total || 0);
      } catch (e) {
        if (!cancelled) {
          setTokens([]);
          setTokenCount(0);
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [visible]);

  const goTokenPage = () => {
    navigate('/console/token');
  };

  const copyText = async (text) => {
    if (await copy(text)) {
      Toast.success({ content: t('已复制到剪贴板！') });
    } else {
      Toast.error({ content: t('复制失败') });
    }
  };

  const fetchTokenKey = async (record) => {
    const tokenId = record?.id;
    if (!tokenId) {
      throw new Error(t('令牌不存在'));
    }

    if (resolvedTokenKeys[tokenId]) {
      return resolvedTokenKeys[tokenId];
    }

    if (keyRequestsRef.current[tokenId]) {
      return keyRequestsRef.current[tokenId];
    }

    const request = (async () => {
      setLoadingTokenKeys((prev) => ({ ...prev, [tokenId]: true }));
      try {
        const fullKey = await fetchTokenKeyById(tokenId);
        setResolvedTokenKeys((prev) => ({ ...prev, [tokenId]: fullKey }));
        return fullKey;
      } finally {
        delete keyRequestsRef.current[tokenId];
        setLoadingTokenKeys((prev) => {
          const next = { ...prev };
          delete next[tokenId];
          return next;
        });
      }
    })();

    keyRequestsRef.current[tokenId] = request;
    return request;
  };

  const toggleTokenVisibility = async (record) => {
    const tokenId = record?.id;
    if (!tokenId) {
      return;
    }

    if (showKeys[tokenId]) {
      setShowKeys((prev) => ({ ...prev, [tokenId]: false }));
      return;
    }

    try {
      const fullKey = await fetchTokenKey(record);
      if (fullKey) {
        setShowKeys((prev) => ({ ...prev, [tokenId]: true }));
      }
    } catch (e) {
      Toast.error({ content: e?.message || t('获取令牌密钥失败') });
    }
  };

  const copyTokenKey = async (record) => {
    try {
      const fullKey = await fetchTokenKey(record);
      await copyText(`sk-${fullKey}`);
    } catch (e) {
      Toast.error({ content: e?.message || t('获取令牌密钥失败') });
    }
  };

  const copyTokenConnectionString = async (record) => {
    try {
      const fullKey = await fetchTokenKey(record);
      const connStr = encodeChannelConnectionString(
        `sk-${fullKey}`,
        getServerAddress(),
      );
      await copyText(connStr);
    } catch (e) {
      Toast.error({ content: e?.message || t('获取令牌密钥失败') });
    }
  };

  if (!visible) {
    return null;
  }

  return (
    <Card className='!rounded-2xl shadow-sm border-0 mb-6'>
      <div className='flex items-center justify-between gap-3 mb-3'>
        <div className='flex items-center min-w-0'>
          <Avatar size='small' color='teal' className='mr-2 shadow-md'>
            <IconKey size={16} />
          </Avatar>
          <div>
            <Text className='text-lg font-medium'>{t('我的令牌')}</Text>
            <div className='text-xs text-gray-600'>
              {t('可用于调用上述 API 端点的令牌')}
            </div>
          </div>
        </div>
        <Button size='small' type='tertiary' onClick={goTokenPage}>
          {t('前往令牌管理')}
        </Button>
      </div>
      {tokens.length > 0 ? (
        <div className='space-y-2'>
          {tokens.map((token) => (
            <div
              key={token.id}
              className='flex items-center gap-3 rounded-lg px-3 py-2 overflow-hidden'
              style={{ backgroundColor: 'var(--semi-color-fill-0)' }}
            >
              <div className='min-w-[100px] max-w-[140px] truncate'>
                <Text strong ellipsis={{ showTooltip: true }}>
                  {token.name || `${t('令牌')} #${token.id}`}
                </Text>
              </div>
              <Tag
                size='small'
                color={token.status === 1 ? 'green' : 'grey'}
                shape='circle'
                className='shrink-0'
              >
                {token.status === 1 ? t('启用') : t('禁用')}
              </Tag>
              <div className='shrink-0'>
                {renderTokenKey(
                  token.key,
                  token,
                  showKeys,
                  resolvedTokenKeys,
                  loadingTokenKeys,
                  toggleTokenVisibility,
                  copyTokenKey,
                  copyTokenConnectionString,
                  t,
                )}
              </div>
              <div className='shrink-0 ml-auto'>
                {renderQuotaUsage(token.remain_quota, token, t)}
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className='flex items-center justify-between gap-3 rounded-lg px-3 py-2'>
          <Text type='secondary'>{t('暂无令牌')}</Text>
          <Button type='tertiary' onClick={goTokenPage}>
            {t('前往创建令牌')}
          </Button>
        </div>
      )}
    </Card>
  );
};

export default ModelTokenList;
