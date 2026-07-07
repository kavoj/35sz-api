package model

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/QuantumNous/new-api/common"
)

func setupRecordsTestDB(t *testing.T) {
	t.Helper()
	setupCommissionTestDB(t)
	require.NoError(t, DB.AutoMigrate(
		&UserReferralPath{},
		&UserCommissionStats{},
		&CommissionRecord{},
		&TopUp{},
	))
}

func TestCommissionRecord_InsertIsIdempotent(t *testing.T) {
	setupRecordsTestDB(t)

	rec := CommissionRecord{
		BeneficiaryId:         50,
		SourceUserId:          100,
		SourceTopupId:         999,
		Scope:                 CommissionScopeFirstTopup,
		Level:                 1,
		RatePercent:           20,
		BaseAmountCents:       10000,
		CommissionAmountCents: 2000,
		Status:                CommissionStatusPending,
		FrozenUntil:           1700000000,
	}

	inserted, err := InsertCommissionRecord(DB, &rec)
	require.NoError(t, err)
	require.True(t, inserted)

	// Retrying with the same (topup, beneficiary, level) drops the duplicate.
	rec2 := rec
	rec2.Id = 0 // ensure it isn't a primary-key collision case
	inserted2, err := InsertCommissionRecord(DB, &rec2)
	require.NoError(t, err)
	require.False(t, inserted2)

	var count int64
	require.NoError(t, DB.Model(&CommissionRecord{}).Count(&count).Error)
	require.EqualValues(t, 1, count)
}

func TestIsFirstDomesticSuccessTopup(t *testing.T) {
	setupRecordsTestDB(t)

	// Two topups on the same user: only alipay/wxpay/epay count.
	require.NoError(t, DB.Create(&TopUp{Id: 1, UserId: 42, Money: 100, TradeNo: "t-1", Status: common.TopUpStatusSuccess, PaymentProvider: PaymentProviderAlipay}).Error)
	require.NoError(t, DB.Create(&TopUp{Id: 2, UserId: 42, Money: 50, TradeNo: "t-2", Status: common.TopUpStatusSuccess, PaymentProvider: PaymentProviderStripe}).Error)

	require.True(t, IsFirstDomesticSuccessTopup(42, 1), "id=1 (alipay) is the only domestic success ⇒ first")

	require.NoError(t, DB.Create(&TopUp{Id: 3, UserId: 42, Money: 200, TradeNo: "t-3", Status: common.TopUpStatusSuccess, PaymentProvider: PaymentProviderWechat}).Error)
	require.False(t, IsFirstDomesticSuccessTopup(42, 3), "wechat topup with a prior alipay ⇒ not first")
}

func TestIsFirstDomesticSuccessTopup_MultipleDomestic(t *testing.T) {
	setupRecordsTestDB(t)
	require.NoError(t, DB.Create(&TopUp{Id: 10, UserId: 42, Money: 100, TradeNo: "t-10", Status: common.TopUpStatusSuccess, PaymentProvider: PaymentProviderAlipay}).Error)
	require.NoError(t, DB.Create(&TopUp{Id: 11, UserId: 42, Money: 200, TradeNo: "t-11", Status: common.TopUpStatusSuccess, PaymentProvider: PaymentProviderWechat}).Error)

	require.False(t, IsFirstDomesticSuccessTopup(42, 11))
}

func TestIsFirstDomesticSuccessTopup_PendingIgnored(t *testing.T) {
	setupRecordsTestDB(t)
	require.NoError(t, DB.Create(&TopUp{Id: 20, UserId: 42, Money: 100, TradeNo: "t-20", Status: common.TopUpStatusPending, PaymentProvider: PaymentProviderAlipay}).Error)
	require.NoError(t, DB.Create(&TopUp{Id: 21, UserId: 42, Money: 200, TradeNo: "t-21", Status: common.TopUpStatusSuccess, PaymentProvider: PaymentProviderWechat}).Error)

	require.True(t, IsFirstDomesticSuccessTopup(42, 21), "pending topups don't count")
}

func TestFetchPendingDueCommissionRecords(t *testing.T) {
	setupRecordsTestDB(t)

	past := int64(1)      // in the far past
	future := int64(1<<62) // far future
	// One due, one not yet.
	_, err := InsertCommissionRecord(DB, &CommissionRecord{
		BeneficiaryId: 1, SourceUserId: 100, SourceTopupId: 1, Scope: CommissionScopeFirstTopup, Level: 1,
		RatePercent: 20, BaseAmountCents: 10000, CommissionAmountCents: 2000,
		Status: CommissionStatusPending, FrozenUntil: past,
	})
	require.NoError(t, err)
	_, err = InsertCommissionRecord(DB, &CommissionRecord{
		BeneficiaryId: 2, SourceUserId: 101, SourceTopupId: 2, Scope: CommissionScopeFirstTopup, Level: 1,
		RatePercent: 20, BaseAmountCents: 10000, CommissionAmountCents: 2000,
		Status: CommissionStatusPending, FrozenUntil: future,
	})
	require.NoError(t, err)

	batch, err := FetchPendingDueCommissionRecords(100)
	require.NoError(t, err)
	require.Len(t, batch, 1)
	require.Equal(t, 1, batch[0].BeneficiaryId)
}

func TestGetCommissionRecordByID(t *testing.T) {
	setupRecordsTestDB(t)
	_, err := InsertCommissionRecord(DB, &CommissionRecord{
		BeneficiaryId: 1, SourceUserId: 100, SourceTopupId: 1, Scope: CommissionScopeFirstTopup, Level: 1,
		RatePercent: 20, BaseAmountCents: 10000, CommissionAmountCents: 2000,
		Status: CommissionStatusPending, FrozenUntil: 100,
	})
	require.NoError(t, err)

	rec, err := GetCommissionRecordByID(1)
	require.NoError(t, err)
	require.Equal(t, 100, rec.SourceUserId)

	_, err = GetCommissionRecordByID(999)
	require.Error(t, err)
}
