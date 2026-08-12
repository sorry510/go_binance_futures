# Twitter 新闻抓取功能删除审计

## 后台/API/无页面入口

| 编号 | 代码位置 | 现有作用 | 对应页面操作流程（保留原始菜单名/页面名） | 备注 |
|---|---|---|---|---|
| 1 | `crawler/twitter.go`：`StartTwitterCrawler` 及其辅助函数 | 按配置轮询 Twitter/Nitter RSS，解析内容、去重并写入 `news` 表 | 无页面入口；后台定时抓取 | Twitter 抓取专用，可删除整个文件 |
| 2 | `crawler/cleanup.go`：`StartNewsCleanupTask` 及其辅助函数 | 定时删除超过保留天数的新闻记录 | 无页面入口；后台定时清理 | 当前仅与 Twitter 抓取配套，可删除整个文件 |
| 3 | `main.go`：`registerModels` 中的 `models.News` | 注册 `news` ORM 模型 | 无页面入口；服务启动时注册模型 | 删除模型代码后同步移除注册；不执行数据库删表 |
| 4 | `main.go`：已注释的 Twitter 抓取和新闻清理启动块 | 原计划启动抓取和清理协程 | 无页面入口；当前不会执行 | 死代码，可删除 |
| 5 | `models/news.go`：`News` | 定义抓取结果对应的 ORM 模型 | 无页面入口 | 仅被 Twitter 抓取和新闻清理使用；删除代码不删除已有数据库表 |
| 6 | `conf/app.conf.example`：`[crawler]` | 提供 Twitter 抓取、RSS 镜像和新闻清理示例配置 | 无页面入口；手动编辑配置 | 整个配置段仅服务该功能，可删除；不修改 `conf/app.conf` |
| 7 | `README.md`：`检查 binance 和 x 的新闻` | 记录新闻检查关键词和账号待办 | README > TODO | 属于该功能的过期待办，可删除 |

## 前端检查

`/Users/zhz/work/binance/go_binance_futrues_new_ui` 中未发现 Twitter、Tweet、Nitter 或新闻抓取相关页面、API、菜单和多语言文案，无需修改前端。

## 待确认事项

无。现有数据库中的 `news` 表及历史数据保留，不新增删表迁移。
