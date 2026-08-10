# suixingAPI capability baseline

Branch base: `main` at `823e2630`.

This document is the upgrade acceptance baseline. A migration is complete only
when the applicable capability and pricing checks below pass.

## Five model capabilities

| Capability | Stable client surface | Baseline verification |
|---|---|---|
| Text and Responses | `/v1/chat/completions`, `/v1/responses` | relay channel and conversion tests |
| Embeddings | `/v1/embeddings` | relay embedding tests |
| Images | `/v1/images/*` | image adaptor and billing tests |
| Video | task creation and polling routes | task adaptor tests |
| Audio | `/v1/audio/speech`, `/v1/audio/transcriptions` | audio adaptor tests |

## Routing invariants

- Standard multipart transcription defaults to TOS upload followed by BigASR
  submit/query and uses `volc.bigasr.auc`.
- Only explicit `metadata.mode=stream` selects SeedASR WebSocket and
  `volc.seedasr.sauc.duration`.
- Agent Plan base URLs use their own path layout and do not acquire a second
  `/api/v3` prefix.

## Structured pricing contract

- `pricing_kind` supports chat, multimodal-chat, image-gen, video-gen,
  audio-in, audio-out, and embedding.
- ImagePricing, VideoPricing, AudioInPricing, and AudioOutPricing are stored as
  structured settings and explicit administrator values take precedence.
- `GET /v1/models/pricing` exposes a stable machine-readable contract.
- Missing structured prices are marked incomplete rather than silently billed.
- Static reference prices are fallback data, not a replacement for explicit
  administrator configuration.
- Storage remains compatible with SQLite, MySQL, and PostgreSQL.

## Frontend migration order

1. Five-capability model management and structured pricing.
2. Vendor inference, capability tags, and context-window metadata.
3. Alipay, WeChat Pay, and CNY presentation.
4. Referral commission management.
5. Home and model-market visual upgrades.

The migration targets `web/src`. Retired `web/classic` and `web/default`
directories are not runtime or source-of-truth frontends.

## Verification boundary

The root Go package embeds generated `web/dist`. Root-package tests and builds
require a completed frontend build. Tests using `httptest` listeners require an
execution environment that permits local loopback sockets.
