# 企业认证与 Token 严谨计量 规划（V1 讨论稿）

> 面向随星知 API（token.35sz.top）。目标：把「谁在用、用来干什么、花了多少」从当前的"单用户额度"升级为"企业级可审计计量"。

---

## 一、现状盘点（基于当前代码，不含推测）

| 能力 | 现状 | 关键位置 |
| --- | --- | --- |
| 账号体系 | 单账号 + 角色（guest/user/admin/root） | `common/constants.go` RoleXxx |
| 凭证 | API Key + 访问令牌（带 scope），支持过期 | `model/user_access_token.go` |
| 额度 | 用户级 `quota`，按分组倍率折算 | `model/user.go`、分组倍率 |
| 计费模式 | 按 token / 按次 / 分档表达式（`tiered_expr`），缓存读写单独计价 | `constant/pricing.go`、`pkg/billing*` |
| 模型分类 | `model_type` 单值 + 新增 `model_types` 多值（文本/语音/图像/视频/向量） | `model/model_meta.go` |
| 用量记录 | 请求日志 + 用量日志（可按令牌/模型/时间查询） | `logs`、`model/log.go` |
| 任务类用量 | 插件上报 usage（秒/张/次），任务轮询结算 | `plugins/tasks/*`、`service/task_polling.go` |
| 生成记录 | `generation_records`（图像/视频产物与成本） | `model/generation_record.go` |

**缺口（正是这次要补的）**：

1. **主体是企业，不是个人**——现在只能给一个账号充值，企业无法把额度分给 10 个开发者并分别设上限。
2. **计量口径不统一**——文本按 token、图像按张、视频按秒、按次计费混在一条链路里，缺少一张能横向比较的"标准用量"账。
3. **缺少对账**——本地结算数与上游返回的 usage 之间没有逐条比对，异常只能靠人工发现。
4. **审计粒度不足**——日志可查，但不可导出为财务口径的账单；也没有"谁批准了这个额度"的痕迹。

---

## 二、企业认证（V1）

### 2.1 数据模型

```
enterprise                 企业主体
  id / name / credit_code(统一社会信用代码) / status / owner_user_id
enterprise_member          成员与授权
  enterprise_id / user_id / role(owner|admin|billing|member) / status
enterprise_kyc             认证材料
  enterprise_id / type(营业执照|对公账户|授权书) / file_id / status / reviewed_by / reviewed_at
enterprise_quota_policy    额度策略
  enterprise_id / monthly_quota / per_member_quota / model_allowlist / rate_limit
enterprise_invoice         开票（可后置）
  enterprise_id / period / amount / status
```

设计要点：

- **企业不替代用户**：一个 user 可以同时属于一个 enterprise 和自己的个人账户，请求时按 API Key 归属判定走哪套额度。
- **额度双上限**：企业月度总额 + 成员月度上限，任一触顶即拒（返回明确错误码，便于调用方区分）。
- **模型白名单**：按 `model_types` 与具体模型两级允许，便于"只给财务部开文本模型"这类场景。

### 2.2 认证流程（状态机）

```
draft → submitted → reviewing → approved
                             ↘ rejected(可补充材料回到 submitted)
approved → suspended（欠费/违规）→ approved（恢复）
```

- 提交时校验：统一社会信用代码格式、营业执照 OCR 与填写信息一致性（V1 可先人工核验）。
- 审核留痕：`reviewed_by`、`reviewed_at`、拒绝原因，全部进审计日志。
- **认证不是使用前置条件**：未认证用户仍可用个人额度；认证后才能开企业额度、开票、加成员。

### 2.3 权限

| 角色 | 能做什么 |
| --- | --- |
| owner | 转让企业、审批额度、开票、删除成员 |
| admin | 加/停成员、设成员上限、配模型白名单 |
| billing | 只读账单与用量，可导出 |
| member | 只能创建自己的 Key，看自己的用量 |

所有写操作记录 `actor / action / before / after`，与现有管理员操作日志同表。

---

## 三、Token 严谨计量（V1）

### 3.1 统一计量口径

一张账要能横向比较，必须先定"标准单位"。建议：

| 模型类型 | 计量单位 | 说明 |
| --- | --- | --- |
| 文本 / 多模态 | `token` | 分 prompt / completion / cached(read) / cache-write / reasoning |
| 向量 | `token`（输入） | 输出恒为 0 |
| 图像 | `image`（张） + 可选 token | 张数由上游 usage 或请求参数确定，取两者较大值 |
| 视频 | `second`（秒） × 分辨率档 | 分辨率进档位，避免 480p 与 1080p 同价 |
| 语音 | `second`（秒）或 `character` | 按上游返回优先，缺失时按音频时长估算并打 `estimated: true` |

所有用量最终折算为**整数微美元（µUSD）**存账，避免浮点累计误差；展示层再乘分组倍率与汇率。

### 3.2 采集与结算链路

```
客户端请求
  → 分发/插件决定计费模式
  → 预扣（reserve）：按预估上限扣减，防超支
  → 上游返回 usage
  → 结算（settle）：实扣 = min(实测算出的额度, 预扣) ，差额回滚
  → 写 usage 明细（含 request_id、上游 usage 原文、折算结果、是否估算）
```

三条硬要求：

1. **幂等**：以 `request_id` 为唯一键，重试不重复扣费（插件结算失败必须能补偿）。
2. **可回溯**：明细里保留上游 usage 原始 JSON（脱敏后），任何一笔都能重算。
3. **异常兜底**：上游不返回 usage 时，标记 `estimated` 并按保守值计费，同时进"待人工复核"队列。

### 3.3 对账

每日定时（建议 02:00）：

- 本地结算总额 vs 上游账单（有 API 的渠道）逐渠道比对；
- 差异超过阈值（如 0.5% 或绝对值 > $1）自动生成对账异常单；
- 异常单必须由管理员标记「已知/已修正/上游误差」，闭环后才允许月结。

### 3.4 与模型类型的联动

`model_types` 多选直接决定计量器：

- 含 `video` → 秒 + 分辨率档；
- 含 `image` → 张数；
- 含 `audio` → 秒/字符；
- 含 `embedding` → 仅输入 token；
- 其余按 token。

这样"新增一个模型，选好类型就自动有了计量口径"，不需要每次改代码——这正是模型类型多选的产品价值。

---

## 四、分阶段落地

| 阶段 | 内容 | 验收标准 |
| --- | --- | --- |
| P1（1 周） | 计量口径统一 + µUSD 存账 + `estimated` 标记 | 任取一笔请求，能重算出与账单一致的数字；估算订单可见 |
| P2（1 周） | 日对账 + 异常单 | 连续 3 天对账差异 < 0.5%，异常单可闭环 |
| P3（2 周） | 企业主体 + 成员 + 双上限额度 | 企业主能给成员分配额度，成员超限被正确拒绝 |
| P4（1 周） | 企业认证 + 审核留痕 | 走完 draft→approved 全流程，材料与审核记录可查 |
| P5（后置） | 账单导出（CSV/PDF）+ 开票 | 财务可直接使用，字段与发票口径一致 |

---

## 五、风险与对策

| 风险 | 对策 |
| --- | --- |
| 改计量口径影响历史账单 | 新口径只对生效日之后的请求生效，历史数据保留旧口径并标注版本号 |
| 上游 usage 缺失导致少收/多收 | `estimated` 标记 + 保守计费 + 人工复核队列 |
| 企业额度被恶意耗尽 | 双上限 + 成员级限流 + 异常用量告警（环比突增 3 倍） |
| 认证材料合规 | 材料只存对象存储、访问走签名 URL、审核人可追溯；到期自动清理 |
| 与上游合并冲突 | 企业相关表与接口独立命名空间（`/api/enterprise/*`），不侵入现有 user/quota 逻辑 |

---

## 六、下一步（需要决策）

1. 企业版是否**独立定价**（企业倍率 vs 个人倍率）？
2. 认证 V1 是否接受**纯人工审核**（不做 OCR 自动比对）？
3. 开票是走第三方（如诺诺）还是先出 PDF 对账单？
4. 对账阈值定多少（0.5% / 1%）？

> 本文是讨论稿，未改动任何线上逻辑。P1 的计量改造会触碰结算主链路，建议单独开分支做灰度。
