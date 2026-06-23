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
import * as React from 'react'
import * as z from 'zod'
import { useForm, type Resolver } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Code2, Copy, Eye, Save, ShieldAlert } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { cn } from '@/lib/utils'
import {
  Alert,
  AlertAction,
  AlertDescription,
  AlertTitle,
} from '@/components/ui/alert'
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
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/ui/tabs'
import { RiskAcknowledgementDialog } from '@/components/risk-acknowledgement-dialog'
import {
  confirmPaymentCompliance,
  createPaymentConfig,
  getPaymentConfigs,
  updatePaymentConfig,
} from '../api'
import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import { safeNumberFieldProps } from '../utils/numeric-field'
import { AmountDiscountVisualEditor } from './amount-discount-visual-editor'
import { AmountOptionsVisualEditor } from './amount-options-visual-editor'
import { CreemProductsVisualEditor } from './creem-products-visual-editor'
import { PaymentMethodsVisualEditor } from './payment-methods-visual-editor'
import { detectWechatPemKindByContent } from './wechat-pay-file-upload'
import {
  formatJsonForEditor,
  getJsonError,
  normalizeJsonForComparison,
  removeTrailingSlash,
} from './utils'
import { saveWaffoPancakeConfig } from './waffo-pancake-api'
import {
  WaffoPancakeSettingsSection,
  type WaffoPancakeBinding,
  type WaffoPancakeSettingsValues,
} from './waffo-pancake-settings-section'
import type { PaymentConfig } from '../types'
import {
  type PayMethod,
  WaffoSettingsSection,
  type WaffoSettingsValues,
} from './waffo-settings-section'

const paymentSchema = z.object({
  PayAddress: z.string().refine((value) => {
    const trimmed = value.trim()
    if (!trimmed) return true
    return /^https?:\/\//.test(trimmed)
  }, 'Provide a valid callback URL starting with http:// or https://'),
  EpayId: z.string(),
  EpayKey: z.string(),
  Price: z.coerce.number().min(0),
  MinTopUp: z.coerce.number().min(0),
  CustomCallbackAddress: z.string().refine((value) => {
    const trimmed = value.trim()
    if (!trimmed) return true
    return /^https?:\/\//.test(trimmed)
  }, 'Provide a valid URL starting with http:// or https://'),
  PayMethods: z.string().superRefine((value, ctx) => {
    const error = getJsonError(value)
    if (error) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: error,
      })
    }
  }),
  AmountOptions: z.string().superRefine((value, ctx) => {
    const error = getJsonError(value, (parsed) => Array.isArray(parsed))
    if (error) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: error,
      })
    }
  }),
  AmountDiscount: z.string().superRefine((value, ctx) => {
    const error = getJsonError(
      value,
      (parsed) =>
        !!parsed && typeof parsed === 'object' && !Array.isArray(parsed)
    )
    if (error) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: error,
      })
    }
  }),
  StripeApiSecret: z.string(),
  StripeWebhookSecret: z.string(),
  StripePriceId: z.string(),
  StripeUnitPrice: z.coerce.number().min(0),
  StripeMinTopUp: z.coerce.number().min(0),
  StripePromotionCodesEnabled: z.boolean(),
  CreemApiKey: z.string(),
  CreemWebhookSecret: z.string(),
  CreemTestMode: z.boolean(),
  CreemProducts: z.string().superRefine((value, ctx) => {
    const error = getJsonError(value, (parsed) => Array.isArray(parsed))
    if (error) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: error,
      })
    }
  }),
  WaffoEnabled: z.boolean(),
  WaffoApiKey: z.string(),
  WaffoPrivateKey: z.string(),
  WaffoPublicCert: z.string(),
  WaffoSandboxPublicCert: z.string(),
  WaffoSandboxApiKey: z.string(),
  WaffoSandboxPrivateKey: z.string(),
  WaffoSandbox: z.boolean(),
  WaffoMerchantId: z.string(),
  WaffoCurrency: z.string(),
  WaffoUnitPrice: z.coerce.number().min(0),
  WaffoMinTopUp: z.coerce.number().min(1),
  WaffoNotifyUrl: z.string(),
  WaffoReturnUrl: z.string(),
  WaffoPancakeMerchantID: z.string(),
  WaffoPancakePrivateKey: z.string(),
  WaffoPancakeReturnURL: z.string(),
})

type PaymentFormValues = z.infer<typeof paymentSchema>
type WaffoFormFieldValues = Omit<WaffoSettingsValues, 'WaffoPayMethods'>
type PaymentBaseFormValues = Omit<
  PaymentFormValues,
  keyof WaffoFormFieldValues | keyof WaffoPancakeSettingsValues
>

const CURRENT_COMPLIANCE_TERMS_VERSION = 'v1'

type PaymentComplianceDefaults = {
  confirmed: boolean
  termsVersion: string
  confirmedAt: number
  confirmedBy: number
}

function GatewaySwitchRow(props: {
  title: string
  description: string
  checked: boolean
  onCheckedChange: (checked: boolean) => void
}) {
  return (
    <SettingsSwitchItem>
      <SettingsSwitchContent>
        <FormLabel>{props.title}</FormLabel>
        <FormDescription>{props.description}</FormDescription>
      </SettingsSwitchContent>
      <Switch checked={props.checked} onCheckedChange={props.onCheckedChange} />
    </SettingsSwitchItem>
  )
}

type NativePaymentProvider = 'alipay' | 'wxpay'

function getNativePaymentDisplayName(provider: NativePaymentProvider) {
  return provider === 'alipay' ? 'Alipay' : 'WeChat Pay'
}

function RequiredLabel(props: { children: React.ReactNode }) {
  return (
    <Label>
      {props.children}
      <span className='text-destructive ml-1'>*</span>
    </Label>
  )
}

function WechatPemUploadField(props: {
  expectedKind: 'cert' | 'key'
  onLoad: (content: string) => void
}) {
  const { t } = useTranslation()
  const inputRef = React.useRef<HTMLInputElement | null>(null)
  const expectedName =
    props.expectedKind === 'cert' ? 'apiclient_cert.pem' : 'apiclient_key.pem'

  const handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]
    if (!file) return

    const reader = new FileReader()
    reader.onload = (loadEvent) => {
      if (typeof loadEvent.target?.result !== 'string') {
        return
      }
      const content = loadEvent.target.result
      // Authoritative check by PEM content, not filename. This accepts
      // renamed files (e.g. "apiclient_cert (1).pem") and rejects files
      // whose content does not match the expected kind.
      const kind = detectWechatPemKindByContent(content)
      if (kind !== props.expectedKind) {
        toast.error(
          t('Please upload {{fileName}}', {
            fileName: expectedName,
          })
        )
        return
      }
      props.onLoad(content)
      toast.success(t('File loaded'))
    }
    reader.readAsText(file)
    event.target.value = ''
  }

  return (
    <div className='flex items-center gap-2'>
      <input
        ref={inputRef}
        type='file'
        accept='.pem,text/plain'
        className='hidden'
        onChange={handleChange}
      />
      <Button type='button' variant='outline' onClick={() => inputRef.current?.click()}>
        {t('Upload {{fileName}}', { fileName: expectedName })}
      </Button>
      <span className='text-muted-foreground text-xs'>{expectedName}</span>
    </div>
  )
}

function PaymentIconUploadField(props: {
  label: string
  value?: string
  onChange: (value: string) => void
}) {
  const { t } = useTranslation()
  const iconFileInputRef = React.useRef<HTMLInputElement | null>(null)

  const handleIconFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]

    if (!file) {
      return
    }

    const maxIconSize = 100 * 1024

    if (file.size > maxIconSize) {
      toast.error(t('Icon file must be 100 KB or smaller'))
      event.target.value = ''
      return
    }

    const reader = new FileReader()
    reader.onload = (loadEvent) => {
      props.onChange(
        typeof loadEvent.target?.result === 'string'
          ? loadEvent.target.result
          : ''
      )
    }
    reader.readAsDataURL(file)
    event.target.value = ''
  }

  return (
    <div className='space-y-2'>
      <Label>{props.label}</Label>
      <div className='flex flex-wrap items-center gap-3'>
        {props.value ? (
          <img
            src={props.value}
            alt={props.label}
            className='h-10 w-10 rounded border object-contain p-1'
          />
        ) : (
          <div className='bg-muted text-muted-foreground flex h-10 w-10 items-center justify-center rounded border text-xs'>
            {t('Icon')}
          </div>
        )}
        <input
          ref={iconFileInputRef}
          type='file'
          accept='image/png,image/jpeg,image/svg+xml,image/webp'
          className='hidden'
          onChange={handleIconFileChange}
        />
        <Button
          type='button'
          variant='outline'
          onClick={() => iconFileInputRef.current?.click()}
        >
          {t('Upload icon')}
        </Button>
        {props.value ? (
          <Button type='button' variant='outline' onClick={() => props.onChange('')}>
            {t('Clear icon')}
          </Button>
        ) : null}
      </div>
      <p className='text-muted-foreground text-xs'>
        {t(
          'Supports PNG, JPG, SVG, or WebP. Recommended size: 128×128 or smaller.'
        )}
      </p>
    </div>
  )
}

type PaymentSettingsSectionProps = {
  defaultValues: PaymentBaseFormValues
  waffoDefaultValues: WaffoSettingsValues
  waffoPancakeDefaultValues: WaffoPancakeSettingsValues
  waffoPancakeProvisionedStoreID?: string
  waffoPancakeProvisionedProductID?: string
  complianceDefaults: PaymentComplianceDefaults
}

function parseWaffoPayMethods(value: string): PayMethod[] {
  try {
    const parsed = JSON.parse(value || '[]')
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

const paymentConfigComparableFields: Array<keyof PaymentConfig> = [
  'name',
  'display_name',
  'icon_url',
  'enabled',
  'sort_order',
  'app_id',
  'app_private_key',
  'alipay_public_key',
  'alipay_app_public_cert',
  'alipay_public_cert',
  'alipay_root_cert',
  'wechat_app_id',
  'wechat_app_secret',
  'wechat_mch_id',
  'wechat_api_key',
  'wechat_serial_no',
  'wechat_private_key',
  'wechat_auth_mode',
  'wechat_public_key_id',
  'wechat_public_key',
  'gateway_url',
  'notify_url',
  'return_url',
]

function hasPaymentConfigChanges(
  current: Partial<PaymentConfig>,
  saved: PaymentConfig | null
) {
  if (!saved) return Boolean(current.enabled)
  return paymentConfigComparableFields.some(
    (field) => (current[field] ?? '') !== (saved[field] ?? '')
  )
}

export function PaymentSettingsSection({
  defaultValues,
  waffoDefaultValues,
  waffoPancakeDefaultValues,
  waffoPancakeProvisionedStoreID,
  waffoPancakeProvisionedProductID,
  complianceDefaults,
}: PaymentSettingsSectionProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const updateOption = useUpdateOption()
  const initialFormValues = React.useMemo<PaymentFormValues>(
    () => ({
      ...defaultValues,
      ...waffoDefaultValues,
      ...waffoPancakeDefaultValues,
    }),
    [defaultValues, waffoDefaultValues, waffoPancakeDefaultValues]
  )
  const initialRef = React.useRef(initialFormValues)
  const defaultsSignature = React.useMemo(
    () => JSON.stringify(initialFormValues),
    [initialFormValues]
  )

  const [payMethodsVisualMode, setPayMethodsVisualMode] = React.useState(true)
  const [amountOptionsVisualMode, setAmountOptionsVisualMode] =
    React.useState(true)
  const [amountDiscountVisualMode, setAmountDiscountVisualMode] =
    React.useState(true)
  const [creemProductsVisualMode, setCreemProductsVisualMode] =
    React.useState(true)
  const [showComplianceDialog, setShowComplianceDialog] = React.useState(false)
  const [activePaymentGateway, setActivePaymentGateway] = React.useState('epay')
  const [epayEnabled, setEpayEnabled] = React.useState(() =>
    Boolean(
      initialFormValues.PayAddress ||
        initialFormValues.EpayId ||
        initialFormValues.EpayKey
    )
  )
  const [stripeEnabled, setStripeEnabled] = React.useState(() =>
    Boolean(
      initialFormValues.StripeApiSecret ||
      initialFormValues.StripePriceId
    )
  )
  const [creemEnabled, setCreemEnabled] = React.useState(() =>
    Boolean(
      initialFormValues.CreemApiKey ||
      initialFormValues.CreemProducts
    )
  )
  // Waffo already has WaffoEnabled in form values
  const [waffoPancakeEnabled, setWaffoPancakeEnabled] = React.useState(() =>
    Boolean(
      initialFormValues.WaffoPancakeMerchantID ||
      waffoPancakeProvisionedStoreID
    )
  )
  const [alipayConfig, setAlipayConfig] = React.useState<PaymentConfig | null>(null)
  const [wechatConfig, setWechatConfig] = React.useState<PaymentConfig | null>(null)
  const [alipayForm, setAlipayForm] = React.useState<Partial<PaymentConfig>>({
    provider: 'alipay',
    name: 'Alipay',
    display_name: getNativePaymentDisplayName('alipay'),
    enabled: false,
    sort_order: 10,
  })
  const [wechatForm, setWechatForm] = React.useState<Partial<PaymentConfig>>({
    provider: 'wxpay',
    name: 'WeChat Pay',
    display_name: getNativePaymentDisplayName('wxpay'),
    enabled: false,
    sort_order: 20,
    wechat_auth_mode: 'certificate',
  })
  const [waffoPayMethods, setWaffoPayMethods] = React.useState<PayMethod[]>(
    () => parseWaffoPayMethods(waffoDefaultValues.WaffoPayMethods)
  )
  const [waffoPancakeSelection, setWaffoPancakeSelection] =
    React.useState<WaffoPancakeBinding>({
      storeID: waffoPancakeProvisionedStoreID ?? '',
      productID: waffoPancakeProvisionedProductID ?? '',
    })
  const [waffoPancakeSavedBinding, setWaffoPancakeSavedBinding] =
    React.useState<WaffoPancakeBinding>({
      storeID: waffoPancakeProvisionedStoreID ?? '',
      productID: waffoPancakeProvisionedProductID ?? '',
    })

  const { data: paymentConfigs = [], refetch: refetchPaymentConfigs } = useQuery({
    queryKey: ['payment-configs'],
    queryFn: async () => {
      const response = await getPaymentConfigs()
      return response.success ? response.data || [] : []
    },
  })

  const createPaymentConfigMutation = useMutation({
    mutationFn: createPaymentConfig,
    onSuccess: () => {
      toast.success(t('Payment configuration saved successfully'))
      refetchPaymentConfigs()
    },
    onError: () => {},
  })

  const updatePaymentConfigMutation = useMutation({
    mutationFn: ({ id, data }: { id: number; data: PaymentConfig }) =>
      updatePaymentConfig(id, data),
    onSuccess: () => {
      toast.success(t('Payment configuration saved successfully'))
      refetchPaymentConfigs()
    },
    onError: () => {},
  })

  React.useEffect(() => {
    const alipay = paymentConfigs.find(c => c.provider === 'alipay')
    const wechat = paymentConfigs.find(c => c.provider === 'wxpay')
    if (alipay) {
      setAlipayConfig(alipay)
      setAlipayForm({
        ...alipay,
        display_name: getNativePaymentDisplayName('alipay'),
      })
    }
    if (wechat) {
      setWechatConfig(wechat)
      setWechatForm({
        ...wechat,
        display_name: getNativePaymentDisplayName('wxpay'),
        wechat_auth_mode: wechat.wechat_auth_mode || 'certificate',
      })
    }
  }, [paymentConfigs])

  React.useEffect(() => {
    setWaffoPayMethods(parseWaffoPayMethods(waffoDefaultValues.WaffoPayMethods))
  }, [waffoDefaultValues.WaffoPayMethods])

  React.useEffect(() => {
    const nextBinding = {
      storeID: waffoPancakeProvisionedStoreID ?? '',
      productID: waffoPancakeProvisionedProductID ?? '',
    }
    setWaffoPancakeSelection(nextBinding)
    setWaffoPancakeSavedBinding(nextBinding)
  }, [waffoPancakeProvisionedProductID, waffoPancakeProvisionedStoreID])

  const complianceStatements = React.useMemo(
    () => [
      t(
        'You have legally obtained authorization for the connected model APIs, accounts, keys, and quotas.'
      ),
      t(
        'You commit to using upstream APIs, accounts, keys, quotas, and service capabilities only within the scope of lawful authorization obtained from upstream service providers, model service providers, or relevant rights holders, and will not conduct unauthorized resale, trafficking, distribution, or other non-compliant commercialization.'
      ),
      t(
        'If you provide generative AI services to the public in mainland China, you will fulfill legal obligations including filing, security assessment, content safety, complaint handling, generated content labeling, log retention, and personal information protection.'
      ),
      t(
        'You commit not to use this system to implement, assist with, or indirectly implement acts that violate applicable laws and regulations, regulatory requirements, platform rules, public interests, or the lawful rights and interests of third parties.'
      ),
      t(
        'You understand and independently bear legal responsibility arising from deployment, operation, and charging behavior.'
      ),
      t(
        'You understand this compliance reminder is only for risk notice and does not constitute legal advice, a compliance review conclusion, or a guarantee of the legality of your use of this system; you should consult professional legal or compliance advisors based on your actual business scenario.'
      ),
    ],
    [t]
  )

  const complianceRequiredText = t(
    'I have read and understood the above compliance reminder, acknowledge the related legal risks, and confirm that I bear legal responsibility arising from deployment, operation, and charging behavior.'
  )
  const complianceRequiredTextParts = React.useMemo(
    () => [
      {
        type: 'input' as const,
        text: t('I have read and understood the above compliance reminder'),
      },
      { type: 'static' as const, text: t('，') },
      {
        type: 'input' as const,
        text: t('acknowledge the related legal risks'),
      },
      { type: 'static' as const, text: t('，and ') },
      {
        type: 'input' as const,
        text: t(
          'confirm that I bear legal responsibility arising from deployment'
        ),
      },
      { type: 'static' as const, text: t('、') },
      {
        type: 'input' as const,
        text: t('operation and charging behavior'),
      },
    ],
    [t]
  )

  const complianceConfirmed =
    complianceDefaults.confirmed &&
    complianceDefaults.termsVersion === CURRENT_COMPLIANCE_TERMS_VERSION

  const confirmComplianceMutation = useMutation({
    mutationFn: confirmPaymentCompliance,
    onSuccess: (data) => {
      if (data.success) {
        toast.success(t('Compliance confirmed successfully'))
        setShowComplianceDialog(false)
        queryClient.invalidateQueries({ queryKey: ['system-options'] })
      } else {
        toast.error(data.message || t('Failed to confirm compliance'))
      }
    },
    onError: (error: Error) => {
      toast.error(error.message || t('Failed to confirm compliance'))
    },
  })

  const form = useForm<PaymentFormValues>({
    resolver: zodResolver(paymentSchema) as Resolver<PaymentFormValues>,
    mode: 'onChange', // Enable real-time validation
    defaultValues: {
      ...initialFormValues,
      PayMethods: formatJsonForEditor(initialFormValues.PayMethods),
      AmountOptions: formatJsonForEditor(initialFormValues.AmountOptions),
      AmountDiscount: formatJsonForEditor(initialFormValues.AmountDiscount),
      CreemProducts: formatJsonForEditor(initialFormValues.CreemProducts),
    },
  })

  const { isSubmitting } = form.formState
  const isPaymentSaving =
    updateOption.isPending ||
    createPaymentConfigMutation.isPending ||
    updatePaymentConfigMutation.isPending ||
    isSubmitting

  const setPaymentValue = React.useCallback(
    (
      key: keyof PaymentFormValues,
      value: PaymentFormValues[keyof PaymentFormValues]
    ) => {
      form.setValue(
        key as Parameters<typeof form.setValue>[0],
        value as Parameters<typeof form.setValue>[1],
        {
          shouldDirty: true,
          shouldValidate: true,
        }
      )
    },
    [form]
  )

  const setWaffoValue = React.useCallback(
    <K extends keyof WaffoFormFieldValues>(
      key: K,
      value: WaffoFormFieldValues[K]
    ) => {
      setPaymentValue(
        key as keyof PaymentFormValues,
        value as PaymentFormValues[keyof PaymentFormValues]
      )
    },
    [setPaymentValue]
  )

  const setWaffoPancakeValue = React.useCallback(
    <K extends keyof WaffoPancakeSettingsValues>(
      key: K,
      value: WaffoPancakeSettingsValues[K]
    ) => {
      setPaymentValue(
        key as keyof PaymentFormValues,
        value as PaymentFormValues[keyof PaymentFormValues]
      )
    },
    [setPaymentValue]
  )

  const saveAlipayConfig = async () => {
    if (!alipayForm.provider) return
    const data = {
      ...alipayForm,
      display_name: getNativePaymentDisplayName('alipay'),
    } as PaymentConfig
    try {
      if (alipayConfig?.id) {
        await updatePaymentConfigMutation.mutateAsync({
          id: alipayConfig.id,
          data,
        })
      } else {
        await createPaymentConfigMutation.mutateAsync(data)
      }
      toast.success(t('Alipay configuration saved successfully'))
      await refetchPaymentConfigs()
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to save Alipay configuration')
      )
    }
  }

  const saveWechatConfig = async () => {
    if (!wechatForm.provider) return
    const data = {
      ...wechatForm,
      display_name: getNativePaymentDisplayName('wxpay'),
      wechat_auth_mode: wechatForm.wechat_auth_mode || 'certificate',
    } as PaymentConfig
    try {
      if (wechatConfig?.id) {
        await updatePaymentConfigMutation.mutateAsync({
          id: wechatConfig.id,
          data,
        })
      } else {
        await createPaymentConfigMutation.mutateAsync(data)
      }
      toast.success(t('WeChat Pay configuration saved successfully'))
      await refetchPaymentConfigs()
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to save WeChat Pay configuration')
      )
    }
  }

  React.useEffect(() => {
    const parsedDefaults = JSON.parse(defaultsSignature) as PaymentFormValues
    initialRef.current = parsedDefaults
    form.reset({
      ...parsedDefaults,
      PayMethods: formatJsonForEditor(parsedDefaults.PayMethods),
      AmountOptions: formatJsonForEditor(parsedDefaults.AmountOptions),
      AmountDiscount: formatJsonForEditor(parsedDefaults.AmountDiscount),
      CreemProducts: formatJsonForEditor(parsedDefaults.CreemProducts),
    })
  }, [defaultsSignature, form])

  const onSubmit = async (values: PaymentFormValues) => {
    const sanitized = {
      PayAddress: removeTrailingSlash(values.PayAddress),
      EpayId: values.EpayId.trim(),
      EpayKey: values.EpayKey.trim(),
      Price: values.Price,
      MinTopUp: values.MinTopUp,
      CustomCallbackAddress: removeTrailingSlash(values.CustomCallbackAddress),
      PayMethods: values.PayMethods.trim(),
      AmountOptions: values.AmountOptions.trim(),
      AmountDiscount: values.AmountDiscount.trim(),
      StripeApiSecret: values.StripeApiSecret.trim(),
      StripeWebhookSecret: values.StripeWebhookSecret.trim(),
      StripePriceId: values.StripePriceId.trim(),
      StripeUnitPrice: values.StripeUnitPrice,
      StripeMinTopUp: values.StripeMinTopUp,
      StripePromotionCodesEnabled: values.StripePromotionCodesEnabled,
      CreemApiKey: values.CreemApiKey.trim(),
      CreemWebhookSecret: values.CreemWebhookSecret.trim(),
      CreemTestMode: values.CreemTestMode,
      CreemProducts: values.CreemProducts.trim(),
      WaffoEnabled: values.WaffoEnabled,
      WaffoSandbox: values.WaffoSandbox,
      WaffoMerchantId: values.WaffoMerchantId.trim(),
      WaffoCurrency: values.WaffoCurrency.trim() || 'USD',
      WaffoUnitPrice: values.WaffoUnitPrice,
      WaffoMinTopUp: values.WaffoMinTopUp,
      WaffoNotifyUrl: values.WaffoNotifyUrl.trim(),
      WaffoReturnUrl: values.WaffoReturnUrl.trim(),
      WaffoPublicCert: values.WaffoPublicCert.trim(),
      WaffoSandboxPublicCert: values.WaffoSandboxPublicCert.trim(),
      WaffoApiKey: values.WaffoApiKey.trim(),
      WaffoPrivateKey: values.WaffoPrivateKey.trim(),
      WaffoSandboxApiKey: values.WaffoSandboxApiKey.trim(),
      WaffoSandboxPrivateKey: values.WaffoSandboxPrivateKey.trim(),
      WaffoPayMethods: JSON.stringify(waffoPayMethods),
      WaffoPancakeMerchantID: values.WaffoPancakeMerchantID.trim(),
      WaffoPancakePrivateKey: values.WaffoPancakePrivateKey.trim(),
      WaffoPancakeReturnURL: removeTrailingSlash(
        values.WaffoPancakeReturnURL.trim()
      ),
    }

    const initial = {
      PayAddress: removeTrailingSlash(initialRef.current.PayAddress),
      EpayId: initialRef.current.EpayId.trim(),
      EpayKey: initialRef.current.EpayKey.trim(),
      Price: initialRef.current.Price,
      MinTopUp: initialRef.current.MinTopUp,
      CustomCallbackAddress: removeTrailingSlash(
        initialRef.current.CustomCallbackAddress
      ),
      PayMethods: initialRef.current.PayMethods.trim(),
      AmountOptions: initialRef.current.AmountOptions.trim(),
      AmountDiscount: initialRef.current.AmountDiscount.trim(),
      StripeApiSecret: initialRef.current.StripeApiSecret.trim(),
      StripeWebhookSecret: initialRef.current.StripeWebhookSecret.trim(),
      StripePriceId: initialRef.current.StripePriceId.trim(),
      StripeUnitPrice: initialRef.current.StripeUnitPrice,
      StripeMinTopUp: initialRef.current.StripeMinTopUp,
      StripePromotionCodesEnabled:
        initialRef.current.StripePromotionCodesEnabled,
      CreemApiKey: initialRef.current.CreemApiKey.trim(),
      CreemWebhookSecret: initialRef.current.CreemWebhookSecret.trim(),
      CreemTestMode: initialRef.current.CreemTestMode,
      CreemProducts: initialRef.current.CreemProducts.trim(),
      WaffoEnabled: initialRef.current.WaffoEnabled,
      WaffoSandbox: initialRef.current.WaffoSandbox,
      WaffoMerchantId: initialRef.current.WaffoMerchantId.trim(),
      WaffoCurrency: initialRef.current.WaffoCurrency.trim() || 'USD',
      WaffoUnitPrice: initialRef.current.WaffoUnitPrice,
      WaffoMinTopUp: initialRef.current.WaffoMinTopUp,
      WaffoNotifyUrl: initialRef.current.WaffoNotifyUrl.trim(),
      WaffoReturnUrl: initialRef.current.WaffoReturnUrl.trim(),
      WaffoPublicCert: initialRef.current.WaffoPublicCert.trim(),
      WaffoSandboxPublicCert: initialRef.current.WaffoSandboxPublicCert.trim(),
      WaffoApiKey: initialRef.current.WaffoApiKey.trim(),
      WaffoPrivateKey: initialRef.current.WaffoPrivateKey.trim(),
      WaffoSandboxApiKey: initialRef.current.WaffoSandboxApiKey.trim(),
      WaffoSandboxPrivateKey: initialRef.current.WaffoSandboxPrivateKey.trim(),
      WaffoPayMethods: JSON.stringify(
        parseWaffoPayMethods(waffoDefaultValues.WaffoPayMethods)
      ),
      WaffoPancakeMerchantID: initialRef.current.WaffoPancakeMerchantID.trim(),
      WaffoPancakePrivateKey: initialRef.current.WaffoPancakePrivateKey.trim(),
      WaffoPancakeReturnURL: removeTrailingSlash(
        initialRef.current.WaffoPancakeReturnURL.trim()
      ),
    }

    const updates: Array<{ key: string; value: string | number | boolean }> = []

    if (sanitized.PayAddress !== initial.PayAddress) {
      updates.push({ key: 'PayAddress', value: sanitized.PayAddress })
    }

    if (sanitized.EpayId !== initial.EpayId) {
      updates.push({ key: 'EpayId', value: sanitized.EpayId })
    }

    if (sanitized.EpayKey && sanitized.EpayKey !== initial.EpayKey) {
      updates.push({ key: 'EpayKey', value: sanitized.EpayKey })
    }

    if (sanitized.Price !== initial.Price) {
      updates.push({ key: 'Price', value: sanitized.Price })
    }

    if (sanitized.MinTopUp !== initial.MinTopUp) {
      updates.push({ key: 'MinTopUp', value: sanitized.MinTopUp })
    }

    if (sanitized.CustomCallbackAddress !== initial.CustomCallbackAddress) {
      updates.push({
        key: 'CustomCallbackAddress',
        value: sanitized.CustomCallbackAddress,
      })
    }

    if (
      normalizeJsonForComparison(sanitized.PayMethods) !==
      normalizeJsonForComparison(initial.PayMethods)
    ) {
      updates.push({ key: 'PayMethods', value: sanitized.PayMethods })
    }

    if (
      normalizeJsonForComparison(sanitized.AmountOptions) !==
      normalizeJsonForComparison(initial.AmountOptions)
    ) {
      updates.push({
        key: 'payment_setting.amount_options',
        value: sanitized.AmountOptions,
      })
    }

    if (
      normalizeJsonForComparison(sanitized.AmountDiscount) !==
      normalizeJsonForComparison(initial.AmountDiscount)
    ) {
      updates.push({
        key: 'payment_setting.amount_discount',
        value: sanitized.AmountDiscount,
      })
    }

    if (
      sanitized.StripeApiSecret &&
      sanitized.StripeApiSecret !== initial.StripeApiSecret
    ) {
      updates.push({ key: 'StripeApiSecret', value: sanitized.StripeApiSecret })
    }

    if (
      sanitized.StripeWebhookSecret &&
      sanitized.StripeWebhookSecret !== initial.StripeWebhookSecret
    ) {
      updates.push({
        key: 'StripeWebhookSecret',
        value: sanitized.StripeWebhookSecret,
      })
    }

    if (sanitized.StripePriceId !== initial.StripePriceId) {
      updates.push({ key: 'StripePriceId', value: sanitized.StripePriceId })
    }

    if (sanitized.StripeUnitPrice !== initial.StripeUnitPrice) {
      updates.push({ key: 'StripeUnitPrice', value: sanitized.StripeUnitPrice })
    }

    if (sanitized.StripeMinTopUp !== initial.StripeMinTopUp) {
      updates.push({ key: 'StripeMinTopUp', value: sanitized.StripeMinTopUp })
    }

    if (
      sanitized.StripePromotionCodesEnabled !==
      initial.StripePromotionCodesEnabled
    ) {
      updates.push({
        key: 'StripePromotionCodesEnabled',
        value: sanitized.StripePromotionCodesEnabled,
      })
    }

    if (
      sanitized.CreemApiKey &&
      sanitized.CreemApiKey !== initial.CreemApiKey
    ) {
      updates.push({ key: 'CreemApiKey', value: sanitized.CreemApiKey })
    }

    if (
      sanitized.CreemWebhookSecret &&
      sanitized.CreemWebhookSecret !== initial.CreemWebhookSecret
    ) {
      updates.push({
        key: 'CreemWebhookSecret',
        value: sanitized.CreemWebhookSecret,
      })
    }

    if (sanitized.CreemTestMode !== initial.CreemTestMode) {
      updates.push({ key: 'CreemTestMode', value: sanitized.CreemTestMode })
    }

    if (
      normalizeJsonForComparison(sanitized.CreemProducts) !==
      normalizeJsonForComparison(initial.CreemProducts)
    ) {
      updates.push({ key: 'CreemProducts', value: sanitized.CreemProducts })
    }

    if (sanitized.WaffoEnabled !== initial.WaffoEnabled) {
      updates.push({ key: 'WaffoEnabled', value: sanitized.WaffoEnabled })
    }

    if (sanitized.WaffoSandbox !== initial.WaffoSandbox) {
      updates.push({ key: 'WaffoSandbox', value: sanitized.WaffoSandbox })
    }

    if (sanitized.WaffoMerchantId !== initial.WaffoMerchantId) {
      updates.push({ key: 'WaffoMerchantId', value: sanitized.WaffoMerchantId })
    }

    if (sanitized.WaffoCurrency !== initial.WaffoCurrency) {
      updates.push({ key: 'WaffoCurrency', value: sanitized.WaffoCurrency })
    }

    if (sanitized.WaffoUnitPrice !== initial.WaffoUnitPrice) {
      updates.push({ key: 'WaffoUnitPrice', value: sanitized.WaffoUnitPrice })
    }

    if (sanitized.WaffoMinTopUp !== initial.WaffoMinTopUp) {
      updates.push({ key: 'WaffoMinTopUp', value: sanitized.WaffoMinTopUp })
    }

    if (sanitized.WaffoNotifyUrl !== initial.WaffoNotifyUrl) {
      updates.push({ key: 'WaffoNotifyUrl', value: sanitized.WaffoNotifyUrl })
    }

    if (sanitized.WaffoReturnUrl !== initial.WaffoReturnUrl) {
      updates.push({ key: 'WaffoReturnUrl', value: sanitized.WaffoReturnUrl })
    }

    if (sanitized.WaffoPublicCert !== initial.WaffoPublicCert) {
      updates.push({ key: 'WaffoPublicCert', value: sanitized.WaffoPublicCert })
    }

    if (sanitized.WaffoSandboxPublicCert !== initial.WaffoSandboxPublicCert) {
      updates.push({
        key: 'WaffoSandboxPublicCert',
        value: sanitized.WaffoSandboxPublicCert,
      })
    }

    if (sanitized.WaffoApiKey) {
      updates.push({ key: 'WaffoApiKey', value: sanitized.WaffoApiKey })
    }

    if (sanitized.WaffoPrivateKey) {
      updates.push({ key: 'WaffoPrivateKey', value: sanitized.WaffoPrivateKey })
    }

    if (sanitized.WaffoSandboxApiKey) {
      updates.push({
        key: 'WaffoSandboxApiKey',
        value: sanitized.WaffoSandboxApiKey,
      })
    }

    if (sanitized.WaffoSandboxPrivateKey) {
      updates.push({
        key: 'WaffoSandboxPrivateKey',
        value: sanitized.WaffoSandboxPrivateKey,
      })
    }

    if (
      normalizeJsonForComparison(sanitized.WaffoPayMethods) !==
      normalizeJsonForComparison(initial.WaffoPayMethods)
    ) {
      updates.push({ key: 'WaffoPayMethods', value: sanitized.WaffoPayMethods })
    }

    const hasWaffoPancakeChanges =
      sanitized.WaffoPancakeMerchantID !== initial.WaffoPancakeMerchantID ||
      sanitized.WaffoPancakePrivateKey.length > 0 ||
      sanitized.WaffoPancakeReturnURL !== initial.WaffoPancakeReturnURL ||
      waffoPancakeSelection.storeID !== waffoPancakeSavedBinding.storeID ||
      waffoPancakeSelection.productID !== waffoPancakeSavedBinding.productID
    const hasAlipayChanges = hasPaymentConfigChanges(alipayForm, alipayConfig)
    const hasWechatChanges = hasPaymentConfigChanges(wechatForm, wechatConfig)

    if (
      updates.length === 0 &&
      !hasWaffoPancakeChanges &&
      !hasAlipayChanges &&
      !hasWechatChanges
    ) {
      toast.info(t('No changes to save'))
      return
    }

    for (const update of updates) {
      await updateOption.mutateAsync(update)
    }

    if (hasAlipayChanges) {
      await saveAlipayConfig()
    }

    if (hasWechatChanges) {
      await saveWechatConfig()
    }

    if (!hasWaffoPancakeChanges) {
      return
    }

    if (!sanitized.WaffoPancakeMerchantID) {
      toast.error(t('Merchant ID is required'))
      return
    }

    if (!waffoPancakeSelection.storeID || !waffoPancakeSelection.productID) {
      toast.error(t('Pick or create both a store and a product before saving.'))
      return
    }

    try {
      const body = await saveWaffoPancakeConfig({
        merchantID: sanitized.WaffoPancakeMerchantID,
        privateKey: sanitized.WaffoPancakePrivateKey,
        returnURL: sanitized.WaffoPancakeReturnURL,
        storeID: waffoPancakeSelection.storeID,
        productID: waffoPancakeSelection.productID,
      })

      if (
        body?.message === 'success' &&
        typeof body.data === 'object' &&
        body.data
      ) {
        const saved = body.data as { product_id: string; store_id: string }
        const savedBinding = {
          storeID: saved.store_id,
          productID: saved.product_id,
        }
        setWaffoPancakeSavedBinding(savedBinding)
        setWaffoPancakeSelection(savedBinding)
        queryClient.invalidateQueries({ queryKey: ['system-options'] })
        toast.success(t('Waffo Pancake settings saved'))
        return
      }

      const reason = typeof body?.data === 'string' ? body.data : undefined
      toast.error(
        reason
          ? `${t('Waffo Pancake save failed')}: ${reason}`
          : t('Waffo Pancake save failed')
      )
    } catch (error) {
      toast.error(
        `${t('Waffo Pancake save failed')}: ${
          error instanceof Error ? error.message : String(error)
        }`
      )
    }
  }

  const currentFormValues = form.watch()
  const wechatOfficialAccountServerURL =
    typeof window === 'undefined'
      ? 'https://api.35sz.top/api/auth/wechat-callback'
      : `${window.location.origin}/api/auth/wechat-callback`
  const waffoValues: WaffoSettingsValues = {
    WaffoEnabled: currentFormValues.WaffoEnabled,
    WaffoApiKey: currentFormValues.WaffoApiKey,
    WaffoPrivateKey: currentFormValues.WaffoPrivateKey,
    WaffoPublicCert: currentFormValues.WaffoPublicCert,
    WaffoSandboxPublicCert: currentFormValues.WaffoSandboxPublicCert,
    WaffoSandboxApiKey: currentFormValues.WaffoSandboxApiKey,
    WaffoSandboxPrivateKey: currentFormValues.WaffoSandboxPrivateKey,
    WaffoSandbox: currentFormValues.WaffoSandbox,
    WaffoMerchantId: currentFormValues.WaffoMerchantId,
    WaffoCurrency: currentFormValues.WaffoCurrency,
    WaffoUnitPrice: currentFormValues.WaffoUnitPrice,
    WaffoMinTopUp: currentFormValues.WaffoMinTopUp,
    WaffoNotifyUrl: currentFormValues.WaffoNotifyUrl,
    WaffoReturnUrl: currentFormValues.WaffoReturnUrl,
    WaffoPayMethods: JSON.stringify(waffoPayMethods),
  }
  const waffoPancakeValues: WaffoPancakeSettingsValues = {
    WaffoPancakeMerchantID: currentFormValues.WaffoPancakeMerchantID,
    WaffoPancakePrivateKey: currentFormValues.WaffoPancakePrivateKey,
    WaffoPancakeReturnURL: currentFormValues.WaffoPancakeReturnURL,
  }

  return (
    <>
    <SettingsSection title={t('Payment Gateway')}>
      {!complianceConfirmed ? (
        <Alert variant='destructive' className='mb-6'>
          <ShieldAlert className='h-4 w-4' />
          <AlertTitle>{t('Compliance confirmation required')}</AlertTitle>
          <AlertDescription>
            <div className='space-y-3'>
              <p>
                {t(
                  'Payment, redemption codes, subscription plans, and invitation rewards are locked until the root administrator confirms the compliance terms.'
                )}
              </p>
              <ol className='list-decimal space-y-1 pl-5'>
                {complianceStatements.map((statement) => (
                  <li key={statement}>{statement}</li>
                ))}
              </ol>
            </div>
          </AlertDescription>
          <AlertAction>
            <Button
              type='button'
              size='sm'
              variant='destructive'
              onClick={() => setShowComplianceDialog(true)}
            >
              {t('Confirm compliance')}
            </Button>
          </AlertAction>
        </Alert>
      ) : (
        <Alert className='mb-6'>
          <AlertTitle>{t('Compliance confirmed')}</AlertTitle>
          <AlertDescription>
            {t('Confirmed at {{time}} by user #{{userId}}', {
              time: complianceDefaults.confirmedAt
                ? new Date(
                    complianceDefaults.confirmedAt * 1000
                  ).toLocaleString()
                : '-',
              userId: complianceDefaults.confirmedBy || '-',
            })}
          </AlertDescription>
        </Alert>
      )}

      <RiskAcknowledgementDialog
        open={showComplianceDialog}
        onOpenChange={setShowComplianceDialog}
        title={t('Confirm compliance terms')}
        description={t(
          'This confirmation unlocks payment, redemption code, subscription plan, and invitation reward features. Please read the statements carefully.'
        )}
        items={complianceStatements}
        requiredText={complianceRequiredText}
        requiredTextParts={complianceRequiredTextParts}
        inputPrompt={t('Please type the following text to confirm:')}
        inputPlaceholder={t('Type the confirmation text here')}
        mismatchHint={t('The entered text does not match the required text.')}
        confirmText={t('Confirm and enable')}
        isLoading={confirmComplianceMutation.isPending}
        onConfirm={() => confirmComplianceMutation.mutate()}
      />

      <Form {...form}>
        <SettingsForm
          onSubmit={form.handleSubmit(onSubmit)}
          className={cn(
            'gap-y-8',
            !complianceConfirmed && 'pointer-events-none opacity-40'
          )}
          data-no-autosubmit='true'
        >
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={isPaymentSaving}
            saveLabel='Save all settings'
          />
          <div className='space-y-4'>
            <div>
              <h3 className='text-lg font-medium'>{t('General Settings')}</h3>
              <p className='text-muted-foreground text-sm'>
                {t('Shared configuration for all payment gateways')}
              </p>
            </div>

            <div className='grid gap-6 md:grid-cols-2'>
              <FormField
                control={form.control}
                name='Price'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Price (local currency / USD)')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        step='0.01'
                        min={0}
                        {...safeNumberFieldProps(field)}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'How much to charge for each US dollar of balance (Epay)'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='MinTopUp'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Minimum top-up (USD)')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        step='0.01'
                        min={0}
                        {...safeNumberFieldProps(field)}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('Smallest USD amount users can recharge (Epay)')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            <div className='grid gap-6 md:grid-cols-2 md:items-start'>
              <FormField
                control={form.control}
                name='AmountOptions'
                render={({ field }) => (
                  <FormItem>
                    <div className='mb-2 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between'>
                      <FormLabel>{t('Top-up amount options')}</FormLabel>
                      <Button
                        type='button'
                        variant='outline'
                        size='sm'
                        onClick={() =>
                          setAmountOptionsVisualMode(!amountOptionsVisualMode)
                        }
                        className='w-full sm:w-auto'
                      >
                        {amountOptionsVisualMode ? (
                          <>
                            <Code2 className='mr-2 h-3 w-3' />
                            {t('JSON Editor')}
                          </>
                        ) : (
                          <>
                            <Eye className='mr-2 h-3 w-3' />
                            {t('Visual Editor')}
                          </>
                        )}
                      </Button>
                    </div>
                    <FormControl>
                      {amountOptionsVisualMode ? (
                        <AmountOptionsVisualEditor
                          value={field.value}
                          onChange={field.onChange}
                        />
                      ) : (
                        <Textarea
                          rows={4}
                          placeholder='[10, 20, 50, 100]'
                          {...field}
                          onChange={(event) =>
                            field.onChange(event.target.value)
                          }
                        />
                      )}
                    </FormControl>
                    <FormDescription>
                      {t('Preset recharge amounts (JSON array)')}{' '}
                      {t(
                        'Displayed on the wallet page only when at least one amount-based payment gateway is enabled.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='AmountDiscount'
                render={({ field }) => (
                  <FormItem>
                    <div className='mb-2 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between'>
                      <FormLabel>{t('Amount discount')}</FormLabel>
                      <Button
                        type='button'
                        variant='outline'
                        size='sm'
                        onClick={() =>
                          setAmountDiscountVisualMode(!amountDiscountVisualMode)
                        }
                        className='w-full sm:w-auto'
                      >
                        {amountDiscountVisualMode ? (
                          <>
                            <Code2 className='mr-2 h-3 w-3' />
                            {t('JSON Editor')}
                          </>
                        ) : (
                          <>
                            <Eye className='mr-2 h-3 w-3' />
                            {t('Visual Editor')}
                          </>
                        )}
                      </Button>
                    </div>
                    <FormControl>
                      {amountDiscountVisualMode ? (
                        <AmountDiscountVisualEditor
                          value={field.value}
                          onChange={field.onChange}
                        />
                      ) : (
                        <Textarea
                          rows={4}
                          placeholder='{"100":0.95,"200":0.9}'
                          {...field}
                          onChange={(event) =>
                            field.onChange(event.target.value)
                          }
                        />
                      )}
                    </FormControl>
                    <FormDescription>
                      {t('Discount map by recharge amount (JSON object)')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
          </div>

          <Separator />

          <div className='space-y-4'>
            <div className='flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between'>
              <div>
                <h3 className='text-lg font-medium'>{t('Payment Channels')}</h3>
                <p className='text-muted-foreground text-sm'>
                  {t('Enable a payment channel first, then fill in the required configuration fields.')}
                </p>
              </div>
              <Button
                type='button'
                variant='outline'
                size='sm'
                onClick={form.handleSubmit(onSubmit)}
                disabled={isPaymentSaving}
                className='w-full sm:w-auto'
              >
                <Save className='mr-2 h-3 w-3' />
                {isPaymentSaving
                  ? t('Saving...')
                  : t('Save payment channel settings')}
              </Button>
            </div>

            <Tabs value={activePaymentGateway} onValueChange={setActivePaymentGateway}>
              <TabsList className='grid w-full grid-cols-2 gap-1 sm:grid-cols-4 lg:grid-cols-7'>
                <TabsTrigger value='epay'>{t('Epay')}</TabsTrigger>
                <TabsTrigger value='alipay'>{t('Alipay')}</TabsTrigger>
                <TabsTrigger value='wechat'>{t('WeChat Pay')}</TabsTrigger>
                <TabsTrigger value='stripe'>{t('Stripe')}</TabsTrigger>
                <TabsTrigger value='creem'>{t('Creem')}</TabsTrigger>
                <TabsTrigger value='waffo'>{t('Waffo')}</TabsTrigger>
                <TabsTrigger value='waffo-pancake'>{t('Waffo Pancake')}</TabsTrigger>
              </TabsList>

              <TabsContent value='epay' className='space-y-4 pt-4'>
                <div>
                  <h3 className='text-lg font-medium'>{t('Epay Gateway')}</h3>
                  <p className='text-muted-foreground text-sm'>
                    {t('Configuration for Epay payment integration')}
                  </p>
                </div>
                <GatewaySwitchRow
                  title={t('Enable Epay Gateway')}
                  description={t('Enable legacy Epay payment methods for wallet recharge.')}
                  checked={epayEnabled}
                  onCheckedChange={setEpayEnabled}
                />

                {epayEnabled && (
                  <>
            <div className='grid gap-6 md:grid-cols-2'>
              <FormField
                control={form.control}
                name='PayAddress'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Epay endpoint')}</FormLabel>
                    <FormControl>
                      <Input
                        placeholder={t('https://pay.example.com')}
                        {...field}
                        onChange={(event) => field.onChange(event.target.value)}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('Base address provided by your Epay service')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='CustomCallbackAddress'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Callback address')}</FormLabel>
                    <FormControl>
                      <Input
                        placeholder={t('https://gateway.example.com')}
                        {...field}
                        onChange={(event) => field.onChange(event.target.value)}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Optional callback override. Leave blank to use server address'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            <div className='grid gap-6 md:grid-cols-2'>
              <FormField
                control={form.control}
                name='EpayId'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Epay merchant ID')}</FormLabel>
                    <FormControl>
                      <Input
                        placeholder='10001'
                        autoComplete='off'
                        {...field}
                        onChange={(event) => field.onChange(event.target.value)}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='EpayKey'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Epay secret key')}</FormLabel>
                    <FormControl>
                      <Input
                        type='password'
                        placeholder={t('Enter new key to update')}
                        autoComplete='new-password'
                        {...field}
                        onChange={(event) => field.onChange(event.target.value)}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('Leave blank unless rotating the secret')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            <FormField
              control={form.control}
              name='PayMethods'
              render={({ field }) => (
                <FormItem>
                  <div className='mb-2 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between'>
                    <FormLabel>{t('Epay payment methods')}</FormLabel>
                    <Button
                      type='button'
                      variant='outline'
                      size='sm'
                      onClick={() =>
                        setPayMethodsVisualMode(!payMethodsVisualMode)
                      }
                      className='w-full sm:w-auto'
                    >
                      {payMethodsVisualMode ? (
                        <>
                          <Code2 className='mr-2 h-3 w-3' />
                          {t('JSON Editor')}
                        </>
                      ) : (
                        <>
                          <Eye className='mr-2 h-3 w-3' />
                          {t('Visual Editor')}
                        </>
                      )}
                    </Button>
                  </div>
                  <FormControl>
                    {payMethodsVisualMode ? (
                      <PaymentMethodsVisualEditor
                        value={field.value}
                        onChange={field.onChange}
                      />
                    ) : (
                      <Textarea
                        rows={4}
                        placeholder={t(
                          '[{"name":"支付宝","type":"alipay","color":"#1677FF"}]'
                        )}
                        {...field}
                        onChange={(event) => field.onChange(event.target.value)}
                      />
                    )}
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Only used by the legacy Epay gateway. Native Alipay and WeChat Pay are configured in their own gateway tabs.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
                  </>
                )}
          </TabsContent>

          <TabsContent value='alipay' className='space-y-4 pt-4'>
            <div>
              <h3 className='text-lg font-medium'>
                {t('Alipay Gateway')}
              </h3>
              <p className='text-muted-foreground text-sm'>
                {t('Configure Alipay merchant credentials for wallet recharge.')}
              </p>
            </div>
            <GatewaySwitchRow
              title={t('Enable Alipay')}
              description={t('Enable Alipay payment method for wallet top-up.')}
              checked={alipayForm.enabled ?? false}
              onCheckedChange={(checked) => setAlipayForm(prev => ({ ...prev, enabled: checked }))}
            />

            {(alipayForm.enabled || alipayConfig?.enabled) ? (
              <div className='space-y-4 pt-2'>
                <div className='grid gap-4 sm:grid-cols-2'>
                  <div className='space-y-2'>
                    <Label>{t('Display name')}</Label>
                    <Input value={getNativePaymentDisplayName('alipay')} disabled />
                    <p className='text-muted-foreground text-xs'>
                      {t('Display name is fixed by the selected payment channel.')}
                    </p>
                  </div>
                  <div className='space-y-2'>
                    <Label>{t('Sort order')}</Label>
                    <Input
                      type='number'
                      value={alipayForm.sort_order ?? 10}
                      onChange={(event) => setAlipayForm(prev => ({ ...prev, sort_order: Number(event.target.value) || 0 }))}
                    />
                  </div>
                </div>

                <PaymentIconUploadField
                  label={t('Icon')}
                  value={alipayForm.icon_url || ''}
                  onChange={(value) =>
                    setAlipayForm((prev) => ({ ...prev, icon_url: value }))
                  }
                />

                <div className='space-y-2'>
                  <Label>{t('App ID')}</Label>
                  <Input
                    value={alipayForm.app_id || ''}
                    onChange={(event) => setAlipayForm(prev => ({ ...prev, app_id: event.target.value }))}
                  />
                </div>
                <div className='space-y-2'>
                  <Label>{t('App private key')}</Label>
                  <Textarea
                    rows={4}
                    value={alipayForm.app_private_key || ''}
                    onChange={(event) => setAlipayForm(prev => ({ ...prev, app_private_key: event.target.value }))}
                    placeholder={alipayConfig ? t('Leave blank to keep the existing key') : t('Enter App private key')}
                    className='font-mono text-xs'
                  />
                </div>
                <div className='space-y-2'>
                  <Label>{t('Alipay public key')}</Label>
                  <Textarea
                    rows={4}
                    value={alipayForm.alipay_public_key || ''}
                    onChange={(event) => setAlipayForm(prev => ({ ...prev, alipay_public_key: event.target.value }))}
                    className='font-mono text-xs'
                  />
                </div>
                <div className='grid gap-4 sm:grid-cols-3'>
                  <div className='space-y-2'>
                    <Label>{t('App public cert')}</Label>
                    <Textarea
                      rows={3}
                      value={alipayForm.alipay_app_public_cert || ''}
                      onChange={(event) => setAlipayForm(prev => ({ ...prev, alipay_app_public_cert: event.target.value }))}
                      className='font-mono text-xs'
                    />
                  </div>
                  <div className='space-y-2'>
                    <Label>{t('Alipay public cert')}</Label>
                    <Textarea
                      rows={3}
                      value={alipayForm.alipay_public_cert || ''}
                      onChange={(event) => setAlipayForm(prev => ({ ...prev, alipay_public_cert: event.target.value }))}
                      className='font-mono text-xs'
                    />
                  </div>
                  <div className='space-y-2'>
                    <Label>{t('Alipay root cert')}</Label>
                    <Textarea
                      rows={3}
                      value={alipayForm.alipay_root_cert || ''}
                      onChange={(event) => setAlipayForm(prev => ({ ...prev, alipay_root_cert: event.target.value }))}
                      className='font-mono text-xs'
                    />
                  </div>
                </div>

                <div className='grid gap-4 sm:grid-cols-3'>
                  <div className='space-y-2'>
                    <Label>{t('Gateway URL')}</Label>
                    <Input
                      value={alipayForm.gateway_url || ''}
                      onChange={(event) => setAlipayForm(prev => ({ ...prev, gateway_url: event.target.value }))}
                    />
                  </div>
                  <div className='space-y-2'>
                    <Label>{t('Notify URL')}</Label>
                    <Input
                      value={alipayForm.notify_url || ''}
                      onChange={(event) => setAlipayForm(prev => ({ ...prev, notify_url: event.target.value }))}
                    />
                  </div>
                  <div className='space-y-2'>
                    <Label>{t('Return URL')}</Label>
                    <Input
                      value={alipayForm.return_url || ''}
                      onChange={(event) => setAlipayForm(prev => ({ ...prev, return_url: event.target.value }))}
                    />
                  </div>
                </div>
              </div>
            ) : null}
          </TabsContent>

          <TabsContent value='wechat' className='space-y-4 pt-4'>
            <div>
              <h3 className='text-lg font-medium'>
                {t('WeChat Pay Gateway')}
              </h3>
              <p className='text-muted-foreground text-sm'>
                {t('Configure WeChat Pay merchant credentials for wallet recharge.')}
              </p>
            </div>
            <GatewaySwitchRow
              title={t('Enable WeChat Pay')}
              description={t('Enable WeChat Pay payment method for wallet top-up.')}
              checked={wechatForm.enabled ?? false}
              onCheckedChange={(checked) => setWechatForm(prev => ({ ...prev, enabled: checked }))}
            />

            {(wechatForm.enabled || wechatConfig?.enabled) ? (
              <div className='space-y-4 pt-2'>
                <div className='grid gap-4 sm:grid-cols-2'>
                  <div className='space-y-2'>
                    <Label>{t('Display name')}</Label>
                    <Input value={getNativePaymentDisplayName('wxpay')} disabled />
                    <p className='text-muted-foreground text-xs'>
                      {t('Display name is fixed by the selected payment channel.')}
                    </p>
                  </div>
                  <div className='space-y-2'>
                    <Label>{t('Sort order')}</Label>
                    <Input
                      type='number'
                      value={wechatForm.sort_order ?? 20}
                      onChange={(event) => setWechatForm(prev => ({ ...prev, sort_order: Number(event.target.value) || 0 }))}
                    />
                  </div>
                </div>

                <PaymentIconUploadField
                  label={t('Icon')}
                  value={wechatForm.icon_url || ''}
                  onChange={(value) =>
                    setWechatForm((prev) => ({ ...prev, icon_url: value }))
                  }
                />

                <div className='space-y-3 rounded-md border p-4'>
                  <div>
                    <h4 className='text-sm font-medium'>{t('WeChat application configuration')}</h4>
                    <p className='text-muted-foreground text-xs'>
                      {t('Enter the AppID of the Official Account, Mini Program, or WeChat Open Platform website application bound to the WeChat merchant ID. Native QR payment only requires AppID; AppSecret is only used for WeChat login, web authorization, or JSAPI payment.')}
                    </p>
                  </div>
                  <div className='grid gap-4 sm:grid-cols-2'>
                    <div className='space-y-2'>
                      <RequiredLabel>{t('App ID')}</RequiredLabel>
                      <Input
                        value={wechatForm.wechat_app_id || ''}
                        onChange={(event) => setWechatForm(prev => ({ ...prev, wechat_app_id: event.target.value }))}
                      />
                    </div>
                    <div className='space-y-2'>
                      <Label>{t('AppSecret')}</Label>
                      <Input
                        type='password'
                        value={wechatForm.wechat_app_secret || ''}
                        onChange={(event) => setWechatForm(prev => ({ ...prev, wechat_app_secret: event.target.value }))}
                        placeholder={wechatConfig ? t('Leave blank to keep the existing key') : t('Enter AppSecret')}
                      />
                      <p className='text-muted-foreground text-xs'>
                        {t('Native QR payment does not require AppSecret. It is only needed for Official Account authorization, WeChat login, or JSAPI payment.')}
                      </p>
                    </div>
                  </div>
                </div>

                <div className='space-y-3 rounded-md border p-4'>
                  <div>
                    <h4 className='text-sm font-medium'>{t('WeChat merchant platform configuration')}</h4>
                    <p className='text-muted-foreground text-xs'>
                      {t('Configure merchant credentials from WeChat Pay merchant platform.')}
                    </p>
                  </div>
                  <div className='space-y-2'>
                    <RequiredLabel>{t('WeChat merchant ID (MCHID)')}</RequiredLabel>
                    <Input
                      value={wechatForm.wechat_mch_id || ''}
                      onChange={(event) => setWechatForm(prev => ({ ...prev, wechat_mch_id: event.target.value }))}
                    />
                  </div>
                  <div className='space-y-2'>
                    <Label>{t('Authentication mode')}</Label>
                    <Select
                      items={[
                        { value: 'certificate', label: t('Platform certificate mode') },
                        { value: 'public_key', label: t('WeChat Pay public key mode') },
                      ]}
                      value={wechatForm.wechat_auth_mode || 'certificate'}
                      onValueChange={(value) =>
                        value &&
                        setWechatForm((prev) => ({
                          ...prev,
                          wechat_auth_mode: value as 'certificate' | 'public_key',
                        }))
                      }
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value='certificate'>
                          {t('Platform certificate mode')}
                        </SelectItem>
                        <SelectItem value='public_key'>
                          {t('WeChat Pay public key mode')}
                        </SelectItem>
                      </SelectContent>
                    </Select>
                    <p className='text-muted-foreground text-xs'>
                      {t('Public key mode uses PUB_KEY_ID and the WeChat Pay public key for response verification.')}
                    </p>
                  </div>

                  <div className='space-y-2'>
                    <RequiredLabel>{t('WeChat Pay certificate (apiclient_cert.pem)')}</RequiredLabel>
                    <WechatPemUploadField
                      expectedKind='cert'
                      onLoad={(content) =>
                        setWechatForm((prev) => ({ ...prev, wechat_cert: content }))
                      }
                    />
                    <Textarea
                      rows={4}
                      value={wechatForm.wechat_cert || ''}
                      onChange={(event) => setWechatForm(prev => ({ ...prev, wechat_cert: event.target.value }))}
                      placeholder='-----BEGIN CERTIFICATE-----'
                      className='font-mono text-xs'
                    />
                    <p className='text-muted-foreground text-xs'>
                      {t('Paste apiclient_cert.pem content. The merchant API certificate serial number can be parsed from it.')}
                    </p>
                  </div>
                </div>

                <div className='space-y-2'>
                  {(wechatForm.wechat_auth_mode || 'certificate') === 'public_key' ? (
                    <Label>{t('Merchant API key (paySignKey) optional')}</Label>
                  ) : (
                    <RequiredLabel>{t('Merchant API key (paySignKey)')}</RequiredLabel>
                  )}
                  <WechatPemUploadField
                    expectedKind='key'
                    onLoad={(content) =>
                      setWechatForm((prev) => ({ ...prev, wechat_api_key: content }))
                    }
                  />
                  <Textarea
                    rows={3}
                    value={wechatForm.wechat_api_key || ''}
                    onChange={(event) => setWechatForm(prev => ({ ...prev, wechat_api_key: event.target.value }))}
                    placeholder={wechatConfig ? t('Leave blank to keep the existing key') : t('Enter APIv3 Key')}
                    className='font-mono text-xs'
                  />
                  <p className='text-muted-foreground text-xs'>
                    {(wechatForm.wechat_auth_mode || 'certificate') === 'public_key'
                      ? t('Public key mode does not use APIv3 Key for response signature verification. Keep it only if encrypted notifications need to be decrypted.')
                      : t('APIv3 Key is used to decrypt payment notifications. It is different from PUB_KEY_ID.')}
                  </p>
                </div>

                {(wechatForm.wechat_auth_mode || 'certificate') === 'public_key' ? (
                  <div className='grid gap-4 sm:grid-cols-2'>
                    <div className='space-y-2'>
                      <RequiredLabel>{t('WeChat Pay public key ID')}</RequiredLabel>
                      <Input
                        value={wechatForm.wechat_public_key_id || ''}
                        onChange={(event) => setWechatForm(prev => ({ ...prev, wechat_public_key_id: event.target.value }))}
                        placeholder='PUB_KEY_ID_...'
                      />
                    </div>
                    <div className='space-y-2 sm:col-span-2'>
                      <RequiredLabel>{t('WeChat Pay public key')}</RequiredLabel>
                      <Textarea
                        rows={5}
                        value={wechatForm.wechat_public_key || ''}
                        onChange={(event) => setWechatForm(prev => ({ ...prev, wechat_public_key: event.target.value }))}
                        placeholder='-----BEGIN PUBLIC KEY-----'
                        className='font-mono text-xs'
                      />
                    </div>
                  </div>
                ) : null}
                <div className='space-y-2'>
                  <RequiredLabel>{t('WeChat Pay certificate private key (apiclient_key.pem)')}</RequiredLabel>
                  <WechatPemUploadField
                    expectedKind='key'
                    onLoad={(content) =>
                      setWechatForm((prev) => ({ ...prev, wechat_private_key: content }))
                    }
                  />
                  <Textarea
                    rows={4}
                    value={wechatForm.wechat_private_key || ''}
                    onChange={(event) => setWechatForm(prev => ({ ...prev, wechat_private_key: event.target.value }))}
                    className='font-mono text-xs'
                  />
                </div>

                  <div className='space-y-2'>
                    <Label>{t('Official Account server URL')}</Label>
                    <div className='flex gap-2'>
                      <Input value={wechatOfficialAccountServerURL} readOnly />
                      <Button
                        type='button'
                        variant='outline'
                        size='icon'
                        onClick={() => {
                          void navigator.clipboard.writeText(
                            wechatOfficialAccountServerURL
                          )
                          toast.success(t('Copied'))
                        }}
                      >
                        <Copy className='h-4 w-4' />
                      </Button>
                    </div>
                    <p className='text-muted-foreground text-xs'>
                      {t('Log in to WeChat Official Account Platform, then go to Development > Basic Configuration > Server Configuration and paste this URL.')}
                    </p>
                  </div>
                </div>
            ) : null}
          </TabsContent>

          <TabsContent value='stripe' className='space-y-4 pt-4'>
            <div>
              <h3 className='text-lg font-medium'>{t('Stripe Gateway')}</h3>
              <p className='text-muted-foreground text-sm'>
                {t('Configuration for Stripe payment integration')}
              </p>
            </div>
            <GatewaySwitchRow
              title={t('Enable Stripe')}
              description={t('Enable Stripe payment method for wallet top-up.')}
              checked={stripeEnabled}
              onCheckedChange={setStripeEnabled}
            />

            {stripeEnabled ? (
              <>
                <div className='rounded-md bg-blue-50 p-4 text-sm text-blue-900 dark:bg-blue-950 dark:text-blue-100'>
              <p className='mb-2 font-medium'>{t('Webhook Configuration:')}</p>
              <ul className='list-inside list-disc space-y-1'>
                <li>
                  {t('Webhook URL:')}{' '}
                  <code className='rounded bg-blue-100 px-1 py-0.5 text-xs dark:bg-blue-900'>
                    {'<ServerAddress>/api/stripe/webhook'}
                  </code>
                </li>
                <li>
                  {t('Required events:')}{' '}
                  <code className='rounded bg-blue-100 px-1 py-0.5 text-xs dark:bg-blue-900'>
                    {t('checkout.session.completed')}
                  </code>{' '}
                  {t('and')}{' '}
                  <code className='rounded bg-blue-100 px-1 py-0.5 text-xs dark:bg-blue-900'>
                    {t('checkout.session.expired')}
                  </code>
                </li>
                <li>
                  {t('Configure at:')}{' '}
                  <a
                    href='https://dashboard.stripe.com/developers'
                    target='_blank'
                    rel='noreferrer'
                    className='underline hover:no-underline'
                  >
                    {t('Stripe Dashboard')}
                  </a>
                </li>
              </ul>
            </div>

            <div className='grid gap-6 md:grid-cols-3'>
              <FormField
                control={form.control}
                name='StripeApiSecret'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('API secret')}</FormLabel>
                    <FormControl>
                      <Input
                        type='password'
                        placeholder={t('sk_xxx or rk_xxx')}
                        autoComplete='new-password'
                        {...field}
                        onChange={(event) => field.onChange(event.target.value)}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('Stripe API key (leave blank unless updating)')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='StripeWebhookSecret'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Webhook secret')}</FormLabel>
                    <FormControl>
                      <Input
                        type='password'
                        placeholder={t('whsec_xxx')}
                        autoComplete='new-password'
                        {...field}
                        onChange={(event) => field.onChange(event.target.value)}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Webhook signing secret (leave blank unless updating)'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='StripePriceId'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Price ID')}</FormLabel>
                    <FormControl>
                      <Input
                        placeholder={t('price_xxx')}
                        {...field}
                        onChange={(event) => field.onChange(event.target.value)}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('Stripe product price ID')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            <div className='grid gap-6 md:grid-cols-3'>
              <FormField
                control={form.control}
                name='StripeUnitPrice'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t('Unit price (local currency / USD)')}
                    </FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        step='0.01'
                        min={0}
                        {...safeNumberFieldProps(field)}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('e.g., 8 means 8 local currency per USD')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='StripeMinTopUp'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Minimum top-up (USD)')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        step='0.01'
                        min={0}
                        {...safeNumberFieldProps(field)}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('Minimum recharge amount in USD')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='StripePromotionCodesEnabled'
                render={({ field }) => (
                  <SettingsSwitchItem>
                    <SettingsSwitchContent>
                      <FormLabel>{t('Promotion codes')}</FormLabel>
                      <FormDescription>
                        {t('Allow users to enter promo codes')}
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
              </>
            ) : null}
          </TabsContent>

          <TabsContent value='creem' className='space-y-4 pt-4'>
            <div>
              <h3 className='text-lg font-medium'>{t('Creem Gateway')}</h3>
              <p className='text-muted-foreground text-sm'>
                {t('Configuration for Creem payment integration')}
              </p>
            </div>
            <GatewaySwitchRow
              title={t('Enable Creem')}
              description={t('Enable Creem payment method for wallet top-up.')}
              checked={creemEnabled}
              onCheckedChange={setCreemEnabled}
            />

            {creemEnabled ? (
              <>
                <div className='rounded-md bg-blue-50 p-4 text-sm text-blue-900 dark:bg-blue-950 dark:text-blue-100'>
              <p className='mb-2 font-medium'>{t('Webhook Configuration:')}</p>
              <ul className='list-inside list-disc space-y-1'>
                <li>
                  {t('Webhook URL:')}{' '}
                  <code className='rounded bg-blue-100 px-1 py-0.5 text-xs dark:bg-blue-900'>
                    {'<ServerAddress>/api/creem/webhook'}
                  </code>
                </li>
                <li>{t('Configure in your Creem dashboard')}</li>
              </ul>
            </div>

            <div className='grid gap-6 md:grid-cols-2'>
              <FormField
                control={form.control}
                name='CreemApiKey'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('API Key')}</FormLabel>
                    <FormControl>
                      <Input
                        type='password'
                        placeholder={t('Enter Creem API key')}
                        autoComplete='new-password'
                        {...field}
                        onChange={(event) => field.onChange(event.target.value)}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('Creem API key (leave blank unless updating)')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='CreemWebhookSecret'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Webhook Secret')}</FormLabel>
                    <FormControl>
                      <Input
                        type='password'
                        placeholder={t('Enter webhook secret')}
                        autoComplete='new-password'
                        {...field}
                        onChange={(event) => field.onChange(event.target.value)}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Webhook signing secret (leave blank unless updating)'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            <FormField
              control={form.control}
              name='CreemTestMode'
              render={({ field }) => (
                <SettingsSwitchItem>
                  <SettingsSwitchContent>
                    <FormLabel>{t('Test Mode')}</FormLabel>
                    <FormDescription>
                      {t('Enable test mode for Creem payments')}
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

            <FormField
              control={form.control}
              name='CreemProducts'
              render={({ field }) => (
                <FormItem>
                  <div className='mb-2 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between'>
                    <FormLabel>{t('Products')}</FormLabel>
                    <Button
                      type='button'
                      variant='outline'
                      size='sm'
                      onClick={() =>
                        setCreemProductsVisualMode(!creemProductsVisualMode)
                      }
                      className='w-full sm:w-auto'
                    >
                      {creemProductsVisualMode ? (
                        <>
                          <Code2 className='mr-2 h-3 w-3' />
                          {t('JSON Editor')}
                        </>
                      ) : (
                        <>
                          <Eye className='mr-2 h-3 w-3' />
                          {t('Visual Editor')}
                        </>
                      )}
                    </Button>
                  </div>
                  <FormControl>
                    {creemProductsVisualMode ? (
                      <CreemProductsVisualEditor
                        value={field.value}
                        onChange={field.onChange}
                      />
                    ) : (
                      <Textarea
                        rows={4}
                        placeholder='[{"name":"Basic","productId":"prod_xxx","price":10,"quota":500000,"currency":"USD"}]'
                        {...field}
                        onChange={(event) => field.onChange(event.target.value)}
                      />
                    )}
                  </FormControl>
                  <FormDescription>
                    {t('Configure Creem products. Provide a JSON array.')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
              </>
            ) : null}
          </TabsContent>

          <TabsContent value='waffo-pancake' className='space-y-4 pt-4'>
            <GatewaySwitchRow
              title={t('Enable Waffo Pancake')}
              description={t('Enable Waffo Pancake MoR for wallet top-up.')}
              checked={waffoPancakeEnabled}
              onCheckedChange={setWaffoPancakeEnabled}
            />

            {waffoPancakeEnabled ? (
              <WaffoPancakeSettingsSection
                defaultValues={waffoPancakeDefaultValues}
                values={waffoPancakeValues}
                onValueChange={setWaffoPancakeValue}
                selectedBinding={waffoPancakeSelection}
                savedBinding={waffoPancakeSavedBinding}
                onSelectedBindingChange={setWaffoPancakeSelection}
              />
            ) : null}
          </TabsContent>

          <TabsContent value='waffo' className='space-y-4 pt-4'>
            <GatewaySwitchRow
              title={t('Enable Waffo')}
              description={t('Enable Waffo payment aggregator for wallet top-up.')}
              checked={currentFormValues.WaffoEnabled}
              onCheckedChange={(checked) => setPaymentValue('WaffoEnabled', checked)}
            />

            {currentFormValues.WaffoEnabled ? (
              <WaffoSettingsSection
                values={waffoValues}
                onValueChange={setWaffoValue}
                payMethods={waffoPayMethods}
                onPayMethodsChange={setWaffoPayMethods}
              />
            ) : null}
          </TabsContent>
        </Tabs>
      </div>
        </SettingsForm>
      </Form>
    </SettingsSection>
    </>
  )
}
