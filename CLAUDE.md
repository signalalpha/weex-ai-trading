# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

WEEX AI Trading is an AI-driven automated trading system built for the WEEX AI Trading Hackathon. It's a high-performance trading bot written in Go that integrates with the WEEX exchange API to execute algorithmic trading strategies.

## Essential Commands

### Build and Development
```bash
# Build current platform binary
make build                    # Output: bin/trader

# Build for Linux server deployment
make build-linux              # Output: bin/trader-linux-amd64

# Build for all platforms
make build-all

# Development
make run                      # Run in development mode
go run cmd/trader/main.go     # Alternative development run

# Code quality
make fmt                      # Format code
make vet                      # Run go vet
make lint                     # Format + vet
make test                     # Run tests
make test-race                # Run tests with race detection

# Dependencies
make deps                     # Download and tidy dependencies
```

### CLI Commands (Built Binary)
```bash
# Account and market data
./bin/trader account                                    # Query account info
./bin/trader price --symbol cmt_btcusdt                # Get price ticker

# Trading operations
./bin/trader leverage --symbol cmt_btcusdt --long 1 --short 1 --mode 1
./bin/trader order --symbol cmt_btcusdt --side buy --type market --size 10
./bin/trader order --symbol cmt_btcusdt --side buy --type limit --size 10 --price 80000

# Testing
./bin/trader test              # Run official API test flow (required for hackathon)

# System
./bin/trader run               # Start trading system (not yet implemented)
```

### Python API Testing (tests/api/)
```bash
cd tests/api

# Single account test
python3 official_api_test.py

# Batch testing multiple accounts
python3 official_api_test_batch.py --api-keys api_keys.json --proxy http://proxy:3128
```

## Architecture

### Application Structure
- **cmd/trader/main.go**: CLI entry point using urfave/cli framework. Defines all subcommands and orchestrates API calls.
- **internal/config**: Configuration management using Viper. Supports both YAML files and environment variables, with env vars taking precedence.
- **internal/api**: Placeholder for API client (currently uses external github.com/signalalpha/weex-go SDK).
- **internal/monitor**: Logging infrastructure using logrus.
- **internal/strategy**: Strategy engine (not yet implemented).
- **internal/execution**: Order execution engine (not yet implemented).
- **internal/risk**: Risk management system (not yet implemented).
- **internal/collector**: Market data collection (not yet implemented).
- **internal/ml**: Machine learning models (not yet implemented).

### Configuration System
Configuration is loaded in this priority order (highest to lowest):
1. Environment variables (WEEX_API_KEY, WEEX_SECRET_KEY, etc.)
2. YAML config file (./configs/config.yaml or specified with --config)
3. Default values

The config system in internal/config/config.go explicitly re-applies environment variables after unmarshaling to ensure they override file values.

### WEEX API Integration
Uses the external SDK `github.com/signalalpha/weex-go` (v0.1.0) for all WEEX API calls. Key types:
- `weexgo.OrderSide`: OrderSideBuy, OrderSideSell
- `weexgo.OrderType`: OrderTypeMarket, OrderTypeLimit
- `weexgo.CreateOrderRequest`: Main order creation struct

### Official Test Flow
The `test` command (cmdOfficialTest in main.go) executes the required hackathon API test sequence:
1. Get account assets
2. Set leverage (1x, cross margin mode)
3. Get ticker price
4. Place market order (10 USDT minimum)
5. Wait 3 seconds for execution
This satisfies the official competition requirements.

## Configuration

### Environment Variables (Required)
```bash
WEEX_API_KEY=your_api_key
WEEX_SECRET_KEY=your_secret_key
WEEX_PASSPHRASE=your_passphrase
WEEX_ENV=testnet              # or production
```

### Optional Configuration
```bash
LOG_LEVEL=info                # debug, info, warn, error
LOG_OUTPUT=console            # console, file, both
DEFAULT_SYMBOL=cmt_btcusdt
```

Configuration files are located in `./configs/` directory. Use `config.example.yaml` as a template.

## Development Patterns

### Adding New CLI Commands
1. Define command structure in cmd/trader/main.go `Commands` slice
2. Create handler function (e.g., `cmdNewFeature`)
3. Use `getClient(c)` helper to obtain configured WEEX API client
4. Use `printJSON()` helper for consistent JSON output

### Error Handling
Commands return errors which are automatically displayed by the CLI framework. Use descriptive error wrapping:
```go
return fmt.Errorf("failed to create order: %w", err)
```

### Logging
Access logger from context metadata:
```go
logger := c.App.Metadata["logger"].(*monitor.Logger)
logger.Info("message")
```

## Testing Infrastructure

### Python Test Scripts
Located in `tests/api/`:
- **official_api_test.py**: Single account test following official requirements
- **official_api_test_batch.py**: Batch testing for multiple accounts with proxy support
- Supports JSON and CSV format for API keys
- Includes automatic order cleanup to prevent leverage setting failures

Test flow includes:
1. Check balance
2. Cancel all active orders (prevents leverage adjustment failures)
3. Set leverage to 20x
4. Place limit buy order at 95% price (won't execute)
5. Query pending orders
6. Place market buy order (executes immediately)
7. Place market sell order (closes position)
8. Query order history and trade details
9. Cancel remaining limit orders
10. Final cleanup

## Deployment

```bash
# Build for Linux servers
make build-linux

# Check deployment readiness
make deploy-check

# Transfer to server
scp bin/trader-linux-amd64 user@server:/path/to/destination/

# On server
chmod +x trader-linux-amd64
./trader-linux-amd64 --help
```

## Project Status

Currently in active development for the hackathon. Core API integration and CLI are complete. The following modules are placeholders:
- Strategy engine (internal/strategy)
- Execution engine (internal/execution)
- Risk management (internal/risk)
- Data collector (internal/collector)
- ML models (internal/ml)

The main trader system (`trader run` command) is not yet implemented - see cmdRun in cmd/trader/main.go:429.

## Key Dependencies

- **github.com/signalalpha/weex-go**: WEEX exchange API client SDK
- **github.com/urfave/cli/v2**: CLI framework
- **github.com/spf13/viper**: Configuration management
- **github.com/sirupsen/logrus**: Structured logging

Go version: 1.25.3
