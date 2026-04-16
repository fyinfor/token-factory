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

import { useCallback, useEffect, useState } from 'react';
import { API } from '../../helpers';

const POLL_INTERVAL_MS = 2 * 60 * 1000;

// useUserMessageUnreadCount 轮询当前用户未读站内消息数量。
export const useUserMessageUnreadCount = (user) => {
  const [unreadCount, setUnreadCount] = useState(0);

  // refreshUnreadCount 主动刷新当前用户未读站内消息数量。
  const refreshUnreadCount = useCallback(async () => {
    if (!user?.id) {
      setUnreadCount(0);
      return 0;
    }
    try {
      const res = await API.get('/api/user/messages/unread_count', {
        skipErrorHandler: true,
      });
      if (res?.data?.success) {
        const count = Number(res.data.data?.unread_count || 0);
        setUnreadCount(count);
        return count;
      }
    } catch (_) {
      // ignore
    }
    setUnreadCount(0);
    return 0;
  }, [user?.id]);

  useEffect(() => {
    // 未登录用户不轮询，避免无效请求。
    if (!user?.id) {
      setUnreadCount(0);
      return undefined;
    }
    const timer = setInterval(refreshUnreadCount, POLL_INTERVAL_MS);
    refreshUnreadCount();
    return () => {
      clearInterval(timer);
    };
  }, [refreshUnreadCount, user?.id]);

  return { unreadCount, refreshUnreadCount };
};

