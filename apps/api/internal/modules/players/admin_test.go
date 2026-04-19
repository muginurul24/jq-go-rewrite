package players

import (
	"encoding/json"
	"testing"
)

func TestExtractBalanceParsesDecimalStringsLikeLegacy(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		payload map[string]any
		want    int64
	}{
		{
			name:    "int string",
			payload: map[string]any{"balance": "9000"},
			want:    9000,
		},
		{
			name:    "decimal string",
			payload: map[string]any{"balance": "9000.87"},
			want:    9000,
		},
		{
			name:    "float",
			payload: map[string]any{"balance": 9000.87},
			want:    9000,
		},
		{
			name:    "json number integer",
			payload: map[string]any{"balance": json.Number("60")},
			want:    60,
		},
		{
			name:    "json number decimal",
			payload: map[string]any{"balance": json.Number("9000.87")},
			want:    9000,
		},
		{
			name:    "missing",
			payload: map[string]any{},
			want:    0,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := extractBalance(testCase.payload)
			if got != testCase.want {
				t.Fatalf("extractBalance(%v) = %d, want %d", testCase.payload, got, testCase.want)
			}
		})
	}
}
