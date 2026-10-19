package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/PipeM113/bakery-manager/internal/shared/kernel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Reasons of an adjustment.
const (
	ReasonMerma      = "merma"
	ReasonCorrection = "correccion"
	ReasonOther      = "otro"
)

// ErrNotFound means the ingredient does not exist.
var ErrNotFound = errors.New("ingredient not found")

// ValidationError is a request the owner can fix.
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

type Adjustment struct {
	ID             string    `json:"id"`
	IngredientID   string    `json:"ingredient_id"`
	IngredientName string    `json:"ingredient_name"`
	Reason         string    `json:"reason"`
	QuantityDelta  float64   `json:"quantity_delta"`
	Unit           string    `json:"unit"`
	StockBefore    float64   `json:"stock_before"`
	StockAfter     float64   `json:"stock_after"`
	LossAmount     int64     `json:"loss_amount"`
	Note           string    `json:"note"`
	AdjustedOn     string    `json:"adjusted_on"`
	CreatedBy      *string   `json:"created_by"`
	CreatedAt      time.Time `json:"created_at"`
}

// Input of an adjustment: Quantity (merma), CountedQuantity (correccion) or QuantityDelta (otro).
type Input struct {
	CreatedBy       string
	IngredientID    string
	Reason          string
	Quantity        *float64
	CountedQuantity *float64
	QuantityDelta   *float64
	Unit            string
	Note            string
	AdjustedOn      string // YYYY-MM-DD, already checked by the caller
}

type Repository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func round3(x float64) float64 { return math.Round(x*1000) / 1000 }

// price of one unit of the ingredient, exact: the package price over its size when it has both.
func unitPrice(packagePrice, packageSize, pricePerUnit float64) *big.Rat {
	rat := func(f float64) *big.Rat {
		r, _ := new(big.Rat).SetString(strconv.FormatFloat(f, 'f', -1, 64))
		return r
	}
	if packagePrice > 0 && packageSize > 0 {
		return new(big.Rat).Quo(rat(packagePrice), rat(packageSize))
	}
	return rat(pricePerUnit)
}

// lossOf is quantity x price, rounded once to whole pesos, halves up.
func lossOf(quantity float64, price *big.Rat) int64 {
	q, _ := new(big.Rat).SetString(strconv.FormatFloat(quantity, 'f', -1, 64))
	v := new(big.Rat).Mul(q, price)
	v.Add(v, big.NewRat(1, 2))
	return new(big.Int).Quo(v.Num(), v.Denom()).Int64()
}

const columns = `a.id, a.ingredient_id, i.name, a.reason, a.quantity_delta, a.unit, a.stock_before, a.stock_after,
	a.loss_amount, a.note, to_char(a.adjusted_on, 'YYYY-MM-DD'), a.created_by, a.created_at`

func scan(row pgx.Row) (Adjustment, error) {
	var a Adjustment
	err := row.Scan(&a.ID, &a.IngredientID, &a.IngredientName, &a.Reason, &a.QuantityDelta, &a.Unit,
		&a.StockBefore, &a.StockAfter, &a.LossAmount, &a.Note, &a.AdjustedOn, &a.CreatedBy, &a.CreatedAt)
	return a, err
}

// Record applies one adjustment to the stock of an ingredient and keeps the record of it. The row
// of the ingredient stays locked meanwhile, so adjustments at the same time wait for each other.
func (r *Repository) Record(ctx context.Context, in Input) (Adjustment, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Adjustment{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var unit string
	var packageSize, packagePrice, pricePerUnit, stock float64
	err = tx.QueryRow(ctx, `
		SELECT default_unit, package_size, package_price, price_per_unit, stock_quantity
		FROM ingredients WHERE id = $1 FOR UPDATE`, in.IngredientID,
	).Scan(&unit, &packageSize, &packagePrice, &pricePerUnit, &stock)
	if errors.Is(err, pgx.ErrNoRows) {
		return Adjustment{}, ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "22P02" { // not even a valid id
		return Adjustment{}, ErrNotFound
	}
	if err != nil {
		return Adjustment{}, fmt.Errorf("lock ingredient: %w", err)
	}

	given := in.Unit
	if given == "" {
		given = unit
	}
	toIngredientUnit := func(q float64) (float64, error) {
		v, err := kernel.ConvertToBase(q, given, unit)
		if err != nil {
			return 0, &ValidationError{err.Error()}
		}
		return round3(v), nil
	}

	note := strings.TrimSpace(in.Note)
	var after float64
	var loss int64
	switch in.Reason {
	case ReasonMerma:
		if in.Quantity == nil || *in.Quantity <= 0 {
			return Adjustment{}, &ValidationError{"la merma necesita una cantidad mayor a 0"}
		}
		q, err := toIngredientUnit(*in.Quantity)
		if err != nil {
			return Adjustment{}, err
		}
		if q > stock {
			return Adjustment{}, &ValidationError{fmt.Sprintf("no hay tanto stock: hay %v %s y la merma es de %v %s. Corrige el stock primero", stock, unit, q, unit)}
		}
		after = round3(stock - q)
		loss = lossOf(q, unitPrice(packagePrice, packageSize, pricePerUnit))
	case ReasonCorrection:
		if in.CountedQuantity == nil || *in.CountedQuantity < 0 {
			return Adjustment{}, &ValidationError{"la corrección necesita la cantidad contada (0 o más)"}
		}
		counted, err := toIngredientUnit(*in.CountedQuantity)
		if err != nil {
			return Adjustment{}, err
		}
		after = counted
	case ReasonOther:
		if note == "" {
			return Adjustment{}, &ValidationError{"el motivo \"otro\" necesita una nota"}
		}
		if in.QuantityDelta == nil || *in.QuantityDelta == 0 {
			return Adjustment{}, &ValidationError{"el ajuste necesita una diferencia distinta de 0"}
		}
		delta, err := toIngredientUnit(*in.QuantityDelta)
		if err != nil {
			return Adjustment{}, err
		}
		after = round3(stock + delta)
		if after < 0 {
			return Adjustment{}, &ValidationError{"el stock no puede quedar negativo"}
		}
	default:
		return Adjustment{}, &ValidationError{`el motivo debe ser "merma", "correccion" u "otro"`}
	}

	stock = round3(stock)
	if _, err := tx.Exec(ctx, `UPDATE ingredients SET stock_quantity = $1, updated_at = NOW() WHERE id = $2`, after, in.IngredientID); err != nil {
		return Adjustment{}, fmt.Errorf("update stock: %w", err)
	}
	var id string
	if err := tx.QueryRow(ctx, `
		INSERT INTO stock_adjustments (created_by, ingredient_id, reason, quantity_delta, unit, stock_before, stock_after, loss_amount, note, adjusted_on)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::date) RETURNING id`,
		in.CreatedBy, in.IngredientID, in.Reason, round3(after-stock), unit, stock, after, loss, note, in.AdjustedOn,
	).Scan(&id); err != nil {
		return Adjustment{}, fmt.Errorf("insert adjustment: %w", err)
	}
	a, err := scan(tx.QueryRow(ctx, `SELECT `+columns+` FROM stock_adjustments a JOIN ingredients i ON i.id = a.ingredient_id WHERE a.id = $1`, id))
	if err != nil {
		return Adjustment{}, fmt.Errorf("read adjustment: %w", err)
	}
	return a, tx.Commit(ctx)
}

func (r *Repository) list(ctx context.Context, where string, args ...any) ([]Adjustment, error) {
	rows, err := r.db.Query(ctx, `SELECT `+columns+`
		FROM stock_adjustments a JOIN ingredients i ON i.id = a.ingredient_id
		WHERE `+where+` ORDER BY a.adjusted_on DESC, a.created_at DESC`, args...)
	if err != nil {
		return nil, fmt.Errorf("list adjustments: %w", err)
	}
	defer rows.Close()
	out := []Adjustment{}
	for rows.Next() {
		a, err := scan(rows)
		if err != nil {
			return nil, fmt.Errorf("scan adjustment: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ForIngredient is the whole history of one ingredient.
func (r *Repository) ForIngredient(ctx context.Context, ingredientID string) ([]Adjustment, error) {
	return r.list(ctx, `a.ingredient_id = $1`, ingredientID)
}

// ForMonth is what was adjusted during the month that starts on `month` ("YYYY-MM-01").
func (r *Repository) ForMonth(ctx context.Context, month string) ([]Adjustment, error) {
	return r.list(ctx, `a.adjusted_on >= $1::date AND a.adjusted_on < ($1::date + interval '1 month')`, month)
}
