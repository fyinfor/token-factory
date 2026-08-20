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
import { useTranslation } from 'react-i18next';
import DOMPurify from 'dompurify';
import { marked } from 'marked';
import {
  CheckCircle2,
  CircleAlert,
  Info,
  Megaphone,
  TriangleAlert,
} from 'lucide-react';
import { getRelativeTime } from '../../helpers/utils';
import { getAnnouncementKey } from '../../hooks/common/useNotifications';
import { getLocalizedAnnouncement } from '../../helpers/announcement';

const TYPE_META = {
  success: { icon: CheckCircle2, className: 'is-success' },
  warning: { icon: TriangleAlert, className: 'is-warning' },
  error: { icon: CircleAlert, className: 'is-error' },
  ongoing: { icon: Megaphone, className: 'is-ongoing' },
  default: { icon: Info, className: 'is-default' },
};

const renderMarkdown = (content) =>
  DOMPurify.sanitize(marked.parse(content || ''), {
    USE_PROFILES: { html: true },
  });

const formatTime = (publishDate) => {
  if (!publishDate) {
    return '';
  }
  const date = new Date(publishDate);
  if (Number.isNaN(date.getTime())) {
    return publishDate;
  }
  const absolute = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`;
  const relative = getRelativeTime(publishDate);
  return relative ? `${relative} · ${absolute}` : absolute;
};

const AnnouncementList = ({ items, unreadKeys = [], fallbackContent = '' }) => {
  const { i18n } = useTranslation();
  const unreadSet = useMemo(() => new Set(unreadKeys), [unreadKeys]);
  const fallbackHtml = useMemo(
    () => renderMarkdown(fallbackContent),
    [fallbackContent],
  );

  if (items.length === 0 && fallbackContent) {
    return (
      <article className='notification-announcement-item is-default'>
        <div className='notification-announcement-rail'>
          <span className='notification-announcement-icon'>
            <Info size={17} />
          </span>
        </div>
        <div className='notification-announcement-copy'>
          <div
            className='notification-rich-text'
            dangerouslySetInnerHTML={{ __html: fallbackHtml }}
          />
        </div>
      </article>
    );
  }

  return items.map((item) => {
    const localizedItem = getLocalizedAnnouncement(item, i18n.language);
    const key = getAnnouncementKey(localizedItem);
    const meta = TYPE_META[localizedItem?.type] || TYPE_META.default;
    const Icon = meta.icon;
    const contentHtml = renderMarkdown(localizedItem?.content);
    const extraHtml = localizedItem?.extra
      ? renderMarkdown(localizedItem.extra)
      : '';
    return (
      <article
        className={`notification-announcement-item ${meta.className}`}
        key={key}
      >
        <div className='notification-announcement-rail'>
          <span className='notification-announcement-icon'>
            <Icon size={17} />
          </span>
          <span className='notification-announcement-line' />
        </div>
        <div className='notification-announcement-copy'>
          <div className='notification-announcement-meta'>
            <time>{formatTime(localizedItem?.publishDate)}</time>
            {unreadSet.has(key) ? (
              <span className='notification-unread-dot' aria-hidden='true' />
            ) : null}
          </div>
          <div
            className='notification-rich-text'
            dangerouslySetInnerHTML={{ __html: contentHtml }}
          />
          {extraHtml ? (
            <div
              className='notification-announcement-extra'
              dangerouslySetInnerHTML={{ __html: extraHtml }}
            />
          ) : null}
        </div>
      </article>
    );
  });
};

export default AnnouncementList;
