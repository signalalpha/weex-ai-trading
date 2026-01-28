# WEEX订单同步服务

独立的订单同步服务，用于定时批量获取多个用户的成交订单并保存到PostgreSQL数据库。

## 特性

- ✅ **独立服务**: 与交易服务完全分离，可独立部署和运行
- ✅ **独立配置**: 使用专门的配置文件，不混入交易服务配置
- ✅ **批量同步**: 支持多个用户、多个交易对同时同步
- ✅ **增量同步**: 只同步新订单，避免重复数据
- ✅ **完整日志**: 详细的同步日志和状态记录

## 快速开始

### 1. 编译

```bash
# 编译同步服务
go build -o weex-sync cmd/sync/main.go

# 或使用Makefile（如果已配置）
make build-sync
```

### 2. 配置

复制示例配置文件：

```bash
cp configs/sync_config.example.yaml configs/sync_config.yaml
```

编辑 `configs/sync_config.yaml`：

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

### 3. 运行

```bash
# 使用默认配置文件
./weex-sync

# 指定配置文件
./weex-sync -c configs/sync_config.yaml

# 指定日志级别
./weex-sync -c configs/sync_config.yaml -l debug
```

## 配置说明

### 数据库配置

```yaml
database:
  host: "localhost"      # 数据库主机
  port: 5432            # 数据库端口
  user: "postgres"       # 数据库用户
  password: "password"   # 数据库密码
  dbname: "weex_trading" # 数据库名
  sslmode: "disable"     # SSL模式
```

也可以通过环境变量配置：
- `SYNC_DB_HOST` 或 `DB_HOST`
- `SYNC_DB_PORT` 或 `DB_PORT`
- `SYNC_DB_USER` 或 `DB_USER`
- `SYNC_DB_PASSWORD` 或 `DB_PASSWORD`
- `SYNC_DB_NAME` 或 `DB_NAME`

### 同步配置

```yaml
sync:
  interval_seconds: 60    # 同步间隔（秒）
  page_size: 100          # 每次获取的记录数
  symbols:                 # 交易对列表
    - "cmt_btcusdt"
  users:                   # 用户列表
    - user_id: "user1"
      api_key: "..."
      secret_key: "..."
      passphrase: "..."
      enabled: true
```

**注意**: 如果不配置 `users`，将使用环境变量 `WEEX_API_KEY`、`WEEX_SECRET_KEY`、`WEEX_PASSPHRASE` 作为默认用户。

## 与交易服务的区别

| 特性 | 交易服务 (trader) | 同步服务 (weex-sync) |
|------|------------------|---------------------|
| 主要功能 | 执行交易决策和下单 | 收集和存储成交数据 |
| 配置关注点 | 单个用户交易参数 | 多用户数据收集 |
| 数据库需求 | 不需要 | 必需 |
| 运行模式 | 实时交易 | 定时同步 |
| 配置文件 | `config.yaml` | `sync_config.yaml` |

## 查询数据

同步的数据保存在PostgreSQL的 `trade_fills` 表中，可以使用SQL查询：

```sql
-- 查看所有成交订单
SELECT * FROM trade_fills ORDER BY trade_time DESC LIMIT 100;

-- 查看某个用户的成交订单
SELECT * FROM trade_fills WHERE user_id = 'user1' ORDER BY trade_time DESC;

-- 查看同步状态
SELECT * FROM sync_status ORDER BY last_sync_time DESC LIMIT 20;
```

更多查询示例请参考 `cmd/trader/SYNC_README.md`。

## 故障排查

### 数据库连接失败
- 检查数据库配置是否正确
- 确认PostgreSQL服务是否运行
- 检查网络连接

### 同步失败
- 查看日志中的错误信息
- 检查API凭证是否正确
- 查看 `sync_status` 表中的错误信息

## 部署建议

### 作为系统服务

创建 systemd 服务文件 `/etc/systemd/system/weex-sync.service`:

```ini
[Unit]
Description=WEEX Order Sync Service
After=network.target postgresql.service

[Service]
Type=simple
User=your_user
WorkingDirectory=/path/to/weex-ai-trading
ExecStart=/path/to/weex-sync -c /path/to/configs/sync_config.yaml
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

启动服务：
```bash
sudo systemctl enable weex-sync
sudo systemctl start weex-sync
sudo systemctl status weex-sync
```

### Docker部署

可以创建独立的Docker容器运行同步服务，与交易服务分离。

## 更多信息

- 详细设计文档: `docs/ORDER_SYNC_DESIGN.md`
- 数据库表结构: `internal/database/schema.sql`
- 使用说明: `cmd/trader/SYNC_README.md`
