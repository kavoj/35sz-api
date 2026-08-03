<div align="center">

![new-api](/web/default/public/logo.png)

# New API

🍥 **新一代大模型网关与AI资产管理系统**

<p align="center">
  简体中文 |
  <a href="./README.zh_TW.md">繁體中文</a> |
  <a href="./README.md">English</a> |
  <a href="./README.fr.md">Français</a> |
  <a href="./README.ja.md">日本語</a>
</p>

<p align="center">
  <a href="https://raw.githubusercontent.com/Calcium-Ion/new-api/main/LICENSE">
    <img src="https://img.shields.io/github/license/Calcium-Ion/new-api?color=brightgreen" alt="license">
  </a><!--
  --><a href="https://github.com/Calcium-Ion/new-api/releases/latest">
    <img src="https://img.shields.io/github/v/release/Calcium-Ion/new-api?color=brightgreen&include_prereleases" alt="release">
  </a><!--
  --><a href="https://hub.docker.com/r/CalciumIon/new-api">
    <img src="https://img.shields.io/badge/docker-dockerHub-blue" alt="docker">
  </a><!--
  --><a href="https://goreportcard.com/report/github.com/Calcium-Ion/new-api">
    <img src="https://goreportcard.com/badge/github.com/Calcium-Ion/new-api" alt="GoReportCard">
  </a>
</p>

<p align="center">
  <a href="https://trendshift.io/repositories/20180" target="_blank">
    <img src="https://trendshift.io/api/badge/repositories/20180" alt="QuantumNous%2Fnew-api | Trendshift" style="width: 250px; height: 55px;" width="250" height="55"/>
  </a>
  <br>
  <a href="https://hellogithub.com/repository/QuantumNous/new-api" target="_blank">
    <img src="https://api.hellogithub.com/v1/widgets/recommend.svg?rid=539ac4217e69431684ad4a0bab768811&claim_uid=tbFPfKIDHpc4TzR" alt="Featured｜HelloGitHub" style="width: 250px; height: 54px;" width="250" height="54" />
  </a><!--
  --><a href="https://www.producthunt.com/products/new-api/launches/new-api?embed=true&utm_source=badge-featured&utm_medium=badge&utm_campaign=badge-new-api" target="_blank" rel="noopener noreferrer">
    <img src="https://api.producthunt.com/widgets/embed-image/v1/featured.svg?post_id=1047693&theme=light&t=1769577875005" alt="New API - All-in-one AI asset management gateway. | Product Hunt" style="width: 250px; height: 54px;" width="250" height="54" />
  </a>
</p>

<p align="center">
  <a href="#-快速开始">快速开始</a> •
  <a href="#-主要特性">主要特性</a> •
  <a href="#-部署">部署</a> •
  <a href="#-文档">文档</a> •
  <a href="#-平台亮点与选型对比">平台亮点</a> •
  <a href="#-帮助支持">帮助</a>
</p>

</div>

## 📝 项目说明

New API 是面向合法授权场景的 AI API 网关与 AI 资产管理平台：将多家上游模型统一到一个入口，集中处理渠道、模型、令牌、路由、价格、额度、日志与运维。它适合企业私有化部署、组织内部 AI 服务、受控的客户 API 服务和多供应商成本管理；平台本身不提供上游模型授权，渠道密钥和模型权限仍由使用者向上游合法申请。

### 一条配置主线

```text
创建渠道 → 添加/同步模型 → 设置价格 → 创建用户/令牌 → 调用模型 → 查看日志与成本
```

- [渠道创建与配置](./docs/guide/channel-setup.zh_CN.md)
- [模型添加与管理](./docs/guide/model-management.zh_CN.md)
- [模型调用](./docs/guide/model-invocation.zh_CN.md)
- [价格与计费](./docs/guide/pricing-and-billing.zh_CN.md)
- [部署硬件与运行环境](./docs/deployment-requirements.zh_CN.md)

> [!IMPORTANT]
> - 本项目仅面向合法授权的 AI API 网关、组织内部鉴权、多模型管理、用量统计、成本核算和私有化部署场景。
> - 使用者必须合法取得上游 API Key、账号、模型服务或接口权限，并遵守上游服务条款及适用法律法规。
> - 使用者应确保其使用方式符合上游服务条款及适用法律法规。
> - 面向公众提供生成式人工智能服务时，使用者应遵守[《生成式人工智能服务管理暂行办法》](http://www.cac.gov.cn/2023-07/13/c_1690898327029107.htm)等监管要求，自行完成所在司法辖区要求的备案、许可、内容安全、实名、日志留存、税务和上游授权等合规义务。

---

## 🤝 我们信任的合作伙伴

<p align="center">
  <em>排名不分先后</em>
</p>

<p align="center">
  <a href="https://www.cherry-ai.com/" target="_blank">
    <img src="./docs/images/cherry-studio.png" alt="Cherry Studio" height="80" />
  </a><!--
  --><a href="https://github.com/iOfficeAI/AionUi/" target="_blank">
    <img src="./docs/images/aionui.png" alt="Aion UI" height="80" />
  </a><!--
  --><a href="https://bda.pku.edu.cn/" target="_blank">
    <img src="./docs/images/pku.png" alt="北京大学" height="80" />
  </a><!--
  --><a href="https://www.compshare.cn/?ytag=GPU_yy_gh_newapi" target="_blank">
    <img src="./docs/images/ucloud.png" alt="UCloud 优刻得" height="80" />
  </a><!--
  --><a href="https://www.aliyun.com/" target="_blank">
    <img src="./docs/images/aliyun.png" alt="阿里云" height="80" />
  </a><!--
  --><a href="https://io.net/" target="_blank">
    <img src="./docs/images/io-net.png" alt="IO.NET" height="80" />
  </a>
</p>

---

## 🙏 特别鸣谢

<p align="center">
  <a href="https://www.jetbrains.com/?from=new-api" target="_blank">
    <img src="https://resources.jetbrains.com/storage/products/company/brand/logos/jb_beam.png" alt="JetBrains Logo" width="120" />
  </a>
</p>

<p align="center">
  <strong>感谢 <a href="https://www.jetbrains.com/?from=new-api">JetBrains</a> 为本项目提供免费的开源开发许可证</strong>
</p>

---

## 🚀 快速开始

### 使用 Docker Compose（推荐）

```bash
# 克隆项目
git clone https://github.com/QuantumNous/new-api.git
cd new-api

# 编辑 docker-compose.yml 配置
nano docker-compose.yml

# 启动服务
docker-compose up -d
```

<details>
<summary><strong>使用 Docker 命令</strong></summary>

```bash
# 拉取最新镜像
docker pull calciumion/new-api:latest

# 使用 SQLite（默认）
docker run --name new-api -d --restart always \
  -p 3000:3000 \
  -e TZ=Asia/Shanghai \
  -v ./data:/data \
  calciumion/new-api:latest

# 使用 MySQL
docker run --name new-api -d --restart always \
  -p 3000:3000 \
  -e SQL_DSN="root:123456@tcp(localhost:3306)/oneapi" \
  -e TZ=Asia/Shanghai \
  -v ./data:/data \
  calciumion/new-api:latest
```

> **💡 提示：** `-v ./data:/data` 会将数据保存在当前目录的 `data` 文件夹中，你也可以改为绝对路径如 `-v /your/custom/path:/data`

</details>

---

🎉 部署完成后，访问 `http://localhost:3000` 即可使用！

> [!WARNING]
> 将本项目作为面向公众的生成式 AI 服务或 API 转售服务运营时，使用者应先完成备案、内容安全、实名、日志留存、税务、支付和上游授权等合规义务。

📖 更多部署方式请参考 [部署指南](https://docs.newapi.pro/zh/docs/installation)

---

## 📚 文档

<div align="center">

### 📖 [官方文档](https://docs.newapi.pro/zh/docs) | [![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/QuantumNous/new-api)

</div>

**快速导航：**

| 分类 | 链接 |
|------|------|
| 🚀 部署指南 | [安装文档](https://docs.newapi.pro/zh/docs/installation) |
| ⚙️ 环境配置 | [环境变量](https://docs.newapi.pro/zh/docs/installation/config-maintenance/environment-variables) |
| 📡 接口文档 | [API 文档](https://docs.newapi.pro/zh/docs/api) |
| 🧭 渠道配置 | [渠道创建与配置](./docs/guide/channel-setup.zh_CN.md) |
| 🧩 模型管理 | [模型添加与管理](./docs/guide/model-management.zh_CN.md) |
| 🧪 模型调用 | [模型调用指南](./docs/guide/model-invocation.zh_CN.md) |
| 💰 价格计费 | [价格与计费指南](./docs/guide/pricing-and-billing.zh_CN.md) |
| 🖥️ 部署要求 | [硬件与运行环境](./docs/deployment-requirements.zh_CN.md) |
| ❓ 常见问题 | [FAQ](https://docs.newapi.pro/zh/docs/support/faq) |
| 💬 社区交流 | [交流渠道](https://docs.newapi.pro/zh/docs/support/community-interaction) |

---

## ✨ 主要特性

> 详细特性请参考 [特性说明](https://docs.newapi.pro/zh/docs/guide/wiki/basic-concepts/features-introduction)

### 🎨 核心功能

| 特性 | 说明 |
|------|------|
| 🎨 全新 UI | 现代化的用户界面设计 |
| 🌍 多语言 | 支持中文、英文、法语、日语 |
| 🔄 数据兼容 | 完全兼容原版 One API 数据库 |
| 📈 数据看板 | 可视化控制台与统计分析 |
| 🔒 权限管理 | 令牌分组、模型限制、用户管理 |

### 💰 授权用量与成本管理

- ✅ 合法授权场景下的内部充值与额度分配（易支付、Stripe）
- ✅ 组织内按次、按量或缓存命中成本核算
- ✅ 支持 OpenAI、Azure、DeepSeek、Claude、Qwen 等模型的缓存计费统计
- ✅ 面向内部管理或企业客户的灵活计费策略配置

### 🔐 授权与安全

- 😈 Discord 授权登录
- 🤖 LinuxDO 授权登录
- 📱 Telegram 授权登录
- 🔑 OIDC 统一认证
- 🔍 Key 查询使用额度（配合 [new-api-key-tool](https://github.com/Calcium-Ion/new-api-key-tool)）

### 🚀 高级功能

**API 格式支持：**
- ⚡ [OpenAI Responses](https://docs.newapi.pro/zh/docs/api/ai-model/chat/openai/create-response)
- ⚡ [OpenAI Realtime API](https://docs.newapi.pro/zh/docs/api/ai-model/realtime/create-realtime-session)（含 Azure）
- ⚡ [Claude Messages](https://docs.newapi.pro/zh/docs/api/ai-model/chat/create-message)
- ⚡ [Google Gemini](https://doc.newapi.pro/api/google-gemini-chat)
- 🔄 [Rerank 模型](https://docs.newapi.pro/zh/docs/api/ai-model/rerank/create-rerank)（Cohere、Jina）

**智能路由：**
- ⚖️ 渠道加权随机
- 🔄 失败自动重试
- 🚦 用户级别模型限流

**格式转换：**
- 🔄 **OpenAI Compatible ⇄ Claude Messages**
- 🔄 **OpenAI Compatible → Google Gemini**
- 🔄 **Google Gemini → OpenAI Compatible** - 仅支持文本，暂不支持函数调用
- 🚧 **OpenAI Compatible ⇄ OpenAI Responses** - 开发中
- 🔄 **思考转内容功能**

**Reasoning Effort 支持：**

<details>
<summary>查看详细配置</summary>

**OpenAI 系列模型：**
- `o3-mini-high` - High reasoning effort
- `o3-mini-medium` - Medium reasoning effort
- `o3-mini-low` - Low reasoning effort
- `gpt-5-high` - High reasoning effort
- `gpt-5-medium` - Medium reasoning effort
- `gpt-5-low` - Low reasoning effort

**Claude 思考模型：**
- `claude-3-7-sonnet-20250219-thinking` - 启用思考模式

**Google Gemini 系列模型：**
- `gemini-2.5-flash-thinking` - 启用思考模式
- `gemini-2.5-flash-nothinking` - 禁用思考模式
- `gemini-2.5-pro-thinking` - 启用思考模式
- `gemini-2.5-pro-thinking-128` - 启用思考模式，并设置思考预算为128tokens
- 也可以直接在 Gemini 模型名称后追加 `-low` / `-medium` / `-high` 来控制思考力度（无需再设置思考预算后缀）

</details>

---

## 🌟 平台亮点与选型对比

### 功能亮点

| 方向 | 平台能力 | 对企业/团队的价值 |
|---|---|---|
| <span style="color:#16a34a">● 接入聚合</span> | 统一接入 OpenAI、Claude、Gemini、Azure、AWS Bedrock、火山方舟、千问等多类上游，并支持自定义合法授权渠道 | 减少供应商切换成本，避免业务系统绑定单一上游 |
| <span style="color:#16a34a">● 路由容灾</span> | 渠道分组、优先级、权重、失败重试、自动禁用和模型映射 | 将成本、容量和可用性策略集中管理 |
| <span style="color:#16a34a">● 统一计费</span> | 按模型、用户/令牌、分组和能力类型核算；支持聊天、Embedding、图片、视频、音频等结构化价格 | 让内部成本分摊和客户计费使用同一套网关口径 |
| <span style="color:#16a34a">● 协议转换</span> | OpenAI Compatible、Responses、Claude Messages、Gemini、Realtime、音频、图片、视频和 Rerank 等入口 | 客户端改造更少，供应商协议差异由适配器隔离 |
| <span style="color:#16a34a">● 私有化管理</span> | 用户、令牌、分组、权限、日志、充值/额度、渠道和模型集中管理 | 数据、密钥和运营规则可留在组织自己的基础设施中 |
| <span style="color:#2563eb">● 可观测</span> | 使用日志、任务日志、渠道测试、响应耗时、用量和扣费记录 | 便于定位上游故障、价格异常和调用链路问题 |
| <span style="color:#d97706">● 依赖配置</span> | 实际模型能力、价格、限流、音色、配额和 SLA 取决于上游授权与套餐 | 平台负责统一治理，不替代供应商权限和服务承诺 |

### 与 New API、OpenRouter 的定位对比

> 说明：New API 是本项目的上游项目基线；OpenRouter 的产品能力、套餐和模型目录会随版本变化。表格比较的是典型产品定位，不构成第三方 SLA、价格或模型数量承诺。

图例：<span style="color:#16a34a">● 原生优势</span>　<span style="color:#2563eb">● 兼容/共同能力</span>　<span style="color:#d97706">● 需配置或依赖上游</span>　<span style="color:#6b7280">— 非平台定位/不适用</span>

| 能力 | 本平台 | New API（上游项目基线） | OpenRouter | 适用价值/差异说明 |
|---|---|---|---|---|
| 私有化部署 | <span style="color:#16a34a">● 原生支持</span> | <span style="color:#2563eb">● 支持</span> | <span style="color:#6b7280">— 主要是托管服务</span> | 对数据、密钥、网络和审计有自主控制 |
| 多上游渠道与路由 | <span style="color:#16a34a">● 渠道/分组/权重/优先级/重试</span> | <span style="color:#2563eb">● 基础能力</span> | <span style="color:#2563eb">● 托管聚合</span> | 本平台更适合组织自有渠道治理 |
| 渠道级 API Key、模型映射 | <span style="color:#16a34a">● 可配置</span> | <span style="color:#2563eb">● 可配置</span> | <span style="color:#d97706">● 受托管平台规则约束</span> | 适合拆分供应商、区域和环境 |
| 用户、令牌、分组和权限 | <span style="color:#16a34a">● 平台内置</span> | <span style="color:#2563eb">● 平台内置</span> | <span style="color:#d97706">● 依赖账户/团队套餐</span> | 适合企业内部多团队隔离 |
| 按模型/用户/分组计费 | <span style="color:#16a34a">● 可配置</span> | <span style="color:#2563eb">● 可配置</span> | <span style="color:#d97706">● 以托管账单规则为准</span> | 支持内部成本中心和客户套餐 |
| 图片/视频/音频结构化价格 | <span style="color:#16a34a">● 支持配置</span> | <span style="color:#2563eb">● 取版本能力</span> | <span style="color:#d97706">● 以托管模型价格为准</span> | 适合非 Token 模型的单位换算 |
| 数据库与日志自主管理 | <span style="color:#16a34a">● SQLite/MySQL/PostgreSQL，可扩展 Redis/日志库</span> | <span style="color:#2563eb">● 支持</span> | <span style="color:#d97706">● 由服务方管理</span> | 适合合规、备份和长期审计 |
| OpenAI/Claude/Gemini/Responses 等协议 | <span style="color:#16a34a">● 多协议适配</span> | <span style="color:#2563eb">● 继承并持续扩展</span> | <span style="color:#2563eb">● 提供统一入口</span> | 具体模型能力仍取决于上游 |
| 免运维托管与全球接入 | <span style="color:#6b7280">— 需自行部署</span> | <span style="color:#6b7280">— 需自行部署</span> | <span style="color:#16a34a">● 托管优势</span> | OpenRouter 更适合快速接入和减少基础设施运维 |
| 上游授权、模型配额和 SLA | <span style="color:#d97706">● 由使用者负责</span> | <span style="color:#d97706">● 由使用者负责</span> | <span style="color:#d97706">● 受托管供应关系约束</span> | 任何平台都不能替代上游授权 |

## 🤖 模型支持

> 详情请参考 [接口文档 - 网关接口](https://docs.newapi.pro/zh/docs/api)

| 模型类型 | 说明 | 文档 |
|---------|------|------|
| 🤖 OpenAI-Compatible | OpenAI 兼容模型 | [文档](https://docs.newapi.pro/zh/docs/api/ai-model/chat/openai/createchatcompletion) |
| 🤖 OpenAI Responses | OpenAI Responses 格式 | [文档](https://docs.newapi.pro/zh/docs/api/ai-model/chat/openai/createresponse) |
| 🎨 Midjourney-Proxy | [Midjourney-Proxy(Plus)](https://github.com/novicezk/midjourney-proxy) | [文档](https://doc.newapi.pro/api/midjourney-proxy-image) |
| 🎵 Suno-API | [Suno API](https://github.com/Suno-API/Suno-API) | [文档](https://doc.newapi.pro/api/suno-music) |
| 🔄 Rerank | Cohere、Jina | [文档](https://docs.newapi.pro/zh/docs/api/ai-model/rerank/create-rerank) |
| 💬 Claude | Messages 格式 | [文档](https://docs.newapi.pro/zh/docs/api/ai-model/chat/createmessage) |
| 🌐 Gemini | Google Gemini 格式 | [文档](https://docs.newapi.pro/zh/docs/api/ai-model/chat/gemini/geminirelayv1beta) |
| 🔧 Dify | ChatFlow 模式 | - |
| 🎯 自定义上游 | 支持配置合法授权的上游接口地址 | - |

### 📡 支持的接口

<details>
<summary>查看完整接口列表</summary>

- [聊天接口 (Chat Completions)](https://docs.newapi.pro/zh/docs/api/ai-model/chat/openai/createchatcompletion)
- [响应接口 (Responses)](https://docs.newapi.pro/zh/docs/api/ai-model/chat/openai/createresponse)
- [图像接口 (Image)](https://docs.newapi.pro/zh/docs/api/ai-model/images/openai/post-v1-images-generations)
- [音频接口 (Audio)](https://docs.newapi.pro/zh/docs/api/ai-model/audio/openai/create-transcription)
- [视频接口 (Video)](https://docs.newapi.pro/zh/docs/api/ai-model/video/openai/create-video)
- [嵌入接口 (Embeddings)](https://docs.newapi.pro/zh/docs/api/ai-model/embeddings/createembedding)
- [重排序接口 (Rerank)](https://docs.newapi.pro/zh/docs/api/ai-model/rerank/creatererank)
- [实时对话 (Realtime)](https://docs.newapi.pro/zh/docs/api/ai-model/realtime/createrealtimesession)
- [Claude 聊天](https://docs.newapi.pro/zh/docs/api/ai-model/chat/createmessage)
- [Google Gemini 聊天](https://docs.newapi.pro/zh/docs/api/ai-model/chat/gemini/geminirelayv1beta)

</details>

---

## 🚢 部署

> [!TIP]
> **最新版 Docker 镜像：** `calciumion/new-api:latest`

### 📋 部署要求与硬件建议

基础要求：64 位系统、Docker/Docker Compose、持久化 `/data`、可访问上游的网络；数据库支持 SQLite、MySQL ≥ 5.7.8、PostgreSQL ≥ 9.6。生产环境建议使用独立数据库和 Redis。

| 场景 | 网关计算 | 内存 | SSD | 说明 |
| --- | ---: | ---: | ---: | --- |
| 开发验证 | 1–2 vCPU | 2–4 GB | 20 GB | 低并发、SQLite、短期日志 |
| 小规模生产 | 2–4 vCPU | 4–8 GB | 50–100 GB | 小团队、独立备份 |
| 中规模生产 | 4–8 vCPU | 8–16 GB | 100 GB+ | 多用户、多渠道、较长日志 |

以上是网关本身的容量建议，不包含本地模型推理服务。并发、流式连接、图片/视频响应大小、日志留存、数据库和 Redis 数据量会改变实际需求。详见[部署硬件与运行环境](./docs/deployment-requirements.zh_CN.md)。

| 组件 | 要求 |
| ------ | ------ |
| **本地数据库** | SQLite（Docker 需挂载 `/data` 目录） |
| **远程数据库** | MySQL ≥ 5.7.8 或 PostgreSQL ≥ 9.6 |
| **容器引擎** | Docker / Docker Compose |
| **系统架构** | 仅支持 64 位系统（amd64 / arm64），不支持 32 位系统 |

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
| `PYROSCOPE_APP_NAME` | Pyroscope 应用名                                        | `new-api` |
| `PYROSCOPE_BASIC_AUTH_USER` | Pyroscope Basic Auth 用户名                        | - |
| `PYROSCOPE_BASIC_AUTH_PASSWORD` | Pyroscope Basic Auth 密码                  | - |
| `PYROSCOPE_MUTEX_RATE` | Pyroscope mutex 采样率                               | `5` |
| `PYROSCOPE_BLOCK_RATE` | Pyroscope block 采样率                               | `5` |
| `HOSTNAME` | Pyroscope 标签里的主机名                                          | `new-api` |

📖 **完整配置：** [环境变量文档](https://docs.newapi.pro/zh/docs/installation/config-maintenance/environment-variables)

</details>

### 🔧 部署方式

<details>
<summary><strong>方式 1：Docker Compose（推荐）</strong></summary>

```bash
# 克隆项目
git clone https://github.com/QuantumNous/new-api.git
cd new-api

# 编辑配置
nano docker-compose.yml

# 启动服务
docker-compose up -d
```

</details>

<details>
<summary><strong>方式 2：Docker 命令</strong></summary>

**使用 SQLite：**
```bash
docker run --name new-api -d --restart always \
  -p 3000:3000 \
  -e TZ=Asia/Shanghai \
  -v ./data:/data \
  calciumion/new-api:latest
```

**使用 MySQL：**
```bash
docker run --name new-api -d --restart always \
  -p 3000:3000 \
  -e SQL_DSN="root:123456@tcp(localhost:3306)/oneapi" \
  -e TZ=Asia/Shanghai \
  -v ./data:/data \
  calciumion/new-api:latest
```

> **💡 路径说明：**
> - `./data:/data` - 相对路径，数据保存在当前目录的 data 文件夹
> - 也可使用绝对路径，如：`/your/custom/path:/data`

</details>

<details>
<summary><strong>方式 3：宝塔面板</strong></summary>

1. 安装宝塔面板（≥ 9.2.0 版本）
2. 在应用商店搜索 **New-API**
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

## 🔗 相关项目

### 上游项目

| 项目 | 说明 |
|------|------|
| [One API](https://github.com/songquanpeng/one-api) | 原版项目基础 |
| [Midjourney-Proxy](https://github.com/novicezk/midjourney-proxy) | Midjourney 接口支持 |

### 配套工具

| 项目 | 说明 |
|------|------|
| [new-api-key-tool](https://github.com/Calcium-Ion/new-api-key-tool) | Key 额度查询工具 |
| [new-api-horizon](https://github.com/Calcium-Ion/new-api-horizon) | New API 高性能优化版 |

---

## 💬 帮助支持

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

## 📜 许可证

本项目采用 [GNU Affero 通用公共许可证 v3.0 (AGPLv3)](./LICENSE) 授权。

本项目为开源项目，在 [One API](https://github.com/songquanpeng/one-api)（MIT 许可证）的基础上进行二次开发。

如果您所在的组织政策不允许使用 AGPLv3 许可的软件，或您希望规避 AGPLv3 的开源义务，请发送邮件至：[support@quantumnous.com](mailto:support@quantumnous.com)

---

## 🌟 Star History

<div align="center">

[![Star History Chart](https://api.star-history.com/svg?repos=Calcium-Ion/new-api&type=Date)](https://star-history.com/#Calcium-Ion/new-api&Date)

</div>

---

<div align="center">

### 💖 感谢使用 New API

如果这个项目对你有帮助，欢迎给我们一个 ⭐️ Star！

**[官方文档](https://docs.newapi.pro/zh/docs)** • **[问题反馈](https://github.com/Calcium-Ion/new-api/issues)** • **[最新发布](https://github.com/Calcium-Ion/new-api/releases)**

<sub>Built with ❤️ by QuantumNous</sub>

</div>
