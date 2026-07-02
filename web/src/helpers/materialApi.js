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
