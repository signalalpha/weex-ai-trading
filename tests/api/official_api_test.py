#!/usr/bin/env python3
"""
WEEX AI Trading Hackathon - Official API Test Script
根据官方文档完成所有必需的 API 测试步骤

官方文档: https://www.weex.com/zh-CN/news/detail/ai-wars-weex-alpha-awakens-weex-global-hackathon-api-test-process-guide-266016

要求: 完成至少 10 USDT 的交易
"""

import time
import hmac
import hashlib
import base64
import requests
import json
import os
import uuid

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

# Validate that all required environment variables are set
if not api_key or not secret_key or not access_passphrase:
    raise ValueError(
        "Missing required environment variables. Please set:\n"
        "  - WEEX_API_KEY\n"
        "  - WEEX_SECRET_KEY\n"
        "  - WEEX_PASSPHRASE\n"
    )

BASE_URL = "https://api-contract.weex.com"
SYMBOL = "cmt_btcusdt"  # 官方测试交易对


def generate_signature(secret_key, timestamp, method, request_path, query_string, body=""):
    """生成 API 签名"""
    message = timestamp + method.upper() + request_path + query_string + str(body)
    signature = hmac.new(secret_key.encode(), message.encode(), hashlib.sha256).digest()
    return base64.b64encode(signature).decode()


def send_request(method, request_path, query_string="", body=None):
    """发送 API 请求"""
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
        url += query_string
    
    if method == "GET":
        response = requests.get(url, headers=headers)
    elif method == "POST":
        response = requests.post(url, headers=headers, data=body_str)
    
    return response


def print_response(step_name, response):
    """打印响应结果"""
    print(f"\n{'='*60}")
    print(f"步骤: {step_name}")
    print(f"状态码: {response.status_code}")
    print(f"响应内容:")
    try:
        data = response.json()
        print(json.dumps(data, indent=2, ensure_ascii=False))
        return data
    except:
        print(response.text)
        return None


def step1_check_domain():
    """步骤 1: 检查域名和路径"""
    print("\n[步骤 1] 检查 API 域名和路径")
    print(f"API Base URL: {BASE_URL}")
    print(f"交易对: {SYMBOL}")
    return True


def step2_check_account_balance():
    """步骤 2: 检查账户余额"""
    print("\n[步骤 2] 检查账户余额")
    request_path = "/capi/v2/account/assets"
    response = send_request("GET", request_path)
    data = print_response("检查账户余额", response)
    # 返回示例（直接是数组，没有 data 层级）:
    # [
    #   {
    #     "coinName": "USDT",
    #     "available": "5413.06877369",
    #     "equity": "5696.49288823",
    #     "frozen": "81.28240000",
    #     "unrealizePnl": "-34.55300000"
    #   }
    # ]
    if response.status_code == 200 and data:
        print(f"\n✅ 账户信息获取成功")
        # data 直接就是数组
        if isinstance(data, list):
            print(f"\n账户资产:")
            for asset in data:
                coin_name = asset.get('coinName', 'N/A')
                available = asset.get('available', '0')
                equity = asset.get('equity', '0')
                frozen = asset.get('frozen', '0')
                unrealize_pnl = asset.get('unrealizePnl', '0')
                print(f"  {coin_name}:")
                print(f"    可用余额: {available}")
                print(f"    权益: {equity}")
                print(f"    冻结: {frozen}")
                print(f"    未实现盈亏: {unrealize_pnl}")
        return data
    else:
        print(f"❌ 获取账户余额失败")
        return None


def step3_set_leverage():
    """步骤 3: 设置杠杆为 1x（全仓模式）"""
    print("\n[步骤 3] 设置杠杆")
    request_path = "/capi/v2/account/leverage"
    body = {
        "symbol": SYMBOL,
        "marginMode": 1,  # 1 = 全仓模式
        "longLeverage": "1",
        "shortLeverage": "1"
    }
    response = send_request("POST", request_path, body=body)
    data = print_response("设置杠杆 (1x, 全仓模式)", response)
    
    if response.status_code == 200:
        print(f"✅ 杠杆设置成功: 1x (全仓)")
        return True
    else:
        print(f"⚠️  杠杆设置可能失败，继续执行...")
        return False


def step4_get_asset_price():
    """步骤 4: 获取资产价格"""
    print("\n[步骤 4] 获取资产价格")
    request_path = "/capi/v2/market/ticker"
    query_string = f"?symbol={SYMBOL}"
    response = send_request("GET", request_path, query_string=query_string)
    data = print_response("获取资产价格", response)
    
    if response.status_code == 200 and data:
        # data 直接就是对象，没有 data 层级
        if isinstance(data, dict):
            last_price = data.get('last') or data.get('lastPrice')
            if last_price:
                print(f"\n✅ 当前价格: {last_price} USDT")
                return float(last_price)
    return None


def step5_place_order(price=None):
    """步骤 5: 下单（至少 10 USDT）"""
    print("\n[步骤 5] 下单")
    
    # 如果没有价格，先获取价格
    if price is None:
        price_data = step4_get_asset_price()
        if price_data:
            price = price_data
        else:
            print("❌ 无法获取价格，无法下单")
            return None
    
    # 订单数量：0.0001 BTC
    order_value = 0.0001  # 0.0001 BTC
    
    # 生成唯一的客户端订单ID
    client_oid = str(int(time.time() * 1000))
    
    request_path = "/capi/v2/order/placeOrder"
    # 比市价低 1% 下限价单
    body = {
        "symbol": SYMBOL,
        "client_oid": client_oid,
        "size": str(order_value),  # 0.0001 BTC
        "type": "1",  # 1 = 开多, 2 = 开空
        "order_type": "1",  # 0:普通，1:只做maker；2:全部成交或立即取消；3:立即成交并取消剩余
        "match_price": "0",  # 0:限价，1:市价
        "price": str(int(price))  # 价格（市价单时可能不生效，但需要提供）
    }
    
    print(f"下单参数: {json.dumps(body, indent=2, ensure_ascii=False)}")
    
    response = send_request("POST", request_path, body=body)
    data = print_response("下单", response)
    # 返回示例:
    # {
    #     "client_oid": null,
    #     "order_id": "596471064624628269"
    # }
    if response.status_code == 200 and data:
        order_id = None
        if isinstance(data, dict):
            order_id = data.get('order_id')
        
        if order_id:
            print(f"\n✅ 订单提交成功! 订单ID: {order_id}")
            return order_id
        else:
            print(f"\n⚠️  订单可能已提交，但未获取到订单ID")
            print(f"响应数据: {data}")
            return "unknown"
    else:
        print(f"❌ 下单失败")
        return None


def step6_get_current_orders():
    """步骤 6: 获取当前委托"""
    print("\n[步骤 6] 获取当前委托")
    request_path = "/capi/v2/order/current"
    query_string = f"?symbol={SYMBOL}"
    response = send_request("GET", request_path, query_string=query_string)
    data = print_response("获取当前委托", response)
    
    if response.status_code == 200:
        print(f"✅ 当前委托查询成功")
        return data
    else:
        print(f"⚠️  获取当前委托失败")
        return None


def step7_get_order_history():
    """步骤 7: 获取历史委托"""
    print("\n[步骤 7] 获取历史委托")
    request_path = "/capi/v2/order/history"
    query_string = f"?symbol={SYMBOL}&pageSize=10"
    response = send_request("GET", request_path, query_string=query_string)
    data = print_response("获取历史委托", response)
    
    if response.status_code == 200:
        print(f"✅ 历史委托查询成功")
        return data
    else:
        print(f"⚠️  获取历史委托失败")
        return None


def step8_get_trade_details():
    """步骤 8: 获取交易详情"""
    print("\n[步骤 8] 获取交易详情")
    request_path = "/capi/v2/order/fills"
    query_string = f"?symbol={SYMBOL}&pageSize=10"
    response = send_request("GET", request_path, query_string=query_string)
    data = print_response("获取交易详情", response)
    
    if response.status_code == 200:
        print(f"✅ 交易详情查询成功")
        return data
    else:
        print(f"⚠️  获取交易详情失败")
        return None


def main():
    """主测试流程"""
    print("="*60)
    print("WEEX AI Trading Hackathon - API 测试")
    print("="*60)
    print(f"\n交易对: {SYMBOL}")
    print(f"要求: 完成至少 10 USDT 的交易")
    print(f"\n开始测试...")
    
    results = {}
    
    # 步骤 1: 检查域名
    step1_check_domain()
    
    # 步骤 2: 检查账户余额
    balance_data = step2_check_account_balance()
    results['balance'] = balance_data
    
    # 步骤 3: 设置杠杆
    leverage_success = step3_set_leverage()
    results['leverage'] = leverage_success
    
    # 步骤 4: 获取价格
    price = step4_get_asset_price()
    results['price'] = price
    
    if price:
        # 步骤 5: 下单
        order_id = step5_place_order(price)
        results['order_id'] = order_id
        
        # 等待订单执行
        if order_id:
            print(f"\n等待 3 秒让订单执行...")
            time.sleep(3)
    
    # 步骤 6: 查询当前委托
    current_orders = step6_get_current_orders()
    results['current_orders'] = current_orders
    
    # 步骤 7: 查询历史委托
    history = step7_get_order_history()
    results['history'] = history
    
    # 步骤 8: 查询交易详情
    trade_details = step8_get_trade_details()
    results['trade_details'] = trade_details
    
    # 总结
    print("\n" + "="*60)
    print("测试总结")
    print("="*60)
    print(f"账户余额查询: {'✅' if balance_data else '❌'}")
    print(f"杠杆设置: {'✅' if leverage_success else '⚠️'}")
    print(f"价格获取: {'✅' if price else '❌'}")
    print(f"订单提交: {'✅' if results.get('order_id') else '❌'}")
    print(f"当前委托查询: {'✅' if current_orders else '⚠️'}")
    print(f"历史委托查询: {'✅' if history else '⚠️'}")
    print(f"交易详情查询: {'✅' if trade_details else '⚠️'}")
    
    print("\n" + "="*60)
    print("测试完成!")
    print("="*60)
    print("\n如果所有步骤都成功完成，您应该已经满足了官方要求。")
    print("请检查订单是否成功执行了至少 10 USDT 的交易。")


if __name__ == '__main__':
    main()

