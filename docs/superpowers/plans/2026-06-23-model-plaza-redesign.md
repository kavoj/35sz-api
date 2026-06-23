# 模型广场重设计与端点能力化 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 重设计 `/pricing` 模型广场（顶部能力 Tab 主导布局），将模型能力统一锚定到 `model.tags`，新增上下文窗口字段（后端真实值+前端兜底），并在 `/models/metadata` 编辑抽屉补全端点能力模板说明。

**Architecture:** 能力来源唯一化——顶部能力 Tab 与卡片能力 chip 都读 `model.tags` 中的能力标签（chat/completion/vision/image/audio/video/embedding/code/reasoning）；删除原按协议端点的筛选器；厂商/分组/计费类型筛选收进顶部下拉。后端 `Model` 表新增 `context_length` 列经 AutoMigrate 自动建列，`Pricing` 透传，前端缺失时回退 `model-metadata.ts` 推断。

**Tech Stack:** Go 1.22 + GORM v2（SQLite/MySQL/PostgreSQL）；React 19 + TypeScript + Rsbuild + Base UI + Tailwind；i18next；Bun。

**参考规范文档:** `docs/superpowers/specs/2026-06-23-model-plaza-redesign-design.md`

---

## File Structure

新增/修改文件及职责：

**任务 4（端点能力模板说明，纯前端）**
- Modify `web/default/src/features/models/constants.ts` — 新增 `CAPABILITY_ENDPOINT_HINTS`（能力中文名→端点模板 key + 说明）。
- Modify `web/default/src/features/models/components/drawers/model-mutate-drawer.tsx` — Endpoints 区块加「能力→端点」对照说明 + 中文模板说明。

**任务 3（删除端点协议筛选器，纯前端）**
- Modify `web/default/src/features/pricing/hooks/use-filters.ts` — 移除 `endpointTypeFilter` 相关。
- Modify `web/default/src/features/pricing/lib/filters.ts` — 移除 `filterByEndpointType` 调用。
- Modify `web/default/src/features/pricing/components/pricing-sidebar.tsx`、`pricing-toolbar.tsx`、`index.tsx` — 移除端点筛选 UI 与 props。
- Modify `web/default/src/routes/pricing/index.tsx` — 移除 search schema 的 `endpointType`。

**任务 2（上下文窗口，全栈）**
- Modify `model/model_meta.go` — `Model` 加 `ContextLength int`；`Update()` Select 加该列。
- Modify `model/pricing.go` — `Pricing` 加 `ContextLength int`；组装时透传 `meta.ContextLength`。
- Modify `web/default/src/features/models/types.ts` — `Model` 接口 + `modelFormSchema` 加 `context_length`。
- Modify `web/default/src/features/models/components/drawers/model-mutate-drawer.tsx` — 基础信息区加数字输入。
- Modify `web/default/src/features/pricing/components/pricing-columns.tsx`、`model-card.tsx` — 上下文窗口展示替换原标签列/标签位。

**任务 1（模型广场重设计，前端）**
- Create `web/default/src/features/pricing/lib/capabilities.ts` — 能力标签词典 + Tab 归类逻辑。
- Create `web/default/src/features/pricing/components/capability-tabs.tsx` — 顶部能力 Tab 组件。
- Create `web/default/src/features/pricing/components/pricing-hero.tsx` — Hero 横幅组件。
- Modify `web/default/src/features/pricing/hooks/use-filters.ts` — 加 `capabilityTab` 状态 + `filterByCapabilityTab`。
- Modify `web/default/src/features/pricing/lib/filters.ts` — 加 `filterByCapabilityTab`。
- Modify `web/default/src/features/pricing/index.tsx` — 单栏布局 + Hero + Tab + 顶部下拉，移除 `PricingSidebar`。
- Modify `web/default/src/routes/pricing/index.tsx` — search schema 加 `capability`。
- Modify i18n：`web/default/src/i18n/locales/zh.json`、`static-keys.ts`。

---

## 能力标签词典（贯穿任务 1/2/3/4）

| tag | 中文 | 顶部 Tab |
|-----|------|---------|
| chat | 对话 | 文本 |
| completion | 补全 | 文本 |
| reasoning | 推理 | 文本 |
| embedding | 向量 | 文本 |
| code | 代码 | 编码 |
| vision | 视觉 | 多模态 |
| audio | 音频 | 多模态 |
| image | 图像 | 图片 |
| video | 视频 | 视频 |

Tab 值集合：`all` / `text` / `code` / `multimodal` / `image` / `video`。

---

## Task 1: 端点能力模板说明常量（任务 4 基础）

**Files:**
- Modify: `web/default/src/features/models/constants.ts`（在 `ENDPOINT_TEMPLATES` 之后、`TAG_PRESETS` 之前）

- [ ] **Step 1: 新增 `CAPABILITY_ENDPOINT_HINTS` 常量**

在 `web/default/src/features/models/constants.ts` 中 `ENDPOINT_TEMPLATES` 对象闭合 `}` 之后插入：

```typescript
// ============================================================================
// Capability → Endpoint Hints（能力到端点模板的对照说明）
// ============================================================================

export interface CapabilityEndpointHint {
  /** 能力英文 key（用于 i18n） */
  capability: string
  /** 推荐填充的 ENDPOINT_TEMPLATES key 列表 */
  templateKeys: string[]
  /** 一句话说明（i18n key） */
  descriptionKey: string
}

export const CAPABILITY_ENDPOINT_HINTS: CapabilityEndpointHint[] = [
  {
    capability: 'chat',
    templateKeys: ['openai', 'anthropic', 'gemini'],
    descriptionKey: 'Conversational models. Use chat/messages endpoints.',
  },
  {
    capability: 'completion',
    templateKeys: ['openai-completions'],
    descriptionKey: 'Legacy text completion endpoint.',
  },
  {
    capability: 'vision',
    templateKeys: ['openai', 'anthropic', 'gemini'],
    descriptionKey: 'Multimodal input. Image understanding via chat endpoints.',
  },
  {
    capability: 'image',
    templateKeys: ['image-generation'],
    descriptionKey: 'Image generation endpoint.',
  },
  {
    capability: 'audio',
    templateKeys: ['audio-speech', 'audio-transcriptions'],
    descriptionKey: 'Audio speech / transcription endpoints.',
  },
  {
    capability: 'video',
    templateKeys: ['openai'],
    descriptionKey: 'Video generation/understanding endpoints.',
  },
  {
    capability: 'embedding',
    templateKeys: ['embeddings'],
    descriptionKey: 'Text embedding endpoint.',
  },
  {
    capability: 'code',
    templateKeys: ['openai', 'anthropic'],
    descriptionKey: 'Code-focused models via chat endpoints.',
  },
  {
    capability: 'reasoning',
    templateKeys: ['openai', 'anthropic'],
    descriptionKey: 'Reasoning models via chat endpoints.',
  },
]
```

注意：`ENDPOINT_TEMPLATES` 中需存在 `audio-speech`、`audio-transcriptions`、`moderations` key（已存在，见文件 205-219 行）。

- [ ] **Step 2: 类型检查**

Run: `cd web/default && bunx tsc --noEmit 2>&1 | grep -i "constants.ts" || echo "no constants.ts errors"`
Expected: `no constants.ts errors`

- [ ] **Step 3: Commit**

```bash
git add web/default/src/features/models/constants.ts
git commit -m "feat(models): add capability-to-endpoint hints constant"
```

---

## Task 2: 编辑抽屉端点能力对照说明（任务 4 UI）

**Files:**
- Modify: `web/default/src/features/models/components/drawers/model-mutate-drawer.tsx`

- [ ] **Step 1: 引入常量**

修改第 87 行的 import（已含 `getTagLabel, getTagCategoryLabel`），追加 `CAPABILITY_ENDPOINT_HINTS`：

```typescript
import { getNameRuleOptions, ENDPOINT_TEMPLATES, TAG_PRESETS, getTagLabel, getTagCategoryLabel, CAPABILITY_ENDPOINT_HINTS } from '../../constants'
```

- [ ] **Step 2: 在 Endpoints 区块的「配置格式说明」卡片后追加能力对照表**

在 `model-mutate-drawer.tsx` 中找到 Endpoints 区块里 `{t('Configuration Format')}` 所在的说明卡片（含 `<pre>` 的 `div`，约 1116-1132 行），在该 `div` 闭合 `</div>` 之后、`<FormDescription>` 之前插入：

```tsx
                    <div className="mt-2 space-y-2 rounded-lg border border-border bg-card/50 p-3">
                      <p className="text-muted-foreground text-xs font-medium">
                        {t('Capability → Endpoint reference')}
                      </p>
                      <div className="flex flex-col gap-1.5">
                        {CAPABILITY_ENDPOINT_HINTS.map((hint) => (
                          <div
                            key={hint.capability}
                            className="flex items-center justify-between gap-2"
                          >
                            <div className="flex flex-col">
                              <span className="text-foreground text-xs font-medium">
                                {getTagLabel(t, hint.capability)}
                              </span>
                              <span className="text-muted-foreground text-[11px]">
                                {t(hint.descriptionKey)}
                              </span>
                            </div>
                            <div className="flex flex-wrap gap-1">
                              {hint.templateKeys.map((key) => (
                                <button
                                  key={key}
                                  type="button"
                                  onClick={() => handleFillEndpointTemplate(key)}
                                  className="rounded-full border border-border px-2 py-0.5 text-[11px] text-muted-foreground transition-colors hover:border-primary hover:text-primary"
                                >
                                  {key}
                                </button>
                              ))}
                            </div>
                          </div>
                        ))}
                      </div>
                    </div>
```

（`handleFillEndpointTemplate` 已在组件内定义，见约 764 行。）

- [ ] **Step 3: 类型检查与构建**

Run: `cd web/default && bunx tsc --noEmit 2>&1 | grep -i "model-mutate-drawer" || echo "no drawer errors"`
Expected: `no drawer errors`

- [ ] **Step 4: 补 i18n 中文键**

在 `web/default/src/i18n/locales/zh.json` 的能力标签翻译区块（第一轮已加的 `"unfiltered": "未过滤"` 之后）追加：

```json
    "Capability → Endpoint reference": "能力 → 端点对照",
    "Conversational models. Use chat/messages endpoints.": "对话模型，使用 chat/messages 端点。",
    "Legacy text completion endpoint.": "传统文本补全端点。",
    "Multimodal input. Image understanding via chat endpoints.": "多模态输入，通过 chat 端点实现图像理解。",
    "Image generation endpoint.": "图像生成端点。",
    "Audio speech / transcription endpoints.": "音频合成 / 转写端点。",
    "Video generation/understanding endpoints.": "视频生成/理解端点。",
    "Text embedding endpoint.": "文本向量端点。",
    "Code-focused models via chat endpoints.": "面向代码的模型，通过 chat 端点调用。",
    "Reasoning models via chat endpoints.": "推理模型，通过 chat 端点调用。"
```

注意：插入位置需在 zh.json 顶层 `"translation"` 对象内，确保前一行以 `,` 结尾、本块作为已有键值后续项；最后一项 `"Reasoning..."` 后若非对象末尾需补 `,`。

- [ ] **Step 5: 补 static-keys**

在 `web/default/src/i18n/static-keys.ts` 第一轮新增的标签键区块（`'unfiltered',` 之后）追加：

```typescript
  'Capability → Endpoint reference',
  'Conversational models. Use chat/messages endpoints.',
  'Legacy text completion endpoint.',
  'Multimodal input. Image understanding via chat endpoints.',
  'Image generation endpoint.',
  'Audio speech / transcription endpoints.',
  'Video generation/understanding endpoints.',
  'Text embedding endpoint.',
  'Code-focused models via chat endpoints.',
  'Reasoning models via chat endpoints.',
```

- [ ] **Step 6: Commit**

```bash
git add web/default/src/features/models/components/drawers/model-mutate-drawer.tsx web/default/src/i18n/locales/zh.json web/default/src/i18n/static-keys.ts
git commit -m "feat(models): add capability-endpoint reference to endpoint config"
```

---

## Task 3: 删除 pricing 端点协议筛选器（前端）

**Files:**
- Modify: `web/default/src/features/pricing/hooks/use-filters.ts`
- Modify: `web/default/src/features/pricing/lib/filters.ts`
- Modify: `web/default/src/features/pricing/components/pricing-toolbar.tsx`
- Modify: `web/default/src/features/pricing/components/pricing-sidebar.tsx`
- Modify: `web/default/src/features/pricing/index.tsx`
- Modify: `web/default/src/routes/pricing/index.tsx`

> 说明：`pricing-sidebar.tsx` 在任务 6（布局重构）中会整体移除使用，但本任务先把它内部对 endpoint 的引用清掉以保证类型通过；若任务顺序中 sidebar 已删可跳过该文件。

- [ ] **Step 1: `lib/filters.ts` 移除 `filterByEndpointType`**

删除 `web/default/src/features/pricing/lib/filters.ts` 中的 `filterByEndpointType` 函数（约 88-99 行）与 `filterAndSortModels` 内的调用行 `result = filterByEndpointType(result, filters.endpointType)`（约 153 行），并从 `filters` 参数类型中删除 `endpointType: string`。同时删除顶部 import 中的 `ENDPOINT_TYPES`（若仅此处用）。

修改后 `filterAndSortModels` 签名与体：

```typescript
export function filterAndSortModels(
  models: PricingModel[],
  filters: {
    search: string
    vendor: string
    group: string
    quotaType: string
    tag: string
    sortBy: string
  }
): PricingModel[] {
  let result = filterBySearch(models, filters.search)
  result = filterByVendor(result, filters.vendor)
  result = filterByGroup(result, filters.group)
  result = filterByQuotaType(result, filters.quotaType)
  result = filterByTag(result, filters.tag)
  result = sortModels(result, filters.sortBy)

  return result
}
```

- [ ] **Step 2: `use-filters.ts` 移除 endpoint 相关**

在 `web/default/src/features/pricing/hooks/use-filters.ts`：
- 删除 import 中的 `ENDPOINT_TYPES`（第 25 行）。
- 删除 `FilterState` 的 `endpointType?: string`（第 40 行）。
- 删除 `filterState` 初始化里的 `endpointType: search.endpointType`（第 60 行）。
- 删除 `endpointTypeFilter` 派生（第 73 行）。
- 删除 `setEndpointTypeFilter`（第 114-120 行）。
- 在 `filterAndSortModels(...)` 调用里删 `endpointType: endpointTypeFilter`（第 153 行）及 deps（第 162 行）。
- `hasActiveFilters` / `activeFilterCount` / `clearFilters` 中删除 `endpointTypeFilter !== ENDPOINT_TYPES.ALL` 与 `endpointType: undefined`（第 173、183、193 行）。
- 返回对象删 `endpointTypeFilter`、`setEndpointTypeFilter`（第 207、217 行）。

- [ ] **Step 3: `routes/pricing/index.tsx` 移除 schema 字段**

删除 `web/default/src/routes/pricing/index.tsx` 第 31 行 `endpointType: z.string().optional(),`。

- [ ] **Step 4: `index.tsx` 移除 props 传递**

在 `web/default/src/features/pricing/index.tsx`：
- 删除解构里的 `endpointTypeFilter`、`setEndpointTypeFilter`（第 61、71 行）。
- 删除 `<PricingSidebar>` 与 `<PricingToolbar>` 上的 `endpointTypeFilter={...}` 与 `onEndpointTypeChange={...}` 属性（第 207-208、239-240 行附近）。

- [ ] **Step 5: `pricing-toolbar.tsx` 与 `pricing-sidebar.tsx` 移除端点 UI**

- `pricing-toolbar.tsx`：删除 props 类型中的 `endpointTypeFilter`、`onEndpointTypeChange`（第 76 行及对应），删除传给 sidebar 的 `endpointTypeFilter={props.endpointTypeFilter}`（第 288 行）及对应 `onEndpointTypeChange`。
- `pricing-sidebar.tsx`：删除 import 的 `ENDPOINT_TYPES`、`getEndpointTypeLabels`（第 32、35 行），删除 props 的 `endpointTypeFilter`、`onEndpointTypeChange`，删除 `endpointTypeLabels`（第 160 行）、`endpointOptions`（第 228-244 行）与渲染该 FilterSection 的块（含 `title={t('Endpoint Type')}` 的 `<FilterSection>`）。

- [ ] **Step 6: `constants.ts` 清理（确认无其他消费方后）**

Run: `cd web/default && grep -rn "ENDPOINT_TYPES\|getEndpointTypeLabels\|EndpointTypeOption" src/features/pricing/ | grep -v "\.test\."`
Expected: 仅 `constants.ts` 自身出现（其他消费方已删）。
若确认，删除 `web/default/src/features/pricing/constants.ts` 中 `ENDPOINT_TYPES`、`EndpointTypeOption`、`getEndpointTypeLabels`（第 66-97 行）及 `FILTER_SECTIONS.ENDPOINT_TYPE`（第 102 行）。
注意：`model-details-api.tsx` 用的是字符串字面量 `'openai-response'` 等，不依赖 `ENDPOINT_TYPES`，无需改。

- [ ] **Step 7: 类型检查**

Run: `cd web/default && bunx tsc --noEmit 2>&1 | grep -iE "pricing|use-filters|filters.ts" || echo "no pricing errors"`
Expected: `no pricing errors`

- [ ] **Step 8: Commit**

```bash
git add web/default/src/features/pricing/ web/default/src/routes/pricing/index.tsx
git commit -m "refactor(pricing): remove endpoint-type protocol filter"
```

---

## Task 4: 后端新增 context_length 字段

**Files:**
- Modify: `model/model_meta.go`
- Modify: `model/pricing.go`

- [ ] **Step 1: `Model` struct 加字段**

在 `model/model_meta.go` 的 `Model` struct 中，`NameRule` 字段（第 68 行）之后追加：

```go
	ContextLength int `json:"context_length,omitempty" gorm:"default:0"`
```

- [ ] **Step 2: `Update()` Select 列表加列**

在 `model/model_meta.go` 的 `Update()` 方法（第 106-113 行），把 `Select(...)` 调用中追加 `"context_length"`：

```go
	return DB.Model(&Model{}).Where("id = ?", mi.Id).
		Select("model_name", "description", "icon", "tags", "model_type", "vendor_id", "endpoints", "status", "sync_official", "name_rule", "context_length", "updated_time").
		Updates(mi).Error
```

- [ ] **Step 3: `Pricing` struct 加字段**

在 `model/pricing.go` 的 `Pricing` struct（第 18-39 行），`PricingVersion` 之后追加：

```go
	ContextLength int `json:"context_length,omitempty"`
```

- [ ] **Step 4: 组装 pricing 时透传**

在 `model/pricing.go` 中，给 pricing 补充元数据的块（第 297-306 行 `if meta, ok := metaMap[model]; ok {` 内），在 `pricing.VendorID = meta.VendorID` 之后追加：

```go
				pricing.ContextLength = meta.ContextLength
```

- [ ] **Step 5: 编译验证**

Run: `cd /Users/perjac/WorkF/tokenJ/35sz-api && go build ./model/... 2>&1 | head`
Expected: 无输出（编译通过）。

- [ ] **Step 6: 模型包测试**

Run: `go test ./model/... 2>&1 | tail -5`
Expected: `ok` 或 `PASS`（无新增失败）。

- [ ] **Step 7: Commit**

```bash
git add model/model_meta.go model/pricing.go
git commit -m "feat(model): add context_length field to model meta and pricing"
```

---

## Task 5: 前端 context_length 表单与类型

**Files:**
- Modify: `web/default/src/features/models/types.ts`
- Modify: `web/default/src/features/models/components/drawers/model-mutate-drawer.tsx`

- [ ] **Step 1: `Model` 接口加字段**

在 `web/default/src/features/models/types.ts` 的 `Model` 接口（第 38-58 行），`name_rule: number` 之后追加：

```typescript
  context_length?: number
```

- [ ] **Step 2: `modelFormSchema` 加字段**

在同文件 `modelFormSchema`（第 236-247 行），`sync_official` 之后追加：

```typescript
  context_length: z.number().optional(),
```

- [ ] **Step 3: 抽屉 `extendedModelFormSchema` 加字段**

在 `model-mutate-drawer.tsx` 的 `extendedModelFormSchema`（第 100-118 行），`sync_official: z.boolean(),` 之后追加：

```typescript
  context_length: z.number().optional(),
```

- [ ] **Step 4: 表单 defaultValues 与 reset 加字段**

在 `model-mutate-drawer.tsx`：
- `useForm` 的 `defaultValues`（第 232-249 行）追加 `context_length: undefined,`。
- 编辑加载 `baseModelData`（第 406-424 行）追加 `context_length: model.context_length,`。
- 新建 reset（第 511-528 行）追加 `context_length: undefined,`。

- [ ] **Step 5: 基础信息区加数字输入（紧随 Tags 字段后）**

在 `model-mutate-drawer.tsx` Tags 的 `FormField`（`name="tags"`）闭合后、`</SideDrawerSection>`（基础信息区结束，约 1013 行）之前插入：

```tsx
              <FormField
                control={form.control}
                name="context_length"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Context window')}</FormLabel>
                    <FormControl>
                      <Input
                        type="number"
                        placeholder="128000"
                        value={field.value ?? ''}
                        onChange={(e) => {
                          const v = e.target.value
                          field.onChange(v === '' ? undefined : Number(v))
                        }}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('Maximum context tokens supported by this model (e.g. 128000).')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
```

- [ ] **Step 6: 提交 payload 带上 context_length**

在 `model-mutate-drawer.tsx` 的 `onSubmit` 内 `submitData` 组装（第 543-549 行），确认 `...values` 已含 `context_length`（因 schema 已加，无需额外改）；但 `modelData` 解构（第 552-561 行）会把 ratio 字段剔除，`context_length` 会保留在 `...modelData` 中随 create/update 提交。无需改动，确认即可。

- [ ] **Step 7: i18n 中文键**

在 `web/default/src/i18n/locales/zh.json` 能力标签区块追加：

```json
    "Context window": "上下文窗口",
    "Maximum context tokens supported by this model (e.g. 128000).": "该模型支持的最大上下文 token 数（如 128000）。"
```

在 `web/default/src/i18n/static-keys.ts` 追加：

```typescript
  'Context window',
  'Maximum context tokens supported by this model (e.g. 128000).',
```

- [ ] **Step 8: 类型检查**

Run: `cd web/default && bunx tsc --noEmit 2>&1 | grep -iE "model-mutate-drawer|types.ts" || echo "no errors"`
Expected: `no errors`

- [ ] **Step 9: Commit**

```bash
git add web/default/src/features/models/types.ts web/default/src/features/models/components/drawers/model-mutate-drawer.tsx web/default/src/i18n/locales/zh.json web/default/src/i18n/static-keys.ts
git commit -m "feat(models): add context window input to model edit drawer"
```

---

## Task 6: pricing 展示上下文窗口（替换标签列）

**Files:**
- Modify: `web/default/src/features/pricing/components/pricing-columns.tsx`
- Modify: `web/default/src/features/pricing/components/model-card.tsx`

- [ ] **Step 1: 表格标签列改为上下文窗口列**

在 `web/default/src/features/pricing/components/pricing-columns.tsx`：
- 顶部 import 追加：

```typescript
import { formatTokenCount, inferModelMetadata } from '../lib/model-metadata'
```

- 找到第一轮已改的 Tags 列（`accessorKey: 'tags'`，含 `parseTags` 与 `renderLimitedTags`），整列替换为：

```tsx
    // Context window column
    {
      id: 'context_length',
      meta: { label: t('Context') },
      header: t('Context'),
      cell: ({ row }) => {
        const model = row.original
        const ctx =
          model.context_length && model.context_length > 0
            ? model.context_length
            : inferModelMetadata(model).context_length
        return (
          <span className="font-mono text-sm tabular-nums">
            {formatTokenCount(ctx)}
          </span>
        )
      },
      size: 110,
      enableSorting: false,
    },
```

若 `parseTags`、`renderLimitedTags` 在删除 Tags 列后无其他引用，删除其 import 与定义以免 lint 报未使用。

Run: `cd web/default && grep -n "parseTags\|renderLimitedTags" src/features/pricing/components/pricing-columns.tsx`
Expected: 无残留引用（如有则一并删除定义）。

- [ ] **Step 2: 卡片底部标签位改为上下文窗口**

在 `web/default/src/features/pricing/components/model-card.tsx`：
- 顶部 import 追加：

```typescript
import { formatTokenCount, inferModelMetadata } from '../lib/model-metadata'
```

- 第一轮已加 `const translatedTags = tags.map(tag => t(tag))`；在其后追加：

```typescript
  const contextLength =
    props.model.context_length && props.model.context_length > 0
      ? props.model.context_length
      : inferModelMetadata(props.model).context_length
```

- 找到 `bottomTags`（第一轮改为 `[...endpoints.slice(0, 2), ...translatedTags.slice(0, 2)]`），保留能力 chip，但在底部信息行追加上下文窗口展示。定位卡片底部 `bottomTags.map(...)` 渲染块（约 252-257 行），在该 `{bottomTags.map(...)}` 之后追加：

```tsx
          <span className="text-muted-foreground/70 text-xs">
            {t('Context')} {formatTokenCount(contextLength)}
          </span>
```

- [ ] **Step 3: i18n 中文键**

`web/default/src/i18n/locales/zh.json` 追加 `"Context": "上下文"`；`static-keys.ts` 追加 `'Context',`。

- [ ] **Step 4: 类型检查**

Run: `cd web/default && bunx tsc --noEmit 2>&1 | grep -iE "pricing-columns|model-card" || echo "no errors"`
Expected: `no errors`

- [ ] **Step 5: Commit**

```bash
git add web/default/src/features/pricing/components/pricing-columns.tsx web/default/src/features/pricing/components/model-card.tsx web/default/src/i18n/locales/zh.json web/default/src/i18n/static-keys.ts
git commit -m "feat(pricing): show context window instead of tags column"
```

---

## Task 7: 能力词典与 Tab 归类逻辑

**Files:**
- Create: `web/default/src/features/pricing/lib/capabilities.ts`
- Test: `web/default/src/features/pricing/lib/capabilities.test.ts`

- [ ] **Step 1: 写失败测试**

创建 `web/default/src/features/pricing/lib/capabilities.test.ts`（使用项目既有的 `node:test` 风格，非 vitest）：

```typescript
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { matchesCapabilityTab, CAPABILITY_TABS } from './capabilities'
import type { PricingModel } from '../types'

function model(tags: string): PricingModel {
  return {
    id: 1,
    model_name: 'm',
    quota_type: 0,
    model_ratio: 1,
    completion_ratio: 1,
    enable_groups: [],
    tags,
  } as PricingModel
}

describe('matchesCapabilityTab', () => {
  test('all tab matches everything', () => {
    assert.equal(matchesCapabilityTab(model(''), 'all'), true)
  })
  test('text tab matches chat tag', () => {
    assert.equal(matchesCapabilityTab(model('chat'), 'text'), true)
  })
  test('code tab matches code tag', () => {
    assert.equal(matchesCapabilityTab(model('code,chat'), 'code'), true)
  })
  test('multimodal tab matches vision tag', () => {
    assert.equal(matchesCapabilityTab(model('vision'), 'multimodal'), true)
  })
  test('image tab matches image tag', () => {
    assert.equal(matchesCapabilityTab(model('image'), 'image'), true)
  })
  test('video tab matches video tag', () => {
    assert.equal(matchesCapabilityTab(model('video'), 'video'), true)
  })
  test('text tab does not match pure image model', () => {
    assert.equal(matchesCapabilityTab(model('image'), 'text'), false)
  })
  test('exposes 6 tabs', () => {
    assert.deepEqual(
      CAPABILITY_TABS.map((t) => t.value),
      ['all', 'text', 'code', 'multimodal', 'image', 'video']
    )
  })
})
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd web/default && node --import tsx --test src/features/pricing/lib/capabilities.test.ts 2>&1 | tail -15`
Expected: FAIL（模块不存在 / 函数未定义）。

- [ ] **Step 3: 实现 `capabilities.ts`**

创建 `web/default/src/features/pricing/lib/capabilities.ts`（版权头复用项目其他文件头部）：

```typescript
/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { parseTags } from './filters'
import type { PricingModel } from '../types'

export type CapabilityTabValue =
  | 'all'
  | 'text'
  | 'code'
  | 'multimodal'
  | 'image'
  | 'video'

export interface CapabilityTab {
  value: CapabilityTabValue
  /** i18n label key */
  labelKey: string
}

export const CAPABILITY_TABS: CapabilityTab[] = [
  { value: 'all', labelKey: 'All' },
  { value: 'text', labelKey: 'Text' },
  { value: 'code', labelKey: 'Code' },
  { value: 'multimodal', labelKey: 'Multimodal' },
  { value: 'image', labelKey: 'Image' },
  { value: 'video', labelKey: 'Video' },
]

/** tag → 所属 Tab 集合 */
const TAB_TAGS: Record<Exclude<CapabilityTabValue, 'all'>, string[]> = {
  text: ['chat', 'completion', 'reasoning', 'embedding'],
  code: ['code'],
  multimodal: ['vision', 'audio'],
  image: ['image'],
  video: ['video'],
}

export function matchesCapabilityTab(
  model: PricingModel,
  tab: CapabilityTabValue
): boolean {
  if (tab === 'all') return true
  const tags = parseTags(model.tags).map((t) => t.toLowerCase())
  return TAB_TAGS[tab].some((t) => tags.includes(t))
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd web/default && bunx vitest run src/features/pricing/lib/capabilities.test.ts 2>&1 | tail -15`
Expected: PASS（8 个测试通过）。

- [ ] **Step 5: Commit**

```bash
git add web/default/src/features/pricing/lib/capabilities.ts web/default/src/features/pricing/lib/capabilities.test.ts
git commit -m "feat(pricing): add capability tab dictionary and matcher"
```

---

## Task 8: useFilters 集成 capabilityTab

**Files:**
- Modify: `web/default/src/features/pricing/hooks/use-filters.ts`
- Modify: `web/default/src/features/pricing/lib/filters.ts`
- Modify: `web/default/src/routes/pricing/index.tsx`

- [ ] **Step 1: `filters.ts` 加 capability 筛选**

在 `web/default/src/features/pricing/lib/filters.ts` 顶部 import 追加：

```typescript
import { matchesCapabilityTab, type CapabilityTabValue } from './capabilities'
```

在 `filterAndSortModels` 的 `filters` 类型加 `capability: CapabilityTabValue`，并在 `filterBySearch` 之前加一行筛选：

```typescript
export function filterAndSortModels(
  models: PricingModel[],
  filters: {
    search: string
    vendor: string
    group: string
    quotaType: string
    tag: string
    capability: CapabilityTabValue
    sortBy: string
  }
): PricingModel[] {
  let result = models.filter((m) => matchesCapabilityTab(m, filters.capability))
  result = filterBySearch(result, filters.search)
  result = filterByVendor(result, filters.vendor)
  result = filterByGroup(result, filters.group)
  result = filterByQuotaType(result, filters.quotaType)
  result = filterByTag(result, filters.tag)
  result = sortModels(result, filters.sortBy)

  return result
}
```

- [ ] **Step 2: `use-filters.ts` 加 capabilityTab 状态**

在 `web/default/src/features/pricing/hooks/use-filters.ts`：
- import 追加：`import type { CapabilityTabValue } from '../lib/capabilities'`
- `FilterState` 加 `capability?: CapabilityTabValue`
- `filterState` 初始化加 `capability: search.capability as CapabilityTabValue | undefined`
- 派生：`const capabilityTab: CapabilityTabValue = filterState.capability || 'all'`
- setter：

```typescript
  const setCapabilityTab = useCallback(
    (v: CapabilityTabValue) =>
      updateFilters({ capability: v === 'all' ? undefined : v }),
    [updateFilters]
  )
```

- `filteredModels` 的 `filterAndSortModels({...})` 调用加 `capability: capabilityTab,`，deps 数组加 `capabilityTab`。
- 返回对象加 `capabilityTab`、`setCapabilityTab`。

- [ ] **Step 3: route schema 加 capability**

在 `web/default/src/routes/pricing/index.tsx` 的 `pricingSearchSchema` 加：

```typescript
  capability: z.enum(['all', 'text', 'code', 'multimodal', 'image', 'video']).optional(),
```

- [ ] **Step 4: 类型检查**

Run: `cd web/default && bunx tsc --noEmit 2>&1 | grep -iE "use-filters|filters.ts|routes/pricing" || echo "no errors"`
Expected: `no errors`

- [ ] **Step 5: Commit**

```bash
git add web/default/src/features/pricing/hooks/use-filters.ts web/default/src/features/pricing/lib/filters.ts web/default/src/routes/pricing/index.tsx
git commit -m "feat(pricing): wire capability tab into filters"
```

---

## Task 9: Hero 横幅与能力 Tab 组件

**Files:**
- Create: `web/default/src/features/pricing/components/pricing-hero.tsx`
- Create: `web/default/src/features/pricing/components/capability-tabs.tsx`
- Modify: `web/default/src/features/pricing/components/index.ts`

- [ ] **Step 1: 创建 `pricing-hero.tsx`**

创建 `web/default/src/features/pricing/components/pricing-hero.tsx`（含版权头，同 Task 7 头部）：

```tsx
// (版权头同其他文件)
import { useTranslation } from 'react-i18next'

export interface PricingHeroProps {
  modelCount: number
  vendorCount: number
}

export function PricingHero(props: PricingHeroProps) {
  const { t } = useTranslation()
  const stats = [
    { value: `${props.modelCount}`, label: t('Models') },
    { value: `${props.vendorCount}`, label: t('Vendors') },
    { value: '99%', label: t('Service availability') },
  ]
  return (
    <header className="mx-auto mb-8 max-w-3xl pt-6 text-center sm:pt-10">
      <p className="text-muted-foreground mb-3 text-xs font-medium tracking-[0.3em] uppercase">
        MODEL PLAZA
      </p>
      <h1 className="text-[clamp(2rem,5.5vw,3.5rem)] leading-[1.15] font-bold tracking-tight">
        {t('Model Square')}
      </h1>
      <p className="text-muted-foreground/80 mt-3 text-sm sm:text-base">
        {t('One gateway to access global AI models.')}
      </p>
      <div className="mt-6 flex items-center justify-center gap-8">
        {stats.map((s) => (
          <div key={s.label} className="flex flex-col">
            <span className="text-foreground text-2xl font-bold sm:text-3xl">
              {s.value}
            </span>
            <span className="text-muted-foreground text-xs">{s.label}</span>
          </div>
        ))}
      </div>
    </header>
  )
}
```

- [ ] **Step 2: 创建 `capability-tabs.tsx`**

创建 `web/default/src/features/pricing/components/capability-tabs.tsx`（含版权头）：

```tsx
// (版权头同其他文件)
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import {
  CAPABILITY_TABS,
  type CapabilityTabValue,
} from '../lib/capabilities'

export interface CapabilityTabsProps {
  value: CapabilityTabValue
  onChange: (value: CapabilityTabValue) => void
  counts?: Record<CapabilityTabValue, number>
}

export function CapabilityTabs(props: CapabilityTabsProps) {
  const { t } = useTranslation()
  return (
    <div className="flex flex-wrap items-center gap-2">
      {CAPABILITY_TABS.map((tab) => {
        const active = props.value === tab.value
        const count = props.counts?.[tab.value]
        return (
          <button
            key={tab.value}
            type="button"
            onClick={() => props.onChange(tab.value)}
            className={cn(
              'inline-flex items-center gap-1.5 rounded-full border px-3.5 py-1.5 text-sm font-medium transition-all',
              active
                ? 'border-foreground/30 bg-foreground/5 text-foreground shadow-sm'
                : 'border-border/70 bg-background text-muted-foreground hover:border-border hover:text-foreground'
            )}
          >
            {t(tab.labelKey)}
            {count != null && (
              <span
                className={cn(
                  'rounded-md px-1.5 py-0.5 text-[10px]',
                  active
                    ? 'bg-background text-foreground'
                    : 'bg-muted text-muted-foreground'
                )}
              >
                {count}
              </span>
            )}
          </button>
        )
      })}
    </div>
  )
}
```

- [ ] **Step 3: 导出组件**

在 `web/default/src/features/pricing/components/index.ts` 追加：

```typescript
export { PricingHero } from './pricing-hero'
export { CapabilityTabs } from './capability-tabs'
```

- [ ] **Step 4: i18n 中文键**

`web/default/src/i18n/locales/zh.json` 追加：

```json
    "Text": "文本",
    "Code": "编码",
    "Multimodal": "多模态",
    "One gateway to access global AI models.": "一处接入全球 AI 模型。",
    "Vendors": "厂商",
    "Service availability": "服务可用性"
```

（`All`、`Image`、`Video`、`Models`、`Model Square` 已存在；`Code` 注意与已有键不冲突——若 zh.json 已有 `"Code"` 键则跳过该行。）

`static-keys.ts` 追加：

```typescript
  'Text',
  'Code',
  'Multimodal',
  'One gateway to access global AI models.',
  'Vendors',
  'Service availability',
```

- [ ] **Step 5: 类型检查**

Run: `cd web/default && bunx tsc --noEmit 2>&1 | grep -iE "pricing-hero|capability-tabs" || echo "no errors"`
Expected: `no errors`

- [ ] **Step 6: Commit**

```bash
git add web/default/src/features/pricing/components/pricing-hero.tsx web/default/src/features/pricing/components/capability-tabs.tsx web/default/src/features/pricing/components/index.ts web/default/src/i18n/locales/zh.json web/default/src/i18n/static-keys.ts
git commit -m "feat(pricing): add hero banner and capability tabs components"
```

---

## Task 10: 重构 pricing 页面布局（单栏 + Hero + Tab + 顶部下拉）

**Files:**
- Modify: `web/default/src/features/pricing/index.tsx`

- [ ] **Step 1: 引入新组件与 capabilityTab**

在 `web/default/src/features/pricing/index.tsx`：
- 从 `./components` 引入追加 `PricingHero`、`CapabilityTabs`，移除 `PricingSidebar`（不再使用）。
- 从 `useFilters` 解构追加 `capabilityTab`、`setCapabilityTab`。

- [ ] **Step 2: 用 Hero 替换原 header，加能力 Tab，移除侧边栏两栏布局**

把 `index.tsx` 中 `<header className='mx-auto mb-5 ...'>...</header>`（含 `Models Directory`/`Model Square`/`SearchBar` 的整块，约 176-202 行）替换为：

```tsx
          <PricingHero
            modelCount={models?.length || 0}
            vendorCount={vendors?.length || 0}
          />
          <SearchBar
            value={searchInput}
            onChange={setSearchInput}
            onClear={clearSearch}
            placeholder={t('Search model name, provider, endpoint, or tag...')}
            className='mx-auto mb-4 max-w-2xl'
          />
          <div className='mb-4'>
            <CapabilityTabs
              value={capabilityTab}
              onChange={setCapabilityTab}
            />
          </div>
```

把两栏 `<div className='grid gap-4 xl:grid-cols-[330px_minmax(0,1fr)]'>`（约 204 行）及其内的 `<PricingSidebar .../>` 块整体替换为单栏：保留其中的 `<main className='min-w-0 space-y-4'>...</main>`（含 `PricingToolbar` 与 `renderPricingContent()`），去掉外层 grid 与 sidebar。即：

```tsx
          <main className='min-w-0 space-y-4'>
            <PricingToolbar
              filteredCount={filteredModels.length}
              totalCount={models?.length}
              sortBy={sortBy}
              onSortChange={setSortBy}
              tokenUnit={tokenUnit}
              onTokenUnitChange={setTokenUnit}
              showRechargePrice={showRechargePrice}
              onRechargePriceChange={setShowRechargePrice}
              viewMode={viewMode}
              onViewModeChange={setViewMode}
              quotaTypeFilter={quotaTypeFilter}
              vendorFilter={vendorFilter}
              groupFilter={groupFilter}
              tagFilter={tagFilter}
              onQuotaTypeChange={setQuotaTypeFilter}
              onVendorChange={setVendorFilter}
              onGroupChange={setGroupFilter}
              onTagChange={setTagFilter}
              vendors={vendors || []}
              groups={availableGroups}
              groupRatios={groupRatio}
              tags={availableTags}
              models={models || []}
              hasActiveFilters={hasActiveFilters}
              activeFilterCount={activeFilterCount}
              onClearFilters={clearFilters}
            />

            {renderPricingContent()}
          </main>
```

（`PricingToolbar` 已在任务 3 移除 `endpointType` 相关 props，本处不再传。`PricingToolbar` 内部已含厂商/分组/计费类型的下拉触发，满足「收进顶部下拉」需求。）

- [ ] **Step 3: 确认 PricingSidebar 引用已清除**

Run: `cd web/default && grep -n "PricingSidebar" src/features/pricing/index.tsx`
Expected: 无输出。

- [ ] **Step 4: 类型检查 + 构建**

Run: `cd web/default && bunx tsc --noEmit 2>&1 | grep -iE "pricing/index" || echo "no errors"`
Expected: `no errors`

Run: `cd web/default && bun run build 2>&1 | tail -8`
Expected: 构建成功（无 error）。

- [ ] **Step 5: Commit**

```bash
git add web/default/src/features/pricing/index.tsx
git commit -m "feat(pricing): redesign model plaza with hero and capability tabs"
```

---

## Task 11: i18n 同步与端到端验证

**Files:**
- 验证为主，必要时补 `zh.json` / `static-keys.ts`

- [ ] **Step 1: i18n 同步检查**

Run: `cd web/default && bun run i18n:sync 2>&1 | tail -15`
Expected: 无缺失键报错（如报缺失，按提示补 zh.json）。

- [ ] **Step 2: 完整前端构建**

Run: `cd web/default && bun run build 2>&1 | tail -8`
Expected: 构建成功。

- [ ] **Step 3: 后端完整构建**

Run: `cd /Users/perjac/WorkF/tokenJ/35sz-api && go build ./... 2>&1 | head`
Expected: 无输出。

- [ ] **Step 4: 浏览器手动验证（用 run/verify 技能或 playwright）**

验证清单：
1. `/models/metadata` 编辑模型：标签栏中文显示；端点区出现「能力→端点对照」可点击填充；「上下文窗口」数字输入可保存。
2. 保存带 context_length 的模型后，`/api/pricing` 返回该值（`curl` 或网络面板）。
3. `/pricing`：Hero 横幅（MODEL PLAZA + 模型/厂商/99% 统计）；能力 Tab（全部/文本/编码/多模态/图片/视频）切换正确过滤；卡片显示中文能力 chip + 上下文窗口；无端点协议筛选器；厂商/分组/计费类型在顶部 toolbar 下拉可用。

- [ ] **Step 5: 最终提交（若有补漏）**

```bash
git add -A
git commit -m "chore: i18n sync and final adjustments for model plaza redesign"
```

---

## Self-Review Notes

- **Spec 覆盖**：任务 4（端点说明）→ Task 1-2；任务 3（删端点筛选）→ Task 3；任务 2（上下文窗口）→ Task 4-6；任务 1（广场重设计）→ Task 7-10；i18n/验证 → Task 11。全部覆盖。
- **类型一致性**：`CapabilityTabValue`、`matchesCapabilityTab`、`CAPABILITY_TABS` 在 Task 7 定义，Task 8/9 一致引用；`context_length`（蛇形）贯穿前后端 JSON，前端 `Model.context_length` 与 `PricingModel.context_length` 一致。
- **第一轮已完成项**：标签中文翻译、tags 中文化展示已在历史提交中；本计划在其基础上叠加，注意 i18n 键去重（zh.json 中 `All/Image/Video/Models/Model Square/Code` 可能已存在，插入前 grep 确认）。
