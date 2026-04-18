package operationalfees

import "testing"

func TestCalculateOperationalFeeDeduction(t *testing.T) {
	testCases := []struct {
		name   string
		settle int64
		want   int64
	}{
		{name: "zero settle", settle: 0, want: 0},
		{name: "negative settle", settle: -1, want: 0},
		{name: "less than monthly fee", settle: 50_000, want: 50_000},
		{name: "exact monthly fee", settle: 100_000, want: 100_000},
		{name: "more than monthly fee", settle: 250_000, want: 100_000},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := calculateOperationalFeeDeduction(testCase.settle)
			if got != testCase.want {
				t.Fatalf("calculateOperationalFeeDeduction(%d) = %d, want %d", testCase.settle, got, testCase.want)
			}
		})
	}
}
