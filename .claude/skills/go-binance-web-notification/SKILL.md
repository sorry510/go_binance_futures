---
name: go-binance-web-notification
description: Implement and verify persisted browser notifications with authenticated WebSocket delivery across go_binance_futures and go_binance_futrues_new_ui.
version: 1.0.1
---
# Go Binance Web Notification

Use this skill when adding or changing browser notification persistence, real-time delivery, read state, or the notification bell across the linked backend and frontend repositories.

## Repositories

- Backend: `/Users/zhz/work/binance/go_binance_futures`
- Frontend: `/Users/zhz/work/binance/go_binance_futrues_new_ui`
- Preserve the intentional `futrues` spelling.
- Never modify or delete the real backend `conf/app.conf`.

## Backend checklist

1. Define the ORM model and explicitly register it in `main.go` so `RunSyncdb` creates or updates the table at startup.
2. Keep notification persistence separate from WebSocket fan-out. Insert first; broadcast only after a successful insert.
3. Reuse the existing notification producers rather than duplicating business rules. Audit their call graph before choosing the persistence hook. This project selects exactly one `Pusher` implementation through `GetNotifyChannel`, so publish from both `DingDingApi` and `SlackApi`; only the selected implementation runs for a logical message.
4. Implement a bounded WebSocket client send queue, ping/pong deadlines, unregister on disconnect, and one shared hub.
5. Expose paginated history, unread count, read-one, read-all, and WebSocket routes. Put the static read-all route before the parameter route.
6. Browser WebSocket clients cannot set an Authorization header. Accept the existing `Bearer <jwt>` value through the `token` query parameter and validate it inside the WebSocket controller. Excluding the WS route from header middleware is safe only when the controller performs this validation before upgrading. Reject malformed or doubled Bearer prefixes.
7. Normalize markdown and HTML intended for remote notification channels before storing browser-visible title and content. Do not log optional missing notification content.

## Frontend checklist

1. Add typed history and read-state API helpers.
2. Build the WebSocket URL from the current page protocol and host, switching `http` to `ws` and `https` to `wss`. URL-encode the stored token.
3. Load recent history before or alongside opening the socket. Display an unread badge, notification popover, read-one, read-all, and a toast for newly received messages.
4. Deduplicate events by notification ID before incrementing unread state. A reconnect can deliver the same event again.
5. Reconnect with capped exponential backoff and clear timers and sockets on component unmount.
6. Render the notification component in every supported navbar layout and add both Chinese and English locale keys.
7. Configure the Vite `/ws` proxy with `ws: true` for local development.

## Verification

- Run focused Go tests for model schema, JWT validation, persistence, and WebSocket broadcast. Build and execute tests using an isolated temporary Beego config directory; never use the real `conf/app.conf`.
- Run focused `go vet` for changed backend packages.
- Run frontend `pnpm typecheck`, `pnpm build`, and `git status` after each because generated artifacts can appear.
- Use a temporary HTTP and WebSocket mock plus a real browser to verify history, real-time toast, popover contents, read actions, and reconnect behavior. Wait through at least one reconnect and confirm the unread count does not increase for a duplicate ID. Confirm the browser console has no errors.
- Remove only artifacts created by the verification run. Preserve unrelated worktree changes.
- Finish with `git diff --check` in both repositories and `git diff -- conf/app.conf` in the backend.

## Retention, search, and module switches

- Keep browser notifications for 30 days. Run cleanup once at task startup and every 6 hours afterwards; delete rows whose `create_time` is strictly before the cutoff.
- A normal history page should support title/content keyword, module, read status, start/end time, pagination, read-one, and read-all, with Chinese and English labels.
- `notify_config.enable` is the module's external-channel switch. Query the latest active-channel row without filtering by `enable`; no row means enabled by default, while an existing row with `enable = 0` must return before DingTalk or Slack network I/O. Web persistence must happen before this return.
- On create, distinguish an omitted `enable` field from an explicit zero: omitted defaults to enabled, explicit zero remains disabled.
