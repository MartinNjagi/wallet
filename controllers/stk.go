package controllers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
	"wallet/data"
	"wallet/library"
	"wallet/models"

	"github.com/sirupsen/logrus"
)

// InitiateSTKPush starts a Safaricom STK push request.
func (ctr *Controller) InitiateSTKPush(ctx context.Context, req data.STKRequest) (*data.STKPushResponse, error) {
	code, codeErr := library.GenerateFriendlyCode(req.SessionID)
	if codeErr != nil {
		logrus.WithContext(ctx).Errorf("GenerateFriendlyCode failed: %s", codeErr.Error())
	}

	stk := models.STKPushRequest{
		MSISDN:        req.MSISDN,
		Amount:        req.Amount,
		AccountRef:    fmt.Sprintf("LCK%d", time.Now().Unix()),
		ReferenceCode: code,
		SessionID:     req.SessionID,
		Status:        data.STKStatusPending,
	}

	if err := ctr.DB.Create(&stk).Error; err != nil {
		return nil, fmt.Errorf("failed to create STK request: %w", err)
	}

	// Respond immediately
	resp := &data.STKPushResponse{
		ReferenceCode: code,
		Message:       fmt.Sprintf("Payment prompt sent. If not received, use Paybill %s with %s as account number.", ctr.Config.PaybillNumber, code),
	}

	stkData := data.STKPushRequest{
		Phone:         stk.MSISDN,
		Amount:        stk.Amount,
		ReferenceCode: stk.ReferenceCode,
		SessionID:     stk.SessionID,
		StkID:         int64(stk.ID),
		Combination:   req.Combination,
	}

	if err := ctr.CallSafaricomSTK(ctx, stkData); err != nil {
		logrus.WithContext(ctx).WithError(err).Error("STK push failed")

		ctr.DB.Model(&stk).Update("status", data.STKStatusFailed)

		return nil, err
	}

	return resp, nil
}

// CallSafaricomSTK performs the external M-PESA STK push API call.
func (ctr *Controller) CallSafaricomSTK(ctx context.Context, stk data.STKPushRequest) error {
	logrus.WithContext(ctx).Infof("Sending STK push to %d for KES %.2f (session %d)",
		stk.Phone, stk.Amount, stk.SessionID)

	// Load environment variables
	shortCode := os.Getenv("c2b_paybill")
	passKey := os.Getenv("c2b_passkey")
	transactionType := os.Getenv("transaction_type")
	consumerKey := os.Getenv("c2b_consumer_key")
	consumerSecret := os.Getenv("c2b_consumer_secret")
	mpesaExpressUrl := os.Getenv("mpesa_stk_push_endpoint")
	mpesaExpressCallbackUrl := os.Getenv("mpesa_stk_push_callback")

	// Get access token
	accessToken, err := ctr.getAccessToken(ctx, shortCode, consumerKey, consumerSecret)
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	// Construct callback URL
	callbackURL := strings.Replace(mpesaExpressCallbackUrl, "{transactionID}", fmt.Sprintf("%d", stk.StkID), 1)

	// Build password
	timestamp := library.MpesaTimestamp()
	auth := fmt.Sprintf("%s%s%s", shortCode, passKey, timestamp)
	password := base64.StdEncoding.EncodeToString([]byte(auth))

	partyB := os.Getenv("c2b_party_b")
	if partyB == "" {
		partyB = shortCode
	}

	logrus.WithContext(ctx).Infof("STK Push details - ShortCode: %s, Timestamp: %s, Password: [REDACTED], Amount: %v, PartyA: %d, PartyB: %s, CallbackURL: %s accessToken:%s",
		shortCode, timestamp, stk.Amount, stk.Phone, partyB, callbackURL, accessToken,
	)

	// Build request payload
	request := data.MPESAExpressRequest{
		BusinessShortCode: shortCode,
		Password:          password,
		Timestamp:         timestamp,
		TransactionType:   transactionType,
		Amount:            fmt.Sprintf("%d", int64(stk.Amount)),
		PartyA:            fmt.Sprintf("%d", stk.Phone),
		PartyB:            partyB,
		PhoneNumber:       fmt.Sprintf("%d", stk.Phone),
		CallBackURL:       callbackURL,
		AccountReference:  stk.Combination,
		TransactionDesc:   fmt.Sprintf("Bill #%d", stk.SessionID),
	}

	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", accessToken),
		"Content-Type":  "application/json",
	}
	jsonRequest, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal STK request: %w", err)
	}
	log.Printf("stk request payload %s", string(jsonRequest))
	// Send HTTP POST request
	//status, response, err := library.HttpPostJSON(mpesaExpressUrl, headers, request)
	response, status, err := library.HTTPRequest(ctx, "POST", mpesaExpressUrl, request, headers)
	if err != nil {
		return fmt.Errorf("failed to make payment request: %w", err)
	}

	if status < 200 || status >= 300 {
		logrus.WithContext(ctx).Errorf("Invalid status from M-PESA API: %d | %s", status, string(response))
		return fmt.Errorf("invalid response status from M-PESA API: %d", status)
	}

	// Parse response
	var payload map[string]interface{}
	if err := json.Unmarshal(response, &payload); err != nil {
		return fmt.Errorf("failed to parse M-PESA response: %w", err)
	}

	logrus.WithContext(ctx).Debugf("STK Push response: %+v", payload)

	// Process response
	errorCode := library.GetString(payload, "errorCode")
	update := map[string]interface{}{}

	if errorCode != "" {
		update = map[string]interface{}{
			"status":              -1,
			"description":         library.GetString(payload, "errorMessage"),
			"merchant_request_id": "",
			"checkout_request_id": library.GetString(payload, "requestId"),
			"result_code":         errorCode,
			"result_description":  library.GetString(payload, "errorMessage"),
		}
	} else {
		update = map[string]interface{}{
			"status":              1,
			"description":         library.GetString(payload, "ResponseDescription"),
			"merchant_request_id": library.GetString(payload, "MerchantRequestID"),
			"checkout_request_id": library.GetString(payload, "CheckoutRequestID"),
			"result_code":         library.GetString(payload, "ResponseCode"),
			"result_description":  library.GetString(payload, "ResponseDescription"),
		}
	}

	// Update database
	if err := ctr.DB.Model(&models.STKPushRequest{}).
		Where("session_id = ?", stk.SessionID).
		Updates(update).Error; err != nil {
		return fmt.Errorf("failed to update STK push request: %w", err)
	}

	return nil
}

// getAccessToken retrieves or generates an M-PESA access token with Redis caching.
func (ctr *Controller) getAccessToken(ctx context.Context, shortCode, consumerKey, consumerSecret string) (string, error) {
	redisKey := fmt.Sprintf("mpesa-paybill:%s:access-token", shortCode)

	// Try to get from cache
	if cachedToken, err := library.GetRedisKey(ctr.Redis, redisKey); err == nil && len(cachedToken) > 0 {
		return cachedToken, nil
	}

	// Generate new token
	url := os.Getenv("mpesa_access_token_url")
	auth := fmt.Sprintf("%s:%s", consumerKey, consumerSecret)
	basicAuth := base64.StdEncoding.EncodeToString([]byte(auth))

	headers := map[string]string{
		"Authorization": fmt.Sprintf("Basic %s", basicAuth),
	}

	params := map[string]string{
		"grant_type": "client_credentials",
	}

	status, response := library.HTTPGet(url, headers, params)
	if status < 200 || status > 210 {
		return "", fmt.Errorf("invalid status %d | %s", status, response)
	}

	var payload data.AccessTokenPayload
	if err := json.Unmarshal([]byte(response), &payload); err != nil {
		logrus.WithContext(ctx).Errorf("Error parsing access token response: %s", err.Error())
		return "", fmt.Errorf("failed to parse access token response: %w", err)
	}

	// Parse expiry and cache token
	expiryInSeconds, err := strconv.ParseInt(payload.ExpiresIn, 10, 64)
	if err != nil {
		logrus.WithContext(ctx).Warnf("Error parsing expiry time: %s, using default", err.Error())
		expiryInSeconds = 3000
	}

	// Use 75% of expiry time for safety margin
	expiryInSeconds = int64(float64(expiryInSeconds) * 0.75)

	if err := library.SetRedisKeyWithExpiry(ctr.Redis, redisKey, payload.AccessToken, int(expiryInSeconds)); err != nil {
		logrus.WithContext(ctx).Warnf("Failed to cache access token: %s", err.Error())
	}

	return payload.AccessToken, nil
}
