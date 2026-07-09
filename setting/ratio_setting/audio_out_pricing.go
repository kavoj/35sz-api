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

// AudioOutPricing captures per-character pricing for text-to-speech models.
//
// Note: kept as "per 1M characters" (not per token) because that matches the
// public pricing units of OpenAI TTS, ElevenLabs, and Doubao seed-tts. Token
// counting for audio output is inconsistent across providers, but character
// counts are stable.
//
// Billing formula
// ---------------
//   totalUSD = charCount / 1_000_000
//            × PricePerMillionChars
//            × VoiceMultipliers[voice]  (default 1)
//
// Units
// -----
// - PricePerMillionChars is USD per 1,000,000 characters of input text.
// - VoiceMultipliers keys are provider-specific voice tokens
//   ("nova"/"echo"/"clone"). Missing key ⇒ 1.
type AudioOutPricing struct {
	PricePerMillionChars float64            `json:"price_per_million_chars"`
	VoiceMultipliers     map[string]float64 `json:"voice_multipliers,omitempty"`
}

var audioOutPricingMap = types.NewRWMap[string, AudioOutPricing]()

// defaultAudioOutPricing — OpenAI TTS + common providers.
var defaultAudioOutPricing = map[string]AudioOutPricing{
	"tts-1": {
		PricePerMillionChars: 15.0,
	},
	"tts-1-hd": {
		PricePerMillionChars: 30.0,
	},
	"gpt-4o-mini-tts": {
		PricePerMillionChars: 10.0,
	},
}

func GetAudioOutPricing(name string) (AudioOutPricing, bool) {
	return audioOutPricingMap.Get(strings.TrimSpace(name))
}

func GetAudioOutPricingCopy() map[string]AudioOutPricing {
	return audioOutPricingMap.ReadAll()
}

func AudioOutPricing2JSONString() string {
	return audioOutPricingMap.MarshalJSONString()
}

func UpdateAudioOutPricingByJSONString(jsonStr string) error {
	return types.LoadFromJsonStringWithCallback(audioOutPricingMap, jsonStr, InvalidateExposedDataCache)
}

func SeedDefaultAudioOutPricing() {
	for name, defaults := range defaultAudioOutPricing {
		if _, exists := audioOutPricingMap.Get(name); exists {
			continue
		}
		audioOutPricingMap.Set(name, defaults)
	}
}
