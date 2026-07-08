package controller

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/commission"
)

// ---------- User-facing endpoints ----------

// GetCommissionStats returns the caller's commission counters plus their
// AffCode so the frontend can render the invite link/QR without a second
// round-trip.
func GetCommissionStats(c *gin.Context) {
	uid := c.GetInt("id")
	user, err := model.GetUserById(uid, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	stats, err := model.GetCommissionStats(uid)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if stats == nil {
		stats = &model.UserCommissionStats{UserId: uid}
	}
	common.ApiSuccess(c, gin.H{
		"aff_code":                  user.AffCode,
		"commission_balance_cents":  stats.CommissionBalanceCents,
		"commission_pending_cents":  stats.CommissionPendingCents,
		"commission_lifetime_cents": stats.CommissionLifetimeCents,
		"commission_redeemed_cents": stats.CommissionRedeemedCents,
	})
}

// GetMyCommissionRecords returns the caller's commission ledger, filtered by
// an optional status query parameter and paginated by (page, size).
func GetMyCommissionRecords(c *gin.Context) {
	uid := c.GetInt("id")
	status := c.Query("status")
	page, size := parsePagination(c)

	q := model.DB.Model(&model.CommissionRecord{}).Where("beneficiary_id = ?", uid)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	q.Count(&total)

	var records []model.CommissionRecord
	if err := q.Order("id DESC").Limit(size).Offset((page - 1) * size).Find(&records).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"records": records, "total": total, "page": page, "size": size})
}

// GetMyRedemptions returns the caller's commission→wallet conversion history.
func GetMyRedemptions(c *gin.Context) {
	uid := c.GetInt("id")
	page, size := parsePagination(c)
	list, err := model.ListRedemptionsByUser(uid, size, (page-1)*size)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"records": list, "page": page, "size": size})
}

// GetMyDownlines returns the caller's L1 or L2 downlines with PII masked.
// The masking policy matches the design spec: username keeps its first rune,
// email keeps its first char + full domain.
func GetMyDownlines(c *gin.Context) {
	uid := c.GetInt("id")
	level := c.DefaultQuery("level", "1")
	page, size := parsePagination(c)

	q := model.DB.Table("user_referral_paths urp").
		Select("urp.user_id, urp.created_at, u.username, u.email").
		Joins("LEFT JOIN users u ON u.id = urp.user_id")
	switch level {
	case "1":
		q = q.Where("urp.l1_user_id = ?", uid)
	case "2":
		q = q.Where("urp.l2_user_id = ?", uid)
	default:
		common.ApiErrorMsg(c, "level must be 1 or 2")
		return
	}
	var total int64
	q.Count(&total)

	rows := []struct {
		UserId    int    `json:"user_id"`
		CreatedAt int64  `json:"created_at"`
		Username  string `json:"username"`
		Email     string `json:"email"`
	}{}
	if err := q.Order("urp.created_at DESC").Limit(size).Offset((page - 1) * size).Scan(&rows).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	for i := range rows {
		rows[i].Username = maskUsername(rows[i].Username)
		rows[i].Email = maskEmail(rows[i].Email)
	}
	common.ApiSuccess(c, gin.H{"rows": rows, "total": total, "page": page, "size": size})
}

// GetQuotaPreview returns the wallet quota the caller would receive if they
// redeemed `cents` right now, using the live system exchange rate.
func GetQuotaPreview(c *gin.Context) {
	cents, _ := strconv.ParseInt(c.Query("cents"), 10, 64)
	quota, rate, qpu, err := commission.PreviewQuota(cents)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, gin.H{"quota_credited": quota, "usd_exchange_rate": rate, "quota_per_unit": qpu})
}

type redeemRequest struct {
	Cents int64 `json:"cents"`
}

// RedeemCommission converts commission balance into wallet quota. Delegates
// to the service layer where the balance guard + rate snapshot live.
func RedeemCommission(c *gin.Context) {
	uid := c.GetInt("id")
	var req redeemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	q, err := commission.Redeem(uid, req.Cents)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, gin.H{"quota_credited": q})
}

// ---------- PII masking helpers ----------
//
// The rules are deliberately simple: the goal is enough anonymity that a user
// browsing their downlines can't harass them, not cryptographic privacy.

func maskUsername(s string) string {
	rs := []rune(s)
	if len(rs) <= 1 {
		return s
	}
	return string(rs[:1]) + strings.Repeat("*", len(rs)-1)
}

func maskEmail(s string) string {
	at := strings.Index(s, "@")
	if at <= 1 {
		return s
	}
	return s[:1] + "***" + s[at:]
}

// ---------- shared helpers ----------

func parsePagination(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	return page, size
}
