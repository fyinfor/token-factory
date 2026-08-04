import { normalizeLanguage } from '../i18n/language';

export const VIDEO_ENDPOINT_TYPES = new Set([
  'openai-video',
  'hidream-video',
  'tokenfactory-video',
  'videogenerator',
  'tencentcloud-vod-video',
  'ali-video',
]);

const TRANSLATABLE_MODEL_TAGS = new Set([
  '文本',
  '视频',
  '图片',
  '多模态',
  '热门',
  'ASR',
]);

/** 按界面语言选择中文/英文内容（英文缺失时回退中文） */
export function getLocalizedContent(zhContent, enContent, language) {
  const zh = String(zhContent ?? '').trim();
  const en = String(enContent ?? '').trim();
  const lang = normalizeLanguage(language);
  if (lang === 'zh-CN' || lang === 'zh-TW') {
    return zh;
  }
  return en || zh;
}

/** 模型描述：中文界面用 description，其他语言优先 description_en */
export function getModelDescription(record, language) {
  return getLocalizedContent(record?.description, record?.description_en, language);
}

const TRANSLATABLE_SUPPLIER_TYPES = new Set([
  '公有云',
  'AIDC',
  '企业中转站',
  '个人中转站',
]);

/** 模型标签展示文案：内置标签（文本/视频/图片/多模态/热门）走 i18n，其余原样返回 */
export function getModelTagLabel(tag, t) {
  const trimmed = String(tag ?? '').trim();
  if (!trimmed) return trimmed;
  if (TRANSLATABLE_MODEL_TAGS.has(trimmed)) {
    return t(trimmed);
  }
  return trimmed;
}

/** 供应商类型展示文案：内置枚举值走 i18n，其余原样返回 */
export function getSupplierTypeLabel(supplierType, t) {
  const trimmed = String(supplierType ?? '').trim();
  if (!trimmed) return trimmed;
  if (TRANSLATABLE_SUPPLIER_TYPES.has(trimmed) && t) {
    return t(trimmed);
  }
  return trimmed;
}

export const hasNumericValue = (value) =>
  value != null && value !== '' && Number.isFinite(Number(value));

export const isVideoPricingModel = (model) => {
  if (!model) return false;
  const endpointTypes = Array.isArray(model.supported_endpoint_types)
    ? model.supported_endpoint_types
    : [];
  return (
    endpointTypes.some((type) => VIDEO_ENDPOINT_TYPES.has(type)) ||
    hasNumericValue(model.video_ratio) ||
    hasNumericValue(model.video_completion_ratio) ||
    hasNumericValue(model.video_price) ||
    !!model.video_flat_clip_hint
  );
};

/** ASR 语音识别定价模型：配置了按秒单价（asr_price，美元/秒） */
export const isASRPricingModel = (model) => {
  if (!model) return false;
  return hasNumericValue(model.asr_price);
};
