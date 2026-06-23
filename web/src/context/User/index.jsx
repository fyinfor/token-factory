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

import React, { useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { reducer, initialState } from './reducer';
import { normalizeLanguage } from '../../i18n/language';
import { API, mergeSelfResponseIntoLocalUser } from '../../helpers';
import AdminInitialSetupModal from '../../components/auth/AdminInitialSetupModal';

const USER_SELF_REFRESH_INTERVAL = 60 * 1000;
const USER_SELF_REFRESH_DEBOUNCE = 2 * 1000;

export const UserContext = React.createContext({
  state: initialState,
  dispatch: () => null,
});

export const UserProvider = ({ children }) => {
  const [state, dispatch] = React.useReducer(reducer, initialState);
  const { i18n } = useTranslation();

  // 已登录时拉取服务端最新资料（含 role），避免仅依赖本地缓存：例如管理员将代理降级后首页仍显示「已是代理」、不显示申请入口
  useEffect(() => {
    if (!localStorage.getItem('user')) return;
    let cancelled = false;
    (async () => {
      try {
        const res = await API.get('/api/user/self');
        if (cancelled || !res.data.success || !res.data.data) return;
        mergeSelfResponseIntoLocalUser(res.data.data, dispatch);
      } catch (e) {
        // 未登录 / token 失效时忽略
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [dispatch]);

  useEffect(() => {
    if (!state.user?.id) return;

    let cancelled = false;
    let refreshing = false;
    let lastRefreshAt = 0;

    const refreshSelf = async () => {
      if (cancelled || refreshing) return;
      if (!localStorage.getItem('user')) return;
      if (document.visibilityState === 'hidden') return;

      const now = Date.now();
      if (now - lastRefreshAt < USER_SELF_REFRESH_DEBOUNCE) return;

      refreshing = true;
      lastRefreshAt = now;
      try {
        const res = await API.get('/api/user/self', { skipErrorHandler: true });
        if (cancelled || !res.data.success || !res.data.data) return;
        mergeSelfResponseIntoLocalUser(res.data.data, dispatch);
      } catch (e) {
        // Ignore transient refresh failures.
      } finally {
        refreshing = false;
      }
    };

    const handleVisibilityChange = () => {
      if (document.visibilityState === 'visible') {
        refreshSelf();
      }
    };

    const refreshTimer = window.setInterval(
      refreshSelf,
      USER_SELF_REFRESH_INTERVAL,
    );

    document.addEventListener('visibilitychange', handleVisibilityChange);
    window.addEventListener('focus', refreshSelf);

    return () => {
      cancelled = true;
      window.clearInterval(refreshTimer);
      document.removeEventListener('visibilitychange', handleVisibilityChange);
      window.removeEventListener('focus', refreshSelf);
    };
  }, [state.user?.id, dispatch]);

  // Sync language preference when user data is loaded
  useEffect(() => {
    if (state.user?.setting) {
      try {
        const settings = JSON.parse(state.user.setting);
        const normalizedLanguage = normalizeLanguage(settings.language);
        if (normalizedLanguage && normalizedLanguage !== i18n.language) {
          i18n.changeLanguage(normalizedLanguage);
        }
        if (normalizedLanguage) {
          localStorage.setItem('i18nextLng', normalizedLanguage);
        }
      } catch (e) {
        // Ignore parse errors
      }
    }
  }, [state.user?.setting, i18n]);

  return (
    <UserContext.Provider value={[state, dispatch]}>
      {children}
      <AdminInitialSetupModal />
    </UserContext.Provider>
  );
};
