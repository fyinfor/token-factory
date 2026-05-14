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

/** 与后端 video_flat_clip_hint.lane 一致，值为 i18n 中文 key */export const VIDEO_FLAT_LANE_I18N_KEY = {
  text_to_video: '文生视频',
  image_to_video: '图生视频',
  video_to_video: '视频生视频',
  text_to_video_legacy: '文生视频（旧）',
  image_to_video_legacy: '图生视频（旧）',
  video_to_video_input_legacy: '视频接入（旧）',
  video_to_video_output_legacy: '视频输出（旧）',
  text_to_video_per_second: '文生视频（按秒）',
  image_to_video_per_second: '图生视频（按秒）',
  video_to_video_per_second: '视频生视频（按秒）',
};

const LANE_FAMILY_BUCKETS = {
  text_to_video: [
    'text_to_video',
    'text_to_video_legacy',
    'text_to_video_per_second',
  ],
  image_to_video: [
    'image_to_video',
    'image_to_video_legacy',
    'image_to_video_per_second',
  ],
  video_to_video: [
    'video_to_video',
    'video_to_video_input_legacy',
    'video_to_video_output_legacy',
    'video_to_video_per_second',
  ],
};

/** 文生 / 图生 / 视频生 / 其他（侧栏分表标题用 i18n key） */
export const VIDEO_FLAT_FAMILY_ORDER = [
  'text_to_video',
  'image_to_video',
  'video_to_video',
  'other',
];

export const VIDEO_FLAT_FAMILY_TITLE_KEY = {
  text_to_video: '文生视频',
  image_to_video: '图生视频',
  video_to_video: '视频生视频',
  other: '其他',
};

export function laneToVideoFlatFamily(lane) {
  const L = String(lane || '');
  if (!L) return 'other';
  for (const [fam, list] of Object.entries(LANE_FAMILY_BUCKETS)) {
    if (list.includes(L)) return fam;
  }
  return 'other';
}

/** 按文生→图生→视频生顺序分组，组内保持 tiers 原顺序 */
export function groupVideoFlatTiersByFamily(tiers) {
  const buckets = {
    text_to_video: [],
    image_to_video: [],
    video_to_video: [],
    other: [],
  };
  (Array.isArray(tiers) ? tiers : []).forEach((row) => {
    const fam = laneToVideoFlatFamily(row.lane);
    buckets[fam].push(row);
  });
  return VIDEO_FLAT_FAMILY_ORDER.filter((f) => buckets[f].length > 0).map(
    (f) => ({ family: f, rows: buckets[f] }),
  );
}

export function pickVideoFlatClipHintForChannel(modelData, channel) {
  if (!channel || !modelData) return null;
  if (channel.video_flat_clip_hint) return channel.video_flat_clip_hint;
  const list = modelData.channel_list || [];
  if (
    list.length === 1 &&
    Number(list[0]?.channel_id) === Number(channel?.channel_id)
  ) {
    return modelData.video_flat_clip_hint || null;
  }
  return null;
}

export function hasVideoFlatClipTierTable(hint) {
  if (!hint) return false;
  const n = Number(hint.tier_count);
  if (Number.isFinite(n) && n > 0) return true;
  return Array.isArray(hint.tiers) && hint.tiers.length > 0;
}
