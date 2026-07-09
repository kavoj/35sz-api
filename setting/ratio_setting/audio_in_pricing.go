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

// AudioInPricing captures per-minute pricing for automatic speech recognition
// (Whisper family) and other transcription-style models. Distinct from
// AudioRatio (which handles chat-embedded audio input tokens on multimodal
// models like gpt-4o-audio) because ASR bills on audio duration, not tokens.
//
// Billing formula
// ---------------
//   totalUSD = max(durationMinutes, MinBillMinutes) × PricePerMinute
//
// Units
// -----
// - PricePerMinute is USD per minute (base USD).
// - MinBillMinutes lets providers enforce a minimum charge (e.g. 15 seconds =
//   0.25 min). 0 disables the floor.
type AudioInPricing struct {
	PricePerMinute float64 `json:"price_per_minute"`
	MinBillMinutes float64 `json:"min_bill_minutes,omitempty"`
}

var audioInPricingMap = types.NewRWMap[string, AudioInPricing]()

// defaultAudioInPricing — Whisper family + common ASR models.
var defaultAudioInPricing = map[string]AudioInPricing{
	"whisper-1": {
		PricePerMinute: 0.006,
	},
	"gpt-4o-transcribe": {
		PricePerMinute: 0.006,
	},
	"gpt-4o-mini-transcribe": {
		PricePerMinute: 0.003,
	},
}

func GetAudioInPricing(name string) (AudioInPricing, bool) {
	return audioInPricingMap.Get(strings.TrimSpace(name))
}

func GetAudioInPricingCopy() map[string]AudioInPricing {
	return audioInPricingMap.ReadAll()
}

func AudioInPricing2JSONString() string {
	return audioInPricingMap.MarshalJSONString()
}

func UpdateAudioInPricingByJSONString(jsonStr string) error {
	return types.LoadFromJsonStringWithCallback(audioInPricingMap, jsonStr, InvalidateExposedDataCache)
}

func SeedDefaultAudioInPricing() {
	for name, defaults := range defaultAudioInPricing {
		if _, exists := audioInPricingMap.Get(name); exists {
			continue
		}
		audioInPricingMap.Set(name, defaults)
	}
}
