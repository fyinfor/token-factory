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
  Input,
  InputNumber,
  Modal,
  Popconfirm,
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

const emptyForm = {
  id: 0,
  user_id: undefined,
  model_name: '',
  price_discount_percent: 100,
  operating_cost_percent: 0,
  markup_discount_rate: 0,
  enabled: true,
};

const emptyImportForm = {
  enabled: true,
};

/**
 * UserModelPricingSettings 用户指定价管理（按用户视角）：
 * 先选用户，再看/管该用户下的模型指定价；支持一键导入当前已定价模型并统一三项折扣。
 */
export default function UserModelPricingSettings() {
  const { t } = useTranslation();
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(false);
  const [filterModel, setFilterModel] = useState('');

  const [pricingUsers, setPricingUsers] = useState([]);
  const [selectedUserId, setSelectedUserId] = useState(undefined);
  const [userOptions, setUserOptions] = useState([]);

  const [modalVisible, setModalVisible] = useState(false);
  const [saving, setSaving] = useState(false);
  const [form, setForm] = useState(emptyForm);
  const [editing, setEditing] = useState(false);
  const [modelOptions, setModelOptions] = useState([]);

  const [importVisible, setImportVisible] = useState(false);
  const [importForm, setImportForm] = useState(emptyImportForm);
  const [importing, setImporting] = useState(false);
  const [importPreview, setImportPreview] = useState([]);
  const [importPreviewLoading, setImportPreviewLoading] = useState(false);

  const [preview, setPreview] = useState(null);
  const [previewLoading, setPreviewLoading] = useState(false);

  const totalPercent = useMemo(
    () =>
      (Number(form.price_discount_percent) || 0) +
      (Number(form.operating_cost_percent) || 0) +
      (Number(form.markup_discount_rate) || 0),
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
      setItems([]);
      return;
    }
    setLoading(true);
    try {
      const res = await API.get(`/api/user_model_pricing/?user_id=${uid}`);
      const { success, message, data } = res.data;
      if (success) {
        setItems(data || []);
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
    if (selectedUserId) {
      loadList(selectedUserId);
    } else {
      setItems([]);
    }
  }, [selectedUserId]);

  const refreshAll = async () => {
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
    setEditing(false);
    setPreview(null);
    setModalVisible(true);
  };

  const openEdit = (row) => {
    setForm({
      id: row.id,
      user_id: row.user_id,
      model_name: row.model_name,
      price_discount_percent: row.price_discount_percent,
      operating_cost_percent: row.operating_cost_percent,
      markup_discount_rate: row.markup_discount_rate,
      enabled: row.enabled,
    });
    setEditing(true);
    setPreview(null);
    setModalVisible(true);
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

  const doPreview = async (f) => {
    const target = f || form;
    if (!target.model_name) {
      showError(t('请先选择模型'));
      return;
    }
    setPreviewLoading(true);
    try {
      const params = new URLSearchParams({
        model_name: target.model_name,
        price_discount_percent: String(target.price_discount_percent ?? 100),
        operating_cost_percent: String(target.operating_cost_percent ?? 0),
        markup_discount_rate: String(target.markup_discount_rate ?? 0),
      });
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

  const doSave = async () => {
    const uid = form.user_id || selectedUserId;
    if (!uid || !form.model_name) {
      showError(t('请选择用户和模型'));
      return;
    }
    setSaving(true);
    try {
      const res = await API.post('/api/user_model_pricing/', {
        user_id: uid,
        model_name: form.model_name,
        price_discount_percent: Number(form.price_discount_percent) || 0,
        operating_cost_percent: Number(form.operating_cost_percent) || 0,
        markup_discount_rate: Number(form.markup_discount_rate) || 0,
        enabled: !!form.enabled,
      });
      if (res.data.success) {
        showSuccess(t('保存成功'));
        setModalVisible(false);
        refreshAll();
      } else {
        showError(res.data.message);
      }
    } catch (e) {
      showError(e?.response?.data?.message || t('保存失败'));
    } finally {
      setSaving(false);
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
        refreshAll();
      } else {
        showError(res.data.message);
      }
    } catch (e) {
      showError(e?.response?.data?.message || t('导入失败'));
    } finally {
      setImporting(false);
    }
  };

  const doDelete = async (row) => {
    try {
      const res = await API.delete(`/api/user_model_pricing/${row.id}`);
      if (res.data.success) {
        showSuccess(t('删除成功'));
        refreshAll();
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
        refreshAll();
      } else {
        showError(res.data.message);
      }
    } catch (e) {
      showError(e?.response?.data?.message || t('清空失败'));
    }
  };

  const filteredItems = useMemo(() => {
    if (!filterModel) return items;
    const kw = filterModel.toLowerCase();
    return items.filter((it) =>
      (it.model_name || '').toLowerCase().includes(kw),
    );
  }, [items, filterModel]);

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

  const columns = [
    {
      title: t('模型'),
      dataIndex: 'model_name',
      render: (text) => <Tag color='blue'>{text}</Tag>,
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
        <Tag color='green' size='large'>
          {Math.round(v * 100) / 100}%
        </Tag>
      ),
    },
    {
      title: t('状态'),
      dataIndex: 'enabled',
      render: (v) =>
        v ? (
          <Tag color='green'>{t('启用')}</Tag>
        ) : (
          <Tag color='grey'>{t('禁用')}</Tag>
        ),
    },
    {
      title: t('更新时间'),
      dataIndex: 'updated_time',
      render: (v) => (v ? new Date(v * 1000).toLocaleString() : '-'),
    },
    {
      title: t('操作'),
      render: (_, row) => (
        <Space>
          <Button size='small' onClick={() => openEdit(row)}>
            {t('编辑')}
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
      ),
    },
  ];

  const setField = (key, value) => {
    setForm((prev) => ({ ...prev, [key]: value }));
  };

  const setImportField = (key, value) => {
    setImportForm((prev) => ({ ...prev, [key]: value }));
  };

  return (
    <div>
      <Banner
        type='info'
        closeIcon={null}
        className='!rounded-lg mb-3'
        description={t(
          '按用户管理指定价：先选择用户，再配置该用户下各模型的固定折扣。「一键导入」会为每个已定价模型，抄录当前最便宜启用渠道的三项折扣（不是统一手填覆盖）。计费按「全局官方价 × 总折扣」，超价渠道对该用户不可用。',
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
          }}
          onClear={() => {
            setSelectedUserId(undefined);
            setItems([]);
          }}
        />
        {selectedUserId && (
          <Tag color='purple' size='large'>
            {selectedUserLabel} · {items.length} {t('个模型')}
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
              onClick={() => setSelectedUserId(u.user_id)}
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
              onChange={setFilterModel}
              showClear
            />
            <Space wrap>
              <Button onClick={refreshAll}>{t('刷新')}</Button>
              <Button onClick={openImport}>{t('一键导入当前折扣')}</Button>
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
            pagination={{ pageSize: 20 }}
            empty={t('该用户暂无指定价，可用「一键导入当前折扣」从最便宜渠道批量绑定')}
          />
        </>
      )}

      <Modal
        title={editing ? t('编辑模型指定价') : t('新增模型指定价')}
        visible={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={doSave}
        okText={t('保存')}
        cancelText={t('取消')}
        confirmLoading={saving}
        width={640}
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
              onChange={(v) => setField('model_name', v)}
              disabled={editing}
            />
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
          <div>
            <Button loading={previewLoading} onClick={() => doPreview()} block>
              {t('预览可用渠道（价格上限校验）')}
            </Button>
          </div>
          {preview && (
            <div>
              <Banner
                type={
                  !preview.cap_defined
                    ? 'warning'
                    : preview.within_count > 0
                      ? 'success'
                      : 'danger'
                }
                closeIcon={null}
                className='!rounded-lg'
                description={
                  !preview.cap_defined
                    ? t(
                        '该模型未配置全局官方价，指定价上限无法计算：计费将回退渠道基价，选路不做价格限制。建议先在「价格设置」中配置全局价。',
                      )
                    : `${t('价格上限')} ${Math.round(preview.cap * 1e6) / 1e6}，${preview.within_count}/${preview.total_channels} ${t('个渠道在上限内可被调用')}`
                }
              />
              {preview.cap_defined && (preview.channels || []).length > 0 && (
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
                  ]}
                  dataSource={preview.channels}
                  rowKey='channel_id'
                  pagination={{ pageSize: 5 }}
                />
              )}
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
