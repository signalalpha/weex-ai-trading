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
- **存储**: (待定)
- **配置**: Viper
- **日志**: Logrus/Zap

## 项目状态

🚧 **开发中** - 当前处于早期开发阶段

- [x] 项目初始化
- [ ] API集成
- [ ] 策略开发
- [ ] 系统集成
- [ ] 测试优化

## 快速开始

### 前置要求

- Go 1.21 或更高版本
- WEEX API Key 和 Secret Key

### 安装

```bash
git clone https://github.com/signalalpha/weex-ai-trading.git
cd weex-ai-trading
go mod download
```

### 配置

1. 复制配置文件示例：
```bash
cp .env.example .env
```

2. 编辑 `.env` 文件，填入你的 API Key：
```env
WEEX_API_KEY=your_api_key
WEEX_SECRET_KEY=your_secret_key
WEEX_ENV=production  # 或 testnet
```

### 运行

```bash
go run cmd/trader/main.go
```

## 项目结构

```
weex-ai-trading/
├── cmd/              # 应用程序入口
├── internal/         # 内部包
│   ├── api/         # API客户端
│   ├── collector/   # 数据采集
│   ├── strategy/    # 策略引擎
│   ├── execution/   # 执行引擎
│   ├── risk/        # 风控系统
│   └── config/      # 配置管理
├── pkg/             # 可复用的包
├── configs/         # 配置文件
├── docs/            # 文档
└── tests/           # 测试
```

## 使用说明

（待补充）

## 开发计划

详细开发计划请参考：[项目实施计划](../roadmap/项目实施计划.md)

## 许可证

MIT License

## 免责声明

本项目仅用于学习和研究目的。使用本软件进行交易存在风险，作者不对任何交易损失负责。请谨慎使用，并充分了解相关风险。

## 贡献

欢迎提交 Issue 和 Pull Request！

## 联系方式

- 项目主页: https://github.com/signalalpha/weex-ai-trading

---

**注意**: 本项目正在积极开发中，API 和功能可能会发生变化。
