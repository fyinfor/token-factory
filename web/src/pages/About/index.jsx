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

import React, { useEffect, useMemo, useState } from 'react';
import { API, showError, getLocalizedContent } from '../../helpers';
import { marked } from 'marked';
import { Empty } from '@douyinfe/semi-ui';
import {
  IllustrationConstruction,
  IllustrationConstructionDark,
} from '@douyinfe/semi-illustrations';
import { useTranslation } from 'react-i18next';

const ABOUT_CACHE_KEY = 'about_content_v2';

const parseAboutContent = (raw) => {
  const text = String(raw ?? '').trim();
  if (!text) return '';
  if (text.startsWith('https://')) {
    return text;
  }
  return marked.parse(text);
};

const normalizeAboutPayload = (data) => {
  if (data && typeof data === 'object' && !Array.isArray(data)) {
    return {
      about: String(data.about ?? ''),
      about_en: String(data.about_en ?? ''),
    };
  }
  return {
    about: String(data ?? ''),
    about_en: '',
  };
};

const About = () => {
  const { t, i18n } = useTranslation();
  const [aboutSource, setAboutSource] = useState({ about: '', about_en: '' });
  const [aboutLoaded, setAboutLoaded] = useState(false);

  useEffect(() => {
    let cancelled = false;

    const loadAbout = async () => {
      try {
        const cached = localStorage.getItem(ABOUT_CACHE_KEY);
        if (cached) {
          const parsed = normalizeAboutPayload(JSON.parse(cached));
          if (!cancelled) {
            setAboutSource(parsed);
          }
        }
      } catch {
        /* ignore invalid cache */
      }

      try {
        const res = await API.get('/api/about');
        const { success, message, data } = res.data;
        if (success) {
          const payload = normalizeAboutPayload(data);
          if (!cancelled) {
            setAboutSource(payload);
            localStorage.setItem(ABOUT_CACHE_KEY, JSON.stringify(payload));
          }
        } else if (!cancelled) {
          showError(message);
        }
      } catch {
        if (!cancelled) {
          showError(t('加载关于内容失败...'));
        }
      } finally {
        if (!cancelled) {
          setAboutLoaded(true);
        }
      }
    };

    loadAbout();
    return () => {
      cancelled = true;
    };
  }, [t]);

  const rawAbout = useMemo(
    () =>
      getLocalizedContent(
        aboutSource.about,
        aboutSource.about_en,
        i18n.language,
      ),
    [aboutSource.about, aboutSource.about_en, i18n.language],
  );

  const about = useMemo(() => parseAboutContent(rawAbout), [rawAbout]);

  const emptyStyle = {
    padding: '24px',
  };

  const customDescription = (
    <div style={{ textAlign: 'center' }}>
      <p>{t('可在设置页面设置关于内容，支持 HTML & Markdown')}</p>
    </div>
  );

  return (
    <div className='mt-[60px] px-2'>
      {aboutLoaded && about === '' ? (
        <div className='flex justify-center items-center h-screen p-8'>
          <Empty
            image={
              <IllustrationConstruction style={{ width: 150, height: 150 }} />
            }
            darkModeImage={
              <IllustrationConstructionDark
                style={{ width: 150, height: 150 }}
              />
            }
            description={t('管理员暂时未设置任何关于内容')}
            style={emptyStyle}
          >
            {customDescription}
          </Empty>
        </div>
      ) : (
        <>
          {about.startsWith('https://') ? (
            <iframe
              src={about}
              style={{ width: '100%', height: '100vh', border: 'none' }}
              title={t('关于')}
            />
          ) : (
            <div
              style={{ fontSize: 'larger' }}
              dangerouslySetInnerHTML={{ __html: about }}
            ></div>
          )}
        </>
      )}
    </div>
  );
};

export default About;
