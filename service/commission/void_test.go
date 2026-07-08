package commission

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/QuantumNous/new-api/model"
)

// insertRec builds a record + its idempotent insert. Different from
// insertPending in settle_test.go: we may want a settled or a pending row
// here.
func insertRec(t *testing.T, id int64, ben int, cents int64, status string, frozen int64) {
	t.Helper()
	rec := &model.CommissionRecord{
		Id: id, BeneficiaryId: ben, SourceUserId: 999, SourceTopupId: int(id),
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
	require.NotZero(t, rec.VoidedAt)

	s, _ := model.GetCommissionStats(50)
	require.EqualValues(t, 0, s.CommissionPendingCents)
}

func TestVoid_SettledDeductsBalance(t *testing.T) {
	model.SetupTestDBForCommissionTests(t)
	require.NoError(t, model.DB.Transaction(func(tx *gorm.DB) error {
		if err := model.AddCommissionPending(tx, 50, 1000); err != nil {
			return err
		}
		return model.PendingToBalance(tx, 50, 1000)
	}))
	insertRec(t, 1, 50, 1000, model.CommissionStatusSettled, time.Now().Unix()-3600)

	require.NoError(t, Void(1, "clawback"))

	s, _ := model.GetCommissionStats(50)
	require.EqualValues(t, 0, s.CommissionBalanceCents)
	require.EqualValues(t, 1000, s.CommissionLifetimeCents, "lifetime never decreases")
	require.EqualValues(t, 0, s.CommissionRedeemedCents, "clawback must not inflate redeemed counter")
}

func TestVoid_AlreadyRedeemedRejected(t *testing.T) {
	model.SetupTestDBForCommissionTests(t)
	require.NoError(t, model.DB.Transaction(func(tx *gorm.DB) error {
		if err := model.AddCommissionPending(tx, 50, 1000); err != nil {
			return err
		}
		if err := model.PendingToBalance(tx, 50, 1000); err != nil {
			return err
		}
		return model.RedeemFromBalance(tx, 50, 1000)
	}))
	insertRec(t, 1, 50, 1000, model.CommissionStatusSettled, 0)

	err := Void(1, "too late")
	require.Error(t, err)

	// Record must not have flipped to voided.
	rec, _ := model.GetCommissionRecordByID(1)
	require.Equal(t, model.CommissionStatusSettled, rec.Status)
}

func TestVoid_EmptyReasonRejected(t *testing.T) {
	model.SetupTestDBForCommissionTests(t)
	insertRec(t, 1, 50, 1000, model.CommissionStatusPending, time.Now().Unix()+3600)
	require.Error(t, Void(1, ""))
	require.Error(t, Void(1, "   "))
}

func TestVoid_AlreadyVoidedRejected(t *testing.T) {
	model.SetupTestDBForCommissionTests(t)
	insertRec(t, 1, 50, 1000, model.CommissionStatusVoided, 0)

	err := Void(1, "again")
	require.Error(t, err)
	require.Contains(t, err.Error(), "already voided")
}

func TestVoid_UnknownRecord(t *testing.T) {
	model.SetupTestDBForCommissionTests(t)
	err := Void(999, "test")
	require.Error(t, err)
}
