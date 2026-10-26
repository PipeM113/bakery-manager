package kernel

import "fmt"

// ConvertToBase convierte una cantidad a la unidad base del insumo.
// Ejemplo: 1 kg → 1000 gr, 1 lt → 1000 ml
// Retorna error si las unidades son incompatibles (ej. gr → ml).
func ConvertToBase(quantity float64, fromUnit string, toUnit string) (float64, error) {
	from := normalize(fromUnit)
	to := normalize(toUnit)

	if from == to {
		return quantity, nil
	}

	// Peso
	if from == "kg" && to == "gr" {
		return quantity * 1000, nil
	}
	if from == "gr" && to == "kg" {
		return quantity / 1000, nil
	}

	// Volumen
	if from == "lt" && to == "ml" {
		return quantity * 1000, nil
	}
	if from == "ml" && to == "lt" {
		return quantity / 1000, nil
	}

	return 0, fmt.Errorf("no se puede convertir de %q a %q: unidades incompatibles", fromUnit, toUnit)
}

func normalize(u string) string {
	switch u {
	case "g", "gr", "grs", "gramo", "gramos":
		return "gr"
	case "k", "kg", "kilo", "kilos":
		return "kg"
	case "l", "lt", "lts", "litro", "litros":
		return "lt"
	case "ml", "mls", "mililitro", "mililitros":
		return "ml"
	case "und", "unidad", "unidades", "u", "unit", "units":
		return "und"
	default:
		return u
	}
}

// Equivalence is what lets an ingredient be used in another family of units: the weight of one litre
// (gr <-> ml) and the weight of one unit (und <-> gr). Zero means "not known"; nothing is preloaded,
// because there is no source for the density of the products a bakery buys.
type Equivalence struct {
	GramsPerLiter float64
	GramsPerUnit  float64
}

// EquivalenceError says that the conversion is possible but needs one of the two equivalences.
// Need is "grams_per_liter" or "grams_per_unit".
type EquivalenceError struct {
	Need string
	From string
	To   string
}

func (e *EquivalenceError) Error() string {
	if e.Need == "grams_per_liter" {
		return fmt.Sprintf("para pasar de %s a %s falta el peso de 1 litro del insumo", e.From, e.To)
	}
	return fmt.Sprintf("para pasar de %s a %s falta el peso de 1 unidad del insumo", e.From, e.To)
}

type family int

const (
	famUnknown family = iota
	famMass
	famVolume
	famCount
)

func familyOf(unit string) family {
	switch normalize(unit) {
	case "gr", "kg":
		return famMass
	case "ml", "lt":
		return famVolume
	case "und":
		return famCount
	}
	return famUnknown
}

// Convert converts between any two known units. Inside a family it is plain arithmetic; across families
// it goes through grams with the equivalences, and returns an *EquivalenceError when one is missing.
func Convert(quantity float64, from, to string, eq Equivalence) (float64, error) {
	ff, ft := familyOf(from), familyOf(to)
	if ff == famUnknown || ft == famUnknown {
		return 0, fmt.Errorf("no se puede convertir de %q a %q: unidades incompatibles", from, to)
	}
	if ff == ft {
		return ConvertToBase(quantity, from, to)
	}
	need := func(f family) (string, bool) {
		switch f {
		case famVolume:
			return "grams_per_liter", eq.GramsPerLiter <= 0
		case famCount:
			return "grams_per_unit", eq.GramsPerUnit <= 0
		}
		return "", false
	}
	for _, f := range []family{ff, ft} {
		if n, missing := need(f); missing {
			return 0, &EquivalenceError{Need: n, From: from, To: to}
		}
	}
	var grams float64
	switch ff {
	case famMass:
		grams, _ = ConvertToBase(quantity, from, "gr")
	case famVolume:
		ml, _ := ConvertToBase(quantity, from, "ml")
		grams = ml * eq.GramsPerLiter / 1000
	case famCount:
		grams = quantity * eq.GramsPerUnit
	}
	switch ft {
	case famMass:
		return ConvertToBase(grams, "gr", to)
	case famVolume:
		return ConvertToBase(grams*1000/eq.GramsPerLiter, "ml", to)
	}
	return grams / eq.GramsPerUnit, nil
}
