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

// ImagePricing captures per-image pricing for a text-to-image model.
//
// Billing formula
// ---------------
//   totalUSD = n
//            × PricePerImage
//            × QualityMultipliers[quality]  (default 1)
//            × SizeMultipliers[size]        (default 1)
//
// Units
// -----
// - PricePerImage is USD per generated image, base USD (admin UI converts
//   from CNY/CUSTOM before save).
// - QualityMultipliers keys are provider-specific quality tokens
//   ("low"/"medium"/"high"/"hd"/"standard"). Missing key ⇒ 1.
// - SizeMultipliers keys are size strings ("1024x1024", "1024x1792",
//   "1792x1024"). Missing key ⇒ 1.
type ImagePricing struct {
	PricePerImage      float64            `json:"price_per_image"`
	QualityMultipliers map[string]float64 `json:"quality_multipliers,omitempty"`
	SizeMultipliers    map[string]float64 `json:"size_multipliers,omitempty"`
}

var imagePricingMap = types.NewRWMap[string, ImagePricing]()

// defaultImagePricing seeds well-known image-generation models with prices
// that match Stripe-tier public pricing. Base USD values, not CNY.
var defaultImagePricing = map[string]ImagePricing{
	"dall-e-3": {
		PricePerImage: 0.04,
		QualityMultipliers: map[string]float64{
			"standard": 1.0,
			"hd":       2.0,
		},
		SizeMultipliers: map[string]float64{
			"1024x1024": 1.0,
			"1024x1792": 2.0,
			"1792x1024": 2.0,
		},
	},
	"gpt-image-1": {
		PricePerImage: 0.02,
		QualityMultipliers: map[string]float64{
			"low":    1.0,
			"medium": 2.0,
			"high":   4.0,
		},
	},
	// Flux family — flat pricing, no size/quality variation.
	"flux-1.1-pro": {
		PricePerImage: 0.04,
	},
	"flux-dev": {
		PricePerImage: 0.003,
	},
}

// GetImagePricing returns the pricing profile for an image model.
func GetImagePricing(name string) (ImagePricing, bool) {
	return imagePricingMap.Get(strings.TrimSpace(name))
}

// GetImagePricingCopy returns a defensive copy of the whole map.
func GetImagePricingCopy() map[string]ImagePricing {
	return imagePricingMap.ReadAll()
}

// ImagePricing2JSONString serializes the map for OptionMap.
func ImagePricing2JSONString() string {
	return imagePricingMap.MarshalJSONString()
}

// UpdateImagePricingByJSONString ingests the option-table JSON blob.
func UpdateImagePricingByJSONString(jsonStr string) error {
	return types.LoadFromJsonStringWithCallback(imagePricingMap, jsonStr, InvalidateExposedDataCache)
}

// SeedDefaultImagePricing merges compile-time defaults without overwriting
// any admin-set entries. Called once from InitRatioMap.
func SeedDefaultImagePricing() {
	for name, defaults := range defaultImagePricing {
		if _, exists := imagePricingMap.Get(name); exists {
			continue
		}
		imagePricingMap.Set(name, defaults)
	}
}
