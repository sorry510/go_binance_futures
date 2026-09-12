# V3-4 补充审查：下载 / ZIP-CSV 解析路径的循环终止性分析

- **审查对象**：`service/historicalmarket/public_data.go`（本轮由 629 → 768 行）、`resolution_provider.go`（383 → 410 行）、`sparse_repository.go`（215 → 225 行）
- **基线**：HEAD `cd950cf feat: ai agent v3-4`，其上叠加工作区未提交改动（`public_data.go` +172、`environment.go` 68 行、两个测试文件 +141）
- **审查类型**：**review-only**（仅审查，未修改任何代码）
- **核心问题**：下载与 ZIP/CSV 解析逻辑是否会造成死循环？
- **结论**：**不会造成死循环。** 所有循环均具备严格单调的推进变量或有限迭代边界；新引入的 `publicDataCSVScanner` 状态机的 `pending` 分支设计上不存在"消费→重建→再消费"的同调用内自循环。另发现 **3 项非死循环类缺陷**（1 项状态污染、1 项语义瑕疵、1 项测试缺口）。

---

## 1. 逐循环终止性证明

| # | 循环                                                              | 位置                              | 推进变量 / 退出条件                                                                                                             | 结论 |
| - | --------------------------------------------------------------- | ------------------------------- | -------------------------------------------------------------------------------------------------------------------- | -- |
| 1 | `for _, char := range value`                                    | `public_data.go:213`            | range 有限迭代                                                                                                            | ✅  |
| 2 | `for attempt := 0; attempt <= client.maxRetries; attempt++`     | `public_data.go:343`（`downloadFile`） | `attempt` 严格递增；`maxRetries` 在构造函数中被规范化（`<0→0`，`==0→2`），恒为有界正整数，循环上界 = maxRetries+1 次                                                    | ✅  |
| 3 | `for attempt := 0; attempt <= client.maxRetries; attempt++`     | `public_data.go:396`（`downloadSmall`） | 同上                                                                                                                     | ✅  |
| 4 | `for _, item := range reader.File`                              | `public_data.go:627`            | range 有限迭代；且命中首个 `.csv` 后**立即 `return`**，不会继续遍历其余条目                                                                     | ✅  |
| 5 | **`for {`**（`scanRange`）                                          | `public_data.go:684`            | **见 §2 逐路径论证**                                                                                                        | ✅  |
| 6 | `for start := 0; start < len(normalized); start += repo.chunkSize(17)` | `sparse_repository.go:79`       | `chunkSize()` 实现 `if size < 1 { return 1 }`（`repository.go:594-603`），**保证 >= 1**，`start` 严格递增；MySQL 分支固定返回 200                                      | ✅  |
| 7 | `for start := 0; ...; start += repo.chunkSize(13)`              | `sparse_repository.go:154`      | 同上                                                                                                                     | ✅  |
| 8 | `for _, trade := range ordered`                                 | `resolution_provider.go:380`    | range 有限迭代                                                                                                            | ✅  |
| 9 | `for fundingIndex < len(dataset.Funding) && ...FundingTime <= until` | `engine.go:87`（`advanceFunding`）  | `fundingIndex++` 单调递增（`position == nil` 的 `continue` 发生在自增之后），上界为 `len(Funding)`                                                | ✅  |
| 10 | `for _, item := range funding`                                 | `adaptive_engine.go:166`        | range 有限迭代                                                                                                            | ✅  |
| 11 | `for _, second := range seconds` / `for _, trade := range trades` | `adaptive_engine.go:101/207`    | range 有限迭代                                                                                                            | ✅  |

---

## 2. `scanRange` 状态机逐路径论证（本次改动的高风险点）

`scanRange`（`public_data.go:683-751`）的结构：

```
for {
  A. ctx.Err() != nil                        → return err      （退出）
  B. scanner.eof                             → return nil      （退出）
  C. pending != nil                          → 取出并置 nil     （状态变化，fall-through 到 E）
     else                                    → reader.Read()
                                                 EOF → eof=true; return nil   （退出）
                                                 err → return err             （退出）
                                                 成功 → scanner.rowNumber++    （推进）
  D. timeColumn >= len(record)               → return err      （退出）
  E. parseArchiveInt64(record[timeColumn]) 失败：
        rowNumber == 1                       → visit(); continue  （回到 A）
        rowNumber != 1                       → return err      （退出）
  F. end > 0 && recordTime > end             → 设 pending; return nil（退出）
  G. recordTime > lastTime                   → 更新 lastTime
  H. start > 0 && recordTime < start         → continue        （回到 A）
  I. visit(record, rowNumber)                → errStopCSV → return nil（退出）
                                               err        → return err（退出）
                                               nil        → 继续循环
}
```

### 关键论证 1：两条 `continue` 路径都伴随 `reader.Read()` 推进

- 路径 **E（`rowNumber == 1`）**：进入该分支的前提是本轮走的是 `else` 分支并成功 `Read()`，因此 `scanner.rowNumber` 已自增为 1。`continue` 时 **pending 为 nil**，下一轮必然再次 `Read()`，`rowNumber` 变为 2。由于 `rowNumber == 1` 是单调条件，该分支**最多进入一次**；第二行若时间解析仍失败则走 `rowNumber != 1` 的 `return err`。→ 不会无限循环。
- 路径 **H（`recordTime < start`）**：记录来源只有两种，且两种都在本轮使状态变化 ——
  - 来自 `pending`：`pending` 已在 C 中被置 `nil`，下轮必然 `Read()`；
  - 来自 `Read()`：`rowNumber` 已自增，下轮 `Read()` 得到新行。
  → 每轮都有实质推进，不会无限循环。

### 关键论证 2：`pending` 机制不存在自循环

这是本改动的核心优化（避免对同一 daily ZIP 反复从字节 0 解压）：

- **消费 pending 的分支（C）没有 `continue`**，它 fall-through 到 F/G/H/I，正常推进；
- **设置 pending 的分支（F）必定紧接 `return nil`**，不会在同一调用内再次进入循环体。

因此不存在「取出 pending → 重新写入 pending → 再取出」的同调用内闭环。`pending` 跨调用保留的设计只会让下一次调用**从 pending 处继续**，而每次调用最多设置一次 pending 即返回。

### 关键论证 3：`start` 回退时的正确性（重开而非死循环）

`readPublicDataCSVRange`（:662-681）在
`scanner == nil || scanner.archiveURL != archive.URL || (start > 0 && scanner.lastTime > 0 && start < scanner.lastTime)`
时关闭旧 scanner 并**重新打开**（`openPublicDataCSVScanner`）。即"调用方范围回退"被显式转换为"整包重开"，是**有界的一次性动作**，不是循环。

### 关键论证 4：无死锁

- `readPublicDataCSVRange` 全程持 `client.parseMu`；在持锁期间**不获取** `client.mu`。
- `client.mu` 仅在 `cachedArchive` / `archiveUnavailable` / `markArchiveUnavailable` 中短暂持有，且**不嵌套** `parseMu`。
- `Close()` 先取 `parseMu` 关闭 scanner，再释放，之后才做 `os.RemoveAll`，不与 `mu` 形成交叉等待。
→ 锁序无环，无死锁。

---

## 3. 验证结果

```
go test -count=1 -timeout 120s -race -run "TestPublicData|TestResolutionProvider|TestSparseTrade|TestAggregateTrades" \
    ./service/historicalmarket/
```

- 结果：**18 个用例全部 PASS，耗时 2.018s**，未触发 `-timeout`（若存在死循环会以 timeout 失败）。
- 覆盖了下载重试（5xx）、404 不重试、CHECKSUM 失败 fail-closed、singleflight 去重、ctx cancel、scanner 跨 range 续读等关键路径。
- 其中 `TestPublicDataTradeScannerStreamsForwardAcrossRanges` 与 `TestPublicDataKlineScannerStreamsForwardAcrossRanges` 直接验证了 `pending`/续读机制：断言第二次请求复用**同一个 scanner 实例**且 `rowNumber` 不回退。

---

## 4. 发现的问题（均非死循环）

### 缺陷 1（P2，状态污染 —— 会导致持久性失败但不会循环）

**位置**：`public_data.go:670-681`

```go
scanner := *scannerSlot
if scanner == nil || scanner.archiveURL != archive.URL || (...) {
    closePublicDataCSVScanner(scanner)     // 关闭旧 scanner
    var err error
    scanner, err = openPublicDataCSVScanner(archive)
    if err != nil {
        return err                          // ← *scannerSlot 仍指向已 close 的 scanner
    }
    *scannerSlot = scanner
}
```

**问题**：`openPublicDataCSVScanner` 失败时提前 `return`，但 `*scannerSlot` 未被清空，仍指向刚刚被关闭的 scanner。后续若 `archive.URL` 相同且 `start >= lastTime`，条件判定为"可复用"，于是复用这个**已关闭**的 scanner —— `reader.Read()` 会返回 "file already closed" 类错误，且**每次调用都在同一处失败**（状态无法自愈，除非 URL 变化或 `start` 回退触发重开）。

**影响**：archive 文件不可读时会进入持久失败态，而非在下次调用时重新尝试打开。
**建议**：在失败分支 `return err` 之前补 `*scannerSlot = nil`。

### 缺陷 2（P2，语义瑕疵）

**位置**：`public_data.go:716-725`

```go
if rowNumber == 1 {
    if err := visit(record, rowNumber); err != nil && !errors.Is(err, errStopCSV) {
        return err
    }
    continue
}
```

**问题**：第一行调用 `visit` 时若返回 `errStopCSV`（语义为"立即停止扫描"），会被条件式**静默忽略**并继续扫描。当前 `ParseKlines`/`ParseTrades` 的 header 路径都返回 `nil`（不在首行返回 `errStopCSV`），因此**实际不触发**；但语义上首行不应例外。
**建议**：改为对 `errStopCSV` 显式 `return nil`，与 `scanRange:744-749` 保持一致。

### 缺陷 3（P3，测试缺口）

现有两个 scanner 测试只覆盖**范围单调前进**的场景。**未覆盖**：
- `start < scanner.lastTime` 时的回退重开路径（含重开后结果正确性）；
- `openPublicDataCSVScanner` 失败后的状态（对应缺陷 1）。
**建议**：补一条"range 回退 → scanner 重开 → 结果仍正确"的用例。

---

## 5. 正向确认（相对上一轮 review 的改进）

本次改动顺带修复了上一版 `CODE_REVIEW_V3-4.md` 中提出的两处问题：

1. **404 负缓存**：新增 `unavailable map[string]struct{}`（`public_data.go:93`）与 `markArchiveUnavailable`（:279-283），`FetchArchive` 在 :234 与 :246 提前短路 —— 同一 Run 内同一缺失 archive 不再重复发起 HTTP 404（原 I-7 类问题）。
2. **SHA256 合法性校验**：新增 `validSHA256Hex`（`sparse_repository.go:218-225`，`hex.DecodeString` + 长度双重校验），并用于 `sparseTradeRangeCached`（`resolution_provider.go:284`）—— 修复原 I-9（仅校验长度）。
3. **范围覆盖判定**：`sparseTradeRangeCached` 改为解析 `sourceRefRange` 并检查 `coverageStart <= start && coverageEnd >= end`（:283-286），使"父区间缓存可被子区间请求复用"，减少重复下载。

**其它边界说明（不构成缺陷）**：
- `waitRetry` 的 `delay := retryBackoff * time.Duration(1<<attempt)`（:436）在极端 `MaxRetries`（≥ 33）配置下位移溢出会使退避失效（timer 立即触发），但 `attempt` 仍单调递增、循环仍有界，**不会死循环**；默认 `maxRetries=2` 不受影响。
- `MaxRetries == 0` 被强制为 2（:142-144），无法通过配置显式禁用重试 —— 配置语义问题。
- `readPublicDataCSVRange` 使用单一 `parseMu` 串行化 klines 与 trades 解析，二者不能并行；为性能取舍，非缺陷。
- `io.Copy` 与 `csv.Reader` 底层均有标准库的 `ErrNoProgress` 保护（连续 100 次空读即报错），进一步排除了"底层 reader 不推进导致挂死"的可能。

---

## 6. 结论

**下载与 ZIP/CSV 解析逻辑不会造成死循环。**

- 4 类下载重试/遍历循环均有严格有界的推进变量；
- `scanRange` 的 `for {}` 是唯一的事件驱动循环，其两条 `continue` 路径均伴随 `reader.Read()` 推进，`pending` 机制不存在同调用内自循环；
- 分块入库循环依赖 `chunkSize() >= 1` 的硬保证；
- 无死锁（锁序无环）；
- `-race -timeout 120s` 下 18 个相关用例全部通过。

建议修复缺陷 1（`*scannerSlot` 失败未清空，P2）与缺陷 2（首行 `errStopCSV` 语义，P2），并补充缺陷 3 的回退重开测试。其余为配置语义与性能取舍，不影响正确性。


## 7. 复核处理记录（2026-09-12）

本轮已按审查结果完成处理：

- **缺陷 1（P2）已修复**：scanner 关闭后会立即清空 slot；若重新打开 ZIP 失败，不会残留已关闭 scanner，后续请求可重新尝试。
- **缺陷 2（P2）已修复**：CSV 首行 visitor 返回 `errStopCSV` 时立即正常结束扫描，与后续行语义一致。
- **缺陷 3（P3）已补测试**：增加 range 回退后重新打开 scanner 并验证结果正确；同时覆盖 scanner 重开失败后 slot 必须为 nil。
- 下载/解析阶段现已向 Backtest Run 暴露 `downloading_intrabar_archive` / `parsing_intrabar_archive`，便于前端区分高精度数据读取、ZIP 下载和 ZIP 解析。
- 针对长回测性能，新增 ROI-only 动态平仓规则证明：当平仓规则除 ROI 外只依赖已完成 `[1+]` 历史输入且分钟内 MarketCondition 不变化时，会先证明当前 1m/1s ROI 范围内规则是否可能成立，不可能成立则不进入 1s/trades。

真实 BTCUSDT 2026-01-01 ~ 2026-01-15 benchmark：Dataset 约 0.36s，Adaptive Engine 约 50.3s；1s 下钻 55 分钟、Trades 下钻 2,859 秒。相比问题 Run 同阶段约 2,290 个高精度分钟，候选量显著下降。
