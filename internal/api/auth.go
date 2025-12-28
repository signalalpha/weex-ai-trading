package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"time"
)

// Signature generates the signature for WEEX API requests
func Signature(secretKey, timestamp, method, requestPath, queryString, body string) string {
	// Create the message to sign: timestamp + method + requestPath + queryString + body
	message := timestamp + method + requestPath + queryString + body

	// Generate HMAC-SHA256 signature
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(message))
	signature := mac.Sum(nil)

	// Base64 encode the signature
	return base64.StdEncoding.EncodeToString(signature)
}

// GenerateTimestamp generates a timestamp in milliseconds
func GenerateTimestamp() string {
	return strconv.FormatInt(time.Now().UnixMilli(), 10)
}

// GetAuthHeaders returns the authentication headers for API requests
func GetAuthHeaders(apiKey, secretKey, passphrase, method, requestPath, queryString, body string) (map[string]string, error) {
	timestamp := GenerateTimestamp()
	signature := Signature(secretKey, timestamp, method, requestPath, queryString, body)

	headers := map[string]string{
		"ACCESS-KEY":       apiKey,
		"ACCESS-SIGN":      signature,
		"ACCESS-TIMESTAMP": timestamp,
		"Content-Type":     "application/json",
		"locale":           "zh-CN",
	}

	// Add passphrase if provided
	if passphrase != "" {
		headers["ACCESS-PASSPHRASE"] = passphrase
	}

	return headers, nil
}
