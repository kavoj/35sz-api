package model

import (
	"time"

	"gorm.io/gorm"
)

// CommissionRedemption is an append-only trail of "commission → wallet quota"
// conversions. The exchange rate and QuotaPerUnit at that instant are
// snapshotted so later operator adjustments do not rewrite historical
// accounting.
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

// InsertCommissionRedemption stamps CreatedAt on the caller's struct so the
// caller gets back the value that was persisted.
func InsertCommissionRedemption(tx *gorm.DB, r *CommissionRedemption) error {
	r.CreatedAt = time.Now().Unix()
	return tx.Create(r).Error
}

// ListRedemptionsByUser returns the most recent redemptions for a user,
// paginated. Newest first.
func ListRedemptionsByUser(userId, limit, offset int) ([]CommissionRedemption, error) {
	var out []CommissionRedemption
	err := DB.Where("user_id = ?", userId).
		Order("id DESC").
		Limit(limit).Offset(offset).
		Find(&out).Error
	return out, err
}
