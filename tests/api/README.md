# WEEX API Python 测试脚本

这是 WEEX 官方提供的 API 测试示例代码，已修改为支持从 `.env` 文件或环境变量读取 API 凭证。

## 安装依赖

首先安装所需的 Python 包：

```bash
pip install -r requirements.txt
```

或者只安装必需的包：

```bash
pip install requests python-dotenv
```

## 使用方法

### 方法 1: 使用 .env 文件（推荐）

1. 在当前目录创建 `.env` 文件：

```bash
cat > .env << EOF
WEEX_API_KEY=your_api_key
WEEX_SECRET_KEY=your_secret_key
WEEX_PASSPHRASE=your_passphrase
EOF
```

2. 运行脚本：

```bash
python3 api_testing.py
```

脚本会自动从 `.env` 文件读取配置。

### 方法 2: 使用环境变量

如果你不想使用 `.env` 文件，也可以直接在 shell 中设置环境变量：

```bash
export WEEX_API_KEY="your_api_key"
export WEEX_SECRET_KEY="your_secret_key"
export WEEX_PASSPHRASE="your_passphrase"

python3 api_testing.py
```

## 功能说明

### api_testing.py
这是官方提供的原始示例代码，包含基础的 GET 和 POST 请求示例。

### official_api_test.py ⭐ **推荐使用**
这是根据官方文档创建的完整测试脚本，包含所有必需的测试步骤：

1. ✅ 检查账户余额
2. ✅ 设置杠杆（1x，全仓模式）
3. ✅ 获取资产价格
4. ✅ 下单（至少 10 USDT）
5. ✅ 查询当前委托
6. ✅ 查询历史委托
7. ✅ 查询交易详情

**运行完整测试：**
```bash
python3 official_api_test.py
```

这将完成官方要求的所有 API 测试步骤，满足参赛资格要求。

## 工作原理

脚本会按以下顺序尝试加载配置：
1. 尝试加载当前目录的 `.env` 文件（如果安装了 `python-dotenv`）
2. 读取系统环境变量

如果没有安装 `python-dotenv`，脚本仍然可以使用环境变量，但无法自动加载 `.env` 文件。

## 安全提醒

⚠️ **重要**: 不要将 API 凭证提交到代码仓库。确保：
- `.env` 文件已添加到 `.gitignore`（已在项目中配置）
- 环境变量中的凭证不会被记录到日志
- 不要将包含凭证的代码分享到公共平台
- 定期检查和轮换 API 密钥

