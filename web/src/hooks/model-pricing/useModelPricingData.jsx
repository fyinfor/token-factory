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

import {
  useState,
  useEffect,
  useContext,
  useRef,
  useMemo,
  useCallback,
} from 'react';
import { useTranslation } from 'react-i18next';
import { API, copy, showError, showInfo, showSuccess } from '../../helpers';
import { fetchPerfMetricsSummary } from '../../helpers/perfMetrics';
import { Modal } from '@douyinfe/semi-ui';
import { UserContext } from '../../context/User';
import { StatusContext } from '../../context/Status';
import { modelMatchesSearchTerm } from '../../components/table/model-pricing/utils/channelRoute';
import {
  formatModelPriceNumber,
  MODEL_CARD_PRICE_MAX_DECIMALS,
} from '../../components/table/model-pricing/utils/priceDisplay';
import {
  getRelevantModelHotScore,
  getTopHotChannels,
  isTopHotModel,
  LIVE_HOT_FILTER,
} from '../../components/table/model-pricing/utils/modelHeat';

const mergeUniqueValues = (values) =>
  Array.from(
    new Set(
      values.filter(
        (value) => value !== undefined && value !== null && value !== '',
      ),
    ),
  );

const mergeTags = (models) =>
  mergeUniqueValues(
    models.flatMap((model) =>
      String(model.tags || '')
        .split(/[,;|]+/)
        .map((tag) => tag.trim())
        .filter(Boolean),
    ),
  ).join(',');

const getChannelMergeKey = (channel, fallbackIndex) => {
  if (channel?.channel_id !== undefined && channel?.channel_id !== null) {
    return `channel:${channel.channel_id}`;
  }
  return [
    'fallback',
    channel?.supplier_application_id,
    channel?.channel_no,
    channel?.route_slug,
    fallbackIndex,
  ]
    .map((part) => String(part ?? ''))
    .join(':');
};

const getChannelPriceScore = (channel, model) => {
  const mediaPrice = Number(
    channel?.video_flat_clip_hint?.min_usd_after_channel_discount ??
      channel?.image_per_image_hint?.min_usd_after_channel_discount,
  );
  if (Number.isFinite(mediaPrice) && mediaPrice >= 0) return mediaPrice;

  const isFixedPrice = model?.quota_type === 1 || channel?.quota_type === 1;
  const channelBase = Number(
    isFixedPrice ? channel?.model_price : channel?.model_ratio,
  );
  if (!Number.isFinite(channelBase)) return Number.POSITIVE_INFINITY;

  const rootBase = Number(
    isFixedPrice ? model?.model_price : model?.model_ratio,
  );
  const costDiscount = Number(channel?.price_discount_percent ?? 100) / 100;
  const markupRate = Number(channel?.markup_discount_rate || 0) / 100;
  const effective =
    channelBase * costDiscount +
    (Number.isFinite(rootBase) ? rootBase * markupRate : 0);
  return Number.isFinite(effective) ? effective : Number.POSITIVE_INFINITY;
};

const mergeModelsByName = (models) => {
  const groups = new Map();

  models.forEach((model) => {
    const modelName = String(model?.model_name || '');
    if (!groups.has(modelName)) groups.set(modelName, []);
    groups.get(modelName).push(model);
  });

  return Array.from(groups.values()).map((sameNameModels) => {
    const channelMap = new Map();
    const supplierMap = new Map();

    sameNameModels.forEach((model, modelIndex) => {
      const channels = Array.isArray(model.channel_list)
        ? model.channel_list
        : [];
      channels.forEach((channel, channelIndex) => {
        const decoratedChannel = {
          ...channel,
          ...(model.video_flat_clip_hint
            ? { video_flat_clip_hint: model.video_flat_clip_hint }
            : {}),
          ...(model.image_per_image_hint
            ? { image_per_image_hint: model.image_per_image_hint }
            : {}),
        };
        const key = getChannelMergeKey(
          decoratedChannel,
          `${modelIndex}-${channelIndex}`,
        );
        if (!channelMap.has(key)) channelMap.set(key, decoratedChannel);
      });

      const suppliers = Array.isArray(model.supplier_list)
        ? model.supplier_list
        : [];
      suppliers.forEach((supplier, supplierIndex) => {
        const key = String(
          supplier?.supplier_id ??
            `${supplier?.supplier_alias || 'supplier'}-${supplierIndex}`,
        );
        if (!supplierMap.has(key)) supplierMap.set(key, supplier);
      });
    });

    const channels = Array.from(channelMap.values()).sort((a, b) => {
      const scoreA = getChannelPriceScore(a, sameNameModels[0]);
      const scoreB = getChannelPriceScore(b, sameNameModels[0]);
      if (scoreA !== scoreB) return scoreA - scoreB;
      return Number(a?.channel_id || 0) - Number(b?.channel_id || 0);
    });
    const cheapestChannel = channels[0];
    const cheapestSource =
      sameNameModels.find((model) =>
        (model.channel_list || []).some(
          (channel) =>
            getChannelMergeKey(channel, '') ===
            getChannelMergeKey(cheapestChannel, ''),
        ),
      ) || sameNameModels[0];

    return {
      ...cheapestSource,
      key: cheapestSource.model_name,
      channel_list: channels,
      supplier_list: Array.from(supplierMap.values()),
      enable_groups: mergeUniqueValues(
        sameNameModels.flatMap((model) => model.enable_groups || []),
      ),
      supported_endpoint_types: mergeUniqueValues(
        sameNameModels.flatMap((model) => model.supported_endpoint_types || []),
      ),
      tags: mergeTags(sameNameModels),
      video_flat_clip_hint: cheapestChannel?.video_flat_clip_hint,
      image_per_image_hint: cheapestChannel?.image_per_image_hint,
    };
  });
};

export const useModelPricingData = (options = {}) => {
  const { defaultSortKey = 'default', mergeChannelsByModel = false } = options;
  const { t } = useTranslation();
  const [searchValue, setSearchValue] = useState('');
  const compositionRef = useRef({ isComposition: false });
  const [selectedRowKeys, setSelectedRowKeys] = useState([]);
  const [modalImageUrl, setModalImageUrl] = useState('');
  const [isModalOpenurl, setIsModalOpenurl] = useState(false);
  const [selectedGroup, setSelectedGroup] = useState('all');
  const [showModelDetail, setShowModelDetail] = useState(false);
  const [selectedModel, setSelectedModel] = useState(null);
  const [modelDetailSection, setModelDetailSection] = useState('general');
  const [filterGroup, setFilterGroup] = useState('all'); // 用于 Table 的可用分组筛选，"all" 表示不过滤
  const [filterQuotaType, setFilterQuotaType] = useState('all'); // 计费类型筛选: 'all' | 0 | 1
  const [filterEndpointType, setFilterEndpointType] = useState('all'); // 端点类型筛选: 'all' | string
  const [filterVendor, setFilterVendor] = useState('all'); // 供应商筛选: 'all' | 'unknown' | string
  const [filterTag, setFilterTag] = useState('all'); // 模型标签筛选: 'all' | string
  const [filterSupplierType, setFilterSupplierType] = useState('all'); // 供应商类型筛选: 'all' | string
  const [filterSupplier, setFilterSupplier] = useState('all'); // 供应商渠道筛选: 'all' | string (supplier_alias)
  // 排序键: 'default' | 'hot' | 'price' | 'supplier_grade' | 'latency' | 'discount'
  const [sortKey, setSortKey] = useState(defaultSortKey);
  const [pageSize, setPageSize] = useState(20);
  const [currentPage, setCurrentPage] = useState(1);
  const [currency, setCurrency] = useState('USD');
  const [showWithRecharge, setShowWithRecharge] = useState(false);
  const [tokenUnit, setTokenUnit] = useState('M');
  const [models, setModels] = useState([]);
  const [perfMetricsMap, setPerfMetricsMap] = useState({});
  const [vendorsMap, setVendorsMap] = useState({});
  const [loading, setLoading] = useState(true);
  const [groupRatio, setGroupRatio] = useState({});
  const [groupModelPrice, setGroupModelPrice] = useState({});
  const [groupModelRatio, setGroupModelRatio] = useState({});
  const [channelModelPrice, setChannelModelPrice] = useState({});
  const [channelModelRatio, setChannelModelRatio] = useState({});
  const [channelCompletionRatio, setChannelCompletionRatio] = useState({});
  const [channelCacheRatio, setChannelCacheRatio] = useState({});
  const [channelCreateCacheRatio, setChannelCreateCacheRatio] = useState({});
  const [channelImageRatio, setChannelImageRatio] = useState({});
  const [channelImagePrice, setChannelImagePrice] = useState({});
  const [channelAudioRatio, setChannelAudioRatio] = useState({});
  const [channelAudioCompletionRatio, setChannelAudioCompletionRatio] =
    useState({});
  const [channelVideoRatio, setChannelVideoRatio] = useState({});
  const [channelVideoCompletionRatio, setChannelVideoCompletionRatio] =
    useState({});
  const [channelVideoPrice, setChannelVideoPrice] = useState({});
  const [channelModelRequestTierPricing, setChannelModelRequestTierPricing] =
    useState({});
  const [globalModelRequestTierPricing, setGlobalModelRequestTierPricing] =
    useState({});
  const [pricingChannels, setPricingChannels] = useState([]);
  const [usableGroup, setUsableGroup] = useState({});
  const [endpointMap, setEndpointMap] = useState({});
  const [autoGroups, setAutoGroups] = useState([]);
  const [hotModelLimit, setHotModelLimit] = useState(8);
  const hasTimePricingRef = useRef(false);

  const [statusState] = useContext(StatusContext);
  const [userState] = useContext(UserContext);

  // 支持 /pricing?model=xxx 从首页广告等入口预填搜索
  useEffect(() => {
    try {
      const sp = new URLSearchParams(window.location.search);
      const m = sp.get('model');
      if (m) {
        setSearchValue(decodeURIComponent(m));
      }
    } catch {
      /* ignore */
    }
  }, []);

  // 充值汇率（price）与美元兑人民币汇率（usd_exchange_rate）
  const priceRate = useMemo(
    () => statusState?.status?.price ?? 1,
    [statusState],
  );
  const usdExchangeRate = useMemo(
    () => statusState?.status?.usd_exchange_rate ?? priceRate,
    [statusState, priceRate],
  );
  const customExchangeRate = useMemo(
    () => statusState?.status?.custom_currency_exchange_rate ?? 1,
    [statusState],
  );
  const customCurrencySymbol = useMemo(
    () => statusState?.status?.custom_currency_symbol ?? '¤',
    [statusState],
  );

  // 默认货币与站点展示类型同步；TOKENS 由视图层走倍率展示
  const siteDisplayType = useMemo(
    () => statusState?.status?.quota_display_type || 'USD',
    [statusState],
  );
  useEffect(() => {
    if (
      siteDisplayType === 'USD' ||
      siteDisplayType === 'CNY' ||
      siteDisplayType === 'CUSTOM'
    ) {
      setCurrency(siteDisplayType);
    }
  }, [siteDisplayType]);

  useEffect(() => {
    if (siteDisplayType === 'TOKENS') {
      setShowWithRecharge(false);
      setCurrency('USD');
    }
  }, [siteDisplayType]);

  const activeChannelHeatFilters = useMemo(
    () => ({ filterSupplier, filterSupplierType }),
    [filterSupplier, filterSupplierType],
  );
  const hotChannelResult = useMemo(
    () => getTopHotChannels(models, hotModelLimit, activeChannelHeatFilters),
    [models, hotModelLimit, activeChannelHeatFilters],
  );
  const hotChannelScoreMap = hotChannelResult.scoreMap;
  const hotPrimaryChannelMap = hotChannelResult.primaryChannelMap;
  const hotSourceMap = hotChannelResult.sourceMap;

  const filteredModels = useMemo(() => {
    let result = models;

    // 分组筛选
    if (filterGroup !== 'all') {
      result = result.filter((model) =>
        model.enable_groups.includes(filterGroup),
      );
    }

    // 计费类型筛选
    if (filterQuotaType !== 'all') {
      result = result.filter((model) => model.quota_type === filterQuotaType);
    }

    // 端点类型筛选
    if (filterEndpointType !== 'all') {
      result = result.filter(
        (model) =>
          model.supported_endpoint_types &&
          model.supported_endpoint_types.includes(filterEndpointType),
      );
    }

    // 供应商筛选
    if (filterVendor !== 'all') {
      if (filterVendor === 'unknown') {
        result = result.filter((model) => !model.vendor_name);
      } else {
        result = result.filter((model) => model.vendor_name === filterVendor);
      }
    }

    // 渠道供应商筛选（channel_list supplier_alias）
    if (filterSupplier !== 'all') {
      result = result.filter((model) => {
        if (!model.channel_list || model.channel_list.length === 0)
          return false;
        return model.channel_list.some(
          (ch) => (ch?.supplier_alias || '') === filterSupplier,
        );
      });
    }

    // 供应商类型筛选（channel_list supplier_type）
    if (filterSupplierType !== 'all') {
      result = result.filter((model) => {
        if (!model.channel_list || model.channel_list.length === 0)
          return false;
        return model.channel_list.some(
          (ch) => (ch?.supplier_type || '') === filterSupplierType,
        );
      });
    }

    // 标签筛选
    if (filterTag !== 'all') {
      result = result.filter((model) => {
        if (filterTag === LIVE_HOT_FILTER) {
          return isTopHotModel(
            model,
            hotChannelScoreMap,
            activeChannelHeatFilters,
          );
        }
        const tagLower = filterTag.toLowerCase();
        if (!model.tags) return false;
        const tagsArr = model.tags
          .toLowerCase()
          .split(/[,;|]+/)
          .map((tag) => tag.trim())
          .filter(Boolean);
        return tagsArr.includes(tagLower);
      });
    }

    // 搜索筛选（含渠道路径、route_slug、supplier_alias 等）
    if (searchValue.length > 0) {
      result = result.filter((model) =>
        modelMatchesSearchTerm(model, searchValue),
      );
    }

    if (sortKey && sortKey !== 'default') {
      const supplierGradeRank = (alias) => {
        if (!alias) return Number.POSITIVE_INFINITY;
        const m = /^P(\d+)$/i.exec(String(alias).trim());
        if (m) return parseInt(m[1], 10);
        // 非 P 等级（自定义别名）排到最后
        return Number.POSITIVE_INFINITY;
      };

      const modelUnitPrice = (m) => {
        const imageHint = m.image_per_image_hint;
        const imageMin = Number(imageHint?.min_usd_after_channel_discount);
        if (Number.isFinite(imageMin) && imageMin > 0) {
          return imageMin;
        }
        const list = Array.isArray(m.channel_list) ? m.channel_list : [];
        const pickField = m.quota_type === 1 ? 'model_price' : 'model_ratio';
        let best = Number.POSITIVE_INFINITY;
        for (const ch of list) {
          const v = Number(ch?.[pickField]);
          if (Number.isFinite(v) && v < best) best = v;
        }
        if (best === Number.POSITIVE_INFINITY) {
          const fallback = m.quota_type === 1 ? m.model_price : m.model_ratio;
          const v = Number(fallback);
          if (Number.isFinite(v)) best = v;
        }
        return best;
      };

      const modelMinSupplierRank = (m) => {
        const list = Array.isArray(m.channel_list) ? m.channel_list : [];
        let best = Number.POSITIVE_INFINITY;
        for (const ch of list) {
          const r = supplierGradeRank(ch?.supplier_alias);
          if (r < best) best = r;
        }
        return best;
      };

      const modelMinLatency = (m) => {
        const list = Array.isArray(m.channel_list) ? m.channel_list : [];
        let best = Number.POSITIVE_INFINITY;
        for (const ch of list) {
          const v = Number(ch?.test_response_time_ms);
          // 0 / 缺失视为未知，放最后
          if (Number.isFinite(v) && v > 0 && v < best) best = v;
        }
        return best;
      };

      const getTierDiscountRatio = (row) => {
        const current = Number(row?.usd_after_channel_discount || 0);
        const official = Number(row?.usd_official || 0);
        if (
          !Number.isFinite(current) ||
          !Number.isFinite(official) ||
          official <= 0 ||
          current >= official
        ) {
          return 0;
        }
        return 1 - current / official;
      };

      const getMaxTierDiscountRatio = (hint) => {
        const tiers = Array.isArray(hint?.tiers) ? hint.tiers : [];
        let best = 0;
        for (const row of tiers) {
          const discount = getTierDiscountRatio(row);
          if (discount > best) best = discount;
        }
        return best;
      };

      // 折扣率：按详情页同口径计算最终渠道价相对根价格的实际折扣
      const modelDiscountRatio = (m) => {
        const videoTierDiscount = getMaxTierDiscountRatio(
          m.video_flat_clip_hint,
        );
        if (videoTierDiscount > 0) return videoTierDiscount;

        const imageTierDiscount = getMaxTierDiscountRatio(
          m.image_per_image_hint,
        );
        if (imageTierDiscount > 0) return imageTierDiscount;

        const list = Array.isArray(m.channel_list) ? m.channel_list : [];
        const channel = list[0];
        if (!channel) return 0;

        const isFixedPrice = m.quota_type === 1 || channel.quota_type === 1;
        const channelBase = Number(
          isFixedPrice ? channel.model_price : channel.model_ratio,
        );
        const rootBase = Number(isFixedPrice ? m.model_price : m.model_ratio);
        if (
          !Number.isFinite(channelBase) ||
          !Number.isFinite(rootBase) ||
          rootBase <= 0
        ) {
          return 0;
        }

        const costDisc =
          (channel.price_discount_percent != null
            ? Number(channel.price_discount_percent)
            : 100) / 100;
        const markupRate = (Number(channel.markup_discount_rate) || 0) / 100;
        const effective = channelBase * costDisc + rootBase * markupRate;
        if (!Number.isFinite(effective) || effective >= rootBase) return 0;
        return 1 - effective / rootBase;
      };

      const tieBreak = (a, b) => {
        if (a.quota_type !== b.quota_type) return a.quota_type - b.quota_type;
        return String(a.model_name).localeCompare(String(b.model_name));
      };

      const cmpAsc = (av, bv, a, b) => {
        if (av === bv) return tieBreak(a, b);
        return av < bv ? -1 : 1;
      };

      const cmpDesc = (av, bv, a, b) => {
        if (av === bv) return tieBreak(a, b);
        return av > bv ? -1 : 1;
      };

      // 折扣率排序：按实际折扣降序，相同则按实时热度降序。
      const cmpDiscount = (a, b) => {
        const discountA = modelDiscountRatio(a);
        const discountB = modelDiscountRatio(b);
        if (discountA !== discountB) {
          return discountA > discountB ? -1 : 1;
        }
        const heatA = getRelevantModelHotScore(
          a,
          hotChannelScoreMap,
          activeChannelHeatFilters,
        );
        const heatB = getRelevantModelHotScore(
          b,
          hotChannelScoreMap,
          activeChannelHeatFilters,
        );
        if (heatA !== heatB) {
          return heatA > heatB ? -1 : 1;
        }
        return tieBreak(a, b);
      };

      result = [...result].sort((a, b) => {
        switch (sortKey) {
          case 'hot':
            return cmpDesc(
              getRelevantModelHotScore(
                a,
                hotChannelScoreMap,
                activeChannelHeatFilters,
              ),
              getRelevantModelHotScore(
                b,
                hotChannelScoreMap,
                activeChannelHeatFilters,
              ),
              a,
              b,
            );
          case 'price':
            return cmpAsc(modelUnitPrice(a), modelUnitPrice(b), a, b);
          case 'supplier_grade':
            return cmpAsc(
              modelMinSupplierRank(a),
              modelMinSupplierRank(b),
              a,
              b,
            );
          case 'latency':
            return cmpAsc(modelMinLatency(a), modelMinLatency(b), a, b);
          case 'discount':
            return cmpDiscount(a, b);
          default:
            return tieBreak(a, b);
        }
      });
    }

    return result;
  }, [
    models,
    searchValue,
    filterGroup,
    filterQuotaType,
    filterEndpointType,
    filterVendor,
    filterTag,
    filterSupplier,
    filterSupplierType,
    sortKey,
    hotChannelScoreMap,
    activeChannelHeatFilters,
  ]);

  const rowSelection = useMemo(
    () => ({
      selectedRowKeys,
      onChange: (keys) => {
        setSelectedRowKeys(keys);
      },
    }),
    [selectedRowKeys],
  );

  const displayPrice = useCallback(
    (usdPrice, { precision = MODEL_CARD_PRICE_MAX_DECIMALS } = {}) => {
      let priceInUSD = usdPrice;
      if (showWithRecharge) {
        priceInUSD = (usdPrice * priceRate) / usdExchangeRate;
      }

      if (currency === 'CNY') {
        return `¥${formatModelPriceNumber(
          priceInUSD * usdExchangeRate,
          precision,
        )}`;
      } else if (currency === 'CUSTOM') {
        return `${customCurrencySymbol}${formatModelPriceNumber(
          priceInUSD * customExchangeRate,
          precision,
        )}`;
      }
      return `$${formatModelPriceNumber(priceInUSD, precision)}`;
    },
    [
      currency,
      customCurrencySymbol,
      customExchangeRate,
      priceRate,
      showWithRecharge,
      usdExchangeRate,
    ],
  );

  const getModelRowKey = (model, index) => {
    const parts = [
      model.model_name,
      model.quota_type,
      model.vendor_id,
      Array.isArray(model.supported_endpoint_types)
        ? model.supported_endpoint_types.join(',')
        : '',
      index,
    ];
    return parts
      .map((part) => (part === undefined || part === null ? '' : String(part)))
      .join('::');
  };

  const setModelsFormat = (models, groupRatio, vendorMap) => {
    const formattedModels = models.map((model) => ({ ...model }));
    for (let i = 0; i < formattedModels.length; i++) {
      const m = formattedModels[i];
      m.key = getModelRowKey(m, i);
      m.group_ratio = groupRatio[m.model_name];

      if (m.vendor_id && vendorMap[m.vendor_id]) {
        const vendor = vendorMap[m.vendor_id];
        m.vendor_name = vendor.name;
        m.vendor_name_en = vendor.name_en;
        m.vendor_icon = vendor.icon;
        m.vendor_description = vendor.description;
      }

      if (!m.channel_list) {
        m.channel_list = [];
      }
    }
    const displayModels = mergeChannelsByModel
      ? mergeModelsByName(formattedModels)
      : formattedModels;

    displayModels.sort((a, b) => {
      return a.quota_type - b.quota_type;
    });

    displayModels.sort((a, b) => {
      if (a.model_name.startsWith('gpt') && !b.model_name.startsWith('gpt')) {
        return -1;
      } else if (
        !a.model_name.startsWith('gpt') &&
        b.model_name.startsWith('gpt')
      ) {
        return 1;
      } else {
        return a.model_name.localeCompare(b.model_name);
      }
    });

    setModels(displayModels);
    setSelectedModel((current) => {
      if (!current) return current;
      const updated = displayModels.find(
        (model) => model.model_name === current.model_name,
      );
      if (!updated) return current;
      return current.preferred_hot_channel_id
        ? {
            ...updated,
            preferred_hot_channel_id: current.preferred_hot_channel_id,
          }
        : updated;
    });
  };

  const loadPricing = async ({ silent = false } = {}) => {
    if (!silent) setLoading(true);
    let url = '/api/pricing';
    const res = await API.get(url);
    const {
      success,
      message,
      data,
      vendors,
      group_ratio,
      group_model_price,
      group_model_ratio,
      channel_model_price,
      channel_model_ratio,
      channel_completion_ratio,
      channel_cache_ratio,
      channel_create_cache_ratio,
      channel_image_ratio,
      channel_image_price,
      channel_audio_ratio,
      channel_audio_completion_ratio,
      channel_video_ratio,
      channel_video_completion_ratio,
      channel_video_price,
      channel_model_request_tier_pricing,
      model_request_tier_pricing,
      channels,
      usable_group,
      supported_endpoint,
      auto_groups,
      hot_model_limit,
    } = res.data;
    if (success) {
      hasTimePricingRef.current = Array.isArray(data)
        ? data.some((model) =>
            (model?.channel_list || []).some(
              (channel) => channel?.time_pricing?.has_schedules === true,
            ),
          )
        : false;
      setGroupRatio(group_ratio);
      setGroupModelPrice(group_model_price || {});
      setGroupModelRatio(group_model_ratio || {});
      setChannelModelPrice(channel_model_price || {});
      setChannelModelRatio(channel_model_ratio || {});
      setChannelCompletionRatio(channel_completion_ratio || {});
      setChannelCacheRatio(channel_cache_ratio || {});
      setChannelCreateCacheRatio(channel_create_cache_ratio || {});
      setChannelImageRatio(channel_image_ratio || {});
      setChannelImagePrice(channel_image_price || {});
      setChannelAudioRatio(channel_audio_ratio || {});
      setChannelAudioCompletionRatio(channel_audio_completion_ratio || {});
      setChannelVideoRatio(channel_video_ratio || {});
      setChannelVideoCompletionRatio(channel_video_completion_ratio || {});
      setChannelVideoPrice(channel_video_price || {});
      setChannelModelRequestTierPricing(
        channel_model_request_tier_pricing || {},
      );
      setGlobalModelRequestTierPricing(model_request_tier_pricing || {});
      setPricingChannels(channels || []);
      setUsableGroup(usable_group);
      if (!silent) setSelectedGroup('all');
      // 构建供应商 Map 方便查找
      const vendorMap = {};
      if (Array.isArray(vendors)) {
        vendors.forEach((v) => {
          vendorMap[v.id] = v;
        });
      }
      setVendorsMap(vendorMap);
      setEndpointMap(supported_endpoint || {});
      setAutoGroups(auto_groups || []);
      setHotModelLimit(
        Number(hot_model_limit) > 0 ? Number(hot_model_limit) : 8,
      );
      setModelsFormat(data, group_ratio, vendorMap);
    } else {
      if (!silent) showError(message);
    }
    if (!silent) setLoading(false);
  };

  const loadPerfMetrics = async () => {
    try {
      const map = await fetchPerfMetricsSummary(24);
      setPerfMetricsMap(map);
    } catch (err) {
      setPerfMetricsMap({});
    }
  };

  const refresh = async () => {
    await Promise.all([loadPricing(), loadPerfMetrics()]);
  };

  const copyText = useCallback(
    async (text) => {
      const copyValue = Array.isArray(text)
        ? text
            .map(
              (key) =>
                models.find((model) => model.key === key)?.model_name ?? key,
            )
            .join('\n')
        : text;
      if (await copy(copyValue)) {
        showSuccess(t('已复制：') + copyValue);
      } else {
        Modal.error({
          title: t('无法复制到剪贴板，请手动复制'),
          content: copyValue,
        });
      }
    },
    [models, t],
  );

  const handleChange = (value) => {
    const newSearchValue = value ? value : '';
    setSearchValue(newSearchValue);
    compositionRef.current.isComposition = false;
  };

  const handleCompositionStart = () => {
    compositionRef.current.isComposition = true;
  };

  const handleCompositionEnd = (event) => {
    compositionRef.current.isComposition = false;
    const value = event.target.value;
    const newSearchValue = value ? value : '';
    setSearchValue(newSearchValue);
  };

  const handleGroupClick = (group) => {
    setSelectedGroup(group);
    setFilterGroup(group);
    if (group === 'all') {
      showInfo(t('已切换至最优倍率视图，每个模型使用其最低倍率分组'));
    } else {
      showInfo(
        t('当前查看的分组为：{{group}}，倍率为：{{ratio}}', {
          group: group,
          ratio: groupRatio[group] ?? 1,
        }),
      );
    }
  };

  const openModelDetail = useCallback(
    (model, section = 'general') => {
      const preferredHotChannelId = hotPrimaryChannelMap.get(
        String(model?.model_name || ''),
      );
      setModelDetailSection(section === 'basic' ? 'basic' : 'general');
      setSelectedModel(
        preferredHotChannelId
          ? { ...model, preferred_hot_channel_id: preferredHotChannelId }
          : model,
      );
      setShowModelDetail(true);
    },
    [hotPrimaryChannelMap],
  );

  const closeModelDetail = useCallback(() => {
    setShowModelDetail(false);
    setTimeout(() => {
      setSelectedModel(null);
    }, 300);
  }, []);

  useEffect(() => {
    refresh().then();
  }, []);

  useEffect(() => {
    const intervalId = window.setInterval(() => {
      if (!hasTimePricingRef.current) return;
      loadPricing({ silent: true }).catch(() => {});
    }, 60000);
    return () => window.clearInterval(intervalId);
  }, []);

  // 当筛选/排序变化时重置到第一页
  useEffect(() => {
    setCurrentPage(1);
  }, [
    filterGroup,
    filterQuotaType,
    filterEndpointType,
    filterVendor,
    filterTag,
    filterSupplier,
    filterSupplierType,
    searchValue,
    sortKey,
  ]);

  return {
    // 状态
    searchValue,
    setSearchValue,
    selectedRowKeys,
    setSelectedRowKeys,
    modalImageUrl,
    setModalImageUrl,
    isModalOpenurl,
    setIsModalOpenurl,
    selectedGroup,
    setSelectedGroup,
    showModelDetail,
    setShowModelDetail,
    selectedModel,
    setSelectedModel,
    modelDetailSection,
    setModelDetailSection,
    filterGroup,
    setFilterGroup,
    filterQuotaType,
    setFilterQuotaType,
    filterEndpointType,
    setFilterEndpointType,
    filterVendor,
    setFilterVendor,
    filterTag,
    setFilterTag,
    filterSupplierType,
    setFilterSupplierType,
    filterSupplier,
    setFilterSupplier,
    sortKey,
    setSortKey,
    pageSize,
    setPageSize,
    currentPage,
    setCurrentPage,
    currency,
    setCurrency,
    siteDisplayType,
    showWithRecharge,
    setShowWithRecharge,
    tokenUnit,
    setTokenUnit,
    models,
    hotChannelScoreMap,
    hotPrimaryChannelMap,
    hotSourceMap,
    hotModelLimit,
    perfMetricsMap,
    loading,
    groupRatio,
    groupModelPrice,
    groupModelRatio,
    channelModelPrice,
    channelModelRatio,
    channelCompletionRatio,
    channelCacheRatio,
    channelCreateCacheRatio,
    channelImageRatio,
    channelImagePrice,
    channelAudioRatio,
    channelAudioCompletionRatio,
    channelVideoRatio,
    channelVideoCompletionRatio,
    channelVideoPrice,
    channelModelRequestTierPricing,
    globalModelRequestTierPricing,
    pricingChannels,
    usableGroup,
    endpointMap,
    autoGroups,

    // 计算属性
    priceRate,
    usdExchangeRate,
    filteredModels,
    rowSelection,

    // 供应商
    vendorsMap,

    // 用户和状态
    userState,
    statusState,

    // 方法
    displayPrice,
    refresh,
    copyText,
    handleChange,
    handleCompositionStart,
    handleCompositionEnd,
    handleGroupClick,
    openModelDetail,
    closeModelDetail,

    // 引用
    compositionRef,

    // 国际化
    t,
  };
};
