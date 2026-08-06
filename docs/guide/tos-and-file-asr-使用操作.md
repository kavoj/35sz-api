# 火山 TOS 与录音文件识别使用操作手册

> 适用范围：New API 当前版本中新增的火山对象存储（TOS）临时音频中转和火山录音文件识别功能。
>
> 本文只使用占位符和公开的示例配置，不包含真实 Access Key、Secret Key、平台令牌或预签名 URL。

## 1. 功能概述

录音文件识别模式用于将智能体平台上传的音频文件提交给火山录音文件识别 HTTP 接口，并轮询获取识别结果。

完整链路如下：

```text
智能体平台
    │ POST /v1/audio/transcriptions
    ▼
New API
    │ 上传临时音频
    ▼
火山 TOS 私有 Bucket
    │ 生成短期预签名 GET URL
    ▼
火山 OpenSpeech submit/query
    │ 轮询识别结果
    ▼
New API 返回统一转写结果
    │ 删除临时对象
    ▼
TOS 临时音频对象清理
```

TOS 只作为火山服务端读取音频的临时中转，不作为业务音频长期归档存储。默认推荐使用私有 Bucket 和短期预签名 URL。

## 2. 前置条件

开始配置前，确认以下条件均已满足：

1. 已在火山引擎创建 TOS Bucket。
2. TOS Bucket 所在地域与 Region 一致。
3. 服务端拥有 TOS 的上传、读取签名和删除临时对象权限。
4. 已在火山控制台开通录音文件识别对应能力，并确认账号、模型和 Resource-Id 有权限使用。
5. New API 服务端可以访问：
   - TOS Endpoint；
   - 火山 OpenSpeech 录音文件识别接口；
   - 火山控制台配置的网络出口或代理。
6. 智能体平台持有的是 New API 平台令牌，而不是 TOS Secret Key 或火山上游密钥。

## 3. 创建和准备 TOS

### 3.1 推荐配置方式

以广州地域、Bucket 名称 `35sz` 为例：

| 配置项 | 填写值 |
|---|---|
| TOS 原生 Endpoint | `https://tos-cn-guangzhou.volces.com` |
| Region | `cn-guangzhou` |
| Bucket | `35sz` |
| 对象前缀 | `recording-asr` |
| 预签名 URL 有效期 | `900` 秒 |
| Public Base URL | 留空 |

Bucket 名称字段只填写 `35sz`，不要填写完整域名。

### 3.2 三种地址的区别

火山控制台可能同时显示以下地址，它们用途不同：

```text
TOS 原生 Endpoint:
https://tos-cn-guangzhou.volces.com

S3 Endpoint:
https://tos-s3-cn-guangzhou.volces.com

Bucket 域名:
https://35sz.tos-cn-guangzhou.volces.com
```

当前 New API 的 TOS 实现使用火山 TOS 原生 SDK，因此配置时：

- `Endpoint` 填 TOS 原生 Endpoint；
- `Region` 填 `cn-guangzhou`；
- `Bucket` 只填 `35sz`；
- S3 Endpoint 不填入 Endpoint；
- Bucket 域名不填入 Bucket 字段；
- 不要把 TOS Endpoint 填到火山 ASR 渠道的 Base URL。

### 3.3 权限建议

建议为 New API 服务端单独创建访问身份，仅授予目标 Bucket 的最小权限：

- 允许上传临时音频对象；
- 允许读取用于生成或使用预签名 URL 的对象；
- 允许删除临时音频对象；
- 允许执行 Bucket 连通性检查所需的操作。

不要为此功能授予不相关 Bucket、账号管理、密钥管理或永久公开读权限。实际权限名称和授权方式以火山 TOS 控制台当前版本为准。

## 4. 在 New API 后台配置 TOS

进入管理端：

```text
系统设置 → 运维设置 → 性能
```

在页面中找到 **Volcengine TOS Object Storage** 区域，填写：

| 页面字段 | 示例 | 说明 |
|---|---|---|
| Enable Volcengine TOS | 开启 | 文件模式必须开启 |
| TOS Endpoint | `https://tos-cn-guangzhou.volces.com` | 必须使用 HTTPS |
| TOS Region | `cn-guangzhou` | 与 Bucket 地域一致 |
| TOS Bucket | `35sz` | 只填 Bucket 名称 |
| TOS Access Key | `<TOS_ACCESS_KEY>` | 仅保存于服务端配置 |
| TOS Secret Key | `<TOS_SECRET_KEY>` | 页面按密码字段处理 |
| TOS Object Prefix | `recording-asr` | 临时对象的逻辑前缀 |
| Signed URL Expiry | `900` | 取值范围为 1 至 604800 秒 |
| TOS Public Base URL | 留空 | 使用私有桶预签名 URL |

填写后点击页面顶部的保存按钮，再点击 **Test TOS connection**。

测试连接只执行配置和 Bucket 检查，不会上传智能体平台的用户音频。

### 4.1 Secret Key 处理规则

- 页面不会回显服务端已有的 Secret Key。
- 修改其他字段时，Secret Key 留空表示保留原值。
- 只有明确使用清空操作时，才会删除原 Secret Key。
- 不要将 Secret Key 复制到浏览器控制台、工单、截图、日志或 Git 仓库。
- 如果怀疑密钥泄露，应在火山控制台轮换密钥，并同步更新后台配置。

## 5. 创建火山录音识别渠道

进入管理端：

```text
渠道 → 创建渠道
```

创建一个火山方舟/VolcEngine 渠道，基本字段按上游实际开通情况填写：

| 字段 | 配置建议 |
|---|---|
| 渠道类型 | 选择火山方舟或 VolcEngine 对应渠道类型 |
| Base URL | 填火山 Ark/Agent Plan 的渠道地址，不要填 TOS Endpoint |
| API Key | 填火山上游凭证，不要填 TOS Access Key |
| 模型列表 | 填控制台实际开通的录音识别模型名称 |
| 分组 | 与智能体平台令牌可用分组保持一致 |
| 权重/优先级 | 按生产路由策略设置 |

### 5.1 凭证格式

当前适配器支持两类火山认证方式：

#### 新版 Ark API Key

通常以 `ark-` 开头，直接填写：

```text
ark-<VOLCENGINE_API_KEY>
```

适配器会使用 `X-Api-Key` 请求头。

#### 旧版语音认证

如果上游明确要求旧版 App ID 和 Access Token，可按以下格式填写：

```text
<VOLCENGINE_APP_ID>|<VOLCENGINE_ACCESS_TOKEN>
```

适配器会拆分为：

```http
X-Api-App-Key: <VOLCENGINE_APP_ID>
X-Api-Access-Key: <VOLCENGINE_ACCESS_TOKEN>
```

不要在同一个渠道中混用两种认证格式。以火山控制台当前显示的认证方式为准。

### 5.2 Resource-Id 和模型

BigASR 标准录音文件识别使用：

```text
volc.bigasr.auc
```

当前代码会根据模型名称选择 BigASR Resource-Id；其他 Seed/Agent Plan 录音文件识别能力的 Resource-Id，必须以火山控制台和对应官方文档中实际开通的值为准。不要仅根据模型名称自行推断或填写未开通的 Resource-Id。

如果出现 Resource-Id 不匹配、模型不存在或 403，应先在火山控制台确认：

- 模型是否已开通；
- 当前账号是否有调用权限；
- 录音文件识别能力是否与该模型匹配；
- 渠道使用的 API Key 是否属于同一账号或项目；
- 控制台给出的 Resource-Id 是否需要更新。

## 6. 调用录音文件识别接口

### 6.1 文件模式入口

智能体平台调用 New API 的统一接口：

```text
POST /v1/audio/transcriptions
```

必须使用 `multipart/form-data`，并提交 `metadata` 指定文件识别模式：

```json
{"mode":"file"}
```

未指定 `metadata.mode=file` 时，火山渠道默认保留实时 WebSocket ASR 路径，不会进入 submit/query 文件识别流程。

### 6.2 cURL 示例

下面的示例只使用占位符：

```bash
curl --fail-with-body "https://<NEW_API_DOMAIN>/v1/audio/transcriptions" \
  -H "Authorization: Bearer <NEW_API_TOKEN>" \
  -F "file=@./sample.mp3" \
  -F "model=<CONFIGURED_ASR_MODEL>" \
  -F 'metadata={"mode":"file"}' \
  -F "response_format=verbose_json"
```

不要把以下内容放入客户端请求：

- TOS Access Key；
- TOS Secret Key；
- 火山上游 API Key；
- TOS 预签名 URL。

这些凭证和中转逻辑由 New API 服务端处理。

### 6.3 响应格式

#### 默认或 text

适合只需要转写文本的调用方：

```json
{
  "text": "这里是录音文件的识别文本"
}
```

#### json / verbose_json

`json` 或 `verbose_json` 会返回更完整的上游识别结果。当前结果可能包含：

- `text`：全文；
- `utterances`：逐句结果；
- `start_time`、`end_time`：时间戳；
- `words`：逐字结果及其时间戳。

调用方应兼容上游结果中的可选字段，不应假设每个音频都一定包含 speaker、channel 或 words 信息。

## 7. 服务端处理流程

文件模式请求在 New API 内部按以下顺序执行：

1. 解析 multipart 表单。
2. 校验音频不为空，且不超过 128 MiB。
3. 使用随机对象键上传到 TOS。
4. 为 TOS 对象生成短期 GET 预签名 URL。
5. 调用火山录音文件识别 submit 接口。
6. 保存 submit 返回的请求关联 ID。
7. 调用 query 接口轮询识别结果。
8. 空响应或 `{}` 视为处理中，继续轮询。
9. 返回识别文本或详细 JSON。
10. 无论成功、失败、超时或取消，均尝试删除临时 TOS 对象。

文件识别请求的总超时时间和音频大小受服务端限制。超出限制时，调用方应拆分音频或改用异步任务方案，不要无限重试同一个请求。

## 8. submit/query 协议调试要点

火山录音文件识别 HTTP 流程使用以下接口：

```text
POST https://openspeech.bytedance.com/api/v3/auc/bigmodel/submit
POST https://openspeech.bytedance.com/api/v3/auc/bigmodel/query
```

关键请求头包括：

```http
Content-Type: application/json
X-Api-Resource-Id: <RESOURCE_ID>
X-Api-Request-Id: <REQUEST_ID>
X-Api-Connect-Id: <REQUEST_ID>
```

认证请求头根据凭证类型选择：

```http
X-Api-Key: <ARK_API_KEY>
```

或：

```http
X-Api-App-Key: <APP_ID>
X-Api-Access-Key: <ACCESS_TOKEN>
```

submit 和 query 必须使用同一个 `X-Api-Request-Id` 和 `X-Api-Connect-Id`。调试时只记录脱敏后的请求 ID、Resource-Id、HTTP 状态和耗时，不要记录完整预签名 URL 或请求体中的敏感内容。

## 9. 推荐调试顺序

### 第一步：检查后台登录和配置入口

确认访问的是正确服务实例，并已登录管理员账号：

```text
/system-settings/operations/performance
```

如果页面没有 TOS 区域，检查运行中的前端是否为包含 TOS 配置的版本，并执行前端重新构建及服务重启。

### 第二步：测试 TOS 连接

在性能设置页面点击 **Test TOS connection**。

- 如果提示 TOS disabled：打开 TOS 开关并保存。
- 如果 Bucket 检查失败：检查 Endpoint、Region、Bucket、AK/SK 和 Bucket 权限。
- 如果返回成功：说明服务端可以访问目标 Bucket，但不代表火山 ASR 模型权限已经正确。

### 第三步：测试渠道连接

确认渠道：

- 类型为 VolcEngine/火山方舟；
- Base URL 是火山 API 地址；
- API Key 格式正确；
- 模型列表和实际调用的 `model` 一致；
- 分组允许当前平台令牌使用。

### 第四步：使用小音频调用

先使用小于 128 MiB、时长较短的 MP3 或 WAV 文件，传入：

```text
metadata={"mode":"file"}
```

先使用 `response_format=text` 验证主链路，再使用 `verbose_json` 检查逐句、逐字和时间戳。

### 第五步：按链路定位错误

| 现象 | 优先检查 |
|---|---|
| TOS disabled | 性能设置中的 TOS 开关是否保存成功 |
| TOS bucket check failed | Endpoint、Region、Bucket、AK/SK、Bucket 权限和网络 |
| 401 | New API 平台令牌、火山 API Key 或旧版凭证格式 |
| 403 | TOS 权限、火山模型授权、Resource-Id、账号项目归属 |
| 404 | 渠道 Base URL 是否误填 TOS 地址，或 Ark/Agent Plan 地址混用 |
| audio file is empty | multipart 是否包含 `file`，文件是否为空 |
| audio file exceeds limit | 音频是否超过 128 MiB |
| 未进入文件识别 | `metadata` 是否为合法 JSON 且 `mode` 是否严格为 `file` |
| query 长时间超时 | submit/query request ID 是否一致、上游任务状态、音频大小和网络 |
| 无逐字结果 | 上游模型或音频未提供 words 字段，不能仅凭客户端参数强制生成 |
| 删除失败告警 | TOS 删除权限、网络和 Bucket 状态；不影响已返回的识别结果，但需清理残留对象 |

## 10. Endpoint 填写错误示例

以下填写方式不要使用：

```text
Endpoint = https://tos-s3-cn-guangzhou.volces.com
```

原因：这是 S3 兼容 Endpoint，不是当前实现使用的原生 TOS Endpoint。

```text
Endpoint = https://35sz.tos-cn-guangzhou.volces.com
```

原因：这是包含 Bucket 的访问域名。当前配置应将域名中的 Bucket 部分拆出，分别填写 Endpoint 和 Bucket。

```text
火山 ASR 渠道 Base URL = https://tos-cn-guangzhou.volces.com
```

原因：TOS Endpoint 仅用于对象存储，不能作为火山 ASR/Ark API 渠道地址。

## 11. 安全要求

1. TOS Bucket 默认保持私有，不要为了调试长期开启公共读。
2. Public Base URL 建议留空，优先使用短期预签名 URL。
3. 预签名 URL 有效期只需覆盖 submit/query 所需时间，不要无理由设置为最大值。
4. 不要在前端、智能体平台、日志、截图、Issue 或 Git 中保存 TOS Secret Key。
5. 不要把完整签名 URL写入日志；签名查询参数具有访问能力。
6. 临时音频包含个人信息、会议内容或业务机密时，应按组织的数据合规要求处理。
7. 生产环境使用专用账号、最小权限和定期密钥轮换。
8. 不要通过 `file_url` 或其他未确认的外部 URL 绕过服务端中转和安全校验。

## 12. 相关官方文档

- [火山引擎 TOS 对接文档](https://docs.volcengine.com/docs/6349/93480?lang=zh)
- [火山引擎录音文件识别标准版 HTTP](https://docs.volcengine.com/docs/6561/1354868?lang=zh)
- [火山引擎方舟控制台](https://console.volcengine.com/ark/)

官方接口字段、模型名称、Resource-Id、权限和计费规则可能随控制台版本或开通产品变化。遇到模型或 Resource-Id 不匹配时，以火山控制台当前显示值和对应官方文档为准，不要根据本文自行新增协议字段。
