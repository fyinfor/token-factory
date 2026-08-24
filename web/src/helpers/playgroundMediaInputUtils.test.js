import { describe, expect, test } from 'bun:test';
import {
  PLAYGROUND_MEDIA_UNLIMITED_COUNT,
  PLAYGROUND_VIDEO_FRAME_MAX_COUNT,
} from '../constants/playground.constants';
import {
  appendMediaUrlsWithLimit,
  appendUploadedMediaUrl,
  canAcceptMoreMediaUrls,
  canAddMediaUrlRow,
  isUnlimitedMediaCount,
} from './playgroundMediaInputUtils';

describe('playground media count helpers', () => {
  test('unlimited count never blocks add/upload', () => {
    expect(isUnlimitedMediaCount(PLAYGROUND_MEDIA_UNLIMITED_COUNT)).toBe(true);
    const urls = ['https://a.png', 'https://b.png', 'https://c.png'];
    expect(canAddMediaUrlRow(urls, PLAYGROUND_MEDIA_UNLIMITED_COUNT)).toBe(
      true,
    );
    expect(
      canAcceptMoreMediaUrls(urls, PLAYGROUND_MEDIA_UNLIMITED_COUNT),
    ).toBe(true);
  });

  test('frames tab cannot add a third filled image', () => {
    const urls = ['https://a.png', 'https://b.png'];
    expect(canAddMediaUrlRow(urls, PLAYGROUND_VIDEO_FRAME_MAX_COUNT)).toBe(
      false,
    );
    expect(
      canAcceptMoreMediaUrls(urls, PLAYGROUND_VIDEO_FRAME_MAX_COUNT),
    ).toBe(false);
    const appended = appendUploadedMediaUrl(
      urls,
      'https://c.png',
      PLAYGROUND_VIDEO_FRAME_MAX_COUNT,
    );
    expect(appended.ok).toBe(false);
  });

  test('appendMediaUrlsWithLimit keeps extra items out of frames list', () => {
    const result = appendMediaUrlsWithLimit(
      [''],
      ['https://a.png', 'https://b.png', 'https://c.png'],
      PLAYGROUND_VIDEO_FRAME_MAX_COUNT,
    );
    expect(result.added).toBe(2);
    expect(result.skipped).toBe(1);
    expect(result.urls.filter(Boolean)).toEqual([
      'https://a.png',
      'https://b.png',
    ]);
  });
});
