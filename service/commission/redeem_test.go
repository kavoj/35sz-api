package commission

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

// seedBalance credits a settled balance directly, bypassing the normal
// pending→settle flow so the test can focus on Redeem-specific behavior.
func seedBalance(t *testing.T, userId int, cents int64) {
	t.Helper()
	require.NoError(t, model.DB.Transaction(func(tx *gorm.DB) error {
		if err := model.AddCommissionPending(tx, userId, cents); err != nil {
			return err
		}
		return model.PendingToBalance(tx, userId, cents)
	}))
}

func TestRedeem_HappyPath(t *testing.T) {
	model.SetupTestDBForCommissionTests(t)
	require.NoError(t, model.DB.Create(&model.User{Id: 42, Quota: 0, Username: "u42", AffCode: "a42"}).Error)
	seedBalance(t, 42, 2500)

	oldRate, oldQPU := operation_setting.USDExchangeRate, common.QuotaPerUnit
	defer func() { operation_setting.USDExchangeRate, common.QuotaPerUnit = oldRate, oldQPU }()
	operation_setting.USDExchangeRate = 7.2
	common.QuotaPerUnit = 500000

	q, err := Redeem(42, 2500)
	require.NoError(t, err)
	expected := int64(math.Floor(2500.0 / 100 / 7.2 * 500000))
	require.Equal(t, expected, q)

	u := model.User{}
	require.NoError(t, model.DB.First(&u, 42).Error)
	require.EqualValues(t, expected, u.Quota)

	s, _ := model.GetCommissionStats(42)
	require.EqualValues(t, 0, s.CommissionBalanceCents)
	require.EqualValues(t, 2500, s.CommissionRedeemedCents)

	rs, _ := model.ListRedemptionsByUser(42, 10, 0)
	require.Len(t, rs, 1)
	require.InDelta(t, 7.2, rs[0].USDExchangeRate, 0.001)
	require.EqualValues(t, 500000, rs[0].QuotaPerUnit)
	require.EqualValues(t, expected, rs[0].QuotaCredited)
}

func TestRedeem_InsufficientBalance(t *testing.T) {
	model.SetupTestDBForCommissionTests(t)
	require.NoError(t, model.DB.Create(&model.User{Id: 42, Quota: 0, Username: "u42", AffCode: "a42"}).Error)
	seedBalance(t, 42, 100)
	oldRate, oldQPU := operation_setting.USDExchangeRate, common.QuotaPerUnit
	defer func() { operation_setting.USDExchangeRate, common.QuotaPerUnit = oldRate, oldQPU }()
	operation_setting.USDExchangeRate = 7.2
	common.QuotaPerUnit = 500000

	_, err := Redeem(42, 500)
	require.Error(t, err)
}

func TestRedeem_ZeroCentsRejected(t *testing.T) {
	model.SetupTestDBForCommissionTests(t)
	_, err := Redeem(42, 0)
	require.Error(t, err)
}

func TestRedeem_NegativeCentsRejected(t *testing.T) {
	model.SetupTestDBForCommissionTests(t)
	_, err := Redeem(42, -100)
	require.Error(t, err)
}

func TestRedeem_ZeroRateRejected(t *testing.T) {
	model.SetupTestDBForCommissionTests(t)
	require.NoError(t, model.DB.Create(&model.User{Id: 42, Quota: 0, Username: "u42", AffCode: "a42"}).Error)
	seedBalance(t, 42, 1000)
	oldRate := operation_setting.USDExchangeRate
	defer func() { operation_setting.USDExchangeRate = oldRate }()
	operation_setting.USDExchangeRate = 0

	_, err := Redeem(42, 500)
	require.Error(t, err)
	require.Contains(t, err.Error(), "汇率")
}

func TestRedeem_LaterRateChangeDoesNotRewriteHistory(t *testing.T) {
	model.SetupTestDBForCommissionTests(t)
	require.NoError(t, model.DB.Create(&model.User{Id: 42, Quota: 0, Username: "u42", AffCode: "a42"}).Error)
	seedBalance(t, 42, 5000)
	oldRate, oldQPU := operation_setting.USDExchangeRate, common.QuotaPerUnit
	defer func() { operation_setting.USDExchangeRate, common.QuotaPerUnit = oldRate, oldQPU }()
	common.QuotaPerUnit = 500000

	operation_setting.USDExchangeRate = 7.2
	_, err := Redeem(42, 2500)
	require.NoError(t, err)

	operation_setting.USDExchangeRate = 6.5
	_, err = Redeem(42, 2500)
	require.NoError(t, err)

	rs, _ := model.ListRedemptionsByUser(42, 10, 0)
	require.Len(t, rs, 2)
	// Newest first.
	require.InDelta(t, 6.5, rs[0].USDExchangeRate, 0.001)
	require.InDelta(t, 7.2, rs[1].USDExchangeRate, 0.001)
}
