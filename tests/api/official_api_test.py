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
        # 如果 query_string 已经包含 ?，直接拼接；否则添加 ?
        if query_string.startswith("?"):
            url += query_string
        else:
            url += "?" + query_string
    
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


def step2_5_cancel_all_active_orders():
    """步骤 2.5: 取消所有活跃订单（开单时无法调整杠杆）"""
    print("\n[步骤 2.5] 检查并取消所有活跃订单")
    
    # 获取当前委托（使用与 step9 相同的方式）
    request_path = "/capi/v2/order/current"
    query_string = f"?symbol={SYMBOL}"  # 与 step9 保持一致，包含 ?
    response = send_request("GET", request_path, query_string=query_string)
    data = print_response("获取当前委托", response)
    
    # 如果 HTTP 状态码不是 200，才是真正的失败
    if response.status_code != 200:
        print(f"⚠️  获取当前委托失败（状态码: {response.status_code}），跳过取消订单")
        return 0
    
    # HTTP 200 但数据为空是正常情况（没有活跃订单），按成功处理
    if not data:
        print(f"✅ 没有活跃订单，无需取消")
        return 0
    
    # 解析订单列表
    orders = []
    if isinstance(data, list):
        orders = data if len(data) > 0 else []
    elif isinstance(data, dict) and 'data' in data:
        orders = data['data'] if isinstance(data['data'], list) and len(data['data']) > 0 else []
    elif isinstance(data, dict) and 'list' in data:
        orders = data['list'] if isinstance(data['list'], list) and len(data['list']) > 0 else []
    
    if not orders:
        print(f"✅ 没有活跃订单，无需取消")
        return 0
    
    print(f"\n发现 {len(orders)} 个活跃订单，开始取消...")
    
    cancelled_count = 0
    for order in orders:
        order_id = None
        # 尝试不同的订单ID字段名
        if 'orderId' in order:
            order_id = order['orderId']
        elif 'order_id' in order:
            order_id = order['order_id']
        elif 'id' in order:
            order_id = order['id']
        
        if not order_id:
            print(f"⚠️  订单缺少ID字段，跳过: {order}")
            continue
        
        print(f"  取消订单: {order_id}")
        
        # 取消订单
        cancel_path = "/capi/v2/order/cancel_order"
        cancel_body = {
            "orderId": str(order_id)
        }
        cancel_response = send_request("POST", cancel_path, body=cancel_body)
        
        if cancel_response.status_code == 200:
            print(f"    ✅ 订单 {order_id} 取消成功")
            cancelled_count += 1
        else:
            print(f"    ⚠️  订单 {order_id} 取消失败: {cancel_response.status_code}")
            try:
                error_data = cancel_response.json()
                print(f"    错误信息: {json.dumps(error_data, ensure_ascii=False)}")
            except:
                print(f"    错误信息: {cancel_response.text}")
        
        # 稍微延迟，避免请求过快
        time.sleep(0.2)
    
    print(f"\n✅ 成功取消 {cancelled_count}/{len(orders)} 个订单")
    
    if cancelled_count > 0:
        print(f"\n等待 2 秒确保订单取消完成...")
        time.sleep(2)
    
    return cancelled_count


def step3_set_leverage():
    """步骤 3: 设置杠杆为 20x（全仓模式）"""
    print("\n[步骤 3] 设置杠杆")
    request_path = "/capi/v2/account/leverage"
    body = {
        "symbol": SYMBOL,
        "marginMode": 1,  # 1 = 全仓模式
        "longLeverage": "20",
        "shortLeverage": "20"
    }
    response = send_request("POST", request_path, body=body)
    data = print_response("设置杠杆 (20x, 全仓模式)", response)
    
    if response.status_code == 200:
        print(f"✅ 杠杆设置成功: 20x (全仓)")
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


def place_order(price, size, order_type, side="1", order_type_flag="0", match_price="0"):
    """
    下单函数
    
    Args:
        price: 价格（限价单使用，市价单时也需要提供但可能不生效）
        size: 订单数量
        order_type: 订单类型描述（用于日志）
        side: "1"=开多(买), "2"=开空(卖)
        order_type_flag: "0"=普通, "1"=只做maker, "2"=全部成交或立即取消, "3"=立即成交并取消剩余
        match_price: "0"=限价, "1"=市价
    """
    client_oid = str(int(time.time() * 1000))
    
    request_path = "/capi/v2/order/placeOrder"
    body = {
        "symbol": SYMBOL,
        "client_oid": client_oid,
        "size": str(size),
        "type": side,  # "1" = 开多(买), "2" = 开空(卖)
        "order_type": order_type_flag,
        "match_price": match_price,  # "0"=限价, "1"=市价
        "price": str(int(price))
    }
    
    print(f"下单参数: {json.dumps(body, indent=2, ensure_ascii=False)}")
    
    response = send_request("POST", request_path, body=body)
    data = print_response(f"下单 ({order_type})", response)
    
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


def step5_place_limit_buy_order(price):
    """步骤 5: 下限价买单（市价5%以下，确保能查询到当前委托）"""
    print("\n[步骤 5] 下限价买单（市价5%以下）")
    
    # 计算限价：当前价格的95%（比市价低5%）
    limit_price = price * 0.95
    order_size = 0.005  # 0.005 BTC
    
    print(f"当前价格: {price} USDT")
    print(f"限价: {limit_price} USDT (95% of current price)")
    print(f"订单数量: {order_size} BTC")
    
    order_id = place_order(
        price=limit_price,
        size=order_size,
        order_type="限价买单",
        side="1",  # 买
        order_type_flag="1",  # 只做maker
        match_price="0"  # 限价
    )
    
    return order_id


def step6_place_market_buy_order(price):
    """步骤 6: 下市价买单（确保成交后能查到交易历史）"""
    print("\n[步骤 6] 下市价买单")
    
    order_size = 0.005  # 0.005 BTC
    print(f"当前价格: {price} USDT")
    print(f"订单数量: {order_size} BTC")
    
    order_id = place_order(
        price=price,
        size=order_size,
        order_type="市价买单",
        side="1",  # 买
        order_type_flag="0",  # 普通
        match_price="1"  # 市价
    )
    
    return order_id


def step7_place_market_sell_order(price):
    """步骤 7: 下市价卖单（平仓）"""
    print("\n[步骤 7] 下市价卖单（平仓）")
    
    order_size = 0.005  # 0.005 BTC
    print(f"当前价格: {price} USDT")
    print(f"订单数量: {order_size} BTC")
    
    order_id = place_order(
        price=price,
        size=order_size,
        order_type="市价卖单",
        side="2",  # 卖
        order_type_flag="0",  # 普通
        match_price="1"  # 市价
    )
    
    return order_id


def step8_cancel_order(order_id):
    """步骤 8: 取消订单"""
    print(f"\n[步骤 8] 取消订单 (订单ID: {order_id})")
    
    request_path = "/capi/v2/order/cancel_order"
    body = {
        "orderId": order_id
    }
    
    response = send_request("POST", request_path, body=body)
    data = print_response("取消订单", response)
    
    if response.status_code == 200:
        print(f"\n✅ 订单取消成功")
        return True
    else:
        print(f"\n⚠️  订单取消失败")
        return False


def step9_get_current_orders():
    """步骤 9: 获取当前委托"""
    print("\n[步骤 9] 获取当前委托")
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


def step10_get_order_history():
    """步骤 10: 获取历史委托"""
    print("\n[步骤 10] 获取历史委托")
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


def step11_get_trade_details():
    """步骤 11: 获取交易详情"""
    print("\n[步骤 11] 获取交易详情")
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
    print(f"测试流程: 检查余额 -> 取消活跃订单 -> 设置杠杆 -> 限价买单 -> 查询当前委托 -> 市价买单 -> 市价卖单 -> 查询历史 -> 取消限价单 -> 最终清理")
    print(f"\n开始测试...")
    
    results = {}
    
    # 步骤 1: 检查域名
    step1_check_domain()
    
    # 步骤 2: 检查账户余额
    balance_data = step2_check_account_balance()
    results['balance'] = balance_data
    
    # 步骤 2.5: 取消所有活跃订单（开单时无法调整杠杆）
    cancelled_count_before = step2_5_cancel_all_active_orders()
    results['cancelled_before'] = cancelled_count_before
    
    # 步骤 3: 设置杠杆
    leverage_success = step3_set_leverage()
    results['leverage'] = leverage_success
    
    # 步骤 4: 获取价格
    price = step4_get_asset_price()
    results['price'] = price
    
    limit_order_id = None
    
    if price:
        # 步骤 5: 下限价买单（市价5%以下，确保能查询到当前委托）
        limit_order_id = step5_place_limit_buy_order(price)
        results['limit_order_id'] = limit_order_id
        
        if limit_order_id:
            print(f"\n等待 2 秒...")
            time.sleep(2)
            
            # 步骤 6: 查询当前委托（确认限价单存在）
            current_orders = step9_get_current_orders()
            results['current_orders'] = current_orders
            
            print(f"\n等待 2 秒...")
            time.sleep(2)
            
            # 步骤 7: 下市价买单（确保成交）
            market_buy_order_id = step6_place_market_buy_order(price)
            results['market_buy_order_id'] = market_buy_order_id
            
            if market_buy_order_id:
                print(f"\n等待 3 秒让市价买单成交...")
                time.sleep(3)
                
                # 步骤 8: 下市价卖单（平仓，恢复账户状态）
                market_sell_order_id = step7_place_market_sell_order(price)
                results['market_sell_order_id'] = market_sell_order_id
                
                if market_sell_order_id:
                    print(f"\n等待 3 秒让市价卖单成交...")
                    time.sleep(3)
                    
                    # 步骤 9: 查询历史委托（确认市价单已成交）
                    history = step10_get_order_history()
                    results['history'] = history
                    
                    # 步骤 10: 查询交易详情
                    trade_details = step11_get_trade_details()
                    results['trade_details'] = trade_details
                    
                    print(f"\n等待 1 秒...")
                    time.sleep(1)
                    
                    # 步骤 11: 取消限价单（清理未成交的限价单）
                    if limit_order_id and limit_order_id != "unknown":
                        cancel_success = step8_cancel_order(limit_order_id)
                        results['cancel_success'] = cancel_success
    
    # 步骤 12: 最终清理 - 再次检查并取消所有活跃订单
    print("\n" + "="*60)
    print("[最终清理] 检查并取消所有活跃订单，确保账户干净")
    print("="*60)
    cancelled_count_after = step2_5_cancel_all_active_orders()
    results['cancelled_after'] = cancelled_count_after
    
    # 总结
    print("\n" + "="*60)
    print("测试总结")
    print("="*60)
    print(f"账户余额查询: {'✅' if balance_data else '❌'}")
    print(f"初始清理订单数: {cancelled_count_before}")
    print(f"杠杆设置: {'✅' if leverage_success else '⚠️'}")
    print(f"价格获取: {'✅' if price else '❌'}")
    print(f"限价买单: {'✅' if results.get('limit_order_id') else '❌'}")
    print(f"当前委托查询: {'✅' if results.get('current_orders') else '⚠️'}")
    print(f"市价买单: {'✅' if results.get('market_buy_order_id') else '❌'}")
    print(f"市价卖单: {'✅' if results.get('market_sell_order_id') else '❌'}")
    print(f"历史委托查询: {'✅' if results.get('history') else '⚠️'}")
    print(f"交易详情查询: {'✅' if results.get('trade_details') else '⚠️'}")
    print(f"限价单取消: {'✅' if results.get('cancel_success') else '⚠️' if results.get('limit_order_id') else '跳过'}")
    print(f"最终清理订单数: {cancelled_count_after}")
    
    print("\n" + "="*60)
    print("测试完成!")
    print("="*60)
    print("\n✅ 测试流程已完成:")
    print("  1. 检查账户余额")
    print("  2. 取消所有活跃订单（开单时无法调整杠杆）")
    print("  3. 设置杠杆（20x）")
    print("  4. 下限价买单（95%价格，不会立即成交）")
    print("  5. 查询当前委托（确认限价单存在）")
    print("  6. 下市价买单（立即成交）")
    print("  7. 下市价卖单（立即成交，恢复账户状态）")
    print("  8. 查询历史委托和交易详情")
    print("  9. 取消限价单（清理未成交订单）")
    print("  10. 最终清理（确保没有挂单）")
    print("\n账户应该已经恢复到接近初始状态，且没有挂单。")


if __name__ == '__main__':
    main()

