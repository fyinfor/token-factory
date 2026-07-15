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

// 素材资产类型枚举（AssetType，与后端 / 上游素材库接口严格对齐，禁止硬编码字符串）。
export const MaterialAssetType = Object.freeze({
  IMAGE: 'Image',
  VIDEO: 'Video',
  AUDIO: 'Audio',
});

// 素材状态枚举（Status，与后端 / 上游素材库接口严格对齐）。
export const MaterialStatus = Object.freeze({
  ACTIVE: 'Active',
  FAILED: 'Failed',
  PENDING: 'Pending',
});

// 上传方式枚举：本地文件 / 在线资源链接。
export const MaterialUploadMode = Object.freeze({
  LOCAL: 'local',
  URL: 'url',
});

// 允许上传的图片扩展名（与后端校验保持一致）。
export const MATERIAL_IMAGE_EXTS = [
  '.jpg',
  '.jpeg',
  '.png',
  '.webp',
  '.gif',
  '.bmp',
];

// 允许上传的视频扩展名（与后端校验保持一致）。
export const MATERIAL_VIDEO_EXTS = [
  '.mp4',
  '.mov',
  '.webm',
  '.mkv',
  '.avi',
  '.m4v',
];

// 允许上传的音频扩展名（SD 素材库上游仅接受 mp3 / wav，与后端校验保持一致）。
export const MATERIAL_AUDIO_EXTS = ['.mp3', '.wav'];

// 素材库不接受的音频扩展名（上游仅 mp3/wav；用于链接/文件名提前拦截）。
export const MATERIAL_UNSUPPORTED_AUDIO_EXTS = [
  '.aif',
  '.aiff',
  '.aifc',
  '.flac',
  '.ogg',
  '.oga',
  '.m4a',
  '.ape',
  '.wma',
  '.aac',
];

// Upload 组件 accept：图片 / 视频 + 明确音频后缀（禁止 audio/* 以免误选 aif/flac/ogg 等）。
export const MATERIAL_UPLOAD_ACCEPT =
  'image/*,video/*,.mp3,.wav,audio/mpeg,audio/wav,audio/x-wav';

// 根据文件名 / URL 推断素材类型，返回 Image / Video / Audio，无法识别返回空串。
export const detectAssetTypeByName = (name = '') => {
  const lower = String(name).toLowerCase();
  // 去除查询参数后再取扩展名，兼容带签名参数的在线链接。
  const clean = lower.split('?')[0].split('#')[0];
  const dot = clean.lastIndexOf('.');
  if (dot < 0) return '';
  const ext = clean.slice(dot);
  if (MATERIAL_UNSUPPORTED_AUDIO_EXTS.includes(ext)) return '';
  if (MATERIAL_IMAGE_EXTS.includes(ext)) return MaterialAssetType.IMAGE;
  if (MATERIAL_VIDEO_EXTS.includes(ext)) return MaterialAssetType.VIDEO;
  if (MATERIAL_AUDIO_EXTS.includes(ext)) return MaterialAssetType.AUDIO;
  return '';
};

// 是否为素材库明确不支持的音频格式（用于上传前提示）。
export const isUnsupportedMaterialAudioName = (name = '') => {
  const lower = String(name).toLowerCase();
  const clean = lower.split('?')[0].split('#')[0];
  const dot = clean.lastIndexOf('.');
  if (dot < 0) return false;
  return MATERIAL_UNSUPPORTED_AUDIO_EXTS.includes(clean.slice(dot));
};

// 判断是否为业务允许的素材类型（图片 / 视频 / 音频）。
export const isAllowedAssetType = (assetType) =>
  assetType === MaterialAssetType.IMAGE ||
  assetType === MaterialAssetType.VIDEO ||
  assetType === MaterialAssetType.AUDIO;
