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
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { ChevronDown, Loader2 } from 'lucide-react'
import { useEffect, useRef, useState, useCallback, useMemo } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

import {
  SideDrawerSection,
  sideDrawerContentClassName,
  sideDrawerFooterClassName,
  sideDrawerFormClassName,
  sideDrawerHeaderClassName,
  sideDrawerSwitchItemClassName,
} from '@/components/drawer-layout'
import { IconSelector } from '@/components/icon-selector'
import { JsonEditor } from '@/components/json-editor'
import { TagInput } from '@/components/tag-input'
import { Button } from '@/components/ui/button'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
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
import { Label } from '@/components/ui/label'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import {
  useSystemOptions,
  getOptionValue,
} from '@/features/system-settings/hooks/use-system-options'
import { useUpdateOption } from '@/features/system-settings/hooks/use-update-option'
import { normalizeJsonString } from '@/features/system-settings/models/utils'
import type { ModelSettings } from '@/features/system-settings/types'
import { safeJsonParse } from '@/features/system-settings/utils/json-parser'
import { useSystemConfig } from '@/hooks/use-system-config'
import {
  convertBillingDisplayToUSD,
  convertUSDToBillingDisplay,
  formatBillingCurrencyFromUSD,
} from '@/lib/currency'
import { getLobeIcon } from '@/lib/lobe-icon'
import { cn } from '@/lib/utils'

import {
  createModel,
  updateModel,
  getModel,
  getVendors,
  getModels,
} from '../../api'
import {
  getNameRuleOptions,
  ENDPOINT_TEMPLATES,
  TAG_PRESETS,
  getTagLabel,
  getTagCategoryLabel,
  CAPABILITY_ENDPOINT_HINTS,
} from '../../constants'
import { modelsQueryKeys, vendorsQueryKeys, parseModelTags } from '../../lib'
import {
  getCompletionPriceLabelKey,
  getFixedPriceLabelKey,
  getPricingCurrencyLabel,
  getPricingModeLabelKey,
  getPromptPriceLabelKey,
} from '../../lib/pricing-currency-label'
import { inferVendorName } from '../../lib/vendor-inference'
import type { Model } from '../../types'
import { OfficialPricingReference } from '../pricing/official-pricing-reference'

// Extended schema for ratio configuration (internal form state only)
const extendedModelFormSchema = z.object({
  id: z.number().optional(),
  model_name: z.string().min(1, 'Model name is required'),
  description: z.string(),
  icon: z.string(),
  tags: z.array(z.string()),
  vendor_id: z.number().optional(),
  endpoints: z.string(),
  name_rule: z.number(),
  status: z.boolean(),
  sync_official: z.boolean(),
  context_length: z.number().optional(),
  price: z.string().optional(),
  ratio: z.string().optional(),
  cacheRatio: z.string().optional(),
  completionRatio: z.string().optional(),
  imageRatio: z.string().optional(),
  audioRatio: z.string().optional(),
  audioCompletionRatio: z.string().optional(),
})

type ExtendedModelFormValues = z.infer<typeof extendedModelFormSchema>

type PricingMode = 'per-token' | 'per-request'
type PricingSubMode = 'ratio' | 'price'

type ModelMutateDrawerProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  currentRow?: Model | null
}

export function ModelMutateDrawer({
  open,
  onOpenChange,
  currentRow,
}: ModelMutateDrawerProps) {
  const { t } = useTranslation()
  const { currency } = useSystemConfig()
  const queryClient = useQueryClient()
  const isEditing = Boolean(currentRow?.id)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [pricingMode, setPricingMode] = useState<PricingMode>('per-token')
  const [pricingSubMode, setPricingSubMode] = useState<PricingSubMode>('ratio')
  const [advancedOpen, setAdvancedOpen] = useState(false)
  const [promptPrice, setPromptPrice] = useState('')
  const [completionPrice, setCompletionPrice] = useState('')
  const [oldModelName, setOldModelName] = useState<string>('')

  // Fetch vendors for dropdown
  const { data: vendorsData } = useQuery({
    queryKey: vendorsQueryKeys.list(),
    queryFn: () => getVendors({ page_size: 1000 }),
    enabled: open,
  })

  const vendors = vendorsData?.data?.items || []

  // Fetch models for vendor statistics
  const { data: allModelsData } = useQuery({
    queryKey: modelsQueryKeys.list({}),
    queryFn: () => getModels({ page_size: 10000 }),
    enabled: open,
  })

  const allModels = allModelsData?.data?.items || []

  // Fetch model detail if editing
  const { data: modelData } = useQuery({
    queryKey: modelsQueryKeys.detail(currentRow?.id || 0),
    queryFn: () => getModel(currentRow!.id),
    enabled: open && isEditing,
  })

  // Fetch system options for ratio configuration
  const { data: systemOptionsData } = useSystemOptions()

  const updateOption = useUpdateOption()

  // Get model settings from system options
  const modelSettings = useMemo(() => {
    if (!systemOptionsData?.data) return null
    const defaultModelSettings: ModelSettings = {
      'global.pass_through_request_enabled': false,
      'global.thinking_model_blacklist': '[]',
      'global.chat_completions_to_responses_policy': '{}',
      'general_setting.ping_interval_enabled': false,
      'general_setting.ping_interval_seconds': 60,
      'gemini.safety_settings': '',
      'gemini.version_settings': '',
      'gemini.supported_imagine_models': '',
      'gemini.thinking_adapter_enabled': false,
      'gemini.thinking_adapter_budget_tokens_percentage': 0.6,
      'gemini.function_call_thought_signature_enabled': false,
      'gemini.remove_function_response_id_enabled': true,
      'claude.model_headers_settings': '',
      'claude.default_max_tokens': '',
      'claude.thinking_adapter_enabled': true,
      'claude.thinking_adapter_budget_tokens_percentage': 0.8,
      ModelPrice: '',
      ModelRatio: '',
      CacheRatio: '',
      CompletionRatio: '',
      ImageRatio: '',
      AudioRatio: '',
      AudioCompletionRatio: '',
      ExposeRatioEnabled: false,
      'billing_setting.billing_mode': '{}',
      'billing_setting.billing_expr': '{}',
      'tool_price_setting.prices': '{}',
      TopupGroupRatio: '',
      GroupRatio: '',
      UserUsableGroups: '',
      GroupGroupRatio: '',
      AutoGroups: '',
      DefaultUseAutoGroup: false,
      CreateCacheRatio: '',
      'group_ratio_setting.group_special_usable_group': '{}',
      'grok.violation_deduction_enabled': false,
      'grok.violation_deduction_amount': 0,
      'channel_affinity_setting.enabled': false,
      'channel_affinity_setting.switch_on_success': true,
      'channel_affinity_setting.max_entries': 100000,
      'channel_affinity_setting.default_ttl_seconds': 3600,
      'channel_affinity_setting.rules': '[]',
      'model_deployment.ionet.api_key': '',
      'model_deployment.ionet.enabled': false,
    }
    return getOptionValue(systemOptionsData.data, defaultModelSettings)
  }, [systemOptionsData])

  const form = useForm<ExtendedModelFormValues>({
    resolver: zodResolver(extendedModelFormSchema),
    defaultValues: {
      model_name: '',
      description: '',
      icon: '',
      tags: [],
      vendor_id: undefined,
      endpoints: '',
      name_rule: 0,
      status: true,
      sync_official: true,
      context_length: undefined,
      price: '',
      ratio: '',
      cacheRatio: '',
      completionRatio: '',
      imageRatio: '',
      audioRatio: '',
      audioCompletionRatio: '',
    },
  })

  // 自动匹配的 vendor ID：图标优先，然后按模型名称推断。
  // 与后端 constant.LookupVendorByIcon / InferVendorNameByModelName 保持一致。
  const iconValue = form.watch('icon')
  const modelNameValue = form.watch('model_name')
  const autoMatchedVendorId = useMemo(() => {
    if (vendors.length === 0) return undefined
    const inferredName = inferVendorName({
      icon: iconValue,
      modelName: modelNameValue,
    })
    if (!inferredName) return undefined
    const match = vendors.find(
      (v) => v.name.toLowerCase() === inferredName.toLowerCase()
    )
    return match?.id
  }, [iconValue, modelNameValue, vendors])

  // 当图标或模型名变化导致自动匹配结果变化时，自动写入 vendor_id。
  // 用户仍可通过下拉框手动改回；只在自动匹配结果切换时更新，避免抹掉手动选择。
  const lastAutoMatchRef = useRef<number | undefined>(undefined)
  useEffect(() => {
    if (!open) return
    const current = form.getValues('vendor_id')
    // 首次进入编辑：只有当模型此前没有 vendor 或已知匹配一致时才自动填
    if (autoMatchedVendorId === undefined) {
      lastAutoMatchRef.current = undefined
      return
    }
    // 首次自动填：当前无值 → 填入
    if (!current) {
      form.setValue('vendor_id', autoMatchedVendorId, { shouldDirty: true })
      lastAutoMatchRef.current = autoMatchedVendorId
      return
    }
    // 自动匹配结果变化（如换了图标）且当前值等于上一次自动匹配值 → 跟随更新
    if (
      current === lastAutoMatchRef.current &&
      lastAutoMatchRef.current !== autoMatchedVendorId
    ) {
      form.setValue('vendor_id', autoMatchedVendorId, { shouldDirty: true })
      lastAutoMatchRef.current = autoMatchedVendorId
    }
  }, [autoMatchedVendorId, open, form])

  // Sort vendors by usage and display name
  const sortedVendors = useMemo(() => {
    // Calculate vendor usage
    const vendorUsage: Record<number, number> = {}
    for (const model of allModels) {
      if (model.vendor_id) {
        vendorUsage[model.vendor_id] = (vendorUsage[model.vendor_id] || 0) + 1
      }
    }

    // Sort vendors with usage and then alphabetically
    return [...vendors].sort((a, b) => {
      const usageA = vendorUsage[a.id] || 0
      const usageB = vendorUsage[b.id] || 0

      if (usageA !== usageB) {
        return usageB - usageA
      }

      const nameA = a.display_name || a.name
      const nameB = b.display_name || b.name
      return nameA.localeCompare(nameB)
    })
  }, [vendors, allModels])

  const validateNumber = (value: string) => {
    if (value === '') return true
    return !isNaN(parseFloat(value))
  }

  const pricingDisplayType = currency?.quotaDisplayType || 'USD'
  const pricingCurrencyLabel = getPricingCurrencyLabel(
    pricingDisplayType,
    currency?.customCurrencySymbol
  )

  const formatModelPriceRule = (amountUSD: number) =>
    t('Billing rule: {{price}} / 1M tokens', {
      price: formatBillingCurrencyFromUSD(amountUSD, {
        abbreviate: false,
        digitsLarge: 4,
        digitsSmall: 4,
      }),
    })

  const formatRequestPriceRule = (amountUSD: number) =>
    t('Billing rule: {{price}} / request', {
      price: formatBillingCurrencyFromUSD(amountUSD, {
        abbreviate: false,
        digitsLarge: 4,
        digitsSmall: 4,
      }),
    })

  const handlePromptPriceChange = (value: string) => {
    setPromptPrice(value)
    if (value && !isNaN(parseFloat(value))) {
      // Input arrives in the currently displayed currency; convert to base USD
      // before deriving the ratio so `ModelRatio` stays USD-anchored, matching
      // commission redeem (service/commission/redeem.go) and topup accounting.
      const usdPrice = convertBillingDisplayToUSD(parseFloat(value))
      const ratio = usdPrice / 2
      form.setValue('ratio', ratio.toString())
    } else {
      form.setValue('ratio', '')
    }
  }

  const handleCompletionPriceChange = (value: string) => {
    setCompletionPrice(value)
    if (
      value &&
      !isNaN(parseFloat(value)) &&
      promptPrice &&
      !isNaN(parseFloat(promptPrice)) &&
      parseFloat(promptPrice) > 0
    ) {
      // completion / prompt is a dimensionless ratio — currency-independent.
      const completionRatio = parseFloat(value) / parseFloat(promptPrice)
      form.setValue('completionRatio', completionRatio.toString())
    } else {
      form.setValue('completionRatio', '')
    }
  }

  /**
   * Apply official reference-price values into the form. Payload fields are
   * USD-anchored ratios (or a USD per-request price); the form stores them
   * verbatim, and the display-currency mirror state gets refreshed so the
   * price-mode inputs update alongside.
   */
  const applyOfficialPricing = useCallback(
    (payload: {
      ratio?: number
      completionRatio?: number
      cacheRatio?: number
      imageRatio?: number
      audioRatio?: number
      audioCompletionRatio?: number
      modelPrice?: number
    }) => {
      if (payload.modelPrice !== undefined) {
        // Per-request: show the value in the current display currency,
        // mirroring what a manual entry would look like.
        const displayPrice = convertUSDToBillingDisplay(payload.modelPrice)
        form.setValue('price', displayPrice.toString())
        return
      }
      if (payload.ratio !== undefined) {
        form.setValue('ratio', payload.ratio.toString())
        const inputPriceDisplay = convertUSDToBillingDisplay(payload.ratio * 2)
        setPromptPrice(inputPriceDisplay.toString())
        if (payload.completionRatio !== undefined) {
          form.setValue('completionRatio', payload.completionRatio.toString())
          setCompletionPrice(
            (inputPriceDisplay * payload.completionRatio).toString()
          )
        } else {
          form.setValue('completionRatio', '')
          setCompletionPrice('')
        }
      }
      const setOrClear = (
        name:
          | 'cacheRatio'
          | 'imageRatio'
          | 'audioRatio'
          | 'audioCompletionRatio',
        val: number | undefined
      ) => {
        form.setValue(name, val !== undefined ? val.toString() : '')
      }
      setOrClear('cacheRatio', payload.cacheRatio)
      setOrClear('imageRatio', payload.imageRatio)
      setOrClear('audioRatio', payload.audioRatio)
      setOrClear('audioCompletionRatio', payload.audioCompletionRatio)
      if (
        payload.cacheRatio !== undefined ||
        payload.imageRatio !== undefined ||
        payload.audioRatio !== undefined ||
        payload.audioCompletionRatio !== undefined
      ) {
        setAdvancedOpen(true)
      }
    },
    [form]
  )

  // Load model data for editing and ratio configuration
  useEffect(() => {
    if (open && isEditing && modelData?.data) {
      const model = modelData.data
      setOldModelName(model.model_name)

      // Base model data reset
      const baseModelData = {
        id: model.id,
        model_name: model.model_name,
        description: model.description || '',
        icon: model.icon || '',
        tags: parseModelTags(model.tags),
        vendor_id: model.vendor_id,
        endpoints: model.endpoints || '',
        name_rule: model.name_rule || 0,
        status: model.status === 1,
        sync_official: model.sync_official === 1,
        context_length: model.context_length,
        price: '',
        ratio: '',
        cacheRatio: '',
        completionRatio: '',
        imageRatio: '',
        audioRatio: '',
        audioCompletionRatio: '',
      }

      // Parse ratio configurations from system settings if available
      if (modelSettings) {
        const priceMap = safeJsonParse<Record<string, number>>(
          modelSettings.ModelPrice,
          { fallback: {}, silent: true }
        )
        const ratioMap = safeJsonParse<Record<string, number>>(
          modelSettings.ModelRatio,
          { fallback: {}, silent: true }
        )
        const cacheMap = safeJsonParse<Record<string, number>>(
          modelSettings.CacheRatio,
          { fallback: {}, silent: true }
        )
        const completionMap = safeJsonParse<Record<string, number>>(
          modelSettings.CompletionRatio,
          { fallback: {}, silent: true }
        )
        const imageMap = safeJsonParse<Record<string, number>>(
          modelSettings.ImageRatio,
          { fallback: {}, silent: true }
        )
        const audioMap = safeJsonParse<Record<string, number>>(
          modelSettings.AudioRatio,
          { fallback: {}, silent: true }
        )
        const audioCompletionMap = safeJsonParse<Record<string, number>>(
          modelSettings.AudioCompletionRatio,
          { fallback: {}, silent: true }
        )

        // Extract ratio config for this model
        const modelName = model.model_name
        const price = priceMap[modelName]
        const ratio = ratioMap[modelName]
        const cacheRatio = cacheMap[modelName]
        const completionRatio = completionMap[modelName]
        const imageRatio = imageMap[modelName]
        const audioRatio = audioMap[modelName]
        const audioCompletionRatio = audioCompletionMap[modelName]

        // Determine pricing mode
        if (price !== undefined && price !== null) {
          setPricingMode('per-request')
          // `price` in DB is base USD/request. When admin views in CNY/CUSTOM,
          // prefill the input with the display-currency equivalent so the
          // number they see matches what they'd type in.
          const displayPrice = convertUSDToBillingDisplay(price)
          form.reset({
            ...baseModelData,
            price: displayPrice.toString(),
          })
        } else {
          setPricingMode('per-token')
          if (ratio !== undefined && ratio !== null) {
            // ratio is USD-anchored (1 unit = $2 / 1M tokens). Convert to the
            // current display currency for the visible price input.
            const tokenPriceUSD = ratio * 2
            const tokenPriceDisplay = convertUSDToBillingDisplay(tokenPriceUSD)
            setPromptPrice(tokenPriceDisplay.toString())
            if (completionRatio !== undefined && completionRatio !== null) {
              const compPriceDisplay = tokenPriceDisplay * completionRatio
              setCompletionPrice(compPriceDisplay.toString())
            }
          }
          form.reset({
            ...baseModelData,
            ratio: ratio?.toString() || '',
            cacheRatio: cacheRatio?.toString() || '',
            completionRatio: completionRatio?.toString() || '',
            imageRatio: imageRatio?.toString() || '',
            audioRatio: audioRatio?.toString() || '',
            audioCompletionRatio: audioCompletionRatio?.toString() || '',
          })
          setAdvancedOpen(
            !!(cacheRatio || imageRatio || audioRatio || audioCompletionRatio)
          )
        }
      } else {
        // If system settings not loaded yet, just load base model data
        setPricingMode('per-token')
        form.reset(baseModelData)
        setAdvancedOpen(false)
      }
    } else if (open && !isEditing) {
      // Pre-fill model name if passed from missing models
      setOldModelName('')
      setPricingMode('per-token')
      setPricingSubMode('ratio')
      setPromptPrice('')
      setCompletionPrice('')
      setAdvancedOpen(false)
      form.reset({
        model_name: currentRow?.model_name || '',
        description: '',
        icon: '',
        tags: [],
        vendor_id: undefined,
        endpoints: '',
        name_rule: 0,
        status: true,
        sync_official: true,
        context_length: undefined,
        price: '',
        ratio: '',
        cacheRatio: '',
        completionRatio: '',
        imageRatio: '',
        audioRatio: '',
        audioCompletionRatio: '',
      })
    }
  }, [open, isEditing, modelData, currentRow, form, modelSettings])

  const onSubmit = useCallback(
    async (values: ExtendedModelFormValues): Promise<void> => {
      setIsSubmitting(true)
      try {
        const submitData = {
          ...values,
          id: isEditing ? currentRow!.id : undefined,
          tags: Array.isArray(values.tags) ? values.tags.join(',') : '',
          status: values.status ? 1 : 0,
          sync_official: values.sync_official ? 1 : 0,
        }

        // Remove ratio fields from model data (they're stored in system settings)
        const {
          price,
          ratio,
          cacheRatio,
          completionRatio,
          imageRatio,
          audioRatio,
          audioCompletionRatio,
          ...modelData
        } = submitData

        const response = isEditing
          ? await updateModel({ ...modelData, id: currentRow!.id })
          : await createModel(modelData)

        if (response.success) {
          // Handle ratio configuration updates in system settings
          const finalModelName = values.model_name
          const hasRatioConfig =
            (pricingMode === 'per-request' &&
              values.price &&
              values.price !== '') ||
            (pricingMode === 'per-token' &&
              (values.ratio ||
                values.cacheRatio ||
                values.completionRatio ||
                values.imageRatio ||
                values.audioRatio ||
                values.audioCompletionRatio))

          // Always process system settings updates if we have modelSettings
          if (modelSettings) {
            const priceMap = safeJsonParse<Record<string, number>>(
              modelSettings.ModelPrice,
              { fallback: {}, silent: true }
            )
            const ratioMap = safeJsonParse<Record<string, number>>(
              modelSettings.ModelRatio,
              { fallback: {}, silent: true }
            )
            const cacheMap = safeJsonParse<Record<string, number>>(
              modelSettings.CacheRatio,
              { fallback: {}, silent: true }
            )
            const completionMap = safeJsonParse<Record<string, number>>(
              modelSettings.CompletionRatio,
              { fallback: {}, silent: true }
            )
            const imageMap = safeJsonParse<Record<string, number>>(
              modelSettings.ImageRatio,
              { fallback: {}, silent: true }
            )
            const audioMap = safeJsonParse<Record<string, number>>(
              modelSettings.AudioRatio,
              { fallback: {}, silent: true }
            )
            const audioCompletionMap = safeJsonParse<Record<string, number>>(
              modelSettings.AudioCompletionRatio,
              { fallback: {}, silent: true }
            )

            if (isEditing && oldModelName && oldModelName !== finalModelName) {
              delete priceMap[oldModelName]
              delete ratioMap[oldModelName]
              delete cacheMap[oldModelName]
              delete completionMap[oldModelName]
              delete imageMap[oldModelName]
              delete audioMap[oldModelName]
              delete audioCompletionMap[oldModelName]
            }

            delete priceMap[finalModelName]
            delete ratioMap[finalModelName]
            delete cacheMap[finalModelName]
            delete completionMap[finalModelName]
            delete imageMap[finalModelName]
            delete audioMap[finalModelName]
            delete audioCompletionMap[finalModelName]

            if (hasRatioConfig) {
              if (
                pricingMode === 'per-request' &&
                values.price &&
                values.price !== ''
              ) {
                // Per-request price is entered in the display currency; store
                // as base USD so it stays consistent with commission redemption
                // and topup accounting (both anchored to USDExchangeRate).
                priceMap[finalModelName] = convertBillingDisplayToUSD(
                  parseFloat(values.price)
                )
              } else if (pricingMode === 'per-token') {
                if (values.ratio && values.ratio !== '') {
                  ratioMap[finalModelName] = parseFloat(values.ratio)
                }
                if (values.cacheRatio && values.cacheRatio !== '') {
                  cacheMap[finalModelName] = parseFloat(values.cacheRatio)
                }
                if (values.completionRatio && values.completionRatio !== '') {
                  completionMap[finalModelName] = parseFloat(
                    values.completionRatio
                  )
                }
                if (values.imageRatio && values.imageRatio !== '') {
                  imageMap[finalModelName] = parseFloat(values.imageRatio)
                }
                if (values.audioRatio && values.audioRatio !== '') {
                  audioMap[finalModelName] = parseFloat(values.audioRatio)
                }
                if (
                  values.audioCompletionRatio &&
                  values.audioCompletionRatio !== ''
                ) {
                  audioCompletionMap[finalModelName] = parseFloat(
                    values.audioCompletionRatio
                  )
                }
              }
            }

            const updates: Array<{ key: string; value: string }> = []

            const newModelPrice = normalizeJsonString(JSON.stringify(priceMap))
            if (
              newModelPrice !== normalizeJsonString(modelSettings.ModelPrice)
            ) {
              updates.push({ key: 'ModelPrice', value: newModelPrice })
            }

            const newModelRatio = normalizeJsonString(JSON.stringify(ratioMap))
            if (
              newModelRatio !== normalizeJsonString(modelSettings.ModelRatio)
            ) {
              updates.push({ key: 'ModelRatio', value: newModelRatio })
            }

            const newCacheRatio = normalizeJsonString(JSON.stringify(cacheMap))
            if (
              newCacheRatio !== normalizeJsonString(modelSettings.CacheRatio)
            ) {
              updates.push({ key: 'CacheRatio', value: newCacheRatio })
            }

            const newCompletionRatio = normalizeJsonString(
              JSON.stringify(completionMap)
            )
            if (
              newCompletionRatio !==
              normalizeJsonString(modelSettings.CompletionRatio)
            ) {
              updates.push({
                key: 'CompletionRatio',
                value: newCompletionRatio,
              })
            }

            const newImageRatio = normalizeJsonString(JSON.stringify(imageMap))
            if (
              newImageRatio !== normalizeJsonString(modelSettings.ImageRatio)
            ) {
              updates.push({ key: 'ImageRatio', value: newImageRatio })
            }

            const newAudioRatio = normalizeJsonString(JSON.stringify(audioMap))
            if (
              newAudioRatio !== normalizeJsonString(modelSettings.AudioRatio)
            ) {
              updates.push({ key: 'AudioRatio', value: newAudioRatio })
            }

            const newAudioCompletionRatio = normalizeJsonString(
              JSON.stringify(audioCompletionMap)
            )
            if (
              newAudioCompletionRatio !==
              normalizeJsonString(modelSettings.AudioCompletionRatio)
            ) {
              updates.push({
                key: 'AudioCompletionRatio',
                value: newAudioCompletionRatio,
              })
            }

            for (const update of updates) {
              await updateOption.mutateAsync(update)
            }
          }

          toast.success(
            isEditing
              ? 'Model updated successfully'
              : 'Model created successfully'
          )
          queryClient.invalidateQueries({ queryKey: modelsQueryKeys.lists() })
          queryClient.invalidateQueries({ queryKey: ['system-options'] })
          onOpenChange(false)
        } else {
          toast.error(response.message || 'Operation failed')
        }
      } catch (error: unknown) {
        toast.error((error as Error)?.message || 'Operation failed')
      } finally {
        setIsSubmitting(false)
      }
    },
    [
      isEditing,
      currentRow,
      queryClient,
      onOpenChange,
      pricingMode,
      oldModelName,
      modelSettings,
      updateOption,
    ]
  )

  const handleFillEndpointTemplate = (templateKey: string) => {
    const template = ENDPOINT_TEMPLATES[templateKey]
    if (template) {
      let newConfig: Record<string, unknown> = { [templateKey]: template }
      try {
        const currentEndpoints = form.getValues('endpoints')
        if (currentEndpoints && currentEndpoints.trim()) {
          const existingConfig = JSON.parse(currentEndpoints)
          if (typeof existingConfig === 'object' && existingConfig !== null) {
            newConfig = { ...existingConfig, [templateKey]: template }
          }
        }
      } catch {
        // If parsing fails, just use the new template
      }
      const templateJson = JSON.stringify(newConfig, null, 2)
      form.setValue('endpoints', templateJson)
    }
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className={sideDrawerContentClassName('sm:max-w-2xl')}>
        <SheetHeader className={sideDrawerHeaderClassName()}>
          <SheetTitle>
            {isEditing ? t('Edit Model') : t('Create Model')}
          </SheetTitle>
          <SheetDescription>
            {isEditing
              ? t("Update model configuration and click save when you're done.")
              : t(
                  'Add a new model to the system by providing the necessary information.'
                )}
          </SheetDescription>
        </SheetHeader>

        <Form {...form}>
          <form
            id='model-form'
            onSubmit={form.handleSubmit(
              onSubmit as Parameters<typeof form.handleSubmit>[0]
            )}
            className={sideDrawerFormClassName()}
          >
            <SideDrawerSection>
              <h3 className='text-sm font-semibold'>
                {t('Basic Information')}
              </h3>

              <FormField
                control={form.control}
                name='model_name'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Model Name *')}</FormLabel>
                    <FormControl>
                      <Input
                        placeholder={t('gpt-4, claude-3-opus, etc.')}
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('The unique identifier for this model')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='description'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Description')}</FormLabel>
                    <FormControl>
                      <Textarea
                        placeholder={t('Describe this model...')}
                        rows={3}
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='icon'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Icon')}</FormLabel>
                    <FormControl>
                      <IconSelector
                        value={field.value}
                        onChange={field.onChange}
                        placeholder={t('Select or upload icon')}
                      />
                    </FormControl>
                    <FormDescription className='text-xs'>
                      {t('Select from @lobehub/icons or upload custom')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='vendor_id'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel className='flex items-center gap-2'>
                      {t('Vendor')}
                      {autoMatchedVendorId &&
                        field.value === autoMatchedVendorId && (
                          <span className='text-primary text-xs font-medium'>
                            {t('Auto-matched')}
                          </span>
                        )}
                    </FormLabel>
                    <Select
                      items={[
                        ...sortedVendors.map((vendor) => ({
                          value: String(vendor.id),
                          label: vendor.display_name || vendor.name,
                        })),
                      ]}
                      onValueChange={(value) =>
                        field.onChange(value ? parseInt(value) : undefined)
                      }
                      value={field.value ? String(field.value) : undefined}
                    >
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue placeholder={t('Select vendor')}>
                            {(() => {
                              const selected = sortedVendors.find(
                                (v) => v.id === field.value
                              )
                              if (!selected) return null
                              return (
                                <span className='flex items-center gap-2'>
                                  {selected.icon && (
                                    <span className='flex items-center'>
                                      {getLobeIcon(selected.icon, 14)}
                                    </span>
                                  )}
                                  <span>
                                    {selected.display_name || selected.name}
                                  </span>
                                </span>
                              )
                            })()}
                          </SelectValue>
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent alignItemWithTrigger={false}>
                        <SelectGroup>
                          {sortedVendors.map((vendor) => {
                            const isRecommended =
                              !field.value && vendor.id === autoMatchedVendorId
                            const modelCount = allModels.filter(
                              (model) => model.vendor_id === vendor.id
                            ).length
                            return (
                              <SelectItem
                                key={vendor.id}
                                value={String(vendor.id)}
                              >
                                <div className='flex items-center gap-2'>
                                  {vendor.icon && (
                                    <span className='flex items-center'>
                                      {getLobeIcon(vendor.icon, 14)}
                                    </span>
                                  )}
                                  <div className='flex flex-col items-start'>
                                    <div className='flex items-center gap-1'>
                                      {vendor.display_name || vendor.name}
                                      {isRecommended && (
                                        <span className='text-primary text-xs font-medium'>
                                          ★
                                        </span>
                                      )}
                                    </div>
                                    {(vendor.display_name ||
                                      modelCount > 0) && (
                                      <div className='flex items-center gap-1'>
                                        {vendor.display_name && (
                                          <span className='text-muted-foreground text-xs'>
                                            {vendor.name}
                                          </span>
                                        )}
                                        {modelCount > 0 && (
                                          <span className='text-muted-foreground text-xs'>
                                            ({modelCount} {t('models')})
                                          </span>
                                        )}
                                      </div>
                                    )}
                                  </div>
                                </div>
                              </SelectItem>
                            )
                          })}
                        </SelectGroup>
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='tags'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Tags')}</FormLabel>
                    <FormControl>
                      <TagInput
                        value={field.value || []}
                        onChange={field.onChange}
                        placeholder={t('Add tags...')}
                      />
                    </FormControl>
                    <div className='mt-2 space-y-2'>
                      {TAG_PRESETS.map((category) => (
                        <div key={category.category} className='space-y-1'>
                          <p className='text-muted-foreground text-xs font-medium'>
                            {getTagCategoryLabel(t, category.category)}
                          </p>
                          <div className='flex flex-wrap gap-1'>
                            {category.tags.map((tag) => {
                              const isSelected = field.value?.includes(tag)
                              return (
                                <button
                                  key={tag}
                                  type='button'
                                  onClick={() => {
                                    const currentTags = field.value || []
                                    const newTags = isSelected
                                      ? currentTags.filter((t) => t !== tag)
                                      : [...currentTags, tag]
                                    field.onChange(newTags)
                                  }}
                                  className={cn(
                                    'rounded-full border px-2 py-0.5 text-xs transition-colors',
                                    isSelected
                                      ? 'border-primary bg-primary/10 text-primary'
                                      : 'border-border text-muted-foreground hover:border-border hover:text-foreground'
                                  )}
                                >
                                  {getTagLabel(t, tag)}
                                </button>
                              )
                            })}
                          </div>
                        </div>
                      ))}
                    </div>
                    <FormDescription className='mt-2'>
                      {t(
                        'Press Enter or comma to add tags, or click tags below to add'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='context_length'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Context window')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        placeholder='128000'
                        value={field.value ?? ''}
                        onChange={(e) => {
                          const v = e.target.value
                          field.onChange(v === '' ? undefined : Number(v))
                        }}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Maximum context tokens supported by this model (e.g. 128000).'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </SideDrawerSection>

            <SideDrawerSection>
              <h3 className='text-sm font-semibold'>{t('Matching Rules')}</h3>

              <FormField
                control={form.control}
                name='name_rule'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Name Rule')}</FormLabel>
                    <FormControl>
                      <RadioGroup
                        onValueChange={(value) =>
                          field.onChange(parseInt(value))
                        }
                        value={String(field.value)}
                        className='grid grid-cols-2 gap-4'
                      >
                        {getNameRuleOptions(t).map((option) => (
                          <div
                            key={option.value}
                            className='flex items-center space-x-2'
                          >
                            <RadioGroupItem
                              value={String(option.value)}
                              id={`rule-${option.value}`}
                            />
                            <Label
                              htmlFor={`rule-${option.value}`}
                              className='cursor-pointer font-normal'
                            >
                              {option.label}
                            </Label>
                          </div>
                        ))}
                      </RadioGroup>
                    </FormControl>
                    <FormDescription>
                      {t('How this model name should match requests')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </SideDrawerSection>

            <SideDrawerSection>
              <div className='flex items-center justify-between'>
                <h3 className='text-sm font-semibold'>{t('Endpoints')}</h3>
                <Select<string>
                  items={[
                    ...Object.entries(ENDPOINT_TEMPLATES).map(
                      ([key, _template]) => ({
                        value: key,
                        label: key,
                      })
                    ),
                  ]}
                  onValueChange={(value) =>
                    value !== null && handleFillEndpointTemplate(value)
                  }
                >
                  <SelectTrigger size='sm' className='w-[200px]'>
                    <SelectValue placeholder={t('Load template...')} />
                  </SelectTrigger>
                  <SelectContent alignItemWithTrigger={false}>
                    <SelectGroup>
                      {Object.entries(ENDPOINT_TEMPLATES).map(
                        ([key, template]) => (
                          <SelectItem key={key} value={key}>
                            <div className='flex flex-col'>
                              <span>{key}</span>
                              {template.description && (
                                <span className='text-muted-foreground text-xs'>
                                  {template.description}
                                </span>
                              )}
                            </div>
                          </SelectItem>
                        )
                      )}
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </div>

              <FormField
                control={form.control}
                name='endpoints'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Endpoint Configuration')}</FormLabel>
                    <FormControl>
                      <JsonEditor
                        value={field.value || ''}
                        onChange={field.onChange}
                        keyPlaceholder='endpoint_type (e.g., chat, completions)'
                        valuePlaceholder='{"path": "/v1/...", "method": "POST"}'
                        keyLabel='Endpoint Type'
                        valueLabel='Configuration'
                        valueType='any'
                        emptyMessage={t(
                          'No endpoints configured. Switch to JSON mode or add rows to define endpoints.'
                        )}
                      />
                    </FormControl>
                    <div className='border-border bg-card/50 mt-2 space-y-2 rounded-lg border p-3'>
                      <p className='text-muted-foreground text-xs font-medium'>
                        {t('Configuration Format')}
                      </p>
                      <pre className='text-muted-foreground overflow-x-auto text-xs'>
                        {`{
  "chat": {
    "path": "/v1/chat/completions",
    "method": "POST"
  },
  "completions": {
    "path": "/v1/completions",
    "method": "POST"
  }
}`}
                      </pre>
                    </div>
                    <div className='border-border bg-card/50 mt-2 space-y-2 rounded-lg border p-3'>
                      <p className='text-muted-foreground text-xs font-medium'>
                        {t('Capability → Endpoint reference')}
                      </p>
                      <div className='flex flex-col gap-1.5'>
                        {CAPABILITY_ENDPOINT_HINTS.map((hint) => (
                          <div
                            key={hint.capability}
                            className='flex items-center justify-between gap-2'
                          >
                            <div className='flex flex-col'>
                              <span className='text-foreground text-xs font-medium'>
                                {getTagLabel(t, hint.capability)}
                              </span>
                              <span className='text-muted-foreground text-[11px]'>
                                {t(hint.descriptionKey)}
                              </span>
                            </div>
                            <div className='flex flex-wrap gap-1'>
                              {hint.templateKeys.map((key) => (
                                <button
                                  key={key}
                                  type='button'
                                  onClick={() =>
                                    handleFillEndpointTemplate(key)
                                  }
                                  className='border-border text-muted-foreground hover:border-primary hover:text-primary rounded-full border px-2 py-0.5 text-[11px] transition-colors'
                                >
                                  {key}
                                </button>
                              ))}
                            </div>
                          </div>
                        ))}
                      </div>
                    </div>
                    <FormDescription>
                      {t(
                        'Define API endpoints for this model. Each endpoint type maps to a path and HTTP method.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </SideDrawerSection>

            <SideDrawerSection>
              <h3 className='text-sm font-semibold'>
                {t('Pricing Configuration')}
              </h3>

              <OfficialPricingReference
                modelName={form.watch('model_name')}
                pricingMode={pricingMode}
                current={{
                  ratio: form.watch('ratio'),
                  completionRatio: form.watch('completionRatio'),
                  cacheRatio: form.watch('cacheRatio'),
                  imageRatio: form.watch('imageRatio'),
                  audioRatio: form.watch('audioRatio'),
                  audioCompletionRatio: form.watch('audioCompletionRatio'),
                  price: form.watch('price'),
                }}
                onApply={applyOfficialPricing}
              />

              <div className='space-y-4'>
                <Label>{t('Pricing mode')}</Label>
                <RadioGroup
                  value={pricingMode}
                  onValueChange={(value) =>
                    setPricingMode(value as PricingMode)
                  }
                >
                  <div className='flex items-center space-x-2'>
                    <RadioGroupItem value='per-token' id='per-token' />
                    <Label htmlFor='per-token' className='font-normal'>
                      {t('Per-token (ratio based)')}
                    </Label>
                  </div>
                  <div className='flex items-center space-x-2'>
                    <RadioGroupItem value='per-request' id='per-request' />
                    <Label htmlFor='per-request' className='font-normal'>
                      {t('Per-request (fixed price)')}
                    </Label>
                  </div>
                </RadioGroup>
              </div>

              {pricingMode === 'per-request' ? (
                <FormField
                  control={form.control}
                  name='price'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        {t(getFixedPriceLabelKey(pricingDisplayType), {
                          currency: pricingCurrencyLabel,
                        })}
                      </FormLabel>
                      <FormControl>
                        <Input
                          type='text'
                          placeholder='0.01'
                          {...field}
                          onChange={(e) => {
                            const value = e.target.value
                            if (validateNumber(value)) {
                              field.onChange(value)
                            }
                          }}
                        />
                      </FormControl>
                      <FormDescription>
                        {field.value && !isNaN(parseFloat(field.value))
                          ? formatRequestPriceRule(
                              convertBillingDisplayToUSD(
                                parseFloat(field.value)
                              )
                            )
                          : t(
                              'Enter price in {{currency}} per request; stored as base USD after conversion.',
                              { currency: pricingCurrencyLabel }
                            )}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              ) : (
                <>
                  <div className='space-y-4'>
                    <Label>{t('Input mode')}</Label>
                    <RadioGroup
                      value={pricingSubMode}
                      onValueChange={(value) =>
                        setPricingSubMode(value as PricingSubMode)
                      }
                    >
                      <div className='flex items-center space-x-2'>
                        <RadioGroupItem value='ratio' id='ratio' />
                        <Label htmlFor='ratio' className='font-normal'>
                          {t('Ratio mode')}
                        </Label>
                      </div>
                      <div className='flex items-center space-x-2'>
                        <RadioGroupItem value='price' id='price' />
                        <Label htmlFor='price' className='font-normal'>
                          {t(getPricingModeLabelKey(pricingDisplayType), {
                            currency: pricingCurrencyLabel,
                          })}
                        </Label>
                      </div>
                    </RadioGroup>
                  </div>

                  {pricingSubMode === 'ratio' ? (
                    <>
                      <FormField
                        control={form.control}
                        name='ratio'
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t('Model ratio')}</FormLabel>
                            <FormControl>
                              <Input
                                type='text'
                                placeholder='1.0'
                                {...field}
                                onChange={(e) => {
                                  const value = e.target.value
                                  if (validateNumber(value)) {
                                    field.onChange(value)
                                    if (value) {
                                      // ratio field is USD-anchored; mirror
                                      // the equivalent display-currency price
                                      // so a mode-switch shows the right value.
                                      const usdPrice = parseFloat(value) * 2
                                      setPromptPrice(
                                        convertUSDToBillingDisplay(
                                          usdPrice
                                        ).toString()
                                      )
                                    } else {
                                      setPromptPrice('')
                                    }
                                  }
                                }}
                              />
                            </FormControl>
                            <FormDescription>
                              {field.value && !isNaN(parseFloat(field.value))
                                ? formatModelPriceRule(
                                    parseFloat(field.value) * 2
                                  )
                                : t('Multiplier for prompt tokens.')}
                            </FormDescription>
                            <FormMessage />
                          </FormItem>
                        )}
                      />

                      <FormField
                        control={form.control}
                        name='completionRatio'
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t('Completion ratio')}</FormLabel>
                            <FormControl>
                              <Input
                                type='text'
                                placeholder='1.0'
                                {...field}
                                onChange={(e) => {
                                  const value = e.target.value
                                  if (validateNumber(value)) {
                                    field.onChange(value)
                                    const ratio = form.getValues('ratio')
                                    if (value && ratio) {
                                      // Mirror completion price in display
                                      // currency for consistency across mode
                                      // switches.
                                      const compPriceUSD =
                                        parseFloat(ratio) *
                                        2 *
                                        parseFloat(value)
                                      setCompletionPrice(
                                        convertUSDToBillingDisplay(
                                          compPriceUSD
                                        ).toString()
                                      )
                                    } else {
                                      setCompletionPrice('')
                                    }
                                  }
                                }}
                              />
                            </FormControl>
                            <FormDescription>
                              {field.value &&
                              !isNaN(parseFloat(field.value)) &&
                              form.getValues('ratio') &&
                              !isNaN(parseFloat(form.getValues('ratio') || ''))
                                ? formatModelPriceRule(
                                    parseFloat(form.getValues('ratio') || '0') *
                                      2 *
                                      parseFloat(field.value)
                                  )
                                : t('Multiplier for completion tokens.')}
                            </FormDescription>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                    </>
                  ) : (
                    <>
                      <div className='space-y-4'>
                        <div className='space-y-2'>
                          <Label>
                            {t(getPromptPriceLabelKey(pricingDisplayType), {
                              currency: pricingCurrencyLabel,
                            })}
                          </Label>
                          <Input
                            type='text'
                            placeholder='2.0'
                            value={promptPrice}
                            onChange={(e) =>
                              handlePromptPriceChange(e.target.value)
                            }
                          />
                          <p className='text-muted-foreground text-sm'>
                            {promptPrice && !isNaN(parseFloat(promptPrice))
                              ? t(
                                  'Calculated ratio: {{ratio}} (stored as base USD)',
                                  {
                                    ratio: (
                                      convertBillingDisplayToUSD(
                                        parseFloat(promptPrice)
                                      ) / 2
                                    ).toFixed(4),
                                  }
                                )
                              : t(
                                  'Enter input price in {{currency}}; stored as base USD after conversion.',
                                  { currency: pricingCurrencyLabel }
                                )}
                          </p>
                        </div>

                        <div className='space-y-2'>
                          <Label>
                            {t(getCompletionPriceLabelKey(pricingDisplayType), {
                              currency: pricingCurrencyLabel,
                            })}
                          </Label>
                          <Input
                            type='text'
                            placeholder='4.0'
                            value={completionPrice}
                            onChange={(e) =>
                              handleCompletionPriceChange(e.target.value)
                            }
                          />
                          <p className='text-muted-foreground text-sm'>
                            {completionPrice &&
                            !isNaN(parseFloat(completionPrice)) &&
                            promptPrice &&
                            !isNaN(parseFloat(promptPrice)) &&
                            parseFloat(promptPrice) > 0
                              ? t(
                                  'Calculated ratio: {{ratio}} (stored as base USD)',
                                  {
                                    ratio: (
                                      parseFloat(completionPrice) /
                                      parseFloat(promptPrice)
                                    ).toFixed(4),
                                  }
                                )
                              : t(
                                  'Enter completion price in {{currency}}; stored as base USD after conversion.',
                                  { currency: pricingCurrencyLabel }
                                )}
                          </p>
                        </div>
                      </div>
                    </>
                  )}

                  <Collapsible
                    open={advancedOpen}
                    onOpenChange={setAdvancedOpen}
                  >
                    <CollapsibleTrigger
                      render={
                        <Button
                          type='button'
                          variant='outline'
                          className='flex w-full items-center justify-between'
                        />
                      }
                    >
                      {t('Advanced options')}
                      <ChevronDown
                        className={`h-4 w-4 transition-transform duration-200 ${
                          advancedOpen ? 'rotate-180' : ''
                        }`}
                      />
                    </CollapsibleTrigger>
                    <CollapsibleContent className='flex flex-col gap-4 pt-4'>
                      <FormField
                        control={form.control}
                        name='cacheRatio'
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t('Cache ratio')}</FormLabel>
                            <FormControl>
                              <Input
                                type='text'
                                placeholder='0.1'
                                {...field}
                                onChange={(e) => {
                                  const value = e.target.value
                                  if (validateNumber(value)) {
                                    field.onChange(value)
                                  }
                                }}
                              />
                            </FormControl>
                            <FormDescription>
                              {t('Discount ratio for cache hits.')}
                            </FormDescription>
                            <FormMessage />
                          </FormItem>
                        )}
                      />

                      <FormField
                        control={form.control}
                        name='imageRatio'
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t('Image ratio')}</FormLabel>
                            <FormControl>
                              <Input
                                type='text'
                                placeholder='1.0'
                                {...field}
                                onChange={(e) => {
                                  const value = e.target.value
                                  if (validateNumber(value)) {
                                    field.onChange(value)
                                  }
                                }}
                              />
                            </FormControl>
                            <FormDescription>
                              {t('Multiplier for image processing.')}
                            </FormDescription>
                            <FormMessage />
                          </FormItem>
                        )}
                      />

                      <FormField
                        control={form.control}
                        name='audioRatio'
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t('Audio ratio')}</FormLabel>
                            <FormControl>
                              <Input
                                type='text'
                                placeholder='1.0'
                                {...field}
                                onChange={(e) => {
                                  const value = e.target.value
                                  if (validateNumber(value)) {
                                    field.onChange(value)
                                  }
                                }}
                              />
                            </FormControl>
                            <FormDescription>
                              {t('Multiplier for audio inputs.')}
                            </FormDescription>
                            <FormMessage />
                          </FormItem>
                        )}
                      />

                      <FormField
                        control={form.control}
                        name='audioCompletionRatio'
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t('Audio completion ratio')}</FormLabel>
                            <FormControl>
                              <Input
                                type='text'
                                placeholder='1.0'
                                {...field}
                                onChange={(e) => {
                                  const value = e.target.value
                                  if (validateNumber(value)) {
                                    field.onChange(value)
                                  }
                                }}
                              />
                            </FormControl>
                            <FormDescription>
                              {t('Multiplier for audio outputs.')}
                            </FormDescription>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                    </CollapsibleContent>
                  </Collapsible>
                </>
              )}
            </SideDrawerSection>

            <SideDrawerSection>
              <h3 className='text-sm font-semibold'>{t('Status & Sync')}</h3>

              <FormField
                control={form.control}
                name='status'
                render={({ field }) => (
                  <FormItem className={sideDrawerSwitchItemClassName()}>
                    <div className='flex flex-col gap-0.5'>
                      <FormLabel className='text-base'>
                        {t('Enabled')}
                      </FormLabel>
                      <FormDescription>
                        {t('Enable or disable this model')}
                      </FormDescription>
                    </div>
                    <FormControl>
                      <Switch
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </FormControl>
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='sync_official'
                render={({ field }) => (
                  <FormItem className={sideDrawerSwitchItemClassName()}>
                    <div className='flex flex-col gap-0.5'>
                      <FormLabel className='text-base'>
                        {t('Official Sync')}
                      </FormLabel>
                      <FormDescription>
                        {t(
                          'Include this model in bulk metadata sync (description, icon, tags, vendor). Pricing is managed separately in the section above.'
                        )}
                      </FormDescription>
                    </div>
                    <FormControl>
                      <Switch
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </FormControl>
                  </FormItem>
                )}
              />
            </SideDrawerSection>
          </form>
        </Form>

        <SheetFooter className={sideDrawerFooterClassName()}>
          <SheetClose
            render={<Button variant='outline' disabled={isSubmitting} />}
          >
            {t('Cancel')}
          </SheetClose>
          <Button form='model-form' type='submit' disabled={isSubmitting}>
            {isSubmitting && <Loader2 className='mr-2 h-4 w-4 animate-spin' />}
            {isEditing ? t('Update Model') : t('Save changes')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
