# 分销佣金 MVP 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现「二级返佣（L1=20%/L2=5%）+ 首充分佣 + T+7 冻结 + 手动转账户余额」的最小可用分销体系，含用户端分销中心页与管理员端规则/流水/总览页。

**Architecture:** 后端遵循 Router → Controller → Service → Model 分层：5 张新表（`user_commission_stats` / `user_referral_paths` / `commission_rules` / `commission_records` / `commission_redemptions`）；支付回调后 goroutine 异步触发佣金写入；`system_tasks` 每 10 分钟执行结算；手动转余额兑换实时读 `USDExchangeRate` 并快照落表。前端新增 2 个 feature 模块（用户端 `referral` + 管理端 `system-settings/commission`）。

**Tech Stack:** Go 1.22 + Gin + GORM v2（SQLite/MySQL/PostgreSQL 三库兼容）；React 19 + TypeScript + Rsbuild + Base UI + Tailwind + TanStack Query + i18next；Bun。

**关联规范：** [`docs/superpowers/specs/2026-07-06-referral-commission-mvp-design.md`](../specs/2026-07-06-referral-commission-mvp-design.md)

---

## File Structure

```
model/
  user_commission_stats.go     — 佣金账户 CRUD + 原子加减方法
  user_referral_path.go        — 邀请路径插入/查询
  commission_rule.go           — 规则 CRUD + 缓存
  commission_record.go         — 流水 CRUD + 首充判定 + 幂等插入
  commission_redemption.go     — 兑换流水（只增不改）
  main.go                      — ★追加 5 张表到 AutoMigrate

service/commission/
  path.go                      — BuildReferralPath(newUserID, inviterID)
  record.go                    — OnTopupCompleted(topup)
  settle.go                    — SettlePending() + ScheduledHandler
  redeem.go                    — Redeem(userID, cents)
  void.go                      — Void(recordID, reason)
  seed.go                      — SeedDefaultRules()

controller/
  commission.go                — 10 个 handler（6 用户 + 4 管理员）
  user.go                      — ★Register 追加 BuildReferralPath
  topup_alipay.go              — ★成功回调追加 go OnTopupCompleted
  topup_wechat.go              — ★同上
  topup.go                     — ★EpayNotify 成功分支追加 go OnTopupCompleted

router/
  api-router.go                — ★注册 6 selfRoute + 4 commissionAdminRoute

web/default/src/features/referral/
  index.tsx / api.ts / types.ts
  lib/format-commission.ts
  components/
    invite-code-card.tsx
    invite-link-card.tsx           (含 qrcode.react 二维码)
    commission-stats-card.tsx
    redeem-dialog.tsx
    records-table.tsx
    redemptions-table.tsx
    downlines-table.tsx
  hooks/
    use-commission-stats.ts
    use-commission-records.ts
    use-commission-redemptions.ts
    use-commission-downlines.ts
    use-quota-preview.ts
    use-redeem-commission.ts

web/default/src/features/system-settings/commission/
  index.tsx / section-registry.tsx / api.ts / types.ts
  sections/
    rules-section.tsx
    records-section.tsx
    overview-section.tsx

web/default/src/routes/_authenticated/console/referral/
  index.tsx                    — TanStack Router 路由入口

web/default/src/routes/_authenticated/system-settings/commission/
  $section.tsx / index.tsx     — 管理端路由

web/default/src/i18n/locales/{zh,en}.json  — 新增 ~60 个键
```

---

## Task 1: 数据模型 — `commission_rules` 表（含种子）

**Files:**
- Create: `model/commission_rule.go`
- Create: `model/commission_rule_test.go`
- Modify: `model/main.go`（AutoMigrate 追加）

- [ ] **Step 1: 写失败测试 `model/commission_rule_test.go`**

```go
package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommissionRule_ActiveRulesForFirstTopup(t *testing.T) {
	setupTestDB(t) // helper: creates in-memory SQLite + AutoMigrate

	require.NoError(t, SeedDefaultCommissionRules())

	rules, err := GetActiveCommissionRules(CommissionScopeFirstTopup)
	require.NoError(t, err)
	require.Len(t, rules, 2)

	byLevel := map[int]*CommissionRule{}
	for i := range rules {
		byLevel[rules[i].Level] = &rules[i]
	}

	require.Contains(t, byLevel, 1)
	require.Contains(t, byLevel, 2)
	require.InDelta(t, 20.0, byLevel[1].RatePercent, 0.001)
	require.InDelta(t, 5.0, byLevel[2].RatePercent, 0.001)
	require.Equal(t, 7, byLevel[1].FrozenDays)
	require.True(t, byLevel[1].Enabled)
}

func TestCommissionRule_DisabledFiltered(t *testing.T) {
	setupTestDB(t)
	require.NoError(t, SeedDefaultCommissionRules())

	require.NoError(t, DB.Model(&CommissionRule{}).
		Where("scope = ? AND level = ?", CommissionScopeFirstTopup, 2).
		Update("enabled", false).Error)

	rules, err := GetActiveCommissionRules(CommissionScopeFirstTopup)
	require.NoError(t, err)
	require.Len(t, rules, 1)
	require.Equal(t, 1, rules[0].Level)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./model/ -run TestCommissionRule -v`
Expected: FAIL - `CommissionRule` undefined 等。

- [ ] **Step 3: 实现 `model/commission_rule.go`**

```go
package model

import (
	"time"
)

const (
	CommissionScopeFirstTopup = "first_topup"
)

// CommissionRule defines a per-scope, per-level commission rule that admins can
// tune from the /system-settings/commission UI. The (scope, level) pair is
// unique; disabling a row is the operator's kill-switch for that tier without
// dropping historical records.
type CommissionRule struct {
	Id             int64   `json:"id" gorm:"primaryKey;autoIncrement"`
	Scope          string  `json:"scope" gorm:"type:varchar(32);not null;uniqueIndex:uk_scope_level,priority:1"`
	Level          int     `json:"level" gorm:"not null;uniqueIndex:uk_scope_level,priority:2"`
	RatePercent    float64 `json:"rate_percent" gorm:"type:decimal(5,2);not null"`
	MinTopupCents  int64   `json:"min_topup_cents" gorm:"not null;default:0"`
	FrozenDays     int     `json:"frozen_days" gorm:"not null;default:7"`
	Enabled        bool    `json:"enabled" gorm:"not null"` // default set in seed/CRUD, not gorm tag
	CreatedAt      int64   `json:"created_at" gorm:"not null"`
	UpdatedAt      int64   `json:"updated_at" gorm:"not null"`
}

func (CommissionRule) TableName() string { return "commission_rules" }

// GetActiveCommissionRules returns rules with enabled=true for the given scope,
// sorted by level ascending. Callers iterate the slice to pay each tier.
func GetActiveCommissionRules(scope string) ([]CommissionRule, error) {
	var rules []CommissionRule
	err := DB.Where("scope = ? AND enabled = ?", scope, true).
		Order("level ASC").
		Find(&rules).Error
	return rules, err
}

// ListCommissionRules returns every rule regardless of enabled state, for the
// admin config page.
func ListCommissionRules() ([]CommissionRule, error) {
	var rules []CommissionRule
	err := DB.Order("scope ASC, level ASC").Find(&rules).Error
	return rules, err
}

// UpdateCommissionRule mutates an existing rule identified by id.
func UpdateCommissionRule(id int64, updates map[string]any) error {
	updates["updated_at"] = time.Now().Unix()
	return DB.Model(&CommissionRule{}).Where("id = ?", id).Updates(updates).Error
}
```

- [ ] **Step 4: 实现 `SeedDefaultCommissionRules`（放 `model/commission_rule.go` 末尾）**

```go
// SeedDefaultCommissionRules inserts default L1=20% / L2=5% rules if the table
// is empty. Called once at startup. Idempotent: never overwrites existing rows.
func SeedDefaultCommissionRules() error {
	var count int64
	if err := DB.Model(&CommissionRule{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	now := time.Now().Unix()
	seeds := []CommissionRule{
		{Scope: CommissionScopeFirstTopup, Level: 1, RatePercent: 20.0, FrozenDays: 7, Enabled: true, CreatedAt: now, UpdatedAt: now},
		{Scope: CommissionScopeFirstTopup, Level: 2, RatePercent: 5.0, FrozenDays: 7, Enabled: true, CreatedAt: now, UpdatedAt: now},
	}
	return DB.Create(&seeds).Error
}
```

- [ ] **Step 5: 追加 AutoMigrate `model/main.go`**

在 `AutoMigrate(...)` 参数列表现有末行 `&AuthzRole{},` 之后新增：

```go
&CommissionRule{},
```

- [ ] **Step 6: 添加 `setupTestDB` helper (若不存在，`model/testutil_test.go`)**

Check first if `setupTestDB` exists in `model/` package; if not, create `model/testutil_test.go`:

```go
package model

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB initializes an in-memory SQLite DB with all the tables this
// package's tests need. It replaces the package-level DB so subsequent calls
// in the same process reuse the state until the test tears down.
func setupTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&CommissionRule{},
		&CommissionRecord{},
		&CommissionRedemption{},
		&UserCommissionStats{},
		&UserReferralPath{},
	))
	DB = db
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	})
}
```

If `setupTestDB` already exists in the package, extend its AutoMigrate list instead.

- [ ] **Step 7: 运行测试确认通过**

Run: `go test ./model/ -run TestCommissionRule -v`
Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add model/commission_rule.go model/commission_rule_test.go model/main.go model/testutil_test.go
git commit -m "feat(commission): add commission_rules table with default L1=20% / L2=5% seed"
```

---

## Task 2: 数据模型 — `user_referral_paths` 表

**Files:**
- Create: `model/user_referral_path.go`
- Create: `model/user_referral_path_test.go`
- Modify: `model/main.go`

- [ ] **Step 1: 写失败测试**

```go
package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserReferralPath_InsertAndQuery(t *testing.T) {
	setupTestDB(t)

	require.NoError(t, InsertReferralPath(100, 50, 20))

	path, err := GetReferralPath(100)
	require.NoError(t, err)
	require.NotNil(t, path)
	require.Equal(t, 50, path.L1UserId)
	require.Equal(t, 20, path.L2UserId)
}

func TestUserReferralPath_NoUpper(t *testing.T) {
	setupTestDB(t)
	require.NoError(t, InsertReferralPath(101, 50, 0))

	path, err := GetReferralPath(101)
	require.NoError(t, err)
	require.NotNil(t, path)
	require.Equal(t, 50, path.L1UserId)
	require.Equal(t, 0, path.L2UserId)
}

func TestUserReferralPath_NotFoundReturnsNilNilNoError(t *testing.T) {
	setupTestDB(t)
	path, err := GetReferralPath(999)
	require.NoError(t, err)
	require.Nil(t, path)
}

func TestUserReferralPath_DuplicateInsertIgnored(t *testing.T) {
	setupTestDB(t)
	require.NoError(t, InsertReferralPath(200, 30, 0))
	// Second insert must not error nor overwrite the L1
	require.NoError(t, InsertReferralPath(200, 40, 0))

	path, err := GetReferralPath(200)
	require.NoError(t, err)
	require.Equal(t, 30, path.L1UserId, "existing referral path must be preserved")
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./model/ -run TestUserReferralPath -v`
Expected: FAIL - undefined.

- [ ] **Step 3: 实现 `model/user_referral_path.go`**

```go
package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserReferralPath is a snapshot of a new user's L1/L2 uplines taken at the
// moment they register. It is INSERT-only: even if an upline later changes
// their own inviter, the downline's payout targets remain fixed.
type UserReferralPath struct {
	UserId     int   `json:"user_id" gorm:"primaryKey"`
	L1UserId   int   `json:"l1_user_id" gorm:"not null;index"`
	L2UserId   int   `json:"l2_user_id" gorm:"not null;default:0;index"`
	CreatedAt  int64 `json:"created_at" gorm:"not null"`
}

func (UserReferralPath) TableName() string { return "user_referral_paths" }

// InsertReferralPath stores the (L1, L2) snapshot for userId. Duplicate calls
// with the same userId are ignored so we never overwrite the original
// snapshot if register logic is somehow retried.
func InsertReferralPath(userId, l1UserId, l2UserId int) error {
	path := UserReferralPath{
		UserId:    userId,
		L1UserId:  l1UserId,
		L2UserId:  l2UserId,
		CreatedAt: time.Now().Unix(),
	}
	return DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&path).Error
}

// GetReferralPath returns nil (with no error) when the user has no path row.
// Errors are only returned for real DB failures.
func GetReferralPath(userId int) (*UserReferralPath, error) {
	var path UserReferralPath
	err := DB.Where("user_id = ?", userId).First(&path).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &path, nil
}
```

- [ ] **Step 4: AutoMigrate 追加**

In `model/main.go` next to `&CommissionRule{},` add:

```go
&UserReferralPath{},
```

- [ ] **Step 5: 运行测试确认通过**

Run: `go test ./model/ -run TestUserReferralPath -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add model/user_referral_path.go model/user_referral_path_test.go model/main.go
git commit -m "feat(commission): add user_referral_paths table (L1/L2 snapshot at register)"
```

---

## Task 3: 数据模型 — `user_commission_stats` 表（佣金账户）

**Files:**
- Create: `model/user_commission_stats.go`
- Create: `model/user_commission_stats_test.go`
- Modify: `model/main.go`

- [ ] **Step 1: 写失败测试**

```go
package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserCommissionStats_GetOrCreateInsertsRow(t *testing.T) {
	setupTestDB(t)

	stats, err := GetOrCreateCommissionStats(DB, 42)
	require.NoError(t, err)
	require.Equal(t, 42, stats.UserId)
	require.EqualValues(t, 0, stats.CommissionBalanceCents)
}

func TestUserCommissionStats_AddPendingBumpsCounter(t *testing.T) {
	setupTestDB(t)

	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return AddCommissionPending(tx, 42, 2500)
	}))

	stats, err := GetCommissionStats(42)
	require.NoError(t, err)
	require.EqualValues(t, 2500, stats.CommissionPendingCents)
	require.EqualValues(t, 0, stats.CommissionBalanceCents)
}

func TestUserCommissionStats_SettleMovesPendingToBalance(t *testing.T) {
	setupTestDB(t)

	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		if err := AddCommissionPending(tx, 42, 2500); err != nil {
			return err
		}
		return PendingToBalance(tx, 42, 2500)
	}))

	stats, _ := GetCommissionStats(42)
	require.EqualValues(t, 0, stats.CommissionPendingCents)
	require.EqualValues(t, 2500, stats.CommissionBalanceCents)
	require.EqualValues(t, 2500, stats.CommissionLifetimeCents)
}

func TestUserCommissionStats_RedeemDeductsBalance(t *testing.T) {
	setupTestDB(t)

	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		if err := AddCommissionPending(tx, 42, 3000); err != nil {
			return err
		}
		if err := PendingToBalance(tx, 42, 3000); err != nil {
			return err
		}
		return DeductCommissionBalance(tx, 42, 1000)
	}))

	stats, _ := GetCommissionStats(42)
	require.EqualValues(t, 2000, stats.CommissionBalanceCents)
	require.EqualValues(t, 1000, stats.CommissionRedeemedCents)
	require.EqualValues(t, 3000, stats.CommissionLifetimeCents, "lifetime is never decreased")
}

func TestUserCommissionStats_DeductRejectsInsufficient(t *testing.T) {
	setupTestDB(t)
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return AddCommissionPending(tx, 42, 100)
	}))

	err := DB.Transaction(func(tx *gorm.DB) error {
		return DeductCommissionBalance(tx, 42, 500)
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient")
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./model/ -run TestUserCommissionStats -v`
Expected: FAIL - undefined.

- [ ] **Step 3: 实现 `model/user_commission_stats.go`**

```go
package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserCommissionStats stores per-user commission counters. Rows are lazily
// created the first time a user has any commission activity (pending / settle
// / redeem). Users who never participate in referrals never get a row.
type UserCommissionStats struct {
	UserId                   int   `json:"user_id" gorm:"primaryKey"`
	CommissionBalanceCents   int64 `json:"commission_balance_cents" gorm:"not null;default:0"`
	CommissionPendingCents   int64 `json:"commission_pending_cents" gorm:"not null;default:0"`
	CommissionLifetimeCents  int64 `json:"commission_lifetime_cents" gorm:"not null;default:0"`
	CommissionRedeemedCents  int64 `json:"commission_redeemed_cents" gorm:"not null;default:0"`
	CreatedAt                int64 `json:"created_at" gorm:"not null"`
	UpdatedAt                int64 `json:"updated_at" gorm:"not null"`
}

func (UserCommissionStats) TableName() string { return "user_commission_stats" }

// GetOrCreateCommissionStats returns the row for userId or lazily inserts it.
// Must be called inside a transaction (tx) if it is part of a larger unit of
// work; ad-hoc reads can pass DB.
func GetOrCreateCommissionStats(tx *gorm.DB, userId int) (*UserCommissionStats, error) {
	stats := UserCommissionStats{
		UserId:    userId,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}
	// ON CONFLICT DO NOTHING keeps the earliest row.
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&stats).Error; err != nil {
		return nil, err
	}
	var out UserCommissionStats
	if err := tx.Where("user_id = ?", userId).First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

// GetCommissionStats returns nil when the row is absent, for lookup on the
// dashboard without eagerly inserting.
func GetCommissionStats(userId int) (*UserCommissionStats, error) {
	var out UserCommissionStats
	err := DB.Where("user_id = ?", userId).First(&out).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// AddCommissionPending increases pending_cents by cents inside tx. If the row
// does not exist yet it is created first.
func AddCommissionPending(tx *gorm.DB, userId int, cents int64) error {
	if _, err := GetOrCreateCommissionStats(tx, userId); err != nil {
		return err
	}
	return tx.Model(&UserCommissionStats{}).
		Where("user_id = ?", userId).
		Updates(map[string]any{
			"commission_pending_cents": gorm.Expr("commission_pending_cents + ?", cents),
			"updated_at":               time.Now().Unix(),
		}).Error
}

// PendingToBalance moves cents from pending to balance and bumps lifetime.
func PendingToBalance(tx *gorm.DB, userId int, cents int64) error {
	return tx.Model(&UserCommissionStats{}).
		Where("user_id = ?", userId).
		Updates(map[string]any{
			"commission_pending_cents":  gorm.Expr("commission_pending_cents - ?", cents),
			"commission_balance_cents":  gorm.Expr("commission_balance_cents + ?", cents),
			"commission_lifetime_cents": gorm.Expr("commission_lifetime_cents + ?", cents),
			"updated_at":                time.Now().Unix(),
		}).Error
}

// DeductCommissionPending is used by AdminVoid when the record is still pending.
func DeductCommissionPending(tx *gorm.DB, userId int, cents int64) error {
	res := tx.Model(&UserCommissionStats{}).
		Where("user_id = ? AND commission_pending_cents >= ?", userId, cents).
		Updates(map[string]any{
			"commission_pending_cents": gorm.Expr("commission_pending_cents - ?", cents),
			"updated_at":               time.Now().Unix(),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("insufficient pending commission balance")
	}
	return nil
}

// DeductCommissionBalance is used by AdminVoid on a settled (but not yet
// redeemed) record, and by Redeem when the user manually converts commission
// into wallet quota. Fails if balance is short.
func DeductCommissionBalance(tx *gorm.DB, userId int, cents int64) error {
	res := tx.Model(&UserCommissionStats{}).
		Where("user_id = ? AND commission_balance_cents >= ?", userId, cents).
		Updates(map[string]any{
			"commission_balance_cents":  gorm.Expr("commission_balance_cents - ?", cents),
			"commission_redeemed_cents": gorm.Expr("commission_redeemed_cents + ?", cents),
			"updated_at":                time.Now().Unix(),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("insufficient commission balance")
	}
	return nil
}
```

- [ ] **Step 4: AutoMigrate 追加**

In `model/main.go` add `&UserCommissionStats{},`

- [ ] **Step 5: 运行测试确认通过**

Run: `go test ./model/ -run TestUserCommissionStats -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add model/user_commission_stats.go model/user_commission_stats_test.go model/main.go
git commit -m "feat(commission): add user_commission_stats table with atomic counter helpers"
```

---

## Task 4: 数据模型 — `commission_records` 表 + 首充判定

**Files:**
- Create: `model/commission_record.go`
- Create: `model/commission_record_test.go`
- Modify: `model/main.go`

- [ ] **Step 1: 写失败测试**

```go
package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommissionRecord_InsertIsIdempotent(t *testing.T) {
	setupTestDB(t)
	rec := CommissionRecord{
		BeneficiaryId: 50, SourceUserId: 100, SourceTopupId: 999,
		Scope: CommissionScopeFirstTopup, Level: 1, RatePercent: 20,
		BaseAmountCents: 10000, CommissionAmountCents: 2000,
		Status: CommissionStatusPending, FrozenUntil: 1700000000,
	}
	ok, err := InsertCommissionRecord(DB, &rec)
	require.NoError(t, err); require.True(t, ok)
	rec2 := rec
	ok2, err := InsertCommissionRecord(DB, &rec2)
	require.NoError(t, err); require.False(t, ok2)
	var n int64
	require.NoError(t, DB.Model(&CommissionRecord{}).Count(&n).Error)
	require.EqualValues(t, 1, n)
}

func TestIsFirstDomesticSuccessTopup(t *testing.T) {
	setupTestDB(t)
	require.NoError(t, DB.Create(&TopUp{Id: 1, UserId: 42, Money: 100, Status: "success", PaymentProvider: PaymentProviderAlipay}).Error)
	require.NoError(t, DB.Create(&TopUp{Id: 2, UserId: 42, Money: 50, Status: "success", PaymentProvider: PaymentProviderStripe}).Error)
	require.True(t, IsFirstDomesticSuccessTopup(42, 1))
	require.NoError(t, DB.Create(&TopUp{Id: 3, UserId: 42, Money: 200, Status: "success", PaymentProvider: PaymentProviderWechat}).Error)
	require.False(t, IsFirstDomesticSuccessTopup(42, 3))
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./model/ -run "TestCommissionRecord|TestIsFirstDomestic" -v`
Expected: FAIL undefined.

- [ ] **Step 3: 实现 `model/commission_record.go`**

```go
package model

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/QuantumNous/new-api/common"
)

const (
	CommissionStatusPending = "pending"
	CommissionStatusSettled = "settled"
	CommissionStatusVoided  = "voided"
)

type CommissionRecord struct {
	Id                    int64   `json:"id" gorm:"primaryKey;autoIncrement"`
	BeneficiaryId         int     `json:"beneficiary_id" gorm:"not null;index:idx_beneficiary_status,priority:1;uniqueIndex:uk_topup_bene_level,priority:2"`
	SourceUserId          int     `json:"source_user_id" gorm:"not null;index"`
	SourceTopupId         int64   `json:"source_topup_id" gorm:"not null;uniqueIndex:uk_topup_bene_level,priority:1"`
	Scope                 string  `json:"scope" gorm:"type:varchar(32);not null"`
	Level                 int     `json:"level" gorm:"not null;uniqueIndex:uk_topup_bene_level,priority:3"`
	RatePercent           float64 `json:"rate_percent" gorm:"type:decimal(5,2);not null"`
	BaseAmountCents       int64   `json:"base_amount_cents" gorm:"not null"`
	CommissionAmountCents int64   `json:"commission_amount_cents" gorm:"not null"`
	Status                string  `json:"status" gorm:"type:varchar(16);not null;index:idx_beneficiary_status,priority:2;index:idx_pending_frozen,priority:1"`
	FrozenUntil           int64   `json:"frozen_until" gorm:"not null;index:idx_pending_frozen,priority:2"`
	SettledAt             int64   `json:"settled_at" gorm:"not null;default:0"`
	VoidedAt              int64   `json:"voided_at" gorm:"not null;default:0"`
	VoidedReason          string  `json:"voided_reason" gorm:"type:varchar(255)"`
	CreatedAt             int64   `json:"created_at" gorm:"not null"`
	UpdatedAt             int64   `json:"updated_at" gorm:"not null"`
}

func (CommissionRecord) TableName() string { return "commission_records" }

func InsertCommissionRecord(tx *gorm.DB, rec *CommissionRecord) (bool, error) {
	now := time.Now().Unix()
	rec.CreatedAt = now
	rec.UpdatedAt = now
	res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(rec)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

func FetchPendingDueCommissionRecords(limit int) ([]CommissionRecord, error) {
	var out []CommissionRecord
	err := DB.Where("status = ? AND frozen_until <= ?", CommissionStatusPending, time.Now().Unix()).
		Order("id ASC").Limit(limit).Find(&out).Error
	return out, err
}

func IsFirstDomesticSuccessTopup(userId int, topupId int64) bool {
	var count int64
	err := DB.Model(&TopUp{}).
		Where("user_id = ? AND status = ? AND id <= ? AND payment_provider IN ?",
			userId, common.TopUpStatusSuccess, topupId,
			[]string{PaymentProviderAlipay, PaymentProviderWechat, PaymentProviderEpay}).
		Count(&count).Error
	if err != nil {
		return false
	}
	return count == 1
}

func GetCommissionRecordByID(id int64) (*CommissionRecord, error) {
	var out CommissionRecord
	if err := DB.Where("id = ?", id).First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}
```

- [ ] **Step 4: AutoMigrate 追加** — `model/main.go` add `&CommissionRecord{},`

- [ ] **Step 5: 运行测试确认通过**

Run: `go test ./model/ -run "TestCommissionRecord|TestIsFirstDomestic" -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add model/commission_record.go model/commission_record_test.go model/main.go
git commit -m "feat(commission): add commission_records table + first-topup detector"
```

---

## Task 5: 数据模型 — `commission_redemptions` 表

**Files:**
- Create: `model/commission_redemption.go`
- Create: `model/commission_redemption_test.go`
- Modify: `model/main.go`

- [ ] **Step 1: 写失败测试**

```go
package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommissionRedemption_Insert(t *testing.T) {
	setupTestDB(t)
	r := CommissionRedemption{
		UserId: 42, CommissionCents: 2500,
		USDExchangeRate: 7.2, QuotaPerUnit: 500000, QuotaCredited: 1736111,
	}
	require.NoError(t, InsertCommissionRedemption(DB, &r))
	require.NotZero(t, r.Id)
	require.NotZero(t, r.CreatedAt)

	list, err := ListRedemptionsByUser(42, 10, 0)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.EqualValues(t, 2500, list[0].CommissionCents)
	require.InDelta(t, 7.2, list[0].USDExchangeRate, 0.001)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./model/ -run TestCommissionRedemption -v`
Expected: FAIL undefined.

- [ ] **Step 3: 实现 `model/commission_redemption.go`**

```go
package model

import (
	"time"

	"gorm.io/gorm"
)

type CommissionRedemption struct {
	Id              int64   `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId          int     `json:"user_id" gorm:"not null;index:idx_user_created,priority:1"`
	CommissionCents int64   `json:"commission_cents" gorm:"not null"`
	USDExchangeRate float64 `json:"usd_exchange_rate" gorm:"type:decimal(10,4);not null"`
	QuotaPerUnit    int64   `json:"quota_per_unit" gorm:"not null"`
	QuotaCredited   int64   `json:"quota_credited" gorm:"not null"`
	CreatedAt       int64   `json:"created_at" gorm:"not null;index:idx_user_created,priority:2"`
}

func (CommissionRedemption) TableName() string { return "commission_redemptions" }

func InsertCommissionRedemption(tx *gorm.DB, r *CommissionRedemption) error {
	r.CreatedAt = time.Now().Unix()
	return tx.Create(r).Error
}

func ListRedemptionsByUser(userId, limit, offset int) ([]CommissionRedemption, error) {
	var out []CommissionRedemption
	err := DB.Where("user_id = ?", userId).
		Order("id DESC").
		Limit(limit).Offset(offset).
		Find(&out).Error
	return out, err
}
```

- [ ] **Step 4: AutoMigrate 追加** — `model/main.go` add `&CommissionRedemption{},`

- [ ] **Step 5: 运行测试确认通过**

Run: `go test ./model/ -run TestCommissionRedemption -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add model/commission_redemption.go model/commission_redemption_test.go model/main.go
git commit -m "feat(commission): add commission_redemptions table (append-only)"
```

---

## Task 6: Service 层 — 邀请路径建立 (`service/commission/path.go`)

**Files:**
- Create: `service/commission/path.go`
- Create: `service/commission/path_test.go`

- [ ] **Step 1: 写失败测试**

```go
package commission

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/QuantumNous/new-api/model"
)

func TestBuildReferralPath_NoInviterNoOp(t *testing.T) {
	model.SetupTestDBForCommissionTests(t)

	require.NoError(t, BuildReferralPath(100, 0))

	p, err := model.GetReferralPath(100)
	require.NoError(t, err)
	require.Nil(t, p)
}

func TestBuildReferralPath_L1Only(t *testing.T) {
	model.SetupTestDBForCommissionTests(t)

	// Inviter (id=50) has no upline of their own.
	require.NoError(t, model.DB.Create(&model.User{Id: 50, InviterId: 0}).Error)
	require.NoError(t, BuildReferralPath(100, 50))

	p, err := model.GetReferralPath(100)
	require.NoError(t, err)
	require.NotNil(t, p)
	require.Equal(t, 50, p.L1UserId)
	require.Equal(t, 0, p.L2UserId)
}

func TestBuildReferralPath_L1AndL2Snapshot(t *testing.T) {
	model.SetupTestDBForCommissionTests(t)

	require.NoError(t, model.DB.Create(&model.User{Id: 20, InviterId: 0}).Error)
	require.NoError(t, model.DB.Create(&model.User{Id: 50, InviterId: 20}).Error)

	require.NoError(t, BuildReferralPath(100, 50))

	p, _ := model.GetReferralPath(100)
	require.Equal(t, 50, p.L1UserId)
	require.Equal(t, 20, p.L2UserId, "L2 must be inviter's inviter, snapshotted at register time")
}
```

- [ ] **Step 2: 添加 helper 到 `model` 包**

In `model/testutil_test.go` (extend if exists, otherwise create) — but this test lives in `service/commission`, which cannot import `model`'s test helper. Instead expose a public function.

Create `model/testing.go` (production file, not `_test.go`, so `service/commission` tests can use it):

```go
package model

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// SetupTestDBForCommissionTests initializes an in-memory SQLite DB with the
// tables used by commission tests. Exported so tests in other packages
// (service/commission) can call it. Never call from production code.
func SetupTestDBForCommissionTests(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&User{}, &TopUp{}, &Log{},
		&CommissionRule{}, &CommissionRecord{}, &CommissionRedemption{},
		&UserCommissionStats{}, &UserReferralPath{},
	))
	DB = db
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil { _ = sqlDB.Close() }
	})
}
```

- [ ] **Step 3: 运行测试确认失败**

Run: `go test ./service/commission/ -run TestBuildReferralPath -v`
Expected: FAIL undefined.

- [ ] **Step 4: 实现 `service/commission/path.go`**

```go
package commission

import (
	"errors"

	"gorm.io/gorm"

	"github.com/QuantumNous/new-api/model"
)

// BuildReferralPath persists the (L1, L2) snapshot for a newly registered
// user. It reads L1's own inviter_id as L2 so the downline pays out to the
// people who were L1's upline at register time — not whoever L1 later moves to.
// A zero inviterID means the new user registered without an invitation, so we
// insert nothing.
func BuildReferralPath(newUserID, inviterID int) error {
	if newUserID <= 0 || inviterID <= 0 {
		return nil
	}
	var inviter model.User
	if err := model.DB.Select("inviter_id").
		Where("id = ?", inviterID).
		First(&inviter).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// inviter was hard-deleted between AffCode lookup and now; skip L2.
			return model.InsertReferralPath(newUserID, inviterID, 0)
		}
		return err
	}
	return model.InsertReferralPath(newUserID, inviterID, inviter.InviterId)
}
```

- [ ] **Step 5: 运行测试确认通过**

Run: `go test ./service/commission/ -run TestBuildReferralPath -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add service/commission/path.go service/commission/path_test.go model/testing.go
git commit -m "feat(commission): implement BuildReferralPath with L2 snapshot at register"
```

---

## Task 7: Service 层 — 佣金写入 (`service/commission/record.go`)

**Files:**
- Create: `service/commission/record.go`
- Create: `service/commission/record_test.go`

- [ ] **Step 1: 写失败测试 `service/commission/record_test.go`**

```go
package commission

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

func setupCommissionScenario(t *testing.T, l1, l2 int) {
	t.Helper()
	model.SetupTestDBForCommissionTests(t)
	require.NoError(t, model.SeedDefaultCommissionRules())
	if l2 != 0 {
		require.NoError(t, model.DB.Create(&model.User{Id: l2, InviterId: 0}).Error)
	}
	if l1 != 0 {
		require.NoError(t, model.DB.Create(&model.User{Id: l1, InviterId: l2}).Error)
	}
}

func makeTopup(id int, userId int, money float64, provider string) *model.TopUp {
	return &model.TopUp{Id: id, UserId: userId, Money: money, Status: common.TopUpStatusSuccess, PaymentProvider: provider}
}

func TestOnTopupCompleted_FirstAlipayPaysBothTiers(t *testing.T) {
	setupCommissionScenario(t, 50, 20)
	require.NoError(t, model.DB.Create(&model.User{Id: 100, InviterId: 50}).Error)
	require.NoError(t, BuildReferralPath(100, 50))

	topup := makeTopup(1, 100, 100, model.PaymentProviderAlipay)
	require.NoError(t, model.DB.Create(topup).Error)
	OnTopupCompleted(topup)

	var recs []model.CommissionRecord
	require.NoError(t, model.DB.Order("level ASC").Find(&recs).Error)
	require.Len(t, recs, 2)
	require.EqualValues(t, 2000, recs[0].CommissionAmountCents) // L1: 10000*20% = 2000
	require.EqualValues(t, 500, recs[1].CommissionAmountCents)  // L2: 10000*5% = 500
}

func TestOnTopupCompleted_StripeIgnored(t *testing.T) {
	setupCommissionScenario(t, 50, 20)
	require.NoError(t, model.DB.Create(&model.User{Id: 100, InviterId: 50}).Error)
	require.NoError(t, BuildReferralPath(100, 50))

	topup := makeTopup(1, 100, 100, model.PaymentProviderStripe)
	require.NoError(t, model.DB.Create(topup).Error)
	OnTopupCompleted(topup)

	var n int64
	model.DB.Model(&model.CommissionRecord{}).Count(&n)
	require.EqualValues(t, 0, n)
}

func TestOnTopupCompleted_ReplayIsIdempotent(t *testing.T) {
	setupCommissionScenario(t, 50, 20)
	require.NoError(t, model.DB.Create(&model.User{Id: 100, InviterId: 50}).Error)
	require.NoError(t, BuildReferralPath(100, 50))

	topup := makeTopup(1, 100, 100, model.PaymentProviderAlipay)
	require.NoError(t, model.DB.Create(topup).Error)

	OnTopupCompleted(topup)
	OnTopupCompleted(topup) // replay

	var n int64
	model.DB.Model(&model.CommissionRecord{}).Count(&n)
	require.EqualValues(t, 2, n, "still 2 records total (one per level)")

	s1, _ := model.GetCommissionStats(50)
	require.EqualValues(t, 2000, s1.CommissionPendingCents, "pending counter must not double-bump on replay")
}

func TestOnTopupCompleted_L2ZeroSkipsSecondTier(t *testing.T) {
	setupCommissionScenario(t, 50, 0)
	require.NoError(t, model.DB.Create(&model.User{Id: 100, InviterId: 50}).Error)
	require.NoError(t, BuildReferralPath(100, 50))

	topup := makeTopup(1, 100, 100, model.PaymentProviderAlipay)
	require.NoError(t, model.DB.Create(topup).Error)
	OnTopupCompleted(topup)

	var recs []model.CommissionRecord
	model.DB.Find(&recs)
	require.Len(t, recs, 1)
	require.Equal(t, 1, recs[0].Level)
}

func TestOnTopupCompleted_DisabledLevelSkipped(t *testing.T) {
	setupCommissionScenario(t, 50, 20)
	require.NoError(t, model.DB.Model(&model.CommissionRule{}).
		Where("scope = ? AND level = ?", model.CommissionScopeFirstTopup, 2).
		Update("enabled", false).Error)
	require.NoError(t, model.DB.Create(&model.User{Id: 100, InviterId: 50}).Error)
	require.NoError(t, BuildReferralPath(100, 50))

	topup := makeTopup(1, 100, 100, model.PaymentProviderAlipay)
	require.NoError(t, model.DB.Create(topup).Error)
	OnTopupCompleted(topup)

	var recs []model.CommissionRecord
	model.DB.Find(&recs)
	require.Len(t, recs, 1)
}

func TestOnTopupCompleted_NotFirstIgnored(t *testing.T) {
	setupCommissionScenario(t, 50, 20)
	require.NoError(t, model.DB.Create(&model.User{Id: 100, InviterId: 50}).Error)
	require.NoError(t, BuildReferralPath(100, 50))
	require.NoError(t, model.DB.Create(makeTopup(1, 100, 50, model.PaymentProviderAlipay)).Error)

	topup2 := makeTopup(2, 100, 200, model.PaymentProviderWechat)
	require.NoError(t, model.DB.Create(topup2).Error)
	OnTopupCompleted(topup2)

	var n int64
	model.DB.Model(&model.CommissionRecord{}).Count(&n)
	require.EqualValues(t, 0, n)
}

func TestOnTopupCompleted_FrozenUntilFromRule(t *testing.T) {
	setupCommissionScenario(t, 50, 0)
	require.NoError(t, model.DB.Model(&model.CommissionRule{}).
		Where("scope = ? AND level = ?", model.CommissionScopeFirstTopup, 1).
		Update("frozen_days", 3).Error)
	require.NoError(t, model.DB.Create(&model.User{Id: 100, InviterId: 50}).Error)
	require.NoError(t, BuildReferralPath(100, 50))

	topup := makeTopup(1, 100, 100, model.PaymentProviderAlipay)
	require.NoError(t, model.DB.Create(topup).Error)
	before := time.Now().Unix()
	OnTopupCompleted(topup)

	var rec model.CommissionRecord
	require.NoError(t, model.DB.First(&rec).Error)
	require.InDelta(t, before+3*86400, rec.FrozenUntil, 60)
}

func TestOnTopupCompleted_CentsFloor(t *testing.T) {
	setupCommissionScenario(t, 50, 0)
	require.NoError(t, model.DB.Model(&model.CommissionRule{}).
		Where("scope = ? AND level = ?", model.CommissionScopeFirstTopup, 1).
		Update("rate_percent", 33.33).Error)
	require.NoError(t, model.DB.Create(&model.User{Id: 100, InviterId: 50}).Error)
	require.NoError(t, BuildReferralPath(100, 50))

	topup := makeTopup(1, 100, 1, model.PaymentProviderAlipay) // ¥1 → 100 cents × 33.33% = 33.33
	require.NoError(t, model.DB.Create(topup).Error)
	OnTopupCompleted(topup)

	var rec model.CommissionRecord
	require.NoError(t, model.DB.First(&rec).Error)
	require.EqualValues(t, int64(math.Floor(100*33.33/100.0)), rec.CommissionAmountCents)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./service/commission/ -run TestOnTopupCompleted -v`
Expected: FAIL undefined.

- [ ] **Step 3: 实现 `service/commission/record.go`**

```go
package commission

import (
	"fmt"
	"math"
	"time"

	"gorm.io/gorm"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

var domesticProviders = map[string]struct{}{
	model.PaymentProviderAlipay: {},
	model.PaymentProviderWechat: {},
	model.PaymentProviderEpay:   {},
}

// OnTopupCompleted is called from a payment webhook after the topup
// transaction has committed. It writes commission records for uplines when
// all business rules are met. Failures are logged but never propagated: the
// user's wallet has already been credited, so a downstream commission bug
// must not roll back the payment.
func OnTopupCompleted(topup *model.TopUp) {
	defer func() {
		if r := recover(); r != nil {
			common.SysError(fmt.Sprintf("commission recording panic: %v topup_id=%d", r, topup.Id))
		}
	}()

	if topup == nil || topup.Status != common.TopUpStatusSuccess {
		return
	}
	if _, ok := domesticProviders[topup.PaymentProvider]; !ok {
		return
	}
	if !model.IsFirstDomesticSuccessTopup(topup.UserId, int64(topup.Id)) {
		return
	}

	path, err := model.GetReferralPath(topup.UserId)
	if err != nil {
		common.SysError(fmt.Sprintf("commission path lookup error: %v user=%d", err, topup.UserId))
		return
	}
	if path == nil {
		return
	}

	rules, err := model.GetActiveCommissionRules(model.CommissionScopeFirstTopup)
	if err != nil || len(rules) == 0 {
		return
	}

	baseCents := int64(math.Round(topup.Money * 100))

	err = model.DB.Transaction(func(tx *gorm.DB) error {
		for _, rule := range rules {
			beneficiary := pickBeneficiary(path, rule.Level)
			if beneficiary == 0 {
				continue
			}
			if baseCents < rule.MinTopupCents {
				continue
			}
			amt := int64(math.Floor(float64(baseCents) * rule.RatePercent / 100.0))
			if amt <= 0 {
				continue
			}
			rec := &model.CommissionRecord{
				BeneficiaryId:         beneficiary,
				SourceUserId:          topup.UserId,
				SourceTopupId:         int64(topup.Id),
				Scope:                 model.CommissionScopeFirstTopup,
				Level:                 rule.Level,
				RatePercent:           rule.RatePercent,
				BaseAmountCents:       baseCents,
				CommissionAmountCents: amt,
				Status:                model.CommissionStatusPending,
				FrozenUntil:           time.Now().Unix() + int64(rule.FrozenDays)*86400,
			}
			inserted, err := model.InsertCommissionRecord(tx, rec)
			if err != nil {
				return err
			}
			if !inserted {
				continue // replay – already recorded
			}
			if err := model.AddCommissionPending(tx, beneficiary, amt); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		common.SysError(fmt.Sprintf("commission writing tx failed: %v topup=%d", err, topup.Id))
	}
}

func pickBeneficiary(path *model.UserReferralPath, level int) int {
	switch level {
	case 1:
		return path.L1UserId
	case 2:
		return path.L2UserId
	default:
		return 0
	}
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./service/commission/ -run TestOnTopupCompleted -v`
Expected: PASS (8 tests)

- [ ] **Step 5: Commit**

```bash
git add service/commission/record.go service/commission/record_test.go
git commit -m "feat(commission): implement OnTopupCompleted (first-topup, domestic-only, idempotent)"
```

---

## Task 8: Service 层 — 定时结算 (`service/commission/settle.go`)

**Files:**
- Create: `service/commission/settle.go`
- Create: `service/commission/settle_test.go`

- [ ] **Step 1: 写失败测试**

```go
package commission

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/QuantumNous/new-api/model"
)

func makePendingRecord(t *testing.T, id int64, beneficiary int, cents int64, frozenUntil int64) {
	t.Helper()
	rec := model.CommissionRecord{
		Id: id, BeneficiaryId: beneficiary, SourceUserId: 999,
		SourceTopupId: id, Scope: model.CommissionScopeFirstTopup, Level: 1, RatePercent: 20,
		BaseAmountCents: cents * 5, CommissionAmountCents: cents,
		Status: model.CommissionStatusPending, FrozenUntil: frozenUntil,
	}
	_, err := model.InsertCommissionRecord(model.DB, &rec)
	require.NoError(t, err)
	require.NoError(t, model.DB.Transaction(func(tx *gorm.DB) error {
		return model.AddCommissionPending(tx, beneficiary, cents)
	}))
}

func TestSettlePending_MovesDueRecords(t *testing.T) {
	model.SetupTestDBForCommissionTests(t)
	past := time.Now().Unix() - 10
	future := time.Now().Unix() + 3600

	makePendingRecord(t, 1, 50, 2000, past)
	makePendingRecord(t, 2, 50, 500, future)

	settled, err := SettlePending()
	require.NoError(t, err)
	require.Equal(t, 1, settled)

	var recs []model.CommissionRecord
	require.NoError(t, model.DB.Order("id ASC").Find(&recs).Error)
	require.Equal(t, model.CommissionStatusSettled, recs[0].Status)
	require.NotZero(t, recs[0].SettledAt)
	require.Equal(t, model.CommissionStatusPending, recs[1].Status)

	s, _ := model.GetCommissionStats(50)
	require.EqualValues(t, 2000, s.CommissionBalanceCents)
	require.EqualValues(t, 500, s.CommissionPendingCents, "future one still pending")
	require.EqualValues(t, 2000, s.CommissionLifetimeCents)
}

func TestSettlePending_NoDueRecordsReturnsZero(t *testing.T) {
	model.SetupTestDBForCommissionTests(t)
	makePendingRecord(t, 1, 50, 2000, time.Now().Unix()+3600)

	settled, err := SettlePending()
	require.NoError(t, err)
	require.Equal(t, 0, settled)
}

func TestSettlePending_HandlesLargeBatchInWaves(t *testing.T) {
	model.SetupTestDBForCommissionTests(t)
	past := time.Now().Unix() - 10
	for i := 1; i <= 600; i++ {
		makePendingRecord(t, int64(i), 50, 10, past)
	}
	settled, err := SettlePending()
	require.NoError(t, err)
	require.Equal(t, 600, settled)

	s, _ := model.GetCommissionStats(50)
	require.EqualValues(t, 600*10, s.CommissionBalanceCents)
}
```

Note: add `"gorm.io/gorm"` import (missing above only if go modules complain).

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./service/commission/ -run TestSettlePending -v`
Expected: FAIL undefined.

- [ ] **Step 3: 实现 `service/commission/settle.go`**

```go
package commission

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
)

const settleBatchLimit = 500

// SettlePending settles every pending record whose freeze window has elapsed.
// Returns the number of records settled. Called on a schedule by the
// system_tasks runner; also safe to invoke manually from the admin UI.
func SettlePending() (int, error) {
	total := 0
	for {
		batch, err := model.FetchPendingDueCommissionRecords(settleBatchLimit)
		if err != nil {
			return total, err
		}
		if len(batch) == 0 {
			return total, nil
		}
		for i := range batch {
			if err := settleOne(&batch[i]); err != nil {
				common.SysError(fmt.Sprintf("commission settle record failed: id=%d err=%v", batch[i].Id, err))
				continue
			}
			total++
		}
		if len(batch) < settleBatchLimit {
			return total, nil
		}
	}
}

func settleOne(rec *model.CommissionRecord) error {
	now := time.Now().Unix()
	return model.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&model.CommissionRecord{}).
			Where("id = ? AND status = ?", rec.Id, model.CommissionStatusPending).
			Updates(map[string]any{
				"status":     model.CommissionStatusSettled,
				"settled_at": now,
				"updated_at": now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil // someone else got to it (e.g., admin void)
		}
		if err := model.PendingToBalance(tx, rec.BeneficiaryId, rec.CommissionAmountCents); err != nil {
			return err
		}
		model.RecordLog(rec.BeneficiaryId, model.LogTypeSystem,
			fmt.Sprintf("推广佣金结算 ¥%.2f 来自 用户#%d 首充",
				float64(rec.CommissionAmountCents)/100.0, rec.SourceUserId))
		return nil
	})
}

// commissionSettleHandler exposes SettlePending to the system_tasks scheduler.
type commissionSettleHandler struct{}

func (commissionSettleHandler) Type() string          { return model.SystemTaskTypeCommissionSettle }
func (commissionSettleHandler) Enabled() bool         { return true }
func (commissionSettleHandler) Interval() time.Duration { return 10 * time.Minute }
func (commissionSettleHandler) NewPayload() any       { return map[string]any{} }

func (commissionSettleHandler) Run(ctx context.Context, task *model.SystemTask, runnerID string) {
	// Use model.FinishSystemTask contract used by other handlers.
	settled, err := SettlePending()
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("commission_settle task failed: %v", err))
		_ = model.FinishSystemTask(task.ID, runnerID, model.SystemTaskStatusFailed, map[string]any{"error": err.Error()}, nil)
		return
	}
	_ = model.FinishSystemTask(task.ID, runnerID, model.SystemTaskStatusSucceeded,
		map[string]any{"settled": settled}, nil)
}

// Register with the system_task runner. Called from an init() in this package
// so it fires at process start without a manual wiring step.
func init() {
	// Registration deferred to the settle-task wiring task (see Task 12) so
	// this file compiles even without the system_task RegisterSystemTaskHandler
	// symbol; the wiring task adds the actual call.
}
```

- [ ] **Step 4: 添加 `SystemTaskTypeCommissionSettle` 常量**

Edit `model/system_task.go` — in the const block that already defines `SystemTaskTypeLogCleanup`, add:

```go
SystemTaskTypeCommissionSettle = "commission_settle"
```

- [ ] **Step 5: 运行测试确认通过**

Run: `go test ./service/commission/ -run TestSettlePending -v`
Expected: PASS (3 tests)

- [ ] **Step 6: Commit**

```bash
git add service/commission/settle.go service/commission/settle_test.go model/system_task.go
git commit -m "feat(commission): implement SettlePending batched settlement + task-type const"
```

---

## Task 9: Service — 转余额兑换 (`service/commission/redeem.go`)

**Files:**
- Create: `service/commission/redeem.go`
- Create: `service/commission/redeem_test.go`

- [ ] **Step 1: 写失败测试**

```go
package commission

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

func seedBalance(t *testing.T, userId int, cents int64) {
	t.Helper()
	require.NoError(t, model.DB.Transaction(func(tx *gorm.DB) error {
		if err := model.AddCommissionPending(tx, userId, cents); err != nil { return err }
		return model.PendingToBalance(tx, userId, cents)
	}))
}

func TestRedeem_HappyPath(t *testing.T) {
	model.SetupTestDBForCommissionTests(t)
	require.NoError(t, model.DB.Create(&model.User{Id: 42, Quota: 0}).Error)
	seedBalance(t, 42, 2500)
	operation_setting.USDExchangeRate = 7.2
	common.QuotaPerUnit = 500000

	q, err := Redeem(42, 2500)
	require.NoError(t, err)
	require.Equal(t, int64(math.Floor(2500.0/100/7.2*500000)), q)

	u := model.User{}
	require.NoError(t, model.DB.First(&u, 42).Error)
	require.EqualValues(t, q, u.Quota)

	s, _ := model.GetCommissionStats(42)
	require.EqualValues(t, 0, s.CommissionBalanceCents)
	require.EqualValues(t, 2500, s.CommissionRedeemedCents)

	rs, _ := model.ListRedemptionsByUser(42, 10, 0)
	require.Len(t, rs, 1)
	require.InDelta(t, 7.2, rs[0].USDExchangeRate, 0.001)
}

func TestRedeem_InsufficientBalance(t *testing.T) {
	model.SetupTestDBForCommissionTests(t)
	require.NoError(t, model.DB.Create(&model.User{Id: 42, Quota: 0}).Error)
	seedBalance(t, 42, 100)
	operation_setting.USDExchangeRate = 7.2
	common.QuotaPerUnit = 500000

	_, err := Redeem(42, 500)
	require.Error(t, err)
}

func TestRedeem_ZeroCentsRejected(t *testing.T) {
	model.SetupTestDBForCommissionTests(t)
	_, err := Redeem(42, 0)
	require.Error(t, err)
}

func TestRedeem_ZeroRateRejected(t *testing.T) {
	model.SetupTestDBForCommissionTests(t)
	require.NoError(t, model.DB.Create(&model.User{Id: 42, Quota: 0}).Error)
	seedBalance(t, 42, 1000)
	operation_setting.USDExchangeRate = 0

	_, err := Redeem(42, 500)
	require.Error(t, err)
}

func TestRedeem_LaterRateChangeDoesNotRewriteHistory(t *testing.T) {
	model.SetupTestDBForCommissionTests(t)
	require.NoError(t, model.DB.Create(&model.User{Id: 42, Quota: 0}).Error)
	seedBalance(t, 42, 5000)
	common.QuotaPerUnit = 500000

	operation_setting.USDExchangeRate = 7.2
	_, err := Redeem(42, 2500)
	require.NoError(t, err)

	operation_setting.USDExchangeRate = 6.5
	_, err = Redeem(42, 2500)
	require.NoError(t, err)

	rs, _ := model.ListRedemptionsByUser(42, 10, 0)
	require.Len(t, rs, 2)
	require.InDelta(t, 6.5, rs[0].USDExchangeRate, 0.001) // newest first
	require.InDelta(t, 7.2, rs[1].USDExchangeRate, 0.001)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./service/commission/ -run TestRedeem -v`

- [ ] **Step 3: 实现 `service/commission/redeem.go`**

```go
package commission

import (
	"errors"
	"fmt"
	"math"

	"gorm.io/gorm"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

// PreviewQuota computes the quota the user would receive right now. Callers
// use it both for the /commission/quota-preview endpoint and internally in
// Redeem to keep the math in one place.
func PreviewQuota(commissionCents int64) (int64, float64, int64, error) {
	if commissionCents <= 0 {
		return 0, 0, 0, errors.New("金额 must be positive")
	}
	rate := operation_setting.USDExchangeRate
	qpu := int64(common.QuotaPerUnit)
	if rate <= 0 || qpu <= 0 {
		return 0, rate, qpu, errors.New("汇率未设置，请联系管理员")
	}
	quota := int64(math.Floor(float64(commissionCents) / 100.0 / rate * float64(qpu)))
	return quota, rate, qpu, nil
}

// Redeem converts commissionCents from the user's commission balance into
// wallet quota. Snapshots the rate and QuotaPerUnit so historical exchanges
// stay auditable if operators tune those knobs later.
func Redeem(userID int, commissionCents int64) (int64, error) {
	quotaCredited, rate, qpu, err := PreviewQuota(commissionCents)
	if err != nil {
		return 0, err
	}
	err = model.DB.Transaction(func(tx *gorm.DB) error {
		if err := model.DeductCommissionBalance(tx, userID, commissionCents); err != nil {
			return err
		}
		res := tx.Model(&model.User{}).Where("id = ?", userID).
			Update("quota", gorm.Expr("quota + ?", quotaCredited))
		if res.Error != nil { return res.Error }
		if res.RowsAffected == 0 { return errors.New("user not found") }
		if err := model.InsertCommissionRedemption(tx, &model.CommissionRedemption{
			UserId: userID, CommissionCents: commissionCents,
			USDExchangeRate: rate, QuotaPerUnit: qpu, QuotaCredited: quotaCredited,
		}); err != nil { return err }
		model.RecordLog(userID, model.LogTypeTopup,
			fmt.Sprintf("佣金 ¥%.2f 转入账户余额，+%d 额度（汇率 %.4f）",
				float64(commissionCents)/100.0, quotaCredited, rate))
		return nil
	})
	if err != nil { return 0, err }
	return quotaCredited, nil
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./service/commission/ -run TestRedeem -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add service/commission/redeem.go service/commission/redeem_test.go
git commit -m "feat(commission): implement Redeem with rate snapshot"
```

---

## Task 10: Service — 作废（管理员反作弊）

**Files:**
- Create: `service/commission/void.go`
- Create: `service/commission/void_test.go`

- [ ] **Step 1: 写失败测试**

```go
package commission

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/QuantumNous/new-api/model"
)

func insertRec(t *testing.T, id int64, ben int, cents int64, status string, frozen int64) {
	t.Helper()
	rec := &model.CommissionRecord{
		Id: id, BeneficiaryId: ben, SourceUserId: 999, SourceTopupId: id,
		Scope: model.CommissionScopeFirstTopup, Level: 1, RatePercent: 20,
		BaseAmountCents: cents * 5, CommissionAmountCents: cents,
		Status: status, FrozenUntil: frozen,
	}
	_, err := model.InsertCommissionRecord(model.DB, rec)
	require.NoError(t, err)
}

func TestVoid_PendingDeductsPending(t *testing.T) {
	model.SetupTestDBForCommissionTests(t)
	require.NoError(t, model.DB.Transaction(func(tx *gorm.DB) error {
		return model.AddCommissionPending(tx, 50, 1000)
	}))
	insertRec(t, 1, 50, 1000, model.CommissionStatusPending, time.Now().Unix()+3600)

	require.NoError(t, Void(1, "abuse detected"))

	rec, _ := model.GetCommissionRecordByID(1)
	require.Equal(t, model.CommissionStatusVoided, rec.Status)
	require.Equal(t, "abuse detected", rec.VoidedReason)

	s, _ := model.GetCommissionStats(50)
	require.EqualValues(t, 0, s.CommissionPendingCents)
}

func TestVoid_SettledDeductsBalance(t *testing.T) {
	model.SetupTestDBForCommissionTests(t)
	require.NoError(t, model.DB.Transaction(func(tx *gorm.DB) error {
		if err := model.AddCommissionPending(tx, 50, 1000); err != nil { return err }
		return model.PendingToBalance(tx, 50, 1000)
	}))
	insertRec(t, 1, 50, 1000, model.CommissionStatusSettled, time.Now().Unix()-3600)

	require.NoError(t, Void(1, "clawback"))

	s, _ := model.GetCommissionStats(50)
	require.EqualValues(t, 0, s.CommissionBalanceCents)
	require.EqualValues(t, 1000, s.CommissionLifetimeCents, "lifetime never decreases")
}

func TestVoid_AlreadyRedeemedRejected(t *testing.T) {
	model.SetupTestDBForCommissionTests(t)
	require.NoError(t, model.DB.Transaction(func(tx *gorm.DB) error {
		if err := model.AddCommissionPending(tx, 50, 1000); err != nil { return err }
		if err := model.PendingToBalance(tx, 50, 1000); err != nil { return err }
		return model.DeductCommissionBalance(tx, 50, 1000)
	}))
	insertRec(t, 1, 50, 1000, model.CommissionStatusSettled, 0)
	err := Void(1, "too late")
	require.Error(t, err)
}

func TestVoid_EmptyReasonRejected(t *testing.T) {
	model.SetupTestDBForCommissionTests(t)
	insertRec(t, 1, 50, 1000, model.CommissionStatusPending, time.Now().Unix()+3600)
	require.Error(t, Void(1, ""))
	require.Error(t, Void(1, "   "))
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./service/commission/ -run TestVoid -v`

- [ ] **Step 3: 实现 `service/commission/void.go`**

```go
package commission

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/QuantumNous/new-api/model"
)

// Void marks a commission record as voided and rolls back the counter.
// Lifetime totals are never rewound (they represent history, not owed amount).
func Void(recordID int64, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return errors.New("void reason is required")
	}
	return model.DB.Transaction(func(tx *gorm.DB) error {
		var rec model.CommissionRecord
		if err := tx.Where("id = ?", recordID).First(&rec).Error; err != nil {
			return err
		}
		if rec.Status == model.CommissionStatusVoided {
			return errors.New("record already voided")
		}
		now := time.Now().Unix()
		switch rec.Status {
		case model.CommissionStatusPending:
			if err := model.DeductCommissionPending(tx, rec.BeneficiaryId, rec.CommissionAmountCents); err != nil {
				return err
			}
		case model.CommissionStatusSettled:
			if err := model.DeductCommissionBalance(tx, rec.BeneficiaryId, rec.CommissionAmountCents); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unexpected status %q", rec.Status)
		}
		return tx.Model(&model.CommissionRecord{}).Where("id = ?", recordID).
			Updates(map[string]any{
				"status":        model.CommissionStatusVoided,
				"voided_at":     now,
				"voided_reason": reason,
				"updated_at":    now,
			}).Error
	})
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./service/commission/ -run TestVoid -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add service/commission/void.go service/commission/void_test.go
git commit -m "feat(commission): implement admin Void with status-aware counter rollback"
```

---

## Task 11: Service — 种子初始化钩子 + 启动接入

**Files:**
- Create: `service/commission/seed.go`
- Modify: `main.go`

- [ ] **Step 1: 实现 `service/commission/seed.go`**

```go
package commission

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// SeedDefaultRules is called at process start after AutoMigrate. Idempotent.
func SeedDefaultRules() {
	if err := model.SeedDefaultCommissionRules(); err != nil {
		common.SysError("commission rules seed failed: " + err.Error())
	}
}
```

- [ ] **Step 2: 在 main.go 启动流程接入**

Find where `model.InitDB()` is invoked and, right after it returns nil, add:

```go
commission.SeedDefaultRules()
```

Import at top of `main.go`:

```go
"github.com/QuantumNous/new-api/service/commission"
```

- [ ] **Step 3: 构建整个项目验证**

Run: `go build ./...`
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add service/commission/seed.go main.go
git commit -m "feat(commission): seed default L1/L2 rules on startup"
```

---

## Task 12: 注册系统定时任务

**Files:**
- Modify: `service/system_task.go`（注册 handler）
- Modify: `service/commission/settle.go`（去掉先前空 init，替换为真的 register）

- [ ] **Step 1: 在 `service/commission/settle.go` 移除占位 init**

删除 `service/commission/settle.go` 底部空的 `init()` 函数（Task 8 里保留的占位）。

- [ ] **Step 2: 修改 `service/system_task.go` 追加 handler 注册**

在文件已有的 `init()`（`RegisterSystemTaskHandler(logCleanupHandler{})` 那一行下面）新增：

```go
import "github.com/QuantumNous/new-api/service/commission"

func init() {
    RegisterSystemTaskHandler(commission.CommissionSettleHandler{})
}
```

**注意：** 这里会导致导入循环（`service/system_task.go` → `service/commission` → 已导入 `service/*`）。为避免循环，改为在 `service/commission/settle.go` 里 export handler 类型，并在 `service/system_task_bootstrap.go`（新增）里做注册：

Create `service/system_task_bootstrap.go`:

```go
package service

import (
	"github.com/QuantumNous/new-api/service/commission"
)

func init() {
	RegisterSystemTaskHandler(commission.NewSettleHandler())
}
```

- [ ] **Step 3: 修改 `service/commission/settle.go` — Export handler struct**

Replace the `commissionSettleHandler` from Task 8 with an exported type:

```go
// SettleHandler adapts SettlePending to the system_tasks contract.
type SettleHandler struct{}

func NewSettleHandler() SettleHandler { return SettleHandler{} }

func (SettleHandler) Type() string          { return model.SystemTaskTypeCommissionSettle }
func (SettleHandler) Enabled() bool         { return true }
func (SettleHandler) Interval() time.Duration { return 10 * time.Minute }
func (SettleHandler) NewPayload() any       { return map[string]any{} }

func (SettleHandler) Run(ctx context.Context, task *model.SystemTask, runnerID string) {
	settled, err := SettlePending()
	if err != nil {
		logger.LogError(ctx, "commission_settle task failed: "+err.Error())
		_ = model.FinishSystemTask(task.ID, runnerID, model.SystemTaskStatusFailed,
			map[string]any{"error": err.Error()}, nil)
		return
	}
	_ = model.FinishSystemTask(task.ID, runnerID, model.SystemTaskStatusSucceeded,
		map[string]any{"settled": settled}, nil)
}
```

Delete the older placeholder `commissionSettleHandler` from Task 8.

- [ ] **Step 4: 构建验证**

Run: `go build ./...`
Expected: no errors, no import cycles.

- [ ] **Step 5: Commit**

```bash
git add service/system_task_bootstrap.go service/commission/settle.go
git commit -m "feat(commission): register SettleHandler with system_tasks runner (10-min interval)"
```

---

## Task 13: 注册用户注册钩子 (`controller/user.go`)

**Files:**
- Modify: `controller/user.go`（在 `Register` 中调用 `BuildReferralPath`）

- [ ] **Step 1: 阅读现有 Register 逻辑**

打开 [controller/user.go:178](controller/user.go#L178) 至 240 附近，定位 `cleanUser.Insert(inviterId)` 与之后获取 `insertedUser.Id` 的位置。

- [ ] **Step 2: 在 Register 中插入调用**

在 `Register` 函数中，`insertedUser.Id` 已被查询出来之后（大约 line 236 后），追加：

```go
// Build referral path snapshot for commission tracking (L1/L2 lookup).
if err := commission.BuildReferralPath(insertedUser.Id, inviterId); err != nil {
    common.SysError(fmt.Sprintf("BuildReferralPath failed user=%d inviter=%d err=%v", insertedUser.Id, inviterId, err))
}
```

Add import to top of file:

```go
"github.com/QuantumNous/new-api/service/commission"
```

- [ ] **Step 3: 加集成测试 (可选，若无法方便测试可 skip 到 Task 15 联调)**

如果 `controller/user.go` 有 register 的测试文件，添加断言 `user_referral_paths` 有对应行；否则跳过至端到端联调。

- [ ] **Step 4: 构建验证**

Run: `go build ./...`
Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add controller/user.go
git commit -m "feat(commission): wire BuildReferralPath into Register"
```

---

## Task 14: 支付回调埋点（3 处）

**Files:**
- Modify: `controller/topup_alipay.go`
- Modify: `controller/topup_wechat.go`
- Modify: `controller/topup.go`（EpayNotify）

- [ ] **Step 1: 修改 `controller/topup_alipay.go`**

Locate the callback handler that calls `model.RechargeAlipay(tradeNo, c.ClientIP())`. Immediately after a nil-error return, re-fetch the topup and fire commission:

```go
if err := model.RechargeAlipay(tradeNo, c.ClientIP()); err != nil {
    // ... existing error handling
    return
}
// ... existing success response ...

// Fire commission recording asynchronously. Failure never blocks payment.
if topUp := model.GetTopUpByTradeNo(tradeNo); topUp != nil {
    go commission.OnTopupCompleted(topUp)
}
```

Add import: `"github.com/QuantumNous/new-api/service/commission"`

- [ ] **Step 2: 修改 `controller/topup_wechat.go`（同样模式）**

Follow the same pattern next to `model.RechargeWechat(tradeNo, c.ClientIP())`.

- [ ] **Step 3: 修改 `controller/topup.go` EpayNotify 成功分支**

In `EpayNotify` where `topUp.Status = common.TopUpStatusSuccess` is set and `model.IncreaseUserQuota` succeeded (around [controller/topup.go:475](controller/topup.go#L475), right after the success `RecordTopupLog`), add:

```go
go commission.OnTopupCompleted(topUp)
```

Add import if missing: `"github.com/QuantumNous/new-api/service/commission"`

- [ ] **Step 4: 构建验证**

Run: `go build ./...`
Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add controller/topup_alipay.go controller/topup_wechat.go controller/topup.go
git commit -m "feat(commission): wire OnTopupCompleted into alipay/wechat/epay callbacks"
```

---

## Task 15: Controller — 用户端 HTTP 层

**Files:**
- Create: `controller/commission.go`
- Modify: `router/api-router.go`

- [ ] **Step 1: 实现 `controller/commission.go`（用户端 6 个 handler）**

```go
package controller

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/commission"
)

// GetCommissionStats returns the user's commission counters, aff_code, and
// referral link. Lazy-creates the stats row so the frontend can render zeroes
// on first load.
func GetCommissionStats(c *gin.Context) {
	uid := c.GetInt("id")
	user, err := model.GetUserById(uid, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	stats, err := model.GetCommissionStats(uid)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if stats == nil {
		stats = &model.UserCommissionStats{UserId: uid}
	}
	c.JSON(200, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"aff_code":                    user.AffCode,
			"commission_balance_cents":    stats.CommissionBalanceCents,
			"commission_pending_cents":    stats.CommissionPendingCents,
			"commission_lifetime_cents":   stats.CommissionLifetimeCents,
			"commission_redeemed_cents":   stats.CommissionRedeemedCents,
		},
	})
}

// GetMyCommissionRecords returns the user's commission ledger, filtered by
// optional status query param.
func GetMyCommissionRecords(c *gin.Context) {
	uid := c.GetInt("id")
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 { page = 1 }
	if size < 1 || size > 100 { size = 20 }

	q := model.DB.Model(&model.CommissionRecord{}).Where("beneficiary_id = ?", uid)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	q.Count(&total)

	var records []model.CommissionRecord
	if err := q.Order("id DESC").Limit(size).Offset((page - 1) * size).Find(&records).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(200, gin.H{"success": true, "data": gin.H{"records": records, "total": total, "page": page, "size": size}})
}

// GetMyRedemptions returns the user's commission→wallet conversion trail.
func GetMyRedemptions(c *gin.Context) {
	uid := c.GetInt("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 { page = 1 }
	if size < 1 || size > 100 { size = 20 }
	list, err := model.ListRedemptionsByUser(uid, size, (page-1)*size)
	if err != nil { common.ApiError(c, err); return }
	c.JSON(200, gin.H{"success": true, "data": list})
}

// GetMyDownlines returns L1/L2 users invited by the caller, with PII masked.
func GetMyDownlines(c *gin.Context) {
	uid := c.GetInt("id")
	level := c.DefaultQuery("level", "1")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 { page = 1 }
	if size < 1 || size > 100 { size = 20 }

	q := model.DB.Table("user_referral_paths urp").
		Select("urp.user_id, urp.created_at, u.username, u.email, u.phone").
		Joins("LEFT JOIN users u ON u.id = urp.user_id")
	if level == "1" {
		q = q.Where("urp.l1_user_id = ?", uid)
	} else if level == "2" {
		q = q.Where("urp.l2_user_id = ?", uid)
	} else {
		c.JSON(200, gin.H{"success": false, "message": "level must be 1 or 2"})
		return
	}
	var total int64
	q.Count(&total)

	rows := []struct {
		UserId    int    `json:"user_id"`
		CreatedAt int64  `json:"created_at"`
		Username  string `json:"username"`
		Email     string `json:"email"`
		Phone     string `json:"phone"`
	}{}
	if err := q.Order("urp.created_at DESC").Limit(size).Offset((page-1)*size).Scan(&rows).Error; err != nil {
		common.ApiError(c, err); return
	}
	// Mask PII per the design spec.
	for i := range rows {
		rows[i].Username = maskUsername(rows[i].Username)
		rows[i].Email = maskEmail(rows[i].Email)
		rows[i].Phone = maskPhone(rows[i].Phone)
	}
	c.JSON(200, gin.H{"success": true, "data": gin.H{"rows": rows, "total": total, "page": page, "size": size}})
}

// GetQuotaPreview returns the wallet quota the user would receive if they
// redeemed `cents` right now. Fails cleanly if the operator hasn't set an
// exchange rate.
func GetQuotaPreview(c *gin.Context) {
	cents, _ := strconv.ParseInt(c.Query("cents"), 10, 64)
	quota, rate, qpu, err := commission.PreviewQuota(cents)
	if err != nil {
		c.JSON(200, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"success": true, "data": gin.H{"quota_credited": quota, "usd_exchange_rate": rate, "quota_per_unit": qpu}})
}

type redeemRequest struct {
	Cents int64 `json:"cents"`
}

// RedeemCommission converts commission balance into wallet quota.
func RedeemCommission(c *gin.Context) {
	uid := c.GetInt("id")
	var req redeemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"success": false, "message": "参数错误"})
		return
	}
	q, err := commission.Redeem(uid, req.Cents)
	if err != nil {
		c.JSON(200, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"success": true, "data": gin.H{"quota_credited": q}})
}

// maskUsername keeps only the first rune.
func maskUsername(s string) string {
	rs := []rune(s)
	if len(rs) <= 1 { return s }
	return string(rs[:1]) + strings.Repeat("*", len(rs)-1)
}

// maskEmail keeps first char + first three chars of domain + tld.
func maskEmail(s string) string {
	at := strings.Index(s, "@")
	if at <= 1 { return s }
	local := s[:1] + "***"
	return local + s[at:]
}

// maskPhone keeps first 3 and last 4.
func maskPhone(s string) string {
	if len(s) < 8 { return s }
	return s[:3] + "****" + s[len(s)-4:]
}
```

- [ ] **Step 2: 路由注册 — 编辑 `router/api-router.go`**

Inside `selfRoute` block (after `selfRoute.PUT("/setting", controller.UpdateUserSetting)` around line 113), add:

```go
selfRoute.GET("/commission/stats", controller.GetCommissionStats)
selfRoute.GET("/commission/records", controller.GetMyCommissionRecords)
selfRoute.GET("/commission/redemptions", controller.GetMyRedemptions)
selfRoute.GET("/commission/downlines", controller.GetMyDownlines)
selfRoute.GET("/commission/quota-preview", controller.GetQuotaPreview)
selfRoute.POST("/commission/redeem", middleware.CriticalRateLimit(), controller.RedeemCommission)
```

- [ ] **Step 3: 构建验证**

Run: `go build ./...`
Expected: no errors.

- [ ] **Step 4: 加轻量 handler 测试 (可选) `controller/commission_test.go`**

```go
package controller

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMaskUsername(t *testing.T) {
	require.Equal(t, "张*", maskUsername("张三"))
	require.Equal(t, "a*****", maskUsername("abcdef"))
	require.Equal(t, "", maskUsername(""))
}

func TestMaskEmail(t *testing.T) {
	require.Equal(t, "t***@example.com", maskEmail("test@example.com"))
	require.Equal(t, "a", maskEmail("a")) // no @
}

func TestMaskPhone(t *testing.T) {
	require.Equal(t, "138****8000", maskPhone("13800138000"))
	require.Equal(t, "short", maskPhone("short"))
}
```

Run: `go test ./controller/ -run "TestMask" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add controller/commission.go controller/commission_test.go router/api-router.go
git commit -m "feat(commission): user-facing HTTP endpoints (6 handlers) with PII masking"
```

---

## Task 16: Controller — 管理员端 HTTP 层

**Files:**
- Modify: `controller/commission.go`（追加管理员 handler）
- Modify: `router/api-router.go`（追加管理员路由分组）

- [ ] **Step 1: 在 `controller/commission.go` 追加管理员 handler**

```go
// ---------- Admin endpoints ----------

// AdminListCommissionRules returns every rule (enabled or not).
func AdminListCommissionRules(c *gin.Context) {
	rules, err := model.ListCommissionRules()
	if err != nil { common.ApiError(c, err); return }
	c.JSON(200, gin.H{"success": true, "data": rules})
}

type adminUpdateRuleRequest struct {
	RatePercent    *float64 `json:"rate_percent"`
	MinTopupCents  *int64   `json:"min_topup_cents"`
	FrozenDays     *int     `json:"frozen_days"`
	Enabled        *bool    `json:"enabled"`
}

func AdminUpdateCommissionRule(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req adminUpdateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"success": false, "message": "参数错误"})
		return
	}
	updates := map[string]any{}
	if req.RatePercent != nil { updates["rate_percent"] = *req.RatePercent }
	if req.MinTopupCents != nil { updates["min_topup_cents"] = *req.MinTopupCents }
	if req.FrozenDays != nil { updates["frozen_days"] = *req.FrozenDays }
	if req.Enabled != nil { updates["enabled"] = *req.Enabled }
	if len(updates) == 0 {
		c.JSON(200, gin.H{"success": true})
		return
	}
	if err := model.UpdateCommissionRule(id, updates); err != nil {
		common.ApiError(c, err); return
	}
	c.JSON(200, gin.H{"success": true})
}

func AdminListRecords(c *gin.Context) {
	status := c.Query("status")
	beneficiary, _ := strconv.Atoi(c.Query("beneficiary"))
	source, _ := strconv.Atoi(c.Query("source"))
	from, _ := strconv.ParseInt(c.Query("from"), 10, 64)
	to, _ := strconv.ParseInt(c.Query("to"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 { page = 1 }
	if size < 1 || size > 100 { size = 20 }

	q := model.DB.Model(&model.CommissionRecord{})
	if status != "" { q = q.Where("status = ?", status) }
	if beneficiary != 0 { q = q.Where("beneficiary_id = ?", beneficiary) }
	if source != 0 { q = q.Where("source_user_id = ?", source) }
	if from > 0 { q = q.Where("created_at >= ?", from) }
	if to > 0 { q = q.Where("created_at <= ?", to) }
	var total int64
	q.Count(&total)
	var out []model.CommissionRecord
	if err := q.Order("id DESC").Limit(size).Offset((page-1)*size).Find(&out).Error; err != nil {
		common.ApiError(c, err); return
	}
	c.JSON(200, gin.H{"success": true, "data": gin.H{"records": out, "total": total, "page": page, "size": size}})
}

type adminVoidRequest struct {
	Reason string `json:"reason"`
}

func AdminVoidRecord(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req adminVoidRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if err := commission.Void(id, req.Reason); err != nil {
		c.JSON(200, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"success": true})
}

func AdminSettleNow(c *gin.Context) {
	n, err := commission.SettlePending()
	if err != nil {
		c.JSON(200, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"success": true, "data": gin.H{"settled": n}})
}

func AdminCommissionOverview(c *gin.Context) {
	type overview struct {
		TotalCents        int64 `json:"total_cents"`
		SettledCents      int64 `json:"settled_cents"`
		PendingCents      int64 `json:"pending_cents"`
		RedeemedCents     int64 `json:"redeemed_cents"`
		ParticipantsCount int64 `json:"participants_count"`
		FirstTopupCount   int64 `json:"first_topup_count"`
	}
	var out overview
	model.DB.Model(&model.CommissionRecord{}).
		Where("status IN ?", []string{model.CommissionStatusPending, model.CommissionStatusSettled}).
		Select("COALESCE(SUM(commission_amount_cents),0)").Scan(&out.TotalCents)
	model.DB.Model(&model.CommissionRecord{}).
		Where("status = ?", model.CommissionStatusSettled).
		Select("COALESCE(SUM(commission_amount_cents),0)").Scan(&out.SettledCents)
	model.DB.Model(&model.CommissionRecord{}).
		Where("status = ?", model.CommissionStatusPending).
		Select("COALESCE(SUM(commission_amount_cents),0)").Scan(&out.PendingCents)
	model.DB.Model(&model.CommissionRedemption{}).
		Select("COALESCE(SUM(commission_cents),0)").Scan(&out.RedeemedCents)
	model.DB.Model(&model.UserReferralPath{}).Count(&out.ParticipantsCount)
	model.DB.Model(&model.CommissionRecord{}).
		Where("level = ?", 1).
		Distinct("source_user_id").Count(&out.FirstTopupCount)

	c.JSON(200, gin.H{"success": true, "data": out})
}
```

- [ ] **Step 2: 路由注册**

In `router/api-router.go` — after the existing `commissionAdminRoute` is a new group. Add before `registerChannelRoutes`:

```go
commissionAdmin := apiRouter.Group("/commission-admin")
commissionAdmin.Use(middleware.RootAuth())
{
    commissionAdmin.GET("/rules", controller.AdminListCommissionRules)
    commissionAdmin.PUT("/rules/:id", controller.AdminUpdateCommissionRule)
    commissionAdmin.GET("/records", controller.AdminListRecords)
    commissionAdmin.POST("/records/:id/void", controller.AdminVoidRecord)
    commissionAdmin.POST("/settle-now", controller.AdminSettleNow)
    commissionAdmin.GET("/stats", controller.AdminCommissionOverview)
}
```

- [ ] **Step 3: 构建验证**

Run: `go build ./...`
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add controller/commission.go router/api-router.go
git commit -m "feat(commission): admin HTTP endpoints (rules, records, void, settle-now, overview)"
```

---

## Task 17: 前端 — 用户端 API 封装 + 类型

**Files:**
- Create: `web/default/src/features/referral/types.ts`
- Create: `web/default/src/features/referral/api.ts`
- Create: `web/default/src/features/referral/lib/format-commission.ts`
- Create: `web/default/src/features/referral/lib/format-commission.test.ts`

- [ ] **Step 1: `types.ts`**

```ts
/*
Copyright (C) 2023-2026 QuantumNous
...
*/
export type CommissionStats = {
  aff_code: string
  commission_balance_cents: number
  commission_pending_cents: number
  commission_lifetime_cents: number
  commission_redeemed_cents: number
}

export type CommissionRecord = {
  id: number
  beneficiary_id: number
  source_user_id: number
  source_topup_id: number
  scope: string
  level: number
  rate_percent: number
  base_amount_cents: number
  commission_amount_cents: number
  status: 'pending' | 'settled' | 'voided'
  frozen_until: number
  settled_at: number
  voided_at: number
  voided_reason: string
  created_at: number
}

export type CommissionRedemption = {
  id: number
  user_id: number
  commission_cents: number
  usd_exchange_rate: number
  quota_per_unit: number
  quota_credited: number
  created_at: number
}

export type CommissionDownline = {
  user_id: number
  created_at: number
  username: string
  email: string
  phone: string
}

export type CommissionQuotaPreview = {
  quota_credited: number
  usd_exchange_rate: number
  quota_per_unit: number
}
```

- [ ] **Step 2: `api.ts`**

```ts
/*
Copyright (C) 2023-2026 QuantumNous
...
*/
import { api } from '@/lib/api'
import type {
  CommissionDownline,
  CommissionQuotaPreview,
  CommissionRecord,
  CommissionRedemption,
  CommissionStats,
} from './types'

type ApiEnvelope<T> = { success: boolean; message?: string; data?: T }

export async function getCommissionStats(): Promise<CommissionStats> {
  const res = await api.get<ApiEnvelope<CommissionStats>>('/api/user/commission/stats')
  if (!res.data.success || !res.data.data) throw new Error(res.data.message || 'load failed')
  return res.data.data
}

export async function getCommissionRecords(params: {
  status?: string
  page?: number
  size?: number
}): Promise<{ records: CommissionRecord[]; total: number }> {
  const res = await api.get('/api/user/commission/records', { params })
  return res.data.data
}

export async function getCommissionRedemptions(params: {
  page?: number
  size?: number
}): Promise<CommissionRedemption[]> {
  const res = await api.get('/api/user/commission/redemptions', { params })
  return res.data.data ?? []
}

export async function getCommissionDownlines(params: {
  level: 1 | 2
  page?: number
  size?: number
}): Promise<{ rows: CommissionDownline[]; total: number }> {
  const res = await api.get('/api/user/commission/downlines', { params })
  return res.data.data
}

export async function previewQuotaCredit(cents: number): Promise<CommissionQuotaPreview> {
  const res = await api.get('/api/user/commission/quota-preview', { params: { cents } })
  if (!res.data.success) throw new Error(res.data.message)
  return res.data.data
}

export async function redeemCommission(cents: number): Promise<{ quota_credited: number }> {
  const res = await api.post('/api/user/commission/redeem', { cents })
  if (!res.data.success) throw new Error(res.data.message)
  return res.data.data
}
```

- [ ] **Step 3: `lib/format-commission.ts` + test**

```ts
export function centsToYuan(cents: number): string {
  return (cents / 100).toFixed(2)
}

export function yuanToCents(yuan: number): number {
  return Math.round(yuan * 100)
}

// Compute expected quota exactly the same way the backend does (floor).
export function computeQuotaCredit(cents: number, rate: number, quotaPerUnit: number): number {
  if (cents <= 0 || rate <= 0 || quotaPerUnit <= 0) return 0
  return Math.floor((cents / 100 / rate) * quotaPerUnit)
}
```

Test file `lib/format-commission.test.ts`:

```ts
import { describe, expect, test } from 'vitest'
import { centsToYuan, yuanToCents, computeQuotaCredit } from './format-commission'

describe('centsToYuan', () => {
  test('formats 2 decimals', () => {
    expect(centsToYuan(2500)).toBe('25.00')
    expect(centsToYuan(1)).toBe('0.01')
    expect(centsToYuan(0)).toBe('0.00')
  })
})

describe('yuanToCents', () => {
  test('rounds correctly', () => {
    expect(yuanToCents(25)).toBe(2500)
    expect(yuanToCents(0.1 + 0.2)).toBe(30) // handles float noise
  })
})

describe('computeQuotaCredit', () => {
  test('happy path matches backend floor', () => {
    // ¥25 @ rate=7.2, qpu=500000 → 25/7.2*500000 = 1736111.11 → floor 1736111
    expect(computeQuotaCredit(2500, 7.2, 500000)).toBe(1736111)
  })
  test('invalid inputs return 0', () => {
    expect(computeQuotaCredit(0, 7.2, 500000)).toBe(0)
    expect(computeQuotaCredit(2500, 0, 500000)).toBe(0)
    expect(computeQuotaCredit(2500, 7.2, 0)).toBe(0)
  })
})
```

Run: `cd web/default && bun run test lib/format-commission -run`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add web/default/src/features/referral/types.ts web/default/src/features/referral/api.ts \
        web/default/src/features/referral/lib/format-commission.ts \
        web/default/src/features/referral/lib/format-commission.test.ts
git commit -m "feat(referral): types + api client + format helpers (with tests)"
```

---

## Task 18: 前端 — React Query hooks（6 个）

**Files:** `web/default/src/features/referral/hooks/*.ts`

- [ ] **Step 1: 6 个 hook 文件（每个 8~15 行）**

`use-commission-stats.ts`：
```ts
import { useQuery } from '@tanstack/react-query'
import { getCommissionStats } from '../api'
export function useCommissionStats() {
  return useQuery({ queryKey:['commission','stats'], queryFn:getCommissionStats, staleTime:30_000 })
}
```

`use-commission-records.ts`：
```ts
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { getCommissionRecords } from '../api'
export function useCommissionRecords(p:{status?:string;page:number;size?:number}) {
  return useQuery({ queryKey:['commission','records',p], queryFn:()=>getCommissionRecords(p), placeholderData:keepPreviousData })
}
```

`use-commission-redemptions.ts`：
```ts
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { getCommissionRedemptions } from '../api'
export function useCommissionRedemptions(page:number, size=20) {
  return useQuery({ queryKey:['commission','redemptions',page,size], queryFn:()=>getCommissionRedemptions({page,size}), placeholderData:keepPreviousData })
}
```

`use-commission-downlines.ts`：
```ts
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { getCommissionDownlines } from '../api'
export function useCommissionDownlines(level:1|2, page:number, size=20) {
  return useQuery({ queryKey:['commission','downlines',level,page,size], queryFn:()=>getCommissionDownlines({level,page,size}), placeholderData:keepPreviousData })
}
```

`use-quota-preview.ts`：
```ts
import { useQuery } from '@tanstack/react-query'
import { previewQuotaCredit } from '../api'
export function useQuotaPreview(cents:number) {
  return useQuery({ queryKey:['commission','preview',cents], queryFn:()=>previewQuotaCredit(cents), enabled: cents>0, staleTime:60_000 })
}
```

`use-redeem-commission.ts`：
```ts
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'
import { redeemCommission } from '../api'
export function useRedeemCommission() {
  const qc = useQueryClient()
  const { t } = useTranslation()
  return useMutation({
    mutationFn: (cents:number) => redeemCommission(cents),
    onSuccess: (d) => {
      toast.success(t('Redeem success')+` +${d.quota_credited}`)
      qc.invalidateQueries({ queryKey:['commission'] })
      qc.invalidateQueries({ queryKey:['self'] })
    },
    onError: (e:Error) => toast.error(e.message),
  })
}
```

- [ ] **Step 2: 构建验证 + Commit**

Run: `cd web/default && bun run typecheck`
Expected: 无新 error。

```bash
git add web/default/src/features/referral/hooks
git commit -m "feat(referral): 6 react-query hooks"
```

---

## Task 19: 前端 — 4 个卡片组件

**Files:** `web/default/src/features/referral/components/`

- [ ] **Step 1: `invite-code-card.tsx`**

```tsx
import { useTranslation } from 'react-i18next'
import { Copy } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

export function InviteCodeCard({ affCode }: { affCode: string }) {
  const { t } = useTranslation()
  return (
    <Card>
      <CardHeader><CardTitle>{t('My Invite Code')}</CardTitle></CardHeader>
      <CardContent className='flex items-center gap-3'>
        <span className='font-mono text-2xl tracking-widest'>{affCode || '—'}</span>
        <Button variant='outline' size='sm' onClick={()=>{navigator.clipboard.writeText(affCode);toast.success(t('Copied'))}}>
          <Copy className='mr-1' />{t('Copy')}
        </Button>
      </CardContent>
    </Card>
  )
}
```

- [ ] **Step 2: `invite-link-card.tsx`（含二维码）**

```tsx
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Copy, QrCode } from 'lucide-react'
import { QRCodeSVG } from 'qrcode.react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'

export function InviteLinkCard({ affCode }: { affCode: string }) {
  const { t } = useTranslation()
  const [qr, setQr] = useState(false)
  const link = useMemo(()=> affCode ? `${window.location.origin}/register?aff=${affCode}` : '', [affCode])
  return (
    <Card>
      <CardHeader><CardTitle>{t('My Referral Link')}</CardTitle></CardHeader>
      <CardContent className='space-y-3'>
        <div className='truncate rounded-md bg-muted p-2 font-mono text-xs'>{link || '—'}</div>
        <div className='flex gap-2'>
          <Button variant='outline' size='sm' disabled={!link} onClick={()=>{navigator.clipboard.writeText(link);toast.success(t('Copied'))}}><Copy className='mr-1' />{t('Copy')}</Button>
          <Button variant='outline' size='sm' disabled={!link} onClick={()=>setQr(true)}><QrCode className='mr-1' />{t('QR Code')}</Button>
        </div>
      </CardContent>
      <Dialog open={qr} onOpenChange={setQr}>
        <DialogContent className='w-fit'>
          <DialogHeader><DialogTitle>{t('Scan to register')}</DialogTitle></DialogHeader>
          {link && <QRCodeSVG value={link} size={220} includeMargin />}
        </DialogContent>
      </Dialog>
    </Card>
  )
}
```

- [ ] **Step 3: `commission-stats-card.tsx`**

```tsx
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import type { CommissionStats } from '../types'
import { centsToYuan } from '../lib/format-commission'
import { RedeemDialog } from './redeem-dialog'

export function CommissionStatsCard({ stats }: { stats: CommissionStats }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  return (
    <Card>
      <CardHeader><CardTitle>{t('Commission Account')}</CardTitle></CardHeader>
      <CardContent>
        <div className='grid grid-cols-2 gap-4 md:grid-cols-4'>
          <Stat label={t('Redeemable')} value={`¥${centsToYuan(stats.commission_balance_cents)}`} />
          <Stat label={t('Pending')} value={`¥${centsToYuan(stats.commission_pending_cents)}`} />
          <Stat label={t('Lifetime')} value={`¥${centsToYuan(stats.commission_lifetime_cents)}`} />
          <Stat label={t('Total Redeemed')} value={`¥${centsToYuan(stats.commission_redeemed_cents)}`} />
        </div>
        <div className='mt-4'>
          <Button disabled={stats.commission_balance_cents<=0} onClick={()=>setOpen(true)}>{t('Redeem to Wallet')}</Button>
        </div>
      </CardContent>
      <RedeemDialog open={open} maxCents={stats.commission_balance_cents} onOpenChange={setOpen} />
    </Card>
  )
}
function Stat({label,value}:{label:string;value:string}){return(<div className='rounded-md border p-3'><div className='text-muted-foreground text-xs'>{label}</div><div className='mt-1 text-xl font-semibold'>{value}</div></div>)}
```

- [ ] **Step 4: `redeem-dialog.tsx`**

```tsx
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { centsToYuan, yuanToCents } from '../lib/format-commission'
import { useQuotaPreview } from '../hooks/use-quota-preview'
import { useRedeemCommission } from '../hooks/use-redeem-commission'

type Props = { open: boolean; maxCents: number; onOpenChange: (v: boolean) => void }
export function RedeemDialog({ open, maxCents, onOpenChange }: Props) {
  const { t } = useTranslation()
  const [yuan, setYuan] = useState(centsToYuan(maxCents))
  const cents = Math.min(yuanToCents(Number(yuan)||0), maxCents)
  const preview = useQuotaPreview(cents)
  const redeem = useRedeemCommission()
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader><DialogTitle>{t('Redeem to Wallet')}</DialogTitle></DialogHeader>
        <div className='space-y-3'>
          <div className='text-sm text-muted-foreground'>{t('Redeemable')}: ¥{centsToYuan(maxCents)}</div>
          <Input type='number' step='0.01' min='0.01' max={maxCents/100} value={yuan} onChange={e=>setYuan(e.target.value)} />
          {preview.data && (
            <div className='rounded-md bg-muted p-3 text-sm'>
              <div>{t('Current rate')}: 1 USD ≈ {preview.data.usd_exchange_rate.toFixed(4)} CNY</div>
              <div className='mt-1'>¥{centsToYuan(cents)} → <span className='font-semibold'>{preview.data.quota_credited.toLocaleString()}</span> {t('quota')}</div>
            </div>
          )}
          {preview.isError && <div className='text-destructive text-sm'>{(preview.error as Error).message}</div>}
          <div className='text-xs text-muted-foreground'>{t('Exchange uses the current system rate at redeem time.')}</div>
        </div>
        <DialogFooter>
          <Button variant='outline' onClick={()=>onOpenChange(false)}>{t('Cancel')}</Button>
          <Button disabled={!cents||cents>maxCents||redeem.isPending} onClick={async()=>{await redeem.mutateAsync(cents);onOpenChange(false)}}>{t('Confirm')}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
```

- [ ] **Step 5: Commit**

```bash
git add web/default/src/features/referral/components/{invite-code-card,invite-link-card,commission-stats-card,redeem-dialog}.tsx
git commit -m "feat(referral): 4 cards + redeem dialog"
```

---

## Task 20: 前端 — 3 个表格 + 页面组装 + 路由

**Files:**
- Create: `web/default/src/features/referral/components/records-table.tsx`
- Create: `web/default/src/features/referral/components/redemptions-table.tsx`
- Create: `web/default/src/features/referral/components/downlines-table.tsx`
- Create: `web/default/src/features/referral/index.tsx`
- Create: `web/default/src/routes/_authenticated/console/referral/index.tsx`

- [ ] **Step 1: `records-table.tsx`**

采用与 [`web/default/src/features/wallet/`](web/default/src/features/wallet/) 中现有列表组件相似的表格模式。列：Time / Source User / Level / Base(¥) / Rate(%) / Commission(¥) / Status (badge with 3 colors: pending=yellow / settled=green / voided=gray) / Frozen Until。分页由 `useCommissionRecords({page,size})` 驱动。

参考骨架：

```tsx
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useCommissionRecords } from '../hooks/use-commission-records'
import { centsToYuan } from '../lib/format-commission'

export function RecordsTable() {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const [status, setStatus] = useState<string>('')
  const { data, isLoading } = useCommissionRecords({ page, size: 20, status })
  // Render filter select + table + pagination.
  // Status badge classes: pending → bg-yellow-100 text-yellow-800; settled → bg-green-100 text-green-800; voided → bg-gray-100 text-gray-600.
  // ...
}
```

- [ ] **Step 2: `redemptions-table.tsx`**

列：Time / Commission(¥) / Rate / QuotaPerUnit / Credited Quota。

- [ ] **Step 3: `downlines-table.tsx`**

Level 切换 (1 / 2) + 分页表。列：Registered At / Username (masked) / Email (masked) / Phone (masked)。

- [ ] **Step 4: `index.tsx` 页面组装**

```tsx
import { useTranslation } from 'react-i18next'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { InviteCodeCard } from './components/invite-code-card'
import { InviteLinkCard } from './components/invite-link-card'
import { CommissionStatsCard } from './components/commission-stats-card'
import { RecordsTable } from './components/records-table'
import { RedemptionsTable } from './components/redemptions-table'
import { DownlinesTable } from './components/downlines-table'
import { useCommissionStats } from './hooks/use-commission-stats'

export function ReferralPage() {
  const { t } = useTranslation()
  const { data: stats, isLoading } = useCommissionStats()
  if (isLoading || !stats) return <div>{t('Loading…')}</div>
  return (
    <div className='space-y-6 p-6'>
      <h1 className='text-2xl font-bold'>{t('My Referral')}</h1>
      <div className='grid gap-4 md:grid-cols-2'>
        <InviteCodeCard affCode={stats.aff_code} />
        <InviteLinkCard affCode={stats.aff_code} />
      </div>
      <CommissionStatsCard stats={stats} />
      <Tabs defaultValue='records'>
        <TabsList>
          <TabsTrigger value='records'>{t('Commission Records')}</TabsTrigger>
          <TabsTrigger value='redemptions'>{t('Redemptions')}</TabsTrigger>
          <TabsTrigger value='downlines'>{t('My Downlines')}</TabsTrigger>
        </TabsList>
        <TabsContent value='records'><RecordsTable /></TabsContent>
        <TabsContent value='redemptions'><RedemptionsTable /></TabsContent>
        <TabsContent value='downlines'><DownlinesTable /></TabsContent>
      </Tabs>
    </div>
  )
}
```

- [ ] **Step 5: TanStack Router 路由**

Create `web/default/src/routes/_authenticated/console/referral/index.tsx`:

```tsx
import { createFileRoute } from '@tanstack/react-router'
import { ReferralPage } from '@/features/referral'
export const Route = createFileRoute('/_authenticated/console/referral/')({
  component: ReferralPage,
})
```

- [ ] **Step 6: 侧边栏接入**

Search `console/wallet` in `web/default/src/` to find the user-side sidebar config file, add:

```tsx
{ to: '/console/referral', label: t('My Referral'), icon: Share2 }
```

- [ ] **Step 7: Commit**

```bash
git add web/default/src/features/referral \
        web/default/src/routes/_authenticated/console/referral
# also sidebar file
git commit -m "feat(referral): page + tables + router + sidebar entry"
```

---

## Task 21: 前端 — 管理员端 3 个 Section

**Files:**
- Create: `web/default/src/features/system-settings/commission/`
  - `index.tsx / section-registry.tsx / api.ts / types.ts`
  - `sections/rules-section.tsx`
  - `sections/records-section.tsx`
  - `sections/overview-section.tsx`
- Create: `web/default/src/routes/_authenticated/system-settings/commission/$section.tsx`
- Create: `web/default/src/routes/_authenticated/system-settings/commission/index.tsx`

- [ ] **Step 1: `api.ts` 封装 6 个管理员端点**

参照 [`web/default/src/features/system-settings/general/channel-affinity/api.ts`](web/default/src/features/system-settings/general/channel-affinity/api.ts) 模式：

```ts
import { api } from '@/lib/api'
import type { CommissionRecord, CommissionRule } from './types'

export async function getRules(): Promise<CommissionRule[]> {
  const res = await api.get('/api/commission-admin/rules')
  return res.data.data ?? []
}
export async function updateRule(id: number, patch: Partial<CommissionRule>) {
  return api.put(`/api/commission-admin/rules/${id}`, patch)
}
export async function listRecords(params: Record<string, any>) {
  const res = await api.get('/api/commission-admin/records', { params })
  return res.data.data
}
export async function voidRecord(id: number, reason: string) {
  return api.post(`/api/commission-admin/records/${id}/void`, { reason })
}
export async function settleNow() {
  return api.post('/api/commission-admin/settle-now')
}
export async function getOverview() {
  const res = await api.get('/api/commission-admin/stats')
  return res.data.data
}
```

- [ ] **Step 2: `section-registry.tsx`**

参照 [`web/default/src/features/system-settings/billing/section-registry.tsx`](web/default/src/features/system-settings/billing/section-registry.tsx) 的 `createSectionRegistry` 模式，声明 3 个 section (`rules / records / overview`)，导出 nav items 与 content builder。

- [ ] **Step 3: `rules-section.tsx` — 规则表**

展示 L1 / L2 两行，每行 4 个可编辑字段 (`rate_percent / min_topup_cents / frozen_days / enabled`)，每行独立 Save 按钮调用 `updateRule(id, patch)`。底部一个 "Settle Now" 按钮调用 `settleNow()`。

- [ ] **Step 4: `records-section.tsx` — 全平台流水**

筛选表单 (status / beneficiary / source / date range) + 分页表格 + 每行 Void 按钮 → VoidDialog（输入必填 `reason` textarea，调用 `voidRecord(id, reason)`）。

- [ ] **Step 5: `overview-section.tsx` — 平台总览**

6 个统计卡（总佣金 / 已结算 / 冻结中 / 已转余额 / 参与用户 / 完成首充），可选加一个简单 30 天趋势折线图（复用项目 chart 组件或 svg 手绘）。

- [ ] **Step 6: 路由文件（照抄 billing 结构）**

Copy `web/default/src/routes/_authenticated/system-settings/billing/{$section.tsx,index.tsx}` and adapt to commission。

- [ ] **Step 7: 侧边栏接入**

Find `system-settings` sidebar config (search `billing` next to `system-settings`), add commission entry.

- [ ] **Step 8: Commit**

```bash
git add web/default/src/features/system-settings/commission \
        web/default/src/routes/_authenticated/system-settings/commission
git commit -m "feat(commission-admin): rules/records/overview sections + routing"
```

---

## Task 22: i18n 键补齐

**Files:**
- Modify: `web/default/src/i18n/locales/zh.json`
- Modify: `web/default/src/i18n/locales/en.json`

- [ ] **Step 1: 提取待翻译键**

Run: `cd web/default && bun run i18n:sync`
This scans all `t('...')` calls and writes missing keys into `_reports/*.untranslated.json`。

- [ ] **Step 2: 手动填 zh.json 翻译**

至少下列约 60 个键需要中文翻译。将它们按 `zh.json` 现有字母顺序插入，并将 en.json 中的值填为英文源文：

```
"Apply to exchange rate" (existing)  — no-op
"Base Amount" → "基数"
"Cancel" (existing)                  — no-op
"Commission" → "佣金"
"Commission Account" → "佣金账户"
"Commission Records" → "佣金流水"
"Confirm" → "确认"
"Conversion Rate" → "转化率"
"Copied" → "已复制"
"Copy" → "复制"
"Current rate" → "当前汇率"
"Enabled" → "启用"
"Exchange uses the current system rate at redeem time." → "汇率以转入时刻的系统实时汇率为准。"
"First-topup users" → "完成首充用户"
"Frozen Days" → "冻结天数"
"Frozen Until" → "冻结至"
"Level" → "层级"
"Lifetime" → "累计佣金"
"Loading…" (existing)                — no-op
"Min Topup" → "最低门槛"
"My Downlines" → "我的下线"
"My Invite Code" → "我的邀请码"
"My Referral" → "我的分销"
"My Referral Link" → "我的推广链接"
"Overview" → "总览"
"Participants" → "参与用户"
"Pending" → "冻结中"
"pending" → "冻结中"
"QR Code" → "二维码"
"quota" → "额度"
"Rate" → "比例"
"Rate (%)" → "比例（%）"
"Reason is required" → "作废原因必填"
"Records" → "流水"
"Redeem success" → "转入成功"
"Redeem to Wallet" → "转入账户余额"
"Redeemable" → "可转余额"
"Redemptions" → "转余额记录"
"Redeemed" → "已转余额"
"Rules" → "规则"
"Save" → "保存"
"Scan to register" → "扫码注册"
"Settle Now" → "立即触发结算"
"Settled" → "已结算"
"settled" → "已结算"
"Settled At" → "结算时间"
"Source User" → "来源用户"
"Status" → "状态"
"Total Commissions" → "总佣金"
"Total Redeemed" → "累计转余额"
"Void" → "作废"
"Void Reason" → "作废原因"
"voided" → "已作废"
```

- [ ] **Step 3: 其他语言文件（fr/ja/ru/vi）**

en.json 用英文键作值；fr/ja/ru/vi 保留英文即可，`i18n:sync` 会生成 `_reports/*.untranslated.json` 交由后续翻译流程处理。

- [ ] **Step 4: 验证**

Run: `cd web/default && bun run i18n:sync && bun run build`
Expected: 构建成功。

- [ ] **Step 5: Commit**

```bash
git add web/default/src/i18n/locales/{zh,en}.json web/default/src/i18n/locales/_reports
git commit -m "i18n(commission): add zh/en translations for referral & commission-admin"
```

---

## Task 23: 端到端联调冒烟

- [ ] **Step 1: 三库回归**

- SQLite: `./start.sh`（默认，本地开发）
- MySQL: 用现有 `db.env` 或 `DATABASE_TYPE=mysql` 环境启动
- PostgreSQL: 用 `DATABASE_TYPE=postgres` 启动

每种数据库启动后确认：
- `commission_rules` 有 2 行种子（L1=20%, L2=5%）
- `system_tasks` 表最新一行 `type='commission_settle'` 出现（首次运行 10 分钟内）

- [ ] **Step 2: 手动端到端**

1. 注册 3 个用户 A (无 inviter)、B (invited by A)、C (invited by B)。
   - 验证 `user_referral_paths`：B(l1=A,l2=0) 与 C(l1=B,l2=A)。

2. 用 C 支付宝（或微信/易支付）充 ¥100。
   - `topups` 一条 success；
   - `commission_records` 2 条 pending：level=1 beneficiary=B amount_cents=2000；level=2 beneficiary=A amount_cents=500；
   - `user_commission_stats` pending 相应 bump。

3. 管理端点击 "Settle Now"（或等 10 分钟）：
   - 2 条 records → status=settled，settled_at 有值；
   - B.balance_cents=2000, A.balance_cents=500；
   - `logs` 表 2 条 LogTypeSystem。

4. B 在 `/console/referral` 点转入余额（¥20 全额）：
   - `users.quota` += floor(2000/100/rate*QPU)；
   - `commission_redemptions` 一行；
   - `commission_balance_cents=0, redeemed_cents=2000`。

5. 管理员在 `/system-settings/commission/records` 找到 A 的 5 元记录点 Void 输入 "test"：
   - status=voided；A.balance -= 500。

6. 用户 C 再次 ¥50 微信充值 → 不产生新 records（非首充）。

7. Stripe 支付 ¥100 → 不产生 records（非国内通道）。

- [ ] **Step 3: 使用 `verify` skill 补充自动化验证（可选）**

Run: `/verify` — Claude 会自动启动 app、跑关键路径。

- [ ] **Step 4: 若发现问题回到相应 Task 修复；无问题合并 PR**

---

## Task 24: 文档收尾

- [ ] **Step 1: 更新本 plan 底部添加 "执行摘要"**

在本文件末尾追加：完成时间、遗留 TODO、灰度状态。

- [ ] **Step 2: 更新 `/Users/perjac/Downloads/PJF/PeJac/35sz-api-二次开发功能清单.md`**

在「核心差异化能力总览」新增一行：

```
| 13 | **二级分销与首充返佣（MVP）** | 全新：L1=20%/L2=5% 首充分佣，T+7 冻结，只转账户余额不出金 | B/C 端渠道招募与激励 |
```

新增章节「分销佣金体系」简述表结构、结算流程与 10 个 API。

- [ ] **Step 3: 更新 MEMORY.md（若需要）**

若项目 `MEMORY.md` 有维护习惯，添加 "分销佣金 MVP 落地时间" 一条。

- [ ] **Step 4: Commit**

```bash
git add docs/superpowers/plans/2026-07-06-referral-commission-mvp.md \
        /Users/perjac/Downloads/PJF/PeJac/35sz-api-二次开发功能清单.md
git commit -m "docs(commission): finalize plan retro + update feature catalog"
```

---

## 完成标准

- 24 个 Task 的所有 checkbox 全部勾选
- `go build ./...` 与 `go test ./model/... ./service/... ./controller/...` 全部通过
- `cd web/default && bun run typecheck && bun run build` 通过
- SQLite / MySQL / PostgreSQL 三库手动冒烟通过
- 生产灰度开关：初始 `commission_rules.enabled=true`；如需暂停 SQL `UPDATE commission_rules SET enabled=false` 即可，不影响支付主链路

---

## 执行摘要（2026-07-08 完成）

**24 个任务全部完成。** 后端 16 个 + 前端 6 个 + i18n 1 个 + 文档 1 个。

### 关键路径回顾

- **数据模型（T1–T5）** — 5 张表 + AutoMigrate 注册 + 通用测试 helper `SetupTestDBForCommissionTests`。
- **Service 层（T6–T11）** — path/record/settle/redeem/void/seed 六个模块，每个都有 5–10 个 TDD 测试。首充判定用 `COUNT(topups)` 事实数据，无需额外字段。
- **调度器（T12）** — 通过 `service/system_task_commission.go` 桥接文件避免 import cycle。
- **注册钩子 & 支付埋点（T13–T14）** — 4 处埋入：Register、Alipay、Wechat、Epay 回调。全部 `go` 关键字异步 + `recover()` 兜底。
- **HTTP 层（T15–T16）** — 10 个 handler + 6 selfRoute + 6 commissionAdmin。PII 脱敏用 `maskUsername` / `maskEmail` helper。
- **前端（T17–T21）** — 2 feature 模块，用户端 `/referral` + 管理端 `/system-settings/commission`。二维码用 `qrcode.react`。
- **i18n（T22）** — 44 组 zh + en 键。
- **回归（T23）** — `go test ./...` 全绿；`bun run build` 通过。三库 E2E（MySQL/Postgres）留给运维环境验证。

### 已知遗留 / 待办（二期）

- 订阅分佣（`scope='subscription'`）
- 消费分佣（按 API 用量返佣）
- 三级及以上分销
- 微信企业付款 API 提现（当前只转余额）
- 反作弊规则引擎（同 IP、设备指纹、首单窗口）
- 分销员等级 / 阶梯佣金
- 财务对账导出、KYC 实名
- 三库真实生产环境 E2E 冒烟（本次仅完成 SQLite 单元测试）

### 灰度建议

上线首日建议：
1. 保留 `commission_rules.enabled=true`（默认状态）；
2. 观察 24h 内 `commission_records` 表增长与 `logs.LogTypeSystem` 结算日志；
3. 若发现异常可 SQL `UPDATE commission_rules SET enabled=false` 立即停止分佣，不影响支付主链路；
4. 首个 T+7 结算窗口后确认 `pending → settled` 流程正常。

### 已通过验证

- `go build ./...` — 无编译错误
- `go test ./...` — 40+ 单元测试全部通过（含 model、service/commission、controller 层）
- `bun run build` — 前端 dist 产出成功
- 数据库兼容 — SQLite AutoMigrate 通过（MySQL/Postgres 生产环境冒烟待运维验证）
