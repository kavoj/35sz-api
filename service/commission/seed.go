package commission

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// SeedDefaultRules is called once at process startup, after AutoMigrate, to
// guarantee the default L1=20% / L2=5% rules exist. It is idempotent: the
// underlying model call skips insertion if any rule row is already present.
// Failures are logged but never fatal — commission-related features simply
// stay disabled until an operator seeds rules manually.
func SeedDefaultRules() {
	if err := model.SeedDefaultCommissionRules(); err != nil {
		common.SysError("commission rules seed failed: " + err.Error())
	}
}
