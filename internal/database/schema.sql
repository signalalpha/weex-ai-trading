-- WEEX交易大赛成交订单表
-- 用于存储批量用户的成交订单数据，供分析查询

CREATE TABLE IF NOT EXISTS trade_fills (
    id BIGSERIAL PRIMARY KEY,
    
    -- 用户标识（API Key的前几位或自定义标识）
    user_id VARCHAR(100) NOT NULL,
    
    -- 订单信息
    order_id VARCHAR(100) NOT NULL,
    trade_id VARCHAR(100),
    symbol VARCHAR(50) NOT NULL,
    
    -- 交易方向: "1"=买, "2"=卖
    side VARCHAR(10) NOT NULL,
    
    -- 成交信息
    price DECIMAL(30, 8) NOT NULL,
    size DECIMAL(30, 8) NOT NULL,
    fee DECIMAL(30, 8) NOT NULL,
    fee_coin VARCHAR(20) NOT NULL,
    
    -- 时间戳
    trade_time BIGINT NOT NULL,  -- 毫秒时间戳
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- 索引
    CONSTRAINT unique_trade_fill UNIQUE (user_id, order_id, trade_id, trade_time)
);

-- 创建索引以优化查询性能
CREATE INDEX IF NOT EXISTS idx_trade_fills_user_id ON trade_fills(user_id);
CREATE INDEX IF NOT EXISTS idx_trade_fills_symbol ON trade_fills(symbol);
CREATE INDEX IF NOT EXISTS idx_trade_fills_trade_time ON trade_fills(trade_time);
CREATE INDEX IF NOT EXISTS idx_trade_fills_created_at ON trade_fills(created_at);
CREATE INDEX IF NOT EXISTS idx_trade_fills_user_symbol_time ON trade_fills(user_id, symbol, trade_time DESC);

-- 用户配置表（可选，用于管理多个用户的API凭证）
CREATE TABLE IF NOT EXISTS user_configs (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(100) UNIQUE NOT NULL,
    api_key VARCHAR(200) NOT NULL,
    secret_key VARCHAR(200) NOT NULL,
    passphrase VARCHAR(200) NOT NULL,
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 同步状态表（记录每次同步的状态）
CREATE TABLE IF NOT EXISTS sync_status (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(100) NOT NULL,
    symbol VARCHAR(50) NOT NULL,
    last_sync_time TIMESTAMP NOT NULL,
    last_trade_time BIGINT,  -- 最后同步的成交时间戳
    records_count INT DEFAULT 0,  -- 本次同步的记录数
    status VARCHAR(20) DEFAULT 'success',  -- success, error
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sync_status_user_symbol ON sync_status(user_id, symbol);
CREATE INDEX IF NOT EXISTS idx_sync_status_last_sync_time ON sync_status(last_sync_time DESC);
