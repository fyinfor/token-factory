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

import localforage from 'localforage';

const STORE_KEY_PREFIX = 'playground_mode_messages';

const playgroundMessageStore = localforage.createInstance({
  name: 'playgroundDB',
  storeName: 'messages',
  description: 'Playground messages storage',
});

export const getMessageStorageKey = (userId) =>
  `${STORE_KEY_PREFIX}_${userId || 'guest'}`;

export const saveModeMessages = async (userId, modeMessages) => {
  try {
    await playgroundMessageStore.setItem(getMessageStorageKey(userId), {
      ...modeMessages,
      timestamp: new Date().toISOString(),
      userId: userId || 'guest',
    });
  } catch (error) {
    console.error('保存操练场消息失败:', error);
  }
};

export const loadModeMessages = async (userId) => {
  try {
    const saved = await playgroundMessageStore.getItem(
      getMessageStorageKey(userId),
    );
    if (!saved || typeof saved !== 'object') return null;
    return saved;
  } catch (error) {
    console.error('加载操练场消息失败:', error);
    return null;
  }
};

export const clearModeMessages = async (userId) => {
  try {
    await playgroundMessageStore.removeItem(getMessageStorageKey(userId));
  } catch (error) {
    console.error('清除操练场消息失败:', error);
  }
};
