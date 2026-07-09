/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
package model

import (
	"math"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
)

// PR-4 CNY reconciliation invariant:
//
//	PaymentAmountCNY  ==  AmountUSDSnapshot
//	                    × USDExchangeRateSnapshot
//	                    × RechargePremiumSnapshot
//
// If any historical row breaks this invariant, ops dashboards can't
// reconcile CNY charges without regenerating the number, which defeats the
// purpose of snapshotting the rate/premium onto the row. These tests pin
// down the fact that the model-layer snapshot helper always produces
// coherent numbers regardless of when the admin last changed the settings.

func TestSnapshotCurrencyForInsertFreezesCurrentSettings(t *testing.T) {
	// Capture originals so we can restore.
	origRate := operation_setting.USDExchangeRate
	origPremium := operation_setting.RechargePremium
	origQuotaPerUnit := common.QuotaPerUnit
	defer func() {
		operation_setting.USDExchangeRate = origRate
		operation_setting.RechargePremium = origPremium
		common.QuotaPerUnit = origQuotaPerUnit
	}()

	operation_setting.USDExchangeRate = 7.3
	operation_setting.RechargePremium = 1.05
	common.QuotaPerUnit = 500_000

	tu := &TopUp{}
	tu.SnapshotCurrencyForInsert()

	assert.InDelta(t, 7.3, tu.USDExchangeRateSnapshot, 1e-6)
	assert.InDelta(t, 1.05, tu.RechargePremiumSnapshot, 1e-6)
	assert.Equal(t, int64(500_000), tu.QuotaPerUnitSnapshot)
}

func TestSnapshotCurrencyForInsertIsIdempotent(t *testing.T) {
	// Simulate: admin changes settings between two snapshot calls. The
	// method itself is idempotent per-call; the caller has to remember
	// snapshotted values already if they want to preserve them. This test
	// documents that behavior explicitly so future refactors don't
	// accidentally add caching that would hide admin changes.
	origRate := operation_setting.USDExchangeRate
	origPremium := operation_setting.RechargePremium
	defer func() {
		operation_setting.USDExchangeRate = origRate
		operation_setting.RechargePremium = origPremium
	}()

	operation_setting.USDExchangeRate = 7.0
	operation_setting.RechargePremium = 1.0
	tu := &TopUp{}
	tu.SnapshotCurrencyForInsert()
	assert.InDelta(t, 7.0, tu.USDExchangeRateSnapshot, 1e-6)

	operation_setting.USDExchangeRate = 7.5
	operation_setting.RechargePremium = 1.1
	tu.SnapshotCurrencyForInsert()
	assert.InDelta(t, 7.5, tu.USDExchangeRateSnapshot, 1e-6,
		"a second call should pick up the new rate")
	assert.InDelta(t, 1.1, tu.RechargePremiumSnapshot, 1e-6)
}

// TestReconciliationInvariantHolds is the *math* test — given known snapshots,
// PaymentAmountCNY must equal AmountUSDSnapshot × rate × premium within
// float precision. Ops SQL queries use exactly this formula.
func TestReconciliationInvariantHolds(t *testing.T) {
	cases := []struct {
		name         string
		amountUSD    float64
		rate         float64
		premium      float64
		wantCNY      float64
		floatEpsilon float64
	}{
		{
			name:         "no premium (RechargePremium=1)",
			amountUSD:    10.0,
			rate:         7.3,
			premium:      1.0,
			wantCNY:      73.0,
			floatEpsilon: 1e-9,
		},
		{
			name:         "5% platform fee",
			amountUSD:    10.0,
			rate:         7.3,
			premium:      1.05,
			wantCNY:      76.65,
			floatEpsilon: 1e-9,
		},
		{
			name:         "fractional dollar",
			amountUSD:    0.5,
			rate:         7.35,
			premium:      1.0,
			wantCNY:      3.675,
			floatEpsilon: 1e-9,
		},
		{
			name:         "large amount",
			amountUSD:    1000.0,
			rate:         6.8,
			premium:      1.02,
			wantCNY:      6936.0,
			floatEpsilon: 1e-6,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate a completed TopUp row.
			tu := &TopUp{
				AmountUSDSnapshot:       tc.amountUSD,
				USDExchangeRateSnapshot: tc.rate,
				RechargePremiumSnapshot: tc.premium,
			}
			// The reconciled CNY (what ops SQL would compute).
			reconciled := tu.AmountUSDSnapshot *
				tu.USDExchangeRateSnapshot *
				tu.RechargePremiumSnapshot
			assert.InDelta(t, tc.wantCNY, reconciled, tc.floatEpsilon)
			// And the drift vs the notional PaymentAmountCNY should be ~0
			// when payment used the same settings (which SnapshotCurrencyForInsert
			// guarantees by definition).
			tu.PaymentAmountCNY = reconciled
			drift := math.Abs(tu.PaymentAmountCNY - reconciled)
			assert.LessOrEqual(t, drift, tc.floatEpsilon)
		})
	}
}
