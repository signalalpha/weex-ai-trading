# Trader CLI 使用说明

## 安装依赖

首先安装依赖：

```bash
go mod tidy
```

## 可用命令

### 查询账户信息
```bash
go run cmd/trader/main.go account
```

### 获取价格
```bash
go run cmd/trader/main.go price --symbol cmt_btcusdt
```

### 设置杠杆
```bash
go run cmd/trader/main.go leverage --symbol cmt_btcusdt --long 1 --short 1 --mode 1
```

### 下单
```bash
# 市价单
go run cmd/trader/main.go order --symbol cmt_btcusdt --side buy --type market --size 10

# 限价单
go run cmd/trader/main.go order --symbol cmt_btcusdt --side buy --type limit --size 10 --price 80000
```

### 运行完整测试（官方要求）
```bash
go run cmd/trader/main.go test
```

这将自动完成所有官方要求的测试步骤。

### 启动交易系统
```bash
go run cmd/trader/main.go run
```

## 全局选项

- `--log-level, -l`: 设置日志级别 (debug, info, warn, error)
- `--config, -c`: 指定配置文件路径

## 示例

```bash
# 查询账户余额
go run cmd/trader/main.go account

# 获取 BTC/USDT 价格
go run cmd/trader/main.go price -s cmt_btcusdt

# 设置杠杆为 1x
go run cmd/trader/main.go leverage -s cmt_btcusdt --long 1 --short 1

# 下单 10 USDT 市价买单
go run cmd/trader/main.go order -s cmt_btcusdt -d buy -t market -z 10

# 运行完整测试流程
go run cmd/trader/main.go test
```

