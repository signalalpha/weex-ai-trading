import time
import hmac
import hashlib
import base64
import requests
import json
import os

# Try to load .env file if python-dotenv is available
try:
    from dotenv import load_dotenv
    # Try to load .env file from current directory
    load_dotenv()
except ImportError:
    # If python-dotenv is not installed, just use environment variables
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
        "\nYou can:\n"
        "  1. Create a .env file in project root with these variables, or\n"
        "  2. Set them in your shell:\n"
        "     export WEEX_API_KEY='your_api_key'\n"
        "     export WEEX_SECRET_KEY='your_secret_key'\n"
        "     export WEEX_PASSPHRASE='your_passphrase'\n"
        "\nIf you want to use .env file, install python-dotenv:\n"
        "  pip install python-dotenv"
    )


def generate_signature(secret_key, timestamp, method, request_path, query_string, body):
  message = timestamp + method.upper() + request_path + query_string + str(body)
  signature = hmac.new(secret_key.encode(), message.encode(), hashlib.sha256).digest()
  # print(base64.b64encode(signature).decode())
  return base64.b64encode(signature).decode()


def generate_signature_get(secret_key, timestamp, method, request_path, query_string):
  message = timestamp + method.upper() + request_path + query_string
  signature = hmac.new(secret_key.encode(), message.encode(), hashlib.sha256).digest()
  # print(base64.b64encode(signature).decode())
  return base64.b64encode(signature).decode()


def send_request_post(api_key, secret_key, access_passphrase, method, request_path, query_string, body):
  timestamp = str(int(time.time() * 1000))
  # print(timestamp)
  body = json.dumps(body)
  signature = generate_signature(secret_key, timestamp, method, request_path, query_string, body)

  headers = {
        "ACCESS-KEY": api_key,
        "ACCESS-SIGN": signature,
        "ACCESS-TIMESTAMP": timestamp,
        "ACCESS-PASSPHRASE": access_passphrase,
        "Content-Type": "application/json",
        "locale": "en-US"
  }

  url = "https://api-contract.weex.com"  # WEEX Contract API base URL
  if method == "GET":
    response = requests.get(url + request_path, headers=headers)
  elif method == "POST":
    response = requests.post(url + request_path, headers=headers, data=body)
  return response

def send_request_get(api_key, secret_key, access_passphrase, method, request_path, query_string):
  timestamp = str(int(time.time() * 1000))
  # print(timestamp)
  signature = generate_signature_get(secret_key, timestamp, method, request_path, query_string)

  headers = {
        "ACCESS-KEY": api_key,
        "ACCESS-SIGN": signature,
        "ACCESS-TIMESTAMP": timestamp,
        "ACCESS-PASSPHRASE": access_passphrase,
        "Content-Type": "application/json",
        "locale": "en-US"
  }

  url = "https://api-contract.weex.com"  # WEEX Contract API base URL
  if method == "GET":
    response = requests.get(url + request_path + query_string, headers=headers)
  return response

def get():
    # Example of calling a GET request
    request_path = "/capi/v2/account/position/singlePosition"
    query_string = '?symbol=cmt_btcusdt'
    response = send_request_get(api_key, secret_key, access_passphrase, "GET", request_path, query_string)
    print(response.status_code)
    print(response.text)

def post():
    # Example of calling a POST request
    request_path = "/capi/v2/order/placeOrder"
    body = {
	"symbol": "cmt_btcusdt",
	"client_oid": "71557515757447",
	"size": "0.01",
	"type": "1",
	"order_type": "0",
	"match_price": "1",
	"price": "80000"}
    query_string = ""
    response = send_request_post(api_key, secret_key, access_passphrase, "POST", request_path, query_string, body)
    print(response.status_code)
    print(response.text)

if __name__ == '__main__':
    get()
    post()