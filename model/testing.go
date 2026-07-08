package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// SetupTestDBForCommissionTests initializes an in-memory SQLite DB with every
// table the commission subsystem's tests need across packages. Exported so
// tests in service/commission and controller can call it. Never invoke from
// production code — it swaps the package-level DB pointer.
func SetupTestDBForCommissionTests(t *testing.T) {
	t.Helper()
	oldDB := DB
	oldType := common.MainDatabaseType()
	DB = nil
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	t.Cleanup(func() {
		DB = oldDB
		common.SetMainDatabaseType(oldType)
	})
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	DB = db
	require.NoError(t, DB.AutoMigrate(
		&User{}, &TopUp{}, &Log{},
		&CommissionRule{}, &CommissionRecord{}, &CommissionRedemption{},
		&UserCommissionStats{}, &UserReferralPath{},
	))
}
