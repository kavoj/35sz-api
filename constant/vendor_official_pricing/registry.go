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

package vendor_official_pricing

import (
	"regexp"
	"strings"
)

// Registry indexes the compiled-in vendor catalogs. New vendors just
// register a new file with a top-level map[string]VendorPricingEntry and
// append themselves here — no per-vendor lookup code needed.
//
// The map key is a vendor slug that matches what /api/vendor_mapping
// exposes and what admins pick in the drawer's Vendor dropdown. Keeping
// them 1:1 lets the handler filter by vendor when the model is
// registered but doesn't identify itself through a name prefix.
var Registry = map[string]map[string]VendorPricingEntry{
	"doubao":     DoubaoStaticPricing,
	"volcengine": DoubaoStaticPricing, // Volcengine's Ark platform = Doubao
	"bytedance":  DoubaoStaticPricing,
}

// Lookup finds a static baseline entry for the given model name. It tries
// three matching strategies in order and returns the first hit:
//
//   1. Exact key match against every registered vendor's catalog.
//   2. Normalized fuzzy match (dash-versions to dot-versions, lowercase,
//      substring in either direction). This is the same rule frontend
//      model-name-normalize.ts uses so both sides find the same entries.
//   3. Miss → returns zero entry + ok=false.
//
// The bool value is what the caller checks: false means "no static
// baseline for this model, use the pricing_incomplete fallback".
//
// vendorHint is optional. When provided, we search that vendor's catalog
// first and skip others. When empty, we scan every vendor in Registry
// (rare — model_name usually carries enough signal for the fuzzy match
// to disambiguate).
func Lookup(modelName, vendorHint string) (VendorPricingEntry, bool) {
	if modelName == "" {
		return VendorPricingEntry{}, false
	}

	// Fast path: exact match against a hinted vendor.
	if vendorHint != "" {
		if catalog, ok := Registry[strings.ToLower(vendorHint)]; ok {
			if entry, ok := catalog[modelName]; ok {
				return entry, true
			}
		}
	}

	// Full scan for exact match.
	for _, catalog := range Registry {
		if entry, ok := catalog[modelName]; ok {
			return entry, true
		}
	}

	// Fuzzy scan. Precompute normalized target once, then compare each
	// catalog key's normalized form. Longest overlap wins so
	// `doubao-seed-1.6-vision` beats bare `doubao-seed-1.6`.
	target := normalizeName(modelName)
	if target == "" {
		return VendorPricingEntry{}, false
	}

	var bestEntry VendorPricingEntry
	bestOverlap := 0
	for _, catalog := range Registry {
		for key, entry := range catalog {
			nk := normalizeName(key)
			if nk == "" {
				continue
			}
			overlap := 0
			if strings.Contains(target, nk) || strings.Contains(nk, target) {
				overlap = min(len(target), len(nk))
			}
			if overlap > bestOverlap {
				bestOverlap = overlap
				bestEntry = entry
			}
		}
	}
	if bestOverlap > 0 {
		return bestEntry, true
	}

	return VendorPricingEntry{}, false
}

// versionDashPattern matches `-N-M-` mid-string and `-N-M` at
// end-of-string. Digit runs are limited to 1–3 characters so we don't
// stitch back model-size suffixes like `-7b` or date stamps like
// `-260628`. Mirrors versionDashesToDots in
// web/default/src/features/models/lib/model-name-normalize.ts.
var (
	versionDashMid = regexp.MustCompile(`(-\d{1,3})-(\d{1,3})(?:-)`)
	versionDashEnd = regexp.MustCompile(`(-\d{1,3})-(\d{1,3})$`)
)

func normalizeName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	// Mid-string first, terminal second — same order as the frontend
	// implementation so the two produce identical output.
	s = versionDashMid.ReplaceAllString(s, "$1.$2-")
	s = versionDashEnd.ReplaceAllString(s, "$1.$2")
	return s
}
