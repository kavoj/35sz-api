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

// VideoPricing captures per-second pricing for a video generation model.
//
// This replaces two hardcoded tables that previously lived in relay code:
//
//   - `relay/channel/task/gemini/billing.go::VeoResolutionRatio` — Google Veo
//     4K uplift ratios were embedded as if/else on model name.
//   - `relay/channel/task/doubao/constants.go::videoPriceTable` — Doubao
//     Seedance per-resolution + video-input pricing table.
//
// Both are now expressible as admin-editable JSON, and the relay layer looks
// them up here first before falling back to the hardcoded defaults.
//
// Storage
// -------
// - Persisted as JSON in OptionMap["VideoPricing"] alongside the existing
//   OptionMap["ModelPrice"] etc.
// - Key = model name (matches ModelRatio, CompletionRatio, etc.)
// - Value = this struct.
//
// Units
// -----
// - PricePerSecond is USD per second, ALREADY-CONVERTED to base USD by the
//   admin UI (see web/default/src/lib/currency.ts::convertBillingDisplayToUSD).
//   Same convention as ModelPrice — the setting layer never touches CNY.
// - ResolutionMultipliers is dimensionless. Keys are lowercase strings like
//   "480p" / "720p" / "1080p" / "4k". Missing keys imply 1.0 (base pricing).
// - HasAudioMultiplier applies when the video model produces audio+video
//   output; e.g. Veo 3.1 charges 1.5× on audio-on renders. 0 or 1 means "no
//   audio uplift" (audio-off pricing).
//
// Billing formula
// ---------------
//   totalUSD = durationSeconds
//            × PricePerSecond
//            × ResolutionMultipliers[resolution]  (default 1)
//            × (HasAudioMultiplier if audio-on else 1)
type VideoPricing struct {
	PricePerSecond        float64            `json:"price_per_second"`
	ResolutionMultipliers map[string]float64 `json:"resolution_multipliers"`
	HasAudioMultiplier    float64            `json:"has_audio_multiplier,omitempty"`
}

// videoPricingMap is the runtime cache. Populated by
// UpdateVideoPricingByJSONString on option reload and seeded in InitRatioMap.
var videoPricingMap = types.NewRWMap[string, VideoPricing]()

// defaultVideoPricing seeds the map with values that mirror the hardcoded
// tables so an admin who never edits the JSON gets identical behavior to
// pre-refactor.
//
// The "base USD" values below were derived from the doubao CNY prices divided
// by USDExchangeRate=7.3 (matches the audit-verified rate used by commission
// redemption and topup). We keep six-decimal precision to avoid drift when
// the admin ratio table is re-derived on the frontend.
//
// Veo entries come from VeoResolutionRatio and Google's public per-second
// pricing (base USD per second for the audio-off variant); the 4K uplift is
// applied through ResolutionMultipliers["4k"].
var defaultVideoPricing = map[string]VideoPricing{
	// -------- Google Veo family --------
	"veo-3.1-generate": {
		PricePerSecond: 0.40,
		ResolutionMultipliers: map[string]float64{
			"720p":  1.0,
			"1080p": 1.0,
			"4k":    1.5, // $0.60 / $0.40
		},
		HasAudioMultiplier: 1.0, // audio included in base price
	},
	"veo-3.1-fast-generate": {
		PricePerSecond: 0.15,
		ResolutionMultipliers: map[string]float64{
			"720p":  1.0,
			"1080p": 1.0,
			"4k":    2.333333, // $0.35 / $0.15
		},
		HasAudioMultiplier: 1.0,
	},
	"veo-3.0-generate": {
		PricePerSecond: 0.40,
		ResolutionMultipliers: map[string]float64{
			"720p":  1.0,
			"1080p": 1.0,
			// veo-3.0 does not support 4K — leave key unset so lookups return 1.
		},
		HasAudioMultiplier: 1.0,
	},

	// -------- Doubao Seedance --------
	// Converted from CNY/1M-token pricing at rate 7.3; the resolution
	// multipliers here mirror the per-tier ratios that videoPriceTable
	// encoded before (e.g. 1080p/base = 51/46 ≈ 1.10870 for seedance-2.0).
	"doubao-seedance-2-0-260128": {
		PricePerSecond: 6.301369, // 46 CNY / 7.3
		ResolutionMultipliers: map[string]float64{
			"480p":  1.0,
			"720p":  1.0,
			"1080p": 1.108696, // 51/46
			"4k":    0.565217, // 26/46 (yes — 4K is cheaper on Seedance 2.0)
		},
	},
	"doubao-seedance-2-0-fast-260128": {
		PricePerSecond: 5.068493, // 37 CNY / 7.3
		ResolutionMultipliers: map[string]float64{
			"480p": 1.0,
			"720p": 1.0,
		},
	},
}

// GetVideoPricing returns the pricing profile for a video generation model.
// The second return value distinguishes "no entry configured" from "entry
// with zero price"; callers that want a hard fallback should check the bool.
func GetVideoPricing(name string) (VideoPricing, bool) {
	return videoPricingMap.Get(strings.TrimSpace(name))
}

// ResolutionMultiplier returns the multiplier for a specific resolution on
// a specific model, along with whether an explicit entry was found. Missing
// entries return 1.0 so callers can compose the value directly into a price
// formula without branching.
func ResolutionMultiplier(name, resolution string) (float64, bool) {
	pricing, ok := videoPricingMap.Get(strings.TrimSpace(name))
	if !ok {
		return 1.0, false
	}
	if pricing.ResolutionMultipliers == nil {
		return 1.0, false
	}
	mult, hit := pricing.ResolutionMultipliers[strings.ToLower(strings.TrimSpace(resolution))]
	if !hit || mult <= 0 {
		return 1.0, false
	}
	return mult, true
}

// GetVideoPricingCopy returns a defensive copy of the whole pricing map.
// Used by admin export flows.
func GetVideoPricingCopy() map[string]VideoPricing {
	return videoPricingMap.ReadAll()
}

// VideoPricing2JSONString serializes the current map for OptionMap.
func VideoPricing2JSONString() string {
	return videoPricingMap.MarshalJSONString()
}

// UpdateVideoPricingByJSONString ingests the JSON blob from the option table
// on startup and when an admin edits the field. Callback invalidates the
// exposed pricing cache used by the /pricing frontend page.
func UpdateVideoPricingByJSONString(jsonStr string) error {
	return types.LoadFromJsonStringWithCallback(videoPricingMap, jsonStr, InvalidateExposedDataCache)
}

// SeedDefaultVideoPricing merges the compile-time defaults into the runtime
// map without overwriting any admin-set values. Called once from
// InitRatioMap so an operator who never touches the JSON still gets sane
// defaults that mirror the pre-refactor hardcoded tables.
//
// The merge semantics match ratio_setting.AddAll conventions elsewhere in
// this package: existing keys win over defaults.
func SeedDefaultVideoPricing() {
	for name, defaults := range defaultVideoPricing {
		if _, exists := videoPricingMap.Get(name); exists {
			continue
		}
		videoPricingMap.Set(name, defaults)
	}
}

// videoPricingHardcodedFallback is called by the billing layer when neither
// the admin-configured map nor the seed data has an entry. Returns the
// pre-refactor hardcoded ratio (from VeoResolutionRatio for the Veo family,
// or 1.0 for anything else). We keep this as a package-private safety net so
// a mis-configured deployment can't accidentally bill $0 for a video.
func videoPricingHardcodedFallback(modelName, resolution string) float64 {
	res := strings.ToLower(strings.TrimSpace(resolution))
	if res != "4k" {
		return 1.0
	}
	// Values kept in sync with relay/channel/task/gemini/billing.go.
	name := strings.ToLower(strings.TrimSpace(modelName))
	if strings.Contains(name, "3.1-fast-generate") {
		return 2.333333
	}
	if strings.Contains(name, "3.1-generate") || strings.Contains(name, "3.1") {
		return 1.5
	}
	return 1.0
}

// ResolutionMultiplierWithFallback is the billing-layer entry point:
// checks the admin map first, then falls back to hardcoded logic. Always
// returns a positive multiplier so callers can multiply unconditionally.
func ResolutionMultiplierWithFallback(name, resolution string) float64 {
	if mult, ok := ResolutionMultiplier(name, resolution); ok {
		return mult
	}
	return videoPricingHardcodedFallback(name, resolution)
}
