package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/signalalpha/weex-ai-trading/internal/database"
	"github.com/signalalpha/weex-ai-trading/internal/monitor"
	"github.com/signalalpha/weex-ai-trading/internal/sync"
	"github.com/urfave/cli/v2"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	app := &cli.App{
		Name:    "weex-sync",
		Usage:   "WEEX交易大赛订单同步服务",
		Version: fmt.Sprintf("%s (build: %s, commit: %s)", Version, BuildTime, GitCommit),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   "configs/sync_config.yaml",
				Usage:   "配置文件路径",
			},
			&cli.StringFlag{
				Name:    "log-level",
				Aliases: []string{"l"},
				Value:   "info",
				Usage:   "日志级别 (debug, info, warn, error)",
			},
		},
		Action: runSync,
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runSync(c *cli.Context) error {
	// Load configuration
	configPath := c.String("config")
	cfg, err := sync.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Override log level if specified
	logLevel := c.String("log-level")
	if logLevel != "" {
		cfg.Log.Level = logLevel
	}

	// Create logger
	logger := monitor.NewLogger(cfg.Log.Level, cfg.Log.Output)

	logger.Info("🚀 启动WEEX订单同步服务")
	logger.WithFields(map[string]interface{}{
		"config_file": configPath,
		"log_level":   cfg.Log.Level,
	}).Info("📋 配置信息")

	// Validate database configuration
	if cfg.Database.Host == "" {
		return fmt.Errorf("database configuration is required (set SYNC_DB_HOST or database.host in config)")
	}

	// Create database connection
	dbConfig := database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
		SSLMode:  cfg.Database.SSLMode,
	}

	db, err := database.New(dbConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	logger.Info("✅ 数据库连接成功")

	// Create sync service
	syncService := sync.NewService(db, cfg, logger)

	// Start sync service
	if err := syncService.Start(); err != nil {
		return fmt.Errorf("failed to start sync service: %w", err)
	}

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("✅ 订单同步服务已启动，按 Ctrl+C 停止")
	logger.WithFields(map[string]interface{}{
		"interval_seconds": cfg.Sync.IntervalSeconds,
		"page_size":        cfg.Sync.PageSize,
		"symbols":          cfg.Sync.Symbols,
		"users_count":      len(cfg.Sync.Users),
	}).Info("📊 同步配置")

	// Wait for stop signal
	<-sigChan
	logger.Info("\n收到停止信号，正在关闭...")

	// Stop sync service
	syncService.Stop()

	logger.Info("👋 订单同步服务已停止")
	return nil
}
