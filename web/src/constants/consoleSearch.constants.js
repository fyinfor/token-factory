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

const consoleSearchItem = (
  key,
  label,
  url,
  sectionLabel,
  keywords = [],
  parentLabel = null,
) => ({
  id: `console:${key}`,
  type: 'console',
  label,
  url,
  pathLabels: ['控制台', sectionLabel, parentLabel].filter(Boolean),
  keywords,
});

export const CONSOLE_SEARCH_ITEMS = [
  consoleSearchItem('dashboard', '数据看板', '/console', '工作区', [
    '仪表盘',
    '概览',
    '统计',
    'Dashboard',
  ]),
  consoleSearchItem('playground', '操练场', '/console/playground', '聊天', [
    'Playground',
    '接口调试',
    'API 测试',
    '模型测试',
  ]),
  consoleSearchItem('token', '令牌管理', '/console/token', '工作区', [
    'API Key',
    '密钥',
    'Token',
    '访问令牌',
  ]),
  consoleSearchItem('log', '使用日志', '/console/log', '工作区', [
    '请求日志',
    '消费日志',
    '调用记录',
    'Usage',
  ]),
  consoleSearchItem('midjourney', '绘图日志', '/console/midjourney', '工作区', [
    '绘图',
    '图片生成',
    'Midjourney',
    'MJ',
  ]),
  consoleSearchItem('task', '任务日志', '/console/task', '工作区', [
    '异步任务',
    '任务记录',
  ]),
  consoleSearchItem('topup', '钱包管理', '/console/topup', '个人中心', [
    '充值',
    '余额',
    '支付',
    '钱包',
  ]),
  consoleSearchItem('invoice', '发票管理', '/console/invoice', '个人中心', [
    '开票',
    '发票申请',
  ]),
  consoleSearchItem('personal', '个人设置', '/console/personal', '个人中心', [
    '账号设置',
    '密码',
    '安全设置',
    '绑定',
  ]),
  consoleSearchItem(
    'route-policy',
    '智能路由',
    '/console/route-policy',
    '个人中心',
    ['路由策略', '权重', '模型路由'],
  ),
  consoleSearchItem(
    'seedance-material',
    'SD 素材库',
    '/console/seedance/material',
    '个人中心',
    [
      'Seedance',
      '素材',
      '视频素材',
      '合规素材',
      '人像素材',
      '虚拟人',
      '虚拟人像',
      '数字人',
      '真人',
      '真人人像',
      '真人素材',
      '真人认证',
      '真人实名认证',
      '真人分组',
      '角色素材',
    ],
  ),
  consoleSearchItem('channel', '渠道管理', '/console/channel', '管理员', [
    '上游渠道',
    'Provider',
    'API 渠道',
    '渠道模型',
  ]),
  consoleSearchItem(
    'aliyun-guardrail',
    '安全护栏',
    '/console/aliyun-guardrail',
    '管理员',
    ['阿里云', '内容安全', 'Guardrail', '安全策略'],
  ),
  consoleSearchItem(
    'subscription',
    '订阅管理',
    '/console/subscription',
    '管理员',
    ['订阅', '套餐', '周期额度'],
  ),
  consoleSearchItem('models', '模型管理', '/console/models', '管理员', [
    '模型配置',
    '上游模型',
    '模型列表',
  ]),
  consoleSearchItem('deployment', '模型部署', '/console/deployment', '管理员', [
    'io.net',
    'GPU',
    '算力',
    '部署服务',
  ]),
  consoleSearchItem('model-heat', '热度配置', '/console/model-heat', '管理员', [
    '模型热度',
    '推荐',
    '排序',
    '热门模型',
  ]),
  consoleSearchItem(
    'redemption',
    '兑换码管理',
    '/console/redemption',
    '管理员',
    ['兑换码', '礼品码', '充值码'],
  ),
  consoleSearchItem('user', '用户管理', '/console/user', '管理员', [
    '用户',
    '账号',
    '额度',
    '封禁',
    '用户分组',
  ]),
  consoleSearchItem(
    'invoice-admin',
    '发票审批',
    '/console/invoice-admin',
    '管理员',
    ['开票审批', '充值订单', '发票申请'],
  ),
  consoleSearchItem(
    'invoice-feature-toggle',
    '发票功能开关',
    '/console/setting?category=site&page=navigation&item=sidebar',
    '系统设置',
    ['发票管理开关', '发票审批开关', '开票功能', '侧边栏管理'],
    '导航与模块',
  ),
  consoleSearchItem(
    'settlement-export',
    '结算单导出',
    '/console/settlement-export',
    '管理员',
    ['结算', '对账', '导出', '账单'],
  ),
  consoleSearchItem(
    'distributor',
    '代理管理',
    '/console/distributor/admin',
    '管理员',
    ['代理', '分销', '佣金', '提现'],
  ),
  consoleSearchItem(
    'supplier-application',
    '申请审批',
    '/console/supplier-application',
    '管理员',
    ['供应商申请', '供应商审批', '入驻审批'],
    '供应商管理',
  ),
  consoleSearchItem(
    'suppliers',
    '供应商列表',
    '/console/suppliers',
    '管理员',
    ['供应商', '供应商账号', '供应商管理'],
    '供应商管理',
  ),
  consoleSearchItem(
    'supplier-dashboard',
    '供应商数据看板',
    '/console/supplier/dashboard',
    '管理员',
    ['供应商统计', '供应商模型', '供应商数据'],
    '供应商管理',
  ),
  consoleSearchItem('setting', '系统设置', '/console/setting', '管理员', [
    '后台设置',
    '系统配置',
    '全局设置',
  ]),
];

export const QUICK_SEARCH_FEATURED_IDS = [
  'console:dashboard',
  'console:channel',
  'console:user',
  'console:models',
  'console:log',
  'setting:system:infrastructure:general',
];
