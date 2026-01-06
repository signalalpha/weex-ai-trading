package trader

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"time"
)

// PerformanceTracker 性能追踪器
type PerformanceTracker struct {
	metrics   PerformanceMetrics
	trades    []Trade
	mu        sync.RWMutex
	returns   []float64 // 每次交易的收益率，用于计算夏普比率
}

// NewPerformanceTracker 创建性能追踪器
func NewPerformanceTracker(initialBalance float64) *PerformanceTracker {
	now := time.Now()
	return &PerformanceTracker{
		metrics: PerformanceMetrics{
			InitialBalance:  initialBalance,
			CurrentBalance:  initialBalance,
			PeakBalance:     initialBalance,
			StartTime:       now,
			LastUpdateTime:  now,
			TradingDays:     0,
		},
		trades:  make([]Trade, 0),
		returns: make([]float64, 0),
	}
}

// RecordTrade 记录交易
func (p *PerformanceTracker) RecordTrade(trade Trade) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 添加到交易历史
	p.trades = append(p.trades, trade)

	// 更新交易计数
	p.metrics.TotalTrades++

	// 计算盈亏
	if trade.Profit > 0 {
		p.metrics.WinningTrades++
		p.metrics.TotalProfit += trade.Profit
	} else if trade.Profit < 0 {
		p.metrics.LosingTrades++
		p.metrics.TotalLoss += math.Abs(trade.Profit)
	}

	// 更新手续费
	feeAmount := parsePrice(trade.Fee)
	p.metrics.TotalFees += feeAmount

	// 更新余额
	p.metrics.CurrentBalance += trade.Profit

	// 更新峰值余额
	if p.metrics.CurrentBalance > p.metrics.PeakBalance {
		p.metrics.PeakBalance = p.metrics.CurrentBalance
	}

	// 计算收益率并添加到returns
	if p.metrics.InitialBalance > 0 {
		returnRate := trade.Profit / p.metrics.InitialBalance
		p.returns = append(p.returns, returnRate)
	}

	// 重新计算所有指标
	p.recalculateMetrics()

	// 更新时间
	p.metrics.LastUpdateTime = time.Now()
}

// recalculateMetrics 重新计算所有指标
func (p *PerformanceTracker) recalculateMetrics() {
	// 胜率
	if p.metrics.TotalTrades > 0 {
		p.metrics.WinRate = float64(p.metrics.WinningTrades) / float64(p.metrics.TotalTrades)
	}

	// 净利润
	p.metrics.NetProfit = p.metrics.TotalProfit - p.metrics.TotalLoss

	// ROI
	if p.metrics.InitialBalance > 0 {
		p.metrics.ROI = p.metrics.NetProfit / p.metrics.InitialBalance
	}

	// 最大回撤
	p.metrics.MaxDrawdown = p.calculateMaxDrawdown()

	// 当前回撤
	if p.metrics.PeakBalance > 0 {
		p.metrics.CurrentDrawdown = (p.metrics.PeakBalance - p.metrics.CurrentBalance) / p.metrics.PeakBalance
	}

	// 平均每笔交易
	if p.metrics.TotalTrades > 0 {
		p.metrics.AverageTrade = p.metrics.NetProfit / float64(p.metrics.TotalTrades)
	}

	// 平均盈利
	if p.metrics.WinningTrades > 0 {
		p.metrics.AverageWin = p.metrics.TotalProfit / float64(p.metrics.WinningTrades)
	}

	// 平均亏损
	if p.metrics.LosingTrades > 0 {
		p.metrics.AverageLoss = p.metrics.TotalLoss / float64(p.metrics.LosingTrades)
	}

	// 盈亏比
	if p.metrics.TotalLoss > 0 {
		p.metrics.ProfitFactor = p.metrics.TotalProfit / p.metrics.TotalLoss
	}

	// 夏普比率
	p.metrics.SharpeRatio = p.calculateSharpeRatio()

	// 交易天数
	p.metrics.TradingDays = int(time.Since(p.metrics.StartTime).Hours() / 24)
	if p.metrics.TradingDays < 1 {
		p.metrics.TradingDays = 1
	}

	// 日均收益率
	if p.metrics.TradingDays > 0 && p.metrics.InitialBalance > 0 {
		p.metrics.DailyReturnRate = p.metrics.NetProfit / p.metrics.InitialBalance / float64(p.metrics.TradingDays)
	}

	// 月化收益率
	p.metrics.MonthlyReturnRate = p.metrics.DailyReturnRate * 30
}

// calculateMaxDrawdown 计算最大回撤
func (p *PerformanceTracker) calculateMaxDrawdown() float64 {
	if len(p.trades) == 0 {
		return 0
	}

	maxDrawdown := 0.0
	peak := p.metrics.InitialBalance
	balance := p.metrics.InitialBalance

	for _, trade := range p.trades {
		balance += trade.Profit

		if balance > peak {
			peak = balance
		}

		if peak > 0 {
			drawdown := (peak - balance) / peak
			if drawdown > maxDrawdown {
				maxDrawdown = drawdown
			}
		}
	}

	return maxDrawdown
}

// calculateSharpeRatio 计算夏普比率
func (p *PerformanceTracker) calculateSharpeRatio() float64 {
	if len(p.returns) < 2 {
		return 0
	}

	// 计算平均收益率
	var sum float64
	for _, r := range p.returns {
		sum += r
	}
	meanReturn := sum / float64(len(p.returns))

	// 计算标准差
	var variance float64
	for _, r := range p.returns {
		variance += math.Pow(r-meanReturn, 2)
	}
	variance /= float64(len(p.returns) - 1)
	stdDev := math.Sqrt(variance)

	// 夏普比率 = (平均收益 - 无风险收益率) / 标准差
	// 假设无风险收益率为0
	if stdDev == 0 {
		return 0
	}

	// 年化夏普比率（假设每天交易一次）
	sharpe := (meanReturn / stdDev) * math.Sqrt(365)

	return sharpe
}

// GetMetrics 获取性能指标
func (p *PerformanceTracker) GetMetrics() PerformanceMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.metrics
}

// GetTrades 获取所有交易记录
func (p *PerformanceTracker) GetTrades() []Trade {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// 返回副本
	trades := make([]Trade, len(p.trades))
	copy(trades, p.trades)
	return trades
}

// GetRecentTrades 获取最近N笔交易
func (p *PerformanceTracker) GetRecentTrades(n int) []Trade {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if n <= 0 || n > len(p.trades) {
		n = len(p.trades)
	}

	trades := make([]Trade, n)
	copy(trades, p.trades[len(p.trades)-n:])
	return trades
}

// PrintSummary 打印性能摘要
func (p *PerformanceTracker) PrintSummary() {
	p.mu.RLock()
	defer p.mu.RUnlock()

	m := p.metrics

	fmt.Println("\n" + strings.Repeat("═", 60))
	fmt.Println("📊 交易性能摘要")
	fmt.Println(strings.Repeat("═", 60))

	// 基础统计
	fmt.Printf("\n【交易统计】\n")
	fmt.Printf("  总交易次数: %d\n", m.TotalTrades)
	fmt.Printf("  盈利次数: %d | 亏损次数: %d\n", m.WinningTrades, m.LosingTrades)
	fmt.Printf("  胜率: %.2f%%\n", m.WinRate*100)

	// 盈亏统计
	fmt.Printf("\n【盈亏统计】\n")
	fmt.Printf("  初始资金: %.2f USDT\n", m.InitialBalance)
	fmt.Printf("  当前资金: %.2f USDT\n", m.CurrentBalance)
	fmt.Printf("  峰值资金: %.2f USDT\n", m.PeakBalance)
	fmt.Printf("  总盈利: %.2f USDT\n", m.TotalProfit)
	fmt.Printf("  总亏损: %.2f USDT\n", m.TotalLoss)
	fmt.Printf("  净利润: %.2f USDT (%.2f%%)\n", m.NetProfit, m.ROI*100)
	fmt.Printf("  总手续费: %.2f USDT\n", m.TotalFees)

	// 风险指标
	fmt.Printf("\n【风险指标】\n")
	fmt.Printf("  最大回撤: %.2f%%\n", m.MaxDrawdown*100)
	fmt.Printf("  当前回撤: %.2f%%\n", m.CurrentDrawdown*100)
	fmt.Printf("  夏普比率: %.2f\n", m.SharpeRatio)
	fmt.Printf("  盈亏比: %.2f\n", m.ProfitFactor)

	// 交易分析
	fmt.Printf("\n【交易分析】\n")
	fmt.Printf("  平均每笔: %.2f USDT\n", m.AverageTrade)
	fmt.Printf("  平均盈利: %.2f USDT\n", m.AverageWin)
	fmt.Printf("  平均亏损: %.2f USDT\n", m.AverageLoss)

	// 收益率
	fmt.Printf("\n【收益率】\n")
	fmt.Printf("  交易天数: %d 天\n", m.TradingDays)
	fmt.Printf("  日均收益率: %.2f%%\n", m.DailyReturnRate*100)
	fmt.Printf("  月化收益率: %.2f%%\n", m.MonthlyReturnRate*100)
	fmt.Printf("  年化收益率: %.2f%%\n", m.MonthlyReturnRate*12*100)

	fmt.Println(strings.Repeat("═", 60) + "\n")
}

// PrintRecentTrades 打印最近的交易
func (p *PerformanceTracker) PrintRecentTrades(n int) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.trades) == 0 {
		fmt.Println("暂无交易记录")
		return
	}

	if n <= 0 || n > len(p.trades) {
		n = len(p.trades)
	}

	recentTrades := p.trades[len(p.trades)-n:]

	fmt.Printf("\n最近 %d 笔交易:\n", len(recentTrades))
	fmt.Println(strings.Repeat("─", 100))
	fmt.Printf("%-20s %-10s %-10s %-12s %-12s %-40s\n",
		"时间", "方向", "价格", "数量", "盈亏", "决策理由")
	fmt.Println(strings.Repeat("─", 100))

	for _, trade := range recentTrades {
		profitStr := fmt.Sprintf("%.2f (%.2f%%)", trade.Profit, trade.ProfitRate*100)
		if trade.Profit > 0 {
			profitStr = "✅ " + profitStr
		} else if trade.Profit < 0 {
			profitStr = "❌ " + profitStr
		} else {
			profitStr = "➖ " + profitStr
		}

		// 截断理由到35个字符
		reason := trade.ClaudeReason
		if len(reason) > 35 {
			reason = reason[:32] + "..."
		}

		fmt.Printf("%-20s %-10s %-10s %-12s %-12s %-40s\n",
			trade.Timestamp.Format("01-02 15:04:05"),
			trade.Side,
			trade.Price,
			trade.Amount,
			profitStr,
			reason,
		)
	}
	fmt.Println(strings.Repeat("─", 100) + "\n")
}

// UpdateBalance 更新当前余额（用于实时同步）
func (p *PerformanceTracker) UpdateBalance(newBalance float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.metrics.CurrentBalance = newBalance

	// 更新峰值
	if newBalance > p.metrics.PeakBalance {
		p.metrics.PeakBalance = newBalance
	}

	// 重新计算回撤
	if p.metrics.PeakBalance > 0 {
		p.metrics.CurrentDrawdown = (p.metrics.PeakBalance - newBalance) / p.metrics.PeakBalance
	}

	p.metrics.LastUpdateTime = time.Now()
}

// GetROI 获取当前ROI
func (p *PerformanceTracker) GetROI() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.metrics.ROI
}

// GetWinRate 获取胜率
func (p *PerformanceTracker) GetWinRate() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.metrics.WinRate
}

// GetNetProfit 获取净利润
func (p *PerformanceTracker) GetNetProfit() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.metrics.NetProfit
}
