package kernel

import (
	"errors"
	"math"
	"testing"
)

// Sprint 14, E6: an ingredient may carry the weight of one litre and the weight of one unit, and then it
// can be used in the other families (gr <-> ml, und <-> gr). Without them the families stay apart.

func TestE6_ConvertThroughTheEquivalences(t *testing.T) {
	eq := Equivalence{GramsPerLiter: 1250, GramsPerUnit: 60}
	cases := []struct {
		name     string
		quantity float64
		from, to string
		want     float64
	}{
		{"625 gr of a 1250 gr/lt liquid are half a litre", 625, "gr", "lt", 0.5},
		{"the same in ml", 625, "gr", "ml", 500},
		{"a litre weighs 1250 gr", 1, "lt", "gr", 1250},
		{"half a kilo in ml", 0.5, "kg", "ml", 400},
		{"2 units of 60 gr", 2, "und", "gr", 120},
		{"120 gr are 2 units", 120, "gr", "und", 2},
		{"a kilo in units", 1.2, "kg", "und", 20},
		{"a litre in units goes through the weight", 1.2, "lt", "und", 25},
		{"the same family keeps working", 1.5, "kg", "gr", 1500},
		{"the same unit is untouched", 7, "ml", "ml", 7},
		{"zero stays zero", 0, "gr", "ml", 0},
	}
	for _, c := range cases {
		got, err := Convert(c.quantity, c.from, c.to, eq)
		if err != nil {
			t.Errorf("%s: unexpected error %v", c.name, err)
			continue
		}
		if math.Abs(got-c.want) > 1e-9 {
			t.Errorf("%s: %v %s -> %s = %v, want %v", c.name, c.quantity, c.from, c.to, got, c.want)
		}
	}
}

func TestE6_WithoutTheEquivalenceTheErrorSaysWhichOne(t *testing.T) {
	cases := []struct {
		name     string
		from, to string
		eq       Equivalence
		need     string
	}{
		{"gr to ml without the litre", "gr", "ml", Equivalence{}, "grams_per_liter"},
		{"lt to kg without the litre", "lt", "kg", Equivalence{GramsPerUnit: 60}, "grams_per_liter"},
		{"und to gr without the unit", "und", "gr", Equivalence{}, "grams_per_unit"},
		{"gr to und without the unit", "gr", "und", Equivalence{GramsPerLiter: 1250}, "grams_per_unit"},
		{"ml to und with only the litre", "ml", "und", Equivalence{GramsPerLiter: 1250}, "grams_per_unit"},
		{"ml to und with only the unit", "ml", "und", Equivalence{GramsPerUnit: 60}, "grams_per_liter"},
	}
	for _, c := range cases {
		_, err := Convert(1, c.from, c.to, c.eq)
		var need *EquivalenceError
		if !errors.As(err, &need) {
			t.Errorf("%s: error = %v, want an EquivalenceError", c.name, err)
			continue
		}
		if need.Need != c.need {
			t.Errorf("%s: needs %q, want %q", c.name, need.Need, c.need)
		}
	}
}

func TestE6_UnknownUnitsAreStillErrors(t *testing.T) {
	_, err := Convert(1, "taza", "gr", Equivalence{GramsPerLiter: 1000, GramsPerUnit: 50})
	if err == nil {
		t.Fatal("an unknown unit must be an error")
	}
	var need *EquivalenceError
	if errors.As(err, &need) {
		t.Error("an unknown unit is not a missing equivalence")
	}
}

func TestE6_TheMessagesAreInPlainSpanish(t *testing.T) {
	_, err := Convert(1, "gr", "ml", Equivalence{})
	if err == nil || !containsFold(err.Error(), "peso de 1 litro") {
		t.Errorf("message = %v, want it to ask for the weight of 1 litre", err)
	}
	_, err = Convert(1, "und", "gr", Equivalence{})
	if err == nil || !containsFold(err.Error(), "peso de 1 unidad") {
		t.Errorf("message = %v, want it to ask for the weight of 1 unit", err)
	}
}

func containsFold(s, sub string) bool {
	ls, lsub := []rune(s), []rune(sub)
	for i := 0; i+len(lsub) <= len(ls); i++ {
		match := true
		for j := range lsub {
			a, b := ls[i+j], lsub[j]
			if a != b && a+32 != b && a != b+32 {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
