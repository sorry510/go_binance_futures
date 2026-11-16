# System Configuration and Logs

[中文](../zh-CN/08-系统配置与日志.md) · [Back to English README](../../../README.EN.md)

## System Settings

- The **System Configuration** page contains sensitive values such as API keys, database passwords, and notification tokens. Do not include it in screenshots, issues, or Git commits.

---

## System Diagnostics and Log Cleanup

When the Web UI is unavailable or you need quick diagnostics on the server, run the read-only command:

~~~bash
./go_binance_futures doctor
~~~

`doctor` only reads the database, current configuration, and external connectivity. It does not start Web, WebSocket, Scheduler, or trading loops, perform reconciliation, or silently migrate the schema.

When `Overall=error`, `doctor` exits with status `1`. Healthy, Warning, and Disabled return `0`, which scripts can use to distinguish critical failures.

To clean up old log-only records after extended operation, run:

~~~bash
./go_binance_futures cleanup logs --before-days 90
~~~

This removes only allowlisted records older than 90 days: `agent_observations`, `agent_change_events`, `agent_task_events`, `agent_alert_pipeline_traces`, and `notifications`. It does not delete market data, Funding, Backtest, Strategy, Paper/Live Trade, Ownership, Agent Trade Proposals/Executions/Audits, Opportunities, Conversations, Memory, Skill/MCP settings, or Workflows. `--before-days` must be greater than zero.

---

## Logs

While the service runs, logs are written both to the console and to `logs/app.log` in the process working directory. Files rotate daily or at 20 MiB, and historical logs are retained for at most seven days. The viewer scans at most 2 MiB per request and returns at most 1,000 lines, with severity and keyword filters and automatic refresh every five seconds.

A small amount of third-party output written directly to stdout/stderr rather than through Beego remains available in PM2's console logs.

---
