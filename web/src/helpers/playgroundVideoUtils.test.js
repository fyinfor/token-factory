import { describe, expect, test } from 'bun:test';
import {
  PLAYGROUND_MEDIA_UNLIMITED_COUNT,
  PLAYGROUND_VIDEO_FRAME_MAX_COUNT,
  PLAYGROUND_VIDEO_IMAGE_TABS,
} from '../constants/playground.constants';
import {
  applyVideoFrameMetadata,
  assertExclusiveVideoImagePayload,
  buildImageMediaItems,
  resolvePlaygroundVideoImages,
  validatePlaygroundVideoImageParams,
} from './playgroundVideoUtils';

describe('playground video image exclusivity', () => {
  test('reference tab only emits images, even if frames are also filled', () => {
    const resolved = resolvePlaygroundVideoImages({
      videoImageTab: PLAYGROUND_VIDEO_IMAGE_TABS.REFERENCE,
      imageUrls: ['https://cdn.example.com/ref-1.png', 'https://cdn.example.com/ref-2.png'],
      frameImageUrls: [
        'https://cdn.example.com/first.png',
        'https://cdn.example.com/last.png',
      ],
    });
    expect(resolved.mode).toBe(PLAYGROUND_VIDEO_IMAGE_TABS.REFERENCE);
    expect(resolved.images).toEqual([
      'https://cdn.example.com/ref-1.png',
      'https://cdn.example.com/ref-2.png',
    ]);
    expect(resolved.frameImages).toEqual([]);
    expect(resolved.referenceImages).toHaveLength(2);
  });

  test('frames tab only emits first/last frames, even if reference images are filled', () => {
    const resolved = resolvePlaygroundVideoImages({
      videoImageTab: PLAYGROUND_VIDEO_IMAGE_TABS.FRAMES,
      imageUrls: ['https://cdn.example.com/ref.png'],
      frameImageUrls: [
        'https://cdn.example.com/first.png',
        'https://cdn.example.com/last.png',
      ],
    });
    expect(resolved.mode).toBe(PLAYGROUND_VIDEO_IMAGE_TABS.FRAMES);
    expect(resolved.images).toEqual([]);
    expect(resolved.frameImages).toEqual([
      'https://cdn.example.com/first.png',
      'https://cdn.example.com/last.png',
    ]);
  });

  test('frames metadata does not include reference images', () => {
    const metadata = applyVideoFrameMetadata(
      { duration: 5 },
      [
        'https://cdn.example.com/first.png',
        'https://cdn.example.com/last.png',
        'https://cdn.example.com/extra.png',
      ],
    );
    expect(metadata.first_frame_url).toBe('https://cdn.example.com/first.png');
    expect(metadata.last_frame_url).toBe('https://cdn.example.com/last.png');
    expect(metadata.images).toBeUndefined();
  });

  test('buildImageMediaItems no longer mixes reference and frames', () => {
    const refs = buildImageMediaItems(
      [
        'https://cdn.example.com/a.png',
        'https://cdn.example.com/b.png',
        'https://cdn.example.com/c.png',
      ],
      PLAYGROUND_VIDEO_IMAGE_TABS.REFERENCE,
    );
    expect(refs.every((item) => item.type === 'reference_image')).toBe(true);
    expect(refs).toHaveLength(3);

    const frames = buildImageMediaItems(
      [
        'https://cdn.example.com/first.png',
        'https://cdn.example.com/last.png',
      ],
      PLAYGROUND_VIDEO_IMAGE_TABS.FRAMES,
    );
    expect(frames.map((item) => item.type)).toEqual([
      'first_frame',
      'last_frame',
    ]);
  });

  test('validate rejects more than 2 frame images', () => {
    const result = validatePlaygroundVideoImageParams({
      videoImageTab: PLAYGROUND_VIDEO_IMAGE_TABS.FRAMES,
      frameImageUrls: [
        'https://cdn.example.com/1.png',
        'https://cdn.example.com/2.png\nhttps://cdn.example.com/3.png',
      ],
    });
    expect(result.ok).toBe(false);
    expect(result.message).toContain('2');
  });

  test('assertExclusiveVideoImagePayload blocks mixed request fields', () => {
    const mixed = assertExclusiveVideoImagePayload({
      images: ['https://cdn.example.com/ref.png'],
      first_frame_url: 'https://cdn.example.com/first.png',
      metadata: { last_frame_url: 'https://cdn.example.com/last.png' },
    });
    expect(mixed.ok).toBe(false);

    const framesOnly = assertExclusiveVideoImagePayload({
      images: [],
      first_frame_url: 'https://cdn.example.com/first.png',
    });
    expect(framesOnly.ok).toBe(true);

    const refsOnly = assertExclusiveVideoImagePayload({
      images: ['https://cdn.example.com/ref.png'],
      metadata: { duration: 5 },
    });
    expect(refsOnly.ok).toBe(true);
  });

  test('frame max constant stays at 2 and reference remains unlimited', () => {
    expect(PLAYGROUND_VIDEO_FRAME_MAX_COUNT).toBe(2);
    expect(Number.isFinite(PLAYGROUND_MEDIA_UNLIMITED_COUNT)).toBe(false);
  });
});
