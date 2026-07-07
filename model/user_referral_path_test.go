package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func setupReferralPathTestDB(t *testing.T) {
	t.Helper()
	setupCommissionTestDB(t) // base migrate (CommissionRule)
	require.NoError(t, DB.AutoMigrate(&UserReferralPath{}))
}

func TestUserReferralPath_InsertAndQuery(t *testing.T) {
	setupReferralPathTestDB(t)

	require.NoError(t, InsertReferralPath(100, 50, 20))

	path, err := GetReferralPath(100)
	require.NoError(t, err)
	require.NotNil(t, path)
	require.Equal(t, 50, path.L1UserId)
	require.Equal(t, 20, path.L2UserId)
}

func TestUserReferralPath_NoUpper(t *testing.T) {
	setupReferralPathTestDB(t)
	require.NoError(t, InsertReferralPath(101, 50, 0))

	path, err := GetReferralPath(101)
	require.NoError(t, err)
	require.NotNil(t, path)
	require.Equal(t, 50, path.L1UserId)
	require.Equal(t, 0, path.L2UserId)
}

func TestUserReferralPath_NotFoundReturnsNilNilNoError(t *testing.T) {
	setupReferralPathTestDB(t)
	path, err := GetReferralPath(999)
	require.NoError(t, err)
	require.Nil(t, path)
}

func TestUserReferralPath_DuplicateInsertIgnored(t *testing.T) {
	setupReferralPathTestDB(t)
	require.NoError(t, InsertReferralPath(200, 30, 0))
	// Second insert must not error nor overwrite the L1.
	require.NoError(t, InsertReferralPath(200, 40, 0))

	path, err := GetReferralPath(200)
	require.NoError(t, err)
	require.Equal(t, 30, path.L1UserId, "existing referral path must be preserved")
}
