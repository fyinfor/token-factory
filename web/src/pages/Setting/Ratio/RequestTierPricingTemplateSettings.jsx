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
  Button,
  Card,
  Form,
  Modal,
  Select,
  Space,
  Table,
  Typography,
} from '@douyinfe/semi-ui';
import {
  IconDelete,
  IconEdit,
  IconPlus,
  IconSave,
} from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../../helpers';
import {
  emptyTierTemplate,
  normalizeTierPricing,
  normalizeTierRule,
  parseJSONMap,
  serializeTierRule,
  summarizeTierPricing,
  validateTierPricing,
  CURRENCY_OPTIONS,
  getCurrencySymbol,
} from './utils/requestTierPricing';
import TierRowsEditor from './components/TierRowsEditor';

const { Text } = Typography;

// ============================================================
// RequestTierPricingTemplateSettings — 阶梯计费模板管理（v2 重构版）
// ============================================================
// 新模型：仅以输入Token区间划分档位，每个档位绑定4项价格。
// 顶部新增货币汇率选择器，统一表格编辑。
// ============================================================

/**
 * 生成唯一模板 ID
 */
const createTemplateId = (templates) => {
  const base = `tpl_${Date.now()}`;
  let id = base;
  let suffix = 1;
  while (Object.prototype.hasOwnProperty.call(templates, id)) {
    suffix += 1;
    id = `${base}_${suffix}`;
  }
  return id;
};

/**
 * 将旧格式模板（4 种类别）转换为新格式
 */
const migrateOldTemplate = (template) => {
  const normalized = normalizeTierRule(template);
  // 旧模板没有 currency，默认 USD
  if (!normalized.currency || !['USD', 'CNY', 'CUSTOM'].includes(normalized.currency)) {
    normalized.currency = 'USD';
  }
  return {
    name: template?.name || '',
    ...normalized,
  };
};

export default function RequestTierPricingTemplateSettings({
  options,
  refresh,
}) {
  const { t } = useTranslation();
  const [templates, setTemplates] = useState({});
  const [editing, setEditing] = useState(null);
  const [loading, setLoading] = useState(false);
  const [localCurrency, setLocalCurrency] = useState('USD');

  // 从 options 加载模板，兼容旧格式
  useEffect(() => {
    const raw = parseJSONMap(options.RequestTierPricingTemplates);
    const migrated = {};
    for (const [id, tpl] of Object.entries(raw)) {
      migrated[id] = migrateOldTemplate(tpl);
    }
    setTemplates(migrated);
  }, [options.RequestTierPricingTemplates]);

  // 模板列表数据
  const data = useMemo(
    () =>
      Object.entries(templates).map(([id, tpl]) => ({
        id,
        ...tpl,
        // 保留原始 tiers 用于表格展示
        _tierCount: tpl.tiers?.length || 0,
        _currency: tpl.currency || 'USD',
      })),
    [templates],
  );

  /** 持久化模板到后端 */
  const save = async (nextTemplates) => {
    setLoading(true);
    try {
      const res = await API.put('/api/option/', {
        key: 'RequestTierPricingTemplates',
        value: JSON.stringify(nextTemplates, null, 2),
      });
      if (!res?.data?.success)
        throw new Error(res?.data?.message || t('保存失败'));
      showSuccess(t('保存成功'));
      await refresh();
    } catch (error) {
      showError(error?.message || t('保存失败'));
    } finally {
      setLoading(false);
    }
  };

  /** 提交编辑 */
  const handleSubmit = async () => {
    if (!editing) return;

    const error = validateTierPricing(editing, t);
    if (error) {
      showError(error);
      return;
    }

    const template = {
      name: editing.name || t('未命名模板'),
      mode: editing.mode || 'progressive',
      currency: editing.currency || 'USD',
      tiers: editing.tiers || [],
    };

    const id = String(editing._id || '').trim();
    setEditing(null);

    if (id) {
      const next = { ...templates, [id]: template };
      setTemplates(next);
      await save(next);
      return;
    }

    const next = { ...templates, [createTemplateId(templates)]: template };
    setTemplates(next);
    await save(next);
  };

  /** 删除模板 */
  const removeTemplate = async (id) => {
    const next = { ...templates };
    delete next[id];
    setTemplates(next);
    await save(next);
  };

  /** 新增模板 */
  const beginCreate = () => {
    setLocalCurrency('USD');
    setEditing({
      ...emptyTierTemplate(),
      _id: '',
    });
  };

  /** 编辑已有模板 */
  const beginEdit = (row) => {
    setLocalCurrency(row.currency || row._currency || 'USD');
    setEditing({
      ...row,
      _id: row.id,
      currency: row.currency || row._currency || 'USD',
      tiers: Array.isArray(row.tiers) ? [...row.tiers] : [],
      name: row.name || '',
    });
  };

  /** 更新编辑中的 tiers */
  const updateTiers = (nextTiers) => {
    setEditing((prev) => (prev ? { ...prev, tiers: nextTiers } : prev));
  };

  /** 货币切换 */
  const handleCurrencyChange = (value) => {
    setLocalCurrency(value);
    setEditing((prev) => (prev ? { ...prev, currency: value } : prev));
  };

  const currencyOptionList = CURRENCY_OPTIONS.map((c) => ({
    label: c.label,
    value: c.key,
  }));

  return (
    <Card>
      <Space vertical align='start' style={{ width: '100%' }}>
        {/* 操作栏 */}
        <Space>
          <Button icon={<IconPlus />} onClick={beginCreate}>
            {t('添加模板')}
          </Button>
          <Button
            icon={<IconSave />}
            loading={loading}
            onClick={() => save(templates)}
          >
            {t('保存模板')}
          </Button>
        </Space>

        <Text type='secondary'>
          {t('模板仅用于前端快速套用，模型保存和主站同步都会写入完整阶梯规则。')}
        </Text>

        {/* 模板列表 */}
        <Table
          dataSource={data}
          rowKey='id'
          pagination={false}
          columns={[
            { title: t('模板 ID'), dataIndex: 'id', width: 200 },
            { title: t('模板名称'), dataIndex: 'name', width: 160 },
            {
              title: t('货币'),
              width: 100,
              render: (_, row) =>
                getCurrencySymbol(row._currency || 'USD'),
            },
            {
              title: t('档位'),
              width: 80,
              render: (_, row) => `${row._tierCount || 0}${t('档')}`,
            },
            {
              title: t('规则摘要'),
              render: (_, row) => summarizeTierPricing(row, t),
            },
            {
              title: t('操作'),
              width: 120,
              render: (_, row) => (
                <Space>
                  <Button
                    size='small'
                    icon={<IconEdit />}
                    onClick={() => beginEdit(row)}
                  />
                  <Button
                    size='small'
                    type='danger'
                    icon={<IconDelete />}
                    onClick={() => removeTemplate(row.id)}
                  />
                </Space>
              ),
            },
          ]}
        />
      </Space>

      {/* 编辑弹窗 */}
      <Modal
        title={t('编辑阶梯计费模板')}
        visible={Boolean(editing)}
        onCancel={() => setEditing(null)}
        onOk={handleSubmit}
        size='large'
      >
        {editing ? (
          <Form labelPosition='left'>
            {/* 模板名称 */}
            <Form.Input
              label={t('模板名称')}
              field='name'
              initValue={editing.name}
              onChange={(v) =>
                setEditing((prev) => (prev ? { ...prev, name: v } : prev))
              }
            />

            {/* 货币汇率选择器 */}
            <Form.Slot label={t('货币单位')}>
              <Select
                value={localCurrency}
                optionList={currencyOptionList}
                onChange={handleCurrencyChange}
                style={{ width: 200 }}
              />
            </Form.Slot>

            <div className='my-3 text-xs text-gray-500'>
              {t('阶梯区间从 0 开始；最后一档固定为无限且不能删除。每个档位统一配置 4 项价格。')}
            </div>

            {/* 统一档位编辑器 */}
            <Card
              style={{
                width: '100%',
                background: 'var(--semi-color-fill-0)',
              }}
            >
              <TierRowsEditor
                t={t}
                value={editing.tiers || []}
                onChange={updateTiers}
                currency={localCurrency}
              />
            </Card>
          </Form>
        ) : null}
      </Modal>
    </Card>
  );
}
