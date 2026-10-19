// Package money holds the rules every amount of pesos obeys on its way in: whole pesos,
// not negative, and small enough to fit the INTEGER columns of the database.
package money

// MaxPesos is the largest amount accepted (just under what a 32 bit INTEGER column holds).
const MaxPesos int64 = 2_000_000_000

// Valid reports whether amount is an acceptable amount of pesos. Zero is accepted only
// when allowZero is true (a sale given away); everything else must be positive.
func Valid(amount int64, allowZero bool) bool {
	if amount < 0 || amount > MaxPesos {
		return false
	}
	return amount > 0 || allowZero
}
