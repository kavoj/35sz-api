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

// CommissionRecord is one line of a commission ledger. The composite unique
// index (source_topup_id, beneficiary_id, level) makes retries of the payment
// webhook harmless: the second insert is dropped instead of paying twice.
type CommissionRecord struct {
	Id                    int64   `json:"id" gorm:"primaryKey;autoIncrement"`
	BeneficiaryId         int     `json:"beneficiary_id" gorm:"not null;index:idx_beneficiary_status,priority:1;uniqueIndex:uk_topup_bene_level,priority:2"`
	SourceUserId          int     `json:"source_user_id" gorm:"not null;index"`
	SourceTopupId         int     `json:"source_topup_id" gorm:"not null;uniqueIndex:uk_topup_bene_level,priority:1"`
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

// InsertCommissionRecord writes rec, returning (true, nil) on a new row or
// (false, nil) when the unique index dropped a duplicate. Used by the payment
// callback so replays don't double-pay.
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

// FetchPendingDueCommissionRecords returns up to `limit` records whose freeze
// window has elapsed. The batch is capped so the settle task never runs one
// huge transaction.
func FetchPendingDueCommissionRecords(limit int) ([]CommissionRecord, error) {
	var out []CommissionRecord
	err := DB.Where("status = ? AND frozen_until <= ?", CommissionStatusPending, time.Now().Unix()).
		Order("id ASC").
		Limit(limit).
		Find(&out).Error
	return out, err
}

// IsFirstDomesticSuccessTopup returns true iff topupId is the earliest
// successful topup for this user across the three domestic providers
// (alipay/wxpay/epay). All other providers (stripe/creem/waffo/admin/balance)
// are ignored, so a Stripe topup does not disqualify a later domestic one from
// being counted as "first".
func IsFirstDomesticSuccessTopup(userId int, topupId int) bool {
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

// GetCommissionRecordByID fetches one record for the admin void endpoint.
func GetCommissionRecordByID(id int64) (*CommissionRecord, error) {
	var out CommissionRecord
	if err := DB.Where("id = ?", id).First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}
