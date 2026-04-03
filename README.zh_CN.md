<div align="center">

![token-factory](/web/public/logo.png)

# TokenFactory

**自托管的 AI 网关：** 把多家模型供应商收口到统一 API，配套用户/令牌/配额与 Web 管理台。

---

### 上游仓库（请务必阅读）

**TokenFactory 派生自 [QuantumNous/new-api](https://github.com/QuantumNous/new-api)（New API）。** 该仓库是设计理念、协议覆盖与社区演进的主要来源；本 fork 可能与之不同，行为与接口请以官方文档为基准，并结合你实际运行的版本验证。

| | |
| --- | --- |
| **上游仓库** | **[github.com/QuantumNous/new-api](https://github.com/QuantumNous/new-api)** |
| **许可** | [GNU AGPL v3.0](./LICENSE) — 本仓库修改同样适用；详见 [`NOTICE`](./NOTICE) |
| **网络提供服务** | 若向他人提供修改版的网络访问，须遵守 **AGPL 第 13 条**，提供对应完整源代码（同等许可）。 |

---

<p align="center">
  简体中文 |
  <a href="./README.zh_TW.md">繁體中文</a> |
  <a href="./README.md">English</a> |
  <a href="./README.fr.md">Français</a> |
  <a href="./README.ja.md">日本語</a>
</p>

<p align="center">
  <a href="./LICENSE"><img src="https://img.shields.io/badge/License-AGPL--v3-blue.svg" alt="AGPL-3.0"></a>
  &nbsp;
  <a href="https://github.com/QuantumNous/new-api"><img src="https://img.shields.io/badge/上游-QuantumNous%2Fnew--api-555555?logo=github" alt="上游 QuantumNous/new-api"></a>
  &nbsp;
  <a href="https://github.com/QuantumNous/token-factory"><img src="https://img.shields.io/badge/本仓库-TokenFactory-2ea043?logo=github" alt="本仓库"></a>
</p>

<p align="center">
  <a href="#快速开始">快速开始</a> •
  <a href="#文档">文档</a> •
  <a href="#部署">部署</a> •
  <a href="#许可证">许可证</a> •
  <a href="#帮助">帮助</a>
</p>

</div>

## 关于本仓库

TokenFactory 提供的是一套可 **私有化部署的控制面**：集中配置渠道、模型映射、访问策略与用量观测，由你本地运行服务并对外暴露端点。

本 README 只描述 **本 fork** 的定位与入口，**不**替代上游的功能清单与法律文本——后者始终与 **[QuantumNous/new-api](https://github.com/QuantumNous/new-api)** 及本树中的许可文件绑定。

## 合规与免责

- 使用须遵守各模型/平台条款（如 OpenAI [使用条款](https://openai.com/policies/terms-of-use)）及所在地**法律法规**，禁止违法或滥用。
- 在中国大陆请遵守生成式人工智能服务备案等要求（如 [《生成式人工智能服务管理暂行办法》](http://www.cac.gov.cn/2023-07/13/c_1690898327029107.htm)），勿向公众提供未按规定完成的生成式 AI 服务。
- **不作稳定性或服务承诺**：除非你自行或第三方提供支持，否则按基础设施软件自行运维。

---

## 快速开始

### 使用 Docker Compose（推荐）

```bash
# 克隆项目
git clone https://github.com/QuantumNous/token-factory.git
cd token-factory

# 编辑 docker-compose.yml 配置
nano docker-compose.yml

# 启动服务
docker-compose up -d
```

<details>
<summary><strong>使用 Docker 命令</strong></summary>

```bash
# 拉取最新镜像
docker pull ghcr.io/fyinfor/token-factory:latest

# 使用 SQLite（默认）
docker run --name token-factory -d --restart always \
  -p 3000:3000 \
  -e TZ=Asia/Shanghai \
  -v ./data:/data \
  ghcr.io/fyinfor/token-factory:latest

# 使用 MySQL
docker run --name token-factory -d --restart always \
  -p 3000:3000 \
  -e SQL_DSN="root:123456@tcp(localhost:3306)/oneapi" \
  -e TZ=Asia/Shanghai \
  -v ./data:/data \
  ghcr.io/fyinfor/token-factory:latest
```

> **💡 提示：** `-v ./data:/data` 会将数据保存在当前目录的 `data` 文件夹中，你也可以改为绝对路径如 `-v /your/custom/path:/data`

</details>

---

服务就绪后访问 **`http://localhost:3000`**。更多安装方式见 **[安装文档](https://docs.newapi.pro/zh/docs/installation)**。

---

## 文档

**[QuantumNous/new-api](https://github.com/QuantumNous/new-api)** 生态站点发布 API、模型与运维说明；TokenFactory 继承该栈，请以官方文档为**准绳**，并结合你本地构建验证。

| | |
| --- | --- |
| 手册（中 / 英） | [简体中文](https://docs.newapi.pro/zh/docs) · [English](https://docs.newapi.pro/en/docs) |
| 环境变量 | [配置说明](https://docs.newapi.pro/zh/docs/installation/config-maintenance/environment-variables) |
| 中继 / REST | [API 文档](https://docs.newapi.pro/zh/docs/api) |
| 功能总览 | [特性说明](https://docs.newapi.pro/zh/docs/guide/wiki/basic-concepts/features-introduction) |
| 问答与社区 | [FAQ](https://docs.newapi.pro/zh/docs/support/faq) · [交流渠道](https://docs.newapi.pro/zh/docs/support/community-interaction) |
| 第三方深度梳理 | [DeepWiki — QuantumNous/new-api](https://deepwiki.com/QuantumNous/new-api) |

**本仓库：** 打包、默认配置、CI 等 **fork 特有问题** 请在本仓库提 Issue。若与上游行为一致，请先在 **[QuantumNous/new-api](https://github.com/QuantumNous/new-api)** 复现并按其规范反馈。

---

## 能力概览（精简）

以下为对上游能力的**摘要**，非完整规格：

- **中继** — 多家供应商适配器，统一对外 API（含 OpenAI 兼容及其他格式，详见上游）。
- **控制台** — 渠道、模型映射、用户与令牌、用量与计费相关配置。
- **策略** — 配额、限流、失败重试；启用 Redis 时可使用缓存等能力（见上游文档）。
- **存储** — SQLite / MySQL / PostgreSQL；可选 Redis（会话、缓存、加解密等按文档配置）。

逐模型、逐接口细节请查阅 **[API 文档](https://docs.newapi.pro/zh/docs/api)** 与 **[QuantumNous/new-api](https://github.com/QuantumNous/new-api)** 的发行说明。

---

## 部署

> [!TIP]
> **最新版 Docker 镜像：** `ghcr.io/fyinfor/token-factory:latest`

### 📋 部署要求

| 组件 | 要求 |
|------|------|
| **本地数据库** | SQLite（Docker 需挂载 `/data` 目录）|
| **远程数据库** | MySQL ≥ 5.7.8 或 PostgreSQL ≥ 9.6 |
| **容器引擎** | Docker / Docker Compose |

### ⚙️ 环境变量配置

<details>
<summary>常用环境变量配置</summary>

| 变量名 | 说明                                                           | 默认值 |
|--------|--------------------------------------------------------------|--------|
| `SESSION_SECRET` | 会话密钥（多机部署必须）                                                 | - |
| `CRYPTO_SECRET` | 加密密钥（Redis 必须）                                               | - |
| `SQL_DSN` | 数据库连接字符串                                                     | - |
| `REDIS_CONN_STRING` | Redis 连接字符串                                                  | - |
| `STREAMING_TIMEOUT` | 流式超时时间（秒）                                                    | `300` |
| `STREAM_SCANNER_MAX_BUFFER_MB` | 流式扫描器单行最大缓冲（MB），图像生成等超大 `data:` 片段（如 4K 图片 base64）需适当调大 | `64` |
| `MAX_REQUEST_BODY_MB` | 请求体最大大小（MB，**解压后**计；防止超大请求/zip bomb 导致内存暴涨），超过将返回 `413` | `32` |
| `AZURE_DEFAULT_API_VERSION` | Azure API 版本                                                 | `2025-04-01-preview` |
| `ERROR_LOG_ENABLED` | 错误日志开关                                                       | `false` |
| `PYROSCOPE_URL` | Pyroscope 服务地址                                            | - |
| `PYROSCOPE_APP_NAME` | Pyroscope 应用名                                        | `token-factory` |
| `PYROSCOPE_BASIC_AUTH_USER` | Pyroscope Basic Auth 用户名                        | - |
| `PYROSCOPE_BASIC_AUTH_PASSWORD` | Pyroscope Basic Auth 密码                  | - |
| `PYROSCOPE_MUTEX_RATE` | Pyroscope mutex 采样率                               | `5` |
| `PYROSCOPE_BLOCK_RATE` | Pyroscope block 采样率                               | `5` |
| `HOSTNAME` | Pyroscope 标签里的主机名                                          | `token-factory` |

📖 **完整配置：** [环境变量文档](https://docs.newapi.pro/zh/docs/installation/config-maintenance/environment-variables)

</details>

### 🔧 部署方式

**Docker Compose：** 见上文 [快速开始](#快速开始)（克隆 → 编辑 `docker-compose.yml` → `docker-compose up -d`）。

<details>
<summary><strong>备选：直接 docker run</strong></summary>

**使用 SQLite：**
```bash
docker run --name token-factory -d --restart always \
  -p 3000:3000 \
  -e TZ=Asia/Shanghai \
  -v ./data:/data \
  ghcr.io/fyinfor/token-factory:latest
```

**使用 MySQL：**
```bash
docker run --name token-factory -d --restart always \
  -p 3000:3000 \
  -e SQL_DSN="root:123456@tcp(localhost:3306)/oneapi" \
  -e TZ=Asia/Shanghai \
  -v ./data:/data \
  ghcr.io/fyinfor/token-factory:latest
```

> **💡 路径说明：**
> - `./data:/data` - 相对路径，数据保存在当前目录的 data 文件夹
> - 也可使用绝对路径，如：`/your/custom/path:/data`

</details>

<details>
<summary><strong>宝塔面板</strong></summary>

1. 安装宝塔面板（≥ 9.2.0 版本）
2. 在应用商店搜索 **TokenFactory**
3. 一键安装

📖 [图文教程](./docs/installation/BT.md)

</details>

### ⚠️ 多机部署注意事项

> [!WARNING]
> - **必须设置** `SESSION_SECRET` - 否则登录状态不一致
> - **公用 Redis 必须设置** `CRYPTO_SECRET` - 否则数据无法解密

### 🔄 渠道重试与缓存

**重试配置：** `设置 → 运营设置 → 通用设置 → 失败重试次数`

**缓存配置：**
- `REDIS_CONN_STRING`：Redis 缓存（推荐）
- `MEMORY_CACHE_ENABLED`：内存缓存

---

## 谱系

| 仓库 | 角色 |
| --- | --- |
| **[QuantumNous/new-api](https://github.com/QuantumNous/new-api)** | **上游** — New API（AGPL-3.0）。**非本 fork 专属问题**、功能设计与主社区请优先关注此处。 |
| [One API](https://github.com/songquanpeng/one-api) | 同族谱系中更早的 MIT 许可实现。 |
| [Midjourney-Proxy](https://github.com/novicezk/midjourney-proxy) | 可选 Midjourney 对接（详见上游文档）。 |

周边工具（如 [neko-api-key-tool](https://github.com/Calcium-Ion/neko-api-key-tool)）见上游与社区说明。

---

## 帮助

### 📖 文档资源

| 资源 | 链接 |
|------|------|
| 📘 常见问题 | [FAQ](https://docs.newapi.pro/zh/docs/support/faq) |
| 💬 社区交流 | [交流渠道](https://docs.newapi.pro/zh/docs/support/community-interaction) |
| 🐛 反馈问题 | [问题反馈](https://docs.newapi.pro/zh/docs/support/feedback-issues) |
| 📚 完整文档 | [官方文档](https://docs.newapi.pro/zh/docs) |

### 🤝 贡献指南

欢迎各种形式的贡献！

- 🐛 报告 Bug
- 💡 提出新功能
- 📝 改进文档
- 🔧 提交代码

---

## 许可证

本项目（**TokenFactory**）采用 [GNU Affero 通用公共许可证 v3.0 (AGPLv3)](./LICENSE) 授权；后续修改与再衍生作品在 AGPL-3.0 下继续适用，除非您另行取得著作权人的商业许可。

**署名说明：** TokenFactory 派生自 [QuantumNous/new-api](https://github.com/QuantumNous/new-api)（New API），上游亦为 AGPL-3.0；项目链条中更早的基础为 [One API](https://github.com/songquanpeng/one-api)（MIT 许可证）。请保留上游与本仓库的版权声明、[`LICENSE`](./LICENSE) 及 [`NOTICE`](./NOTICE)。**AGPL 第 13 条：** 若您将修改版以网络服务形式向他人提供，须向其提供对应完整源代码（同等许可）。

如果您所在的组织政策不允许使用 AGPLv3 许可的软件，或您希望规避 AGPLv3 的开源义务，请发送邮件至：[support@quantumnous.com](mailto:support@quantumnous.com)

---

<div align="center">

**TokenFactory** — 本 fork 提供的自托管 AI 网关发行版。

**上游：** **[QuantumNous/new-api](https://github.com/QuantumNous/new-api)** · **文档：** [docs.newapi.pro](https://docs.newapi.pro/zh/docs) · **本仓库：** [Issues](https://github.com/QuantumNous/token-factory/issues)

<sub>New API 项目由 **QuantumNous** 与贡献者维护。JetBrains 通过免费 IDE 许可支持开源开发。</sub>

</div>
