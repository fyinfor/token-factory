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

import React from 'react';
import { Button, Form, Tabs, TextArea, Typography } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import LegalContentRenderer, {
  LEGAL_CONTENT_FORMATS,
  looksLikeHtml,
  normalizeLegalContentFormat,
} from '../legal/LegalContentRenderer';
import SimpleRichTextEditor from '../common/ui/SimpleRichTextEditor';
import './LegalContentEditor.css';

const { Text } = Typography;

const EDITOR_MODE_OPTIONS = [
  { itemKey: LEGAL_CONTENT_FORMATS.markdown, tab: 'Markdown' },
  { itemKey: LEGAL_CONTENT_FORMATS.html, tab: 'HTML' },
  { itemKey: LEGAL_CONTENT_FORMATS.richtext, tab: '富文本' },
];

function resolveEditorMode(format, value) {
  const normalized = normalizeLegalContentFormat(format);
  if (normalized !== LEGAL_CONTENT_FORMATS.auto) {
    return normalized;
  }
  return looksLikeHtml(value)
    ? LEGAL_CONTENT_FORMATS.html
    : LEGAL_CONTENT_FORMATS.markdown;
}

export default function LegalContentEditor({
  title,
  value,
  format,
  styleId,
  placeholder,
  helpText,
  saveText,
  loading,
  onChange,
  onFormatChange,
  onSave,
}) {
  const { t } = useTranslation();
  const content = String(value || '');
  const editorMode = resolveEditorMode(format, content);
  const renderEditor = () => {
    if (editorMode === LEGAL_CONTENT_FORMATS.richtext) {
      return (
        <div className='legal-content-editor-richtext'>
          <SimpleRichTextEditor
            value={content}
            onChange={onChange}
            placeholder={placeholder}
            minHeight={360}
          />
        </div>
      );
    }

    return (
      <TextArea
        className='legal-content-editor-textarea'
        value={content}
        placeholder={placeholder}
        onChange={onChange}
        style={{
          width: '100%',
          height: '100%',
          fontFamily: 'JetBrains Mono, Consolas',
        }}
      />
    );
  };

  return (
    <Form.Slot>
      <div className='legal-content-editor'>
        <div className='legal-content-editor-header'>
          <div className='legal-content-editor-title-actions'>
            <Text strong>{title}</Text>
            <Button onClick={onSave} loading={loading}>
              {saveText}
            </Button>
          </div>
          <div className='legal-content-editor-mode-tabs'>
            <Tabs
              type='button'
              activeKey={editorMode}
              onChange={(mode) => onFormatChange?.(mode)}
            >
              {EDITOR_MODE_OPTIONS.map((option) => (
                <Tabs.TabPane
                  key={option.itemKey}
                  itemKey={option.itemKey}
                  tab={t(option.tab)}
                />
              ))}
            </Tabs>
          </div>
        </div>
        {helpText ? (
          <Text
            type='tertiary'
            size='small'
            className='legal-content-editor-help'
          >
            {helpText}
          </Text>
        ) : null}
        <div className='legal-content-editor-grid'>
          <div className='legal-content-editor-panel legal-content-editor-input-panel'>
            {renderEditor()}
          </div>
          <div className='legal-content-editor-preview-panel legal-content-editor-panel'>
            <Text strong>{t('预览')}</Text>
            <div className='legal-content-editor-preview-scroll'>
              {content.trim() ? (
                <LegalContentRenderer
                  content={content}
                  format={editorMode}
                  styleId={styleId}
                  title={title}
                />
              ) : (
                <Text type='tertiary'>{t('暂无内容')}</Text>
              )}
            </div>
          </div>
        </div>
      </div>
    </Form.Slot>
  );
}
