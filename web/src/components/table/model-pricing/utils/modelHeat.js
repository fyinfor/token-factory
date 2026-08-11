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
export const HOT_OVERRIDE_AUTO = 'auto';
export const HOT_OVERRIDE_FORCE_HOT = 'force_hot';
export const HOT_OVERRIDE_FORCE_NOT_HOT = 'force_not_hot';

const MANUAL_HOT_SCORE_BASE = 8000000000000000;
const MANUAL_HOT_RANK_STEP = 1000000000;
const UNRANKED_MANUAL_HOT = 1000000;

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
    channel?.channel_id,
    channel?.ChannelID,
    channel?.id,
    channel?.ChannelNo,
    channel?.channel_no,
  ].find(
    (value) =>
      value !== undefined && value !== null && String(value).trim() !== '',
  );
  return identity === undefined ? '' : String(identity).trim();
};

const getHotOverride = (channel) =>
  String(channel?.hot_override ?? channel?.HotOverride ?? '')
    .trim()
    .toLowerCase();

const getManualRank = (channel) => {
  const rank = Number(channel?.hot_manual_rank ?? channel?.HotManualRank ?? 0);
  return Number.isFinite(rank) && rank > 0
    ? Math.min(Math.floor(rank), UNRANKED_MANUAL_HOT)
    : UNRANKED_MANUAL_HOT;
};

export const getChannelHeatScore = (channel) => {
  const requestCount = Number(
    channel?.channel_heat_score ??
      channel?.ChannelHeatScore ??
      channel?.req_count_7d ??
      channel?.RequestCount7d ??
      channel?.auto_req_count ??
      channel?.AutoReqCount ??
      0,
  );
  return Number.isFinite(requestCount) ? Math.max(0, requestCount) : 0;
};

export const getChannelHeatKey = (model, channel) => {
  const modelName = getModelName(model);
  const channelIdentity = getChannelIdentity(channel);
  if (!modelName || !channelIdentity) return '';
  return `${channelIdentity}:${modelName}`;
};

const getManualHotScore = (channel) => {
  const rank = getManualRank(channel);
  return (
    MANUAL_HOT_SCORE_BASE -
    rank * MANUAL_HOT_RANK_STEP +
    Math.min(getChannelHeatScore(channel), MANUAL_HOT_RANK_STEP - 1)
  );
};

export const getTopHotChannels = (
  models,
  limit = HOME_HOT_CHANNEL_LIMIT,
  filters = {},
) => {
  const modelCandidates = new Map();

  (Array.isArray(models) ? models : []).forEach((model, modelIndex) => {
    getChannelList(model).forEach((channel, channelIndex) => {
      if (!channelMatchesHeatFilters(channel, filters)) return;
      const key = getChannelHeatKey(model, channel);
      const score = getChannelHeatScore(channel);
      if (!key) return;

      const modelName = getModelName(model);
      if (!modelCandidates.has(modelName)) {
        modelCandidates.set(modelName, {
          modelName,
          modelIndex,
          forcedChannels: [],
          bestAutoChannel: null,
        });
      }
      const candidate = modelCandidates.get(modelName);
      candidate.modelIndex = Math.min(candidate.modelIndex, modelIndex);

      const entry = {
        key,
        score,
        modelName,
        channel,
        modelIndex,
        channelIndex,
      };

      const override = getHotOverride(channel);
      if (override === HOT_OVERRIDE_FORCE_HOT) {
        entry.manualRank = getManualRank(channel);
        entry.effectiveScore = getManualHotScore(channel);
        candidate.forcedChannels.push(entry);
        return;
      }
      if (override === HOT_OVERRIDE_FORCE_NOT_HOT || score <= 0) return;

      const existing = candidate.bestAutoChannel;
      if (
        !existing ||
        score > existing.score ||
        (score === existing.score && channelIndex < existing.channelIndex)
      ) {
        entry.effectiveScore = score;
        candidate.bestAutoChannel = entry;
      }
    });
  });

  const forcedModels = [];
  const automaticModels = [];
  modelCandidates.forEach((candidate) => {
    if (candidate.forcedChannels.length > 0) {
      candidate.forcedChannels.sort(
        (a, b) =>
          a.manualRank - b.manualRank ||
          b.score - a.score ||
          a.channelIndex - b.channelIndex,
      );
      forcedModels.push({
        ...candidate,
        primary: candidate.forcedChannels[0],
      });
      return;
    }
    if (candidate.bestAutoChannel) {
      automaticModels.push({
        ...candidate,
        primary: candidate.bestAutoChannel,
      });
    }
  });

  forcedModels.sort(
    (a, b) =>
      b.primary.effectiveScore - a.primary.effectiveScore ||
      a.modelIndex - b.modelIndex,
  );
  automaticModels.sort(
    (a, b) => b.primary.score - a.primary.score || a.modelIndex - b.modelIndex,
  );

  const normalizedLimit = Number.isFinite(Number(limit))
    ? Math.max(0, Math.floor(Number(limit)))
    : HOME_HOT_CHANNEL_LIMIT;
  const automaticSlots = Math.max(0, normalizedLimit - forcedModels.length);
  const selectedModels = [
    ...forcedModels,
    ...automaticModels.slice(0, automaticSlots),
  ];
  const entries = selectedModels.map((candidate) => candidate.primary);
  const scoreMap = new Map();
  const primaryChannelMap = new Map();
  const sourceMap = new Map();

  selectedModels.forEach((candidate) => {
    const isManual = candidate.forcedChannels.length > 0;
    const selectedChannels = isManual
      ? candidate.forcedChannels
      : [candidate.primary];
    selectedChannels.forEach((entry) => {
      scoreMap.set(entry.key, entry.effectiveScore);
    });
    primaryChannelMap.set(
      candidate.modelName,
      getChannelIdentity(candidate.primary.channel),
    );
    sourceMap.set(candidate.modelName, isManual ? 'manual' : 'auto');
  });

  return {
    entries,
    scoreMap,
    primaryChannelMap,
    sourceMap,
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
