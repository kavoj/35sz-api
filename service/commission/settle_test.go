package commission

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/QuantumNous/new-api/model"
)

// insertPending seeds one record + bumps the stats.pending counter as the
// production code path would have. Callers control the frozen_until so the
// same helper works for both due and not-due scenarios.
func insertPending(t *testing.T, id int64, beneficiary int, cents int64, frozenUntil int64) {
	t.Helper()
	rec := &model.CommissionRecord{
		Id: id, BeneficiaryId: beneficiary, SourceUserId: 999,
		SourceTopupId: int(id), Scope: model.CommissionScopeFirstTopup, Level: 1,
		RatePercent: 20, BaseAmountCents: cents * 5, CommissionAmountCents: cents,
		Status: model.CommissionStatusPending, FrozenUntil: frozenUntil,
	}
	_, err := model.InsertCommissionRecord(model.DB, rec)
	require.NoError(t, err)
	require.NoError(t, model.DB.Transaction(func(tx *gorm.DB) error {
		return model.AddCommissionPending(tx, beneficiary, cents)
	}))
}

func TestSettlePending_MovesDueRecords(t *testing.T) {
	model.SetupTestDBForCommissionTests(t)

	past := time.Now().Unix() - 10
	future := time.Now().Unix() + 3600
	insertPending(t, 1, 50, 2000, past)
	insertPending(t, 2, 50, 500, future)

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
	insertPending(t, 1, 50, 2000, time.Now().Unix()+3600)

	settled, err := SettlePending()
	require.NoError(t, err)
	require.Equal(t, 0, settled)
}

func TestSettlePending_HandlesLargeBatchInWaves(t *testing.T) {
	model.SetupTestDBForCommissionTests(t)

	past := time.Now().Unix() - 10
	for i := 1; i <= 600; i++ {
		insertPending(t, int64(i), 50, 10, past)
	}
	settled, err := SettlePending()
	require.NoError(t, err)
	require.Equal(t, 600, settled)

	s, _ := model.GetCommissionStats(50)
	require.EqualValues(t, 600*10, s.CommissionBalanceCents)
}

func TestSettlePending_SkipsAlreadySettled(t *testing.T) {
	model.SetupTestDBForCommissionTests(t)
	past := time.Now().Unix() - 10
	insertPending(t, 1, 50, 2000, past)

	require.NoError(t, model.DB.Model(&model.CommissionRecord{}).
		Where("id = ?", 1).
		Updates(map[string]any{"status": model.CommissionStatusSettled}).Error)

	settled, err := SettlePending()
	require.NoError(t, err)
	require.Equal(t, 0, settled, "already-settled records are filtered by the WHERE clause")
}
