package commission

import (
	"errors"

	"gorm.io/gorm"

	"github.com/QuantumNous/new-api/model"
)

// BuildReferralPath persists the (L1, L2) snapshot for a newly registered
// user. It reads L1's own inviter_id as L2 so the downline pays out to the
// people who were L1's upline at register time — not whoever L1 later moves
// to. A zero inviterID means the new user registered without an invitation,
// so we insert nothing.
func BuildReferralPath(newUserID, inviterID int) error {
	if newUserID <= 0 || inviterID <= 0 {
		return nil
	}
	var inviter model.User
	if err := model.DB.Select("inviter_id").
		Where("id = ?", inviterID).
		First(&inviter).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Inviter was hard-deleted between AffCode lookup and now; still
			// record L1 so first-level commission works. L2 defaults to 0.
			return model.InsertReferralPath(newUserID, inviterID, 0)
		}
		return err
	}
	return model.InsertReferralPath(newUserID, inviterID, inviter.InviterId)
}
