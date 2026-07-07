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
	UserId                  int   `json:"user_id" gorm:"primaryKey"`
	CommissionBalanceCents  int64 `json:"commission_balance_cents" gorm:"not null;default:0"`
	CommissionPendingCents  int64 `json:"commission_pending_cents" gorm:"not null;default:0"`
	CommissionLifetimeCents int64 `json:"commission_lifetime_cents" gorm:"not null;default:0"`
	CommissionRedeemedCents int64 `json:"commission_redeemed_cents" gorm:"not null;default:0"`
	CreatedAt               int64 `json:"created_at" gorm:"not null"`
	UpdatedAt               int64 `json:"updated_at" gorm:"not null"`
}

func (UserCommissionStats) TableName() string { return "user_commission_stats" }

// GetOrCreateCommissionStats returns the row for userId, lazily inserting it
// on first touch. Call inside a transaction (tx) whenever the caller is going
// to mutate counters afterwards; ad-hoc reads may pass DB.
func GetOrCreateCommissionStats(tx *gorm.DB, userId int) (*UserCommissionStats, error) {
	now := time.Now().Unix()
	stats := UserCommissionStats{
		UserId:    userId,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&stats).Error; err != nil {
		return nil, err
	}
	var out UserCommissionStats
	if err := tx.Where("user_id = ?", userId).First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

// GetCommissionStats returns nil when the row is absent, for dashboard reads
// that don't want to eagerly insert.
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
// Fails if the counter would go negative.
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

// RedeemFromBalance moves cents out of balance into redeemed. Used by the
// user-initiated wallet conversion. Fails if balance is short.
func RedeemFromBalance(tx *gorm.DB, userId int, cents int64) error {
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

// DeductCommissionBalance is used by AdminVoid on a settled record: pull the
// amount off the user's balance without bumping the "redeemed" counter — a
// clawback is not a user-initiated conversion. Fails if balance is short.
// (`commission_lifetime_cents` intentionally never rolls back so historical
// totals remain auditable.)
func DeductCommissionBalance(tx *gorm.DB, userId int, cents int64) error {
	res := tx.Model(&UserCommissionStats{}).
		Where("user_id = ? AND commission_balance_cents >= ?", userId, cents).
		Updates(map[string]any{
			"commission_balance_cents": gorm.Expr("commission_balance_cents - ?", cents),
			"updated_at":               time.Now().Unix(),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("insufficient commission balance")
	}
	return nil
}
