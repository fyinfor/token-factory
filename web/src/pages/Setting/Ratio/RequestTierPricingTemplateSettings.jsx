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
import { Button, Card, Form, Input, Modal, Space, Table, Typography } from '@douyinfe/semi-ui';
import { IconDelete, IconEdit, IconPlus, IconSave } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../../helpers';
import {
  TIER_CATEGORIES,
  ensureFinalInfinityTierRows,
  emptyTierRule,
  normalizeTierRule,
  parseJSONMap,
  serializeTierRule,
  summarizeTierRule,
  validateTierRule,
} from './utils/requestTierPricing';

const { Text } = Typography;

const createEmptyTemplate = () => ({
  name: '',
  ...emptyTierRule(),
});

function TierRowsEditor({ t, value, onChange }) {
  const rule = normalizeTierRule(value);
  const updateRows = (key, rows) => onChange({ ...rule, [key]: ensureFinalInfinityTierRows(rows) });
  const insertTopRow = (key, rows) => {
    const firstFinite = rows.find((row) => Number(row.up_to) > 0);
    const upTo = firstFinite ? Math.max(1, Math.floor(Number(firstFinite.up_to) / 2)) : 128000;
    const next = rows.length ? [{ up_to: upTo, ratio: 1 }, ...rows] : [{ up_to: 0, ratio: 1 }];
    updateRows(key, next);
  };
  return (
    <Space vertical align='start' style={{ width: '100%' }}>
      {TIER_CATEGORIES.map(({ key, label }) => {
        const rows = ensureFinalInfinityTierRows(rule[key] || []);
        return (
          <Card key={key} title={t(label)} bodyStyle={{ padding: 12 }} style={{ width: '100%' }}>
            <Table
              size='small'
              pagination={false}
              dataSource={rows.map((row, index) => ({ ...row, _idx: index }))}
              rowKey='_idx'
              columns={[
                {
                  title: t('区间 token'),
                  render: (_, row) => {
                    const previous = row._idx === 0 ? 0 : rows[row._idx - 1]?.up_to || 0;
                    const current = row.up_to || '∞';
                    return `${previous}～${current}`;
                  },
                },
                {
                  title: t('上限 token'),
                  dataIndex: 'up_to',
                  render: (_, row) => (
                    <Input
                      value={row._idx === rows.length - 1 ? '∞' : String(row.up_to ?? '')}
                      placeholder={t('最后一档固定无限')}
                      disabled={row._idx === rows.length - 1}
                      onChange={(v) => {
                        const next = [...rows];
                        next[row._idx] = { ...next[row._idx], up_to: v };
                        updateRows(key, next);
                      }}
                    />
                  ),
                },
                {
                  title: t('阶梯倍率'),
                  dataIndex: 'ratio',
                  render: (_, row) => (
                    <Input
                      value={String(row.ratio ?? '')}
                      placeholder='1'
                      onChange={(v) => {
                        const next = [...rows];
                        next[row._idx] = { ...next[row._idx], ratio: v };
                        updateRows(key, next);
                      }}
                    />
                  ),
                },
                {
                  title: t('操作'),
                  render: (_, row) => (
                    <Button
                      type='danger'
                      size='small'
                      icon={<IconDelete />}
                      disabled={row._idx === rows.length - 1}
                      onClick={() => updateRows(key, rows.filter((_, idx) => idx !== row._idx))}
                    />
                  ),
                },
              ]}
            />
            <Button
              className='mt-2'
              size='small'
              icon={<IconPlus />}
              onClick={() => insertTopRow(key, rows)}
            >
              {t('添加档位')}
            </Button>
          </Card>
        );
      })}
    </Space>
  );
}

export default function RequestTierPricingTemplateSettings({ options, refresh }) {
  const { t } = useTranslation();
  const [templates, setTemplates] = useState({});
  const [editing, setEditing] = useState(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setTemplates(parseJSONMap(options.RequestTierPricingTemplates));
  }, [options.RequestTierPricingTemplates]);

  const data = useMemo(
    () => Object.entries(templates).map(([id, tpl]) => ({ id, ...tpl })),
    [templates],
  );

  const save = async (nextTemplates) => {
    setLoading(true);
    try {
      const res = await API.put('/api/option/', {
        key: 'RequestTierPricingTemplates',
        value: JSON.stringify(nextTemplates, null, 2),
      });
      if (!res?.data?.success) throw new Error(res?.data?.message || t('保存失败'));
      showSuccess(t('保存成功'));
      await refresh();
    } catch (error) {
      showError(error?.message || t('保存失败'));
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = async () => {
    const id = String(editing?.id || '').trim();
    const rule = serializeTierRule(editing);
    const error = validateTierRule(rule, t);
    if (error) {
      showError(error);
      return;
    }
    const template = {
      name: editing.name || t('未命名模板'),
      ...rule,
    };
    setEditing(null);
    if (id) {
      const next = {
        ...templates,
        [id]: template,
      };
      setTemplates(next);
      await save(next);
      return;
    }
    await save([...Object.values(templates), template]);
  };

  const removeTemplate = async (id) => {
    const next = { ...templates };
    delete next[id];
    setTemplates(next);
    await save(next);
  };

  return (
    <Card>
      <Space vertical align='start' style={{ width: '100%' }}>
        <Space>
          <Button icon={<IconPlus />} onClick={() => setEditing(createEmptyTemplate())}>
            {t('添加模板')}
          </Button>
          <Button icon={<IconSave />} loading={loading} onClick={() => save(templates)}>
            {t('保存模板')}
          </Button>
        </Space>
        <Text type='secondary'>{t('模板仅用于前端快速套用，模型保存和主站同步都会写入完整阶梯规则。')}</Text>
        <Table
          dataSource={data}
          rowKey='id'
          pagination={false}
          columns={[
            { title: t('模板 ID'), dataIndex: 'id' },
            { title: t('模板名称'), dataIndex: 'name' },
            { title: t('规则摘要'), render: (_, row) => summarizeTierRule(row, t) },
            {
              title: t('操作'),
              render: (_, row) => (
                <Space>
                  <Button size='small' icon={<IconEdit />} onClick={() => setEditing(row)} />
                  <Button size='small' type='danger' icon={<IconDelete />} onClick={() => removeTemplate(row.id)} />
                </Space>
              ),
            },
          ]}
        />
      </Space>
      <Modal
        title={t('编辑阶梯计费模板')}
        visible={Boolean(editing)}
        onCancel={() => setEditing(null)}
        onOk={handleSubmit}
        size='large'
      >
        {editing ? (
          <Form labelPosition='left'>
            <Form.Input label={t('模板名称')} field='name' initValue={editing.name} onChange={(v) => setEditing({ ...editing, name: v })} />
            <TierRowsEditor t={t} value={editing} onChange={(v) => setEditing({ ...editing, ...v })} />
          </Form>
        ) : null}
      </Modal>
    </Card>
  );
}
