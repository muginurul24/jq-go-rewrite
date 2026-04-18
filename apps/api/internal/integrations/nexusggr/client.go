package nexusggr

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
	agentCode  string
	agentToken string
	httpClient *http.Client
}

func NewClient(cfg config.NexusGGRConfig) *Client {
	return &Client{
		baseURL:    strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"),
		agentCode:  strings.TrimSpace(cfg.AgentCode),
		agentToken: strings.TrimSpace(cfg.AgentToken),
		httpClient: &http.Client{Timeout: requestTimeout},
	}
}

func (c *Client) ProviderList(ctx context.Context) (*ProviderListResponse, error) {
	var response ProviderListResponse
	if err := c.call(ctx, "provider_list", nil, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) GameList(ctx context.Context, providerCode string) (*GameListResponse, error) {
	var response GameListResponse
	if err := c.call(ctx, "game_list", map[string]any{
		"provider_code": providerCode,
	}, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) GameListV2(ctx context.Context, providerCode string) (*GameListResponse, error) {
	var response GameListResponse
	if err := c.call(ctx, "game_list_v2", map[string]any{
		"provider_code": providerCode,
	}, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) GameLaunch(ctx context.Context, userCode string, providerCode string, lang string, gameCode *string) (*GameLaunchResponse, error) {
	params := map[string]any{
		"user_code":     userCode,
		"provider_code": providerCode,
		"lang":          lang,
	}
	if gameCode != nil && strings.TrimSpace(*gameCode) != "" {
		params["game_code"] = strings.TrimSpace(*gameCode)
	}

	var response GameLaunchResponse
	if err := c.call(ctx, "game_launch", params, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) MoneyInfo(ctx context.Context, userCode *string, allUsers bool) (*MoneyInfoResponse, error) {
	params := map[string]any{}
	if userCode != nil && strings.TrimSpace(*userCode) != "" {
		params["user_code"] = strings.TrimSpace(*userCode)
	}
	if allUsers {
		params["all_users"] = true
	}

	var response MoneyInfoResponse
	if err := c.call(ctx, "money_info", params, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) GetGameLog(ctx context.Context, userCode string, gameType string, start string, end string, page int, perPage int) (*GameLogResponse, error) {
	var response GameLogResponse
	if err := c.call(ctx, "get_game_log", map[string]any{
		"user_code": userCode,
		"game_type": gameType,
		"start":     start,
		"end":       end,
		"page":      page,
		"perPage":   perPage,
	}, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) UserCreate(ctx context.Context, userCode string) (*UserCreateResponse, error) {
	var response UserCreateResponse
	if err := c.call(ctx, "user_create", map[string]any{
		"user_code": userCode,
	}, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) UserDeposit(ctx context.Context, userCode string, amount int64, agentSign *string) (*UserBalanceMutationResponse, error) {
	params := map[string]any{
		"user_code": userCode,
		"amount":    amount,
	}
	if agentSign != nil && strings.TrimSpace(*agentSign) != "" {
		params["agent_sign"] = strings.TrimSpace(*agentSign)
	}

	var response UserBalanceMutationResponse
	if err := c.call(ctx, "user_deposit", params, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) UserWithdraw(ctx context.Context, userCode string, amount int64, agentSign *string) (*UserBalanceMutationResponse, error) {
	params := map[string]any{
		"user_code": userCode,
		"amount":    amount,
	}
	if agentSign != nil && strings.TrimSpace(*agentSign) != "" {
		params["agent_sign"] = strings.TrimSpace(*agentSign)
	}

	var response UserBalanceMutationResponse
	if err := c.call(ctx, "user_withdraw", params, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) UserWithdrawReset(ctx context.Context, userCode *string, allUsers bool) (*UserWithdrawResetResponse, error) {
	params := map[string]any{}
	if userCode != nil && strings.TrimSpace(*userCode) != "" {
		params["user_code"] = strings.TrimSpace(*userCode)
	}
	if allUsers {
		params["all_users"] = true
	}

	var response UserWithdrawResetResponse
	if err := c.call(ctx, "user_withdraw_reset", params, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) TransferStatus(ctx context.Context, userCode string, agentSign string) (*TransferStatusResponse, error) {
	var response TransferStatusResponse
	if err := c.call(ctx, "transfer_status", map[string]any{
		"user_code":  userCode,
		"agent_sign": agentSign,
	}, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) CallPlayers(ctx context.Context) (*CallPlayersResponse, error) {
	var response CallPlayersResponse
	if err := c.call(ctx, "call_players", nil, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) CallList(ctx context.Context, providerCode string, gameCode string) (*CallListResponse, error) {
	var response CallListResponse
	if err := c.call(ctx, "call_list", map[string]any{
		"provider_code": providerCode,
		"game_code":     gameCode,
	}, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) CallApply(ctx context.Context, providerCode string, gameCode string, userCode string, callRTP int, callType int) (*CallApplyResponse, error) {
	var response CallApplyResponse
	if err := c.call(ctx, "call_apply", map[string]any{
		"provider_code": providerCode,
		"game_code":     gameCode,
		"user_code":     userCode,
		"call_rtp":      callRTP,
		"call_type":     callType,
	}, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) CallHistory(ctx context.Context, offset int, limit int) (*CallHistoryResponse, error) {
	var response CallHistoryResponse
	if err := c.call(ctx, "call_history", map[string]any{
		"offset": offset,
		"limit":  limit,
	}, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) CallCancel(ctx context.Context, callID int) (*CallCancelResponse, error) {
	var response CallCancelResponse
	if err := c.call(ctx, "call_cancel", map[string]any{
		"call_id": callID,
	}, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) ControlRtp(ctx context.Context, providerCode string, userCode string, rtp float64) (*ControlRtpResponse, error) {
	var response ControlRtpResponse
	if err := c.call(ctx, "control_rtp", map[string]any{
		"provider_code": providerCode,
		"user_code":     userCode,
		"rtp":           rtp,
	}, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) ControlUsersRtp(ctx context.Context, userCodes []string, rtp float64) (*ControlRtpResponse, error) {
	encodedUserCodes, err := json.Marshal(userCodes)
	if err != nil {
		return nil, fmt.Errorf("marshal user codes: %w", err)
	}

	var response ControlRtpResponse
	if err := c.call(ctx, "control_users_rtp", map[string]any{
		"user_codes": string(encodedUserCodes),
		"rtp":        rtp,
	}, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) call(ctx context.Context, method string, params map[string]any, target any) error {
	if c.baseURL == "" {
		return errors.New("nexusggr base url is not configured")
	}

	payload := map[string]any{
		"method":      method,
		"agent_code":  c.agentCode,
		"agent_token": c.agentToken,
	}
	for key, value := range params {
		payload[key] = value
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal nexusggr payload: %w", err)
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/", bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("create nexusggr request: %w", err)
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
				return fmt.Errorf("call nexusggr method %s: %w", method, err)
			}
		}
	}

	return fmt.Errorf("call nexusggr method %s: %w", method, lastErr)
}

func decodeResponse(response *http.Response, target any) error {
	decoder := json.NewDecoder(response.Body)
	decoder.UseNumber()

	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode nexusggr response: %w", err)
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
