package qris

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/config"
)

const (
	requestTimeout = 30 * time.Second
	retryDelay     = 100 * time.Millisecond
	maxAttempts    = 3
)

type Client struct {
	baseURL    string
	client     string
	clientKey  string
	globalUUID string
	httpClient *http.Client
}

func NewClient(cfg config.QRISConfig) *Client {
	return &Client{
		baseURL:    strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"),
		client:     strings.TrimSpace(cfg.Client),
		clientKey:  strings.TrimSpace(cfg.ClientKey),
		globalUUID: strings.TrimSpace(cfg.GlobalUUID),
		httpClient: &http.Client{Timeout: requestTimeout},
	}
}

func (c *Client) Generate(ctx context.Context, username string, amount int64, expire int, customRef *string) (*GenerateResponse, error) {
	params := map[string]any{
		"username": username,
		"amount":   amount,
		"uuid":     c.globalUUID,
		"expire":   expire,
	}
	if customRef != nil && strings.TrimSpace(*customRef) != "" {
		params["custom_ref"] = strings.TrimSpace(*customRef)
	}

	var response GenerateResponse
	if err := c.call(ctx, http.MethodPost, "/api/generate", params, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) CheckStatus(ctx context.Context, trxID string) (*CheckStatusResponse, error) {
	var response CheckStatusResponse
	if err := c.call(ctx, http.MethodPost, "/api/checkstatus/v2/"+trxID, map[string]any{
		"uuid":       c.globalUUID,
		"client":     c.client,
		"client_key": c.clientKey,
	}, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) Balance(ctx context.Context) (*BalanceResponse, error) {
	var response BalanceResponse
	if err := c.call(ctx, http.MethodPost, "/api/balance/"+c.globalUUID, map[string]any{
		"client": c.client,
	}, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) Inquiry(ctx context.Context, amount int64, bankCode string, accountNumber string, transferType int) (*InquiryResponse, error) {
	var response InquiryResponse
	if err := c.call(ctx, http.MethodPost, "/api/inquiry", map[string]any{
		"client":         c.client,
		"client_key":     c.clientKey,
		"uuid":           c.globalUUID,
		"amount":         amount,
		"bank_code":      strings.TrimSpace(bankCode),
		"account_number": strings.TrimSpace(accountNumber),
		"type":           transferType,
	}, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) Transfer(ctx context.Context, amount int64, bankCode string, accountNumber string, transferType int, inquiryID int64) (*TransferResponse, error) {
	var response TransferResponse
	if err := c.call(ctx, http.MethodPost, "/api/transfer", map[string]any{
		"client":         c.client,
		"client_key":     c.clientKey,
		"uuid":           c.globalUUID,
		"amount":         amount,
		"bank_code":      strings.TrimSpace(bankCode),
		"account_number": strings.TrimSpace(accountNumber),
		"type":           transferType,
		"inquiry_id":     inquiryID,
	}, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) call(ctx context.Context, method string, path string, payload any, target any) error {
	if c.baseURL == "" {
		return errors.New("qris base url is not configured")
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal qris payload: %w", err)
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("create qris request: %w", err)
		}

		request.Header.Set("Accept", "application/json")
		request.Header.Set("Content-Type", "application/json")

		response, err := c.httpClient.Do(request)
		if err == nil {
			lastErr = decodeResponse(response, target)
			_ = response.Body.Close()
			if lastErr == nil {
				return nil
			}
		} else {
			lastErr = err
		}

		if attempt < maxAttempts {
			if err := sleepWithContext(ctx, retryDelay); err != nil {
				return fmt.Errorf("call qris path %s: %w", path, err)
			}
		}
	}

	return fmt.Errorf("call qris path %s: %w", path, lastErr)
}

func decodeResponse(response *http.Response, target any) error {
	decoder := json.NewDecoder(response.Body)
	decoder.UseNumber()

	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode qris response: %w", err)
	}

	return nil
}

func sleepWithContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
