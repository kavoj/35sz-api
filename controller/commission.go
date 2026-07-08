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

// ---------- Admin endpoints ----------
//
// Guarded by middleware.RootAuth at the route layer.

// AdminListCommissionRules returns every rule (enabled or not) so the config
// UI can render disabled rows too.
func AdminListCommissionRules(c *gin.Context) {
	rules, err := model.ListCommissionRules()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, rules)
}

type adminUpdateRuleRequest struct {
	RatePercent   *float64 `json:"rate_percent"`
	MinTopupCents *int64   `json:"min_topup_cents"`
	FrozenDays    *int     `json:"frozen_days"`
	Enabled       *bool    `json:"enabled"`
}

// AdminUpdateCommissionRule mutates the tunable fields on one rule. Uses
// pointer fields so an omitted key is not accidentally reset to zero.
func AdminUpdateCommissionRule(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req adminUpdateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	updates := map[string]any{}
	if req.RatePercent != nil {
		updates["rate_percent"] = *req.RatePercent
	}
	if req.MinTopupCents != nil {
		updates["min_topup_cents"] = *req.MinTopupCents
	}
	if req.FrozenDays != nil {
		updates["frozen_days"] = *req.FrozenDays
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if len(updates) == 0 {
		common.ApiSuccess(c, nil)
		return
	}
	if err := model.UpdateCommissionRule(id, updates); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

// AdminListRecords is the platform-wide commission ledger view. Accepts
// optional status / beneficiary / source_user / created_at range filters.
func AdminListRecords(c *gin.Context) {
	status := c.Query("status")
	beneficiary, _ := strconv.Atoi(c.Query("beneficiary"))
	source, _ := strconv.Atoi(c.Query("source"))
	from, _ := strconv.ParseInt(c.Query("from"), 10, 64)
	to, _ := strconv.ParseInt(c.Query("to"), 10, 64)
	page, size := parsePagination(c)

	q := model.DB.Model(&model.CommissionRecord{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if beneficiary != 0 {
		q = q.Where("beneficiary_id = ?", beneficiary)
	}
	if source != 0 {
		q = q.Where("source_user_id = ?", source)
	}
	if from > 0 {
		q = q.Where("created_at >= ?", from)
	}
	if to > 0 {
		q = q.Where("created_at <= ?", to)
	}
	var total int64
	q.Count(&total)

	var out []model.CommissionRecord
	if err := q.Order("id DESC").Limit(size).Offset((page - 1) * size).Find(&out).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"records": out, "total": total, "page": page, "size": size})
}

type adminVoidRequest struct {
	Reason string `json:"reason"`
}

// AdminVoidRecord flips a commission record to voided and rolls back the
// beneficiary's pending/balance counters. Delegates to commission.Void so all
// the state-machine rules live in one place.
func AdminVoidRecord(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req adminVoidRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if err := commission.Void(id, req.Reason); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, nil)
}

// AdminSettleNow drains the pending-and-due queue on demand instead of
// waiting for the scheduled runner. Used by the admin "settle now" button.
func AdminSettleNow(c *gin.Context) {
	n, err := commission.SettlePending()
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, gin.H{"settled": n})
}

// AdminCommissionOverview returns platform-wide totals for the admin
// dashboard. Uses coalesce/sum so zero-row tables still return a document.
func AdminCommissionOverview(c *gin.Context) {
	type overview struct {
		TotalCents        int64 `json:"total_cents"`
		SettledCents      int64 `json:"settled_cents"`
		PendingCents      int64 `json:"pending_cents"`
		RedeemedCents     int64 `json:"redeemed_cents"`
		ParticipantsCount int64 `json:"participants_count"`
		FirstTopupCount   int64 `json:"first_topup_count"`
	}
	var out overview
	model.DB.Model(&model.CommissionRecord{}).
		Where("status IN ?", []string{model.CommissionStatusPending, model.CommissionStatusSettled}).
		Select("COALESCE(SUM(commission_amount_cents),0)").Scan(&out.TotalCents)
	model.DB.Model(&model.CommissionRecord{}).
		Where("status = ?", model.CommissionStatusSettled).
		Select("COALESCE(SUM(commission_amount_cents),0)").Scan(&out.SettledCents)
	model.DB.Model(&model.CommissionRecord{}).
		Where("status = ?", model.CommissionStatusPending).
		Select("COALESCE(SUM(commission_amount_cents),0)").Scan(&out.PendingCents)
	model.DB.Model(&model.CommissionRedemption{}).
		Select("COALESCE(SUM(commission_cents),0)").Scan(&out.RedeemedCents)
	model.DB.Model(&model.UserReferralPath{}).Count(&out.ParticipantsCount)
	model.DB.Model(&model.CommissionRecord{}).
		Where("level = ?", 1).
		Distinct("source_user_id").Count(&out.FirstTopupCount)

	common.ApiSuccess(c, out)
}
