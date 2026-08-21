import { describe, expect, test } from 'bun:test';
import {
  getFirstMediaPreviewCandidate,
  isPlaygroundPreviewableUrl,
} from './playgroundMediaPreview';

describe('playground media preview helpers', () => {
  test('uses the first filled URI when a field contains multiple lines', () => {
    expect(
      getFirstMediaPreviewCandidate(
        'https://cdn.example.com/a.png\nhttps://cdn.example.com/b.png',
      ),
    ).toBe('https://cdn.example.com/a.png');
  });

  test('accepts http(s), data, blob and asset URIs', () => {
    expect(isPlaygroundPreviewableUrl('https://cdn.example.com/a.png')).toBe(
      true,
    );
    expect(isPlaygroundPreviewableUrl('data:image/png;base64,abc')).toBe(true);
    expect(isPlaygroundPreviewableUrl('asset://asset-123')).toBe(true);
    expect(isPlaygroundPreviewableUrl('blob:https://example.com/1')).toBe(true);
    expect(isPlaygroundPreviewableUrl('not-a-url')).toBe(false);
    expect(isPlaygroundPreviewableUrl('')).toBe(false);
  });
});
