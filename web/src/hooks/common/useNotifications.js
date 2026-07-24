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

const ANNOUNCEMENT_READ_STORAGE_KEY = 'notice_read_keys';
export const ANNOUNCEMENT_READ_EVENT = 'announcement-read-state-changed';

export const getAnnouncementKey = (announcement) =>
  `${announcement?.publishDate || ''}-${(announcement?.content || '').slice(0, 30)}`;

const readStoredKeys = () => {
  try {
    const keys = JSON.parse(
      localStorage.getItem(ANNOUNCEMENT_READ_STORAGE_KEY) || '[]',
    );
    return Array.isArray(keys) ? keys : [];
  } catch (_) {
    return [];
  }
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
  const [readVersion, setReadVersion] = useState(0);
  const announcements = useMemo(
    () => statusState?.status?.announcements || [],
    [statusState?.status?.announcements],
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
