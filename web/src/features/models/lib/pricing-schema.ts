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
export const PRICING_KINDS = [
  'chat',
  'multimodal-chat',
  'image-gen',
  'video-gen',
  'audio-in',
  'audio-out',
  'embedding',
] as const

export type PricingKind = (typeof PRICING_KINDS)[number]

export function normalizePricingKind(value?: string): PricingKind {
  return PRICING_KINDS.includes(value as PricingKind)
    ? (value as PricingKind)
    : 'chat'
}

export function isStructuredPricingKind(
  value: PricingKind
): value is 'image-gen' | 'video-gen' | 'audio-in' | 'audio-out' {
  return ['image-gen', 'video-gen', 'audio-in', 'audio-out'].includes(value)
}

export function getKindLabelKey(value: PricingKind): string {
  const labels: Record<PricingKind, string> = {
    chat: 'Chat (per-token)',
    'multimodal-chat': 'Multimodal chat (per-token)',
    'image-gen': 'Image generation (per-image)',
    'video-gen': 'Video generation (per-second)',
    'audio-in': 'Speech recognition (per-minute)',
    'audio-out': 'Speech synthesis (per-1M-characters)',
    embedding: 'Embedding / Reranking (per-token)',
  }
  return labels[value]
}
