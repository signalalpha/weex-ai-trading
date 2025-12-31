# WEEX AI Trading

WEEX AI Trading 黑客松大赛参赛项目 - 基于人工智能的自动化交易系统

## 项目简介

这是一个参加 WEEX AI Trading 黑客松大赛的项目，旨在开发一个高性能、智能化的自动化交易系统。系统使用 Go 语言开发，集成了机器学习模型，能够实时分析市场数据并执行交易决策。

## 功能特性

- 🤖 **AI驱动**: 集成机器学习模型进行市场分析和交易决策
- ⚡ **高性能**: 基于 Go 语言的高并发架构
- 📊 **实时数据**: WebSocket 实时市场数据采集
- 🛡️ **风险控制**: 完善的止损、止盈和仓位管理机制
- 📈 **回测系统**: 内置策略回测功能
- 🔍 **监控告警**: 实时监控和异常告警

## 技术栈

- **语言**: Go 1.21+
- **API**: WEEX REST API & WebSocket
- **配置**: Viper
- **日志**: Logrus
- **CLI**: urfave/cli

## 快速开始

### 前置要求

- Go 1.21 或更高版本
- WEEX API Key、Secret Key 和 Passphrase

### 安装

```bash
git clone https://github.com/signalalpha/weex-ai-trading.git
cd weex-ai-trading
make deps  # 或: go mod download
```

### 配置

1. 复制环境变量文件：
```bash
cp env.example .env
```

2. 编辑 `.env` 文件，填入你的 API Key：
```env
WEEX_API_KEY=your_api_key
WEEX_SECRET_KEY=your_secret_key
WEEX_PASSPHRASE=your_passphrase
WEEX_ENV=testnet  # 或 production
```

### 构建

#### 构建当前平台版本
```bash
make build
# 或
go build -o bin/trader cmd/trader/main.go
```

#### 构建 Linux AMD64 版本（用于服务器部署）
```bash
make build-linux
# 二进制文件: bin/trader-linux-amd64
```

#### 查看所有可用命令
```bash
make help
```

### 使用 CLI 命令

#### 查询账户信息
```bash
./bin/trader account
```

#### 获取价格
```bash
./bin/trader price --symbol cmt_btcusdt
```

#### 设置杠杆
```bash
./bin/trader leverage --symbol cmt_btcusdt --long 1 --short 1 --mode 1
```

#### 下单
```bash
# 市价单
./bin/trader order --symbol cmt_btcusdt --side buy --type market --size 10

# 限价单
./bin/trader order --symbol cmt_btcusdt --side buy --type limit --size 10 --price 80000
```

#### 运行完整 API 测试（官方要求）
```bash
./bin/trader test
```

### 部署到服务器

1. 构建 Linux 版本：
```bash
make build-linux
```

2. 拷贝到服务器：
```bash
scp bin/trader-linux-amd64 user@server:/path/to/destination/
```

3. 在服务器上设置权限并运行：
```bash
chmod +x trader-linux-amd64
./trader-linux-amd64 --help
```

或者使用部署检查命令查看详细步骤：
```bash
make deploy-check
```

## 项目结构

```
weex-ai-trading/
├── cmd/              # 应用程序入口
│   └── trader/       # 主程序
├── internal/         # 内部包
│   ├── api/         # API客户端
│   ├── collector/   # 数据采集
│   ├── strategy/    # 策略引擎
│   ├── execution/   # 执行引擎
│   ├── risk/        # 风控系统
│   ├── config/      # 配置管理
│   └── monitor/     # 监控日志
├── pkg/             # 可复用的包
├── configs/         # 配置文件
├── tests/           # 测试
│   └── api/         # API 测试脚本（Python）
├── scripts/         # 脚本文件
├── bin/             # 构建输出目录
├── Makefile         # 构建脚本
├── go.mod           # Go 模块定义
└── README.md        # 本文档
```

## Makefile 命令

- `make help` - 显示帮助信息
- `make build` - 构建当前平台版本
- `make build-linux` - 构建 Linux AMD64 版本（推荐用于服务器部署）
- `make build-all` - 构建多个平台版本
- `make clean` - 清理构建文件
- `make deps` - 下载并整理依赖
- `make fmt` - 格式化代码
- `make vet` - 运行 go vet
- `make lint` - 运行代码检查
- `make test` - 运行测试
- `make run` - 运行程序（开发模式）
- `make deploy-check` - 构建并检查部署文件

## 开发

```bash
# 设置开发环境
make dev-setup

# 运行程序
make run

# 或使用 go run
go run cmd/trader/main.go --help
```

## 项目状态

🚧 **开发中** - 当前处于开发阶段

- [x] 项目初始化
- [x] API 客户端基础框架
- [x] CLI 命令集成
- [ ] 数据采集模块
- [ ] 策略引擎
- [ ] 系统集成
- [ ] 测试优化

## 许可证

MIT License

## 免责声明

本项目仅用于学习和研究目的。使用本软件进行交易存在风险，作者不对任何交易损失负责。请谨慎使用，并充分了解相关风险。

## 贡献

欢迎提交 Issue 和 Pull Request！

---

**注意**: 本项目正在积极开发中，API 和功能可能会发生变化。
