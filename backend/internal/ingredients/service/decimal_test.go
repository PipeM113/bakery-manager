package service

import "testing"

// Sprint 2a, rule P8: prices and sizes in the Excel file are read the way a Chilean user
// writes them: "." groups thousands, "," separates decimals, a "$" may come in front.
func TestP8_ParseCLDecimal(t *testing.T) {
	good := []struct {
		in   string
		want float64
	}{
		{"1500", 1500},
		{" 2000 ", 2000},
		{"1.500", 1500},
		{"1.500.000", 1500000},
		{"1.500,50", 1500.5},
		{"2,5", 2.5},
		{"2.5", 2.5},
		{"0,25", 0.25},
		{"$1.500", 1500},
		{"$ 1.500", 1500},
		{"0", 0},
		{"-200", -200},
	}
	for _, c := range good {
		got, err := ParseCLDecimal(c.in)
		if err != nil {
			t.Errorf("ParseCLDecimal(%q) failed: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParseCLDecimal(%q) = %v, want %v", c.in, got, c.want)
		}
	}

	bad := []string{"", "   ", "abc", "1,2,3", "1.5.5", "1.5000.000", "12abc", "NaN", "Inf", "1e3", "--5", "1.500,5,0"}
	for _, in := range bad {
		if got, err := ParseCLDecimal(in); err == nil {
			t.Errorf("ParseCLDecimal(%q) = %v, want an error", in, got)
		}
	}
}
