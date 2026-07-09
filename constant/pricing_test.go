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

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// PricingKind is persisted on `model_meta.pricing_kind` and consumed by every
// billing dispatch site. These tests pin down the invariant that "empty or
// unknown → chat" — legacy rows that predate the column MUST keep working
// under the fallback path.

func TestPricingKindConstantsAreUnique(t *testing.T) {
	kinds := []string{
		PricingKindChat,
		PricingKindMultimodalChat,
		PricingKindImageGen,
		PricingKindVideoGen,
		PricingKindAudioIn,
		PricingKindAudioOut,
		PricingKindEmbedding,
	}
	seen := make(map[string]struct{}, len(kinds))
	for _, k := range kinds {
		_, dup := seen[k]
		require.Falsef(t, dup, "duplicate pricing kind constant: %q", k)
		require.NotEmpty(t, k, "pricing kind constant must not be empty string")
		seen[k] = struct{}{}
	}
}

func TestIsValidPricingKind(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  bool
	}{
		{"chat is valid", PricingKindChat, true},
		{"video-gen is valid", PricingKindVideoGen, true},
		{"embedding is valid", PricingKindEmbedding, true},
		{"empty string is invalid", "", false},
		{"whitespace is invalid", " chat ", false},
		{"unknown value is invalid", "voice-cloning", false},
		{"case sensitive - uppercase is invalid", "CHAT", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IsValidPricingKind(tc.input))
		})
	}
}

func TestNormalizePricingKindFallsBackToChat(t *testing.T) {
	// The whole point of NormalizePricingKind is guarding downstream switch
	// statements: they should NEVER see empty or garbage values.
	cases := []struct {
		input string
		want  string
	}{
		{"", PricingKindChat},
		{"unknown", PricingKindChat},
		{"CHAT", PricingKindChat},
		{PricingKindChat, PricingKindChat},
		{PricingKindVideoGen, PricingKindVideoGen},
		{PricingKindAudioOut, PricingKindAudioOut},
	}
	for _, tc := range cases {
		assert.Equalf(t, tc.want, NormalizePricingKind(tc.input),
			"NormalizePricingKind(%q)", tc.input)
	}
}
