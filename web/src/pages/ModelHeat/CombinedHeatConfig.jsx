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

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Button,
  Card,
  Col,
  Input,
  InputNumber,
  Modal,
  Row,
  Select,
  Space,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { Clock, Flame, RefreshCw, Save, SlidersHorizontal } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { API, getLobeHubIcon, showError, showSuccess } from '@/helpers';
import {
  getChannelHeatKey,
  getTopHotChannels,
  HOT_OVERRIDE_AUTO,
  HOT_OVERRIDE_FORCE_HOT,
  HOT_OVERRIDE_FORCE_NOT_HOT,
} from '../../components/table/model-pricing/utils/modelHeat';

const { Text } = Typography;

const getRowKey = (channelId, modelName) => `${channelId}:${modelName}`;

const CombinedHeatConfig = () => {
  const { t } = useTranslation();

  const overrideOptions = [
    { value: HOT_OVERRIDE_AUTO, label: t('自动') },
    { value: HOT_OVERRIDE_FORCE_HOT, label: t('强制热门') },
    { value: HOT_OVERRIDE_FORCE_NOT_HOT, label: t('强制非热门') },
  ];
  const [loading, setLoading] = useState(false);
  const [rows, setRows] = useState([]);
  const [searchModel, setSearchModel] = useState('');
  const [searchChannel, setSearchChannel] = useState('');
  const [overrideFilter, setOverrideFilter] = useState('all');
  const [selectedRowKeys, setSelectedRowKeys] = useState([]);
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [settings, setSettings] = useState({ period: '7d', limit: 8 });
  const [pendingSettings, setPendingSettings] = useState({
    period: '7d',
    limit: 8,
  });
  const [settingsVisible, setSettingsVisible] = useState(false);
  const [batchVisible, setBatchVisible] = useState(false);
  const [batchMode, setBatchMode] = useState(HOT_OVERRIDE_AUTO);
  const [batchRank, setBatchRank] = useState(0);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const [pricingRes, overridesRes, settingsRes] = await Promise.all([
        API.get('/api/pricing'),
        API.get('/api/channel/hot-overrides'),
        API.get('/api/channel/hot-settings'),
      ]);
      if (!pricingRes.data.success) {
        throw new Error(pricingRes.data.message || t('加载失败'));
      }
      if (!overridesRes.data.success) {
        throw new Error(overridesRes.data.message || t('加载失败'));
      }
      if (!settingsRes.data.success) {
        throw new Error(settingsRes.data.message || t('加载失败'));
      }

      const overrideMap = new Map();
      (overridesRes.data.data || []).forEach((item) => {
        overrideMap.set(getRowKey(item.channel_id, item.model_name), item);
      });
      const vendorMap = new Map();
      (pricingRes.data.vendors || []).forEach((vendor) => {
        vendorMap.set(vendor.id, vendor);
      });

      const nextRows = [];
      (pricingRes.data.data || []).forEach((model) => {
        const channelList = Array.isArray(model.channel_list)
          ? model.channel_list
          : [];
        channelList.forEach((channel) => {
          const channelId = Number(channel.channel_id);
          if (!(channelId > 0)) return;
          const key = getRowKey(channelId, model.model_name);
          const override = overrideMap.get(key);
          nextRows.push({
            key,
            model_name: model.model_name,
            model_icon: model.icon || '',
            vendor: vendorMap.get(model.vendor_id),
            tags: model.tags || '',
            channel_id: channelId,
            channel_no: channel.channel_no || '',
            route_slug: channel.route_slug || '',
            supplier_alias: channel.supplier_alias || '',
            supplier_type: channel.supplier_type || '',
            company_logo_url: channel.company_logo_url || '',
            auto_req_count: Number(channel.auto_req_count || 0),
            override_mode: override?.override_mode || HOT_OVERRIDE_AUTO,
            manual_rank: Number(override?.manual_rank || 0),
            channel: {
              ...channel,
              hot_override: override?.override_mode || '',
              hot_manual_rank: Number(override?.manual_rank || 0),
            },
          });
        });
      });
      setRows(nextRows);

      const nextSettings = {
        period: settingsRes.data.data?.period || '7d',
        limit: Number(settingsRes.data.data?.limit || 8),
      };
      setSettings(nextSettings);
      setPendingSettings(nextSettings);
    } catch (error) {
      showError(error.message || t('加载失败'));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const previewModels = useMemo(() => {
    const modelMap = new Map();
    rows.forEach((row) => {
      if (!modelMap.has(row.model_name)) {
        modelMap.set(row.model_name, {
          model_name: row.model_name,
          channel_list: [],
        });
      }
      modelMap.get(row.model_name).channel_list.push({
        ...row.channel,
        channel_id: row.channel_id,
        auto_req_count: row.auto_req_count,
        channel_heat_score: row.auto_req_count,
        hot_override:
          row.override_mode === HOT_OVERRIDE_AUTO ? '' : row.override_mode,
        hot_manual_rank: row.manual_rank,
      });
    });
    return Array.from(modelMap.values());
  }, [rows]);

  const hotPreview = useMemo(
    () => getTopHotChannels(previewModels, settings.limit),
    [previewModels, settings.limit],
  );

  const updateLocalOverride = useCallback((key, mode, rank) => {
    setRows((previous) =>
      previous.map((row) =>
        row.key === key
          ? {
              ...row,
              override_mode: mode,
              manual_rank: rank,
            }
          : row,
      ),
    );
  }, []);

  const saveOverride = useCallback(
    async (row, mode = row.override_mode, rank = row.manual_rank) => {
      try {
        const normalizedRank = mode === HOT_OVERRIDE_FORCE_HOT ? rank || 0 : 0;
        const res = await API.put('/api/channel/hot-override', {
          channel_id: row.channel_id,
          model_name: row.model_name,
          override_mode: mode,
          manual_rank: normalizedRank,
        });
        if (!res.data.success) {
          showError(res.data.message || t('保存失败'));
          return false;
        }
        updateLocalOverride(row.key, mode, normalizedRank);
        showSuccess(t('保存成功'));
        return true;
      } catch (error) {
        showError(error.message || t('保存失败'));
        return false;
      }
    },
    [t, updateLocalOverride],
  );

  const saveSettings = async () => {
    try {
      const res = await API.put('/api/channel/hot-settings', pendingSettings);
      if (!res.data.success) {
        showError(res.data.message || t('保存失败'));
        return;
      }
      setSettings(pendingSettings);
      setSettingsVisible(false);
      showSuccess(t('保存成功'));
      loadData();
    } catch (error) {
      showError(error.message || t('保存失败'));
    }
  };

  const saveBatch = async () => {
    const selectedRows = rows.filter((row) =>
      selectedRowKeys.includes(row.key),
    );
    if (selectedRows.length === 0) return;
    try {
      const normalizedRank =
        batchMode === HOT_OVERRIDE_FORCE_HOT ? batchRank || 0 : 0;
      const res = await API.put('/api/channel/hot-overrides/batch', {
        overrides: selectedRows.map((row) => ({
          channel_id: row.channel_id,
          model_name: row.model_name,
          override_mode: batchMode,
          manual_rank: normalizedRank,
        })),
      });
      if (!res.data.success) {
        showError(res.data.message || t('保存失败'));
        return;
      }
      const selectedSet = new Set(selectedRowKeys);
      setRows((previous) =>
        previous.map((row) =>
          selectedSet.has(row.key)
            ? {
                ...row,
                override_mode: batchMode,
                manual_rank: normalizedRank,
              }
            : row,
        ),
      );
      setSelectedRowKeys([]);
      setBatchVisible(false);
      showSuccess(t('批量保存成功'));
    } catch (error) {
      showError(error.message || t('批量保存失败'));
    }
  };

  const filteredRows = useMemo(() => {
    const modelKeyword = searchModel.trim().toLowerCase();
    const channelKeyword = searchChannel.trim().toLowerCase();
    return rows.filter((row) => {
      if (
        modelKeyword &&
        !row.model_name.toLowerCase().includes(modelKeyword)
      ) {
        return false;
      }
      if (channelKeyword) {
        const channelText = [
          row.channel_id,
          row.channel_no,
          row.route_slug,
          row.supplier_alias,
          row.supplier_type,
        ]
          .filter(Boolean)
          .join(' ')
          .toLowerCase();
        if (!channelText.includes(channelKeyword)) return false;
      }
      return overrideFilter === 'all' || row.override_mode === overrideFilter;
    });
  }, [overrideFilter, rows, searchChannel, searchModel]);

  useEffect(() => {
    setCurrentPage(1);
  }, [searchModel, searchChannel, overrideFilter]);

  const paginatedRows = useMemo(() => {
    const start = (currentPage - 1) * pageSize;
    return filteredRows.slice(start, start + pageSize);
  }, [currentPage, filteredRows, pageSize]);

  const getFinalStatus = useCallback(
    (row) => {
      const key = getChannelHeatKey(
        { model_name: row.model_name },
        { channel_id: row.channel_id },
      );
      if (!hotPreview.scoreMap.has(key)) {
        return { label: t('非热门'), color: 'grey' };
      }
      if (row.override_mode === HOT_OVERRIDE_FORCE_HOT) {
        return { label: t('人工热门'), color: 'red' };
      }
      return { label: t('自动热门'), color: 'orange' };
    },
    [hotPreview.scoreMap, t],
  );

  const columns = useMemo(
    () => [
      {
        title: t('模型 / 渠道'),
        dataIndex: 'model_name',
        width: 320,
        render: (_, row) => (
          <div className='flex min-w-0 flex-col gap-1'>
            <Space spacing={8}>
              {getLobeHubIcon(
                row.model_icon || row.vendor?.icon || 'Layers',
                20,
              )}
              <Text strong copyable ellipsis={{ showTooltip: true }}>
                {row.model_name}
              </Text>
            </Space>
            <Text
              type='secondary'
              size='small'
              ellipsis={{ showTooltip: true }}
            >
              {[
                row.supplier_alias,
                row.route_slug || row.channel_no,
                `ID ${row.channel_id}`,
              ]
                .filter(Boolean)
                .join(' · ')}
            </Text>
          </div>
        ),
      },
      {
        title: t('实际调用'),
        dataIndex: 'auto_req_count',
        width: 120,
        sorter: (a, b) => a.auto_req_count - b.auto_req_count,
        render: (value) => Number(value || 0).toLocaleString(),
      },
      {
        title: t('人工覆盖'),
        dataIndex: 'override_mode',
        width: 150,
        render: (value, row) => (
          <Select
            value={value}
            optionList={overrideOptions}
            onChange={(mode) => saveOverride(row, mode, row.manual_rank)}
            style={{ width: 130 }}
          />
        ),
      },
      {
        title: t('人工顺序'),
        dataIndex: 'manual_rank',
        width: 120,
        render: (value, row) => (
          <InputNumber
            value={value}
            min={0}
            max={1000000}
            step={1}
            disabled={row.override_mode !== HOT_OVERRIDE_FORCE_HOT}
            onChange={(rank) =>
              updateLocalOverride(row.key, row.override_mode, Number(rank || 0))
            }
            onBlur={() => {
              const latest = rows.find((item) => item.key === row.key) || row;
              saveOverride(latest);
            }}
            style={{ width: 90 }}
          />
        ),
      },
      {
        title: t('最终状态'),
        dataIndex: 'final_status',
        width: 120,
        render: (_, row) => {
          const status = getFinalStatus(row);
          return <Tag color={status.color}>{status.label}</Tag>;
        },
      },
    ],
    [getFinalStatus, rows, saveOverride, t, updateLocalOverride],
  );

  return (
    <>
      <Card
        title={
          <div className='flex items-center text-orange-500'>
            <Flame size={16} className='mr-2' />
            <span>{t('首页热门配置')}</span>
          </div>
        }
        headerExtraContent={
          <Text type='secondary'>
            {t('自动热度按实际调用统计，人工覆盖优先于自动排名')}
          </Text>
        }
      >
        <Row gutter={[12, 12]} className='mb-4'>
          <Col xs={24} md={6}>
            <Input
              value={searchModel}
              onChange={setSearchModel}
              showClear
              placeholder={t('搜索模型')}
            />
          </Col>
          <Col xs={24} md={6}>
            <Input
              value={searchChannel}
              onChange={setSearchChannel}
              showClear
              placeholder={t('搜索渠道')}
            />
          </Col>
          <Col xs={24} md={5}>
            <Select
              value={overrideFilter}
              onChange={setOverrideFilter}
              optionList={[
                { value: 'all', label: t('全部状态') },
                ...overrideOptions,
              ]}
              style={{ width: '100%' }}
            />
          </Col>
          <Col xs={24} md={7}>
            <div className='flex justify-end'>
              <Space wrap>
                <Button
                  icon={<Clock size={16} />}
                  onClick={() => {
                    setPendingSettings(settings);
                    setSettingsVisible(true);
                  }}
                >
                  {settings.period} · {t('自动热门')} {settings.limit}
                </Button>
                <Button
                  icon={<RefreshCw size={16} />}
                  onClick={loadData}
                  loading={loading}
                >
                  {t('刷新')}
                </Button>
                <Button
                  type='primary'
                  icon={<SlidersHorizontal size={16} />}
                  disabled={selectedRowKeys.length === 0}
                  onClick={() => setBatchVisible(true)}
                >
                  {t('批量设置')} ({selectedRowKeys.length})
                </Button>
              </Space>
            </div>
          </Col>
        </Row>

        <Table
          columns={columns}
          dataSource={paginatedRows}
          loading={loading}
          rowKey='key'
          rowSelection={{ selectedRowKeys, onChange: setSelectedRowKeys }}
          pagination={{
            currentPage,
            pageSize,
            total: filteredRows.length,
            showSizeChanger: true,
            pageSizeOpts: [20, 50, 100],
            onChange: (page, size) => {
              setCurrentPage(page);
              setPageSize(size);
            },
          }}
        />
      </Card>

      <Modal
        title={t('热门统计设置')}
        visible={settingsVisible}
        onCancel={() => setSettingsVisible(false)}
        onOk={saveSettings}
        centered
      >
        <div className='space-y-4'>
          <div>
            <Text strong>{t('统计周期')}</Text>
            <Select
              value={pendingSettings.period}
              onChange={(period) =>
                setPendingSettings((previous) => ({ ...previous, period }))
              }
              optionList={[
                { value: '7d', label: t('最近 7 天') },
                { value: '30d', label: t('最近 30 天') },
                { value: 'all', label: t('全部历史') },
              ]}
              style={{ width: '100%', marginTop: 8 }}
            />
          </div>
          <div>
            <Text strong>{t('自动热门模型数量')}</Text>
            <InputNumber
              value={pendingSettings.limit}
              min={1}
              max={100}
              onChange={(limit) =>
                setPendingSettings((previous) => ({
                  ...previous,
                  limit: Number(limit || 1),
                }))
              }
              style={{ width: '100%', marginTop: 8 }}
            />
          </div>
        </div>
      </Modal>

      <Modal
        title={t('批量设置')}
        visible={batchVisible}
        onCancel={() => setBatchVisible(false)}
        onOk={saveBatch}
        okButtonProps={{ icon: <Save size={16} /> }}
        centered
      >
        <div className='space-y-4'>
          <Select
            value={batchMode}
            onChange={setBatchMode}
            optionList={overrideOptions}
            style={{ width: '100%' }}
          />
          {batchMode === HOT_OVERRIDE_FORCE_HOT ? (
            <InputNumber
              value={batchRank}
              min={0}
              max={1000000}
              step={1}
              onChange={(rank) => setBatchRank(Number(rank || 0))}
              placeholder={t('人工顺序，0 表示未指定')}
              style={{ width: '100%' }}
            />
          ) : null}
        </div>
      </Modal>
    </>
  );
};

export default CombinedHeatConfig;
