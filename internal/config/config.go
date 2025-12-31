package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	WEEX    WEEXConfig    `mapstructure:"weex"`
	Log     LogConfig     `mapstructure:"log"`
	Trading TradingConfig `mapstructure:"trading"`
	Risk    RiskConfig    `mapstructure:"risk"`
}

// WEEXConfig contains WEEX API configuration
type WEEXConfig struct {
	APIKey     string `mapstructure:"api_key"`
	SecretKey  string `mapstructure:"secret_key"`
	Passphrase string `mapstructure:"passphrase"`
	Env        string `mapstructure:"env"` // production or testnet
	APIBaseURL string `mapstructure:"api_base_url"`
	WSURL      string `mapstructure:"ws_url"`
}

// LogConfig contains logging configuration
type LogConfig struct {
	Level  string `mapstructure:"level"`  // debug, info, warn, error
	Output string `mapstructure:"output"` // console, file, both
}

// TradingConfig contains trading configuration
type TradingConfig struct {
	DefaultSymbol string `mapstructure:"default_symbol"`
}

// RiskConfig contains risk management configuration
type RiskConfig struct {
	MaxPositionSize      float64 `mapstructure:"max_position_size"`
	MaxDrawdown          float64 `mapstructure:"max_drawdown"`
	StopLossPercentage   float64 `mapstructure:"stop_loss_percentage"`
	TakeProfitPercentage float64 `mapstructure:"take_profit_percentage"`
}

// Load loads configuration from file and environment variables
// If configPath is empty, it will search in default locations (./configs, .)
func Load(configPath ...string) (*Config, error) {
	viper.SetConfigType("yaml")

	// Set defaults
	setDefaults()

	// Read from environment variables
	viper.SetEnvPrefix("WEEX")
	viper.AutomaticEnv()

	// Bind environment variables
	bindEnvVars()

	// If config path is provided, use it directly
	if len(configPath) > 0 && configPath[0] != "" {
		viper.SetConfigFile(configPath[0])
	} else {
		// Otherwise, use default search paths
		viper.SetConfigName("config")
		viper.AddConfigPath("./configs")
		viper.AddConfigPath(".")
	}

	// Try to read config file (optional)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found is OK if we have env vars
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// After unmarshaling, manually check and apply environment variables
	// This ensures env vars override config file values
	if envAPIKey := os.Getenv("WEEX_API_KEY"); envAPIKey != "" {
		cfg.WEEX.APIKey = envAPIKey
	}
	if envSecretKey := os.Getenv("WEEX_SECRET_KEY"); envSecretKey != "" {
		cfg.WEEX.SecretKey = envSecretKey
	}
	if envPassphrase := os.Getenv("WEEX_PASSPHRASE"); envPassphrase != "" {
		cfg.WEEX.Passphrase = envPassphrase
	}
	if envEnv := os.Getenv("WEEX_ENV"); envEnv != "" {
		cfg.WEEX.Env = envEnv
	}

	// Validate required fields
	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func setDefaults() {
	viper.SetDefault("weex.env", "testnet")
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.output", "console")
	viper.SetDefault("trading.default_symbol", "cmt_btcusdt")
	viper.SetDefault("risk.max_position_size", 1000.0)
	viper.SetDefault("risk.max_drawdown", 0.1)
	viper.SetDefault("risk.stop_loss_percentage", 0.02)
	viper.SetDefault("risk.take_profit_percentage", 0.05)
}

func bindEnvVars() {
	viper.BindEnv("weex.api_key", "WEEX_API_KEY")
	viper.BindEnv("weex.secret_key", "WEEX_SECRET_KEY")
	viper.BindEnv("weex.passphrase", "WEEX_PASSPHRASE")
	viper.BindEnv("weex.env", "WEEX_ENV")
	viper.BindEnv("log.level", "LOG_LEVEL")
	viper.BindEnv("log.output", "LOG_OUTPUT")
	viper.BindEnv("trading.default_symbol", "DEFAULT_SYMBOL")
}

func validate(cfg *Config) error {
	// Check API Key - viper should have merged config file and env vars
	// So we check the final value in cfg, which should have env vars if they exist
	apiKey := cfg.WEEX.APIKey
	if apiKey == "" || apiKey == "your_api_key_here" {
		// Also check environment variable directly as fallback
		if envKey := os.Getenv("WEEX_API_KEY"); envKey != "" {
			// Env var exists, update the config
			cfg.WEEX.APIKey = envKey
		} else {
			return fmt.Errorf("WEEX_API_KEY is required (set via environment variable or config file)")
		}
	}

	secretKey := cfg.WEEX.SecretKey
	if secretKey == "" || secretKey == "your_secret_key_here" {
		if envKey := os.Getenv("WEEX_SECRET_KEY"); envKey != "" {
			cfg.WEEX.SecretKey = envKey
		} else {
			return fmt.Errorf("WEEX_SECRET_KEY is required (set via environment variable or config file)")
		}
	}

	passphrase := cfg.WEEX.Passphrase
	if passphrase == "" || passphrase == "your_passphrase_here" {
		if envKey := os.Getenv("WEEX_PASSPHRASE"); envKey != "" {
			cfg.WEEX.Passphrase = envKey
		}
		// Passphrase might be optional for some endpoints, but usually required
		// Uncomment the following lines if it's mandatory:
		// else {
		// 	return fmt.Errorf("WEEX_PASSPHRASE is required")
		// }
	}

	if cfg.WEEX.Env != "production" && cfg.WEEX.Env != "testnet" {
		return fmt.Errorf("invalid WEEX_ENV: must be 'production' or 'testnet'")
	}

	return nil
}
