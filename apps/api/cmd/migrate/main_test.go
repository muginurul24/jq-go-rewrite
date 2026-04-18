package main

import "testing"

func TestNeedsGooseVersionTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		command      string
		resetAndSeed bool
		want         bool
	}{
		{name: "seed flag", command: "up", resetAndSeed: true, want: true},
		{name: "seed command", command: "seed", want: true},
		{name: "down", command: "down", want: true},
		{name: "redo", command: "redo", want: true},
		{name: "reset", command: "reset", want: true},
		{name: "status", command: "status", want: true},
		{name: "up", command: "up", want: false},
		{name: "baseline", command: "baseline", want: false},
		{name: "version", command: "version", want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := needsGooseVersionTable(tt.command, tt.resetAndSeed)
			if got != tt.want {
				t.Fatalf("needsGooseVersionTable(%q, %t) = %t, want %t", tt.command, tt.resetAndSeed, got, tt.want)
			}
		})
	}
}
