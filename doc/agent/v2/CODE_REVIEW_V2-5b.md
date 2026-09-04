# V2-5 复查报告（CODE_REVIEW_V2-5b）

> 触发：用户提交 V2-5 更新代码（新增 Interactive OAuth 2.0 授权流程），要求重新 review。
> 前置：`CODE_REVIEW_V2-5.md` 为初版（Conditional Pass，F1 AT RISK / F2 待确认 / F7 FAIL）。
> 范围：仅 code review，**未做任何代码修改、未写内存、未做后续任务**（遵循"review only"约束）。
> 复查时间：2026-09-04

---

## 0. 结论速览

| 项目 | 初版 (V2-5) | 本次复查 (V2-5b) |
| --- | --- | --- |
| 整体 Gate 结论 | Conditional Pass | **PASS** |
| F1（custom_header 凭据明文） | AT RISK | **已撤回/降级**（原前提错误） |
| F2（更新路径未保留凭据） | 待确认 | **已修复** |
| 新增 Interactive OAuth | 不在原范围内 | **符合安全 Gate，全部通过**（phase doc 已纳入范围） |
| F7（前端 TS6053） | FAIL（repo 既有问题） | 仍 FAIL，但**确认非 V2-5 回归** |
| 后端 build | — | PASS |
| 后端 race 测试（聚焦包） | — | PASS |
| 后端 `go test ./...` 全量 | — | PASS（全绿） |

**最终判定：V2-5 更新可进入下一 Phase（V2-6）。** 唯一遗留项 F7 为仓库既有问题，与本次 MCP 工作无关，建议另行单独处理。

---

## 1. Gate 基线变更说明（重要）

本次复查前，`doc/agent/v2/05-phase-v2-5-mcp-integration.md` 与 `doc/agent/v2/README.md` 已被更新：
- phase doc §4 鉴权模式新增 `oauth2 interactive Authorization Code + PKCE`；
- §12 新增 Interactive OAuth 部署配置与 Binance 官方 MCP 兼容性说明；
- §13 回归覆盖清单新增 13 条 OAuth 相关 E2E 用例；
- §14 Phase Gate 全部勾选（含 OAuth 相关 4 项）；
- README V2-5 标为 ✅。

因此 **Interactive OAuth 现属 V2-5 正式范围**，不再按"超出 phase 文档"处理。复查据此以更新后 phase doc §14 为 Gate 基线。

---

## 2. 验证执行结果（本次实际跑过）

| 验证项 | 命令 | 结果 |
| --- | --- | --- |
| 后端全量编译 | `go build ./...` | PASS（无输出，exit 0） |
| 聚焦包 race 测试 | `go test -count=1 -race ./agent/mcpclient/... ./agent/runtime/... ./agent/contextengine/... ./agent/toolruntime/... ./models/...` | 5 包全部 `ok`（含新增 `TestInteractiveOAuthPKCEPersistenceAndReconnect`） |
| 全量测试 | `go test -count=1 ./...` | **全绿**，无 FAIL（仅 `ld: malformed LC_DYSYMTAB` 链接器警告，无害） |
| 前端 typecheck | `npx vue-tsc --noEmit --skipLibCheck` | 唯一报错 `TS6053: src/views/permission/page/index.vue not found`（repo 既有，非 V2-5 文件） |

> 说明：`ld: warning ... malformed LC_DYSYMTAB` 为 macOS 链接器对带 cgo 测试二进制的无害告警，不影响测试结果；所有包均 `ok`。

---

## 3. F1 撤回说明（关键纠正）

**初版 F1 前提错误，本次撤回并降级为低敏观察。**

初版 F1 假定 `custom_header` 字段存储的是 `Authorization: Bearer <token>` 这类"凭据值"，会明文落库并经 API 泄露。复查实际代码后该前提不成立：

- `agent/mcpclient/security.go` `authRoundTripper.RoundTrip`（L98-99）：
  ```go
  case AuthCustomHeader:
      clone.Header.Set(rt.header, rt.secret)
  ```
  其中 `rt.header = strings.TrimSpace(server.CustomHeader)`（**Header 名称**，如 `X-API-Key`），`rt.secret = resolver(ctx, server.SecretRef)`（**凭据值**，运行时解析）。
- `agent/mcpclient/store.go` `serverView`（L151-155）：`row.SecretRef = ""` 后返回，**绝不返回凭据值**；`CustomHeader` 仅携带 Header 名称。
- `agent/mcpclient/security.go` `buildHTTPClient`（L185-187）：`resolver` 默认 `ResolveManagedSecret`，故 `mcpdb:oauth:<id>` 引用也能正确解析。

**结论**：`CustomHeader` 列 + API 返回 + 前端回显只承载 Header **名称**，凭据值始终隔离在 `SecretRef`（`env:` 或加密 `mcpdb:oauth:`），从不以 `custom_header` 形式存储。Header 名称属低敏信息，返回可接受。**F1 撤回**。

---

## 4. F2 修复确认

`agent/mcpclient/store.go` `SaveServer` 更新路径：

- L114：`sameCredentialScope = existing.AuthType == row.AuthType && existing.Endpoint == row.Endpoint`；
- L115-117：`row.SecretRef == "" && sameCredentialScope` 时沿用 `existing.SecretRef` → **凭据在 AuthType/Endpoint 不变时保留**；
- L121-129：当 scope 变化且原 `SecretRef` 为 `mcpdb:oauth:` 前缀时，**删除 `AgentMCPSecret` 与 `AgentMCPOAuthState`** 并清空 `SecretRef`，避免静默复用旧 token；
- L73-75：API 提交的 `secret_ref` 强制 `env:` 前缀，不接受裸凭据；
- L79-81：非 `custom_header` 鉴权时清空 `CustomHeader`。

**结论：F2 已修复**，凭据在编辑保存时不丢失、scope 切换时不串用。

---

## 5. 新增 Interactive OAuth 2.0 安全评估

逐条对照 phase doc §5/§14 安全 Gate，直接读 `oauth.go` / `security.go` / `store.go` / `controllers/agent_mcp.go` / `middlewares/auth.go` 验证：

### 5.1 加密（token / state 落库）
- `encryptOAuthPayload` / `decryptOAuthPayload`（L107-157）：AES-256-GCM（key 32 字节），**每记录随机 nonce**，`"v1:"+base64.RawURLEncoding` 封装，绑定 AAD（`mcp-secret:<id>:oauth2` / `mcp-oauth-state:<hash>`）。
- `oauthEncryptionKey`（L89-105）：强制 `mcp::oauth_encryption_key` 为 hex(64) 或 base64 的 **32 字节**，否则 fail-closed。
- DB 不存 token 明文：测试 `TestInteractiveOAuthPKCEPersistenceAndReconnect` 断言 `Ciphertext` 不含 `access-1`/`refresh-1`。

### 5.2 PKCE / CSRF state
- `cfg.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier), ...)`（L292）：强制 **S256** PKCE。
- `discoverOAuthMetadata`（L392-394）：若授权服务器 `code_challenge_methods_supported` 不含 `S256` 直接报错拒绝。
- `consumeOAuthPending`（L508-531）：`stateHash`=SHA256；读取后即 `o.Delete(&row)` → **一次性消费**；再校验 `ExpiresAt`；**replay 被拒**（测试二次 `CompleteOAuth` 报错）。
- `saveOAuthPending`（L502-503）：写入前清理过期 state 与同 server 旧 state；TTL 10 分钟。

### 5.3 Issuer 校验
- `CompleteOAuth`（L550-558）：`RequireIssuerResponse`（AS 声明支持 `iss` 响应参数）时要求 `iss` 非空且等于 `pending.Issuer`；其余情况若返回 `iss` 且不一致亦拒绝。

### 5.4 SSRF 边界
- `discoverOAuthMetadata`（L358-396）、`restrictedHTTPClient`（L420-426）、token exchange（L563）：metadata / token / DCR 端点全部经 `buildHTTPClient`（同一套 `ValidateEndpoint` + `validateResolvedIP` + `transport.Proxy = nil`），**不继承系统代理**，对实际解析 IP 做强校验。
- `oauthPublicURLs`（L63-82）：`mcp::oauth_public_base_url` 默认强制 HTTPS，仅 loopback (`localhost` / `127.0.0.1` / `::1`) 允许 HTTP 本地开发；禁 userinfo/query/fragment → `redirect_uri` 不可被篡改。
- MCP transport 与 OAuth 出站均不继承 `HTTP_PROXY/HTTPS_PROXY`（security.go L191）。

### 5.5 客户端身份（CIMD / DCR）
- `resolveOAuthClient`（L428-453）：优先 Client ID Metadata Document（`token_endpoint_auth_method: none`）；若 AS 声称支持 CIMD 却不支持 `none` token auth → **直接拒绝**；否则回退 DCR，`chooseTokenAuthMethod` 优先 `none`→`client_secret_post`→`client_secret_basic`。避免静默落入弱认证。

### 5.6 回调页 / 路由 / 错误处理
- `writeOAuthCallbackPage`（controllers L187-196）：HTML 仅插入**系统固定状态文案**，`serverID` 为 int64 不渲染，用户查询参数 `state/code/error/...` **绝不进入 HTML** → 无反射型 XSS。
- `middlewares/auth.go` L22-23：`/agents/mcp/oauth/client-metadata` 与 `/agents/mcp/oauth/callback` 精确（非 `/*` 前缀）排除 JWT；`oauth/start` 仍走 JWT（管理员专属）。排除面最小、无过度放开。
- `markOAuthRequired`（L590-600）：错误写入 DB 前经 `security.RedactText` 脱敏，防敏感信息落库。

### 5.7 Refresh 旋转持久化
- `savingOAuthTokenSource`（security.go L120-148）：token 变化（access/refresh/expiry）时重新加密写回 `AgentMCPSecret`；`buildHTTPClient` 在 `mcpdb:oauth:` 引用下挂载 `persist`，进程重启后仍可继续用。

### 5.8 评估结论
OAuth 实现工程完整、纵深防御到位，**所有 phase doc §14 OAuth 相关 Gate 通过**。无阻塞性安全问题。

---

## 6. 遗留 / 观察项（均非阻塞）

| 项 | 说明 | 处理建议 |
| --- | --- | --- |
| F7 前端 TS6053 | `src/views/permission/page/index.vue` 缺失，仅被 `sso.ts` 注释 URL 引用，V2-5 未触碰该文件；`vue-tsc` 因此 FAIL。V2-5 自身文件零类型错误。 | **repo 既有问题**，与本次无关；建议单独补文件或修正 tsconfig include，不在 V2-5 内解决。 |
| pending.ClientSecret 短期落库 | `oauthPending` 含 `ClientSecret`，以加密形式存于 `AgentMCPOAuthState` 的 10 分钟 pending 窗口；消费即删、且加密。 | 可接受；如有合规要求可改为仅 DCR 时临时持有、不落库。 |
| OAuth 配置缺失 fail-closed | `StartOAuth` 在缺 `conf/app.conf` 中的 `mcp::oauth_public_base_url` / `mcp::oauth_encryption_key` 时直接报错（L256-265）。 | 符合预期，部署文档已说明（phase doc §12）。 |

---

## 7. 最终判定

- Gate 基线（更新后 phase doc §14）：**全部满足**。
- 验证：build / race / 全量测试 **全绿**；前端 typecheck 仅 repo 既有 TS6053。
- 历史发现：F1 撤回、F2 修复；新增 OAuth 安全 Gate 全通过。
- **V2-5 更新 = PASS，可进入 V2-6 Imported / Portable Skills。**

> 备注：本复查未修改任何代码、未写入项目/用户内存，未执行 V2-5 范围外任务。F7 待仓库侧单独处理。
