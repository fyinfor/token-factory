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
  Input,
} from '@douyinfe/semi-ui';
import {
  IconChevronDown,
  IconChevronUp,
  IconKey,
  IconCopy,
  IconEyeClosed,
  IconEyeOpened,
} from '@douyinfe/semi-icons';
import { API, copy } from '../../../../../helpers';
import { fetchTokenKey as fetchTokenKeyById } from '../../../../../helpers/token';
import { renderQuotaUsage } from '../../../tokens/TokensColumnDefs';
import { useNavigate } from 'react-router-dom';

const { Text } = Typography;

const StepTitle = ({ label, title, desc }) => (
  <div className='flex items-start gap-3'>
    <div
      className='flex items-center justify-center gap-1.5 shrink-0 rounded-full font-semibold text-xs px-3'
      style={{
        height: 30,
        width: 84,
        color: 'var(--semi-color-bg-0)',
        backgroundColor: 'var(--semi-color-primary)',
        boxShadow: '0 6px 14px rgba(var(--semi-blue-5), 0.24)',
      }}
    >
      <IconKey size={14} />
      {label}
    </div>
    <div className='min-w-0'>
      <div className='flex items-center gap-1'>
        <Text className='text-lg font-medium'>{title}</Text>
      </div>
      <div className='text-xs text-gray-600 mt-0.5'>{desc}</div>
    </div>
  </div>
);

/** 令牌数量达到该阈值时，列表支持折叠 */
const TOKEN_LIST_COLLAPSE_THRESHOLD = 2;

/** 模型详情侧栏「我的令牌」列表，2 个及以上时可折叠 */
const ModelTokenList = ({
  visible,
  t,
  stepLabel,
  title,
  description,
  flat = false,
  showLoginPrompt = false,
}) => {
  const [tokens, setTokens] = useState([]);
  const [tokenCount, setTokenCount] = useState(0);
  const [showKeys, setShowKeys] = useState({});
  const [resolvedTokenKeys, setResolvedTokenKeys] = useState({});
  const [loadingTokenKeys, setLoadingTokenKeys] = useState({});
  const [listExpanded, setListExpanded] = useState(false);
  const keyRequestsRef = useRef({});
  const navigate = useNavigate();

  const canCollapse = !flat && tokens.length >= TOKEN_LIST_COLLAPSE_THRESHOLD;

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
    setListExpanded(true);
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
      await fetchTokenKey(record);
      setShowKeys((prev) => ({ ...prev, [tokenId]: true }));
    } catch (e) {
      Toast.error({ content: e?.message || t('获取令牌密钥失败') });
    }
  };

  /** 复制 API Key */
  const copyTokenKey = async (record) => {
    try {
      const fullKey = await fetchTokenKey(record);
      await copyText(`sk-${fullKey}`);
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
          className='grid min-w-0 grid-cols-1 gap-2 rounded-lg px-3 py-2 sm:grid-cols-[minmax(90px,130px)_auto_minmax(0,1fr)_auto] sm:items-center'
          style={{ backgroundColor: 'var(--semi-color-fill-0)' }}
        >
          {(() => {
            const tokenId = token.id;
            const revealed = !!showKeys[tokenId];
            const fullKey = resolvedTokenKeys[tokenId]
              ? `sk-${resolvedTokenKeys[tokenId]}`
              : '';
            return (
              <>
                <div className='min-w-0 truncate'>
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
                <div className='flex min-w-0 items-center gap-2'>
                  <Input
                    className='min-w-0 flex-1'
                    readOnly
                    size='small'
                    value={revealed && fullKey ? fullKey : 'sk-********'}
                    suffix={
                      <Button
                        theme='borderless'
                        size='small'
                        type='tertiary'
                        icon={revealed ? <IconEyeClosed /> : <IconEyeOpened />}
                        loading={!!loadingTokenKeys[token.id]}
                        aria-label={t('查看API Key')}
                        onClick={(e) => {
                          e.stopPropagation();
                          toggleTokenVisibility(token);
                        }}
                      />
                    }
                  />
                  <Button
                    size='small'
                    type='primary'
                    theme='light'
                    icon={<IconCopy />}
                    loading={!!loadingTokenKeys[token.id]}
                    aria-label={t('复制API Key')}
                    onClick={(e) => {
                      e.stopPropagation();
                      copyTokenKey(token);
                    }}
                  >
                    {t('复制')}
                  </Button>
                </div>
                <div className='min-w-0 sm:ml-auto'>
                  {renderQuotaUsage(token.remain_quota, token, t)}
                </div>
              </>
            );
          })()}
        </div>
      ))}
    </div>
  );

  if (!visible && !showLoginPrompt) {
    return null;
  }

  const loginPrompt = (
    <>
      <div className='mb-3'>
        <StepTitle
          label={stepLabel || t('第三步')}
          title={title || t('复制API Key')}
          desc={t('登录后可查看并复制 API Key')}
        />
      </div>
      <div className='model-login-prompt flex items-center justify-between gap-3 rounded-lg px-3 py-2.5'>
        <Text type='secondary'>{t('请先登录后继续')}</Text>
        <Button
          className='model-login-prompt-button'
          type='primary'
          theme='light'
          onClick={() => navigate('/login')}
        >
          {t('去登录')}
        </Button>
      </div>
    </>
  );

  if (!visible) {
    return flat ? (
      <section className='mb-6 pb-3'>{loginPrompt}</section>
    ) : (
      <Card className='!rounded-2xl shadow-sm border-0 mb-6'>
        {loginPrompt}
      </Card>
    );
  }

  const content = (
    <>
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
          {stepLabel ? (
            <StepTitle
              label={stepLabel}
              title={title || t('复制API Key')}
              desc={
                canCollapse && !listExpanded
                  ? renderTokenCountLabel()
                  : description || t('复制可用于调用上述 API 端点的 API Key')
              }
            />
          ) : (
            <>
              <Avatar size='small' color='teal' className='mr-2 shadow-md'>
                <IconKey size={16} />
              </Avatar>
              <div>
                <div className='flex items-center gap-1'>
                  <Text className='text-lg font-medium'>
                    {t('复制API Key')}
                  </Text>
                </div>
                <div className='text-xs text-gray-600'>
                  {canCollapse && !listExpanded
                    ? renderTokenCountLabel()
                    : t('复制可用于调用上述 API 端点的 API Key')}
                </div>
              </div>
            </>
          )}
          {stepLabel && canCollapse ? (
            <span className='ml-1'>
              {canCollapse ? (
                listExpanded ? (
                  <IconChevronUp size='small' className='text-gray-500' />
                ) : (
                  <IconChevronDown size='small' className='text-gray-500' />
                )
              ) : null}
            </span>
          ) : null}
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
    </>
  );

  if (flat) {
    return (
      <section className='mb-6 border-b border-semi-color-border pb-6'>
        {content}
      </section>
    );
  }

  return (
    <Card className='!rounded-2xl shadow-sm border-0 mb-6'>{content}</Card>
  );
};

export default ModelTokenList;
