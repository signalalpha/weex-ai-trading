package main

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

func main() {
	// Test 1: Try to access a public endpoint (market time) without authentication
	fmt.Println("Test 1: Accessing public endpoint (server time)...")
	url := "https://api-contract.weex.com/capi/v2/market/time"
	
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}
	
	req.Header.Set("User-Agent", "weex-ai-trading/1.0")
	req.Header.Set("locale", "zh-CN")
	
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	bodyBytes, _ := io.ReadAll(resp.Body)
	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Response: %s\n", string(bodyBytes))
	
	// Test 2: Try with different base URL (spot API)
	fmt.Println("\nTest 2: Trying spot API endpoint...")
	url2 := "https://api.weex.com/v1/public/time"
	req2, _ := http.NewRequest("GET", url2, nil)
	req2.Header.Set("User-Agent", "weex-ai-trading/1.0")
	
	resp2, err := client.Do(req2)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp2.Body.Close()
	
	bodyBytes2, _ := io.ReadAll(resp2.Body)
	fmt.Printf("Status: %d\n", resp2.StatusCode)
	fmt.Printf("Response: %s\n", string(bodyBytes2))
}

