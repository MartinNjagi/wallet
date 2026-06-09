package internal

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
	"wallet/data"
)

type InternalClient struct {
	httpClient *http.Client
	baseURLs   map[string]string // service name → base URL
	secret     string
}

func NewInternalClient(cfg *data.AppConfig) *InternalClient {
	return &InternalClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		secret:     cfg.InternalServiceToken,
		baseURLs: map[string]string{
			"identity": cfg.IdentityServiceURL,
			"wallet":   cfg.WalletServiceURL,
			"sms":      cfg.SMSServiceURL,
		},
	}
}

// Post signs and sends an internal request, decoding into `out` (pass nil to ignore body)
func (c *InternalClient) Post(ctx context.Context, service, path string, payload, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	baseURL, ok := c.baseURLs[service]
	if !ok {
		return fmt.Errorf("unknown internal service: %s", service)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	sig, ts := computeHMAC(body, c.secret)
	req.Header.Set("X-Signature", sig)
	req.Header.Set("X-Timestamp", ts)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request to %s%s: %w", service, path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("service %s returned %d", service, resp.StatusCode)
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

func computeHMAC(body []byte, secret string) (signature, timestamp string) {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(fmt.Sprintf("%s.%s", ts, string(body))))
	return hex.EncodeToString(mac.Sum(nil)), ts
}
