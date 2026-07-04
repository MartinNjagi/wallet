package library

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HTTPRequest makes an HTTP request with the given parameters
func HTTPRequest(ctx context.Context, method, url string, body interface{}, headers map[string]string) ([]byte, int, error) {
	var requestBody []byte
	var err error

	// Marshal body if provided
	if body != nil {
		requestBody, err = json.Marshal(body)
		log.Printf("request body %s", string(requestBody))
		if err != nil {
			return nil, 0, fmt.Errorf("error marshalling request body: %w", err)
		}
	}
	log.Printf("request body %s", string(requestBody))

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, 0, fmt.Errorf("error creating HTTP request: %w", err)
	}

	// Set headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Make request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("error reading response body: %w", err)
	}

	st := resp.StatusCode
	hd := resp.Header

	logRequest("POST", url, headers, body, st, hd, string(responseBody))

	return responseBody, resp.StatusCode, nil
}

// HTTPRequestWithBearer is a convenience wrapper for requests with Bearer token authentication
func HTTPRequestWithBearer(ctx context.Context, method, url string, body interface{}, bearerToken string) ([]byte, int, error) {
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": fmt.Sprintf("Bearer %s", bearerToken),
	}
	return HTTPRequest(ctx, method, url, body, headers)
}

func HTTPGet(remoteURL string, headers map[string]string, payload map[string]string) (httpStatus int, response string) {

	var fields []string

	if payload != nil {

		for key, value := range payload {

			val := fmt.Sprintf("%s=%v", key, url.QueryEscape(value))

			fields = append(fields, val)
		}
	}

	params := strings.Join(fields, "&")

	endpoint := fmt.Sprintf("%s?%s", remoteURL, params)

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {

		log.Printf("got error making http request %s", err.Error())
		return 0, ""
	}

	logHeaders := make(map[string]string)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	logHeaders["Content-Type"] = "application/json"
	logHeaders["Accept"] = "application/json"

	if headers != nil {

		for k, v := range headers {

			req.Header.Set(k, v)
			logHeaders[k] = v
		}
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("got error making GET http request %s", err.Error())
		return 0, ""
	}

	st := resp.StatusCode
	body, err := io.ReadAll(resp.Body)
	if err != nil {

		log.Printf("got error making http request %s", err.Error())
		return st, ""
	}

	logRequest("GET", endpoint, logHeaders, nil, st, req.Header, string(body))

	return st, string(body)
}
