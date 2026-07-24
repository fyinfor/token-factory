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

import {
  CreditCard,
  LayoutDashboard,
  Server,
  Shapes,
  Shield,
} from 'lucide-react';

const settingItem = (key, group, section, label, keywords = []) => ({
  key,
  group,
  section,
  label,
  keywords,
});

const settingPage = (key, label, items, keywords = []) => ({
  key,
  label,
  items,
  keywords,
});

export const SETTING_CATEGORIES = [
  {
    key: 'site',
    label: '站点与界面',
    icon: LayoutDashboard,
    pages: [
      settingPage('dashboard', '仪表盘设置', [
        settingItem('panels', 'dashboard', 'panels', '仪表盘内容配置', [
          '公告',
          'API信息',
          '常见问答',
          'Uptime Kuma',
          '数据看板',
        ]),
        settingItem('home-hero', 'dashboard', 'home-hero', '首页沉浸主轮播', [
          '轮播图',
          '首页主图',
        ]),
        settingItem('banner', 'dashboard', 'banner', '首页广告设置', [
          '广告',
          'Banner',
        ]),
      ]),
      settingPage('navigation', '导航与模块', [
        settingItem('header', 'operation', 'header', '顶栏管理', [
          '顶部导航',
          '菜单',
        ]),
        settingItem('sidebar', 'operation', 'sidebar', '侧边栏管理', [
          '左侧菜单',
          '全局控制',
        ]),
      ]),
      settingPage('tools', '工具与绘图', [
        settingItem('connections', 'chats', 'connections', '工具连接设置', [
          '聊天应用',
          'Chat',
        ]),
        settingItem('drawing', 'drawing', 'drawing', '绘图设置', [
          'Midjourney',
          '图片生成',
        ]),
      ]),
      settingPage('content', '内容与品牌', [
        settingItem('general', 'other', 'general', '公告与协议', [
          '公告',
          '用户协议',
          '隐私政策',
          '默认语言',
        ]),
        settingItem('appearance', 'other', 'appearance', '品牌与外观', [
          '系统名称',
          'Logo',
          '首页内容',
          '关于',
          '页脚',
        ]),
        settingItem('docs', 'other', 'docs', '文档配置', ['文档站', 'Docs']),
        settingItem('material', 'other', 'material', '素材设置', [
          '素材库',
          'Seedance',
        ]),
      ]),
    ],
  },
  {
    key: 'operation',
    label: '运营与支付',
    icon: CreditCard,
    pages: [
      settingPage('strategy', '运营设置', [
        settingItem('general', 'operation', 'general', '通用设置', [
          '注册',
          '额度展示',
          '重试',
          '演示站点',
        ]),
        settingItem('credit', 'operation', 'credit', '额度设置', [
          '新用户额度',
          '邀请额度',
          '预消费',
        ]),
        settingItem('distributor', 'operation', 'distributor', '代理设置', [
          '分销',
          '佣金',
          '提现',
        ]),
        settingItem('checkin', 'operation', 'checkin', '签到设置', [
          '签到奖励',
        ]),
      ]),
      settingPage('payment', '支付设置', [
        settingItem('general', 'payment', 'general', '基础设置', [
          '服务器地址',
          '支付回调地址',
        ]),
        settingItem('gateway', 'payment', 'gateway', '易支付 / Yipay', [
          '易支付',
          'Yipay',
          '充值价格',
          '充值方式',
        ]),
        settingItem('stripe', 'payment', 'stripe', 'Stripe 设置', [
          'Stripe',
          'Webhook',
          '促销码',
        ]),
        settingItem('creem', 'payment', 'creem', 'Creem 设置', [
          'Creem',
          '产品配置',
        ]),
        settingItem('waffo', 'payment', 'waffo', 'Waffo 设置', [
          'Waffo',
          '沙盒',
          '支付方式',
        ]),
        settingItem('ucoin', 'payment', 'ucoin', 'U币支付设置', [
          'U币',
          '加密货币',
          '币种',
        ]),
      ]),
    ],
  },
  {
    key: 'model',
    label: '模型与计费',
    icon: Shapes,
    pages: [
      settingPage('pricing', '分组与模型定价设置', [
        settingItem('visual', 'ratio', 'visual', '价格设置', [
          '可视化价格',
          '模型价格',
        ]),
        settingItem('model', 'ratio', 'model', '模型倍率设置', [
          '倍率',
          '缓存倍率',
          '补全倍率',
        ]),
        settingItem('group', 'ratio', 'group', '分组相关设置', [
          '分组倍率',
          '用户分组',
        ]),
        settingItem('unset-models', 'ratio', 'unset_models', '未设置价格模型', [
          '漏价模型',
        ]),
        settingItem(
          'tier-templates',
          'ratio',
          'request_tier_templates',
          '阶梯计费模板',
          ['阶梯价格', '请求分层计费'],
        ),
        settingItem('upstream-sync', 'ratio', 'upstream_sync', '上游倍率同步', [
          '同步价格',
          'models.dev',
        ]),
      ]),
      settingPage('models', '模型相关设置', [
        settingItem('global', 'models', 'global', '全局设置', [
          '透传请求',
          '思考模型',
          'Responses',
        ]),
        settingItem('gemini', 'models', 'gemini', 'Gemini 设置', [
          'Google',
          'Gemini',
          '安全设置',
        ]),
        settingItem('claude', 'models', 'claude', 'Claude 设置', [
          'Anthropic',
          'Claude',
        ]),
        settingItem('grok', 'models', 'grok', 'Grok 设置', ['xAI', 'Grok']),
        settingItem(
          'deployment',
          'model-deployment',
          'deployment',
          '模型部署设置',
          ['io.net', 'GPU'],
        ),
      ]),
      settingPage('routing', '速率限制设置', [
        settingItem('affinity', 'models', 'affinity', '渠道亲和性', [
          '路由',
          '粘性',
          '亲和',
        ]),
        settingItem(
          'request-rate',
          'ratelimit',
          'request',
          '模型请求速率限制',
          ['RPM', '请求限速'],
        ),
      ]),
    ],
  },
  {
    key: 'security',
    label: '安全与身份',
    icon: Shield,
    pages: [
      settingPage('protection', '安全防护', [
        settingItem('sensitive', 'operation', 'sensitive', '屏蔽词过滤设置', [
          '敏感词',
          '阿里云内容安全',
        ]),
        settingItem('ssrf', 'system', 'ssrf', 'SSRF 防护设置', [
          '私有IP',
          '域名白名单',
          '端口',
        ]),
        settingItem('api-limits', 'api-rate-limit', 'limits', '接口限流设置', [
          'API限流',
          '全局限流',
        ]),
        settingItem(
          'temporary-blacklist',
          'api-rate-limit',
          'blacklist',
          '临时黑名单',
          ['封禁', 'IP黑名单'],
        ),
      ]),
      settingPage('identity', '身份与验证', [
        settingItem('real-name', 'operation', 'real-name', '实名认证', [
          '阿里云实名认证',
          '实名奖励',
        ]),
        settingItem('login', 'system', 'login', '登录注册设置', [
          '密码登录',
          '邮箱验证',
          '注册',
        ]),
        settingItem('sms', 'system', 'sms', '短信配置', [
          '阿里云短信',
          '手机验证',
        ]),
        settingItem('passkey', 'system', 'passkey', 'Passkey 配置', [
          'WebAuthn',
          '通行密钥',
        ]),
        settingItem(
          'email-domain',
          'system',
          'email-domain',
          '邮箱域名白名单',
          ['邮箱后缀', 'Email domain'],
        ),
        settingItem('smtp', 'system', 'smtp', 'SMTP 配置', [
          '邮件',
          '邮箱服务器',
        ]),
        settingItem('turnstile', 'system', 'turnstile', 'Turnstile 配置', [
          'Cloudflare',
          '人机验证',
          '验证码',
        ]),
      ]),
      settingPage('oauth', '第三方登录', [
        settingItem('oidc', 'system', 'oidc', 'OIDC 配置', ['OpenID Connect']),
        settingItem('github', 'system', 'github', 'GitHub OAuth', ['GitHub']),
        settingItem('discord', 'system', 'discord', 'Discord OAuth', [
          'Discord',
        ]),
        settingItem('linuxdo', 'system', 'linuxdo', 'Linux DO OAuth', [
          'LinuxDO',
        ]),
        settingItem('custom-oauth', 'system', 'custom-oauth', '自定义 OAuth', [
          'OAuth提供商',
        ]),
        settingItem('wechat', 'system', 'wechat', 'WeChat 登录', [
          '微信',
          'WeChat Server',
        ]),
        settingItem('telegram', 'system', 'telegram', 'Telegram 登录', [
          'Telegram Bot',
        ]),
      ]),
    ],
  },
  {
    key: 'system',
    label: '系统运维',
    icon: Server,
    pages: [
      settingPage('infrastructure', '系统设置', [
        settingItem('general', 'system', 'general', '基础设置', ['服务器地址']),
        settingItem('proxy', 'system', 'proxy', '代理设置', [
          'Worker',
          '图片代理',
          'Webhook代理',
        ]),
        settingItem('oss', 'system', 'oss', '文件上传配置', [
          'OSS',
          '对象存储',
          '上传',
        ]),
      ]),
      settingPage('performance', '性能设置', [
        settingItem('monitoring', 'operation', 'monitoring', '渠道监控设置', [
          '自动禁用渠道',
          '自动测试',
          '余额预警',
        ]),
        settingItem('cache', 'performance', 'cache', '磁盘缓存设置', [
          '磁盘换内存',
          '缓存路径',
        ]),
        settingItem('guard', 'performance', 'guard', '系统性能保护', [
          '性能保护',
          '系统负载',
        ]),
        settingItem('stats', 'performance', 'stats', '性能监控', [
          '统计',
          '运行指标',
        ]),
      ]),
      settingPage('logs', '日志设置', [
        settingItem('business', 'operation', 'log', '业务日志设置', [
          '消费日志',
        ]),
        settingItem('server', 'performance', 'logs', '服务器日志管理', [
          '服务端日志',
          '日志文件',
        ]),
      ]),
      settingPage('about', '系统信息', [
        settingItem('info', 'other', 'system-info', '系统信息', [
          '当前版本',
          '启动时间',
          '检查更新',
        ]),
        settingItem('changelog', 'other', 'changelog', '更新日志', [
          '版本记录',
          '发布说明',
        ]),
      ]),
    ],
  },
];

export const DEFAULT_SETTING_CATEGORY = SETTING_CATEGORIES[0];
export const DEFAULT_SETTING_PAGE = DEFAULT_SETTING_CATEGORY.pages[0];
export const DEFAULT_SETTING_ITEM = DEFAULT_SETTING_PAGE.items[0];

const LEGACY_DEFAULT_SECTIONS = {
  operation: 'general',
  dashboard: 'panels',
  chats: 'connections',
  drawing: 'drawing',
  payment: 'general',
  ratio: 'visual',
  ratelimit: 'request',
  models: 'global',
  'model-deployment': 'deployment',
  performance: 'cache',
  'api-rate-limit': 'limits',
  system: 'general',
  other: 'system-info',
};

const findSettingSelectionByGroup = (group, section) => {
  const targetSection = section || LEGACY_DEFAULT_SECTIONS[group];
  const defaultSection = LEGACY_DEFAULT_SECTIONS[group];
  let groupFallback = null;
  let defaultSelection = null;

  for (const category of SETTING_CATEGORIES) {
    for (const page of category.pages) {
      for (const item of page.items) {
        if (item.group !== group) continue;
        if (!groupFallback) {
          groupFallback = { category, page, item };
        }
        if (
          !defaultSelection &&
          (item.section === defaultSection || item.key === defaultSection)
        ) {
          defaultSelection = { category, page, item };
        }
        if (item.section === targetSection || item.key === targetSection) {
          return { category, page, item };
        }
      }
    }
  }

  return defaultSelection || groupFallback;
};

export const getSettingSelection = (search) => {
  const params = new URLSearchParams(search);
  const legacySelection = findSettingSelectionByGroup(
    params.get('tab'),
    params.get('section'),
  );
  const category =
    SETTING_CATEGORIES.find((entry) => entry.key === params.get('category')) ||
    legacySelection?.category ||
    DEFAULT_SETTING_CATEGORY;
  const page =
    category.pages.find((entry) => entry.key === params.get('page')) ||
    (legacySelection?.category === category ? legacySelection.page : null) ||
    category.pages[0];
  const item =
    page.items.find((entry) => entry.key === params.get('item')) ||
    (legacySelection?.page === page ? legacySelection.item : null) ||
    page.items[0];

  return { category, page, item };
};

export const getSettingUrl = (categoryKey, pageKey, itemKey) => {
  const category =
    SETTING_CATEGORIES.find((entry) => entry.key === categoryKey) ||
    DEFAULT_SETTING_CATEGORY;
  const page =
    category.pages.find((entry) => entry.key === pageKey) || category.pages[0];
  const item =
    page.items.find((entry) => entry.key === itemKey) || page.items[0];

  return `/console/setting?category=${category.key}&page=${page.key}&item=${item.key}`;
};

export const SETTING_SEARCH_ITEMS = SETTING_CATEGORIES.flatMap((category) =>
  category.pages.flatMap((page) =>
    page.items.map((item) => ({
      ...item,
      categoryKey: category.key,
      categoryLabel: category.label,
      pageKey: page.key,
      pageLabel: page.label,
      url: getSettingUrl(category.key, page.key, item.key),
    })),
  ),
);
