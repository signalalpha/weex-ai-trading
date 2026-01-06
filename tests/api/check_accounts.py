#!/usr/bin/env python3
"""
WEEX 账户批量检查工具
功能：
1. 批量检查账户余额
2. 批量检查交易量是否达到 10 USDT

使用方法:
    python3 check_accounts.py --api-keys api_keys.csv --proxy http://proxy.example.com:3128
    python3 check_accounts.py --api-keys api_keys.json --min-volume 10
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
SYMBOL = "cmt_btcusdt"  # 默认检查的交易对


def mask_proxy_url(proxy_url: str) -> str:
    """安全地显示代理 URL，隐藏密码部分"""
    if '@' not in proxy_url:
        return proxy_url
    try:
        protocol, rest = proxy_url.split('://', 1)
        if '@' in rest:
            auth, host_port = rest.rsplit('@', 1)
            username = auth.split(':', 1)[0] if ':' in auth else auth
            return f"{protocol}://{username}:***@{host_port}"
    except Exception:
        pass
    return proxy_url


class WEEXAccountChecker:
    """WEEX 账户检查器"""
    
    def __init__(self, api_key: str, secret_key: str, passphrase: str, proxy: Optional[str] = None):
        self.api_key = api_key
        self.secret_key = secret_key
        self.passphrase = passphrase
        self.session = requests.Session()
        
        # 配置代理
        if proxy:
            self.proxies = {
                'http': proxy,
                'https': proxy,
            }
            self.session.proxies = self.proxies
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
        
        if method == "GET":
            response = self.session.get(url, headers=headers, proxies=self.proxies, timeout=120)
        elif method == "POST":
            response = self.session.post(url, headers=headers, data=body_str, proxies=self.proxies, timeout=120)
        else:
            raise ValueError(f"Unsupported HTTP method: {method}")
        
        return response
    
    def check_balance(self) -> Dict:
        """检查账户余额"""
        request_path = "/capi/v2/account/assets"
        response = self.send_request("GET", request_path)
        
        result = {
            'success': False,
            'balance': None,
            'usdt_balance': None,
            'error': None
        }
        
        if response.status_code == 200:
            try:
                data = response.json()
                if isinstance(data, list):
                    result['success'] = True
                    result['balance'] = data
                    # 查找 USDT 余额
                    for asset in data:
                        if asset.get('coinName', '').upper() == 'USDT':
                            result['usdt_balance'] = float(asset.get('available', 0))
                            break
            except Exception as e:
                result['error'] = f"解析响应失败: {str(e)}"
        else:
            result['error'] = f"HTTP {response.status_code}: {response.text[:200]}"
        
        return result
    
    def check_trading_volume(self, symbol: str = SYMBOL, min_volume: float = 10.0) -> Dict:
        """检查交易量是否达到要求
        
        Args:
            symbol: 交易对符号
            min_volume: 最小交易量（USDT）
        
        Returns:
            包含交易量信息的字典
        """
        result = {
            'success': False,
            'total_volume': 0.0,
            'trade_count': 0,
            'meets_requirement': False,
            'error': None
        }
        
        # 查询交易历史（获取所有成交记录）
        request_path = "/capi/v2/order/fills"
        query_string = f"?symbol={symbol}&pageSize=100"  # 先查询最近100条
        
        try:
            response = self.send_request("GET", request_path, query_string=query_string)
            
            if response.status_code == 200:
                data = response.json()
                
                # 处理响应数据
                # 实际 API 返回格式：
                # {
                #     "list": [
                #         {
                #             "tradeId": 0,
                #             "orderId": 0,
                #             "symbol": "cmt_btcusdt",
                #             "fillValue": "12",  // 成交金额（USDT）
                #             "fillSize": "67",
                #             ...
                #         }
                #     ],
                #     "nextFlag": false,
                #     "totals": 0
                # }
                trades = []
                if isinstance(data, dict) and 'list' in data:
                    # 标准格式：数据在 list 字段中
                    trades = data['list'] if isinstance(data['list'], list) else []
                elif isinstance(data, list):
                    # 兼容格式：直接是数组
                    trades = data
                elif isinstance(data, dict) and 'data' in data:
                    # 兼容格式：数据在 data 字段中
                    trades = data['data'] if isinstance(data['data'], list) else []
                
                # 计算总交易量（使用 fillValue 字段，单位：USDT）
                total_volume = 0.0
                for trade in trades:
                    # 使用 fillValue 字段（成交金额，单位：USDT）
                    if 'fillValue' in trade:
                        try:
                            fill_value = float(trade.get('fillValue', 0))
                            total_volume += fill_value
                        except (ValueError, TypeError):
                            # 如果 fillValue 无法转换为数字，跳过这条记录
                            continue
                
                result['success'] = True
                result['total_volume'] = total_volume
                result['trade_count'] = len(trades)
                result['meets_requirement'] = total_volume >= min_volume
                
            else:
                result['error'] = f"HTTP {response.status_code}: {response.text[:200]}"
                
        except Exception as e:
            result['error'] = f"查询交易量失败: {str(e)}"
        
        return result
    
    def check_all(self, symbol: str = SYMBOL, min_volume: float = 10.0) -> Dict:
        """检查账户余额和交易量"""
        result = {
            'api_key': self.api_key[:10] + "..." if len(self.api_key) > 10 else self.api_key,
            'balance_check': self.check_balance(),
            'volume_check': self.check_trading_volume(symbol, min_volume)
        }
        return result


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
        for row_num, row in enumerate(reader, start=2):
            api_key = row.get('api_key') or row.get('WEEX_API_KEY') or row.get('apiKey')
            secret_key = row.get('secret_key') or row.get('WEEX_SECRET_KEY') or row.get('secretKey')
            passphrase = row.get('passphrase') or row.get('WEEX_PASSPHRASE') or row.get('Passphrase')
            
            if not api_key and not secret_key and not passphrase:
                continue
            
            if not api_key or not secret_key or not passphrase:
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


def format_balance_result(balance_check: Dict) -> str:
    """格式化余额检查结果"""
    if balance_check['success']:
        usdt = balance_check.get('usdt_balance')
        if usdt is not None:
            return f"USDT余额: {usdt:.2f}"
        else:
            return "余额查询成功（未找到USDT）"
    else:
        return f"❌ 失败: {balance_check.get('error', '未知错误')}"


def format_volume_result(volume_check: Dict, min_volume: float) -> str:
    """格式化交易量检查结果"""
    if volume_check['success']:
        volume = volume_check['total_volume']
        count = volume_check['trade_count']
        meets = volume_check['meets_requirement']
        status = "✅" if meets else "⚠️"
        return f"{status} 交易量: {volume:.2f} USDT ({count}笔交易) {'≥' if meets else '<'} {min_volume} USDT"
    else:
        return f"❌ 失败: {volume_check.get('error', '未知错误')}"


def main():
    parser = argparse.ArgumentParser(
        description='WEEX 账户批量检查工具',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
示例:
  # 检查余额和交易量（默认最小交易量 10 USDT）
  python3 check_accounts.py --api-keys api_keys.csv --proxy http://proxy.example.com:3128
  
  # 自定义最小交易量
  python3 check_accounts.py --api-keys api_keys.csv --min-volume 20 --proxy http://proxy.example.com:3128
  
  # 仅检查余额
  python3 check_accounts.py --api-keys api_keys.csv --balance-only --proxy http://proxy.example.com:3128

API Keys 文件格式与 official_api_test_batch.py 相同（JSON 或 CSV）
        """
    )
    
    parser.add_argument(
        '--api-keys',
        type=str,
        required=True,
        help='API keys 文件路径（JSON 或 CSV 格式）'
    )
    
    parser.add_argument(
        '--proxy',
        type=str,
        help='代理地址，例如: http://proxy.example.com:3128'
    )
    
    parser.add_argument(
        '--min-volume',
        type=float,
        default=10.0,
        help='最小交易量（USDT），默认: 10.0'
    )
    
    parser.add_argument(
        '--symbol',
        type=str,
        default=SYMBOL,
        help=f'交易对符号，默认: {SYMBOL}'
    )
    
    parser.add_argument(
        '--balance-only',
        action='store_true',
        help='仅检查余额，不检查交易量'
    )
    
    parser.add_argument(
        '--output',
        type=str,
        help='输出结果到 JSON 文件（可选）'
    )
    
    args = parser.parse_args()
    
    # 加载 API keys
    print(f"📁 从文件加载 API keys: {args.api_keys}")
    api_keys_list = load_api_keys(args.api_keys)
    print(f"✅ 加载了 {len(api_keys_list)} 个 API key\n")
    
    # 显示配置信息
    if args.proxy:
        print(f"🌐 使用代理: {mask_proxy_url(args.proxy)}\n")
    
    if not args.balance_only:
        print(f"📊 最小交易量要求: {args.min_volume} USDT")
        print(f"📈 交易对: {args.symbol}\n")
    
    # 批量检查
    all_results = []
    
    for idx, creds in enumerate(api_keys_list, 1):
        api_key_short = creds['api_key'][:15] + "..." if len(creds['api_key']) > 15 else creds['api_key']
        print(f"[{idx}/{len(api_keys_list)}] 检查账户: {api_key_short}")
        
        checker = WEEXAccountChecker(
            api_key=creds['api_key'],
            secret_key=creds['secret_key'],
            passphrase=creds['passphrase'],
            proxy=args.proxy
        )
        
        if args.balance_only:
            balance_check = checker.check_balance()
            result = {
                'api_key': creds['api_key'],
                'balance_check': balance_check
            }
            print(f"  {format_balance_result(balance_check)}")
        else:
            result = checker.check_all(symbol=args.symbol, min_volume=args.min_volume)
            print(f"  {format_balance_result(result['balance_check'])}")
            print(f"  {format_volume_result(result['volume_check'], args.min_volume)}")
        
        all_results.append(result)
        print()
        
        # 避免请求过快
        if idx < len(api_keys_list):
            time.sleep(0.5)
    
    # 统计结果
    print("=" * 80)
    print("检查结果统计")
    print("=" * 80)
    
    balance_success = sum(1 for r in all_results if r.get('balance_check', {}).get('success', False))
    print(f"余额检查成功: {balance_success}/{len(all_results)}")
    
    if not args.balance_only:
        volume_success = sum(1 for r in all_results if r.get('volume_check', {}).get('success', False))
        volume_meets = sum(1 for r in all_results if r.get('volume_check', {}).get('meets_requirement', False))
        print(f"交易量检查成功: {volume_success}/{len(all_results)}")
        print(f"交易量达到要求 (≥{args.min_volume} USDT): {volume_meets}/{len(all_results)}")
    
    # 保存结果
    if args.output:
        output_data = {
            'check_time': datetime.now().isoformat(),
            'total_count': len(all_results),
            'balance_success_count': balance_success,
            'min_volume': args.min_volume if not args.balance_only else None,
            'symbol': args.symbol if not args.balance_only else None,
            'volume_meets_count': volume_meets if not args.balance_only else None,
            'proxy': mask_proxy_url(args.proxy) if args.proxy else None,
            'results': all_results
        }
        
        with open(args.output, 'w', encoding='utf-8') as f:
            json.dump(output_data, f, indent=2, ensure_ascii=False)
        
        print(f"\n📄 结果已保存到: {args.output}")


if __name__ == '__main__':
    main()

