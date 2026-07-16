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
import { useQuery } from '@tanstack/react-query'
import { Code2, Eye, RotateCcw, Save } from 'lucide-react'
import { memo, useCallback, useMemo, useRef, useState } from 'react'
import { type UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { JsonCodeEditor } from '@/components/json-code-editor'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Switch } from '@/components/ui/switch'
import { getModels } from '@/features/models/api'
import { modelsQueryKeys } from '@/features/models/lib'

import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import {
  ModelRatioVisualEditor,
  type ModelRatioVisualEditorHandle,
} from './model-ratio-visual-editor'

type ModelFormValues = {
  ModelPrice: string
  ModelRatio: string
  CacheRatio: string
  CreateCacheRatio: string
  CompletionRatio: string
  ImageRatio: string
  AudioRatio: string
  AudioCompletionRatio: string
  ExposeRatioEnabled: boolean
  BillingMode: string
  BillingExpr: string
}

type ModelRatioFormProps = {
  form: UseFormReturn<ModelFormValues>
  savedValues: ModelFormValues
  onSave: (values: ModelFormValues) => Promise<void>
  onReset: () => void
  isSaving: boolean
  isResetting: boolean
}

type ModelJsonFieldName =
  | 'ModelPrice'
  | 'ModelRatio'
  | 'CacheRatio'
  | 'CreateCacheRatio'
  | 'CompletionRatio'
  | 'ImageRatio'
  | 'AudioRatio'
  | 'AudioCompletionRatio'
  | 'ImagePricing'
  | 'VideoPricing'
  | 'AudioInPricing'
  | 'AudioOutPricing'

const modelJsonFields: Array<{
  name: ModelJsonFieldName
  labelKey: string
  descriptionKey: string
}> = [
  {
    name: 'ModelPrice',
    labelKey: 'Model fixed pricing',
    descriptionKey:
      'JSON map of model → USD cost per request. Takes precedence over ratio based billing.',
  },
  {
    name: 'ModelRatio',
    labelKey: 'Model ratio',
    descriptionKey: 'JSON map of model → multiplier applied to quota billing.',
  },
  {
    name: 'CacheRatio',
    labelKey: 'Prompt cache ratio',
    descriptionKey: 'Optional ratio used when upstream cache hits occur.',
  },
  {
    name: 'CreateCacheRatio',
    labelKey: 'Create cache ratio',
    descriptionKey:
      'Ratio applied when creating cache entries for supported models.',
  },
  {
    name: 'CompletionRatio',
    labelKey: 'Completion ratio',
    descriptionKey:
      'Applies to custom completion endpoints. JSON map of model → ratio.',
  },
  {
    name: 'ImageRatio',
    labelKey: 'Image ratio',
    descriptionKey: 'Configure per-model ratio for image inputs or outputs.',
  },
  {
    name: 'AudioRatio',
    labelKey: 'Audio ratio',
    descriptionKey:
      'Ratio applied to audio inputs where supported by the upstream model.',
  },
  {
    name: 'AudioCompletionRatio',
    labelKey: 'Audio completion ratio',
    descriptionKey: 'Ratio applied to audio completions for streaming models.',
  },
  {
    name: 'ImagePricing',
    labelKey: 'Image generation pricing',
    descriptionKey:
      'Structured pricing for image generation models (per-image base + multipliers).',
  },
  {
    name: 'VideoPricing',
    labelKey: 'Video generation pricing',
    descriptionKey:
      'Structured pricing for video generation models (per-second base + multipliers).',
  },
  {
    name: 'AudioInPricing',
    labelKey: 'Audio input pricing',
    descriptionKey:
      'Structured pricing for speech recognition models (per-minute base).',
  },
  {
    name: 'AudioOutPricing',
    labelKey: 'Audio output pricing',
    descriptionKey:
      'Structured pricing for text-to-speech models (per-1M-chars base + multipliers).',
  },
]

function ModelJsonTextareaField(props: {
  form: UseFormReturn<ModelFormValues>
  name: ModelJsonFieldName
  label: string
  description: string
}) {
  return (
    <FormField
      control={props.form.control}
      name={props.name}
      render={({ field }) => (
        <FormItem className='flex min-w-0 flex-col gap-2'>
          <FormLabel>{props.label}</FormLabel>
          <FormControl>
            <JsonCodeEditor
              value={field.value}
              onChange={(value) => field.onChange(value)}
            />
          </FormControl>
          <FormDescription className='text-xs leading-5'>
            {props.description}
          </FormDescription>
          <FormMessage />
        </FormItem>
      )}
    />
  )
}

export const ModelRatioForm = memo(function ModelRatioForm({
  form,
  savedValues,
  onSave,
  onReset,
  isSaving,
  isResetting,
}: ModelRatioFormProps) {
  const { t } = useTranslation()
  const [editMode, setEditMode] = useState<'visual' | 'json'>('visual')
  const visualEditorRef = useRef<ModelRatioVisualEditorHandle>(null)

  // "Added only" toggle — default true so admins see just the models they've
  // registered in the model catalogue. Toggling off reveals every OptionMap
  // pricing key (including legacy models the admin removed from the catalogue
  // but hasn't cleared from pricing). State persists in localStorage so the
  // preference survives navigation.
  const ADDED_ONLY_STORAGE_KEY = 'model-ratio-filter-added-only'
  const [showAddedOnly, setShowAddedOnly] = useState<boolean>(() => {
    try {
      const raw = localStorage.getItem(ADDED_ONLY_STORAGE_KEY)
      return raw === null ? true : raw === 'true'
    } catch {
      return true
    }
  })
  const handleToggleAddedOnly = useCallback((next: boolean) => {
    setShowAddedOnly(next)
    try {
      localStorage.setItem(ADDED_ONLY_STORAGE_KEY, String(next))
    } catch {
      /* localStorage unavailable — ignore */
    }
  }, [])

  // Fetch the registered models set for the filter. `page_size: 10000` is the
  // convention used by other admin screens (e.g. model-mutate-drawer) to grab
  // the full catalogue in one round-trip; the API caps at a large enough
  // number that pagination isn't a concern for the 100-500 model range this
  // deployment sees. `enabled: showAddedOnly` skips the request when the
  // filter is off, so the toggle is truly zero-cost when disabled.
  const { data: registeredModelsData } = useQuery({
    queryKey: modelsQueryKeys.list({ pageSize: 10000, forFilter: true }),
    queryFn: () => getModels({ page_size: 10000 }),
    enabled: showAddedOnly,
    staleTime: 60_000,
  })
  const registeredModelNames = useMemo(() => {
    if (!registeredModelsData?.data?.items) return undefined
    return new Set(
      registeredModelsData.data.items.map((m) => m.model_name),
    ) as ReadonlySet<string>
  }, [registeredModelsData])

  const handleFieldChange = useCallback(
    (field: keyof ModelFormValues, value: string) => {
      form.setValue(field, value, {
        shouldValidate: true,
        shouldDirty: true,
      })
    },
    [form]
  )

  const toggleEditMode = useCallback(() => {
    setEditMode((prev) => (prev === 'visual' ? 'json' : 'visual'))
  }, [])

  const handleSave = useCallback(async () => {
    if (editMode === 'visual') {
      const committed = await visualEditorRef.current?.commitOpenEditor()
      if (committed === false) return
    }

    await form.handleSubmit(onSave)()
  }, [editMode, form, onSave])

  return (
    <div className='space-y-6'>
      <div className='flex flex-wrap items-center justify-end gap-3'>
        {/* Added-only filter toggle. Placed on the left of the toolbar so
          * admins scanning left-to-right see the filter state before the
          * destructive Reset action. Hidden in JSON edit mode because the
          * JSON editor already shows the raw OptionMap and can't render a
          * filtered view. */}
        {editMode === 'visual' && (
          <label className='mr-auto flex items-center gap-2 text-sm text-muted-foreground'>
            <Switch
              checked={showAddedOnly}
              onCheckedChange={handleToggleAddedOnly}
            />
            <span>{t('Show only registered models')}</span>
          </label>
        )}
        <Button
          type='button'
          variant='destructive'
          size='sm'
          onClick={onReset}
          disabled={isResetting}
        >
          <RotateCcw data-icon='inline-start' />
          {t('Reset prices')}
        </Button>
        {editMode === 'json' && (
          <Button
            type='button'
            size='sm'
            onClick={handleSave}
            disabled={isSaving}
          >
            <Save data-icon='inline-start' />
            {isSaving ? t('Saving...') : t('Save model prices')}
          </Button>
        )}
        <Button variant='outline' size='sm' onClick={toggleEditMode}>
          {editMode === 'visual' ? (
            <>
              <Code2 className='mr-2 h-4 w-4' />
              {t('Switch to JSON')}
            </>
          ) : (
            <>
              <Eye className='mr-2 h-4 w-4' />
              {t('Switch to Visual')}
            </>
          )}
        </Button>
      </div>

      <Form {...form}>
        {editMode === 'visual' ? (
          <div className='space-y-6'>
            <ModelRatioVisualEditor
              ref={visualEditorRef}
              savedModelPrice={savedValues.ModelPrice}
              savedModelRatio={savedValues.ModelRatio}
              savedCacheRatio={savedValues.CacheRatio}
              savedCreateCacheRatio={savedValues.CreateCacheRatio}
              savedCompletionRatio={savedValues.CompletionRatio}
              savedImageRatio={savedValues.ImageRatio}
              savedAudioRatio={savedValues.AudioRatio}
              savedAudioCompletionRatio={savedValues.AudioCompletionRatio}
              savedBillingMode={savedValues.BillingMode}
              savedBillingExpr={savedValues.BillingExpr}
              modelPrice={form.watch('ModelPrice')}
              modelRatio={form.watch('ModelRatio')}
              cacheRatio={form.watch('CacheRatio')}
              createCacheRatio={form.watch('CreateCacheRatio')}
              completionRatio={form.watch('CompletionRatio')}
              imageRatio={form.watch('ImageRatio')}
              audioRatio={form.watch('AudioRatio')}
              audioCompletionRatio={form.watch('AudioCompletionRatio')}
              imagePricing={form.watch('ImagePricing')}
              videoPricing={form.watch('VideoPricing')}
              audioInPricing={form.watch('AudioInPricing')}
              audioOutPricing={form.watch('AudioOutPricing')}
              billingMode={form.watch('BillingMode')}
              billingExpr={form.watch('BillingExpr')}
              filterAddedOnly={showAddedOnly}
              registeredModelNames={registeredModelNames}
              onSave={handleSave}
              isSaving={isSaving}
              onChange={(field, value) => {
                const fieldMap: Record<string, keyof ModelFormValues> = {
                  'billing_setting.billing_mode': 'BillingMode',
                  'billing_setting.billing_expr': 'BillingExpr',
                }
                const formField =
                  fieldMap[field] || (field as keyof ModelFormValues)
                handleFieldChange(formField, value)
              }}
            />

            <FormField
              control={form.control}
              name='ExposeRatioEnabled'
              render={({ field }) => (
                <SettingsSwitchItem>
                  <SettingsSwitchContent>
                    <FormLabel>{t('Expose ratio API')}</FormLabel>
                    <FormDescription>
                      {t(
                        'Allow clients to query configured ratios via `/api/ratio`.'
                      )}
                    </FormDescription>
                  </SettingsSwitchContent>
                  <FormControl>
                    <Switch
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  </FormControl>
                </SettingsSwitchItem>
              )}
            />
          </div>
        ) : (
          <SettingsForm onSubmit={form.handleSubmit(onSave)}>
            <div className='grid min-w-0 gap-x-5 gap-y-8 lg:grid-cols-2 2xl:grid-cols-3'>
              {modelJsonFields.map((config) => (
                <ModelJsonTextareaField
                  key={config.name}
                  form={form}
                  name={config.name}
                  label={t(config.labelKey)}
                  description={t(config.descriptionKey)}
                />
              ))}
            </div>

            <FormField
              control={form.control}
              name='ExposeRatioEnabled'
              render={({ field }) => (
                <SettingsSwitchItem>
                  <SettingsSwitchContent>
                    <FormLabel>{t('Expose ratio API')}</FormLabel>
                    <FormDescription>
                      {t(
                        'Allow clients to query configured ratios via `/api/ratio`.'
                      )}
                    </FormDescription>
                  </SettingsSwitchContent>
                  <FormControl>
                    <Switch
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  </FormControl>
                </SettingsSwitchItem>
              )}
            />
          </SettingsForm>
        )}
      </Form>
    </div>
  )
})
