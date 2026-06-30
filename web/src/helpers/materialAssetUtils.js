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

/**
 * 素材 asset:// 协议工具方法。
 * 统一处理 asset URI 的构建、解析、拼接、分隔以及预览地址转换。
 */

// asset:// 协议前缀（与后端 toMaterialAssetResponse 一致：asset:// + 上游 asset_id）
export const ASSET_URI_SCHEME = 'asset://';

/** @deprecated 请使用 ASSET_URI_SCHEME；保留别名避免旧引用报错 */
export const ASSET_URI_PREFIX = ASSET_URI_SCHEME;

// 多素材拼接分隔符（换行符，与操练场多行输入兼容）
export const ASSET_URI_SEPARATOR = '\n';

/**
 * 构建 asset:// 协议地址（上游 asset_id，如 asset-xxxx）
 * @param {string} upstreamAssetId 上游素材 ID
 * @returns {string} asset://asset-xxxx 格式地址
 */
export const buildAssetUri = (upstreamAssetId) => {
  const id = String(upstreamAssetId || '').trim();
  if (!id) return '';
  return ASSET_URI_SCHEME + id;
};

/**
 * 从素材对象获取真实 asset:// 地址（优先 API 返回的 asset_uri）
 * @param {{ asset_uri?: string, asset_id?: string }} asset
 * @returns {string}
 */
export const getMaterialAssetUri = (asset) => {
  if (!asset) return '';
  const uri = String(asset.asset_uri || '').trim();
  if (uri) return uri;
  return buildAssetUri(asset.asset_id);
};

/**
 * 从 asset:// 协议地址中解析上游素材 ID
 * @param {string} uri asset://asset-xxxx 格式地址
 * @returns {string} 上游素材 ID，无法解析返回空字符串
 */
export const parseAssetId = (uri) => {
  const s = String(uri || '').trim();
  if (!s.startsWith(ASSET_URI_SCHEME)) return '';
  return s.slice(ASSET_URI_SCHEME.length).trim();
};

/**
 * 判断值是否为 asset:// 协议地址
 * @param {string} value
 * @returns {boolean}
 */
export const isAssetUri = (value) => {
  const s = String(value || '').trim();
  return s.length > ASSET_URI_SCHEME.length && s.startsWith(ASSET_URI_SCHEME);
};

/**
 * 【需求5】多素材 URI 拼接
 * @param {string[]} uris asset URI 数组
 * @returns {string} 以分隔符拼接的字符串
 */
export const joinAssetUris = (uris) => {
  if (!Array.isArray(uris)) return '';
  return uris
    .map((u) => String(u || '').trim())
    .filter(Boolean)
    .join(ASSET_URI_SEPARATOR);
};

/**
 * 拆分多素材拼接字符串为单个 URI 数组
 * @param {string} value 可能包含分隔符的字符串
 * @returns {string[]} 拆分后的 URI 数组
 */
export const splitAssetUris = (value) => {
  const s = String(value || '').trim();
  if (!s) return [];
  return s
    .split(/\n+/)
    .map((u) => u.trim())
    .filter(Boolean);
};

/**
 * 【需求6】构建上游素材 ID -> 在线预览URL 映射表
 * @param {Array} assets 素材列表
 * @returns {Object<string, string>} { upstreamAssetId: previewUrl }
 */
export const buildAssetMap = (assets) => {
  if (!Array.isArray(assets)) return {};
  const map = {};
  for (const asset of assets) {
    if (!asset?.url) continue;
    const upstreamId = String(asset.asset_id || parseAssetId(asset.asset_uri) || '').trim();
    if (upstreamId) {
      map[upstreamId] = String(asset.url);
    }
  }
  return map;
};

/**
 * 【需求6】将单个 asset:// 地址转换为在线预览URL
 * 素材库不存在对应素材ID时，保留原始地址，不报错、不空白。
 * @param {string} uri asset:// 协议地址
 * @param {Object<string, string>} assetMap 素材ID -> 预览URL 映射
 * @returns {string} 解析后的预览URL
 */
export const resolveAssetUriToUrl = (uri, assetMap) => {
  const s = String(uri || '').trim();
  if (!isAssetUri(s)) return s;
  const assetId = parseAssetId(s);
  if (!assetId) return s;
  const resolvedUrl = assetMap?.[assetId];
  return resolvedUrl || s;
};

/**
 * 【需求6】批量解析 URL 数组中的 asset:// 地址
 * @param {string[]} urls URL 数组
 * @param {Object<string, string>} assetMap 素材映射表
 * @returns {string[]} 解析后的 URL 数组
 */
export const resolveAssetUrisInArray = (urls, assetMap) => {
  if (!Array.isArray(urls)) return [];
  const result = [];
  for (const rawUrl of urls) {
    const s = String(rawUrl || '').trim();
    if (!s) continue;
    const parts = splitAssetUris(s);
    for (const part of parts) {
      const resolved = resolveAssetUriToUrl(part, assetMap);
      if (resolved) result.push(resolved);
    }
  }
  return result;
};

