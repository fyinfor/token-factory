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
  getMarkdownCodeBlockCount,
  getMarkdownTableColumnCounts,
  isValidPolishedMarkdown,
  readAssistantContent,
  readAssistantStreamChunk,
  stripReasoningContent,
} from './documentAiUtils';

const validDocument = `# API 文档

{{base_url}} {{model}} {{api_key}}

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| model | string | 是 | 模型 |

\`\`\`http
POST /v1/demo
\`\`\`
`;

describe('document AI result handling', () => {
  test('removes complete and unclosed reasoning blocks', () => {
    expect(stripReasoningContent('<think>draft</think>\n\n# Final')).toBe(
      '# Final',
    );
    expect(stripReasoningContent('# Prefix\n<think>unfinished')).toBe(
      '# Prefix',
    );
  });

  test('ignores structured reasoning parts and outer Markdown fences', () => {
    const result = readAssistantContent({
      data: {
        choices: [
          {
            message: {
              content: [
                { type: 'reasoning', text: 'private draft' },
                {
                  type: 'text',
                  text: `\`\`\`markdown\n${validDocument}\n\`\`\``,
                },
              ],
            },
          },
        ],
      },
    });
    expect(result).toBe(validDocument.trim());
  });

  test('reads text deltas and ignores structured stream reasoning', () => {
    expect(
      readAssistantStreamChunk({
        choices: [{ delta: { content: '# API' } }],
      }),
    ).toBe('# API');
    expect(
      readAssistantStreamChunk({
        choices: [
          {
            delta: {
              content: [
                { type: 'reasoning', text: 'draft' },
                { type: 'text', text: 'final' },
              ],
            },
          },
        ],
      }),
    ).toBe('final');
  });

  test('counts table columns outside code fences', () => {
    expect(getMarkdownTableColumnCounts(validDocument)).toEqual([4]);
    expect(
      getMarkdownTableColumnCounts(
        '| A | B | C | D | E |\n| --- | --- | --- | --- | --- |',
      ),
    ).toEqual([5]);
    expect(getMarkdownCodeBlockCount(validDocument)).toBe(1);
  });

  test('accepts a useful, moderately condensed result', () => {
    const source = `${validDocument}\n${'原始内容'.repeat(2200)}`;
    const result = `${validDocument}\n${'整理内容'.repeat(1200)}`;
    expect(isValidPolishedMarkdown(source, result)).toBe(true);
  });

  test('rejects leaked reasoning, invalid structure, and weak compression', () => {
    const source = `${validDocument}\n${'原始内容'.repeat(2200)}`;
    expect(
      isValidPolishedMarkdown(source, `${validDocument}\n# 第二份文档`),
    ).toBe(false);
    expect(
      isValidPolishedMarkdown(
        source,
        validDocument.replace(
          '| 字段 | 类型 | 必填 | 说明 |',
          '| 字段 | 类型 | 必填 | 位置 | 说明 |',
        ),
      ),
    ).toBe(false);
    expect(
      isValidPolishedMarkdown(source, `${validDocument}\n\`\`\`json`),
    ).toBe(false);
    expect(
      isValidPolishedMarkdown(
        source,
        `${validDocument}\n${'几乎没有精简'.repeat(1800)}`,
      ),
    ).toBe(false);
  });

  test('rejects long documents that lose examples or completed responses', () => {
    const requestBlock = (index) =>
      `\n\n\`\`\`http\nPOST /v1/demo/${index}\n\`\`\``;
    const source = `${validDocument}\n\n\`\`\`json\n{"status":"completed"}\n\`\`\`${Array.from(
      { length: 9 },
      (_, index) => requestBlock(index),
    ).join('')}\n${'原始内容'.repeat(2000)}`;
    const paddedResult = (body) => `${body}\n${'整理内容'.repeat(1300)}`;
    const tooFewExamples = `${validDocument}\n\n\`\`\`json\n{"status":"completed"}\n\`\`\``;
    const enoughExamples = `${tooFewExamples}${Array.from(
      { length: 5 },
      (_, index) => requestBlock(index),
    ).join('')}`;

    expect(isValidPolishedMarkdown(source, paddedResult(tooFewExamples))).toBe(
      false,
    );
    expect(
      isValidPolishedMarkdown(
        source,
        paddedResult(enoughExamples).replace('{"status":"completed"}', '{}'),
      ),
    ).toBe(false);
    expect(isValidPolishedMarkdown(source, paddedResult(enoughExamples))).toBe(
      true,
    );
  });
});
