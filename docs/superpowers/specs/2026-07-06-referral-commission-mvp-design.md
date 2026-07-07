# 分销佣金 MVP 设计规范

**日期：** 2026-07-06
**作者：** PerJac（AI 协作产出）
**状态：** 已通过用户设计评审，待写实施计划
**关联：** 二次开发功能清单 — `分销、佣金、只转余额、不出金`

---

## 1. 目标与范围

**目标：** 为 `token.35sz.top` 平台建设「二级返佣 + 佣金转账户余额」的最小可用分销体系，用作面向渠道商/推广者的增长工具。

**范围（MVP）：**

- ✅ 二级返佣：直接上级（L1）20%、上上级（L2）5%，可运营配置
- ✅ 只对**首次成功充值** 且通过 **支付宝 / 微信 / 易支付** 三个国内通道产生的充值分佣
- ✅ 佣金 T+7 冻结（可配到规则表 `frozen_days`），到期由定时任务转到用户「佣金账户」
- ✅ 佣金只能**手动转入账户余额**（`users.quota`），账户余额只能被 API 消费扣款
- ✅ 佣金账户结构独立，不与现有 `AffQuota`（注册邀请赠额度）合并
- ✅ 管理员规则页 + 全平台流水页（含作废操作）+ 总览页
- ✅ 用户端分销中心页（邀请码、推广链接、佣金账户、流水、下线、二维码）
- ✅ 灰度开关：`commission_rules.enabled`，粒度到层级

**明确不做（推迟到二期）：**

- ❌ 订阅分佣（`scope='subscription'`）
- ❌ 消费分佣（按 API 用量返佣）
- ❌ 3 级及以上分销
- ❌ 微信/支付宝真实提现（企业付款 API）
- ❌ 反作弊规则引擎（同 IP / 设备指纹 / 首单窗口）
- ❌ 分销员等级 / 阶梯佣金
- ❌ 财务对账导出、KYC 实名认证
- ❌ 运营活动玩法（"首充双倍佣金"等）

**为什么可以放心排除：**
数据模型（`commission_rules.scope` 枚举、`level` 字段、`commission_records` 完整流水）已为二期能力预留扩展点，届时只需 `INSERT` 新规则 + 加新 handler，不改现有表结构。

---

## 2. 核心业务规则

### 2.1 邀请链路建立

- 用户 A 注册时填写 B 的邀请码（沿用现有 `users.aff_code` 与 `users.inviter_id` 字段）
- **注册瞬间冗余锁定** L1 = B，L2 = B 当时的 `users.inviter_id` 快照
- 即使 B 后续变更上级，A 的链路保持不变（不追溯）
- 若 B 无上级（`inviter_id = 0`），则 `l2_user_id = 0`，A 的二级佣金机会直接放弃

### 2.2 佣金触发条件

**同时满足全部条件才产生佣金：**

1. `topups.status = 'success'`
2. `topups.payment_provider ∈ {'alipay', 'wechat', 'epay'}`
3. 该用户历史上首次成功充值（`COUNT(topups WHERE user_id=? AND status='success' AND provider IN (...)) == 1`）
4. `user_referral_paths` 存在对应用户的邀请路径
5. `commission_rules(scope='first_topup', level=N, enabled=true)` 规则存在
6. `topups.money >= min_topup_cents`（规则中的门槛）

**佣金基数：** `topups.money`（人民币分单位，与 `topups` 表字段类型一致）

**佣金计算：** `commission_amount_cents = floor(base_amount_cents * rate_percent / 100)`

**幂等约束：** `UNIQUE (source_topup_id, beneficiary_id, level)` 防止支付回调重放导致重复分佣

### 2.3 冻结 → 结算

- 佣金写入时状态 = `pending`，`frozen_until = now + regulations.frozen_days * 86400`
- 定时任务 `commission_settle`（每 10 分钟）扫 `pending AND frozen_until <= now`
- 每条独立事务：
  - `records.status = 'settled'`, `settled_at = now`
  - `user_commission_stats.pending_cents -= amt`
  - `user_commission_stats.balance_cents += amt`
  - `user_commission_stats.lifetime_cents += amt`
  - 写 `logs` 表 `LogTypeSystem` "推广佣金结算 ¥X.XX 来自 <用户#N> 首充"

### 2.4 佣金 → 余额兑换

**汇率与 QuotaPerUnit 实时读取，兑换时刻快照落表：**

```go
rate     := operation_setting.USDExchangeRate
quotaPer := int64(common.QuotaPerUnit)

usdAmount := float64(commissionCents) / 100.0 / rate
quotaCredited := int64(math.Floor(usdAmount * float64(quotaPer)))
```

**示例：** ¥25 佣金（2500 cents），`USDExchangeRate=7.2`，`QuotaPerUnit=500000`
→ `quotaCredited = floor(2500/100/7.2 * 500000) = 1,736,111`

**约束：**
- `commissionCents > 0`
- `user_commission_stats.balance_cents >= commissionCents`（`SELECT ... FOR UPDATE`）
- `USDExchangeRate > 0`（否则拒绝，返回"系统未设置汇率"）
- 换算取 `floor`（用户不占运营便宜）
- 兑换后不可撤销

### 2.5 作废（管理员反作弊）

- 冻结中的记录：直接扣减 `pending_cents`
- 已结算的记录：扣减 `balance_cents`（要求 `balance_cents >= amount`，否则拒绝）
- 已被转余额过的记录不允许作废（判定：查询 `commission_redemptions` 时序）
- 必填 `voided_reason`（写入 `records.voided_reason` + `logs` 审计）

---

## 3. 数据模型（共 5 张新表）

### 3.1 `user_commission_stats` — 佣金账户主表

```sql
CREATE TABLE user_commission_stats (
  user_id                    INT PRIMARY KEY,
  commission_balance_cents   BIGINT NOT NULL DEFAULT 0,   -- 可转余额（已结算）
  commission_pending_cents   BIGINT NOT NULL DEFAULT 0,   -- 冻结中
  commission_lifetime_cents  BIGINT NOT NULL DEFAULT 0,   -- 累计已结算
  commission_redeemed_cents  BIGINT NOT NULL DEFAULT 0,   -- 累计已转余额
  updated_at                 BIGINT NOT NULL,
  created_at                 BIGINT NOT NULL
);
```

**懒建策略：** 只有当用户首次触发佣金相关操作（写入 pending / 结算 / 转余额）时才 `INSERT`。查询用 `LEFT JOIN`，未参与分销的用户不占行。

### 3.2 `user_referral_paths` — 邀请链路快照表

```sql
CREATE TABLE user_referral_paths (
  user_id     INT PRIMARY KEY,
  l1_user_id  INT NOT NULL,
  l2_user_id  INT NOT NULL DEFAULT 0,
  created_at  BIGINT NOT NULL,
  INDEX idx_l1 (l1_user_id),
  INDEX idx_l2 (l2_user_id)
);
```

**注册时一次性插入，永不更新。**

### 3.3 `commission_rules` — 佣金规则表（可运营）

```sql
CREATE TABLE commission_rules (
  id                BIGINT PRIMARY KEY AUTO_INCREMENT,
  scope             VARCHAR(32) NOT NULL,      -- MVP 只有 'first_topup'
  level             INT NOT NULL,              -- 1 or 2
  rate_percent      DECIMAL(5,2) NOT NULL,
  min_topup_cents   BIGINT NOT NULL DEFAULT 0,
  frozen_days       INT NOT NULL DEFAULT 7,
  enabled           BOOLEAN NOT NULL DEFAULT true,
  created_at        BIGINT NOT NULL,
  updated_at        BIGINT NOT NULL,
  UNIQUE KEY uk_scope_level (scope, level)
);
```

**种子数据（`service/commission/seed.go`，首次启动检测表空时插入）：**

```
('first_topup', 1, 20.00, 0, 7, true)
('first_topup', 2,  5.00, 0, 7, true)
```

**灰度开关：** 单条规则的 `enabled=false` 即停止对应层级的分佣，不影响其他层级；两条都 `false` = 全局停用（佣金逻辑走完但不产生金额）。

### 3.4 `commission_records` — 佣金流水表

```sql
CREATE TABLE commission_records (
  id                      BIGINT PRIMARY KEY AUTO_INCREMENT,
  beneficiary_id          INT NOT NULL,
  source_user_id          INT NOT NULL,
  source_topup_id         BIGINT NOT NULL,
  scope                   VARCHAR(32) NOT NULL,
  level                   INT NOT NULL,
  rate_percent            DECIMAL(5,2) NOT NULL,
  base_amount_cents       BIGINT NOT NULL,          -- = topups.money
  commission_amount_cents BIGINT NOT NULL,
  status                  VARCHAR(16) NOT NULL,     -- pending | settled | voided
  frozen_until            BIGINT NOT NULL,
  settled_at              BIGINT NOT NULL DEFAULT 0,
  voided_at               BIGINT NOT NULL DEFAULT 0,
  voided_reason           VARCHAR(255),
  created_at              BIGINT NOT NULL,
  updated_at              BIGINT NOT NULL,
  UNIQUE KEY uk_topup_beneficiary_level (source_topup_id, beneficiary_id, level),
  INDEX idx_beneficiary_status (beneficiary_id, status),
  INDEX idx_pending_frozen (status, frozen_until)
);
```

### 3.5 `commission_redemptions` — 佣金转余额记录表

```sql
CREATE TABLE commission_redemptions (
  id                    BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id               INT NOT NULL,
  commission_cents      BIGINT NOT NULL,          -- 本次转出佣金
  usd_exchange_rate     DECIMAL(10,4) NOT NULL,   -- 转入时刻实时汇率快照
  quota_per_unit        BIGINT NOT NULL,          -- 转入时刻 QuotaPerUnit 快照
  quota_credited        BIGINT NOT NULL,          -- 实际入 users.quota
  created_at            BIGINT NOT NULL,
  INDEX idx_user_created (user_id, created_at)
);
```

**关键点：** 汇率快照落表 → 汇率后期变化不影响历史记录可追溯。

---

## 4. 组件与代码位置

### 4.1 后端分层

```
model/
  user_commission_stats.go     CRUD + GetOrCreate + atomic 加减方法
  user_referral_path.go        单点插入/查询
  commission_rule.go           CRUD + 活动规则缓存 GetActiveRules(scope)
  commission_record.go         CRUD + IsFirstSuccessfulTopup + 幂等插入
  commission_redemption.go     只增不改
  main.go                      追加 5 张表到 AutoMigrate

service/commission/
  path.go                      BuildReferralPath(newUserID, inviterID) — 注册时调用
  record.go                    OnTopupCompleted(topup) — 支付回调后异步调用
  settle.go                    SettlePending() — 定时任务执行体
  redeem.go                    Redeem(userID, cents) — 用户手动转入
  void.go                      Void(recordID, reason) — 管理员作废
  seed.go                      SeedDefaultRules() — 启动时兜底初始化

controller/
  commission.go                用户 6 + 管理员 4 = 10 个 handler
  user.go                      Register 追加 BuildReferralPath 调用
  topup_alipay.go
  topup_wechat.go              支付回调 goroutine 追加 OnTopupCompleted
  (model/topup.go RechargeEpay 与 controller/user.go EpayNotify 同)

setting/system_task/
  commission_settle_task.go    注册 TaskTypeCommissionSettle handler

router/api-router.go           追加 6 个 selfRoute + 6 个 commissionAdminRoute
```

### 4.2 服务层核心逻辑

**`service/commission/record.go`：**

```go
func OnTopupCompleted(topup *model.TopUp) {
    defer func() {
        if r := recover(); r != nil {
            common.SysError(fmt.Sprintf("commission recording panic: %v", r))
        }
    }()

    // 1) 国内通道白名单
    if !isDomesticPayment(topup.PaymentProvider) {
        return
    }
    // 2) 首充判定
    if !model.IsFirstSuccessfulTopup(topup.UserId, topup.Id) {
        return
    }
    // 3) 读邀请路径
    path, err := model.GetReferralPath(topup.UserId)
    if err != nil || path == nil {
        return
    }
    // 4) 读活动规则（缓存）
    rules := model.GetActiveCommissionRules("first_topup")
    // 5) 单事务写 pending 记录 + user_commission_stats.pending_cents += amt
    _ = model.DB.Transaction(func(tx *gorm.DB) error {
        for _, rule := range rules {
            beneficiary := pickBeneficiary(path, rule.Level)
            if beneficiary == 0 {
                continue
            }
            if topup.Money < rule.MinTopupCents {
                continue
            }
            amt := int64(math.Floor(float64(topup.Money) * rule.RatePercent / 100))
            if err := insertCommissionRecord(tx, topup, beneficiary, rule, amt); err != nil {
                return err
            }
            if err := stats.AddPending(tx, beneficiary, amt); err != nil {
                return err
            }
        }
        return nil
    })
}
```

**`service/commission/settle.go`：**

```go
func SettlePending() (settled int, err error) {
    for {
        var batch []model.CommissionRecord
        model.DB.Where("status = ? AND frozen_until <= ?", "pending", time.Now().Unix()).
            Limit(500).Find(&batch)
        if len(batch) == 0 {
            return
        }
        for _, r := range batch {
            _ = model.DB.Transaction(func(tx *gorm.DB) error {
                if err := markSettled(tx, r.ID); err != nil {
                    return err
                }
                if err := stats.PendingToBalance(tx, r.BeneficiaryID, r.CommissionAmountCents); err != nil {
                    return err
                }
                return recordLog(tx, r)
            })
            settled++
        }
        if len(batch) < 500 {
            return
        }
    }
}
```

**`service/commission/redeem.go`：**

```go
func Redeem(userID int, commissionCents int64) (quotaCredited int64, err error) {
    if commissionCents <= 0 {
        return 0, errors.New("金额必须大于 0")
    }
    rate := operation_setting.USDExchangeRate
    qpu := int64(common.QuotaPerUnit)
    if rate <= 0 || qpu <= 0 {
        return 0, errors.New("系统未设置汇率，请联系管理员")
    }
    quotaCredited = int64(math.Floor(float64(commissionCents) / 100.0 / rate * float64(qpu)))

    err = model.DB.Transaction(func(tx *gorm.DB) error {
        var s model.UserCommissionStats
        if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
            Where("user_id = ?", userID).First(&s).Error; err != nil {
            return err
        }
        if s.CommissionBalanceCents < commissionCents {
            return errors.New("佣金余额不足")
        }
        // 更新 stats
        tx.Model(&s).Updates(map[string]any{
            "commission_balance_cents":   gorm.Expr("commission_balance_cents - ?", commissionCents),
            "commission_redeemed_cents":  gorm.Expr("commission_redeemed_cents + ?", commissionCents),
            "updated_at":                 time.Now().Unix(),
        })
        // 增加 users.quota
        tx.Model(&model.User{}).Where("id = ?", userID).
            Update("quota", gorm.Expr("quota + ?", quotaCredited))
        // 落 redemption 快照
        tx.Create(&model.CommissionRedemption{
            UserID:           userID,
            CommissionCents:  commissionCents,
            USDExchangeRate:  rate,
            QuotaPerUnit:     qpu,
            QuotaCredited:    quotaCredited,
            CreatedAt:        time.Now().Unix(),
        })
        // 写 logs
        model.RecordLog(userID, model.LogTypeTopup,
            fmt.Sprintf("佣金 ¥%.2f 转入余额 → +%d 额度", float64(commissionCents)/100, quotaCredited))
        return nil
    })
    return
}
```

### 4.3 支付回调埋点（3 处）

在既有事务提交之后异步触发：

```go
// model/topup.go RechargeAlipay / RechargeWechat 事务后
go commission.OnTopupCompleted(topup)

// controller/user.go EpayNotify 事务后同上
```

**关键：**
- `go` 关键字异步 → 不阻塞回调响应
- `OnTopupCompleted` 内部 `recover()` 兜底
- Stripe/Creem/Waffo 不改代码，天然不进分佣

### 4.4 定时任务集成

复用 `system_tasks` 调度器，注册 `TaskTypeCommissionSettle` handler，频率每 10 分钟。

---

## 5. API 契约

### 5.1 用户端（`middleware.UserAuth()`）

| Method | Path | 用途 |
| --- | --- | --- |
| GET | `/api/user/commission/stats` | 佣金账户 4 项统计 + 我的邀请码/链接 |
| GET | `/api/user/commission/records?status=&page=&size=` | 我的佣金流水（分页） |
| GET | `/api/user/commission/redemptions?page=&size=` | 我的转余额记录 |
| GET | `/api/user/commission/downlines?level=&page=&size=` | 我的下线（脱敏） |
| GET | `/api/user/commission/quota-preview?cents=` | 转入前预览换算 |
| POST | `/api/user/commission/redeem` | 转入余额 body: `{cents: number}` |

### 5.2 管理员端（`middleware.RootAuth()`，路由前缀 `/api/commission-admin`）

| Method | Path | 用途 |
| --- | --- | --- |
| GET | `/rules` | 列出所有规则 |
| PUT | `/rules/:id` | 更新规则（含 enabled 灰度开关） |
| GET | `/records?status=&beneficiary=&source=&from=&to=&page=` | 全平台流水（含筛选） |
| POST | `/records/:id/void` | 作废记录 body: `{reason: string}` |
| POST | `/settle-now` | 手动触发结算 |
| GET | `/stats` | 平台总览（总佣金、参与用户数、转化率、近 30 天趋势） |

### 5.3 下线列表脱敏规则

**用户端（自己看下线）：**
- 用户名：`张三` → `张*`（保留首字）
- 邮箱：`test@example.com` → `t***@example.com`
- 手机：`13800138000` → `138****8000`
- 只展示：注册时间、层级（L1/L2）、是否已首充（不展示金额）

**管理员端：** 有权限查看完整手机号与邮箱（新增专用接口或复用用户管理页）。

---

## 6. 前端组件

### 6.1 用户端 `/console/referral`

```
web/default/src/features/referral/
  index.tsx                              页面入口 + 卡片布局
  api.ts                                 6 个 API 封装
  types.ts                               类型定义
  lib/format-commission.ts               分↔元、可转额度预览换算
  components/
    invite-code-card.tsx                 邀请码 + 复制
    invite-link-card.tsx                 推广链接 + 复制 + 二维码（前端 qrcode 库）
    commission-stats-card.tsx            4 项统计 + 换算预览 + 转入按钮
    redeem-dialog.tsx                    转入余额确认对话框
    records-table.tsx                    佣金流水表
    redemptions-table.tsx                转余额记录表
    downlines-table.tsx                  下线列表（脱敏展示）
  hooks/
    use-commission-stats.ts
    use-commission-records.ts
    use-commission-redemptions.ts
    use-commission-downlines.ts
    use-quota-preview.ts                 debounce, 实时预览
    use-redeem-commission.ts             useMutation + toast
```

**页面布局要点：**

- 顶部两卡：邀请码 + 推广链接（二维码用 `qrcode` npm 包在浏览器端生成 SVG）
- 中部大卡：佣金账户（`balance / pending / lifetime / redeemed` 四项）+ 当前汇率显示 + 「转入账户余额」按钮
- 下部 Tab：佣金流水 / 转余额记录 / 我的下线
- 转入对话框：显示当前系统实时汇率、兑换预估、"汇率以转入时刻的系统实时汇率为准"警示

### 6.2 管理员端 `/system-settings/commission/*`

```
web/default/src/features/system-settings/commission/
  index.tsx
  section-registry.tsx
  sections/
    rules-section.tsx        规则表单（含 enabled 灰度开关）
    records-section.tsx      全平台流水 + 作废对话框
    overview-section.tsx     总览（统计卡 + 近 30 天趋势折线图）
```

**接入方式：** 使用既有 `createSectionRegistry` 模式（与 `billing`、`content` 并列），管理员通过既有 `/system-settings` 分屏进入。

### 6.3 侧边栏 / 导航接入

- 用户端 sidebar：`/console/wallet`（钱包）下方增加 `/console/referral`（我的分销），图标 `Users2` 或 `Share2`
- 管理员端 sidebar：`billing`（计费）下方增加 `commission`（佣金）

### 6.4 i18n

约 60 个新翻译键，`zh.json` + `en.json` + 4 个其他语言 fallback。

---

## 7. 事务与并发一致性

| 场景 | 锁策略 |
| --- | --- |
| 支付回调重放 → 写 pending | `UNIQUE (source_topup_id, beneficiary_id, level)` + `clause.OnConflict{DoNothing: true}` |
| 定时结算逐条处理 | 每条独立事务，`SELECT FOR UPDATE` 该 record |
| 用户转余额 | `SELECT user_commission_stats FOR UPDATE`（`clause.Locking{Strength:"UPDATE"}`） |
| 管理员作废 | `SELECT record + user_commission_stats FOR UPDATE`（同事务） |
| 并发注册相同邀请码 | `users.aff_code` 已有 `uniqueIndex` 保障 |

---

## 8. 错误处理与降级

| 场景 | 处理 | 用户可见影响 |
| --- | --- | --- |
| 佣金写入 goroutine panic | `recover()` + SysLog | 无（充值正常到账） |
| 邀请路径查询失败 | 静默跳过 | 无（只损失佣金机会） |
| 结算任务某条事务失败 | 该条保留 pending、日志报警、继续 | 无（下次调度重试） |
| Redeem `SELECT FOR UPDATE` 超时 | rollback + toast "系统繁忙" | 用户重试 |
| `USDExchangeRate = 0` | Redeem 拒绝 + toast "未设置汇率" | 用户看到提示 |
| 管理员作废时余额不足 | 事务内校验，返回错误 | 管理员看到提示 |

---

## 9. 测试策略

**后端测试文件：**

```
model/user_referral_path_test.go        锁定路径插入
model/commission_rule_test.go           CRUD + 种子存在
model/commission_record_test.go         UNIQUE 幂等、状态机
service/commission/record_test.go       ★ 首充判定 · 通道白名单 · 无路径静默 · 规则禁用
service/commission/settle_test.go       ★ 冻结时间、事务原子、分批
service/commission/redeem_test.go       ★ 汇率快照、余额校验、并发、floor
service/commission/void_test.go         作废状态机
controller/commission_test.go           HTTP 层权限校验
```

**前端测试文件：**

```
web/default/src/features/referral/lib/format-commission.test.ts
web/default/src/features/referral/components/redeem-dialog.test.tsx
web/default/src/features/system-settings/commission/sections/rules-section.test.tsx
```

**遵守 `AGENTS.md` 后端测试规约：** 用 `testify/require + assert`；只测行为契约与幂等性，不测私有函数、无 fuzz、无压力测试。

---

## 10. 三库兼容性核对

| 特性 | SQLite | MySQL 5.7+ | PostgreSQL 9.6+ | 处理 |
| --- | --- | --- | --- | --- |
| `BIGINT AUTO_INCREMENT` | ✓ | ✓ | ✓ | GORM `primaryKey` |
| `DECIMAL(5,2)` / `(10,4)` | ✓ | ✓ | ✓ | `type:decimal(5,2)` |
| 复合 `UNIQUE` | ✓ | ✓ | ✓ | GORM 复合唯一索引 tag |
| `SELECT FOR UPDATE` | ⚠ 无锁 | ✓ | ✓ | `clause.Locking{Strength:"UPDATE"}` |
| 布尔默认值 | ✓ | ⚠ | ⚠ | 不用 `default:true` gorm tag，改 Go 代码默认 |
| `ALTER TABLE ADD COLUMN` | ✓ | ✓ | ✓ | AutoMigrate 处理 |

**种子数据：** `commission_rules` 用 "if not exists 插入" 模式，避免重启重复插入。

---

## 11. 部署与迁移

**首次上线：**

1. `AutoMigrate` 建 5 张新表
2. `service/commission/seed.go` 首次启动检测表空 → 插入 L1=20% / L2=5%
3. `system_tasks` 注册 `commission_settle` 任务（每 10 分钟）
4. 前端 sidebar 新增用户端 `/console/referral` 与管理端 `/system-settings/commission`

**灰度：** `commission_rules.enabled = false` 即停用对应层级，不影响支付回调主流程。

**回滚：**

- 立即停用：SQL 更新 `enabled=false`
- 完全回退：drop 5 张表 + 移除 API 路由 + 移除前端 feature 目录

---

## 12. 交付规模估算

| 分类 | 数量 |
| --- | --- |
| 新增表 | 5（`user_commission_stats`, `user_referral_paths`, `commission_rules`, `commission_records`, `commission_redemptions`） |
| 新增 Model 文件 | 5 |
| 新增 Service 文件 | 6 |
| 新增 Controller | 1（含 10 个 API） |
| 修改现有 Controller | 4 点埋入（Register + 3 处支付回调） |
| 新增前端 feature | 2（`referral` + `system-settings/commission`） |
| 新增前端组件 | ~12 |
| 新增翻译键 | ~60 |
| 后端测试文件 | ~8 |
| 前端测试文件 | ~3 |

**预计工作量：** 10 人日（后端 4 天 · 前端 4 天 · 测试联调 2 天）。

---

## 13. 与既有能力的关系

- **`users.inviter_id / aff_code`：** 继续沿用（邀请码来源）
- **`users.aff_quota / aff_history_quota`：** 保留不动，仍是「注册邀请赠额度」渠道
- **佣金账户与 AffQuota 并列存在，互不干扰**
- **`topups.money` 字段：** 直接作为佣金基数（人民币分单位）
- **`system_tasks` 调度框架：** 复用，新增 `commission_settle` 类型
- **`operation_setting.USDExchangeRate` / `common.QuotaPerUnit`：** 转余额时实时读取，快照落 `commission_redemptions`

---

**设计评审通过：** 2026-07-06
**下一步：** 由 `writing-plans` skill 产出可执行的实施计划文档。
