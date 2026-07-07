package model

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupStatsTestDB(t *testing.T) {
	t.Helper()
	setupCommissionTestDB(t)
	require.NoError(t, DB.AutoMigrate(&UserReferralPath{}, &UserCommissionStats{}))
}

func TestUserCommissionStats_GetOrCreateInsertsRow(t *testing.T) {
	setupStatsTestDB(t)

	stats, err := GetOrCreateCommissionStats(DB, 42)
	require.NoError(t, err)
	require.Equal(t, 42, stats.UserId)
	require.EqualValues(t, 0, stats.CommissionBalanceCents)

	// Second call returns the existing row.
	stats2, err := GetOrCreateCommissionStats(DB, 42)
	require.NoError(t, err)
	require.EqualValues(t, stats.CreatedAt, stats2.CreatedAt)
}

func TestUserCommissionStats_AddPendingBumpsCounter(t *testing.T) {
	setupStatsTestDB(t)

	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return AddCommissionPending(tx, 42, 2500)
	}))

	stats, err := GetCommissionStats(42)
	require.NoError(t, err)
	require.EqualValues(t, 2500, stats.CommissionPendingCents)
	require.EqualValues(t, 0, stats.CommissionBalanceCents)
}

func TestUserCommissionStats_SettleMovesPendingToBalance(t *testing.T) {
	setupStatsTestDB(t)

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
	setupStatsTestDB(t)

	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		if err := AddCommissionPending(tx, 42, 3000); err != nil {
			return err
		}
		if err := PendingToBalance(tx, 42, 3000); err != nil {
			return err
		}
		return RedeemFromBalance(tx, 42, 1000)
	}))

	stats, _ := GetCommissionStats(42)
	require.EqualValues(t, 2000, stats.CommissionBalanceCents)
	require.EqualValues(t, 1000, stats.CommissionRedeemedCents)
	require.EqualValues(t, 3000, stats.CommissionLifetimeCents, "lifetime never decreases")
}

func TestUserCommissionStats_RedeemRejectsInsufficient(t *testing.T) {
	setupStatsTestDB(t)
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return AddCommissionPending(tx, 42, 100)
	}))

	err := DB.Transaction(func(tx *gorm.DB) error {
		return RedeemFromBalance(tx, 42, 500)
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient")
}

func TestUserCommissionStats_DeductBalanceClawback(t *testing.T) {
	setupStatsTestDB(t)

	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		if err := AddCommissionPending(tx, 42, 2000); err != nil {
			return err
		}
		if err := PendingToBalance(tx, 42, 2000); err != nil {
			return err
		}
		return DeductCommissionBalance(tx, 42, 500)
	}))

	stats, _ := GetCommissionStats(42)
	require.EqualValues(t, 1500, stats.CommissionBalanceCents)
	require.EqualValues(t, 0, stats.CommissionRedeemedCents, "clawback must not inflate redeemed counter")
	require.EqualValues(t, 2000, stats.CommissionLifetimeCents)
}

func TestUserCommissionStats_DeductPendingClawback(t *testing.T) {
	setupStatsTestDB(t)

	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		if err := AddCommissionPending(tx, 42, 2000); err != nil {
			return err
		}
		return DeductCommissionPending(tx, 42, 500)
	}))

	stats, _ := GetCommissionStats(42)
	require.EqualValues(t, 1500, stats.CommissionPendingCents)
}

func TestUserCommissionStats_DeductPendingInsufficient(t *testing.T) {
	setupStatsTestDB(t)
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return AddCommissionPending(tx, 42, 100)
	}))

	err := DB.Transaction(func(tx *gorm.DB) error {
		return DeductCommissionPending(tx, 42, 500)
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient")
}
