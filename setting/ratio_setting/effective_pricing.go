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

import "strings"

// EffectivePerRequestPrice returns the "per-request" USD price for a model
// by consulting the structured pricing tables introduced in PR-1:
//
//   image-gen  → ImagePricing.PricePerImage
//   video-gen  → VideoPricing.PricePerSecond
//   audio-in   → AudioInPricing.PricePerMinute
//   audio-out  → AudioOutPricing.PricePerMillionChars
//
// This is called by `GetModelPrice()` as the FIRST lookup — before the legacy
// `ModelPrice` map — so admins configuring native units through the drawer's
// StructuredPricingEditor see the value take effect at billing time.
//
// Semantics of "per-request":
//   - This function returns the BASE price (single unit). The billing layer is
//     responsible for multiplying by the actual request quantity
//     (seconds / images / minutes / chars) and applying resolution / quality
//     multipliers from the same structured entry.
//   - For video-gen the layer above should compose:
//       total = pricePerSecond × durationSeconds × resolutionMult × audioMult
//   - For image-gen: total = pricePerImage × n × qualityMult × sizeMult
//   - For audio-in:  total = max(minMinutes, actualMinutes) × pricePerMinute
//   - For audio-out: total = charCount / 1_000_000 × pricePerMillionChars × voiceMult
//
// When no structured entry exists, returns (0, false) and the caller falls
// through to the legacy `ModelPrice` map. This guarantees zero-behavior-change
// for chat / multimodal-chat / embedding models that never saw the new
// tables, and for structured-kind models an admin hasn't explicitly priced.
func EffectivePerRequestPrice(name string) (float64, bool) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return 0, false
	}

	// The order of the four probes doesn't matter for correctness — the
	// same model name should only appear in exactly one map at a time
	// (the drawer's save flow enforces this: switching kinds deletes the
	// row from the other three maps). But we probe in the order most
	// likely to hit for the model catalog we ship, to minimize the number
	// of RWMap.Get() calls on hot paths.

	if p, ok := GetVideoPricing(trimmed); ok && p.PricePerSecond > 0 {
		return p.PricePerSecond, true
	}
	if p, ok := GetImagePricing(trimmed); ok && p.PricePerImage > 0 {
		return p.PricePerImage, true
	}
	if p, ok := GetAudioInPricing(trimmed); ok && p.PricePerMinute > 0 {
		return p.PricePerMinute, true
	}
	if p, ok := GetAudioOutPricing(trimmed); ok && p.PricePerMillionChars > 0 {
		return p.PricePerMillionChars, true
	}
	return 0, false
}
