# WEEX API 批量测试工具使用指南

## 功能特性

- ✅ 支持批量处理多个 API key
- ✅ 支持 Squid 代理（解决 IP 白名单问题）
- ✅ 支持 JSON 和 CSV 格式的 API key 文件
- ✅ 自动生成测试报告（JSON 格式）
- ✅ 每个 API key 独立测试，互不影响

## 安装依赖

```bash
pip install -r requirements.txt
```

## 使用方法

### 1. 准备 API Keys 文件

#### JSON 格式 (推荐)

创建 `api_keys.json` 文件：

```json
[
  {
    "api_key": "weex_xxxxxxxxxxxxxxxxxxxxxxxx",
    "secret_key": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "passphrase": "your_passphrase_here"
  },
  {
    "api_key": "weex_yyyyyyyyyyyyyyyyyyyyyyyy",
    "secret_key": "yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy",
    "passphrase": "your_passphrase_here"
  }
]
```

#### CSV 格式

创建 `api_keys.csv` 文件：

```csv
api_key,secret_key,passphrase
weex_xxxxxxxxxxxxxxxxxxxxxxxx,xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx,your_passphrase_here
weex_yyyyyyyyyyyyyyyyyyyyyyyy,yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy,your_passphrase_here
```

### 2. 运行批量测试

#### 使用代理（推荐）

```bash
python3 official_api_test_batch.py \
  --api-keys api_keys.json \
  --proxy http://proxy.example.com:3128
```

#### 不使用代理

```bash
python3 official_api_test_batch.py --api-keys api_keys.json
```

#### 使用环境变量（单个 API key）

```bash
export WEEX_API_KEY="your_api_key"
export WEEX_SECRET_KEY="your_secret_key"
export WEEX_PASSPHRASE="your_passphrase"

python3 official_api_test_batch.py --proxy http://proxy.example.com:3128
```

### 3. 查看测试结果

测试结果会自动保存到 `test_results.json`（可通过 `--output` 参数指定其他文件名）。

```bash
# 查看测试结果
cat test_results.json | python3 -m json.tool
```

结果文件包含：
- 测试时间
- 总数量、成功数量、失败数量
- 每个 API key 的详细测试结果

## 命令行参数

```
--api-keys FILE     API keys 文件路径（JSON 或 CSV 格式）
--proxy URL         代理地址，例如: http://proxy.example.com:3128
--output FILE       测试结果输出文件（默认: test_results.json）
```

## 代理配置

### Squid 代理格式

```
http://proxy.example.com:3128
```

### 带认证的代理

如果代理需要认证，格式为：

```
http://username:password@proxy.example.com:3128
```

### 环境变量方式

也可以通过环境变量设置代理：

```bash
export HTTP_PROXY="http://proxy.example.com:3128"
export HTTPS_PROXY="http://proxy.example.com:3128"
```

## 测试流程

每个 API key 会执行以下步骤：

1. ✅ 检查账户余额
2. ✅ 取消所有活跃订单（开单时无法调整杠杆）
3. ✅ 设置杠杆（20x）
4. ✅ 下限价买单（95%价格，不会立即成交）
5. ✅ 查询当前委托（确认限价单存在）
6. ✅ 下市价买单（立即成交）
7. ✅ 下市价卖单（立即成交，恢复账户状态）
8. ✅ 查询历史委托和交易详情
9. ✅ 取消限价单（清理未成交订单）
10. ✅ 最终清理（确保没有挂单）

## 注意事项

1. **IP 白名单**: 确保代理 IP 已添加到 WEEX API 白名单
2. **API Key 安全**: 不要将包含真实 API key 的文件提交到代码仓库
3. **测试顺序**: API keys 会按顺序测试，每个之间间隔 3 秒
4. **错误处理**: 如果某个 API key 测试失败，会继续测试下一个
5. **订单数量**: 每个 API key 会下单 0.005 BTC（约 10 USDT，取决于价格）

## 示例输出

```
================================================================================
测试 API Key 1/3
================================================================================
✅ 已配置代理: http://proxy.example.com:3128
============================================================
WEEX AI Trading Hackathon - API 测试
============================================================

API Key: weex_xxxxx...
交易对: cmt_btcusdt
...

测试结果: ✅ 成功
--------------------------------------------------------------------------------

等待 3 秒后继续下一个 API key...

================================================================================
批量测试完成!
================================================================================
总计: 3 个 API key
成功: 3 个
失败: 0 个
结果已保存到: test_results.json
```

## 故障排查

### 问题: 签名错误

- 检查 API key、Secret key 和 Passphrase 是否正确
- 检查代理是否正常工作

### 问题: HTTP 521 错误

- 确认代理 IP 已添加到 WEEX API 白名单
- 检查代理连接是否正常

### 问题: 连接超时

- 检查代理地址和端口是否正确
- 检查网络连接

