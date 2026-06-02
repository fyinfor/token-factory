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

import React, { useEffect, useMemo } from 'react';
import { Typography } from '@douyinfe/semi-ui';

const { Title } = Typography;

const parseLegalHtml = (html) => {
  const tempDiv = document.createElement('div');
  tempDiv.innerHTML = html;

  const styles = Array.from(tempDiv.querySelectorAll('style'))
    .map((style) => style.innerHTML)
    .join('\n');

  const bodyContent = tempDiv.querySelector('body');
  const content = bodyContent ? bodyContent.innerHTML : html;

  return { content, styles };
};

const LegalDocumentPage = ({ title, html, styleId }) => {
  const { content, styles } = useMemo(() => parseLegalHtml(html), [html]);

  useEffect(() => {
    if (!styles) return undefined;

    let styleEl = document.getElementById(styleId);
    if (!styleEl) {
      styleEl = document.createElement('style');
      styleEl.id = styleId;
      styleEl.type = 'text/css';
      document.head.appendChild(styleEl);
    }
    styleEl.innerHTML = styles;

    return () => {
      const el = document.getElementById(styleId);
      if (el) el.remove();
    };
  }, [styles, styleId]);

  return (
    <div className='min-h-screen bg-gray-50'>
      <div className='max-w-4xl mx-auto py-12 px-4 sm:px-6 lg:px-8'>
        <div className='bg-white rounded-lg shadow-sm p-8'>
          <Title heading={2} className='text-center mb-8'>
            {title}
          </Title>
          <div
            className='prose prose-lg max-w-none'
            dangerouslySetInnerHTML={{ __html: content }}
          />
        </div>
      </div>
    </div>
  );
};

export default LegalDocumentPage;
