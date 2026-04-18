package dashboard

import (
	"encoding/json"
	"testing"
)

func TestToInt64RoundsDecimalValues(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		value any
		want  int64
	}{
		{
			name:  "json number decimal",
			value: json.Number("50935716.87"),
			want:  50935717,
		},
		{
			name:  "string decimal",
			value: "50935716.87",
			want:  50935717,
		},
		{
			name:  "float64 decimal",
			value: 50935716.87,
			want:  50935717,
		},
		{
			name:  "int value unchanged",
			value: 1500000,
			want:  1500000,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := toInt64(tc.value)
			if got != tc.want {
				t.Fatalf("toInt64(%v) = %d, want %d", tc.value, got, tc.want)
			}
		})
	}
}
