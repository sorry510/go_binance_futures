# 币安合约自动带单 Stage 7-2 — Lead 私有 WS 只读接线与观察

> 日期：2026-10-11，分支 `feat/lead-trading`。**仅离线代码和 Mock 验收，不允许真实 Lead 交易。**

## 实施及实际证据

- 新增 `service/leadaccount/read_only_host.go`：`LeadReadOnlyHost` 仅支持显式观察，先执行 **SELECT-only** `ReadOnlyInspectOwnership`，读取 Lead account_id 的所有未决订单及包含 CLOSED 的受控仓位历史，随后启动独立私有 WS + 完整 REST 账户/订单/仓位快照。**绝不调用** `LeadSupervisor.Recovery`、`ApplyFill`、`ExecuteOpen`、`Submit`、`Cancel`，无 DB 订单状态修改。
- `main.go` 在 DB 版本启动检查之后加入**显式可选**只读观察接线，通过 `controllers.StartLeadReadOnlyOnExplicitOptIn` 进入，**仅** `BINANCE_LEAD_OBSERVER_MODE=read_only` 并且有安全 Lead AES-GCM 凭证文件/密钥时才可能连接 Lead 私有 API。**未设置环境变量则不读取凭证、不创建 WS、不访问 Lead API**；其余模式一律拒绝。
- Controller `/futures/lead/status` 和 `/metrics` 增加 `read_only_observer`、WS generation、镜像快照及原有 API 预算的真实观测状态；不会宣称观察代表 Portfolio 资格、授权开仓或恢复对账。状态 `live_trade_authorized=false` 永远保持。
- 密钥轮换时先 `StopReadOnlyHost`，避免旧 Key 的 ListenKey/WS 在保存新凭证后继续运行；观察器停止/取消等待是有界的。故障不会影响 Main，Context 取消不会留测试进程。
- 自动化：`TestStage72DisabledObserverDoesNotReadCredentialsOrNetwork`、`TestStage72ReadOnlyHostLifecycleNoExchangeMutations`、`TestStage72CredentialRotationBeforeRunCannotStartOldWS`、`TestStage72OwnershipReadOnlyInspectionDoesNotReconcile`。
- **真实 Gate 尚未测试**：本次不设置 opt-in、不使用用户真实 Key、不连真实 Lead Portfolio、不启动正在监听 3333 的后台新服务。将来部署并启动新编译版本时还需要明确配置和只读在线核查。

## 工作边界

没有修改 `app.conf`，没有运行 `sync db`、修改真实业务库或触发任何真实 Lead Submit/Cancel/REST/WS 测试；没有自动提交/推送 Git，不回退未提交的用户修改。
