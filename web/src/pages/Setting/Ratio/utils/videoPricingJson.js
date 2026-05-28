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

const DEFAULT_TEXT_VIDEO_PIXEL_COMPRESSION = '384';
const DEFAULT_IMAGE_VIDEO_PIXEL_COMPRESSION = '512';
const DEFAULT_VIDEO_VIDEO_PIXEL_COMPRESSION = '384';

/** 简写分辨率 → 内部 canonical（与可视化下拉 value 一致） */
const RESOLUTION_TO_CANONICAL = {
  '480': '854x480',
  '480p': '854x480',
  '854x480': '854x480',
  '540': '960x540',
  '540p': '960x540',
  '960x540': '960x540',
  '720': '1280x720',
  '720p': '1280x720',
  '1280x720': '1280x720',
  '1080': '1920x1080',
  '1080p': '1920x1080',
  '1920x1080': '1920x1080',
  '2k': '2560x1440',
  '2K': '2560x1440',
  '2560x1440': '2560x1440',
  '4k': '3840x2160',
  '4K': '3840x2160',
  '3840x2160': '3840x2160',
};

/** canonical → JSON 简写键 */
const CANONICAL_TO_SHORT = {
  '854x480': '480',
  '960x540': '540',
  '1280x720': '720',
  '1920x1080': '1080',
  '2560x1440': '2k',
  '3840x2160': '4k',
};

export const VIDEO_PRICING_JSON_RESOLUTION_KEYS = [
  '480',
  '540',
  '720',
  '1080',
  '2k',
  '4k',
];

export const VIDEO_PRICING_JSON_BILLING_MODES = ['per-item', 'per-second'];
export const VIDEO_PRICING_JSON_PRICE_UNITS = ['USD', 'CNY', 'CUSTOM'];

const SECTION_META = [
  { key: 'text_to_video', comment: '文生视频' },
  { key: 'image_to_video', comment: '图生视频' },
  { key: 'video_to_video', comment: '视频生视频' },
];

const toNumericString = (value) => {
  if (value === '' || value === null || value === undefined) {
    return '';
  }
  const num = Number(value);
  return Number.isFinite(num) ? String(num) : '';
};

export const normalizeResolutionKey = (key) => {
  const raw = String(key || '').trim();
  if (!raw) return '';
  const lower = raw.toLowerCase();
  if (RESOLUTION_TO_CANONICAL[lower]) {
    return RESOLUTION_TO_CANONICAL[lower];
  }
  if (RESOLUTION_TO_CANONICAL[raw]) {
    return RESOLUTION_TO_CANONICAL[raw];
  }
  if (/^\d+$/.test(raw)) {
    const withP = `${raw}p`;
    if (RESOLUTION_TO_CANONICAL[withP]) {
      return RESOLUTION_TO_CANONICAL[withP];
    }
  }
  return raw.replace(/\s+/g, '');
};

export const resolutionToShortKey = (resolution) => {
  const canon = normalizeResolutionKey(resolution);
  return CANONICAL_TO_SHORT[canon] || canon;
};

/** 去掉 // 行注释（不处理字符串内的 //） */
export const stripJsonLineComments = (text) => {
  let result = '';
  let inString = false;
  let escape = false;
  for (let i = 0; i < text.length; i += 1) {
    const c = text[i];
    const next = text[i + 1];
    if (inString) {
      result += c;
      if (escape) {
        escape = false;
      } else if (c === '\\') {
        escape = true;
      } else if (c === '"') {
        inString = false;
      }
      continue;
    }
    if (c === '"') {
      inString = true;
      result += c;
      continue;
    }
    if (c === '/' && next === '/') {
      while (i < text.length && text[i] !== '\n') {
        i += 1;
      }
      continue;
    }
    result += c;
  }
  return result;
};

export const parseVideoPricingJsonText = (text) => {
  try {
    const stripped = stripJsonLineComments(text || '{}');
    const parsed = JSON.parse(stripped);
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      return { ok: false, message: '不是合法的 JSON 字符串' };
    }
    return { ok: true, data: parsed };
  } catch {
    return { ok: false, message: '不是合法的 JSON 字符串' };
  }
};

const toPriceNumber = (value) => {
  const str = toNumericString(value);
  if (!str) return null;
  return Number(str);
};

const serializePricesFromRow = (row, perItem) => {
  if (!row?.resolution) return null;
  const shortKey = resolutionToShortKey(row.resolution);
  if (row.audioPricingEnabled) {
    const noAudio = toPriceNumber(row.noAudioPrice);
    const withAudio = toPriceNumber(row.withAudioPrice);
    if (noAudio === null && withAudio === null) return null;
    if (
      noAudio !== null &&
      withAudio !== null &&
      Math.abs(noAudio - withAudio) < 1e-9
    ) {
      return { key: shortKey, prices: [noAudio] };
    }
    const prices = [];
    if (noAudio !== null) prices.push(noAudio);
    if (withAudio !== null) prices.push(withAudio);
    if (prices.length === 0) return null;
    if (prices.length === 1) return { key: shortKey, prices };
    return { key: shortKey, prices: [noAudio, withAudio] };
  }
  const unified = perItem
    ? toPriceNumber(row.videoPrice)
    : toPriceNumber(row.tokenPrice);
  if (unified === null) return null;
  return { key: shortKey, prices: [unified] };
};

/** 输出固定包含全部分辨率键，未配置为 [] */
const serializeSectionMapFull = (rows, perItem) => {
  const enabled = {};
  for (const row of rows || []) {
    const item = serializePricesFromRow(row, perItem);
    if (!item) continue;
    enabled[item.key] = item.prices;
  }
  const out = {};
  for (const key of VIDEO_PRICING_JSON_RESOLUTION_KEYS) {
    out[key] = enabled[key] ?? [];
  }
  return out;
};

/** 视频生视频：合并 upload / generate（与后端 video_to_video_* 一致，后者覆盖同分辨率） */
const serializeVideoToVideoSectionFull = (
  uploadRules,
  generateRules,
  perItem,
) => {
  const enabled = {};
  for (const row of uploadRules || []) {
    const item = serializePricesFromRow(row, perItem);
    if (item) enabled[item.key] = item.prices;
  }
  for (const row of generateRules || []) {
    const item = serializePricesFromRow(row, perItem);
    if (item) enabled[item.key] = item.prices;
  }
  const out = {};
  for (const key of VIDEO_PRICING_JSON_RESOLUTION_KEYS) {
    out[key] = enabled[key] ?? [];
  }
  return out;
};

/** 从当前模型状态生成简化 JSON（与可视化同展示币种） */
export const buildVideoPricingJsonEditor = (model) => {
  if (!model) {
    return {};
  }
  const perItem = model.videoBillingMode === 'per-item';
  return {
    billing_mode: model.videoBillingMode || 'per-second',
    price_unit: model.videoPriceUnit || 'USD',
    fixed_price: model.videoFixedPrice || '',
    similarity_threshold: model.videoSimilarityThreshold || '',
    text_to_video: serializeSectionMapFull(model.videoTextToVideoRules, perItem),
    image_to_video: serializeSectionMapFull(
      model.videoImageToVideoRules,
      perItem,
    ),
    video_to_video: serializeVideoToVideoSectionFull(
      model.videoUploadRules,
      model.videoGenerateRules,
      perItem,
    ),
  };
};

/** 带 // 注释的 JSON 文本（支持 // 注释后解析） */
export const stringifyVideoPricingJsonEditor = (model) => {
  const data = buildVideoPricingJsonEditor(model);
  const out = [];
  out.push('{');
  out.push('  // billing_mode: per-item（按条）| per-second（按秒）');
  out.push(`  "billing_mode": ${JSON.stringify(data.billing_mode)},`);
  out.push('  // price_unit: USD | CNY | CUSTOM');
  out.push(`  "price_unit": ${JSON.stringify(data.price_unit)},`);
  out.push('  // 无分辨率表时的兜底单价；不需要可留空 ""');
  out.push(`  "fixed_price": ${JSON.stringify(data.fixed_price)},`);
  out.push('  // 相似分辨率阈值，仅 per-second 有效，如 "0.35"');
  out.push(`  "similarity_threshold": ${JSON.stringify(data.similarity_threshold)},`);
  out.push(
    '  // 三大类：文生 / 图生 / 视频生视频；各分辨率 [] 未启用，[价] 统一，[无音轨, 有音轨] 分开',
  );

  SECTION_META.forEach((section, sectionIndex) => {
    const isLastSection = sectionIndex === SECTION_META.length - 1;
    out.push(`  // ${section.comment}`);
    out.push('  "' + section.key + '": {');
    VIDEO_PRICING_JSON_RESOLUTION_KEYS.forEach((resKey, resIndex) => {
      const prices = data[section.key]?.[resKey] ?? [];
      const resComma =
        resIndex < VIDEO_PRICING_JSON_RESOLUTION_KEYS.length - 1 ? ',' : '';
      out.push(`    "${resKey}": ${JSON.stringify(prices)}${resComma}`);
    });
    out.push(`  }${isLastSection ? '' : ','}`);
  });
  out.push('}');
  return out.join('\n');
};

const parseLegacyRow = (row, perItem, section) => {
  if (!row || typeof row !== 'object' || Array.isArray(row)) {
    return null;
  }
  const resolution = normalizeResolutionKey(row.resolution || '');
  const audioEnabled = Boolean(
    row.audio_pricing_enabled ?? row.audioPricingEnabled,
  );
  const defaultCompression =
    section === 'image'
      ? DEFAULT_IMAGE_VIDEO_PIXEL_COMPRESSION
      : section === 'text'
        ? DEFAULT_TEXT_VIDEO_PIXEL_COMPRESSION
        : DEFAULT_VIDEO_VIDEO_PIXEL_COMPRESSION;
  if (perItem) {
    if (audioEnabled) {
      return {
        resolution,
        audioPricingEnabled: true,
        videoPrice: '',
        noAudioPrice: toNumericString(row.no_audio_price ?? row.noAudioPrice),
        withAudioPrice: toNumericString(
          row.with_audio_price ?? row.withAudioPrice,
        ),
      };
    }
    return {
      resolution,
      audioPricingEnabled: false,
      videoPrice: toNumericString(row.video_price ?? row.videoPrice),
      noAudioPrice: '',
      withAudioPrice: '',
    };
  }
  if (audioEnabled) {
    return {
      resolution,
      audioPricingEnabled: true,
      tokenPrice: '',
      noAudioPrice: toNumericString(row.no_audio_price ?? row.noAudioPrice),
      withAudioPrice: toNumericString(
        row.with_audio_price ?? row.withAudioPrice,
      ),
      pixelCompression: toNumericString(
        row.pixel_compression ?? row.pixelCompression ?? defaultCompression,
      ),
    };
  }
  return {
    resolution,
    audioPricingEnabled: false,
    tokenPrice: toNumericString(row.token_price ?? row.tokenPrice),
    noAudioPrice: '',
    withAudioPrice: '',
    pixelCompression: toNumericString(
      row.pixel_compression ?? row.pixelCompression ?? defaultCompression,
    ),
  };
};

const parseLegacyRowList = (rows, perItem, section) => {
  if (!Array.isArray(rows)) {
    return [];
  }
  return rows
    .map((row) => parseLegacyRow(row, perItem, section))
    .filter((row) => row !== null && row.resolution);
};

const normalizePriceArray = (prices) => {
  if (prices === null || prices === undefined) {
    return [];
  }
  const arr = Array.isArray(prices) ? prices : [prices];
  return arr.map((item) => toPriceNumber(item)).filter((n) => n !== null);
};

const parseSectionMap = (data, perItem, section) => {
  if (Array.isArray(data)) {
    return parseLegacyRowList(data, perItem, section);
  }
  if (!data || typeof data !== 'object') {
    return [];
  }
  const defaultCompression =
    section === 'image'
      ? DEFAULT_IMAGE_VIDEO_PIXEL_COMPRESSION
      : section === 'text'
        ? DEFAULT_TEXT_VIDEO_PIXEL_COMPRESSION
        : DEFAULT_VIDEO_VIDEO_PIXEL_COMPRESSION;
  const rows = [];
  for (const [rawKey, rawPrices] of Object.entries(data)) {
    const resolution = normalizeResolutionKey(rawKey);
    if (!resolution) continue;
    const nums = normalizePriceArray(rawPrices);
    // [] 或未填：未启用
    if (nums.length === 0) continue;
    if (nums.length >= 2) {
      rows.push({
        resolution,
        audioPricingEnabled: true,
        videoPrice: '',
        tokenPrice: '',
        noAudioPrice: String(nums[0]),
        withAudioPrice: String(nums[1]),
        pixelCompression: defaultCompression,
      });
    } else {
      rows.push({
        resolution,
        audioPricingEnabled: false,
        videoPrice: perItem ? String(nums[0]) : '',
        tokenPrice: perItem ? '' : String(nums[0]),
        noAudioPrice: '',
        withAudioPrice: '',
        pixelCompression: defaultCompression,
      });
    }
  }
  return rows;
};

const parseVideoToVideoFromJson = (jsonObj, perItem) => {
  const primary = jsonObj.video_to_video ?? jsonObj.videoToVideo;
  if (primary !== undefined && !Array.isArray(primary)) {
    return parseSectionMap(primary, perItem, 'videoUpload');
  }

  // 兼容旧 JSON：video_upload + video_generate 合并为 video_to_video
  const legacyUpload = jsonObj.video_upload ?? jsonObj.videoUpload;
  const legacyGen = jsonObj.video_generate ?? jsonObj.videoGenerate;
  const mergedObj = {};
  const applyMap = (data) => {
    if (!data || typeof data !== 'object' || Array.isArray(data)) {
      if (Array.isArray(data)) {
        for (const row of parseLegacyRowList(data, perItem, 'videoUpload')) {
          const sk = resolutionToShortKey(row.resolution);
          if (!sk) continue;
          const prices = row.audioPricingEnabled
            ? [
                toPriceNumber(row.noAudioPrice),
                toPriceNumber(row.withAudioPrice),
              ].filter((n) => n !== null)
            : [toPriceNumber(row.videoPrice || row.tokenPrice)].filter(
                (n) => n !== null,
              );
          if (prices.length > 0) mergedObj[sk] = prices;
        }
      }
      return;
    }
    for (const [rawKey, rawPrices] of Object.entries(data)) {
      const sk = resolutionToShortKey(rawKey);
      if (sk) mergedObj[sk] = rawPrices;
    }
  };
  applyMap(legacyUpload);
  applyMap(legacyGen);
  if (Object.keys(mergedObj).length === 0) {
    return [];
  }
  return parseSectionMap(mergedObj, perItem, 'videoUpload');
};

/**
 * 将 JSON 对象合并进模型；返回 { ok, model?, message? }
 */
export const mergeVideoPricingFromJsonEditor = (model, jsonObj) => {
  if (!model) {
    return { ok: false, message: '未选择模型' };
  }
  if (!jsonObj || typeof jsonObj !== 'object' || Array.isArray(jsonObj)) {
    return { ok: false, message: '不是合法的 JSON 字符串' };
  }

  const billingModeRaw = String(
    jsonObj.billing_mode ?? jsonObj.billingMode ?? model.videoBillingMode ?? '',
  ).trim();
  const billingMode =
    billingModeRaw === 'per-item'
      ? 'per-item'
      : billingModeRaw === 'per-second'
        ? 'per-second'
        : model.videoBillingMode || 'per-second';
  const perItem = billingMode === 'per-item';

  const priceUnitRaw = String(
    jsonObj.price_unit ?? jsonObj.priceUnit ?? model.videoPriceUnit ?? 'USD',
  )
    .trim()
    .toUpperCase();
  const priceUnit = ['USD', 'CNY', 'CUSTOM'].includes(priceUnitRaw)
    ? priceUnitRaw
    : model.videoPriceUnit || 'USD';

  const fixedPrice = toNumericString(
    jsonObj.fixed_price ?? jsonObj.fixedPrice ?? model.videoFixedPrice,
  );
  const similarityThreshold = toNumericString(
    jsonObj.similarity_threshold ??
      jsonObj.similarityThreshold ??
      model.videoSimilarityThreshold,
  );

  return {
    ok: true,
    model: {
      ...model,
      videoBillingMode: billingMode,
      videoPriceUnit: priceUnit,
      videoFixedPrice: fixedPrice,
      videoSimilarityThreshold: similarityThreshold,
      videoTextToVideoRules: parseSectionMap(
        jsonObj.text_to_video ?? jsonObj.textToVideo,
        perItem,
        'text',
      ),
      videoImageToVideoRules: parseSectionMap(
        jsonObj.image_to_video ?? jsonObj.imageToVideo,
        perItem,
        'image',
      ),
      ...(() => {
        const videoToVideoRows = parseVideoToVideoFromJson(jsonObj, perItem);
        return {
          videoUploadRules: videoToVideoRows.map((row) => ({ ...row })),
          videoGenerateRules: videoToVideoRows.map((row) => ({ ...row })),
        };
      })(),
    },
  };
};

export const VIDEO_PRICING_JSON_PLACEHOLDER = `{
  // billing_mode: per-item（按条）| per-second（按秒）
  "billing_mode": "per-item",
  // price_unit: USD | CNY | CUSTOM
  "price_unit": "USD",
  // 无分辨率表时的兜底单价
  "fixed_price": "",
  // 相似分辨率阈值，仅 per-second 有效
  "similarity_threshold": "",
  // 三大类：[] 未启用，[价] 统一价，[无音轨, 有音轨] 分开计价
  // 文生视频
  "text_to_video": {
    "480": [],
    "540": [],
    "720": [],
    "1080": [],
    "2k": [],
    "4k": []
  },
  // 图生视频
  "image_to_video": {
    "480": [],
    "540": [],
    "720": [],
    "1080": [],
    "2k": [],
    "4k": []
  },
  // 视频生视频
  "video_to_video": {
    "480": [],
    "540": [],
    "720": [],
    "1080": [],
    "2k": [],
    "4k": []
  }
}`;
