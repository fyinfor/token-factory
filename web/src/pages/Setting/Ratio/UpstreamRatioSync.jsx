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

import React, { useState, useCallback, useMemo, useEffect } from 'react';
import {
  Button,
  Table,
  Tag,
  Empty,
  Checkbox,
  Form,
  Input,
  Tooltip,
  Select,
  Modal,
} from '@douyinfe/semi-ui';
import { IconSearch } from '@douyinfe/semi-icons';
import {
  RefreshCcw,
  CheckSquare,
  AlertTriangle,
  CheckCircle,
} from 'lucide-react';
import {
  API,
  showError,
  showSuccess,
  showWarning,
  stringToColor,
} from '../../../helpers';
import { useIsMobile } from '../../../hooks/common/useIsMobile';
import { DEFAULT_ENDPOINT } from '../../../constants';
import { useTranslation } from 'react-i18next';
import {
  IllustrationNoResult,
  IllustrationNoResultDark,
} from '@douyinfe/semi-illustrations';
import ChannelSelectorModal from '../../../components/settings/ChannelSelectorModal';

const OFFICIAL_RATIO_PRESET_ID = -100;
const OFFICIAL_RATIO_PRESET_NAME = '官方倍率预设';
const OFFICIAL_RATIO_PRESET_BASE_URL = 'https://basellm.github.io';
const OFFICIAL_RATIO_PRESET_ENDPOINT =
  '/llm-metadata/api/newapi/ratio_config-v1-base.json';
const MODELS_DEV_PRESET_ID = -101;
const MODELS_DEV_PRESET_NAME = 'models.dev 价格预设';
const MODELS_DEV_PRESET_BASE_URL = 'https://models.dev';
const MODELS_DEV_PRESET_ENDPOINT = 'https://models.dev/api.json';

const CHANNEL_PRICING_OPTION_KEYS = [
  'ChannelModelPrice',
  'ChannelModelRatio',
  'ChannelCompletionRatio',
  'ChannelCacheRatio',
];

const UPSTREAM_NAME_ID_RE = /\((-?\d+)\)\s*$/;

function parseUpstreamListChannelId(upstreamDisplayName) {
  const m = String(upstreamDisplayName).match(UPSTREAM_NAME_ID_RE);
  if (!m) return null;
  const n = parseInt(m[1], 10);
  return Number.isNaN(n) ? null : n;
}

function parseNestedOption(json) {
  try {
    const o = JSON.parse(json || '{}');
    return o && typeof o === 'object' && !Array.isArray(o) ? o : {};
  } catch {
    return {};
  }
}

function ratioTypeToPascal(ratioType) {
  return ratioType
    .split('_')
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join('');
}

function channelOptionKeyForRatioType(ratioType) {
  return `Channel${ratioTypeToPascal(ratioType)}`;
}

function cloneDeep(obj) {
  return JSON.parse(JSON.stringify(obj));
}

/** 解析勾选的上游列名（兼容 number / string 与上游 JSON 类型差异） */
function findUpstreamColumnName(upMap, value) {
  if (!upMap || value === undefined || value === null) return null;
  const same = (a, b) => {
    if (a === b) return true;
    const na = Number(a);
    const nb = Number(b);
    if (Number.isNaN(na) || Number.isNaN(nb)) return false;
    return na === nb;
  };
  const hit = Object.entries(upMap).find(([_, v]) => same(v, value));
  return hit ? hit[0] : null;
}

function resolutionMatchesUpstream(selected, upstreamVal) {
  if (selected === undefined || selected === null) return false;
  if (selected === upstreamVal) return true;
  const ns = Number(selected);
  const nu = Number(upstreamVal);
  return !Number.isNaN(ns) && !Number.isNaN(nu) && ns === nu;
}

function formatCellNumber(v) {
  if (v === null || v === undefined || v === 'same') return '—';
  const n = Number(v);
  if (Number.isNaN(n)) return String(v);
  const r = Math.round(n * 1e6) / 1e6;
  if (Math.abs(r - Math.round(r)) < 1e-9) return String(Math.round(r));
  return String(r).replace(/(\.\d*?[1-9])0+$/, '$1').replace(/\.$/, '');
}

const RATIO_TO_PRICE_FACTOR = 2;
const PRICE_DECIMAL_PLACES = 4;
const RATIO_DECIMAL_PLACES = 6;

function toFiniteNumber(value) {
  const n = Number(value);
  return Number.isFinite(n) ? n : null;
}

function roundToPlaces(value, places) {
  const n = toFiniteNumber(value);
  if (n === null) return null;
  const factor = 10 ** places;
  return Math.round(n * factor) / factor;
}

function normalizeStoredNumber(value, places = RATIO_DECIMAL_PLACES) {
  const rounded = roundToPlaces(value, places);
  if (rounded === null) return null;
  const nearestInt = Math.round(rounded);
  // Absorb floating-point epsilon like 33.000000000008 -> 33.
  if (Math.abs(rounded - nearestInt) < 1e-9) {
    return nearestInt;
  }
  return rounded;
}

function ratioToDisplayPrice(ratio) {
  const n = toFiniteNumber(ratio);
  if (n === null) return null;
  return roundToPlaces(n * RATIO_TO_PRICE_FACTOR, PRICE_DECIMAL_PLACES);
}

function formatPriceNumber(v) {
  if (v === null || v === undefined || v === 'same') return '—';
  const n = roundToPlaces(v, PRICE_DECIMAL_PLACES);
  if (n === null) return String(v);
  if (Math.abs(n - Math.round(n)) < 1e-9) return String(Math.round(n));
  return String(n).replace(/(\.\d*?[1-9])0+$/, '$1').replace(/\.$/, '');
}

function formatOldNewPair(oldVal, newVal) {
  return `${formatCellNumber(oldVal)}/${formatCellNumber(newVal)}`;
}

function formatPriceOldNewPair(oldVal, newVal) {
  const formatPriceWithSymbol = (value) => {
    const formatted = formatPriceNumber(value);
    return formatted === '—' ? '—' : `$${formatted}`;
  };
  return `${formatPriceWithSymbol(oldVal)}/${formatPriceWithSymbol(newVal)}`;
}

/** 是否可勾选同步：存在上游新价且与当前生效价不同 */
function isUpstreamCellSelectable(oldVal, newVal) {
  if (newVal === null || newVal === undefined || newVal === 'same')
    return false;
  const nu = Number(newVal);
  if (Number.isNaN(nu)) return false;
  if (oldVal === null || oldVal === undefined) return true;
  return !resolutionMatchesUpstream(oldVal, newVal);
}

function ConflictConfirmModal({ t, visible, items, onOk, onCancel }) {
  const isMobile = useIsMobile();
  const columns = [
    { title: t('渠道'), dataIndex: 'channel' },
    { title: t('模型'), dataIndex: 'model' },
    {
      title: t('当前计费'),
      dataIndex: 'current',
      render: (text) => <div style={{ whiteSpace: 'pre-wrap' }}>{text}</div>,
    },
    {
      title: t('修改为'),
      dataIndex: 'newVal',
      render: (text) => <div style={{ whiteSpace: 'pre-wrap' }}>{text}</div>,
    },
  ];

  return (
    <Modal
      title={t('确认冲突项修改')}
      visible={visible}
      onCancel={onCancel}
      onOk={onOk}
      size={isMobile ? 'full-width' : 'large'}
    >
      <Table
        columns={columns}
        dataSource={items}
        pagination={false}
        size='small'
      />
    </Modal>
  );
}

export default function UpstreamRatioSync(props) {
  const { t } = useTranslation();
  const [modalVisible, setModalVisible] = useState(false);
  const [loading, setLoading] = useState(false);
  const [syncLoading, setSyncLoading] = useState(false);
  const isMobile = useIsMobile();

  // 渠道选择相关
  const [allChannels, setAllChannels] = useState([]);
  const [selectedChannelIds, setSelectedChannelIds] = useState([]);

  // 渠道端点配置
  const [channelEndpoints, setChannelEndpoints] = useState({}); // { channelId: endpoint }

  // 差异数据和测试结果
  const [differences, setDifferences] = useState({});
  const [resolutions, setResolutions] = useState({});

  // 是否已经执行过同步
  const [hasSynced, setHasSynced] = useState(false);

  // 分页相关状态
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  // 搜索相关状态
  const [searchKeyword, setSearchKeyword] = useState('');

  // 倍率类型过滤
  const [ratioTypeFilter, setRatioTypeFilter] = useState('');

  // 冲突确认弹窗相关
  const [confirmVisible, setConfirmVisible] = useState(false);
  const [conflictItems, setConflictItems] = useState([]); // {channel, model, current, newVal, ratioType}

  const channelSelectorRef = React.useRef(null);

  useEffect(() => {
    setCurrentPage(1);
  }, [ratioTypeFilter, searchKeyword]);

  const fetchAllChannels = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/ratio_sync/channels');

      if (res.data.success) {
        const channels = res.data.data || [];

        const transferData = channels.map((channel) => ({
          key: channel.id,
          label: channel.name,
          value: channel.id,
          disabled: false,
          _originalData: channel,
        }));

        setAllChannels(transferData);

        // 合并已有 endpoints，避免每次打开弹窗都重置
        setChannelEndpoints((prev) => {
          const merged = { ...prev };
          transferData.forEach((channel) => {
            const id = channel.key;
            const base = channel._originalData?.base_url || '';
            const name = channel.label || '';
            const channelType = channel._originalData?.type;
            const isOfficialRatioPreset =
              id === OFFICIAL_RATIO_PRESET_ID ||
              base === OFFICIAL_RATIO_PRESET_BASE_URL ||
              name === OFFICIAL_RATIO_PRESET_NAME;
            const isModelsDevPreset =
              id === MODELS_DEV_PRESET_ID ||
              base === MODELS_DEV_PRESET_BASE_URL ||
              name === MODELS_DEV_PRESET_NAME;
            const isOpenRouter = channelType === 20;
            if (!merged[id]) {
              if (isModelsDevPreset) {
                merged[id] = MODELS_DEV_PRESET_ENDPOINT;
              } else if (isOfficialRatioPreset) {
                merged[id] = OFFICIAL_RATIO_PRESET_ENDPOINT;
              } else if (isOpenRouter) {
                merged[id] = 'openrouter';
              } else {
                merged[id] = DEFAULT_ENDPOINT;
              }
            }
          });
          return merged;
        });
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(t('获取渠道失败：') + error.message);
    } finally {
      setLoading(false);
    }
  };

  const confirmChannelSelection = () => {
    const selected = allChannels
      .filter((ch) => selectedChannelIds.includes(ch.value))
      .map((ch) => ch._originalData);

    if (selected.length === 0) {
      showWarning(t('请至少选择一个渠道'));
      return;
    }

    setModalVisible(false);
    fetchRatiosFromChannels(selected);
  };

  const fetchRatiosFromChannels = async (channelList) => {
    setSyncLoading(true);

    const upstreams = channelList.map((ch) => ({
      id: ch.id,
      name: ch.name,
      base_url: ch.base_url,
      endpoint: channelEndpoints[ch.id] || DEFAULT_ENDPOINT,
    }));

    const payload = {
      upstreams: upstreams,
      timeout: 10,
      // 应用同步后再次拉取时仍返回已与上游一致的模型行，避免 differences 被清空
      include_aligned: true,
    };

    try {
      const res = await API.post('/api/ratio_sync/fetch', payload);

      if (!res.data.success) {
        showError(res.data.message || t('后端请求失败'));
        setSyncLoading(false);
        return;
      }

      const { differences = {}, test_results = [] } = res.data.data;

      const errorResults = test_results.filter((r) => r.status === 'error');
      if (errorResults.length > 0) {
        showWarning(
          t('部分渠道测试失败：') +
            errorResults.map((r) => `${r.name}: ${r.error}`).join(', '),
        );
      }

      setDifferences(differences);
      setResolutions({});
      setHasSynced(true);

      if (Object.keys(differences).length === 0) {
        showSuccess(t('未找到差异化倍率，无需同步'));
      }
    } catch (e) {
      showError(t('请求后端接口失败：') + e.message);
    } finally {
      setSyncLoading(false);
    }
  };

  function getBillingCategory(ratioType) {
    return ratioType === 'model_price' ? 'price' : 'ratio';
  }

  const selectValue = useCallback(
    (model, ratioType, value) => {
      const category = getBillingCategory(ratioType);

      setResolutions((prev) => {
        const newModelRes = { ...(prev[model] || {}) };

        Object.keys(newModelRes).forEach((rt) => {
          if (getBillingCategory(rt) !== category) {
            delete newModelRes[rt];
          }
        });

        newModelRes[ratioType] = value;

        return {
          ...prev,
          [model]: newModelRes,
        };
      });
    },
    [setResolutions],
  );

  const buildPricingBaseline = useCallback(() => {
    const global = {
      ModelRatio: parseNestedOption(props.options.ModelRatio),
      CompletionRatio: parseNestedOption(props.options.CompletionRatio),
      CacheRatio: parseNestedOption(props.options.CacheRatio),
      ModelPrice: parseNestedOption(props.options.ModelPrice),
    };
    const channel = {};
    CHANNEL_PRICING_OPTION_KEYS.forEach((k) => {
      channel[k] = parseNestedOption(props.options[k]);
    });
    return { global, channel };
  }, [props.options]);

  const applySync = async () => {
    const baseline = buildPricingBaseline();

    const conflicts = [];

    const findSourceChannel = (model, ratioType, value) => {
      const upMap = differences[model]?.[ratioType]?.upstreams;
      return findUpstreamColumnName(upMap, value) || t('未知');
    };

    const billingCategoryForChannel = (model, channelIdStr, chMaps) => {
      const priceMap = chMaps.ChannelModelPrice[channelIdStr];
      if (priceMap && priceMap[model] !== undefined) return 'price';
      const mr = chMaps.ChannelModelRatio[channelIdStr];
      const cr = chMaps.ChannelCompletionRatio[channelIdStr];
      const cache = chMaps.ChannelCacheRatio[channelIdStr];
      if (
        (mr && mr[model] !== undefined) ||
        (cr && cr[model] !== undefined) ||
        (cache && cache[model] !== undefined)
      )
        return 'ratio';
      return null;
    };

    const billingCategoryGlobal = (model, g) => {
      if (g.ModelPrice[model] !== undefined) return 'price';
      if (
        g.ModelRatio[model] !== undefined ||
        g.CompletionRatio[model] !== undefined ||
        g.CacheRatio[model] !== undefined
      )
        return 'ratio';
      return null;
    };

    Object.entries(resolutions).forEach(([model, ratios]) => {
      const groups = new Map();
      Object.entries(ratios).forEach(([ratioType, value]) => {
        const src = findSourceChannel(model, ratioType, value);
        const cid = parseUpstreamListChannelId(src);
        const tkey = cid !== null && cid > 0 ? `c:${cid}` : 'global';
        if (!groups.has(tkey)) {
          groups.set(tkey, { cid: cid > 0 ? cid : null, ratios: {} });
        }
        groups.get(tkey).ratios[ratioType] = value;
      });

      groups.forEach(({ cid, ratios: r }) => {
        const newCat = 'model_price' in r ? 'price' : 'ratio';
        const localCat =
          cid != null && cid > 0
            ? billingCategoryForChannel(model, String(cid), baseline.channel)
            : billingCategoryGlobal(model, baseline.global);

        if (localCat && localCat !== newCat) {
          let currentDesc = '';
          if (cid != null && cid > 0) {
            const idStr = String(cid);
            if (localCat === 'price') {
              currentDesc = `${t('固定价格')} : ${baseline.channel.ChannelModelPrice[idStr]?.[model]}`;
            } else {
              currentDesc = `${t('模型倍率')} : ${baseline.channel.ChannelModelRatio[idStr]?.[model] ?? '-'}\n${t('输出倍率')} : ${baseline.channel.ChannelCompletionRatio[idStr]?.[model] ?? '-'}`;
            }
          } else if (localCat === 'price') {
            currentDesc = `${t('固定价格')} : ${baseline.global.ModelPrice[model]}`;
          } else {
            currentDesc = `${t('模型倍率')} : ${baseline.global.ModelRatio[model] ?? '-'}\n${t('输出倍率')} : ${baseline.global.CompletionRatio[model] ?? '-'}`;
          }

          let newDesc = '';
          if (newCat === 'price') {
            newDesc = `${t('固定价格')} : ${r['model_price']}`;
          } else {
            const newModelRatio = r['model_ratio'] ?? '-';
            const newCompRatio = r['completion_ratio'] ?? '-';
            newDesc = `${t('模型倍率')} : ${newModelRatio}\n${t('输出倍率')} : ${newCompRatio}`;
          }

          const channels = Object.entries(r)
            .map(([rt, val]) => findSourceChannel(model, rt, val))
            .filter((v, idx, arr) => arr.indexOf(v) === idx)
            .join(', ');

          conflicts.push({
            channel: channels,
            model,
            current: currentDesc,
            newVal: newDesc,
          });
        }
      });
    });

    if (conflicts.length > 0) {
      setConflictItems(conflicts);
      setConfirmVisible(true);
      return;
    }

    await performSync(baseline);
  };

  const performSync = useCallback(
    async (baseline) => {
      const findSourceChannel = (model, ratioType, value) => {
        const upMap = differences[model]?.[ratioType]?.upstreams;
        return findUpstreamColumnName(upMap, value) || t('未知');
      };

      const finalRatios = {
        ModelRatio: { ...baseline.global.ModelRatio },
        CompletionRatio: { ...baseline.global.CompletionRatio },
        CacheRatio: { ...baseline.global.CacheRatio },
        ModelPrice: { ...baseline.global.ModelPrice },
      };

      const finalChannel = {};
      CHANNEL_PRICING_OPTION_KEYS.forEach((k) => {
        finalChannel[k] = cloneDeep(baseline.channel[k] || {});
      });

      const getCurrentEffectiveModelRatio = (model, cid) => {
        if (cid != null && cid > 0) {
          const channelRatio = baseline.channel.ChannelModelRatio?.[String(cid)]?.[
            model
          ];
          const channelRatioNum = toFiniteNumber(channelRatio);
          if (channelRatioNum !== null) {
            return channelRatioNum;
          }
        }
        return toFiniteNumber(baseline.global.ModelRatio?.[model]);
      };

      const normalizeRatioSelection = (model, cid, rawRatios) => {
        const normalized = { ...rawRatios };
        const selectedModelRatio = toFiniteNumber(normalized.model_ratio);

        let effectiveModelRatio = selectedModelRatio;
        if (effectiveModelRatio !== null) {
          const normalizedInputPrice = ratioToDisplayPrice(effectiveModelRatio);
          if (normalizedInputPrice !== null) {
            effectiveModelRatio = roundToPlaces(
              normalizedInputPrice / RATIO_TO_PRICE_FACTOR,
              RATIO_DECIMAL_PLACES,
            );
            normalized.model_ratio = effectiveModelRatio;
          }
        } else {
          effectiveModelRatio = getCurrentEffectiveModelRatio(model, cid);
        }

        const selectedCompletionRatio = toFiniteNumber(
          normalized.completion_ratio,
        );
        if (
          selectedCompletionRatio !== null &&
          effectiveModelRatio !== null &&
          effectiveModelRatio > 0
        ) {
          const normalizedOutputPrice = ratioToDisplayPrice(
            effectiveModelRatio * selectedCompletionRatio,
          );
          if (normalizedOutputPrice !== null) {
            const inputPrice = effectiveModelRatio * RATIO_TO_PRICE_FACTOR;
            if (inputPrice > 0) {
              normalized.completion_ratio = roundToPlaces(
                normalizedOutputPrice / inputPrice,
                RATIO_DECIMAL_PLACES,
              );
            }
          }
        }

        return normalized;
      };

      Object.entries(resolutions).forEach(([model, ratios]) => {
        const groups = new Map();
        Object.entries(ratios).forEach(([ratioType, value]) => {
          const src = findSourceChannel(model, ratioType, value);
          const cid = parseUpstreamListChannelId(src);
          const tkey = cid !== null && cid > 0 ? `c:${cid}` : 'global';
          if (!groups.has(tkey)) {
            groups.set(tkey, { cid: cid > 0 ? cid : null, ratios: {} });
          }
          groups.get(tkey).ratios[ratioType] = value;
        });

        groups.forEach(({ cid, ratios: r }) => {
          const normalizedRatios = normalizeRatioSelection(model, cid, r);
          const selectedTypes = Object.keys(normalizedRatios);
          const hasPrice = selectedTypes.includes('model_price');
          const hasRatio = selectedTypes.some((rt) => rt !== 'model_price');

          if (cid != null && cid > 0) {
            const idStr = String(cid);
            if (hasPrice) {
              ['ChannelModelRatio', 'ChannelCompletionRatio', 'ChannelCacheRatio'].forEach(
                (rk) => {
                  if (finalChannel[rk][idStr]) {
                    delete finalChannel[rk][idStr][model];
                  }
                },
              );
            }
            if (hasRatio && finalChannel.ChannelModelPrice[idStr]) {
              delete finalChannel.ChannelModelPrice[idStr][model];
            }

            Object.entries(normalizedRatios).forEach(([ratioType, value]) => {
              const ck = channelOptionKeyForRatioType(ratioType);
              if (!finalChannel[ck][idStr]) finalChannel[ck][idStr] = {};
              finalChannel[ck][idStr][model] =
                normalizeStoredNumber(value) ?? value;
            });
          } else {
            if (hasPrice) {
              delete finalRatios.ModelRatio[model];
              delete finalRatios.CompletionRatio[model];
              delete finalRatios.CacheRatio[model];
            }
            if (hasRatio) {
              delete finalRatios.ModelPrice[model];
            }

            Object.entries(normalizedRatios).forEach(([ratioType, value]) => {
              const gk = ratioTypeToPascal(ratioType);
              finalRatios[gk][model] = normalizeStoredNumber(value) ?? value;
            });
          }
        });
      });

      setLoading(true);
      try {
        const updates = [];
        const globalKeys = [
          'ModelRatio',
          'CompletionRatio',
          'CacheRatio',
          'ModelPrice',
        ];
        globalKeys.forEach((key) => {
          const before = JSON.stringify(baseline.global[key] || {});
          const after = JSON.stringify(finalRatios[key] || {});
          if (before !== after) {
            updates.push(
              API.put('/api/option/', {
                key,
                value: JSON.stringify(finalRatios[key], null, 2),
              }),
            );
          }
        });

        CHANNEL_PRICING_OPTION_KEYS.forEach((key) => {
          const before = JSON.stringify(baseline.channel[key] || {});
          const after = JSON.stringify(finalChannel[key] || {});
          if (before !== after) {
            updates.push(
              API.put('/api/option/', {
                key,
                value: JSON.stringify(finalChannel[key], null, 2),
              }),
            );
          }
        });

        if (updates.length === 0) {
          showWarning(t('没有需要保存的变更'));
          setLoading(false);
          return;
        }

        const results = await Promise.all(updates);

        if (results.every((res) => res.data.success)) {
          showSuccess(t('同步成功'));
          props.refresh();

          setDifferences((prevDifferences) => {
            const next = cloneDeep(prevDifferences);

            Object.entries(resolutions).forEach(([model, ratios]) => {
              Object.entries(ratios).forEach(([ratioType, value]) => {
                const diff = next[model]?.[ratioType];
                if (!diff || typeof diff.upstreams !== 'object') return;

                const srcName = findUpstreamColumnName(diff.upstreams, value);
                if (!srcName) return;

                const num = parseFloat(value);
                const synced = Number.isNaN(num)
                  ? value
                  : normalizeStoredNumber(num) ?? num;
                if (!diff.upstream_old) diff.upstream_old = {};
                diff.upstream_old[srcName] = synced;
                diff.upstreams[srcName] = synced;
              });
            });

            return next;
          });

          setResolutions({});
        } else {
          showError(t('部分保存失败'));
        }
      } catch (error) {
        showError(t('保存失败'));
      } finally {
        setLoading(false);
      }
    },
    [resolutions, differences, props.refresh, t],
  );

  const getCurrentPageData = (dataSource) => {
    const startIndex = (currentPage - 1) * pageSize;
    const endIndex = startIndex + pageSize;
    return dataSource.slice(startIndex, endIndex);
  };

  const renderHeader = () => (
    <div className='flex flex-col w-full'>
      <div className='flex flex-col md:flex-row justify-between items-center gap-4 w-full'>
        <div className='flex flex-col md:flex-row gap-2 w-full md:w-auto order-2 md:order-1'>
          <Button
            icon={<RefreshCcw size={14} />}
            className='w-full md:w-auto mt-2'
            onClick={() => {
              setModalVisible(true);
              if (allChannels.length === 0) {
                fetchAllChannels();
              }
            }}
          >
            {t('选择同步渠道')}
          </Button>

          {(() => {
            const hasSelections = Object.keys(resolutions).length > 0;

            return (
              <Button
                icon={<CheckSquare size={14} />}
                type='secondary'
                onClick={applySync}
                disabled={!hasSelections}
                className='w-full md:w-auto mt-2'
              >
                {t('应用同步')}
              </Button>
            );
          })()}

          <div className='flex flex-col sm:flex-row gap-2 w-full md:w-auto mt-2'>
            <Input
              prefix={<IconSearch size={14} />}
              placeholder={t('搜索模型名称')}
              value={searchKeyword}
              onChange={setSearchKeyword}
              className='w-full sm:w-64'
              showClear
            />

            <Select
              placeholder={t('按倍率类型筛选')}
              value={ratioTypeFilter}
              onChange={setRatioTypeFilter}
              className='w-full sm:w-48'
              showClear
              onClear={() => setRatioTypeFilter('')}
            >
              <Select.Option value='model_ratio'>{t('模型倍率')}</Select.Option>
              <Select.Option value='completion_ratio'>
                {t('输出倍率')}
              </Select.Option>
              <Select.Option value='cache_ratio'>{t('缓存倍率')}</Select.Option>
              <Select.Option value='model_price'>{t('固定价格')}</Select.Option>
            </Select>
          </div>
        </div>
      </div>
    </div>
  );

  const renderDifferenceTable = () => {
    const getModelRatioForDisplay = (model, upstreamName, mode) => {
      const modelRatioDiff = differences?.[model]?.model_ratio;
      if (!modelRatioDiff) return null;

      if (mode === 'current') {
        return toFiniteNumber(modelRatioDiff.current);
      }

      if (mode === 'old') {
        const upstreamOldVal = modelRatioDiff.upstream_old?.[upstreamName];
        if (upstreamOldVal !== undefined && upstreamOldVal !== null) {
          return toFiniteNumber(upstreamOldVal);
        }
      }

      if (mode === 'new') {
        const upstreamNewVal = modelRatioDiff.upstreams?.[upstreamName];
        if (upstreamNewVal !== undefined && upstreamNewVal !== null) {
          return toFiniteNumber(upstreamNewVal);
        }
      }

      return toFiniteNumber(modelRatioDiff.current);
    };

    const formatCurrentDisplayValue = (record) => {
      if (record.current === null || record.current === undefined) {
        return t('未设置');
      }

      if (record.ratioType === 'model_ratio') {
        const inputPrice = ratioToDisplayPrice(record.current);
        return inputPrice === null
          ? t('未设置')
          : `$${formatPriceNumber(inputPrice)}`;
      }

      if (record.ratioType === 'completion_ratio') {
        const completionRatio = toFiniteNumber(record.current);
        const inputRatio = getModelRatioForDisplay(record.model, null, 'current');
        if (completionRatio === null || inputRatio === null) return t('未设置');
        const outputPrice = ratioToDisplayPrice(inputRatio * completionRatio);
        return outputPrice === null
          ? t('未设置')
          : `$${formatPriceNumber(outputPrice)}`;
      }

      return String(record.current);
    };

    const formatUpstreamPairDisplay = (record, upstreamName) => {
      const oldVal = record.upstream_old?.[upstreamName];
      const newVal = record.upstreams?.[upstreamName];

      if (record.ratioType === 'model_ratio') {
        return formatPriceOldNewPair(
          ratioToDisplayPrice(oldVal),
          ratioToDisplayPrice(newVal),
        );
      }

      if (record.ratioType === 'completion_ratio') {
        const oldCompletion = toFiniteNumber(oldVal);
        const newCompletion = toFiniteNumber(newVal);
        const oldInputRatio = getModelRatioForDisplay(
          record.model,
          upstreamName,
          'old',
        );
        const newInputRatio = getModelRatioForDisplay(
          record.model,
          upstreamName,
          'new',
        );
        const oldOutputPrice =
          oldCompletion !== null && oldInputRatio !== null
            ? ratioToDisplayPrice(oldCompletion * oldInputRatio)
            : null;
        const newOutputPrice =
          newCompletion !== null && newInputRatio !== null
            ? ratioToDisplayPrice(newCompletion * newInputRatio)
            : null;
        return formatPriceOldNewPair(oldOutputPrice, newOutputPrice);
      }

      return formatOldNewPair(oldVal, newVal);
    };

    const dataSource = useMemo(() => {
      const tmp = [];

      Object.entries(differences).forEach(([model, ratioTypes]) => {
        const hasPrice = 'model_price' in ratioTypes;
        const hasOtherRatio = [
          'model_ratio',
          'completion_ratio',
          'cache_ratio',
        ].some((rt) => rt in ratioTypes);
        const billingConflict = hasPrice && hasOtherRatio;

        Object.entries(ratioTypes).forEach(([ratioType, diff]) => {
          tmp.push({
            key: `${model}_${ratioType}`,
            model,
            ratioType,
            current: diff.current,
            upstream_old: diff.upstream_old || {},
            upstreams: diff.upstreams,
            confidence: diff.confidence || {},
            billingConflict,
          });
        });
      });

      return tmp;
    }, [differences]);

    const filteredDataSource = useMemo(() => {
      if (!searchKeyword.trim() && !ratioTypeFilter) {
        return dataSource;
      }

      return dataSource.filter((item) => {
        const matchesKeyword =
          !searchKeyword.trim() ||
          item.model.toLowerCase().includes(searchKeyword.toLowerCase().trim());

        const matchesRatioType =
          !ratioTypeFilter || item.ratioType === ratioTypeFilter;

        return matchesKeyword && matchesRatioType;
      });
    }, [dataSource, searchKeyword, ratioTypeFilter]);

    const upstreamNames = useMemo(() => {
      const set = new Set();
      filteredDataSource.forEach((row) => {
        Object.keys(row.upstreams || {}).forEach((name) => set.add(name));
      });
      return Array.from(set);
    }, [filteredDataSource]);

    if (filteredDataSource.length === 0) {
      return (
        <Empty
          image={<IllustrationNoResult style={{ width: 150, height: 150 }} />}
          darkModeImage={
            <IllustrationNoResultDark style={{ width: 150, height: 150 }} />
          }
          description={
            searchKeyword.trim()
              ? t('未找到匹配的模型')
              : Object.keys(differences).length === 0
                ? hasSynced
                  ? t('暂无差异化倍率显示')
                  : t('请先选择同步渠道')
                : t('请先选择同步渠道')
          }
          style={{ padding: 30 }}
        />
      );
    }

    const columns = [
      {
        title: t('模型'),
        dataIndex: 'model',
        fixed: 'left',
      },
      {
        title: t('倍率类型'),
        dataIndex: 'ratioType',
        render: (text, record) => {
          const typeMap = {
            model_ratio: t('输入价格（由倍率换算）'),
            completion_ratio: t('输出价格（由倍率换算）'),
            cache_ratio: t('缓存倍率'),
            model_price: t('固定价格'),
          };
          const baseTag = (
            <Tag color={stringToColor(text)} shape='circle'>
              {typeMap[text] || text}
            </Tag>
          );
          if (record?.billingConflict) {
            return (
              <div className='flex items-center gap-1'>
                {baseTag}
                <Tooltip
                  position='top'
                  content={t(
                    '该模型存在固定价格与倍率计费方式冲突，请确认选择',
                  )}
                >
                  <AlertTriangle size={14} className='text-yellow-500' />
                </Tooltip>
              </div>
            );
          }
          return baseTag;
        },
      },
      {
        title: t('置信度'),
        dataIndex: 'confidence',
        render: (_, record) => {
          const allConfident = Object.values(record.confidence || {}).every(
            (v) => v !== false,
          );

          if (allConfident) {
            return (
              <Tooltip content={t('所有上游数据均可信')}>
                <Tag
                  color='green'
                  shape='circle'
                  type='light'
                  prefixIcon={<CheckCircle size={14} />}
                >
                  {t('可信')}
                </Tag>
              </Tooltip>
            );
          } else {
            const untrustedSources = Object.entries(record.confidence || {})
              .filter(([_, isConfident]) => isConfident === false)
              .map(([name]) => name)
              .join(', ');

            return (
              <Tooltip
                content={t('以下上游数据可能不可信：') + untrustedSources}
              >
                <Tag
                  color='yellow'
                  shape='circle'
                  type='light'
                  prefixIcon={<AlertTriangle size={14} />}
                >
                  {t('谨慎')}
                </Tag>
              </Tooltip>
            );
          }
        },
      },
      {
        title: t('当前值（系统默认）'),
        dataIndex: 'current',
        render: (_, record) => (
          <Tag
            color={
              record.current !== null && record.current !== undefined
                ? 'blue'
                : 'default'
            }
            shape='circle'
          >
            {formatCurrentDisplayValue(record)}
          </Tag>
        ),
      },
      ...upstreamNames.map((upName) => {
        const channelStats = (() => {
          let selectableCount = 0;
          let selectedCount = 0;

          filteredDataSource.forEach((row) => {
            const upstreamVal = row.upstreams?.[upName];
            const oldVal = row.upstream_old?.[upName];
            if (isUpstreamCellSelectable(oldVal, upstreamVal)) {
              selectableCount++;
              const isSelected = resolutionMatchesUpstream(
                resolutions[row.model]?.[row.ratioType],
                upstreamVal,
              );
              if (isSelected) {
                selectedCount++;
              }
            }
          });

          return {
            selectableCount,
            selectedCount,
            allSelected:
              selectableCount > 0 && selectedCount === selectableCount,
            partiallySelected:
              selectedCount > 0 && selectedCount < selectableCount,
            hasSelectableItems: selectableCount > 0,
          };
        })();

        const handleBulkSelect = (checked) => {
          if (checked) {
            filteredDataSource.forEach((row) => {
              const upstreamVal = row.upstreams?.[upName];
              const oldVal = row.upstream_old?.[upName];
              if (isUpstreamCellSelectable(oldVal, upstreamVal)) {
                selectValue(row.model, row.ratioType, upstreamVal);
              }
            });
          } else {
            setResolutions((prev) => {
              const newRes = { ...prev };
              filteredDataSource.forEach((row) => {
                if (newRes[row.model]) {
                  delete newRes[row.model][row.ratioType];
                  if (Object.keys(newRes[row.model]).length === 0) {
                    delete newRes[row.model];
                  }
                }
              });
              return newRes;
            });
          }
        };

        return {
          title: channelStats.hasSelectableItems ? (
            <Checkbox
              checked={channelStats.allSelected}
              indeterminate={channelStats.partiallySelected}
              onChange={(e) => handleBulkSelect(e.target.checked)}
            >
              {upName}
            </Checkbox>
          ) : (
            <span>{upName}</span>
          ),
          dataIndex: upName,
          render: (_, record) => {
            const upstreamVal = record.upstreams?.[upName];
            const oldVal = record.upstream_old?.[upName];
            const isConfident = record.confidence?.[upName] !== false;
            const pairText = formatUpstreamPairDisplay(record, upName);

            if (upstreamVal === null || upstreamVal === undefined) {
              return (
                <Tag color='default' shape='circle'>
                  {t('未设置')}
                </Tag>
              );
            }

            const selectable = isUpstreamCellSelectable(oldVal, upstreamVal);
            const isSelected = resolutionMatchesUpstream(
              resolutions[record.model]?.[record.ratioType],
              upstreamVal,
            );

            const pairEl = (
              <Tooltip content={t('旧价新价对照说明')}>
                <span
                  className='font-mono text-sm tabular-nums'
                  style={{ letterSpacing: '0.02em' }}
                >
                  {pairText}
                </span>
              </Tooltip>
            );

            if (!selectable) {
              return (
                <div className='flex items-center gap-2'>
                  {pairEl}
                  {!isConfident && (
                    <Tooltip
                      position='left'
                      content={t('该数据可能不可信，请谨慎使用')}
                    >
                      <AlertTriangle size={16} className='text-yellow-500' />
                    </Tooltip>
                  )}
                </div>
              );
            }

            return (
              <div className='flex items-center gap-2'>
                <Checkbox
                  checked={isSelected}
                  onChange={(e) => {
                    const isChecked = e.target.checked;
                    if (isChecked) {
                      selectValue(record.model, record.ratioType, upstreamVal);
                    } else {
                      setResolutions((prev) => {
                        const newRes = { ...prev };
                        if (newRes[record.model]) {
                          delete newRes[record.model][record.ratioType];
                          if (Object.keys(newRes[record.model]).length === 0) {
                            delete newRes[record.model];
                          }
                        }
                        return newRes;
                      });
                    }
                  }}
                >
                  {pairEl}
                </Checkbox>
                {!isConfident && (
                  <Tooltip
                    position='left'
                    content={t('该数据可能不可信，请谨慎使用')}
                  >
                    <AlertTriangle size={16} className='text-yellow-500' />
                  </Tooltip>
                )}
              </div>
            );
          },
        };
      }),
    ];

    return (
      <Table
        columns={columns}
        dataSource={getCurrentPageData(filteredDataSource)}
        pagination={{
          currentPage: currentPage,
          pageSize: pageSize,
          total: filteredDataSource.length,
          showSizeChanger: true,
          showQuickJumper: true,
          pageSizeOptions: ['5', '10', '20', '50'],
          onChange: (page, size) => {
            setCurrentPage(page);
            setPageSize(size);
          },
          onShowSizeChange: (current, size) => {
            setCurrentPage(1);
            setPageSize(size);
          },
        }}
        scroll={{ x: 'max-content' }}
        size='middle'
        loading={loading || syncLoading}
      />
    );
  };

  const updateChannelEndpoint = useCallback((channelId, endpoint) => {
    setChannelEndpoints((prev) => ({ ...prev, [channelId]: endpoint }));
  }, []);

  const handleModalClose = () => {
    setModalVisible(false);
    if (channelSelectorRef.current) {
      channelSelectorRef.current.resetPagination();
    }
  };

  return (
    <>
      <Form.Section text={renderHeader()}>
        {renderDifferenceTable()}
      </Form.Section>

      <ChannelSelectorModal
        ref={channelSelectorRef}
        t={t}
        visible={modalVisible}
        onCancel={handleModalClose}
        onOk={confirmChannelSelection}
        allChannels={allChannels}
        selectedChannelIds={selectedChannelIds}
        setSelectedChannelIds={setSelectedChannelIds}
        channelEndpoints={channelEndpoints}
        updateChannelEndpoint={updateChannelEndpoint}
      />

      <ConflictConfirmModal
        t={t}
        visible={confirmVisible}
        items={conflictItems}
        onOk={async () => {
          setConfirmVisible(false);
          await performSync(buildPricingBaseline());
        }}
        onCancel={() => setConfirmVisible(false)}
      />
    </>
  );
}
