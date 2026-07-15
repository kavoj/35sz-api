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
package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// GET /v1/models/pricing
//
// Downstream platforms that resell 35sz-api capabilities (e.g. BuildingAI
// Agent Platform) need a single authoritative endpoint to pull per-model
// pricing in NATIVE UNITS — $/1M tokens for chat, $/image for image-gen,
// $/second for video-gen, $/minute for ASR, $/1M chars for TTS. The
// existing GET /api/pricing returns ratio numbers geared toward the
// in-app pricing table UI and mixes multiple pricing paradigms together;
// it's fine for humans reading the pricing table but not for machine
// integration.
//
// This endpoint pulls the structured VideoPricing / ImagePricing /
// AudioInPricing / AudioOutPricing tables (introduced by PR-1) plus the
// legacy ModelRatio for chat models, and stamps each entry with:
//
//   - model_type — text / image / video / audio / embedding (surface for
//     downstream filter/routing).
//   - pricing_kind — one of the 7 PricingKind constants (chat / video-gen
//     / image-gen / audio-in / audio-out / embedding / multimodal-chat);
//     the exact vocabulary the downstream Agent needs to decide which
//     onSuccess callback shape to use.
//   - pricing_type — "token" | "per_image" | "per_second" | "per_minute"
//     | "per_million_chars", so a naive downstream that doesn't grok
//     pricing_kind can still know the unit.
//   - pricing — a per-kind sub-struct with the actual numbers (see
//     doc comments below).
//
// Auth: requires admin. This is a machine-to-machine endpoint that leaks
// upstream cost data; it MUST NOT be exposed to regular users.
//
// Cache: caller-side. This endpoint reads the live in-memory maps
// synchronously; if downstream traffic becomes hot we can add a
// Cache-Control header later.
// ============================================================================

// ModelPricingResponse is the top-level envelope. `data` is a map from
// model name to its pricing struct — a map (not an array) so downstream
// callers can look up by O(1). `updated_at` marks the moment we snapshot
// the maps for reproducibility, and `pricing_version` is a hash for
// change detection (downstream can skip re-caching if hash unchanged).
type ModelPricingResponse struct {
	Data           map[string]ModelPricingEntry `json:"data"`
	UpdatedAt      int64                        `json:"updated_at"`
	PricingVersion string                       `json:"pricing_version,omitempty"`
}

// ModelPricingEntry is the per-model unit pricing. Every field is
// populated only when relevant to the model's kind, so downstream can
// tell "field missing" from "field zero" easily. The Pricing sub-struct
// carries the actual numbers — chat has ratios, image has $/image, etc.
type ModelPricingEntry struct {
	ModelType   string  `json:"model_type"`
	PricingKind string  `json:"pricing_kind"`
	PricingType string  `json:"pricing_type"`
	Pricing     Pricing `json:"pricing"`
}

// Pricing is a discriminated union — the fields relevant to the entry's
// PricingKind are populated; the rest are zero. Keeping this as a flat
// struct (rather than four separate types with json.RawMessage
// polymorphism) trades a bit of clarity for a stable JSON shape that
// downstream can parse without a type-switch.
type Pricing struct {
	// --- chat / multimodal-chat / embedding (token-based) ---
	ModelRatio           float64  `json:"model_ratio,omitempty"`
	CompletionRatio      float64  `json:"completion_ratio,omitempty"`
	CacheRatio           *float64 `json:"cache_ratio,omitempty"`
	ImageRatio           *float64 `json:"image_ratio,omitempty"`
	AudioRatio           *float64 `json:"audio_ratio,omitempty"`
	AudioCompletionRatio *float64 `json:"audio_completion_ratio,omitempty"`
	// InputPerMillionTokens = ModelRatio × 2 USD (denormalized for
	// downstream convenience — one line in Agent's PricingService instead
	// of a magic "× 2" repeated everywhere).
	InputPerMillionTokens  float64 `json:"input_per_million_tokens,omitempty"`
	OutputPerMillionTokens float64 `json:"output_per_million_tokens,omitempty"`

	// --- image-gen ---
	PricePerImage      float64            `json:"price_per_image,omitempty"`
	QualityMultipliers map[string]float64 `json:"quality_multipliers,omitempty"`
	SizeMultipliers    map[string]float64 `json:"size_multipliers,omitempty"`

	// --- video-gen ---
	PricePerSecond        float64            `json:"price_per_second,omitempty"`
	ResolutionMultipliers map[string]float64 `json:"resolution_multipliers,omitempty"`
	HasAudioMultiplier    float64            `json:"has_audio_multiplier,omitempty"`

	// --- audio-in ---
	PricePerMinute float64 `json:"price_per_minute,omitempty"`
	MinBillMinutes float64 `json:"min_bill_minutes,omitempty"`

	// --- audio-out ---
	PricePerMillionChars float64            `json:"price_per_million_chars,omitempty"`
	VoiceMultipliers     map[string]float64 `json:"voice_multipliers,omitempty"`

	// --- per-request fallback (legacy ModelPrice map) ---
	// Non-zero means the model is billed per-request with this flat USD
	// price, regardless of its pricing_kind. Downstream should honor
	// this over the finer-grained fields when both are populated.
	FlatPricePerRequest float64 `json:"flat_price_per_request,omitempty"`

	// --- data-quality flag ---
	// PricingIncomplete = true means the model has pricing_kind set to a
	// structured kind (image-gen / video-gen / audio-in / audio-out) but
	// the corresponding *Pricing option row is empty or missing. This can
	// happen when:
	//   - The admin classified the model as video-gen in the metadata UI
	//     but hasn't opened the pricing drawer to enter numbers yet.
	//   - A seed data set covers only a subset of the model catalogue.
	//   - Legacy rows migrated from ModelPrice but never got a structured
	//     entry.
	// Downstream integrators MUST fall back to their local pricing table
	// (or refuse to bill) when this flag is true — the flat / ratio
	// fields cannot be trusted to represent the model's real billing
	// shape in that state.
	PricingIncomplete bool `json:"pricing_incomplete,omitempty"`
}

// pricingTypeForKind maps a PricingKind to a stable "pricing_type" string
// that lets a downstream integrator route without knowing our enum. The
// mapping intentionally hides the multimodal-chat / chat distinction —
// downstream cares about billing unit, not conversation type.
func pricingTypeForKind(kind string) string {
	switch kind {
	case constant.PricingKindImageGen:
		return "per_image"
	case constant.PricingKindVideoGen:
		return "per_second"
	case constant.PricingKindAudioIn:
		return "per_minute"
	case constant.PricingKindAudioOut:
		return "per_million_chars"
	case constant.PricingKindEmbedding,
		constant.PricingKindMultimodalChat,
		constant.PricingKindChat:
		return "token"
	}
	return "token"
}

// modelTypeForKind maps PricingKind to the coarse-grained model_type used
// by the /models/metadata drawer filter chips. When a model has an
// explicit model_type in the DB we honor that; this is a fallback for
// legacy rows that predate PR-1.
func modelTypeForKind(kind string) string {
	switch kind {
	case constant.PricingKindImageGen:
		return model.ModelTypeImage
	case constant.PricingKindVideoGen:
		return model.ModelTypeVideo
	case constant.PricingKindAudioIn, constant.PricingKindAudioOut:
		return model.ModelTypeAudio
	case constant.PricingKindEmbedding:
		return model.ModelTypeEmbedding
	}
	return model.ModelTypeText
}

// GetModelsPricing is the HTTP handler for GET /v1/models/pricing.
// It's a pure aggregator — no writes, no side effects, no leaking of
// upstream provider credentials.
func GetModelsPricing(c *gin.Context) {
	// GetPricing() is cached with a 1-minute stale window (see
	// model.GetPricing). We piggyback on that so /v1/models/pricing is
	// cheap even under 5-minute-polling downstream calls.
	pricings := model.GetPricing()

	// Load structured pricing maps up front — one copy each, then O(1)
	// lookups in the loop below.
	imageMap := ratio_setting.GetImagePricingCopy()
	videoMap := ratio_setting.GetVideoPricingCopy()
	audioInMap := ratio_setting.GetAudioInPricingCopy()
	audioOutMap := ratio_setting.GetAudioOutPricingCopy()

	// Pull model_type + pricing_kind from the models table. Two rows
	// with the same name (edge case) resolve to whichever is last;
	// pricing.go's metaMap has the same policy.
	metaKindMap := make(map[string]string)
	metaTypeMap := make(map[string]string)
	models, err := model.GetAllModels(0, 100000)
	if err == nil {
		for _, m := range models {
			metaKindMap[m.ModelName] = constant.NormalizePricingKind(m.PricingKind)
			metaTypeMap[m.ModelName] = m.ModelType
		}
	}

	data := make(map[string]ModelPricingEntry, len(pricings))
	for _, p := range pricings {
		kind := metaKindMap[p.ModelName]
		if kind == "" {
			kind = constant.PricingKindChat
		}
		modelType := metaTypeMap[p.ModelName]
		if modelType == "" {
			modelType = modelTypeForKind(kind)
		}

		entry := ModelPricingEntry{
			ModelType:   modelType,
			PricingKind: kind,
			PricingType: pricingTypeForKind(kind),
			Pricing:     buildPricingBody(p, kind, imageMap, videoMap, audioInMap, audioOutMap),
		}
		data[p.ModelName] = entry
	}

	c.JSON(200, gin.H{
		"success":         true,
		"data":            data,
		"updated_at":      common.GetTimestamp(),
		"pricing_version": firstPricingVersion(pricings),
	})
}

// firstPricingVersion returns the pricing_version stamp shared by every
// entry (GetPricing computes it once per refresh). Guards against the
// empty-list edge case that trips up `pricings[0].PricingVersion`.
func firstPricingVersion(pricings []model.Pricing) string {
	if len(pricings) == 0 {
		return ""
	}
	return pricings[0].PricingVersion
}

// buildPricingBody populates the fields relevant to `kind`. Fields
// irrelevant to the kind stay at their Go zero value, which JSON
// serialization omits (thanks to the omitempty tags on Pricing).
func buildPricingBody(
	p model.Pricing,
	kind string,
	imageMap map[string]ratio_setting.ImagePricing,
	videoMap map[string]ratio_setting.VideoPricing,
	audioInMap map[string]ratio_setting.AudioInPricing,
	audioOutMap map[string]ratio_setting.AudioOutPricing,
) Pricing {
	body := Pricing{
		// Legacy flat ModelPrice — carried on every entry so a downstream
		// can honor per-request pricing regardless of kind. Zero means
		// "no flat override, use kind-specific fields".
		FlatPricePerRequest: p.ModelPrice,
	}

	// Effective USD/1M ratios — always populated for chat-shaped kinds;
	// for image/video/audio-gen kinds they'd be misleading (the model
	// isn't billed per token), so we leave them zero.
	switch kind {
	case constant.PricingKindChat,
		constant.PricingKindMultimodalChat,
		constant.PricingKindEmbedding:
		body.ModelRatio = p.ModelRatio
		body.CompletionRatio = p.CompletionRatio
		body.CacheRatio = p.CacheRatio
		body.ImageRatio = p.ImageRatio
		body.AudioRatio = p.AudioRatio
		body.AudioCompletionRatio = p.AudioCompletionRatio
		// 1 ModelRatio unit = $2 / 1M tokens. Convention baked into
		// setting/ratio_setting/model_ratio.go. Denormalized here so
		// downstream doesn't need to know the "× 2" magic constant.
		body.InputPerMillionTokens = p.ModelRatio * 2
		if p.CompletionRatio > 0 {
			body.OutputPerMillionTokens = p.ModelRatio * 2 * p.CompletionRatio
		} else {
			body.OutputPerMillionTokens = p.ModelRatio * 2
		}

	case constant.PricingKindImageGen:
		if ip, ok := imageMap[p.ModelName]; ok && ip.PricePerImage > 0 {
			body.PricePerImage = ip.PricePerImage
			body.QualityMultipliers = ip.QualityMultipliers
			body.SizeMultipliers = ip.SizeMultipliers
		} else if body.FlatPricePerRequest <= 0 {
			body.PricingIncomplete = true
		}

	case constant.PricingKindVideoGen:
		if vp, ok := videoMap[p.ModelName]; ok && vp.PricePerSecond > 0 {
			body.PricePerSecond = vp.PricePerSecond
			body.ResolutionMultipliers = vp.ResolutionMultipliers
			body.HasAudioMultiplier = vp.HasAudioMultiplier
		} else if body.FlatPricePerRequest <= 0 {
			body.PricingIncomplete = true
		}

	case constant.PricingKindAudioIn:
		if ap, ok := audioInMap[p.ModelName]; ok && ap.PricePerMinute > 0 {
			body.PricePerMinute = ap.PricePerMinute
			body.MinBillMinutes = ap.MinBillMinutes
		} else if body.FlatPricePerRequest <= 0 {
			body.PricingIncomplete = true
		}

	case constant.PricingKindAudioOut:
		if ap, ok := audioOutMap[p.ModelName]; ok && ap.PricePerMillionChars > 0 {
			body.PricePerMillionChars = ap.PricePerMillionChars
			body.VoiceMultipliers = ap.VoiceMultipliers
		} else if body.FlatPricePerRequest <= 0 {
			body.PricingIncomplete = true
		}
	}

	return body
}

// Reserved for future: rate-limit and record downstream polling for
// analytics. For now the endpoint is behind admin auth, and the number
// of admin API keys polling a given deployment is small, so we skip.
var _ = operation_setting.USDExchangeRate
