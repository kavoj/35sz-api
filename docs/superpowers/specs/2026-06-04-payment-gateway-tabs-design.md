# Payment Gateway Tabs Design

## Context

The payment settings page currently mixes several payment systems in a long form. Native Alipay/WeChat gateway settings are shown as cards with a separate configuration dialog, while Epay, Stripe, Creem, Waffo, and Waffo Pancake have their own inline sections. This makes it hard to understand which gateway is enabled and why `/wallet` may show "online topup is not enabled".

The OAuth settings page already uses a clearer pattern: one tab per integration, a visible enable switch at the top, and configuration fields shown in context. The payment page should follow the same interaction model.

## Goal

Refactor `/system-settings/billing/payment` so every payment gateway is represented by one tab with a top-level enable/disable switch. Only when a gateway is enabled should its detailed configuration fields be shown.

Gateways:

- Epay
- Alipay
- WeChat Pay
- Stripe
- Creem
- Waffo
- Waffo Pancake

## Design

### Layout

Add a `Tabs` section inside `PaymentSettingsSection`:

```text
Payment Gateways
[Epay] [Alipay] [WeChat] [Stripe] [Creem] [Waffo] [Waffo Pancake]
```

Each tab begins with a `SettingsSwitchItem`-style row:

```text
Enable <Gateway>
Description of what enabling this gateway does.
[Switch]
```

If disabled, show only the switch row and a short hint. If enabled, render the gateway's config fields below the switch.

### Backend persistence

No new backend table is required for legacy option-based gateways. Reuse existing option keys:

- Epay uses `PayAddress`, `EpayId`, `EpayKey`, `PayMethods`.
- Stripe uses `StripeApiSecret`, `StripeWebhookSecret`, `StripePriceId`, `StripeUnitPrice`, `StripeMinTopUp`, `StripePromotionCodesEnabled`.
- Creem uses `CreemApiKey`, `CreemWebhookSecret`, `CreemTestMode`, `CreemProducts`.
- Waffo uses existing `Waffo*` option keys and `WaffoEnabled`.
- Waffo Pancake uses existing `WaffoPancake*` option keys.
- Alipay and WeChat use the existing new `payment_configs` API and `enabled` field.

For gateways that do not currently have a dedicated enabled flag:

- Epay: add frontend-only `EpayEnabled` state derived from whether PayAddress/EpayId/EpayKey or PayMethods exist. Saving with disabled clears only gateway-specific values if the user explicitly disables it.
- Stripe: add frontend-only `StripeEnabled` derived from required Stripe fields.
- Creem: add frontend-only `CreemEnabled` derived from required Creem fields/products.
- Waffo Pancake: add frontend-only `WaffoPancakeEnabled` derived from merchant/private key/product binding.

To avoid accidental destructive config clearing, disabling a legacy gateway should set the effective gateway disabled state but preserve existing values unless the existing backend already has a dedicated enabled key. A future cleanup can add explicit backend enable flags.

### Alipay / WeChat

Move the current `PaymentConfigDialog` fields inline into tabs. The switch maps directly to `PaymentConfig.enabled`. Saving calls:

- `POST /api/payment-config/` if no config exists.
- `PUT /api/payment-config/:id` if config exists.

Sensitive masked fields remain unchanged by preserving existing backend behavior.

### Wallet behavior

After saving:

- `/api/user/topup/info` should expose `enable_alipay_topup` / `enable_wechat_topup` when native gateways are enabled and valid.
- The wallet UI already includes these flags in the online topup visibility condition after the recent fix.

## Scope

This design changes frontend structure and reuses existing backend APIs. It does not add new payment providers and does not change payment callback behavior.

## Verification

- `cd web/default && bun run typecheck`
- `cd web/default && bun run build`
- `./restart.sh`
- Open `/system-settings/billing/payment` and verify one tab per gateway.
- Toggle Alipay/WeChat on, fill required fields, save, reload, and verify enabled state persists.
- Toggle legacy gateways and verify fields hide/show without breaking save behavior.
- Open `/wallet` and verify enabled gateways appear as recharge options when backend config is valid.
