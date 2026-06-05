# Payment Channels Copy Design

## Context

The payment settings page currently shows both an outer `Payment Gateway` section title and an inner `Payment Gateways` tabs title. In Chinese both appear as `支付网关`, which creates duplicated wording and makes the hierarchy unclear.

## Goal

Keep the outer payment settings module as `Payment Gateway / 支付网关`, and rename the inner tabs area to `Payment Channels / 支付渠道` so it clearly refers to specific payment channels such as Epay, Alipay, WeChat Pay, Stripe, Creem, Waffo, and Waffo Pancake.

## Design

Change the inner tabs area only:

- `Payment Gateways` → `Payment Channels`
- `Enable a gateway first, then fill in the required configuration fields.` → `Enable a payment channel first, then fill in the required configuration fields.`
- `Save payment gateway settings` → `Save payment channel settings`

Add Chinese translations:

- `Payment Channels` → `支付渠道`
- `Enable a payment channel first, then fill in the required configuration fields.` → `请先启用一个支付渠道，然后填写所需的配置字段。`
- `Save payment channel settings` → `保存支付渠道设置`

## Files

- `web/default/src/features/system-settings/integrations/payment-settings-section.tsx`
- `web/default/src/i18n/locales/en.json`
- `web/default/src/i18n/locales/zh.json`

## Verification

- `cd web/default && bun run typecheck`
- `cd web/default && bun run build`
