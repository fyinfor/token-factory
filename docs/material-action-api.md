# Tokease 素材库 Action API 使用文档

> 本文档面向中转站接入 Tokease 素材库 API 的开发者，介绍素材组与素材的全生命周期管理、真人认证等接口。
> 所有素材库 Action 接口通过统一的网关地址调用，采用 Action 路由 + Bearer Token 鉴权。

---

## 一、接口基础说明

### 请求域名

| 环境 | Base URL |
|------|----------|
| 生产环境 | `https://tokease.cn` |

所有素材库 Action 接口统一地址：

```http
POST {BaseURL}/api/material?Action={ActionName}
```

### 统一请求方式

| 项目 | 说明 |
|------|------|
| HTTP 方法 | `POST` |
| Content-Type | `application/json` |
| Action 传参 | Query 参数 `Action`，取值见下表 |
| 鉴权 | `Authorization: Bearer sk-{your_api_token}` |

### 支持的 Action 列表

按业务场景分组，推荐按下列顺序阅读与调用：

#### 真人认证（真人素材组）

| Action | 说明 |
|--------|------|
| `CreateVisualValidateSession` | 创建真人认证 H5 会话 |
| `GetVisualValidateResult` | 查询真人认证结果（轮询获取 GroupId） |

#### 素材组管理

| Action | 说明 |
|--------|------|
| `CreateAssetGroup` | 创建虚拟素材组 |
| `GetAssetGroup` | 查询素材组详情 |
| `UpdateAssetGroup` | 更新素材组名称/描述 |
| `DeleteAssetGroup` | 删除素材组 |

#### 素材管理

| Action | 说明 |
|--------|------|
| `CreateAsset` | 上传素材（通过公网 URL） |
| `GetAsset` | 查询素材详情与处理状态 |
| `UpdateAsset` | 更新素材名称 |
| `DeleteAsset` | 删除素材 |

### 全局返回格式

HTTP 状态码固定 **200**，业务成败看 `code` 字段：

```json
{
  "code": 0,
  "msg": "success",
  "data": {}
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `code` | int | `0` 成功；非 0 见错误码表 |
| `msg` | string | 成功时为 `"success"`；失败时为可读错误描述 |
| `data` | object | 业务数据，结构与上游 Result 对齐 |

### 通用错误码

| code | 说明 |
|------|------|
| `0` | 成功 |
| `400` | 参数无效（缺必填、JSON 格式错误、Action 不支持等） |
| `401` | 未授权（Token 缺失/无效） |
| `403` | 无权操作（资源不属于当前账号） |
| `404` | 资源不存在 |
| `500` | 服务内部错误或上游调用失败 |
| `503` | 素材库功能未启用或未开通 |

### 枚举参考

#### AssetType（素材类型）

| 值 | 说明 |
|----|------|
| `Image` | 图片 |
| `Video` | 视频 |
| `Audio` | 音频 |

#### Asset Status（素材状态）

| 值 | 说明 |
|----|------|
| `Active` | 资产可用，可用于视频生成引用 |
| `Pending` | 处理中，需继续轮询 `GetAsset` |
| `Failed` | 创建/处理失败，详见返回中的 `Error` 字段 |

### 特别说明

1. **`CreateVisualValidateSession` 与 `CreateAssetGroup` 的区别**：前者用于创建**真人资产组**认证会话；后者用于创建**虚拟素材组**。请根据业务场景选择，不可混用。
2. **真人素材组创建流程**：先调用 `CreateVisualValidateSession` 获取 H5 认证链接 → 用户完成手机实名认证 → 调用 `GetVisualValidateResult` 轮询直至返回 `GroupId`，**无需再填写其他信息**。
3. **获得 GroupId 后**：可直接调用 `CreateAsset` 上传素材，并通过 `GetAsset` / `UpdateAsset` / `DeleteAsset` 管理素材生命周期。
4. **素材上传耗时**：`CreateAsset` 会同步完成「文件下载 → 上传 OSS → 创建资产」全流程，耗时取决于文件大小和网络状况；上传后若 `Status` 为 `Pending`，请轮询 `GetAsset` 直至变为 `Active`。
5. **视频生成引用**：建议在调用 Seedance 视频生成前，先通过 `GetAsset` 确认素材状态为 `Active`，再使用 `asset://{AssetId}` 协议引用。
6. **BytedToken 安全**：`BytedToken` 仅供后端轮询 `GetVisualValidateResult` 使用，**不要在前端页面展示给用户**。
7. **认证轮询**：`GetVisualValidateResult` 建议 **3 秒**间隔轮询，最长 **5 分钟**；认证未完成时返回 `Status: pending`。

---

## 二、接口详细说明

### 真人认证

#### 1. CreateVisualValidateSession — 创建真人认证 H5 会话

| 项目 | 值 |
|------|-----|
| 接口地址 | `POST /api/material?Action=CreateVisualValidateSession` |
| Action | `CreateVisualValidateSession` |

**请求 Header**

```http
Authorization: Bearer sk-xxxxxxxx
Content-Type: application/json
```

**请求 Body**

无需参数，传空 JSON 即可：

```json
{}
```

**返回示例**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "BytedToken": "2026062515501465178B138B02A64E1A",
    "H5Link": "https://tokease.cn/real-validate?token=rt_...",
    "QrCode": "data:image/png;base64,iVBORw..."
  }
}
```

| 字段 | 说明 |
|------|------|
| `BytedToken` | 轮询令牌，供 `GetVisualValidateResult` 使用（**禁止前端展示**） |
| `H5Link` | 真人认证 H5 页面地址（约 5 分钟有效） |
| `QrCode` | H5 链接二维码（base64 PNG） |

**说明**：创建真人资产组认证会话。用户在 H5 页面完成手机实名认证后，通过 `GetVisualValidateResult` 轮询获取 `GroupId`。

---

#### 2. GetVisualValidateResult — 查询真人认证结果

| 项目 | 值 |
|------|-----|
| 接口地址 | `POST /api/material?Action=GetVisualValidateResult` |
| Action | `GetVisualValidateResult` |

**请求 Body**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `BytedToken` | string | 是 | 创建会话时返回的令牌 |

```json
{
  "BytedToken": "2026062515501465178B138B02A64E1A"
}
```

**返回示例（认证成功）**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "GroupId": "group-20260625155055-5h78r",
    "Status": "success"
  }
}
```

**返回示例（认证中）**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "Status": "pending"
  }
}
```

**轮询建议**：间隔 **3 秒**，最长 **5 分钟**。认证成功后系统自动创建真人素材分组并返回 `GroupId`。

---

### 素材组管理

#### 3. CreateAssetGroup — 创建虚拟素材组

| 项目 | 值 |
|------|-----|
| 接口地址 | `POST /api/material?Action=CreateAssetGroup` |
| Action | `CreateAssetGroup` |

**请求 Body**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `Name` | string | 是 | 素材组名称，最长 128 字符 |
| `Description` | string | 否 | 描述，最长 512 字符 |

```json
{
  "Name": "我的素材组",
  "Description": "用于存放角色立绘"
}
```

**返回示例**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "Id": "group-20260612195842-84j62"
  }
}
```

**说明**：创建**虚拟素材组**（非真人认证分组），返回素材组 ID 并绑定当前账号。

---

#### 4. GetAssetGroup — 查询素材组

| 项目 | 值 |
|------|-----|
| 接口地址 | `POST /api/material?Action=GetAssetGroup` |
| Action | `GetAssetGroup` |

**请求 Body**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `Id` | string | 是 | 素材组 ID |

```json
{
  "Id": "group-20260612195842-84j62"
}
```

**返回示例**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "Id": "group-20260612195842-84j62",
    "Name": "我的素材组",
    "Description": "用于存放角色立绘",
    "GroupType": "AIGC",
    "ProjectName": "default",
    "CreateTime": "2026-06-12T11:58:32Z",
    "UpdateTime": "2026-06-12T11:58:32Z"
  }
}
```

| 字段 | 说明 |
|------|------|
| `GroupType` | 虚拟素材组固定为 `AIGC` |
| `ProjectName` | 固定 `default` |
| `CreateTime` / `UpdateTime` | ISO 8601 时间格式 |

**说明**：查询指定素材组详情，仅可查询当前账号拥有的素材组。

---

#### 5. UpdateAssetGroup — 更新素材组

| 项目 | 值 |
|------|-----|
| 接口地址 | `POST /api/material?Action=UpdateAssetGroup` |
| Action | `UpdateAssetGroup` |

**请求 Body**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `Id` | string | 是 | 素材组 ID |
| `Name` | string | 否 | 新名称（与 Description 至少填一项） |
| `Description` | string | 否 | 新描述 |

```json
{
  "Id": "group-20260708214759-b62zr",
  "Name": "new-name",
  "Description": "new-description"
}
```

**返回示例**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "Id": "group-20260708214759-b62zr"
  }
}
```

---

#### 6. DeleteAssetGroup — 删除素材组

| 项目 | 值 |
|------|-----|
| 接口地址 | `POST /api/material?Action=DeleteAssetGroup` |
| Action | `DeleteAssetGroup` |

**请求 Body**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `Id` | string | 是 | 素材组 ID |

```json
{
  "Id": "group-20260617154226-dfxll"
}
```

**返回示例**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "Id": "group-20260617154226-dfxll"
  }
}
```

**说明**：删除整个素材组，请谨慎操作；仅可删除当前账号拥有的素材组。

---

### 素材管理

#### 7. CreateAsset — 上传素材

| 项目 | 值 |
|------|-----|
| 接口地址 | `POST /api/material?Action=CreateAsset` |
| Action | `CreateAsset` |

**请求 Body**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `GroupId` | string | 是 | 目标素材组 ID（虚拟组或真人认证返回的 GroupId） |
| `URL` | string | 是 | 素材文件公网 URL（支持 http/https） |
| `Name` | string | 是 | 素材名称 |
| `AssetType` | string | 是 | 资产类型：`Image` / `Video` / `Audio` |

```json
{
  "GroupId": "group-20260612195842-84j62",
  "URL": "https://example.com/image.jpg",
  "Name": "角色立绘01",
  "AssetType": "Image"
}
```

**返回示例**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "Id": "asset-20260612200812-mmq8g"
  }
}
```

**说明**：

- 接口会同步完成「文件下载 → 上传 OSS → 创建资产」全流程，耗时取决于文件大小和网络状况。
- 返回的 `Id` 即为后续 `GetAsset` / `UpdateAsset` / `DeleteAsset` 及视频生成 `asset://` 引用的 Asset ID。
- 上传完成后建议调用 `GetAsset` 确认 `Status` 为 `Active` 后再用于视频生成。

---

#### 8. GetAsset — 查询素材

| 项目 | 值 |
|------|-----|
| 接口地址 | `POST /api/material?Action=GetAsset` |
| Action | `GetAsset` |

**请求 Body**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `Id` | string | 是 | 素材 ID（由 CreateAsset 返回） |

```json
{
  "Id": "asset-20260612200812-mmq8g"
}
```

**返回示例**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "Id": "asset-20260612200812-mmq8g",
    "Name": "角色立绘01",
    "URL": "https://example.com/image.jpg",
    "AssetType": "Image",
    "GroupId": "group-20260612195842-84j62",
    "Status": "Active",
    "Moderation": {
      "Strategy": "Default"
    },
    "Error": {
      "Code": "",
      "Message": ""
    },
    "ProjectName": "default",
    "CreateTime": "2026-06-12T12:08:02Z",
    "UpdateTime": "2026-06-12T12:08:02Z"
  }
}
```

| 字段 | 说明 |
|------|------|
| `Status` | `Active` / `Pending` / `Failed`，见枚举参考 |
| `Moderation.Strategy` | 审核策略，固定 `Default` |
| `Error.Code` / `Error.Message` | 处理失败时的错误信息，正常时为空 |
| `ProjectName` | 固定 `default` |

**说明**：查询素材详情与处理状态。若 `Status` 为 `Pending`，请间隔 1～3 秒继续轮询直至变为 `Active` 或 `Failed`。

---

#### 9. UpdateAsset — 更新素材

| 项目 | 值 |
|------|-----|
| 接口地址 | `POST /api/material?Action=UpdateAsset` |
| Action | `UpdateAsset` |

**请求 Body**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `Id` | string | 是 | 素材 ID |
| `Name` | string | 是 | 新素材名称 |

```json
{
  "Id": "asset-20260708214814-hdbss",
  "Name": "new-name"
}
```

**返回示例**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "Id": "asset-20260708214814-hdbss"
  }
}
```

---

#### 10. DeleteAsset — 删除素材

| 项目 | 值 |
|------|-----|
| 接口地址 | `POST /api/material?Action=DeleteAsset` |
| Action | `DeleteAsset` |

**请求 Body**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `Id` | string | 是 | 素材 ID（`asset-xxxx`） |

```json
{
  "Id": "asset-20260617155448-fkgww"
}
```

**返回示例**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "Id": "asset-20260617155448-fkgww"
  }
}
```

---

## 三、终端用户素材库接入说明

### 3.1 接入前置准备

1. 在 Tokease 平台注册账号并创建 API Token（格式 `sk-xxx`）。
2. 确认您的账号已开通素材库功能权限（如未开通，请联系平台开通）。
3. 准备 HTTPS 可访问环境；所有请求使用 JSON Body + Bearer Token。
4. 素材文件需先上传至公网可访问的 URL（如 OSS、CDN），再调用 `CreateAsset` 传入。

### 3.2 调用鉴权方式

```http
Authorization: Bearer sk-your-api-token-here
Content-Type: application/json
```

Token 与账号一一绑定；平台根据 Token 识别您的身份，所有资源操作强制校验归属。

### 3.3 虚拟素材完整流程

```
1. CreateAssetGroup         → 获得 group-xxx（虚拟素材组）
2. CreateAsset              → 传入 GroupId + 公网 URL，获得 asset-xxx
3. GetAsset                 → 轮询确认 Status = Active
4. UpdateAsset              → （可选）修改素材名称
5. DeleteAsset              → （可选）删除单个素材
6. UpdateAssetGroup         → （可选）修改分组名称/描述
7. DeleteAssetGroup         → （可选）删除整个分组
```

### 3.4 真人认证 + 素材上传完整流程

```
1. CreateVisualValidateSession
      ↓ 返回 H5Link / QrCode / BytedToken（BytedToken 仅后端保存）
2. 引导用户在手机打开 H5Link 完成实名认证
3. 每 3s 调用 GetVisualValidateResult（携带 BytedToken）
      ↓ Status=pending 继续轮询
      ↓ Status=success 获得 GroupId（真人分组）
4. CreateAsset               → 向该 GroupId 上传素材
5. GetAsset                  → 确认 Status = Active
6. 在视频生成中使用 asset://{AssetId} 引用
```

### 3.5 资源权限隔离规则

| 规则 | 说明 |
|------|------|
| 账号隔离 | 仅能操作当前账号创建或绑定的分组/素材/认证会话 |
| 跨账号拒绝 | 使用他人 `Id` / `BytedToken` 返回 `code=403` |
| 分组归属校验 | `CreateAsset` 仅可向当前账号拥有的 GroupId 上传 |
| BytedToken 绑定 | 每个 BytedToken 仅对应创建时的账号，不可跨账号轮询 |

### 3.6 常见调用报错排查

| msg / 现象 | 可能原因 | 处理建议 |
|------------|----------|----------|
| `未授权，请携带有效的 API Token` | Header 缺失或 Token 无效 | 检查 `Authorization: Bearer sk-xxx` |
| `Action 参数不能为空` | URL 未带 Action | 追加 `?Action=CreateAssetGroup` |
| `GroupId 不能为空` | CreateAsset 缺参 | 补全 GroupId / URL / Name / AssetType |
| `分组不存在或无权操作` | Id 错误或归属他人 | 确认 Id 来自当前账号创建结果 |
| `认证会话不存在或无权操作` | BytedToken 错误或跨账号 | 使用本会话创建接口返回的 Token |
| `素材库功能未启用...` (503) | 账号未开通素材库功能 | 联系平台开通 |
| `上传素材失败: ...` (500) | URL 不可访问或上游异常 | 确认 URL 公网可达，稍后重试 |
| GetAsset 一直 `Pending` | 上游仍在处理 | 间隔 1～3s 继续轮询 |
| 认证一直 `pending` | 用户未完成 H5 认证 | 确认 H5 页面操作成功，5 分钟内有效 |

---

## 四、curl 快速验证

```bash
# 1. 创建虚拟素材组
curl -X POST "https://tokease.cn/api/material?Action=CreateAssetGroup" \
  -H "Authorization: Bearer sk-xxx" \
  -H "Content-Type: application/json" \
  -d '{"Name":"测试组","Description":"描述"}'

# 2. 上传素材
curl -X POST "https://tokease.cn/api/material?Action=CreateAsset" \
  -H "Authorization: Bearer sk-xxx" \
  -H "Content-Type: application/json" \
  -d '{"GroupId":"group-20260612195842-84j62","URL":"https://example.com/image.jpg","Name":"角色立绘01","AssetType":"Image"}'

# 3. 查询素材状态
curl -X POST "https://tokease.cn/api/material?Action=GetAsset" \
  -H "Authorization: Bearer sk-xxx" \
  -H "Content-Type: application/json" \
  -d '{"Id":"asset-20260612200812-mmq8g"}'

# 4. 查询素材组
curl -X POST "https://tokease.cn/api/material?Action=GetAssetGroup" \
  -H "Authorization: Bearer sk-xxx" \
  -H "Content-Type: application/json" \
  -d '{"Id":"group-20260612195842-84j62"}'
```
