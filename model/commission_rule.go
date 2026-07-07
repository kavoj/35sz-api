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
	Id            int64   `json:"id" gorm:"primaryKey;autoIncrement"`
	Scope         string  `json:"scope" gorm:"type:varchar(32);not null;uniqueIndex:uk_scope_level,priority:1"`
	Level         int     `json:"level" gorm:"not null;uniqueIndex:uk_scope_level,priority:2"`
	RatePercent   float64 `json:"rate_percent" gorm:"type:decimal(5,2);not null"`
	MinTopupCents int64   `json:"min_topup_cents" gorm:"not null;default:0"`
	FrozenDays    int     `json:"frozen_days" gorm:"not null;default:7"`
	// Enabled default is intentionally set at the seed/CRUD layer rather than a
	// gorm default tag: MySQL/PostgreSQL normalize boolean defaults differently
	// and would cause AutoMigrate to churn `ALTER TABLE` on every restart.
	Enabled   bool  `json:"enabled" gorm:"not null"`
	CreatedAt int64 `json:"created_at" gorm:"not null"`
	UpdatedAt int64 `json:"updated_at" gorm:"not null"`
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
