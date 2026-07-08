package commission

import (
	"fmt"
	"math"
	"time"

	"gorm.io/gorm"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// domesticProviders is the set of PaymentProvider values that produce
// commission. Overseas gateways (stripe/creem/waffo/waffo_pancake) are
// intentionally excluded from this MVP; admin manual top-ups and balance
// transfers are excluded too.
var domesticProviders = map[string]struct{}{
	model.PaymentProviderAlipay: {},
	model.PaymentProviderWechat: {},
	model.PaymentProviderEpay:   {},
}

// OnTopupCompleted is called from a payment webhook after the topup's own
// transaction has already committed. It writes commission records for uplines
// when all business rules are met. Failures are logged but never propagated:
// the user's wallet has already been credited, so a downstream commission bug
// must not roll back the payment.
func OnTopupCompleted(topup *model.TopUp) {
	defer func() {
		if r := recover(); r != nil {
			id := 0
			if topup != nil {
				id = topup.Id
			}
			common.SysError(fmt.Sprintf("commission recording panic: %v topup_id=%d", r, id))
		}
	}()

	if topup == nil || topup.Status != common.TopUpStatusSuccess {
		return
	}
	if _, ok := domesticProviders[topup.PaymentProvider]; !ok {
		return
	}
	if !model.IsFirstDomesticSuccessTopup(topup.UserId, topup.Id) {
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

	// TopUp.Money is a yuan float on the topups table. Round to the nearest
	// cent so downstream math stays in integers.
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
				SourceTopupId:         topup.Id,
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
				// The unique index dropped a duplicate — a prior invocation
				// already recorded this line. Do not double-bump the counter.
				continue
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

// pickBeneficiary maps rule levels (1, 2) to the L1/L2 slots recorded in the
// user's referral path snapshot.
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
