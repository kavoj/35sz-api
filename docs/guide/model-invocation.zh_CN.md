# 模型调用指南

> 适用版本：以当前仓库版本为准。示例中的 `https://gateway.example.com` 和 `sk-your-token` 均为占位符。

## 一、调用前检查

1. 管理端已创建渠道，并在渠道模型列表中启用目标模型。
2. 模型所属分组与当前用户/令牌可用分组一致。
3. 上游密钥具有目标模型权限。
4. 已为模型设置价格，或确认该模型使用平台默认价格策略。
5. 创建 API Key 后，将客户端 Base URL 指向 New API 服务地址。

不要把真实上游 API Key 放到客户端。客户端只使用 New API 发放的令牌，New API 再根据渠道配置访问真实上游。

## 二、OpenAI Chat Completions

```bash
curl https://gateway.example.com/v1/chat/completions \
  -H "Authorization: Bearer sk-your-token" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "your-model",
    "messages": [{"role": "user", "content": "你好"}],
    "stream": false
  }'
```

流式调用：

```bash
curl https://gateway.example.com/v1/chat/completions \
  -H "Authorization: Bearer sk-your-token" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "your-model",
    "messages": [{"role": "user", "content": "写一段摘要"}],
    "stream": true
  }'
```

## 三、Responses、Embedding、图片与 Rerank

### Responses

```bash
curl https://gateway.example.com/v1/responses \
  -H "Authorization: Bearer sk-your-token" \
  -H "Content-Type: application/json" \
  -d '{"model":"your-responses-model","input":"介绍一下这个模型"}'
```

### Embeddings

```bash
curl https://gateway.example.com/v1/embeddings \
  -H "Authorization: Bearer sk-your-token" \
  -H "Content-Type: application/json" \
  -d '{"model":"your-embedding-model","input":["第一段文本","第二段文本"]}'
```

### 图片生成

```bash
curl https://gateway.example.com/v1/images/generations \
  -H "Authorization: Bearer sk-your-token" \
  -H "Content-Type: application/json" \
  -d '{"model":"your-image-model","prompt":"一座海边灯塔","size":"1024x1024"}'
```

### Rerank

```bash
curl https://gateway.example.com/v1/rerank \
  -H "Authorization: Bearer sk-your-token" \
  -H "Content-Type: application/json" \
  -d '{"model":"your-rerank-model","query":"什么是网关？","documents":["网关统一管理 API。","数据库保存业务数据。"]}'
```

## 四、音频

### 语音识别

```bash
curl https://gateway.example.com/v1/audio/transcriptions \
  -H "Authorization: Bearer sk-your-token" \
  -F "file=@./sample.wav" \
  -F "model=your-asr-model"
```

### 语音合成

```bash
curl https://gateway.example.com/v1/audio/speech \
  -H "Authorization: Bearer sk-your-token" \
  -H "Content-Type: application/json" \
  -d '{"model":"your-tts-model","input":"你好，这是语音测试。","voice":"alloy","response_format":"mp3"}' \
  --output speech.mp3
```

火山方舟 Agent Plan 语音模型由渠道适配器转发到真实的 OpenSpeech WebSocket。客户端仍使用统一的 OpenAI 音频接口；上游 Resource-Id、API Key 和音色在渠道中配置。

## 五、Claude、Gemini 与第三方客户端

如果渠道支持原生 Claude 或 Gemini 格式，使用对应的协议路径和请求结构；如果供应商提供 OpenAI 兼容接口，可统一使用 `/v1/chat/completions`。第三方客户端通常只需设置：

- Base URL：`https://gateway.example.com/v1`
- API Key：New API 发放的令牌
- Model：渠道模型列表中的平台模型名

实际支持的协议和字段以[接口文档](https://docs.newapi.pro/zh/docs/api)与 [`docs/openapi/api.json`](../openapi/api.json) 为准。

## 六、路由、重试与日志

- 同模型多渠道时，平台按分组、渠道状态、优先级、权重等选择上游。
- 上游失败时，若重试策略允许，平台会尝试其他可用渠道。
- 渠道自动禁用可避免持续把请求发送到故障上游。
- 管理端使用日志应核对模型、渠道、响应状态、耗时和扣费；不要在日志中打印原始 API Key。

常见状态：

| 现象 | 优先检查 |
|---|---|
| 401/403 | New API 令牌、渠道上游密钥、模型权限 |
| 404 | 模型名、Base URL、端点类型、模型映射 |
| 429 | 上游限流、渠道权重、重试和用户限流 |
| 5xx/超时 | 上游状态、代理、流式超时、渠道自动禁用 |
| 返回为空 | 请求格式、上游响应格式、音频字段和模型能力 |

返回：[README.zh_CN.md](../../README.zh_CN.md) · [渠道配置](channel-setup.zh_CN.md) · [模型管理](model-management.zh_CN.md) · [价格与计费](pricing-and-billing.zh_CN.md)
