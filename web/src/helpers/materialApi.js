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

/**
 * 素材管理接口请求封装层。
 * 统一复用项目现有 axios 实例（helpers/api.js 的 API），约定返回 { success, message, data }。
 * 业务组件 / 状态管理只依赖本文件，便于统一维护接口路径与错误约定。
 */
import { API } from './api';

// 后端素材模块 REST 路径（前端 -> 本系统后端，由后端再转发上游 ?Action= 接口）。
const MATERIAL_API = {
  CONFIG: '/api/material/config',
  GROUP: '/api/material/group',
  ASSETS: '/api/material/assets',
  UPLOAD: '/api/material/upload',
  UPLOAD_URL: '/api/material/upload-url',
  // 删除：DELETE /api/material/asset/:asset_id
  ASSET: (assetId) => `/api/material/asset/${encodeURIComponent(assetId)}`,
  // 真人认证会话（Web 控制台）。
  VISUAL_SESSION: '/api/material/visual/session',
  VISUAL_RESULT: '/api/material/visual/result',
  // 真人分组与素材管理（Web 控制台）。
  REAL_GROUPS: '/api/material/real/groups',
  REAL_GROUP: (groupId) => `/api/material/real/groups/${encodeURIComponent(groupId)}`,
  REAL_ASSETS: '/api/material/real/assets',
  REAL_UPLOAD: '/api/material/real/upload',
  REAL_UPLOAD_URL: '/api/material/real/upload-url',
  REAL_ASSET: (assetId) => `/api/material/real/asset/${encodeURIComponent(assetId)}`,
};

/**
 * 拉取素材库前端配置（启用状态、上传大小上限、合规协议文案等）。
 * @returns {Promise<{success:boolean,data:object}>}
 */
export const getMaterialConfig = async () => {
  const res = await API.get(MATERIAL_API.CONFIG);
  return res.data;
};

/**
 * 分页查询当前用户素材列表。
 * @param {{page?:number,pageSize?:number}} params
 */
export const listMaterialAssets = async ({ page = 1, pageSize = 100 } = {}) => {
  const res = await API.get(MATERIAL_API.ASSETS, {
    params: { p: page, size: pageSize },
  });
  return res.data;
};

/**
 * 本地文件上传素材。
 * 注意：上传成功后端会轮询上游 GetAsset 拉取永久素材 URL 后再落库返回。
 * @param {File} file 本地文件实例
 * @param {boolean} agreed 是否已勾选合规协议
 * @param {{onUploadProgress?: (ev: import('axios').AxiosProgressEvent) => void}} [options]
 */
export const uploadMaterialFile = async (file, agreed, options = {}) => {
  const fd = new FormData();
  fd.append('file', file);
  fd.append('agreed', agreed ? 'true' : 'false');
  const res = await API.post(MATERIAL_API.UPLOAD, fd, {
    skipErrorHandler: true,
    onUploadProgress: options.onUploadProgress,
  });
  return res.data;
};

/**
 * 在线资源链接上传素材。
 * @param {{url:string,name?:string,assetType?:string,agreed:boolean}} payload
 */
export const uploadMaterialByURL = async ({ url, name, assetType, agreed }) => {
  const res = await API.post(
    MATERIAL_API.UPLOAD_URL,
    {
      url,
      name: name || '',
      asset_type: assetType || '',
      agreed: !!agreed,
    },
    { skipErrorHandler: true },
  );
  return res.data;
};

/**
 * 查询单个素材详情（按上游 asset_id）。
 * 若素材仍待同步，后端会 best-effort 向上游刷新一次。
 * @param {string} assetId 上游素材 ID
 */
export const getMaterialAsset = async (assetId) => {
  const res = await API.get(MATERIAL_API.ASSET(assetId), {
    skipErrorHandler: true,
  });
  return res.data;
};

/**
 * 删除素材（按上游 asset_id）。
 * 后端会先调用上游 DeleteAsset 删除资产，再移除本地记录与临时文件。
 * @param {string} assetId 上游素材 ID
 */
export const deleteMaterialAsset = async (assetId) => {
  const res = await API.delete(MATERIAL_API.ASSET(assetId), {
    skipErrorHandler: true,
  });
  return res.data;
};

/**
 * 素材改名（虚拟人像）：上游 UpdateAsset + 本地同步。
 * @param {string} assetId 上游素材 ID
 * @param {string} name 新名称
 */
export const updateMaterialAsset = async (assetId, name) => {
  const res = await API.put(
    MATERIAL_API.ASSET(assetId),
    { name },
    { skipErrorHandler: true },
  );
  return res.data;
};

// ---------------------------------------------------------------------------
// 真人认证模块 API（Web 控制台，BytedToken 仅后端存储，前端仅持有 session_id）
// ---------------------------------------------------------------------------

/**
 * 创建真人认证会话（CreateVisualValidateSession）。
 * 后端存储 BytedToken，返回 session_id + H5Link + QrCode，前端展示二维码/链接。
 * @returns {Promise<{success:boolean,data:{session_id:number,h5_link:string,qr_code:string,expires_at:number,status:string}}>}
 */
export const createVisualSession = async () => {
  const res = await API.post(MATERIAL_API.VISUAL_SESSION, {}, {
    skipErrorHandler: true,
  });
  return res.data;
};

/**
 * 轮询真人认证结果。前端每 3s 调用一次，最大 5 分钟。
 * @param {number} sessionId 后端返回的会话 ID
 * @returns {Promise<{success:boolean,data:{status:string,group_id?:string,message?:string}}>}
 */
export const pollVisualResult = async (sessionId) => {
  const res = await API.get(MATERIAL_API.VISUAL_RESULT, {
    params: { session_id: sessionId },
    skipErrorHandler: true,
  });
  return res.data;
};

/**
 * 查询当前用户的所有真人认证分组。
 * @returns {Promise<{success:boolean,data:{items:Array}}>}
 */
export const listRealGroups = async () => {
  const res = await API.get(MATERIAL_API.REAL_GROUPS);
  return res.data;
};

/**
 * 删除真人认证分组。
 * @param {string} groupId 上游分组 ID
 * @returns {Promise<{success:boolean,data:{group_id:string}}>}
 */
export const deleteRealGroup = async (groupId) => {
  const res = await API.delete(MATERIAL_API.REAL_GROUP(groupId), {
    skipErrorHandler: true,
  });
  return res.data;
};

/**
 * 更新真人认证分组（名称、描述）。
 * @param {string} groupId 上游分组 ID
 * @param {{group_name?:string, description?:string}} data
 * @returns {Promise<{success:boolean,data:object}>}
 */
export const updateRealGroup = async (groupId, data) => {
  const res = await API.put(MATERIAL_API.REAL_GROUP(groupId), data, {
    skipErrorHandler: true,
  });
  return res.data;
};

/**
 * 分页查询真人素材列表。
 * @param {{groupId?:string,page?:number,pageSize?:number}} params
 */
export const listRealAssets = async ({ groupId, page = 1, pageSize = 100 } = {}) => {
  const params = { p: page, size: pageSize };
  if (groupId) params.group_id = groupId;
  const res = await API.get(MATERIAL_API.REAL_ASSETS, { params });
  return res.data;
};

/**
 * 本地文件上传真人素材。
 * @param {File} file 本地文件实例
 * @param {boolean} agreed 是否已勾选合规协议
 * @param {string} groupId 真人分组 ID
 * @param {{onUploadProgress?: (ev: import('axios').AxiosProgressEvent) => void}} [options]
 */
export const uploadRealMaterialFile = async (file, agreed, groupId, options = {}) => {
  const fd = new FormData();
  fd.append('file', file);
  fd.append('agreed', agreed ? 'true' : 'false');
  fd.append('group_id', groupId);
  const res = await API.post(MATERIAL_API.REAL_UPLOAD, fd, {
    skipErrorHandler: true,
    onUploadProgress: options.onUploadProgress,
  });
  return res.data;
};

/**
 * 在线资源链接上传真人素材。
 * @param {{url:string,name?:string,assetType?:string,agreed:boolean,groupId:string}} payload
 */
export const uploadRealMaterialByURL = async ({ url, name, assetType, agreed, groupId }) => {
  const res = await API.post(
    MATERIAL_API.REAL_UPLOAD_URL,
    {
      url,
      name: name || '',
      asset_type: assetType || '',
      agreed: !!agreed,
      group_id: groupId,
    },
    { skipErrorHandler: true },
  );
  return res.data;
};

/**
 * 查询单个真人素材详情（按上游 asset_id）。
 * @param {string} assetId 上游素材 ID
 */
export const getRealMaterial = async (assetId) => {
  const res = await API.get(MATERIAL_API.REAL_ASSET(assetId), {
    skipErrorHandler: true,
  });
  return res.data;
};

/**
 * 删除真人素材（按上游 asset_id）。
 * @param {string} assetId 上游素材 ID
 */
export const deleteRealMaterial = async (assetId) => {
  const res = await API.delete(MATERIAL_API.REAL_ASSET(assetId), {
    skipErrorHandler: true,
  });
  return res.data;
};

/**
 * 真人素材改名：上游 UpdateAsset + 本地同步。
 * @param {string} assetId 上游素材 ID
 * @param {string} name 新名称
 */
export const updateRealMaterial = async (assetId, name) => {
  const res = await API.put(
    MATERIAL_API.REAL_ASSET(assetId),
    { name },
    { skipErrorHandler: true },
  );
  return res.data;
};
