package kernel

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// InUseError says that something cannot be deleted because other records depend on it. Message is meant
// for the person ("No se puede eliminar: se usa en las recetas Pan y Torta").
type InUseError struct{ Message string }

func (e *InUseError) Error() string { return e.Message }

// IsForeignKeyViolation reports whether err is the database refusing a delete because of a reference.
func IsForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

// Count writes a quantity with its noun in the right number: Count(1, "venta", "ventas") is "1 venta".
func Count(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return itoa(n) + " " + many
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for ; n > 0; n /= 10 {
		i--
		b[i] = byte('0' + n%10)
	}
	return string(b[i:])
}
