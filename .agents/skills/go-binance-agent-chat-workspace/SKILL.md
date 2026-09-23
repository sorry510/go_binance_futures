---
name: go-binance-agent-chat-workspace
description: Maintain and verify the go_binance_futures Agent Chat Workspace and its go_binance_futrues_new_ui contract, including attached chat-capable skills, Auto and Explicit routing, model selection, conversation deletion, migrations, and regression tests.
version: 1.0.0
---
# Go Binance Agent Chat Workspace

Use this skill when changing, repairing, or reviewing Agent Chat Workspace behavior in `go_binance_futures` or its linked `go_binance_futrues_new_ui` frontend.

## Guardrails

- Never modify `conf/app.conf`; use isolated `/tmp` caches or configuration for verification.
- Preserve unrelated dirty changes in both repositories. The frontend directory is intentionally spelled `go_binance_futrues_new_ui`.
- Treat attached state, chat capability, enablement, and tool permission as separate boundaries. Attaching a Skill must never grant tools automatically.
- A message starts exactly one Primary Skill task. Do not merge multiple System Prompts or silently expand to Team execution.
- Keep `general_chat` attached and non-removable, chat-capable, selectable in Explicit mode, and the deterministic Auto fallback.

## Discovery

1. Index the repository with codebase-memory-mcp and use graph search and call tracing before text search.
2. Trace the UI to route, controller, app service, conversation store, runtime, and persistence.
3. Inspect the linked frontend whenever API fields, catalogs, model options, or visible behavior can change.
4. Check both worktrees before editing and preserve pre-existing generated assets and source changes.

## Skill routing invariants

1. A chat catalog entry must be enabled, chat-enabled, and implement `skill.ChatAdapter`. Add a compile-time assertion for native implementations.
2. Explicit routing must first require the requested Skill to be attached, then require it to be present in the current chat-capable catalog.
3. Auto routing may score only attached entries that remain present in the current chat-capable catalog. Unattached, disabled, or non-chat-enabled Skills are never candidates.
4. Keep the scoring algorithm deterministic. Cover the fallback threshold, exact Skill name, display name, and `@name` or `/name` priority with table-driven tests before tuning weights.
5. Legacy exported message entry points must delegate to the validated options entry point; do not maintain a bypass path. Preserve their historical default model semantics explicitly.
6. `general_chat` input remains plain text and must not gain tools. Normalize surrounding whitespace consistently with other chat adapters.

## Model and deletion invariants

1. Conversation `model_config_id = 0` means Model Gateway routing; a positive ID means strict selection with no silent fallback.
2. A message-level override affects only that new task unless the model preference endpoint is called.
3. Propagate the selected model config through parent, child, and supervisor tasks and retain final provider and model audit fields.
4. Reject deleting a running conversation. Delete conversation messages and Skill bindings but retain Task, Trace, and Audit history.

## Backend and frontend synchronization

1. Keep request and response fields synchronized across router, controller, app service, frontend API types, composer, conversation view, and both locale files.
2. If the frontend derives Explicit choices from the backend chat catalog, a backend catalog fix may require no frontend source change; verify that data flow instead of duplicating special cases.
3. For additive schema changes, bump the schema version once, keep migration SQL idempotent, and add upgrade coverage for both schema and data backfill.
4. Update phase README and implementation or review reports only after the corresponding Gate is verified. Distinguish historical audit evidence from post-fix status.

## Verification

- Run focused tests for affected app, conversation, Skill, manager, and Team packages.
- Run table-driven routing boundary tests and interface contract tests.
- Run targeted `-race` tests for changed concurrent or shared-state packages.
- Run `go vet ./...`, `go build ./...`, `gofmt -l` on intentionally changed Go files, and `git diff --check`.
- Run `go test -count=1 ./...` with `GOCACHE` under `/tmp`. Some suites bind loopback temporary ports; if sandbox binding is denied, rerun the same command with the required local-port permission rather than treating it as a product failure.
- Run frontend typecheck and build only when frontend source changes, then recheck both worktrees for generated files.
- Finish by confirming `git status --short -- conf/app.conf` is empty.
