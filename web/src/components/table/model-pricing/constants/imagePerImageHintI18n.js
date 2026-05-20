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

/** 与后端 image_per_image_hint.lane 一致 */
export const IMAGE_PER_IMAGE_LANE_I18N_KEY = {
  text_to_image: '文生图',
  image_to_image: '图生图',
};

export const IMAGE_PER_IMAGE_FAMILY_ORDER = ['text_to_image', 'image_to_image'];

export const IMAGE_PER_IMAGE_FAMILY_TITLE_KEY = {
  text_to_image: '文生图',
  image_to_image: '图生图',
};

export function laneToImagePerImageFamily(lane) {
  const L = String(lane || '');
  if (L === 'text_to_image' || L === 'image_to_image') return L;
  return 'text_to_image';
}

export function groupImagePerImageTiersByFamily(tiers) {
  const buckets = {
    text_to_image: [],
    image_to_image: [],
  };
  (tiers || []).forEach((row) => {
    const fam = laneToImagePerImageFamily(row.lane);
    if (buckets[fam]) {
      buckets[fam].push(row);
    }
  });
  return IMAGE_PER_IMAGE_FAMILY_ORDER.filter((f) => buckets[f].length > 0).map(
    (f) => ({ family: f, rows: buckets[f] }),
  );
}

export function pickImagePerImageHintForChannel(modelData, channel) {
  if (!channel || !modelData) return null;
  if (channel.image_per_image_hint) return channel.image_per_image_hint;
  const list = modelData.channel_list || [];
  if (
    list.length === 1 &&
    Number(list[0]?.channel_id) === Number(channel?.channel_id)
  ) {
    return modelData.image_per_image_hint || null;
  }
  return null;
}

export function hasImagePerImageTierTable(hint) {
  if (!hint) return false;
  const n = Number(hint.tier_count);
  if (Number.isFinite(n) && n > 0) return true;
  return Array.isArray(hint.tiers) && hint.tiers.length > 0;
}
