package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// DB wraps the database connection
type DB struct {
	*sql.DB
}

// Config holds database configuration
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// New creates a new database connection
func New(cfg Config) (*DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	return &DB{sqlDB}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}

// InitSchema initializes the database schema
func (db *DB) InitSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS trade_fills (
		id BIGSERIAL PRIMARY KEY,
		user_id VARCHAR(100) NOT NULL,
		order_id VARCHAR(100) NOT NULL,
		trade_id VARCHAR(100),
		symbol VARCHAR(50) NOT NULL,
		side VARCHAR(10) NOT NULL,
		price DECIMAL(30, 8) NOT NULL,
		size DECIMAL(30, 8) NOT NULL,
		fee DECIMAL(30, 8) NOT NULL,
		fee_coin VARCHAR(20) NOT NULL,
		trade_time BIGINT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		CONSTRAINT unique_trade_fill UNIQUE (user_id, order_id, trade_id, trade_time)
	);

	CREATE INDEX IF NOT EXISTS idx_trade_fills_user_id ON trade_fills(user_id);
	CREATE INDEX IF NOT EXISTS idx_trade_fills_symbol ON trade_fills(symbol);
	CREATE INDEX IF NOT EXISTS idx_trade_fills_trade_time ON trade_fills(trade_time);
	CREATE INDEX IF NOT EXISTS idx_trade_fills_created_at ON trade_fills(created_at);
	CREATE INDEX IF NOT EXISTS idx_trade_fills_user_symbol_time ON trade_fills(user_id, symbol, trade_time DESC);

	CREATE TABLE IF NOT EXISTS user_configs (
		id SERIAL PRIMARY KEY,
		user_id VARCHAR(100) UNIQUE NOT NULL,
		api_key VARCHAR(200) NOT NULL,
		secret_key VARCHAR(200) NOT NULL,
		passphrase VARCHAR(200) NOT NULL,
		enabled BOOLEAN DEFAULT TRUE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS sync_status (
		id SERIAL PRIMARY KEY,
		user_id VARCHAR(100) NOT NULL,
		symbol VARCHAR(50) NOT NULL,
		last_sync_time TIMESTAMP NOT NULL,
		last_trade_time BIGINT,
		records_count INT DEFAULT 0,
		status VARCHAR(20) DEFAULT 'success',
		error_message TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_sync_status_user_symbol ON sync_status(user_id, symbol);
	CREATE INDEX IF NOT EXISTS idx_sync_status_last_sync_time ON sync_status(last_sync_time DESC);
	`

	_, err := db.Exec(schema)
	return err
}
