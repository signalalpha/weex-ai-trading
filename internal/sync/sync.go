package sync

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/signalalpha/weex-ai-trading/internal/database"
	"github.com/signalalpha/weex-ai-trading/internal/monitor"
	weexgo "github.com/signalalpha/weex-go"
)

// Service handles syncing trade fills from WEEX API to database
type Service struct {
	db     *database.DB
	config *Config
	logger *monitor.Logger
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewService creates a new sync service
func NewService(db *database.DB, cfg *Config, logger *monitor.Logger) *Service {
	return &Service{
		db:     db,
		config: cfg,
		logger: logger,
		stopCh: make(chan struct{}),
	}
}

// Start starts the sync service
func (s *Service) Start() error {
	// Initialize database schema
	if err := s.db.InitSchema(); err != nil {
		return fmt.Errorf("failed to initialize database schema: %w", err)
	}

	s.logger.Info("✅ 数据库表初始化成功")

	// Start sync loop
	s.wg.Add(1)
	go s.syncLoop()

	s.logger.Info("✅ 订单同步服务已启动")
	return nil
}

// Stop stops the sync service
func (s *Service) Stop() {
	close(s.stopCh)
	s.wg.Wait()
	s.logger.Info("👋 订单同步服务已停止")
}

// syncLoop runs the main sync loop
func (s *Service) syncLoop() {
	defer s.wg.Done()

	interval := time.Duration(s.config.Sync.IntervalSeconds) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run immediately on start
	s.syncAll()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.syncAll()
		}
	}
}

// syncAll syncs all users and symbols
func (s *Service) syncAll() {
	s.logger.Info("🔄 开始批量同步成交订单...")

	users := s.config.Sync.Users
	if len(users) == 0 {
		// If no users configured, use environment variables as single user
		apiKey := os.Getenv("WEEX_API_KEY")
		secretKey := os.Getenv("WEEX_SECRET_KEY")
		passphrase := os.Getenv("WEEX_PASSPHRASE")

		if apiKey == "" || secretKey == "" {
			s.logger.Error("❌ 未配置用户且环境变量WEEX_API_KEY/WEEX_SECRET_KEY未设置")
			return
		}

		users = []UserConfig{
			{
				UserID:     "default",
				APIKey:     apiKey,
				SecretKey:  secretKey,
				Passphrase: passphrase,
				Enabled:    true,
			},
		}
	}

	symbols := s.config.Sync.Symbols
	if len(symbols) == 0 {
		symbols = []string{"cmt_btcusdt"} // Default symbol
	}

	// Sync each user
	for _, user := range users {
		if !user.Enabled {
			s.logger.WithFields(map[string]interface{}{
				"user_id": user.UserID,
			}).Info("⏭️  跳过已禁用的用户")
			continue
		}

		// Sync each symbol for this user
		for _, symbol := range symbols {
			s.syncUserSymbol(user, symbol)
		}
	}

	s.logger.Info("✅ 批量同步完成")
}

// syncUserSymbol syncs trade fills for a specific user and symbol
func (s *Service) syncUserSymbol(user UserConfig, symbol string) {
	startTime := time.Now()

	s.logger.WithFields(map[string]interface{}{
		"user_id": user.UserID,
		"symbol":  symbol,
	}).Info("📥 同步用户成交订单")

	// Create WEEX client for this user
	opts := []weexgo.ClientOption{
		weexgo.WithAPIKey(user.APIKey),
		weexgo.WithSecretKey(user.SecretKey),
		weexgo.WithPassphrase(user.Passphrase),
	}

	// Add proxy if configured
	if s.config.Sync.WEEX.Proxy != "" {
		opts = append(opts, weexgo.WithProxy(s.config.Sync.WEEX.Proxy))
	}

	// Add base URL if configured
	if s.config.Sync.WEEX.APIBaseURL != "" {
		opts = append(opts, weexgo.WithBaseURL(s.config.Sync.WEEX.APIBaseURL))
	}

	client, err := weexgo.NewClient(opts...)
	if err != nil {
		s.logger.WithError(err).WithFields(map[string]interface{}{
			"user_id": user.UserID,
			"symbol":  symbol,
		}).Error("❌ 创建WEEX客户端失败")

		s.db.SaveSyncStatus(user.UserID, symbol, 0, 0, "error", err.Error())
		return
	}

	// Get last trade time from database
	lastTradeTime, err := s.db.GetLastTradeTime(user.UserID, symbol)
	if err != nil {
		s.logger.WithError(err).WithFields(map[string]interface{}{
			"user_id": user.UserID,
			"symbol":  symbol,
		}).Warn("⚠️  获取最后交易时间失败，将获取所有记录")
	}

	// Fetch trade fills from API
	fills, err := client.GetTradeFills(symbol, nil)
	if err != nil {
		s.logger.WithError(err).WithFields(map[string]interface{}{
			"user_id": user.UserID,
			"symbol":  symbol,
		}).Error("❌ 获取成交订单失败")

		s.db.SaveSyncStatus(user.UserID, symbol, lastTradeTime, 0, "error", err.Error())
		return
	}

	if len(fills) == 0 {
		s.logger.WithFields(map[string]interface{}{
			"user_id": user.UserID,
			"symbol":  symbol,
		}).Info("ℹ️  没有新的成交订单")

		s.db.SaveSyncStatus(user.UserID, symbol, lastTradeTime, 0, "success", "")
		return
	}

	// Filter fills that are newer than last trade time
	var newFills weexgo.TradeFills
	var maxTradeTime int64 = lastTradeTime

	for _, fill := range fills {
		if fill.CreatedTime > lastTradeTime {
			newFills = append(newFills, fill)
			if fill.CreatedTime > maxTradeTime {
				maxTradeTime = fill.CreatedTime
			}
		}
	}

	if len(newFills) == 0 {
		s.logger.WithFields(map[string]interface{}{
			"user_id": user.UserID,
			"symbol":  symbol,
			"total":   len(fills),
		}).Info("ℹ️  没有新的成交订单（所有记录都已同步）")

		s.db.SaveSyncStatus(user.UserID, symbol, lastTradeTime, 0, "success", "")
		return
	}

	// Save to database
	savedCount, err := s.db.SaveTradeFills(user.UserID, newFills)
	if err != nil {
		s.logger.WithError(err).WithFields(map[string]interface{}{
			"user_id": user.UserID,
			"symbol":  symbol,
			"count":   len(newFills),
		}).Error("❌ 保存成交订单失败")

		s.db.SaveSyncStatus(user.UserID, symbol, maxTradeTime, savedCount, "error", err.Error())
		return
	}

	duration := time.Since(startTime)
	s.logger.WithFields(map[string]interface{}{
		"user_id":     user.UserID,
		"symbol":      symbol,
		"fetched":     len(fills),
		"new":         len(newFills),
		"saved":       savedCount,
		"duration_ms": duration.Milliseconds(),
	}).Info("✅ 同步成功")

	// Save sync status
	s.db.SaveSyncStatus(user.UserID, symbol, maxTradeTime, savedCount, "success", "")
}
