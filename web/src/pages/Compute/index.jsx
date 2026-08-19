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

import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Empty, Spin } from '@douyinfe/semi-ui';
import {
  IllustrationNoContent,
  IllustrationNoContentDark,
} from '@douyinfe/semi-illustrations';
import { useTranslation } from 'react-i18next';

const COMPUTE_PAGE_SANDBOX = 'allow-same-origin allow-forms';

function getComputePageSandbox(allowJavaScript, allowPopups) {
  const permissions = [COMPUTE_PAGE_SANDBOX];
  if (allowPopups) {
    permissions.push('allow-popups', 'allow-popups-to-escape-sandbox');
  }
  if (allowJavaScript) permissions.push('allow-scripts');
  return permissions.join(' ');
}

export default function ComputePage() {
  const { t } = useTranslation();
  const [state, setState] = useState('loading');
  const [allowJavaScript, setAllowJavaScript] = useState(false);
  const [allowPopups, setAllowPopups] = useState(false);
  const [contentURL, setContentURL] = useState('');
  const iframeRef = useRef(null);
  const resizeObserverRef = useRef(null);
  const copyCleanupRef = useRef(null);

  const syncIframeHeight = useCallback(() => {
    const iframe = iframeRef.current;
    const documentElement = iframe?.contentDocument?.documentElement;
    const body = iframe?.contentDocument?.body;
    if (!iframe || !documentElement || !body) return;

    iframe.style.height = `${Math.max(
      documentElement.scrollHeight,
      body.scrollHeight,
    )}px`;
  }, []);

  const handleIframeLoad = useCallback(() => {
    resizeObserverRef.current?.disconnect();
    copyCleanupRef.current?.();
    syncIframeHeight();

    const iframeDocument = iframeRef.current?.contentDocument;
    const documentElement = iframeDocument?.documentElement;
    if (!documentElement || typeof ResizeObserver === 'undefined') return;

    const observer = new ResizeObserver(syncIframeHeight);
    observer.observe(documentElement);
    resizeObserverRef.current = observer;

    const copyButton = iframeDocument.querySelector('[data-copy-phone]');
    if (!copyButton) return;

    let resetTimer;
    const handleCopy = async () => {
      const phone = copyButton.dataset.copyPhone;
      try {
        await navigator.clipboard.writeText(phone);
      } catch {
        const input = document.createElement('textarea');
        input.value = phone;
        input.style.position = 'fixed';
        input.style.opacity = '0';
        document.body.appendChild(input);
        input.select();
        document.execCommand('copy');
        input.remove();
      }

      window.clearTimeout(resetTimer);
      copyButton.classList.add('is-copied');
      copyButton.querySelector('.zh').textContent = '已复制';
      copyButton.querySelector('.en').textContent = 'Copied';
      resetTimer = window.setTimeout(() => {
        copyButton.classList.remove('is-copied');
        copyButton.querySelector('.zh').textContent = '复制';
        copyButton.querySelector('.en').textContent = 'Copy';
      }, 1600);
    };

    copyButton.addEventListener('click', handleCopy);
    copyCleanupRef.current = () => {
      window.clearTimeout(resetTimer);
      copyButton.removeEventListener('click', handleCopy);
    };
  }, [syncIframeHeight]);

  useEffect(() => {
    let cancelled = false;
    fetch('/api/compute-page/status', { cache: 'no-store' })
      .then((response) => {
        if (!response.ok) throw new Error('Failed to load compute page status');
        return response.json();
      })
      .then((payload) => {
        if (cancelled) return;
        setAllowJavaScript(Boolean(payload?.data?.allow_javascript));
        setAllowPopups(Boolean(payload?.data?.allow_popups));
        setContentURL(payload?.data?.content_url || '');
        setState(
          payload?.success && payload?.data?.enabled ? 'enabled' : 'disabled',
        );
      })
      .catch(() => {
        if (!cancelled) setState('disabled');
      });
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    window.addEventListener('resize', syncIframeHeight);
    return () => {
      window.removeEventListener('resize', syncIframeHeight);
      resizeObserverRef.current?.disconnect();
      copyCleanupRef.current?.();
    };
  }, [syncIframeHeight]);

  if (state === 'loading') {
    return (
      <div className='compute-page-state'>
        <Spin size='large' />
      </div>
    );
  }

  if (state !== 'enabled') {
    return (
      <div className='compute-page-state'>
        <Empty
          image={<IllustrationNoContent style={{ width: 220, height: 220 }} />}
          darkModeImage={
            <IllustrationNoContentDark style={{ width: 220, height: 220 }} />
          }
          title={t('算力页面暂未开放')}
        />
      </div>
    );
  }

  return (
    <iframe
      ref={iframeRef}
      className='compute-page-frame'
      src={contentURL || '/api/compute-page/content'}
      title={t('算力')}
      sandbox={getComputePageSandbox(allowJavaScript, allowPopups)}
      onLoad={handleIframeLoad}
      referrerPolicy='no-referrer'
    />
  );
}
