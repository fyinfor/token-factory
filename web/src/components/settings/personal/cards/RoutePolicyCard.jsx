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
  Modal,
  Spin,
  Toast,
} from '@douyinfe/semi-ui';
import { Route, ChevronDown, ChevronRight, Plus, Trash2, Info, Search, GripVertical } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { API, showSuccess, showError } from '../../../../helpers';
import { getCurrencyConfig } from '../../../../helpers/render';
import { getUsedGroupContext } from '../../../../helpers/utils';
import { computeChannelBillingRates } from '../../../../helpers/billingFormula';
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

const resolveDefaultWeight = (channel) => {
  if (channel.global_configured) return channel.global_weight || 0;
  return 0;
};

const resolveDefaultEnabled = (channel) => {
  if (channel.global_configured) return channel.global_enabled;
  return true;
};

const isWeightAtDefault = (channel) =>
  resolveDisplayWeight(channel) === resolveDefaultWeight(channel) &&
  resolveEffectiveEnabled(channel) === resolveDefaultEnabled(channel);

const ChannelModelsCell = ({ models, t }) => {
  const modelsInGroup = models || [];
  if (modelsInGroup.length === 0) {
    return (
      <Typography.Text size='small' type='quaternary'>
        —
      </Typography.Text>
    );
  }
  return (
    <div className='flex flex-wrap gap-1'>
      {modelsInGroup.map((modelName) => (
        <Tag key={modelName} size='small' color='grey'>
          {modelName}
        </Tag>
      ))}
    </div>
  );
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

// 后端 model_prices 对倍率型模型存的是「模型输入倍率」；与计费口径一致：
// USD / 1M tokens = 倍率 × 2（参见 render.jsx 的 inputRatioPrice = modelRatio × 2）。
// 仅在 /api/pricing 无对应数据时作为兜底展示。
const RATIO_TO_USD_PER_1M = 2;

// ── 与首页模型卡片定价统一 ──────────────────────────────────────
// 首页卡片「平台价」= computeChannelBillingRates(...).inputRatioPrice × usedGroupRatio
//   （= (渠道倍率×成本折扣% + 全局倍率×加价%) × 2 × 分组倍率），分组倍率取该模型可用分组中最便宜者。
// 这里基于 /api/pricing 同源数据、同一套 helper 复刻该价，确保两处展示完全一致。

// 卡片精度：与首页一致（保留 2 位有效小数四舍五入）。
const fmtCardPrice = (value) => {
  const v = Number(value);
  if (!Number.isFinite(v) || v <= 0) return '—';
  return String(parseFloat(v.toFixed(2)));
};

// buildPricingIndex 把 /api/pricing 的 data 打平为 channelId -> (modelName -> { value, perRequest })。
const buildPricingIndex = (data, groupRatio) => {
  const index = new Map();
  (data || []).forEach((item) => {
    if (!item) return;
    const { usedGroupRatio } = getUsedGroupContext(item, 'all', groupRatio || {});
    (item.channel_list || []).forEach((ch) => {
      if (!ch || ch.channel_id == null) return;
      const rates = computeChannelBillingRates({
        channelModelRatio: ch.model_ratio,
        channelCompletionRatio: ch.completion_ratio,
        channelModelPrice:
          ch.model_price != null && ch.model_price > 0 ? ch.model_price : -1,
        priceDiscountPercent: ch.price_discount_percent,
        markupDiscountPercent: ch.markup_discount_rate,
        globalModelRatio: item.model_ratio,
        globalModelPrice: item.model_price,
        globalCompletionRatio: item.completion_ratio,
      });
      const perRequest = item.quota_type === 1;
      const value = perRequest
        ? (rates.mp > 0 ? rates.mp : 0) * usedGroupRatio
        : rates.inputRatioPrice * usedGroupRatio;
      if (!(value > 0)) return;
      if (!index.has(ch.channel_id)) index.set(ch.channel_id, new Map());
      const perChannel = index.get(ch.channel_id);
      const prev = perChannel.get(item.model_name);
      if (prev == null || value < prev.value) {
        perChannel.set(item.model_name, { value, perRequest });
      }
    });
  });
  return index;
};

// resolveUnifiedChannelPrice 取某渠道在归类内各模型中的最低价（与后端 ch.Price 取 min 一致）。
const resolveUnifiedChannelPrice = (pricingIndex, channel) => {
  if (!pricingIndex) return null;
  const perChannel = pricingIndex.get(channel.channel_id);
  if (!perChannel) return null;
  const models = channel.models_in_group || [];
  let best = null;
  models.forEach((m) => {
    const entry = perChannel.get(m);
    if (entry && entry.value > 0 && (best == null || entry.value < best.value)) {
      best = entry;
    }
  });
  return best;
};

const RoutePolicyCard = ({
  t,
  variant = 'self',
  managedUserId = null,
  hideHeader = false,
}) => {
  const { t: translate } = useTranslation();
  const [userState] = useContext(UserContext);
  const [statusState] = useContext(StatusContext);
  const isAdmin = (userState?.user?.role || 0) >= 10;
  const statusLoaded = statusState?.status != null;
  const routeEnabled = statusState?.status?.tokenfactory_route_enabled === true;
  const isAdminTemplate = variant === 'admin-template';
  const isAdminUser = variant === 'admin-user' && Number(managedUserId) > 0;
  const canManageGroupRoute = !isAdminTemplate;

  const apiPaths = useMemo(() => {
    if (isAdminTemplate) {
      return {
        getPolicy: '/api/admin/route-policy/template',
        putMode: '/api/admin/route-policy/config',
        putGroupRoute: null,
        postWeight: '/api/admin/route-policy/template/weights',
        deleteWeight: (id) => `/api/admin/route-policy/template/weights/${id}`,
        postOverride: '/api/admin/route-policy/template/overrides',
        deleteOverride: (id) => `/api/admin/route-policy/template/overrides/${id}`,
      };
    }
    if (isAdminUser) {
      const base = `/api/admin/route-policy/users/${managedUserId}`;
      return {
        getPolicy: base,
        putMode: `${base}/mode`,
        putGroupRoute: `${base}/group-route`,
        postWeight: `${base}/weights`,
        deleteWeight: (id) => `${base}/weights/${id}`,
        postOverride: `${base}/overrides`,
        deleteOverride: (id) => `${base}/overrides/${id}`,
      };
    }
    return {
      getPolicy: '/api/user/route-policy',
      putMode: '/api/user/route-policy/mode',
      putGroupRoute: '/api/user/route-policy/group-route',
      postWeight: '/api/user/route-policy/weights',
      deleteWeight: (id) => `/api/user/route-policy/weights/${id}`,
      postOverride: '/api/user/route-policy/overrides',
      deleteOverride: (id) => `/api/user/route-policy/overrides/${id}`,
    };
  }, [isAdminTemplate, isAdminUser, managedUserId]);

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
  const [pricingIndex, setPricingIndex] = useState(null);
  const [resettingWeights, setResettingWeights] = useState(false);
  const groupRefs = useRef({});

  const tLocal = useCallback(
    (key, options) => (t ? t(key, options) : translate(key, options)),
    [t, translate]
  );

  const fetchPolicy = useCallback(async ({ silent = false } = {}) => {
    if (!silent) {
      setLoading(true);
      setLoadError('');
    }
    try {
      const res = await API.get(apiPaths.getPolicy);
      const { data } = res;
      if (data.success !== false) {
        setMode(data.mode ?? '');
        setGlobalMode(data.global_mode ?? 'price');
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
  }, [tLocal, apiPaths.getPolicy]);

  // 拉取 /api/pricing，构建「渠道×模型 → 平台价」索引，使价格优模式展示与首页卡片完全一致。
  const fetchPricing = useCallback(async () => {
    try {
      const res = await API.get('/api/pricing');
      const { success, data, group_ratio } = res.data || {};
      if (success !== false && Array.isArray(data)) {
        setPricingIndex(buildPricingIndex(data, group_ratio));
      }
    } catch (err) {
      // 定价同步失败时静默回退到 gRPC 快照价（fmtUsdPer1M），不阻塞页面。
      setPricingIndex(null);
    }
  }, []);

  useEffect(() => {
    if (!statusLoaded) {
      return;
    }
    if (!routeEnabled) {
      setLoading(false);
      setLoadError('');
      return;
    }
    if (variant === 'admin-user' && !(Number(managedUserId) > 0)) {
      setLoading(false);
      setLoadError('');
      setGroups([]);
      return;
    }
    fetchPolicy();
    fetchPricing();
  }, [statusLoaded, routeEnabled, fetchPolicy, fetchPricing, variant, managedUserId]);

  // mode === '' 时跟随系统全局模式；界面直接展示 effectiveMode。
  const effectiveMode = mode === '' ? globalMode : mode;

  const handleModeChange = async (rawValue) => {
    const newMode = rawValue?.target?.value ?? rawValue;
    if (!newMode || newMode === effectiveMode) return;

    const prevMode = mode;
    setMode(newMode);
    setSavingMode(true);
    try {
      const res = await API.put(apiPaths.putMode, { mode: newMode });
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

  const handleGroupRouteToggle = async (groupKey, enabled) => {
    if (!apiPaths.putGroupRoute) return;
    const disabled = !enabled;
    const prevGroups = groups;
    setGroups((list) =>
      list.map((g) =>
        g.group_key === groupKey ? { ...g, route_disabled: disabled } : g,
      ),
    );
    try {
      const res = await API.put(apiPaths.putGroupRoute, {
        group_key: groupKey,
        disabled,
      });
      if (res.data.success) {
        showSuccess(
          disabled
            ? tLocal('route_policy.group_route_disabled')
            : tLocal('route_policy.group_route_enabled'),
        );
      } else {
        showError(res.data.error || tLocal('route_policy.save_failed'));
        setGroups(prevGroups);
      }
    } catch (err) {
      showError(
        err.response?.data?.error || tLocal('route_policy.save_failed'),
      );
      setGroups(prevGroups);
    }
  };

  const handleWeightChange = async (groupKey, channelID, weight, enabled, { quiet = false } = {}) => {
    try {
      const res = await API.post(apiPaths.postWeight, {
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

  const handleDeleteWeight = async (weightID, { quiet = false } = {}) => {
    try {
      const res = await API.delete(apiPaths.deleteWeight(weightID));
      if (res.data.success) {
        if (!quiet) {
          showSuccess(tLocal('route_policy.weight_deleted'));
          fetchPolicy({ silent: true });
        }
        return true;
      }
      if (!quiet) {
        showError(res.data.error || tLocal('route_policy.delete_failed'));
      }
      return false;
    } catch (err) {
      if (!quiet) {
        showError(tLocal('route_policy.delete_failed'));
      }
      return false;
    }
  };

  const hasNonDefaultWeights = useMemo(
    () =>
      groups.some((group) =>
        (group.channels || []).some((channel) => !isWeightAtDefault(channel)),
      ),
    [groups],
  );

  const handleResetAllWeights = () => {
    const updates = [];
    groups.forEach((group) => {
      (group.channels || []).forEach((channel) => {
        if (isWeightAtDefault(channel)) return;
        updates.push({
          groupKey: group.group_key,
          channelID: channel.channel_id,
          weight: resolveDefaultWeight(channel),
          enabled: resolveDefaultEnabled(channel),
        });
      });
    });

    if (updates.length === 0) {
      showError(tLocal('route_policy.reset_all_weights_none'));
      return;
    }

    Modal.confirm({
      title: tLocal('route_policy.reset_all_weights'),
      content: tLocal('route_policy.reset_all_weights_confirm'),
      okText: tLocal('route_policy.reset_all_weights'),
      cancelText: tLocal('取消'),
      onOk: async () => {
        setResettingWeights(true);
        let changed = 0;
        for (const item of updates) {
          const ok = await handleWeightChange(
            item.groupKey,
            item.channelID,
            item.weight,
            item.enabled,
            { quiet: true },
          );
          if (ok) changed += 1;
        }
        setResettingWeights(false);
        if (changed > 0) {
          showSuccess(tLocal('route_policy.reset_all_weights_success'));
          fetchPolicy({ silent: true });
        } else {
          showError(tLocal('route_policy.save_failed'));
        }
      },
    });
  };

  const handleAddOverride = async () => {
    if (!newOverrideModel.trim() || !newOverrideGroup.trim()) {
      showError(tLocal('route_policy.override_required'));
      return;
    }
    setAddingOverride(true);
    try {
      const res = await API.post(apiPaths.postOverride, {
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
      const res = await API.delete(apiPaths.deleteOverride(overrideID));
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
          {isUser
            ? tLocal('route_policy.my_override')
            : tLocal('route_policy.global_override')}
        </Tag>
      ),
    },
    {
      title: '',
      key: 'action',
      width: '15%',
      render: (_, record) => {
        const canDelete = isAdminTemplate
          ? !record.is_user
          : Boolean(record.is_user);
        if (!canDelete) return null;
        return (
          <Button
            type='danger'
            size='small'
            icon={<Trash2 size={14} />}
            onClick={() => handleDeleteOverride(record.id)}
          />
        );
      },
    },
  ];

  const allOverrides = isAdminTemplate
    ? globalOverrides.map((o) => ({ ...o, is_user: false }))
    : [
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
      {!hideHeader && (
        <div className='flex items-center mb-4'>
          <Avatar size='small' color='green' className='mr-3 shadow-md'>
            <Route size={16} />
          </Avatar>
          <div>
            <Typography.Text className='text-lg font-medium'>
              {isAdminTemplate
                ? tLocal('route_policy.admin_template_title')
                : isAdminUser
                  ? tLocal('route_policy.admin_user_title')
                  : tLocal('route_policy.title')}
            </Typography.Text>
            <div className='text-xs text-gray-600 dark:text-gray-400'>
              {isAdminTemplate
                ? tLocal('route_policy.admin_template_description')
                : isAdminUser
                  ? tLocal('route_policy.admin_user_description')
                  : tLocal('route_policy.description')}
            </div>
          </div>
        </div>
      )}

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
                <div className='min-w-0 flex-1'>
                  <div className='flex flex-wrap items-center gap-2'>
                    <Typography.Title heading={6}>
                      {tLocal('route_policy.model_groups')}
                    </Typography.Title>
                    {effectiveMode === 'weight' && (
                      <Button
                        size='small'
                        type='tertiary'
                        loading={resettingWeights}
                        disabled={resettingWeights || !hasNonDefaultWeights}
                        onClick={handleResetAllWeights}
                      >
                        {tLocal('route_policy.reset_all_weights')}
                      </Button>
                    )}
                  </div>
                  <Typography.Text type='tertiary' size='small' className='block mt-1'>
                    {effectiveMode === 'weight'
                      ? tLocal('route_policy.drag_sort_hint')
                      : tLocal('route_policy.mode_price_hint')}
                  </Typography.Text>
                  <div className='mt-2 flex items-start gap-1.5 rounded-lg bg-blue-50/80 dark:bg-blue-950/30 px-3 py-2 text-xs text-blue-700 dark:text-blue-300'>
                    <Info size={13} className='mt-0.5 shrink-0' />
                    <span>{tLocal('route_policy.group_explainer')}</span>
                  </div>
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
                              {tLocal('route_policy.search_maps_to', {
                                groupKey: item.groupKey,
                                displayName: item.displayName,
                              })}
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
                        <Tag size='small' color='cyan'>
                          {group.models.length} {tLocal('route_policy.models')}
                        </Tag>
                        <code
                          className='text-xs text-gray-500 dark:text-gray-400 font-mono truncate max-w-[140px]'
                          title={tLocal('route_policy.group_key_label')}
                        >
                          {group.group_key}
                        </code>
                        <Tag size='small' color='blue'>
                          {group.channel_count} {tLocal('route_policy.channels')}
                        </Tag>
                        {group.route_disabled ? (
                          <Tag size='small' color='orange'>
                            {tLocal('route_policy.group_route_off_tag')}
                          </Tag>
                        ) : null}
                      </div>
                      {canManageGroupRoute ? (
                        <div
                          className='flex items-center gap-2 shrink-0 ml-3'
                          onClick={(e) => e.stopPropagation()}
                        >
                          <Typography.Text size='small' type='tertiary'>
                            {tLocal('route_policy.group_route_switch')}
                          </Typography.Text>
                          <Switch
                            checked={!group.route_disabled}
                            onChange={(checked) =>
                              handleGroupRouteToggle(group.group_key, checked)
                            }
                            aria-label={tLocal('route_policy.group_route_switch')}
                          />
                        </div>
                      ) : null}
                    </div>
                    <Collapsible isOpen={expandedGroups[group.group_key]}>
                      <div className='px-4 pb-3'>
                        <div className='mb-3 rounded-lg border border-dashed dark:border-gray-600 bg-gray-50/60 dark:bg-gray-900/30 px-3 py-2.5'>
                          <Typography.Text size='small' strong className='block mb-1'>
                            {tLocal('route_policy.group_call_models')}
                          </Typography.Text>
                          <Typography.Text type='tertiary' size='small' className='block mb-2'>
                            {tLocal('route_policy.group_models_hint', {
                              count: group.models.length,
                              groupKey: group.group_key,
                            })}
                          </Typography.Text>
                          <div className='flex flex-wrap gap-1'>
                            {(group.models || []).map((modelName) => (
                              <Tag
                                key={modelName}
                                size='small'
                                color='light-blue'
                                className='cursor-pointer'
                                onClick={(event) => {
                                  event.stopPropagation();
                                  setModelSearch(modelName);
                                }}
                              >
                                {modelName}
                              </Tag>
                            ))}
                          </div>
                        </div>
                        {effectiveMode === 'weight' ? (
                          <WeightChannelTable
                            groupKey={group.group_key}
                            channels={group.channels}
                            isAdmin={isAdmin}
                            onWeightChange={handleWeightChange}
                            onBatchWeightChange={handleBatchWeightChange}
                            onDeleteWeight={handleDeleteWeight}
                            onSaved={() => fetchPolicy({ silent: true })}
                            t={tLocal}
                          />
                        ) : (
                          <PriceChannelTable
                            channels={group.channels}
                            pricingIndex={pricingIndex}
                            t={tLocal}
                          />
                        )}
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
            <th className='py-2 pr-2 w-9' />
            <th className='py-2 pr-3'>{t('route_policy.provider')}</th>
            <th className='py-2 pr-3'>{t('route_policy.route_slug')}</th>
            <th className='py-2 pr-3'>{t('route_policy.group_models')}</th>
            <th className='py-2 pr-3'>{t('route_policy.user_weight')}</th>
            <th className='py-2 pr-3'>{t('route_policy.enabled')}</th>
            <th className='py-2 pr-3'>{t('route_policy.global_weight')}</th>
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
              t={t}
            />
          ))}
        </tbody>
      </table>
    </div>
  );
};

// PriceChannelTable shows read-only channel prices sorted ascending within a group.
// 单价口径与首页模型卡片「平台价」一致：优先用 /api/pricing 索引（含分组倍率），
// 无对应数据时回退到 gRPC 快照价（fmtUsdPer1M）。
const PriceChannelTable = ({ channels, pricingIndex, t }) => {
  const { symbol: currencySymbol, rate: currencyRate } = getCurrencyConfig();

  // 计算每个渠道的统一展示价（USD/1M 或按次），并按价升序（无价排最后）。
  const rows = useMemo(() => {
    const withPrice = (channels || []).map((channel) => {
      const unified = resolveUnifiedChannelPrice(pricingIndex, channel);
      let value = null;
      let perRequest = false;
      if (unified && unified.value > 0) {
        value = unified.value;
        perRequest = unified.perRequest;
      } else if (channel.price > 0) {
        // 兜底：gRPC 快照价为有效输入倍率，×2 得 USD/1M。
        value = channel.price * RATIO_TO_USD_PER_1M;
      }
      return { channel, value, perRequest };
    });
    return withPrice.sort((a, b) => {
      const pa = a.value > 0 ? a.value : Number.POSITIVE_INFINITY;
      const pb = b.value > 0 ? b.value : Number.POSITIVE_INFINITY;
      if (pa !== pb) return pa - pb;
      return a.channel.channel_id - b.channel.channel_id;
    });
  }, [channels, pricingIndex]);

  return (
    <div className='overflow-x-auto'>
      <table className='w-full text-sm'>
        <thead>
          <tr className='border-b dark:border-gray-600 text-left text-xs text-gray-500'>
            <th className='py-2 pr-3 w-14'>#</th>
            <th className='py-2 pr-3'>{t('route_policy.provider')}</th>
            <th className='py-2 pr-3'>{t('route_policy.route_slug')}</th>
            <th className='py-2 pr-3'>{t('route_policy.group_models')}</th>
            <th className='py-2 pr-3'>
              {t('route_policy.price')} ({currencySymbol}/1M)
            </th>
          </tr>
        </thead>
        <tbody>
          {rows.map(({ channel, value, perRequest }, index) => {
            const modelsInGroup = channel.models_in_group || [];
            const priceText =
              value > 0
                ? `${currencySymbol}${fmtCardPrice(value * currencyRate)}${perRequest ? '/次' : '/1M'}`
                : '—';
            return (
              <tr
                key={channel.channel_id}
                className='border-b dark:border-gray-700 hover:bg-gray-50/60 dark:hover:bg-gray-800/40'
              >
                <td className='py-2 pr-3 align-top'>
                  <Tag size='small' color={index === 0 ? 'green' : 'grey'}>
                    #{index + 1}
                  </Tag>
                </td>
                <td className='py-2 pr-3 align-top'>
                  <Typography.Text size='small'>
                    {resolveSupplierLabel(channel)}
                  </Typography.Text>
                </td>
                <td className='py-2 pr-3 align-top'>
                  <Typography.Text size='small' className='font-mono'>
                    {resolveRouteSlugLabel(channel)}
                  </Typography.Text>
                </td>
                <td className='py-2 pr-3 align-top'>
                  <ChannelModelsCell models={modelsInGroup} t={t} />
                </td>
                <td className='py-2 pr-3 align-top font-mono'>
                  <Typography.Text size='small'>{priceText}</Typography.Text>
                </td>
              </tr>
            );
          })}
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
      </td>
      <td className='py-2 pr-3'>
        <Typography.Text size='small'>{supplierLabel}</Typography.Text>
      </td>
      <td className='py-2 pr-3'>
        <Typography.Text size='small' className='font-mono'>
          {routeSlugLabel}
        </Typography.Text>
      </td>
      <td className='py-2 pr-3 align-top'>
        <ChannelModelsCell models={channel.models_in_group} t={t} />
      </td>
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
      <td className='py-2 pr-3'>
        <Switch
          size='small'
          checked={enabled}
          onChange={handleToggle}
          disabled={disabled}
        />
      </td>
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
    </tr>
  );
};

export default RoutePolicyCard;
