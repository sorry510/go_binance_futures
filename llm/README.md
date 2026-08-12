# LLM 基础接入

业务代码只依赖 `Client` 接口，通过 `conf/app.conf` 的 `llm.provider` 切换实现。

支持的 provider：

- `openai`、`chatgpt`：OpenAI Chat Completions API
- `openai_compatible`：OpenAI Chat Completions 兼容协议
- `anthropic`、`claude`：Claude Messages API
- `claude_sdk`：Claude Agent SDK 子进程桥接
- `codex_sdk`：Codex SDK 子进程桥接

基础调用：

```go
client, err := llm.NewFromConfig()
if err != nil {
	return err
}

response, err := client.Generate(ctx, llm.Request{
	System: "You are a concise assistant.",
	Messages: []llm.Message{
		{Role: llm.RoleUser, Content: "Summarize the market data."},
	},
})
```

SDK 桥接需要 Node.js 18 或更高版本，并在首次使用前安装依赖：

```bash
cd llm/bridge
npm install
```

Claude/Codex SDK 桥接默认禁用写入权限。`Response.SessionID` 可传回下一次请求的 `Request.SessionID`，用于恢复 SDK 会话。
