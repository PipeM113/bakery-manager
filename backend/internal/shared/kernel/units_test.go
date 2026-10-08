package kernel

import "testing"

// AC3: characterization of ConvertToBase. These tests pin what the code does today;
// they are not a statement that every case is desirable (see the CURRENT notes).
func TestAC3_ConvertToBase(t *testing.T) {
	cases := []struct {
		name     string
		quantity float64
		from, to string
		want     float64
	}{
		{"kg to gr", 1.5, "kg", "gr", 1500},
		{"gr to kg", 250, "gr", "kg", 0.25},
		{"lt to ml", 2, "lt", "ml", 2000},
		{"ml to lt", 500, "ml", "lt", 0.5},
		{"same unit is untouched", 7, "gr", "gr", 7},
		{"same unknown unit is untouched", 3, "taza", "taza", 3},
		{"alias kilos", 2, "kilos", "gr", 2000},
		{"alias g", 1000, "g", "kg", 1},
		{"alias litros", 1, "litros", "ml", 1000},
		{"alias und and unidades are the same unit", 4, "und", "unidades", 4},
		{"zero stays zero", 0, "kg", "gr", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ConvertToBase(c.quantity, c.from, c.to)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Fatalf("ConvertToBase(%v, %q, %q) = %v, want %v", c.quantity, c.from, c.to, got, c.want)
			}
		})
	}
}

func TestAC3_ConvertToBaseRejectsIncompatibleUnits(t *testing.T) {
	cases := []struct{ name, from, to string }{
		{"weight to volume", "gr", "ml"},
		{"volume to weight", "lt", "kg"},
		{"unknown to weight", "taza", "gr"},
		{"weight to unit", "kg", "und"},
		// CURRENT: unit names are case-sensitive, so "Kg" is not recognised.
		{"CURRENT uppercase is not normalised", "Kg", "gr"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ConvertToBase(10, c.from, c.to)
			if err == nil {
				t.Fatalf("expected an error, got %v", got)
			}
			if got != 0 {
				t.Fatalf("on error the value must be 0, got %v", got)
			}
		})
	}
}
