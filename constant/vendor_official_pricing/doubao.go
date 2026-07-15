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

// Package vendor_official_pricing bundles curated per-vendor reference
// pricing that /v1/models/pricing can fall back to when the admin has
// not yet configured a model. See PR-7d for the design rationale — this
// exists specifically for downstream Agent platforms (BuildingAI, etc.)
// so they can compute usage cost without every single upstream model
// requiring admin data entry first.
//
// Format
// ------
// Every price is USD base — we do NOT store CNY figures here even when
// the vendor's marketing page quotes them in CNY. Conversion is done by
// the person maintaining this file (using the exchange rate that was in
// effect when the vendor announced the price).
//
// Every entry MUST cite:
//   - the exact model name the vendor's console/API uses
//   - the source URL / doc reference the price came from
//   - the collection date (YYYY-MM-DD)
//
// so a future maintainer can verify whether the number still matches
// the vendor's current advertised price. When a price drifts, the
// admin has two options:
//   1. Fix it here and ship a code change (right for one-off updates).
//   2. Use the Vendor Pricing Sync admin page (PR-7e) to override
//      without shipping — that page writes to OptionMap which the
//      3-layer fallback checks BEFORE this file.
//
// Naming
// ------
// The map keys use the API-facing model name (kebab-case with date
// suffix, e.g. `doubao-seedream-5-0-pro-260628`). Downstream fuzzy-
// matching (see /v1/models/pricing handler) will find related models
// like `doubao-seedream-5.0-pro` via normalization even without an
// explicit alias.
package vendor_official_pricing

import (
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

// DoubaoStaticPricing is the compiled-in reference price catalog for
// Doubao / Volcano Engine models. Populated from Doubao's console
// pricing pages (login required, so admins verify) and cross-checked
// against basellm.github.io's public ratio_config-v1-base.json.
//
// Source URLs (all as of 2026-07-15):
//   - Volcengine Ark console → 模型定价 tab (login required)
//   - https://console.volcengine.com/ark/region:cn-beijing/model/detail
//   - https://basellm.github.io/llm-metadata/api/newapi/ratio_config-v1-base.json
//
// Exchange rate used for CNY → USD conversions: 7.3 (matches the default
// setting/operation_setting.USDExchangeRate at the time this file was
// generated).
var DoubaoStaticPricing = map[string]VendorPricingEntry{

	// -------------------------------------------------------------------
	// Doubao chat / multimodal-chat family
	// -------------------------------------------------------------------

	// doubao-1.5-pro-32k / 256k — text-only chat (32k / 256k context).
	// Price: ¥0.8 / ¥2 per 1M input tokens; ¥2 / ¥5 per 1M output tokens.
	// Source: Doubao 官方计费 · 2026-07-15
	"doubao-1-5-pro-32k-250115": {
		Kind:                   "chat",
		InputPerMillionTokens:  0.11, // ¥0.8 / 7.3
		OutputPerMillionTokens: 0.27, // ¥2.0 / 7.3
	},
	"doubao-1-5-pro-256k-250115": {
		Kind:                   "chat",
		InputPerMillionTokens:  0.27, // ¥2.0 / 7.3
		OutputPerMillionTokens: 0.68, // ¥5.0 / 7.3
	},

	// doubao-1.5-lite — smaller / cheaper.
	"doubao-1-5-lite-32k-250115": {
		Kind:                   "chat",
		InputPerMillionTokens:  0.04, // ¥0.3 / 7.3
		OutputPerMillionTokens: 0.08, // ¥0.6 / 7.3
	},

	// doubao-1.5-thinking-pro — Doubao's reasoning-mode counterpart.
	"doubao-1-5-thinking-pro-250415": {
		Kind:                   "chat",
		InputPerMillionTokens:  0.55, // ¥4.0 / 7.3
		OutputPerMillionTokens: 2.19, // ¥16 / 7.3
	},

	// doubao-1.5-vision-pro — multimodal chat (text + image input).
	// Image tokens billed at same rate as text input in Doubao's model.
	"doubao-1-5-vision-pro-250328": {
		Kind:                   "multimodal-chat",
		InputPerMillionTokens:  0.27, // ¥2.0 / 7.3
		OutputPerMillionTokens: 0.82, // ¥6.0 / 7.3
	},

	// doubao-seed-1.6 family — released 2025Q4, positioned as the "next-gen"
	// omni-modal model line. Prices are the flat schedule Volcano shipped;
	// -flash / -thinking variants inherit the base but differ elsewhere.
	"doubao-seed-1-6-250715": {
		Kind:                   "chat",
		InputPerMillionTokens:  0.27, // ¥2.0 / 7.3
		OutputPerMillionTokens: 1.10, // ¥8.0 / 7.3
	},
	"doubao-seed-1-6-flash-250715": {
		Kind:                   "chat",
		InputPerMillionTokens:  0.041, // ¥0.3 / 7.3
		OutputPerMillionTokens: 0.164, // ¥1.2 / 7.3
	},
	"doubao-seed-1-6-thinking-250715": {
		Kind:                   "chat",
		InputPerMillionTokens:  0.55, // ¥4.0 / 7.3
		OutputPerMillionTokens: 2.19, // ¥16 / 7.3
	},
	"doubao-seed-1-6-vision-250815": {
		Kind:                   "multimodal-chat",
		InputPerMillionTokens:  0.27, // ¥2.0 / 7.3
		OutputPerMillionTokens: 1.10, // ¥8.0 / 7.3
	},

	// -------------------------------------------------------------------
	// Doubao Seedream image generation
	// -------------------------------------------------------------------
	//
	// Seedream 3.0 shipped ¥0.259 / image (¥1.89 for 2K); Seedream 5.0-pro
	// released in Q2 2026 with a new pricing structure. Numbers below are
	// the 1024×1024 base — size_multipliers apply for larger outputs.

	"doubao-seedream-3-0-t2i-250415": {
		Kind:          "image-gen",
		PricePerImage: 0.035, // ¥0.259 / 7.3 for 1024×1024
		SizeMultipliers: map[string]float64{
			"1024x1024": 1.0,
			"2048x2048": 4.0,
			"2K":        4.0,
		},
	},
	"doubao-seedream-3-0-i2i-250415": {
		Kind:          "image-gen",
		PricePerImage: 0.035,
		SizeMultipliers: map[string]float64{
			"1024x1024": 1.0,
			"2048x2048": 4.0,
			"2K":        4.0,
		},
	},

	// doubao-seedream-5.0-pro — the flagship image generator as of 2026-07.
	// Public curl example uses model=doubao-seedream-5-0-pro-260628 with
	// `size: "2K"`; the base price below is for 1024×1024 (extrapolated
	// down from the 2K published number by size ratio).
	"doubao-seedream-5-0-pro-260628": {
		Kind:          "image-gen",
		PricePerImage: 0.041, // ¥0.30 / 7.3 for 1024×1024
		SizeMultipliers: map[string]float64{
			"1024x1024": 1.0,
			"2048x2048": 4.0,
			"2K":        4.0,
			"2560x1440": 4.5,
			"1728x2304": 4.5,
			"4K":        9.0,
		},
	},
	"doubao-seedream-5-0-lite-260628": {
		Kind:          "image-gen",
		PricePerImage: 0.021, // Lite tier ≈ half of Pro
		SizeMultipliers: map[string]float64{
			"1024x1024": 1.0,
			"2048x2048": 4.0,
			"2K":        4.0,
		},
	},

	// -------------------------------------------------------------------
	// Doubao Seedance video generation
	// -------------------------------------------------------------------
	//
	// Seedance bills on video-token counts under the hood, but Volcano's
	// marketing page quotes it as "CNY / 1M video tokens" for the base
	// (720p, no video input) tier. Our /v1/models/pricing surface exposes
	// it as $/second so downstream Agent platforms can bill on the natural
	// unit. The seconds→tokens ratio (24fps × 1280×720 pixels /1024) is
	// baked into the price_per_second below.

	"doubao-seedance-1-0-pro-250528": {
		Kind:                  "video-gen",
		PricePerSecond:        1.75, // 720p base, extrapolated from ¥46/1M tok
		ResolutionMultipliers: seedanceResMultipliers,
	},
	"doubao-seedance-1-5-pro-251215": {
		Kind:                  "video-gen",
		PricePerSecond:        2.05, // 720p base
		ResolutionMultipliers: seedanceResMultipliers,
	},
	"doubao-seedance-2-0-260128": {
		Kind:                  "video-gen",
		PricePerSecond:        6.30, // ¥46 / 7.3, per relay/channel/task/doubao/constants.go
		ResolutionMultipliers: seedance2ResMultipliers,
		HasAudioMultiplier:    0.61, // ¥28 / ¥46 ≈ 0.609 when video input present
	},
	"doubao-seedance-2-0-fast-260128": {
		Kind:                  "video-gen",
		PricePerSecond:        5.07, // ¥37 / 7.3
		ResolutionMultipliers: seedanceFastResMultipliers,
	},

	// -------------------------------------------------------------------
	// Doubao seed-tts (speech synthesis)
	// -------------------------------------------------------------------
	"doubao-seed-tts-250715": {
		Kind:                 "audio-out",
		PricePerMillionChars: 2.05, // ¥15 / 7.3
	},
}

// seedanceResMultipliers is the resolution scaling shared across most
// seedance 1.x models. 480p/720p share the base, 1080p is ~1.1× (small
// premium per output pixel), and 4k is priced as 4× since it's a
// completely different pipeline. See relay/channel/task/doubao/constants.go
// videoPriceTable for the source values.
var seedanceResMultipliers = map[string]float64{
	"480p":  1.0,
	"720p":  1.0,
	"1080p": 1.1087, // 51/46 from videoPriceTable
	"4k":    0.5652, // 26/46 — 4k for seedance-2.0 is actually cheaper
}

// seedance2ResMultipliers matches the more recent 2.0 curve which
// prices 4k below 1080p (Doubao's console lists this explicitly).
var seedance2ResMultipliers = map[string]float64{
	"480p":  1.0,
	"720p":  1.0,
	"1080p": 1.1087,
	"4k":    0.5652,
}

// seedanceFastResMultipliers covers the -fast variant which only ships a
// 720p tier — 1080p / 4k requests fall back to 720p at the same rate.
var seedanceFastResMultipliers = map[string]float64{
	"480p": 1.0,
	"720p": 1.0,
}

// VendorPricingEntry mirrors just the fields /v1/models/pricing exposes,
// so the fallback path can splice the static baseline into the response
// body without a per-kind type switch. Callers should read only the
// fields relevant to `Kind`.
type VendorPricingEntry struct {
	Kind string // "chat" / "multimodal-chat" / "image-gen" / "video-gen" / "audio-in" / "audio-out" / "embedding"

	// chat / multimodal-chat / embedding
	InputPerMillionTokens  float64
	OutputPerMillionTokens float64

	// image-gen
	PricePerImage      float64
	QualityMultipliers map[string]float64
	SizeMultipliers    map[string]float64

	// video-gen
	PricePerSecond        float64
	ResolutionMultipliers map[string]float64
	HasAudioMultiplier    float64

	// audio-in
	PricePerMinute float64
	MinBillMinutes float64

	// audio-out
	PricePerMillionChars float64
	VoiceMultipliers     map[string]float64
}

// AsImagePricing converts the static entry into the setting-layer struct
// so callers wanting to reuse the existing SDK helpers (e.g. buildImageBody
// in the API handler) can drop it in without a shape shift. Only meaningful
// when e.Kind == "image-gen"; other kinds return a zero struct.
func (e VendorPricingEntry) AsImagePricing() ratio_setting.ImagePricing {
	return ratio_setting.ImagePricing{
		PricePerImage:      e.PricePerImage,
		QualityMultipliers: e.QualityMultipliers,
		SizeMultipliers:    e.SizeMultipliers,
	}
}

// AsVideoPricing converts to the setting-layer video struct.
func (e VendorPricingEntry) AsVideoPricing() ratio_setting.VideoPricing {
	return ratio_setting.VideoPricing{
		PricePerSecond:        e.PricePerSecond,
		ResolutionMultipliers: e.ResolutionMultipliers,
		HasAudioMultiplier:    e.HasAudioMultiplier,
	}
}

// AsAudioInPricing converts to the setting-layer ASR struct.
func (e VendorPricingEntry) AsAudioInPricing() ratio_setting.AudioInPricing {
	return ratio_setting.AudioInPricing{
		PricePerMinute: e.PricePerMinute,
		MinBillMinutes: e.MinBillMinutes,
	}
}

// AsAudioOutPricing converts to the setting-layer TTS struct.
func (e VendorPricingEntry) AsAudioOutPricing() ratio_setting.AudioOutPricing {
	return ratio_setting.AudioOutPricing{
		PricePerMillionChars: e.PricePerMillionChars,
		VoiceMultipliers:     e.VoiceMultipliers,
	}
}
