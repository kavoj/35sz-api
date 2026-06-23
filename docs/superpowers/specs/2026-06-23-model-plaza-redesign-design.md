# 模型广场重设计与端点能力化 — 设计文档

日期：2026-06-23
分支：newpay

## 背景

参考 https://aijiniu.com/model.html 的模型分类与视觉风格，对 `/pricing` 模型广场页面进行重设计，并统一「端点类型」与「模型能力」的表述。本设计覆盖四项需求：

1. **美化模型广场**：全面对齐参考页布局（Hero 横幅 + 顶部能力 Tab + 简洁卡片）。
2. **标签列改为显示上下文窗口**：用后端真实值 + 前端推断兜底。
3. **端点类型改名为模型能力**：筛选器展示层改名并映射中文。
4. **端点配置增加说明 + 能力模板**：在 `/models/metadata` 编辑抽屉补全模型能力对应的端点模板与说明。

## 关键决策（已与用户确认）

- 上下文窗口数据源：**后端返回真实值 + 前端推断兜底**。
- 端点筛选器：**仅改名为「模型能力」并把协议值映射为中文**，筛选逻辑不变。
- 美化程度：**全面对齐参考页布局**。
- 筛选器去留：**顶部能力 Tab 主导，厂商/分组/计费类型收进顶部右侧下拉菜单**，移除左侧 `PricingSidebar`。
- **模型能力的数据来源 = 模型 tags 中设定的能力标签**（chat/completion/vision/image/audio/video/embedding/code/reasoning，即第一轮已做中文翻译的标签集）。`/models/metadata` 编辑抽屉的「标签」字段即设置/纠正模型归类的入口——不准确时由管理员在标签里增删能力标签来修正，**不依赖按模型名推断**。

## 能力中文映射（统一口径）

协议端点 → 中文能力名（任务 3、4 共用）：

| 协议端点值 | 中文能力 |
|-----------|---------|
| openai | 对话 |
| openai-response | 对话 |
| openai-completions | 补全 |
| anthropic | 对话 |
| gemini | 对话 |
| image-generation | 图像 |
| embeddings | 向量 |
| jina-rerank | 重排 |
| audio-speech | 音频 |
| audio-transcriptions | 音频 |
| openai-video | 视频 |
| moderations | 审核 |

用户列出的能力词（对话/补全/视觉/图像/音频/视频/向量/代码/推理）作为通用能力词典，在编辑抽屉的能力→端点对照说明中使用。

### 能力标签词典（顶部 Tab 与卡片能力 chip 的真实来源）

模型能力**统一来源于模型 tags**（`model.tags` 逗号分隔字符串）中的能力标签，与第一轮已翻译的标签集一致：

| 能力标签 (tag) | 中文 | 归入顶部 Tab |
|---------------|------|-------------|
| chat | 对话 | 文本 |
| completion | 补全 | 文本 |
| code | 代码 | 编码 |
| reasoning | 推理 | 文本 |
| vision | 视觉 | 多模态 |
| image | 图像 | 图片 |
| audio | 音频 | 多模态 |
| video | 视频 | 视频 |
| embedding | 向量 | 文本 |

顶部能力 Tab 分类（参考页：全部/文本/编码/多模态/图片/视频），**按模型 tags 中的能力标签归类**（不按模型名推断）：

| Tab | 归类规则（基于 tags） |
|-----|---------|
| 全部 | 不筛选 |
| 文本 | tags 含 chat / completion / reasoning / embedding |
| 编码 | tags 含 code |
| 多模态 | tags 含 vision / audio（即文本之外的输入能力） |
| 图片 | tags 含 image |
| 视频 | tags 含 video |

归类不准时，管理员在 `/models/metadata` 编辑抽屉的「标签」字段增删对应能力标签即可修正——该字段是唯一的归类入口。模型无能力标签时归入「全部」，不出现在具体能力 Tab。

## 任务 4：端点配置说明 + 能力模板（最先做，零后端依赖）

文件：`web/default/src/features/models/constants.ts`、`web/default/src/features/models/components/drawers/model-mutate-drawer.tsx`

- 在 `constants.ts` 现有 `ENDPOINT_TEMPLATES` 旁新增 `CAPABILITY_ENDPOINT_HINTS`：能力中文名 → 推荐端点模板 key 列表 + 一句话说明。
- 编辑抽屉 Endpoints 区块：
  - 在「配置格式说明」卡片中加入**能力 → 端点对照表**（中文能力名、对应协议端点、点击一键填充模板）。
  - 现有模板下拉 `SelectItem` 的 description 改用中文能力说明。
- 不改后端，不改数据结构。

## 任务 3：以 tags 能力标签的顶部 Tab 取代端点协议筛选器

文件：`web/default/src/features/pricing/hooks/use-filters.ts`、`pricing-toolbar.tsx`、`index.tsx`、`constants.ts`

> 决策（已确认）：「模型能力」统一由顶部能力 Tab（来自 `model.tags`）承载，**原按协议端点（`supported_endpoint_types`）的筛选器与顶部 Tab 职能重叠，予以删除**。这也与参考页一致（参考页只有能力 Tab，无协议筛选器）。

- 移除 `useFilters` 中的 `endpointTypeFilter` 状态、`filterByEndpointType` 调用、相关 `setEndpointTypeFilter` 与 URL 同步。
- 移除 toolbar/侧边栏中 Endpoint Type 筛选 UI。
- `constants.ts` 的 `ENDPOINT_TYPES` / `getEndpointTypeLabels` 若无其他引用则一并清理（先 grep 确认无其他消费方）。
- 顶部能力 Tab 的中文能力名沿用「能力标签词典」表（对话/补全/代码/推理/视觉/图像/音频/视频/向量）。
- i18n：Tab 文案（全部/文本/编码/多模态/图片/视频）加入 `zh.json` 与 `static-keys.ts`。

## 任务 2：上下文窗口（后端真实值 + 前端兜底）

### 后端
- `model/model_meta.go`：`Model` struct 新增 `ContextLength int \`json:"context_length,omitempty" gorm:"default:0"\``。`Model` 已在 `model/main.go` 的 `AutoMigrate` 列表中，SQLite/MySQL/PostgreSQL 三库自动建列，默认 0，向后兼容。
- `Model.Update()` 的 `Select(...)` 列表加入 `"context_length"`，确保零值可写。
- `model/pricing.go`：`Pricing` struct 新增 `ContextLength int \`json:"context_length,omitempty"\``；在组装 pricing 时 `pricing.ContextLength = meta.ContextLength`（来自 `metaMap[model]`）。
- 遵循 Rule 1：本任务无 JSON marshal/unmarshal 手写代码，GORM 标签即可。

### 前端
- `web/default/src/features/pricing/types.ts`：`PricingModel.context_length` 已存在（可选），保持。
- 编辑抽屉基础信息区新增「上下文窗口」数字输入（占位如 `128000`），写入 `model.context_length`。
  - `web/default/src/features/models/types.ts` 的 `modelFormSchema` 与抽屉内 `extendedModelFormSchema` 加 `context_length: z.number().optional()`。
  - 提交 payload 带上 `context_length`。
- 展示：卡片/表格原「标签列」位置改为显示上下文窗口。
  - 取值：`model.context_length && model.context_length > 0 ? model.context_length : inferModelMetadata(model).context_length`。
  - 用现有 `formatTokenCount()` 格式化为 `128K` / `1M`。

## 任务 1：美化模型广场（顶部 Tab 主导）

文件：`web/default/src/features/pricing/index.tsx`、`components/*`、`hooks/use-filters.ts`

### 布局结构（自上而下）
1. **Hero 横幅**：小标题 `MODEL PLAZA` + 主标题「模型广场 · 一处接入全球 AI」+ 三项统计（启用模型数 / 厂商数 / 99% 可用性 占位），保留现有渐变背景，居中。
2. **搜索框**：保留现有 `SearchBar`。
3. **顶部能力 Tab**：全部/文本/编码/多模态/图片/视频（横向 tablist）。
4. **顶部右侧下拉区**：厂商、分组、计费类型三个下拉（替代左侧 `PricingSidebar`），加排序 + Token 单位 + 卡片/表格视图切换（复用现 toolbar 能力）。
5. **卡片网格 / 表格**：保留 card/table 双视图。

### 卡片视觉（对齐参考页）
- 头部：图标 + 模型名（mono）+ 厂商名。
- 一句话能力描述（复用 description，缺失时按能力生成「支持 X、Y 能力，按 Token 透明计费」）。
- 能力 chip 行（中文能力名，来源于 `model.tags` 中的能力标签）。
- 输入/输出价格行（保留现有 `formatPrice`）。
- 上下文窗口 chip（任务 2）。
- 「立即调用 →」按钮（指向控制台/详情）。

### 组件改动
- 移除 `PricingSidebar` 的左侧使用；新增顶部 Tab + 下拉筛选区（可在 `PricingToolbar` 内扩展或新增 `PricingTopFilters`）。
- `useFilters` 增加 `capabilityTab` 状态及 `filterByCapabilityTab` 逻辑（**基于 `model.tags` 中的能力标签归类**，见上文「能力标签词典」表，不按模型名推断），与现有 search/vendor/group/quotaType/endpointType 组合。
- `index.tsx` 的 `grid xl:grid-cols-[330px_1fr]` 两栏布局改为单栏纵向。
- 标签筛选（tag）：原侧边栏的 Model Tags 独立筛选移除；标签语义已由顶部能力 Tab 承载，标签关键词搜索仍由搜索框覆盖。

## 实施顺序

1. 任务 4（纯前端，独立）
2. 任务 3（纯前端，独立）
3. 任务 2 后端（struct + 迁移 + pricing 填充 + Update）
4. 任务 2 前端（表单输入 + 展示替换）
5. 任务 1（布局重构，依赖任务 2/3 的展示与中文能力）

## 测试策略

- 后端：`go build ./...` 通过；`go test ./model/...` 通过；手动验证 SQLite 启动自动建列、编辑模型写入 context_length 后 `/api/pricing` 返回该值。
- 前端：`bun run build` 通过；`bun run i18n:sync` 无缺失键；浏览器验证 /pricing Hero、能力 Tab 切换、下拉筛选、卡片上下文窗口显示、/models/metadata 端点说明与模板填充。

## 不做（YAGNI）

- 不引入 max_output_tokens、知识截止日期等新后端字段（本次仅 context_length）。
- 不改 billing/relay 逻辑。
- 不动 classic 主题前端。
- 不做参考页未要求的标签筛选保留（标签位让位给上下文窗口）。

## 受保护信息

严格保留 QuantumNous / new-api 相关版权头与标识（CLAUDE.md Rule 5）。
