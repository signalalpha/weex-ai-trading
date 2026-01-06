# 快速开始 - WEEX AI Trading

这是一个由 Claude AI 驱动的自动交易系统，专为 WEEX AI Trading 黑客松设计。

## 🚀 5分钟快速启动

### 1. 配置环境变量

```bash
# 复制环境变量模板
cp env.example .env

# 编辑 .env 文件，填入你的凭证
vi .env
```

**必需的配置：**
```bash
# WEEX API 凭证
WEEX_API_KEY=your_api_key_here
WEEX_SECRET_KEY=your_secret_key_here
WEEX_PASSPHRASE=your_passphrase_here
WEEX_ENV=testnet  # 或 production

# Claude API Key（必须！）
CLAUDE_API_KEY=sk-ant-api03-xxxxx...
```

### 2. 构建程序

```bash
# 方式1：使用 Make（推荐）
make build

# 方式2：直接编译
go build -o bin/trader cmd/trader/main.go
```

### 3. 启动AI交易

#### 模拟模式（推荐先测试）

```bash
# 加载环境变量
source .env

# 启动模拟交易（不会实际下单）
./bin/trader run --dry-run
```

#### 实盘模式

```bash
# 确认配置正确后，启动实盘交易
./bin/trader run
```

**停止交易：** 按 `Ctrl+C` 优雅停止

---

## 📊 工作原理

```
┌──────────────┐
│  市场数据采集  │  每60秒采集一次市场数据
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ Claude AI分析 │  调用Claude API分析市场趋势
└──────┬───────┘  返回：买/卖/持有 + 信心分数 + 理由
       │
       ▼
┌──────────────┐
│   风控检查    │  10重风控：信心分数、回撤、仓位、频率...
└──────┬───────┘
       │
       ▼
┌──────────────┐
│   执行交易    │  通过WEEX API下单
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  性能追踪    │  记录盈亏、胜率、夏普比率等指标
└──────────────┘
```

---

## 🎯 核心特性

### 1. Claude AI 决策
- 使用 Claude 3.5 Sonnet 分析市场
- 综合考虑：价格趋势、成交量、买卖压力
- 每次决策都有详细理由和信心分数
- 智能缓存机制，节省API调用成本

### 2. 多层风控
- ✅ **信心分数过滤**：只有 >= 60分 才执行交易
- ✅ **最大回撤限制**：超过 15% 自动停止
- ✅ **仓位限制**：单次最多 80% 仓位
- ✅ **交易频率限制**：每小时最多 10 笔
- ✅ **最小间隔**：两次交易至少间隔 30 秒
- ✅ **止损/止盈**：3% 止损，5% 止盈
- ✅ **单日亏损限制**：单日最多亏损 10%
- ✅ **禁止做空**：仅做多，降低风险
- ✅ **紧急停止**：支持手动紧急停止
- ✅ **余额检查**：确保资金充足

### 3. 实时监控
- 📈 实时显示决策过程
- 💰 每笔交易的盈亏记录
- 📊 每10笔交易自动打印性能摘要
- 🔍 详细的日志输出

---

## 📈 监控输出示例

```
🤖 WEEX AI Trading Engine - Powered by Claude
============================================================
【交易配置】
  交易对: cmt_btcusdt
  决策间隔: 60 秒
  最大持仓: 0.010000 BTC
  Claude模型: claude-3-5-sonnet-20241022
  模拟模式: false

【风控配置】
  最大仓位比例: 80%
  最大回撤限制: 15%
  最低信心分数: 60
  交易频率限制: 10 次/小时
  止损: 3% | 止盈: 5%

【初始状态】
  初始资金: 1000.00 USDT
  开始时间: 2026-01-06 14:00:00
============================================================

正在调用 Claude API 分析市场...
📋 Claude 决策: buy | 数量: 0.000100 BTC | 信心: 75% | 理由: 突破关键阻力位，买盘压力强
✅ 风控通过: 信心分数充足
📤 提交订单: buy cmt_btcusdt 0.000100 BTC
✅ 订单成功: ID=12345, 状态=filled

...
```

---

## ⚙️ 高级配置

### 调整决策间隔

编辑 `cmd/trader/main.go:451`:
```go
DecisionInterval: 60, // 改为你想要的秒数
```

### 调整最大持仓

编辑 `cmd/trader/main.go:452`:
```go
MaxPosition: 0.01, // 改为你想要的BTC数量
```

### 调整风控参数

编辑 `internal/trader/engine.go:41-50`:
```go
riskConfig := RiskConfig{
	MaxPositionRatio:  0.8,   // 最大仓位比例
	MaxDrawdown:       0.15,  // 最大回撤
	MinConfidence:     60,    // 最低信心分数
	MaxTradesPerHour:  10,    // 每小时最多交易次数
	MinTradeInterval:  30,    // 最小交易间隔（秒）
	StopLossPercent:   0.03,  // 止损百分比
	TakeProfitPercent: 0.05,  // 止盈百分比
	DailyLossLimit:    0.10,  // 单日亏损限制
}
```

---

## 🐛 故障排查

### 问题：CLAUDE_API_KEY is required

**解决：**
```bash
# 确保设置了环境变量
export CLAUDE_API_KEY=sk-ant-api03-xxxxx...

# 或者在 .env 文件中配置后
source .env
```

### 问题：failed to get account assets

**解决：**
1. 检查 WEEX API 凭证是否正确
2. 确认 API Key 有足够的权限
3. 检查 WEEX_ENV 是否设置为 testnet 或 production

### 问题：Claude API call failed

**解决：**
1. 检查 Claude API Key 是否有效
2. 确认 API Key 有足够余额
3. 检查网络连接

### 问题：风控拒绝交易

这是**正常现象**，说明风控系统在工作。常见原因：
- 信心分数不足 (<60)
- 回撤过大
- 交易频率过高
- 仓位超限

---

## 💰 成本估算

### Claude API 调用成本
- 模型：Claude 3.5 Sonnet
- 每次决策：约 1000 tokens 输入 + 200 tokens 输出
- 成本：约 $0.01-0.02 每次决策

**每日估算：**
- 决策频率：每分钟 1 次 = 每天 1440 次
- 缓存命中率：约 50%（价格变化小时使用缓存）
- 实际 API 调用：约 720 次/天
- 每日成本：$7-15
- 比赛期间（假设7天）：$50-100

**降低成本建议：**
1. 增加决策间隔到 120 秒
2. 提高缓存阈值（修改 claude.go:cacheThreshold）
3. 只在市场波动大时频繁决策

---

## 📝 其他命令

```bash
# 查看账户信息
./bin/trader account

# 查看当前价格
./bin/trader price --symbol cmt_btcusdt

# 设置杠杆
./bin/trader leverage --symbol cmt_btcusdt --long 1 --short 1

# 手动下单
./bin/trader order --symbol cmt_btcusdt --side buy --type market --size 10

# 运行官方API测试
./bin/trader test
```

---

## 🎯 参赛建议

### 1. 先模拟测试
在实盘前，至少运行 24 小时模拟交易：
```bash
./bin/trader run --dry-run
```

### 2. 监控性能
密切关注：
- 胜率（目标 > 55%）
- 夏普比率（目标 > 1.0）
- 最大回撤（控制在 < 10%）

### 3. 调整策略
根据市场情况调整：
- 震荡市：提高信心分数阈值（如 70）
- 趋势市：降低信心分数阈值（如 55）
- 高波动：减少仓位、增加止损

### 4. 优化 Prompt
最关键的优化点是 `internal/trader/claude.go:buildPrompt`：
- 添加更多技术指标
- 优化市场描述
- 调整决策建议

---

## ⚠️ 风险提示

1. **比赛用途**：本系统专为比赛设计，请勿用于实盘大额交易
2. **API 成本**：Claude API 有调用成本，请控制预算
3. **市场风险**：加密货币波动大，存在亏损风险
4. **持续监控**：运行期间请持续监控，出现异常及时停止

---

## 📞 支持

遇到问题？
1. 查看日志文件
2. 检查 CLAUDE.md 文档
3. 提交 GitHub Issue

---

**祝你在 WEEX AI Trading 黑客松中取得好成绩！🏆**
