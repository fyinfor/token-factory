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

import { describe, expect, test } from 'bun:test';
import {
  getChannelHeatKey,
  getRelevantModelHotScore,
  getTopHotChannels,
  HOT_OVERRIDE_FORCE_HOT,
  HOT_OVERRIDE_FORCE_NOT_HOT,
  isTopHotModel,
} from './modelHeat';

const channel = (channelId, requestCount, extra = {}) => ({
  channel_id: channelId,
  auto_req_count: requestCount,
  ...extra,
});

describe('homepage channel-model hot ranking', () => {
  test('selects automatic hot entries by distinct model', () => {
    const models = [
      {
        model_name: 'model-a',
        channel_list: [channel(1, 100), channel(2, 90)],
      },
      { model_name: 'model-b', channel_list: [channel(3, 80)] },
    ];

    const result = getTopHotChannels(models, 1);

    expect(result.entries).toHaveLength(1);
    expect(result.entries[0].modelName).toBe('model-a');
    expect(result.primaryChannelMap.get('model-a')).toBe('1');
    expect(isTopHotModel(models[0], result.scoreMap)).toBe(true);
    expect(isTopHotModel(models[1], result.scoreMap)).toBe(false);
  });

  test('manual hot models take priority and automatic ranking fills remaining slots', () => {
    const models = [
      { model_name: 'model-a', channel_list: [channel(1, 100)] },
      { model_name: 'model-b', channel_list: [channel(2, 80)] },
      {
        model_name: 'model-c',
        channel_list: [
          channel(3, 1, {
            hot_override: HOT_OVERRIDE_FORCE_HOT,
            hot_manual_rank: 1,
          }),
        ],
      },
    ];

    const result = getTopHotChannels(models, 2);

    expect(result.entries.map((entry) => entry.modelName)).toEqual([
      'model-c',
      'model-a',
    ]);
    expect(result.sourceMap.get('model-c')).toBe('manual');
    expect(result.sourceMap.get('model-a')).toBe('auto');
    expect(
      getRelevantModelHotScore(models[2], result.scoreMap),
    ).toBeGreaterThan(getRelevantModelHotScore(models[0], result.scoreMap));
  });

  test('force-not-hot excludes only the configured channel', () => {
    const model = {
      model_name: 'model-a',
      channel_list: [
        channel(1, 1000, { hot_override: HOT_OVERRIDE_FORCE_NOT_HOT }),
        channel(2, 50),
      ],
    };

    const result = getTopHotChannels([model], 1);

    expect(result.primaryChannelMap.get('model-a')).toBe('2');
    expect(
      result.scoreMap.has(getChannelHeatKey(model, model.channel_list[0])),
    ).toBe(false);
    expect(
      result.scoreMap.has(getChannelHeatKey(model, model.channel_list[1])),
    ).toBe(true);
  });

  test('supplier filters can hide a manual channel and fall back to automatic ranking', () => {
    const model = {
      model_name: 'model-a',
      channel_list: [
        channel(1, 10, {
          supplier_alias: 'P1',
        }),
        channel(2, 1, {
          supplier_alias: 'P2',
          hot_override: HOT_OVERRIDE_FORCE_HOT,
          hot_manual_rank: 1,
        }),
      ],
    };

    const result = getTopHotChannels([model], 1, { filterSupplier: 'P1' });

    expect(result.sourceMap.get('model-a')).toBe('auto');
    expect(result.primaryChannelMap.get('model-a')).toBe('1');
  });

  test('all forced models remain hot even when they exceed the automatic limit', () => {
    const models = ['a', 'b', 'c'].map((name, index) => ({
      model_name: name,
      channel_list: [
        channel(index + 1, 0, {
          hot_override: HOT_OVERRIDE_FORCE_HOT,
          hot_manual_rank: index + 1,
        }),
      ],
    }));

    const result = getTopHotChannels(models, 1);

    expect(result.entries).toHaveLength(3);
    expect(result.entries.map((entry) => entry.modelName)).toEqual([
      'a',
      'b',
      'c',
    ]);
  });
});
