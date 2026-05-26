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

import React, { useCallback, useMemo, useRef } from 'react';
import './JsonCodeEditor.css';

const escapeHtml = (str) =>
  String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;');

const findLineCommentStart = (line) => {
  let inString = false;
  let escape = false;
  for (let i = 0; i < line.length - 1; i += 1) {
    const c = line[i];
    if (inString) {
      if (escape) {
        escape = false;
      } else if (c === '\\') {
        escape = true;
      } else if (c === '"') {
        inString = false;
      }
      continue;
    }
    if (c === '"') {
      inString = true;
      continue;
    }
    if (c === '/' && line[i + 1] === '/') {
      return i;
    }
  }
  return -1;
};

const highlightJsonTokens = (str) => {
  const tokenRegex =
    /("(\\u[a-zA-Z0-9]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\d+(?:\.\d*)?(?:[eE][+-]?\d+)?)/g;

  let result = '';
  let lastIndex = 0;
  let match = tokenRegex.exec(str);

  while (match !== null) {
    result += escapeHtml(str.slice(lastIndex, match.index));
    const token = match[0];
    let className = 'json-code-number';
    if (/^"/.test(token)) {
      className = /:$/.test(token) ? 'json-code-key' : 'json-code-string';
    } else if (/true|false|null/.test(token)) {
      className = 'json-code-keyword';
    }
    result += `<span class="${className}">${escapeHtml(token)}</span>`;
    lastIndex = tokenRegex.lastIndex;
    match = tokenRegex.exec(str);
  }

  result += escapeHtml(str.slice(lastIndex));
  return result;
};

export const highlightJsonWithComments = (str) => {
  if (!str) return '';
  return str
    .split('\n')
    .map((line) => {
      const commentStart = findLineCommentStart(line);
      if (commentStart >= 0) {
        const codePart = line.slice(0, commentStart);
        const commentPart = line.slice(commentStart);
        return `${highlightJsonTokens(codePart)}<span class="json-code-comment">${escapeHtml(commentPart)}</span>`;
      }
      return highlightJsonTokens(line);
    })
    .join('\n');
};

const JsonCodeEditor = ({
  value = '',
  onChange,
  placeholder = '',
  minRows = 18,
  maxRows = 36,
  className = '',
}) => {
  const textareaRef = useRef(null);
  const preRef = useRef(null);

  const highlightedHtml = useMemo(
    () => highlightJsonWithComments(value),
    [value],
  );

  const lineCount = useMemo(() => {
    const lines = (value || '').split('\n').length;
    return Math.min(Math.max(lines, minRows), maxRows);
  }, [value, minRows, maxRows]);

  const syncScroll = useCallback(() => {
    const textarea = textareaRef.current;
    const pre = preRef.current;
    if (!textarea || !pre) return;
    pre.scrollTop = textarea.scrollTop;
    pre.scrollLeft = textarea.scrollLeft;
  }, []);

  const handleChange = useCallback(
    (event) => {
      onChange?.(event.target.value);
    },
    [onChange],
  );

  const handleKeyDown = useCallback((event) => {
    if (event.key !== 'Tab') return;
    event.preventDefault();
    const textarea = textareaRef.current;
    if (!textarea) return;
    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;
    const nextValue = `${value.slice(0, start)}  ${value.slice(end)}`;
    onChange?.(nextValue);
    requestAnimationFrame(() => {
      textarea.selectionStart = start + 2;
      textarea.selectionEnd = start + 2;
    });
  }, [onChange, value]);

  const editorHeight = Math.max(lineCount * 1.5 * 13 + 24, minRows * 1.5 * 13 + 24);

  return (
    <div
      className={`json-code-editor ${className}`.trim()}
      style={{ height: `${editorHeight}px`, minHeight: `${editorHeight}px` }}
    >
      <pre
        ref={preRef}
        className='json-code-editor__highlight'
        aria-hidden='true'
        dangerouslySetInnerHTML={{ __html: highlightedHtml + '\n' }}
      />
      <textarea
        ref={textareaRef}
        className='json-code-editor__input'
        value={value}
        onChange={handleChange}
        onScroll={syncScroll}
        onKeyDown={handleKeyDown}
        placeholder={placeholder}
        spellCheck={false}
        autoComplete='off'
        autoCorrect='off'
        autoCapitalize='off'
      />
    </div>
  );
};

export default JsonCodeEditor;
