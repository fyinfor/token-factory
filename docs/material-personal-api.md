# 个人素材 API 接口文档

> **基础域名**：`https://tokease.cn`  
> **鉴权方式**：所有接口均通过用户 API 令牌（`sk-xxx`）鉴权，请求头统一携带：  
> `Authorization: Bearer sk-xxxxxxxxxxxxxxxxxxxxxxxx`

---

## 接口概览

| 接口 | 请求方式 | 路径 | 说明 |
|------|----------|------|------|
| 个人素材上传（本地文件） | POST | `/api/material/personal/upload` | 上传图片/视频文件，仅归属当前 Token 用户 |
| 个人素材上传（在线链接） | POST | `/api/material/personal/upload-url` | 通过在线图片/视频链接上传素材，仅归属当前 Token 用户 |
| 个人素材列表查询 | GET | `/api/material/personal/assets` | 分页查询当前 Token 用户的个人素材列表 |
| 个人素材删除 | DELETE | `/api/material/personal/asset/:asset_id` | 删除指定 `asset_id` 的个人素材 |
| 个人素材详情查询 | GET | `/api/material/personal/asset/:asset_id` | 查询指定 `asset_id` 的个人素材详情 |

---

## 通用说明

### 请求头

| 名称 | 必填 | 示例值 | 说明 |
|------|------|--------|------|
| `Authorization` | 是 | `Bearer sk-abc123...` | 用户 API 令牌，用于识别归属用户 |
| `Content-Type` | 视接口 | `multipart/form-data` / 无需设置 | 上传接口使用 `multipart/form-data` |

### 通用响应结构

```json
{
  "success": true,
  "message": "",
  "data": { }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `success` | boolean | 请求是否成功 |
| `message` | string | 失败时的错误描述，成功时为空 |
| `data` | any | 业务数据 |

### 错误码说明

| HTTP 状态码 | 错误说明 |
|-------------|----------|
| `200` 但 `success=false` | 业务错误，详见 `message` |
| `401 Unauthorized` | `Authorization` 缺失、格式错误、Token 无效或已过期 |
| `403 Forbidden` | Token 用户被封禁或无权访问 |

常见业务错误：

| message | 说明 |
|---------|------|
| 素材库功能未启用或基础地址未配置，请联系管理员 | 系统未开启素材库 |
| 未授权 | Token 未识别到用户 |
| 用户无效 | Token 归属用户不存在 |
| 无上传权限 | 用户角色无文件上传权限 |
| 请先阅读并勾选同意虚拟人像合规协议 | 缺少 `agreed=true` |
| 请选择文件字段 file | 未上传文件 |
| 仅支持上传图片或视频格式... | 文件扩展名不在白名单 |
| 文件超过大小限制（最大 N MB） | 单文件超出系统限制 |
| 文件上传未启用 | 运营设置中未启用文件上传 |
| 素材不存在或无权操作 | 素材 ID 不存在或不属于当前用户 |

---

## 1. 个人素材上传

### 接口地址

```http
POST https://tokease.cn/api/material/personal/upload
```

### 请求头

```http
Authorization: Bearer sk-xxxxxxxxxxxxxxxxxxxxxxxx
Content-Type: multipart/form-data
```

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `file` | File | 是 | 本地图片或视频文件 |
| `agreed` | string | 是 | 是否已同意虚拟人像合规协议，固定传 `"true"` |

### 请求示例（curl）

```bash
curl -X POST https://tokease.cn/api/material/personal/upload \
  -H "Authorization: Bearer sk-xxxxxxxxxxxxxxxxxxxxxxxx" \
  -F "file=@/path/to/portrait.jpg" \
  -F "agreed=true"
```

### 成功响应示例

```json
{
  "success": true,
  "message": "",
  "data": {
    "asset_id": "asset-a1b2c3d4",
    "asset_uri": "asset://asset-a1b2c3d4",
    "name": "portrait.jpg",
    "asset_type": "Image",
    "url": "https://tokease.cn/api/uploads/seedance/2026/06/25/xxxxxxxx.jpg",
    "status": "Active",
    "created_at": 1750838400
  }
}
```

### 失败响应示例

```json
{
  "success": false,
  "message": "仅支持上传图片或视频格式（图片：jpg/jpeg/png/webp/gif/bmp；视频：mp4/mov/webm/mkv/avi/m4v）",
  "data": null
}
```

---

## 2. 个人素材上传（在线链接）

### 接口地址

```http
POST https://tokease.cn/api/material/personal/upload-url
```

### 请求头

```http
Authorization: Bearer sk-xxxxxxxxxxxxxxxxxxxxxxxx
Content-Type: application/json
```

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `url` | string | 是 | 在线图片或视频链接，必须为 `http` 或 `https` 绝对地址 |
| `name` | string | 否 | 素材名称；未填写时自动取链接文件名 |
| `asset_type` | string | 否 | 素材类型：`Image` / `Video`；未填写时自动按链接扩展名识别，默认 `Image` |
| `agreed` | boolean | 是 | 是否已同意虚拟人像合规协议，固定传 `true` |

### 请求示例（curl）

```bash
curl -X POST https://tokease.cn/api/material/personal/upload-url \
  -H "Authorization: Bearer sk-xxxxxxxxxxxxxxxxxxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com/portrait.jpg",
    "name": "portrait.jpg",
    "asset_type": "Image",
    "agreed": true
  }'
```

### 成功响应示例

```json
{
  "success": true,
  "message": "",
  "data": {
    "asset_id": "asset-a1b2c3d4",
    "asset_uri": "asset://asset-a1b2c3d4",
    "name": "portrait.jpg",
    "asset_type": "Image",
    "url": "https://example.com/portrait.jpg",
    "status": "Active",
    "created_at": 1750838400
  }
}
```

### 失败响应示例

```json
{
  "success": false,
  "message": "请输入合法的在线资源链接（http/https）",
  "data": null
}
```

---

## 3. 个人素材列表查询

### 接口地址

```http
GET https://tokease.cn/api/material/personal/assets
```

### 请求头

```http
Authorization: Bearer sk-xxxxxxxxxxxxxxxxxxxxxxxx
```

### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `page` | integer | 否 | 当前页码，默认 `1` |
| `page_size` | integer | 否 | 每页条数，默认 `10` |

### 请求示例（curl）

```bash
curl -X GET "https://tokease.cn/api/material/personal/assets?page=1&page_size=10" \
  -H "Authorization: Bearer sk-xxxxxxxxxxxxxxxxxxxxxxxx"
```

### 成功响应示例

```json
{
  "success": true,
  "message": "",
  "data": {
    "page": 1,
    "page_size": 10,
    "total": 2,
    "items": [
      {
        "asset_id": "asset-a1b2c3d4",
        "asset_uri": "asset://asset-a1b2c3d4",
        "name": "portrait.jpg",
        "asset_type": "Image",
        "url": "https://tokease.cn/api/uploads/seedance/2026/06/25/xxxxxxxx.jpg",
        "status": "Active",
        "created_at": 1750838400
      },
      {
        "asset_id": "asset-e5f6g7h8",
        "asset_uri": "asset://asset-e5f6g7h8",
        "name": "portrait.jpg",
        "asset_type": "Image",
        "url": "https://example.com/portrait.jpg",
        "status": "Active",
        "created_at": 1750838400
      }
    ]
  }
}
```

### 失败响应示例

```json
{
  "success": false,
  "message": "未授权",
  "data": null
}
```

---

## 4. 个人素材删除

### 接口地址

```http
DELETE https://tokease.cn/api/material/personal/asset/:id
```

### 请求头

```http
Authorization: Bearer sk-xxxxxxxxxxxxxxxxxxxxxxxx
```

### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `asset_id` | string | 是 | 素材对外唯一标识（即上传接口返回的 `asset_id`） |

### 请求示例（curl）

```bash
curl -X DELETE https://tokease.cn/api/material/personal/asset/asset-a1b2c3d4 \
  -H "Authorization: Bearer sk-xxxxxxxxxxxxxxxxxxxxxxxx"
```

### 成功响应示例

```json
{
  "success": true,
  "message": "",
  "data": {
    "asset_id": "asset-a1b2c3d4"
  }
}
```

### 失败响应示例

```json
{
  "success": false,
  "message": "素材不存在或无权操作",
  "data": null
}
```

---

## 5. 个人素材详情查询

### 接口地址

```http
GET https://tokease.cn/api/material/personal/asset/:id
```

### 请求头

```http
Authorization: Bearer sk-xxxxxxxxxxxxxxxxxxxxxxxx
```

### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `asset_id` | string | 是 | 素材对外唯一标识（即上传接口返回的 `asset_id`） |

### 请求示例（curl）

```bash
curl -X GET https://tokease.cn/api/material/personal/asset/asset-a1b2c3d4 \
  -H "Authorization: Bearer sk-xxxxxxxxxxxxxxxxxxxxxxxx"
```

### 成功响应示例

```json
{
  "success": true,
  "message": "",
  "data": {
    "asset_id": "asset-a1b2c3d4",
    "asset_uri": "asset://asset-a1b2c3d4",
    "name": "portrait.jpg",
    "asset_type": "Image",
    "url": "https://tokease.cn/api/uploads/seedance/2026/06/25/xxxxxxxx.jpg",
    "status": "Active",
    "created_at": 1750838400
  }
}
```

### 失败响应示例

```json
{
  "success": false,
  "message": "素材不存在或无权操作",
  "data": null
}
```

---

## 数据字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `asset_id` | string | 素材对外唯一标识，后续删除/查询接口均使用该字段 |
| `asset_uri` | string | 业务资源地址，格式为 `asset://asset_id`，可用于视频生成请求替换素材 |
| `name` | string | 素材名称 |
| `asset_type` | string | 素材类型：`Image` / `Video` |
| `url` | string | 素材公网访问地址 |
| `status` | string | 素材状态：`Active` / `Pending` / `Failed` |
| `created_at` | integer | 创建时间戳（秒） |

---

## 注意事项

1. **数据隔离**：所有接口均通过 `Authorization` 中的 Token 自动识别归属用户，所有操作仅作用于该用户的个人素材，无法访问或操作其他用户的素材。
2. **上传方式**：支持两种上传方式，本地文件上传请使用 `/api/material/personal/upload`，在线图片/视频链接上传请使用 `/api/material/personal/upload-url`。
3. **文件限制**：支持的图片格式为 `jpg/jpeg/png/webp/gif/bmp`，视频格式为 `mp4/mov/webm/mkv/avi/m4v`，单文件大小由系统运营设置决定（默认最大 10MB）。
4. **合规协议**：上传接口必须携带 `agreed=true`，表示已阅读并同意虚拟人像合规协议。
5. **状态刷新**：列表查询与详情查询接口会对仍处于 `Pending` 或本地临时 URL 状态的素材执行一次上游状态刷新。
