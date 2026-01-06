#!/usr/bin/env python3
"""
WEEX AI Trading Hackathon - Official API Test Script (Batch Mode)
支持批量处理多个 API key 和使用代理

使用方法:
    python3 official_api_test_batch.py --api-keys api_keys.json --proxy http://proxy.example.com:3128
    python3 official_api_test_batch.py --api-keys api_keys.csv --proxy http://proxy.example.com:3128
    python3 official_api_test_batch.py --api-keys api_keys.json  # 不使用代理
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
import csv
from datetime import datetime
from typing import Dict, List, Optional

# Try to load .env file if python-dotenv is available
try:
    from dotenv import load_dotenv
    load_dotenv()
except ImportError:
    pass

BASE_URL = "https://api-contract.weex.com"
SYMBOL = "cmt_btcusdt"  # 官方测试交易对


def mask_proxy_url(proxy_url: str) -> str:
    """安全地显示代理 URL，隐藏密码部分"""
    if '@' not in proxy_url:
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


class WEEXAPIClient:
    """WEEX API 客户端，支持代理
    
    代理格式支持：
    - 不带认证: http://proxy.example.com:3128
    - 带认证: http://username:password@proxy.example.com:3128
    - Squid 代理完全支持以上两种格式
    """
    
    def __init__(self, api_key: str, secret_key: str, passphrase: str, proxy: Optional[str] = None):
        self.api_key = api_key
        self.secret_key = secret_key
        self.passphrase = passphrase
        self.proxy = proxy
        self.session = requests.Session()
        
        # 配置代理（requests 库原生支持带认证的代理 URL）
        if proxy:
            self.proxies = {
                'http': proxy,   # 同时设置 HTTP 和 HTTPS，Squid 代理需要
                'https': proxy,  # requests 会自动通过 HTTP CONNECT 方法处理 HTTPS
            }
            # 同时设置到 session.proxies（虽然我们会在请求时显式传递，但保留此设置作为备用）
            self.session.proxies = self.proxies
            print(f"✅ 已配置代理: {mask_proxy_url(proxy)}")
        else:
            self.proxies = None
    
    def generate_signature(self, timestamp: str, method: str, request_path: str, query_string: str, body: str = "") -> str:
        """生成 API 签名"""
        message = timestamp + method.upper() + request_path + query_string + str(body)
        signature = hmac.new(self.secret_key.encode(), message.encode(), hashlib.sha256).digest()
        return base64.b64encode(signature).decode()
    
    def send_request(self, method: str, request_path: str, query_string: str = "", body: Optional[Dict] = None) -> requests.Response:
        """发送 API 请求"""
        timestamp = str(int(time.time() * 1000))
        body_str = json.dumps(body) if body else ""
        
        signature = self.generate_signature(timestamp, method, request_path, query_string, body_str)
        
        headers = {
            "ACCESS-KEY": self.api_key,
            "ACCESS-SIGN": signature,
            "ACCESS-TIMESTAMP": timestamp,
            "ACCESS-PASSPHRASE": self.passphrase,
            "Content-Type": "application/json",
            "locale": "zh-CN"
        }
        
        url = BASE_URL + request_path
        if query_string:
            if query_string.startswith("?"):
                url += query_string
            else:
                url += "?" + query_string
        
        # 显式传递 proxies 参数，确保代理生效（与 test_week.py 一致）
        if method == "GET":
            response = self.session.get(url, headers=headers, proxies=self.proxies, timeout=120)
        elif method == "POST":
            response = self.session.post(url, headers=headers, data=body_str, proxies=self.proxies, timeout=120)
        else:
            raise ValueError(f"Unsupported HTTP method: {method}")
        
        return response
    
    def print_response(self, step_name: str, response: requests.Response) -> Optional[Dict]:
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
    
    def step1_check_domain(self) -> bool:
        """步骤 1: 检查域名和路径"""
        print("\n[步骤 1] 检查 API 域名和路径")
        print(f"API Base URL: {BASE_URL}")
        print(f"交易对: {SYMBOL}")
        return True
    
    def step2_check_account_balance(self) -> Optional[List]:
        """步骤 2: 检查账户余额"""
        print("\n[步骤 2] 检查账户余额")
        request_path = "/capi/v2/account/assets"
        response = self.send_request("GET", request_path)
        data = self.print_response("检查账户余额", response)
        
        if response.status_code == 200 and data:
            print(f"\n✅ 账户信息获取成功")
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
    
    def step2_5_cancel_all_active_orders(self) -> int:
        """步骤 2.5: 取消所有活跃订单（开单时无法调整杠杆）"""
        print("\n[步骤 2.5] 检查并取消所有活跃订单")
        
        request_path = "/capi/v2/order/current"
        query_string = f"?symbol={SYMBOL}"
        response = self.send_request("GET", request_path, query_string=query_string)
        data = self.print_response("获取当前委托", response)
        
        if response.status_code != 200:
            print(f"⚠️  获取当前委托失败（状态码: {response.status_code}），跳过取消订单")
            return 0
        
        if not data:
            print(f"✅ 没有活跃订单，无需取消")
            return 0
        
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
            
            cancel_path = "/capi/v2/order/cancel_order"
            cancel_body = {"orderId": str(order_id)}
            cancel_response = self.send_request("POST", cancel_path, body=cancel_body)
            
            if cancel_response.status_code == 200:
                print(f"    ✅ 订单 {order_id} 取消成功")
                cancelled_count += 1
            else:
                print(f"    ⚠️  订单 {order_id} 取消失败: {cancel_response.status_code}")
            
            time.sleep(0.2)
        
        print(f"\n✅ 成功取消 {cancelled_count}/{len(orders)} 个订单")
        
        if cancelled_count > 0:
            print(f"\n等待 2 秒确保订单取消完成...")
            time.sleep(2)
        
        return cancelled_count
    
    def step3_set_leverage(self) -> bool:
        """步骤 3: 设置杠杆为 20x（全仓模式）"""
        print("\n[步骤 3] 设置杠杆")
        request_path = "/capi/v2/account/leverage"
        body = {
            "symbol": SYMBOL,
            "marginMode": 1,
            "longLeverage": "20",
            "shortLeverage": "20"
        }
        response = self.send_request("POST", request_path, body=body)
        data = self.print_response("设置杠杆 (20x, 全仓模式)", response)
        
        if response.status_code == 200:
            print(f"✅ 杠杆设置成功: 20x (全仓)")
            return True
        else:
            print(f"⚠️  杠杆设置可能失败，继续执行...")
            return False
    
    def step4_get_asset_price(self) -> Optional[float]:
        """步骤 4: 获取资产价格"""
        print("\n[步骤 4] 获取资产价格")
        request_path = "/capi/v2/market/ticker"
        query_string = f"?symbol={SYMBOL}"
        response = self.send_request("GET", request_path, query_string=query_string)
        data = self.print_response("获取资产价格", response)
        
        if response.status_code == 200 and data:
            if isinstance(data, dict):
                last_price = data.get('last') or data.get('lastPrice')
                if last_price:
                    print(f"\n✅ 当前价格: {last_price} USDT")
                    return float(last_price)
        return None
    
    def place_order(self, price: float, size: float, order_type: str, side: str = "1", 
                   order_type_flag: str = "0", match_price: str = "0") -> Optional[str]:
        """下单函数"""
        client_oid = str(int(time.time() * 1000))
        
        request_path = "/capi/v2/order/placeOrder"
        body = {
            "symbol": SYMBOL,
            "client_oid": client_oid,
            "size": str(size),
            "type": side,
            "order_type": order_type_flag,
            "match_price": match_price,
            "price": str(int(price))
        }
        
        print(f"下单参数: {json.dumps(body, indent=2, ensure_ascii=False)}")
        
        response = self.send_request("POST", request_path, body=body)
        data = self.print_response(f"下单 ({order_type})", response)
        
        if response.status_code == 200 and data:
            order_id = None
            if isinstance(data, dict):
                order_id = data.get('order_id')
            
            if order_id:
                print(f"\n✅ 订单提交成功! 订单ID: {order_id}")
                return order_id
            else:
                print(f"\n⚠️  订单可能已提交，但未获取到订单ID")
                return "unknown"
        else:
            print(f"❌ 下单失败")
            return None
    
    def step5_place_limit_buy_order(self, price: float) -> Optional[str]:
        """步骤 5: 下限价买单"""
        print("\n[步骤 5] 下限价买单（市价5%以下）")
        limit_price = price * 0.95
        order_size = 0.005
        
        print(f"当前价格: {price} USDT")
        print(f"限价: {limit_price} USDT (95% of current price)")
        print(f"订单数量: {order_size} BTC")
        
        return self.place_order(
            price=limit_price,
            size=order_size,
            order_type="限价买单",
            side="1",
            order_type_flag="1",
            match_price="0"
        )
    
    def step6_place_market_buy_order(self, price: float) -> Optional[str]:
        """步骤 6: 下市价买单"""
        print("\n[步骤 6] 下市价买单")
        order_size = 0.005
        print(f"当前价格: {price} USDT")
        print(f"订单数量: {order_size} BTC")
        
        return self.place_order(
            price=price,
            size=order_size,
            order_type="市价买单",
            side="1",
            order_type_flag="0",
            match_price="1"
        )
    
    def step7_place_market_sell_order(self, price: float) -> Optional[str]:
        """步骤 7: 下市价卖单"""
        print("\n[步骤 7] 下市价卖单（平仓）")
        order_size = 0.005
        print(f"当前价格: {price} USDT")
        print(f"订单数量: {order_size} BTC")
        
        return self.place_order(
            price=price,
            size=order_size,
            order_type="市价卖单",
            side="2",
            order_type_flag="0",
            match_price="1"
        )
    
    def step8_cancel_order(self, order_id: str) -> bool:
        """步骤 8: 取消订单"""
        print(f"\n[步骤 8] 取消订单 (订单ID: {order_id})")
        request_path = "/capi/v2/order/cancel_order"
        body = {"orderId": order_id}
        response = self.send_request("POST", request_path, body=body)
        data = self.print_response("取消订单", response)
        
        if response.status_code == 200:
            print(f"\n✅ 订单取消成功")
            return True
        else:
            print(f"\n⚠️  订单取消失败")
            return False
    
    def step9_get_current_orders(self) -> Optional[Dict]:
        """步骤 9: 获取当前委托"""
        print("\n[步骤 9] 获取当前委托")
        request_path = "/capi/v2/order/current"
        query_string = f"?symbol={SYMBOL}"
        response = self.send_request("GET", request_path, query_string=query_string)
        data = self.print_response("获取当前委托", response)
        
        if response.status_code == 200:
            print(f"✅ 当前委托查询成功")
            return data
        else:
            print(f"⚠️  获取当前委托失败")
            return None
    
    def step10_get_order_history(self) -> Optional[Dict]:
        """步骤 10: 获取历史委托"""
        print("\n[步骤 10] 获取历史委托")
        request_path = "/capi/v2/order/history"
        query_string = f"?symbol={SYMBOL}&pageSize=10"
        response = self.send_request("GET", request_path, query_string=query_string)
        data = self.print_response("获取历史委托", response)
        
        if response.status_code == 200:
            print(f"✅ 历史委托查询成功")
            return data
        else:
            print(f"⚠️  获取历史委托失败")
            return None
    
    def step11_get_trade_details(self) -> Optional[Dict]:
        """步骤 11: 获取交易详情"""
        print("\n[步骤 11] 获取交易详情")
        request_path = "/capi/v2/order/fills"
        query_string = f"?symbol={SYMBOL}&pageSize=10"
        response = self.send_request("GET", request_path, query_string=query_string)
        data = self.print_response("获取交易详情", response)
        
        if response.status_code == 200:
            print(f"✅ 交易详情查询成功")
            return data
        else:
            print(f"⚠️  获取交易详情失败")
            return None
    
    def run_test(self) -> Dict:
        """运行完整的测试流程"""
        print("="*60)
        print("WEEX AI Trading Hackathon - API 测试")
        print("="*60)
        # 安全显示 API key
        api_key_display = self.api_key[:10] + "..." if self.api_key and len(self.api_key) > 10 else (self.api_key or "N/A")
        print(f"\nAPI Key: {api_key_display}")
        print(f"交易对: {SYMBOL}")
        print(f"测试流程: 检查余额 -> 取消活跃订单 -> 设置杠杆 -> 限价买单 -> 查询当前委托 -> 市价买单 -> 市价卖单 -> 查询历史 -> 取消限价单 -> 最终清理")
        print(f"\n开始测试...")
        
        results = {
            'api_key': self.api_key,
            'start_time': datetime.now().isoformat(),
            'success': False,
            'error': None
        }
        
        try:
            # 步骤 1: 检查域名
            self.step1_check_domain()
            
            # 步骤 2: 检查账户余额
            balance_data = self.step2_check_account_balance()
            results['balance'] = balance_data is not None
            
            # 步骤 2.5: 取消所有活跃订单
            cancelled_count_before = self.step2_5_cancel_all_active_orders()
            results['cancelled_before'] = cancelled_count_before
            
            # 步骤 3: 设置杠杆
            leverage_success = self.step3_set_leverage()
            results['leverage'] = leverage_success
            
            # 步骤 4: 获取价格
            price = self.step4_get_asset_price()
            results['price'] = price is not None
            
            limit_order_id = None
            
            if price:
                # 步骤 5: 下限价买单
                limit_order_id = self.step5_place_limit_buy_order(price)
                results['limit_order_id'] = limit_order_id
                
                if limit_order_id:
                    time.sleep(2)
                    current_orders = self.step9_get_current_orders()
                    results['current_orders'] = current_orders is not None
                    
                    time.sleep(2)
                    market_buy_order_id = self.step6_place_market_buy_order(price)
                    results['market_buy_order_id'] = market_buy_order_id
                    
                    if market_buy_order_id:
                        time.sleep(3)
                        market_sell_order_id = self.step7_place_market_sell_order(price)
                        results['market_sell_order_id'] = market_sell_order_id
                        
                        if market_sell_order_id:
                            time.sleep(3)
                            history = self.step10_get_order_history()
                            results['history'] = history is not None
                            
                            trade_details = self.step11_get_trade_details()
                            results['trade_details'] = trade_details is not None
                            
                            time.sleep(1)
                            if limit_order_id and limit_order_id != "unknown":
                                cancel_success = self.step8_cancel_order(limit_order_id)
                                results['cancel_success'] = cancel_success
            
            # 步骤 12: 最终清理
            print("\n" + "="*60)
            print("[最终清理] 检查并取消所有活跃订单，确保账户干净")
            print("="*60)
            cancelled_count_after = self.step2_5_cancel_all_active_orders()
            results['cancelled_after'] = cancelled_count_after
            
            results['success'] = True
            results['end_time'] = datetime.now().isoformat()
            
        except Exception as e:
            results['error'] = str(e)
            results['end_time'] = datetime.now().isoformat()
            print(f"\n❌ 测试过程中发生错误: {e}")
            import traceback
            traceback.print_exc()
        
        return results


def load_api_keys_from_json(file_path: str) -> List[Dict[str, str]]:
    """从 JSON 文件加载 API keys"""
    with open(file_path, 'r', encoding='utf-8') as f:
        data = json.load(f)
    
    if isinstance(data, list):
        return data
    elif isinstance(data, dict) and 'api_keys' in data:
        return data['api_keys']
    else:
        raise ValueError("JSON 文件格式错误，应该是数组或包含 'api_keys' 字段的对象")


def load_api_keys_from_csv(file_path: str) -> List[Dict[str, str]]:
    """从 CSV 文件加载 API keys"""
    api_keys = []
    with open(file_path, 'r', encoding='utf-8') as f:
        reader = csv.DictReader(f)
        for row_num, row in enumerate(reader, start=2):  # 从第2行开始（第1行是标题）
            # 尝试多种字段名
            api_key = row.get('api_key') or row.get('WEEX_API_KEY') or row.get('apiKey')
            secret_key = row.get('secret_key') or row.get('WEEX_SECRET_KEY') or row.get('secretKey')
            passphrase = row.get('passphrase') or row.get('WEEX_PASSPHRASE') or row.get('Passphrase')
            
            # 跳过空行
            if not api_key and not secret_key and not passphrase:
                continue
            
            # 验证必需的字段
            if not api_key or not secret_key or not passphrase:
                print(f"⚠️  警告: CSV 第 {row_num} 行缺少必需的字段，已跳过")
                print(f"    api_key: {'有' if api_key else '缺失'}, secret_key: {'有' if secret_key else '缺失'}, passphrase: {'有' if passphrase else '缺失'}")
                continue
            
            api_keys.append({
                'api_key': api_key.strip(),
                'secret_key': secret_key.strip(),
                'passphrase': passphrase.strip()
            })
    return api_keys


def load_api_keys(file_path: str) -> List[Dict[str, str]]:
    """根据文件扩展名自动选择加载方式"""
    if file_path.endswith('.json'):
        return load_api_keys_from_json(file_path)
    elif file_path.endswith('.csv'):
        return load_api_keys_from_csv(file_path)
    else:
        raise ValueError(f"不支持的文件格式: {file_path}，请使用 .json 或 .csv")


def main():
    parser = argparse.ArgumentParser(
        description='WEEX API 批量测试工具',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
示例:
  # 使用 JSON 文件和代理
  python3 official_api_test_batch.py --api-keys api_keys.json --proxy http://proxy.example.com:3128
  
  # 使用 CSV 文件，不使用代理
  python3 official_api_test_batch.py --api-keys api_keys.csv
  
  # 使用环境变量（单个 API key）
  python3 official_api_test_batch.py --proxy http://proxy.example.com:3128

API Keys 文件格式:

JSON 格式 (api_keys.json):
  [
    {
      "api_key": "weex_xxx",
      "secret_key": "xxx",
      "passphrase": "xxx"
    },
    ...
  ]

CSV 格式 (api_keys.csv):
  api_key,secret_key,passphrase
  weex_xxx,xxx,xxx
  ...
        """
    )
    
    parser.add_argument(
        '--api-keys',
        type=str,
        help='API keys 文件路径（JSON 或 CSV 格式）。如果不提供，则从环境变量读取单个 API key'
    )
    
    parser.add_argument(
        '--proxy',
        type=str,
        help='代理地址，例如: http://proxy.example.com:3128'
    )
    
    parser.add_argument(
        '--output',
        type=str,
        default='test_results.json',
        help='测试结果输出文件（JSON 格式），默认: test_results.json'
    )
    
    args = parser.parse_args()
    
    # 加载 API keys
    if args.api_keys:
        print(f"📁 从文件加载 API keys: {args.api_keys}")
        api_keys_list = load_api_keys(args.api_keys)
        print(f"✅ 加载了 {len(api_keys_list)} 个 API key")
    else:
        # 从环境变量读取单个 API key
        api_key = os.environ.get("WEEX_API_KEY")
        secret_key = os.environ.get("WEEX_SECRET_KEY")
        passphrase = os.environ.get("WEEX_PASSPHRASE")
        
        if not api_key or not secret_key or not passphrase:
            print("❌ 错误: 未提供 --api-keys 文件，且环境变量中缺少 API 凭证")
            print("请使用 --api-keys 参数指定文件，或设置环境变量:")
            print("  WEEX_API_KEY")
            print("  WEEX_SECRET_KEY")
            print("  WEEX_PASSPHRASE")
            sys.exit(1)
        
        api_keys_list = [{
            'api_key': api_key,
            'secret_key': secret_key,
            'passphrase': passphrase
        }]
        print("✅ 从环境变量加载 API key")
    
    # 显示代理信息
    if args.proxy:
        print(f"🌐 使用代理: {mask_proxy_url(args.proxy)}")
    else:
        print("⚠️  未使用代理，如果 IP 不在白名单中可能会失败")
    
    # 批量测试
    all_results = []
    
    for idx, creds in enumerate(api_keys_list, 1):
        print("\n" + "="*80)
        print(f"测试 API Key {idx}/{len(api_keys_list)}")
        print("="*80)
        
        # 验证 API key 数据
        if not creds.get('api_key') or not creds.get('secret_key') or not creds.get('passphrase'):
            print(f"⚠️  错误: API Key {idx} 数据不完整，已跳过")
            print(f"    api_key: {'有' if creds.get('api_key') else '缺失'}")
            print(f"    secret_key: {'有' if creds.get('secret_key') else '缺失'}")
            print(f"    passphrase: {'有' if creds.get('passphrase') else '缺失'}")
            all_results.append({
                'api_key': creds.get('api_key', 'N/A'),
                'start_time': datetime.now().isoformat(),
                'success': False,
                'error': 'API key 数据不完整',
                'end_time': datetime.now().isoformat()
            })
            continue
        
        client = WEEXAPIClient(
            api_key=creds['api_key'],
            secret_key=creds['secret_key'],
            passphrase=creds['passphrase'],
            proxy=args.proxy
        )
        
        result = client.run_test()
        all_results.append(result)
        
        # 显示简要结果
        print("\n" + "-"*80)
        api_key_display = creds['api_key'][:10] + "..." if creds['api_key'] and len(creds['api_key']) > 10 else (creds['api_key'] or 'N/A')
        print(f"API Key: {api_key_display}")
        print(f"测试结果: {'✅ 成功' if result['success'] else '❌ 失败'}")
        if result.get('error'):
            print(f"错误信息: {result['error']}")
        print("-"*80)
        
        # 在 API keys 之间稍作延迟
        if idx < len(api_keys_list):
            print(f"\n等待 3 秒后继续下一个 API key...")
            time.sleep(3)
    
    # 保存结果
    output_data = {
        'test_time': datetime.now().isoformat(),
        'total_count': len(all_results),
        'success_count': sum(1 for r in all_results if r['success']),
        'fail_count': sum(1 for r in all_results if not r['success']),
        'proxy': args.proxy,
        'results': all_results
    }
    
    with open(args.output, 'w', encoding='utf-8') as f:
        json.dump(output_data, f, indent=2, ensure_ascii=False)
    
    # 显示总结
    print("\n" + "="*80)
    print("批量测试完成!")
    print("="*80)
    print(f"总计: {len(all_results)} 个 API key")
    print(f"成功: {output_data['success_count']} 个")
    print(f"失败: {output_data['fail_count']} 个")
    print(f"结果已保存到: {args.output}")


if __name__ == '__main__':
    main()

