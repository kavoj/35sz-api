/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import * as LobeIcons from '@lobehub/icons'
import { Search, Upload, X } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { getLobeIcon } from '@/lib/lobe-icon'
import { cn } from '@/lib/utils'

const POPULAR_ICONS = [
  'OpenAI.Color',
  'Anthropic.Color',
  'Google.Color',
  'Gemini.Color',
  'Qwen.Color',
  'Doubao.Color',
  'DeepSeek.Color',
  'Alibaba.Color',
  'ByteDance.Color',
  'Moonshot.Color',
  'Zhipu.Color',
  'Mistral.Color',
  'Cohere.Color',
  'Kling.Color',
  'Jimeng.Color',
  'Vidu.Color',
]

function getIconKeys() {
  const keys: string[] = []
  for (const [key, value] of Object.entries(LobeIcons)) {
    if (key.startsWith('_') || !/^[A-Z]/.test(key)) continue
    if (!value || (typeof value !== 'object' && typeof value !== 'function')) continue
    if (typeof value === 'object' && !('$$typeof' in value)) continue
    keys.push(key)
    if (typeof value === 'object') {
      if ('Color' in value) keys.push(`${key}.Color`)
      if ('Avatar' in value) keys.push(`${key}.Avatar`)
    }
  }
  return keys
}

type IconSelectorProps = {
  value?: string
  onChange: (value: string) => void
  placeholder?: string
  className?: string
}

export function IconSelector({
  value,
  onChange,
  placeholder,
  className,
}: IconSelectorProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState('')
  const [customValue, setCustomValue] = useState(value ?? '')
  const [activeTab, setActiveTab] = useState<'popular' | 'custom' | 'upload'>('popular')
  const allIcons = useMemo(() => getIconKeys(), [])
  const filteredIcons = useMemo(() => {
    const keyword = search.trim().toLowerCase()
    return (keyword ? allIcons.filter((icon) => icon.toLowerCase().includes(keyword)) : allIcons).slice(0, 96)
  }, [allIcons, search])

  const select = (icon: string) => {
    onChange(icon)
    setOpen(false)
    setSearch('')
  }

  const upload = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]
    if (!file) return
    const reader = new FileReader()
    reader.onload = () => select(String(reader.result ?? ''))
    reader.readAsDataURL(file)
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger>
        <Button
          type='button'
          variant='outline'
          role='combobox'
          aria-expanded={open}
          className={cn('w-full justify-between', className)}
        >
          <span className='flex min-w-0 items-center gap-2'>
            {value ? getLobeIcon(value, 20) : <span className='text-muted-foreground'>{placeholder || t('Select icon')}</span>}
            {value && !value.startsWith('data:') && <span className='truncate text-sm text-muted-foreground'>{value}</span>}
          </span>
          {value && <X className='size-4 shrink-0' onClick={(event) => { event.stopPropagation(); onChange('') }} />}
        </Button>
      </PopoverTrigger>
      <PopoverContent className='w-96 p-3' align='start'>
        <div className='mb-3 grid grid-cols-3 gap-1'>
          {(['popular', 'custom', 'upload'] as const).map((tab) => (
            <Button key={tab} type='button' size='sm' variant={activeTab === tab ? 'secondary' : 'ghost'} onClick={() => setActiveTab(tab)}>
              {t(tab === 'popular' ? 'Popular' : tab === 'custom' ? 'Custom' : 'Upload')}
            </Button>
          ))}
        </div>
        {activeTab === 'popular' && (
          <>
            <div className='relative mb-2'>
              <Search className='absolute top-1/2 left-2 size-4 -translate-y-1/2 text-muted-foreground' />
              <Input className='pl-8' placeholder={t('Search icons...')} value={search} onChange={(event) => setSearch(event.target.value)} />
            </div>
            <div className='grid max-h-64 grid-cols-4 gap-2 overflow-y-auto'>
              {(search ? filteredIcons : POPULAR_ICONS.filter((icon) => allIcons.includes(icon))).map((icon) => (
                <Button key={icon} type='button' variant={value === icon ? 'secondary' : 'ghost'} className='h-16 flex-col gap-1' title={icon} onClick={() => select(icon)}>
                  {getLobeIcon(icon, 26)}
                  <span className='max-w-full truncate text-[10px]'>{icon.split('.')[0]}</span>
                </Button>
              ))}
            </div>
          </>
        )}
        {activeTab === 'custom' && (
          <div className='space-y-2'>
            <Input placeholder={t('@lobehub/icons key')} value={customValue} onChange={(event) => setCustomValue(event.target.value)} />
            <Button type='button' className='w-full' onClick={() => select(customValue.trim())} disabled={!customValue.trim()}>{t('Apply')}</Button>
          </div>
        )}
        {activeTab === 'upload' && (
          <label className='flex cursor-pointer flex-col items-center gap-2 rounded-md border border-dashed p-6 text-sm text-muted-foreground'>
            <Upload className='size-5' />
            {t('Upload icon')}
            <input className='hidden' type='file' accept='image/*' onChange={upload} />
          </label>
        )}
      </PopoverContent>
    </Popover>
  )
}
