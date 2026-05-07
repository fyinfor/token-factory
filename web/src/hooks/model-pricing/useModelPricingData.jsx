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

import { useState, useEffect, useContext, useRef, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { API, copy, showError, showInfo, showSuccess } from '../../helpers';
import { Modal } from '@douyinfe/semi-ui';
import { UserContext } from '../../context/User';
import { StatusContext } from '../../context/Status';

export const useModelPricingData = () => {
  const { t } = useTranslation();
  const [searchValue, setSearchValue] = useState('');
  const compositionRef = useRef({ isComposition: false });
  const [selectedRowKeys, setSelectedRowKeys] = useState([]);
  const [modalImageUrl, setModalImageUrl] = useState('');
  const [isModalOpenurl, setIsModalOpenurl] = useState(false);
  const [selectedGroup, setSelectedGroup] = useState('all');
  const [showModelDetail, setShowModelDetail] = useState(false);
  const [selectedModel, setSelectedModel] = useState(null);
  const [filterGroup, setFilterGroup] = useState('all'); // 用于 Table 的可用分组筛选，"all" 表示不过滤
  const [filterQuotaType, setFilterQuotaType] = useState('all'); // 计费类型筛选: 'all' | 0 | 1
  const [filterEndpointType, setFilterEndpointType] = useState('all'); // 端点类型筛选: 'all' | string
  const [filterVendor, setFilterVendor] = useState('all'); // 供应商筛选: 'all' | 'unknown' | string
  const [filterTag, setFilterTag] = useState('all'); // 模型标签筛选: 'all' | string
  // 排序键: 'default' | 'price' | 'supplier_grade' | 'latency' | 'discount'
  const [sortKey, setSortKey] = useState('default');
  const [pageSize, setPageSize] = useState(20);
  const [currentPage, setCurrentPage] = useState(1);
  const [currency, setCurrency] = useState('USD');
  const [showWithRecharge, setShowWithRecharge] = useState(false);
  const [tokenUnit, setTokenUnit] = useState('M');
  const [models, setModels] = useState([]);
  const [vendorsMap, setVendorsMap] = useState({});
  const [loading, setLoading] = useState(true);
  const [groupRatio, setGroupRatio] = useState({});
  const [groupModelPrice, setGroupModelPrice] = useState({});
  const [groupModelRatio, setGroupModelRatio] = useState({});
  const [channelModelPrice, setChannelModelPrice] = useState({});
  const [channelModelRatio, setChannelModelRatio] = useState({});
  const [pricingChannels, setPricingChannels] = useState([]);
  const [usableGroup, setUsableGroup] = useState({});
  const [endpointMap, setEndpointMap] = useState({});
  const [autoGroups, setAutoGroups] = useState([]);

  const [statusState] = useContext(StatusContext);
  const [userState] = useContext(UserContext);

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

    // 标签筛选
    if (filterTag !== 'all') {
      const tagLower = filterTag.toLowerCase();
      result = result.filter((model) => {
        if (!model.tags) return false;
        const tagsArr = model.tags
          .toLowerCase()
          .split(/[,;|]+/)
          .map((tag) => tag.trim())
          .filter(Boolean);
        return tagsArr.includes(tagLower);
      });
    }

    // 搜索筛选
    if (searchValue.length > 0) {
      const searchTerm = searchValue.toLowerCase();
      result = result.filter(
        (model) =>
          (model.model_name &&
            model.model_name.toLowerCase().includes(searchTerm)) ||
          (model.description &&
            model.description.toLowerCase().includes(searchTerm)) ||
          (model.tags && model.tags.toLowerCase().includes(searchTerm)) ||
          (model.vendor_name &&
            model.vendor_name.toLowerCase().includes(searchTerm)),
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

      // 折扣率：1 - 渠道最低价 / 根价格；无折扣或数据缺失返回 0
      const modelDiscountRatio = (m) => {
        const list = Array.isArray(m.channel_list) ? m.channel_list : [];
        const pickField = m.quota_type === 1 ? 'model_price' : 'model_ratio';
        const rootVal = Number(m[pickField]);
        if (!Number.isFinite(rootVal) || rootVal <= 0) return 0;
        let minChannel = Number.POSITIVE_INFINITY;
        for (const ch of list) {
          const v = Number(ch?.[pickField]);
          if (Number.isFinite(v) && v < minChannel) minChannel = v;
        }
        if (!Number.isFinite(minChannel) || minChannel >= rootVal) return 0;
        return 1 - minChannel / rootVal;
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

      result = [...result].sort((a, b) => {
        switch (sortKey) {
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
            return cmpDesc(modelDiscountRatio(a), modelDiscountRatio(b), a, b);
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
    sortKey,
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

  const displayPrice = (usdPrice) => {
    let priceInUSD = usdPrice;
    if (showWithRecharge) {
      priceInUSD = (usdPrice * priceRate) / usdExchangeRate;
    }

    if (currency === 'CNY') {
      return `¥${parseFloat((priceInUSD * usdExchangeRate).toFixed(2))}`;
    } else if (currency === 'CUSTOM') {
      return `${customCurrencySymbol}${parseFloat((priceInUSD * customExchangeRate).toFixed(2))}`;
    }
    return `$${parseFloat(priceInUSD.toFixed(2))}`;
  };

  const setModelsFormat = (models, groupRatio, vendorMap) => {
    for (let i = 0; i < models.length; i++) {
      const m = models[i];
      m.key = m.model_name;
      m.group_ratio = groupRatio[m.model_name];

      if (m.vendor_id && vendorMap[m.vendor_id]) {
        const vendor = vendorMap[m.vendor_id];
        m.vendor_name = vendor.name;
        m.vendor_icon = vendor.icon;
        m.vendor_description = vendor.description;
      }

      if (!m.channel_list) {
        m.channel_list = [];
      }
    }
    models.sort((a, b) => {
      return a.quota_type - b.quota_type;
    });

    models.sort((a, b) => {
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

    setModels(models);
  };

  const loadPricing = async () => {
    setLoading(true);
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
      channels,
      usable_group,
      supported_endpoint,
      auto_groups,
    } = res.data;
    if (success) {
      setGroupRatio(group_ratio);
      setGroupModelPrice(group_model_price || {});
      setGroupModelRatio(group_model_ratio || {});
      setChannelModelPrice(channel_model_price || {});
      setChannelModelRatio(channel_model_ratio || {});
      setPricingChannels(channels || []);
      setUsableGroup(usable_group);
      setSelectedGroup('all');
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
      setModelsFormat(data, group_ratio, vendorMap);
    } else {
      showError(message);
    }
    setLoading(false);
  };

  const refresh = async () => {
    await loadPricing();
  };

  const copyText = async (text) => {
    if (await copy(text)) {
      showSuccess(t('已复制：') + text);
    } else {
      Modal.error({ title: t('无法复制到剪贴板，请手动复制'), content: text });
    }
  };

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

  const openModelDetail = (model) => {
    setSelectedModel(model);
    setShowModelDetail(true);
  };

  const closeModelDetail = () => {
    setShowModelDetail(false);
    setTimeout(() => {
      setSelectedModel(null);
    }, 300);
  };

  useEffect(() => {
    refresh().then();
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
    loading,
    groupRatio,
    groupModelPrice,
    groupModelRatio,
    channelModelPrice,
    channelModelRatio,
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
