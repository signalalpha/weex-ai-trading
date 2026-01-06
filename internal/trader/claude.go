package trader

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// ClaudeAnalyzer Claude 分析器
type ClaudeAnalyzer struct {
	client         *anthropic.Client
	model          string
	cache          *DecisionCache
	enableCache    bool
	cacheThreshold float64 // 价格变化阈值，低于此值使用缓存
}

// NewClaudeAnalyzer 创建 Claude 分析器
func NewClaudeAnalyzer(apiKey, model string) *ClaudeAnalyzer {
	client := anthropic.NewClient(
		option.WithAPIKey(apiKey),
	)

	if model == "" {
		model = "claude-3-5-sonnet-20241022"
	}

	return &ClaudeAnalyzer{
		client:         client,
		model:          model,
		enableCache:    true,
		cacheThreshold: 0.001, // 0.1% 价格变化阈值
		cache: &DecisionCache{
			LastUpdate: time.Now().Add(-1 * time.Hour), // 初始化为1小时前
		},
	}
}

// Analyze 分析市场数据并给出交易决策
func (c *ClaudeAnalyzer) Analyze(ctx context.Context, marketData MarketData, account AccountInfo) (Decision, error) {
	// 检查缓存
	if c.shouldUseCache(marketData) {
		c.cache.HitCount++
		return c.cache.LastDecision, nil
	}

	// 构建 prompt
	prompt := c.buildPrompt(marketData, account)

	// 调用 Claude API
	resp, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.F(anthropic.Model(c.model)),
		MaxTokens: anthropic.F(int64(2048)),
		Messages: anthropic.F([]anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		}),
	})

	if err != nil {
		return Decision{}, fmt.Errorf("Claude API call failed: %w", err)
	}

	// 解析响应
	decision, err := c.parseResponse(resp)
	if err != nil {
		return Decision{}, fmt.Errorf("failed to parse Claude response: %w", err)
	}

	// 更新缓存
	c.updateCache(decision, marketData)

	return decision, nil
}

// buildPrompt 构建 Claude prompt
func (c *ClaudeAnalyzer) buildPrompt(data MarketData, account AccountInfo) string {
	// 构建K线数据摘要
	candles1mSummary := c.summarizeCandles(data.Candles1m, "1分钟")
	candles5mSummary := c.summarizeCandles(data.Candles5m, "5分钟")
	candles15mSummary := c.summarizeCandles(data.Candles15m, "15分钟")

	// 构建订单簿分析
	orderBookAnalysis := fmt.Sprintf(`
订单簿深度分析:
- 买盘压力: %.2f (前10档买单总量)
- 卖盘压力: %.2f (前10档卖单总量)
- 买卖压力比: %.2f (>1.2看涨, <0.8看跌)`,
		data.OrderBookData.BuyPressure,
		data.OrderBookData.SellPressure,
		data.OrderBookData.PressureRatio,
	)

	prompt := "你是一个专业的加密货币量化交易员，专注于 BTC/USDT 短线交易。你的目标是通过精准的市场分析实现稳定盈利。\n\n" +
		"## 当前市场概况\n" +
		fmt.Sprintf("交易对: %s\n", data.Symbol) +
		fmt.Sprintf("当前时间: %s\n", data.Timestamp.Format("2006-01-02 15:04:05")) +
		fmt.Sprintf("当前价格: %s USDT\n", data.Price) +
		fmt.Sprintf("买一价: %s | 卖一价: %s\n", data.BidPrice, data.AskPrice) +
		fmt.Sprintf("24h涨跌幅: %s%%\n", data.Change24h) +
		fmt.Sprintf("24h最高: %s | 最低: %s\n", data.High24h, data.Low24h) +
		fmt.Sprintf("24h成交量: %s\n\n", data.Volume24h) +
		orderBookAnalysis + "\n\n" +
		"## 多时间框架K线分析\n" +
		candles1mSummary + "\n\n" +
		candles5mSummary + "\n\n" +
		candles15mSummary + "\n\n" +
		"## 当前持仓状况\n" +
		fmt.Sprintf("USDT余额: %.2f\n", account.USDTBalance) +
		fmt.Sprintf("BTC余额: %.6f\n", account.BTCBalance) +
		fmt.Sprintf("总资产价值: %.2f USDT\n", account.TotalValue) +
		fmt.Sprintf("可用USDT: %.2f\n", account.AvailableUSDT) +
		fmt.Sprintf("可用BTC: %.6f\n", account.AvailableBTC) +
		fmt.Sprintf("最大允许持仓: %.6f BTC\n\n", account.MaxPositionBTC) +
		"## 你的任务\n" +
		"基于以上市场数据和持仓情况，给出下一步交易建议。请综合考虑：\n\n" +
		"1. **趋势判断**: 分析多时间框架K线，判断短期趋势（看涨/看跌/震荡）\n" +
		"2. **买卖压力**: 根据订单簿数据，评估市场买卖力量对比\n" +
		"3. **风险控制**:\n" +
		"   - 单次交易不要超过总资产的20%\n" +
		"   - 如果当前有持仓，考虑是否需要止盈或止损\n" +
		"   - 避免在震荡市频繁交易\n" +
		"4. **信心评估**: 只有在有较高把握时才建议交易（信心分数>=60）\n\n" +
		"## 交易策略建议\n" +
		"- **买入时机**: 上升趋势确立 + 买盘压力强 + 突破关键阻力位\n" +
		"- **卖出时机**: 下降趋势确立 + 卖盘压力强 + 跌破关键支撑位\n" +
		"- **持有时机**: 趋势不明朗 + 信心不足 + 市场震荡\n\n" +
		"## 返回格式（必须是有效JSON，不要有任何额外文字）\n" +
		"{\n" +
		"  \"action\": \"buy\" | \"sell\" | \"hold\",\n" +
		"  \"amount\": 交易数量（BTC，最多6位小数），\n" +
		"  \"confidence\": 0-100的信心分数（只有>=60才建议实际交易），\n" +
		"  \"reason\": \"一句话说明核心理由（30字以内）\",\n" +
		"  \"stop_loss\": \"止损价格（可选，字符串）\",\n" +
		"  \"take_profit\": \"止盈价格（可选，字符串）\"\n" +
		"}\n\n" +
		"**重要提示**:\n" +
		"1. 请直接返回JSON，不要有任何markdown格式或其他文字\n" +
		"2. amount必须考虑账户余额，不要超过可用资金\n" +
		"3. 如果action是\"hold\"，amount设为0\n" +
		"4. 信心分数要真实反映市场确定性，不要虚高\n\n" +
		"现在请给出你的交易建议:"

	return prompt
}

// summarizeCandles 汇总K线数据
func (c *ClaudeAnalyzer) summarizeCandles(candles []Candle, timeframe string) string {
	if len(candles) == 0 {
		return fmt.Sprintf("### %sK线: 暂无数据", timeframe)
	}

	// 计算涨跌
	first := candles[0]
	last := candles[len(candles)-1]

	return fmt.Sprintf(`### %sK线 (最近%d根)
开盘: %s -> 收盘: %s
最高: %s | 最低: %s
总成交量: %s
趋势: %s`,
		timeframe,
		len(candles),
		first.Open,
		last.Close,
		c.findHighest(candles),
		c.findLowest(candles),
		c.sumVolume(candles),
		c.detectTrend(candles),
	)
}

// findHighest 找最高价
func (c *ClaudeAnalyzer) findHighest(candles []Candle) string {
	if len(candles) == 0 {
		return "N/A"
	}
	highest := candles[0].High
	for _, candle := range candles {
		if candle.High > highest {
			highest = candle.High
		}
	}
	return highest
}

// findLowest 找最低价
func (c *ClaudeAnalyzer) findLowest(candles []Candle) string {
	if len(candles) == 0 {
		return "N/A"
	}
	lowest := candles[0].Low
	for _, candle := range candles {
		if candle.Low < lowest {
			lowest = candle.Low
		}
	}
	return lowest
}

// sumVolume 计算总成交量
func (c *ClaudeAnalyzer) sumVolume(candles []Candle) string {
	// 简化处理，返回第一根K线的成交量
	if len(candles) == 0 {
		return "N/A"
	}
	return candles[0].Volume
}

// detectTrend 简单趋势检测
func (c *ClaudeAnalyzer) detectTrend(candles []Candle) string {
	if len(candles) < 3 {
		return "数据不足"
	}

	first := candles[0]
	last := candles[len(candles)-1]

	// 简单比较首尾价格
	if last.Close > first.Open {
		return "上涨 ↗"
	} else if last.Close < first.Open {
		return "下跌 ↘"
	}
	return "震荡 ↔"
}

// parseResponse 解析 Claude 响应
func (c *ClaudeAnalyzer) parseResponse(resp *anthropic.Message) (Decision, error) {
	if len(resp.Content) == 0 {
		return Decision{}, fmt.Errorf("empty response from Claude")
	}

	// 获取文本内容
	textBlock, ok := resp.Content[0].AsUnion().(anthropic.TextBlock)
	if !ok {
		return Decision{}, fmt.Errorf("unexpected response type from Claude")
	}

	responseText := textBlock.Text

	// 清理可能的 markdown 格式
	responseText = strings.TrimSpace(responseText)
	responseText = strings.TrimPrefix(responseText, "```json")
	responseText = strings.TrimPrefix(responseText, "```")
	responseText = strings.TrimSuffix(responseText, "```")
	responseText = strings.TrimSpace(responseText)

	// 尝试找到第一个 { 和最后一个 }
	startIdx := strings.Index(responseText, "{")
	endIdx := strings.LastIndex(responseText, "}")

	if startIdx == -1 || endIdx == -1 || startIdx >= endIdx {
		return Decision{}, fmt.Errorf("no valid JSON found in response: %s", responseText)
	}

	jsonStr := responseText[startIdx : endIdx+1]

	// 解析 JSON
	var decision Decision
	if err := json.Unmarshal([]byte(jsonStr), &decision); err != nil {
		return Decision{}, fmt.Errorf("failed to unmarshal JSON: %w, response: %s", err, jsonStr)
	}

	// 验证决策
	if err := c.validateDecision(&decision); err != nil {
		return Decision{}, fmt.Errorf("invalid decision: %w", err)
	}

	return decision, nil
}

// validateDecision 验证决策的合法性
func (c *ClaudeAnalyzer) validateDecision(decision *Decision) error {
	// 验证 action
	action := strings.ToLower(decision.Action)
	if action != "buy" && action != "sell" && action != "hold" {
		return fmt.Errorf("invalid action: %s (must be buy/sell/hold)", decision.Action)
	}
	decision.Action = action

	// 验证 confidence
	if decision.Confidence < 0 || decision.Confidence > 100 {
		return fmt.Errorf("invalid confidence: %d (must be 0-100)", decision.Confidence)
	}

	// 验证 amount
	if decision.Amount < 0 {
		return fmt.Errorf("invalid amount: %f (must be >= 0)", decision.Amount)
	}

	// hold 动作必须 amount = 0
	if action == "hold" && decision.Amount != 0 {
		decision.Amount = 0
	}

	return nil
}

// shouldUseCache 判断是否应该使用缓存
func (c *ClaudeAnalyzer) shouldUseCache(data MarketData) bool {
	if !c.enableCache {
		return false
	}

	// 如果缓存太旧（超过2分钟），不使用
	if time.Since(c.cache.LastUpdate) > 2*time.Minute {
		return false
	}

	// 如果上次决策是 hold，且时间不超过1分钟，直接使用缓存
	if c.cache.LastDecision.Action == "hold" && time.Since(c.cache.LastUpdate) < 1*time.Minute {
		return true
	}

	// 计算价格变化
	if c.cache.LastPrice == 0 {
		return false
	}

	currentPrice := parsePrice(data.Price)
	priceChange := abs(currentPrice-c.cache.LastPrice) / c.cache.LastPrice

	// 如果价格变化小于阈值，使用缓存
	return priceChange < c.cacheThreshold
}

// updateCache 更新缓存
func (c *ClaudeAnalyzer) updateCache(decision Decision, data MarketData) {
	c.cache.LastDecision = decision
	c.cache.LastPrice = parsePrice(data.Price)
	c.cache.LastUpdate = time.Now()
}

// GetCacheStats 获取缓存统计
func (c *ClaudeAnalyzer) GetCacheStats() (hitCount int, lastUpdate time.Time) {
	return c.cache.HitCount, c.cache.LastUpdate
}

// parsePrice 解析价格字符串为 float64
func parsePrice(priceStr string) float64 {
	var price float64
	fmt.Sscanf(priceStr, "%f", &price)
	return price
}

// abs 绝对值
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
