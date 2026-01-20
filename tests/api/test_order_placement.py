#!/usr/bin/env python3
"""
快速测试不同币种的下单接口
用于快速验证哪些币种可以正常下单，哪些会失败
"""

import time
import hmac
import hashlib
import base64
import requests
import json
import os
import sys
import argparse

# Try to load .env file if python-dotenv is available
try:
    from dotenv import load_dotenv
    load_dotenv()
except ImportError:
    pass

# Read API credentials from environment variables
api_key = os.environ.get("WEEX_API_KEY")
secret_key = os.environ.get("WEEX_SECRET_KEY")
access_passphrase = os.environ.get("WEEX_PASSPHRASE")

# Read proxy from environment variables (支持 HTTP_PROXY, HTTPS_PROXY, 或 WEEX_PROXY)
proxy_url = os.environ.get("WEEX_PROXY") or os.environ.get("HTTPS_PROXY") or os.environ.get("HTTP_PROXY")

# Validate that all required environment variables are set
if not api_key or not secret_key or not access_passphrase:
    raise ValueError(
        "Missing required environment variables. Please set:\n"
        "  - WEEX_API_KEY\n"
        "  - WEEX_SECRET_KEY\n"
        "  - WEEX_PASSPHRASE\n"
        "\n可选代理设置:\n"
        "  - WEEX_PROXY (优先) 或 HTTP_PROXY/HTTPS_PROXY\n"
    )

BASE_URL = "https://api-contract.weex.com"

# 全局代理设置（可以通过命令行参数或环境变量设置）
GLOBAL_PROXY = proxy_url


def mask_proxy_url(proxy_url: str) -> str:
    """安全地显示代理 URL，隐藏密码部分"""
    if not proxy_url or '@' not in proxy_url:
        return proxy_url  # 没有认证信息，直接返回
    try:
        # 格式: http://username:password@host:port
        protocol, rest = proxy_url.split('://', 1)
        if '@' in rest:
            auth, host_port = rest.rsplit('@', 1)
            username = auth.split(':', 1)[0] if ':' in auth else auth
            return f"{protocol}://{username}:***@{host_port}"
    except Exception:
        pass  # 解析失败，返回原 URL
    return proxy_url

# 默认测试的交易对列表（比赛官方指定的8个交易对）
DEFAULT_SYMBOLS = [
    "cmt_btcusdt",
    "cmt_ethusdt",
    "cmt_solusdt",
    "cmt_dogeusdt",
    "cmt_xrpusdt",
    "cmt_adausdt",
    "cmt_bnbusdt",
    "cmt_ltcusdt",
]

# 交易对的精度配置（根据错误信息整理）
# key: symbol, value: {"price_step": 价格步长, "size_step": 数量步长, "min_size": 最小数量}
SYMBOL_PRECISION = {
    "cmt_btcusdt": {"price_step": 0.1, "size_step": 0.001, "min_size": 0.001},  # price stepSize = 0.1
    "cmt_ethusdt": {"price_step": 0.01, "size_step": 0.001, "min_size": 0.001},
    "cmt_solusdt": {"price_step": 0.01, "size_step": 0.1, "min_size": 0.1},  # size stepSize = 0.1
    "cmt_dogeusdt": {"price_step": 0.00001, "size_step": 100, "min_size": 100},  # size stepSize = 100
    "cmt_xrpusdt": {"price_step": 0.0001, "size_step": 10, "min_size": 10},  # size stepSize = 10
    "cmt_adausdt": {"price_step": 0.0001, "size_step": 10, "min_size": 10},  # size stepSize = 10
    "cmt_bnbusdt": {"price_step": 0.01, "size_step": 0.1, "min_size": 0.1},  # size stepSize = 0.1
    "cmt_ltcusdt": {"price_step": 0.01, "size_step": 0.1, "min_size": 0.1},  # size stepSize = 0.1
}


def round_to_step(value: float, step: float) -> float:
    """将值四舍五入到指定步长"""
    if step <= 0:
        return value
    return round(value / step) * step


def adjust_price_to_precision(price: float, symbol: str) -> float:
    """根据交易对的精度调整价格"""
    precision = SYMBOL_PRECISION.get(symbol, {"price_step": 0.01})
    price_step = precision["price_step"]
    return round_to_step(price, price_step)


def adjust_size_to_precision(size: float, symbol: str) -> float:
    """根据交易对的精度调整数量，并确保不小于最小值"""
    precision = SYMBOL_PRECISION.get(symbol, {"size_step": 0.001, "min_size": 0.001})
    size_step = precision["size_step"]
    min_size = precision["min_size"]
    
    # 先调整到步长
    adjusted_size = round_to_step(size, size_step)
    
    # 确保不小于最小值
    if adjusted_size < min_size:
        adjusted_size = min_size
    
    return adjusted_size


def format_price(price: float, symbol: str) -> str:
    """根据交易对的精度格式化价格字符串"""
    precision = SYMBOL_PRECISION.get(symbol, {"price_step": 0.01})
    price_step = precision["price_step"]
    
    # 根据步长确定小数位数
    if price_step >= 1:
        # 步长 >= 1，使用整数
        return str(int(price))
    elif price_step >= 0.1:
        # 步长 >= 0.1，保留1位小数
        return f"{price:.1f}"
    elif price_step >= 0.01:
        # 步长 >= 0.01，保留2位小数
        return f"{price:.2f}"
    elif price_step >= 0.001:
        # 步长 >= 0.001，保留3位小数
        return f"{price:.3f}"
    elif price_step >= 0.0001:
        # 步长 >= 0.0001，保留4位小数
        return f"{price:.4f}"
    elif price_step >= 0.00001:
        # 步长 >= 0.00001，保留5位小数
        return f"{price:.5f}"
    else:
        # 步长更小，保留更多小数位
        decimals = len(str(price_step).rstrip('0').split('.')[-1]) if '.' in str(price_step) else 0
        return f"{price:.{decimals}f}"


def generate_signature(secret_key, timestamp, method, request_path, query_string, body=""):
    """生成 API 签名"""
    message = timestamp + method.upper() + request_path + query_string + str(body)
    signature = hmac.new(secret_key.encode(), message.encode(), hashlib.sha256).digest()
    return base64.b64encode(signature).decode()


def send_request(method, request_path, query_string="", body=None, verbose=False, proxy=None):
    """
    发送 API 请求
    
    Args:
        method: HTTP 方法 (GET/POST)
        request_path: 请求路径
        query_string: 查询字符串
        body: 请求体（字典）
        verbose: 是否打印详细信息
        proxy: 代理URL（如果为None，使用全局代理设置）
    
    Returns:
        response 对象，如果 verbose=True，返回 (response, request_info) 元组
    """
    timestamp = str(int(time.time() * 1000))
    body_str = json.dumps(body) if body else ""
    
    signature = generate_signature(secret_key, timestamp, method, request_path, query_string, body_str)
    
    headers = {
        "ACCESS-KEY": api_key,
        "ACCESS-SIGN": signature,
        "ACCESS-TIMESTAMP": timestamp,
        "ACCESS-PASSPHRASE": access_passphrase,
        "Content-Type": "application/json",
        "locale": "zh-CN"
    }
    
    url = BASE_URL + request_path
    if query_string:
        if query_string.startswith("?"):
            url += query_string
        else:
            url += "?" + query_string
    
    # 确定使用的代理（优先使用传入的proxy参数，否则使用全局代理）
    use_proxy = proxy if proxy is not None else GLOBAL_PROXY
    
    # 配置代理（requests 库原生支持带认证的代理 URL）
    proxies = None
    if use_proxy:
        proxies = {
            'http': use_proxy,   # 同时设置 HTTP 和 HTTPS
            'https': use_proxy,  # requests 会自动通过 HTTP CONNECT 方法处理 HTTPS
        }
    
    request_info = {
        "method": method,
        "url": url,
        "endpoint": request_path,
        "headers": headers.copy(),
        "body": body,
        "body_str": body_str,
        "query_string": query_string,
        "proxy": mask_proxy_url(use_proxy) if use_proxy else None
    }
    
    if verbose:
        # 隐藏敏感信息（签名、密钥等）但保留用于验证
        safe_headers = headers.copy()
        # 保留签名但标记（签名是必要的，官方可能需要验证）
        request_info["headers"] = safe_headers
    
    if method == "GET":
        response = requests.get(url, headers=headers, proxies=proxies)
    elif method == "POST":
        response = requests.post(url, headers=headers, data=body_str, proxies=proxies)
    
    if verbose:
        return response, request_info
    return response


def get_symbol_price(symbol):
    """获取交易对的当前价格"""
    try:
        request_path = "/capi/v2/market/ticker"
        query_string = f"?symbol={symbol}"
        # get_symbol_price 不需要详细输出，使用普通模式
        response = send_request("GET", request_path, query_string=query_string, verbose=False)
        
        if response.status_code == 200:
            data = response.json()
            if isinstance(data, dict):
                last_price = data.get('last') or data.get('lastPrice')
                if last_price:
                    return float(last_price)
        return None
    except Exception as e:
        print(f"    ⚠️  获取价格失败: {e}")
        return None


def test_place_order(symbol, price=None, size=None, order_type="限价买单测试", side="1", order_type_flag="1", match_price="0", proxy=None):
    """
    测试下单接口
    
    Args:
        symbol: 交易对符号
        price: 价格（如果为None，则尝试获取当前价格并设置为95%）
        size: 订单数量（如果为None，使用默认小数量）
        order_type: 订单类型描述
        side: "1"=开多(买), "2"=开空(卖)
        order_type_flag: "0"=普通, "1"=只做maker, "2"=全部成交或立即取消, "3"=立即成交并取消剩余
        match_price: "0"=限价, "1"=市价
    """
    # 如果价格为None，尝试获取当前价格
    if price is None:
        print(f"    获取 {symbol} 当前价格...")
        current_price = get_symbol_price(symbol)
        if current_price:
            # 限价单设置为当前价格的95%，确保不会立即成交
            price = current_price * 0.95
        else:
            print(f"    ❌ 无法获取价格，跳过下单测试")
            return {"success": False, "error": "无法获取价格"}
    
    # 保存原始价格和数量
    original_price = price
    
    # 根据交易对精度调整价格
    price = adjust_price_to_precision(price, symbol)
    if abs(price) < 1e-10:  # 价格接近0或为0
        print(f"    ⚠️  价格调整后为0（原始: {original_price}），可能步长设置不正确")
        return {"success": False, "error": f"价格调整后为0，原始价格: {original_price}"}
    
    # 如果数量为None，使用该交易对的最小数量
    if size is None:
        precision = SYMBOL_PRECISION.get(symbol, {"min_size": 0.001})
        size = precision["min_size"]
    
    # 保存原始数量
    original_size = size
    
    # 根据交易对精度调整数量
    size = adjust_size_to_precision(size, symbol)
    
    client_oid = str(int(time.time() * 1000))
    
    request_path = "/capi/v2/order/placeOrder"
    body = {
        "symbol": symbol,
        "client_oid": client_oid,
        "size": str(size),
        "type": side,
        "order_type": order_type_flag,
        "match_price": match_price,
        "price": format_price(price, symbol)  # 根据精度格式化价格
    }
    
    print(f"\n    {'-'*70}")
    print(f"    📋 下单请求详情")
    print(f"    {'-'*70}")
    print(f"    交易对: {symbol}")
    precision = SYMBOL_PRECISION.get(symbol, {})
    price_step = precision.get("price_step", "未知")
    size_step = precision.get("size_step", "未知")
    print(f"    价格精度: stepSize={price_step}")
    print(f"    数量精度: stepSize={size_step}")
    print(f"    价格: {price} (原始值: {original_price}, 调整后: {price})")
    print(f"    数量: {size} (原始值: {original_size}, 调整后: {size})")
    print(f"    订单类型: {order_type}")
    print(f"    方向: {'开多(买)' if side == '1' else '开空(卖)'}")
    print(f"    限价/市价: {'限价' if match_price == '0' else '市价'}")
    
    try:
        # 使用 verbose 模式获取请求信息（传入代理参数）
        response, request_info = send_request("POST", request_path, body=body, verbose=True, proxy=proxy)
        
        # 打印请求信息
        print(f"\n    🔗 请求端点 (Endpoint):")
        print(f"        {request_info['method']} {request_info['url']}")
        print(f"        路径: {request_info['endpoint']}")
        
        print(f"\n    📤 请求参数 (Request Parameters):")
        print(f"        Body (JSON):")
        print(f"        {json.dumps(request_info['body'], indent=10, ensure_ascii=False)}")
        
        print(f"\n    🔑 请求头 (Request Headers):")
        # 打印请求头，但隐藏敏感信息的值（只显示字段名和长度）
        safe_headers = {}
        for key, value in request_info['headers'].items():
            if key in ['ACCESS-KEY', 'ACCESS-SIGN', 'ACCESS-PASSPHRASE']:
                safe_headers[key] = f"[已设置, 长度: {len(str(value))}]"
            else:
                safe_headers[key] = value
        print(f"        {json.dumps(safe_headers, indent=10, ensure_ascii=False)}")
        
        # 打印完整的原始请求信息（用于调试和官方沟通）
        print(f"\n    🔍 完整请求信息 (用于与官方沟通，含完整签名):")
        print(f"        【完整URL】")
        print(f"        {request_info['url']}")
        print(f"\n        【HTTP方法】")
        print(f"        {request_info['method']}")
        if request_info.get('proxy'):
            print(f"\n        【代理 (Proxy)】")
            print(f"        {request_info['proxy']}")
        print(f"\n        【请求头 (完整，含签名)】")
        for key, value in request_info['headers'].items():
            print(f"        {key}: {value}")
        print(f"\n        【请求体 (原始JSON字符串)】")
        print(f"        {request_info['body_str']}")
        print(f"\n        【请求体 (格式化JSON)】")
        print(f"        {json.dumps(request_info['body'], indent=8, ensure_ascii=False)}")
        
        # 打印响应信息
        print(f"\n    📥 响应信息 (Response):")
        print(f"        【HTTP 状态码】")
        print(f"        {response.status_code} {response.reason}")
        print(f"\n        【响应头 (Response Headers)】")
        for key, value in response.headers.items():
            print(f"        {key}: {value}")
        
        result = {
            "symbol": symbol,
            "status_code": response.status_code,
            "success": False,
            "order_id": None,
            "error": None,
            "response_data": None,
            "request_info": request_info
        }
        
        try:
            data = response.json()
            result["response_data"] = data
            
            print(f"\n        【响应体 (Response Body - JSON)】")
            print(f"        {json.dumps(data, indent=8, ensure_ascii=False)}")
            print(f"\n        【响应体 (原始文本)】")
            print(f"        {response.text}")
            
            if response.status_code == 200 and data:
                order_id = None
                if isinstance(data, dict):
                    order_id = data.get('order_id') or data.get('orderId')
                
                if order_id:
                    result["success"] = True
                    result["order_id"] = order_id
                else:
                    result["error"] = "响应中未找到订单ID"
            else:
                # 尝试从响应中提取错误信息
                if isinstance(data, dict):
                    result["error"] = data.get('msg') or data.get('message') or data.get('error') or str(data)
                else:
                    result["error"] = str(data) if data else f"HTTP {response.status_code}"
        except json.JSONDecodeError:
            response_text = response.text
            print(f"\n        【响应体 (Response Body - 非JSON，原始文本)】")
            print(f"        {response_text}")
            result["error"] = f"响应不是有效的JSON: {response_text[:200]}"
        
        print(f"    {'-'*70}\n")
        
        return result
        
    except Exception as e:
        return {
            "symbol": symbol,
            "success": False,
            "error": f"请求异常: {str(e)}",
            "response_data": None
        }


def cancel_order(symbol, order_id):
    """取消订单"""
    try:
        request_path = "/capi/v2/order/cancel_order"
        body = {
            "orderId": str(order_id)
        }
        # cancel_order 不需要详细输出，使用普通模式
        response = send_request("POST", request_path, body=body, verbose=False)
        return response.status_code == 200
    except:
        return False


def test_symbols(symbols, cancel_orders=True, proxy=None):
    """
    批量测试多个交易对的下单功能
    
    Args:
        symbols: 交易对列表
        cancel_orders: 是否在下单后立即取消订单（清理）
        proxy: 代理URL（如果为None，使用全局代理设置）
    """
    global GLOBAL_PROXY  # 在函数开头声明 global
    
    print("=" * 80)
    print("快速下单接口测试")
    print("=" * 80)
    
    # 确定使用的代理（优先使用传入的proxy参数，否则使用全局代理）
    use_proxy = proxy if proxy is not None else GLOBAL_PROXY
    if use_proxy:
        print(f"\n🌐 使用代理: {mask_proxy_url(use_proxy)}")
        # 更新全局代理设置
        GLOBAL_PROXY = use_proxy
    
    print(f"\n测试交易对数量: {len(symbols)}")
    print(f"交易对列表: {', '.join(symbols)}")
    print(f"\n开始测试...\n")
    
    results = []
    success_count = 0
    fail_count = 0
    
    for i, symbol in enumerate(symbols, 1):
        print(f"[{i}/{len(symbols)}] 测试 {symbol}...")
        
        # 测试下单（传入代理参数）
        result = test_place_order(
            symbol=symbol,
            price=None,  # 自动获取价格
            size=None,   # 使用默认小数量
            order_type="限价买单测试",
            side="1",
            order_type_flag="1",  # 只做maker，确保不会立即成交
            match_price="0",  # 限价单
            proxy=use_proxy  # 传递代理参数
        )
        
        if result["success"]:
            print(f"    ✅ 下单成功! 订单ID: {result['order_id']}")
            success_count += 1
            
            # 如果设置了取消订单，尝试取消
            if cancel_orders and result["order_id"]:
                print(f"    取消订单 {result['order_id']}...")
                if cancel_order(symbol, result["order_id"]):
                    print(f"    ✅ 订单已取消")
                else:
                    print(f"    ⚠️  订单取消失败（可能需要手动取消）")
        else:
            print(f"    ❌ 下单失败: {result.get('error', '未知错误')}")
            fail_count += 1
            if result.get("response_data"):
                print(f"    响应数据: {json.dumps(result['response_data'], ensure_ascii=False, indent=6)}")
        
        results.append(result)
        
        # 稍微延迟，避免请求过快
        if i < len(symbols):
            time.sleep(0.5)
    
    # 打印总结
    print("\n" + "=" * 80)
    print("测试总结")
    print("=" * 80)
    print(f"总测试数: {len(symbols)}")
    print(f"成功: {success_count} ✅")
    print(f"失败: {fail_count} ❌")
    
    if fail_count > 0:
        print(f"\n失败的交易对:")
        for result in results:
            if not result["success"]:
                error_msg = result.get("error", "未知错误")
                print(f"  - {result['symbol']}: {error_msg}")
    
    print("\n" + "=" * 80)
    
    return results


def main():
    parser = argparse.ArgumentParser(
        description="快速测试不同币种的下单接口",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
示例:
  # 测试默认交易对列表
  python test_order_placement.py

  # 测试指定的交易对
  python test_order_placement.py -s cmt_btcusdt cmt_ethusdt

  # 从文件读取交易对列表（每行一个）
  python test_order_placement.py -f symbols.txt

  # 测试后不取消订单
  python test_order_placement.py --no-cancel

  # 使用代理测试
  python test_order_placement.py --proxy http://proxy.example.com:3128

  # 使用带认证的代理测试
  python test_order_placement.py --proxy http://username:password@proxy.example.com:3128
        """
    )
    
    parser.add_argument(
        "-s", "--symbols",
        nargs="+",
        help="指定要测试的交易对列表（例如: cmt_btcusdt cmt_ethusdt）"
    )
    
    parser.add_argument(
        "-f", "--file",
        help="从文件读取交易对列表（每行一个）"
    )
    
    parser.add_argument(
        "--no-cancel",
        action="store_true",
        help="下单后不自动取消订单"
    )
    
    parser.add_argument(
        "--proxy",
        help="代理地址，例如: http://proxy.example.com:3128 或 http://username:password@proxy.example.com:3128"
    )
    
    args = parser.parse_args()
    
    # 确定要测试的交易对列表
    symbols = []
    
    if args.file:
        # 从文件读取
        try:
            with open(args.file, 'r', encoding='utf-8') as f:
                symbols = [line.strip() for line in f if line.strip() and not line.startswith('#')]
            print(f"从文件 {args.file} 读取了 {len(symbols)} 个交易对")
        except Exception as e:
            print(f"❌ 读取文件失败: {e}")
            sys.exit(1)
    elif args.symbols:
        # 从命令行参数读取
        symbols = args.symbols
    else:
        # 使用默认列表
        symbols = DEFAULT_SYMBOLS
    
    if not symbols:
        print("❌ 没有指定要测试的交易对")
        sys.exit(1)
    
    # 确定使用的代理（命令行参数优先于环境变量）
    global GLOBAL_PROXY  # 在函数开头声明 global
    proxy = args.proxy if args.proxy else None
    if proxy:
        # 更新全局代理设置
        GLOBAL_PROXY = proxy
    
    # 执行测试
    results = test_symbols(symbols, cancel_orders=not args.no_cancel, proxy=proxy)
    
    # 返回适当的退出码
    if all(r["success"] for r in results):
        sys.exit(0)
    else:
        sys.exit(1)


if __name__ == '__main__':
    main()
