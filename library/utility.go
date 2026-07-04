package library

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// GetString Helper: safely extracts string from map[string]interface{}
func GetString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// MpesaTimestamp Helper: Get current MPESA timestamp
func MpesaTimestamp() string {
	return time.Now().Format("20060102150405")
}

func logRequest(method, endpoint string, requestHeaders map[string]string, requestBody interface{}, responseStatus int, responseHeader http.Header, responseBody string) {

	if os.Getenv("debug") == "1" || os.Getenv("DEBUG") == "1" {

		responseHeaders := make(map[string]string)

		for k, v := range responseHeader {

			responseHeaders[k] = strings.Join(v, ",")

		}

		var heads, rheads []string
		for k, v := range requestHeaders {

			heads = append(heads, fmt.Sprintf("\t%s : %s", k, v))
		}

		for k, v := range responseHeaders {

			rheads = append(rheads, fmt.Sprintf("\t%s : %s", k, v))
		}

		body := "none"

		if requestBody != nil {

			jsonData, _ := json.Marshal(requestBody)
			body = string(jsonData)

		}

		log.Printf("**** BEGIN HTTP %s REQUEST ****\n"+
			"Remote Url : %s\n"+
			"Request Headers:\n"+
			"%s\n"+
			"Request Payload\n"+
			"\t%s\n"+
			"Response Status: %d\n"+
			"Response Headers\n"+
			"%s\n"+
			"Response Body\n"+
			"**** END HTTP %s REQUEST ****\n"+
			"\t%s", strings.ToUpper(method), endpoint, strings.Join(heads, "\n"), body, responseStatus, strings.Join(rheads, "\n"), strings.ToUpper(method), responseBody)
	}

}
