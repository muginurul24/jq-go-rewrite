package nexusggrtopup

import "testing"

func TestResolveTopupRatioUsesTieredThreshold(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		amount int64
		want   int64
	}{
		{name: "below threshold", amount: 500_000, want: defaultTopupRatio},
		{name: "exact threshold", amount: discountedTopupThreshold, want: defaultTopupRatio},
		{name: "above threshold", amount: discountedTopupThreshold + 1, want: discountedTopupRatio},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := ResolveTopupRatio(tc.amount); got != tc.want {
				t.Fatalf("ResolveTopupRatio(%d) = %d, want %d", tc.amount, got, tc.want)
			}
		})
	}
}
