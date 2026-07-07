package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func setupRedemptionTestDB(t *testing.T) {
	t.Helper()
	setupCommissionTestDB(t)
	require.NoError(t, DB.AutoMigrate(&CommissionRedemption{}))
}

func TestCommissionRedemption_Insert(t *testing.T) {
	setupRedemptionTestDB(t)

	r := CommissionRedemption{
		UserId:          42,
		CommissionCents: 2500,
		USDExchangeRate: 7.2,
		QuotaPerUnit:    500000,
		QuotaCredited:   1736111,
	}
	before := time.Now().Unix()
	require.NoError(t, InsertCommissionRedemption(DB, &r))
	require.NotZero(t, r.Id)
	require.GreaterOrEqual(t, r.CreatedAt, before)

	list, err := ListRedemptionsByUser(42, 10, 0)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.EqualValues(t, 2500, list[0].CommissionCents)
	require.InDelta(t, 7.2, list[0].USDExchangeRate, 0.001)
	require.EqualValues(t, 500000, list[0].QuotaPerUnit)
	require.EqualValues(t, 1736111, list[0].QuotaCredited)
}

func TestCommissionRedemption_ListReturnsNewestFirst(t *testing.T) {
	setupRedemptionTestDB(t)

	require.NoError(t, InsertCommissionRedemption(DB, &CommissionRedemption{UserId: 1, CommissionCents: 100, USDExchangeRate: 7.2, QuotaPerUnit: 500000, QuotaCredited: 1}))
	require.NoError(t, InsertCommissionRedemption(DB, &CommissionRedemption{UserId: 1, CommissionCents: 200, USDExchangeRate: 6.5, QuotaPerUnit: 500000, QuotaCredited: 2}))

	list, err := ListRedemptionsByUser(1, 10, 0)
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.EqualValues(t, 200, list[0].CommissionCents, "newest first")
	require.EqualValues(t, 100, list[1].CommissionCents)
}

func TestCommissionRedemption_ListPaginates(t *testing.T) {
	setupRedemptionTestDB(t)

	for i := 0; i < 5; i++ {
		require.NoError(t, InsertCommissionRedemption(DB, &CommissionRedemption{
			UserId: 7, CommissionCents: int64(i + 1),
			USDExchangeRate: 7, QuotaPerUnit: 500000, QuotaCredited: int64(i + 1),
		}))
	}
	page1, err := ListRedemptionsByUser(7, 2, 0)
	require.NoError(t, err)
	require.Len(t, page1, 2)

	page2, err := ListRedemptionsByUser(7, 2, 2)
	require.NoError(t, err)
	require.Len(t, page2, 2)
	require.NotEqual(t, page1[0].Id, page2[0].Id)
}

func TestCommissionRedemption_ListFiltersByUser(t *testing.T) {
	setupRedemptionTestDB(t)

	require.NoError(t, InsertCommissionRedemption(DB, &CommissionRedemption{UserId: 1, CommissionCents: 100, USDExchangeRate: 7, QuotaPerUnit: 500000, QuotaCredited: 1}))
	require.NoError(t, InsertCommissionRedemption(DB, &CommissionRedemption{UserId: 2, CommissionCents: 200, USDExchangeRate: 7, QuotaPerUnit: 500000, QuotaCredited: 2}))

	list, err := ListRedemptionsByUser(1, 10, 0)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, 1, list[0].UserId)
}
