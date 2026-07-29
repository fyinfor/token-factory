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

export const stripOuterMarkdownFence = (content) => {
  const value = String(content || '').trim();
  const match = value.match(/^```(?:markdown|md)?\s*\n([\s\S]*?)\n```$/i);
  return (match ? match[1] : value).trim();
};

export const stripReasoningContent = (content) => {
  let value = String(content || '');
  value = value.replace(
    /<(think|thinking|analysis|reasoning)>[\s\S]*?<\/\1>/gi,
    '',
  );
  const unclosedReasoning = value.search(
    /<(think|thinking|analysis|reasoning)>/i,
  );
  if (unclosedReasoning >= 0) {
    value = value.slice(0, unclosedReasoning);
  }
  return value.trim();
};

export const readAssistantContent = (response) => {
  const content = response?.data?.choices?.[0]?.message?.content;
  if (typeof content === 'string') {
    return stripOuterMarkdownFence(stripReasoningContent(content));
  }
  if (Array.isArray(content)) {
    return stripOuterMarkdownFence(
      stripReasoningContent(
        content
          .filter(
            (part) =>
              typeof part === 'string' ||
              !['analysis', 'reasoning', 'thinking'].includes(part?.type),
          )
          .map((part) => (typeof part === 'string' ? part : part?.text || ''))
          .join(''),
      ),
    );
  }
  return '';
};

export const readAssistantStreamChunk = (payload) => {
  const choice = payload?.choices?.[0];
  const content = choice?.delta?.content ?? choice?.message?.content;
  if (typeof content === 'string') return content;
  if (!Array.isArray(content)) return '';
  return content
    .filter(
      (part) =>
        typeof part === 'string' ||
        !['analysis', 'reasoning', 'thinking'].includes(part?.type),
    )
    .map((part) => (typeof part === 'string' ? part : part?.text || ''))
    .join('');
};

export const getMarkdownTableColumnCounts = (content) => {
  const lines = String(content || '').split(/\r?\n/);
  const counts = [];
  let inCodeFence = false;
  lines.forEach((line, index) => {
    if (/^\s*```/.test(line)) {
      inCodeFence = !inCodeFence;
      return;
    }
    if (inCodeFence || index + 1 >= lines.length) return;
    const nextLine = lines[index + 1];
    if (
      /^\s*\|.+\|\s*$/.test(line) &&
      /^\s*\|(?:\s*:?-{3,}:?\s*\|)+\s*$/.test(nextLine)
    ) {
      counts.push(
        line
          .trim()
          .slice(1, -1)
          .split(/(?<!\\)\|/).length,
      );
    }
  });
  return counts;
};

export const getMarkdownCodeBlockCount = (content) =>
  Math.floor((String(content || '').match(/^\s*```/gm) || []).length / 2);

export const isValidPolishedMarkdown = (source, result) => {
  const h1Count = (result.match(/^#\s+.+$/gm) || []).length;
  const fenceCount = (result.match(/^\s*```/gm) || []).length;
  const codeGroupOpenCount = (result.match(/^\s*:::code-group\s*$/gm) || [])
    .length;
  const codeGroupCloseCount = (result.match(/^\s*:::\s*$/gm) || []).length;
  const maxTableColumns = Math.max(0, ...getMarkdownTableColumnCounts(result));
  const hasRequiredTemplates = ['base_url', 'model', 'api_key'].every((name) =>
    result.includes(`{{${name}}}`),
  );
  const sourceCodeBlockCount = getMarkdownCodeBlockCount(source);
  const resultCodeBlockCount = getMarkdownCodeBlockCount(result);
  const reasonableLength =
    source.length < 8000
      ? result.length <= Math.max(source.length * 1.2, 4000)
      : result.length >= source.length * 0.45 &&
        result.length <= source.length * 0.85;
  const preservesExampleCoverage =
    source.length < 8000 ||
    sourceCodeBlockCount < 8 ||
    resultCodeBlockCount >= Math.min(8, Math.ceil(sourceCodeBlockCount * 0.5));
  const preservesCompletedResponse =
    !/"status"\s*:\s*"completed"/i.test(source) ||
    /"status"\s*:\s*"completed"/i.test(result);

  return (
    h1Count === 1 &&
    result.startsWith('# ') &&
    fenceCount % 2 === 0 &&
    codeGroupOpenCount === codeGroupCloseCount &&
    maxTableColumns <= 4 &&
    hasRequiredTemplates &&
    reasonableLength &&
    preservesExampleCoverage &&
    preservesCompletedResponse &&
    !/<\/?(?:think|thinking|analysis|reasoning)>/i.test(result)
  );
};
