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

export const LIVE_HOT_FILTER = '__live_hot__';
export const HOME_HOT_CHANNEL_LIMIT = 8;

const getModelName = (model) =>
  String(model?.model_name ?? model?.ModelName ?? '').trim();

const getChannelList = (model) =>
  Array.isArray(model?.channel_list)
    ? model.channel_list
    : Array.isArray(model?.ChannelList)
      ? model.ChannelList
      : [];

const getChannelIdentity = (channel) => {
  const identity = [
    channel?.ChannelNo,
    channel?.channel_no,
    channel?.channel_id,
    channel?.ChannelID,
    channel?.id,
  ].find(
    (value) =>
      value !== undefined && value !== null && String(value).trim() !== '',
  );
  return identity === undefined ? '' : String(identity).trim();
};

// Matches the score shown and used for sorting in the model-heat console.
export const getChannelHeatScore = (channel) => {
  const manualBase = Number(
    channel?.manual_base_req_count ?? channel?.ManualBaseReqCount ?? 0,
  );
  const requestCount = Number(
    channel?.req_count_7d ??
      channel?.RequestCount7d ??
      channel?.auto_req_count ??
      channel?.AutoReqCount ??
      0,
  );
  const channelWeight = Number(
    channel?.channel_sort_weight ??
      channel?.ChannelSortWeight ??
      channel?.sort_weight ??
      channel?.SortWeight ??
      1,
  );

  const base = Number.isFinite(manualBase) ? manualBase : 0;
  const requests = Number.isFinite(requestCount) ? requestCount : 0;
  const weight = Number.isFinite(channelWeight) ? channelWeight : 1;
  return (base + requests) * weight;
};

export const getChannelHeatKey = (model, channel) => {
  const modelName = getModelName(model);
  const channelIdentity = getChannelIdentity(channel);
  if (!modelName || !channelIdentity) return '';
  return `${channelIdentity}:${modelName}`;
};

export const getTopHotChannels = (models, limit = HOME_HOT_CHANNEL_LIMIT) => {
  const rankedChannelMap = new Map();

  (Array.isArray(models) ? models : []).forEach((model, modelIndex) => {
    getChannelList(model).forEach((channel, channelIndex) => {
      const key = getChannelHeatKey(model, channel);
      const score = getChannelHeatScore(channel);
      if (!key || !Number.isFinite(score) || score <= 0) return;

      const entry = {
        key,
        score,
        modelName: getModelName(model),
        channel,
        modelIndex,
        channelIndex,
      };
      const existing = rankedChannelMap.get(key);
      if (!existing || score > existing.score) {
        rankedChannelMap.set(key, entry);
      }
    });
  });

  const rankedChannels = Array.from(rankedChannelMap.values());

  rankedChannels.sort(
    (a, b) =>
      b.score - a.score ||
      a.modelIndex - b.modelIndex ||
      a.channelIndex - b.channelIndex,
  );

  const entries = rankedChannels.slice(0, Math.max(0, limit));
  return {
    entries,
    scoreMap: new Map(entries.map((entry) => [entry.key, entry.score])),
  };
};

export const channelMatchesHeatFilters = (
  channel,
  { filterSupplier = 'all', filterSupplierType = 'all' } = {},
) => {
  if (
    filterSupplier !== 'all' &&
    String(channel?.supplier_alias ?? channel?.SupplierAlias ?? '') !==
      String(filterSupplier)
  ) {
    return false;
  }

  if (
    filterSupplierType !== 'all' &&
    String(channel?.supplier_type ?? channel?.SupplierType ?? '') !==
      String(filterSupplierType)
  ) {
    return false;
  }

  return true;
};

export const getRelevantModelHotScore = (
  model,
  hotChannelScoreMap,
  filters = {},
) => {
  if (!(hotChannelScoreMap instanceof Map)) return 0;

  return getChannelList(model).reduce((highestScore, channel) => {
    if (!channelMatchesHeatFilters(channel, filters)) return highestScore;
    const key = getChannelHeatKey(model, channel);
    const score = Number(hotChannelScoreMap.get(key) ?? 0);
    return Number.isFinite(score)
      ? Math.max(highestScore, score)
      : highestScore;
  }, 0);
};

export const isTopHotModel = (model, hotChannelScoreMap, filters = {}) =>
  getRelevantModelHotScore(model, hotChannelScoreMap, filters) > 0;
