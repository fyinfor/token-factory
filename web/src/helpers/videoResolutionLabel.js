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

  if (/^\d+p$/i.test(compact)) {
    const n = parseInt(compact, 10);
    if (Number.isFinite(n)) return { short: n, long: n };
  }

  if (lower === '8k') return { short: 4320, long: 7680 };
  if (lower === '4k') return { short: 2160, long: 3840 };
  if (lower === '2k') return { short: 1440, long: 2560 };

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

  if (/^\d+p$/i.test(compact)) {
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
  if (short >= 720) return '720p';
  if (short >= 540) return '540p';
  if (short >= 480) return '480p';
  if (short >= 360) return '360p';
  if (short >= 240) return '240p';
  return `${short}p`;
}

const VIDEO_RESOLUTION_DIMENSIONS = {
  '480p': { width: 854, height: 480 },
  '540p': { width: 960, height: 540 },
  '720p': { width: 1280, height: 720 },
  '1080p': { width: 1920, height: 1080 },
  '2K': { width: 2560, height: 1440 },
  '4K': { width: 3840, height: 2160 },
  '8K': { width: 7680, height: 4320 },
};

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
  const p = compact.match(/^(\d+)p$/i);
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

export function getPlaygroundVideoSizeForTier(
  resolution,
  orientation = 'landscape',
) {
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
      { label: '1080p', value: '1080p', rawResolution: '1080p' },
    ];
  }

  return options.sort((a, b) =>
    compareVideoResolutionAsc(a.rawResolution, b.rawResolution),
  );
}
