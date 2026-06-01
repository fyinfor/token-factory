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

const KNOWN_APP_LABELS = {
  cherrystudio: 'Cherry Studio',
  aionui: 'AionUI',
  ama: 'AMA 问天',
  opencat: 'OpenCat',
};

/** @returns {boolean} */
export function isCustomProtocolUrl(url) {
  if (!url || typeof url !== 'string') return false;
  const trimmed = url.trim();
  if (trimmed.startsWith('http://') || trimmed.startsWith('https://')) {
    return false;
  }
  return /^[a-z][a-z0-9+.-]*:/i.test(trimmed);
}

/** @returns {string|null} */
export function getAppLabelFromUrl(url) {
  const match = String(url || '').match(/^([a-z][a-z0-9+.-]*):/i);
  if (!match) return null;
  const scheme = match[1].toLowerCase();
  return KNOWN_APP_LABELS[scheme] || scheme;
}

/**
 * Open a link. For custom protocol URLs, use blur/visibility heuristics to guess
 * whether a registered handler opened (browsers cannot query installation directly).
 * Callbacks fire immediately / asynchronously — callers should not block UI on the promise.
 *
 * @returns {Promise<'opened'|'maybe-not-installed'|'navigated'>}
 */
export function openAppLink(
  url,
  {
    onOpening,
    onOpened,
    onMaybeNotInstalled,
    detectTimeoutMs = 1200,
  } = {},
) {
  if (!url) return Promise.resolve('navigated');

  if (!isCustomProtocolUrl(url)) {
    window.open(url, '_blank');
    return Promise.resolve('navigated');
  }

  const appLabel = getAppLabelFromUrl(url);
  onOpening?.(appLabel);

  return new Promise((resolve) => {
    let settled = false;
    const finish = (result) => {
      if (settled) return;
      settled = true;
      window.removeEventListener('blur', onBlur);
      document.removeEventListener('visibilitychange', onVisibility);
      if (result === 'opened') {
        onOpened?.(appLabel);
      }
      resolve(result);
    };

    const onBlur = () => finish('opened');
    const onVisibility = () => {
      if (document.hidden) finish('opened');
    };

    window.addEventListener('blur', onBlur);
    document.addEventListener('visibilitychange', onVisibility);

    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.style.display = 'none';
    document.body.appendChild(anchor);
    anchor.click();
    document.body.removeChild(anchor);

    window.setTimeout(() => {
      if (settled) return;
      finish('maybe-not-installed');
      onMaybeNotInstalled?.(appLabel);
    }, detectTimeoutMs);
  });
}
