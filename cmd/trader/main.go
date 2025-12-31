package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/signalalpha/weex-ai-trading/internal/api"
	"github.com/signalalpha/weex-ai-trading/internal/config"
	"github.com/signalalpha/weex-ai-trading/internal/monitor"
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
				Name:   "run",
				Usage:  "启动交易系统",
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

func getClient(c *cli.Context) (*api.Client, error) {
	cfg := c.App.Metadata["config"].(*config.Config)
	return api.NewClient(cfg)
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
	accountInfo, err := client.GetAccountInfo()
	if err != nil {
		return fmt.Errorf("failed to get account info: %w", err)
	}

	fmt.Println("\n账户信息:")
	printJSON(accountInfo)
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

	var side api.OrderSide
	if sideStr == "buy" {
		side = api.OrderSideBuy
	} else if sideStr == "sell" {
		side = api.OrderSideSell
	} else {
		return fmt.Errorf("invalid side: %s (must be buy or sell)", sideStr)
	}

	var orderType api.OrderType
	if orderTypeStr == "market" {
		orderType = api.OrderTypeMarket
	} else if orderTypeStr == "limit" {
		orderType = api.OrderTypeLimit
	} else {
		return fmt.Errorf("invalid order type: %s (must be market or limit)", orderTypeStr)
	}

	req := &api.CreateOrderRequest{
		Symbol:    symbol,
		Side:      side,
		OrderType: orderType,
		Quantity:  size,
	}

	if orderType == api.OrderTypeLimit {
		price := c.String("price")
		if price == "" {
			return fmt.Errorf("price is required for limit orders")
		}
		req.Price = price
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
	// TODO: 需要实现 GetCurrentOrders 方法
	return fmt.Errorf("not implemented yet")
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
	accountInfo, err := client.GetAccountInfo()
	if err != nil {
		return fmt.Errorf("failed to get account info: %w", err)
	}
	fmt.Println("✅ 账户信息获取成功")
	printJSON(accountInfo)

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
	orderReq := &api.CreateOrderRequest{
		Symbol:    symbol,
		Side:      api.OrderSideBuy,
		OrderType: api.OrderTypeMarket,
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
	logger := c.App.Metadata["logger"].(*monitor.Logger)
	logger.Info("Starting WEEX AI Trading system...")

	// TODO: Initialize and start trading system
	logger.Info("Trading system initialized. Waiting for shutdown signal...")
	logger.Info("(Trading system not yet implemented)")

	return nil
}
