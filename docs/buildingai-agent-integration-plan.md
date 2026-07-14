# BuildingAI Agent 平台接入 35sz-api 统一定价方案

**版本**：V1.0
**日期**：2026-07-14
**依赖**：`35sz-api` PR-6 已合入（GET /v1/models/pricing 接口就绪）
**目标读者**：BuildingAI Agent 平台后端开发工程师
**估工**：**2 天完成 P0 全量对接，2 小时可临时兜底修复图片扣费 BUG**

---

## 一、背景摘要

Agent 平台当前状态（`/Users/perjac/Downloads/随星AI-Agent平台统一计费方案.md` 已分析）：

| 模型类型 | 扣费状态 | 根因 |
|---|---|---|
| **LLM** | ✅ 正常按 token 扣费 | onFinish + calculatePower + deductUserPower 链路完整 |
| **视频** | ⚠️ 有回调但不扣费 | 后台未配置 `powerPerVideo`，`billingRule.power` 为 0 |
| **图片** | ❌ 完全不扣费（BUG） | `createSeedreamImageGenerationTool` 未传 `onSuccess` 回调 |

35sz-api 侧现状（截至 PR-6 已合入）：

- 已建立 `pricing_kind`：`chat / multimodal-chat / image-gen / video-gen / audio-in / audio-out / embedding` 7 分类
- 已建立 4 张结构化定价表：`VideoPricing / ImagePricing / AudioInPricing / AudioOutPricing`
- 已开放机器可读接口：`GET https://token.35sz.top/v1/models/pricing` 需 admin API key 鉴权

---

## 二、接入总览

```
BuildingAI Agent 后端
├── ① ModelPricingService (新建)
│   ├── 启动时 & 每 5 分钟拉取 35sz-api /v1/models/pricing
│   ├── 内存缓存 <modelId, PricingEntry>
│   └── getPrice(modelId, params) 匹配阶梯价
├── ② BillingPolicy (新建 - 三级优先级)
│   ├── 优先级 1: 管理员手动积分价 (per_use_credits 字段)
│   ├── 优先级 2: 上游拉取 → 售价倍率×汇率 换算为积分
│   └── 优先级 3: 代码兜底默认价
├── ③ 修图片工具 BUG (P0)
│   └── seedreamImageGenerationTool 加 onSuccess 回调
├── ④ 视频/音频工具接入 (P0)
│   └── 从 ModelPricingService 拉阶梯价，替换 billingRule.power
└── ⑤ 分账流水字段扩展 (P1)
    ├── cost_amount / sell_amount / profit_amount
    └── upstream_channel / upstream_model_id / usage_data / billing_status
```

---

## 三、上游接口契约

### 3.1 请求

```http
GET https://token.35sz.top/v1/models/pricing
Authorization: Bearer <ADMIN_ACCESS_TOKEN>
Accept: application/json
```

- **鉴权**：管理员 Access Token（在 35sz-api 后台"个人设置 → 访问令牌"页面生成）。**注意：不是 `sk-` 开头的 sk key，而是 admin 的 access_token；错用会返回 401**。
- **建议频率**：每 5 分钟轮询一次（正常情况下 35sz-api 侧数据变更也是分钟级）。
- **超时**：建议客户端超时 10s；失败降级到上一次缓存或代码兜底价。

### 3.2 响应结构

```json
{
  "success": true,
  "updated_at": 1720000000,
  "pricing_version": "a42d372ccf0b5dd13ecf71203521f9d2",
  "data": {
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
        },
        "has_audio_multiplier": 1.0
      }
    },

    "doubao-seedream-3-0": {
      "model_type": "image",
      "pricing_kind": "image-gen",
      "pricing_type": "per_image",
      "pricing": {
        "price_per_image": 0.02,
        "size_multipliers": {
          "1024x1024": 1.0,
          "2048x2048": 2.0,
          "2560x1440": 2.25
        }
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
        "model_ratio": 0.02,
        "input_per_million_tokens": 0.04,
        "output_per_million_tokens": 0.04
      }
    }
  }
}
```

### 3.3 关键字段说明

| 字段 | 含义 | 用于 |
|---|---|---|
| `model_type` | 前端类型：text/image/video/audio/embedding | Agent 平台 UI filter chip |
| `pricing_kind` | 定价类型 (7 枚举) | Agent 平台业务侧路由 onSuccess 参数结构 |
| `pricing_type` | 简化单位：token / per_image / per_second / per_minute / per_million_chars | 简单集成方可直接看这一字段 |
| `pricing.price_per_*` | 基础单价（USD） | 主计费公式 |
| `pricing.*_multipliers` | 阶梯倍率（分辨率/尺寸/音色/质量） | 阶梯计费 |
| `pricing.input_per_million_tokens` | 已换算好的 $/1M tokens | LLM 直接用，无需 `× 2` |
| `pricing.flat_price_per_request` | Legacy ModelPrice fallback（USD） | 管理员 escape hatch，> 0 时**覆盖**其他字段 |

**关键约定**：所有金额均为 **base USD**，Agent 平台需要根据汇率与积分兑换比例换算成积分。

---

## 四、Agent 平台实现方案

### 4.1 环境变量（新增）

```dotenv
# .env / config
UPSTREAM_PRICING_API_BASE=https://token.35sz.top
UPSTREAM_PRICING_API_KEY=<admin_access_token>
UPSTREAM_PRICING_POLL_INTERVAL_MS=300000     # 5 分钟
UPSTREAM_PRICING_TIMEOUT_MS=10000            # 10 秒
UPSTREAM_PRICING_FALLBACK_MODE=cache         # cache | code_default | fail

# 换算规则
USD_TO_CNY_RATE=7.3                          # 与 35sz-api 保持一致
CREDITS_PER_CNY=10                           # 1 元 = 10 积分
SELL_RATIO_DEFAULT=1.5                       # 默认售价倍率 (成本 × 1.5)
```

### 4.2 `ModelPricingService` (NestJS 示例)

**关键文件**：`packages/api/src/modules/pricing/model-pricing.service.ts` (新建)

```typescript
import { Injectable, Logger, OnModuleInit } from '@nestjs/common';
import { HttpService } from '@nestjs/axios';
import { ConfigService } from '@nestjs/config';
import { firstValueFrom } from 'rxjs';

// -----------------------------------------------------------------------------
// 上游响应类型 —— 与 35sz-api controller/models_pricing.go 1:1 对应
// -----------------------------------------------------------------------------

export type PricingKind =
  | 'chat'
  | 'multimodal-chat'
  | 'image-gen'
  | 'video-gen'
  | 'audio-in'
  | 'audio-out'
  | 'embedding';

export type PricingType =
  | 'token'
  | 'per_image'
  | 'per_second'
  | 'per_minute'
  | 'per_million_chars';

export interface UpstreamPricingBody {
  // chat / multimodal-chat / embedding
  model_ratio?: number;
  completion_ratio?: number;
  cache_ratio?: number;
  image_ratio?: number;
  audio_ratio?: number;
  audio_completion_ratio?: number;
  input_per_million_tokens?: number;
  output_per_million_tokens?: number;
  // image-gen
  price_per_image?: number;
  quality_multipliers?: Record<string, number>;
  size_multipliers?: Record<string, number>;
  // video-gen
  price_per_second?: number;
  resolution_multipliers?: Record<string, number>;
  has_audio_multiplier?: number;
  // audio-in
  price_per_minute?: number;
  min_bill_minutes?: number;
  // audio-out
  price_per_million_chars?: number;
  voice_multipliers?: Record<string, number>;
  // legacy fallback
  flat_price_per_request?: number;
}

export interface UpstreamPricingEntry {
  model_type: 'text' | 'image' | 'video' | 'audio' | 'embedding' | 'file';
  pricing_kind: PricingKind;
  pricing_type: PricingType;
  pricing: UpstreamPricingBody;
}

// -----------------------------------------------------------------------------
// 计费结果 —— Agent 业务代码消费
// -----------------------------------------------------------------------------

export interface CalculatePriceParams {
  modelId: string;
  // 图像
  size?: string;
  quality?: string;
  imageCount?: number;
  // 视频
  resolution?: string;
  duration?: number;
  aspectRatio?: string;
  hasAudio?: boolean;
  // 音频
  voice?: string;
  durationSeconds?: number;
  charCount?: number;
  // LLM
  inputTokens?: number;
  outputTokens?: number;
}

export interface PriceResult {
  /** 上游成本（USD） */
  costUSD: number;
  /** 上游成本（分，向上取整） */
  costCents: number;
  /** 应扣积分（考虑售价倍率） */
  sellCredits: number;
  /** 结构化明细，便于日志/流水记录 */
  breakdown: {
    kind: PricingKind;
    unit: PricingType;
    unitPrice: number;      // 单价 USD
    quantity: number;       // 数量：秒/张/分/token/字符
    multiplier: number;     // 汇总倍率（分辨率×音频×尺寸×质量×音色）
    exchangeRate: number;
    sellRatio: number;
  };
  /** 如果从上游拉取失败或模型未配置，此字段非空 */
  fallbackReason?: 'code_default' | 'cache_stale' | 'not_found';
}

// -----------------------------------------------------------------------------
// Service
// -----------------------------------------------------------------------------

@Injectable()
export class ModelPricingService implements OnModuleInit {
  private readonly logger = new Logger(ModelPricingService.name);
  private cache = new Map<string, UpstreamPricingEntry>();
  private lastUpdatedAt = 0;
  private pricingVersion = '';
  private pollTimer?: NodeJS.Timeout;

  constructor(
    private readonly http: HttpService,
    private readonly config: ConfigService,
  ) {}

  async onModuleInit() {
    await this.refreshFromUpstream();
    const intervalMs = this.config.get<number>(
      'UPSTREAM_PRICING_POLL_INTERVAL_MS',
      5 * 60 * 1000,
    );
    this.pollTimer = setInterval(() => this.refreshFromUpstream(), intervalMs);
  }

  /**
   * 从 35sz-api 拉取最新定价并覆盖内存缓存。失败时保留旧缓存。
   */
  async refreshFromUpstream(): Promise<void> {
    const baseUrl = this.config.get<string>('UPSTREAM_PRICING_API_BASE');
    const apiKey = this.config.get<string>('UPSTREAM_PRICING_API_KEY');
    if (!baseUrl || !apiKey) {
      this.logger.warn('UPSTREAM_PRICING_API_BASE/KEY 未配置，跳过刷新');
      return;
    }
    try {
      const url = `${baseUrl}/v1/models/pricing`;
      const resp = await firstValueFrom(
        this.http.get(url, {
          headers: { Authorization: `Bearer ${apiKey}` },
          timeout: this.config.get<number>('UPSTREAM_PRICING_TIMEOUT_MS', 10000),
        }),
      );
      if (!resp.data?.success) {
        this.logger.warn(`上游返回 success=false, message=${resp.data?.message}`);
        return;
      }
      const nextCache = new Map<string, UpstreamPricingEntry>();
      Object.entries<UpstreamPricingEntry>(resp.data.data ?? {}).forEach(
        ([modelId, entry]) => {
          nextCache.set(modelId, entry);
        },
      );
      // 版本未变则跳过 swap，避免不必要的 GC 压力
      const nextVersion = resp.data.pricing_version ?? '';
      if (nextVersion === this.pricingVersion && this.cache.size > 0) {
        this.lastUpdatedAt = resp.data.updated_at ?? Date.now() / 1000;
        return;
      }
      this.cache = nextCache;
      this.lastUpdatedAt = resp.data.updated_at ?? Date.now() / 1000;
      this.pricingVersion = nextVersion;
      this.logger.log(
        `定价缓存已刷新: ${nextCache.size} 个模型, version=${nextVersion}`,
      );
    } catch (err) {
      this.logger.warn(
        `拉取上游定价失败，沿用旧缓存: ${(err as Error).message}`,
      );
    }
  }

  // ---------------------------------------------------------------------------
  // 查询接口
  // ---------------------------------------------------------------------------

  /**
   * 计算价格。业务侧只需传 modelId 和参数，返回成本 + 应扣积分 + 明细。
   */
  calculate(params: CalculatePriceParams): PriceResult {
    const entry = this.cache.get(params.modelId);
    if (!entry) {
      return this.codeDefault(params, 'not_found');
    }

    // flat_price_per_request 是 admin escape hatch，> 0 时覆盖其他字段
    if (entry.pricing.flat_price_per_request && entry.pricing.flat_price_per_request > 0) {
      return this.buildResult({
        kind: entry.pricing_kind,
        unit: entry.pricing_type,
        unitPrice: entry.pricing.flat_price_per_request,
        quantity: 1,
        multiplier: 1,
      });
    }

    switch (entry.pricing_kind) {
      case 'video-gen':
        return this.calcVideo(entry, params);
      case 'image-gen':
        return this.calcImage(entry, params);
      case 'audio-in':
        return this.calcAudioIn(entry, params);
      case 'audio-out':
        return this.calcAudioOut(entry, params);
      case 'chat':
      case 'multimodal-chat':
      case 'embedding':
      default:
        return this.calcToken(entry, params);
    }
  }

  private calcVideo(
    entry: UpstreamPricingEntry,
    params: CalculatePriceParams,
  ): PriceResult {
    const base = entry.pricing.price_per_second ?? 0;
    const seconds = Math.max(1, params.duration ?? 5);
    const resMul =
      entry.pricing.resolution_multipliers?.[
        (params.resolution ?? '720p').toLowerCase()
      ] ?? 1;
    const audMul =
      params.hasAudio && entry.pricing.has_audio_multiplier
        ? entry.pricing.has_audio_multiplier
        : 1;
    return this.buildResult({
      kind: 'video-gen',
      unit: 'per_second',
      unitPrice: base,
      quantity: seconds,
      multiplier: resMul * audMul,
    });
  }

  private calcImage(
    entry: UpstreamPricingEntry,
    params: CalculatePriceParams,
  ): PriceResult {
    const base = entry.pricing.price_per_image ?? 0;
    const n = Math.max(1, params.imageCount ?? 1);
    const sizeMul = entry.pricing.size_multipliers?.[params.size ?? ''] ?? 1;
    const qMul =
      entry.pricing.quality_multipliers?.[params.quality ?? ''] ?? 1;
    return this.buildResult({
      kind: 'image-gen',
      unit: 'per_image',
      unitPrice: base,
      quantity: n,
      multiplier: sizeMul * qMul,
    });
  }

  private calcAudioIn(
    entry: UpstreamPricingEntry,
    params: CalculatePriceParams,
  ): PriceResult {
    const perMin = entry.pricing.price_per_minute ?? 0;
    const durationSec = params.durationSeconds ?? 0;
    const minBill = entry.pricing.min_bill_minutes ?? 0;
    const minutes = Math.max(minBill, durationSec / 60);
    return this.buildResult({
      kind: 'audio-in',
      unit: 'per_minute',
      unitPrice: perMin,
      quantity: minutes,
      multiplier: 1,
    });
  }

  private calcAudioOut(
    entry: UpstreamPricingEntry,
    params: CalculatePriceParams,
  ): PriceResult {
    const per1M = entry.pricing.price_per_million_chars ?? 0;
    const chars = params.charCount ?? 0;
    const voiceMul =
      entry.pricing.voice_multipliers?.[params.voice ?? ''] ?? 1;
    return this.buildResult({
      kind: 'audio-out',
      unit: 'per_million_chars',
      unitPrice: per1M,
      quantity: chars / 1_000_000,
      multiplier: voiceMul,
    });
  }

  private calcToken(
    entry: UpstreamPricingEntry,
    params: CalculatePriceParams,
  ): PriceResult {
    // 使用 denormalized $/1M tokens 字段，Agent 平台不用知道 "× 2" 约定
    const inputPer1M = entry.pricing.input_per_million_tokens ?? 0;
    const outputPer1M = entry.pricing.output_per_million_tokens ?? 0;
    const inputCost = ((params.inputTokens ?? 0) / 1_000_000) * inputPer1M;
    const outputCost = ((params.outputTokens ?? 0) / 1_000_000) * outputPer1M;
    const totalUnitPrice = inputCost + outputCost;
    return this.buildResult({
      kind: entry.pricing_kind,
      unit: 'token',
      unitPrice: totalUnitPrice,
      quantity: 1,
      multiplier: 1,
    });
  }

  // ---------------------------------------------------------------------------
  // 结果封装 —— USD → CNY → 积分
  // ---------------------------------------------------------------------------

  private buildResult(args: {
    kind: PricingKind;
    unit: PricingType;
    unitPrice: number;
    quantity: number;
    multiplier: number;
  }): PriceResult {
    const costUSD = args.unitPrice * args.quantity * args.multiplier;
    const exchangeRate = this.config.get<number>('USD_TO_CNY_RATE', 7.3);
    const creditsPerCny = this.config.get<number>('CREDITS_PER_CNY', 10);
    const sellRatio = this.config.get<number>('SELL_RATIO_DEFAULT', 1.5);

    const costCny = costUSD * exchangeRate;
    const costCents = Math.ceil(costCny * 100); // 分为单位记录
    const sellCredits = Math.ceil(costCny * sellRatio * creditsPerCny);

    return {
      costUSD,
      costCents,
      sellCredits,
      breakdown: {
        kind: args.kind,
        unit: args.unit,
        unitPrice: args.unitPrice,
        quantity: args.quantity,
        multiplier: args.multiplier,
        exchangeRate,
        sellRatio,
      },
    };
  }

  private codeDefault(
    params: CalculatePriceParams,
    reason: PriceResult['fallbackReason'],
  ): PriceResult {
    // 兜底默认价：所有生成类模型 10 积分（管理员应尽快补齐）
    this.logger.warn(
      `模型 ${params.modelId} 使用代码兜底默认价, reason=${reason}`,
    );
    return {
      costUSD: 0,
      costCents: 0,
      sellCredits: 10,
      breakdown: {
        kind: 'chat',
        unit: 'token',
        unitPrice: 0,
        quantity: 1,
        multiplier: 1,
        exchangeRate: 0,
        sellRatio: 0,
      },
      fallbackReason: reason,
    };
  }

  // ---------------------------------------------------------------------------
  // 观测接口
  // ---------------------------------------------------------------------------

  getStats() {
    return {
      cachedModels: this.cache.size,
      lastUpdatedAt: this.lastUpdatedAt,
      pricingVersion: this.pricingVersion,
    };
  }
}
```

### 4.3 三级优先级策略（`BillingPolicyService`）

**关键文件**：`packages/api/src/modules/pricing/billing-policy.service.ts` (新建)

```typescript
@Injectable()
export class BillingPolicyService {
  constructor(
    private readonly modelPricing: ModelPricingService,
    private readonly modelRepo: ModelRepository,  // 现有 model 表 CRUD
  ) {}

  /**
   * 主入口：给定模型和参数，返回最终应扣积分。
   *
   * 优先级:
   *   1. 管理员手动积分价 (per_use_credits > 0 且 sync_upstream=false)
   *   2. 上游同步价 × 售价倍率
   *   3. 代码兜底 10 积分
   */
  async resolveSellCredits(params: CalculatePriceParams): Promise<PriceResult> {
    const model = await this.modelRepo.findByModelId(params.modelId);

    // ① 手动模式
    if (model && !model.sync_upstream && model.per_use_credits > 0) {
      return {
        costUSD: 0,
        costCents: 0,
        sellCredits: model.per_use_credits,
        breakdown: {
          kind: 'chat',
          unit: 'token',
          unitPrice: 0,
          quantity: 1,
          multiplier: 1,
          exchangeRate: 0,
          sellRatio: 0,
        },
      };
    }

    // ② 自动同步模式（或未指定则默认 auto）
    return this.modelPricing.calculate(params);
  }
}
```

### 4.4 图片工具接入（P0 - 修 BUG）

**关键文件**：`packages/api/src/modules/ai/chat/services/ai-chat-completion.service.ts` 第 1596-1604 行

**改动前**：

```typescript
tools.seedreamImageGeneration = createSeedreamImageGenerationTool(
  imageProviderInstance,
  imageModels,
  { referenceImages },
);
```

**改动后**：

```typescript
tools.seedreamImageGeneration = createSeedreamImageGenerationTool(
  imageProviderInstance,
  imageModels,
  {
    referenceImages,
    onSuccess: async ({ modelId, size, imageCount }) => {
      if (!userId) return;

      const price = await this.billingPolicy.resolveSellCredits({
        modelId,
        size,
        imageCount: imageCount ?? 1,
      });
      if (price.sellCredits <= 0) return;

      try {
        await this.appBillingService.deductUserPower({
          userId,
          amount: price.sellCredits,
          accountType: ACCOUNT_LOG_TYPE.PLUGIN_DEC,
          source: {
            type: ACCOUNT_LOG_SOURCE.PLUGIN,
            source: '图片生成',
          },
          remark: `图片生成消耗（${modelId}, ${size}, ${imageCount ?? 1}张），成本${price.costCents}分`,
          associationNo: conversationId,
          // ① 分账字段 —— 见 4.6 数据库改动
          costAmount: price.costCents,
          upstreamChannel: 'token.35sz.top',
          upstreamModelId: modelId,
          usageData: {
            size,
            imageCount: imageCount ?? 1,
            breakdown: price.breakdown,
          },
        });
      } catch (err) {
        this.logger.warn(
          `图片计费失败: userId=${userId}, credits=${price.sellCredits}, err=${err.message}`,
        );
      }
    },
  },
);
```

**同步修改 tool 定义**：`packages/@buildingai/ai-toolkit/src/tools/seedream-image.tools.ts`

```typescript
export interface SeedreamImageGenerationToolOptions {
  referenceImages?: string[];
  onSuccess?: (event: {
    modelId: string;
    size: string;
    imageCount: number;
  }) => Promise<void>;
}

// 在 tool 内部 execute 成功后调用：
export function createSeedreamImageGenerationTool(
  provider: any,
  models: any[],
  options: SeedreamImageGenerationToolOptions,
) {
  return tool({
    // ...
    execute: async ({ prompt, size, n = 1 }, { toolCallId }) => {
      const result = await provider.generateImage({ prompt, size, n });
      // ↓ 新增
      if (options.onSuccess) {
        try {
          await options.onSuccess({
            modelId: result.model,
            size,
            imageCount: result.images?.length ?? n,
          });
        } catch (err) {
          console.warn(`onSuccess 回调失败`, err);
          // 计费失败不能影响图片返回给用户
        }
      }
      return result;
    },
  });
}
```

### 4.5 视频/音频工具接入（P0）

**视频**：`ai-chat-completion.service.ts` 第 1646-1666 行，已有 onSuccess，只需换实现：

```typescript
onSuccess: async ({ modelId, duration, resolution, aspectRatio, hasAudio }) => {
  if (!userId) return;

  const price = await this.billingPolicy.resolveSellCredits({
    modelId,
    duration,
    resolution,
    aspectRatio,
    hasAudio,
  });
  if (price.sellCredits <= 0) return;

  try {
    await this.appBillingService.deductUserPower({
      userId,
      amount: price.sellCredits,
      accountType: ACCOUNT_LOG_TYPE.PLUGIN_DEC,
      source: { type: ACCOUNT_LOG_SOURCE.PLUGIN, source: '视频生成' },
      remark: `视频生成（${modelId}, ${resolution}, ${duration}s, audio=${hasAudio}），成本${price.costCents}分`,
      associationNo: conversationId,
      costAmount: price.costCents,
      upstreamChannel: 'token.35sz.top',
      upstreamModelId: modelId,
      usageData: {
        duration,
        resolution,
        aspectRatio,
        hasAudio,
        breakdown: price.breakdown,
      },
    });
  } catch (err) {
    this.logger.warn(
      `视频计费失败: userId=${userId}, err=${err.message}`,
    );
  }
},
```

**ASR (audio-in)**：转录工具在 onSuccess 中传入音频时长秒数：

```typescript
onSuccess: async ({ modelId, durationSeconds }) => {
  const price = await this.billingPolicy.resolveSellCredits({
    modelId,
    durationSeconds,
  });
  await this.appBillingService.deductUserPower({
    userId,
    amount: price.sellCredits,
    // ...
  });
},
```

**TTS (audio-out)**：合成工具传入字符数：

```typescript
onSuccess: async ({ modelId, charCount, voice }) => {
  const price = await this.billingPolicy.resolveSellCredits({
    modelId,
    charCount,
    voice,
  });
  // ...
},
```

### 4.6 分账流水字段扩展（P1）

**数据库 migration**：`account_log` 表新增 4 列

```sql
ALTER TABLE account_log ADD COLUMN upstream_channel VARCHAR(50);
ALTER TABLE account_log ADD COLUMN upstream_model_id VARCHAR(100);
ALTER TABLE account_log ADD COLUMN cost_amount INTEGER DEFAULT 0;
ALTER TABLE account_log ADD COLUMN usage_data JSONB;
ALTER TABLE account_log ADD COLUMN billing_status VARCHAR(20) DEFAULT 'pending';

CREATE INDEX idx_account_log_upstream_model ON account_log(upstream_channel, upstream_model_id);
CREATE INDEX idx_account_log_billing_status ON account_log(billing_status);
```

**流水查询**：财务看板 SQL 示例

```sql
-- 按日汇总收入/成本/利润
SELECT
  DATE(created_at) AS day,
  upstream_model_id,
  COUNT(*) AS request_count,
  SUM(amount) AS total_sell_credits,
  SUM(cost_amount) AS total_cost_cents,
  SUM(amount * 10) - SUM(cost_amount) AS profit_cents  -- 假设 1 积分 = 10 分
FROM account_log
WHERE accountType IN ('CHAT_DEC', 'PLUGIN_DEC')
  AND upstream_channel = 'token.35sz.top'
  AND created_at BETWEEN '2026-07-01' AND '2026-07-31'
GROUP BY DATE(created_at), upstream_model_id
ORDER BY day DESC, total_sell_credits DESC;
```

### 4.7 后台"手动/自动"开关（P0）

**位置**：Agent 后台"模型管理 → 编辑模型"页面

**改动前**：

```
消耗积分：[  ] 积分/次
```

**改动后**（一行单选 + 输入框联动）：

```
价格来源：
  ○ 自动同步上游（当前上游价：18 积分/次，售价倍率 1.5x）
  ● 手动设置固定积分价：[10] 积分/次
```

- 选"自动同步"时输入框灰掉，显示 API 拉取换算后的积分
- 选"手动设置"时输入框可编辑，优先级最高
- 现有已配置模型默认选中"手动设置"（完全兼容老配置）

**model 表新增字段**：

```sql
ALTER TABLE model ADD COLUMN sync_upstream BOOLEAN DEFAULT false;
```

---

## 五、落地步骤

### 5.1 P0 · 紧急修复（2 小时）

**目标**：图片扣费 BUG 立刻修，不依赖上游接口拉取

- [ ] 修改 `packages/@buildingai/ai-toolkit/src/tools/seedream-image.tools.ts`：`SeedreamImageGenerationToolOptions` 增加 `onSuccess` 回调，`execute` 成功后调用
- [ ] 修改 `packages/api/src/modules/ai/chat/services/ai-chat-completion.service.ts` 第 1596-1604 行：图片工具注册增加 onSuccess 回调，先用 hardcode `priceMap` 兜底
- [ ] 部署验证：调用图片工具后检查 `account_log` 表有对应扣费记录

### 5.2 P0 · 上游对接（1 天）

**目标**：Agent 平台自动从 35sz-api 拉取定价

- [ ] 新建 `ModelPricingService`（4.2）
- [ ] 新建 `BillingPolicyService`（4.3）
- [ ] 环境变量新增：`UPSTREAM_PRICING_API_BASE / KEY / POLL_INTERVAL_MS / TIMEOUT_MS`
- [ ] 图片/视频/ASR/TTS 四个工具的 onSuccess 改为调用 `BillingPolicyService`
- [ ] 后台模型编辑页加"自动/手动"单选（4.7）
- [ ] 数据库 migration：`model` 表加 `sync_upstream`

### 5.3 P1 · 分账与看板（2 天）

**目标**：财务能对账，运营能看利润

- [ ] `account_log` 表新增分账字段（4.6）
- [ ] `deductUserPower` 接口支持新字段
- [ ] 财务看板 SQL 汇总（4.6 的示例）
- [ ] 定时任务：每日凌晨 2 点拉取 35sz-api 前日调用日志与本地对账（暂缓，等 35sz-api 侧对账接口）

### 5.4 P2 · 后续优化（按需）

- [ ] 上游 5% 差异告警
- [ ] 阶梯定价更多参数（如 fps、种子偏好）
- [ ] 支持多个上游供应商切换

---

## 六、注意事项与坑点

### 6.1 汇率一致性

Agent 侧的 `USD_TO_CNY_RATE` **必须与 35sz-api 侧的 `USDExchangeRate` 保持一致**（默认 7.3）。若不同步会造成：
- 用户看到的积分与实际调用成本不匹配
- 分账时利润计算错误

**推荐做法**：在 Agent 后台加一个"从上游同步汇率"按钮，读取 35sz-api 的 `GET /api/status` 里的 `usd_exchange_rate` 字段。

### 6.2 缓存过期

`ModelPricingService` 缓存 5 分钟。如果 35sz-api 侧管理员刚改了价格，Agent 侧最长会有 5 分钟延迟。

**加速方案（可选）**：
- 35sz-api 在管理员保存定价时主动 POST 通知 Agent 平台 webhook
- Agent 平台加一个"强制刷新定价"管理员按钮

### 6.3 兜底价格

`ModelPricingService.codeDefault` 返回 10 积分。这个值太保守会导致某些模型（如 4K 视频）严重亏损。

**推荐做法**：`code_default` 分模型类型 fallback：
```typescript
const CODE_DEFAULT_BY_KIND: Record<PricingKind, number> = {
  'chat': 5,
  'multimodal-chat': 10,
  'image-gen': 20,
  'video-gen': 100,
  'audio-in': 10,
  'audio-out': 20,
  'embedding': 1,
};
```

### 6.4 失败退款

- **LLM**：流式中断或上游报错不走 onFinish，天然不扣费 ✅
- **图片/视频/音频**：任务失败不触发 onSuccess，天然不扣费 ✅
- **异常情况**：上游扣费成功但下游返回失败（网络断），需要 Agent 平台监听 35sz-api 侧的失败日志异步退款

### 6.5 用户余额预检查

参考 3.4 节，所有工具调用前先检查余额是否足够：

```typescript
const estimated = await billingPolicy.resolveSellCredits({ modelId, ...args });
const balance = await this.appBillingService.getUserPower(userId);
if (balance < estimated.sellCredits) {
  throw HttpErrorFactory.badRequest(
    `积分不足，本次生成需要 ${estimated.sellCredits} 积分，当前余额 ${balance}`,
  );
}
```

---

## 七、验证清单

### 7.1 单元测试

- [ ] `ModelPricingService.calculate` 各 kind 分支
- [ ] `BillingPolicyService.resolveSellCredits` 三级优先级
- [ ] 上游失败时降级到缓存
- [ ] flat_price_per_request 覆盖其他字段

### 7.2 集成测试（用 mock 35sz-api）

- [ ] 用 nock/msw mock `GET /v1/models/pricing`，验证冷启动加载
- [ ] 模拟上游 500 → 使用缓存不影响业务
- [ ] 模拟上游超时 → 降级到 code_default

### 7.3 手动 E2E

- [ ] 用户对话触发 LLM → account_log 有正确 CHAT_DEC 流水
- [ ] 用户触发 seedream 图片生成 → account_log 有 PLUGIN_DEC + cost_amount 非零 + usage_data 完整
- [ ] 用户触发 seedance 视频生成 5s 1080p → 扣费 ≈ 6.301369 × 5 × 1.1087 USD × 7.3 CNY × 1.5 sellRatio × 10 积分 ≈ 405 积分
- [ ] 用户触发 whisper 转录 2min → 扣费 ≈ 0.006 × 2 × 7.3 × 1.5 × 10 ≈ 1.3 积分
- [ ] 35sz-api 侧修改 doubao-seedance 单价 → 5 分钟内 Agent 侧生效
- [ ] 断网测试：35sz-api 不可达 → Agent 沿用缓存，业务不中断

---

## 八、上下游联动清单

### 35sz-api 侧已完成

- [x] PR-1 引入 pricing_kind + 4 张结构化定价表
- [x] PR-2 relay 计费路径接入 admin 视频定价
- [x] PR-3.x 前端 UI 支持结构化定价编辑
- [x] PR-4 CNY 充值对账加固
- [x] PR-5 relay 计费消费结构化定价表
- [x] **PR-6 GET /v1/models/pricing 机器接口** ← 本文档基础

### 35sz-api 侧待办（Agent 平台需要时补做）

- [ ] `GET /api/status` 返回 usd_exchange_rate（**已有**，无需改）
- [ ] Webhook 通知 Agent 平台定价变更（P2，可选）
- [ ] 定期出账接口 `GET /admin/billing/daily-usage`（P2，用于双端对账）

### BuildingAI Agent 平台侧待办

- [ ] **P0**：图片工具 onSuccess 回调（2 小时）
- [ ] **P0**：ModelPricingService + BillingPolicyService（1 天）
- [ ] **P0**：视频/音频工具接入 BillingPolicyService（半天）
- [ ] **P0**：后台"自动/手动"开关（半天）
- [ ] **P1**：account_log 分账字段（半天）
- [ ] **P1**：财务看板 SQL（半天）
- [ ] **P2**：告警与对账（后续）

---

## 九、风险与回滚

### 9.1 潜在风险

| 风险 | 概率 | 影响 | 缓解 |
|---|---|---|---|
| 上游接口挂 | 中 | Agent 扣费停滞 | 5 分钟缓存 + 代码兜底价 |
| 汇率漂移 | 低 | 收入误差 | Agent 后台可强制同步 |
| 计费公式不一致 | 中 | 双端对账差异 | 用 pricing_version 校验，> 0.1% 差异告警 |
| 图片工具兼容性 | 低 | 上线后图片调用失败 | 灰度发布，可回滚 tool 定义 |

### 9.2 回滚流程

- **P0 图片修复回滚**：还原 `seedream-image.tools.ts` + `ai-chat-completion.service.ts` 相关行，图片工具恢复不扣费状态（回到当前 BUG 状态，但业务不中断）
- **ModelPricingService 回滚**：环境变量 `UPSTREAM_PRICING_FALLBACK_MODE=code_default`，跳过所有上游拉取，全部走代码兜底
- **数据库迁移回滚**：新增字段可保留（无害），删除对应索引即可

---

**文档结束**

如有疑问，联系 35sz-api 侧维护者。

> Co-Authored-By: Claude <noreply@anthropic.com>
