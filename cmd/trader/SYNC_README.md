# 订单同步服务使用说明

## 功能概述

订单同步服务用于定时批量获取多个用户的成交订单，并保存到PostgreSQL数据库中，供后续分析和查询使用。

## 主要特性

- ✅ 支持批量用户同步
- ✅ 定时自动同步（可配置间隔）
- ✅ 增量同步（只同步新订单，避免重复）
- ✅ 同步状态记录（记录每次同步的结果）
- ✅ 错误处理和重试机制
- ✅ 完整的日志记录

## 数据库表结构

### trade_fills（成交订单表）
存储所有成交订单的详细信息：
- `user_id`: 用户标识
- `order_id`: 订单ID
- `trade_id`: 成交ID
- `symbol`: 交易对
- `side`: 交易方向（"1"=买, "2"=卖）
- `price`: 成交价格
- `size`: 成交数量
- `fee`: 手续费
- `fee_coin`: 手续费币种
- `trade_time`: 成交时间（毫秒时间戳）

### sync_status（同步状态表）
记录每次同步的状态：
- `user_id`: 用户标识
- `symbol`: 交易对
- `last_sync_time`: 最后同步时间
- `last_trade_time`: 最后同步的成交时间戳
- `records_count`: 本次同步的记录数
- `status`: 同步状态（success/error）
- `error_message`: 错误信息（如果有）

## 配置说明

### 数据库配置

在 `config.yaml` 中配置PostgreSQL连接信息：

```yaml
database:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "your_password"
  dbname: "weex_trading"
  sslmode: "disable"
```

也可以通过环境变量配置：
- `DB_HOST` 或 `POSTGRES_HOST`
- `DB_PORT` 或 `POSTGRES_PORT`
- `DB_USER` 或 `POSTGRES_USER`
- `DB_PASSWORD` 或 `POSTGRES_PASSWORD`
- `DB_NAME` 或 `POSTGRES_DB`
- `DB_SSLMODE`

### 同步服务配置

```yaml
sync:
  interval_seconds: 60    # 同步间隔（秒）
  page_size: 100          # 每次获取的记录数
  symbols:                # 要同步的交易对
    - "cmt_btcusdt"
  users:                  # 用户列表（可选）
    - user_id: "user1"
      api_key: "..."
      secret_key: "..."
      passphrase: "..."
      enabled: true
```

如果不配置 `users`，将使用默认的WEEX配置作为单个用户。

## 使用方法

### 1. 初始化数据库

首先需要创建数据库和表结构：

```bash
# 连接到PostgreSQL
psql -U postgres

# 创建数据库
CREATE DATABASE weex_trading;

# 使用数据库
\c weex_trading

# 执行SQL脚本创建表（可选，服务会自动创建）
\i internal/database/schema.sql
```

### 2. 配置服务

编辑 `configs/config.yaml`，配置数据库和同步参数。

### 3. 启动同步服务

```bash
# 使用配置文件
./trader sync -c configs/config.yaml

# 或使用环境变量
export DB_HOST=localhost
export DB_USER=postgres
export DB_PASSWORD=your_password
export DB_NAME=weex_trading
./trader sync
```

### 4. 查看日志

服务会输出详细的同步日志，包括：
- 每次同步的开始和结束
- 每个用户和交易对的同步状态
- 获取和保存的记录数
- 错误信息（如果有）

## 查询数据示例

### 查询某个用户的所有成交订单

```sql
SELECT * FROM trade_fills 
WHERE user_id = 'user1' 
ORDER BY trade_time DESC 
LIMIT 100;
```

### 查询某个交易对的成交订单

```sql
SELECT * FROM trade_fills 
WHERE symbol = 'cmt_btcusdt' 
ORDER BY trade_time DESC;
```

### 查询某个时间段的成交订单

```sql
SELECT * FROM trade_fills 
WHERE user_id = 'user1' 
  AND symbol = 'cmt_btcusdt'
  AND trade_time >= 1704067200000  -- 开始时间（毫秒时间戳）
  AND trade_time <= 1704153600000  -- 结束时间（毫秒时间戳）
ORDER BY trade_time DESC;
```

### 统计每个用户的成交数量

```sql
SELECT 
  user_id,
  symbol,
  COUNT(*) as trade_count,
  SUM(size::numeric) as total_size,
  SUM(fee::numeric) as total_fee
FROM trade_fills
GROUP BY user_id, symbol
ORDER BY trade_count DESC;
```

### 查看同步状态

```sql
SELECT * FROM sync_status 
ORDER BY last_sync_time DESC 
LIMIT 20;
```

## 注意事项

1. **API限流**: 注意WEEX API的调用频率限制，合理设置同步间隔
2. **数据库性能**: 如果数据量很大，建议定期清理旧数据或使用分区表
3. **错误处理**: 如果某个用户的同步失败，不会影响其他用户的同步
4. **增量同步**: 服务会自动记录最后同步的时间，只获取新订单，避免重复
5. **唯一约束**: 数据库使用 `(user_id, order_id, trade_id, trade_time)` 作为唯一约束，避免重复插入

## 故障排查

### 数据库连接失败
- 检查数据库配置是否正确
- 确认PostgreSQL服务是否运行
- 检查网络连接和防火墙设置

### 同步失败
- 查看日志中的错误信息
- 检查API凭证是否正确
- 确认网络连接是否正常
- 查看 `sync_status` 表中的错误信息

### 没有新数据
- 确认用户是否有新的成交订单
- 检查 `last_trade_time` 是否正确
- 查看API返回的数据是否为空
