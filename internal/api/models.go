package api

// AccountBalance represents account balance information
// Response format from /capi/v2/account/assets:
//
//	[
//	  {
//	    "coinName": "USDT",
//	    "available": "5413.06877369",
//	    "equity": "5696.49288823",
//	    "frozen": "81.28240000",
//	    "unrealizePnl": "-34.55300000"
//	  }
//	]
type AccountBalance struct {
	CoinName     string `json:"coinName"`
	Available    string `json:"available"`
	Equity       string `json:"equity"`
	Frozen       string `json:"frozen"`
	UnrealizePnl string `json:"unrealizePnl"`
}

// AccountInfo represents account information
// It's a slice of AccountBalance (direct array response from /capi/v2/account/assets)
type AccountInfo []AccountBalance

// Ticker represents market ticker data
type Ticker struct {
	Symbol    string `json:"symbol"`
	LastPrice string `json:"lastPrice"`
	High24h   string `json:"high24h"`
	Low24h    string `json:"low24h"`
	Volume24h string `json:"volume24h"`
	Change24h string `json:"change24h"`
}

// OrderSide represents order side (buy or sell)
type OrderSide string

const (
	OrderSideBuy  OrderSide = "buy"
	OrderSideSell OrderSide = "sell"
)

// OrderType represents order type
type OrderType string

const (
	OrderTypeMarket OrderType = "market"
	OrderTypeLimit  OrderType = "limit"
)

// CreateOrderRequest represents a request to create an order
type CreateOrderRequest struct {
	Symbol    string    `json:"symbol"`
	Side      OrderSide `json:"side"`
	OrderType OrderType `json:"orderType"`
	Quantity  string    `json:"quantity"`
	Price     string    `json:"price,omitempty"` // Required for limit orders
}

// Order represents an order
type Order struct {
	OrderID    string    `json:"orderId"`
	Symbol     string    `json:"symbol"`
	Side       OrderSide `json:"side"`
	OrderType  OrderType `json:"orderType"`
	Quantity   string    `json:"quantity"`
	Price      string    `json:"price"`
	Status     string    `json:"status"`
	CreateTime int64     `json:"createTime"`
}

// APIResponse represents a generic API response
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
