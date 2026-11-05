# 测试策略平仓成功通知误报与重复 WS 通知排查（2026-11-05）

## 现象与范围

用户反馈：ARM 服务器测试策略持续通过 WS 推送“正常平仓”，但测试持仓没有被结算。

经只读 SSH + MySQL 8.0.35 查询，生产库名称为 `go_binance`，`test_strategy_results` 有 278 条记录，其中 `close_price='0'` 共 3 条（查询时间约北京时间 2026-11-05 10:25），涉及 OPUSDT、MORPHOUSDT 和 XMRUSDT。线上日志没有记录原 ORM 更新的返回值。本次只读调查没有重写或结算这些记录，也没有修改线上服务。

**测试策略平仓**属于本地模拟交易记录结算，与 Binance 上真实合约仓位的平仓订单不同；此链路不会向交易所提交平仓。

## 已确认的代码缺陷

`feature/feature_test_strategy.go` 的 `CheckTestResults` 包含两条平仓路径：
1. 命中 `close_long / close_short` 的自定义策略；
2. 未配置平仓策略时触发系统 ROI 兜底条件。

旧实现都先修改内存对象，随后调用 `orm.NewOrm().Update(result)`，但**既不检查 error，也不检查 affected rows**，直接发送标记为 `futures_test` 的平仓 WS/通知，且计算连续盈利次数。

当 DB 更新失败或没有成功更新该行时，`close_price` 依然是 `0`，后续轮询重复加载并发送“已平仓”通知。这一逻辑缺陷已经直接确认；由于旧程序丢弃实际错误，**此次不能从旧日志证明底层 SQL 失败的具体原因**。

## 本地修复

新增 `feature/feature_test_strategy_close.go`：
- 只更新 `close_price`、`close_profit`、平仓策略信息及 `updateTime`；不重新写入开仓时的行情/指标/策略快照，降低旧数据不兼容风险。
- 使用 `WHERE id = ? AND close_price = '0'` 的原子条件更新（CAS），防止两个检查周期重复结算。
- 仅在 `affected == 1 && err == nil` 时修改内存对象，并允许发送成功通知以及增加连续盈利计数。
- `affected == 0` 视为该记录已被其他任务处理或不存在，不再次通知；出现 SQL 错误则记录包含模拟仓位 id、symbol 的错误日志。
- 拒绝非正数、NaN/Inf 平仓价格及非法收益，并在本地校验策略字段长度。

`feature/feature_test_strategy.go` 两条平仓路径均已调用公共保存逻辑，删除原先忽略错误的 `Update(result)`。

## 验证及上线注意事项

新增 `feature/feature_test_strategy_close_test.go`：在独立 SQLite 测试库检验第一次结算成功、二次结算被 CAS 拒绝、开仓快照保留、非法值不写入。

**没有执行真实下单、生产库 UPDATE、生产数据库迁移或线上服务重启**。应先部署包含本次改动的服务二进制；若同类事故再发生，新日志会清楚输出底层 SQL 错误，而不误报“平仓成功”。部署后核对测试页面的 `close_price` 是否真的不再为零，及对应 WS 通知是否只发生一次。

若用户所说的是 **Binance 真实仓位尚未平仓**，需另行核对 Binance 实盘订单/持仓数据；本次 `futures_test` 的 WS 通知不是实盘平仓成交证明。
