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

/**
 * UserModelPricingSettings 用户指定价管理：
 * 对「用户 × 模型」单独覆盖 成本折扣/经营成本/加价折扣 三项，
 * 计费按「全局官方价 × 三折扣总和」，智能路由只允许单价 ≤ 上限的渠道。
 */
export default function UserModelPricingSettings() {
  const { t } = useTranslation();
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(false);
  const [filterModel, setFilterModel] = useState('');

  const [modalVisible, setModalVisible] = useState(false);
  const [saving, setSaving] = useState(false);
  const [form, setForm] = useState(emptyForm);
  const [editing, setEditing] = useState(false);

  const [userOptions, setUserOptions] = useState([]);
  const [modelOptions, setModelOptions] = useState([]);

  const [preview, setPreview] = useState(null);
  const [previewLoading, setPreviewLoading] = useState(false);

  const totalPercent = useMemo(
    () =>
      (Number(form.price_discount_percent) || 0) +
      (Number(form.operating_cost_percent) || 0) +
      (Number(form.markup_discount_rate) || 0),
    [form],
  );

  const loadList = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/user_model_pricing/');
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
        setUserOptions(
          users
            .filter((u) => !u.DeletedAt) // 搜索接口 Unscoped，剔除已注销用户
            .map((u) => ({
              value: u.id,
              label: `${u.username}${u.display_name ? ` (${u.display_name})` : ''} #${u.id}`,
            })),
        );
      }
    } catch (e) {
      // 静默失败，输入框可继续搜索
    }
  };

  useEffect(() => {
    loadList();
    loadModels();
  }, []);

  const openAdd = () => {
    setForm({ ...emptyForm });
    setEditing(false);
    setPreview(null);
    searchUsers('');
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
    setUserOptions([
      { value: row.user_id, label: `${row.username || ''} #${row.user_id}` },
    ]);
    setEditing(true);
    setPreview(null);
    setModalVisible(true);
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
    if (!form.user_id || !form.model_name) {
      showError(t('请选择用户和模型'));
      return;
    }
    setSaving(true);
    try {
      const res = await API.post('/api/user_model_pricing/', {
        user_id: form.user_id,
        model_name: form.model_name,
        price_discount_percent: Number(form.price_discount_percent) || 0,
        operating_cost_percent: Number(form.operating_cost_percent) || 0,
        markup_discount_rate: Number(form.markup_discount_rate) || 0,
        enabled: !!form.enabled,
      });
      if (res.data.success) {
        showSuccess(t('保存成功'));
        setModalVisible(false);
        loadList();
      } else {
        showError(res.data.message);
      }
    } catch (e) {
      showError(e?.response?.data?.message || t('保存失败'));
    } finally {
      setSaving(false);
    }
  };

  const doDelete = async (row) => {
    try {
      const res = await API.delete(`/api/user_model_pricing/${row.id}`);
      if (res.data.success) {
        showSuccess(t('删除成功'));
        loadList();
      } else {
        showError(res.data.message);
      }
    } catch (e) {
      showError(e?.response?.data?.message || t('删除失败'));
    }
  };

  const filteredItems = useMemo(() => {
    if (!filterModel) return items;
    const kw = filterModel.toLowerCase();
    return items.filter(
      (it) =>
        (it.model_name || '').toLowerCase().includes(kw) ||
        (it.username || '').toLowerCase().includes(kw) ||
        String(it.user_id) === filterModel.trim(),
    );
  }, [items, filterModel]);

  const columns = [
    {
      title: t('用户'),
      dataIndex: 'username',
      render: (text, row) => (
        <Space>
          <Text strong>{text || t('未知用户')}</Text>
          <Tag size='small' color='white'>
            #{row.user_id}
          </Tag>
        </Space>
      ),
    },
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
            content={t('删除后该用户恢复默认渠道定价与选路')}
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

  return (
    <div>
      <Banner
        type='info'
        closeIcon={null}
        className='!rounded-lg mb-3'
        description={t(
          '用户指定价：对指定用户调用指定模型时，按「全局官方价 × (成本折扣 + 经营成本 + 加价折扣)」计费（渠道无关、不叠加分组倍率）；智能路由仅允许调用有效单价不超过该上限的渠道，超价渠道对该用户不可用。',
        )}
      />
      <div className='flex items-center justify-between mb-3'>
        <Input
          style={{ width: 280 }}
          placeholder={t('按用户名 / 用户ID / 模型名筛选')}
          value={filterModel}
          onChange={setFilterModel}
          showClear
        />
        <Space>
          <Button onClick={loadList}>{t('刷新')}</Button>
          <Button theme='solid' type='primary' onClick={openAdd}>
            {t('新增指定价')}
          </Button>
        </Space>
      </div>
      <Table
        columns={columns}
        dataSource={filteredItems}
        loading={loading}
        rowKey='id'
        pagination={{ pageSize: 20 }}
        empty={t('暂无用户指定价配置')}
      />

      <Modal
        title={editing ? t('编辑用户指定价') : t('新增用户指定价')}
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
            <Select
              style={{ width: '100%' }}
              filter
              remote
              placeholder={t('选择用户，支持输入用户名 / ID 搜索')}
              optionList={userOptions}
              value={form.user_id}
              onSearch={searchUsers}
              onChange={(v) => setField('user_id', v)}
              disabled={editing}
            />
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
              <Text type='tertiary' size='small'>
                {t('= 用户最终价 / 全局官方价')}
              </Text>
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
            <Button
              loading={previewLoading}
              onClick={() => doPreview()}
              block
            >
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
    </div>
  );
}
