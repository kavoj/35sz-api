package service

import (
	"github.com/QuantumNous/new-api/service/commission"
)

// Registering the commission settlement handler here keeps the reference to
// service/commission out of system_task.go itself, avoiding a package cycle
// as future task types accumulate.
func init() {
	RegisterSystemTaskHandler(commission.NewSettleHandler())
}
