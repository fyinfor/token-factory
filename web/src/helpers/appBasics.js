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

import { Toast } from '@douyinfe/semi-ui';
import { USER_ROLES } from '../constants/user.constants';
import { normalizeLanguage } from '../i18n/language';

export function getSystemName(language) {
  const lang = normalizeLanguage(
    language || localStorage.getItem('i18nextLng') || 'zh-CN',
  );
  if (lang === 'zh-CN' || lang === 'zh-TW') {
    return localStorage.getItem('system_name') || '词元工厂';
  }
  return (
    localStorage.getItem('system_name_en') ||
    localStorage.getItem('system_name') ||
    'TokenFactory'
  );
}

export const getLogo = () => localStorage.getItem('logo') || '/logo.png';
export const getFooterHTML = () => localStorage.getItem('footer_html');

export function getUserIdFromLocalStorage() {
  try {
    const user = JSON.parse(localStorage.getItem('user') || 'null');
    return user?.id ?? -1;
  } catch {
    return -1;
  }
}

export const userIsSupplierUser = (user) =>
  Boolean(
    user &&
    user.supplier_id != null &&
    user.supplier_id !== 0 &&
    user.supplier_id !== '0',
  );

export function isAdmin() {
  try {
    return JSON.parse(localStorage.getItem('user') || 'null')?.role >= 10;
  } catch {
    return false;
  }
}

export const showSuccess = (message) => Toast.success(message);
export const showInfo = (message) => Toast.info(message);

export function showError(error) {
  console.error(error);
  if (error?.message) {
    if (error.name === 'AxiosError') {
      switch (error.response?.status) {
        case 401:
          localStorage.removeItem('user');
          window.location.href = '/login?expired=true';
          break;
        case 429:
          Toast.error('错误：请求次数过多，请稍后再试！');
          break;
        case 500:
          Toast.error('错误：服务器内部错误，请联系管理员！');
          break;
        case 405:
          Toast.info('本站仅作演示之用，无服务端！');
          break;
        default:
          Toast.error('错误：' + error.message);
      }
      return;
    }
    Toast.error('错误：' + error.message);
    return;
  }
  Toast.error('错误：' + error);
}

export const userIsDistributorUser = (user) =>
  Boolean(
    user &&
    (user.is_distributor === 1 ||
      user.is_distributor === true ||
      user.role === USER_ROLES.DISTRIBUTOR),
  );
