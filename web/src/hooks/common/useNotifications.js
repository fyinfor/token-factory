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

import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { getLocalizedAnnouncement } from '../../helpers/announcement';

const ANNOUNCEMENT_READ_STORAGE_KEY = 'notice_read_keys';
const ANNOUNCEMENT_POPUP_ACK_STORAGE_KEY = 'notice_popup_acknowledged_keys';
export const ANNOUNCEMENT_READ_EVENT = 'announcement-read-state-changed';
export const OPEN_NOTIFICATION_CENTER_EVENT = 'open-notification-center';

const readStoredList = (storageKey) => {
  try {
    const values = JSON.parse(localStorage.getItem(storageKey) || '[]');
    return Array.isArray(values) ? values : [];
  } catch (_) {
    return [];
  }
};

const hashString = (value) => {
  let hash = 2166136261;
  for (let index = 0; index < value.length; index += 1) {
    hash ^= value.charCodeAt(index);
    hash = Math.imul(hash, 16777619);
  }
  return `${value.length}-${(hash >>> 0).toString(36)}`;
};

export const getAnnouncementKey = (announcement) =>
  `${announcement?.publishDate || ''}-${(announcement?.sourceContent || announcement?.content || '').slice(0, 30)}`;

const readStoredKeys = () => {
  return readStoredList(ANNOUNCEMENT_READ_STORAGE_KEY);
};

export const getAnnouncementPopupKey = (announcement) =>
  `announcement-${hashString(
    JSON.stringify({
      id: announcement?.id ?? null,
      publishDate: announcement?.publishDate || '',
      type: announcement?.type || 'default',
      content: announcement?.sourceContent || announcement?.content || '',
      extra: announcement?.sourceExtra || announcement?.extra || '',
    }),
  )}`;

export const getLegacyNoticePopupKey = (content) =>
  `legacy-${hashString(content || '')}`;

export const isAnnouncementPopupAcknowledged = (key) =>
  Boolean(
    key && readStoredList(ANNOUNCEMENT_POPUP_ACK_STORAGE_KEY).includes(key),
  );

export const acknowledgeAnnouncementPopup = (key) => {
  if (!key) {
    return;
  }
  const acknowledgedKeys = readStoredList(ANNOUNCEMENT_POPUP_ACK_STORAGE_KEY);
  const nextKeys = Array.from(new Set([...acknowledgedKeys, key])).slice(-200);
  localStorage.setItem(
    ANNOUNCEMENT_POPUP_ACK_STORAGE_KEY,
    JSON.stringify(nextKeys),
  );
};

export const markAnnouncementKeysRead = (keys) => {
  const nextKeys = keys.filter(Boolean);
  if (nextKeys.length === 0) {
    return;
  }
  const mergedKeys = Array.from(new Set([...readStoredKeys(), ...nextKeys]));
  localStorage.setItem(
    ANNOUNCEMENT_READ_STORAGE_KEY,
    JSON.stringify(mergedKeys),
  );
  window.dispatchEvent(new Event(ANNOUNCEMENT_READ_EVENT));
};

export const useNotifications = (statusState) => {
  const { i18n } = useTranslation();
  const [readVersion, setReadVersion] = useState(0);
  const announcements = useMemo(
    () =>
      (statusState?.status?.announcements || []).map((announcement) =>
        getLocalizedAnnouncement(announcement, i18n.language),
      ),
    [i18n.language, statusState?.status?.announcements],
  );

  const unreadKeys = useMemo(() => {
    const readSet = new Set(readStoredKeys());
    return announcements
      .map(getAnnouncementKey)
      .filter((key) => !readSet.has(key));
  }, [announcements, readVersion]);

  const refreshUnreadCount = useCallback(() => {
    setReadVersion((version) => version + 1);
  }, []);

  const markAnnouncementsRead = useCallback((items) => {
    markAnnouncementKeysRead(items.map(getAnnouncementKey));
  }, []);

  useEffect(() => {
    const handleReadStateChanged = () => refreshUnreadCount();
    const handleStorage = (event) => {
      if (event.key === ANNOUNCEMENT_READ_STORAGE_KEY) {
        refreshUnreadCount();
      }
    };
    window.addEventListener(ANNOUNCEMENT_READ_EVENT, handleReadStateChanged);
    window.addEventListener('storage', handleStorage);
    return () => {
      window.removeEventListener(
        ANNOUNCEMENT_READ_EVENT,
        handleReadStateChanged,
      );
      window.removeEventListener('storage', handleStorage);
    };
  }, [refreshUnreadCount]);

  return {
    announcements,
    unreadCount: unreadKeys.length,
    unreadKeys,
    markAnnouncementsRead,
    refreshUnreadCount,
  };
};
