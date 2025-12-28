package api

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

// DebugRequest logs detailed request information
func DebugRequest(req *http.Request) string {
	var debugInfo string
	debugInfo += fmt.Sprintf("Method: %s\n", req.Method)
	debugInfo += fmt.Sprintf("URL: %s\n", req.URL.String())
	debugInfo += "Headers:\n"
	for k, v := range req.Header {
		// Mask sensitive headers
		if k == "ACCESS-KEY" || k == "ACCESS-SIGN" || k == "ACCESS-PASSPHRASE" {
			if len(v) > 0 && len(v[0]) > 10 {
				debugInfo += fmt.Sprintf("  %s: %s...\n", k, v[0][:10])
			} else {
				debugInfo += fmt.Sprintf("  %s: %v\n", k, v)
			}
		} else {
			debugInfo += fmt.Sprintf("  %s: %v\n", k, v)
		}
	}
	if req.Body != nil {
		bodyBytes, _ := io.ReadAll(req.Body)
		if len(bodyBytes) > 0 {
			debugInfo += fmt.Sprintf("Body: %s\n", string(bodyBytes))
			// Reset body for actual request
			req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}
	}
	return debugInfo
}

// DebugResponse logs detailed response information
func DebugResponse(resp *http.Response) (string, error) {
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	// Reset body for parsing
	resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	debugInfo := fmt.Sprintf("Status: %d %s\n", resp.StatusCode, resp.Status)
	debugInfo += "Response Headers:\n"
	for k, v := range resp.Header {
		debugInfo += fmt.Sprintf("  %s: %v\n", k, v)
	}
	debugInfo += fmt.Sprintf("Body: %s\n", string(bodyBytes))
	return debugInfo, nil
}

