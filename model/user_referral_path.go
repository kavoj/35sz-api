package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserReferralPath is a snapshot of a new user's L1/L2 uplines taken at the
// moment they register. It is INSERT-only: even if an upline later changes
// their own inviter, the downline's payout targets remain fixed.
type UserReferralPath struct {
	UserId    int   `json:"user_id" gorm:"primaryKey"`
	L1UserId  int   `json:"l1_user_id" gorm:"not null;index"`
	L2UserId  int   `json:"l2_user_id" gorm:"not null;default:0;index"`
	CreatedAt int64 `json:"created_at" gorm:"not null"`
}

func (UserReferralPath) TableName() string { return "user_referral_paths" }

// InsertReferralPath stores the (L1, L2) snapshot for userId. Duplicate calls
// with the same userId are ignored so we never overwrite the original snapshot
// if register logic is somehow retried.
func InsertReferralPath(userId, l1UserId, l2UserId int) error {
	path := UserReferralPath{
		UserId:    userId,
		L1UserId:  l1UserId,
		L2UserId:  l2UserId,
		CreatedAt: time.Now().Unix(),
	}
	return DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&path).Error
}

// GetReferralPath returns nil (with no error) when the user has no path row.
// Errors are only returned for real DB failures.
func GetReferralPath(userId int) (*UserReferralPath, error) {
	var path UserReferralPath
	err := DB.Where("user_id = ?", userId).First(&path).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &path, nil
}
