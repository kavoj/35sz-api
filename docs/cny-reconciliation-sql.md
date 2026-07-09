# CNY Recharge Reconciliation SQL

PR-4 加固后，每笔 TopUp 行都携带 5 个快照字段。历史订单不受后续 admin
改动 `USDExchangeRate` / `RechargePremium` 影响。

## 快照字段

| 字段 | 类型 | 说明 |
|---|---|---|
| `amount_usd_snapshot` | DECIMAL(12,4) | 用户看到的 USD 数量 |
| `payment_amount_cny` | DECIMAL(12,2) | 实际支付的本地货币金额 |
| `usd_exchange_rate_snapshot` | DECIMAL(10,4) | 支付时的汇率 |
| `recharge_premium_snapshot` | DECIMAL(6,4) | 支付时的平台溢价 |
| `quota_per_unit_snapshot` | BIGINT | 支付时每 USD 兑换的 token 数 |

## 核心恒等式

```
PaymentAmountCNY  =  AmountUSDSnapshot × USDExchangeRateSnapshot × RechargePremiumSnapshot
```

任何一笔历史订单都可按此公式独立复算。

## 单笔核对 SQL

```sql
SELECT
  id,
  trade_no,
  user_id,
  amount_usd_snapshot,
  payment_amount_cny,
  usd_exchange_rate_snapshot,
  recharge_premium_snapshot,
  usd_exchange_rate_snapshot * recharge_premium_snapshot AS effective_rate,
  amount_usd_snapshot
    * usd_exchange_rate_snapshot
    * recharge_premium_snapshot AS reconciled_cny,
  ABS(
    payment_amount_cny
    - amount_usd_snapshot
      * usd_exchange_rate_snapshot
      * recharge_premium_snapshot
  ) AS drift_cny
FROM top_ups
WHERE trade_no = 'USR123NO456';
```

**drift_cny > 0.01** 说明支付时的 `payMoney` 和快照三要素算出来的
CNY 值不一致，需要人工介入（通常是 controller 里绕过了
SnapshotCurrencyForInsert 的代码路径）。

## 期间总收入统计

```sql
-- 2026-Q3 CNY 总收入
SELECT
  DATE_FORMAT(FROM_UNIXTIME(complete_time), '%Y-%m') AS month,
  COUNT(*) AS orders,
  SUM(amount_usd_snapshot) AS total_usd,
  SUM(payment_amount_cny) AS total_cny,
  AVG(usd_exchange_rate_snapshot) AS avg_rate,
  AVG(recharge_premium_snapshot) AS avg_premium
FROM top_ups
WHERE status = 'success'
  AND complete_time BETWEEN UNIX_TIMESTAMP('2026-07-01 00:00:00')
                        AND UNIX_TIMESTAMP('2026-09-30 23:59:59')
GROUP BY month
ORDER BY month;
```

PostgreSQL 版：将 `DATE_FORMAT(FROM_UNIXTIME(...), '%Y-%m')`
替换为 `TO_CHAR(TO_TIMESTAMP(complete_time), 'YYYY-MM')`；
SQLite 版：替换为 `strftime('%Y-%m', complete_time, 'unixepoch')`。

## 未快照的历史行

老 TopUp 行 (PR-4 之前的) 的快照字段可能是 0。这类行可以用
`payment_amount_cny = 0 OR usd_exchange_rate_snapshot = 0` 筛出来：

```sql
SELECT COUNT(*) FROM top_ups
WHERE status = 'success'
  AND (payment_amount_cny = 0 OR usd_exchange_rate_snapshot = 0);
```

如果数量可接受，可用当前系统设置估算历史行 (但注意结果不精确，因为
admin 中途可能改过汇率)：

```sql
UPDATE top_ups
SET
  amount_usd_snapshot = amount,           -- 假设 amount 存 USD 数（Stripe/Epay 均如此，Creem 例外）
  payment_amount_cny = money,             -- 用户实际支付
  usd_exchange_rate_snapshot = 7.3,       -- 用当前 operation_setting.USDExchangeRate 替换
  recharge_premium_snapshot = 1.0,
  quota_per_unit_snapshot = 500000
WHERE status = 'success'
  AND usd_exchange_rate_snapshot = 0;
```

**只在你确认 admin 从未改过汇率时才执行**，否则请直接舍弃对老行的
反查支持。

## 与 Commission Redemption 对齐

`commission_redemptions` 表早在 PR-1 之前就已经有 `USDExchangeRate` +
`QuotaPerUnit` 快照。PR-4 让 `top_ups` 表跟上了同样的加固模式；两张
表 join 之后可以完整审计"用户从下单到消费到分账"的三段货币流转：

```sql
SELECT
  tu.trade_no,
  tu.amount_usd_snapshot AS topup_usd,
  tu.payment_amount_cny AS topup_cny,
  tu.usd_exchange_rate_snapshot AS topup_rate,
  cr.commission_cents AS commission_earned_cny_cents,
  cr.usd_exchange_rate AS commission_rate,
  cr.quota_credited AS commission_quota
FROM top_ups tu
LEFT JOIN commission_redemptions cr ON cr.user_id = tu.user_id
WHERE tu.status = 'success'
  AND tu.complete_time > UNIX_TIMESTAMP('2026-07-01 00:00:00');
```

**注意**：CommissionRedemption 的 `usd_exchange_rate` 快照在 redeem
瞬间冻结，而 TopUp 的 snapshot 在支付瞬间冻结；两者是不同时刻的
汇率，可以差别很大 (admin 中间改过)。这是预期行为，不是 bug。

## 相关

- `db-currency-storage-audit` (memory) — 审计报告及历史
- `cny-reconciliation-plan` (memory) — PR-4 设计文档
- `service/commission/redeem.go` — 参考实现 (commission 是 gold standard)
- `model/topup.go` `SnapshotCurrencyForInsert` — 快照实现
- `model/log.go` `RecordTopupLogWithSnapshot` — 结构化日志（同一份快照写入 `logs.Other`）
