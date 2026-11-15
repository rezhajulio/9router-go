### fix(xiaomi-tokenplan): update model list to current Token Plan catalog

`xiaomi-tokenplan` (`xmtp`) still advertised the pre-2026-10-21 catalog:
`mimo-v2.5-pro` and `mimo-v2.5` were retired upstream and their ids no longer
resolve, and the list also carried `mimo-v2-pro`, `mimo-v2-omni`,
`mimo-v2-tts` and the claude-native `mimo-v2.5-pro-claude` alias of the removed
model. Connecting a Token Plan key and picking any of those ids failed at the
provider.

Replaced with the current Token Plan catalog (upstream 5e54377b, v0.5.119):

- `mimo-v2.6-pro`, `mimo-v2.6-flash` (chat)
- `mimo-v2.5-asr` (kind `stt`, params `["language"]`)
- `mimo-v2.5-tts`, `mimo-v2.5-tts-voiceclone`, `mimo-v2.5-tts-voicedesign`
  (kind `tts`)

`mimo-v2.6-pro-ultraspeed` stays out, matching upstream: it is rate-limited to
customized-service accounts.

Port notes: the registry swap lands in `ProviderModels` for both the id and the
`xmtp` alias, the media ids gain `ProviderModelKinds` rows (otherwise they leak
into the `/v1/models` LLM list), and the dashboard copy in
`web/src/lib/models.ts` follows — enforced by
`TestDashboardCatalogListsEveryRoutedModel`. No capability, pricing or
thinking-level rows were added upstream for the new ids: the chat ids resolve
through the existing global `*mimo*v2.6*` pattern rows, verified against
v0.5.119 with `scripts/gen-thinking-levels.mjs`.

The sibling `xiaomi-mimo` lane is untouched (provider isolation, AGENTS.md §3).

Verified: `go build ./...`, `go test ./...`, new
`TestUpstream20261112TokenPlan*` regression tests, `make vet-svelte` (74/74
baseline unchanged), `bun test src/lib/models.test.ts`, and `bun run build`
(dist 4.4M, brotli JS unchanged in kind).
