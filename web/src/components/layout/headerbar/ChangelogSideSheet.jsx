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

import React, { useEffect, useState } from 'react';
import { Empty, SideSheet, Spin, Tag, Typography } from '@douyinfe/semi-ui';
import { API, showError } from '../../../helpers';
import MarkdownRenderer from '../../common/markdown/MarkdownRenderer';

const { Text } = Typography;

const ChangelogSideSheet = ({ visible, onClose, isMobile, t }) => {
  const [entries, setEntries] = useState([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!visible) {
      return;
    }
    let cancelled = false;
    const loadChangelogs = async () => {
      setLoading(true);
      try {
        const res = await API.get('/api/changelog');
        const { success, message, data } = res.data || {};
        if (!success) {
          showError(message || t('加载更新日志失败'));
          return;
        }
        if (!cancelled) {
          setEntries(Array.isArray(data) ? data : []);
        }
      } catch (error) {
        if (!cancelled) {
          showError(error?.message || t('加载更新日志失败'));
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };
    loadChangelogs();
    return () => {
      cancelled = true;
    };
  }, [visible, t]);

  return (
    <SideSheet
      placement='right'
      title={t('更新日志')}
      visible={visible}
      width={isMobile ? '100%' : 520}
      onCancel={onClose}
      bodyStyle={{
        padding: 0,
        background: 'var(--semi-color-bg-0)',
      }}
    >
      {loading ? (
        <div className='flex h-full items-center justify-center px-6 py-16'>
          <Spin size='large' />
        </div>
      ) : entries.length === 0 ? (
        <div className='flex h-full items-center justify-center px-6 py-16'>
          <Empty title={t('暂无更新日志')} />
        </div>
      ) : (
        <div className='flex flex-col'>
          {entries.map((entry, index) => (
            <article
              key={entry.id}
              className='px-5 py-4'
              style={{
                borderBottom:
                  index === entries.length - 1
                    ? 'none'
                    : '1px solid var(--semi-color-border)',
              }}
            >
              <div className='mb-3 flex items-center gap-2'>
                <Tag color='blue' shape='circle' size='large'>
                  {entry.date}
                </Tag>
              </div>
              <MarkdownRenderer content={entry.content} />
            </article>
          ))}
        </div>
      )}
    </SideSheet>
  );
};

export default ChangelogSideSheet;
