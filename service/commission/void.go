package commission

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/QuantumNous/new-api/model"
)

// Void marks a commission record as voided and rolls back the beneficiary's
// counter. Lifetime totals are intentionally left in place because they
// represent history rather than a claim on funds; only the "available" or
// "pending" counters need to reflect the clawback.
//
// The pending-vs-settled branch matters: for pending records we deduct from
// pending_cents; for settled records we deduct from balance_cents. If the
// user has already converted that settled balance into wallet quota, the
// DeductCommissionBalance guard fails with "insufficient" and Void aborts —
// the plan explicitly excludes clawing back already-spent funds.
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

		return tx.Model(&model.CommissionRecord{}).
			Where("id = ?", recordID).
			Updates(map[string]any{
				"status":        model.CommissionStatusVoided,
				"voided_at":     now,
				"voided_reason": reason,
				"updated_at":    now,
			}).Error
	})
}
