/**
 * 解析分辨率用于排序：返回较短边、较长边（像素级近似）；无法识别则返回 null。
 */
function parseResolutionDimsForSort(raw) {
  if (raw == null) return null;
  const s = String(raw).trim();
  if (!s) return null;
  const compact = s.replace(/\s+/g, '');
  const lower = compact.toLowerCase();

  const wxh = lower.match(/^(\d+)\s*[x×]\s*(\d+)$/i);
  if (wxh) {
    const w = parseInt(wxh[1], 10);
    const h = parseInt(wxh[2], 10);
    if (Number.isFinite(w) && Number.isFinite(h) && w > 0 && h > 0) {
      return { short: Math.min(w, h), long: Math.max(w, h) };
    }
  }

  if (/^\d+p?$/i.test(compact)) {
    const n = parseInt(compact, 10);
    if (Number.isFinite(n)) return { short: n, long: n };
  }

  if (lower === '8k') return { short: 4320, long: 7680 };
  if (lower === '4k') return { short: 2160, long: 3840 };
  if (lower === '2k') return { short: 1440, long: 2560 };
  if (lower === '1k') return { short: 1024, long: 1024 };
  if (lower === '512p' || lower === '512') return { short: 512, long: 512 };

  if (/^\d+k$/i.test(compact)) {
    const n = parseInt(compact, 10);
    if (Number.isFinite(n) && n > 0) {
      const short = n * 720;
      return { short, long: short };
    }
  }

  return null;
}

/**
 * 分辨率从低到高比较（用于定价档位表排序）。
 * 可识别 WxH、720p、4K/2K/8K 等；无法解析的排在已解析项之后，彼此按字符串序。
 */
export function compareVideoResolutionAsc(rawA, rawB) {
  const da = parseResolutionDimsForSort(rawA);
  const db = parseResolutionDimsForSort(rawB);
  if (da && db) {
    if (da.short !== db.short) return da.short - db.short;
    if (da.long !== db.long) return da.long - db.long;
    return String(rawA).localeCompare(String(rawB));
  }
  if (da && !db) return -1;
  if (!da && db) return 1;
  return String(rawA ?? '').localeCompare(String(rawB ?? ''));
}

/**
 * 将配置里的分辨率（如 854x480、1280x720）转成用户易读的 480p / 720p / 4K 等。
 * 已是 720p、2K 等形式则尽量规范化后原样返回。
 */
export function formatVideoResolutionDisplayLabel(raw) {
  if (raw == null) return '';
  const s = String(raw).trim();
  if (!s) return '';

  const compact = s.replace(/\s+/g, '');
  const lower = compact.toLowerCase();

  if (/^\d+p?$/i.test(compact)) {
    const n = parseInt(compact, 10);
    return Number.isFinite(n) ? `${n}p` : s;
  }
  if (/^\d+k$/i.test(compact)) {
    const n = parseInt(compact, 10);
    return Number.isFinite(n) ? `${n}K` : s;
  }

  const m = lower.match(/^(\d+)\s*[x×]\s*(\d+)$/i);
  if (!m) return s;

  const w = parseInt(m[1], 10);
  const h = parseInt(m[2], 10);
  if (!Number.isFinite(w) || !Number.isFinite(h) || w <= 0 || h <= 0) {
    return s;
  }

  const short = Math.min(w, h);
  const long = Math.max(w, h);

  if (long >= 7680 || short >= 4320) return '8K';
  if (short >= 2160) return '4K';
  if (short >= 1440) return '2K';
  if (short >= 1080) return '1080p';
  if (short >= 768) return '768p';
  if (short >= 720) return '720p';
  if (short >= 540) return '540p';
  if (short >= 480) return '480p';
  if (short >= 360) return '360p';
  if (short >= 240) return '240p';
  return `${short}p`;
}

/**
 * 将图片分辨率归一化为 Ai 绘图档位标识（512P / 1K / 2K / 4K）。
 * 与视频分辨率规则相互独立。按实际输出图片短边像素：
 *   512P：短边 ≤ 512px
 *   1K：短边 ≤ 1024px（512＜短边≤1024）
 *   2K：1024px ＜ 短边 ≤ 2048px
 *   4K：短边 ＞ 2048px
 * 例如 512×512→512P，1024×1536→1K，1920×1080→2K，2160×3840→4K。
 * 显式「1K」「1080p」统一展示为 1K（历史计费别名）。
 */
export function formatImageResolutionDisplayLabel(raw) {
  if (raw == null) return '';
  const s = String(raw).trim();
  if (!s) return '';

  const compact = s.replace(/\s+/g, '');
  const lower = compact.toLowerCase();

  // 显式档位别名优先：历史「1080p」≡ 1K，避免把字面量「1080p」按短边 1080 误判为 2K。
  if (lower === '512p' || lower === '512') {
    return '512P';
  }
  if (lower === '1k' || lower === '1080p' || lower === '1080') {
    return '1K';
  }

  if (/^\d+k$/i.test(compact)) {
    const n = parseInt(compact, 10);
    if (!Number.isFinite(n)) return s;
    if (n === 1) return '1K';
    if (n === 2 || n === 4) return `${n}K`;
    return `${n}K`;
  }

  let short = null;
  if (/^\d+p?$/i.test(compact)) {
    const n = parseInt(compact, 10);
    if (Number.isFinite(n) && n > 0) short = n;
  } else {
    const m = lower.match(/^(\d+)\s*[x×]\s*(\d+)$/i);
    if (m) {
      const w = parseInt(m[1], 10);
      const h = parseInt(m[2], 10);
      if (Number.isFinite(w) && Number.isFinite(h) && w > 0 && h > 0) {
        short = Math.min(w, h);
      }
    }
  }
  if (short == null) return s;
  if (short <= 512) return '512P';
  if (short <= 1024) return '1K';
  if (short <= 2048) return '2K';
  return '4K';
}

/**
 * 使用日志分辨率展示：预扣时用户输入的 resolution 原样保留；其余场景归一化为 480p/720p 等。
 */
export function resolveVideoBillingResolutionLabel(raw, fromInput = false) {
  const s = String(raw ?? '').trim();
  if (!s) return '';
  if (fromInput) return s;
  return formatVideoResolutionDisplayLabel(s) || s;
}

/**
 * 使用日志规格展示（分辨率 + 比例）；fromInput 为 true 时不将 1280x720 转为 720p。
 */
export function formatVideoSpecLabelForBilling(
  resolution,
  ratio,
  fromInput = false,
) {
  const resLabel = resolveVideoBillingResolutionLabel(resolution, fromInput);
  const ratioLabel = String(ratio ?? '').trim();
  if (resLabel && ratioLabel) {
    return `${resLabel} ${ratioLabel}`;
  }
  return resLabel || ratioLabel || '';
}

/**
 * 视频规格展示（计费日志规范）：统一展示「分辨率标识 + 画面比例」，禁止渲染像素尺寸。
 * 例：formatVideoSpecLabel('1280x720', '16:9') => '720p 16:9'；
 *     formatVideoSpecLabel('480p', '16:9')     => '480p 16:9'。
 * 任一字段缺失时仅展示已有部分；均缺失返回空串。
 *
 * @param {string|number} resolution 分辨率（支持 480p / 720p / 1280x720 等，像素将被转为分辨率标识）
 * @param {string} ratio 画面比例（如 '16:9'）
 * @returns {string}
 */
export function formatVideoSpecLabel(resolution, ratio) {
  // formatVideoResolutionDisplayLabel 会把 1280x720 等像素尺寸归一为 720p，确保不渲染像素。
  const resLabel = formatVideoResolutionDisplayLabel(resolution);
  const ratioLabel = String(ratio == null ? '' : ratio).trim();
  if (resLabel && ratioLabel) {
    return `${resLabel} ${ratioLabel}`;
  }
  return resLabel || ratioLabel || '';
}

const VIDEO_RESOLUTION_DIMENSIONS = {
  '480p': { width: 854, height: 480 },
  '540p': { width: 960, height: 540 },
  '720p': { width: 1280, height: 720 },
  '768p': { width: 1366, height: 768 },
  '1080p': { width: 1920, height: 1080 },
  '2K': { width: 2560, height: 1440 },
  '4K': { width: 3840, height: 2160 },
  '8K': { width: 7680, height: 4320 },
};

/** Ai 绘图：分辨率档位 × 画面比例 → 固定像素尺寸（与视频档位表相互独立） */
const AI_DRAWING_SIZE_TABLE = {
  '512P': {
    '1:1': { width: 512, height: 512 },
    '16:9': { width: 768, height: 432 },
    '9:16': { width: 432, height: 768 },
    '4:3': { width: 512, height: 384 },
    '3:4': { width: 384, height: 512 },
    '21:9': { width: 768, height: 328 },
  },
  '1K': {
    '1:1': { width: 1024, height: 1024 },
    '16:9': { width: 1536, height: 864 },
    '9:16': { width: 864, height: 1536 },
    '4:3': { width: 1024, height: 768 },
    '3:4': { width: 768, height: 1024 },
    '21:9': { width: 1536, height: 656 },
  },
  '2K': {
    '1:1': { width: 2048, height: 2048 },
    '16:9': { width: 2048, height: 1152 },
    '9:16': { width: 1152, height: 2048 },
    '4:3': { width: 1536, height: 1152 },
    '3:4': { width: 1152, height: 1536 },
    '21:9': { width: 2048, height: 880 },
  },
  '4K': {
    '1:1': { width: 4096, height: 4096 },
    '16:9': { width: 3840, height: 2160 },
    '9:16': { width: 2160, height: 3840 },
    '4:3': { width: 3072, height: 2304 },
    '3:4': { width: 2304, height: 3072 },
    '21:9': { width: 4096, height: 1760 },
  },
};

const AI_DRAWING_TIER_REPRESENTATIVE = {
  '512P': { width: 512, height: 512 },
  '1K': { width: 1024, height: 1024 },
  '2K': { width: 2048, height: 2048 },
  '4K': { width: 4096, height: 4096 },
};

const AI_DRAWING_TIERS = ['512P', '1K', '2K', '4K'];

function parseResolutionDimensions(raw) {
  if (raw == null) return null;
  const compact = String(raw).trim().replace(/\s+/g, '');
  if (!compact) return null;
  const wxh = compact.toLowerCase().match(/^(\d+)\s*[x×]\s*(\d+)$/i);
  if (wxh) {
    const width = parseInt(wxh[1], 10);
    const height = parseInt(wxh[2], 10);
    if (
      Number.isFinite(width) &&
      Number.isFinite(height) &&
      width > 0 &&
      height > 0
    ) {
      return { width, height };
    }
  }
  const label = formatVideoResolutionDisplayLabel(compact);
  if (VIDEO_RESOLUTION_DIMENSIONS[label]) {
    return VIDEO_RESOLUTION_DIMENSIONS[label];
  }
  const p = compact.match(/^(\d+)p?$/i);
  if (p) {
    const short = parseInt(p[1], 10);
    if (Number.isFinite(short) && short > 0) {
      return {
        width: Math.max(2, Math.round((short * 16) / 9 / 2) * 2),
        height: short,
      };
    }
  }
  return null;
}

function parseAspectRatioValue(ratio) {
  const compact = String(ratio || '').trim();
  if (!compact || compact === 'auto') return null;
  const parts = compact.split(':');
  if (parts.length !== 2) return null;
  const widthRatio = Number(parts[0]);
  const heightRatio = Number(parts[1]);
  if (
    !Number.isFinite(widthRatio) ||
    !Number.isFinite(heightRatio) ||
    widthRatio <= 0 ||
    heightRatio <= 0
  ) {
    return null;
  }
  return { widthRatio, heightRatio };
}

function getSizeForAspectRatio(resolution, ratio) {
  const aspect = parseAspectRatioValue(ratio);
  if (!aspect) return null;

  const dims =
    parseResolutionDimensions(resolution) ||
    VIDEO_RESOLUTION_DIMENSIONS['720p'];
  const shortSide = Math.min(dims.width, dims.height);
  const landscape = aspect.widthRatio >= aspect.heightRatio;
  const baseShort = shortSide;
  const baseLong = Math.max(
    2,
    Math.round(
      (baseShort * Math.max(aspect.widthRatio, aspect.heightRatio)) /
        Math.min(aspect.widthRatio, aspect.heightRatio) /
        2,
    ) * 2,
  );
  const width = landscape ? baseLong : baseShort;
  const height = landscape ? baseShort : baseLong;
  return {
    width,
    height,
    size: `${width}x${height}`,
  };
}

export function getPlaygroundVideoSizeForTier(
  resolution,
  orientation = 'landscape',
  ratio = '',
) {
  const ratioSize = getSizeForAspectRatio(resolution, ratio);
  if (ratioSize) {
    return ratioSize;
  }

  const dims =
    parseResolutionDimensions(resolution) ||
    VIDEO_RESOLUTION_DIMENSIONS['720p'];
  const short = Math.min(dims.width, dims.height);
  const long = Math.max(dims.width, dims.height);
  if (orientation === 'portrait') {
    return { width: short, height: long, size: `${short}x${long}` };
  }
  return { width: long, height: short, size: `${long}x${short}` };
}

/** 操练场图片：按 Ai 绘图分辨率档位 + 画面比例查表得到上游 size */
export function getPlaygroundImageSizeForTier(resolution, ratio = '1:1') {
  let tier = formatImageResolutionDisplayLabel(resolution);
  if (!AI_DRAWING_TIERS.includes(tier)) {
    const dims = parseResolutionDimensions(resolution);
    if (dims) {
      tier = formatImageResolutionDisplayLabel(`${dims.width}x${dims.height}`);
    }
  }
  const normalizedTier = AI_DRAWING_TIERS.includes(tier) ? tier : '1K';
  const ratioKey =
    !ratio || ratio === 'auto' ? '1:1' : String(ratio).trim();
  const dims =
    AI_DRAWING_SIZE_TABLE[normalizedTier]?.[ratioKey] ||
    AI_DRAWING_SIZE_TABLE[normalizedTier]?.['1:1'] ||
    AI_DRAWING_TIER_REPRESENTATIVE['1K'];
  return {
    width: dims.width,
    height: dims.height,
    size: `${dims.width}x${dims.height}`,
  };
}

/** 当前分辨率展示：1920x1080 → 1920 x 1080 */
export function formatPlaygroundPixelSizeLabel(size) {
  const compact = String(size ?? '').trim().replace(/\s+/g, '');
  const match = compact.match(/^(\d+)[x×](\d+)$/i);
  if (match) {
    return `${match[1]} x ${match[2]}`;
  }
  return String(size ?? '').trim();
}

/** 解析操练场自定义图片尺寸（如 1280x720 / 1280×720） */
export function parsePlaygroundCustomImageSize(value) {
  const compact = String(value ?? '')
    .trim()
    .replace(/\s+/g, '');
  if (!compact) return null;
  const match = compact.match(/^(\d+)[x×](\d+)$/i);
  if (!match) return null;
  const width = Number(match[1]);
  const height = Number(match[2]);
  if (
    !Number.isFinite(width) ||
    !Number.isFinite(height) ||
    width <= 0 ||
    height <= 0
  ) {
    return null;
  }
  return {
    width,
    height,
    size: `${Math.round(width)}x${Math.round(height)}`,
  };
}

/**
 * 解析操练场最终图片 size：自定义尺寸优先，否则按分辨率档位 + 比例计算。
 */
export function resolvePlaygroundImageSize(inputs = {}) {
  const customSize = parsePlaygroundCustomImageSize(inputs.image_custom_size);
  if (customSize) {
    return customSize;
  }
  const sizeOptions = buildPlaygroundImageSizeOptions(
    inputs.selected_image_pricing_tiers,
  );
  const selectedImageSize = sizeOptions.some(
    (option) => option.value === inputs.image_size,
  )
    ? inputs.image_size
    : preferPlaygroundImageSize(sizeOptions);
  return getPlaygroundImageSizeForTier(
    selectedImageSize,
    inputs.image_ratio === 'auto' || !inputs.image_ratio
      ? '1:1'
      : inputs.image_ratio,
  );
}

/** 写入助手消息的生成元信息（对话区回复下方展示） */
export function buildPlaygroundGenerationMeta(inputs = {}) {
  const mode = inputs.display_mode || 'text';
  const model = String(inputs.model || '').trim();
  if (mode === 'image') {
    const imageSize = resolvePlaygroundImageSize(inputs);
    return {
      mode,
      model,
      size: imageSize?.size || '',
    };
  }
  if (mode === 'video') {
    const resolutionOptions = buildPlaygroundVideoResolutionOptions(
      inputs.selected_video_pricing_tiers,
    );
    const selectedResolution = resolutionOptions.some(
      (option) => option.value === inputs.video_resolution_preset,
    )
      ? inputs.video_resolution_preset
      : resolutionOptions[0]?.value ||
        String(inputs.video_resolution_preset || '').trim() ||
        '720p';
    return {
      mode,
      model,
      resolution: selectedResolution,
      ratio: String(inputs.video_ratio || '').trim(),
    };
  }
  return { mode, model };
}

export function buildPlaygroundVideoResolutionOptions(tiers) {
  const rawResolutions = (Array.isArray(tiers) ? tiers : [])
    .map((tier) => (typeof tier === 'string' ? tier : tier?.resolution))
    .map((resolution) => String(resolution || '').trim())
    .filter(Boolean);

  const seen = new Set();
  const options = [];
  for (const resolution of rawResolutions) {
    const label = formatVideoResolutionDisplayLabel(resolution) || resolution;
    const value = label;
    const sortDims = parseResolutionDimsForSort(resolution);
    if (!sortDims || seen.has(value)) continue;
    seen.add(value);
    options.push({
      label,
      value,
      rawResolution: resolution,
    });
  }

  if (options.length === 0) {
    return [
      { label: '540p', value: '540p', rawResolution: '540p' },
      { label: '720p', value: '720p', rawResolution: '720p' },
      { label: '768p', value: '768p', rawResolution: '768p' },
      { label: '1080p', value: '1080p', rawResolution: '1080p' },
    ];
  }

  return options.sort((a, b) =>
    compareVideoResolutionAsc(a.rawResolution, b.rawResolution),
  );
}

const DEFAULT_PLAYGROUND_IMAGE_SIZE_OPTIONS = [
  { label: '512P', value: '512x512', rawResolution: '512x512' },
  { label: '1K', value: '1024x1024', rawResolution: '1024x1024' },
  { label: '2K', value: '2048x2048', rawResolution: '2048x2048' },
  { label: '4K', value: '4096x4096', rawResolution: '4096x4096' },
];

function getImageSizeValueForResolution(resolution) {
  const tier = formatImageResolutionDisplayLabel(resolution);
  if (AI_DRAWING_TIER_REPRESENTATIVE[tier]) {
    const dims = AI_DRAWING_TIER_REPRESENTATIVE[tier];
    return `${dims.width}x${dims.height}`;
  }
  const dims = parseResolutionDimensions(resolution);
  if (!dims) {
    return String(resolution || '').trim();
  }
  return `${dims.width}x${dims.height}`;
}

export function buildPlaygroundImageSizeOptions(tiers) {
  const rawResolutions = (Array.isArray(tiers) ? tiers : [])
    .map((tier) => (typeof tier === 'string' ? tier : tier?.resolution))
    .map((resolution) => String(resolution || '').trim())
    .filter(Boolean);

  const seen = new Set();
  const options = [];
  for (const resolution of rawResolutions) {
    const label = formatImageResolutionDisplayLabel(resolution) || resolution;
    const value = getImageSizeValueForResolution(resolution);
    if (!value || !label || seen.has(label)) continue;
    seen.add(label);
    options.push({
      label,
      value,
      rawResolution: resolution,
    });
  }

  if (options.length === 0) {
    return DEFAULT_PLAYGROUND_IMAGE_SIZE_OPTIONS;
  }

  return options.sort((a, b) =>
    compareVideoResolutionAsc(a.rawResolution, b.rawResolution),
  );
}

/** 操练场图片默认选中：优先 1K（1024x1024），否则取最低档 */
export function preferPlaygroundImageSize(options) {
  const list = Array.isArray(options) ? options : [];
  if (list.length === 0) return '1024x1024';
  const oneK = list.find((option) => {
    const label = String(option?.label || '').toLowerCase();
    return label === '1k' || label === '1080p' || option?.value === '1024x1024';
  });
  if (oneK?.value) return oneK.value;
  return list[0]?.value || '1024x1024';
}
