import React, { useState, useEffect, useContext, useCallback, useMemo, useRef } from 'react';
import {
  Avatar,
  Button,
  Card,
  Collapsible,
  Input,
  InputNumber,
  Radio,
  RadioGroup,
  Switch,
  Table,
  Tag,
  Typography,
  TagInput,
  Spin,
  Toast,
} from '@douyinfe/semi-ui';
import { Route, ChevronDown, ChevronRight, Plus, Trash2, Info, Search, GripVertical } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { API, showSuccess, showError } from '../../../../helpers';
import { getCurrencyConfig } from '../../../../helpers/render';
import { UserContext } from '../../../../context/User';
import { StatusContext } from '../../../../context/Status';

const ROUTE_MODES = [
  { value: 'default', label_key: 'route_policy.mode_default' },
  { value: 'weight', label_key: 'route_policy.mode_weight' },
  { value: 'price', label_key: 'route_policy.mode_price' },
];

const computeWeightsFromOrder = (channels) => {
  const total = channels.length;
  return channels.map((channel, index) => ({
    channel,
    weight: Math.max(10, (total - index) * 10),
  }));
};

const resolveEffectiveEnabled = (channel) => {
  if (channel.user_configured) return channel.user_enabled;
  if (channel.global_configured) return channel.global_enabled;
  return true;
};

const resolveDisplayWeight = (channel) => {
  if (channel.user_configured) return channel.user_weight || 0;
  if (channel.global_configured) return channel.global_weight || 0;
  return 0;
};

const applyOrderWeightsToChannels = (channels) =>
  computeWeightsFromOrder(channels).map(({ channel, weight }) => ({
    ...channel,
    user_weight: weight,
    user_configured: true,
    user_enabled: resolveEffectiveEnabled(channel),
  }));

const fuzzyMatchModelQuery = (query, model, groupKey, displayName) => {
  const q = query.trim().toLowerCase();
  if (!q) return false;
  const haystacks = [model, groupKey, displayName].filter(Boolean).map((s) => s.toLowerCase());
  return haystacks.some((text) => text.includes(q));
};

const RoutePolicyCard = ({ t }) => {
  const { t: translate } = useTranslation();
  const [userState] = useContext(UserContext);
  const [statusState] = useContext(StatusContext);
  const isAdmin = (userState?.user?.role || 0) >= 10;
  const statusLoaded = statusState?.status != null;
  const routeEnabled = statusState?.status?.tokenfactory_route_enabled === true;

  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState('');
  const [mode, setMode] = useState('');
  const [globalMode, setGlobalMode] = useState('default');
  const [groups, setGroups] = useState([]);
  const [userOverrides, setUserOverrides] = useState([]);
  const [globalOverrides, setGlobalOverrides] = useState([]);
  const [expandedGroups, setExpandedGroups] = useState({});
  const [savingMode, setSavingMode] = useState(false);
  const [newOverrideModel, setNewOverrideModel] = useState('');
  const [newOverrideGroup, setNewOverrideGroup] = useState('');
  const [addingOverride, setAddingOverride] = useState(false);
  const [modelSearch, setModelSearch] = useState('');
  const [highlightedGroupKey, setHighlightedGroupKey] = useState('');
  const groupRefs = useRef({});

  const tLocal = useCallback(
    (key) => t ? t(key) : translate(key),
    [t, translate]
  );

  const fetchPolicy = useCallback(async ({ silent = false } = {}) => {
    if (!silent) {
      setLoading(true);
      setLoadError('');
    }
    try {
      const res = await API.get('/api/user/route-policy');
      const { data } = res;
      if (data.success !== false) {
        setMode(data.mode ?? '');
        setGlobalMode(data.global_mode ?? 'default');
        setGroups(data.groups ?? []);
        setUserOverrides(data.user_overrides ?? []);
        setGlobalOverrides(data.global_overrides ?? []);
      } else if (!silent) {
        const msg = data.error || tLocal('route_policy.load_failed');
        setLoadError(msg);
        showError(msg);
      }
    } catch (err) {
      if (!silent) {
        const msg = tLocal('route_policy.load_failed_with_reason', {
          reason: err.response?.data?.error || err.message || String(err),
        });
        setLoadError(msg);
        showError(msg);
      }
    } finally {
      if (!silent) {
        setLoading(false);
      }
    }
  }, [tLocal]);

  useEffect(() => {
    if (!statusLoaded) {
      return;
    }
    if (!routeEnabled) {
      setLoading(false);
      setLoadError('');
      return;
    }
    fetchPolicy();
  }, [statusLoaded, routeEnabled, fetchPolicy]);

  // mode === '' 时跟随系统全局模式；界面直接展示 effectiveMode。
  const effectiveMode = mode === '' ? globalMode : mode;

  const handleModeChange = async (rawValue) => {
    const newMode = rawValue?.target?.value ?? rawValue;
    if (!newMode || newMode === effectiveMode) return;

    const prevMode = mode;
    setMode(newMode);
    setSavingMode(true);
    try {
      const res = await API.put('/api/user/route-policy/mode', { mode: newMode });
      if (res.data.success) {
        showSuccess(tLocal('route_policy.mode_updated'));
        await fetchPolicy({ silent: true });
      } else {
        showError(res.data.error || tLocal('route_policy.save_failed'));
        setMode(prevMode);
      }
    } catch (err) {
      showError(
        err.response?.data?.error || tLocal('route_policy.save_failed'),
      );
      setMode(prevMode);
    } finally {
      setSavingMode(false);
    }
  };

  const handleWeightChange = async (groupKey, channelID, weight, enabled, { quiet = false } = {}) => {
    try {
      const res = await API.post('/api/user/route-policy/weights', {
        group_key: groupKey,
        channel_id: channelID,
        weight,
        enabled,
      });
      if (res.data.success) {
        if (!quiet) {
          showSuccess(tLocal('route_policy.weight_updated'));
          fetchPolicy({ silent: true });
        }
        return true;
      }
      if (!quiet) {
        showError(res.data.error || tLocal('route_policy.save_failed'));
      }
      return false;
    } catch (err) {
      if (!quiet) {
        showError(tLocal('route_policy.save_failed'));
      }
      return false;
    }
  };

  const handleBatchWeightChange = async (groupKey, updates) => {
    let changed = 0;
    for (const item of updates) {
      const ok = await handleWeightChange(
        groupKey,
        item.channel_id,
        item.weight,
        item.enabled,
        { quiet: true },
      );
      if (ok) changed += 1;
    }
    if (changed > 0) {
      showSuccess(tLocal('route_policy.weight_updated'));
      fetchPolicy({ silent: true });
    }
  };

  const handleDeleteWeight = async (weightID) => {
    try {
      const res = await API.delete(`/api/user/route-policy/weights/${weightID}`);
      if (res.data.success) {
        showSuccess(tLocal('route_policy.weight_deleted'));
        fetchPolicy({ silent: true });
      } else {
        showError(res.data.error || tLocal('route_policy.delete_failed'));
      }
    } catch (err) {
      showError(tLocal('route_policy.delete_failed'));
    }
  };

  const handleAddOverride = async () => {
    if (!newOverrideModel.trim() || !newOverrideGroup.trim()) {
      showError(tLocal('route_policy.override_required'));
      return;
    }
    setAddingOverride(true);
    try {
      const res = await API.post('/api/user/route-policy/overrides', {
        raw_model: newOverrideModel.trim(),
        group_key: newOverrideGroup.trim(),
      });
      if (res.data.success) {
        showSuccess(tLocal('route_policy.override_added'));
        setNewOverrideModel('');
        setNewOverrideGroup('');
        fetchPolicy({ silent: true });
      } else {
        showError(res.data.error || tLocal('route_policy.save_failed'));
      }
    } catch (err) {
      showError(tLocal('route_policy.save_failed'));
    } finally {
      setAddingOverride(false);
    }
  };

  const handleDeleteOverride = async (overrideID) => {
    try {
      const res = await API.delete(`/api/user/route-policy/overrides/${overrideID}`);
      if (res.data.success) {
        showSuccess(tLocal('route_policy.override_deleted'));
        fetchPolicy({ silent: true });
      } else {
        showError(res.data.error || tLocal('route_policy.delete_failed'));
      }
    } catch (err) {
      showError(tLocal('route_policy.delete_failed'));
    }
  };

  const toggleGroup = (groupKey) => {
    setExpandedGroups((prev) => ({ ...prev, [groupKey]: !prev[groupKey] }));
  };

  const modeHint = () => {
    switch (effectiveMode) {
      case 'default':
        return tLocal('route_policy.mode_default_hint');
      case 'weight':
        return tLocal('route_policy.mode_weight_hint');
      case 'price':
        return tLocal('route_policy.mode_price_hint');
      default:
        return tLocal('route_policy.custom_mode_hint');
    }
  };

  const overrideColumns = [
    {
      title: tLocal('route_policy.raw_model'),
      dataIndex: 'raw_model',
      key: 'raw_model',
      width: '40%',
    },
    {
      title: tLocal('route_policy.group_key'),
      dataIndex: 'group_key',
      key: 'group_key',
      width: '30%',
    },
    {
      title: tLocal('route_policy.scope'),
      dataIndex: 'is_user',
      key: 'is_user',
      width: '15%',
      render: (isUser) => (
        <Tag color={isUser ? 'blue' : 'grey'} size='small'>
          {isUser ? tLocal('route_policy.my_override') : tLocal('route_policy.global_override')}
        </Tag>
      ),
    },
    {
      title: '',
      key: 'action',
      width: '15%',
      render: (_, record) =>
        record.is_user ? (
          <Button
            type='danger'
            size='small'
            icon={<Trash2 size={14} />}
            onClick={() => handleDeleteOverride(record.id)}
          />
        ) : null,
    },
  ];

  const allOverrides = [
    ...userOverrides.map((o) => ({ ...o, is_user: true })),
    ...globalOverrides.map((o) => ({ ...o, is_user: false })),
  ];

  const modelIndex = useMemo(() => {
    const items = [];
    groups.forEach((group) => {
      const displayName = group.display_name || group.group_key;
      (group.models || []).forEach((model) => {
        items.push({
          model,
          groupKey: group.group_key,
          displayName,
        });
      });
    });
    return items;
  }, [groups]);

  const modelSearchResults = useMemo(() => {
    const query = modelSearch.trim();
    if (!query) return [];
    return modelIndex
      .filter((item) =>
        fuzzyMatchModelQuery(query, item.model, item.groupKey, item.displayName),
      )
      .slice(0, 80);
  }, [modelIndex, modelSearch]);

  const jumpToModelGroup = (groupKey, modelName = '') => {
    setExpandedGroups((prev) => ({ ...prev, [groupKey]: true }));
    setHighlightedGroupKey(groupKey);
    if (modelName) {
      setModelSearch(modelName);
    }
    window.setTimeout(() => {
      groupRefs.current[groupKey]?.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
    }, 120);
    window.setTimeout(() => setHighlightedGroupKey(''), 1800);
  };

  // 路由能力关闭时不渲染该卡片（须放在所有 hooks 之后，避免 hooks 数量变化报错导致白屏）。
  if (statusLoaded && !routeEnabled) {
    return null;
  }

  return (
    <Card className='!rounded-2xl shadow-sm border-0'>
      {/* Card Header */}
      <div className='flex items-center mb-4'>
        <Avatar size='small' color='green' className='mr-3 shadow-md'>
          <Route size={16} />
        </Avatar>
        <div>
          <Typography.Text className='text-lg font-medium'>
            {tLocal('route_policy.title')}
          </Typography.Text>
          <div className='text-xs text-gray-600 dark:text-gray-400'>
            {tLocal('route_policy.description')}
          </div>
        </div>
      </div>

      {loading ? (
        <div className='flex justify-center py-8'>
          <Spin />
        </div>
      ) : (
        <>
          {loadError && (
            <div className='mb-4 p-3 rounded-lg bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 text-sm flex items-center justify-between gap-2'>
              <span>{loadError}</span>
              <Button size='small' onClick={fetchPolicy}>
                {tLocal('route_policy.retry')}
              </Button>
            </div>
          )}
          {/* Route Mode Selection */}
          <Card className='!rounded-xl border dark:border-gray-700 mb-4'>
            <div className='flex flex-col gap-3'>
              <div className='flex items-center gap-2'>
                <div className='w-12 h-12 rounded-full bg-green-50 dark:bg-green-900/30 flex items-center justify-center mr-4 flex-shrink-0'>
                  <Route size={20} className='text-green-600 dark:text-green-400' />
                </div>
                <div>
                  <Typography.Title heading={6} className='mb-1'>
                    {tLocal('route_policy.mode_label')}
                  </Typography.Title>
                  <Typography.Text type='tertiary' className='text-sm'>
                    {tLocal('route_policy.mode_desc')}
                  </Typography.Text>
                </div>
              </div>
              <div className='ml-16'>
                <RadioGroup
                  type='button'
                  value={effectiveMode}
                  onChange={handleModeChange}
                  disabled={savingMode}
                >
                  {ROUTE_MODES.map((m) => (
                    <Radio key={m.value} value={m.value}>
                      {tLocal(m.label_key)}
                    </Radio>
                  ))}
                </RadioGroup>
                <div className='mt-2'>
                  <div className='text-xs text-gray-500 dark:text-gray-400 flex items-center gap-1'>
                    <Info size={12} />
                    {modeHint()}
                  </div>
                </div>
              </div>
            </div>
          </Card>

          {/* Model Groups with Channels */}
          {(effectiveMode === 'weight' || effectiveMode === 'price') && groups.length > 0 && (
            <Card className='!rounded-xl border dark:border-gray-700 mb-4'>
              <div className='flex flex-col gap-3 mb-3 lg:flex-row lg:items-start lg:justify-between'>
                <div>
                  <Typography.Title heading={6}>
                    {tLocal('route_policy.model_groups')}
                  </Typography.Title>
                  <Typography.Text type='tertiary' size='small' className='block mt-1'>
                    {effectiveMode === 'weight'
                      ? tLocal('route_policy.drag_sort_hint')
                      : tLocal('route_policy.mode_price_hint')}
                  </Typography.Text>
                </div>
                <div className='w-full lg:w-72 shrink-0'>
                  <Typography.Text size='small' className='mb-1 block'>
                    {tLocal('route_policy.model_search')}
                  </Typography.Text>
                  <Input
                    prefix={<Search size={14} className='text-gray-400' />}
                    value={modelSearch}
                    onChange={setModelSearch}
                    placeholder={tLocal('route_policy.model_search_placeholder')}
                    size='small'
                    showClear
                  />
                  <Typography.Text type='tertiary' size='small' className='mt-1 block'>
                    {tLocal('route_policy.model_search_hint')}
                  </Typography.Text>
                  {modelSearch.trim() ? (
                    <div className='mt-2 max-h-56 overflow-y-auto rounded-lg border dark:border-gray-600 bg-gray-50/80 dark:bg-gray-900/40'>
                      {modelSearchResults.length > 0 ? (
                        modelSearchResults.map((item) => (
                          <button
                            key={`${item.groupKey}-${item.model}`}
                            type='button'
                            className='w-full px-3 py-2 text-left text-sm hover:bg-white dark:hover:bg-gray-800 border-b last:border-b-0 dark:border-gray-700'
                            onClick={() => jumpToModelGroup(item.groupKey, item.model)}
                          >
                            <div className='font-medium truncate'>{item.model}</div>
                            <div className='text-xs text-gray-500 dark:text-gray-400 truncate'>
                              {item.displayName}
                              <span className='mx-1'>·</span>
                              <code>{item.groupKey}</code>
                            </div>
                          </button>
                        ))
                      ) : (
                        <div className='px-3 py-4 text-xs text-gray-500 dark:text-gray-400 text-center'>
                          {tLocal('route_policy.model_search_empty')}
                        </div>
                      )}
                    </div>
                  ) : null}
                </div>
              </div>
              <div className='space-y-2'>
                {groups.map((group) => (
                  <div
                    key={group.group_key}
                    ref={(el) => {
                      groupRefs.current[group.group_key] = el;
                    }}
                    className={`border dark:border-gray-600 rounded-lg overflow-hidden transition-colors ${
                      highlightedGroupKey === group.group_key
                        ? 'ring-2 ring-green-400/70 dark:ring-green-500/50'
                        : ''
                    }`}
                  >
                    <div
                      className='flex items-center justify-between px-4 py-3 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800'
                      onClick={() => toggleGroup(group.group_key)}
                    >
                      <div className='flex items-center gap-2 min-w-0'>
                        {expandedGroups[group.group_key] ? (
                          <ChevronDown size={16} />
                        ) : (
                          <ChevronRight size={16} />
                        )}
                        <Typography.Text strong className='truncate'>
                          {group.display_name || group.group_key}
                        </Typography.Text>
                        <Tag size='small' color='blue'>
                          {group.channel_count} {tLocal('route_policy.channels')}
                        </Tag>
                        <Typography.Text type='tertiary' size='small'>
                          {group.models.length} {tLocal('route_policy.models')}
                        </Typography.Text>
                      </div>
                    </div>
                    <Collapsible isOpen={expandedGroups[group.group_key]}>
                      <div className='px-4 pb-3'>
                        <div className='text-xs text-gray-500 dark:text-gray-400 mb-2'>
                          {tLocal('route_policy.group_models')}: {group.models.slice(0, 10).join(', ')}
                          {group.models.length > 10 ? '...' : ''}
                        </div>
                        <WeightChannelTable
                          groupKey={group.group_key}
                          channels={
                            effectiveMode === 'price'
                              ? [...group.channels].sort(
                                  (a, b) => (a.price || 0) - (b.price || 0),
                                )
                              : group.channels
                          }
                          routeMode={effectiveMode}
                          isAdmin={isAdmin}
                          onWeightChange={handleWeightChange}
                          onBatchWeightChange={handleBatchWeightChange}
                          onDeleteWeight={handleDeleteWeight}
                          onSaved={() => fetchPolicy({ silent: true })}
                          t={tLocal}
                        />
                      </div>
                    </Collapsible>
                  </div>
                ))}
              </div>
            </Card>
          )}

          {/* Model Group Overrides */}
          <Card className='!rounded-xl border dark:border-gray-700'>
            <Typography.Title heading={6} className='mb-3'>
              {tLocal('route_policy.overrides')}
            </Typography.Title>
            <Typography.Text type='tertiary' size='small' className='mb-3 block'>
              {tLocal('route_policy.overrides_desc')}
            </Typography.Text>

            {allOverrides.length > 0 && (
              <Table
                columns={overrideColumns}
                dataSource={allOverrides}
                rowKey={(r) => `${r.is_user}-${r.id}`}
                size='small'
                pagination={false}
                className='mb-3'
              />
            )}

            <div className='flex items-end gap-2 mt-3'>
              <div className='flex-1'>
                <Typography.Text size='small' className='mb-1 block'>
                  {tLocal('route_policy.raw_model')}
                </Typography.Text>
                <Input
                  value={newOverrideModel}
                  onChange={setNewOverrideModel}
                  placeholder={tLocal('route_policy.raw_model_placeholder')}
                  size='small'
                />
              </div>
              <div className='flex-1'>
                <Typography.Text size='small' className='mb-1 block'>
                  {tLocal('route_policy.group_key')}
                </Typography.Text>
                <Input
                  value={newOverrideGroup}
                  onChange={setNewOverrideGroup}
                  placeholder={tLocal('route_policy.group_key_placeholder')}
                  size='small'
                />
              </div>
              <Button
                theme='solid'
                type='primary'
                size='small'
                icon={<Plus size={14} />}
                loading={addingOverride}
                onClick={handleAddOverride}
              >
                {tLocal('route_policy.add_override')}
              </Button>
            </div>
          </Card>
        </>
      )}
    </Card>
  );
};

// WeightChannelTable supports pointer-drag to adjust channel weights within a group.
const WeightChannelTable = ({
  groupKey,
  channels,
  routeMode = 'weight',
  isAdmin,
  onWeightChange,
  onBatchWeightChange,
  onDeleteWeight,
  onSaved,
  t,
}) => {
  const [orderedChannels, setOrderedChannels] = useState(channels);
  const [dragIndex, setDragIndex] = useState(null);
  const [dragging, setDragging] = useState(false);
  const [saving, setSaving] = useState(false);
  const orderedChannelsRef = useRef(channels);
  const dragIndexRef = useRef(null);
  const draggingRef = useRef(false);
  const baselineOrderRef = useRef('');

  useEffect(() => {
    setOrderedChannels(channels);
    orderedChannelsRef.current = channels;
    baselineOrderRef.current = channels.map((ch) => ch.channel_id).join(',');
  }, [channels]);

  const persistOrderWeights = async (nextChannels) => {
    if (routeMode !== 'weight') return;

    const updates = computeWeightsFromOrder(nextChannels).map(({ channel, weight }) => ({
      channel_id: channel.channel_id,
      weight,
      enabled: resolveEffectiveEnabled(channel),
    }));

    if (updates.length === 0) return;

    setSaving(true);
    await onBatchWeightChange(groupKey, updates);
    setSaving(false);
  };

  const handleGripPointerDown = (index, event) => {
    if (routeMode !== 'weight') return;
    event.preventDefault();
    event.stopPropagation();
    dragIndexRef.current = index;
    draggingRef.current = true;
    setDragIndex(index);
    setDragging(true);
  };

  const handleRowPointerEnter = (index) => {
    if (!draggingRef.current) return;
    const from = dragIndexRef.current;
    if (from === null || from === index) return;

    setOrderedChannels((prev) => {
      const next = [...prev];
      const [moved] = next.splice(from, 1);
      next.splice(index, 0, moved);
      const weighted = applyOrderWeightsToChannels(next);
      orderedChannelsRef.current = weighted;
      return weighted;
    });
    dragIndexRef.current = index;
    setDragIndex(index);
  };

  useEffect(() => {
    const finishDrag = async () => {
      if (!draggingRef.current) return;

      const nextOrder = orderedChannelsRef.current.map((ch) => ch.channel_id).join(',');
      const orderChanged = nextOrder !== baselineOrderRef.current;

      draggingRef.current = false;
      dragIndexRef.current = null;
      setDragIndex(null);
      setDragging(false);

      if (orderChanged) {
        await persistOrderWeights(orderedChannelsRef.current);
      }
    };

    if (!dragging) return undefined;

    const onPointerUp = () => {
      void finishDrag();
    };
    window.addEventListener('pointerup', onPointerUp);
    return () => window.removeEventListener('pointerup', onPointerUp);
  }, [dragging, groupKey, onBatchWeightChange]);

  return (
    <div className={`overflow-x-auto ${dragging ? 'select-none' : ''}`}>
      <table className='w-full text-sm'>
        <thead>
          <tr className='border-b dark:border-gray-600 text-left text-xs text-gray-500'>
            {routeMode === 'weight' ? <th className='py-2 pr-2 w-9' /> : null}
            <th className='py-2 pr-3'>{t('route_policy.provider')}</th>
            <th className='py-2 pr-3'>{t('route_policy.route_slug')}</th>
            {routeMode === 'weight' ? (
              <>
                <th className='py-2 pr-3'>{t('route_policy.user_weight')}</th>
                <th className='py-2 pr-3'>{t('route_policy.enabled')}</th>
                <th className='py-2 pr-3'>{t('route_policy.global_weight')}</th>
              </>
            ) : (
              <th className='py-2 pr-3'>
                {t('route_policy.price_per_1k', {
                  symbol: getCurrencyConfig().symbol,
                  price: '…',
                })}
              </th>
            )}
          </tr>
        </thead>
        <tbody>
          {orderedChannels.map((channel, index) => (
            <ChannelRow
              key={channel.channel_id}
              channel={channel}
              groupKey={groupKey}
              isAdmin={isAdmin}
              index={index}
              isDragging={dragIndex === index}
              isTableDragging={dragging}
              saving={saving}
              onGripPointerDown={handleGripPointerDown}
              onRowPointerEnter={handleRowPointerEnter}
              onWeightChange={onWeightChange}
              onDeleteWeight={onDeleteWeight}
              onSaved={onSaved}
              routeMode={routeMode}
              t={t}
            />
          ))}
        </tbody>
      </table>
    </div>
  );
};

const resolveSupplierLabel = (channel) =>
  channel.supplier_alias?.trim() || channel.provider_slug?.trim() || '—';

const resolveRouteSlugLabel = (channel) => channel.route_slug?.trim() || '—';

// ChannelRow renders a single channel in the group table with editable weight/switch.
const ChannelRow = ({
  channel,
  groupKey,
  isAdmin,
  index = 0,
  isDragging = false,
  isTableDragging = false,
  saving = false,
  onGripPointerDown,
  onRowPointerEnter,
  onWeightChange,
  onDeleteWeight,
  onSaved,
  routeMode,
  t,
}) => {
  const [weight, setWeight] = useState(() => resolveDisplayWeight(channel));
  const [enabled, setEnabled] = useState(() => resolveEffectiveEnabled(channel));
  const [rowSaving, setRowSaving] = useState(false);

  useEffect(() => {
    setWeight(resolveDisplayWeight(channel));
    setEnabled(resolveEffectiveEnabled(channel));
  }, [
    channel.user_weight,
    channel.user_enabled,
    channel.user_configured,
    channel.global_weight,
    channel.global_enabled,
    channel.global_configured,
  ]);

  const handleWeightSave = async () => {
    setRowSaving(true);
    const ok = await onWeightChange(groupKey, channel.channel_id, weight, enabled);
    if (ok) {
      onSaved?.();
    }
    setRowSaving(false);
  };

  const handleToggle = async (checked) => {
    setEnabled(checked);
    setRowSaving(true);
    const ok = await onWeightChange(groupKey, channel.channel_id, weight, checked);
    if (ok) {
      onSaved?.();
    }
    setRowSaving(false);
  };

  const dragEnabled = routeMode === 'weight' && onGripPointerDown;
  const disabled = saving || rowSaving;
  const supplierLabel = resolveSupplierLabel(channel);
  const routeSlugLabel = resolveRouteSlugLabel(channel);

  return (
    <tr
      className={`border-b dark:border-gray-700 transition-all duration-200 ${
        isDragging
          ? 'relative z-10 bg-white dark:bg-gray-800 shadow-lg ring-1 ring-green-300/70 dark:ring-green-700/60 scale-[1.01]'
          : isTableDragging
            ? 'opacity-90'
            : 'hover:bg-gray-50/60 dark:hover:bg-gray-800/40'
      }`}
      onPointerEnter={() => onRowPointerEnter?.(index)}
    >
      <td className='py-2 pr-2 w-9 align-middle'>
        {dragEnabled ? (
          <div
            role='button'
            tabIndex={-1}
            className={`touch-none inline-flex items-center justify-center rounded-md p-1.5 transition-all duration-200 ${
              isDragging
                ? 'cursor-grabbing bg-green-100 text-green-600 shadow-md ring-1 ring-green-300/80 dark:bg-green-900/50 dark:text-green-400 dark:ring-green-700/60'
                : 'cursor-grab text-gray-400 hover:bg-gray-100 hover:text-green-600 hover:shadow-sm hover:ring-1 hover:ring-gray-200 dark:hover:bg-gray-700/80 dark:hover:text-green-400 dark:hover:ring-gray-600 active:scale-95 active:bg-green-50 active:text-green-600 active:shadow-md dark:active:bg-green-900/40'
            }`}
            title={t('route_policy.drag_handle')}
            onPointerDown={(event) => onGripPointerDown(index, event)}
          >
            <GripVertical size={15} strokeWidth={2.25} />
          </div>
        ) : null}
      </td>
      <td className='py-2 pr-3'>
        <Typography.Text size='small'>{supplierLabel}</Typography.Text>
      </td>
      <td className='py-2 pr-3'>
        <Typography.Text size='small' className='font-mono'>
          {routeSlugLabel}
        </Typography.Text>
      </td>
      {routeMode === 'weight' && (
        <td className='py-2 pr-3'>
          <div className='flex items-center gap-1'>
            <InputNumber
              value={weight}
              onChange={(v) => setWeight(v ?? 0)}
              min={0}
              max={1000}
              size='small'
              style={{ width: 70 }}
              onBlur={handleWeightSave}
              disabled={disabled}
            />
            {channel.user_configured && channel.user_weight_id > 0 && (
              <Button
                size='small'
                type='danger'
                icon={<Trash2 size={12} />}
                onClick={() => onDeleteWeight(channel.user_weight_id)}
                disabled={disabled}
              />
            )}
          </div>
        </td>
      )}
      {routeMode === 'weight' && (
        <td className='py-2 pr-3'>
          <Switch
            size='small'
            checked={enabled}
            onChange={handleToggle}
            disabled={disabled}
          />
        </td>
      )}
      {routeMode === 'weight' && (
        <td className='py-2 pr-3'>
          {channel.global_configured ? (
            <Typography.Text size='small' type='tertiary'>
              {channel.global_weight}
              {!channel.global_enabled && ` (${t('route_policy.disabled')})`}
            </Typography.Text>
          ) : (
            <Typography.Text size='small' type='quaternary'>
              —
            </Typography.Text>
          )}
        </td>
      )}
      {routeMode === 'price' && (
        <td className='py-2 pr-3'>
          {channel.price > 0 ? (
            <Typography.Text size='small'>
              {t('route_policy.price_per_1k', {
                symbol: getCurrencyConfig().symbol,
                price: channel.price.toFixed(4),
              })}
            </Typography.Text>
          ) : (
            <Typography.Text size='small' type='quaternary'>
              —
            </Typography.Text>
          )}
        </td>
      )}
    </tr>
  );
};

export default RoutePolicyCard;
