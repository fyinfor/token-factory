import { describe, expect, test } from 'bun:test';
import {
  mergeVideoPricingFromJsonEditor,
  parseVideoPricingJsonText,
  stringifyVideoPricingJsonEditor,
} from './videoPricingJson';

const baseModel = {
  name: 'seedance-1.0',
  videoBillingMode: 'per-token',
  videoPriceUnit: 'USD',
  videoFixedPrice: '',
  videoSimilarityThreshold: '',
  videoTextToVideoRules: [
    {
      resolution: '1280x720',
      audioPricingEnabled: false,
      tokenPrice: '0.15',
      videoPrice: '',
      noAudioPrice: '',
      withAudioPrice: '',
    },
  ],
  videoImageToVideoRules: [],
  videoUploadRules: [],
  videoGenerateRules: [],
  videoUpscaleRules: [
    {
      resolution: '1280x720',
      sourceResolution: '854x480',
      tokenPrice: '0.02',
    },
  ],
};

describe('video pricing JSON editor', () => {
  test('round-trips per-token billing and upscale rows', () => {
    const text = stringifyVideoPricingJsonEditor(baseModel);
    const parsed = parseVideoPricingJsonText(text);
    expect(parsed.ok).toBe(true);
    expect(parsed.data.billing_mode).toBe('per-token');
    expect(parsed.data.text_to_video['720']).toEqual([0.15]);
    expect(parsed.data.video_upscale).toEqual([
      { source: '480', target: '720', price: 0.02 },
    ]);

    const merged = mergeVideoPricingFromJsonEditor(baseModel, parsed.data);
    expect(merged.ok).toBe(true);
    expect(merged.model.videoBillingMode).toBe('per-token');
    expect(merged.model.videoTextToVideoRules[0].tokenPrice).toBe('0.15');
    expect(merged.model.videoUpscaleRules).toEqual([
      {
        resolution: '1280x720',
        sourceResolution: '854x480',
        tokenPrice: '0.02',
        videoPrice: '',
        noAudioPrice: '',
        withAudioPrice: '',
        audioPricingEnabled: false,
      },
    ]);
  });

  test('keeps existing upscale rows when JSON omits video_upscale', () => {
    const merged = mergeVideoPricingFromJsonEditor(baseModel, {
      billing_mode: 'per-token',
      text_to_video: { '1080': [0.31] },
    });
    expect(merged.ok).toBe(true);
    expect(merged.model.videoTextToVideoRules[0].tokenPrice).toBe('0.31');
    expect(merged.model.videoUpscaleRules).toEqual(baseModel.videoUpscaleRules);
  });

  test('accepts backend video_upscale_per_second payload', () => {
    const merged = mergeVideoPricingFromJsonEditor(baseModel, {
      billing_mode: 'per-token',
      video_upscale_per_second: [
        { resolution: '1080p', source_resolution: '720p', price: 0.08 },
      ],
    });
    expect(merged.ok).toBe(true);
    expect(merged.model.videoUpscaleRules[0].resolution).toBe('1920x1080');
    expect(merged.model.videoUpscaleRules[0].sourceResolution).toBe('1280x720');
    expect(merged.model.videoUpscaleRules[0].tokenPrice).toBe('0.08');
  });
});
