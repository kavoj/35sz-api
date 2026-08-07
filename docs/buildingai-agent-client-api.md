# BuildingAI 智能体调用端接口文档

**版本**：V1.0  
**适用对象**：基于 BuildingAI 二次开发的 Web、后端服务、工作流节点和 MV 歌词转写客户端  
**网关**：35sz-api OpenAI 兼容接口

> 本文定义调用端与网关之间的稳定契约。BuildingAI 的页面和内部服务可以自行封装业务字段，但不得依赖 BuildingAI 官网前端的私有接口。模型、渠道、Resource-Id 和上游协议由网关管理员配置，调用端只提交标准 OpenAI 请求。

## 1. 接入配置

```text
API Base URL: https://<gateway-host>/v1
API Key:      sk-xxxxxxxx
```

请求头：

```http
Authorization: Bearer sk-xxxxxxxx
Content-Type: application/json
```

API Key 只放在 BuildingAI 服务端或受信任的后端，不要嵌入浏览器、移动端或公开工作流配置。

## 2. 模型发现

### GET `/v1/models`

```bash
curl https://<gateway-host>/v1/models \
  -H 'Authorization: Bearer sk-xxxxxxxx'
```

调用端应使用返回的 `id` 作为后续 `model`，不得自行拼接渠道模型名。

响应为 OpenAI 兼容格式：

```json
{
  "object": "list",
  "data": [
    {"id": "doubao-seed-asr-2-0", "object": "model", "owned_by": "volcengine"}
  ]
}
```

## 3. 智能体对话

### POST `/v1/chat/completions`

```json
{
  "model": "doubao-seed-1-6-thinking",
  "messages": [
    {"role": "system", "content": "你是企业知识助手。"},
    {"role": "user", "content": "总结本周项目风险。"}
  ],
  "stream": true,
  "stream_options": {"include_usage": true},
  "user": "buildingai-user-123",
  "metadata": {
    "agent_id": "agent-project-risk",
    "workflow_id": "weekly-risk-review",
    "conversation_id": "conv-20260807-001",
    "trace_id": "trace-001"
  }
}
```

`metadata` 用于审计、计费关联和故障定位；不要放 API Key、完整提示词以外的隐私密钥或其他凭证。网关不应将 prompt/completion 持久化为业务记录。

非流式请求将返回 OpenAI `chat.completion`；流式请求返回 `text/event-stream`，每个事件为 `data: {JSON}`，以 `data: [DONE]` 结束。

## 4. 智能体工具调用

网关透传 OpenAI-compatible `tools`、`tool_choice`、`tool_calls` 字段。BuildingAI 负责执行本地工具并将结果以 `role=tool` 消息回传；工具名和参数必须在 BuildingAI 侧做白名单校验。

## 5. 录音文件识别

### POST `/v1/audio/transcriptions`

```bash
curl https://<gateway-host>/v1/audio/transcriptions \
  -H 'Authorization: Bearer sk-xxxxxxxx' \
  -F 'file=@./meeting.mp3' \
  -F 'model=doubao-seed-asr-2-0' \
  -F 'response_format=verbose_json'
```

调用端只需要上传文件。火山渠道的“自动兼容”模式按以下规则选择上游：

1. 标准 multipart 文件上传默认走 HTTP 录音文件识别。
2. `metadata={"mode":"file"}` 明确要求文件识别，继续兼容旧客户端。
3. 只有 `metadata={"mode":"stream"}` 才请求 WebSocket 流式识别。
4. 文件模式的 Resource-Id 由渠道配置提供，例如 `volc.bigasr.auc_turbo`；调用端不能传 `X-Api-Resource-Id` 覆盖它。

支持响应格式：`json`、`verbose_json`、`text`。`verbose_json` 的 `utterances`、`words` 和说话人字段均为可选，客户端必须允许字段缺失。

## 6. 实时流式识别

实时场景不应伪装成普通文件上传。BuildingAI 可使用网关约定的 WebSocket 转发入口（由部署版本公布），并在会话元数据中指定：

```json
{"mode":"stream","conversation_id":"conv-001"}
```

流式 Resource-Id 由渠道配置选择，例如 `volc.seedasr.sauc.duration`。若部署没有公开 WebSocket 入口，使用文件接口即可，调用端无需直接连接火山 OpenSpeech。

## 7. 错误处理

客户端按 HTTP 状态码处理：

| 状态 | 处理 |
|---|---|
| 400 | 修正请求字段后重试，不要盲目重试 |
| 401 | 检查 BuildingAI 服务端 API Key |
| 403 | 联系管理员检查渠道权限、模型和 Resource-Id |
| 429 | 按 `Retry-After` 或指数退避重试 |
| 502/504 | 使用同一 `trace_id` 重试一次，并记录网关 request id |

错误响应采用：

```json
{"error":{"message":"...","type":"...","code":"..."}}
```

不得把上游 API Key、预签名 TOS URL 或完整请求体写入日志。

## 8. Node.js 示例

```ts
import OpenAI from 'openai'

const client = new OpenAI({
  apiKey: process.env.BUILDINGAI_GATEWAY_KEY,
  baseURL: `${process.env.BUILDINGAI_GATEWAY_URL}/v1`,
})

const stream = await client.chat.completions.create({
  model: 'doubao-seed-1-6-thinking',
  messages: [{ role: 'user', content: '生成项目周报摘要' }],
  stream: true,
  metadata: { agent_id: 'weekly-report', trace_id: crypto.randomUUID() },
})

for await (const chunk of stream) {
  process.stdout.write(chunk.choices[0]?.delta?.content ?? '')
}
```

## 9. 兼容性验收

- 同一套 BuildingAI 代码只配置一个 OpenAI-compatible Base URL。
- 更换火山文件版/流式版 Resource-Id 不需要发布 BuildingAI。
- 普通音频文件上传不传 metadata 也能完成录音文件识别。
- `metadata.mode=stream` 才启用流式协议。
- 所有请求可用 `agent_id`、`workflow_id`、`conversation_id`、`trace_id` 关联审计和计费。
- 失败重试不会重复扣费；服务端以 request id 和消费日志保证幂等。

## 10. 参考

- BuildingAI：<https://www.buildingai.cc/>
- 火山引擎语音模型接入文档：<https://docs.volcengine.com/docs/6561/1354869?lang=zh>
- 本仓库定价与 BuildingAI 集成方案：[buildingai-agent-integration-plan.md](./buildingai-agent-integration-plan.md)
