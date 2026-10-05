package finance

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestComputeImprestIncrease(t *testing.T) {
	d := decimal.RequireFromString
	cases := []struct {
		name                   string
		imprest, balance, amt  string
		want                   string
	}{
		{"zero box: all goes to imprest", "0", "0", "500000", "500000"},
		{"roadmap QA: 1M imprest, 500k left, receives 600k", "1000000", "500000", "600000", "100000"},
		{"exact replenishment: no imprest change", "1000000", "500000", "500000", "0"},
		{"partial replenishment: no imprest change", "1000000", "500000", "200000", "0"},
		{"overspent (negative balance): deficit covered first", "1000000", "-200000", "1500000", "300000"},
		{"zero imprest, overspent: excess above deficit only", "0", "-100000", "300000", "200000"},
		{"balance above imprest: whole amount is excess", "1000000", "1000000", "250000", "250000"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ComputeImprestIncrease(d(c.imprest), d(c.balance), d(c.amt))
			if !got.Equal(d(c.want)) {
				t.Fatalf("got %s want %s", got, c.want)
			}
		})
	}
}
