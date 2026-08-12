---
name: go-binance-mcp-http-api-mirror
description: Maintain the go_binance_futures Streamable HTTP MCP server when its explicit REST allowlist, JWT forwarding, transport, or documentation changes. Use for adding or removing categorized MCP tools while ensuring unrelated HTTP routes remain unexposed and conf/app.conf stays untouched.
version: 1.0.0
trusted: false
---

# Go Binance MCP HTTP API Mirror

Use this workflow when adding or changing MCP coverage in `/Users/zhz/work/binance/go_binance_futures`.

1. Read `routers/router.go`, `mcpserver/`, `middlewares/auth.go`, and the relevant controller inputs. Establish the user-approved MCP allowlist before editing; never infer that every non-login route should be exposed.
2. Keep only explicitly allowed, stable, categorized tools in `mcpserver/api_tools.go`. Keep tool paths fixed in code; never accept an arbitrary target URL. Use `path_params`, `query`, and `body` to preserve each allowed REST contract.
3. Keep MCP transport Streamable HTTP only at `/mcp`. Do not add stdio, command, or a separate local port. Require the existing login JWT and forward the incoming `Authorization` header to the internal HTTP call.
4. Do not register MCP-specific notification, scanner, system, account, order, or other privileged tools unless the user explicitly adds them to the allowlist. Removing an MCP-only notification tool also removes its dedicated adapter and template while preserving the project's ordinary notification services.
5. Test the exact tool-name and method/path allowlist, confirm every allowed operation still exists in `routers/router.go`, and ensure unrelated routes are not required or auto-exposed. Also test `tools/list`, an actual allowed `tools/call`, Authorization propagation, and internal HTTP request construction.
6. Run MCP tests from an isolated `/tmp` config and language directory. Compile from the repository, but never modify or use the real `conf/app.conf` as test setup. Bind only a temporary localhost port for Streamable HTTP protocol tests.
7. Update `doc/mcp.md` with endpoint, JWT setup, common arguments, the exact current allowlist, removed tool groups, and destructive-operation warnings. Verify every registered tool appears in the document and deleted tool names do not remain.
8. Finish with `git diff --check`, focused `go vet`, a main-package compile-only check, an exact residual route/tool comparison, and `git diff --exit-code -- conf/app.conf`. Preserve unrelated worktree changes.
