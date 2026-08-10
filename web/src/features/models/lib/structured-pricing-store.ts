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
import { safeJsonParse } from '@/features/system-settings/utils/json-parser'

type StructuredPricing = Record<string, unknown>

type UpdateStructuredPricingInput<TPricing extends StructuredPricing> = {
  previousModelName?: string
  modelName: string
  pricing: TPricing
  basePriceField: string
}

function parseStructuredPricingMap(
  storedValue: string
): Record<string, StructuredPricing> {
  return safeJsonParse<Record<string, StructuredPricing>>(storedValue, {
    fallback: {},
    silent: true,
  })
}

export function readStructuredPricing<TPricing extends StructuredPricing>(
  storedValue: string,
  modelName: string
): TPricing | undefined {
  return parseStructuredPricingMap(storedValue)[modelName] as
    | TPricing
    | undefined
}

export function updateStructuredPricingMap<TPricing extends StructuredPricing>(
  storedValue: string,
  input: UpdateStructuredPricingInput<TPricing>
): string {
  const pricingMap = parseStructuredPricingMap(storedValue)

  if (input.previousModelName && input.previousModelName !== input.modelName) {
    delete pricingMap[input.previousModelName]
  }

  delete pricingMap[input.modelName]

  const basePrice = input.pricing[input.basePriceField]
  if (typeof basePrice === 'number' && basePrice > 0) {
    pricingMap[input.modelName] = input.pricing
  }

  return JSON.stringify(pricingMap)
}
