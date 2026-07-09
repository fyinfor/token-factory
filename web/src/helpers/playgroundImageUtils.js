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

import { splitAssetUris } from './materialAssetUtils';
import {
  extractImageUrlsFromMessages,
  isAudioURL,
  isVideoURL,
} from './playgroundVideoUtils';

/** 将 b64_json / 裸 base64 转为可在 img src 使用的 data URL */
export function normalizeBase64ImageSrc(value, mime = 'image/png') {
  const s = String(value || '').trim();
  if (!s) return '';
  if (s.startsWith('data:')) return s;
  if (s.startsWith('http://') || s.startsWith('https://')) return s;
  return `data:${mime};base64,${s}`;
}

function itemToImageSrc(item) {
  if (!item) return '';
  if (typeof item === 'string') {
    const s = item.trim();
    if (!s) return '';
    if (s.startsWith('http://') || s.startsWith('https://') || s.startsWith('data:'))
      return s;
    return normalizeBase64ImageSrc(s);
  }
  if (typeof item !== 'object') return '';
  if (typeof item.url === 'string' && item.url.trim()) return item.url.trim();
  if (typeof item.image_url === 'string' && item.image_url.trim())
    return item.image_url.trim();
  const b64 =
    item.b64_json ?? item.base64 ?? item.binary ?? item.image_base64 ?? item.data;
  if (typeof b64 === 'string' && b64.trim()) {
    const mime =
      typeof item.mime_type === 'string' && item.mime_type.trim()
        ? item.mime_type.trim()
        : 'image/png';
    return normalizeBase64ImageSrc(b64, mime);
  }
  return '';
}

function collectSourcesFromList(list) {
  if (!Array.isArray(list)) return [];
  return list.map((item) => itemToImageSrc(item)).filter(Boolean);
}

export function dedupeImageSources(sources) {
  const seen = new Set();
  const out = [];
  for (const src of sources) {
    if (!src || seen.has(src)) continue;
    seen.add(src);
    out.push(src);
  }
  return out;
}

/** 从 OpenAI / 异步任务 / 即梦等响应体中提取全部可展示图片地址 */
export function extractImageSources(obj, depth = 0) {
  if (!obj || typeof obj !== 'object' || depth > 6) return [];

  const results = [];

  results.push(...collectSourcesFromList(obj.data));
  results.push(...collectSourcesFromList(obj.image_urls));
  results.push(...collectSourcesFromList(obj.images));
  results.push(...collectSourcesFromList(obj.outputs));

  if (Array.isArray(obj.binary_data_base64)) {
    for (const b64 of obj.binary_data_base64) {
      const src = normalizeBase64ImageSrc(b64);
      if (src) results.push(src);
    }
  }

  const directKeys = ['url', 'image_url', 'output_url', 'result_url'];
  for (const key of directKeys) {
    const v = obj[key];
    if (typeof v === 'string' && v.trim()) results.push(v.trim());
  }

  if (obj.output && typeof obj.output === 'object') {
    results.push(...extractImageSources(obj.output, depth + 1));
  }
  if (obj.result && typeof obj.result === 'object') {
    results.push(...extractImageSources(obj.result, depth + 1));
  }

  if (obj.data && typeof obj.data === 'object' && !Array.isArray(obj.data)) {
    results.push(...extractImageSources(obj.data, depth + 1));
  }

  return dedupeImageSources(results);
}

/** 操练场生成图 markdown 占位（空 alt，兼容旧版 generated-image-N） */
const GENERATED_IMAGE_MARKDOWN_REGEX =
  /!\[(?:generated-image-\d+)?\]\(([\s\S]*?)\)/g;
const GENERATED_IMAGE_MARKDOWN_STRIP_REGEX =
  /!\[(?:generated-image-\d+)?\]\([\s\S]*?\)\s*/g;

export function buildImageMarkdown(sources) {
  return dedupeImageSources(sources)
    .map((src) => `![](${src})`)
    .join('\n\n');
}

/** 构造操练场消息 patch：generatedImages + markdown 正文 */
export function buildImageMessageContentPatch(sources) {
  const images = dedupeImageSources(sources);
  if (!images.length) return null;
  return {
    generatedImages: images,
    content: buildImageMarkdown(images),
  };
}

/** 从助手消息 markdown 中解析生成图占位地址 */
export function extractGeneratedImagesFromMarkdown(text) {
  if (typeof text !== 'string' || !text.trim()) return [];
  const images = [];
  for (const match of text.matchAll(GENERATED_IMAGE_MARKDOWN_REGEX)) {
    const src = match[1]?.trim();
    if (src) images.push(src);
  }
  return dedupeImageSources(images);
}

/** 合并 message.generatedImages 与 markdown 中的图片（持久化后仍可读） */
export function resolveMessageGeneratedImages(message) {
  const fromField = Array.isArray(message?.generatedImages)
    ? message.generatedImages.filter(Boolean)
    : [];
  if (fromField.length) return dedupeImageSources(fromField);
  if (typeof message?.content === 'string') {
    return extractGeneratedImagesFromMarkdown(message.content);
  }
  return [];
}

/** 展示 markdown 时去掉已由画廊渲染的生成图占位 */
export function stripGeneratedImageMarkdown(text) {
  if (typeof text !== 'string' || !text.trim()) return '';
  return text
    .replace(GENERATED_IMAGE_MARKDOWN_STRIP_REGEX, '')
    .replace(/\n{3,}/g, '\n\n')
    .trim();
}

/**
 * 操练场文本 / 图片模式：合并媒体侧栏与消息内的参考图 URL。
 * 空地址会被过滤，无有效图片时返回空数组（调用方无需传参、也不应报错）。
 */
export function collectPlaygroundImageMediaUrls(sidebarImageUrls, messages) {
  const expandedSidebarUrls = (sidebarImageUrls || []).flatMap((url) => {
    const trimmed = String(url || '').trim();
    if (!trimmed) return [];
    return splitAssetUris(trimmed);
  });
  const merged = [
    ...expandedSidebarUrls.map((url) => String(url || '').trim()).filter(Boolean),
    ...extractImageUrlsFromMessages(messages),
  ];
  const seen = new Set();
  const images = [];
  for (const url of merged) {
    const trimmed = String(url || '').trim();
    if (!trimmed || seen.has(trimmed) || isVideoURL(trimmed) || isAudioURL(trimmed)) {
      continue;
    }
    seen.add(trimmed);
    images.push(trimmed);
  }
  return images;
}
