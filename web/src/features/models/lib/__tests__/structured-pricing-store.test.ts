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
  readStructuredPricing,
  updateStructuredPricingMap,
} from '../structured-pricing-store'

describe('structured model pricing storage', () => {
  test('reads the selected model without changing the stored value', () => {
    const stored = JSON.stringify({
      'image-alpha': {
        price_per_image: 0.04,
        quality_multipliers: { hd: 2 },
        size_multipliers: { '1024x1024': 1 },
      },
      'image-beta': { price_per_image: 0.08 },
    })

    assert.deepEqual(readStructuredPricing(stored, 'image-alpha'), {
      price_per_image: 0.04,
      quality_multipliers: { hd: 2 },
      size_multipliers: { '1024x1024': 1 },
    })
  })

  test('updates one model while preserving unrelated entries', () => {
    const stored = JSON.stringify({
      'video-alpha': { price_per_second: 0.1 },
      'video-beta': { price_per_second: 0.2 },
    })

    const updated = updateStructuredPricingMap(stored, {
      previousModelName: 'video-alpha',
      modelName: 'video-alpha',
      pricing: {
        price_per_second: 0.15,
        resolution_multipliers: { '1080p': 1.5 },
      },
      basePriceField: 'price_per_second',
    })

    assert.deepEqual(JSON.parse(updated), {
      'video-alpha': {
        price_per_second: 0.15,
        resolution_multipliers: { '1080p': 1.5 },
      },
      'video-beta': { price_per_second: 0.2 },
    })
  })

  test('removes the current model when its base price is zero', () => {
    const stored = JSON.stringify({
      'whisper-alpha': { price_per_minute: 0.006 },
      'whisper-beta': { price_per_minute: 0.01 },
    })

    const updated = updateStructuredPricingMap(stored, {
      previousModelName: 'whisper-alpha',
      modelName: 'whisper-alpha',
      pricing: { price_per_minute: 0, min_bill_minutes: 0.25 },
      basePriceField: 'price_per_minute',
    })

    assert.deepEqual(JSON.parse(updated), {
      'whisper-beta': { price_per_minute: 0.01 },
    })
  })

  test('moves pricing to a renamed model without leaving the old key', () => {
    const stored = JSON.stringify({
      'tts-old': { price_per_million_chars: 15 },
      'tts-other': { price_per_million_chars: 30 },
    })

    const updated = updateStructuredPricingMap(stored, {
      previousModelName: 'tts-old',
      modelName: 'tts-new',
      pricing: {
        price_per_million_chars: 18,
        voice_multipliers: { clone: 2 },
      },
      basePriceField: 'price_per_million_chars',
    })

    assert.deepEqual(JSON.parse(updated), {
      'tts-new': {
        price_per_million_chars: 18,
        voice_multipliers: { clone: 2 },
      },
      'tts-other': { price_per_million_chars: 30 },
    })
  })
})
