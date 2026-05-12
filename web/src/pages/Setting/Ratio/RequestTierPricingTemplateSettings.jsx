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
  Input,
  Modal,
  Space,
  Switch,
  Table,
  Typography,
} from '@douyinfe/semi-ui';
import { IconDelete, IconEdit, IconPlus, IconSave, IconSetting } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../../helpers';
import {
  emptyTierRule,
  hasTierRule,
  normalizeTierRule,
  parseJSONMap,
  serializeTierRule,
  summarizeTierRule,
  validateTierRule,
} from './utils/requestTierPricing';
import TierRowsEditor from './components/TierRowsEditor';

const { Text } = Typography;
const DEFAULT_TIER_SEGMENTS = {
  segments: [{ up_to: 0, ratio: 0 }],
};

const createEmptyTemplate = () => ({
  name: '',
  ...emptyTierRule(),
});

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

export default function RequestTierPricingTemplateSettings({
  options,
  refresh,
}) {
  const { t } = useTranslation();
  const [templates, setTemplates] = useState({});
  const [editing, setEditing] = useState(null);
  const [loading, setLoading] = useState(false);
  const [visibleCategories, setVisibleCategories] = useState({
    output: false,
    cache_read: false,
    cache_write: false,
  });

  // 获取汇率
  const exchangeRate = options?.usd_exchange_rate || 1;

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

  const handleSubmit = async () => {
    const id = String(editing?.id || '').trim();
    // 过滤掉开关未打开的类别（除了 input，input 始终保留）
    const filteredEditing = { ...editing };
    if (!visibleCategories.output) delete filteredEditing.output;
    if (!visibleCategories.cache_read) delete filteredEditing.cache_read;
    if (!visibleCategories.cache_write) delete filteredEditing.cache_write;
    const rule = serializeTierRule(filteredEditing);
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
    const next = {
      ...templates,
      [createTemplateId(templates)]: template,
    };
    setTemplates(next);
    await save(next);
  };

  const removeTemplate = async (id) => {
    const next = { ...templates };
    delete next[id];
    setTemplates(next);
    await save(next);
  };

  const beginCreate = () => {
    setVisibleCategories({
      output: false,
      cache_read: false,
      cache_write: false,
    });
    setEditing(createEmptyTemplate());
  };

  const beginEdit = (row) => {
    const rule = normalizeTierRule(row);
    setVisibleCategories({
      output: rule.output.length > 0,
      cache_read: rule.cache_read.length > 0,
      cache_write: rule.cache_write.length > 0,
    });
    setEditing({
      ...row,
      ...rule,
    });
  };

  const updateEditingTier = (key, value) => {
    setEditing((previous) => ({
      ...previous,
      [key]: value?.segments || [],
    }));
  };

  return (
    <Card>
      <Space vertical align='start' style={{ width: '100%' }}>
        <Space>
          <Button
            icon={<IconPlus />}
            onClick={beginCreate}
          >
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
          {t(
            '模板仅用于前端快速套用，模型保存和主站同步都会写入完整阶梯规则。',
          )}
        </Text>
        <Table
          dataSource={data}
          rowKey='id'
          pagination={false}
          columns={[
            { title: t('模板 ID'), dataIndex: 'id' },
            { title: t('模板名称'), dataIndex: 'name' },
            {
              title: t('规则摘要'),
              render: (_, row) => summarizeTierRule(row, t),
            },
            {
              title: t('操作'),
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
      <Modal
        title={t('编辑阶梯计费模板')}
        visible={Boolean(editing)}
        onCancel={() => setEditing(null)}
        onOk={handleSubmit}
        size='large'
      >
        {editing ? (
          <Form labelPosition='left'>
            <Form.Input
              label={t('模板名称')}
              field='name'
              initValue={editing.name}
              onChange={(v) => setEditing({ ...editing, name: v })}
            />
            <Card
              title={<span>{t('输入价格')}</span>}
              style={{ width: '100%', marginBottom: 8, background: 'var(--semi-color-fill-0)' }}
            >
              <TierRowsEditor
                t={t}
                value={{ segments: editing.input || DEFAULT_TIER_SEGMENTS.segments }}
                onChange={(value) => updateEditingTier('input', value)}
                exchangeRate={exchangeRate}
                tierType='model'
              />
            </Card>
            <Card
              title={
                <div className='flex justify-between items-center'>
                  <span>{t('输出价格')}</span>
                  <Switch
                    size='small'
                    checked={visibleCategories.output}
                    onChange={(checked) =>
                      setVisibleCategories((previous) => ({
                        ...previous,
                        output: checked,
                      }))
                    }
                  />
                </div>
              }
              style={{ width: '100%', marginBottom: 8, background: 'var(--semi-color-fill-0)' }}
            >
              {visibleCategories.output ? (
                <TierRowsEditor
                  t={t}
                  value={{ segments: editing.output || DEFAULT_TIER_SEGMENTS.segments }}
                  onChange={(value) => updateEditingTier('output', value)}
                  exchangeRate={exchangeRate}
                  tierType='completion'
                />
              ) : null}
            </Card>
            <Card
              title={
                <div className='flex justify-between items-center'>
                  <span>{t('缓存读取价格')}</span>
                  <Switch
                    size='small'
                    checked={visibleCategories.cache_read}
                    onChange={(checked) =>
                      setVisibleCategories((previous) => ({
                        ...previous,
                        cache_read: checked,
                      }))
                    }
                  />
                </div>
              }
              style={{ width: '100%', marginBottom: 8, background: 'var(--semi-color-fill-0)' }}
            >
              {visibleCategories.cache_read ? (
                <TierRowsEditor
                  t={t}
                  value={{ segments: editing.cache_read || DEFAULT_TIER_SEGMENTS.segments }}
                  onChange={(value) => updateEditingTier('cache_read', value)}
                  exchangeRate={exchangeRate}
                  tierType='cache'
                />
              ) : null}
            </Card>
            <Card
              title={
                <div className='flex justify-between items-center'>
                  <span>{t('缓存写入价格')}</span>
                  <Switch
                    size='small'
                    checked={visibleCategories.cache_write}
                    onChange={(checked) =>
                      setVisibleCategories((previous) => ({
                        ...previous,
                        cache_write: checked,
                      }))
                    }
                  />
                </div>
              }
              style={{ width: '100%', marginBottom: 8, background: 'var(--semi-color-fill-0)' }}
            >
              {visibleCategories.cache_write ? (
                <TierRowsEditor
                  t={t}
                  value={{ segments: editing.cache_write || DEFAULT_TIER_SEGMENTS.segments }}
                  onChange={(value) => updateEditingTier('cache_write', value)}
                  exchangeRate={exchangeRate}
                  tierType='createCache'
                />
              ) : null}
            </Card>
          </Form>
        ) : null}
      </Modal>
    </Card>
  );
}
