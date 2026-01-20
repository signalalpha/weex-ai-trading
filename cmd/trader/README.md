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

### 查询当前活跃订单
```bash
# 查询指定交易对的当前活跃订单
go run cmd/trader/main.go orders --symbol cmt_btcusdt

# 使用简写
go run cmd/trader/main.go orders -s cmt_ethusdt
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

## 代理配置

如果需要通过代理访问 WEEX API，可以通过以下方式设置：

### 方式 1: 环境变量（推荐）

```bash
# 使用 WEEX_PROXY（优先级最高）
export WEEX_PROXY="http://proxy.example.com:3128"

# 或使用 HTTPS_PROXY
export HTTPS_PROXY="http://proxy.example.com:3128"

# 或使用 HTTP_PROXY
export HTTP_PROXY="http://proxy.example.com:3128"

# 带认证的代理
export WEEX_PROXY="http://username:password@proxy.example.com:3128"
```

### 方式 2: 配置文件

在配置文件中添加：

```yaml
weex:
  proxy: "http://proxy.example.com:3128"
```

### 代理格式

- **不带认证**: `http://proxy.example.com:3128`
- **带认证**: `http://username:password@proxy.example.com:3128`

环境变量优先级：`WEEX_PROXY` > `HTTPS_PROXY` > `HTTP_PROXY`

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

# 查询当前活跃订单
go run cmd/trader/main.go orders -s cmt_btcusdt

# 运行完整测试流程
go run cmd/trader/main.go test

# 使用代理运行命令
export WEEX_PROXY="http://proxy.example.com:3128"
go run cmd/trader/main.go account
```

