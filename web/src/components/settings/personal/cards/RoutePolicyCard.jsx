import React, { useState, useEffect, useContext, useCallback } from 'react';
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
import { Route, ChevronDown, ChevronRight, Plus, Trash2, Info } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { API, showSuccess, showError } from '../../../../helpers';
import { UserContext } from '../../../../context/User';

const ROUTE_MODES = [
  { value: '', label_key: 'route_policy.disabled' },
  { value: 'default', label_key: 'route_policy.mode_default' },
  { value: 'weight', label_key: 'route_policy.mode_weight' },
  { value: 'price', label_key: 'route_policy.mode_price' },
];

const RoutePolicyCard = ({ t }) => {
  const { t: translate } = useTranslation();
  const [userState] = useContext(UserContext);
  const isAdmin = (userState?.user?.role || 0) >= 10;

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

  const tLocal = useCallback(
    (key) => t ? t(key) : translate(key),
    [t, translate]
  );

  const fetchPolicy = useCallback(async () => {
    setLoading(true);
    setLoadError('');
    try {
      const res = await API.get('/api/user/route-policy');
      const { data } = res;
      if (data.success !== false) {
        setMode(data.mode ?? '');
        setGlobalMode(data.global_mode ?? 'default');
        setGroups(data.groups ?? []);
        setUserOverrides(data.user_overrides ?? []);
        setGlobalOverrides(data.global_overrides ?? []);
      } else {
        const msg = data.error || tLocal('route_policy.load_failed');
        setLoadError(msg);
        showError(msg);
      }
    } catch (err) {
      const msg =
        tLocal('route_policy.load_failed') +
        ': ' +
        (err.response?.data?.error || err.message || String(err));
      setLoadError(msg);
      showError(msg);
    } finally {
      setLoading(false);
    }
  }, [tLocal]);

  useEffect(() => {
    fetchPolicy();
  }, [fetchPolicy]);

  const handleModeChange = async (newMode) => {
    const prevMode = mode;
    setMode(newMode);
    setSavingMode(true);
    try {
      if (newMode === '') {
        const res = await API.put('/api/user/route-policy/mode', { reset_mode: true });
        if (res.data.success) {
          showSuccess(tLocal('route_policy.mode_reset'));
        } else {
          showError(res.data.error || tLocal('route_policy.save_failed'));
          setMode(prevMode);
        }
      } else {
        const res = await API.put('/api/user/route-policy/mode', { mode: newMode });
        if (res.data.success) {
          showSuccess(tLocal('route_policy.mode_updated'));
        } else {
          showError(res.data.error || tLocal('route_policy.save_failed'));
          setMode(prevMode);
        }
      }
    } catch (err) {
      showError(tLocal('route_policy.save_failed'));
      setMode(prevMode);
    } finally {
      setSavingMode(false);
    }
  };

  const handleWeightChange = async (groupKey, channelID, weight, enabled) => {
    try {
      const res = await API.post('/api/user/route-policy/weights', {
        group_key: groupKey,
        channel_id: channelID,
        weight,
        enabled,
      });
      if (res.data.success) {
        showSuccess(tLocal('route_policy.weight_updated'));
        fetchPolicy();
      } else {
        showError(res.data.error || tLocal('route_policy.save_failed'));
      }
    } catch (err) {
      showError(tLocal('route_policy.save_failed'));
    }
  };

  const handleDeleteWeight = async (weightID) => {
    try {
      const res = await API.delete(`/api/user/route-policy/weights/${weightID}`);
      if (res.data.success) {
        showSuccess(tLocal('route_policy.weight_deleted'));
        fetchPolicy();
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
        fetchPolicy();
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
        fetchPolicy();
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

  // mode === ''：未开启个人 TokenFactory 智能路由（本站原生选路）。
  // mode 为 default/weight/price 时：启用个人策略，对应 TokenFactory 三种模式。
  const radioValue = mode === '' ? '' : mode;
  const smartRoutingEnabled = mode !== '';

  const modeLabel = (m) => {
    switch (m) {
      case 'weight': return tLocal('route_policy.mode_weight');
      case 'price': return tLocal('route_policy.mode_price');
      case 'default': return tLocal('route_policy.mode_default');
      default: return tLocal('route_policy.disabled');
    }
  };

  const modeHint = () => {
    if (mode === '') {
      return tLocal('route_policy.disabled_hint');
    }
    switch (mode) {
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
                  value={radioValue}
                  onChange={(e) => handleModeChange(e.target.value)}
                  disabled={savingMode}
                >
                  {ROUTE_MODES.map((m) => (
                    <Radio key={m.value || 'disabled'} value={m.value}>
                      {tLocal(m.label_key)}
                    </Radio>
                  ))}
                </RadioGroup>
                <div className='mt-2 space-y-1'>
                  <div className='text-xs text-gray-500 dark:text-gray-400 flex items-center gap-1'>
                    <Info size={12} />
                    {modeHint()}
                  </div>
                  {!smartRoutingEnabled && globalMode ? (
                    <div className='text-xs text-gray-400 dark:text-gray-500'>
                      {tLocal('route_policy.site_global_ref')}: {modeLabel(globalMode)}
                    </div>
                  ) : null}
                </div>
              </div>
            </div>
          </Card>

          {/* Model Groups with Channels */}
          {mode === 'weight' && groups.length > 0 && (
            <Card className='!rounded-xl border dark:border-gray-700 mb-4'>
              <Typography.Title heading={6} className='mb-3'>
                {tLocal('route_policy.model_groups')}
              </Typography.Title>
              <div className='space-y-2'>
                {groups.map((group) => (
                  <div
                    key={group.group_key}
                    className='border dark:border-gray-600 rounded-lg overflow-hidden'
                  >
                    <div
                      className='flex items-center justify-between px-4 py-3 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800'
                      onClick={() => toggleGroup(group.group_key)}
                    >
                      <div className='flex items-center gap-2'>
                        {expandedGroups[group.group_key] ? (
                          <ChevronDown size={16} />
                        ) : (
                          <ChevronRight size={16} />
                        )}
                        <Typography.Text strong>
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
                        <div className='overflow-x-auto'>
                          <table className='w-full text-sm'>
                            <thead>
                              <tr className='border-b dark:border-gray-600 text-left text-xs text-gray-500'>
                                <th className='py-2 pr-3'>{tLocal('route_policy.channel_name')}</th>
                                <th className='py-2 pr-3'>{tLocal('route_policy.provider')}</th>
                                {mode === 'weight' && (
                                  <th className='py-2 pr-3'>{tLocal('route_policy.user_weight')}</th>
                                )}
                                {mode === 'weight' && (
                                  <th className='py-2 pr-3'>{tLocal('route_policy.enabled')}</th>
                                )}
                                {mode === 'weight' && (
                                  <th className='py-2 pr-3'>{tLocal('route_policy.global_weight')}</th>
                                )}
                                {mode === 'price' && (
                                  <th className='py-2 pr-3'>{tLocal('route_policy.price')}</th>
                                )}
                              </tr>
                            </thead>
                            <tbody>
                              {group.channels.map((ch) => (
                                <ChannelRow
                                  key={ch.channel_id}
                                  channel={ch}
                                  groupKey={group.group_key}
                                  isAdmin={isAdmin}
                                  onWeightChange={handleWeightChange}
                                  onDeleteWeight={handleDeleteWeight}
                                  routeMode={mode}
                                  t={tLocal}
                                />
                              ))}
                            </tbody>
                          </table>
                        </div>
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

// ChannelRow renders a single channel in the group table with editable weight/switch.
const ChannelRow = ({ channel, groupKey, isAdmin, onWeightChange, onDeleteWeight, routeMode, t }) => {
  const [weight, setWeight] = useState(channel.user_weight || 0);
  const [enabled, setEnabled] = useState(channel.user_enabled);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    setWeight(channel.user_weight || 0);
    setEnabled(channel.user_enabled);
  }, [channel.user_weight, channel.user_enabled]);

  const handleWeightSave = async () => {
    setSaving(true);
    await onWeightChange(groupKey, channel.channel_id, weight, enabled);
    setSaving(false);
  };

  const handleToggle = async (checked) => {
    setEnabled(checked);
    setSaving(true);
    await onWeightChange(groupKey, channel.channel_id, weight, checked);
    setSaving(false);
  };

  const displayName = isAdmin ? channel.name : channel.masked_name;

  return (
    <tr className='border-b dark:border-gray-700'>
      <td className='py-2 pr-3'>
        <Typography.Text size='small'>{displayName}</Typography.Text>
        {channel.route_slug && (
          <Typography.Text size='small' type='tertiary' className='ml-1'>
            /{channel.route_slug}
          </Typography.Text>
        )}
      </td>
      <td className='py-2 pr-3'>
        <Typography.Text size='small' type='tertiary'>
          {channel.provider_slug}
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
              disabled={saving}
            />
            {channel.user_configured && (
              <Button
                size='small'
                type='danger'
                icon={<Trash2 size={12} />}
                onClick={() => onDeleteWeight(0)}
                disabled={saving}
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
            disabled={saving}
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
              ¥{channel.price.toFixed(4)}/1K
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
