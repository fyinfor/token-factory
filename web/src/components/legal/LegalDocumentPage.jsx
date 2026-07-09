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
import { Empty, Spin, Typography } from '@douyinfe/semi-ui';
import {
  IllustrationConstruction,
  IllustrationConstructionDark,
} from '@douyinfe/semi-illustrations';
import { useTranslation } from 'react-i18next';
import { API, showError } from '../../helpers';
import LegalContentRenderer, {
  LEGAL_CONTENT_FORMATS,
} from './LegalContentRenderer';

const { Title } = Typography;

const LegalDocumentPage = ({
  title,
  apiEndpoint,
  styleId,
  defaultContent = '',
  defaultFormat = LEGAL_CONTENT_FORMATS.html,
}) => {
  const { t } = useTranslation();
  const [content, setContent] = useState(defaultContent);
  const [format, setFormat] = useState(defaultFormat);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    const loadContent = async () => {
      setLoading(true);
      try {
        const res = await API.get(apiEndpoint, { skipErrorHandler: true });
        if (cancelled) return;

        const {
          success,
          message,
          data,
          format: contentFormat,
        } = res.data || {};
        if (!success) {
          if (!defaultContent) {
            showError(message || t('加载内容失败'));
          }
          setContent(defaultContent);
          setFormat(defaultFormat);
          return;
        }

        setContent(data || defaultContent);
        setFormat(contentFormat || defaultFormat);
      } catch (error) {
        if (!cancelled) {
          if (!defaultContent) {
            showError(
              error?.response?.data?.message ||
                error?.message ||
                t('加载内容失败'),
            );
          }
          setContent(defaultContent);
          setFormat(defaultFormat);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    if (apiEndpoint) {
      loadContent();
    } else {
      setLoading(false);
    }

    return () => {
      cancelled = true;
    };
  }, [apiEndpoint, defaultContent, defaultFormat, t]);

  if (loading) {
    return (
      <div className='flex justify-center items-center min-h-screen bg-gray-50'>
        <Spin size='large' />
      </div>
    );
  }

  if (!String(content || '').trim()) {
    return (
      <div className='flex justify-center items-center min-h-screen bg-gray-50'>
        <Empty
          title={t('暂无内容')}
          description={title}
          image={
            <IllustrationConstruction style={{ width: 150, height: 150 }} />
          }
          darkModeImage={
            <IllustrationConstructionDark style={{ width: 150, height: 150 }} />
          }
          className='p-8'
        />
      </div>
    );
  }

  return (
    <div className='min-h-screen bg-gray-50'>
      <div className='max-w-4xl mx-auto py-12 px-4 sm:px-6 lg:px-8'>
        <div className='bg-white rounded-lg shadow-sm p-8'>
          <Title heading={2} className='text-center mb-8'>
            {title}
          </Title>
          <LegalContentRenderer
            content={content}
            format={format}
            styleId={styleId}
            title={title}
          />
        </div>
      </div>
    </div>
  );
};

export default LegalDocumentPage;
