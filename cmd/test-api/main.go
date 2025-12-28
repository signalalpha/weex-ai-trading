package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/signalalpha/weex-ai-trading/internal/api"
	"github.com/signalalpha/weex-ai-trading/internal/config"
	"github.com/signalalpha/weex-ai-trading/internal/monitor"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	logger := monitor.NewLogger(cfg.Log.Level, cfg.Log.Output)
	logger.Info("Starting WEEX API Test...")

	// Create API client
	client, err := api.NewClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create API client: %v", err)
	}

	// Test 0: Try public endpoint first (no auth required)
	logger.Info("Test 0: Testing public endpoint (server time)...")
	testURL := "https://api-contract.weex.com/capi/v2/market/time"
	httpClient := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", testURL, nil)
	req.Header.Set("User-Agent", "weex-ai-trading/1.0")
	req.Header.Set("locale", "zh-CN")

	resp, err := httpClient.Do(req)
	if err != nil {
		logger.Errorf("Failed to access public endpoint: %v", err)
	} else {
		defer resp.Body.Close()
		bodyBytes, _ := io.ReadAll(resp.Body)
		logger.Infof("Public endpoint response: Status=%d, Body=%s", resp.StatusCode, string(bodyBytes))
		if resp.StatusCode == 200 {
			fmt.Printf("\n✅ Public endpoint accessible\n")
		} else {
			fmt.Printf("\n⚠️  Public endpoint returned status %d: %s\n", resp.StatusCode, string(bodyBytes))
		}
	}

	// Test 1: Get account info
	logger.Info("\nTest 1: Getting account information...")
	accountInfo, err := client.GetAccountInfo()
	if err != nil {
		logger.Errorf("Failed to get account info: %v", err)
		fmt.Printf("\n❌ Account info failed: %v\n", err)
		fmt.Printf("\n💡 Troubleshooting tips:\n")
		fmt.Printf("  1. Check if your IP is whitelisted in WEEX API settings\n")
		fmt.Printf("  2. Verify API Key, Secret Key, and Passphrase are correct\n")
		fmt.Printf("  3. Check if you're using the correct API endpoint\n")
		fmt.Printf("  4. For 521 errors, it usually means IP whitelist issue\n")
	} else {
		logger.Infof("Account info: %+v", accountInfo)
		fmt.Printf("\n=== Account Information ===\n")
		fmt.Printf("Total Equity: %s\n", accountInfo.TotalEquity)
		if len(accountInfo.Balance) > 0 {
			fmt.Printf("Balances:\n")
			for _, bal := range accountInfo.Balance {
				fmt.Printf("  %s: Available=%s, Frozen=%s\n", bal.Currency, bal.Available, bal.Frozen)
			}
		}
	}

	// Test 2: Get ticker for cmt_btcusdt
	logger.Info("\nTest 2: Getting ticker for cmt_btcusdt...")
	symbol := "cmt_btcusdt"
	ticker, err := client.GetTicker(symbol)
	if err != nil {
		logger.Errorf("Failed to get ticker: %v", err)
	} else {
		logger.Infof("Ticker: %+v", ticker)
		fmt.Printf("\n=== Ticker Information ===\n")
		fmt.Printf("Symbol: %s\n", ticker.Symbol)
		fmt.Printf("Last Price: %s\n", ticker.LastPrice)
		fmt.Printf("24h High: %s\n", ticker.High24h)
		fmt.Printf("24h Low: %s\n", ticker.Low24h)
		fmt.Printf("24h Volume: %s\n", ticker.Volume24h)
		fmt.Printf("24h Change: %s\n", ticker.Change24h)
	}

	// Test 3: Set leverage (if needed)
	if len(os.Args) > 1 && os.Args[1] == "set-leverage" {
		logger.Info("\nTest 3: Setting leverage to 1x...")
		err := client.SetLeverage(symbol, 1, "1", "1")
		if err != nil {
			logger.Errorf("Failed to set leverage: %v", err)
		} else {
			logger.Info("Leverage set successfully")
			fmt.Printf("\n=== Leverage Set ===\n")
			fmt.Printf("Symbol: %s\n", symbol)
			fmt.Printf("Margin Mode: 1 (全仓)\n")
			fmt.Printf("Long/Short Leverage: 1x\n")
		}
	}

	// Test 4: Create test order (only if explicitly requested)
	if len(os.Args) > 1 && os.Args[1] == "test-order" {
		logger.Info("\nTest 4: Creating test order...")

		// Get current price first
		if ticker == nil {
			ticker, err = client.GetTicker(symbol)
			if err != nil {
				logger.Fatalf("Failed to get ticker for order: %v", err)
			}
		}

		// Create a small market buy order (约10 USDT)
		// For market order, we use quantity in USDT value
		orderReq := &api.CreateOrderRequest{
			Symbol:    symbol,
			Side:      api.OrderSideBuy,
			OrderType: api.OrderTypeMarket,
			Quantity:  "10", // 10 USDT worth
		}

		order, err := client.CreateOrder(orderReq)
		if err != nil {
			logger.Errorf("Failed to create order: %v", err)
		} else {
			logger.Infof("Order created: %+v", order)
			fmt.Printf("\n=== Order Created ===\n")
			fmt.Printf("Order ID: %s\n", order.OrderID)
			fmt.Printf("Symbol: %s\n", order.Symbol)
			fmt.Printf("Side: %s\n", order.Side)
			fmt.Printf("Type: %s\n", order.OrderType)
			fmt.Printf("Quantity: %s\n", order.Quantity)
			fmt.Printf("Status: %s\n", order.Status)
		}
	}

	fmt.Println("\n=== API Test Completed ===")
}
