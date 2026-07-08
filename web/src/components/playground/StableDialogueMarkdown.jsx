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

import React, {
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import RehypeHighlight from 'rehype-highlight';
import copyText from 'copy-text-to-clipboard';
import { IconCopyStroked, IconTick } from '@douyinfe/semi-icons';
import * as SemiMarkdownComponents from '@douyinfe/semi-ui/lib/es/markdownRender/components';
import { escapeHtmlInMarkdown } from '@douyinfe/semi-foundation/lib/es/utils/escapeHtml';
import 'highlight.js/styles/github.css';

const STREAM_MARKDOWN_UPDATE_MS = 80;
const REHYPE_PLUGINS = [
  [
    RehypeHighlight,
    {
      aliases: {
        bash: ['shell', 'sh'],
        csharp: ['cs'],
        javascript: ['js', 'jsx'],
        markdown: ['md'],
        typescript: ['ts', 'tsx'],
        yaml: ['yml'],
      },
      detect: false,
      ignoreMissing: true,
    },
  ],
];

const getCodeLanguage = (className) => {
  const languageClass = String(className || '')
    .split(/\s+/)
    .find((item) => item.startsWith('language-'));
  return languageClass ? languageClass.slice('language-'.length) : '';
};

const omitMarkdownInternalProps = (props) => {
  const rest = { ...props };
  delete rest.node;
  delete rest.inline;
  delete rest.ordered;
  delete rest.depth;
  delete rest.index;
  delete rest.checked;
  delete rest.siblingCount;
  delete rest.sourcePosition;
  return rest;
};

const cleanSemiMarkdownComponents = Object.fromEntries(
  Object.entries(SemiMarkdownComponents).map(([name, Component]) => [
    name,
    (props) => <Component {...omitMarkdownInternalProps(props)} />,
  ]),
);

const useThrottledMarkdownRaw = (raw, enabled) => {
  const [value, setValue] = useState(raw);
  const latestRawRef = useRef(raw);
  const lastUpdateRef = useRef(0);
  const timerRef = useRef(null);

  useEffect(() => {
    latestRawRef.current = raw;

    if (!enabled) {
      if (timerRef.current) {
        window.clearTimeout(timerRef.current);
        timerRef.current = null;
      }
      lastUpdateRef.current = Date.now();
      setValue(raw);
      return;
    }

    const now = Date.now();
    const remaining = STREAM_MARKDOWN_UPDATE_MS - (now - lastUpdateRef.current);

    if (remaining <= 0) {
      if (timerRef.current) {
        window.clearTimeout(timerRef.current);
        timerRef.current = null;
      }
      lastUpdateRef.current = now;
      setValue(raw);
      return;
    }

    if (!timerRef.current) {
      timerRef.current = window.setTimeout(() => {
        timerRef.current = null;
        lastUpdateRef.current = Date.now();
        setValue(latestRawRef.current);
      }, remaining);
    }
  }, [enabled, raw]);

  useEffect(
    () => () => {
      if (timerRef.current) {
        window.clearTimeout(timerRef.current);
      }
    },
    [],
  );

  return value;
};

const getTextFromReactNode = (node) => {
  if (Array.isArray(node)) {
    return node.map(getTextFromReactNode).join('');
  }
  if (React.isValidElement(node)) {
    return getTextFromReactNode(node.props.children);
  }
  return node == null ? '' : String(node);
};

const DialogueCode = React.memo((props) => {
  const [copied, setCopied] = useState(false);
  const { children, className, isBlock, ...restProps } =
    omitMarkdownInternalProps(props);
  const code = useMemo(() => getTextFromReactNode(children), [children]);
  const declaredLanguage = useMemo(
    () => getCodeLanguage(className),
    [className],
  );

  const handleCopy = useCallback(() => {
    copyText(code);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 2000);
  }, [code]);

  if (!isBlock) {
    const InlineCode = SemiMarkdownComponents.code;
    return (
      <InlineCode {...restProps} className={className}>
        {children}
      </InlineCode>
    );
  }

  return (
    <div className='semi-ai-chat-dialogue-code'>
      <div className='semi-ai-chat-dialogue-code-topSlot'>
        <span className='semi-ai-chat-dialogue-code-topSlot-type'>
          {declaredLanguage || 'text'}
        </span>
        <span className='semi-ai-chat-dialogue-code-topSlot-copy'>
          <button
            className='semi-ai-chat-dialogue-code-topSlot-copy-wrapper'
            onClick={handleCopy}
            type='button'
            aria-label='Copy code'
          >
            {copied ? <IconTick /> : <IconCopyStroked />}
          </button>
        </span>
      </div>
      <pre className='playground-dialogue-code-pre'>
        <code {...restProps} className={className}>
          {children}
        </code>
      </pre>
    </div>
  );
});

DialogueCode.displayName = 'DialogueCode';

const StableDialogueMarkdown = ({
  raw,
  components,
  className,
  style,
  escapeHtml = false,
  streaming = false,
  onContentRendered,
}) => {
  const throttledRaw = useThrottledMarkdownRaw(raw || '', streaming);
  const markdownRaw = useMemo(
    () =>
      escapeHtml
        ? escapeHtmlInMarkdown(throttledRaw || '')
        : throttledRaw || '',
    [escapeHtml, throttledRaw],
  );

  const mergedComponents = useMemo(
    () => ({
      ...cleanSemiMarkdownComponents,
      pre: ({ children }) => {
        const child = React.Children.toArray(children)[0];
        return React.isValidElement(child) ? (
          React.cloneElement(child, { isBlock: true })
        ) : (
          <>{children}</>
        );
      },
      code: DialogueCode,
      ...(components || {}),
    }),
    [components],
  );

  useLayoutEffect(() => {
    onContentRendered?.();
  }, [markdownRaw, onContentRendered]);

  return (
    <div
      className={['semi-markdownRender', className].filter(Boolean).join(' ')}
      style={style}
    >
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        rehypePlugins={REHYPE_PLUGINS}
        components={mergedComponents}
      >
        {markdownRaw}
      </ReactMarkdown>
    </div>
  );
};

export default React.memo(StableDialogueMarkdown);
