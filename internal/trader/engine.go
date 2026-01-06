package trader

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	weexgo "github.com/signalalpha/weex-go"
	"github.com/signalalpha/weex-ai-trading/internal/monitor"
)

// Engine 交易引擎
type Engine struct {
	config      EngineConfig
	weexClient  *weexgo.Client
	claude      *ClaudeAnalyzer
	risk        *RiskManager
	performance *PerformanceTracker
	logger      *monitor.Logger
	position    Position
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewEngine 创建交易引擎
func NewEngine(config EngineConfig, weexClient *weexgo.Client, logger *monitor.Logger) (*Engine, error) {
	// 创建 Claude 分析器
	claude := NewClaudeAnalyzer(config.ClaudeAPIKey, config.ClaudeModel)

	// 创建风险管理器
	riskConfig := RiskConfig{
		MaxPositionRatio:  0.8,   // 最大80%仓位
		MaxDrawdown:       0.15,  // 最大15%回撤
		MinConfidence:     60,    // 最低信心分数60
		MaxTradesPerHour:  10,    // 每小时最多10笔交易
		MinTradeInterval:  30,    // 最小交易间隔30秒
		StopLossPercent:   0.03,  // 3%止损
		TakeProfitPercent: 0.05,  // 5%止盈
		EmergencyStop:     false,
		DailyLossLimit:    0.10,  // 单日最大亏损10%
		AllowShortSell:    false, // 不允许做空
	}
	risk := NewRiskManager(riskConfig)

	// 获取初始余额
	_, err := weexClient.GetAccountAssets()
	if err != nil {
		return nil, fmt.Errorf("failed to get account assets: %w", err)
	}

	initialBalance := 100.0 // 默认初始余额，实际应从API获取
	// TODO: 解析 accountAssets 获取实际余额
	// 由于不确定 WEEX SDK 的具体结构，暂时使用默认值

	// 创建性能追踪器
	performance := NewPerformanceTracker(initialBalance)

	ctx, cancel := context.WithCancel(context.Background())

	engine := &Engine{
		config:      config,
		weexClient:  weexClient,
		claude:      claude,
		risk:        risk,
		performance: performance,
		logger:      logger,
		position:    Position{Symbol: config.Symbol},
		ctx:         ctx,
		cancel:      cancel,
	}

	return engine, nil
}

// Run 启动交易引擎
func (e *Engine) Run() error {
	e.logger.Info("🚀 交易引擎启动...")
	e.logger.Infof("交易对: %s", e.config.Symbol)
	e.logger.Infof("决策间隔: %d秒", e.config.DecisionInterval)
	e.logger.Infof("模拟模式: %v", e.config.DryRun)

	// 打印初始配置
	e.printStartupInfo()

	// 启动定时决策循环
	ticker := time.NewTicker(time.Duration(e.config.DecisionInterval) * time.Second)
	defer ticker.Stop()

	// 启动每日重置任务
	e.startDailyResetTask()

	// 初始决策
	e.makeDecisionAndExecute()

	for {
		select {
		case <-ticker.C:
			e.makeDecisionAndExecute()
		case <-e.ctx.Done():
			e.logger.Info("⏸️  交易引擎停止")
			return nil
		}
	}
}

// makeDecisionAndExecute 做出决策并执行
func (e *Engine) makeDecisionAndExecute() {
	// 1. 收集市场数据
	marketData, err := e.collectMarketData()
	if err != nil {
		e.logger.Errorf("采集市场数据失败: %v", err)
		return
	}

	// 2. 获取账户信息
	account, err := e.getAccountInfo()
	if err != nil {
		e.logger.Errorf("获取账户信息失败: %v", err)
		return
	}

	// 3. 调用 Claude 分析
	e.logger.Debug("正在调用 Claude API 分析市场...")
	decision, err := e.claude.Analyze(e.ctx, marketData, account)
	if err != nil {
		e.logger.Errorf("Claude 分析失败: %v", err)
		return
	}

	e.logger.Infof("📋 Claude 决策: %s | 数量: %.6f BTC | 信心: %d%% | 理由: %s",
		decision.Action, decision.Amount, decision.Confidence, decision.Reason)

	// 4. 风控检查
	metrics := e.performance.GetMetrics()
	passed, reason := e.risk.CheckDecision(decision, account, metrics)

	if !passed {
		e.logger.Warnf("❌ 风控拒绝: %s", reason)
		return
	}

	e.logger.Infof("✅ 风控通过: %s", reason)

	// 5. 执行交易
	if decision.Action != "hold" {
		e.executeTrade(decision, marketData)
	} else {
		e.logger.Info("💤 持有观望")
	}

	// 6. 定期打印性能摘要
	if e.performance.GetMetrics().TotalTrades%10 == 0 && e.performance.GetMetrics().TotalTrades > 0 {
		e.performance.PrintSummary()
	}
}

// collectMarketData 收集市场数据
func (e *Engine) collectMarketData() (MarketData, error) {
	// 获取 ticker
	ticker, err := e.weexClient.GetTicker(e.config.Symbol)
	if err != nil {
		return MarketData{}, fmt.Errorf("failed to get ticker: %w", err)
	}

	// TODO: 获取K线数据（需要 WEEX SDK 支持）
	// 目前简化处理，只使用 ticker 数据
	// 注意：WEEX SDK 的实际结构可能不同，这里使用默认值
	data := MarketData{
		Symbol:     e.config.Symbol,
		Timestamp:  time.Now(),
		Price:      "100000",  // TODO: 从ticker获取实际价格
		BidPrice:   "99999",   // TODO: 从ticker获取
		AskPrice:   "100001",  // TODO: 从ticker获取
		Change24h:  "0",       // TODO: 从ticker获取
		Volume24h:  "1000",    // TODO: 从ticker获取
		High24h:    "100500",  // TODO: 从ticker获取
		Low24h:     "99500",   // TODO: 从ticker获取
		Candles1m:  []Candle{},  // TODO: 获取实际K线
		Candles5m:  []Candle{},  // TODO: 获取实际K线
		Candles15m: []Candle{},  // TODO: 获取实际K线
		OrderBookData: OrderBook{
			Bids:          [][]string{},
			Asks:          [][]string{},
			BuyPressure:   0,
			SellPressure:  0,
			PressureRatio: 1.0,
		},
	}

	// 如果ticker有数据，尝试使用（结构未知，先注释）
	_ = ticker

	return data, nil
}

// getAccountInfo 获取账户信息
func (e *Engine) getAccountInfo() (AccountInfo, error) {
	assets, err := e.weexClient.GetAccountAssets()
	if err != nil {
		return AccountInfo{}, err
	}

	info := AccountInfo{
		MaxPositionBTC: e.config.MaxPosition,
		USDTBalance:    100.0, // TODO: 从assets获取实际值
		BTCBalance:     0.0,
		TotalValue:     100.0,
		AvailableUSDT:  100.0,
		AvailableBTC:   0.0,
	}

	// TODO: 解析 assets 获取实际余额
	// 由于不确定 WEEX SDK 的具体结构，暂时使用默认值
	_ = assets

	return info, nil
}

// executeTrade 执行交易
func (e *Engine) executeTrade(decision Decision, marketData MarketData) {
	if e.config.DryRun {
		e.logger.Warnf("🧪 [模拟模式] %s %.6f BTC @ %s USDT", decision.Action, decision.Amount, marketData.Price)
		return
	}

	// 确定交易方向
	var side weexgo.OrderSide
	if decision.Action == "buy" {
		side = weexgo.OrderSideBuy
	} else if decision.Action == "sell" {
		side = weexgo.OrderSideSell
	} else {
		e.logger.Warnf("未知交易动作: %s", decision.Action)
		return
	}

	// 创建订单
	req := &weexgo.CreateOrderRequest{
		Symbol:    e.config.Symbol,
		Side:      side,
		OrderType: weexgo.OrderTypeMarket, // 市价单
		Quantity:  fmt.Sprintf("%.6f", decision.Amount),
	}

	e.logger.Infof("📤 提交订单: %s %s %.6f BTC", side, e.config.Symbol, decision.Amount)

	_, err := e.weexClient.CreateOrder(req)
	if err != nil {
		e.logger.Errorf("❌ 下单失败: %v", err)
		return
	}

	// TODO: 订单ID和状态的实际字段名未知，使用默认值
	e.logger.Infof("✅ 订单成功")

	// 记录交易
	trade := Trade{
		ID:           "unknown", // TODO: 从order获取实际ID
		Timestamp:    time.Now(),
		Symbol:       e.config.Symbol,
		Side:         string(side),
		Price:        marketData.Price,
		Amount:       req.Quantity,
		Fee:          "0", // TODO: 从订单响应获取
		Profit:       0,   // TODO: 计算实际盈亏
		Decision:     decision,
		ExecutedAt:   time.Now(),
		Status:       "open",
		ClaudeReason: decision.Reason,
	}

	// 记录到性能追踪器
	e.performance.RecordTrade(trade)

	// 记录到风险管理器
	e.risk.RecordTrade(trade)

	// 更新持仓
	e.updatePosition(trade, marketData)
}

// updatePosition 更新持仓信息
func (e *Engine) updatePosition(trade Trade, marketData MarketData) {
	currentPrice, _ := strconv.ParseFloat(marketData.Price, 64)
	tradeAmount, _ := strconv.ParseFloat(trade.Amount, 64)

	if trade.Side == "buy" {
		// 买入：增加持仓
		if e.position.Amount == 0 {
			// 开新仓
			e.position.Side = "long"
			e.position.EntryPrice = currentPrice
			e.position.Amount = tradeAmount
			e.position.OpenTime = time.Now()
		} else {
			// 加仓
			totalCost := e.position.EntryPrice*e.position.Amount + currentPrice*tradeAmount
			e.position.Amount += tradeAmount
			e.position.EntryPrice = totalCost / e.position.Amount
		}
	} else if trade.Side == "sell" {
		// 卖出：减少持仓
		if e.position.Amount >= tradeAmount {
			// 计算已实现盈亏
			realizedPNL := (currentPrice - e.position.EntryPrice) * tradeAmount
			e.position.RealizedPNL += realizedPNL

			e.position.Amount -= tradeAmount

			if e.position.Amount == 0 {
				// 完全平仓
				e.logger.Infof("💰 平仓完成，已实现盈亏: %.2f USDT", e.position.RealizedPNL)
				e.position = Position{Symbol: e.config.Symbol} // 重置
			}
		}
	}

	// 更新当前价格和未实现盈亏
	e.position.CurrentPrice = currentPrice
	if e.position.Amount > 0 {
		e.position.UnrealizedPNL = (currentPrice - e.position.EntryPrice) * e.position.Amount
		e.logger.Infof("📊 当前持仓: %.6f BTC @ %.2f, 未实现盈亏: %.2f USDT",
			e.position.Amount, e.position.EntryPrice, e.position.UnrealizedPNL)
	}
}

// Stop 停止交易引擎
func (e *Engine) Stop() {
	e.logger.Info("正在停止交易引擎...")
	e.cancel()

	// 打印最终性能摘要
	e.performance.PrintSummary()
	e.performance.PrintRecentTrades(20)
}

// GetStatus 获取引擎状态
func (e *Engine) GetStatus() map[string]interface{} {
	metrics := e.performance.GetMetrics()
	riskStats := e.risk.GetTradeStats()
	cacheHits, lastUpdate := e.claude.GetCacheStats()

	return map[string]interface{}{
		"running":         true,
		"symbol":          e.config.Symbol,
		"dry_run":         e.config.DryRun,
		"total_trades":    metrics.TotalTrades,
		"win_rate":        fmt.Sprintf("%.2f%%", metrics.WinRate*100),
		"net_profit":      fmt.Sprintf("%.2f USDT", metrics.NetProfit),
		"roi":             fmt.Sprintf("%.2f%%", metrics.ROI*100),
		"current_balance": fmt.Sprintf("%.2f USDT", metrics.CurrentBalance),
		"max_drawdown":    fmt.Sprintf("%.2f%%", metrics.MaxDrawdown*100),
		"current_drawdown": fmt.Sprintf("%.2f%%", metrics.CurrentDrawdown*100),
		"position":        e.position,
		"risk_stats":      riskStats,
		"claude_cache_hits": cacheHits,
		"claude_last_update": lastUpdate.Format("15:04:05"),
	}
}

// printStartupInfo 打印启动信息
func (e *Engine) printStartupInfo() {
	metrics := e.performance.GetMetrics()

	fmt.Println("\n" + strings.Repeat("═", 80))
	fmt.Println("🤖 WEEX AI Trading Engine - Powered by Claude")
	fmt.Println(strings.Repeat("═", 80))

	fmt.Printf("\n【交易配置】\n")
	fmt.Printf("  交易对: %s\n", e.config.Symbol)
	fmt.Printf("  决策间隔: %d 秒\n", e.config.DecisionInterval)
	fmt.Printf("  最大持仓: %.6f BTC\n", e.config.MaxPosition)
	fmt.Printf("  Claude模型: %s\n", e.config.ClaudeModel)
	fmt.Printf("  模拟模式: %v\n", e.config.DryRun)

	fmt.Printf("\n【风控配置】\n")
	riskConfig := e.risk.GetConfig()
	fmt.Printf("  最大仓位比例: %.0f%%\n", riskConfig.MaxPositionRatio*100)
	fmt.Printf("  最大回撤限制: %.0f%%\n", riskConfig.MaxDrawdown*100)
	fmt.Printf("  最低信心分数: %d\n", riskConfig.MinConfidence)
	fmt.Printf("  交易频率限制: %d 次/小时\n", riskConfig.MaxTradesPerHour)
	fmt.Printf("  止损: %.0f%% | 止盈: %.0f%%\n", riskConfig.StopLossPercent*100, riskConfig.TakeProfitPercent*100)

	fmt.Printf("\n【初始状态】\n")
	fmt.Printf("  初始资金: %.2f USDT\n", metrics.InitialBalance)
	fmt.Printf("  开始时间: %s\n", metrics.StartTime.Format("2006-01-02 15:04:05"))

	fmt.Println(strings.Repeat("═", 80) + "\n")
}

// startDailyResetTask 启动每日重置任务
func (e *Engine) startDailyResetTask() {
	go func() {
		for {
			now := time.Now()
			// 计算到明天0点的时间
			tomorrow := now.AddDate(0, 0, 1)
			midnight := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, tomorrow.Location())
			duration := midnight.Sub(now)

			select {
			case <-time.After(duration):
				e.logger.Info("🔄 执行每日统计重置")
				e.risk.ResetDailyStats()
			case <-e.ctx.Done():
				return
			}
		}
	}()
}
