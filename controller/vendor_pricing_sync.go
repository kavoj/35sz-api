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

// Package controller — Vendor Pricing Sync
//
// Overview
// --------
// PR-7e introduces a runtime channel for admins to override the compiled-in
// vendor pricing baseline (constant/vendor_official_pricing) without shipping
// a new binary. This is critical for Doubao / Volcano Engine where prices
// change unpredictably; the admin copies the vendor console's pricing table
// into the sync page and the numbers flow immediately into
// /v1/models/pricing responses.
//
// The runtime storage lives in OptionMap["VendorOfficialPricing"] (see
// setting/ratio_setting/vendor_official_pricing.go). This file wires up
// the HTTP endpoints; the /v1/models/pricing handler consults the runtime
// map as layer 2 of its 3-layer fallback (user_configured →
// vendor_official → static_baseline → none).
//
// Auth
// ----
// Both endpoints are admin-only. The frontend page is under
// /system-settings/billing/vendor-pricing-sync which the router already
// gates on admin session.

package controller

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

// vendorPricingSyncRequest is the POST body — a batch of entries the admin
// wants to upsert. `entries` map key must be the exact API-facing model
// name (e.g. `doubao-seedream-5-0-pro-260628`). `replace_all` set to true
// causes the runtime map to be REPLACED wholesale (delete anything not in
// the incoming batch); false only upserts, leaving unmentioned entries
// alone.
//
// The batch shape lets the admin paste ~20 rows at once (typical Doubao
// vendor page has 15-30 models) with a single roundtrip.
type vendorPricingSyncRequest struct {
	Entries    map[string]ratio_setting.VendorOfficialPricingEntry `json:"entries"`
	ReplaceAll bool                                                `json:"replace_all,omitempty"`
	SourceNote string                                              `json:"source_note,omitempty"`
}

// vendorPricingSyncResponse describes the post-sync state and any diff
// with what was already there. Frontend uses `merged_count` /
// `replaced_count` / `unchanged_count` to render a summary toast.
type vendorPricingSyncResponse struct {
	MergedCount    int    `json:"merged_count"`    // upserted entries (new + updated)
	ReplacedCount  int    `json:"replaced_count"`  // pre-existing entries that got overwritten
	UnchangedCount int    `json:"unchanged_count"` // entries that stayed identical
	DeletedCount   int    `json:"deleted_count"`   // only non-zero when replace_all=true
	SourceNote     string `json:"source_note"`     // echoed back for audit UI
}

// GetVendorPricingOverrides returns every runtime override the admin has
// configured. Read-only endpoint used by the sync page to render the
// "current overrides" panel.
func GetVendorPricingOverrides(c *gin.Context) {
	entries := ratio_setting.GetVendorOfficialPricingCopy()
	c.JSON(200, gin.H{
		"success":     true,
		"entries":     entries,
		"total_count": len(entries),
	})
}

// PostVendorPricingSync applies an incoming batch of overrides.
//
// Behavior:
//   - Merges by default. `replace_all=true` deletes anything not in the
//     batch (useful when the admin has a full curated list and wants to
//     match it exactly).
//   - Each entry gets a `Vendor / UpdatedAt / UpdatedBy / SourceNotes`
//     stamp from the request context so the admin panel can show
//     provenance later.
//   - Writes the merged map back to OptionMap["VendorOfficialPricing"]
//     via model.UpdateOption so the change survives restarts.
//
// Validation:
//   - Empty batch → 400, refuse the roundtrip.
//   - Each entry must have a non-empty Kind matching the 7 known values.
//     Malformed rows are dropped with a warning (not fatal — the admin
//     usually wants the batch to still apply for the valid rows).
func PostVendorPricingSync(c *gin.Context) {
	var req vendorPricingSyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if len(req.Entries) == 0 {
		common.ApiErrorMsg(c, "entries is empty")
		return
	}

	adminID := c.GetInt("id")
	nowTs := common.GetTimestamp()

	// Sanitize + stamp provenance.
	validEntries := make(map[string]ratio_setting.VendorOfficialPricingEntry, len(req.Entries))
	for name, entry := range req.Entries {
		key := strings.TrimSpace(name)
		if key == "" {
			continue
		}
		if !isValidPricingKindForVendorSync(entry.Kind) {
			continue
		}
		if entry.UpdatedAt == 0 {
			entry.UpdatedAt = nowTs
		}
		if entry.UpdatedBy == 0 {
			entry.UpdatedBy = adminID
		}
		if entry.SourceNotes == "" && req.SourceNote != "" {
			entry.SourceNotes = req.SourceNote
		}
		validEntries[key] = entry
	}
	if len(validEntries) == 0 {
		common.ApiErrorMsg(c, "no valid entries in payload; each entry must carry a supported 'kind' field")
		return
	}

	// Merge with the current runtime map. When replace_all is true, we
	// clear first so unmentioned keys drop away.
	previous := ratio_setting.GetVendorOfficialPricingCopy()
	deleted := 0
	if req.ReplaceAll {
		for name := range previous {
			if _, keep := validEntries[name]; !keep {
				ratio_setting.DeleteVendorOfficialPricing(name)
				deleted++
			}
		}
	}
	ratio_setting.SetVendorOfficialPricingEntries(validEntries)

	// Persist. We serialize the current whole map (not the incoming batch)
	// because the RWMap holds both merged and pre-existing entries at this
	// point.
	blob := ratio_setting.VendorOfficialPricing2JSONString()
	if err := model.UpdateOption("VendorOfficialPricing", blob); err != nil {
		common.ApiError(c, err)
		return
	}

	// Compute the response summary. `unchanged` is entries where the JSON
	// serialization matched pre-image — used by the UI's "no-op" indicator
	// so the admin can tell when a paste from the vendor console had no
	// real diff.
	merged := 0
	replaced := 0
	unchanged := 0
	for name, next := range validEntries {
		prev, existed := previous[name]
		if !existed {
			merged++
			continue
		}
		if entriesEqual(prev, next) {
			unchanged++
		} else {
			replaced++
			merged++
		}
	}

	c.JSON(200, gin.H{
		"success": true,
		"result": vendorPricingSyncResponse{
			MergedCount:    merged,
			ReplacedCount:  replaced,
			UnchangedCount: unchanged,
			DeletedCount:   deleted,
			SourceNote:     req.SourceNote,
		},
	})
}

// DeleteVendorPricingOverride removes a single runtime override. The model
// then falls back to the compiled-in static baseline (or to
// `pricing_incomplete: true` if nothing there either).
func DeleteVendorPricingOverride(c *gin.Context) {
	name := strings.TrimSpace(c.Param("model_name"))
	if name == "" {
		common.ApiErrorMsg(c, "model_name is required")
		return
	}
	ratio_setting.DeleteVendorOfficialPricing(name)
	blob := ratio_setting.VendorOfficialPricing2JSONString()
	if err := model.UpdateOption("VendorOfficialPricing", blob); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(200, gin.H{"success": true})
}

// isValidPricingKindForVendorSync guards the 7-value enum. We do NOT reuse
// constant.IsValidPricingKind because that file expects the caller to
// have already normalized — here we get raw admin input and want to be
// strict about acceptable spellings.
func isValidPricingKindForVendorSync(k string) bool {
	switch k {
	case "chat", "multimodal-chat", "image-gen", "video-gen",
		"audio-in", "audio-out", "embedding":
		return true
	}
	return false
}

// entriesEqual is a shallow value-comparison for two entries. Used only
// for reporting purposes (unchanged_count in the response). It compares
// the fields most likely to drift and treats maps as equal iff they have
// identical keys AND values — good enough for a UI hint, not a fidelity
// check.
func entriesEqual(a, b ratio_setting.VendorOfficialPricingEntry) bool {
	if a.Kind != b.Kind ||
		a.InputPerMillionTokens != b.InputPerMillionTokens ||
		a.OutputPerMillionTokens != b.OutputPerMillionTokens ||
		a.PricePerImage != b.PricePerImage ||
		a.PricePerSecond != b.PricePerSecond ||
		a.HasAudioMultiplier != b.HasAudioMultiplier ||
		a.PricePerMinute != b.PricePerMinute ||
		a.MinBillMinutes != b.MinBillMinutes ||
		a.PricePerMillionChars != b.PricePerMillionChars {
		return false
	}
	if !stringFloatMapEqual(a.QualityMultipliers, b.QualityMultipliers) ||
		!stringFloatMapEqual(a.SizeMultipliers, b.SizeMultipliers) ||
		!stringFloatMapEqual(a.ResolutionMultipliers, b.ResolutionMultipliers) ||
		!stringFloatMapEqual(a.VoiceMultipliers, b.VoiceMultipliers) {
		return false
	}
	return true
}

func stringFloatMapEqual(a, b map[string]float64) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
