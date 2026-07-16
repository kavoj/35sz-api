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
package ratio_setting

import (
	"strings"

	"github.com/QuantumNous/new-api/types"
)

// VendorOfficialPricingEntry mirrors the compiled-in
// constant/vendor_official_pricing.VendorPricingEntry shape but lives in a
// runtime-mutable OptionMap slot so admins can override the static baseline
// without shipping a new binary. Fed by the Vendor Pricing Sync admin page
// (PR-7e); consumed by the /v1/models/pricing 3-layer fallback in
// controller/models_pricing.go.
//
// The map key is the model name in the API-facing form (kebab-case with
// date suffix, e.g. `doubao-seedream-5-0-pro-260628`). Same normalization
// rules apply as elsewhere — the frontend and static baseline both use the
// `-N-M-` → `-N.M-` transform when matching, so a lookup for
// `doubao-seedream-5.0-pro` finds the -260628 dated entry, and vice versa.
//
// This is a duplicate-by-value of the constant-package struct (not shared)
// because setting/ratio_setting must not import constant/vendor_official_pricing
// (import graph: setting/ratio_setting is used by relay adapters, so pulling
// in the whole vendor catalog would inflate binary size and slow tests).
// Instead the controller layer copies field-by-field when merging.
type VendorOfficialPricingEntry struct {
	Kind string `json:"kind"`

	// chat / multimodal-chat / embedding
	InputPerMillionTokens  float64 `json:"input_per_million_tokens,omitempty"`
	OutputPerMillionTokens float64 `json:"output_per_million_tokens,omitempty"`

	// image-gen
	PricePerImage      float64            `json:"price_per_image,omitempty"`
	QualityMultipliers map[string]float64 `json:"quality_multipliers,omitempty"`
	SizeMultipliers    map[string]float64 `json:"size_multipliers,omitempty"`

	// video-gen
	PricePerSecond        float64            `json:"price_per_second,omitempty"`
	ResolutionMultipliers map[string]float64 `json:"resolution_multipliers,omitempty"`
	HasAudioMultiplier    float64            `json:"has_audio_multiplier,omitempty"`

	// audio-in
	PricePerMinute float64 `json:"price_per_minute,omitempty"`
	MinBillMinutes float64 `json:"min_bill_minutes,omitempty"`

	// audio-out
	PricePerMillionChars float64            `json:"price_per_million_chars,omitempty"`
	VoiceMultipliers     map[string]float64 `json:"voice_multipliers,omitempty"`

	// Provenance — filled by the sync handler when the admin uploads.
	// Not required for lookup, but shown in the admin UI so operators can
	// tell a stale entry from a fresh one and remember which vendor's
	// console they copied it from.
	Vendor      string `json:"vendor,omitempty"`       // e.g. "doubao", "openai"
	UpdatedAt   int64  `json:"updated_at,omitempty"`   // unix seconds
	UpdatedBy   int    `json:"updated_by,omitempty"`   // admin user id
	SourceNotes string `json:"source_notes,omitempty"` // free-form audit trail
}

// vendorOfficialPricingMap holds the runtime state populated from
// OptionMap["VendorOfficialPricing"]. Uses the standard RWMap so it
// behaves like every other OptionMap-backed setting (ModelRatio,
// ImagePricing, ...).
var vendorOfficialPricingMap = types.NewRWMap[string, VendorOfficialPricingEntry]()

// GetVendorOfficialPricing returns a single admin-overridden entry, or
// zero + false if the map has no such model. Callers should treat the
// bool as "does the admin want us to use this override?".
func GetVendorOfficialPricing(name string) (VendorOfficialPricingEntry, bool) {
	return vendorOfficialPricingMap.Get(strings.TrimSpace(name))
}

// GetVendorOfficialPricingCopy returns a snapshot of the full map. Used
// by the admin GET endpoint to display current overrides.
func GetVendorOfficialPricingCopy() map[string]VendorOfficialPricingEntry {
	return vendorOfficialPricingMap.ReadAll()
}

// VendorOfficialPricing2JSONString serializes the whole map for storage
// in OptionMap. Follows the same convention as ImagePricing2JSONString etc.
func VendorOfficialPricing2JSONString() string {
	return vendorOfficialPricingMap.MarshalJSONString()
}

// UpdateVendorOfficialPricingByJSONString replaces the runtime map with
// whatever the JSON blob decodes to. Called on service startup (option
// hydration) and whenever the admin submits the sync form.
func UpdateVendorOfficialPricingByJSONString(jsonStr string) error {
	return types.LoadFromJsonStringWithCallback(
		vendorOfficialPricingMap,
		jsonStr,
		InvalidateExposedDataCache,
	)
}

// SetVendorOfficialPricingEntries is the batched upsert path used by the
// admin sync handler. Overwrites keys with the incoming map's values,
// leaves other keys untouched. Returns the pre-merge state so the caller
// can diff for audit logging.
func SetVendorOfficialPricingEntries(
	entries map[string]VendorOfficialPricingEntry,
) map[string]VendorOfficialPricingEntry {
	before := vendorOfficialPricingMap.ReadAll()
	for k, v := range entries {
		vendorOfficialPricingMap.Set(strings.TrimSpace(k), v)
	}
	InvalidateExposedDataCache()
	return before
}

// DeleteVendorOfficialPricing removes a single override, letting the model
// fall back to the compiled-in static baseline (or to `none` if the vendor
// doesn't carry it either). Used by the admin UI's "Remove" action.
//
// RWMap has no direct Delete method — we copy everything except the
// target key, clear, and re-add. Fine for admin operations; would need a
// proper Delete if this got called on the hot path.
func DeleteVendorOfficialPricing(name string) {
	name = strings.TrimSpace(name)
	current := vendorOfficialPricingMap.ReadAll()
	if _, exists := current[name]; !exists {
		return
	}
	delete(current, name)
	vendorOfficialPricingMap.Clear()
	vendorOfficialPricingMap.AddAll(current)
	InvalidateExposedDataCache()
}
