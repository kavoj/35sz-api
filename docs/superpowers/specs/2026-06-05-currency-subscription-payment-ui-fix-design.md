# Currency and Subscription Payment UI Fix Design

## Context

The billing currency page can set the display mode to CNY and the exchange rate to 7.3. The global currency configuration is correctly exposed by `/api/status` and mapped into the frontend system config store, but several payment/subscription UI components still hard-code `$` or use local number-only formatters.

The subscriptions admin drawer also always shows Stripe, Creem, and Waffo Pancake third-party ID fields even when those gateways are not configured/enabled. Waffo Pancake has an existing auto-create product endpoint; Stripe and Creem do not yet have backend product/price creation endpoints.

## Goal

Fix the current UI-level currency display bugs and make subscription third-party payment configuration conditional on enabled payment gateways, without changing backend payment amount semantics or adding new Stripe/Creem backend creation APIs in this phase.

## Scope

### Included in this phase

- Payment settings top-up amount options show the configured display currency.
- Wallet subscription plan cards and purchase dialog show the configured display currency.
- Subscription admin plan drawer labels/descriptions clarify the actual payment amount currency.
- Subscription admin third-party payment config fields are hidden unless their gateway is enabled/configured.
- Waffo Pancake keeps its existing auto-create product flow.
- Stripe and Creem remain manual ID fields, shown only when enabled/configured.

### Deferred to a later phase

- Auto-create Stripe Product/Price.
- Auto-create Creem Product.
- Backend subscription amount currency normalization/migration.

## Design

### Currency display

Use existing currency utilities from `web/default/src/lib/currency.ts`:

- Use `formatCurrencyFromUSD` for values that represent system USD credit amounts.
- Use `formatLocalCurrencyAmount` for values that are already actual payment amounts.

Apply this as follows:

- `AmountOptionsVisualEditor`: top-up amount options are USD credit amounts, so display with `formatCurrencyFromUSD(amount)`.
- Wallet subscription plan card: `plan.price_amount` is currently used as actual payment amount by most subscription payment backends, so display with `formatLocalCurrencyAmount(plan.price_amount)`.
- Subscription purchase dialog: `Amount Due` is actual payment amount, so display with `formatLocalCurrencyAmount(plan.price_amount)`.

### Subscription admin drawer

Use `useSystemConfig()` to read current currency settings and `getCurrencyLabel()` from `currency.ts` for the label.

Change `Actual Amount` to include the current currency label, for example:

- English: `Actual Amount (CNY)`
- Chinese: `实际支付金额（CNY）`

Add description:

> Used as the actual amount charged by Epay, Alipay, WeChat Pay, and Waffo Pancake. Stripe and Creem use their configured product IDs.

### Conditional third-party payment config

The subscriptions admin provider already loads system options for compliance. Extend that provider to derive gateway availability from the same option data:

- Stripe config field visible when Stripe appears configured/enabled enough for subscription setup.
- Creem config field visible when Creem appears configured/enabled enough for subscription setup.
- Waffo Pancake config field visible when Waffo Pancake credentials are configured enough for product creation.

Pass these flags through subscription context to `SubscriptionsMutateDrawer`.

Field behavior:

- Stripe enabled: show `Stripe Price ID` with manual entry/help text.
- Creem enabled: show `Creem Product ID` with manual entry/help text.
- Waffo Pancake enabled: show `Waffo Pancake Product ID` selector and `+ Create` button using the existing endpoint.
- If none enabled: hide the entire `Third-party Payment Config` section.

### Subscription management notice

Replace the current top notice text with:

> Enabled third-party payment channels may require provider-specific product IDs. Waffo Pancake can create one automatically; Stripe and Creem currently require IDs from their dashboards.

Chinese:

> 已启用的第三方支付渠道可能需要配置平台对应的产品 ID。Waffo Pancake 可自动创建；Stripe 和 Creem 当前需要从各自控制台获取 ID。

## Files

- `web/default/src/features/system-settings/integrations/amount-options-visual-editor.tsx`
- `web/default/src/features/wallet/components/subscription-plans-card.tsx`
- `web/default/src/features/subscriptions/components/dialogs/subscription-purchase-dialog.tsx`
- `web/default/src/features/subscriptions/components/subscriptions-provider.tsx`
- `web/default/src/features/subscriptions/components/subscriptions-mutate-drawer.tsx`
- `web/default/src/features/subscriptions/index.tsx`
- `web/default/src/i18n/locales/en.json`
- `web/default/src/i18n/locales/zh.json`

## Verification

- `cd web/default && bun run typecheck`
- `cd web/default && bun run build`
- Manual checks:
  - Set currency display mode to CNY and exchange rate to 7.3.
  - Payment top-up amount option `100` displays as `¥730`.
  - Wallet subscription plan price displays with `¥`.
  - Subscription purchase dialog amount due displays with `¥`.
  - Subscription admin drawer only shows third-party ID fields for configured/enabled gateways.
  - Waffo Pancake `+ Create` still works when Waffo Pancake is configured.
