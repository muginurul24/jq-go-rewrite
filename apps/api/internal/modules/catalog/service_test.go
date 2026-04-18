package catalog

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	nexusggrintegration "github.com/mugiew/justqiuv2-rewrite/apps/api/internal/integrations/nexusggr"
)

type fakeUpstreamClient struct {
	providerCalls int
	gameCalls     int
	gameV2Calls   int

	providerResponse *nexusggrintegration.ProviderListResponse
	gameResponse     *nexusggrintegration.GameListResponse
	gameV2Response   *nexusggrintegration.GameListResponse
}

func (f *fakeUpstreamClient) ProviderList(_ context.Context) (*nexusggrintegration.ProviderListResponse, error) {
	f.providerCalls++
	return f.providerResponse, nil
}

func (f *fakeUpstreamClient) GameList(_ context.Context, _ string) (*nexusggrintegration.GameListResponse, error) {
	f.gameCalls++
	return f.gameResponse, nil
}

func (f *fakeUpstreamClient) GameListV2(_ context.Context, _ string) (*nexusggrintegration.GameListResponse, error) {
	f.gameV2Calls++
	return f.gameV2Response, nil
}

func TestProviderListCachesAndSanitizesRecords(t *testing.T) {
	t.Parallel()

	cache := newTestRedis(t)
	upstream := &fakeUpstreamClient{
		providerResponse: &nexusggrintegration.ProviderListResponse{
			Status: 1,
			Providers: []map[string]any{
				{
					"code":   "PGSOFT",
					"name":   "PG Soft",
					"status": 1,
					"secret": "hidden",
				},
			},
		},
	}

	service := NewService(cache, upstream)

	firstResponse, err := service.ProviderList(context.Background())
	if err != nil {
		t.Fatalf("ProviderList() error = %v", err)
	}
	secondResponse, err := service.ProviderList(context.Background())
	if err != nil {
		t.Fatalf("ProviderList() second call error = %v", err)
	}

	if upstream.providerCalls != 1 {
		t.Fatalf("expected 1 upstream provider call, got %d", upstream.providerCalls)
	}

	if len(firstResponse.Providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(firstResponse.Providers))
	}

	if firstResponse.Providers[0].Code != "PGSOFT" || firstResponse.Providers[0].Name != "PG Soft" || firstResponse.Providers[0].Status != 1 {
		t.Fatalf("unexpected provider payload: %#v", firstResponse.Providers[0])
	}

	if len(secondResponse.Providers) != 1 {
		t.Fatalf("expected cached provider response, got %#v", secondResponse.Providers)
	}
}

func TestGameListCachesAndSanitizesRecords(t *testing.T) {
	t.Parallel()

	cache := newTestRedis(t)
	upstream := &fakeUpstreamClient{
		gameResponse: &nexusggrintegration.GameListResponse{
			Status: 1,
			Games: []map[string]any{
				{
					"id":        1,
					"game_code": "mahjong",
					"game_name": "Mahjong Ways",
					"banner":    "https://cdn.test/mahjong.png",
					"status":    1,
					"secret":    "hidden",
				},
			},
		},
	}

	service := NewService(cache, upstream)

	firstResponse, err := service.GameList(context.Background(), "PGSOFT")
	if err != nil {
		t.Fatalf("GameList() error = %v", err)
	}
	secondResponse, err := service.GameList(context.Background(), "PGSOFT")
	if err != nil {
		t.Fatalf("GameList() second call error = %v", err)
	}

	if upstream.gameCalls != 1 {
		t.Fatalf("expected 1 upstream game call, got %d", upstream.gameCalls)
	}

	if firstResponse.ProviderCode != "PGSOFT" || len(firstResponse.Games) != 1 {
		t.Fatalf("unexpected game list payload: %#v", firstResponse)
	}

	if firstResponse.Games[0].Banner != "https://cdn.test/mahjong.png" || firstResponse.Games[0].Status != 1 {
		t.Fatalf("unexpected game record: %#v", firstResponse.Games[0])
	}

	if len(secondResponse.Games) != 1 {
		t.Fatalf("expected cached game response, got %#v", secondResponse.Games)
	}
}

func TestGameListV2BypassesCacheAndKeepsLocalizedNames(t *testing.T) {
	t.Parallel()

	cache := newTestRedis(t)
	upstream := &fakeUpstreamClient{
		gameV2Response: &nexusggrintegration.GameListResponse{
			Status: 1,
			Games: []map[string]any{
				{
					"id":        1,
					"game_code": "mahjong",
					"game_name": map[string]any{
						"en": "Mahjong Ways",
						"id": "Mahjong Ways",
					},
				},
			},
		},
	}

	service := NewService(cache, upstream)

	firstResponse, err := service.GameListV2(context.Background(), "PGSOFT")
	if err != nil {
		t.Fatalf("GameListV2() error = %v", err)
	}
	_, err = service.GameListV2(context.Background(), "PGSOFT")
	if err != nil {
		t.Fatalf("GameListV2() second call error = %v", err)
	}

	if upstream.gameV2Calls != 2 {
		t.Fatalf("expected v2 endpoint to bypass cache, got %d upstream calls", upstream.gameV2Calls)
	}

	localizedName, ok := firstResponse.Games[0].GameName.(map[string]any)
	if !ok {
		t.Fatalf("expected localized game_name map, got %#v", firstResponse.Games[0].GameName)
	}

	if localizedName["en"] != "Mahjong Ways" || localizedName["id"] != "Mahjong Ways" {
		t.Fatalf("unexpected localized names: %#v", localizedName)
	}
}

func newTestRedis(t *testing.T) *redis.Client {
	t.Helper()

	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
		server.Close()
	})

	return client
}
