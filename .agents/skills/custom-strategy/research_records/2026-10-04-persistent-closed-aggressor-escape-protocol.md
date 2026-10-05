# RG5：突破前闭合主动成交持续支持（首次收益前冻结）

2026-10-04北京时间01:46:48核对实际Go/Expr结果和完整候选身份，主回测收益尚未读取。goal active。上一goal turn为progress：RG2/RG3/RG4共36完整run及所有审计已失败，旧主不重启。本轮从RG4继续一个开仓确认维度。

## 完整候选与唯一变化

目录temp_strategy/20261004-persistent-closed-aggressor-escape/：族00-persistent-closed-aggressor-escape-family.json SHA24c5d8a6122268c975e07262657a34b4c33cd5a6f92a73d24494ccb3f9dbadd0 /version714d90bc633a69cf2c743922f787e338cb52fb3640d312a6894b8d85ebb8ea74；组合01-v29c-persistent-closed-aggressor-escape.json SHA861906038fd5a3a1e34c30565837962d8121d4a0258a4907167e25680f46f07c /version5095698c9ac0b0182c447af318f1743971cdf8c5ea07eebdc1af86fb0962da0c。族4/组合6启用规则，包括双向开仓与双向平仓。

完整原9指标/1h4h1d配置、AF0基础开仓、全部RG4关闭对象保持相同。只在RG4弱趋势脱离补充开仓增加：突破之前八个闭合小时[2:10]总quote为正，各小时quote/buyQuote有效，累计buyQuote乘2严格大于总quote（LONG）或严格小于（SHORT）。这是按真实quote加权累计方向，不是八小时比例的简单平均；不包括突破小时[1]或forming[0]。原突破小时有效主动额同向占优条件、闭合ADX<20、4h EMA方向、首次8小时极值脱离、放量、实体及live不追价条件全部保留。forming价格/活动[0]有意保留，无forming taker0或hash路由、新指标/周期/系统变量、删空/币别参数或多阈值网格。

假设：RG4频率达标但跨币/年度失败，单根突破小时主动额占优未产生稳定边际；本轮检验突破前累计参与方向是否能避免一次性失真。不是从亏损静态反推可交易逆策略。已读取旧v88净主动额记录和order-flow-coherence的程序/配置，未读其收益；前者比较突破小时单根绝对净额记录、后者分钟相干性，不能宣称主动成交概念全新。373旧portable程序/1614同侧完整程序比较无精确去空白重复（族与组合重复比较计入），不是语义/alpha证明；只读候选程序，不读未观察币收益。

## 已运行矩阵与失败工具证据

34401实际terminal exit0，verification/rg5_expr_checks.go及results/20261004-rg5-expr-checks.json：3206/0。包含唯一变更的整对象断言、原基础可达、原入口边界、累计50%等值/两侧严格边界、quote加权支持/相反、首尾有效索引与负quote/负buy/buy超总额/零累计、排除forming和[10]/ratio字段影响、价格/quote等比例缩放、两侧原关闭及RG4完整ROI/ADX/结构/hash矩阵。合成静态/重复案例，非3206独立市场样本、private selector/顺序缓存/API/live/forward/盈利证明。

一次工具编排嵌套模板字符串语法失败未执行文件操作；随后研究helper说明字符串多引号导致gofmt/Go编译失败，缓存verification/rg5_expr_checks_failed_metadata_quote.go保留失败源。仅修复helper说明引号，没有改候选、容差或收益。前端首次文件路径错误已用实际src/utils/technology.ts及src/views/futures/strategyTemplate.vue核对；输入/序列化允许本轮原9配置及四类型，主动额字段实际后端存在，前端自动补全未列这些字段，不伪称自动补全完整或自行扩展UI。

| 两侧条件 | 关闭 |
| --- | --- |
| 只有ROI到+5/-5/+16/+28/-12，没有确认 | false |
| ROI在±5外、闭合4h ADX<20、当前小时价格反向破坏上根Low/High | true，RG4正常分支原样 |
| 新结构等值、ADX>=20或ROI在(-5,+5) | RG4新增分支false，原确认分支按原逻辑 |
| 原>=16趋势失败或动量+反向冲击、<=-12趋势/动量失败、>=28动量/反向冲击 | true，原样保留 |
| ROI<=-20 | true，唯一ROI-only灾难例外 |

外部profit/loss5/5只决定调用资格，不是±5%必平；AutoStopOrder=false，内区间不调用close。统一关闭不依赖开仓族/hash。没有API/live/forward关闭验证，fresh信号检查不能代替这些资格。

## 冻结撮合与所有硬门槛

AF0/RG4/RG5×BTCUSDT/ETHUSDT/SOLUSDT/XRPUSDT共12完整run，各初始1000现金；UTC2022-09-01含至2026-10-01不含，1491天/213周，每币至少192笔。报告四完整Sep–Aug年、追加2026-09、日历2023/2024/2025/2026 Jan–Sep，以及45月交易cohort归因（不是独立初始化收益）。每币>=0.9次/周、净正、四年稳定、跨币泛化全部联合门槛不放宽。开发币49月已经观察，不是未观察时间验证。AAVE/ATOM/ETC/LINK仍未观察收益、历史资格待核；开发失败不读验证收益。

原backtest_engine_v7/standard_1m，观察分钟close决策、下一分钟Open不利滑点成交；当前可用现金10%保证金、8倍、双边各fee0.0005/slip5bps、真实funding时间/率与原缺mark时已观察结算分钟Close回退、外部5/5及-20例外保持。回退不是精确交易所mark，研究账目算术通过不等于真实成本资格补齐，不补零或修改engine。每研究单列mark回退和分钟非零活动检查；这不是订单簿容量保证。

数据public-canonical-repaired-v2-funding-tail-v1，全public-archive/public-archive与verified-archive版本20261003-v2；每次cache hit重算身份，不重下修正源或修改DB。BTC/ETH/SOL/XRP SHA分别c521ace1ceff44534353fbff6179a929402a8f6f96381482670cf43a1612f457 /7c4bb4e7985cf9a0edb7b5eafdc72241aab954f2839c50d1cbf71271391b4371 /64be83c45564b68a75e344bfa32712dfec20f498f62deea32da993a6f4ef8818 /8aa4acab1e7e3fd8bec851d1e66356a715f358f5f7bc06f901c95e6000b774e3。原replay SHA21cd7e45d5d132671ebd36abd986a2a93dd81733cad3b15ba0838c4cef9012b8；独立data helper SHAa340335a5c780078e071fa2cf285662fd3205b7b530253216ab628853c2adf63；真实funding suffix54/overlap7及原manifest保留。

## 预声明审计、保护与恢复

主输出results/20261004-rg5-development4-canonical-repaired-v2-funding-tail-v1.json。主完成后复核8个AF0/RG4完整旧控制的逐笔/metrics/年度/source和全部会计，重算每币年度/日历/方向/族/集中度、现金复利quantity/fills/fee/实际funding。全部RG5补充入口按canonical闭合小时复核新增[2:10]总额和有效性、原量价/EMA/ATR/ADX，使用原9指标fresh receiver实际200input/199closed seed，不独立数学证明。全部正常关闭复用RG4原Position/gross mark-price-denominator ROI/外部gate审计，forced-end单列；不能用net/margin ROI。fresh非整段顺序缓存/private/API/live/forward/订单簿证明，分支归因非删除后反事实PnL。不得拿静态账目筛选替代完整重撮合。

三库只读39560 exit0，截止北京时间01:39:58.217/01:40:03.394/01:40:08.651，模板17/17/17、结果221/218/7；三份v29摘要同前，go_binance完整技术和策略语义完全相同。只核身份/元数据，不宣称446行完整新前向分析。conf/app.conf、engine/environment/indicator_cache、原helpers、正式技能1.0.8/trusted:false保护SHA未变，新技能行为pending/aggregate null未promote。

禁止App、DB写入/分配/启用、生产/前端/config改动和新仓库测试文件；全部完整候选/失败留temp_strategy，大输出/Go/overlay在仓库外缓存，真实overlay目标仍须不存在。启动后检查点记真实handle，超时poll同handle，不重复主。下次先阶段总结再核goal和进程；paused立即停研究不自行恢复，未达全部门槛不标complete或因困难标blocked。
