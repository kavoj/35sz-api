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
import { zodResolver } from '@hookform/resolvers/zod'
import { AlertTriangle, ChevronDown } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import * as z from 'zod'

import {
  sideDrawerContentClassName,
  sideDrawerFooterClassName,
} from '@/components/drawer-layout'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
} from '@/components/ui/input-group'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { combineBillingExpr } from '@/features/pricing/lib/billing-expr'
import { cn } from '@/lib/utils'

import {
  SettingsControlGroup,
  SettingsSwitchField,
} from '../components/settings-form-layout'
import {
  displayPricingValueToUsd,
  formatModelPricingAmountFromUSD,
  formatPricingNumber,
  getModelPricingCurrencyPrefix,
  getModelPricingUnitLabel,
  usdToDisplayPricingValue,
} from './pricing-format'
import { TieredPricingEditor } from './tiered-pricing-editor'
import {
  convertBillingDisplayToUSD,
  convertUSDToBillingDisplay,
  getCurrencyDisplay,
} from '@/lib/currency'
import { safeJsonParse } from '../utils/json-parser'
import { normalizeJsonString } from './utils'
import {
  ImageGenEditor,
  VideoGenEditor,
  AudioInEditor,
  AudioOutEditor,
} from '@/features/models/components/pricing/structured-pricing-editor'
import {
  emptyImagePricing,
  emptyVideoPricing,
  emptyAudioInPricing,
  emptyAudioOutPricing,
} from '@/features/models/lib/pricing-types'
import type {
  ImagePricing,
  VideoPricing,
  AudioInPricing,
  AudioOutPricing,
} from '@/features/models/lib/pricing-types'

const createModelPricingSchema = (t: (key: string) => string) =>
  z.object({
    name: z.string().min(1, t('Model name is required')),
    price: z.string().optional(),
    ratio: z.string().optional(),
    cacheRatio: z.string().optional(),
    createCacheRatio: z.string().optional(),
    completionRatio: z.string().optional(),
    imageRatio: z.string().optional(),
    audioRatio: z.string().optional(),
    audioCompletionRatio: z.string().optional(),
    imagePricing: z.string().optional(),
    videoPricing: z.string().optional(),
    audioInPricing: z.string().optional(),
    audioOutPricing: z.string().optional(),
  })

type ModelPricingFormValues = z.infer<
  ReturnType<typeof createModelPricingSchema>
>

type PricingMode =
  | 'per-token'
  | 'per-request'
  | 'tiered_expr'
  | 'per-image'
  | 'per-second'
  | 'per-minute'
  | 'per-1m-chars'
type LaneKey =
  | 'completion'
  | 'cache'
  | 'createCache'
  | 'image'
  | 'audioInput'
  | 'audioOutput'

export type ModelRatioData = {
  name: string
  price?: string
  ratio?: string
  cacheRatio?: string
  createCacheRatio?: string
  completionRatio?: string
  imageRatio?: string
  audioRatio?: string
  audioCompletionRatio?: string
  imagePricing?: string
  videoPricing?: string
  audioInPricing?: string
  audioOutPricing?: string
  billingMode?: PricingMode
  billingExpr?: string
  requestRuleExpr?: string
}

type ModelPricingSheetProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSave: (data: ModelRatioData) => void
  onCancel?: () => void
  editData?: ModelRatioData | null
  selectedTargetCount?: number
}

type ModelPricingEditorPanelProps = Omit<
  ModelPricingSheetProps,
  'open' | 'onOpenChange'
> & {
  className?: string
}

type PreviewRow = {
  key: string
  label: string
  value: string
  multiline?: boolean
}

const numericDraftRegex = /^(\d+(\.\d*)?|\.\d*)?$/

const EMPTY_LANE_PRICES: Record<LaneKey, string> = {
  completion: '',
  cache: '',
  createCache: '',
  image: '',
  audioInput: '',
  audioOutput: '',
}

const EMPTY_LANE_ENABLED: Record<LaneKey, boolean> = {
  completion: false,
  cache: false,
  createCache: false,
  image: false,
  audioInput: false,
  audioOutput: false,
}

const ratioFieldByLane: Record<LaneKey, keyof ModelPricingFormValues> = {
  completion: 'completionRatio',
  cache: 'cacheRatio',
  createCache: 'createCacheRatio',
  image: 'imageRatio',
  audioInput: 'audioRatio',
  audioOutput: 'audioCompletionRatio',
}

const laneConfigs: Array<{
  key: LaneKey
  titleKey: string
  descriptionKey: string
  placeholder: string
}> = [
  {
    key: 'completion',
    titleKey: 'Completion price',
    descriptionKey: 'Output token price for generated tokens.',
    placeholder: '15',
  },
  {
    key: 'cache',
    titleKey: 'Cache read price',
    descriptionKey: 'Token price for cache reads.',
    placeholder: '0.3',
  },
  {
    key: 'createCache',
    titleKey: 'Cache write price',
    descriptionKey: 'Token price for creating cache entries.',
    placeholder: '3.75',
  },
  {
    key: 'image',
    titleKey: 'Image input price',
    descriptionKey: 'Token price for image input.',
    placeholder: '2.5',
  },
  {
    key: 'audioInput',
    titleKey: 'Audio input price',
    descriptionKey: 'Token price for audio input.',
    placeholder: '3.81',
  },
  {
    key: 'audioOutput',
    titleKey: 'Audio output price',
    descriptionKey: 'Token price for audio output.',
    placeholder: '15.11',
  },
]

function hasValue(value: unknown): boolean {
  return (
    value !== '' && value !== null && value !== undefined && value !== false
  )
}

function toNumberOrNull(value: unknown): number | null {
  if (!hasValue(value) && value !== 0) return null
  const num = Number(value)
  return Number.isFinite(num) ? num : null
}

function ratioToBasePrice(ratio: unknown): string {
  const num = toNumberOrNull(ratio)
  if (num === null) return ''
  return usdToDisplayPricingValue(num * 2)
}

function deriveLanePrice(
  ratio: unknown,
  denominator: unknown,
  fallback = ''
): string {
  const ratioNumber = toNumberOrNull(ratio)
  const denominatorNumber = displayPricingValueToUsd(denominator)
  if (ratioNumber === null || denominatorNumber === null) return fallback
  return usdToDisplayPricingValue(ratioNumber * denominatorNumber)
}

function createInitialLaneState(data?: ModelRatioData | null) {
  if (!data) {
    return {
      promptPrice: '',
      prices: { ...EMPTY_LANE_PRICES },
      enabled: { ...EMPTY_LANE_ENABLED },
    }
  }

  const promptPrice = ratioToBasePrice(data.ratio)
  const audioInputPrice = deriveLanePrice(data.audioRatio, promptPrice)
  const prices: Record<LaneKey, string> = {
    completion: deriveLanePrice(data.completionRatio, promptPrice),
    cache: deriveLanePrice(data.cacheRatio, promptPrice),
    createCache: deriveLanePrice(data.createCacheRatio, promptPrice),
    image: deriveLanePrice(data.imageRatio, promptPrice),
    audioInput: audioInputPrice,
    audioOutput: deriveLanePrice(data.audioCompletionRatio, audioInputPrice),
  }

  return {
    promptPrice,
    prices,
    enabled: {
      completion: hasValue(data.completionRatio),
      cache: hasValue(data.cacheRatio),
      createCache: hasValue(data.createCacheRatio),
      image: hasValue(data.imageRatio),
      audioInput: hasValue(data.audioRatio),
      audioOutput: hasValue(data.audioCompletionRatio),
    },
  }
}

function getModeLabel(mode: PricingMode) {
  if (mode === 'per-request') return 'Per-request'
  if (mode === 'tiered_expr') return 'Expression'
  return 'Per-token'
}

function getModeBadgeVariant(
  mode: PricingMode
): 'default' | 'secondary' | 'outline' {
  if (mode === 'per-request') return 'secondary'
  if (mode === 'tiered_expr') return 'default'
  return 'outline'
}

function formatDisplayPricingValue(value: unknown): string {
  const valueUsd = displayPricingValueToUsd(value)
  return valueUsd === null ? '' : formatModelPricingAmountFromUSD(valueUsd)
}

function buildPreviewRows(
  values: ModelPricingFormValues,
  mode: PricingMode,
  billingExpr: string,
  requestRuleExpr: string,
  promptPrice: string,
  lanePrices: Record<LaneKey, string>,
  laneEnabled: Record<LaneKey, boolean>,
  t: (key: string) => string
): PreviewRow[] {
  if (mode === 'tiered_expr') {
    const effectiveExpr = combineBillingExpr(billingExpr, requestRuleExpr)
    return [
      { key: 'mode', label: 'BillingMode', value: 'tiered_expr' },
      {
        key: 'expr',
        label: t('Expression'),
        value: effectiveExpr || t('Empty'),
        multiline: true,
      },
    ]
  }

  if (mode === 'per-request') {
    return [
      {
        key: 'price',
        label: 'ModelPrice',
        value: values.price
          ? formatDisplayPricingValue(values.price)
          : t('Empty'),
      },
    ]
  }

  return [
    {
      key: 'inputPrice',
      label: t('Input price'),
      value: promptPrice ? formatDisplayPricingValue(promptPrice) : t('Empty'),
    },
    {
      key: 'completion',
      label: t('Completion price'),
      value:
        laneEnabled.completion && lanePrices.completion
          ? formatDisplayPricingValue(lanePrices.completion)
          : t('Empty'),
    },
    {
      key: 'cache',
      label: t('Cache read price'),
      value:
        laneEnabled.cache && lanePrices.cache
          ? formatDisplayPricingValue(lanePrices.cache)
          : t('Empty'),
    },
    {
      key: 'createCache',
      label: t('Cache write price'),
      value:
        laneEnabled.createCache && lanePrices.createCache
          ? formatDisplayPricingValue(lanePrices.createCache)
          : t('Empty'),
    },
    {
      key: 'image',
      label: t('Image input price'),
      value:
        laneEnabled.image && lanePrices.image
          ? formatDisplayPricingValue(lanePrices.image)
          : t('Empty'),
    },
    {
      key: 'audio',
      label: t('Audio input price'),
      value:
        laneEnabled.audioInput && lanePrices.audioInput
          ? formatDisplayPricingValue(lanePrices.audioInput)
          : t('Empty'),
    },
    {
      key: 'audioCompletion',
      label: t('Audio output price'),
      value:
        laneEnabled.audioOutput && lanePrices.audioOutput
          ? formatDisplayPricingValue(lanePrices.audioOutput)
          : t('Empty'),
    },
  ]
}

export function ModelPricingSheet({
  open,
  onOpenChange,
  onSave,
  onCancel,
  editData,
  selectedTargetCount = 0,
}: ModelPricingSheetProps) {
  const { t } = useTranslation()
  const title = editData ? t('Edit model pricing') : t('Add model pricing')
  const description = editData?.name || t('New model')

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side='right'
        className={sideDrawerContentClassName('sm:max-w-2xl')}
      >
        <SheetHeader className='sr-only'>
          <SheetTitle>{title}</SheetTitle>
          <SheetDescription>{description}</SheetDescription>
        </SheetHeader>
        <ModelPricingEditorPanel
          onSave={onSave}
          editData={editData}
          selectedTargetCount={selectedTargetCount}
          onCancel={() => {
            onCancel?.()
            onOpenChange(false)
          }}
          className='h-full rounded-none border-0'
        />
      </SheetContent>
    </Sheet>
  )
}

export function ModelPricingEditorPanel({
  onSave,
  editData,
  selectedTargetCount = 0,
  onCancel,
  className,
}: ModelPricingEditorPanelProps) {
  const { t } = useTranslation()
  const [pricingMode, setPricingMode] = useState<PricingMode>('per-token')
  const [promptPrice, setPromptPrice] = useState('')
  const [lanePrices, setLanePrices] = useState<Record<LaneKey, string>>({
    ...EMPTY_LANE_PRICES,
  })
  const [laneEnabled, setLaneEnabled] = useState<Record<LaneKey, boolean>>({
    ...EMPTY_LANE_ENABLED,
  })
  const [billingExpr, setBillingExpr] = useState('')
  const [requestRuleExpr, setRequestRuleExpr] = useState('')
  const [previewOpen, setPreviewOpen] = useState(true)
  const [imagePricing, setImagePricing] = useState<ImagePricing>(() =>
    emptyImagePricing()
  )
  const [videoPricing, setVideoPricing] = useState<VideoPricing>(() =>
    emptyVideoPricing()
  )
  const [audioInPricing, setAudioInPricing] = useState<AudioInPricing>(() =>
    emptyAudioInPricing()
  )
  const [audioOutPricing, setAudioOutPricing] = useState<AudioOutPricing>(
    () => emptyAudioOutPricing()
  )
  const isEditMode = !!editData

  const { config: currencyConfig } = getCurrencyDisplay()
  const pricingCurrencyLabel =
    currencyConfig.quotaDisplayType === 'CNY'
      ? '元'
      : currencyConfig.quotaDisplayType === 'CUSTOM'
        ? '自定义'
        : 'USD'

  const form = useForm<ModelPricingFormValues>({
    resolver: zodResolver(createModelPricingSchema(t)),
    defaultValues: {
      name: '',
      price: '',
      ratio: '',
      cacheRatio: '',
      createCacheRatio: '',
      completionRatio: '',
      imageRatio: '',
      audioRatio: '',
      audioCompletionRatio: '',
      imagePricing: '',
      videoPricing: '',
      audioInPricing: '',
      audioOutPricing: '',
    },
  })

  useEffect(() => {
    const nextLaneState = createInitialLaneState(editData)

    if (editData) {
      form.reset({
        name: editData.name,
        price: usdToDisplayPricingValue(editData.price),
        ratio: editData.ratio || '',
        cacheRatio: editData.cacheRatio || '',
        createCacheRatio: editData.createCacheRatio || '',
        completionRatio: editData.completionRatio || '',
        imageRatio: editData.imageRatio || '',
        audioRatio: editData.audioRatio || '',
        audioCompletionRatio: editData.audioCompletionRatio || '',
        imagePricing: editData.imagePricing || '',
        videoPricing: editData.videoPricing || '',
        audioInPricing: editData.audioInPricing || '',
        audioOutPricing: editData.audioOutPricing || '',
      })
      const billingMode = editData.billingMode
      const isStructured =
        billingMode === 'per-image' ||
        billingMode === 'per-second' ||
        billingMode === 'per-minute' ||
        billingMode === 'per-1m-chars'
      setPricingMode(
        isStructured
          ? (billingMode as PricingMode)
          : billingMode === 'tiered_expr'
            ? 'tiered_expr'
            : editData.price
              ? 'per-request'
              : 'per-token'
      )
      setBillingExpr(editData.billingExpr || '')
      setRequestRuleExpr(editData.requestRuleExpr || '')

      // Initialize structured pricing from editData
      const modelName = editData.name
      const imageMap = safeJsonParse<Record<string, ImagePricing>>(
        editData.imagePricing || '{}',
        { fallback: {}, silent: true }
      )
      const rawImg = imageMap[modelName]
      setImagePricing(
        rawImg
          ? {
              ...rawImg,
              price_per_image:
                convertUSDToBillingDisplay(rawImg.price_per_image),
            }
          : emptyImagePricing()
      )
      const videoMap = safeJsonParse<Record<string, VideoPricing>>(
        editData.videoPricing || '{}',
        { fallback: {}, silent: true }
      )
      const rawVid = videoMap[modelName]
      setVideoPricing(
        rawVid
          ? {
              ...rawVid,
              price_per_second:
                convertUSDToBillingDisplay(rawVid.price_per_second),
            }
          : emptyVideoPricing()
      )
      const audioInMap = safeJsonParse<Record<string, AudioInPricing>>(
        editData.audioInPricing || '{}',
        { fallback: {}, silent: true }
      )
      const rawAin = audioInMap[modelName]
      setAudioInPricing(
        rawAin
          ? {
              ...rawAin,
              price_per_minute:
                convertUSDToBillingDisplay(rawAin.price_per_minute),
            }
          : emptyAudioInPricing()
      )
      const audioOutMap = safeJsonParse<Record<string, AudioOutPricing>>(
        editData.audioOutPricing || '{}',
        { fallback: {}, silent: true }
      )
      const rawAout = audioOutMap[modelName]
      setAudioOutPricing(
        rawAout
          ? {
              ...rawAout,
              price_per_million_chars:
                convertUSDToBillingDisplay(
                  rawAout.price_per_million_chars
                ),
            }
          : emptyAudioOutPricing()
      )
    } else {
      form.reset({
        name: '',
        price: '',
        ratio: '',
        cacheRatio: '',
        createCacheRatio: '',
        completionRatio: '',
        imageRatio: '',
        audioRatio: '',
        audioCompletionRatio: '',
        imagePricing: '',
        videoPricing: '',
        audioInPricing: '',
        audioOutPricing: '',
      })
      setPricingMode('per-token')
      setBillingExpr('')
      setRequestRuleExpr('')
      setImagePricing(emptyImagePricing())
      setVideoPricing(emptyVideoPricing())
      setAudioInPricing(emptyAudioInPricing())
      setAudioOutPricing(emptyAudioOutPricing())
    }

    setPromptPrice(nextLaneState.promptPrice)
    setLanePrices(nextLaneState.prices)
    setLaneEnabled(nextLaneState.enabled)
    setPreviewOpen(true)
  }, [editData, form])

  const setFormValue = (field: keyof ModelPricingFormValues, value: string) => {
    form.setValue(field, value, {
      shouldDirty: true,
      shouldValidate: true,
    })
  }

  const deriveLaneRatio = (
    lane: LaneKey,
    price: string,
    nextPromptPrice = promptPrice,
    nextLanePrices = lanePrices
  ) => {
    const priceNumber = displayPricingValueToUsd(price)
    if (priceNumber === null) return ''

    if (lane === 'audioOutput') {
      const audioInputPrice = displayPricingValueToUsd(
        nextLanePrices.audioInput
      )
      if (audioInputPrice === null || audioInputPrice === 0) return ''
      return formatPricingNumber(priceNumber / audioInputPrice)
    }

    const inputPrice = displayPricingValueToUsd(nextPromptPrice)
    if (inputPrice === null || inputPrice === 0) return ''
    return formatPricingNumber(priceNumber / inputPrice)
  }

  const syncLaneRatios = (
    nextPromptPrice = promptPrice,
    nextLanePrices = lanePrices,
    nextLaneEnabled = laneEnabled
  ) => {
    const inputPrice = displayPricingValueToUsd(nextPromptPrice)
    setFormValue(
      'ratio',
      inputPrice !== null ? formatPricingNumber(inputPrice / 2) : ''
    )

    laneConfigs.forEach(({ key }) => {
      const ratioField = ratioFieldByLane[key]
      if (!nextLaneEnabled[key]) {
        setFormValue(ratioField, '')
        return
      }
      setFormValue(
        ratioField,
        deriveLaneRatio(
          key,
          nextLanePrices[key],
          nextPromptPrice,
          nextLanePrices
        )
      )
    })
  }

  const handlePromptPriceChange = (value: string) => {
    if (!numericDraftRegex.test(value)) return
    setPromptPrice(value)
    syncLaneRatios(value, lanePrices, laneEnabled)
  }

  const handleLanePriceChange = (lane: LaneKey, value: string) => {
    if (!numericDraftRegex.test(value)) return
    const nextLanePrices = { ...lanePrices, [lane]: value }
    setLanePrices(nextLanePrices)

    if (laneEnabled[lane]) {
      setFormValue(
        ratioFieldByLane[lane],
        deriveLaneRatio(lane, value, promptPrice, nextLanePrices)
      )
    }

    if (lane === 'audioInput' && laneEnabled.audioOutput) {
      setFormValue(
        'audioCompletionRatio',
        deriveLaneRatio(
          'audioOutput',
          nextLanePrices.audioOutput,
          promptPrice,
          nextLanePrices
        )
      )
    }
  }

  const handleLaneToggle = (lane: LaneKey, checked: boolean) => {
    const nextEnabled = { ...laneEnabled, [lane]: checked }
    let nextPrices = lanePrices

    if (!checked) {
      nextPrices = { ...nextPrices, [lane]: '' }
      setFormValue(ratioFieldByLane[lane], '')
      if (lane === 'audioInput') {
        nextEnabled.audioOutput = false
        nextPrices.audioOutput = ''
        setFormValue('audioCompletionRatio', '')
      }
    }

    setLaneEnabled(nextEnabled)
    setLanePrices(nextPrices)

    if (checked) {
      setFormValue(
        ratioFieldByLane[lane],
        deriveLaneRatio(lane, nextPrices[lane], promptPrice, nextPrices)
      )
    }
  }

  const handleModeChange = (value: string) => {
    const nextMode = value as PricingMode
    setPricingMode(nextMode)
    if (nextMode === 'tiered_expr' && !billingExpr) {
      setBillingExpr('tier("base", p * 0 + c * 0)')
    }
  }

  // Structured pricing change handlers — update local state + persist to form field
  const handleImagePricingChange = (next: ImagePricing) => {
    setImagePricing(next)
    const mName = form.getValues('name') || editData?.name || ''
    if (!mName) return
    const map = safeJsonParse<Record<string, ImagePricing>>(
      form.getValues('imagePricing') || '{}',
      { fallback: {}, silent: true }
    )
    if (next.price_per_image > 0) {
      map[mName] = {
        ...next,
        price_per_image: convertBillingDisplayToUSD(next.price_per_image),
      }
    } else {
      delete map[mName]
    }
    form.setValue(
      'imagePricing',
      normalizeJsonString(JSON.stringify(map)),
      { shouldDirty: true }
    )
  }

  const handleVideoPricingChange = (next: VideoPricing) => {
    setVideoPricing(next)
    const mName = form.getValues('name') || editData?.name || ''
    if (!mName) return
    const map = safeJsonParse<Record<string, VideoPricing>>(
      form.getValues('videoPricing') || '{}',
      { fallback: {}, silent: true }
    )
    if (next.price_per_second > 0) {
      map[mName] = {
        ...next,
        price_per_second: convertBillingDisplayToUSD(next.price_per_second),
      }
    } else {
      delete map[mName]
    }
    form.setValue(
      'videoPricing',
      normalizeJsonString(JSON.stringify(map)),
      { shouldDirty: true }
    )
  }

  const handleAudioInPricingChange = (next: AudioInPricing) => {
    setAudioInPricing(next)
    const mName = form.getValues('name') || editData?.name || ''
    if (!mName) return
    const map = safeJsonParse<Record<string, AudioInPricing>>(
      form.getValues('audioInPricing') || '{}',
      { fallback: {}, silent: true }
    )
    if (next.price_per_minute > 0) {
      map[mName] = {
        ...next,
        price_per_minute: convertBillingDisplayToUSD(
          next.price_per_minute
        ),
      }
    } else {
      delete map[mName]
    }
    form.setValue(
      'audioInPricing',
      normalizeJsonString(JSON.stringify(map)),
      { shouldDirty: true }
    )
  }

  const handleAudioOutPricingChange = (next: AudioOutPricing) => {
    setAudioOutPricing(next)
    const mName = form.getValues('name') || editData?.name || ''
    if (!mName) return
    const map = safeJsonParse<Record<string, AudioOutPricing>>(
      form.getValues('audioOutPricing') || '{}',
      { fallback: {}, silent: true }
    )
    if (next.price_per_million_chars > 0) {
      map[mName] = {
        ...next,
        price_per_million_chars: convertBillingDisplayToUSD(
          next.price_per_million_chars
        ),
      }
    } else {
      delete map[mName]
    }
    form.setValue(
      'audioOutPricing',
      normalizeJsonString(JSON.stringify(map)),
      { shouldDirty: true }
    )
  }

  const watchedValues = form.watch()
  const previewRows = useMemo(
    () =>
      buildPreviewRows(
        watchedValues,
        pricingMode,
        billingExpr,
        requestRuleExpr,
        promptPrice,
        lanePrices,
        laneEnabled,
        t
      ),
    [
      billingExpr,
      laneEnabled,
      lanePrices,
      pricingMode,
      promptPrice,
      requestRuleExpr,
      t,
      watchedValues,
    ]
  )

  const warnings = useMemo(() => {
    const nextWarnings: string[] = []
    const hasConflict =
      !!editData?.price &&
      [
        editData.ratio,
        editData.completionRatio,
        editData.cacheRatio,
        editData.createCacheRatio,
        editData.imageRatio,
        editData.audioRatio,
        editData.audioCompletionRatio,
      ].some(hasValue)

    if (hasConflict) {
      nextWarnings.push(
        t(
          'This model has both fixed-price and token-price settings. Saving the current mode will rewrite the conflicting fields.'
        )
      )
    }

    if (
      pricingMode === 'per-token' &&
      toNumberOrNull(promptPrice) === null &&
      laneConfigs.some(
        ({ key }) => laneEnabled[key] && hasValue(lanePrices[key])
      )
    ) {
      nextWarnings.push(
        t('Input price is required before saving dependent prices.')
      )
    }

    if (
      pricingMode === 'per-token' &&
      laneEnabled.audioOutput &&
      !hasValue(lanePrices.audioInput)
    ) {
      nextWarnings.push(t('Audio output price requires an audio input price.'))
    }

    return nextWarnings
  }, [editData, laneEnabled, lanePrices, pricingMode, promptPrice, t])

  const handleSubmit = (values: ModelPricingFormValues) => {
    if (
      pricingMode === 'per-token' &&
      toNumberOrNull(promptPrice) === null &&
      laneConfigs.some(
        ({ key }) => laneEnabled[key] && hasValue(lanePrices[key])
      )
    ) {
      form.setError('ratio', {
        message: t('Input price is required before saving dependent prices.'),
      })
      return
    }

    if (
      pricingMode === 'per-token' &&
      laneEnabled.audioOutput &&
      !hasValue(lanePrices.audioInput)
    ) {
      form.setError('audioRatio', {
        message: t('Audio output price requires an audio input price.'),
      })
      return
    }

    const fixedPriceUsd = displayPricingValueToUsd(values.price)
    const data: ModelRatioData = {
      name: values.name.trim(),
      billingMode: pricingMode,
      price:
        pricingMode === 'per-request' && fixedPriceUsd !== null
          ? formatPricingNumber(fixedPriceUsd)
          : '',
      ratio: values.ratio || '',
      cacheRatio: values.cacheRatio || '',
      createCacheRatio: values.createCacheRatio || '',
      completionRatio: values.completionRatio || '',
      imageRatio: values.imageRatio || '',
      audioRatio: values.audioRatio || '',
      audioCompletionRatio: values.audioCompletionRatio || '',
      imagePricing: values.imagePricing || '',
      videoPricing: values.videoPricing || '',
      audioInPricing: values.audioInPricing || '',
      audioOutPricing: values.audioOutPricing || '',
    }

    if (pricingMode === 'tiered_expr') {
      data.billingExpr = billingExpr
      data.requestRuleExpr = requestRuleExpr
    }

    onSave(data)
    form.reset()
    onCancel?.()
  }

  const activeName = watchedValues.name || editData?.name || t('New model')

  return (
    <div
      className={cn(
        'bg-background flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl border',
        className
      )}
    >
      <div className='border-b p-4'>
        <div className='flex flex-wrap items-start justify-between gap-3'>
          <div className='min-w-0'>
            <h3 className='truncate text-base font-medium'>
              {isEditMode ? t('Edit model pricing') : t('Add model pricing')}
            </h3>
            <p className='text-muted-foreground truncate text-sm'>
              {activeName}
            </p>
          </div>
          <Badge variant={getModeBadgeVariant(pricingMode)}>
            {t(getModeLabel(pricingMode))}
          </Badge>
        </div>
      </div>

      <Form {...form}>
        <form
          onSubmit={form.handleSubmit(handleSubmit)}
          className='flex min-h-0 flex-1 flex-col'
          autoComplete='off'
        >
          <div className='min-h-0 flex-1 overflow-y-auto p-4'>
            <FieldGroup>
              {warnings.length > 0 && (
                <Alert variant='destructive'>
                  <AlertTriangle data-icon='inline-start' />
                  <AlertDescription>
                    <div className='flex flex-col gap-1'>
                      {warnings.map((warning) => (
                        <span key={warning}>{warning}</span>
                      ))}
                    </div>
                  </AlertDescription>
                </Alert>
              )}

              <FormField
                control={form.control}
                name='name'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Model name')}</FormLabel>
                    <FormControl>
                      <Input
                        placeholder={t('gpt-4')}
                        {...field}
                        disabled={isEditMode}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('The exact model identifier as used in API requests.')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <Tabs value={pricingMode} onValueChange={handleModeChange}>
                <TabsList className='grid w-full grid-cols-4 gap-y-1 md:grid-cols-7'>
                  <TabsTrigger value='per-token'>{t('Per-token')}</TabsTrigger>
                  <TabsTrigger value='per-request'>
                    {t('Per-request')}
                  </TabsTrigger>
                  <TabsTrigger value='tiered_expr'>
                    {t('Expression')}
                  </TabsTrigger>
                  <TabsTrigger value='per-image'>
                    {t('Per-image')}
                  </TabsTrigger>
                  <TabsTrigger value='per-second'>
                    {t('Per-second')}
                  </TabsTrigger>
                  <TabsTrigger value='per-minute'>
                    {t('Per-minute')}
                  </TabsTrigger>
                  <TabsTrigger value='per-1m-chars'>
                    {t('Per-1M chars')}
                  </TabsTrigger>
                </TabsList>

                <TabsContent value='per-token' className='flex flex-col gap-5'>
                  <FieldGroup>
                    <Field>
                      <FieldLabel>{t('Input price')}</FieldLabel>
                      <PriceInput
                        value={promptPrice}
                        placeholder='3'
                        onChange={handlePromptPriceChange}
                      />
                      <FieldDescription>
                        {t(
                          'Price per 1M input tokens in the selected display currency.'
                        )}
                      </FieldDescription>
                    </Field>

                    <div className='grid gap-3 sm:grid-cols-2'>
                      {laneConfigs.map((lane) => {
                        const disabled =
                          lane.key === 'audioOutput' &&
                          (!laneEnabled.audioInput ||
                            !hasValue(lanePrices.audioInput))
                        return (
                          <PriceLane
                            key={lane.key}
                            title={t(lane.titleKey)}
                            description={t(lane.descriptionKey)}
                            placeholder={lane.placeholder}
                            value={lanePrices[lane.key]}
                            enabled={laneEnabled[lane.key]}
                            disabled={disabled}
                            onEnabledChange={(checked) =>
                              handleLaneToggle(lane.key, checked)
                            }
                            onChange={(value) =>
                              handleLanePriceChange(lane.key, value)
                            }
                          />
                        )
                      })}
                    </div>
                  </FieldGroup>
                </TabsContent>

                <TabsContent
                  value='per-request'
                  className='flex flex-col gap-5'
                >
                  <FormField
                    control={form.control}
                    name='price'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Fixed price')}</FormLabel>
                        <FormControl>
                          <InputGroup>
                            <InputGroupAddon>
                              {getModelPricingCurrencyPrefix()}
                            </InputGroupAddon>
                            <InputGroupInput
                              inputMode='decimal'
                              placeholder='0.01'
                              {...field}
                              onChange={(event) => {
                                const value = event.target.value
                                if (numericDraftRegex.test(value)) {
                                  field.onChange(value)
                                }
                              }}
                            />
                            <InputGroupAddon align='inline-end'>
                              {t('per request')}
                            </InputGroupAddon>
                          </InputGroup>
                        </FormControl>
                        <FormDescription>
                          {t(
                            'Cost per request in the selected display currency. It is saved internally as USD.'
                          )}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </TabsContent>

                <TabsContent
                  value='tiered_expr'
                  className='flex flex-col gap-5'
                >
                  <TieredPricingEditor
                    modelName={watchedValues.name}
                    billingExpr={billingExpr}
                    requestRuleExpr={requestRuleExpr}
                    onBillingExprChange={setBillingExpr}
                    onRequestRuleExprChange={setRequestRuleExpr}
                  />
                </TabsContent>

                {/* Structured pricing modes (PR-7c) — inline editors.
                  * Each tab shows the same multi-field editor (base price +
                  * multipliers) that previously required opening the
                  * model-mutate drawer under Models → Metadata → Edit. */}
                {(
                  [
                    'per-image',
                    'per-second',
                    'per-minute',
                    'per-1m-chars',
                  ] as const
                ).map((mode) => (
                  <TabsContent
                    key={mode}
                    value={mode}
                    className='flex flex-col gap-4'
                  >
                    <div className='rounded-md border p-4'>
                      {mode === 'per-image' && (
                        <ImageGenEditor
                          value={imagePricing}
                          onChange={handleImagePricingChange}
                          currencyLabel={pricingCurrencyLabel}
                        />
                      )}
                      {mode === 'per-second' && (
                        <VideoGenEditor
                          value={videoPricing}
                          onChange={handleVideoPricingChange}
                          currencyLabel={pricingCurrencyLabel}
                        />
                      )}
                      {mode === 'per-minute' && (
                        <AudioInEditor
                          value={audioInPricing}
                          onChange={handleAudioInPricingChange}
                          currencyLabel={pricingCurrencyLabel}
                        />
                      )}
                      {mode === 'per-1m-chars' && (
                        <AudioOutEditor
                          value={audioOutPricing}
                          onChange={handleAudioOutPricingChange}
                          currencyLabel={pricingCurrencyLabel}
                        />
                      )}
                    </div>
                  </TabsContent>
                ))}
              </Tabs>

              <Collapsible open={previewOpen} onOpenChange={setPreviewOpen}>
                <CollapsibleTrigger
                  render={
                    <Button
                      type='button'
                      variant='outline'
                      className='flex w-full justify-between'
                    />
                  }
                >
                  <span>{t('Save preview')}</span>
                  <ChevronDown
                    className={cn(
                      'transition-transform',
                      previewOpen && 'rotate-180'
                    )}
                  />
                </CollapsibleTrigger>
                <CollapsibleContent className='pt-3'>
                  <div className='rounded-lg border'>
                    {previewRows.map((row) => (
                      <div
                        key={row.key}
                        className='grid grid-cols-[140px_1fr] gap-3 border-b px-3 py-2 text-sm last:border-b-0'
                      >
                        <span className='text-muted-foreground text-xs'>
                          {row.label}
                        </span>
                        <span
                          className={cn(
                            'min-w-0',
                            row.multiline
                              ? 'font-mono text-xs leading-5 break-words whitespace-pre-wrap'
                              : 'truncate'
                          )}
                        >
                          {row.value}
                        </span>
                      </div>
                    ))}
                  </div>
                </CollapsibleContent>
              </Collapsible>
            </FieldGroup>
          </div>

          <SheetFooter
            className={sideDrawerFooterClassName(
              'grid-cols-1 sm:items-center sm:justify-between'
            )}
          >
            <div className='text-muted-foreground text-xs'>
              {selectedTargetCount > 0
                ? t('{{count}} selected targets available for bulk copy.', {
                    count: selectedTargetCount,
                  })
                : t('Changes are written to the settings draft on save.')}
            </div>
            <div className='flex justify-end gap-2'>
              <Button type='button' variant='outline' onClick={onCancel}>
                {t('Cancel')}
              </Button>
              <Button type='submit'>
                {isEditMode ? t('Update') : t('Add')}
              </Button>
            </div>
          </SheetFooter>
        </form>
      </Form>
    </div>
  )
}

function PriceInput(props: {
  value: string
  placeholder?: string
  disabled?: boolean
  onChange: (value: string) => void
}) {
  const prefix = getModelPricingCurrencyPrefix()
  const unitLabel = getModelPricingUnitLabel()
  return (
    <InputGroup>
      <InputGroupAddon>{prefix}</InputGroupAddon>
      <InputGroupInput
        inputMode='decimal'
        value={props.value}
        placeholder={props.placeholder}
        disabled={props.disabled}
        onChange={(event) => props.onChange(event.target.value)}
      />
      <InputGroupAddon align='inline-end'>{unitLabel}</InputGroupAddon>
    </InputGroup>
  )
}

function PriceLane(props: {
  title: string
  description: string
  placeholder: string
  value: string
  enabled: boolean
  disabled?: boolean
  onEnabledChange: (checked: boolean) => void
  onChange: (value: string) => void
}) {
  const { t } = useTranslation()
  const effectiveDisabled = props.disabled || !props.enabled

  return (
    <SettingsControlGroup
      className={cn('space-y-3', effectiveDisabled && 'opacity-75')}
      data-disabled={effectiveDisabled || undefined}
    >
      <SettingsSwitchField
        checked={props.enabled}
        disabled={props.disabled}
        onCheckedChange={props.onEnabledChange}
        label={props.title}
        description={props.description}
        aria-label={props.title}
      />
      <PriceInput
        value={props.value}
        placeholder={props.placeholder}
        disabled={effectiveDisabled}
        onChange={props.onChange}
      />
      <p className='text-muted-foreground text-xs'>
        {props.enabled
          ? t('Price per 1M tokens in the selected display currency.')
          : t('Disabled lanes are omitted on save.')}
      </p>
    </SettingsControlGroup>
  )
}
