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

export const MESSAGE_STATUS = {
  LOADING: 'loading',
  INCOMPLETE: 'incomplete',
  COMPLETE: 'complete',
  ERROR: 'error',
};

export const MESSAGE_ROLES = {
  USER: 'user',
  ASSISTANT: 'assistant',
  SYSTEM: 'system',
};

/** 操练场图片模式参考图 URL 最多条数 */
export const PLAYGROUND_MEDIA_MAX_COUNT = 3;
/** 操练场视频首尾帧最多张数 */
export const PLAYGROUND_VIDEO_FRAME_MAX_COUNT = 2;
/** 操练场视频参考图 / 视频 / 音频：不限制数量 */
export const PLAYGROUND_MEDIA_UNLIMITED_COUNT = Number.POSITIVE_INFINITY;
/** 操练场视频图片地址 Tab：参考图 / 首尾帧互斥 */
export const PLAYGROUND_VIDEO_IMAGE_TABS = {
  REFERENCE: 'reference',
  FRAMES: 'frames',
};

// 默认消息示例 - 使用函数生成以支持 i18n
export const getDefaultMessages = (t) => [
  {
    role: MESSAGE_ROLES.USER,
    id: '2',
    createAt: 1715676751919,
    content: t('默认用户消息'),
  },
  {
    role: MESSAGE_ROLES.ASSISTANT,
    id: '3',
    createAt: 1715676751919,
    content: t('默认助手消息'),
    reasoningContent: '',
    isReasoningExpanded: false,
  },
];

// 保留旧的导出以保持向后兼容
export const DEFAULT_MESSAGES = [
  {
    role: MESSAGE_ROLES.USER,
    id: '2',
    createAt: 1715676751919,
    content: 'Hello',
  },
  {
    role: MESSAGE_ROLES.ASSISTANT,
    id: '3',
    createAt: 1715676751919,
    content: 'Hello! How can I help you today?',
    reasoningContent: '',
    isReasoningExpanded: false,
  },
];

// ========== UI 相关常量 ==========
export const DEBUG_TABS = {
  PREVIEW: 'preview',
  REQUEST: 'request',
  RESPONSE: 'response',
};

// ========== API 相关常量 ==========
export const API_ENDPOINTS = {
  CHAT_COMPLETIONS: '/api/playground/chat/completions',
  IMAGE_GENERATIONS: '/api/playground/images/generations',
  IMAGE_EDITS: '/api/playground/images/edits',
  IMAGE_GENERATIONS_FETCH_PREFIX: '/api/playground/images/generations',
  VIDEO_GENERATIONS: '/api/playground/videos',
  USER_MODELS: '/api/user/models',
  USER_PLAYGROUND_VIDEO_PRICING_TIERS:
    '/api/user/playground/video-pricing-tiers',
  USER_PLAYGROUND_IMAGE_PRICING_TIERS:
    '/api/user/playground/image-pricing-tiers',
  USER_GROUPS: '/api/user/self/groups',
};

// ========== 配置默认值 ==========
export const DEFAULT_CONFIG = {
  inputs: {
    display_mode: 'text',
    model: 'gpt-4o',
    model_type: '',
    selected_route_slug: '',
    group: '',
    temperature: 0.7,
    top_p: 1,
    max_tokens: 4096,
    frequency_penalty: 0,
    presence_penalty: 0,
    seed: null,
    stream: true,
    imageEnabled: false,
    imageUrls: [''],
    frameImageUrls: [''],
    videoImageTab: PLAYGROUND_VIDEO_IMAGE_TABS.REFERENCE,
    videoUrls: [''],
    audioUrls: [''],
    selected_model_tags: [],
    image_size: '1024x1024',
    image_ratio: '1:1',
    image_custom_size: '',
    image_n: 1,
    selected_image_pricing_tiers: [],
    image_quality: 'standard',
    image_response_format: 'url',
    image_style: 'vivid',
    video_duration: 5,
    video_resolution_preset: '720p',
    video_orientation: 'landscape',
    video_ratio: '16:9',
    selected_video_pricing_tiers: [],
    video_width: 1280,
    video_height: 720,
    video_fps: 24,
    video_motion: 0.4,
    video_n: 1,
    generate_audio: true,
    selected_channel_type: '',
  },
  parameterEnabled: {
    temperature: true,
    top_p: true,
    max_tokens: false,
    frequency_penalty: true,
    presence_penalty: true,
    seed: false,
  },
  systemPrompt: '',
  showDebugPanel: false,
  customRequestMode: false,
  customRequestBody: '',
};

// ========== 正则表达式 ==========
export const THINK_TAG_REGEX = /<think>([\s\S]*?)<\/think>/g;

// ========== 错误消息 ==========
export const ERROR_MESSAGES = {
  NO_TEXT_CONTENT: '此消息没有可复制的文本内容',
  INVALID_MESSAGE_TYPE: '无法复制此类型的消息内容',
  COPY_FAILED: '复制失败，请手动选择文本复制',
  COPY_HTTPS_REQUIRED: '复制功能需要 HTTPS 环境，请手动复制',
  BROWSER_NOT_SUPPORTED: '浏览器不支持复制功能，请手动复制',
  JSON_PARSE_ERROR: '自定义请求体格式错误，请检查JSON格式',
  API_REQUEST_ERROR: '请求发生错误',
  NETWORK_ERROR: '网络连接失败或服务器无响应',
};

/** 操练场文生视频时长（秒）：4～15，默认 5 */
export const PLAYGROUND_VIDEO_DURATION_OPTIONS = Array.from(
  { length: 12 },
  (_, i) => {
    const sec = i + 4;
    return { label: `${sec}s`, value: sec };
  },
);

/** 操练场消息内图片/视频最大宽度 */
export const PLAYGROUND_MEDIA_MAX_WIDTH = 'min(100%, 780px)';
export const PLAYGROUND_MEDIA_MAX_WIDTH_PX = 780;
/** 操练场消息内图片/视频最大高度（保持比例） */
export const PLAYGROUND_MEDIA_MAX_HEIGHT = '60vh';
export const PLAYGROUND_MEDIA_MAX_HEIGHT_RATIO = 0.6;

/** 操练场 AI 对话中用户参考图预览尺寸（较生成图更小） */
export const PLAYGROUND_DIALOGUE_REFERENCE_IMAGE_MAX_WIDTH =
  'min(100%, 240px)';
export const PLAYGROUND_DIALOGUE_REFERENCE_IMAGE_MAX_WIDTH_PX = 240;
export const PLAYGROUND_DIALOGUE_REFERENCE_IMAGE_MAX_HEIGHT = '160px';
export const PLAYGROUND_DIALOGUE_REFERENCE_IMAGE_MAX_HEIGHT_PX = 160;

export function getPlaygroundMediaMaxHeightPx() {
  if (typeof window === 'undefined') {
    return 640;
  }
  return Math.round(window.innerHeight * PLAYGROUND_MEDIA_MAX_HEIGHT_RATIO);
}

// 操练场图片分辨率：Ai 绘图档位 512P / 1K / 2K / 4K（与视频档位相互独立；value 为代表性像素）
export const PLAYGROUND_IMAGE_SIZE_OPTIONS = [
  { label: '512P', value: '512x512' },
  { label: '1K', value: '1024x1024' },
  { label: '2K', value: '2048x2048' },
  { label: '4K', value: '4096x4096' },
];

/** 视频比例选项（含 Auto） */
export const PLAYGROUND_ASPECT_RATIO_OPTIONS = [
  { label: 'Auto', value: 'auto' },
  { label: '16:9', value: '16:9' },
  { label: '4:3', value: '4:3' },
  { label: '1:1', value: '1:1' },
  { label: '3:4', value: '3:4' },
  { label: '9:16', value: '9:16' },
  { label: '21:9', value: '21:9' },
];

/** 图片比例选项（无 Auto，与 Ai 绘图尺寸表一致） */
export const PLAYGROUND_IMAGE_ASPECT_RATIO_OPTIONS = [
  { label: '1:1', value: '1:1' },
  { label: '16:9', value: '16:9' },
  { label: '9:16', value: '9:16' },
  { label: '4:3', value: '4:3' },
  { label: '3:4', value: '3:4' },
  { label: '21:9', value: '21:9' },
];

// ========== 存储键名 ==========
export const STORAGE_KEYS = {
  CONFIG: 'playground_config',
  MESSAGES: 'playground_messages',
};
