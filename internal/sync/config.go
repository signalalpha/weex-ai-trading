package sync

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Config represents the sync service configuration
type Config struct {
	Database DatabaseConfig `mapstructure:"database"`
	Sync     SyncConfig     `mapstructure:"sync"`
	Log      LogConfig      `mapstructure:"log"`
}

// DatabaseConfig contains PostgreSQL database configuration
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

// UserConfig represents a user's API credentials
type UserConfig struct {
	UserID    string `mapstructure:"user_id"`
	APIKey    string `mapstructure:"api_key"`
	SecretKey string `mapstructure:"secret_key"`
	Passphrase string `mapstructure:"passphrase"`
	Enabled   bool   `mapstructure:"enabled"`
}

// SyncConfig contains sync service configuration
type SyncConfig struct {
	IntervalSeconds int          `mapstructure:"interval_seconds"` // 同步间隔（秒）
	PageSize        int          `mapstructure:"page_size"`         // 每次获取的记录数
	Symbols         []string     `mapstructure:"symbols"`           // 要同步的交易对列表
	Users           []UserConfig `mapstructure:"users"`             // 用户列表
	WEEX            WEEXConfig   `mapstructure:"weex"`              // WEEX API配置（用于默认用户）
}

// WEEXConfig contains WEEX API configuration (for default user)
type WEEXConfig struct {
	APIBaseURL string `mapstructure:"api_base_url"`
	Proxy      string `mapstructure:"proxy"`
}

// LogConfig contains logging configuration
type LogConfig struct {
	Level  string `mapstructure:"level"`  // debug, info, warn, error
	Output string `mapstructure:"output"` // console, file, both
}

// Load loads sync service configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	viper.SetConfigType("yaml")

	// Set defaults
	setDefaults()

	// Read from environment variables
	viper.SetEnvPrefix("SYNC")
	viper.AutomaticEnv()

	// Bind environment variables
	bindEnvVars()

	// If config path is provided, use it directly
	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		// Otherwise, use default search paths
		viper.SetConfigName("sync_config")
		viper.AddConfigPath("./configs")
		viper.AddConfigPath(".")
	}

	// Try to read config file
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Apply environment variables (override config file)
	applyEnvVars(&cfg)

	// Validate configuration
	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func setDefaults() {
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "postgres")
	viper.SetDefault("database.password", "")
	viper.SetDefault("database.dbname", "weex_trading")
	viper.SetDefault("database.sslmode", "disable")

	viper.SetDefault("sync.interval_seconds", 60)
	viper.SetDefault("sync.page_size", 100)
	viper.SetDefault("sync.symbols", []string{"cmt_btcusdt"})

	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.output", "console")
}

func bindEnvVars() {
	// Database env vars
	viper.BindEnv("database.host", "SYNC_DB_HOST", "DB_HOST", "POSTGRES_HOST")
	viper.BindEnv("database.port", "SYNC_DB_PORT", "DB_PORT", "POSTGRES_PORT")
	viper.BindEnv("database.user", "SYNC_DB_USER", "DB_USER", "POSTGRES_USER")
	viper.BindEnv("database.password", "SYNC_DB_PASSWORD", "DB_PASSWORD", "POSTGRES_PASSWORD")
	viper.BindEnv("database.dbname", "SYNC_DB_NAME", "DB_NAME", "POSTGRES_DB")
	viper.BindEnv("database.sslmode", "SYNC_DB_SSLMODE", "DB_SSLMODE")

	// Sync env vars
	viper.BindEnv("sync.interval_seconds", "SYNC_INTERVAL_SECONDS")
	viper.BindEnv("sync.page_size", "SYNC_PAGE_SIZE")

	// Log env vars
	viper.BindEnv("log.level", "SYNC_LOG_LEVEL", "LOG_LEVEL")
	viper.BindEnv("log.output", "SYNC_LOG_OUTPUT", "LOG_OUTPUT")
}

func applyEnvVars(cfg *Config) {
	// Database
	if envHost := os.Getenv("SYNC_DB_HOST"); envHost != "" {
		cfg.Database.Host = envHost
	} else if envHost := os.Getenv("DB_HOST"); envHost != "" {
		cfg.Database.Host = envHost
	}

	if envPort := os.Getenv("SYNC_DB_PORT"); envPort != "" {
		// Parse port if needed
	} else if envPort := os.Getenv("DB_PORT"); envPort != "" {
		// Parse port if needed
	}

	if envUser := os.Getenv("SYNC_DB_USER"); envUser != "" {
		cfg.Database.User = envUser
	} else if envUser := os.Getenv("DB_USER"); envUser != "" {
		cfg.Database.User = envUser
	}

	if envPassword := os.Getenv("SYNC_DB_PASSWORD"); envPassword != "" {
		cfg.Database.Password = envPassword
	} else if envPassword := os.Getenv("DB_PASSWORD"); envPassword != "" {
		cfg.Database.Password = envPassword
	}

	if envDBName := os.Getenv("SYNC_DB_NAME"); envDBName != "" {
		cfg.Database.DBName = envDBName
	} else if envDBName := os.Getenv("DB_NAME"); envDBName != "" {
		cfg.Database.DBName = envDBName
	}

	// Log
	if envLevel := os.Getenv("SYNC_LOG_LEVEL"); envLevel != "" {
		cfg.Log.Level = envLevel
	}
}

func validate(cfg *Config) error {
	// Validate database configuration
	if cfg.Database.Host == "" {
		return fmt.Errorf("database.host is required")
	}
	if cfg.Database.DBName == "" {
		return fmt.Errorf("database.dbname is required")
	}

	// Validate sync configuration
	if cfg.Sync.IntervalSeconds <= 0 {
		return fmt.Errorf("sync.interval_seconds must be greater than 0")
	}
	if len(cfg.Sync.Symbols) == 0 {
		return fmt.Errorf("sync.symbols cannot be empty")
	}

	// Validate users
	if len(cfg.Sync.Users) == 0 {
		// If no users configured, that's OK - will use default WEEX config
		// But we need at least API credentials from environment
		if os.Getenv("WEEX_API_KEY") == "" {
			return fmt.Errorf("either sync.users must be configured or WEEX_API_KEY environment variable must be set")
		}
	} else {
		// Validate each user
		for i, user := range cfg.Sync.Users {
			if user.UserID == "" {
				return fmt.Errorf("sync.users[%d].user_id is required", i)
			}
			if user.APIKey == "" {
				return fmt.Errorf("sync.users[%d].api_key is required", i)
			}
			if user.SecretKey == "" {
				return fmt.Errorf("sync.users[%d].secret_key is required", i)
			}
		}
	}

	return nil
}
