# suixingAPI five-capability baseline

Branch base: `main` at `823e2630`.

This branch integrates the five client-facing capability groups already
available in `newpay`, without importing its payment, CNY settlement, pricing
editor, commission, or legacy frontend-directory migrations.

| Capability | Source and retained behavior | Stage verification |
|---|---|---|
| Text | Latest `main` already contains DeepSeek Responses support. | `go test ./relay/channel/deepseek ./relay/common` |
| Audio | SeedASR WebSocket for explicit stream mode; BigASR file mode defaults to TOS upload then submit/query with `volc.bigasr.auc`; Agent Plan TTS remains supported. | `go test ./relay/channel/volcengine ./common/objectstorage` |
| Video | Volcengine native task route and Agent Plan task payload/URL adaptation. | `go test ./relay/channel/task/doubao ./controller` |
| Embeddings | Latest `main` implementation remains intact; Volcengine URL handling includes Agent Plan embeddings. | `go test ./relay/channel/volcengine ./relay/common` |
| Images | Latest `main` already contains the image stream disconnect and billing correction; Seedream channel testing remains image-routed. | `go test ./relay/channel/openai ./controller` |

## Routing invariants

- `POST /v1/audio/transcriptions` without `metadata.mode=stream` uses the
  BigASR file-recognition workflow and `volc.bigasr.auc`.
- Only explicit `metadata.mode=stream` selects SeedASR WebSocket and
  `volc.seedasr.sauc.duration`.
- Agent Plan base URLs use their own path layout and must not acquire a second
  `/api/v3` prefix.
- No migration in this branch changes payment, currency conversion, structured
  pricing, commission, or `web/default` directory ownership.

## Verification boundary

The repository root package embeds generated `web/dist`. Build or root-package
tests need that frontend output first. Package-level backend verification does
not require the generated frontend asset.
