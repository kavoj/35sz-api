# 35sz-api 定价接口调用示例文档

**接口地址**：`https://token.35sz.top/v1/models/pricing`
**更新日期**：2026-07-15
**版本**：V1.0（PR-6 已上线，含 `pricing_incomplete` 数据质量标记）
**面向**：BuildingAI Agent 平台后端开发工程师、下游 SaaS 平台集成商

---

## 一、接口概览

- **方法**：`GET`
- **完整 URL**：`https://token.35sz.top/v1/models/pricing`
- **鉴权**：管理员 Access Token + New-Api-User Header **两个必须都传**
- **返回**：所有已上线模型的原生单位定价 (USD-anchored)
- **建议轮询频率**：每 5 分钟一次
- **建议缓存时长**：客户端本地 5 分钟
- **限流**：无（Admin 内网调用；异常轮询请自查）

---

## 二、鉴权

### 2.1 必需的三样东西

| 项 | 从哪里拿 |
|---|---|
| **Access Token** | 35sz-api 后台 → 个人设置 → 访问令牌 → 生成/复制。**注意：不是 `sk-` 开头的 API Key，那是模型调用用的；这里要用 access_token** |
| **User ID** | 35sz-api 后台 → 个人设置 → 顶部显示的用户 ID（数字，例如 `1` = root 用户） |
| **用户角色** | 该 access_token 对应的账号必须是 **Admin** 或 **Root** 角色，普通用户无权访问 |

### 2.2 Header 格式

```http
GET /v1/models/pricing HTTP/1.1
Host: token.35sz.top
Authorization: Bearer <ACCESS_TOKEN>
New-Api-User: <USER_ID>
```

### 2.3 常见 401 错误原因

| 错误 | 状态码 | 根因 | 修复方式 |
|---|---|---|---|
| `not logged in and no access token provided` | 401 | 未传 `Authorization` header | 补上 `Authorization: Bearer <token>` |
| `access token invalid` | 200 (JSON) | Token 已删除/过期/被禁用 | 后台重新生成 |
| `user id not provided` | 401 | 缺 `New-Api-User` header | 补上 `New-Api-User: <数字 ID>` |
| `user id mismatch` | 401 | `New-Api-User` 值与 access_token 的持有者不匹配 | 用同一账号生成 token 并传对应 ID |
| `insufficient privilege` | 200 (JSON) | 账号角色不是 Admin/Root | 让 root 授予该账号 Admin 权限 |
| `Invalid URL (POST /v1/models/pricing)` | 404 | 生产环境部署的分支不含 PR-6 | 部署 `newpay` 分支或 cherry-pick `b594ba52` 到 main |

---

## 三、请求示例

### 3.1 curl

```bash
curl -s https://token.35sz.top/v1/models/pricing \
  -H "Authorization: Bearer yYdbeXXXXXXXXXXXX" \
  -H "New-Api-User: 1"
```

### 3.2 HTTPie

```bash
http GET https://token.35sz.top/v1/models/pricing \
  Authorization:"Bearer yYdbeXXXXXXXXXXXX" \
  New-Api-User:1
```

### 3.3 Node.js / TypeScript

```typescript
const resp = await fetch('https://token.35sz.top/v1/models/pricing', {
  headers: {
    Authorization: `Bearer ${process.env.UPSTREAM_ACCESS_TOKEN}`,
    'New-Api-User': process.env.UPSTREAM_USER_ID ?? '1',
  },
})
const body = await resp.json()
if (!body.success) throw new Error(body.message ?? 'unknown')
```

### 3.4 Python (requests)

```python
import os, requests
resp = requests.get(
    "https://token.35sz.top/v1/models/pricing",
    headers={
        "Authorization": f"Bearer {os.environ['UPSTREAM_ACCESS_TOKEN']}",
        "New-Api-User": os.environ.get("UPSTREAM_USER_ID", "1"),
    },
    timeout=10,
)
body = resp.json()
assert body["success"], body.get("message")
```

### 3.5 Go

```go
req, _ := http.NewRequest("GET", "https://token.35sz.top/v1/models/pricing", nil)
req.Header.Set("Authorization", "Bearer "+os.Getenv("UPSTREAM_ACCESS_TOKEN"))
req.Header.Set("New-Api-User", os.Getenv("UPSTREAM_USER_ID"))
resp, err := http.DefaultClient.Do(req)
```

---

## 四、响应结构

### 4.1 顶层信封

```json
{
  "success": true,
  "data": { ... },
  "updated_at": 1721123456,
  "pricing_version": "5a90f2b86c08bd983a9a2e6d66c255f4eaef9c4bc934386d2b6ae84ef0ff1f1f"
}
```

| 字段 | 类型 | 说明 |
|---|---|---|
| `success` | bool | 是否成功。**若为 false，请读取 `message` 字段** |
| `data` | object | Key = 模型名 (string)，Value = 见 4.2 单模型结构 |
| `updated_at` | int64 (Unix 秒) | 服务端生成响应的时间戳 |
| `pricing_version` | string (SHA256 hex) | 定价版本哈希。**多次拉取时若该值不变，可直接跳过刷新缓存** |

### 4.2 单模型结构

```json
{
  "model_type": "video",
  "pricing_kind": "video-gen",
  "pricing_type": "per_second",
  "pricing": {
    "price_per_second": 6.301369,
    "resolution_multipliers": {"480p": 1.0, "720p": 1.0, "1080p": 1.1087, "4k": 0.5652},
    "has_audio_multiplier": 1.0
  }
}
```

| 字段 | 值域 | 用途 |
|---|---|---|
| `model_type` | `text` / `image` / `video` / `audio` / `embedding` / `file` | 粗粒度分类，UI filter chip |
| `pricing_kind` | 见下表 7 类 | 精细定价类型，决定 `pricing` 内字段结构 |
| `pricing_type` | `token` / `per_image` / `per_second` / `per_minute` / `per_million_chars` | 简化单位（`pricing_kind` 的降级映射） |
| `pricing` | object | 具体价格数值，字段随 `pricing_kind` 变化 |

### 4.3 `pricing_kind` 7 类枚举

| 值 | 计费维度 | pricing 字段 | 示例模型 |
|---|---|---|---|
| `chat` | 按 token | 6 个 ratio + 2 个 per_million_tokens | `gpt-4o`, `deepseek-v3`, `doubao-1-5-pro` |
| `multimodal-chat` | 按 token（含图/音输入） | 同 chat + 3 个模态 ratio | `gpt-4o` (支持图像), `qwen-vl` |
| `image-gen` | 按张 | `price_per_image` + 尺寸/质量倍率 | `doubao-seedream-5-0-pro`, `dall-e-3`, `flux-1.1` |
| `video-gen` | 按秒 | `price_per_second` + 分辨率/音频倍率 | `doubao-seedance-1-5-pro`, `sora`, `veo-3.1` |
| `audio-in` | 按分钟 | `price_per_minute` | `whisper-1`, `gpt-4o-transcribe` |
| `audio-out` | 按 1M 字符 | `price_per_million_chars` + 音色倍率 | `tts-1`, `gpt-4o-mini-tts` |
| `embedding` | 按 token（无 output） | 同 chat（但 completion_ratio 无意义） | `text-embedding-3-large` |

---

## 五、每种 pricing_kind 的 `pricing` 结构详解

### 5.1 `chat` / `multimodal-chat` (token-based)

**JSON 示例**（gpt-4o）：

```json
"gpt-4o": {
  "model_type": "text",
  "pricing_kind": "chat",
  "pricing_type": "token",
  "pricing": {
    "model_ratio": 2.5,
    "completion_ratio": 4.0,
    "cache_ratio": 0.5,
    "input_per_million_tokens": 5.0,
    "output_per_million_tokens": 20.0
  }
}
```

**字段说明**：

| 字段 | 类型 | 单位 | 说明 |
|---|---|---|---|
| `model_ratio` | float | 无量纲 | Input token 定价倍数。**1 unit = $2/1M tokens** |
| `completion_ratio` | float | 无量纲 | Output token 相对 input 的倍数 |
| `cache_ratio` | *float | 无量纲 | Cache hit 折扣（0.5 = 5 折） |
| `image_ratio` | *float | 无量纲 | 图像输入 token 相对文本 token 的倍数（仅 `multimodal-chat`） |
| `audio_ratio` | *float | 无量纲 | 音频输入相对文本的倍数（仅 `multimodal-chat`） |
| `audio_completion_ratio` | *float | 无量纲 | 音频输出相对音频输入的倍数（仅 `multimodal-chat`） |
| **`input_per_million_tokens`** | float | **USD/1M tokens** | **推荐使用**：已 denormalize（= `model_ratio × 2`），无需下游做 `× 2` |
| **`output_per_million_tokens`** | float | **USD/1M tokens** | **推荐使用**：已 denormalize（= `model_ratio × 2 × completion_ratio`） |

**计费公式**：

```
成本 USD = (input_tokens / 1_000_000) × input_per_million_tokens
        + (output_tokens / 1_000_000) × output_per_million_tokens
        + (cached_tokens / 1_000_000) × input_per_million_tokens × cache_ratio
```

### 5.2 `image-gen`

**JSON 示例**（doubao-seedream-5-0-pro）：

```json
"doubao-seedream-5-0-pro-260628": {
  "model_type": "image",
  "pricing_kind": "image-gen",
  "pricing_type": "per_image",
  "pricing": {
    "price_per_image": 0.041,
    "size_multipliers": {
      "1024x1024": 1.0,
      "2048x2048": 2.0,
      "2560x1440": 2.25,
      "1728x2304": 2.25
    },
    "quality_multipliers": {
      "standard": 1.0,
      "hd": 2.0
    }
  }
}
```

**字段说明**：

| 字段 | 类型 | 单位 | 说明 |
|---|---|---|---|
| `price_per_image` | float | USD/张 | 基础单价（1024x1024 standard 质量） |
| `size_multipliers` | map | 无量纲 | Key = 尺寸串（"WIDTHxHEIGHT"），Value = 相对基础价倍数 |
| `quality_multipliers` | map | 无量纲 | Key = quality 参数值（"standard"/"hd"/"low"/"medium"），Value = 倍数 |

**计费公式**：

```
成本 USD = n × price_per_image
        × (size_multipliers[size] || 1.0)
        × (quality_multipliers[quality] || 1.0)
```

### 5.3 `video-gen`

**JSON 示例**（doubao-seedance-2-0-260128）：

```json
"doubao-seedance-2-0-260128": {
  "model_type": "video",
  "pricing_kind": "video-gen",
  "pricing_type": "per_second",
  "pricing": {
    "price_per_second": 6.301369,
    "resolution_multipliers": {
      "480p": 1.0,
      "720p": 1.0,
      "1080p": 1.1087,
      "4k": 0.5652
    },
    "has_audio_multiplier": 1.0
  }
}
```

**字段说明**：

| 字段 | 类型 | 单位 | 说明 |
|---|---|---|---|
| `price_per_second` | float | USD/秒 | 基础单价（720p，无音频） |
| `resolution_multipliers` | map | 无量纲 | Key = 分辨率（小写：`480p`/`720p`/`1080p`/`4k`），Value = 相对倍数 |
| `has_audio_multiplier` | float | 无量纲 | 视频**含音频**时的倍数（Veo 3.1 = 1.5）；无音频或不区分则为 1.0 或 0 |

**计费公式**：

```
成本 USD = duration_seconds × price_per_second
        × (resolution_multipliers[resolution] || 1.0)
        × (hasAudio ? has_audio_multiplier : 1.0)
```

### 5.4 `audio-in` (ASR)

**JSON 示例**（whisper-1）：

```json
"whisper-1": {
  "model_type": "audio",
  "pricing_kind": "audio-in",
  "pricing_type": "per_minute",
  "pricing": {
    "price_per_minute": 0.006
  }
}
```

**字段说明**：

| 字段 | 类型 | 单位 | 说明 |
|---|---|---|---|
| `price_per_minute` | float | USD/分钟 | 单价 |
| `min_bill_minutes` | float | 分钟 | 最小计费单位（如 0.25 = 15 秒最小值） |

**计费公式**：

```
成本 USD = max(duration_minutes, min_bill_minutes) × price_per_minute
```

### 5.5 `audio-out` (TTS)

**JSON 示例**（tts-1）：

```json
"tts-1": {
  "model_type": "audio",
  "pricing_kind": "audio-out",
  "pricing_type": "per_million_chars",
  "pricing": {
    "price_per_million_chars": 15.0,
    "voice_multipliers": {
      "nova": 1.0,
      "echo": 1.0,
      "clone": 2.0
    }
  }
}
```

**字段说明**：

| 字段 | 类型 | 单位 | 说明 |
|---|---|---|---|
| `price_per_million_chars` | float | USD/1M chars | 单价 |
| `voice_multipliers` | map | 无量纲 | Key = voice 参数，Value = 相对倍数 |

**计费公式**：

```
成本 USD = (char_count / 1_000_000) × price_per_million_chars
        × (voice_multipliers[voice] || 1.0)
```

### 5.6 `embedding`

**JSON 示例**（text-embedding-3-large）：

```json
"text-embedding-3-large": {
  "model_type": "embedding",
  "pricing_kind": "embedding",
  "pricing_type": "token",
  "pricing": {
    "model_ratio": 0.065,
    "input_per_million_tokens": 0.13,
    "output_per_million_tokens": 0.13
  }
}
```

**说明**：Embedding 模型只有 input token（无 output），downstream 忽略 `output_per_million_tokens` 或视其等同于 input 即可。

**计费公式**：

```
成本 USD = (input_tokens / 1_000_000) × input_per_million_tokens
```

### 5.7 `flat_price_per_request` (跨 kind 通用字段)

任何 kind 都可能有该字段（含 `chat`、`image-gen` 等），非零时应**优先于其他字段**：

```json
"some-custom-model": {
  "model_type": "text",
  "pricing_kind": "chat",
  "pricing_type": "token",
  "pricing": {
    "model_ratio": 0,
    "flat_price_per_request": 0.02
  }
}
```

**语义**：本次调用固定收 `$0.02`，忽略 token 数、图张数、视频秒数。

**用途**：管理员的 escape hatch，通常用于：
- 阶梯定价过于复杂的自研模型
- 与上游签的固定包月/包次合同
- Legacy 数据从 `ModelPrice` 迁移过来的模型

**处理规则**：

```typescript
if (pricing.flat_price_per_request && pricing.flat_price_per_request > 0) {
  cost = pricing.flat_price_per_request
} else {
  // 走 pricing_kind 的正常公式
}
```

### 5.8 `pricing_incomplete` (数据质量标记)

若返回：

```json
"some-model": {
  "model_type": "video",
  "pricing_kind": "video-gen",
  "pricing_type": "per_second",
  "pricing": {
    "pricing_incomplete": true
  }
}
```

**含义**：模型被分类为 `video-gen`（或其他结构化 kind），但对应的 `VideoPricing` / `ImagePricing` / `AudioInPricing` / `AudioOutPricing` 表中**没有配置或价格为 0**。可能原因：
1. 管理员刚在 UI 里把模型分类改成了 `video-gen`，但还没填价格
2. Seed 数据没覆盖到这个模型名
3. Legacy 数据从 `ModelPrice` 迁移但未补齐

**下游处理**（重要）：

```typescript
if (pricing.pricing_incomplete) {
  // 不要用 flat_price_per_request 或其他字段（都可能是 0 或误导）
  // 走本地 fallback 兜底表 或 拒绝提供此模型
  logger.warn(`Model ${modelId} pricing incomplete on upstream, using local fallback`)
  return localFallbackPricing[modelId]
}
```

**如果你的运维流程严谨**，可以监控这个字段，非零时向管理员发告警。

---

## 六、完整响应示例

以下是生产环境实际返回的样本（脱敏）：

```json
{
  "success": true,
  "data": {
    "deepseek-v4-flash": {
      "model_type": "text",
      "pricing_kind": "chat",
      "pricing_type": "token",
      "pricing": {
        "model_ratio": 0.07,
        "completion_ratio": 2.0,
        "input_per_million_tokens": 0.14,
        "output_per_million_tokens": 0.28
      }
    },
    "gpt-4o": {
      "model_type": "text",
      "pricing_kind": "multimodal-chat",
      "pricing_type": "token",
      "pricing": {
        "model_ratio": 2.5,
        "completion_ratio": 4.0,
        "cache_ratio": 0.5,
        "image_ratio": 1.5,
        "input_per_million_tokens": 5.0,
        "output_per_million_tokens": 20.0
      }
    },
    "doubao-seedream-5-0-pro-260628": {
      "model_type": "image",
      "pricing_kind": "image-gen",
      "pricing_type": "per_image",
      "pricing": {
        "price_per_image": 0.041,
        "size_multipliers": {"1024x1024": 1.0, "2048x2048": 2.0}
      }
    },
    "doubao-seedance-2-0-260128": {
      "model_type": "video",
      "pricing_kind": "video-gen",
      "pricing_type": "per_second",
      "pricing": {
        "price_per_second": 6.301369,
        "resolution_multipliers": {
          "480p": 1.0,
          "720p": 1.0,
          "1080p": 1.1087,
          "4k": 0.5652
        }
      }
    },
    "doubao-seedance-1-5-pro-251215": {
      "model_type": "video",
      "pricing_kind": "video-gen",
      "pricing_type": "per_second",
      "pricing": {
        "pricing_incomplete": true
      }
    },
    "whisper-1": {
      "model_type": "audio",
      "pricing_kind": "audio-in",
      "pricing_type": "per_minute",
      "pricing": {
        "price_per_minute": 0.006
      }
    },
    "tts-1": {
      "model_type": "audio",
      "pricing_kind": "audio-out",
      "pricing_type": "per_million_chars",
      "pricing": {
        "price_per_million_chars": 15.0,
        "voice_multipliers": {"nova": 1.0, "clone": 2.0}
      }
    },
    "text-embedding-3-large": {
      "model_type": "embedding",
      "pricing_kind": "embedding",
      "pricing_type": "token",
      "pricing": {
        "model_ratio": 0.065,
        "input_per_million_tokens": 0.13,
        "output_per_million_tokens": 0.13
      }
    }
  },
  "updated_at": 1721123456,
  "pricing_version": "5a90f2b86c08bd983a9a2e6d66c255f4eaef9c4bc934386d2b6ae84ef0ff1f1f"
}
```

---

## 七、Agent 平台参考实现

### 7.1 TypeScript 类型定义

```typescript
// 建议放在 packages/api/src/modules/pricing/types.ts
export type PricingKind =
  | 'chat'
  | 'multimodal-chat'
  | 'image-gen'
  | 'video-gen'
  | 'audio-in'
  | 'audio-out'
  | 'embedding'

export type PricingType =
  | 'token'
  | 'per_image'
  | 'per_second'
  | 'per_minute'
  | 'per_million_chars'

export type ModelType =
  | 'text'
  | 'image'
  | 'video'
  | 'audio'
  | 'embedding'
  | 'file'

export interface PricingBody {
  // Token
  model_ratio?: number
  completion_ratio?: number
  cache_ratio?: number
  image_ratio?: number
  audio_ratio?: number
  audio_completion_ratio?: number
  input_per_million_tokens?: number
  output_per_million_tokens?: number
  // Image
  price_per_image?: number
  quality_multipliers?: Record<string, number>
  size_multipliers?: Record<string, number>
  // Video
  price_per_second?: number
  resolution_multipliers?: Record<string, number>
  has_audio_multiplier?: number
  // Audio-in
  price_per_minute?: number
  min_bill_minutes?: number
  // Audio-out
  price_per_million_chars?: number
  voice_multipliers?: Record<string, number>
  // Cross-kind
  flat_price_per_request?: number
  pricing_incomplete?: boolean
}

export interface ModelPricingEntry {
  model_type: ModelType
  pricing_kind: PricingKind
  pricing_type: PricingType
  pricing: PricingBody
}

export interface ModelPricingResponse {
  success: boolean
  message?: string
  data: Record<string, ModelPricingEntry>
  updated_at: number
  pricing_version: string
}
```

### 7.2 计算成本的通用函数

```typescript
export interface CalcInput {
  // 通用
  modelId: string
  // Token 类
  inputTokens?: number
  outputTokens?: number
  cachedTokens?: number
  // Image
  imageCount?: number
  size?: string
  quality?: string
  // Video
  durationSeconds?: number
  resolution?: string
  hasAudio?: boolean
  // Audio-in
  audioMinutes?: number
  // Audio-out
  charCount?: number
  voice?: string
}

export interface CalcResult {
  costUSD: number
  breakdown: string
  incomplete: boolean
}

export function calculateCost(
  entry: ModelPricingEntry,
  input: CalcInput,
): CalcResult {
  const p = entry.pricing

  // 数据不完整 → 上游还没定价 → 交给本地兜底
  if (p.pricing_incomplete) {
    return { costUSD: 0, breakdown: 'incomplete', incomplete: true }
  }

  // Flat override 优先级最高
  if (p.flat_price_per_request && p.flat_price_per_request > 0) {
    return {
      costUSD: p.flat_price_per_request,
      breakdown: `flat=${p.flat_price_per_request}`,
      incomplete: false,
    }
  }

  switch (entry.pricing_kind) {
    case 'chat':
    case 'multimodal-chat':
    case 'embedding': {
      const inputCost =
        ((input.inputTokens ?? 0) / 1_000_000) *
        (p.input_per_million_tokens ?? 0)
      const outputCost =
        ((input.outputTokens ?? 0) / 1_000_000) *
        (p.output_per_million_tokens ?? 0)
      const cacheCost =
        ((input.cachedTokens ?? 0) / 1_000_000) *
        (p.input_per_million_tokens ?? 0) *
        (p.cache_ratio ?? 1)
      return {
        costUSD: inputCost + outputCost + cacheCost,
        breakdown: `in=${inputCost.toFixed(6)} + out=${outputCost.toFixed(6)} + cache=${cacheCost.toFixed(6)}`,
        incomplete: false,
      }
    }

    case 'image-gen': {
      const base = p.price_per_image ?? 0
      const n = input.imageCount ?? 1
      const sizeMul = input.size ? (p.size_multipliers?.[input.size] ?? 1) : 1
      const qMul = input.quality
        ? (p.quality_multipliers?.[input.quality] ?? 1)
        : 1
      return {
        costUSD: n * base * sizeMul * qMul,
        breakdown: `n=${n} × base=${base} × size=${sizeMul} × quality=${qMul}`,
        incomplete: false,
      }
    }

    case 'video-gen': {
      const base = p.price_per_second ?? 0
      const seconds = input.durationSeconds ?? 0
      const resMul = input.resolution
        ? (p.resolution_multipliers?.[input.resolution.toLowerCase()] ?? 1)
        : 1
      const audMul = input.hasAudio ? (p.has_audio_multiplier ?? 1) : 1
      return {
        costUSD: seconds * base * resMul * audMul,
        breakdown: `sec=${seconds} × base=${base} × res=${resMul} × audio=${audMul}`,
        incomplete: false,
      }
    }

    case 'audio-in': {
      const minutes = Math.max(input.audioMinutes ?? 0, p.min_bill_minutes ?? 0)
      const rate = p.price_per_minute ?? 0
      return {
        costUSD: minutes * rate,
        breakdown: `min=${minutes} × rate=${rate}`,
        incomplete: false,
      }
    }

    case 'audio-out': {
      const chars = input.charCount ?? 0
      const rate = p.price_per_million_chars ?? 0
      const voiceMul = input.voice
        ? (p.voice_multipliers?.[input.voice] ?? 1)
        : 1
      return {
        costUSD: (chars / 1_000_000) * rate * voiceMul,
        breakdown: `chars=${chars} × rate=${rate} × voice=${voiceMul}`,
        incomplete: false,
      }
    }
  }
}
```

### 7.3 缓存 + 定时刷新 (NestJS)

```typescript
import { Injectable, Logger, OnModuleInit } from '@nestjs/common'
import { ConfigService } from '@nestjs/config'
import { HttpService } from '@nestjs/axios'
import { firstValueFrom } from 'rxjs'

@Injectable()
export class UpstreamPricingCache implements OnModuleInit {
  private readonly logger = new Logger(UpstreamPricingCache.name)
  private cache = new Map<string, ModelPricingEntry>()
  private lastVersion = ''
  private lastUpdatedAt = 0

  constructor(
    private readonly http: HttpService,
    private readonly config: ConfigService,
  ) {}

  async onModuleInit() {
    await this.refresh()
    setInterval(() => this.refresh(), 5 * 60 * 1000)
  }

  async refresh(): Promise<void> {
    try {
      const url = `${this.config.get('UPSTREAM_BASE')}/v1/models/pricing`
      const resp = await firstValueFrom(
        this.http.get<ModelPricingResponse>(url, {
          headers: {
            Authorization: `Bearer ${this.config.get('UPSTREAM_ACCESS_TOKEN')}`,
            'New-Api-User': String(this.config.get('UPSTREAM_USER_ID', 1)),
          },
          timeout: 10_000,
        }),
      )
      if (!resp.data.success) {
        this.logger.warn(`refresh failed: ${resp.data.message}`)
        return
      }
      if (resp.data.pricing_version === this.lastVersion && this.cache.size) {
        this.lastUpdatedAt = resp.data.updated_at
        return
      }
      const next = new Map<string, ModelPricingEntry>()
      Object.entries(resp.data.data).forEach(([id, entry]) => next.set(id, entry))
      this.cache = next
      this.lastVersion = resp.data.pricing_version
      this.lastUpdatedAt = resp.data.updated_at
      const incomplete = [...next.values()].filter(
        (e) => e.pricing.pricing_incomplete,
      ).length
      this.logger.log(
        `refreshed: ${next.size} models, ${incomplete} incomplete, version=${this.lastVersion}`,
      )
    } catch (err) {
      this.logger.warn(`refresh error, keeping stale cache: ${err.message}`)
    }
  }

  get(modelId: string): ModelPricingEntry | undefined {
    return this.cache.get(modelId)
  }

  stats() {
    return {
      total: this.cache.size,
      lastUpdatedAt: this.lastUpdatedAt,
      version: this.lastVersion,
      incomplete: [...this.cache.values()].filter(
        (e) => e.pricing.pricing_incomplete,
      ).length,
    }
  }
}
```

---

## 八、常见问题

### Q1: 为什么某些模型返回 `pricing_incomplete: true`？

**A**: 见 5.8 节。要么是 admin 分类改了但没填数，要么是 seed 数据不完整。**下游必须走本地兜底**，不能相信这个 entry 的价格字段。

### Q2: 我如何知道数据什么时候刷新？

**A**: 两种方式：

- **主动**：每次拉取都会返回 `pricing_version` 哈希，未变 = 数据没变
- **被动**（未来）：可以订阅 webhook（暂未实现，见路线图）

### Q3: 售价倍率 (markup) 由谁应用？

**A**: **由下游 Agent 平台自己应用**。本接口只返回**上游成本价**（USD），Agent 平台负责：

1. `costUSD × USDExchangeRate` → CNY 成本
2. `CNY 成本 × RechargePremium (默认 1.5)` → CNY 售价
3. `CNY 售价 × CreditsPerCNY (默认 10)` → 应扣积分

### Q4: 如果 35sz-api 暂时不可用怎么办？

**A**: Agent 平台的 `UpstreamPricingCache` 应：
1. 保留上次成功的缓存 (`refresh` 失败时不 clear)
2. 若冷启动时就失败 → 走本地 `fallback-pricing.ts` 兜底
3. 记录告警（每分钟持续失败要通知运维）

### Q5: 汇率不一致导致对账差异

**A**: 汇率必须与 35sz-api 侧保持一致。建议 Agent 平台**从 35sz-api 的 `GET /api/status` 拉取 `usd_exchange_rate` 字段**，而不是硬编码 7.3。

### Q6: 频繁轮询会不会被封？

**A**: 5 分钟一次是安全的。若你想更频繁（如 1 分钟），建议提前和运维沟通。当前实现无 rate-limit，但滥用可能触发 IP 层封锁。

### Q7: `pricing_type` 和 `pricing_kind` 是什么关系？

**A**: `pricing_kind` 更精细（7 类），`pricing_type` 更粗（5 类）。映射规则：

| pricing_kind | pricing_type |
|---|---|
| `chat`, `multimodal-chat`, `embedding` | `token` |
| `image-gen` | `per_image` |
| `video-gen` | `per_second` |
| `audio-in` | `per_minute` |
| `audio-out` | `per_million_chars` |

**下游只关心计费单位时用 `pricing_type`**；**要区分是否走多模态输入时用 `pricing_kind`**。

---

## 九、路线图

| 状态 | 项 | 说明 |
|---|---|---|
| ✅ V1.0 | 基础接口 | 本文档描述的全部功能 |
| ✅ V1.0 | `pricing_incomplete` 标记 | 数据不完整时下游降级 |
| 📅 V1.1 | `Cache-Control: max-age=300` header | 减少下游拉取负担 |
| 📅 V1.1 | webhook 通知定价变更 | 免轮询 |
| 📅 V1.2 | `GET /v1/models/pricing/:model_id` 单模型查询 | 按需拉取 |
| 📅 V1.2 | `updated_at` 精确到毫秒 | 更准的对账时间 |
| 📅 V2.0 | 双端对账接口 `GET /admin/billing/daily-usage` | Agent 平台每日对账 |

---

## 十、变更日志

**2026-07-15 V1.0**
- 初版接口上线（commit `b594ba52`）
- 新增 `pricing_incomplete` 数据质量标记
- 修复 video-gen 返回空对象的问题（现在改为返回 `pricing_incomplete: true`）

**技术支持**：如有疑问联系 35sz-api 维护者。
