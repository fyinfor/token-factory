<div align="center">

![token-factory](/web/public/logo.png)

# TokenFactory

**AI gateway you self-host:** route many model providers through one API surface, with users, keys, quotas, and an admin UI.

---

### Upstream (required reading)

**TokenFactory is derived from the [QuantumNous/new-api](https://github.com/QuantumNous/new-api) project (“New API”).** That repository is the authoritative upstream for design, protocol coverage, and community history. This fork may diverge; for behavior and APIs, treat upstream docs as the baseline and verify against your build.

| | |
| --- | --- |
| **Upstream repository** | **[github.com/QuantumNous/new-api](https://github.com/QuantumNous/new-api)** |
| **License** | [GNU AGPL v3.0](./LICENSE) — same for modifications here; see [`NOTICE`](./NOTICE) |
| **Network use** | If you offer a modified version over a network to others, **AGPL-3.0 section 13** requires you to provide the corresponding full source under the same license. |

---

<p align="center">
  <a href="./README.zh_CN.md">简体中文</a> |
  <a href="./README.zh_TW.md">繁體中文</a> |
  <strong>English</strong> |
  <a href="./README.fr.md">Français</a> |
  <a href="./README.ja.md">日本語</a>
</p>

<p align="center">
  <a href="./LICENSE"><img src="https://img.shields.io/badge/License-AGPL--v3-blue.svg" alt="AGPL-3.0"></a>
  &nbsp;
  <a href="https://github.com/QuantumNous/new-api"><img src="https://img.shields.io/badge/Upstream-QuantumNous%2Fnew--api-555555?logo=github" alt="Upstream: QuantumNous/new-api"></a>
  &nbsp;
  <a href="https://github.com/QuantumNous/token-factory"><img src="https://img.shields.io/badge/This_repo-TokenFactory-2ea043?logo=github" alt="This repository"></a>
</p>

<p align="center">
  <a href="#quick-start">Quick Start</a> •
  <a href="#documentation">Documentation</a> •
  <a href="#deployment">Deployment</a> •
  <a href="#license">License</a> •
  <a href="#help">Help</a>
</p>

</div>

## About this repository

TokenFactory is a **self-hosted control plane** for aggregating upstream AI vendors: one place to configure channels, map models, enforce access, and observe usage. You run the binary (or container), point clients at it, and manage everything from the web console.

This README describes **this fork’s** packaging and pointers. It does not replace the upstream feature list or legal notices—those remain tied to [QuantumNous/new-api](https://github.com/QuantumNous/new-api) and the license files in this tree.

## Compliance & disclaimer

- Use only in line with provider terms (e.g. OpenAI [Terms of Use](https://openai.com/policies/terms-of-use)) and **applicable law**. No illegal or abusive use.
- In China, follow registration and compliance rules for generative AI services (e.g. [《生成式人工智能服务管理暂行办法》](http://www.cac.gov.cn/2023-07/13/c_1690898327029107.htm)); do not offer unregistered public generative AI services where prohibited.
- No warranty: treat this as **self-supported** infrastructure unless you arrange your own support.

---

## Quick Start

### Using Docker Compose (Recommended)

```bash
# Clone the project
git clone https://github.com/QuantumNous/token-factory.git
cd token-factory

# Edit docker-compose.yml configuration
nano docker-compose.yml

# Start the service
docker-compose up -d
```

<details>
<summary><strong>Using Docker Commands</strong></summary>

```bash
# Pull the latest image
docker pull ghcr.io/fyinfor/token-factory:latest

# Using SQLite (default)
docker run --name token-factory -d --restart always \
  -p 3000:3000 \
  -e TZ=Asia/Shanghai \
  -v ./data:/data \
  ghcr.io/fyinfor/token-factory:latest

# Using MySQL
docker run --name token-factory -d --restart always \
  -p 3000:3000 \
  -e SQL_DSN="root:123456@tcp(localhost:3306)/oneapi" \
  -e TZ=Asia/Shanghai \
  -v ./data:/data \
  ghcr.io/fyinfor/token-factory:latest
```

> **💡 Tip:** `-v ./data:/data` will save data in the `data` folder of the current directory, you can also change it to an absolute path like `-v /your/custom/path:/data`

</details>

---

When the stack is healthy, open **`http://localhost:3000`**. More install paths (bare metal, panels, etc.): **[installation docs](https://docs.newapi.pro/en/docs/installation)**.

---

## Documentation

The **[QuantumNous/new-api](https://github.com/QuantumNous/new-api)** ecosystem publishes the reference manuals for APIs, models, and operations. TokenFactory tracks that stack; use the docs as the source of truth and validate against your build.

| | |
| --- | --- |
| Manual (EN / ZH) | [docs.newapi.pro — English](https://docs.newapi.pro/en/docs) · [简体中文](https://docs.newapi.pro/zh/docs) |
| Environment variables | [Configuration reference](https://docs.newapi.pro/en/docs/installation/config-maintenance/environment-variables) |
| Relay / REST API | [API documentation](https://docs.newapi.pro/en/docs/api) |
| Feature overview | [Features introduction](https://docs.newapi.pro/en/docs/guide/wiki/basic-concepts/features-introduction) |
| FAQ & community | [FAQ](https://docs.newapi.pro/en/docs/support/faq) · [Channels](https://docs.newapi.pro/en/docs/support/community-interaction) |
| Deep dive (third-party) | [DeepWiki — QuantumNous/new-api](https://deepwiki.com/QuantumNous/new-api) |

**This repository:** report **fork-specific** bugs (packaging, defaults, CI) here. If the behavior matches upstream, reproduce on **[QuantumNous/new-api](https://github.com/QuantumNous/new-api)** and follow their contribution guidelines.

---

## What you get (at a glance)

Capabilities come from the upstream codebase; this is a **summary**, not an exhaustive spec:

- **Relay** — many vendor adapters behind a unified API surface (OpenAI-compatible and other formats per upstream).
- **Console** — channels, model mapping, users, keys, usage and billing configuration.
- **Policies** — quotas, rate limits, retries, and optional cache when Redis is enabled.
- **Storage** — SQLite, MySQL, or PostgreSQL; optional Redis for sessions/cache/crypto as documented upstream.

For model-by-model and endpoint-by-endpoint detail, use the **[API docs](https://docs.newapi.pro/en/docs/api)** and **[QuantumNous/new-api](https://github.com/QuantumNous/new-api)** releases.

---

## Deployment

> [!TIP]
> **Latest Docker image:** `ghcr.io/fyinfor/token-factory:latest`

### 📋 Deployment Requirements

| Component | Requirement |
|------|------|
| **Local database** | SQLite (Docker must mount `/data` directory)|
| **Remote database** | MySQL ≥ 5.7.8 or PostgreSQL ≥ 9.6 |
| **Container engine** | Docker / Docker Compose |

### ⚙️ Environment Variable Configuration

<details>
<summary>Common environment variable configuration</summary>

| Variable Name | Description | Default Value |
|--------|------|--------|
| `SESSION_SECRET` | Session secret (required for multi-machine deployment) | - |
| `CRYPTO_SECRET` | Encryption secret (required for Redis) | - |
| `SQL_DSN` | Database connection string | - |
| `REDIS_CONN_STRING` | Redis connection string | - |
| `STREAMING_TIMEOUT` | Streaming timeout (seconds) | `300` |
| `STREAM_SCANNER_MAX_BUFFER_MB` | Max per-line buffer (MB) for the stream scanner; increase when upstream sends huge image/base64 payloads | `64` |
| `MAX_REQUEST_BODY_MB` | Max request body size (MB, counted **after decompression**; prevents huge requests/zip bombs from exhausting memory). Exceeding it returns `413` | `32` |
| `AZURE_DEFAULT_API_VERSION` | Azure API version | `2025-04-01-preview` |
| `ERROR_LOG_ENABLED` | Error log switch | `false` |
| `PYROSCOPE_URL` | Pyroscope server address | - |
| `PYROSCOPE_APP_NAME` | Pyroscope application name | `token-factory` |
| `PYROSCOPE_BASIC_AUTH_USER` | Pyroscope basic auth user | - |
| `PYROSCOPE_BASIC_AUTH_PASSWORD` | Pyroscope basic auth password | - |
| `PYROSCOPE_MUTEX_RATE` | Pyroscope mutex sampling rate | `5` |
| `PYROSCOPE_BLOCK_RATE` | Pyroscope block sampling rate | `5` |
| `HOSTNAME` | Hostname tag for Pyroscope | `token-factory` |

📖 **Complete configuration:** [Environment Variables Documentation](https://docs.newapi.pro/en/docs/installation/config-maintenance/environment-variables)

</details>

### 🔧 Deployment Methods

**Docker Compose:** use the [Quick Start](#quick-start) commands above (clone → edit `docker-compose.yml` → `docker-compose up -d`).

<details>
<summary><strong>Alternative: plain Docker run</strong></summary>

**Using SQLite:**
```bash
docker run --name token-factory -d --restart always \
  -p 3000:3000 \
  -e TZ=Asia/Shanghai \
  -v ./data:/data \
  ghcr.io/fyinfor/token-factory:latest
```

**Using MySQL:**
```bash
docker run --name token-factory -d --restart always \
  -p 3000:3000 \
  -e SQL_DSN="root:123456@tcp(localhost:3306)/oneapi" \
  -e TZ=Asia/Shanghai \
  -v ./data:/data \
  ghcr.io/fyinfor/token-factory:latest
```

> **💡 Path explanation:**
> - `./data:/data` - Relative path, data saved in the data folder of the current directory
> - You can also use absolute path, e.g.: `/your/custom/path:/data`

</details>

<details>
<summary><strong>BaoTa Panel</strong></summary>

1. Install BaoTa Panel (≥ 9.2.0 version)
2. Search for **TokenFactory** in the application store
3. One-click installation

📖 [Tutorial with images](./docs/BT.md)

</details>

### ⚠️ Multi-machine Deployment Considerations

> [!WARNING]
> - **Must set** `SESSION_SECRET` - Otherwise login status inconsistent
> - **Shared Redis must set** `CRYPTO_SECRET` - Otherwise data cannot be decrypted

### 🔄 Channel Retry and Cache

**Retry configuration:** `Settings → Operation Settings → General Settings → Failure Retry Count`

**Cache configuration:**
- `REDIS_CONN_STRING`: Redis cache (recommended)
- `MEMORY_CACHE_ENABLED`: Memory cache

---

## Lineage

| Repository | Role |
| --- | --- |
| **[QuantumNous/new-api](https://github.com/QuantumNous/new-api)** | **Upstream** — New API (AGPL-3.0). **Start here** for history, issues that are not fork-specific, and feature design. |
| [One API](https://github.com/songquanpeng/one-api) | Earlier MIT-licensed codebase in the same family tree. |
| [Midjourney-Proxy](https://github.com/novicezk/midjourney-proxy) | Optional Midjourney integration (see upstream docs). |

Tools maintained around the ecosystem (e.g. [neko-api-key-tool](https://github.com/Calcium-Ion/neko-api-key-tool)) are documented upstream.

---

## Help

### 📖 Documentation Resources

| Resource | Link |
|------|------|
| 📘 FAQ | [FAQ](https://docs.newapi.pro/en/docs/support/faq) |
| 💬 Community Interaction | [Communication Channels](https://docs.newapi.pro/en/docs/support/community-interaction) |
| 🐛 Issue Feedback | [Issue Feedback](https://docs.newapi.pro/en/docs/support/feedback-issues) |
| 📚 Complete Documentation | [Official Documentation](https://docs.newapi.pro/en/docs) |

### 🤝 Contribution Guide

Welcome all forms of contribution!

- 🐛 Report Bugs
- 💡 Propose New Features
- 📝 Improve Documentation
- 🔧 Submit Code

---

## License

This project (**TokenFactory**) is licensed under the [GNU Affero General Public License v3.0 (AGPLv3)](./LICENSE). Modifications and further derivatives remain under **AGPL-3.0** unless you obtain a separate commercial license from the copyright holders.

**Attribution:** TokenFactory is derived from [QuantumNous/new-api](https://github.com/QuantumNous/new-api) (New API), which is also under AGPL-3.0. The project chain includes [One API](https://github.com/songquanpeng/one-api) (MIT License) as an earlier base. Please retain upstream notices and this repository’s [`LICENSE`](./LICENSE) and [`NOTICE`](./NOTICE). Under **AGPL-3.0 section 13**, if you run a modified version as a network service for others, you must offer them the corresponding complete source code under the same license.

If your organization's policies do not permit the use of AGPLv3-licensed software, or if you wish to avoid the open-source obligations of AGPLv3, please contact us at: [support@quantumnous.com](mailto:support@quantumnous.com)

---

<div align="center">

**TokenFactory** — self-hosted AI gateway (this fork).

**Upstream:** **[QuantumNous/new-api](https://github.com/QuantumNous/new-api)** · **Docs:** [docs.newapi.pro](https://docs.newapi.pro/en/docs) · **This repo:** [issues](https://github.com/QuantumNous/token-factory/issues)

<sub>The New API project is developed by **QuantumNous** and contributors. JetBrains supports open-source development through free IDE licenses.</sub>

</div>
