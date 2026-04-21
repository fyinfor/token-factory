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

import React from 'react';
import { Navigate, useSearchParams } from 'react-router-dom';
import { history } from './history';
import { userIsDistributorUser } from './utils';

/** 登录/注册页 ?redirect= 仅允许站内相对路径，防止开放重定向 */
export function safeInternalRedirectPath(raw) {
  if (raw == null || typeof raw !== 'string') return null;
  let path = raw.trim();
  if (!path) return null;
  try {
    path = decodeURIComponent(path);
  } catch {
    return null;
  }
  if (!path.startsWith('/') || path.startsWith('//')) return null;
  if (/[\s\r\n]/.test(path)) return null;
  const lower = path.toLowerCase();
  if (lower.startsWith('javascript:') || path.includes('://')) return null;
  return path;
}

/**
 * 顶栏「成为代理 / 提供算力」未登录时会带 ?redirect= 到申请页；登录后按需改写目标：
 * - 分销申请：管理员 → 分销商管理；已是分销商 → 分销中心
 * - 供应商申请：管理员 → 供应商审批
 */
export function redirectApplyIntentToAdminIfNeeded(path, userLike) {
  if (path == null || path === '') return path;
  if (typeof path !== 'string') return path;
  if (!userLike || typeof userLike !== 'object') return path;

  let pathname = path.split('?')[0].trim();
  if (pathname.length > 1 && pathname.endsWith('/')) {
    pathname = pathname.slice(0, -1);
  }

  if (pathname === '/console/distributor/apply') {
    if (typeof userLike.role === 'number' && userLike.role >= 10) {
      return '/console/distributor/admin';
    }
    if (userIsDistributorUser(userLike)) {
      return '/console/distributor/center';
    }
    return path;
  }

  if (pathname === '/console/supplier/apply') {
    if (typeof userLike.role === 'number' && userLike.role >= 10) {
      return '/console/supplier-application';
    }
    return path;
  }

  return path;
}

export function authHeader() {
  // return authorization header with jwt token
  let user = JSON.parse(localStorage.getItem('user'));

  if (user && user.token) {
    return { Authorization: 'Bearer ' + user.token };
  } else {
    return {};
  }
}

export const AuthRedirect = ({ children }) => {
  const [searchParams] = useSearchParams();
  const user = localStorage.getItem('user');

  if (user) {
    const safe = safeInternalRedirectPath(searchParams.get('redirect'));
    try {
      const parsed = JSON.parse(user);
      const next = redirectApplyIntentToAdminIfNeeded(safe, parsed);
      return <Navigate to={next || '/console'} replace />;
    } catch {
      return <Navigate to={safe || '/console'} replace />;
    }
  }

  return children;
};

function PrivateRoute({ children }) {
  if (!localStorage.getItem('user')) {
    return <Navigate to='/login' state={{ from: history.location }} />;
  }
  return children;
}

export function AdminRoute({ children }) {
  const raw = localStorage.getItem('user');
  if (!raw) {
    return <Navigate to='/login' state={{ from: history.location }} />;
  }
  try {
    const user = JSON.parse(raw);
    if (user && typeof user.role === 'number' && user.role >= 10) {
      return children;
    }
  } catch (e) {
    // ignore
  }
  return <Navigate to='/forbidden' replace />;
}

export { PrivateRoute };
