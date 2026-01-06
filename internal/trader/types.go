package trader

import "time"

// MarketData 市场数据快照
type MarketData struct {
	Symbol        string    `json:"symbol"`
	Timestamp     time.Time `json:"timestamp"`
	Price         string    `json:"price"`
	BidPrice      string    `json:"bid_price"`
	AskPrice      string    `json:"ask_price"`
	Change24h     string    `json:"change_24h"`
	Volume24h     string    `json:"volume_24h"`
	High24h       string    `json:"high_24h"`
	Low24h        string    `json:"low_24h"`
	Candles1m     []Candle  `json:"candles_1m"`  // 最近15根1分钟K线
	Candles5m     []Candle  `json:"candles_5m"`  // 最近12根5分钟K线
	Candles15m    []Candle  `json:"candles_15m"` // 最近8根15分钟K线
	OrderBookData OrderBook `json:"order_book"`
}

// Candle K线数据
type Candle struct {
	Timestamp int64  `json:"timestamp"`
	Open      string `json:"open"`
	High      string `json:"high"`
	Low       string `json:"low"`
	Close     string `json:"close"`
	Volume    string `json:"volume"`
}

// OrderBook 订单簿数据
type OrderBook struct {
	Bids          [][]string `json:"bids"` // [[price, size], ...]
	Asks          [][]string `json:"asks"`
	BuyPressure   float64    `json:"buy_pressure"`   // 买盘压力
	SellPressure  float64    `json:"sell_pressure"`  // 卖盘压力
	PressureRatio float64    `json:"pressure_ratio"` // 买卖压力比
}

// AccountInfo 账户信息
type AccountInfo struct {
	USDTBalance    float64 `json:"usdt_balance"`
	BTCBalance     float64 `json:"btc_balance"`
	TotalValue     float64 `json:"total_value"`      // USDT计价总资产
	AvailableUSDT  float64 `json:"available_usdt"`   // 可用USDT
	AvailableBTC   float64 `json:"available_btc"`    // 可用BTC
	MaxPositionBTC float64 `json:"max_position_btc"` // 最大允许持仓
}

// Decision Claude 的交易决策
type Decision struct {
	Action     string  `json:"action"`      // "buy" | "sell" | "hold"
	Amount     float64 `json:"amount"`      // 交易数量（BTC）
	Confidence int     `json:"confidence"`  // 信心分数 0-100
	Reason     string  `json:"reason"`      // 决策理由
	StopLoss   string  `json:"stop_loss"`   // 止损价（可选）
	TakeProfit string  `json:"take_profit"` // 止盈价（可选）
}

// Trade 交易记录
type Trade struct {
	ID           string    `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	Symbol       string    `json:"symbol"`
	Side         string    `json:"side"` // "buy" | "sell"
	Price        string    `json:"price"`
	Amount       string    `json:"amount"`
	Fee          string    `json:"fee"`
	Profit       float64   `json:"profit"`        // 盈亏（USDT）
	ProfitRate   float64   `json:"profit_rate"`   // 盈亏率
	Decision     Decision  `json:"decision"`      // 对应的决策
	ExecutedAt   time.Time `json:"executed_at"`   // 执行时间
	ClosedAt     time.Time `json:"closed_at"`     // 平仓时间（如果已平仓）
	Status       string    `json:"status"`        // "open" | "closed"
	ClaudeReason string    `json:"claude_reason"` // Claude的决策理由
}

// Position 持仓信息
type Position struct {
	Symbol        string    `json:"symbol"`
	Side          string    `json:"side"` // "long" | "short"
	Amount        float64   `json:"amount"`
	EntryPrice    float64   `json:"entry_price"`
	CurrentPrice  float64   `json:"current_price"`
	UnrealizedPNL float64   `json:"unrealized_pnl"` // 未实现盈亏
	RealizedPNL   float64   `json:"realized_pnl"`   // 已实现盈亏
	OpenTime      time.Time `json:"open_time"`
}

// PerformanceMetrics 性能指标
type PerformanceMetrics struct {
	TotalTrades       int       `json:"total_trades"`
	WinningTrades     int       `json:"winning_trades"`
	LosingTrades      int       `json:"losing_trades"`
	WinRate           float64   `json:"win_rate"`
	TotalProfit       float64   `json:"total_profit"`
	TotalLoss         float64   `json:"total_loss"`
	NetProfit         float64   `json:"net_profit"`
	ROI               float64   `json:"roi"`                 // 投资回报率
	MaxDrawdown       float64   `json:"max_drawdown"`        // 最大回撤
	CurrentDrawdown   float64   `json:"current_drawdown"`    // 当前回撤
	SharpeRatio       float64   `json:"sharpe_ratio"`        // 夏普比率
	AverageTrade      float64   `json:"average_trade"`       // 平均每笔交易盈亏
	AverageWin        float64   `json:"average_win"`         // 平均盈利
	AverageLoss       float64   `json:"average_loss"`        // 平均亏损
	ProfitFactor      float64   `json:"profit_factor"`       // 盈亏比
	TotalFees         float64   `json:"total_fees"`          // 总手续费
	StartTime         time.Time `json:"start_time"`          // 开始时间
	LastUpdateTime    time.Time `json:"last_update_time"`    // 最后更新时间
	InitialBalance    float64   `json:"initial_balance"`     // 初始资金
	CurrentBalance    float64   `json:"current_balance"`     // 当前资金
	PeakBalance       float64   `json:"peak_balance"`        // 峰值资金
	TradingDays       int       `json:"trading_days"`        // 交易天数
	DailyReturnRate   float64   `json:"daily_return_rate"`   // 日均收益率
	MonthlyReturnRate float64   `json:"monthly_return_rate"` // 月化收益率
}

// RiskConfig 风控配置
type RiskConfig struct {
	MaxPositionRatio  float64 `json:"max_position_ratio"`   // 最大仓位比例 (0-1)
	MaxDrawdown       float64 `json:"max_drawdown"`         // 最大回撤限制 (0-1)
	MinConfidence     int     `json:"min_confidence"`       // 最小信心分数 (0-100)
	MaxTradesPerHour  int     `json:"max_trades_per_hour"`  // 每小时最大交易次数
	MinTradeInterval  int     `json:"min_trade_interval"`   // 最小交易间隔（秒）
	StopLossPercent   float64 `json:"stop_loss_percent"`    // 止损百分比
	TakeProfitPercent float64 `json:"take_profit_percent"`  // 止盈百分比
	EmergencyStop     bool    `json:"emergency_stop"`       // 紧急停止标志
	DailyLossLimit    float64 `json:"daily_loss_limit"`     // 单日亏损限制
	AllowShortSell    bool    `json:"allow_short_sell"`     // 是否允许做空
}

// EngineConfig 引擎配置
type EngineConfig struct {
	Symbol              string  `json:"symbol"`
	DecisionInterval    int     `json:"decision_interval"`     // 决策间隔（秒）
	MaxPosition         float64 `json:"max_position"`          // 最大持仓（BTC）
	ClaudeModel         string  `json:"claude_model"`          // Claude模型
	ClaudeAPIKey        string  `json:"claude_api_key"`        // Claude API Key
	EnableMultiTimeframe bool   `json:"enable_multi_timeframe"` // 启用多时间框架分析
	EnableOrderBook     bool    `json:"enable_order_book"`     // 启用订单簿分析
	DryRun              bool    `json:"dry_run"`               // 模拟运行（不实际下单）
	LogLevel            string  `json:"log_level"`             // 日志级别
}

// DecisionCache 决策缓存（避免频繁调用API）
type DecisionCache struct {
	LastDecision Decision
	LastPrice    float64
	LastUpdate   time.Time
	HitCount     int
}
