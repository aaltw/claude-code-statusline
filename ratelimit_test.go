package main

import "testing"

func TestFiveHourBudget(t *testing.T) {
	const win = 5 * 60 * 60
	reset := 1_000_000.0
	cases := []struct {
		now  float64
		want int
	}{
		{reset - win, 0},        // window start
		{reset - win/2, 50},     // halfway
		{reset, 100},            // at reset
		{reset - win - 100, 0},  // before start, clamped
		{reset + 100, 100},      // past reset, clamped
	}
	for _, c := range cases {
		if got := int(fiveHourBudget(reset, c.now)); got != c.want {
			t.Errorf("fiveHourBudget(now=%.0f) = %d, want %d", c.now, got, c.want)
		}
	}
}
