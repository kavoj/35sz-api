# Payment Settings Epay Methods Clarity Design

## Context

The payment settings page currently shows `Payment methods` under General Settings. Those methods are persisted as `PayMethods` and are only used by the legacy Epay gateway, but the label can be confused with the native Alipay and WeChat Pay gateway tabs.

The wallet top-up amount presets are also shared amount-based gateway settings, but they only appear on `/wallet` when at least one amount-based payment gateway is enabled. The admin UI should make that condition explicit.

## Goal

Make the payment settings structure clearer without changing backend storage or payment behavior.

## Design

### Epay payment methods placement

Move the existing `PayMethods` editor block from General Settings into the Epay tab, inside the `epayEnabled` configuration area.

Rename the label from `Payment methods` to `Epay payment methods`.

Update the description to:

> Only used by the legacy Epay gateway. Native Alipay and WeChat Pay are configured in their own gateway tabs.

This keeps the same form field, JSON editor/visual editor toggle, validation, and save key (`PayMethods`). Only the UI placement and copy change.

### Top-up amount options helper text

Keep `Top-up amount options` in General Settings because it is shared across amount-based gateways.

Append this explanatory sentence to its description:

> Displayed on the wallet page only when at least one amount-based payment gateway is enabled.

Chinese translation:

> 仅当至少启用一个按金额充值的支付网关时，会在钱包页显示。

### Payment Gateways translated copy

Keep the existing `t(...)` usage for:

- `Payment Gateways`
- `Enable a gateway first, then fill in the required configuration fields.`

Add or correct entries in `web/default/src/i18n/locales/en.json` and `web/default/src/i18n/locales/zh.json` so these strings display correctly in English and Chinese.

## Files

- `web/default/src/features/system-settings/integrations/payment-settings-section.tsx`
- `web/default/src/i18n/locales/en.json`
- `web/default/src/i18n/locales/zh.json`

## Verification

- `cd web/default && bun run typecheck`
- `cd web/default && bun run build`
- Manual UI check:
  - General Settings no longer contains Epay payment methods.
  - Epay tab shows `Epay payment methods` when Epay is enabled.
  - Top-up amount options description includes the wallet display condition.
  - Payment Gateways title and description display in Chinese when the UI language is Chinese.
