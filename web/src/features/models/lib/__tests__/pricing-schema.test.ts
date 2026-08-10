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
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import {
  isStructuredPricingKind,
  normalizePricingKind,
} from '../pricing-schema'

describe('model pricing schema', () => {
  test('normalizes missing and unsupported persisted values to chat', () => {
    assert.equal(normalizePricingKind(), 'chat')
    assert.equal(normalizePricingKind('unsupported'), 'chat')
  })

  test('keeps every structured pricing kind distinct from token pricing', () => {
    assert.equal(isStructuredPricingKind('image-gen'), true)
    assert.equal(isStructuredPricingKind('video-gen'), true)
    assert.equal(isStructuredPricingKind('audio-in'), true)
    assert.equal(isStructuredPricingKind('audio-out'), true)
    assert.equal(isStructuredPricingKind('embedding'), false)
  })
})
