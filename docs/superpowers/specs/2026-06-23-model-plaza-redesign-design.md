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

顶部能力 Tab 分类（参考页：全部/文本/编码/多模态/图片/视频），按 `model_type` + 推断能力归类：

| Tab | 归类规则 |
|-----|---------|
| 全部 | 不筛选 |
| 文本 | model_type=text 且非纯多模态 |
| 编码 | tags 或推断 capabilities 含 code/code_interpreter |
| 多模态 | 输入模态 > 1（含 image/audio/video 输入） |
| 图片 | model_type=image 或 supported_endpoint_types 含 image-generation |
| 视频 | model_type=video 或 supported_endpoint_types 含 openai-video |

## 任务 4：端点配置说明 + 能力模板（最先做，零后端依赖）

文件：`web/default/src/features/models/constants.ts`、`web/default/src/features/models/components/drawers/model-mutate-drawer.tsx`

- 在 `constants.ts` 现有 `ENDPOINT_TEMPLATES` 旁新增 `CAPABILITY_ENDPOINT_HINTS`：能力中文名 → 推荐端点模板 key 列表 + 一句话说明。
- 编辑抽屉 Endpoints 区块：
  - 在「配置格式说明」卡片中加入**能力 → 端点对照表**（中文能力名、对应协议端点、点击一键填充模板）。
  - 现有模板下拉 `SelectItem` 的 description 改用中文能力说明。
- 不改后端，不改数据结构。

## 任务 3：端点筛选器改名「模型能力」+ 中文映射

文件：`web/default/src/features/pricing/constants.ts`、消费方组件

- `getEndpointTypeLabels(t)` 把协议值映射为上表中文能力名（值不变，label 改中文）。
- 顶部右侧下拉中的该筛选标题从 `Endpoint Type` 改为「模型能力」。
- 筛选逻辑 `filterByEndpointType`（按 `supported_endpoint_types`）不变。
- i18n：新增中文键到 `zh.json` 与 `static-keys.ts`。

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
- 能力 chip 行（中文能力名）。
- 输入/输出价格行（保留现有 `formatPrice`）。
- 上下文窗口 chip（任务 2）。
- 「立即调用 →」按钮（指向控制台/详情）。

### 组件改动
- 移除 `PricingSidebar` 的左侧使用；新增顶部 Tab + 下拉筛选区（可在 `PricingToolbar` 内扩展或新增 `PricingTopFilters`）。
- `useFilters` 增加 `capabilityTab` 状态及对应 `filterByCapabilityTab` 逻辑（基于 model_type + `inferModelMetadata` 推断），与现有 search/vendor/group/quotaType/endpointType 组合。
- `index.tsx` 的 `grid xl:grid-cols-[330px_1fr]` 两栏布局改为单栏纵向。
- 标签筛选（tag）：原侧边栏的 Model Tags 筛选移除（任务 2 已把标签列替换为上下文窗口）；如需保留 tag 搜索仍由搜索框覆盖。

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
