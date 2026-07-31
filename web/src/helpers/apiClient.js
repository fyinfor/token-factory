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

import axios from 'axios';
import { getUserIdFromLocalStorage, showError } from './appBasics';

function createAPIInstance() {
  return axios.create({
    baseURL: import.meta.env.VITE_REACT_APP_SERVER_URL || '',
    headers: {
      'New-API-User': getUserIdFromLocalStorage(),
      'Cache-Control': 'no-store',
    },
  });
}

function resolveAcceptLanguage() {
  try {
    const raw =
      (typeof window !== 'undefined' && window.__i18n?.language) ||
      localStorage.getItem('i18nextLng') ||
      'zh-CN';
    const lang = String(raw).trim().toLowerCase();
    if (lang.startsWith('zh-tw')) return 'zh-TW';
    if (lang.startsWith('zh')) return 'zh-CN';
    if (lang.startsWith('en')) return 'en';
    return 'zh-CN';
  } catch {
    return 'zh-CN';
  }
}

function attachAcceptLanguageInterceptor(instance) {
  instance.interceptors.request.use((config) => {
    config.headers = config.headers || {};
    config.headers['Accept-Language'] = resolveAcceptLanguage();
    return config;
  });
}

function patchAPIInstance(instance) {
  const originalGet = instance.get.bind(instance);
  const inFlightGetRequests = new Map();

  instance.get = (url, config = {}) => {
    if (config?.disableDuplicate) {
      return originalGet(url, config);
    }

    const params = config.params ? JSON.stringify(config.params) : '{}';
    const key = `${url}?${params}`;
    if (inFlightGetRequests.has(key)) {
      return inFlightGetRequests.get(key);
    }

    const request = originalGet(url, config).finally(() => {
      inFlightGetRequests.delete(key);
    });
    inFlightGetRequests.set(key, request);
    return request;
  };
}

const tGlobal = (key) =>
  (typeof window !== 'undefined' ? window.__i18n?.t?.(key) : undefined) || key;

function attachResponseInterceptor(instance) {
  instance.interceptors.response.use(
    (response) => response,
    (error) => {
      if (error.config?.skipErrorHandler) {
        return Promise.reject(error);
      }
      const responseData = error.response?.data;
      if (responseData?.data?.require_real_name_verification) {
        showError(responseData.message || tGlobal('充值前请先完成实名认证'));
        window.location.assign(
          responseData.data.redirect || '/console/real-name-verification',
        );
        return Promise.reject(error);
      }
      showError(error);
      return Promise.reject(error);
    },
  );
}

function configureAPI(instance) {
  patchAPIInstance(instance);
  attachAcceptLanguageInterceptor(instance);
  attachResponseInterceptor(instance);
  return instance;
}

export let API = configureAPI(createAPIInstance());

export function updateAPI() {
  API = configureAPI(createAPIInstance());
}
