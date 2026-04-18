package settlements

import "testing"

func TestCalculateSettlementAmountMatchesLegacyRounding(t *testing.T) {
	testCases := []struct {
		name    string
		pending int64
		want    int64
	}{
		{name: "zero pending", pending: 0, want: 0},
		{name: "one rounds up", pending: 1, want: 1},
		{name: "two rounds down", pending: 2, want: 1},
		{name: "five rounds half up", pending: 5, want: 4},
		{name: "ten settles seventy percent", pending: 10, want: 7},
		{name: "odd amount", pending: 11, want: 8},
		{name: "large amount", pending: 1250000, want: 875000},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := calculateSettlementAmount(testCase.pending)
			if got != testCase.want {
				t.Fatalf("calculateSettlementAmount(%d) = %d, want %d", testCase.pending, got, testCase.want)
			}
		})
	}
}
