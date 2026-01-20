package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/signalalpha/weex-ai-trading/internal/config"
	"github.com/signalalpha/weex-ai-trading/internal/monitor"
	"github.com/signalalpha/weex-ai-trading/internal/trader"
	weexgo "github.com/signalalpha/weex-go"
	"github.com/urfave/cli/v2"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	app := &cli.App{
		Name:    "trader",
		Usage:   "WEEX AI Trading 交易系统",
		Version: fmt.Sprintf("%s (build: %s, commit: %s)", Version, BuildTime, GitCommit),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "配置文件路径",
			},
			&cli.StringFlag{
				Name:    "log-level",
				Aliases: []string{"l"},
				Value:   "info",
				Usage:   "日志级别 (debug, info, warn, error)",
			},
		},
		Commands: []*cli.Command{
			{
				Name:   "account",
				Usage:  "查询账户信息",
				Action: cmdAccount,
			},
			{
				Name:  "price",
				Usage: "获取交易对价格",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "symbol",
						Aliases: []string{"s"},
						Value:   "cmt_btcusdt",
						Usage:   "交易对符号",
					},
				},
				Action: cmdPrice,
			},
			{
				Name:  "leverage",
				Usage: "设置杠杆",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "symbol",
						Aliases: []string{"s"},
						Value:   "cmt_btcusdt",
						Usage:   "交易对符号",
					},
					&cli.StringFlag{
						Name:  "long",
						Value: "1",
						Usage: "做多杠杆倍数",
					},
					&cli.StringFlag{
						Name:  "short",
						Value: "1",
						Usage: "做空杠杆倍数",
					},
					&cli.IntFlag{
						Name:  "mode",
						Value: 1,
						Usage: "保证金模式 (1=全仓, 2=逐仓)",
					},
				},
				Action: cmdSetLeverage,
			},
			{
				Name:  "order",
				Usage: "下单",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "symbol",
						Aliases: []string{"s"},
						Value:   "cmt_btcusdt",
						Usage:   "交易对符号",
					},
					&cli.StringFlag{
						Name:    "side",
						Aliases: []string{"d"},
						Value:   "buy",
						Usage:   "交易方向 (buy/sell)",
					},
					&cli.StringFlag{
						Name:    "type",
						Aliases: []string{"t"},
						Value:   "market",
						Usage:   "订单类型 (market/limit)",
					},
					&cli.StringFlag{
						Name:    "size",
						Aliases: []string{"z"},
						Value:   "10",
						Usage:   "订单数量 (USDT)",
					},
					&cli.StringFlag{
						Name:  "price",
						Usage: "限价单价格 (限价单必填)",
					},
				},
				Action: cmdPlaceOrder,
			},
			{
				Name:  "orders",
				Usage: "查询当前委托",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "symbol",
						Aliases: []string{"s"},
						Value:   "cmt_btcusdt",
						Usage:   "交易对符号",
					},
				},
				Action: cmdCurrentOrders,
			},
			{
				Name:  "history",
				Usage: "查询历史委托",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "symbol",
						Aliases: []string{"s"},
						Value:   "cmt_btcusdt",
						Usage:   "交易对符号",
					},
					&cli.IntFlag{
						Name:  "limit",
						Value: 10,
						Usage: "返回记录数",
					},
				},
				Action: cmdOrderHistory,
			},
			{
				Name:  "trades",
				Usage: "查询交易详情",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "symbol",
						Aliases: []string{"s"},
						Value:   "cmt_btcusdt",
						Usage:   "交易对符号",
					},
					&cli.IntFlag{
						Name:  "limit",
						Value: 10,
						Usage: "返回记录数",
					},
				},
				Action: cmdTradeDetails,
			},
			{
				Name:   "test",
				Usage:  "运行完整的 API 测试流程（官方要求）",
				Action: cmdOfficialTest,
			},
			{
				Name:  "run",
				Usage: "启动AI交易系统",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "dry-run",
						Usage: "模拟运行模式（不实际下单）",
						Value: false,
					},
				},
				Action: cmdRun,
			},
		},
		Before: func(c *cli.Context) error {
			// 加载配置
			configPath := c.String("config")
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// 如果指定了日志级别，覆盖配置
			if c.String("log-level") != "" {
				cfg.Log.Level = c.String("log-level")
			}

			// 将配置保存到上下文
			c.App.Metadata["config"] = cfg
			c.App.Metadata["logger"] = monitor.NewLogger(cfg.Log.Level, cfg.Log.Output)

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func getClient(c *cli.Context) (*weexgo.Client, error) {
	cfg := c.App.Metadata["config"].(*config.Config)

	opts := []weexgo.ClientOption{
		weexgo.WithAPIKey(cfg.WEEX.APIKey),
		weexgo.WithSecretKey(cfg.WEEX.SecretKey),
		weexgo.WithPassphrase(cfg.WEEX.Passphrase),
	}

	if cfg.WEEX.APIBaseURL != "" {
		opts = append(opts, weexgo.WithBaseURL(cfg.WEEX.APIBaseURL))
	}

	if cfg.WEEX.Proxy != "" {
		opts = append(opts, weexgo.WithProxy(cfg.WEEX.Proxy))
	}

	return weexgo.NewClient(opts...)
}

func printJSON(data interface{}) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return
	}
	fmt.Println(string(jsonData))
}

func cmdAccount(c *cli.Context) error {
	client, err := getClient(c)
	if err != nil {
		return err
	}

	fmt.Println("查询账户信息...")
	accountAssets, err := client.GetAccountAssets()
	if err != nil {
		return fmt.Errorf("failed to get account assets: %w", err)
	}

	fmt.Println("\n账户信息:")
	printJSON(accountAssets)
	return nil
}

func cmdPrice(c *cli.Context) error {
	client, err := getClient(c)
	if err != nil {
		return err
	}

	symbol := c.String("symbol")
	fmt.Printf("获取 %s 价格...\n", symbol)

	ticker, err := client.GetTicker(symbol)
	if err != nil {
		return fmt.Errorf("failed to get ticker: %w", err)
	}

	fmt.Printf("\n%s 行情信息:\n", symbol)
	printJSON(ticker)
	return nil
}

func cmdSetLeverage(c *cli.Context) error {
	client, err := getClient(c)
	if err != nil {
		return err
	}

	symbol := c.String("symbol")
	longLeverage := c.String("long")
	shortLeverage := c.String("short")
	marginMode := c.Int("mode")

	fmt.Printf("设置 %s 杠杆: 做多=%sx, 做空=%sx, 模式=%d...\n", symbol, longLeverage, shortLeverage, marginMode)

	err = client.SetLeverage(symbol, marginMode, longLeverage, shortLeverage)
	if err != nil {
		return fmt.Errorf("failed to set leverage: %w", err)
	}

	fmt.Println("✅ 杠杆设置成功")
	return nil
}

func cmdPlaceOrder(c *cli.Context) error {
	client, err := getClient(c)
	if err != nil {
		return err
	}

	symbol := c.String("symbol")
	sideStr := c.String("side")
	orderTypeStr := c.String("type")
	size := c.String("size")

	var side weexgo.OrderSide
	if sideStr == "buy" {
		side = weexgo.OrderSideBuy
	} else if sideStr == "sell" {
		side = weexgo.OrderSideSell
	} else {
		return fmt.Errorf("invalid side: %s (must be buy or sell)", sideStr)
	}

	var orderType weexgo.OrderType
	if orderTypeStr == "market" {
		orderType = weexgo.OrderTypeMarket
	} else if orderTypeStr == "limit" {
		orderType = weexgo.OrderTypeLimit
	} else {
		return fmt.Errorf("invalid order type: %s (must be market or limit)", orderTypeStr)
	}

	// 解析数量和价格
	var quantityFloat float64
	if _, err := fmt.Sscanf(size, "%f", &quantityFloat); err != nil {
		return fmt.Errorf("invalid size format: %s", size)
	}

	// 根据交易对精度调整数量
	adjustedSize := trader.AdjustSizeToPrecision(quantityFloat, symbol)
	adjustedSizeStr := fmt.Sprintf("%.6f", adjustedSize)

	req := &weexgo.CreateOrderRequest{
		Symbol:    symbol,
		Side:      side,
		OrderType: orderType,
		Quantity:  adjustedSizeStr,
	}

	if orderType == weexgo.OrderTypeLimit {
		priceStr := c.String("price")
		if priceStr == "" {
			return fmt.Errorf("price is required for limit orders")
		}

		// 解析价格
		var priceFloat float64
		if _, err := fmt.Sscanf(priceStr, "%f", &priceFloat); err != nil {
			return fmt.Errorf("invalid price format: %s", priceStr)
		}

		// 根据交易对精度调整价格
		adjustedPrice := trader.AdjustPriceToPrecision(priceFloat, symbol)
		req.Price = trader.FormatPriceString(adjustedPrice, symbol)
	}

	fmt.Printf("下单: %s %s %s %s USDT...\n", side, orderType, symbol, size)
	fmt.Printf("订单参数: %+v\n", req)

	order, err := client.CreateOrder(req)
	if err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}

	fmt.Println("\n✅ 订单创建成功:")
	printJSON(order)
	return nil
}

func cmdCurrentOrders(c *cli.Context) error {
	client, err := getClient(c)
	if err != nil {
		return err
	}

	symbol := c.String("symbol")
	fmt.Printf("查询 %s 的当前活跃订单...\n", symbol)

	orders, err := client.GetCurrentOrders(symbol)
	if err != nil {
		return fmt.Errorf("failed to get current orders: %w", err)
	}

	if len(orders) == 0 {
		fmt.Println("\n✅ 当前没有活跃订单")
		return nil
	}

	fmt.Printf("\n✅ 找到 %d 个活跃订单:\n\n", len(orders))
	printJSON(orders)
	return nil
}

func cmdOrderHistory(c *cli.Context) error {
	// TODO: 需要实现 GetOrderHistory 方法
	return fmt.Errorf("not implemented yet")
}

func cmdTradeDetails(c *cli.Context) error {
	// TODO: 需要实现 GetTradeDetails 方法
	return fmt.Errorf("not implemented yet")
}

func cmdOfficialTest(c *cli.Context) error {
	client, err := getClient(c)
	if err != nil {
		return err
	}

	symbol := "cmt_btcusdt"
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("WEEX AI Trading Hackathon - API 测试")
	fmt.Println(strings.Repeat("=", 60))

	// 步骤 1: 检查账户余额
	fmt.Println("\n[步骤 1] 检查账户余额")
	accountAssets, err := client.GetAccountAssets()
	if err != nil {
		return fmt.Errorf("failed to get account assets: %w", err)
	}
	fmt.Println("✅ 账户信息获取成功")
	printJSON(accountAssets)

	// 步骤 2: 设置杠杆
	fmt.Println("\n[步骤 2] 设置杠杆 (1x, 全仓模式)")
	err = client.SetLeverage(symbol, 1, "1", "1")
	if err != nil {
		fmt.Printf("⚠️  杠杆设置失败: %v (继续执行...)\n", err)
	} else {
		fmt.Println("✅ 杠杆设置成功")
	}

	// 步骤 3: 获取价格
	fmt.Printf("\n[步骤 3] 获取 %s 价格\n", symbol)
	ticker, err := client.GetTicker(symbol)
	if err != nil {
		return fmt.Errorf("failed to get ticker: %w", err)
	}
	fmt.Println("✅ 价格获取成功")
	printJSON(ticker)

	// 步骤 4: 下单
	fmt.Println("\n[步骤 4] 下单 (10 USDT)")
	orderReq := &weexgo.CreateOrderRequest{
		Symbol:    symbol,
		Side:      weexgo.OrderSideBuy,
		OrderType: weexgo.OrderTypeMarket,
		Quantity:  "10",
	}
	order, err := client.CreateOrder(orderReq)
	if err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}
	fmt.Println("✅ 订单提交成功")
	printJSON(order)

	// 等待订单执行
	fmt.Println("\n等待 3 秒让订单执行...")
	time.Sleep(3 * time.Second)

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("测试完成!")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("\n如果所有步骤都成功完成，您应该已经满足了官方要求。")

	return nil
}

func cmdRun(c *cli.Context) error {
	cfg := c.App.Metadata["config"].(*config.Config)
	logger := c.App.Metadata["logger"].(*monitor.Logger)

	// 检查 Claude API Key
	claudeAPIKey := os.Getenv("CLAUDE_API_KEY")
	if claudeAPIKey == "" {
		return fmt.Errorf("CLAUDE_API_KEY environment variable is required")
	}

	// 创建 WEEX 客户端
	client, err := getClient(c)
	if err != nil {
		return fmt.Errorf("failed to create WEEX client: %w", err)
	}

	// 创建引擎配置
	engineConfig := trader.EngineConfig{
		Symbol:               cfg.Trading.DefaultSymbol,
		DecisionInterval:     60,   // 每60秒决策一次
		MaxPosition:          0.01, // 最大持仓0.01 BTC
		ClaudeModel:          "claude-3-5-sonnet-20241022",
		ClaudeAPIKey:         claudeAPIKey,
		EnableMultiTimeframe: false, // 暂时禁用多时间框架（需要K线API支持）
		EnableOrderBook:      false, // 暂时禁用订单簿（需要API支持）
		DryRun:               c.Bool("dry-run"),
		LogLevel:             cfg.Log.Level,
	}

	// 创建交易引擎
	engine, err := trader.NewEngine(engineConfig, client, logger)
	if err != nil {
		return fmt.Errorf("failed to create trading engine: %w", err)
	}

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动引擎（异步）
	go func() {
		if err := engine.Run(); err != nil {
			logger.Errorf("Engine error: %v", err)
		}
	}()

	logger.Info("✅ 交易引擎已启动，按 Ctrl+C 停止")

	// 等待停止信号
	<-sigChan
	logger.Info("\n收到停止信号，正在关闭...")

	// 停止引擎
	engine.Stop()

	logger.Info("👋 交易引擎已停止")
	return nil
}
