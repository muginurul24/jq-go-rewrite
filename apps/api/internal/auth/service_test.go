package auth

import "testing"

func TestNormalizeTokenableType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "legacy single slash format",
			input: `App\Models\Toko`,
			want:  TokenableTypeToko,
		},
		{
			name:  "rewrite escaped double slash format",
			input: `App\\Models\\Toko`,
			want:  TokenableTypeToko,
		},
		{
			name:  "trim surrounding whitespace",
			input: "  App\\Models\\Toko  ",
			want:  TokenableTypeToko,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := normalizeTokenableType(tt.input); got != tt.want {
				t.Fatalf("normalizeTokenableType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
