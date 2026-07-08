package commission

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// setupCommissionScenario creates a small user tree with default L1=20% /
// L2=5% rules seeded. Callers may pass 0 for l2 to simulate an inviter with no
// upline.
func setupCommissionScenario(t *testing.T, l1, l2 int) {
	t.Helper()
	model.SetupTestDBForCommissionTests(t)
	require.NoError(t, model.SeedDefaultCommissionRules())
	if l2 != 0 {
		require.NoError(t, model.DB.Create(&model.User{Id: l2, InviterId: 0, Username: fmt.Sprintf("u%d", l2), AffCode: fmt.Sprintf("a%d", l2)}).Error)
	}
	if l1 != 0 {
		require.NoError(t, model.DB.Create(&model.User{Id: l1, InviterId: l2, Username: fmt.Sprintf("u%d", l1), AffCode: fmt.Sprintf("a%d", l1)}).Error)
	}
}

func makeTopup(id, userId int, money float64, provider string) *model.TopUp {
	return &model.TopUp{
		Id: id, UserId: userId, Money: money,
		TradeNo:         fmt.Sprintf("t-%d", id),
		Status:          common.TopUpStatusSuccess,
		PaymentProvider: provider,
	}
}

func TestOnTopupCompleted_FirstAlipayPaysBothTiers(t *testing.T) {
	setupCommissionScenario(t, 50, 20)
	require.NoError(t, model.DB.Create(&model.User{Id: 100, InviterId: 50, Username: "u100", AffCode: "a100"}).Error)
	require.NoError(t, BuildReferralPath(100, 50))

	topup := makeTopup(1, 100, 100, model.PaymentProviderAlipay)
	require.NoError(t, model.DB.Create(topup).Error)
	OnTopupCompleted(topup)

	var recs []model.CommissionRecord
	require.NoError(t, model.DB.Order("level ASC").Find(&recs).Error)
	require.Len(t, recs, 2)
	require.Equal(t, 50, recs[0].BeneficiaryId)
	require.EqualValues(t, 2000, recs[0].CommissionAmountCents, "L1 = floor(10000 * 20%) = 2000")
	require.Equal(t, 20, recs[1].BeneficiaryId)
	require.EqualValues(t, 500, recs[1].CommissionAmountCents, "L2 = floor(10000 * 5%) = 500")

	s1, _ := model.GetCommissionStats(50)
	s2, _ := model.GetCommissionStats(20)
	require.EqualValues(t, 2000, s1.CommissionPendingCents)
	require.EqualValues(t, 500, s2.CommissionPendingCents)
}

func TestOnTopupCompleted_NoPathNoOp(t *testing.T) {
	setupCommissionScenario(t, 50, 0)
	require.NoError(t, model.DB.Create(&model.User{Id: 100, InviterId: 0, Username: "u100", AffCode: "a100"}).Error)
	// No BuildReferralPath call → no path row.

	topup := makeTopup(1, 100, 100, model.PaymentProviderAlipay)
	require.NoError(t, model.DB.Create(topup).Error)
	OnTopupCompleted(topup)

	var n int64
	model.DB.Model(&model.CommissionRecord{}).Count(&n)
	require.EqualValues(t, 0, n)
}

func TestOnTopupCompleted_StripeIgnored(t *testing.T) {
	setupCommissionScenario(t, 50, 20)
	require.NoError(t, model.DB.Create(&model.User{Id: 100, InviterId: 50, Username: "u100", AffCode: "a100"}).Error)
	require.NoError(t, BuildReferralPath(100, 50))

	topup := makeTopup(1, 100, 100, model.PaymentProviderStripe)
	require.NoError(t, model.DB.Create(topup).Error)
	OnTopupCompleted(topup)

	var n int64
	model.DB.Model(&model.CommissionRecord{}).Count(&n)
	require.EqualValues(t, 0, n)
}

func TestOnTopupCompleted_ReplayIsIdempotent(t *testing.T) {
	setupCommissionScenario(t, 50, 20)
	require.NoError(t, model.DB.Create(&model.User{Id: 100, InviterId: 50, Username: "u100", AffCode: "a100"}).Error)
	require.NoError(t, BuildReferralPath(100, 50))

	topup := makeTopup(1, 100, 100, model.PaymentProviderAlipay)
	require.NoError(t, model.DB.Create(topup).Error)

	OnTopupCompleted(topup)
	OnTopupCompleted(topup) // replay

	var n int64
	model.DB.Model(&model.CommissionRecord{}).Count(&n)
	require.EqualValues(t, 2, n, "still just two records total (one per level)")

	s1, _ := model.GetCommissionStats(50)
	require.EqualValues(t, 2000, s1.CommissionPendingCents, "pending counter must not double-bump on replay")
}

func TestOnTopupCompleted_L2ZeroSkipsSecondTier(t *testing.T) {
	setupCommissionScenario(t, 50, 0)
	require.NoError(t, model.DB.Create(&model.User{Id: 100, InviterId: 50, Username: "u100", AffCode: "a100"}).Error)
	require.NoError(t, BuildReferralPath(100, 50))

	topup := makeTopup(1, 100, 100, model.PaymentProviderAlipay)
	require.NoError(t, model.DB.Create(topup).Error)
	OnTopupCompleted(topup)

	var recs []model.CommissionRecord
	model.DB.Find(&recs)
	require.Len(t, recs, 1)
	require.Equal(t, 1, recs[0].Level)
}

func TestOnTopupCompleted_DisabledLevelSkipped(t *testing.T) {
	setupCommissionScenario(t, 50, 20)
	require.NoError(t, model.DB.Model(&model.CommissionRule{}).
		Where("scope = ? AND level = ?", model.CommissionScopeFirstTopup, 2).
		Update("enabled", false).Error)
	require.NoError(t, model.DB.Create(&model.User{Id: 100, InviterId: 50, Username: "u100", AffCode: "a100"}).Error)
	require.NoError(t, BuildReferralPath(100, 50))

	topup := makeTopup(1, 100, 100, model.PaymentProviderAlipay)
	require.NoError(t, model.DB.Create(topup).Error)
	OnTopupCompleted(topup)

	var recs []model.CommissionRecord
	model.DB.Find(&recs)
	require.Len(t, recs, 1)
	require.Equal(t, 1, recs[0].Level)
}

func TestOnTopupCompleted_NotFirstIgnored(t *testing.T) {
	setupCommissionScenario(t, 50, 20)
	require.NoError(t, model.DB.Create(&model.User{Id: 100, InviterId: 50, Username: "u100", AffCode: "a100"}).Error)
	require.NoError(t, BuildReferralPath(100, 50))

	// Prior successful alipay topup already exists.
	require.NoError(t, model.DB.Create(makeTopup(1, 100, 50, model.PaymentProviderAlipay)).Error)

	topup2 := makeTopup(2, 100, 200, model.PaymentProviderWechat)
	require.NoError(t, model.DB.Create(topup2).Error)
	OnTopupCompleted(topup2)

	var n int64
	model.DB.Model(&model.CommissionRecord{}).Count(&n)
	require.EqualValues(t, 0, n, "not first success → no commission")
}

func TestOnTopupCompleted_FrozenUntilFromRule(t *testing.T) {
	setupCommissionScenario(t, 50, 0)
	require.NoError(t, model.DB.Model(&model.CommissionRule{}).
		Where("scope = ? AND level = ?", model.CommissionScopeFirstTopup, 1).
		Update("frozen_days", 3).Error)
	require.NoError(t, model.DB.Create(&model.User{Id: 100, InviterId: 50, Username: "u100", AffCode: "a100"}).Error)
	require.NoError(t, BuildReferralPath(100, 50))

	topup := makeTopup(1, 100, 100, model.PaymentProviderAlipay)
	require.NoError(t, model.DB.Create(topup).Error)
	before := time.Now().Unix()
	OnTopupCompleted(topup)

	var rec model.CommissionRecord
	require.NoError(t, model.DB.First(&rec).Error)
	require.InDelta(t, before+int64(3*86400), rec.FrozenUntil, 60)
}

func TestOnTopupCompleted_CentsFloor(t *testing.T) {
	setupCommissionScenario(t, 50, 0)
	require.NoError(t, model.DB.Model(&model.CommissionRule{}).
		Where("scope = ? AND level = ?", model.CommissionScopeFirstTopup, 1).
		Update("rate_percent", 33.33).Error)
	require.NoError(t, model.DB.Create(&model.User{Id: 100, InviterId: 50, Username: "u100", AffCode: "a100"}).Error)
	require.NoError(t, BuildReferralPath(100, 50))

	// ¥1 topup = 100 cents; 100 * 33.33 / 100 = 33.33 → floor = 33.
	topup := makeTopup(1, 100, 1, model.PaymentProviderAlipay)
	require.NoError(t, model.DB.Create(topup).Error)
	OnTopupCompleted(topup)

	var rec model.CommissionRecord
	require.NoError(t, model.DB.First(&rec).Error)
	require.EqualValues(t, int64(math.Floor(100*33.33/100.0)), rec.CommissionAmountCents)
}

func TestOnTopupCompleted_MinTopupGate(t *testing.T) {
	setupCommissionScenario(t, 50, 0)
	require.NoError(t, model.DB.Model(&model.CommissionRule{}).
		Where("scope = ?", model.CommissionScopeFirstTopup).
		Update("min_topup_cents", 20000).Error) // ¥200 threshold
	require.NoError(t, model.DB.Create(&model.User{Id: 100, InviterId: 50, Username: "u100", AffCode: "a100"}).Error)
	require.NoError(t, BuildReferralPath(100, 50))

	// ¥100 topup < ¥200 gate.
	topup := makeTopup(1, 100, 100, model.PaymentProviderAlipay)
	require.NoError(t, model.DB.Create(topup).Error)
	OnTopupCompleted(topup)

	var n int64
	model.DB.Model(&model.CommissionRecord{}).Count(&n)
	require.EqualValues(t, 0, n)
}
