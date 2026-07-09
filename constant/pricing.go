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

package constant

// PricingKind identifies the billing shape for a model row. It's persisted on
// `model_meta.pricing_kind` and drives:
//
//   - which pricing setting map is consulted at billing time
//     (ModelRatio / ImagePricing / VideoPricing / AudioInPricing /
//     AudioOutPricing)
//   - which units the numbers carry ($/1M tokens vs $/second vs $/image vs
//     $/minute vs $/1M chars)
//   - which fields the /models/metadata drawer renders
//
// The zero-value default is "chat" — every legacy model row that predates
// this column keeps its existing per-token billing behavior when GORM applies
// the AutoMigrate default.
const (
	// PricingKindChat: pure text chat models. Billed per token via
	// ModelRatio + CompletionRatio + CacheRatio (existing infrastructure).
	// Also the fallback when a model_meta row lacks a valid pricing_kind.
	PricingKindChat = "chat"

	// PricingKindMultimodalChat: chat models that also charge for image or
	// audio *input*. Billed per token as above, plus ImageRatio for image
	// tokens and AudioRatio/AudioCompletionRatio for audio tokens.
	PricingKindMultimodalChat = "multimodal-chat"

	// PricingKindImageGen: image generation models. Billed per generated
	// image, with optional quality/size multipliers.
	// Storage: setting/ratio_setting.ImagePricing map, keyed by model name.
	PricingKindImageGen = "image-gen"

	// PricingKindVideoGen: video generation models. Billed per second of
	// output, multiplied by a resolution multiplier and optionally an
	// audio-on multiplier (Veo). Replaces the hardcoded
	// videoInputRatioMap / VeoResolutionRatio tables with a per-model
	// admin-editable setting.
	// Storage: setting/ratio_setting.VideoPricing map.
	PricingKindVideoGen = "video-gen"

	// PricingKindAudioIn: automatic speech recognition (whisper family).
	// Billed per minute of input audio.
	// Storage: setting/ratio_setting.AudioInPricing map.
	PricingKindAudioIn = "audio-in"

	// PricingKindAudioOut: text-to-speech / audio generation. Billed per
	// million characters of input text, with optional voice multipliers.
	// Storage: setting/ratio_setting.AudioOutPricing map.
	PricingKindAudioOut = "audio-out"

	// PricingKindEmbedding: embedding / reranking / sentence-similarity
	// models. Billed per token like chat, but the completion / cache
	// dimensions are not applicable — the UI hides them.
	PricingKindEmbedding = "embedding"
)

// allPricingKinds — kept in one place so the validator, tests, and any future
// admin UI enumeration stay in sync when a kind is added.
var allPricingKinds = map[string]struct{}{
	PricingKindChat:           {},
	PricingKindMultimodalChat: {},
	PricingKindImageGen:       {},
	PricingKindVideoGen:       {},
	PricingKindAudioIn:        {},
	PricingKindAudioOut:       {},
	PricingKindEmbedding:      {},
}

// IsValidPricingKind reports whether the supplied string is one of the
// defined PricingKind* constants. Empty strings are NOT valid — callers that
// want to accept "unset" should treat it as PricingKindChat explicitly.
func IsValidPricingKind(kind string) bool {
	_, ok := allPricingKinds[kind]
	return ok
}

// NormalizePricingKind coerces an incoming string into a valid PricingKind,
// defaulting to PricingKindChat when the value is empty or unrecognized. Use
// this at the request-boundary (controller layer) so downstream code can
// assume the value is safe to switch on.
func NormalizePricingKind(kind string) string {
	if IsValidPricingKind(kind) {
		return kind
	}
	return PricingKindChat
}
