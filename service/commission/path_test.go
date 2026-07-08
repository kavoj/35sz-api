package commission

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/QuantumNous/new-api/model"
)

func TestBuildReferralPath_NoInviterNoOp(t *testing.T) {
	model.SetupTestDBForCommissionTests(t)

	require.NoError(t, BuildReferralPath(100, 0))

	p, err := model.GetReferralPath(100)
	require.NoError(t, err)
	require.Nil(t, p)
}

func TestBuildReferralPath_L1Only(t *testing.T) {
	model.SetupTestDBForCommissionTests(t)

	// Inviter (id=50) has no upline of their own.
	require.NoError(t, model.DB.Create(&model.User{Id: 50, InviterId: 0, Username: "u50", AffCode: "aff50"}).Error)
	require.NoError(t, BuildReferralPath(100, 50))

	p, err := model.GetReferralPath(100)
	require.NoError(t, err)
	require.NotNil(t, p)
	require.Equal(t, 50, p.L1UserId)
	require.Equal(t, 0, p.L2UserId)
}

func TestBuildReferralPath_L1AndL2Snapshot(t *testing.T) {
	model.SetupTestDBForCommissionTests(t)

	require.NoError(t, model.DB.Create(&model.User{Id: 20, InviterId: 0, Username: "u20", AffCode: "aff20"}).Error)
	require.NoError(t, model.DB.Create(&model.User{Id: 50, InviterId: 20, Username: "u50", AffCode: "aff50"}).Error)

	require.NoError(t, BuildReferralPath(100, 50))

	p, _ := model.GetReferralPath(100)
	require.Equal(t, 50, p.L1UserId)
	require.Equal(t, 20, p.L2UserId, "L2 must be inviter's inviter, snapshotted at register time")
}

func TestBuildReferralPath_InviterMissingKeepsL1(t *testing.T) {
	model.SetupTestDBForCommissionTests(t)

	// Inviter id 77 was never inserted, simulating a hard delete race.
	require.NoError(t, BuildReferralPath(100, 77))

	p, err := model.GetReferralPath(100)
	require.NoError(t, err)
	require.NotNil(t, p)
	require.Equal(t, 77, p.L1UserId)
	require.Equal(t, 0, p.L2UserId)
}

func TestBuildReferralPath_DuplicateInvocationPreservesSnapshot(t *testing.T) {
	model.SetupTestDBForCommissionTests(t)

	require.NoError(t, model.DB.Create(&model.User{Id: 20, InviterId: 0, Username: "u20", AffCode: "aff20"}).Error)
	require.NoError(t, model.DB.Create(&model.User{Id: 50, InviterId: 20, Username: "u50", AffCode: "aff50"}).Error)
	require.NoError(t, BuildReferralPath(100, 50))

	// Simulate a re-run: the original L2=20 must survive even if we called
	// BuildReferralPath again after some upstream mutation.
	require.NoError(t, model.DB.Model(&model.User{}).Where("id = ?", 50).Update("inviter_id", 999).Error)
	require.NoError(t, BuildReferralPath(100, 50))

	p, _ := model.GetReferralPath(100)
	require.Equal(t, 20, p.L2UserId, "path row is INSERT-only; the second call must not overwrite L2")
}
