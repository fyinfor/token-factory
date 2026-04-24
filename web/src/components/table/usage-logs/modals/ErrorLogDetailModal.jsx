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

import React, { useMemo } from 'react';
import { Modal, Button, Typography, Space, Divider } from '@douyinfe/semi-ui';
import { IconCopy } from '@douyinfe/semi-icons';
import { copy, showSuccess } from '../../../../helpers';

const { Text } = Typography;

/**
 * 使用日志「错误」类记录的完整内容弹窗，便于阅读被列表截断的长错误文本。
 * @param {object} props
 * @param {boolean} props.visible 是否显示
 * @param {object|null} props.logRecord 当前行日志（需含 content 等字段）
 * @param {() => void} props.onClose 关闭回调
 * @param {function} props.t i18n
 * @returns {JSX.Element}
 */
const ErrorLogDetailModal = ({ visible, logRecord, onClose, t }) => {
  const fullText = useMemo(() => {
    if (logRecord == null) {
      return '';
    }
    const c = logRecord.content;
    return c == null ? '' : String(c);
  }, [logRecord]);

  /**
   * 将完整错误内容复制到剪贴板。
   * @param {import('react').MouseEvent} e
   */
  const handleCopyAll = (e) => {
    e.stopPropagation();
    if (!fullText) {
      return;
    }
    copy(fullText).then((ok) => {
      if (ok) {
        showSuccess(t('已复制到剪贴板'));
      }
    });
  };

  return (
    <Modal
      title={t('错误详情')}
      visible={visible}
      onCancel={onClose}
      maskClosable
      width={720}
      footer={
        <Space>
          <Button
            type='tertiary'
            icon={<IconCopy />}
            onClick={handleCopyAll}
            disabled={!fullText}
          >
            {t('复制全文')}
          </Button>
          <Button type='primary' onClick={onClose}>
            {t('关闭')}
          </Button>
        </Space>
      }
    >
      {logRecord ? (
        <div>
          <Space wrap spacing={8}>
            {logRecord.timestamp2string ? (
              <Text type='tertiary' size='small'>
                {t('时间')}
                {': '}
                {logRecord.timestamp2string}
              </Text>
            ) : null}
            {logRecord.model_name ? (
              <Text type='tertiary' size='small'>
                {t('模型')}
                {': '}
                {logRecord.model_name}
              </Text>
            ) : null}
            {logRecord.request_id ? (
              <Text type='tertiary' size='small'>
                {t('Request ID')}
                {': '}
                {logRecord.request_id}
              </Text>
            ) : null}
          </Space>
          <Divider margin='12px' />
          <div
            style={{
              maxHeight: '60vh',
              overflow: 'auto',
              whiteSpace: 'pre-wrap',
              wordBreak: 'break-word',
              lineHeight: 1.6,
              fontSize: 13,
              fontFamily: 'var(--semi-font-family-code)',
            }}
          >
            {fullText || t('无')}
          </div>
        </div>
      ) : null}
    </Modal>
  );
};

export default ErrorLogDetailModal;
