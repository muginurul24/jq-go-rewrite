package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	nexusggrintegration "github.com/mugiew/justqiuv2-rewrite/apps/api/internal/integrations/nexusggr"
)

const cacheTTL = 24 * time.Hour

var ErrUpstreamFailure = errors.New("upstream failure")

type upstreamClient interface {
	ProviderList(ctx context.Context) (*nexusggrintegration.ProviderListResponse, error)
	GameList(ctx context.Context, providerCode string) (*nexusggrintegration.GameListResponse, error)
	GameListV2(ctx context.Context, providerCode string) (*nexusggrintegration.GameListResponse, error)
}

type Service struct {
	cache  *redis.Client
	client upstreamClient
}

type ProviderRecord struct {
	Code   any `json:"code"`
	Name   any `json:"name"`
	Status any `json:"status"`
}

type GameRecord struct {
	ID       any `json:"id"`
	GameCode any `json:"game_code"`
	GameName any `json:"game_name"`
	Banner   any `json:"banner"`
	Status   any `json:"status"`
}

type LocalizedGameRecord struct {
	ID       any `json:"id"`
	GameCode any `json:"game_code"`
	GameName any `json:"game_name"`
}

type ProviderListResponse struct {
	Success   bool             `json:"success"`
	Providers []ProviderRecord `json:"providers"`
}

type GameListResponse struct {
	Success      bool         `json:"success"`
	ProviderCode string       `json:"provider_code"`
	Games        []GameRecord `json:"games"`
}

type GameListV2Response struct {
	Success      bool                  `json:"success"`
	ProviderCode string                `json:"provider_code"`
	Games        []LocalizedGameRecord `json:"games"`
}

func NewService(cache *redis.Client, client upstreamClient) *Service {
	return &Service{
		cache:  cache,
		client: client,
	}
}

func (s *Service) ProviderList(ctx context.Context) (*ProviderListResponse, error) {
	const cacheKey = "nexusggr:provider-list"

	var cachedResponse ProviderListResponse
	if s.loadCachedJSON(ctx, cacheKey, &cachedResponse) {
		return &cachedResponse, nil
	}

	upstreamResponse, err := s.client.ProviderList(ctx)
	if err != nil {
		return nil, fmt.Errorf("provider list: %w", err)
	}
	if upstreamResponse.Status != 1 {
		return nil, ErrUpstreamFailure
	}

	response := &ProviderListResponse{
		Success:   true,
		Providers: mapProviderRecords(upstreamResponse.Providers),
	}
	s.storeCachedJSON(ctx, cacheKey, response)

	return response, nil
}

func (s *Service) GameList(ctx context.Context, providerCode string) (*GameListResponse, error) {
	cacheKey := "nexusggr:game-list:" + providerCode

	var cachedResponse GameListResponse
	if s.loadCachedJSON(ctx, cacheKey, &cachedResponse) {
		return &cachedResponse, nil
	}

	upstreamResponse, err := s.client.GameList(ctx, providerCode)
	if err != nil {
		return nil, fmt.Errorf("game list: %w", err)
	}
	if upstreamResponse.Status != 1 {
		return nil, ErrUpstreamFailure
	}

	response := &GameListResponse{
		Success:      true,
		ProviderCode: providerCode,
		Games:        mapGameRecords(upstreamResponse.Games),
	}
	s.storeCachedJSON(ctx, cacheKey, response)

	return response, nil
}

func (s *Service) GameListV2(ctx context.Context, providerCode string) (*GameListV2Response, error) {
	upstreamResponse, err := s.client.GameListV2(ctx, providerCode)
	if err != nil {
		return nil, fmt.Errorf("localized game list: %w", err)
	}
	if upstreamResponse.Status != 1 {
		return nil, ErrUpstreamFailure
	}

	return &GameListV2Response{
		Success:      true,
		ProviderCode: providerCode,
		Games:        mapLocalizedGameRecords(upstreamResponse.Games),
	}, nil
}

func mapProviderRecords(records []map[string]any) []ProviderRecord {
	mappedRecords := make([]ProviderRecord, 0, len(records))
	for _, record := range records {
		mappedRecords = append(mappedRecords, ProviderRecord{
			Code:   record["code"],
			Name:   record["name"],
			Status: record["status"],
		})
	}

	return mappedRecords
}

func mapGameRecords(records []map[string]any) []GameRecord {
	mappedRecords := make([]GameRecord, 0, len(records))
	for _, record := range records {
		mappedRecords = append(mappedRecords, GameRecord{
			ID:       record["id"],
			GameCode: record["game_code"],
			GameName: record["game_name"],
			Banner:   record["banner"],
			Status:   record["status"],
		})
	}

	return mappedRecords
}

func mapLocalizedGameRecords(records []map[string]any) []LocalizedGameRecord {
	mappedRecords := make([]LocalizedGameRecord, 0, len(records))
	for _, record := range records {
		mappedRecords = append(mappedRecords, LocalizedGameRecord{
			ID:       record["id"],
			GameCode: record["game_code"],
			GameName: record["game_name"],
		})
	}

	return mappedRecords
}

func (s *Service) loadCachedJSON(ctx context.Context, key string, target any) bool {
	if s.cache == nil {
		return false
	}

	payload, err := s.cache.Get(ctx, key).Bytes()
	if err != nil {
		return false
	}

	return json.Unmarshal(payload, target) == nil
}

func (s *Service) storeCachedJSON(ctx context.Context, key string, value any) {
	if s.cache == nil {
		return
	}

	payload, err := json.Marshal(value)
	if err != nil {
		return
	}

	_ = s.cache.Set(ctx, key, payload, cacheTTL).Err()
}
