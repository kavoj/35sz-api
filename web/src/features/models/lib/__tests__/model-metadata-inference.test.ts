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

import { inferModelMetadata } from '../model-metadata-inference'

describe('model metadata inference', () => {
  test('classifies all structured media pricing families', () => {
    assert.deepEqual(inferModelMetadata('doubao-seedream-5-0').tags, ['image'])
    assert.deepEqual(inferModelMetadata('doubao-seedance-2-0').tags, ['video'])
    assert.deepEqual(inferModelMetadata('doubao-seed-asr-2-0').tags, [
      'audio',
      'asr',
    ])
    assert.deepEqual(inferModelMetadata('tts-1-hd').tags, ['audio', 'tts'])
    assert.deepEqual(inferModelMetadata('text-embedding-3-large').tags, [
      'embedding',
    ])
  })

  test('adds multimodal and reasoning capabilities without dropping chat', () => {
    assert.deepEqual(inferModelMetadata('gpt-4o-mini').tags, [
      'chat',
      'vision',
      'function-calling',
    ])
    assert.deepEqual(inferModelMetadata('deepseek-r1').tags, [
      'chat',
      'reasoning',
    ])
  })

  test('returns only strongly inferred context windows', () => {
    assert.equal(inferModelMetadata('gpt-4o-mini').contextLength, 128000)
    assert.equal(inferModelMetadata('claude-3-5-sonnet').contextLength, 200000)
    assert.equal(inferModelMetadata('private-model').contextLength, undefined)
  })
})
