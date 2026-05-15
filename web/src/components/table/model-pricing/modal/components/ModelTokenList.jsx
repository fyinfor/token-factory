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
import { IconChevronDown, IconChevronUp, IconKey } from '@douyinfe/semi-icons';
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

/** 令牌数量达到该阈值时，列表支持折叠 */
const TOKEN_LIST_COLLAPSE_THRESHOLD = 2;

/** 模型详情侧栏「我的令牌」列表，2 个及以上时可折叠 */
const ModelTokenList = ({ visible, t }) => {
  const [tokens, setTokens] = useState([]);
  const [tokenCount, setTokenCount] = useState(0);
  const [showKeys, setShowKeys] = useState({});
  const [resolvedTokenKeys, setResolvedTokenKeys] = useState({});
  const [loadingTokenKeys, setLoadingTokenKeys] = useState({});
  const [listExpanded, setListExpanded] = useState(false);
  const keyRequestsRef = useRef({});
  const navigate = useNavigate();

  const canCollapse = tokens.length >= TOKEN_LIST_COLLAPSE_THRESHOLD;

  useEffect(() => {
    if (!visible) {
      setTokens([]);
      setTokenCount(0);
      setShowKeys({});
      setListExpanded(false);
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

  /** 令牌加载完成后，按数量决定默认展开状态 */
  useEffect(() => {
    if (tokens.length < TOKEN_LIST_COLLAPSE_THRESHOLD) {
      setListExpanded(true);
    } else {
      setListExpanded(false);
    }
  }, [tokens.length]);

  /** 跳转到令牌管理页 */
  const goTokenPage = () => {
    navigate('/console/token');
  };

  /** 切换令牌列表折叠状态 */
  const toggleListExpanded = () => {
    if (!canCollapse) {
      return;
    }
    setListExpanded((prev) => !prev);
  };

  /** 复制文本到剪贴板 */
  const copyText = async (text) => {
    if (await copy(text)) {
      Toast.success({ content: t('已复制到剪贴板！') });
    } else {
      Toast.error({ content: t('复制失败') });
    }
  };

  /** 按需拉取并缓存令牌完整密钥 */
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

  /** 切换单条令牌的密钥可见性 */
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

  /** 复制令牌密钥 */
  const copyTokenKey = async (record) => {
    try {
      const fullKey = await fetchTokenKey(record);
      await copyText(`sk-${fullKey}`);
    } catch (e) {
      Toast.error({ content: e?.message || t('获取令牌密钥失败') });
    }
  };

  /** 复制令牌连接串 */
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
  /** 渲染令牌数量摘要文案 */
  const renderTokenCountLabel = () => {
    const count = tokenCount > tokens.length ? tokenCount : tokens.length;
    return t('共 {{count}} 个令牌', { count });
  };

  /** 渲染令牌行列表 */
  const renderTokenRows = () => (
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
  );


  if (!visible) {
    return null;
  }

  return (
    <Card className='!rounded-2xl shadow-sm border-0 mb-6'>
      <div className='flex items-center justify-between gap-3 mb-3'>
        <div
          className={`flex items-center min-w-0 ${canCollapse ? 'cursor-pointer' : ''}`}
          onClick={toggleListExpanded}
          role={canCollapse ? 'button' : undefined}
          tabIndex={canCollapse ? 0 : undefined}
          onKeyDown={(e) => {
            if (canCollapse && (e.key === 'Enter' || e.key === ' ')) {
              e.preventDefault();
              toggleListExpanded();
            }
          }}
        >
          <Avatar size='small' color='teal' className='mr-2 shadow-md'>
            <IconKey size={16} />
          </Avatar>
          <div>
            <div className='flex items-center gap-1'>
              <Text className='text-lg font-medium'>{t('我的令牌')}</Text>
              {canCollapse ? (
                listExpanded ? (
                  <IconChevronUp size='small' className='text-gray-500' />
                ) : (
                  <IconChevronDown size='small' className='text-gray-500' />
                )
              ) : null}
            </div>
            <div className='text-xs text-gray-600'>
              {canCollapse && !listExpanded
                ? renderTokenCountLabel()
                : t('可用于调用上述 API 端点的令牌')}
            </div>
          </div>
        </div>
        <Button
          size='small'
          type='tertiary'
          onClick={(e) => {
            e.stopPropagation();
            goTokenPage();
          }}
        >
          {t('前往令牌管理')}
        </Button>
      </div>
      {tokens.length > 0 ? (
        !canCollapse || listExpanded ? (
          renderTokenRows()
        ) : null
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
