# 渠道创建与配置指南

> 适用版本：以当前仓库版本为准。本文只使用示例值，不包含真实 API Key。

渠道是 New API 连接上游模型服务的配置单元。一个渠道可以承载多个模型，并通过分组、权重、优先级和模型映射参与路由。完整字段以管理端当前版本和 [`docs/openapi/api.json`](../openapi/api.json) 为准。

## 一、创建渠道

进入管理端 **渠道** → **创建渠道**，按以下顺序填写：

| 字段 | 用途 | 配置建议 |
|---|---|---|
| 渠道名称 | 便于识别供应商、环境或区域 | 使用“供应商-区域-用途”，例如 `OpenAI-生产` |
| 渠道类型 | 选择适配器和协议 | 按上游真实协议选择，不要仅按模型名称选择 |
| Base URL | 上游基础地址 | 只填写适配器要求的基础地址；不要重复拼接 `/v1` 或 `/api/v3` |
| API Key | 上游凭证 | 仅在管理端录入；不要提交到 Git、截图或日志 |
| 模型列表 | 该渠道可提供的模型 | 逗号分隔；模型名必须与调用方使用的名称或映射一致 |
| 分组 | 路由和用户可用范围 | 与令牌/用户分组保持一致 |
| 权重 | 同一模型多渠道的流量比例 | 按容量、成本和稳定性设置；不是优先级替代品 |
| 优先级 | 选择主用/备用渠道 | 先按优先级尝试，再按路由策略在同级渠道间分配 |
| 自动禁用 | 连续失败时临时禁用渠道 | 生产建议开启，同时配置合理的重试次数 |
| 代理 | 当前渠道的网络出口 | 只有确实需要代理时填写，并验证代理可访问上游 |

保存后先点击 **测试连接**，再使用 **测试渠道连接** 按具体模型验证。

## 二、常见渠道示例

### 1. OpenAI 兼容上游

- 渠道类型：选择对应的 OpenAI 兼容类型。
- Base URL：填写上游公开的兼容根地址。
- 模型列表：填写上游模型 ID。
- API Key：填写上游授权密钥。
- 如果上游要求非标准请求头，在渠道的请求头覆盖中配置，并先用单模型测试验证。

### 2. Claude / Gemini

优先选择对应的原生渠道类型，让适配器负责请求格式和响应转换。只有上游明确提供兼容接口时，才选择 OpenAI 兼容渠道。

### 3. 火山方舟 Agent Plan 语音模型

火山方舟新控制台 API Key（通常以 `ark-` 开头）与旧版语音 `appid|access_token` 不是同一种凭证，不能互换。

| 模型 | Resource-Id | Agent Plan 端点 | 说明 |
|---|---|---|---|
| 豆包语音合成模型 2.0 | `seed-tts-2.0` | `wss://openspeech.bytedance.com/api/v3/plan/tts/bidirection` | 新 API Key 使用 `X-Api-Key` |
| 豆包流式语音识别模型 2.0 | `volc.seedasr.sauc.duration` | `wss://openspeech.bytedance.com/api/v3/plan/sauc/bigmodel_async` | 新 API Key 使用 `X-Api-Key` |

配置要点：

1. 渠道类型选择 **火山方舟**。
2. Agent Plan Base URL 使用控制台对应的 Agent Plan 地址；普通 Ark 请求和语音 OpenSpeech WebSocket 由适配器分别路由。
3. 直接填写新控制台 API Key，不要拼接成 `appid|access_token`。
4. 模型列表分别加入实际授权的模型名称。
5. 在火山控制台确认模型已开通、已开启超额后付费，并确认 TTS 音色已授权。
6. 分别测试 TTS 和 ASR；测试成功只说明链路和协议可用，实际转写/合成还受音频内容和音色授权影响。

旧版火山 TTS 渠道若使用 `appid|access_token`，仍走旧版兼容逻辑，不要在同一个渠道混用两套认证。

### 4. 音频渠道

- 语音合成：`POST /v1/audio/speech`。
- 语音识别：`POST /v1/audio/transcriptions`，通常使用 multipart 字段 `file`。
- 自定义上游若只支持 HTTP，使用自定义 Base URL；火山 Agent Plan 语音由火山适配器选择 WebSocket 端点。

## 三、额外设置

渠道额外设置 JSON 可参考 [`docs/channel/other_setting.md`](../channel/other_setting.md)：

```json
{
  "force_format": false,
  "thinking_to_content": false,
  "proxy": ""
}
```

- `force_format`：必要时强制转换为 OpenAI 格式。
- `thinking_to_content`：将推理内容转换为 `<think>` 标签拼接返回。
- `proxy`：为该渠道设置代理地址。

请求头覆盖应只配置合法、必要且不包含真实密钥的模板。不要把 API Key 写入普通文档、代码或客户端仓库。

## 四、测试与排错

1. **测试连接失败**：检查 Base URL、DNS/TLS、代理、密钥和上游权限。
2. **模型不存在**：检查渠道模型列表、模型映射和上游模型 ID。
3. **401/403**：检查密钥类型、Resource-Id、上游订阅或模型授权；火山 Agent Plan 新 key 不要使用旧 `appid|access_token` 格式。
4. **404**：检查是否把普通 Ark API 路径和 Agent Plan 路径混用。
5. **超时**：检查上游响应时间、流式超时、代理和渠道自动禁用状态。
6. **调用成功但无内容**：检查请求格式、音频文件字段、响应格式、模型能力和上游返回结构。

测试完成后到 **使用日志/任务日志** 核对请求模型、渠道、响应状态和耗时。生产环境应定期轮换密钥，并限制管理端访问。

返回：[README.zh_CN.md](../../README.zh_CN.md) · [模型管理](model-management.zh_CN.md) · [模型调用](model-invocation.zh_CN.md) · [价格与计费](pricing-and-billing.zh_CN.md)
