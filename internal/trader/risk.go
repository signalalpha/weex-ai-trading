package trader

import (
	"fmt"
	"sync"
	"time"
)

// RiskManager 风险管理器
type RiskManager struct {
	config         RiskConfig
	mu             sync.RWMutex
	tradeHistory   []time.Time    // 交易时间历史
	dailyPNL       map[string]float64 // 每日盈亏记录，key为日期
	lastTradeTime  time.Time
	emergencyStop  bool
}

// NewRiskManager 创建风险管理器
func NewRiskManager(config RiskConfig) *RiskManager {
	return &RiskManager{
		config:        config,
		tradeHistory:  make([]time.Time, 0),
		dailyPNL:      make(map[string]float64),
		lastTradeTime: time.Time{},
		emergencyStop: false,
	}
}

// CheckDecision 检查决策是否通过风控
func (r *RiskManager) CheckDecision(decision Decision, account AccountInfo, metrics PerformanceMetrics) (bool, string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 1. 检查紧急停止
	if r.emergencyStop || r.config.EmergencyStop {
		return false, "系统处于紧急停止状态"
	}

	// 2. 检查 hold 动作（总是通过）
	if decision.Action == "hold" {
		return true, "hold操作无需风控检查"
	}

	// 3. 检查信心分数
	if decision.Confidence < r.config.MinConfidence {
		return false, fmt.Sprintf("信心分数不足 (需要>=%d, 当前%d)", r.config.MinConfidence, decision.Confidence)
	}

	// 4. 检查最大回撤
	if metrics.CurrentDrawdown > r.config.MaxDrawdown {
		return false, fmt.Sprintf("当前回撤超过限制 (%.2f%% > %.2f%%)", metrics.CurrentDrawdown*100, r.config.MaxDrawdown*100)
	}

	// 5. 检查交易频率
	if !r.checkTradeFrequency() {
		return false, fmt.Sprintf("交易频率超限 (每小时最多%d次)", r.config.MaxTradesPerHour)
	}

	// 6. 检查最小交易间隔
	if !r.checkTradeInterval() {
		return false, fmt.Sprintf("交易间隔太短 (最小间隔%d秒)", r.config.MinTradeInterval)
	}

	// 7. 检查单日亏损限制
	if !r.checkDailyLoss(metrics) {
		return false, fmt.Sprintf("单日亏损超限 (%.2f%% > %.2f%%)", r.getDailyLossPercent(metrics)*100, r.config.DailyLossLimit*100)
	}

	// 8. 检查仓位限制（买入时）
	if decision.Action == "buy" {
		if !r.checkPositionLimit(decision, account) {
			maxAmount := account.MaxPositionBTC * r.config.MaxPositionRatio
			return false, fmt.Sprintf("仓位超限 (目标%.6f BTC > 最大%.6f BTC)", decision.Amount, maxAmount)
		}
	}

	// 9. 检查做空权限
	if decision.Action == "sell" && !r.config.AllowShortSell {
		if account.BTCBalance < decision.Amount {
			return false, "不允许做空，且当前BTC余额不足"
		}
	}

	// 10. 检查账户余额
	if decision.Action == "buy" {
		// 简单估算需要的USDT（这里需要当前价格，实际应该传入）
		// 暂时简化处理
		if account.AvailableUSDT < 10 {
			return false, "USDT余额不足"
		}
	}

	return true, "通过风控检查"
}

// checkTradeFrequency 检查交易频率
func (r *RiskManager) checkTradeFrequency() bool {
	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)

	// 清理1小时前的记录
	validTrades := make([]time.Time, 0)
	for _, t := range r.tradeHistory {
		if t.After(oneHourAgo) {
			validTrades = append(validTrades, t)
		}
	}
	r.tradeHistory = validTrades

	// 检查是否超过限制
	return len(r.tradeHistory) < r.config.MaxTradesPerHour
}

// checkTradeInterval 检查交易间隔
func (r *RiskManager) checkTradeInterval() bool {
	if r.lastTradeTime.IsZero() {
		return true
	}

	elapsed := time.Since(r.lastTradeTime)
	return elapsed.Seconds() >= float64(r.config.MinTradeInterval)
}

// checkDailyLoss 检查单日亏损
func (r *RiskManager) checkDailyLoss(metrics PerformanceMetrics) bool {
	today := time.Now().Format("2006-01-02")
	dailyLoss := r.dailyPNL[today]

	if dailyLoss >= 0 {
		return true // 今日盈利，无需检查
	}

	// 计算亏损比例
	lossPercent := -dailyLoss / metrics.InitialBalance

	return lossPercent <= r.config.DailyLossLimit
}

// checkPositionLimit 检查仓位限制
func (r *RiskManager) checkPositionLimit(decision Decision, account AccountInfo) bool {
	maxAllowed := account.MaxPositionBTC * r.config.MaxPositionRatio
	targetPosition := account.BTCBalance + decision.Amount

	return targetPosition <= maxAllowed
}

// RecordTrade 记录交易
func (r *RiskManager) RecordTrade(trade Trade) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	r.tradeHistory = append(r.tradeHistory, now)
	r.lastTradeTime = now

	// 更新每日盈亏
	today := now.Format("2006-01-02")
	r.dailyPNL[today] += trade.Profit
}

// getDailyLossPercent 获取当日亏损百分比
func (r *RiskManager) getDailyLossPercent(metrics PerformanceMetrics) float64 {
	today := time.Now().Format("2006-01-02")
	dailyPNL := r.dailyPNL[today]

	if dailyPNL >= 0 {
		return 0
	}

	return -dailyPNL / metrics.InitialBalance
}

// TriggerEmergencyStop 触发紧急停止
func (r *RiskManager) TriggerEmergencyStop(reason string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.emergencyStop = true
	r.config.EmergencyStop = true

	// TODO: 发送告警通知
	fmt.Printf("⚠️ 紧急停止触发: %s\n", reason)
}

// ResumeTrading 恢复交易
func (r *RiskManager) ResumeTrading() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.emergencyStop = false
	r.config.EmergencyStop = false

	fmt.Println("✅ 交易已恢复")
}

// IsEmergencyStopped 是否处于紧急停止状态
func (r *RiskManager) IsEmergencyStopped() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.emergencyStop
}

// GetTradeStats 获取交易统计
func (r *RiskManager) GetTradeStats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)

	// 统计过去1小时的交易次数
	recentTrades := 0
	for _, t := range r.tradeHistory {
		if t.After(oneHourAgo) {
			recentTrades++
		}
	}

	// 今日盈亏
	today := now.Format("2006-01-02")
	todayPNL := r.dailyPNL[today]

	// 距离上次交易的时间
	var lastTradeElapsed string
	if r.lastTradeTime.IsZero() {
		lastTradeElapsed = "从未交易"
	} else {
		lastTradeElapsed = time.Since(r.lastTradeTime).Round(time.Second).String()
	}

	return map[string]interface{}{
		"recent_trades_1h":    recentTrades,
		"max_trades_per_hour": r.config.MaxTradesPerHour,
		"today_pnl":           todayPNL,
		"last_trade_elapsed":  lastTradeElapsed,
		"emergency_stop":      r.emergencyStop,
		"total_trades":        len(r.tradeHistory),
	}
}

// ResetDailyStats 重置每日统计（每天0点调用）
func (r *RiskManager) ResetDailyStats() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 保留最近7天的数据
	sevenDaysAgo := time.Now().AddDate(0, 0, -7).Format("2006-01-02")

	newDailyPNL := make(map[string]float64)
	for date, pnl := range r.dailyPNL {
		if date >= sevenDaysAgo {
			newDailyPNL[date] = pnl
		}
	}

	r.dailyPNL = newDailyPNL
}

// UpdateConfig 更新风控配置
func (r *RiskManager) UpdateConfig(config RiskConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.config = config
}

// GetConfig 获取当前配置
func (r *RiskManager) GetConfig() RiskConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.config
}

// ShouldStopLoss 判断是否应该止损
func (r *RiskManager) ShouldStopLoss(position Position) bool {
	if position.Amount == 0 {
		return false
	}

	// 计算当前亏损百分比
	lossPercent := (position.CurrentPrice - position.EntryPrice) / position.EntryPrice

	if position.Side == "long" {
		// 做多：价格下跌超过止损线
		return lossPercent < -r.config.StopLossPercent
	} else {
		// 做空：价格上涨超过止损线
		return lossPercent > r.config.StopLossPercent
	}
}

// ShouldTakeProfit 判断是否应该止盈
func (r *RiskManager) ShouldTakeProfit(position Position) bool {
	if position.Amount == 0 {
		return false
	}

	// 计算当前盈利百分比
	profitPercent := (position.CurrentPrice - position.EntryPrice) / position.EntryPrice

	if position.Side == "long" {
		// 做多：价格上涨超过止盈线
		return profitPercent > r.config.TakeProfitPercent
	} else {
		// 做空：价格下跌超过止盈线
		return profitPercent < -r.config.TakeProfitPercent
	}
}
