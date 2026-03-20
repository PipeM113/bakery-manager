package kernel

// ConvertToBase convierte una cantidad a la unidad base del insumo.
// Ejemplo: 1 kg → 1000 gr, 1 lt → 1000 ml
func ConvertToBase(quantity float64, fromUnit string, toUnit string) float64 {
	from := normalize(fromUnit)
	to := normalize(toUnit)

	if from == to {
		return quantity
	}

	// Peso
	if from == "kg" && to == "gr" {
		return quantity * 1000
	}
	if from == "gr" && to == "kg" {
		return quantity / 1000
	}

	// Volumen
	if from == "lt" && to == "ml" {
		return quantity * 1000
	}
	if from == "ml" && to == "lt" {
		return quantity / 1000
	}

	// Si no hay conversión posible devuelve tal cual
	return quantity
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
	default:
		return u
	}
}
