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

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import {
  Table,
  Card,
  InputNumber,
  Button,
  Space,
  Tag,
  Typography,
  Row,
  Col,
  Input,
  Select,
  Modal,
  Form,
} from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { Flame, Server, TrendingUp, Hash, Save, RefreshCw, Filter, Clock } from 'lucide-react';
import { API, showError, showSuccess, getLobeHubIcon, stringToColor } from '@/helpers';
import { getChannelHeatScore } from '../../components/table/model-pricing/utils/modelHeat';

const { Title, Text } = Typography;
const { Option } = Select;

const CombinedHeatConfig = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState([]);
  const [models, setModels] = useState([]);
  const [channels, setChannels] = useState([]);
  const [vendors, setVendors] = useState([]);
  const [searchVendor, setSearchVendor] = useState('');
  const [searchModel, setSearchModel] = useState('');
  const [searchChannel, setSearchChannel] = useState('');
  const [selectedRowKeys, setSelectedRowKeys] = useState([]);
  const [batchModalVisible, setBatchModalVisible] = useState(false);
  const [batchForm, setBatchForm] = useState({
    model_weight: null,
    channel_weight: null,
    manual_base_7d: null,
  });
  const [heatPeriod, setHeatPeriod] = useState('7d');
  const [periodModalVisible, setPeriodModalVisible] = useState(false);
  const [pendingPeriod, setPendingPeriod] = useState('7d');
  const [pageSize, setPageSize] = useState(20);
  const [currentPage, setCurrentPage] = useState(1);

  // 加载模型列表
  const loadModels = useCallback(async () => {
    try {
      const res = await API.get('/api/pricing/');
      if (res.data.success) {
        const modelList = res.data.data || [];
        setModels(modelList);
        return modelList;
      }
    } catch (error) {
      console.error('加载模型失败', error);
    }
    return [];
  }, []);

  // 加载热度统计周期
  const loadHeatPeriod = useCallback(async () => {
    try {
      const res = await API.get('/api/channel/heat/period');
      if (res.data.success) {
        setHeatPeriod(res.data.data || '7d');
      }
    } catch (error) {
      console.error('加载热度统计周期失败', error);
    }
  }, []);

  // 保存热度统计周期
  const saveHeatPeriod = async (period) => {
    try {
      const res = await API.put('/api/channel/heat/period', { period });
      if (res.data.success) {
        setHeatPeriod(period);
        setPeriodModalVisible(false);
        showSuccess(t('统计周期已更新，重新加载数据中...'));
        loadData();
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(t('保存失败'));
    }
  };

  // 加载渠道列表
  const loadChannels = useCallback(async () => {
    try {
      const res = await API.get('/api/channel/?page_size=1000');
      if (res.data.success) {
        const items = res.data.data?.items || [];
        setChannels(items);
        return items;
      }
    } catch (error) {
      console.error('加载渠道失败', error);
    }
    return [];
  }, []);

  // 加载渠道-模型热力配置
  const loadChannelModelHeats = useCallback(async () => {
    try {
      const res = await API.get('/api/channel/heats');
      if (res.data.success) {
        return res.data.data || [];
      }
    } catch (error) {
      console.error('加载热力配置失败', error);
    }
    return [];
  }, []);

  // 加载厂商列表
  const loadVendors = useCallback(async () => {
    try {
      const res = await API.get('/api/vendors/?page_size=1000');
      if (res.data.success) {
        const vendorList = res.data.data?.items || [];
        setVendors(vendorList);
        return vendorList;
      }
    } catch (error) {
      console.error('加载厂商列表失败', error);
    }
    return [];
  }, []);

  // 合并数据：扁平化显示，每个模型-渠道组合作为独立行
  const mergeData = useCallback((modelList, channelList, heatsMap) => {
    const merged = [];
    
    modelList.forEach((model) => {
      // API 返回的是 channel_list，字段名为 ChannelNo 而非 channel_id
      const channelListFromAPI = model.channel_list || model.ChannelList || [];
      
      if (channelListFromAPI.length > 0) {
        // 有渠道数据时，为每个渠道创建一行
        channelListFromAPI.forEach((ch) => {
          const channelID = ch.ChannelNo || ch.channel_no || ch.channel_id || ch.id;
          const channelInfo = channelList.find((c) => c.id === channelID || c.id === parseInt(channelID));
          
          // 查找该渠道-模型组合的自定义热力配置
          const heatKey = `${channelID}-${model.model_name}`;
          const heatConfig = heatsMap?.get(heatKey);
          
          merged.push({
            key: `channel-${model.model_name}-${channelID}`,
            type: 'channel',
            id: model.id,
            model_name: model.model_name,
            model_icon: model.icon || '',
            vendor_id: model.vendor_id,
            tags: model.tags || '',
            channel_id: channelID,
            channel_name: channelInfo?.name || ch.SupplierAlias || `${t('渠道')}${channelID}`,
            // 全部使用渠道-模型独立配置（channel_model_heats 表）
            model_sort_weight: heatConfig?.model_sort_weight ?? 1,
            channel_sort_weight: heatConfig?.channel_sort_weight ?? 1,
            manual_base_req_count: heatConfig?.manual_base_req_count ?? 0,
            req_count_7d: ch.auto_req_count ?? 0,
            heat_score: ch.ChannelHeatScore ?? ch.channel_heat_score ?? 0,
            enabled: channelInfo?.status === 1,
          });
        });
      } else {
        // 没有渠道数据时，只显示模型行
        // 查找全局模型热力配置（channel_id=0）
        const globalHeatKey = `0-${model.model_name}`;
        const globalHeatConfig = heatsMap?.get(globalHeatKey);
        
        merged.push({
          key: `model-${model.model_name}`,
          type: 'model',
          id: model.id,
          model_name: model.model_name,
          model_icon: model.icon || '',
          vendor_id: model.vendor_id,
          tags: model.tags || '',
          channel_id: null,
          channel_name: '-',
          // 从 channel_model_heats 表读取全局配置（channel_id=0）
          model_sort_weight: globalHeatConfig?.model_sort_weight ?? 1,
          channel_sort_weight: 1,
          sort_weight: globalHeatConfig?.model_sort_weight ?? 1,
          manual_base_req_count: globalHeatConfig?.manual_base_req_count ?? 0,
          req_count_7d: model.auto_req_count ?? 0,
          heat_score: model.model_heat_score ?? 0,
          enabled: false,
        });
      }
    });

    return merged;
  }, [t]);

  // 加载所有数据
  const loadData = useCallback(async () => {
    setLoading(true);
    const [modelList, channelList, heatsList, vendorList] = await Promise.all([
      loadModels(),
      loadChannels(),
      loadChannelModelHeats(),
      loadVendors(),
    ]);
    
    // 构建热力配置 Map: "channelID-modelName" -> heatConfig
    const heatsMap = new Map();
    heatsList.forEach((heat) => {
      const key = `${heat.channel_id}-${heat.model_name}`;
      heatsMap.set(key, heat);
    });

    // 构建厂商 Map: vendor_id -> vendor
    const vendorMap = new Map();
    vendorList.forEach((vendor) => {
      vendorMap.set(vendor.id, vendor);
    });
    
    const merged = mergeData(modelList, channelList, heatsMap);
    // 为合并后的数据添加厂商信息
    const mergedWithVendor = merged.map((item) => ({
      ...item,
      vendor: vendorMap.get(item.vendor_id),
    }));
    setData(mergedWithVendor);
    setLoading(false);
  }, [loadModels, loadChannels, loadChannelModelHeats, loadVendors, mergeData]);

  useEffect(() => {
    loadHeatPeriod();
    loadData();
  }, [loadData, loadHeatPeriod]);

  // 处理字段变更
  const handleFieldChange = (key, field, value) => {
    setData((prev) =>
      prev.map((item) => {
        if (item.key === key) {
          return { ...item, [field]: value };
        }
        if (item.children) {
          return {
            ...item,
            children: item.children.map((child) =>
              child.key === key ? { ...child, [field]: value } : child
            ),
          };
        }
        return item;
      })
    );
  };

  // 计算热度分
  const calculateHeatScore = useCallback(
    (record) => getChannelHeatScore(record),
    [],
  );

  // 保存单个记录
  const handleSave = async (record, { silent = false } = {}) => {
    try {
      if (record.type === 'model') {
        // 保存模型权重 - 使用POST /api/models/batch_weight
        // 保存到 channel_model_heats（全局 model 行，channel_id 为 null 或 0）
        // 这里我们需要创建一个特殊的 key 来存储模型全局配置
        await API.put('/api/channel/heat', {
          channel_id: 0, // 使用 0 表示全局模型配置
          model_name: record.model_name,
          model_sort_weight: record.model_sort_weight,
          channel_sort_weight: 1,
          manual_base_req_count: record.manual_base_req_count,
        });
      } else {
        // 保存渠道-模型组合热力配置
        await API.put('/api/channel/heat', {
          channel_id: parseInt(record.channel_id),
          model_name: record.model_name,
          model_sort_weight: record.model_sort_weight ?? 1,
          channel_sort_weight: record.channel_sort_weight,
          manual_base_req_count: record.manual_base_req_count,
        });
      }
      if (silent) {
        showSuccess(t('操作成功完成！'));
      } else {
        showSuccess(t('保存成功'));
        loadData();
      }
    } catch (error) {
      showError(t('保存失败'));
    }
  };

  // 批量保存
  const handleBatchSave = async () => {
    const updates = [];

    data.forEach((item) => {
      if (selectedRowKeys.includes(item.key)) {
        updates.push({
          type: item.type,
          model_name: item.model_name,
          channel_id: item.channel_id,
          ...batchForm,
        });
      }
    });

    // 分批保存
    try {
      // 所有配置都保存到 channel_model_heats 表
      const heatUpdates = [];
      
      for (const update of updates) {
        if (update.type === 'model') {
          // 模型行：channel_id 为 0
          heatUpdates.push({
            channel_id: 0,
            model_name: update.model_name,
            model_sort_weight: update.model_weight ?? 1,
            channel_sort_weight: 1,
            manual_base_req_count: update.manual_base_7d ?? 0,
          });
        } else {
          // 渠道-模型行
          heatUpdates.push({
            channel_id: parseInt(update.channel_id),
            model_name: update.model_name,
            model_sort_weight: update.model_weight ?? 1,
            channel_sort_weight: update.channel_weight ?? 1,
            manual_base_req_count: update.manual_base_7d ?? 0,
          });
        }
      }
      
      // 批量保存所有热力配置
      if (heatUpdates.length > 0) {
        await API.put('/api/channel/heats/batch', {
          heats: heatUpdates,
        });
      }
      
      showSuccess(t('批量保存成功'));
      setBatchModalVisible(false);
      loadData();
    } catch (error) {
      showError(t('批量保存失败'));
    }
  };

  // 过滤数据（扁平化）
  const filteredData = useMemo(() => {
    let result = [...data];

    if (searchModel) {
      result = result.filter(
        (item) =>
          item.model_name.toLowerCase().includes(searchModel.toLowerCase())
      );
    }

    if (searchChannel) {
      result = result.filter(
        (item) =>
          item.channel_name?.toLowerCase().includes(searchChannel.toLowerCase())
      );
    }

    if (searchVendor) {
      result = result.filter((item) => item.vendor_id === parseInt(searchVendor));
    }

    return result.sort(
      (a, b) => calculateHeatScore(b) - calculateHeatScore(a)
    );
  }, [data, searchModel, searchChannel, searchVendor, calculateHeatScore]);

  const vendorSearchTextByValue = useMemo(() => {
    const map = new Map();
    vendors.forEach((vendor) => {
      map.set(
        vendor.id.toString(),
        [vendor.id, vendor.name, vendor.description].filter(Boolean).join(' ').toLowerCase(),
      );
    });
    return map;
  }, [vendors]);

  const filterVendorOption = useCallback(
    (input, option) => {
      const keyword = String(input ?? '').trim().toLowerCase();
      if (!keyword) return true;
      return (vendorSearchTextByValue.get(String(option?.value ?? '')) || '').includes(keyword);
    },
    [vendorSearchTextByValue],
  );

  const renderVendorOption = (vendor) => (
    <Space>
      {getLobeHubIcon(vendor.icon || 'Layers', 16)}
      {vendor.name}
    </Space>
  );

  const paginatedData = useMemo(() => {
    const startIndex = (currentPage - 1) * pageSize;
    return filteredData.slice(startIndex, startIndex + pageSize);
  }, [filteredData, currentPage, pageSize]);

  useEffect(() => {
    setCurrentPage(1);
  }, [searchModel, searchChannel, searchVendor]);

  // 表格列定义
  const columns = [
    {
      title: t('序号'),
      dataIndex: 'index',
      width: 70,
      render: (text, record, index) => (
        <Text type='secondary'>{(currentPage - 1) * pageSize + index + 1}</Text>
      ),
    },
    {
      title: t('模型 / 渠道'),
      dataIndex: 'model_name',
      width: 300,
      render: (text, record) => (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
          <Space>
            {getLobeHubIcon(record.model_icon || record.vendor?.icon || 'Layers', 20)}
            <Text strong copyable>{text}</Text>
          </Space>
          {record.channel_name && record.channel_name !== '-' ? (
            <Space style={{ paddingLeft: 4 }}>
              <Text type='secondary' size='small' copyable>{record.channel_name}</Text>
            </Space>
          ) : null}
        </div>
      ),
    },
    {
      title: t('模型类型'),
      dataIndex: 'vendor',
      width: 120,
      render: (vendor) => {
        if (!vendor) return '-';
        return (
          <Tag
            color='white'
            shape='circle'
            prefixIcon={getLobeHubIcon(vendor.icon || 'Layers', 14)}
          >
            {vendor.name}
          </Tag>
        );
      },
    },
    {
      title: t('标签'),
      dataIndex: 'tags',
      width: 150,
      render: (tags) => {
        const tagArray = Array.isArray(tags) ? tags : (tags ? String(tags).split(',').filter(Boolean) : []);
        if (tagArray.length === 0) return '-';
        return (
          <Space wrap>
            {tagArray.map((tag, index) => (
              <Tag key={index} size='small' shape='circle' color={stringToColor(tag)}>{tag}</Tag>
            ))}
          </Space>
        );
      },
    },
    {
      title: (
        <Space>
          <TrendingUp size={14} />
          {t('渠道权重')}
        </Space>
      ),
      dataIndex: 'channel_sort_weight',
      width: 120,
      render: (text, record) => (
        <InputNumber
          value={record.channel_sort_weight ?? 1}
          min={0}
          max={10}
          step={0.1}
          precision={2}
          onChange={(value) => handleFieldChange(record.key, 'channel_sort_weight', value)}
          onBlur={() => handleSave(record, { silent: true })}
          style={{ width: 100 }}
          keepFocus={true}
          innerButtons
        />
      ),
    },
    {
      title: (
        <Space>
          <Hash size={14} />
          {t('手动调整')}
        </Space>
      ),
      dataIndex: 'manual_base_req_count',
      width: 130,
      render: (text, record) => (
        <InputNumber
          value={record.manual_base_req_count ?? 0}
          step={1}
          onChange={(value) =>
            handleFieldChange(record.key, 'manual_base_req_count', value)
          }
          onBlur={() => handleSave(record, { silent: true })}
          style={{ width: 100 }}
          keepFocus={true}
          innerButtons
        />
      ),
    },
    {
      title: t('实际调用'),
      dataIndex: 'req_count_7d',
      width: 110,
      render: (text) => <Text type='secondary'>{(text ?? 0).toLocaleString()}</Text>,
    },
    {
      title: (
        <Space>
          <TrendingUp size={14} />
          {t('预估热度分')}
        </Space>
      ),
      dataIndex: 'heat_score',
      width: 150,
      render: (text, record) => {
        const score = calculateHeatScore(record);
        let color = 'default';
        if (score > 10000) color = 'red';
        else if (score > 1000) color = 'orange';
        else if (score > 100) color = 'yellow';
        return <Tag color={color}>{score.toFixed(2)}</Tag>;
      },
    },
  ];

  const rowSelection = {
    selectedRowKeys,
    onChange: setSelectedRowKeys,
  };

  return (
    <>
      <Card
        title={
          <div className='flex items-center text-orange-500'>
            <Flame size={16} className='mr-2' />
            <span>{t('热度配置')}</span>
          </div>
        }
        headerExtraContent={
          <Text type='secondary'>
            {t('热度分 = (手动基数 + 实际调用) × 权重')}
          </Text>
        }
      >

        <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
          <Col span={4}>
            <Select
              placeholder={t('搜索模型')}
              value={searchModel || undefined}
              onChange={(value) => setSearchModel(value || '')}
              showClear
              filter
              maxTagCount={1}
              style={{ width: '100%' }}
            >
              {[...new Set(data.map((item) => item.model_name))].map((name) => (
                <Option key={name} value={name}>
                  {name}
                </Option>
              ))}
            </Select>
          </Col>
          <Col span={4}>
            <Select
              placeholder={t('搜索模型类型')}
              value={searchVendor || undefined}
              onChange={(value) => setSearchVendor(value || '')}
              showClear
              filter={filterVendorOption}
              maxTagCount={1}
              renderSelectedItem={(optionNode) => optionNode?.label || optionNode?.value || ''}
              style={{ width: '100%' }}
            >
              {vendors.map((vendor) => (
                <Option
                  key={vendor.id}
                  value={vendor.id.toString()}
                  label={renderVendorOption(vendor)}
                >
                  {renderVendorOption(vendor)}
                </Option>
              ))}
            </Select>
          </Col>
          <Col span={4}>
            <Select
              placeholder={t('搜索渠道')}
              value={searchChannel || undefined}
              onChange={(value) => setSearchChannel(value || '')}
              showClear
              filter
              maxTagCount={1}
              style={{ width: '100%' }}
            >
              {[...new Set(data.map((item) => item.channel_name).filter(Boolean))].map((name) => (
                <Option key={name} value={name}>
                  {name}
                </Option>
              ))}
            </Select>
          </Col>
          <Col span={12} style={{ textAlign: 'right' }}>
            <Space>
              <Button
                icon={<Clock size={16} />}
                onClick={() => { setPendingPeriod(heatPeriod); setPeriodModalVisible(true); }}
              >
                {t('统计周期')}: {heatPeriod === 'all' ? t('全部历史') : heatPeriod}
              </Button>
              <Button icon={<RefreshCw size={16} />} onClick={loadData} loading={loading}>
                {t('刷新')}
              </Button>
              <Button
                type='primary'
                icon={<Save size={16} />}
                disabled={selectedRowKeys.length === 0}
                onClick={() => setBatchModalVisible(true)}
              >
                {t('批量设置')} ({selectedRowKeys.length})
              </Button>
            </Space>
          </Col>
        </Row>

        <Table
          columns={columns}
          dataSource={paginatedData}
          loading={loading}
          rowSelection={rowSelection}
          pagination={{
            currentPage,
            pageSize,
            total: filteredData.length,
            showSizeChanger: true,
            pageSizeOpts: [20, 50, 100],
            onChange: (page, size) => {
              setCurrentPage(page);
              setPageSize(size);
            },
            onShowSizeChange: (current, size) => {
              setCurrentPage(1);
              setPageSize(size);
            },
          }}
          rowKey='key'
        />
      </Card>

      {/* 批量设置模态框 */}
      <Modal
        title={t('批量设置')}
        visible={batchModalVisible}
        onOk={handleBatchSave}
        onCancel={() => setBatchModalVisible(false)}
        centered
      >
        <Form layout='vertical'>
          <Form.Section text={t('权重配置')}>
            <Row gutter={16}>
              <Col span={24}>
                <Form.InputNumber
                  field='channel_weight'
                  label={t('渠道权重')}
                  min={0}
                  max={10}
                  step={0.1}
                  precision={2}
                  placeholder={t('不修改')}
                  onChange={(value) =>
                    setBatchForm((prev) => ({ ...prev, channel_weight: value }))
                  }
                />
              </Col>
            </Row>
          </Form.Section>
          <Form.Section text={t('手动调整')}>          
            <Row gutter={16}>
              <Col span={12}>
                <Form.InputNumber
                  field='manual_base_7d'
                  label={t('手动调整')}
                  step={1}
                  placeholder={t('不修改')}
                  onChange={(value) =>
                    setBatchForm((prev) => ({ ...prev, manual_base_7d: value }))
                  }
                />
              </Col>
            </Row>
          </Form.Section>
        </Form>
      </Modal>
      {/* 统计周期设置弹框 */}
      <Modal
        title={
          <Space>
            <Clock size={16} />
            {t('统计周期设置')}
          </Space>
        }
        visible={periodModalVisible}
        onOk={() => saveHeatPeriod(pendingPeriod)}
        onCancel={() => setPeriodModalVisible(false)}
        centered
      >
        <div style={{ padding: '16px 0' }}>
          <Text>{t('选择热度统计使用的时间范围，影响"实际调用"列和后端排序计算。')}</Text>
          <div style={{ marginTop: 16 }}>
            <Select
              value={pendingPeriod}
              onChange={setPendingPeriod}
              style={{ width: '100%' }}
            >
              <Option value='7d'>{t('近 7 天')}</Option>
              <Option value='30d'>{t('近 30 天')}</Option>
              <Option value='all'>{t('全部历史')}</Option>
            </Select>
          </div>
        </div>
      </Modal>
    </>
  );
};

export default CombinedHeatConfig;
