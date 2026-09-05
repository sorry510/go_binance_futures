package routers

import (
	"go_binance_futures/controllers"

	"github.com/beego/beego/v2/server/web"
)

func init() {
	web.Router("/login", &controllers.LoginController{}, "post:Post") // 登录

	web.Router("/service/config", &controllers.IndexController{}, "get:GetServiceConfig;put:EditServiceConfig")                          // 服务配置信息
	web.Router("/test-pusher", &controllers.IndexController{}, "post:TestPusher")                                                        // 测试推送
	web.Router("/update-market-condition", &controllers.IndexController{}, "post:UpdateMarketCondition")                                 // 手动触发更新市场状态
	web.Router("/update-market-condition/:taskId", &controllers.IndexController{}, "get:GetMarketConditionUpdateTask")                   // 查询市场状态更新进度
	web.Router("/agents/chat/conversations", &controllers.AgentChatController{}, "get:ListConversations;post:CreateConversation")        // Chat 对话列表/新建
	web.Router("/agents/chat/conversations/:id", &controllers.AgentChatController{}, "put:UpdateConversation;delete:DeleteConversation") // Chat 对话标题更新/删除
	web.Router("/agents/chat/conversations/:id/messages", &controllers.AgentChatController{}, "get:Messages;post:SendMessage")           // Chat 消息/发送
	web.Router("/agents/chat/skills", &controllers.AgentChatController{}, "get:Skills")                                                  // Chat-capable Skill 列表
	web.Router("/agents/tasks", &controllers.AgentController{}, "get:ListTasks;post:StartTask")                                          // 创建/查询统一 Agent 任务
	web.Router("/agents/tasks/:taskId", &controllers.AgentController{}, "get:GetTask")                                                   // 查询统一 Agent 任务
	web.Router("/agents/tasks/:taskId/cancel", &controllers.AgentController{}, "post:CancelTask")                                        // 取消运行中的 Agent 任务
	web.Router("/agents/tasks/:taskId/resume", &controllers.AgentController{}, "post:ResumeTask")                                        // 从安全 Checkpoint 恢复 Agent 任务
	web.Router("/agents/memories", &controllers.AgentMemoryController{}, "get:List;post:Create")                                         // 长期 Memory 列表/新增
	web.Router("/agents/memories/:id", &controllers.AgentMemoryController{}, "put:Update;delete:Delete")                                 // 长期 Memory 更新/删除
	web.Router("/agents/memories/:id/disable", &controllers.AgentMemoryController{}, "post:Disable")                                     // 禁用 Memory
	web.Router("/agents/memories/:id/enable", &controllers.AgentMemoryController{}, "post:Enable")                                       // 启用 Memory
	web.Router("/agents/memories/:id/approve", &controllers.AgentMemoryController{}, "post:Approve")                                     // 审批 candidate Memory
	web.Router("/agents/symbol-analysis/history", &controllers.AgentController{}, "get:GetSymbolAnalysisHistory")                        // 查询单币分析历史
	web.Router("/agents/alerts/status", &controllers.AgentController{}, "get:GetAlertPipelineStatus")                                    // 查询事件报警链路状态与最近追踪
	web.Router("/agents/alerts/traces", &controllers.AgentController{}, "get:ListAlertPipelineTraces")                                   // 分页查询完整事件→信号→任务→通知链路
	web.Router("/agents/scheduler/status", &controllers.AgentController{}, "get:GetSchedulerStatus")                                     // 查询 Agent Scheduler 状态
	web.Router("/agents/governance/status", &controllers.AgentController{}, "get:GetGovernanceStatus")
	web.Router("/agents/observability/summary", &controllers.AgentObservabilityController{}, "get:Summary")                   // V2-10 长期运营指标
	web.Router("/agents/observability/traces", &controllers.AgentObservabilityController{}, "get:Traces")                     // V2-10 持久化节点 Trace
	web.Router("/agents/observability/changes", &controllers.AgentObservabilityController{}, "get:Changes")                   // V2-10 MCP/Skill 变更历史                                   // 查询 Agent 权限、预算和运行指标
	web.Router("/agents/skills/implementations", &controllers.AgentSkillController{}, "get:GetImplementations")               // 可用 Skill implementation
	web.Router("/agents/skills", &controllers.AgentSkillController{}, "get:Get;post:Post")                                    // Agent Skill 配置
	web.Router("/agents/skills/:id", &controllers.AgentSkillController{}, "put:Put;delete:Delete")                            // Agent Skill 更新/删除
	web.Router("/agents/skills/import", &controllers.AgentPortableSkillController{}, "post:Import")                           // 导入标准 Agent Skill ZIP/SKILL.md
	web.Router("/agents/skills/import-directory", &controllers.AgentPortableSkillController{}, "post:ImportDirectory")        // 从受限服务器目录导入标准 Agent Skill
	web.Router("/agents/skills/:id/versions", &controllers.AgentPortableSkillController{}, "get:Versions")                    // Portable Skill 版本历史
	web.Router("/agents/skills/versions/:versionId", &controllers.AgentPortableSkillController{}, "get:VersionDetail")        // Portable Skill revision 详情
	web.Router("/agents/skills/versions/:versionId/file", &controllers.AgentPortableSkillController{}, "get:ReadVersionFile") // 浏览 revision 文本文件
	web.Router("/agents/skills/versions/:versionId/activate", &controllers.AgentPortableSkillController{}, "post:Activate")   // 激活/回滚 Portable Skill revision
	web.Router("/agents/skills/permissions/:id", &controllers.AgentPortableSkillController{}, "put:UpdatePermission")         // Portable Skill requested/granted Tool 审批
	web.Router("/agents/mcp/servers", &controllers.AgentMCPController{}, "get:ListServers;post:CreateServer")                 // 第三方 MCP Server 列表/新增
	web.Router("/agents/mcp/servers/:id", &controllers.AgentMCPController{}, "put:UpdateServer;delete:DeleteServer")          // 第三方 MCP Server 更新/删除
	web.Router("/agents/mcp/servers/:id/catalog", &controllers.AgentMCPController{}, "get:GetCatalog")                        // MCP Catalog
	web.Router("/agents/mcp/servers/:id/test", &controllers.AgentMCPController{}, "post:TestConnection")                      // MCP 连接测试
	web.Router("/agents/mcp/servers/:id/refresh", &controllers.AgentMCPController{}, "post:RefreshCatalog")                   // MCP Catalog 刷新
	web.Router("/agents/mcp/servers/:id/oauth/start", &controllers.AgentMCPController{}, "post:StartOAuth")                   // MCP OAuth 授权开始
	web.Router("/agents/mcp/oauth/client-metadata", &controllers.AgentMCPController{}, "get:OAuthClientMetadata")             // MCP OAuth Client ID Metadata Document
	web.Router("/agents/mcp/oauth/callback", &controllers.AgentMCPController{}, "get:OAuthCallback")                          // MCP OAuth callback
	web.Router("/agents/mcp/tools/:id", &controllers.AgentMCPController{}, "put:UpdateTool")                                  // MCP Tool 分类/治理
	web.Router("/agents/mcp/permissions", &controllers.AgentMCPController{}, "post:SavePermission")                           // Skill -> MCP capability 授权
	web.Router("/llm/router", &controllers.LLMRouterController{}, "get:Get;put:Put")                                          // Model Gateway Router 设置/健康
	web.Router("/llm/configs/presets", &controllers.LLMConfigController{}, "get:GetPresets")                                  // LLM Provider 预设
	web.Router("/llm/configs/test", &controllers.LLMConfigController{}, "post:Test")                                          // 测试 LLM 配置
	web.Router("/llm/configs", &controllers.LLMConfigController{}, "get:Get;post:Post")                                       // LLM 配置列表/新增
	web.Router("/llm/configs/:id/api-key", &controllers.LLMConfigController{}, "get:GetAPIKey")                               // 显式查看单个 LLM API Key
	web.Router("/llm/configs/:id", &controllers.LLMConfigController{}, "put:Put;delete:Delete")                               // LLM 配置更新/删除
	web.Router("/agents/scheduler/jobs/:name/trigger", &controllers.AgentController{}, "post:TriggerSchedulerJob")            // 手动触发 Scheduler Job
	web.Router("/notify-config", &controllers.NotifyConfigController{}, "get:Get;post:Post")                                  // 列表查询和新增
	web.Router("/notify-config/:id", &controllers.NotifyConfigController{}, "delete:Delete;put:Edit")                         // 更新和删除
	web.Router("/notifications", &controllers.NotificationController{}, "get:Get")                                            // 网页通知列表
	web.Router("/notifications/read-all", &controllers.NotificationController{}, "put:ReadAll")                               // 全部标记为已读
	web.Router("/notifications/:id/read", &controllers.NotificationController{}, "put:Read")                                  // 标记单条通知为已读
	web.Router("/ws/notifications", &controllers.NotificationWebSocketController{}, "get:Get")                                // 网页通知 WebSocket

	web.Router("/features", &controllers.FeatureController{}, "get:Get;post:Post")                            // 列表查询和新增
	web.Router("/features-options", &controllers.FeatureController{}, "get:GetOptions")                       // 列表查询
	web.Router("/features/:id", &controllers.FeatureController{}, "delete:Delete;put:Edit;get:Show")          // 更新和删除,查询
	web.Router("/features/enable/:flag", &controllers.FeatureController{}, "put:UpdateEnable")                // 修改所有的合约交易对开启关闭
	web.Router("/features/batch", &controllers.FeatureController{}, "put:BatchEdit")                          // 修改所有的合约交易
	web.Router("/features/strategy-rule/test/:id", &controllers.FeatureController{}, "post:TestStrategyRule") // 测试策略规则

	web.Router("/test-strategy-results", &controllers.TestStrategyResultController{}, "get:Get;delete:Delete")              // 测试策略的下单和平仓,按搜索条件删除
	web.Router("/test-strategy-results/:id", &controllers.TestStrategyResultController{}, "delete:Delete;get:Show")         // 删除某个测试策略结果和获取明细
	web.Router("/test-strategy-results/test/:symbol", &controllers.TestStrategyResultController{}, "post:TestStrategyRule") // 测试策略结果的某个平仓测试是否符合条件

	web.Router("/spots", &controllers.SpotController{}, "get:Get;post:Post")             // 列表查询和新增
	web.Router("/spots/:id", &controllers.SpotController{}, "delete:Delete;put:Edit")    // 更新和删除
	web.Router("/spots/enable/:flag", &controllers.SpotController{}, "put:UpdateEnable") // 修改所有的合约交易对开启关闭
	web.Router("/spots/batch", &controllers.SpotController{}, "put:BatchEdit")           // 修改所有的合约交易
	// web.Router("/spots/strategy-rule/test/:id", &controllers.SpotController{}, "post:TestStrategyRule") // 测试策略规则

	web.Router("/rush", &controllers.RushController{}, "get:Get;post:Post")             // 列表查询和新增
	web.Router("/rush/:id", &controllers.RushController{}, "delete:Delete;put:Edit")    // 更新和删除
	web.Router("/rush/enable/:flag", &controllers.RushController{}, "put:UpdateEnable") // 修改所有的交易对开启关闭

	web.Router("/notice/coin", &controllers.NoticeCoinController{}, "get:Get;post:Post")             // 列表查询和新增
	web.Router("/notice/coin/:id", &controllers.NoticeCoinController{}, "delete:Delete;put:Edit")    // 更新和删除
	web.Router("/notice/coin/enable/:flag", &controllers.NoticeCoinController{}, "put:UpdateEnable") // 修改所有的交易对开启关闭

	web.Router("/listen/coin", &controllers.ListenCoinController{}, "get:Get;post:Post")                         // 列表查询和新增
	web.Router("/listen/coin/:id", &controllers.ListenCoinController{}, "delete:Delete;put:Edit")                // 更新和删除
	web.Router("/listen/coin/kc-chart/:id", &controllers.ListenCoinController{}, "get:GetKcLineChart")           // kcChart
	web.Router("/listen/coin/enable/:flag", &controllers.ListenCoinController{}, "put:UpdateEnable")             // 修改所有的交易对开启关闭
	web.Router("/listen/funding-rates", &controllers.ListenCoinController{}, "get:GetFundingRates")              // 合约费率列表
	web.Router("/listen/funding-rates/:id", &controllers.ListenCoinController{}, "put:EditFundingRates")         // 编辑合约费率
	web.Router("/listen/funding-rate/history", &controllers.ListenCoinController{}, "get:GetFundingRateHistory") // 合约费率历史
	web.Router("/listen/strategy-rule/test/:id", &controllers.ListenCoinController{}, "post:TestStrategyRule")   // 测试策略规则

	web.Router("/orders", &controllers.OrderController{}, "get:Get;delete:DeleteAll") // order list 和 删除所有 order
	web.Router("/orders/:id", &controllers.OrderController{}, "delete:Delete")        // 删除某个订单
	web.Router("/config", &controllers.ConfigController{}, "get:Get;put:Edit")        // config get and edit

	web.Router("/futures/account", &controllers.AccountController{}, "get:GetBinanceFuturesAccount")                                              // 获取合约账户信息
	web.Router("/futures/positions", &controllers.AccountController{}, "get:GetBinanceFuturesPositions")                                          // 获取合约持仓信息
	web.Router("/futures/open-orders", &controllers.AccountController{}, "get:GetBinanceFuturesOpenOrders")                                       // 获取合约挂单信息
	web.Router("/futures/local/positions", &controllers.AccountController{}, "get:GetLocalFuturesPositions")                                      // 获取本地存储的合约持仓信息
	web.Router("/futures/local/positions/:id", &controllers.AccountController{}, "put:EditLocalFuturesPositions;delete:DelLocalFuturesPositions") // 修复和删除本地存储的合约持仓信息
	web.Router("/futures/local/open-orders", &controllers.AccountController{}, "get:GetLocalFuturesOpenOrders")                                   // 获取本地存储的挂单信息
	web.Router("/futures/liquidation-orders", &controllers.FuturesLiquidationOrderController{}, "get:Get")                                        // 合约强平订单查询

	web.Router("/fund-rate/eat", &controllers.EatRateController{}, "get:Get;post:Post")          // 列表查询和新增
	web.Router("/fund-rate/eat/:id", &controllers.EatRateController{}, "delete:Delete;put:Edit") // 更新和删除
	web.Router("/fund-rate/eat/start/:id", &controllers.EatRateController{}, "post:Start")       // start
	web.Router("/fund-rate/eat/end/:id", &controllers.EatRateController{}, "post:End")           // end

	web.Router("/strategy-templates", &controllers.StrategyTemplateController{}, "get:Get;post:Post")                                         // 策略模板
	web.Router("/strategy-templates/import", &controllers.StrategyTemplateController{}, "post:Import")                                        // 导入策略模板 JSON
	web.Router("/strategy-templates/ai-generate", &controllers.StrategyTemplateController{}, "post:StartAIGeneration")                        // 创建 AI 策略模板生成任务
	web.Router("/strategy-templates/ai-generate/:taskId/import", &controllers.StrategyTemplateController{}, "post:ImportAIGeneratedTemplate") // 导入 AI 生成的策略模板并结束对话
	web.Router("/strategy-templates/ai-generate/:taskId", &controllers.StrategyTemplateController{}, "get:GetAIGenerationTask")               // 查询 AI 策略模板生成进度
	web.Router("/strategy-templates/:id", &controllers.StrategyTemplateController{}, "delete:Delete;put:Edit")                                // 策略模板更新
	web.Router("/strategy-templates/test/:symbol", &controllers.StrategyTemplateController{}, "post:TestStrategyRule")                        // 测试策略规则

	web.Router("/start", &controllers.CommandController{}, "post:Start")   // start
	web.Router("/stop", &controllers.CommandController{}, "post:Stop")     // stop
	web.Router("/pull", &controllers.CommandController{}, "post:GitPull")  // git pull
	web.Router("/pm2-log", &controllers.CommandController{}, "get:Pm2Log") // pm2-log
}
