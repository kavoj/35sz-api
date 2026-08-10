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
export type ImagePricing = {
  price_per_image: number
  quality_multipliers?: Record<string, number>
  size_multipliers?: Record<string, number>
}

export type VideoPricing = {
  price_per_second: number
  resolution_multipliers?: Record<string, number>
  has_audio_multiplier?: number
}

export type AudioInPricing = {
  price_per_minute: number
  min_bill_minutes?: number
}

export type AudioOutPricing = {
  price_per_million_chars: number
  voice_multipliers?: Record<string, number>
}

export const emptyImagePricing = (): ImagePricing => ({
  price_per_image: 0,
  quality_multipliers: {},
  size_multipliers: {},
})

export const emptyVideoPricing = (): VideoPricing => ({
  price_per_second: 0,
  resolution_multipliers: {},
  has_audio_multiplier: 0,
})

export const emptyAudioInPricing = (): AudioInPricing => ({
  price_per_minute: 0,
  min_bill_minutes: 0,
})

export const emptyAudioOutPricing = (): AudioOutPricing => ({
  price_per_million_chars: 0,
  voice_multipliers: {},
})
