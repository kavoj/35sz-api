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
import { Plus, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

import type {
  AudioInPricing,
  AudioOutPricing,
  ImagePricing,
  VideoPricing,
} from '../../lib/pricing-types'

type MultiplierEditorProps = {
  label: string
  value: Record<string, number>
  onChange: (value: Record<string, number>) => void
}

function MultiplierEditor(props: MultiplierEditorProps) {
  const { t } = useTranslation()
  const entries = Object.entries(props.value)

  const replaceEntry = (index: number, key: string, multiplier: number) => {
    const next: Record<string, number> = {}
    entries.forEach(([currentKey, currentValue], currentIndex) => {
      next[currentIndex === index ? key : currentKey] =
        currentIndex === index ? multiplier : currentValue
    })
    props.onChange(next)
  }

  return (
    <div className='space-y-2'>
      <Label>{props.label}</Label>
      {entries.map(([key, multiplier], index) => (
        <div className='flex gap-2' key={key || 'new-multiplier'}>
          <Input
            aria-label={t('Multiplier name')}
            value={key}
            onChange={(event) =>
              replaceEntry(index, event.target.value, multiplier)
            }
          />
          <Input
            aria-label={t('Multiplier value')}
            className='w-28'
            min='0'
            step='0.01'
            type='number'
            value={multiplier}
            onChange={(event) =>
              replaceEntry(index, key, Number(event.target.value))
            }
          />
          <Button
            aria-label={t('Remove entry')}
            onClick={() =>
              props.onChange(
                Object.fromEntries(
                  entries.filter((_, currentIndex) => currentIndex !== index)
                )
              )
            }
            size='icon'
            type='button'
            variant='ghost'
          >
            <Trash2 className='h-4 w-4' />
          </Button>
        </div>
      ))}
      <Button
        disabled={Object.hasOwn(props.value, '')}
        onClick={() => props.onChange({ ...props.value, '': 1 })}
        size='sm'
        type='button'
        variant='outline'
      >
        <Plus className='h-4 w-4' />
        {t('Add multiplier')}
      </Button>
    </div>
  )
}

type EditorProps<T> = {
  value: T
  onChange: (value: T) => void
  currencyLabel?: string
}

export function ImagePricingEditor(props: EditorProps<ImagePricing>) {
  const { t } = useTranslation()
  return (
    <div className='space-y-4'>
      <PriceInput
        label={t('Base price per image ({{currency}})', {
          currency: props.currencyLabel ?? 'USD',
        })}
        value={props.value.price_per_image}
        onChange={(value) =>
          props.onChange({ ...props.value, price_per_image: value })
        }
      />
      <MultiplierEditor
        label={t('Quality multipliers')}
        value={props.value.quality_multipliers ?? {}}
        onChange={(value) =>
          props.onChange({ ...props.value, quality_multipliers: value })
        }
      />
      <MultiplierEditor
        label={t('Size multipliers')}
        value={props.value.size_multipliers ?? {}}
        onChange={(value) =>
          props.onChange({ ...props.value, size_multipliers: value })
        }
      />
    </div>
  )
}

export function VideoPricingEditor(props: EditorProps<VideoPricing>) {
  const { t } = useTranslation()
  return (
    <div className='space-y-4'>
      <PriceInput
        label={t('Base price per second ({{currency}})', {
          currency: props.currencyLabel ?? 'USD',
        })}
        value={props.value.price_per_second}
        onChange={(value) =>
          props.onChange({ ...props.value, price_per_second: value })
        }
      />
      <MultiplierEditor
        label={t('Resolution multipliers')}
        value={props.value.resolution_multipliers ?? {}}
        onChange={(value) =>
          props.onChange({ ...props.value, resolution_multipliers: value })
        }
      />
      <PriceInput
        label={t('Audio-on multiplier')}
        value={props.value.has_audio_multiplier ?? 0}
        onChange={(value) =>
          props.onChange({ ...props.value, has_audio_multiplier: value })
        }
      />
    </div>
  )
}

export function AudioInPricingEditor(props: EditorProps<AudioInPricing>) {
  const { t } = useTranslation()
  return (
    <div className='space-y-4'>
      <PriceInput
        label={t('Price per minute ({{currency}})', {
          currency: props.currencyLabel ?? 'USD',
        })}
        value={props.value.price_per_minute}
        onChange={(value) =>
          props.onChange({ ...props.value, price_per_minute: value })
        }
      />
      <PriceInput
        label={t('Minimum billable minutes')}
        value={props.value.min_bill_minutes ?? 0}
        onChange={(value) =>
          props.onChange({ ...props.value, min_bill_minutes: value })
        }
      />
    </div>
  )
}

export function AudioOutPricingEditor(props: EditorProps<AudioOutPricing>) {
  const { t } = useTranslation()
  return (
    <div className='space-y-4'>
      <PriceInput
        label={t('Price per 1M characters ({{currency}})', {
          currency: props.currencyLabel ?? 'USD',
        })}
        value={props.value.price_per_million_chars}
        onChange={(value) =>
          props.onChange({ ...props.value, price_per_million_chars: value })
        }
      />
      <MultiplierEditor
        label={t('Voice multipliers')}
        value={props.value.voice_multipliers ?? {}}
        onChange={(value) =>
          props.onChange({ ...props.value, voice_multipliers: value })
        }
      />
    </div>
  )
}

function PriceInput(props: {
  label: string
  value: number
  onChange: (value: number) => void
}) {
  return (
    <div className='space-y-2'>
      <Label>{props.label}</Label>
      <Input
        min='0'
        step='0.0001'
        type='number'
        value={props.value || ''}
        onChange={(event) => props.onChange(Number(event.target.value))}
      />
    </div>
  )
}
