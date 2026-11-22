# Code Review：统一表达式编译缓存与策略模板保存校验

> 审查对象：当前工作区改动（基线 `HEAD 1a720e0 fix: update code`）
> 声明的修改内容：「优先统一编译缓存和保存时校验」
> 审查方式：隔离副本 `/tmp/phase_audit_exprcache`（827 Go，14 个代码文件 SHA256 全等）+ 全套离线 Gate + 独立正确性探针 + 全局调用点核对
> 未改业务代码、未提交、未跑 `sync db`、未连真实 Binance、未写生产库。

---

## 一、结论

**当前改动：✅ 源码 / 离线 Gate 通过，可有条件放行；但「统一编译缓存」目标未完全实现（1 项 P2），须补充说明或补齐后再宣告完成。**

- 缓存实现本身正确：环境类型签名、LRU 有界、并发安全、函数绑定、错误不缓存均经验证；性能收益显著（热缓存约 11.7 倍、内存约 72 倍下降，合成 benchmark）。
- 保存校验集中在 `validateEditableStrategyTemplate`，Post 与 Edit 两条生产写入路径均已覆盖，AI 导入路径使用既有独立校验（逻辑完整）。
- **P2 阻断宣告「统一」**：全局仍有 7 处 `expr.Compile` 未迁移，其中 4 处是高频规则求值路径（正是缓存收益最大的场景）。以「统一编译缓存」为目标衡量，覆盖率约 10/17（≈59%）。
- 静态产物为正常前端重建（约 90 组 hash 文件 D+??），index.html 引用一致，无缺失。

**允许**：合并当前代码作为阶段性成果，但须明确「优先统一」的范围定义。
**禁止**：在未补齐遗漏路径或未明确声明范围前，宣称「表达式编译缓存已全局统一」。

---

## 二、改动与声明对齐

| 声明项 | 实现 | 判定 |
|---|---|---|
| 统一编译缓存 | 新增 `service/strategy/expr_cache.go`（`CompileRule`：LRU 256、环境签名、并发 Mutex），10 处调用点迁移至缓存 | ⚠️ 部分完成（P2） |
| 保存时校验 | 新增 `controllers/strategy_template_save_validation.go`，Post / Edit 加入校验，JSON 绑定错误返回 400，Edit 支持部分字段 patch | ✅ 完成 |
| 静默配套 | 9 处 controller/service 调用点统一；static 前端重建 | ✅ 一致 |

---

## 三、离线 Gate（全部通过）

| 检查 | 结果 |
|---|---|
| gofmt（14 文件） | ✅ 无未格式化 |
| `go vet`（strategy / controllers / feature / backtest / coinmexecution） | ✅ 退出码 0 |
| `go build ./...`（隔离副本） | ✅ 退出码 0 |
| `go test -count=1` 全量 88 包 | ✅ 76 ok + 12 no test + 0 FAIL |
| `go test ./service/strategy ./controllers -count=5` | ✅ 通过 |
| `go test -race ./service/strategy ./controllers` | ✅ 通过（LC_DYSYMTAB warning 非致命） |

---

## 四、缓存正确性实证

| 声明 | 证据 |
|---|---|
| 缓存键含环境类型签名 | `ruleEnvironmentSignature` 序列化变量名 + `reflect.Type.String()` + PkgPath；独立探针验证同名同结构不同命名类型（probeA/probeB）不共享程序 |
| 函数绑定不陈旧 | 开发方测试 + 独立探针 `TestProbeSharedProgramConcurrentRun`：200 goroutine 共享同一 `*vm.Program`，用不同函数值/价格 Run，结果全部正确；race 通过 |
| 价格/仓位不驻留缓存 | 开发方测试 `TestCompileRuleCacheReuseAndIsolation`：同签名不同价格复用程序，Run 时值正确 |
| 编译选项错误隔离 | 开发方测试验证 `asBool` / `allowUndefined` 属于缓存身份，失败表达式不缓存、修正后可编译 |
| 有界 LRU | `maxCompiledRules=256`，开发方测试注入 296 条，map 与 LRU 均稳定在 256 |
| 并发安全 | `TestCompileRuleConcurrentUse`（12 worker）+ 本次 200 goroutine 探针，race 全通过 |
| nil vs typed | 独立探针 `TestProbeNilThenTypedRuntime`：nil 环境编译、后续 typed 值 Run 正常 |

性能实测（`-benchtime=1000x -benchmem`）：热缓存 883.5 ns/op、152 B/op、10 allocs/op；未缓存 10,327 ns/op、11,048 B/op、69 allocs/op。缓存命中节省约 91% 时间与 98% 内存分配。该数据为合成 benchmark，不代表端到端回测路径收益。

---

## 五、Findings

### F1 / P2 —— 「统一编译缓存」未覆盖全部高频规则求值路径

- **事实**：全局（剪枝 research/specs）仍有 7 处 `expr.Compile`：
  - `controllers/strategy_template_ai_validation.go:62` —— AI 导入校验（一次性，影响小）
  - `feature/feature_debug.go:292` —— 硬编码调试代码（非生产路径）
  - `feature/feature_listen.go:64` —— **自定义规则监听，高频**
  - `feature/feature_test_strategy.go:103` / `:267` —— **模拟策略测试，高频**
  - `spot/spot_listen_rules.go:81` —— **现货监听规则，高频**
  - `service/workflow/data.go:200` —— workflow 规则（频率未知）
- 其中 feature_listen / feature_test_strategy / spot_listen_rules 的 env 均来自 `line.InitParseEnv`，与已迁移的 `feature/strategy/line/line_custom.go` 同构，无技术障碍。
- **影响**：声明目标是「统一编译缓存」，实际覆盖 10/17（约 59%）；监听与模拟测试路径仍每次重新编译，缓存收益最大的场景被遗漏。这不是正确性缺陷，但与声明不符，也会造成性能改进不可复现。
- **修复**（二选一）：①迁移剩余 5 处生产高频路径至 `CompileRule`，并补相应回归测试；②在方案 / 提交说明中明确「优先统一」的具体范围（例如「本轮只覆盖控制器 + 核心交易/回测路径，监听类后续处理」），避免「已全局统一」的误读。

### F2 / P3 —— name-only 编辑跳过规则校验（有意设计，需文档化）

- `Edit()` 仅当 `patch.Technology != nil || patch.Strategy != nil` 时调用 `validateEditableStrategyTemplate`；纯改名允许历史坏模板存在。
- 这是为兼容历史数据有意保留（代码注释明确），不是漏洞；但应在方案或 API 文档中记录，避免未来被误判为校验缺失。

### F3 / P3 —— 保存校验与编译缓存尚未共用同一实现

- `validateEditableStrategyTemplate` 最终调用 `validateStrategyTemplateRuleExpressions`（`strategy_template_ai_validation.go:62`），后者仍用原生 `expr.Compile` + `expr.Run`（校验需立即执行，逻辑正确）。AI 导入路径同理。
- 这不构成 bug（校验路径是一次性、且需要 Run），但意味着「保存时校验」与「编译缓存」是两条独立编译链。若未来规则函数的环境签名规则变化，需同步维护两处。

### F4 / P3 —— 静态产物重建属正常，但须与源码同步提交

约 90 组 `static/static/js/*` 删除+新增（hash 变化）与 `static/index.html` 的 `index-DJ1eOpOH.js` → `index-D3225lYt.js` 一致，CSS 未变，引用完整性验证通过（2/2）。此为前端构建产物替换，与本项目历史 static 重建模式一致。注意不要只提交源码而遗漏 static，否则线上页面仍加载旧入口。

---

## 六、放行 Gate

1. 就 F1 作出选择：补齐 5 处高频路径，或书面明确「优先统一」范围；二者完成其一即可宣告本阶段目标达成。
2. F2/F3 属于文档与维护性事项，不阻塞合并，建议随本次提交补充说明。
3. 静态产物须与源码一并提交，保持 index.html 引用与新 hash 文件成套。
4. 源码 / 离线 / 合成 benchmark 通过 ≠ 真实行情、真实历史回测、MySQL 8 或实盘验证；本轮未涉及实盘交易逻辑改动，也未运行 `sync db`。

**当前状态：源码与离线测试通过，缓存实现正确且收益显著；因 F1 与目标表述相关，判定「有条件放行」，不宣称「全局统一」完成。**

> 审查探针仅写在隔离副本，报告落盘后清理；用户工作区保持交付时状态。


---

## 七、二次复核与 F1 修复结果（2026-11-22）

本节为原审查之后的**追加复核**，不覆盖前述审查时点的事实。

### F1：已修复

原报告列出的全部 7 个遗漏 `expr.Compile` 调用点，现均统一使用 `strategyservice.CompileRule`：

| 调用点 | 修改 |
|---|---|
| `feature/feature_listen.go` | 合约自定义监听规则共用编译缓存 |
| `feature/feature_test_strategy.go` | 模拟开仓、平仓共用编译缓存（2 处） |
| `spot/spot_listen_rules.go` | 现货自定义监听共用编译缓存 |
| `service/workflow/data.go` | 实验规则以原有 `AsBool` 选项调用缓存 |
| `controllers/strategy_template_ai_validation.go` | AI 导入与保存校验以 `AsBool` 调用同一缓存，保留原有 `expr.Run` 验证 |
| `feature/feature_debug.go` | 调试入口也使用统一编译缓存 |

全局重新检索 `feature/`、`spot/`、`controllers/`、`service/` 的非测试 Go 代码后，只剩 `service/strategy/expr_cache.go` 作为中央 `expr.Compile` 实现。**此前 10/17 的部分覆盖现已补全至本次检出的 17/17。** 本次仅替换编译入口，继续由各自原调用方执行 `expr.Run`，不改变业务信号处理、通知、成交或结算逻辑。

### F2 / F3 / F4 处理

- **F2：保留现状（有意行为）。** 仅修改模板 `name` 时不触发历史规则重新校验；修改 `technology` 或 `strategy` 时以合并后的模板完整校验。避免历史旧模板因无关改名而无法保存。
- **F3：已统一。** `validateStrategyTemplateRuleExpressions` 现调用 `strategyservice.CompileRule(rule.Code, env, true, false)`，与普通策略共用缓存；仍立即执行表达式、检查运行错误和布尔返回值，未降低保存校验要求。
- **F4：无须额外修改。** 本轮补齐调用点仅涉及后端 Go 源码，未修改前端源码或再次重建 `static`；仍须在最终提交时把此前已构建的前端静态文件与对应前端源码一起保留。

### 二次复核 Gate

| 检查 | 结果 |
|---|---|
| 全部实际业务 Go 包 `go test -count=1 -p=3`（排除 `research_specs`） | ✅ 75 包测试通过 + 12 包无测试文件，退出码 0 |
| `go test -race`：strategy、feature、spot、workflow、controllers 关键回归用例 | ✅ 通过，macOS 链接器出现非致命 LC_DYSYMTAB warning |
| `go vet`：strategy、feature、spot、workflow、controllers、backtest、coinmexecution | ✅ 通过 |
| Go 后端 `go build -o /tmp/s510_expr_review_backend_build .` | ✅ 通过 |
| `git diff --check` | ✅ 通过 |
| 直接 `expr.Compile` 生产调用点 | ✅ 仅保留中央缓存实现 |

**关于 `go test ./...`：** 工作区现有 `research_specs/` 的多个研究批次目录混放不同包名（如 `main` 与 `backtest`）或将独立测试拷贝置于非原包目录，导致直接 `go test ./...` 在研究资料目录报错。这与本轮 Go 生产代码修改无关；已改用明确的业务包路径执行完整测试。未清理或移动研究文件。

**更新结论：F1 已补齐，F2 按兼容需求保留，F3 已收敛，F4 无新问题。当前二次复核可放行本轮代码变更（以静态离线回归通过为依据）。未执行真实行情/实盘验收，未重启生产服务、未迁移/写入生产数据库，亦未提交 Git。**
