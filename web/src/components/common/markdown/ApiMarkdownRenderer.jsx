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

import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Select, Tabs } from '@douyinfe/semi-ui';
import MarkdownRenderer from './MarkdownRenderer';

const CODE_GROUP_PATTERN = /:::code-group\s*\n([\s\S]*?)\n:::/g;
const FENCED_CODE_PATTERN =
  /```([^\s`\[]+)?(?:\s+\[([^\]\n]+)\])?\s*\n([\s\S]*?)\n```/g;

const defaultCodeLabel = (language) => {
  const normalized = String(language || '').toLowerCase();
  const labels = {
    bash: 'cURL',
    shell: 'cURL',
    curl: 'cURL',
    js: 'JavaScript',
    javascript: 'JavaScript',
    py: 'Python',
    python: 'Python',
    java: 'Java',
    go: 'Go',
    csharp: 'C#',
    cs: 'C#',
  };
  return labels[normalized] || language || 'Code';
};

const parseCodeGroup = (source) => {
  const examples = [];
  let match;
  while ((match = FENCED_CODE_PATTERN.exec(source)) !== null) {
    examples.push({
      language: match[1] || 'text',
      label: String(match[2] || defaultCodeLabel(match[1])).trim(),
      code: match[3],
    });
  }
  FENCED_CODE_PATTERN.lastIndex = 0;
  return examples;
};

const splitDocument = (content) => {
  const segments = [];
  let cursor = 0;
  let match;
  while ((match = CODE_GROUP_PATTERN.exec(content)) !== null) {
    if (match.index > cursor) {
      segments.push({
        type: 'markdown',
        content: content.slice(cursor, match.index),
      });
    }
    const examples = parseCodeGroup(match[1]);
    segments.push(
      examples.length > 0
        ? { type: 'code-group', examples }
        : { type: 'markdown', content: match[0] },
    );
    cursor = match.index + match[0].length;
  }
  CODE_GROUP_PATTERN.lastIndex = 0;
  if (cursor < content.length) {
    segments.push({ type: 'markdown', content: content.slice(cursor) });
  }
  return segments;
};

const extractHeadings = (content) =>
  String(content || '')
    .replace(/```[\s\S]*?```/g, '')
    .split(/\r?\n/)
    .map((line) => line.match(/^(#{1,3})\s+(.+?)\s*#*$/))
    .filter(Boolean)
    .map((match, index) => ({
      value: index,
      level: match[1].length,
      label: match[2]
        .replace(/\[([^\]]+)\]\([^\)]+\)/g, '$1')
        .replace(/[*_`~]/g, '')
        .trim(),
    }));

const ApiMarkdownRenderer = ({
  content = '',
  t,
  showToc = false,
  scrollContainerRef,
  contained = false,
  beforeContent = null,
  toolbarContent = null,
}) => {
  const contentRef = useRef(null);
  const documentRef = useRef(null);
  const scrollFrameRef = useRef();
  const [selectedHeading, setSelectedHeading] = useState();
  const segments = useMemo(
    () => splitDocument(String(content || '')),
    [content],
  );
  const headings = useMemo(() => extractHeadings(content), [content]);

  useEffect(() => {
    setSelectedHeading(undefined);
    if (scrollFrameRef.current) cancelAnimationFrame(scrollFrameRef.current);
    scrollFrameRef.current = undefined;
  }, [content]);

  useEffect(
    () => () => {
      if (scrollFrameRef.current) cancelAnimationFrame(scrollFrameRef.current);
    },
    [],
  );

  const scrollToHeading = (index) => {
    const target =
      documentRef.current?.querySelectorAll('h1, h2, h3')?.[Number(index)];
    if (!target) return;
    const scrollContainer =
      (contained ? contentRef.current : scrollContainerRef?.current) ||
      target.closest('[data-api-docs-scroll-container="true"]') ||
      target.closest('.semi-sidesheet-body');
    if (!scrollContainer) {
      target.scrollIntoView({ behavior: 'auto', block: 'start' });
      return;
    }
    const containerRect = scrollContainer.getBoundingClientRect();
    const targetRect = target.getBoundingClientRect();
    scrollContainer.scrollTo({
      top: scrollContainer.scrollTop + targetRect.top - containerRect.top - 16,
      behavior: 'auto',
    });
  };

  const selectHeading = (index) => {
    setSelectedHeading(index);
    if (index === undefined || index === null) {
      return;
    }
    if (scrollFrameRef.current) cancelAnimationFrame(scrollFrameRef.current);
    scrollFrameRef.current = requestAnimationFrame(() => {
      scrollFrameRef.current = requestAnimationFrame(() => {
        scrollFrameRef.current = undefined;
        scrollToHeading(index);
      });
    });
  };

  return (
    <div
      className={`api-markdown-renderer ${contained ? 'api-markdown-contained' : ''}`}
    >
      {toolbarContent || (showToc && headings.length > 1) ? (
        <div className='api-markdown-toolbar'>
          {toolbarContent}
          {showToc && headings.length > 1 ? (
            <div className='api-markdown-toc'>
              <Select
                showClear
                size='small'
                zIndex={1600}
                dropdownClassName='api-markdown-toc-dropdown'
                prefix={t?.('目录') || 'Contents'}
                placeholder={t?.('跳转到章节') || 'Jump to section'}
                value={selectedHeading}
                optionList={headings.map((heading) => ({
                  value: heading.value,
                  label: `${heading.level > 1 ? '  '.repeat(heading.level - 1) : ''}${heading.label}`,
                }))}
                onChange={selectHeading}
              />
            </div>
          ) : null}
        </div>
      ) : null}
      <div
        ref={contentRef}
        className='api-markdown-content'
        data-api-docs-scroll-container={contained ? 'true' : undefined}
      >
        {beforeContent}
        <div ref={documentRef} className='api-markdown-document'>
          {segments.map((segment, index) =>
            segment.type === 'code-group' ? (
              <div className='api-markdown-code-group' key={`group-${index}`}>
                <Tabs type='line' keepDOM={false}>
                  {segment.examples.map((example, exampleIndex) => (
                    <Tabs.TabPane
                      key={`${example.label}-${exampleIndex}`}
                      itemKey={`${index}-${exampleIndex}`}
                      tab={example.label}
                    >
                      <MarkdownRenderer
                        content={`\`\`\`${example.language}\n${example.code}\n\`\`\``}
                      />
                    </Tabs.TabPane>
                  ))}
                </Tabs>
              </div>
            ) : segment.content.trim() ? (
              <MarkdownRenderer
                key={`markdown-${index}`}
                content={segment.content}
              />
            ) : null,
          )}
        </div>
      </div>
    </div>
  );
};

export default ApiMarkdownRenderer;
