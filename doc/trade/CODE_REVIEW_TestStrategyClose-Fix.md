# 审查：测试策略平仓"通知成功但未结算"修复

- **审查对象**（工作区未提交，基线 `HEAD = 9ffe4f5` / `master`）：
  - 新增 `feature/feature_test_strategy_close.go`（结算公共逻辑）
  - 新增 `feature/feature_test_strategy_close_test.go`（2 个用例）
  - 修改 `feature/feature_test_strategy.go`（两条平仓路径改为调用公共逻辑）
  - 事故说明 `doc/trade/CODE_REVIEW_Incident-TestStrategyClose-FalseWS.md`
- **审查方式**：只读代码审查 + 实跑测试（未改动任何文件）
- **结论**：**修复正确、完整，建议合并**。它不仅修好了"误报"，还顺带消除了**导致更新失败的整类原因**（见 §2.2），这是本次改动最有价值的点。附 4 项 P3，其中 **T2（3 条历史行会在上线后按当前价立即结算）需要你在上线前决策**。

---

## 1. 缺陷与修复对照

| 旧行为 | 风险 | 新行为 |
| --- | --- | --- |
| `orm.NewOrm().Update(result)`：整行覆盖写，**既不检查 error 也不检查 affected rows**，随后直接发 `futures_test` 平仓通知并计算连续盈亏 | ① 更新失败时 `close_price` 仍为 `'0'` → 下一轮重复加载 → **每轮重复推送"已平仓"**（即事故现象）；② 通知与实际结算脱节；③ 真实错误被丢弃，事后无法定位 | `finalizeTestStrategyClose(...)` 返回 `(updated, err)`：<br>· **CAS**：`WHERE id=? AND close_price='0'`，只更新平仓相关字段；<br>· `err != nil` → 记录含 id/symbol 的错误并 `continue`（**不通知**）；<br>· `affected != 1` → warning 并 `continue`（**不通知**）；<br>· 仅 `updated` 时同步内存对象 → 才发通知 / 走 `autoTestToTrade` |

两条平仓路径（自定义 `close_long/close_short` 策略、无平仓策略时 `±10%` 兜底）**都已替换**，没有遗漏的写入口（仓库内已无其它 `close_price` 写入点）。

---

## 2. 正确性核对

### 2.1 CAS 与通知的耦合（核心）
- 只有 `affected == 1 && err == nil` 才置内存字段并通知 ✓ → **"通知即已结算"** 成立 ✓（旧的反向漏洞被消除）。
- `affected == 0` 视为"已被其它周期结算或行已删除" ✓ → 不重复通知 ✓；两周期并发时只有一个能拿到 `1` ✓。
- 失败时 `continue` → 同时跳过 `autoTestToTrade(...)` ✓（不会因一次失败平仓而翻转测试开关）✓；且不会误伤同批其它行的评估 ✓。

### 2.2 从"整行更新"改为"只写平仓字段"——顺带消除了失败面
`test_strategy_results` 保存了开仓时的**大字段快照**（`technology`、`open_strategy`、`strategy` 等 TEXT；多个 `size(64)/size(128)` 的 hash/名称字段）。旧实现每次平仓都把整行重写一遍，**只要任一历史字段与当前列定义不兼容（超长、NULL 约束等）整条 UPDATE 就会失败**，而错误又被丢弃 → 正是"通知了但没结算"的高概率成因。
新实现只写 `close_price/close_profit/close_strategy/close_strategy_name/close_strategy_type/close_strategy_hash/updateTime` 七个字段 ✓ → 与开仓快照解耦 ✓（新增测试也断言"开仓快照未被修改"✓）。**这是根因级修复，而不是给 `Update` 补一个 `if err`**。

### 2.3 价格与数值语义等价
- 旧代码写 `position.MarkPrice`，而 `position.MarkPrice` 本身就是 `strconv.FormatFloat(floatNowPrice, 'f', -1, 64)`（`feature_test_strategy.go:236`）→ 新代码 `priceText` 用**相同格式** ✓ → **DB 值不变** ✓。
- 顺带修正：旧代码 DB 里写 `position.MarkPrice`、通知里写 `floatNowPrice`（两者虽同源但类型不同）；现在两边一致 ✓。
- `updateTime`：旧 `time.Now().Unix()*1000` vs 新 `UnixMilli()` ✓ 等价 ✓。
- 通知里的 `Profit` 取 `result.CloseProfit`（`"%.3f"`）✓ 与成功写入的值一致 ✓。

### 2.4 入参校验
`price > 0 && 有限`、`netProfit` 有限、`code` 非空、`name ≤ 128`、`type ≤ 32`（与列宽一致 ✓）、`result.ID > 0` ✓；全部在**写库前**拒绝 ✓（没有"写了一半"的路径）✓。测试 `TestFinalizeTestStrategyCloseRejectsInvalidInputsWithoutUpdating` 覆盖 ✓。

### 2.5 健壮性
- `Filter("close_price", "0")` 命中 `close_price` 的**主键 + 单列条件** ✓，SQLite/MySQL 行为一致 ✓。
- 更新后只改内存对象的**平仓字段** ✓，不会污染后续循环 ✓。

---

## 3. 验证结果

| 项目 | 结果 |
| --- | --- |
| `go test -count=1 -race ./feature` | ✅ ok（含新增 2 个用例） |
| `go test -count=1 -race ./service/strategy` | ✅ ok |
| 专项 `TestFinalizeTestStrategyCloseOnceOnlyAndFields` / `...RejectsInvalidInputsWithoutUpdating` | ✅ PASS |
| `go vet ./feature`、根包 `go build`、`gofmt -l`（3 个文件） | ✅ 干净 |
| `./controllers` | ⚠️ 5 个失败：`TestQueryStrategyTemplatePageOrdersByIDDescending`（`DataBase alias 'default' already registered`，**测试顺序耦合**）+ 3 个策略模板 AI/校验 + 静态模板。**已在 `git archive HEAD` 干净副本上完整复现同样 5 项** → **与本次改动无关**，属既有问题 |

---

## 4. 建议（P3）

| # | 级别 | 问题 | 建议 |
| --- | --- | --- | --- |
| **T1** | P3 | CAS 只匹配**字面** `'0'`。若存在历史行是 `''` / `'0.0'` / `'0.000'`，将永远无法结算（每轮输出 warning）。 | 改为 `Filter("close_price__in", []string{"0",""})` 或用 `close_price+0 = 0` 的原始条件；一行改动，零风险。 |
| **T2** | **P3（上线前决策）** | 生产那 3 条 `close_price='0'`（OP/MORPHO/XMR）会在**部署后的下一轮**被重新评估：若仍满足平仓条件 → **立即按当前价结算**，可能产生很大的"利润/亏损"并计入统计 | 上线前先决定：① 接受自动结算（并在上线后立即核对该 3 行的 `close_profit`）；② 或先把它们标记为作废/人工平仓，再上线。建议上线后第一件事就是核对这 3 条。 |
| **T3** | P3 | 若底层错误**持久**（例如列类型不匹配），每 2 秒一轮会各打一条 error 日志（含 id/symbol ✓ 无敏感信息 ✓） | 加简单限频（同一 id 每分钟一条）或只在该行错误码变化时打印；避免日志刷屏掩盖其它告警。 |
| **T4** | P3（可接受） | 提交成功后、通知前进程崩溃 → 行已结算但**没有通知**（至多一次通知的弱化版） | 当前规模可接受；若要求严格"必达"，需 outbox/事件表 + 重投，不建议现在做。 |
| **T5** | 诊断建议 | 想确认这 3 条**当初为何更新失败**，可跑一次只读查询比对字段长度：`SELECT id, symbol, CHAR_LENGTH(open_strategy), CHAR_LENGTH(close_strategy), CHAR_LENGTH(strategy), CHAR_LENGTH(open_strategy_name), CHAR_LENGTH(close_strategy_type) FROM test_strategy_results WHERE close_price='0';` 若有值超过列宽（如 `*_name > 128`）即印证 §2.2 推断 | 仅只读，安全 |
| **T6** | 口径 | 本修复只作用于 `futures_test` **本地模拟账本**，与 Binance 真实仓位无关（文档已强调 ✓） | 若用户同时认为"真实仓也没平"，需另查实盘下单/平仓链路（与本次改动无关） |

---

## 5. 审查边界

- 未连接生产库、未执行任何 `UPDATE`、未重启服务；T5 的 SQL 仅作建议。
- 未做浏览器/WS 端到端验证（本次无 UI 改动；通知仅在结算成功后发送，逻辑已由单元测试锁定）。
- `controllers` 的 5 项失败为既有问题（HEAD 干净副本可复现），不在本次范围内，建议单独立项修复（其中 `TestQueryStrategyTemplatePageOrdersByIDDescending` 失败原因是 beego `default` 别名重复注册，属测试隔离缺陷，容易误伤 CI）。
