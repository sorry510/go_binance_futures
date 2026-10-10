# 币安合约自动带单 — Stage 4-1 代码审计报告

- **审计对象**：`doc/trade/币安合约自动带单-Stage4-1-实施记录.md` 对应的 **Stage 4-1 账户配置与只读身份验证**（工作区未提交改动）。基线 `HEAD = 3201634`（`feat: stage3`，分支 `feat/lead-trading`）
- **审计方式**：原地只读审计 + 隔离副本写临时探针（探针与副本已删除，用户仓库与 `strategy_templates/` 未被改动）
- **审查类型**：review-only
- **结论**：**Stage 4-1 完成**：Lead 专属加密凭证仓库与只读身份/账户验证实现扎实，**零生产接线**（无路由/无表/无迁移/无后台任务），`TradingReady` 恒为 false 且默认关闭项无解除入口。未发现 P0/P1；发现 **1 项 P3（超时以 `blocking_reasons` 表达且 error 为 nil 的接口契约）** 与若干 P3/观察。
- **审计日期**：2026-10-10

---

## 1. 结论先行

| 判定项 | 结论 |
| --- | --- |
| 计划 4.1「账户角色固定 lead；lead_enabled=false 默认；启动前须完成身份/白名单/余额/可交易校验」 | ✅ 实现：`Manager` 常量绑定 `lead`；`initialReport()` 默认 `TradingReady=false`、`ReadOnlyChecksPassed=false` + 7 条固定 blocking reason；验证覆盖身份/白名单/账户权限与余额/仓位/挂单/Hedge ✓ |
| 计划 4.1「Lead 与 `FutureEnable` 互不改变」 | ✅ **结构性保证**：`leadaccount` 包不引用 `models.Config`、不注册到启动流程，`NewManagerFromEnv()` 无调用方 ✓ |
| 计划 4.1「不信任用户填写的 portfolio 名称」 | ✅ `PortfolioLabel` 仅加密存储+展示，注释明确 "untrusted UI metadata, not identity"；`PortfolioBindingConfirmed` 恒 false ✓ |
| Gate 4（本阶段相关部分）「无可绕过风控的带单写入口」 | ✅ **结构性满足**：`AccountReader` 接口**只含读方法**（无下单/撤单/杠杆/保证金/WS 启动），编译期断言 `var _ AccountReader = (*binance.AccountClient)(nil)` ✓；`GetPositionModeContext` 为签名 GET 且只发一次 ✓ |
| Gate 4「真单仍需 Stage 7 人工授权」 | ✅ 记录与代码一致：`TradingReady=false` 恒定、固定 reason `stage7_live_authorization_required`、无解除入口 ✓ |
| DB / 配置 / 路由 | ❌ 全部未改：`routers/`、`models/`、`appversion/`（仍 19）、`command/` 无变更 ✓ 与记录 §2 最后一段一致 |
| 是否存在 P0/P1 | ❌ 未发现 |

---

## 2. 实现逐项核对

### 2-A 加密凭证仓库（`service/leadaccount/credentials.go`，241 行）

| 要求（记录 §1/§2） | 判定 | 证据 |
| --- | --- | --- |
| AES-256-GCM + 随机 Nonce + 账户/版本 AAD | ✅ | `aes.NewCipher(32B)` + `cipher.NewGCM`；每次 `Save` 用 `io.ReadFull(rand.Reader, nonce)` 生成新 nonce；AAD = `credentialVersion+":"+LeadAccountID`（`:155`/`:202`） |
| 原子写入（临时文件后 rename） | ✅ | `os.CreateTemp(dir, ".lead-credential-*.tmp")` → `Chmod(0600)` → `Write` → **`Sync`** → `Close` → 再次校验路径 → `os.Rename` → 目录 fsync（best-effort）；`defer os.Remove(name)` 保证失败无残留（`:209-239`） |
| 文件 0600、目录须预先 0700 | ✅ | 写入固定 0600 ✓；`verifyCredentialPath` 要求父目录**非符号链接**且 `perm&0077==0`，目标文件须为常规文件、`perm&0077==0` 且**不可执行**（`perm&0100==0`）✓ |
| 禁止相对路径 / 符号链接目标 | ✅ | 非绝对路径、`/`、base `.` → `ErrUnsafeLocation`；`os.Lstat` 判定 → 符号链接既非目录也非常规文件 → 拒绝 ✓ |
| 任何密钥加载失败 fail-closed | ✅ | 主密钥缺失 → `ErrKeyUnavailable`；base64 非 32 字节 → 报错；文件不存在 → `ErrCredentialsUnavailable`；读失败/超 16KB/版本不符/nonce 长度异常/GCM 解密失败/明文不合法 → **`ErrCredentialsCorrupt`** ✓ |
| 数据不进入普通 JSON 输出 | ✅ | `Credentials.APIKey/APISecret` 标记 `json:"-"`；仅 `sealedCredentialData`（未导出）参与加密前序列化；`MaskedKey()` 只回显后四位 ✓ |
| 凭证来源：环境密钥 + 私有文件路径 | ✅ | `BINANCE_LEAD_CREDENTIAL_KEY`（标准 base64 / 恰 32 字节）与 `BINANCE_LEAD_CREDENTIAL_FILE`（绝对路径）✓；不自动生成主密钥、不写盘、不自动建目录 ✓ |
| 轮换 | ✅ | 覆盖式原子写入；旧文件即失效（旧 key 无法解新密文）✓ |

### 2-B 只读验证与管理器（`service/leadaccount/verify.go`，306 行）

| 要求 | 判定 | 证据 |
| --- | --- | --- |
| 无全局注册、无启动钩子、无后台轮询/路由 | ✅ | `Manager` 仅由 `NewManagerFromEnv()`/`NewManager()` 构造，全仓**无生产调用方**（见 §4）✓；无 goroutine、无 ticker ✓ |
| 显式调用才验证，45 秒整体超时 | ✅ | `VerifyReadOnly(ctx)` 手动调用；`context.WithTimeout(ctx, 45s)` 覆盖全部 6 步 ✓；`verifyMu` 串行化并发验证 ✓ |
| 六步流程与顺序 | ✅ | ① `userStatus` 必须 `Success && IsLeadTrader`，否则**立即返回**（不进入 /fapi ✓）→ ② `leadSymbol` 白名单（USDT 后缀 + quote 校验 + 去重，空则 `lead_usdt_whitelist_empty`）→ ③ 账户 `CanTrade` + USDT 余额/可用余额（`validAmount` 拒绝 NaN/Inf/负值）+ Multi-Asset 双检（非 USDT 资产 MarginBalance>0 → `multi_asset_account_unconfirmed`；`MultiAssetsMargin` → `lead_multi_asset_margin_mode`）→ ④ Fresh 仓位快照（数量合法性校验）→ ⑤ Fresh 未成交订单快照 → ⑥ `positionSide/dual` 校验 Hedge Mode（非双向 → `lead_hedge_mode_required`）✓ |
| 报告不泄漏 Key/Secret/签名/原始交易所错误 | ✅ | `VerificationReport` 字段全部为布尔/数值/固定字符串 + `KeyHint`（掩码）；失败以**固定 reason 字符串**表达，不携带底层错误文本 ✓（探针实测 JSON 无泄漏 ✓） |
| 旋转后旧验证立即失效；旧凭证晚到响应被拒 | ✅ | `SaveCredentials` → `generation++` + `last=initialReport()`（仅保留掩码与 label）✓；`VerifyReadOnly` 前后各取 generation，不一致则**丢弃报告并返回错误** ✓ |
| 再次验证失败不得沿用旧成功 | ✅ | `invalidateVerification(generation, reason)` 重置为初始报告并追加原因 ✓（永久测试 `TestStage41ManagerFailedReverificationClearsEarlierSuccess` ✓） |
| `PortfolioBindingConfirmed` / `TradingReady` 恒 false | ✅ | `finishVerification` 末尾无条件追加 4 条结构性 reason 并强制两字段为 false ✓（探针实测成功路径仍 `trading_ready=false` ✓） |
| 只读方法：`GetPositionModeContext` | ✅ | `NewGetPositionModeService().Do(...)` 经 `doAccountSigned`（读 recvWindow）✓；SDK 测试断言 **GET `/fapi/v1/positionSide/dual`、Lead Key 签名、恰好 1 次请求** ✓ |

---

## 3. 实证过的隐式契约（探针，全部通过）

| 探针 | 实证内容 | 结果 |
| --- | --- | --- |
| **PR-S4-1** 凭证仓库加固（补既有测试缺口） | ① **符号链接父目录** ② **可执行位**的目标文件 ③ 超长 key(600)/label(200) ④ 空 secret ⑤ **envelope 版本被改** ⑥ **临时文件残留** | ✅ ①②③④⑤ 全部拒绝（`ErrUnsafeLocation` / 输入错误 / `ErrCredentialsCorrupt`）；被拒输入**未破坏**既有凭证；保存后目录内**无 `.tmp` 残留**（entries=1） |
| **PR-S4-2** `Status()` 隔离 + 报告不泄漏 | 初始状态、防御性拷贝、成功路径与 JSON 序列化 | ✅ 初始状态含 5 条结构性 reason 且 `TradingReady=false`；外部修改返回的切片**不影响内部**；成功路径 `read_only_checks_passed=true` 但 `trading_ready=false`/`portfolio_binding_confirmed=false`；`key_hint=****ABCD`（掩码）；**报告 JSON 不含 key/secret** ✓ |
| **PR-S4-3** 白名单过滤 + 多资产门禁 | 混入重复/非 USDT（`ETHBTC`、`SOLUSDC`）/空 quote 的名单；非 USDT 资产有保证金；`MultiAssetsMargin=true` | ✅ 只计 2 条 USDT（去重 + 后缀 + quote 校验）；两种多资产信号各自阻断 ✓ |
| **PR-S4-4** 截止时间语义 | 30ms 超时 + 阻塞型 reader | ✅ 31ms 内结束、报告关闭（`read_only_checks_passed=false`，reason `lead_identity_request_failed`）、管理器状态不通过；**但 error 为 nil**（见 F1） |
| 生产接线检查 | `git grep leadaccount.` / `NewManagerFromEnv(` / `VerifyReadOnly(` / `SaveCredentials(` 在非测试代码 | ✅ **0 命中** → 包完全惰性，无启动接线、无路由 ✓ |

> 现有永久测试（`credentials_test.go` 4 个、`verify_test.go` 7 个、`account_client_stage4_1_test.go` 1 个）已覆盖：往返/轮换/密文不含明文/**0600**/错误主密钥/相对路径/0755 目录/符号链接**文件**/空 key/0644 后拒读/篡改/公开 JSON/只读成功仍阻交易/身份失败不读 futures/失败永不通过/非 Hedge+坏余额/旋转失效/晚到响应/再验证失败清成功。**本轮探针补充的是**：符号链接**父目录**、**可执行位**、超长输入、空 secret、**版本不符**、**临时文件卫生**、报告级 JSON 泄漏、白名单 USDT 过滤细节、多资产两分支、以及超时契约（均无永久测试）。

---

## 4. 自动化验证结果

| 项目 | 结果 |
| --- | --- |
| `go test -count=1 -race ./service/leadaccount ./feature/api/binance` | ✅ 均 ok（1.45s / 1.44s） |
| `go test -count=1 -race ./service/leadaccount ./feature/api/binance ./feature ./scanner -skip '^TestStaticStrategyTemplatesCompileWithoutDeprecatedMarketGlobals$'` | ✅ 全部 ok |
| `go vet ./service/leadaccount ./feature/api/binance` | ✅ 干净 |
| 根包 `go build -o /tmp/s41_bin .` | ✅ 成功（未在仓库留产物） |
| 探针 PR-S4-1～S4-4 | ✅ 全部通过；副本去掉探针后 `./service/leadaccount` 原测试 ok |
| 新增路由 / 表 / 迁移 | ✅ 无（`routers/`、`models/`、`appversion/`、`command/` 均未改） |
| 真实 Binance / 真实 Lead Key | ❌ 未执行（Gate 0-LIVE 仍阻塞，记录 §4 已声明） |

---

## 5. 风险与待改进

| # | 级别 | 问题 | 证据 / 建议 |
| --- | --- | --- | --- |
| **F1** | P3（接口契约） | **超时/失败与"验证完成但未通过"在返回值上不可区分**：`VerifyReadOnly` 在截止时间到期或任一阶段失败时返回 **`nil` error** + 关闭的报告（reason 固定为 `lead_identity_request_failed` 等）。若 Stage 6 的调用方只看 error 就认为"验证 OK"，会把超时误判为成功 | 探针 PR-S4-4 实测（30ms 截止 → 31ms 返回、reason=`lead_identity_request_failed`、error=nil）。建议二选一：① 当 `ctx.Err() != nil` 时返回该错误（与"本地失败"同一语义层）；② 或在报告里增加 `completed` 标志/文档明确"nil error 仅表示调用完成，必须读取 `read_only_checks_passed`"。安全性不受影响（`TradingReady` 恒 false ✓） |
| F2 | P3（设计边界，已声明） | 只读检查**无法证明** `/fapi` 数据来自用户选定的那个真实 Portfolio：`PortfolioBindingConfirmed` 恒 false，需人工对照币安页面 | 记录 §3 已明确 ✓；建议在 Stage 6 UI 上把该确认做成显式的人工勾选+证据记录（而不是仅展示文字） |
| F3 | P3（能力缺失，属后续阶段） | 本阶段**没有清除/禁用凭证的入口**（只能覆盖保存），也没有"上次保存时间/凭证指纹"等审计字段 | Stage 6 API 草案含 `DELETE /futures/lead/credentials` ✓；建议届时同时记录 `saved_at`/`key_hint` 以便轮换审计 |
| F4 | P3（可观测性） | 验证期间的 6 次请求分别标记 `lead_trading_sapi` 与 `lead_trading` 来源；但报告中不含本次验证的 API 用量/耗时，Stage 6 无法直接展示"验证消耗了多少权重" | 如需可在 `VerificationReport` 增加字段（或由调用方从 V4-4 快照按 source 取）；非阻塞 |
| F5 | P3（观察） | 45s 是**整体**预算：单个请求默认依赖 SDK/HTTP 客户端超时，若首步耗尽预算，后续步骤会直接以 ctx 取消失败（表现为多条 reason） | 与记录 §3"45 秒整体超时"一致 ✓；建议文档补一句"单请求超时未单独设置，依赖上下文预算" |
| — | 正面 | ① 只读接口边界（`AccountReader` 无写方法 + 编译期断言）使"绕过风控写入口"在本阶段**结构上不可能**；② 加解密路径每一次都重新校验文件权限/符号链接（防运行期被替换）；③ `SaveCredentials` 失败不改变旧验证状态（凭证未换则旧结论仍有效）✓ 逻辑自洽；④ 拒绝输入不破坏既有密文 ✓ |

---

## 6. 审计边界与未验证项

- **未验证**：真实 Lead Portfolio Key 的任何行为（Gate 0-LIVE）；本审计未发起任何真实 Binance 请求。
- **未验证**：真实 MySQL（本阶段无 DB 变更）；前端（无改动）。
- **未验证**：`PortfolioLabel` 与真实 Portfolio 的对照流程（人工步骤，属 Stage 6/7）。
- 未验证：SDK 层在真实网络下的 45s 预算表现（以 fake/阻塞 reader 验证了截止时间语义 ✓）。

---

## 7. 附：Stage 4-1 交付物清单与建议下一步

| 文件 | 规模 | 作用 |
| --- | --- | --- |
| `service/leadaccount/credentials.go` | 241（新） | AES-256-GCM 凭证仓库：环境密钥、绝对路径+权限/符号链接校验、原子写入（0600）、版本/AAD 绑定、fail-closed 解密、掩码与 JSON 隔离 |
| `service/leadaccount/verify.go` | 306（新） | Lead 专属 `Manager`（无全局注册）：显式保存/轮换（generation 失效）、45s 只读验证六步、固定 blocking reason、`TradingReady` 恒 false |
| `service/leadaccount/credentials_test.go` / `verify_test.go` | 139 / 239（新） | 凭证与验证的 11 个永久测试（含篡改、权限、符号链接、旋转失效、晚到响应、再验证失败清成功） |
| `feature/api/binance/account_client.go` | +9 | 新增只读 `GetPositionModeContext`（Hedge Mode 查询，不写账户） |
| `feature/api/binance/account_client_stage4_1_test.go` | 31（新） | 断言 GET `/fapi/v1/positionSide/dual`、Lead Key 签名、仅 1 次请求 |
| 文档 | — | `币安合约自动带单-Stage4-1-实施记录.md`；方案文档 4.1 段追加实施结果 |

**建议下一步**：① 就 **F1（超时契约）** 明确选择"返回 ctx 错误"或"文档化 nil error 语义"，避免 Stage 6 调用方误判；② 继续 **4-2**（账户级名义额/日亏损/保证金与熔断 Gate，仍保持 `enabled=false`）；③ 4-2 落地时把本期探针的 10 项边界（符号链接父目录、可执行位、版本不符、临时文件卫生、报告 JSON、白名单过滤、多资产分支、超时语义等）按需固化为永久测试；④ 真实凭证与真实只读能力仍等 **Gate 0-LIVE**，本阶段结论不得当作交易授权。
