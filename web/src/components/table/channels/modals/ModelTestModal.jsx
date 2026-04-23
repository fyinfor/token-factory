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

import React, { useState, useEffect, useCallback } from 'react';
import {
  Modal,
  Button,
  Input,
  Table,
  Tag,
  Typography,
  Select,
  Switch,
  Banner,
  InputNumber,
} from '@douyinfe/semi-ui';
import { IconSearch, IconInfoCircle } from '@douyinfe/semi-icons';
import {
  API,
  copy,
  showError,
  showInfo,
  showSuccess,
} from '../../../../helpers';
import {
  getStabilityGradeLabel,
  renderModelTestResultSummary,
} from '../../../../helpers/modelStability';
import { MODEL_TABLE_PAGE_SIZE } from '../../../../constants';

const ModelTestModal = ({
  showModelTestModal,
  currentTestChannel,
  handleCloseModal,
  isBatchTesting,
  batchTestModels,
  modelSearchKeyword,
  setModelSearchKeyword,
  selectedModelKeys,
  setSelectedModelKeys,
  modelTestResults,
  testingModels,
  testChannel,
  modelTablePage,
  setModelTablePage,
  selectedEndpointType,
  setSelectedEndpointType,
  isStreamTest,
  setIsStreamTest,
  allSelectingRef,
  isMobile,
  t,
}) => {
  /** 渠道下各模型在 DB 中的单测/运营展示行，key 为 model_name */
  const [mtrByModel, setMtrByModel] = useState({});
  const [overrideVisible, setOverrideVisible] = useState(false);
  const [overrideModel, setOverrideModel] = useState(null);
  const [overrideMs, setOverrideMs] = useState(0);
  const [overrideGrade, setOverrideGrade] = useState(0);
  const [overrideSubmitting, setOverrideSubmitting] = useState(false);

  const loadMtrForChannel = useCallback(async () => {
    if (!currentTestChannel?.id) {
      setMtrByModel({});
      return;
    }
    try {
      const res = await API.get(
        `/api/channel/model-test-results?channel_id=${currentTestChannel.id}`,
      );
      if (!res.data?.success) {
        return;
      }
      const map = {};
      (res.data.data || []).forEach((row) => {
        if (row.model_name) {
          map[row.model_name] = row;
        }
      });
      setMtrByModel(map);
    } catch (e) {
      setMtrByModel({});
    }
  }, [currentTestChannel?.id]);

  useEffect(() => {
    if (!showModelTestModal || !currentTestChannel) {
      setMtrByModel({});
      return;
    }
    const tmr = setTimeout(() => {
      loadMtrForChannel();
    }, 300);
    return () => clearTimeout(tmr);
  }, [
    showModelTestModal,
    currentTestChannel,
    loadMtrForChannel,
    modelTestResults,
  ]);

  const openOverride = (modelName) => {
    setOverrideModel(modelName);
    const row = mtrByModel[modelName];
    setOverrideMs(
      row?.manual_display_response_time > 0
        ? row.manual_display_response_time
        : 0,
    );
    setOverrideGrade(row?.manual_stability_grade > 0 ? row.manual_stability_grade : 0);
    setOverrideVisible(true);
  };

  const submitOverride = async () => {
    if (!currentTestChannel || !overrideModel) {
      return;
    }
    setOverrideSubmitting(true);
    try {
      const res = await API.put('/api/channel/model-test-result-display', {
        channel_id: currentTestChannel.id,
        model_name: overrideModel,
        manual_display_response_time: overrideMs > 0 ? Math.round(overrideMs) : 0,
        manual_stability_grade: overrideGrade,
      });
      if (res.data?.success) {
        showSuccess(t('已保存运营展示数据'));
        setOverrideVisible(false);
        loadMtrForChannel();
      } else {
        showError(res.data?.message || t('保存失败'));
      }
    } catch (e) {
      showError(t('保存失败'));
    } finally {
      setOverrideSubmitting(false);
    }
  };

  const hasChannel = Boolean(currentTestChannel);
  const streamToggleDisabled = [
    'embeddings',
    'image-generation',
    'jina-rerank',
    'openai-response-compact',
  ].includes(selectedEndpointType);

  React.useEffect(() => {
    if (streamToggleDisabled && isStreamTest) {
      setIsStreamTest(false);
    }
  }, [streamToggleDisabled, isStreamTest, setIsStreamTest]);

  const filteredModels = hasChannel
    ? currentTestChannel.models
        .split(',')
        .filter((model) =>
          model.toLowerCase().includes(modelSearchKeyword.toLowerCase()),
        )
    : [];

  const endpointTypeOptions = [
    { value: '', label: t('自动检测') },
    { value: 'openai', label: 'OpenAI (/v1/chat/completions)' },
    { value: 'openai-response', label: 'OpenAI Response (/v1/responses)' },
    {
      value: 'openai-response-compact',
      label: 'OpenAI Response Compaction (/v1/responses/compact)',
    },
    { value: 'anthropic', label: 'Anthropic (/v1/messages)' },
    {
      value: 'gemini',
      label: 'Gemini (/v1beta/models/{model}:generateContent)',
    },
    { value: 'jina-rerank', label: 'Jina Rerank (/v1/rerank)' },
    {
      value: 'image-generation',
      label: t('图像生成') + ' (/v1/images/generations)',
    },
    { value: 'embeddings', label: 'Embeddings (/v1/embeddings)' },
  ];

  const handleCopySelected = () => {
    if (selectedModelKeys.length === 0) {
      showError(t('请先选择模型！'));
      return;
    }
    copy(selectedModelKeys.join(',')).then((ok) => {
      if (ok) {
        showSuccess(
          t('已复制 ${count} 个模型').replace(
            '${count}',
            selectedModelKeys.length,
          ),
        );
      } else {
        showError(t('复制失败，请手动复制'));
      }
    });
  };

  const handleSelectSuccess = () => {
    if (!currentTestChannel) return;
    const successKeys = currentTestChannel.models
      .split(',')
      .filter((m) => m.toLowerCase().includes(modelSearchKeyword.toLowerCase()))
      .filter((m) => {
        const inMemory = modelTestResults[`${currentTestChannel.id}-${m}`];
        if (inMemory) {
          return inMemory.success;
        }
        const persisted = mtrByModel[m];
        return Boolean(persisted?.last_test_success);
      });
    if (successKeys.length === 0) {
      showInfo(t('暂无成功模型'));
    }
    setSelectedModelKeys(successKeys);
  };

  const columns = [
    {
      title: t('模型名称'),
      dataIndex: 'model',
      render: (text) => (
        <div className='flex items-center'>
          <Typography.Text strong>{text}</Typography.Text>
        </div>
      ),
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      render: (text, record) => {
        const testResult =
          modelTestResults[`${currentTestChannel.id}-${record.model}`];
        const persisted = mtrByModel[record.model];
        const isTesting = testingModels.has(record.model);

        if (isTesting) {
          return (
            <Tag color='blue' shape='circle'>
              {t('测试中')}
            </Tag>
          );
        }

        if (!testResult && !persisted) {
          return (
            <Tag color='grey' shape='circle'>
              {t('未开始')}
            </Tag>
          );
        }

        const mergedResult = testResult
          ? {
              success: Boolean(testResult.success),
              time: Number(testResult.time || 0),
            }
          : {
              success: Boolean(persisted?.last_test_success),
              time: Number(persisted?.last_response_time || 0) / 1000,
            };

        return (
          <div className='flex items-center gap-2'>
            <Tag color={mergedResult.success ? 'green' : 'red'} shape='circle'>
              {mergedResult.success ? t('成功') : t('失败')}
            </Tag>
            {mergedResult.success && mergedResult.time > 0 && (
              <Typography.Text type='tertiary'>
                {t('请求时长: ${time}s').replace(
                  '${time}',
                  mergedResult.time.toFixed(2),
                )}
              </Typography.Text>
            )}
          </div>
        );
      },
    },
    {
      title: t('展示状态'),
      dataIndex: 'ops_display',
      width: 220,
      render: (_text, record) => {
        const row = mtrByModel[record.model];
        return (
          <div className='flex flex-col gap-1 items-start max-w-[200px]'>
            {renderModelTestResultSummary(row, t)}
          </div>
        );
      },
    },
    {
      title: '',
      dataIndex: 'operate',
      render: (text, record) => {
        const isTesting = testingModels.has(record.model);
        return (
          <div className='flex items-center gap-2'>
            <Button
              type='tertiary'
              onClick={() =>
                testChannel(
                  currentTestChannel,
                  record.model,
                  selectedEndpointType,
                  isStreamTest,
                )
              }
              loading={isTesting}
              size='small'
            >
              {t('测试')}
            </Button>
            <Button
              type='tertiary'
              size='small'
              onClick={() => openOverride(record.model)}
            >
              {t('手动调整')}
            </Button>
          </div>
        );
      },
    },
  ];

  const dataSource = (() => {
    if (!hasChannel) return [];
    const start = (modelTablePage - 1) * MODEL_TABLE_PAGE_SIZE;
    const end = start + MODEL_TABLE_PAGE_SIZE;
    return filteredModels.slice(start, end).map((model) => ({
      model,
      key: model,
    }));
  })();

  return (
    <Modal
      title={
        hasChannel ? (
          <div className='flex flex-col gap-2 w-full'>
            <div className='flex items-center gap-2'>
              <Typography.Text
                strong
                className='!text-[var(--semi-color-text-0)] !text-base'
              >
                {currentTestChannel.name} {t('渠道的模型测试')}
              </Typography.Text>
              <Typography.Text type='tertiary' size='small'>
                {t('共')} {currentTestChannel.models.split(',').length}{' '}
                {t('个模型')}
              </Typography.Text>
            </div>
          </div>
        ) : null
      }
      visible={showModelTestModal}
      onCancel={handleCloseModal}
      footer={
        hasChannel ? (
          <div className='flex justify-end'>
            {isBatchTesting ? (
              <Button type='danger' onClick={handleCloseModal}>
                {t('停止测试')}
              </Button>
            ) : (
              <Button type='tertiary' onClick={handleCloseModal}>
                {t('取消')}
              </Button>
            )}
            <Button
              onClick={batchTestModels}
              loading={isBatchTesting}
              disabled={isBatchTesting}
            >
              {isBatchTesting
                ? t('测试中...')
                : t('批量测试${count}个模型').replace(
                    '${count}',
                    filteredModels.length,
                  )}
            </Button>
          </div>
        ) : null
      }
      maskClosable={!isBatchTesting}
      className='!rounded-lg'
      size={isMobile ? 'full-width' : 'large'}
    >
      {hasChannel && (
        <div className='model-test-scroll'>
          {/* Endpoint toolbar */}
          <div className='flex flex-col sm:flex-row sm:items-center gap-2 w-full mb-2'>
            <div className='flex items-center gap-2 flex-1 min-w-0'>
              <Typography.Text strong className='shrink-0'>
                {t('端点类型')}:
              </Typography.Text>
              <Select
                value={selectedEndpointType}
                onChange={setSelectedEndpointType}
                optionList={endpointTypeOptions}
                className='!w-full min-w-0'
                placeholder={t('选择端点类型')}
              />
            </div>
            <div className='flex items-center justify-between sm:justify-end gap-2 shrink-0'>
              <Typography.Text strong className='shrink-0'>
                {t('流式')}:
              </Typography.Text>
              <Switch
                checked={isStreamTest}
                onChange={setIsStreamTest}
                size='small'
                disabled={streamToggleDisabled}
                aria-label={t('流式')}
              />
            </div>
          </div>

          <Banner
            type='info'
            closeIcon={null}
            icon={<IconInfoCircle />}
            className='!rounded-lg mb-2'
            description={t(
              '说明：本页测试为非流式请求；若渠道仅支持流式返回，可能出现测试失败，请以实际使用为准。',
            )}
          />

          {/* 搜索与操作按钮 */}
          <div className='flex flex-col sm:flex-row sm:items-center gap-2 w-full mb-2'>
            <Input
              placeholder={t('搜索模型...')}
              value={modelSearchKeyword}
              onChange={(v) => {
                setModelSearchKeyword(v);
                setModelTablePage(1);
              }}
              className='!w-full sm:!flex-1'
              prefix={<IconSearch />}
              showClear
            />

            <div className='flex items-center justify-end gap-2'>
              <Button onClick={handleCopySelected}>{t('复制已选')}</Button>
              <Button type='tertiary' onClick={handleSelectSuccess}>
                {t('选择成功')}
              </Button>
            </div>
          </div>

          <Table
            columns={columns}
            dataSource={dataSource}
            rowSelection={{
              selectedRowKeys: selectedModelKeys,
              onChange: (keys) => {
                if (allSelectingRef.current) {
                  allSelectingRef.current = false;
                  return;
                }
                setSelectedModelKeys(keys);
              },
              onSelectAll: (checked) => {
                allSelectingRef.current = true;
                setSelectedModelKeys(checked ? filteredModels : []);
              },
            }}
            pagination={{
              currentPage: modelTablePage,
              pageSize: MODEL_TABLE_PAGE_SIZE,
              total: filteredModels.length,
              showSizeChanger: false,
              onPageChange: (page) => setModelTablePage(page),
            }}
          />
          <Modal
            title={t('手动调整')}
            visible={overrideVisible}
            onCancel={() => setOverrideVisible(false)}
            maskClosable={!overrideSubmitting}
            footer={
              <div className='flex justify-end gap-2'>
                <Button onClick={() => setOverrideVisible(false)} disabled={overrideSubmitting}>
                  {t('取消')}
                </Button>
                <Button
                  type='tertiary'
                  onClick={async () => {
                    setOverrideMs(0);
                    setOverrideGrade(0);
                    if (!currentTestChannel || !overrideModel) {
                      return;
                    }
                    setOverrideSubmitting(true);
                    try {
                      const res = await API.put('/api/channel/model-test-result-display', {
                        channel_id: currentTestChannel.id,
                        model_name: overrideModel,
                        manual_display_response_time: 0,
                        manual_stability_grade: 0,
                      });
                      if (res.data?.success) {
                        showSuccess(t('已清除运营展示覆盖'));
                        setOverrideVisible(false);
                        loadMtrForChannel();
                      } else {
                        showError(res.data?.message || t('操作失败'));
                      }
                    } catch (e) {
                      showError(t('操作失败'));
                    } finally {
                      setOverrideSubmitting(false);
                    }
                  }}
                  loading={overrideSubmitting}
                >
                  {t('清除覆盖')}
                </Button>
                <Button type='primary' loading={overrideSubmitting} onClick={submitOverride}>
                  {t('保存')}
                </Button>
              </div>
            }
          >
            <div className='flex flex-col gap-3'>
              <Banner type='info' fullWidth closeIcon={null} className='!rounded-lg' description={t('用于模型广场等处的展示；填 0 或「不覆盖」表示使用实测/默认。')} />
              <div>
                <Typography.Text strong>{t('展示耗时（毫秒）')}</Typography.Text>
                <InputNumber
                  value={overrideMs}
                  onChange={(v) =>
                    setOverrideMs(v == null || Number.isNaN(v) ? 0 : v)
                  }
                  min={0}
                  className='w-full mt-1'
                />
              </div>
              <div>
                <Typography.Text strong>{t('稳定性等级')}</Typography.Text>
                <Select
                  value={overrideGrade}
                  onChange={setOverrideGrade}
                  className='w-full mt-1'
                  optionList={[
                    { value: 0, label: t('不覆盖（自动）') },
                    { value: 1, label: getStabilityGradeLabel(1, t) },
                    { value: 2, label: getStabilityGradeLabel(2, t) },
                    { value: 3, label: getStabilityGradeLabel(3, t) },
                    { value: 4, label: getStabilityGradeLabel(4, t) },
                    { value: 5, label: getStabilityGradeLabel(5, t) },
                  ]}
                />
              </div>
            </div>
          </Modal>
        </div>
      )}
    </Modal>
  );
};

export default ModelTestModal;
