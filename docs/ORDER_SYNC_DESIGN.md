# WEEX交易大赛订单同步系统设计文档

## 概述

本系统设计用于WEEX交易大赛中，定时批量获取多个用户的成交订单，并保存到PostgreSQL数据库中，供后续分析和查询使用。

## 系统架构

```
┌─────────────────┐
│   Sync Service  │ 定时任务服务
└────────┬────────┘
         │
    ┌────┴────┐
    │         │
┌───▼───┐  ┌──▼────┐
│ User1 │  │ User2 │  批量用户
│Client │  │Client │  WEEX API客户端
└───┬───┘  └──┬────┘
    │         │
    └────┬────┘
         │
    ┌────▼────┐
    │ WEEX API│  获取成交订单
    │ /order/ │  /fills
    └────┬────┘
         │
    ┌────▼────┐
    │PostgreSQL│  保存订单数据
    │ Database │
    └──────────┘
```

## 核心组件

### 1. weex-go SDK扩展

**文件**: `weex-go/models.go`, `weex-go/client.go`

**功能**:
- 添加 `TradeFill` 结构体，表示成交订单
- 添加 `GetTradeFills()` 方法，调用 `/capi/v2/order/fills` API

**API端点**:
```
GET /capi/v2/order/fills?symbol={symbol}&pageSize={pageSize}
```

### 2. 数据库层

**文件**: 
- `internal/database/db.go` - 数据库连接和初始化
- `internal/database/trade_fill.go` - 订单数据操作
- `internal/database/schema.sql` - 数据库表结构

**表结构**:

#### trade_fills（成交订单表）
```sql
CREATE TABLE trade_fills (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(100) NOT NULL,
    order_id VARCHAR(100) NOT NULL,
    trade_id VARCHAR(100),
    symbol VARCHAR(50) NOT NULL,
    side VARCHAR(10) NOT NULL,  -- "1"=买, "2"=卖
    price DECIMAL(30, 8) NOT NULL,
    size DECIMAL(30, 8) NOT NULL,
    fee DECIMAL(30, 8) NOT NULL,
    fee_coin VARCHAR(20) NOT NULL,
    trade_time BIGINT NOT NULL,  -- 毫秒时间戳
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_trade_fill UNIQUE (user_id, order_id, trade_id, trade_time)
);
```

#### sync_status（同步状态表）
```sql
CREATE TABLE sync_status (
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
```

**关键功能**:
- `SaveTradeFills()` - 批量保存订单（使用事务）
- `GetLastTradeTime()` - 获取最后同步时间（用于增量同步）
- `SaveSyncStatus()` - 记录同步状态
- `GetTradeFills()` - 查询订单（支持多条件过滤）

### 3. 配置管理

**文件**: `internal/config/config.go`

**新增配置项**:

```yaml
database:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "your_password"
  dbname: "weex_trading"
  sslmode: "disable"

sync:
  interval_seconds: 60    # 同步间隔（秒）
  page_size: 100          # 每次获取的记录数
  symbols:                # 交易对列表
    - "cmt_btcusdt"
  users:                  # 用户列表
    - user_id: "user1"
      api_key: "..."
      secret_key: "..."
      passphrase: "..."
      enabled: true
```

### 4. 同步服务

**文件**: `internal/sync/sync.go`

**核心逻辑**:

1. **定时循环**: 根据配置的间隔时间，定时执行同步任务
2. **批量处理**: 遍历所有用户和交易对，逐个同步
3. **增量同步**: 
   - 从数据库获取最后同步时间
   - 只获取比最后时间更新的订单
   - 避免重复数据
4. **错误处理**: 
   - 单个用户失败不影响其他用户
   - 记录错误信息到 `sync_status` 表
5. **状态记录**: 每次同步都记录状态，便于监控和排查

**工作流程**:
```
启动服务
  ↓
初始化数据库表
  ↓
启动定时器（每N秒）
  ↓
遍历所有用户
  ↓
  遍历所有交易对
  ↓
    获取最后同步时间
    ↓
    调用API获取成交订单
    ↓
    过滤新订单（时间 > 最后同步时间）
    ↓
    批量保存到数据库
    ↓
    记录同步状态
  ↓
等待下次定时触发
```

### 5. 命令行接口

**文件**: `cmd/trader/main.go`

**新增命令**:
```bash
./trader sync -c configs/config.yaml
```

## 使用流程

### 1. 环境准备

```bash
# 安装PostgreSQL（如果还没有）
# macOS
brew install postgresql
brew services start postgresql

# 创建数据库
createdb weex_trading
```

### 2. 配置服务

编辑 `configs/config.yaml`:

```yaml
database:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "your_password"
  dbname: "weex_trading"

sync:
  interval_seconds: 60
  page_size: 100
  symbols:
    - "cmt_btcusdt"
  users:
    - user_id: "user1"
      api_key: "user1_api_key"
      secret_key: "user1_secret_key"
      passphrase: "user1_passphrase"
      enabled: true
```

### 3. 安装依赖

```bash
go mod tidy
```

### 4. 启动服务

```bash
# 编译
go build -o trader cmd/trader/main.go

# 运行
./trader sync -c configs/config.yaml
```

### 5. 查询数据

```sql
-- 查看所有成交订单
SELECT * FROM trade_fills ORDER BY trade_time DESC LIMIT 100;

-- 查看某个用户的成交订单
SELECT * FROM trade_fills WHERE user_id = 'user1' ORDER BY trade_time DESC;

-- 查看同步状态
SELECT * FROM sync_status ORDER BY last_sync_time DESC LIMIT 20;
```

## 设计特点

### 1. 增量同步
- 只同步新订单，避免重复数据
- 基于 `trade_time` 时间戳判断
- 使用唯一约束防止重复插入

### 2. 批量处理
- 支持多个用户同时同步
- 支持多个交易对
- 使用事务批量保存，提高性能

### 3. 容错机制
- 单个用户失败不影响其他用户
- 记录详细的错误信息
- 自动重试（通过定时器）

### 4. 可扩展性
- 易于添加新用户（只需配置）
- 易于添加新交易对
- 数据库索引优化查询性能

### 5. 监控和调试
- 详细的日志记录
- 同步状态表记录每次同步结果
- 支持查询历史同步记录

## 性能优化

1. **数据库索引**: 
   - `user_id` 索引
   - `symbol` 索引
   - `trade_time` 索引
   - 复合索引 `(user_id, symbol, trade_time)`

2. **连接池**: 
   - 最大连接数: 25
   - 空闲连接数: 5
   - 连接生命周期: 5分钟

3. **批量操作**: 
   - 使用事务批量插入
   - 使用 `ON CONFLICT DO NOTHING` 避免重复

4. **增量同步**: 
   - 只获取新数据，减少API调用
   - 减少数据库写入量

## 安全考虑

1. **API凭证**: 
   - 配置文件中的凭证应妥善保管
   - 建议使用环境变量或密钥管理服务

2. **数据库安全**: 
   - 使用SSL连接（生产环境）
   - 限制数据库访问权限
   - 定期备份数据

3. **错误信息**: 
   - 日志中不输出敏感信息
   - 错误信息记录在数据库中，便于排查

## 后续优化建议

1. **分页支持**: 
   - 如果订单数量很大，需要支持分页获取
   - 当前实现只获取最新的100条

2. **数据清理**: 
   - 定期清理旧数据
   - 或使用分区表

3. **监控告警**: 
   - 集成监控系统（如Prometheus）
   - 同步失败时发送告警

4. **Web界面**: 
   - 提供Web界面查看同步状态
   - 可视化展示成交数据

5. **数据导出**: 
   - 支持导出为CSV/Excel
   - 支持生成报表

## 总结

本系统提供了一个完整的解决方案，用于在WEEX交易大赛中批量同步和管理成交订单数据。系统设计考虑了性能、可扩展性、容错性和易用性，可以满足大赛期间的数据收集和分析需求。
