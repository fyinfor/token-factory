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
