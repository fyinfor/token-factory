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

import React, { useState, useEffect, useCallback, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Modal,
  Steps,
  Button,
  Space,
  Tag,
  Spin,
  Typography,
  Banner,
  Checkbox,
  CheckboxGroup,
  Divider,
  Toast,
  Avatar,
  Card,
} from '@douyinfe/semi-ui';
import {
  IconTickCircle,
  IconAlertTriangle,
  IconClose,
  IconRefresh,
  IconExternalOpen,
} from '@douyinfe/semi-icons';
import { API, showError, showSuccess } from '../../../../helpers';
import { getChannelIcon } from '../../../../helpers';
import { MODEL_FETCHABLE_CHANNEL_TYPES } from '../../../../constants';

const { Text, Title } = Typography;
const { Step } = Steps;

// ────────────────────────────────────────────────────────────
// 小工具
// ────────────────────────────────────────────────────────────

const StatusRow = ({ label, ok, count, total }) => (
  <div className='flex items-center justify-between py-1'>
    <Text size='small'>{label}</Text>
    <Space spacing={4}>
      {ok ? (
        <Tag color='green' size='small' shape='circle'>
          ✓ {count ?? '—'}
        </Tag>
      ) : (
        <Tag color='red' size='small' shape='circle'>
          ✗ {count ?? '—'}{total != null ? `/${total}` : ''}
        </Tag>
      )}
    </Space>
  </div>
);

// ────────────────────────────────────────────────────────────
// Step 1 — 模型导入
// ────────────────────────────────────────────────────────────

// ────────────────────────────────────────────────────────────
// AutoMeta 推断结果展示
// ────────────────────────────────────────────────────────────

const SOURCE_LABEL = {
  official: { color: 'green', label: '官方预设' },
  inferred: { color: 'blue', label: '名称推断' },
  exists:   { color: 'grey', label: '已存在' },
};

const AutoMetaResult = ({ items, t }) => {
  const created = items.filter((i) => i.source !== 'exists' && !i.err);
  const skipped = items.filter((i) => i.source === 'exists');
  const failed  = items.filter((i) => i.err);

  return (
    <div className='mt-3 p-3 rounded-xl' style={{ background: 'var(--semi-color-fill-0)' }}>
      <div className='flex gap-3 mb-2 text-sm'>
        <Text type='success' size='small'>✓ {t('已创建')} {created.length}</Text>
        <Text type='tertiary' size='small'>— {t('已跳过')} {skipped.length}</Text>
        {failed.length > 0 && <Text type='danger' size='small'>✗ {t('失败')} {failed.length}</Text>}
      </div>
      {created.length > 0 && (
        <div className='flex flex-wrap gap-1 max-h-28 overflow-y-auto'>
          {created.map((item) => {
            const src = SOURCE_LABEL[item.source] ?? SOURCE_LABEL.inferred;
            return (
              <Tag key={item.model_name} size='small' color={src.color} shape='circle'>
                {item.model_name}
                <span className='opacity-60 ml-1 text-xs'>({src.label})</span>
              </Tag>
            );
          })}
        </div>
      )}
      {failed.length > 0 && (
        <div className='mt-1'>
          {failed.map((item) => (
            <Text key={item.model_name} type='danger' size='small' className='block'>
              {item.model_name}: {item.err}
            </Text>
          ))}
        </div>
      )}
    </div>
  );
};

const StepImport = ({ channel, onboardData, reloadOnboard, t }) => {
  const [selected, setSelected] = useState([]);
  const [saving, setSaving] = useState(false);
  const [autoMetaLoading, setAutoMetaLoading] = useState(false);
  const [autoMetaItems, setAutoMetaItems] = useState(null); // null = not run yet

  const imported = onboardData?.models_imported ?? [];
  const available = onboardData?.models_available ?? [];
  const metaLinked = onboardData?.meta_linked ?? [];
  const metaMissing = onboardData?.meta_missing ?? [];

  // 上游中尚未导入的模型
  const notYetImported = available.filter((m) => !imported.includes(m));

  const handleSave = async () => {
    if (selected.length === 0) return;
    setSaving(true);
    try {
      const merged = Array.from(new Set([...imported, ...selected]));
      const res = await API.patch(`/api/channel/${channel.id}/models`, {
        models: merged,
      });
      if (res?.data?.success) {
        showSuccess(t('模型导入成功'));
        setSelected([]);
        await reloadOnboard();
      } else {
        showError(res?.data?.message || t('导入失败'));
      }
    } catch (e) {
      showError(String(e));
    } finally {
      setSaving(false);
    }
  };

  // 自动推断并创建 model_meta
  const handleAutoMeta = async () => {
    setAutoMetaLoading(true);
    setAutoMetaItems(null);
    try {
      const res = await API.post(`/api/channel/${channel.id}/onboard/auto_meta`, {
        models: metaMissing,
      });
      if (res?.data?.success) {
        const { items, created } = res.data.data;
        setAutoMetaItems(items ?? []);
        if (created > 0) {
          showSuccess(t('成功创建 {{n}} 个模型元数据', { n: created }));
          await reloadOnboard();
        } else {
          showSuccess(t('元数据已是最新，无需创建'));
        }
      } else {
        showError(res?.data?.message || t('自动推断失败'));
      }
    } catch (e) {
      showError(String(e));
    } finally {
      setAutoMetaLoading(false);
    }
  };

  const canFetch = MODEL_FETCHABLE_CHANNEL_TYPES?.has
    ? MODEL_FETCHABLE_CHANNEL_TYPES.has(channel?.type)
    : false;

  return (
    <div className='space-y-4'>
      {/* 当前已导入模型汇总 */}
      <Card className='!rounded-xl shadow-sm border-0' bodyStyle={{ padding: '12px 16px' }}>
        <div className='flex items-center gap-2 mb-2'>
          <Avatar size='extra-small' color='blue' shape='square'>
            <span className='text-xs'>✓</span>
          </Avatar>
          <Text strong>{t('已导入模型')}（{imported.length}）</Text>
        </div>
        {imported.length === 0 ? (
          <Text type='tertiary' size='small'>{t('当前渠道尚未配置模型')}</Text>
        ) : (
          <div className='flex flex-wrap gap-1 mt-1 max-h-32 overflow-y-auto'>
            {imported.map((m) => (
              <Tag
                key={m}
                size='small'
                color={metaLinked.includes(m) ? 'green' : 'orange'}
                shape='circle'
              >
                {m}
              </Tag>
            ))}
          </div>
        )}

        {/* 元数据缺失：自动推断入口 */}
        {metaMissing.length > 0 && (
          <div className='mt-3 p-3 rounded-lg' style={{ background: 'var(--semi-color-warning-light-default)', border: '1px solid var(--semi-color-warning-light-active)' }}>
            <div className='flex items-start justify-between gap-2'>
              <div className='flex-1'>
                <Text size='small' strong>{t('{{n}} 个模型缺少元数据配置', { n: metaMissing.length })}</Text>
                <Text type='tertiary' size='small' className='block mt-0.5'>
                  {t('可一键自动推断（优先匹配官方预设，未收录则按名称规则推断），如需精确配置请前往模型管理手动修改')}
                </Text>
                <div className='flex flex-wrap gap-1 mt-1 max-h-16 overflow-y-auto'>
                  {metaMissing.slice(0, 10).map((m) => (
                    <Tag key={m} size='small' color='orange' shape='circle'>{m}</Tag>
                  ))}
                  {metaMissing.length > 10 && (
                    <Tag size='small' color='orange' shape='circle'>…+{metaMissing.length - 10}</Tag>
                  )}
                </div>
              </div>
              <div className='flex flex-col gap-1 shrink-0'>
                <Button
                  size='small'
                  type='primary'
                  theme='solid'
                  loading={autoMetaLoading}
                  onClick={handleAutoMeta}
                >
                  {t('自动推断元数据')}
                </Button>
                <a
                  href='/console/models'
                  target='_blank'
                  rel='noreferrer'
                  className='text-center'
                >
                  <Button size='small' theme='borderless' icon={<IconExternalOpen size='small' />}>
                    {t('手动配置')}
                  </Button>
                </a>
              </div>
            </div>
            {autoMetaItems && (
              <AutoMetaResult items={autoMetaItems} t={t} />
            )}
          </div>
        )}
        {metaMissing.length === 0 && imported.length > 0 && (
          <div className='flex items-center gap-1 mt-2 text-sm' style={{ color: 'var(--semi-color-success)' }}>
            <IconTickCircle />
            <Text type='success' size='small'>{t('所有已导入模型均已配置元数据')}</Text>
          </div>
        )}
      </Card>

      {/* 上游可导入模型 */}
      {canFetch && (
        <Card className='!rounded-xl shadow-sm border-0' bodyStyle={{ padding: '12px 16px' }}>
          <div className='flex items-center justify-between mb-2'>
            <div className='flex items-center gap-2'>
              <Avatar size='extra-small' color='purple' shape='square'>
                <span className='text-xs'>↓</span>
              </Avatar>
              <Text strong>{t('上游可导入模型')}（{notYetImported.length}）</Text>
            </div>
            <Button
              size='small'
              theme='borderless'
              icon={<IconRefresh />}
              onClick={reloadOnboard}
            >
              {t('刷新')}
            </Button>
          </div>

          {notYetImported.length === 0 ? (
            <div className='flex items-center gap-2 text-sm' style={{ color: 'var(--semi-color-success)' }}>
              <IconTickCircle />
              <Text type='success' size='small'>{t('所有上游模型已全部导入')}</Text>
            </div>
          ) : (
            <>
              <div className='max-h-48 overflow-y-auto'>
                <CheckboxGroup
                  value={selected}
                  onChange={setSelected}
                  direction='vertical'
                >
                  {notYetImported.map((m) => (
                    <Checkbox key={m} value={m}>
                      <Text size='small'>{m}</Text>
                    </Checkbox>
                  ))}
                </CheckboxGroup>
              </div>
              <div className='flex items-center justify-between mt-2 pt-2 border-t border-gray-100'>
                <Text type='tertiary' size='small'>
                  {t('已选 {{n}} 个', { n: selected.length })}
                </Text>
                <Space>
                  <Button
                    size='small'
                    theme='borderless'
                    onClick={() => setSelected(notYetImported)}
                  >
                    {t('全选')}
                  </Button>
                  <Button
                    size='small'
                    type='primary'
                    disabled={selected.length === 0}
                    loading={saving}
                    onClick={handleSave}
                  >
                    {t('导入选中')}（{selected.length}）
                  </Button>
                </Space>
              </div>
            </>
          )}
        </Card>
      )}

      {!canFetch && (
        <Banner
          type='info'
          closeIcon={null}
          className='!rounded-xl'
          description={t('该渠道类型不支持自动拉取上游模型，请手动编辑渠道添加模型。')}
        />
      )}
    </div>
  );
};

// ────────────────────────────────────────────────────────────
// Step 2 — 定价配置
// ────────────────────────────────────────────────────────────

const StepPricing = ({ channel, onboardData, reloadOnboard, t }) => {
  const ratioConfigured = onboardData?.ratio_configured ?? [];
  const ratioMissing = onboardData?.ratio_missing ?? [];
  const canSyncRatio = onboardData?.can_sync_ratio ?? false;

  const total = ratioConfigured.length + ratioMissing.length;
  const allOk = ratioMissing.length === 0 && total > 0;

  return (
    <div className='space-y-4'>
      {/* 状态总览 */}
      <Card className='!rounded-xl shadow-sm border-0' bodyStyle={{ padding: '12px 16px' }}>
        <div className='flex items-center gap-2 mb-3'>
          <Avatar size='extra-small' color={allOk ? 'green' : 'yellow'} shape='square'>
            <span className='text-xs'>$</span>
          </Avatar>
          <Text strong>{t('定价配置状态')}</Text>
        </div>
        <StatusRow
          label={t('已配置定价')}
          ok={ratioConfigured.length > 0}
          count={ratioConfigured.length}
          total={total}
        />
        <StatusRow
          label={t('缺少定价')}
          ok={ratioMissing.length === 0}
          count={ratioMissing.length}
          total={total}
        />

        {allOk && (
          <div className='flex items-center gap-2 mt-2 text-sm' style={{ color: 'var(--semi-color-success)' }}>
            <IconTickCircle />
            <Text type='success' size='small'>{t('所有模型均已配置定价')}</Text>
          </div>
        )}

        {ratioMissing.length > 0 && (
          <div className='mt-2'>
            <Text type='tertiary' size='small'>{t('缺少定价的模型：')}</Text>
            <div className='flex flex-wrap gap-1 mt-1 max-h-24 overflow-y-auto'>
              {ratioMissing.map((m) => (
                <Tag key={m} size='small' color='red' shape='circle'>{m}</Tag>
              ))}
            </div>
          </div>
        )}
      </Card>

      {/* 操作指引 */}
      <Card className='!rounded-xl shadow-sm border-0' bodyStyle={{ padding: '12px 16px' }}>
        <div className='flex items-center gap-2 mb-2'>
          <Avatar size='extra-small' color='cyan' shape='square'>
            <span className='text-xs'>⚡</span>
          </Avatar>
          <Text strong>{t('配置方式')}</Text>
        </div>
        <div className='space-y-2 text-sm'>
          <div className='flex items-start gap-2'>
            <span className='mt-0.5 text-purple-500 font-bold'>1.</span>
            <span>
              <Text>{t('前往')}</Text>&nbsp;
              <a
                href='/console/setting?tab=ratio'
                target='_blank'
                rel='noreferrer'
                className='underline text-blue-500'
              >
                {t('倍率设置')} <IconExternalOpen size='small' />
              </a>
              &nbsp;→&nbsp;{t('上游倍率同步')}，{t('选择此渠道或官方预设一键拉取定价')}
            </span>
          </div>
          {canSyncRatio && (
            <div className='flex items-start gap-2'>
              <span className='mt-0.5 text-purple-500 font-bold'>2.</span>
              <Text type='tertiary' size='small'>
                {t('该渠道已配置 Base URL，支持从上游直接拉取倍率配置')}
              </Text>
            </div>
          )}
          <div className='flex items-start gap-2'>
            <span className='mt-0.5 text-purple-500 font-bold'>{canSyncRatio ? '3' : '2'}.</span>
            <Text type='tertiary' size='small'>
              {t('配置完成后，点击"刷新状态"更新此页面的诊断结果')}
            </Text>
          </div>
        </div>
        <div className='mt-3 flex gap-2'>
          <Button
            size='small'
            theme='light'
            type='primary'
            icon={<IconExternalOpen />}
            onClick={() => window.open('/console/setting?tab=ratio', '_blank')}
          >
            {t('前往倍率设置')}
          </Button>
          <Button
            size='small'
            theme='borderless'
            icon={<IconRefresh />}
            onClick={reloadOnboard}
          >
            {t('刷新状态')}
          </Button>
        </div>
      </Card>
    </div>
  );
};

// ────────────────────────────────────────────────────────────
// Step 3 — 连通性测试（全量批测 + 单测）
// ────────────────────────────────────────────────────────────

const LATENCY_COLOR = (s) => {
  if (s <= 1) return 'green';
  if (s <= 3) return 'lime';
  if (s <= 5) return 'yellow';
  return 'red';
};

const StepTest = ({ channel, onboardData, t }) => {
  // results: { [modelName]: { success, time, message, testing, fromHistory, testedAt } }
  const [results, setResults] = useState({});
  const [batchRunning, setBatchRunning] = useState(false);
  const [historyLoading, setHistoryLoading] = useState(false);
  const stopRef = useRef(false);

  const imported = onboardData?.models_imported ?? [];
  const hasRatioWarning = (onboardData?.ratio_missing?.length ?? 0) > 0;

  // 打开时加载历史测试结果
  useEffect(() => {
    if (!channel?.id) return;
    let cancelled = false;
    const loadHistory = async () => {
      setHistoryLoading(true);
      try {
        const res = await API.get(`/api/channel/${channel.id}/test_results`);
        if (cancelled) return;
        const rows = res?.data?.data ?? [];
        if (rows.length > 0) {
          setResults((prev) => {
            const merged = { ...prev };
            rows.forEach((row) => {
              // 不覆盖本次会话已有的测试结果
              if (!merged[row.model_name]) {
                merged[row.model_name] = {
                  success: row.last_test_success,
                  time: (row.last_response_time ?? 0) / 1000,
                  message: row.last_test_message ?? '',
                  testing: false,
                  fromHistory: true,
                  testedAt: row.last_test_time,
                };
              }
            });
            return merged;
          });
        }
      } catch (_) {
        // 历史加载失败不影响功能，静默忽略
      } finally {
        if (!cancelled) setHistoryLoading(false);
      }
    };
    loadHistory();
    return () => { cancelled = true; };
  }, [channel?.id]);

  // 将批测 API 返回的 item 数组合并进 results state
  const applyBulkItems = (items, targetModels) => {
    setResults((prev) => {
      const next = { ...prev };
      items.forEach((item) => {
        next[item.model_name] = {
          success: !!item.success,
          time: item.time ?? 0,
          message: item.message ?? '',
          testing: false,
          fromHistory: false,
        };
      });
      // 未收到结果的目标模型（极少数情况）→ 标记失败
      (targetModels ?? []).forEach((m) => {
        if (next[m]?.testing) {
          next[m] = { success: false, time: 0, message: t('未收到测试结果'), testing: false, fromHistory: false };
        }
      });
      return next;
    });
  };

  // 测试单个模型 —— 1 次 POST，后端串行处理，规避全局限流
  const runOne = async (modelName) => {
    setResults((prev) => ({ ...prev, [modelName]: { testing: true } }));
    try {
      const res = await API.post(`/api/channel/${channel.id}/onboard/test`, { models: [modelName] });
      applyBulkItems(res?.data?.data ?? [], [modelName]);
    } catch (e) {
      setResults((prev) => ({
        ...prev,
        [modelName]: { success: false, time: 0, message: String(e), testing: false, fromHistory: false },
      }));
    }
  };

  // 全量批测 —— 单次 POST 请求，后端串行测试全部模型，彻底规避前端并发限流
  const runAll = async () => {
    if (imported.length === 0) return;
    setBatchRunning(true);
    stopRef.current = false;
    setResults(Object.fromEntries(imported.map((m) => [m, { testing: true }])));
    try {
      const res = await API.post(`/api/channel/${channel.id}/onboard/test`, { models: imported });
      applyBulkItems(res?.data?.data ?? [], imported);
    } catch (e) {
      setResults(Object.fromEntries(
        imported.map((m) => [m, { success: false, time: 0, message: String(e), testing: false, fromHistory: false }]),
      ));
    } finally {
      setBatchRunning(false);
    }
  };

  // 批测为单次后端请求，无法中途停止；保留接口兼容
  const stopBatch = () => { stopRef.current = true; };

  // 统计
  const tested = Object.entries(results).filter(([, v]) => !v.testing);
  const passCount = tested.filter(([, v]) => v.success).length;
  const failCount = tested.filter(([, v]) => !v.success).length;

  return (
    <div className='space-y-3'>
      {hasRatioWarning && (
        <Banner
          type='warning'
          closeIcon={null}
          className='!rounded-xl'
          description={t('部分模型尚未配置定价，测试中 token 费用将使用默认值，结果仅供连通性参考。')}
        />
      )}
      {imported.length === 0 && (
        <Banner
          type='warning'
          closeIcon={null}
          className='!rounded-xl'
          description={t('当前渠道尚未导入任何模型，无法进行模型测试。')}
        />
      )}

      {imported.length > 0 && (
        <Card className='!rounded-xl shadow-sm border-0' bodyStyle={{ padding: '12px 16px' }}>
          {/* 操作栏 */}
          <div className='flex items-center justify-between mb-3'>
            <div className='flex items-center gap-2'>
              <Avatar size='extra-small' color='orange' shape='square'>
                <span className='text-xs'>⚡</span>
              </Avatar>
              <Text strong>{t('连通性测试')}</Text>
              <Text type='tertiary' size='small'>（{imported.length} {t('个模型')}）</Text>
            </div>
            <Space>
              {batchRunning ? (
                <Button size='small' type='danger' theme='light' onClick={stopBatch}>
                  {t('停止')}
                </Button>
              ) : (
                <Button
                  size='small'
                  type='primary'
                  onClick={runAll}
                  icon={<IconTickCircle />}
                >
                  {t('全部测试')}
                </Button>
              )}
            </Space>
          </div>

          {/* 统计行 */}
          {tested.length > 0 && (
            <div className='flex items-center gap-3 mb-2 pb-2 border-b border-gray-100'>
              <Text size='small' type='tertiary'>
                {historyLoading ? t('加载历史结果…') : `${t('已测试')} ${tested.length}/${imported.length}`}
              </Text>
              {passCount > 0 && (
                <Tag size='small' color='green' shape='circle'>✓ {passCount} {t('通过')}</Tag>
              )}
              {failCount > 0 && (
                <Tag size='small' color='red' shape='circle'>✗ {failCount} {t('失败')}</Tag>
              )}
              {/* 说明历史数据来源 */}
              {tested.some(([, v]) => v.fromHistory) && !batchRunning && (
                <Text size='small' type='tertiary' style={{ fontStyle: 'italic' }}>
                  {t('（含上次测试结果）')}
                </Text>
              )}
            </div>
          )}

          {/* 模型结果列表 */}
          <div className='space-y-1 max-h-72 overflow-y-auto'>
            {imported.map((m) => {
              const r = results[m];
              const isTesting = r?.testing;
              const isDone = r && !isTesting;
              const isHistory = isDone && r.fromHistory;

              // 历史时间格式化：秒级时间戳 → 本地日期时间
              const historyLabel = isHistory && r.testedAt
                ? new Date(r.testedAt * 1000).toLocaleString()
                : null;

              return (
                <div
                  key={m}
                  className='flex items-center justify-between py-1 px-2 rounded-lg hover:bg-gray-50 group'
                  style={{ minHeight: 32 }}
                >
                  {/* 模型名 */}
                  <Text size='small' className='truncate flex-1 mr-2' title={m}>
                    {m}
                  </Text>

                  {/* 结果 / 操作 */}
                  <div className='flex items-center gap-1 shrink-0'>
                    {isTesting && (
                      <Tag size='small' color='blue' shape='circle'>
                        {t('测试中…')}
                      </Tag>
                    )}
                    {isDone && r.success && (
                      <Tag
                        size='small'
                        color={LATENCY_COLOR(r.time)}
                        shape='circle'
                        title={historyLabel ? `${t('上次测试')}: ${historyLabel}` : undefined}
                      >
                        {isHistory ? '◷' : '✓'} {r.time.toFixed(2)}s
                      </Tag>
                    )}
                    {isDone && !r.success && (
                      <Tag
                        size='small'
                        color={isHistory ? 'orange' : 'red'}
                        shape='circle'
                        title={r.message + (historyLabel ? `\n${t('上次测试')}: ${historyLabel}` : '')}
                      >
                        {isHistory ? '◷' : '✗'} {t('失败')}
                      </Tag>
                    )}
                    {/* 单测按钮（hover 才显示） */}
                    {!isTesting && (
                      <Button
                        size='small'
                        theme='borderless'
                        type='tertiary'
                        className='opacity-0 group-hover:opacity-100 transition-opacity'
                        onClick={() => runOne(m)}
                      >
                        {t('测试')}
                      </Button>
                    )}
                  </div>
                </div>
              );
            })}
          </div>

          {/* 失败详情展开 */}
          {tested.some(([, v]) => !v.success) && (
            <div className='mt-2 pt-2 border-t border-gray-100'>
              <Text type='tertiary' size='small' strong>{t('失败详情：')}</Text>
              {tested
                .filter(([, v]) => !v.success)
                .slice(0, 5)
                .map(([name, v]) => (
                  <div key={name} className='mt-1'>
                    <Text size='small' type='danger' className='break-all'>
                      <strong>{name}</strong>：{v.message}
                    </Text>
                  </div>
                ))}
            </div>
          )}
        </Card>
      )}
    </div>
  );
};

// ────────────────────────────────────────────────────────────
// 主组件
// ────────────────────────────────────────────────────────────

const STEPS = ['import', 'pricing', 'test'];

const ChannelOnboardModal = ({ visible, channel, onClose, onRefresh }) => {
  const { t } = useTranslation();
  const [currentStep, setCurrentStep] = useState(0);
  const [loading, setLoading] = useState(false);
  const [onboardData, setOnboardData] = useState(null);

  const loadOnboard = useCallback(async () => {
    if (!channel?.id) return;
    setLoading(true);
    try {
      const res = await API.get(`/api/channel/${channel.id}/onboard`);
      if (res?.data?.success) {
        setOnboardData(res.data.data);
      } else {
        showError(res?.data?.message || t('获取上架状态失败'));
      }
    } catch (e) {
      showError(String(e));
    } finally {
      setLoading(false);
    }
  }, [channel?.id, t]);

  useEffect(() => {
    if (visible && channel?.id) {
      setCurrentStep(0);
      setOnboardData(null);
      loadOnboard();
    }
  }, [visible, channel?.id]);

  const handleClose = () => {
    onRefresh?.();
    onClose?.();
  };

  const stepStatus = (idx) => {
    if (!onboardData) return 'wait';
    if (idx === 0) {
      const ok = (onboardData.models_imported?.length ?? 0) > 0;
      return ok ? 'finish' : 'error';
    }
    if (idx === 1) {
      const missing = onboardData.ratio_missing?.length ?? 0;
      const configured = onboardData.ratio_configured?.length ?? 0;
      if (missing === 0 && configured > 0) return 'finish';
      if (configured > 0) return 'process'; // 部分已配置
      return 'error';
    }
    return 'wait';
  };

  const channelName = channel?.name ?? '渠道';
  const channelType = channel?.type;

  return (
    <Modal
      title={
        <div className='flex items-center gap-3'>
          {channelType != null && (
            <span className='text-xl'>{getChannelIcon(channelType)}</span>
          )}
          <div>
            <Title heading={5} className='m-0'>
              {t('上架向导')} — {channelName}
            </Title>
            <Text type='tertiary' size='small'>
              {t('引导完成模型导入、定价配置、连通性测试三个上架步骤')}
            </Text>
          </div>
        </div>
      }
      visible={visible}
      onCancel={handleClose}
      width={620}
      style={{ maxWidth: '95vw' }}
      footer={
        <div className='flex justify-between items-center'>
          <Button
            type='tertiary'
            disabled={currentStep === 0}
            onClick={() => setCurrentStep((s) => s - 1)}
          >
            {t('上一步')}
          </Button>
          <Space>
            <Button onClick={handleClose}>{t('完成')}</Button>
            {currentStep < STEPS.length - 1 && (
              <Button
                type='primary'
                onClick={() => setCurrentStep((s) => s + 1)}
              >
                {t('下一步')}
              </Button>
            )}
          </Space>
        </div>
      }
      closeIcon={<IconClose />}
    >
      {/* Steps header */}
      <Steps current={currentStep} className='mb-4' onChange={setCurrentStep} size='small'>
        <Step
          title={t('导入模型')}
          description={
            onboardData
              ? `${onboardData.models_imported?.length ?? 0} ${t('个已导入')}`
              : ''
          }
          status={currentStep === 0 ? 'process' : stepStatus(0)}
        />
        <Step
          title={t('定价配置')}
          description={
            onboardData
              ? onboardData.ratio_missing?.length === 0
                ? t('已完成')
                : `${onboardData.ratio_missing?.length ?? 0} ${t('个缺失')}`
              : ''
          }
          status={currentStep === 1 ? 'process' : stepStatus(1)}
        />
        <Step
          title={t('连通性测试')}
          description={t('验证渠道可用性')}
          status={currentStep === 2 ? 'process' : 'wait'}
        />
      </Steps>

      <Divider margin='12px' />

      {/* Step content */}
      <Spin spinning={loading}>
        <div style={{ minHeight: 200 }}>
          {!loading && onboardData && (
            <>
              {currentStep === 0 && (
                <StepImport
                  channel={channel}
                  onboardData={onboardData}
                  reloadOnboard={loadOnboard}
                  t={t}
                />
              )}
              {currentStep === 1 && (
                <StepPricing
                  channel={channel}
                  onboardData={onboardData}
                  reloadOnboard={loadOnboard}
                  t={t}
                />
              )}
              {currentStep === 2 && (
                <StepTest
                  channel={channel}
                  onboardData={onboardData}
                  t={t}
                />
              )}
            </>
          )}
          {!loading && !onboardData && (
            <div className='flex flex-col items-center justify-center h-40 gap-2'>
              <Text type='tertiary'>{t('诊断数据加载失败')}</Text>
              <Button size='small' icon={<IconRefresh />} onClick={loadOnboard}>
                {t('重试')}
              </Button>
            </div>
          )}
        </div>
      </Spin>

      {/* Warnings */}
      {onboardData?.warnings?.length > 0 && (
        <div className='mt-3'>
          {onboardData.warnings.map((w, i) => (
            <Banner
              key={i}
              type='warning'
              closeIcon={null}
              className='!rounded-lg mb-1'
              description={<Text size='small'>{w}</Text>}
            />
          ))}
        </div>
      )}
    </Modal>
  );
};

export default ChannelOnboardModal;
