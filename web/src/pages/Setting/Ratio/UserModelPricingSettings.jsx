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

import React, { useEffect, useMemo, useState } from 'react';
import {
  Banner,
  Button,
  Checkbox,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Radio,
  RadioGroup,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../../helpers';

const { Text } = Typography;

const PAGE_SIZE = 20;

const EXPORT_FIELD_OPTIONS = [
  { value: 'model_name', labelKey: '模型', defaultChecked: true },
  { value: 'mode', labelKey: '模式', defaultChecked: true },
  { value: 'price_discount_percent', labelKey: '成本折扣', defaultChecked: true },
  { value: 'operating_cost_percent', labelKey: '经营成本', defaultChecked: true },
  { value: 'markup_discount_rate', labelKey: '加价折扣', defaultChecked: true },
  { value: 'total_percent', labelKey: '总折扣', defaultChecked: true },
  { value: 'enabled', labelKey: '启用', defaultChecked: true },
  { value: 'updated_time', labelKey: '更新时间', defaultChecked: true },
  { value: 'username', labelKey: '用户名', defaultChecked: false },
  { value: 'user_id', labelKey: '用户ID', defaultChecked: false },
];

const defaultExportFields = () =>
  EXPORT_FIELD_OPTIONS.filter((o) => o.defaultChecked).map((o) => o.value);

const formatExportTimestamp = () => {
  const d = new Date();
  const pad = (n) => String(n).padStart(2, '0');
  return `${d.getFullYear()}${pad(d.getMonth() + 1)}${pad(d.getDate())}-${pad(
    d.getHours(),
  )}${pad(d.getMinutes())}${pad(d.getSeconds())}`;
};

const downloadBlob = (blob, filename) => {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  URL.revokeObjectURL(url);
};

const getDownloadFilename = (disposition, fallback) => {
  const text = String(disposition || '');
  const utf8Match = text.match(/filename\*=UTF-8''([^;]+)/i);
  if (utf8Match?.[1]) {
    try {
      return decodeURIComponent(utf8Match[1]);
    } catch {
      return fallback;
    }
  }
  const asciiMatch = text.match(/filename="?([^";]+)"?/i);
  return asciiMatch?.[1] || fallback;
};

const emptyForm = {
  id: 0,
  user_id: undefined,
  model_name: '',
  mode: 'price_cap',
  price_discount_percent: 100,
  operating_cost_percent: 0,
  markup_discount_rate: 0,
  enabled: true,
  channels: [],
};

const emptyImportForm = {
  enabled: true,
};

const MODE_PRICE_CAP = 'price_cap';
const MODE_CHANNEL_LIST = 'channel_list';

const modeLabel = (mode, t) =>
  mode === MODE_CHANNEL_LIST ? t('渠道清单') : t('价格上限');

const pickEditable = (row) => ({
  mode: row.mode === MODE_CHANNEL_LIST ? MODE_CHANNEL_LIST : MODE_PRICE_CAP,
  price_discount_percent: Number(row.price_discount_percent) || 0,
  operating_cost_percent: Number(row.operating_cost_percent) || 0,
  markup_discount_rate: Number(row.markup_discount_rate) || 0,
  enabled: !!row.enabled,
  channels: normalizeChannels(row.channels),
});

const normalizeChannels = (channels) => {
  const list = Array.isArray(channels) ? channels : [];
  const seen = new Set();
  const out = [];
  for (const ch of list) {
    const id = Number(ch?.channel_id) || 0;
    if (id <= 0 || seen.has(id)) continue;
    seen.add(id);
    out.push({ channel_id: id, priority: out.length + 1 });
  }
  return out;
};

const sameChannels = (a, b) => {
  const aa = normalizeChannels(a);
  const bb = normalizeChannels(b);
  if (aa.length !== bb.length) return false;
  for (let i = 0; i < aa.length; i += 1) {
    if (aa[i].channel_id !== bb[i].channel_id) return false;
  }
  return true;
};

const sameEditable = (a, b) =>
  (a.mode || MODE_PRICE_CAP) === (b.mode || MODE_PRICE_CAP) &&
  Number(a.price_discount_percent) === Number(b.price_discount_percent) &&
  Number(a.operating_cost_percent) === Number(b.operating_cost_percent) &&
  Number(a.markup_discount_rate) === Number(b.markup_discount_rate) &&
  !!a.enabled === !!b.enabled &&
  sameChannels(a.channels, b.channels);

const calcTotalPercent = (row) =>
  (Number(row.price_discount_percent) || 0) +
  (Number(row.operating_cost_percent) || 0) +
  (Number(row.markup_discount_rate) || 0);

/**
 * UserModelPricingSettings 用户指定价管理（按用户视角）：
 * 先选用户，再在表格内统一填三项折扣；支持一键导入当前已定价模型。
 */
export default function UserModelPricingSettings() {
  const { t } = useTranslation();
  const [items, setItems] = useState([]);
  const [baselines, setBaselines] = useState({});
  const [loading, setLoading] = useState(false);
  const [filterModel, setFilterModel] = useState('');
  const [currentPage, setCurrentPage] = useState(1);

  const [pricingUsers, setPricingUsers] = useState([]);
  const [selectedUserId, setSelectedUserId] = useState(undefined);
  const [userOptions, setUserOptions] = useState([]);

  const [modalVisible, setModalVisible] = useState(false);
  const [saving, setSaving] = useState(false);
  const [form, setForm] = useState(emptyForm);
  const [modelOptions, setModelOptions] = useState([]);

  const [importVisible, setImportVisible] = useState(false);
  const [importForm, setImportForm] = useState(emptyImportForm);
  const [importing, setImporting] = useState(false);
  const [importPreview, setImportPreview] = useState([]);
  const [importPreviewLoading, setImportPreviewLoading] = useState(false);

  const [exportVisible, setExportVisible] = useState(false);
  const [exportFields, setExportFields] = useState(defaultExportFields);
  const [exportOnlyFiltered, setExportOnlyFiltered] = useState(true);
  const [exporting, setExporting] = useState(false);

  const [preview, setPreview] = useState(null);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [previewVisible, setPreviewVisible] = useState(false);
  const [previewTitle, setPreviewTitle] = useState('');

  const [channelOptions, setChannelOptions] = useState([]);
  const [channelOptionsLoading, setChannelOptionsLoading] = useState(false);

  const [convertingMode, setConvertingMode] = useState(false);
  const [convertVisible, setConvertVisible] = useState(false);
  const [convertModelNames, setConvertModelNames] = useState([]);

  const [savingRowId, setSavingRowId] = useState(null);
  const [savingAll, setSavingAll] = useState(false);

  const totalPercent = useMemo(
    () => calcTotalPercent(form),
    [form],
  );

  const selectedUserLabel = useMemo(() => {
    if (!selectedUserId) return '';
    const fromSummary = pricingUsers.find((u) => u.user_id === selectedUserId);
    if (fromSummary) {
      return `${fromSummary.username || t('未知用户')} #${selectedUserId}`;
    }
    const fromOpts = userOptions.find((o) => o.value === selectedUserId);
    return fromOpts?.label || `#${selectedUserId}`;
  }, [selectedUserId, pricingUsers, userOptions, t]);

  const mapUsersToOptions = (users) =>
    (users || [])
      .filter((u) => !u.DeletedAt)
      .map((u) => ({
        value: u.id,
        label: `${u.username}${u.display_name ? ` (${u.display_name})` : ''} #${u.id}`,
      }));

  const syncItems = (data) => {
    const list = data || [];
    setItems(list);
    const next = {};
    for (const row of list) {
      next[row.id] = pickEditable(row);
    }
    setBaselines(next);
  };

  const loadPricingUsers = async () => {
    try {
      const res = await API.get('/api/user_model_pricing/users');
      if (res.data.success) {
        setPricingUsers(res.data.data || []);
      }
    } catch (e) {
      // 不阻塞主流程
    }
  };

  const loadList = async (userId) => {
    const uid = userId ?? selectedUserId;
    if (!uid) {
      syncItems([]);
      return;
    }
    setLoading(true);
    try {
      const res = await API.get(`/api/user_model_pricing/?user_id=${uid}`);
      const { success, message, data } = res.data;
      if (success) {
        syncItems(data || []);
      } else {
        showError(message);
      }
    } catch (e) {
      showError(e?.response?.data?.message || t('加载失败'));
    } finally {
      setLoading(false);
    }
  };

  const loadModels = async () => {
    try {
      const res = await API.get('/api/channel/models_enabled');
      if (res.data.success) {
        const models = res.data.data || [];
        setModelOptions(
          models.map((m) => (typeof m === 'string' ? m : m.model_name || m.id)),
        );
      }
    } catch (e) {
      // 模型列表加载失败不阻塞页面，仍可手动输入
    }
  };

  // 管理员视角：空关键字返回全量用户列表（首页 100 条），输入后再远程过滤。
  const searchUsers = async (keyword = '') => {
    try {
      const res = await API.get(
        `/api/user/search?keyword=${encodeURIComponent(keyword)}&p=1&page_size=100`,
      );
      if (res.data.success) {
        const users = res.data.data?.items || res.data.data || [];
        setUserOptions(mapUsersToOptions(users));
      }
    } catch (e) {
      // 静默失败，输入框可继续搜索
    }
  };

  useEffect(() => {
    loadPricingUsers();
    loadModels();
    searchUsers('');
  }, []);

  useEffect(() => {
    setCurrentPage(1);
    if (selectedUserId) {
      loadList(selectedUserId);
    } else {
      syncItems([]);
    }
  }, [selectedUserId]);

  const refreshAll = async ({ resetPage = false } = {}) => {
    if (resetPage) {
      setCurrentPage(1);
    }
    await loadPricingUsers();
    if (selectedUserId) {
      await loadList(selectedUserId);
    }
  };

  const openAdd = () => {
    if (!selectedUserId) {
      showError(t('请先选择用户'));
      return;
    }
    setForm({
      ...emptyForm,
      user_id: selectedUserId,
    });
    setChannelOptions([]);
    setPreview(null);
    setModalVisible(true);
  };

  const openEdit = (row) => {
    if (!row) return;
    setForm({
      id: row.id,
      user_id: row.user_id,
      model_name: row.model_name,
      mode: row.mode === MODE_CHANNEL_LIST ? MODE_CHANNEL_LIST : MODE_PRICE_CAP,
      price_discount_percent: Number(row.price_discount_percent) || 0,
      operating_cost_percent: Number(row.operating_cost_percent) || 0,
      markup_discount_rate: Number(row.markup_discount_rate) || 0,
      enabled: !!row.enabled,
      channels: normalizeChannels(row.channels),
    });
    setPreview(null);
    setModalVisible(true);
    if (row.model_name) {
      loadChannelOptions(row.model_name, {
        price_discount_percent: row.price_discount_percent,
        operating_cost_percent: row.operating_cost_percent,
        markup_discount_rate: row.markup_discount_rate,
      });
    }
  };

  const loadChannelOptions = async (modelName, discounts = {}) => {
    if (!modelName) {
      setChannelOptions([]);
      return;
    }
    setChannelOptionsLoading(true);
    try {
      const params = new URLSearchParams({
        model_name: modelName,
        mode: MODE_CHANNEL_LIST,
        price_discount_percent: String(discounts.price_discount_percent ?? 100),
        operating_cost_percent: String(discounts.operating_cost_percent ?? 0),
        markup_discount_rate: String(discounts.markup_discount_rate ?? 0),
      });
      const res = await API.get(`/api/user_model_pricing/preview?${params}`);
      if (res.data.success) {
        setChannelOptions(res.data.data?.channels || []);
      } else {
        showError(res.data.message);
      }
    } catch (e) {
      showError(e?.response?.data?.message || t('加载渠道失败'));
    } finally {
      setChannelOptionsLoading(false);
    }
  };

  const openImport = async () => {
    if (!selectedUserId) {
      showError(t('请先选择用户'));
      return;
    }
    setImportForm({ enabled: true });
    setImportPreview([]);
    setImportVisible(true);
    setImportPreviewLoading(true);
    try {
      const res = await API.get(
        `/api/user_model_pricing/import_preview?user_id=${selectedUserId}`,
      );
      if (res.data.success) {
        setImportPreview(res.data.data?.items || []);
      } else {
        showError(res.data.message);
      }
    } catch (e) {
      showError(e?.response?.data?.message || t('预览失败'));
    } finally {
      setImportPreviewLoading(false);
    }
  };

  const openExport = () => {
    if (!selectedUserId) {
      showError(t('请先选择用户'));
      return;
    }
    if (!items.length) {
      showError(t('该用户暂无指定价配置可导出'));
      return;
    }
    setExportFields(defaultExportFields());
    setExportOnlyFiltered(!!filterModel);
    setExportVisible(true);
  };

  const doExport = async () => {
    if (!selectedUserId) {
      showError(t('请先选择用户'));
      return;
    }
    if (!exportFields.length) {
      showError(t('请至少选择一个导出字段'));
      return;
    }
    const fallbackName = `user-model-pricing-${selectedUserId}-${formatExportTimestamp()}.xlsx`;
    setExporting(true);
    try {
      const params = {
        user_id: selectedUserId,
        fields: exportFields.join(','),
      };
      if (exportOnlyFiltered && filterModel) {
        params.model_name = filterModel;
      }
      const res = await API.get('/api/user_model_pricing/export', {
        responseType: 'blob',
        disableDuplicate: true,
        params,
      });
      const contentType =
        res.headers?.['content-type'] || res.data?.type || '';
      if (contentType.includes('application/json')) {
        const text = await res.data.text();
        try {
          const payload = JSON.parse(text);
          showError(payload?.message || t('导出失败'));
        } catch {
          showError(text || t('导出失败'));
        }
        return;
      }
      const blob = new Blob([res.data], {
        type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
      });
      downloadBlob(
        blob,
        getDownloadFilename(
          res.headers?.['content-disposition'],
          fallbackName,
        ),
      );
      showSuccess(t('导出成功'));
      setExportVisible(false);
    } catch (e) {
      showError(e?.response?.data?.message || t('导出失败'));
    } finally {
      setExporting(false);
    }
  };

  const doPreview = async (target, title = '') => {
    if (!target?.model_name) {
      showError(t('请先选择模型'));
      return;
    }
    setPreviewLoading(true);
    setPreviewTitle(title || target.model_name);
    setPreviewVisible(true);
    try {
      const params = new URLSearchParams({
        model_name: target.model_name,
        mode: target.mode === MODE_CHANNEL_LIST ? MODE_CHANNEL_LIST : MODE_PRICE_CAP,
        price_discount_percent: String(target.price_discount_percent ?? 100),
        operating_cost_percent: String(target.operating_cost_percent ?? 0),
        markup_discount_rate: String(target.markup_discount_rate ?? 0),
      });
      const selected = normalizeChannels(target.channels)
        .map((c) => c.channel_id)
        .join(',');
      if (selected) {
        params.set('selected_channel_ids', selected);
      }
      const res = await API.get(`/api/user_model_pricing/preview?${params}`);
      if (res.data.success) {
        setPreview(res.data.data);
      } else {
        showError(res.data.message);
      }
    } catch (e) {
      showError(e?.response?.data?.message || t('预览失败'));
    } finally {
      setPreviewLoading(false);
    }
  };

  const upsertPricing = async (payload) => {
    const mode =
      payload.mode === MODE_CHANNEL_LIST ? MODE_CHANNEL_LIST : MODE_PRICE_CAP;
    const body = {
      user_id: payload.user_id,
      model_name: payload.model_name,
      mode,
      price_discount_percent: Number(payload.price_discount_percent) || 0,
      operating_cost_percent: Number(payload.operating_cost_percent) || 0,
      markup_discount_rate: Number(payload.markup_discount_rate) || 0,
      enabled: !!payload.enabled,
    };
    if (mode === MODE_CHANNEL_LIST) {
      body.channels = normalizeChannels(payload.channels);
    }
    const res = await API.post('/api/user_model_pricing/', body);
    return res.data;
  };

  const applySavedRow = (saved, fallbackRow) => {
    const next = {
      ...fallbackRow,
      ...(saved || {}),
      mode:
        (saved?.mode || fallbackRow.mode) === MODE_CHANNEL_LIST
          ? MODE_CHANNEL_LIST
          : MODE_PRICE_CAP,
      channels: normalizeChannels(saved?.channels ?? fallbackRow.channels),
      total_percent:
        saved?.total_percent ??
        calcTotalPercent({ ...fallbackRow, ...(saved || {}) }),
    };
    setItems((prev) =>
      prev.map((it) => (it.id === fallbackRow.id ? { ...it, ...next } : it)),
    );
    setBaselines((prev) => ({
      ...prev,
      [fallbackRow.id]: pickEditable(next),
    }));
  };

  const doSave = async () => {
    const uid = form.user_id || selectedUserId;
    if (!uid || !form.model_name) {
      showError(t('请选择用户和模型'));
      return;
    }
    if (
      form.mode === MODE_CHANNEL_LIST &&
      normalizeChannels(form.channels).length === 0
    ) {
      showError(t('渠道清单模式请至少勾选一个渠道'));
      return;
    }
    setSaving(true);
    try {
      const { success, message } = await upsertPricing({
        ...form,
        user_id: uid,
      });
      if (success) {
        showSuccess(t('保存成功'));
        setModalVisible(false);
        await refreshAll({ resetPage: false });
      } else {
        showError(message);
      }
    } catch (e) {
      showError(e?.response?.data?.message || t('保存失败'));
    } finally {
      setSaving(false);
    }
  };

  const doSaveRow = async (row) => {
    if (!row?.user_id || !row?.model_name) {
      showError(t('请选择用户和模型'));
      return;
    }
    if (
      row.mode === MODE_CHANNEL_LIST &&
      normalizeChannels(row.channels).length === 0
    ) {
      showError(t('渠道清单模式请至少勾选一个渠道'));
      return;
    }
    setSavingRowId(row.id);
    try {
      const { success, message, data } = await upsertPricing(row);
      if (success) {
        showSuccess(t('保存成功'));
        applySavedRow(data, row);
        await loadPricingUsers();
      } else {
        showError(message);
      }
    } catch (e) {
      showError(e?.response?.data?.message || t('保存失败'));
    } finally {
      setSavingRowId(null);
    }
  };

  const doSaveAllDirty = async () => {
    const dirty = items.filter((row) => {
      const base = baselines[row.id];
      return base && !sameEditable(row, base);
    });
    if (!dirty.length) {
      showError(t('没有待保存的修改'));
      return;
    }
    setSavingAll(true);
    let ok = 0;
    let fail = 0;
    try {
      for (const row of dirty) {
        try {
          const { success, message, data } = await upsertPricing(row);
          if (success) {
            applySavedRow(data, row);
            ok += 1;
          } else {
            fail += 1;
            showError(`${row.model_name}: ${message}`);
          }
        } catch (e) {
          fail += 1;
          showError(
            `${row.model_name}: ${e?.response?.data?.message || t('保存失败')}`,
          );
        }
      }
      if (ok > 0) {
        showSuccess(
          fail > 0
            ? `${t('已保存')} ${ok} ${t('条')}，${t('失败')} ${fail} ${t('条')}`
            : `${t('已保存')} ${ok} ${t('条修改')}`,
        );
        await loadPricingUsers();
      }
    } finally {
      setSavingAll(false);
    }
  };

  const doImport = async () => {
    if (!selectedUserId) {
      showError(t('请先选择用户'));
      return;
    }
    if (!importPreview.length) {
      showError(t('没有可导入的模型'));
      return;
    }
    setImporting(true);
    try {
      const res = await API.post('/api/user_model_pricing/import', {
        user_id: selectedUserId,
        enabled: !!importForm.enabled,
      });
      if (res.data.success) {
        const d = res.data.data || {};
        showSuccess(
          `${t('导入完成')}：${t('新建')} ${d.created ?? 0}，${t('更新')} ${d.updated ?? 0}（${t('共')} ${d.total_models ?? 0} ${t('个模型')}）`,
        );
        setImportVisible(false);
        await refreshAll({ resetPage: true });
      } else {
        showError(res.data.message);
      }
    } catch (e) {
      showError(e?.response?.data?.message || t('导入失败'));
    } finally {
      setImporting(false);
    }
  };

  const openConvertChannelList = () => {
    if (!selectedUserId) {
      showError(t('请先选择用户'));
      return;
    }
    if (!items.length) {
      showError(t('该用户暂无指定价配置'));
      return;
    }
    setConvertModelNames([]);
    setConvertVisible(true);
  };

  const convertTargetCount = convertModelNames.length
    ? convertModelNames.length
    : items.length;

  const doConvertToChannelList = async () => {
    if (!selectedUserId) {
      showError(t('请先选择用户'));
      return;
    }
    if (!items.length) {
      showError(t('该用户暂无指定价配置'));
      return;
    }
    setConvertingMode(true);
    try {
      const payload = { user_id: selectedUserId };
      if (convertModelNames.length > 0) {
        payload.model_names = convertModelNames;
      }
      const res = await API.post(
        '/api/user_model_pricing/convert_channel_list',
        payload,
      );
      if (res.data.success) {
        const d = res.data.data || {};
        const skipped = d.skipped ?? 0;
        const scopeTip = convertModelNames.length
          ? t('所选模型')
          : t('全部模型');
        showSuccess(
          skipped > 0
            ? `${t('已切换')}（${scopeTip}）${d.converted ?? 0} ${t('个为渠道清单')}，${t('跳过')} ${skipped} ${t('个')}`
            : `${t('已切换')}（${scopeTip}）${d.converted ?? 0} ${t('个为渠道清单')}`,
        );
        setConvertVisible(false);
        await refreshAll({ resetPage: false });
      } else {
        showError(res.data.message);
      }
    } catch (e) {
      showError(e?.response?.data?.message || t('切换失败'));
    } finally {
      setConvertingMode(false);
    }
  };

  const doDelete = async (row) => {
    try {
      const res = await API.delete(`/api/user_model_pricing/${row.id}`);
      if (res.data.success) {
        showSuccess(t('删除成功'));
        setItems((prev) => prev.filter((it) => it.id !== row.id));
        setBaselines((prev) => {
          const next = { ...prev };
          delete next[row.id];
          return next;
        });
        await loadPricingUsers();
      } else {
        showError(res.data.message);
      }
    } catch (e) {
      showError(e?.response?.data?.message || t('删除失败'));
    }
  };

  const doClearUser = async () => {
    if (!selectedUserId) return;
    try {
      const res = await API.delete(
        `/api/user_model_pricing/by_user/${selectedUserId}`,
      );
      if (res.data.success) {
        showSuccess(
          `${t('已清空')} ${res.data.data?.deleted ?? 0} ${t('条指定价')}`,
        );
        await refreshAll({ resetPage: true });
      } else {
        showError(res.data.message);
      }
    } catch (e) {
      showError(e?.response?.data?.message || t('清空失败'));
    }
  };

  const updateRowField = (rowId, key, value) => {
    setItems((prev) =>
      prev.map((it) => {
        if (it.id !== rowId) return it;
        const next = { ...it, [key]: value };
        next.total_percent = calcTotalPercent(next);
        return next;
      }),
    );
  };

  const filteredItems = useMemo(() => {
    if (!filterModel) return items;
    const kw = filterModel.toLowerCase();
    return items.filter((it) =>
      (it.model_name || '').toLowerCase().includes(kw),
    );
  }, [items, filterModel]);

  const dirtyCount = useMemo(() => {
    let count = 0;
    for (const row of items) {
      const base = baselines[row.id];
      if (base && !sameEditable(row, base)) count += 1;
    }
    return count;
  }, [items, baselines]);

  // 顶部用户选择：已有配置的用户优先，合并搜索结果
  const mergedUserOptions = useMemo(() => {
    const map = new Map();
    for (const u of pricingUsers) {
      map.set(u.user_id, {
        value: u.user_id,
        label: `${u.username || t('未知用户')} #${u.user_id}（${u.model_count}${t('个模型')}）`,
      });
    }
    for (const o of userOptions) {
      if (!map.has(o.value)) {
        map.set(o.value, o);
      }
    }
    if (selectedUserId && !map.has(selectedUserId)) {
      map.set(selectedUserId, {
        value: selectedUserId,
        label: selectedUserLabel,
      });
    }
    return Array.from(map.values());
  }, [pricingUsers, userOptions, selectedUserId, selectedUserLabel, t]);

  const convertModelOptions = useMemo(
    () =>
      items.map((row) => ({
        value: row.model_name,
        label: `${row.model_name}（${modeLabel(row.mode, t)} · ${Math.round(calcTotalPercent(row) * 100) / 100}%）`,
      })),
    [items, t],
  );

  const maxPage = Math.max(1, Math.ceil(filteredItems.length / PAGE_SIZE) || 1);
  const safePage = Math.min(currentPage, maxPage);

  useEffect(() => {
    if (currentPage !== safePage) {
      setCurrentPage(safePage);
    }
  }, [currentPage, safePage]);

  const columns = [
    {
      title: t('模型'),
      dataIndex: 'model_name',
      width: 200,
      render: (text) => <Tag color='blue'>{text}</Tag>,
    },
    {
      title: t('模式'),
      dataIndex: 'mode',
      width: 120,
      render: (v) => (
        <Tag color={v === MODE_CHANNEL_LIST ? 'purple' : 'cyan'}>
          {modeLabel(v, t)}
        </Tag>
      ),
    },
    {
      title: t('渠道'),
      width: 110,
      render: (_, row) =>
        row.mode === MODE_CHANNEL_LIST ? (
          <Tag color='purple'>
            {normalizeChannels(row.channels).length} {t('个')}
          </Tag>
        ) : (
          <Text type='tertiary'>{t('按上限自动')}</Text>
        ),
    },
    {
      title: t('成本折扣') + ' (%)',
      dataIndex: 'price_discount_percent',
      width: 130,
      render: (v, row) => (
        <InputNumber
          style={{ width: '100%' }}
          min={0}
          max={1000}
          value={v}
          onChange={(val) =>
            updateRowField(row.id, 'price_discount_percent', val ?? 0)
          }
        />
      ),
    },
    {
      title: t('经营成本') + ' (%)',
      dataIndex: 'operating_cost_percent',
      width: 130,
      render: (v, row) => (
        <InputNumber
          style={{ width: '100%' }}
          min={0}
          max={1000}
          value={v}
          onChange={(val) =>
            updateRowField(row.id, 'operating_cost_percent', val ?? 0)
          }
        />
      ),
    },
    {
      title: t('加价折扣') + ' (%)',
      dataIndex: 'markup_discount_rate',
      width: 130,
      render: (v, row) => (
        <InputNumber
          style={{ width: '100%' }}
          min={0}
          max={1000}
          value={v}
          onChange={(val) =>
            updateRowField(row.id, 'markup_discount_rate', val ?? 0)
          }
        />
      ),
    },
    {
      title: t('总折扣'),
      dataIndex: 'total_percent',
      width: 100,
      render: (_, row) => (
        <Tag color='green' size='large'>
          {Math.round(calcTotalPercent(row) * 100) / 100}%
        </Tag>
      ),
    },
    {
      title: t('状态'),
      dataIndex: 'enabled',
      width: 90,
      render: (v, row) => (
        <Switch
          checked={!!v}
          onChange={(checked) => updateRowField(row.id, 'enabled', checked)}
        />
      ),
    },
    {
      title: t('更新时间'),
      dataIndex: 'updated_time',
      width: 170,
      render: (v) => (v ? new Date(v * 1000).toLocaleString() : '-'),
    },
    {
      title: t('操作'),
      width: 280,
      fixed: 'right',
      render: (_, row) => {
        const dirty =
          baselines[row.id] && !sameEditable(row, baselines[row.id]);
        return (
          <Space wrap>
            <Button
              size='small'
              theme='solid'
              type='primary'
              disabled={!dirty}
              loading={savingRowId === row.id}
              onClick={() => doSaveRow(row)}
            >
              {t('保存')}
            </Button>
            <Button size='small' onClick={() => openEdit(row)}>
              {t('编辑')}
            </Button>
            <Button
              size='small'
              onClick={() => doPreview(row, row.model_name)}
            >
              {t('预览')}
            </Button>
            <Popconfirm
              title={t('确认删除该指定价配置？')}
              content={t('删除后该用户对该模型恢复默认渠道定价与选路')}
              onConfirm={() => doDelete(row)}
            >
              <Button size='small' type='danger'>
                {t('删除')}
              </Button>
            </Popconfirm>
          </Space>
        );
      },
    },
  ];

  const setField = (key, value) => {
    setForm((prev) => ({ ...prev, [key]: value }));
  };

  const toggleFormChannel = (channelId, checked) => {
    setForm((prev) => {
      const current = normalizeChannels(prev.channels);
      let next;
      if (checked) {
        if (current.some((c) => c.channel_id === channelId)) {
          next = current;
        } else {
          next = [...current, { channel_id: channelId, priority: current.length + 1 }];
        }
      } else {
        next = current.filter((c) => c.channel_id !== channelId);
      }
      return { ...prev, channels: normalizeChannels(next) };
    });
  };

  const moveFormChannel = (channelId, direction) => {
    setForm((prev) => {
      const list = normalizeChannels(prev.channels);
      const idx = list.findIndex((c) => c.channel_id === channelId);
      if (idx < 0) return prev;
      const swapWith = idx + direction;
      if (swapWith < 0 || swapWith >= list.length) return prev;
      const next = [...list];
      [next[idx], next[swapWith]] = [next[swapWith], next[idx]];
      return { ...prev, channels: normalizeChannels(next) };
    });
  };

  const selectAllWithinCapChannels = () => {
    const ids = (channelOptions || [])
      .filter((ch) => ch.within_cap)
      .map((ch) => ch.channel_id);
    const existingPri = new Map(
      normalizeChannels(form.channels).map((c) => [c.channel_id, c.priority]),
    );
    // 保留已选顺序，追加未选的 within_cap 渠道
    const ordered = normalizeChannels(form.channels).map((c) => c.channel_id);
    for (const id of ids) {
      if (!existingPri.has(id) && !ordered.includes(id)) {
        ordered.push(id);
      }
    }
    setForm((prev) => ({
      ...prev,
      channels: normalizeChannels(ordered.map((id) => ({ channel_id: id }))),
    }));
  };

  const setImportField = (key, value) => {
    setImportForm((prev) => ({ ...prev, [key]: value }));
  };

  const handleFilterChange = (v) => {
    setFilterModel(v);
    setCurrentPage(1);
  };

  return (
    <div>
      <Banner
        type='info'
        closeIcon={null}
        className='!rounded-lg mb-3'
        description={t(
          '按用户管理指定价：普通用户计费为「全局官方价 × 总折扣」。代理身份仅用指定价约束可选渠道，自用仍按渠道成本价（加价为 0），与无指定价时一致。价格上限模式：自动放行单价 ≤ 指定售价的渠道；渠道清单模式：手动勾选可用渠道并排序，首页模型详情与智能路由仅展示勾选渠道。走智能路由时在勾选集内按智能规则选渠，否则按手动优先级。',
        )}
      />

      <div className='flex flex-wrap items-center gap-3 mb-3'>
        <Text strong>{t('管理用户')}</Text>
        <Select
          style={{ minWidth: 320 }}
          filter
          remote
          showClear
          placeholder={t('选择用户（已配置用户优先列出，可搜索全部）')}
          optionList={mergedUserOptions}
          value={selectedUserId}
          onSearch={searchUsers}
          onChange={(v) => {
            setSelectedUserId(v);
            setFilterModel('');
            setCurrentPage(1);
          }}
          onClear={() => {
            setSelectedUserId(undefined);
            syncItems([]);
            setCurrentPage(1);
          }}
        />
        {selectedUserId && (
          <Tag color='purple' size='large'>
            {selectedUserLabel} · {items.length} {t('个模型')}
          </Tag>
        )}
        {dirtyCount > 0 && (
          <Tag color='orange' size='large'>
            {t('未保存')} {dirtyCount} {t('条')}
          </Tag>
        )}
      </div>

      {pricingUsers.length > 0 && (
        <div className='flex flex-wrap gap-2 mb-3'>
          {pricingUsers.map((u) => (
            <Tag
              key={u.user_id}
              color={selectedUserId === u.user_id ? 'blue' : 'white'}
              style={{ cursor: 'pointer' }}
              onClick={() => {
                setSelectedUserId(u.user_id);
                setFilterModel('');
                setCurrentPage(1);
              }}
            >
              {u.username || `#${u.user_id}`}（{u.model_count}）
            </Tag>
          ))}
        </div>
      )}

      {!selectedUserId ? (
        <Banner
          type='warning'
          closeIcon={null}
          className='!rounded-lg'
          description={t(
            '请先选择要管理的用户。可从上方下拉选择，或点击已有配置用户标签快速切换。',
          )}
        />
      ) : (
        <>
          <div className='flex items-center justify-between mb-3'>
            <Input
              style={{ width: 280 }}
              placeholder={t('按模型名筛选')}
              value={filterModel}
              onChange={handleFilterChange}
              showClear
            />
            <Space wrap>
              <Button
                theme='solid'
                type='secondary'
                disabled={dirtyCount === 0}
                loading={savingAll}
                onClick={doSaveAllDirty}
              >
                {t('保存全部修改')}
                {dirtyCount > 0 ? ` (${dirtyCount})` : ''}
              </Button>
              <Button onClick={() => refreshAll({ resetPage: false })}>
                {t('刷新')}
              </Button>
              <Button
                disabled={!items.length}
                onClick={openExport}
              >
                {t('导出')}
              </Button>
              <Button onClick={openImport}>{t('一键导入当前折扣')}</Button>
              <Button
                type='secondary'
                disabled={!items.length}
                onClick={openConvertChannelList}
              >
                {t('切换为渠道清单')}
              </Button>
              <Button theme='solid' type='primary' onClick={openAdd}>
                {t('新增模型指定价')}
              </Button>
              <Popconfirm
                title={t('确认清空该用户全部指定价？')}
                content={t('删除后该用户恢复所有模型的默认渠道定价与选路')}
                onConfirm={doClearUser}
              >
                <Button type='danger'>{t('清空该用户')}</Button>
              </Popconfirm>
            </Space>
          </div>
          <Table
            columns={columns}
            dataSource={filteredItems}
            loading={loading}
            rowKey='id'
            scroll={{ x: 1400 }}
            pagination={{
              currentPage: safePage,
              pageSize: PAGE_SIZE,
              total: filteredItems.length,
              showSizeChanger: false,
              onPageChange: (page) => setCurrentPage(page),
            }}
            empty={t('该用户暂无指定价，可用「一键导入当前折扣」从最便宜渠道批量绑定')}
          />
        </>
      )}

      <Modal
        title={form.id ? t('编辑模型指定价') : t('新增模型指定价')}
        visible={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={doSave}
        okText={t('保存')}
        cancelText={t('取消')}
        confirmLoading={saving}
        width={760}
      >
        <div className='flex flex-col gap-3'>
          <div>
            <Text strong>{t('用户')}</Text>
            <div className='mt-1'>
              <Tag color='purple' size='large'>
                {selectedUserLabel || `#${form.user_id}`}
              </Tag>
            </div>
          </div>
          <div>
            <Text strong>{t('模型')}</Text>
            <Select
              style={{ width: '100%' }}
              filter
              allowCreate
              placeholder={t('选择或输入模型名')}
              optionList={modelOptions.map((m) => ({ value: m, label: m }))}
              value={form.model_name || undefined}
              disabled={!!form.id}
              onChange={(v) => {
                setField('model_name', v);
                setField('channels', []);
                if (v) {
                  loadChannelOptions(v, form);
                } else {
                  setChannelOptions([]);
                }
              }}
            />
          </div>
          <div>
            <Text strong>{t('选路模式')}</Text>
            <div className='mt-1'>
              <RadioGroup
                type='button'
                value={form.mode || MODE_PRICE_CAP}
                onChange={(event) => {
                  const mode = event.target.value;
                  setField('mode', mode);
                  if (
                    mode === MODE_CHANNEL_LIST &&
                    form.model_name &&
                    channelOptions.length === 0
                  ) {
                    loadChannelOptions(form.model_name, form);
                  }
                }}
              >
                <Radio value={MODE_PRICE_CAP}>{t('价格上限')}</Radio>
                <Radio value={MODE_CHANNEL_LIST}>{t('渠道清单')}</Radio>
              </RadioGroup>
            </div>
            <Text type='tertiary' size='small'>
              {form.mode === MODE_CHANNEL_LIST
                ? t(
                    '手动勾选可用渠道并排序；未勾选不可见也不可调用。智能路由优先于手动排序。',
                  )
                : t('自动放行有效单价不超过「全局官方价 × 总折扣」的渠道。')}
            </Text>
          </div>
          <div className='grid grid-cols-3 gap-3'>
            <div>
              <Text strong>{t('成本折扣')} (%)</Text>
              <InputNumber
                style={{ width: '100%' }}
                min={0}
                max={1000}
                value={form.price_discount_percent}
                onChange={(v) => setField('price_discount_percent', v)}
              />
            </div>
            <div>
              <Text strong>{t('经营成本')} (%)</Text>
              <InputNumber
                style={{ width: '100%' }}
                min={0}
                max={1000}
                value={form.operating_cost_percent}
                onChange={(v) => setField('operating_cost_percent', v)}
              />
            </div>
            <div>
              <Text strong>{t('加价折扣')} (%)</Text>
              <InputNumber
                style={{ width: '100%' }}
                min={0}
                max={1000}
                value={form.markup_discount_rate}
                onChange={(v) => setField('markup_discount_rate', v)}
              />
            </div>
          </div>
          <div className='flex items-center justify-between'>
            <Space>
              <Text strong>{t('总折扣')}:</Text>
              <Tag color='green' size='large'>
                {Math.round(totalPercent * 100) / 100}%
              </Tag>
            </Space>
            <Space>
              <Text strong>{t('启用')}</Text>
              <Switch
                checked={!!form.enabled}
                onChange={(v) => setField('enabled', v)}
              />
            </Space>
          </div>

          {form.mode === MODE_CHANNEL_LIST && (
            <div className='flex flex-col gap-2'>
              <div className='flex items-center justify-between'>
                <Text strong>
                  {t('勾选渠道并排序')}（
                  {normalizeChannels(form.channels).length}）
                </Text>
                <Space>
                  <Button
                    size='small'
                    loading={channelOptionsLoading}
                    onClick={() =>
                      form.model_name && loadChannelOptions(form.model_name, form)
                    }
                  >
                    {t('刷新渠道')}
                  </Button>
                  <Button size='small' onClick={selectAllWithinCapChannels}>
                    {t('勾选未超价渠道')}
                  </Button>
                </Space>
              </div>
              <Banner
                type='warning'
                closeIcon={null}
                className='!rounded-lg'
                description={t(
                  '单价高于指定售价的渠道可勾选但平台可能亏损。排序靠前优先；若用户走智能路由则在勾选集内按智能规则选渠。',
                )}
              />
              <Table
                size='small'
                loading={channelOptionsLoading}
                columns={[
                  {
                    title: t('启用'),
                    width: 70,
                    render: (_, ch) => (
                      <Checkbox
                        checked={normalizeChannels(form.channels).some(
                          (c) => c.channel_id === ch.channel_id,
                        )}
                        onChange={(e) =>
                          toggleFormChannel(ch.channel_id, !!e?.target?.checked)
                        }
                      />
                    ),
                  },
                  {
                    title: t('优先级'),
                    width: 100,
                    render: (_, ch) => {
                      const list = normalizeChannels(form.channels);
                      const idx = list.findIndex(
                        (c) => c.channel_id === ch.channel_id,
                      );
                      if (idx < 0) return '-';
                      return (
                        <Space>
                          <Tag color='blue'>{idx + 1}</Tag>
                          <Button
                            size='small'
                            disabled={idx === 0}
                            onClick={() => moveFormChannel(ch.channel_id, -1)}
                          >
                            ↑
                          </Button>
                          <Button
                            size='small'
                            disabled={idx === list.length - 1}
                            onClick={() => moveFormChannel(ch.channel_id, 1)}
                          >
                            ↓
                          </Button>
                        </Space>
                      );
                    },
                  },
                  { title: t('渠道'), dataIndex: 'channel_name' },
                  { title: 'ID', dataIndex: 'channel_id', width: 70 },
                  {
                    title: t('有效单价'),
                    dataIndex: 'unit_price',
                    width: 110,
                    render: (v) => Math.round(v * 1e6) / 1e6,
                  },
                  {
                    title: t('相对指定售价'),
                    dataIndex: 'within_cap',
                    width: 120,
                    render: (v) =>
                      v ? (
                        <Tag color='green'>{t('未超价')}</Tag>
                      ) : (
                        <Tag color='orange'>{t('高于售价')}</Tag>
                      ),
                  },
                ]}
                dataSource={[...(channelOptions || [])].sort((a, b) => {
                  const pa = normalizeChannels(form.channels).findIndex(
                    (c) => c.channel_id === a.channel_id,
                  );
                  const pb = normalizeChannels(form.channels).findIndex(
                    (c) => c.channel_id === b.channel_id,
                  );
                  const ra = pa < 0 ? 9999 : pa;
                  const rb = pb < 0 ? 9999 : pb;
                  if (ra !== rb) return ra - rb;
                  return (a.channel_id || 0) - (b.channel_id || 0);
                })}
                rowKey='channel_id'
                pagination={{ pageSize: 6 }}
                empty={
                  form.model_name
                    ? t('该模型暂无启用渠道')
                    : t('请先选择模型')
                }
              />
            </div>
          )}

          <div>
            <Button
              loading={previewLoading}
              onClick={() => doPreview(form)}
              block
            >
              {form.mode === MODE_CHANNEL_LIST
                ? t('预览勾选渠道')
                : t('预览可用渠道（价格上限校验）')}
            </Button>
          </div>
        </div>
      </Modal>

      <Modal
        title={`${t('预览可用渠道')}${previewTitle ? ` · ${previewTitle}` : ''}`}
        visible={previewVisible}
        onCancel={() => {
          setPreviewVisible(false);
          setPreview(null);
        }}
        footer={
          <Button onClick={() => setPreviewVisible(false)}>{t('关闭')}</Button>
        }
        width={720}
      >
        {previewLoading && !preview ? (
          <Text type='tertiary'>{t('加载中')}...</Text>
        ) : preview ? (
          <div>
            <Banner
              type={
                preview.mode === 'channel_list'
                  ? preview.selected_count > 0
                    ? 'success'
                    : 'warning'
                  : !preview.cap_defined
                    ? 'warning'
                    : preview.within_count > 0
                      ? 'success'
                      : 'danger'
              }
              closeIcon={null}
              className='!rounded-lg'
              description={
                preview.mode === 'channel_list'
                  ? `${t('渠道清单模式')}：${t('已勾选')} ${preview.selected_count || 0}/${preview.total_channels} ${t('个渠道（用户端仅可见勾选渠道）')}`
                  : !preview.cap_defined
                    ? t(
                        '该模型未配置全局官方价，指定价上限无法计算：计费将回退渠道基价，选路不做价格限制。建议先在「价格设置」中配置全局价。',
                      )
                    : `${t('价格上限')} ${Math.round(preview.cap * 1e6) / 1e6}，${preview.within_count}/${preview.total_channels} ${t('个渠道在上限内可被调用')}`
              }
            />
            {(preview.channels || []).length > 0 && (
              <Table
                size='small'
                className='mt-2'
                columns={[
                  { title: t('渠道'), dataIndex: 'channel_name' },
                  { title: 'ID', dataIndex: 'channel_id', width: 80 },
                  {
                    title: t('有效单价'),
                    dataIndex: 'unit_price',
                    render: (v) => Math.round(v * 1e6) / 1e6,
                  },
                  ...(preview.mode === 'channel_list'
                    ? [
                        {
                          title: t('勾选'),
                          dataIndex: 'selected',
                          render: (v, row) =>
                            v ? (
                              <Tag color='blue'>
                                #{row.priority || '-'}
                              </Tag>
                            ) : (
                              <Tag color='grey'>{t('未勾选')}</Tag>
                            ),
                        },
                        {
                          title: t('相对指定售价'),
                          dataIndex: 'within_cap',
                          render: (v) =>
                            v ? (
                              <Tag color='green'>{t('未超价')}</Tag>
                            ) : (
                              <Tag color='orange'>{t('高于售价')}</Tag>
                            ),
                        },
                      ]
                    : [
                        {
                          title: t('可调用'),
                          dataIndex: 'within_cap',
                          render: (v) =>
                            v ? (
                              <Tag color='green'>{t('是')}</Tag>
                            ) : (
                              <Tag color='red'>{t('超价排除')}</Tag>
                            ),
                        },
                      ]),
                ]}
                dataSource={preview.channels}
                rowKey='channel_id'
                pagination={{ pageSize: 5 }}
              />
            )}
          </div>
        ) : (
          <Text type='tertiary'>{t('暂无预览数据')}</Text>
        )}
      </Modal>

      <Modal
        title={t('切换为渠道清单')}
        visible={convertVisible}
        onCancel={() => setConvertVisible(false)}
        onOk={doConvertToChannelList}
        okText={
          convertModelNames.length
            ? `${t('切换所选')}（${convertModelNames.length}）`
            : `${t('切换全部')}（${items.length}）`
        }
        cancelText={t('取消')}
        confirmLoading={convertingMode}
        width={640}
      >
        <div className='flex flex-col gap-3'>
          <Banner
            type='info'
            closeIcon={null}
            className='!rounded-lg'
            description={t(
              '将指定价改为渠道清单：每个模型勾选当前未超指定售价的渠道，并按单价从低到高排序。已是渠道清单的也会按此规则重写勾选；无可用渠道的模型会跳过。',
            )}
          />
          <div>
            <div className='flex items-center justify-between mb-2'>
              <Text strong>{t('选择模型')}</Text>
              <Space>
                <Button
                  size='small'
                  onClick={() =>
                    setConvertModelNames(items.map((r) => r.model_name))
                  }
                >
                  {t('全选')}
                </Button>
                <Button size='small' onClick={() => setConvertModelNames([])}>
                  {t('清空（默认全切）')}
                </Button>
              </Space>
            </div>
            <Select
              multiple
              filter
              showClear
              style={{ width: '100%' }}
              placeholder={t('不选则默认切换该用户全部模型')}
              optionList={convertModelOptions}
              value={convertModelNames}
              onChange={(v) => setConvertModelNames(v || [])}
              maxTagCount={4}
            />
            <div className='mt-2'>
              <Text type='tertiary' size='small'>
                {convertModelNames.length
                  ? `${t('将切换所选')} ${convertModelNames.length} ${t('个模型')}`
                  : `${t('未选择模型，将切换全部')} ${items.length} ${t('个模型')}`}
                {filterModel
                  ? `（${t('列表筛选不影响范围，以勾选为准')}）`
                  : ''}
              </Text>
            </div>
          </div>
          <Banner
            type='warning'
            closeIcon={null}
            className='!rounded-lg'
            description={`${t('确认后将对')} ${selectedUserLabel} ${t('的')} ${convertTargetCount} ${t('个模型执行切换，三折扣保持不变。')}`}
          />
        </div>
      </Modal>

      <Modal
        title={t('导出用户指定价')}
        visible={exportVisible}
        onCancel={() => setExportVisible(false)}
        onOk={doExport}
        okText={t('确认导出')}
        cancelText={t('取消')}
        confirmLoading={exporting}
        okButtonProps={{ disabled: !exportFields.length }}
        width={560}
      >
        <div className='flex flex-col gap-3'>
          <Banner
            type='info'
            closeIcon={null}
            className='!rounded-lg'
            description={`${t('将导出')} ${selectedUserLabel} ${t('的模型折扣配置为 Excel。可勾选需要的列；未勾选的字段不会出现在文件中。')}`}
          />
          <div>
            <div className='flex items-center justify-between mb-2'>
              <Text strong>{t('导出字段')}</Text>
              <Space>
                <Button
                  size='small'
                  onClick={() =>
                    setExportFields(EXPORT_FIELD_OPTIONS.map((o) => o.value))
                  }
                >
                  {t('全选')}
                </Button>
                <Button
                  size='small'
                  onClick={() => setExportFields(defaultExportFields())}
                >
                  {t('常用')}
                </Button>
                <Button size='small' onClick={() => setExportFields([])}>
                  {t('清空')}
                </Button>
              </Space>
            </div>
            <Checkbox.Group
              value={exportFields}
              onChange={setExportFields}
              direction='horizontal'
              className='flex flex-wrap gap-x-4 gap-y-2'
            >
              {EXPORT_FIELD_OPTIONS.map((opt) => (
                <Checkbox key={opt.value} value={opt.value}>
                  {t(opt.labelKey)}
                </Checkbox>
              ))}
            </Checkbox.Group>
          </div>
          {!!filterModel && (
            <div className='flex items-center justify-between'>
              <div>
                <Text strong>{t('仅导出当前筛选结果')}</Text>
                <div>
                  <Text type='tertiary' size='small'>
                    {t('当前模型筛选')}：{filterModel}（
                    {filteredItems.length}/{items.length}）
                  </Text>
                </div>
              </div>
              <Switch
                checked={exportOnlyFiltered}
                onChange={setExportOnlyFiltered}
              />
            </div>
          )}
        </div>
      </Modal>

      <Modal
        title={t('一键导入当前渠道折扣')}
        visible={importVisible}
        onCancel={() => setImportVisible(false)}
        onOk={doImport}
        okText={t('确认导入')}
        cancelText={t('取消')}
        confirmLoading={importing}
        okButtonProps={{ disabled: importPreviewLoading || !importPreview.length }}
        width={760}
      >
        <div className='flex flex-col gap-3'>
          <Banner
            type='info'
            closeIcon={null}
            className='!rounded-lg'
            description={`${t('将为')} ${selectedUserLabel} ${t('按模型导入当前最便宜启用渠道的三项折扣；已存在配置会被覆盖为对应渠道当前值。')}`}
          />
          <div className='flex items-center justify-between'>
            <Text>
              {t('可导入模型')}：{importPreview.length}
            </Text>
            <Space>
              <Text strong>{t('启用')}</Text>
              <Switch
                checked={!!importForm.enabled}
                onChange={(v) => setImportField('enabled', v)}
              />
            </Space>
          </div>
          <Table
            size='small'
            loading={importPreviewLoading}
            columns={[
              { title: t('模型'), dataIndex: 'model_name' },
              {
                title: t('来源渠道'),
                render: (_, row) => `${row.channel_name} #${row.channel_id}`,
              },
              {
                title: t('成本折扣'),
                dataIndex: 'price_discount_percent',
                render: (v) => `${v}%`,
              },
              {
                title: t('经营成本'),
                dataIndex: 'operating_cost_percent',
                render: (v) => `${v}%`,
              },
              {
                title: t('加价折扣'),
                dataIndex: 'markup_discount_rate',
                render: (v) => `${v}%`,
              },
              {
                title: t('总折扣'),
                dataIndex: 'total_percent',
                render: (v) => (
                  <Tag color='green'>{Math.round(v * 100) / 100}%</Tag>
                ),
              },
            ]}
            dataSource={importPreview}
            rowKey='model_name'
            pagination={{ pageSize: 8 }}
            empty={t('没有可导入的已定价模型')}
          />
        </div>
      </Modal>
    </div>
  );
}
