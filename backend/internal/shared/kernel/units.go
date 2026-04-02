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
