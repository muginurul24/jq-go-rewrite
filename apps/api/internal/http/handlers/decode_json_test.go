package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSONAllowsEmptyBody(t *testing.T) {
	t.Parallel()

	testBodies := []string{"", "null", "[]", "{}"}
	for _, body := range testBodies {
		request := httptest.NewRequest("POST", "/api/v1/money/info", strings.NewReader(body))
		var payload struct {
			Username *string `json:"username"`
		}

		if err := decodeJSON(request, &payload); err != nil {
			t.Fatalf("decodeJSON(%q) error = %v", body, err)
		}
	}
}

func TestDecodeJSONRejectsUnknownFields(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest("POST", "/api/v1/money/info", strings.NewReader(`{"unknown":true}`))
	var payload struct {
		Username *string `json:"username"`
	}

	if err := decodeJSON(request, &payload); err == nil {
		t.Fatal("decodeJSON should reject unknown fields")
	}
}
