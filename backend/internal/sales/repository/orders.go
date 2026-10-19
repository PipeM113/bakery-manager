package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/PipeM113/bakery-manager/pkg/money"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// OrderLine is one product of an order: a sale, with the share of the order total it brought in.
type OrderLine struct {
	SaleID       string `json:"sale_id"`
	RecipeID     string `json:"recipe_id"`
	RecipeName   string `json:"recipe_name"`
	QuantitySold int    `json:"quantity_sold"`
	SoldAs       string `json:"sold_as"`
	UnitPrice    int64  `json:"unit_price"`
	LineTotal    int64  `json:"line_total"`
}

type Order struct {
	ID           string      `json:"id"`
	CreatedBy    *string     `json:"created_by"`
	CustomerName string      `json:"customer_name"`
	Notes        string      `json:"notes"`
	OrderDate    string      `json:"order_date"`
	TotalPrice   int64       `json:"total_price"`
	Lines        []OrderLine `json:"lines"`
	// StockShortfalls lists what the stock could not give; empty when it was enough.
	StockShortfalls []Shortfall `json:"stock_shortfalls"`
	CreatedAt       time.Time   `json:"created_at"`
}

type OrderInput struct {
	CreatedBy    string
	CustomerName string
	Notes        string
	OrderDate    string // YYYY-MM-DD, already checked
	TotalPrice   *int64 // the order's own total; nil means the sum of the lines
	Lines        []LineInput
}

// Distribute shares `total` between lines in proportion to their normal prices, in whole pesos,
// so that the shares add up to exactly `total`. What is left after rounding down goes, one peso
// at a time, to the lines with the largest remainders (the first one wins a tie). Lines without a
// normal price share the total equally.
func Distribute(total int64, normals []int64) []int64 {
	n := int64(len(normals))
	out := make([]int64, len(normals))
	var sum int64
	for _, v := range normals {
		sum += v
	}
	rems := make([]int64, len(normals))
	var given int64
	for i, v := range normals {
		if sum == 0 {
			out[i], rems[i] = total/n, total%n
		} else {
			out[i], rems[i] = total*v/sum, total*v%sum
		}
		given += out[i]
	}
	for left := total - given; left > 0; left-- {
		best := 0
		for i := range rems {
			if rems[i] > rems[best] {
				best = i
			}
		}
		out[best]++
		rems[best] = -1
	}
	return out
}

// CreateOrder registers every line as a sale of the same order, all or nothing.
func (r *SaleRepository) CreateOrder(ctx context.Context, in OrderInput) (Order, error) {
	normals := make([]int64, len(in.Lines))
	var sum int64
	for i, l := range in.Lines {
		normals[i] = int64(l.QuantitySold) * l.UnitPrice
		sum += normals[i]
	}
	if sum > money.MaxPesos {
		return Order{}, &ValidationError{"el total del pedido supera el máximo permitido"}
	}
	total, totals := sum, normals
	if in.TotalPrice != nil {
		total = *in.TotalPrice
		totals = Distribute(total, normals)
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Order{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	order := Order{CustomerName: in.CustomerName, Notes: in.Notes, OrderDate: in.OrderDate, TotalPrice: total}
	if err := tx.QueryRow(ctx, `
		INSERT INTO orders (created_by, customer_name, notes, order_date, total_price)
		VALUES ($1, $2, $3, $4::date, $5) RETURNING id, created_at`,
		in.CreatedBy, in.CustomerName, in.Notes, in.OrderDate, total,
	).Scan(&order.ID, &order.CreatedAt); err != nil {
		return Order{}, fmt.Errorf("insert order: %w", err)
	}

	sales, shortfalls, err := takeLines(ctx, tx, in.CreatedBy, &order.ID, in.OrderDate, in.Notes, in.Lines, totals)
	if err != nil {
		return Order{}, err
	}
	order.StockShortfalls = shortfalls
	order.Lines = make([]OrderLine, len(sales))
	for i, s := range sales {
		order.Lines[i] = OrderLine{SaleID: s.ID, RecipeID: s.RecipeID, RecipeName: s.RecipeName, QuantitySold: s.QuantitySold,
			SoldAs: s.SoldAs, UnitPrice: s.UnitPrice, LineTotal: s.TotalPrice}
	}
	if err := tx.Commit(ctx); err != nil {
		return Order{}, fmt.Errorf("commit tx: %w", err)
	}
	return order, nil
}

const orderColumns = `o.id, o.created_by, o.customer_name, o.notes, o.order_date::text, o.total_price, o.created_at`

func scanOrder(row pgx.Row) (Order, error) {
	var o Order
	err := row.Scan(&o.ID, &o.CreatedBy, &o.CustomerName, &o.Notes, &o.OrderDate, &o.TotalPrice, &o.CreatedAt)
	o.Lines = []OrderLine{}
	o.StockShortfalls = []Shortfall{}
	return o, err
}

// fillLines reads the lines of the orders in one query.
func (r *SaleRepository) fillLines(ctx context.Context, orders []Order) error {
	if len(orders) == 0 {
		return nil
	}
	ids := make([]string, len(orders))
	at := map[string]int{}
	for i, o := range orders {
		ids[i] = o.ID
		at[o.ID] = i
	}
	rows, err := r.db.Query(ctx, `
		SELECT s.order_id, s.id, s.recipe_id, rec.name, s.quantity_sold, s.sold_as, s.unit_price, s.line_total
		FROM sales s JOIN recipes rec ON rec.id = s.recipe_id
		WHERE s.order_id = ANY($1::uuid[])
		ORDER BY s.created_at, s.id`, ids)
	if err != nil {
		return fmt.Errorf("read order lines: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var orderID string
		var l OrderLine
		if err := rows.Scan(&orderID, &l.SaleID, &l.RecipeID, &l.RecipeName, &l.QuantitySold, &l.SoldAs, &l.UnitPrice, &l.LineTotal); err != nil {
			return fmt.Errorf("scan order line: %w", err)
		}
		orders[at[orderID]].Lines = append(orders[at[orderID]].Lines, l)
	}
	return rows.Err()
}

// ListOrders gives the orders of the month that starts on `month` ("YYYY-MM-01"), newest first.
func (r *SaleRepository) ListOrders(ctx context.Context, month string) ([]Order, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+orderColumns+` FROM orders o
		WHERE o.order_date >= $1::date AND o.order_date < ($1::date + interval '1 month')
		ORDER BY o.order_date DESC, o.created_at DESC`, month)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()
	orders := []Order{}
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	return orders, r.fillLines(ctx, orders)
}

func notFound(err error) bool {
	var pgErr *pgconn.PgError
	return errors.Is(err, pgx.ErrNoRows) || (errors.As(err, &pgErr) && pgErr.Code == "22P02")
}

// GetOrder gives one order of the user with its lines.
func (r *SaleRepository) GetOrder(ctx context.Context, id string) (Order, error) {
	o, err := scanOrder(r.db.QueryRow(ctx, `SELECT `+orderColumns+` FROM orders o WHERE o.id = $1`, id))
	if notFound(err) {
		return Order{}, ErrNotFound
	}
	if err != nil {
		return Order{}, fmt.Errorf("get order: %w", err)
	}
	orders := []Order{o}
	if err := r.fillLines(ctx, orders); err != nil {
		return Order{}, err
	}
	return orders[0], nil
}

// DeleteOrder cancels the whole order: each ingredient gets back exactly what the lines took.
func (r *SaleRepository) DeleteOrder(ctx context.Context, id string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var locked string
	if err := tx.QueryRow(ctx, `SELECT id FROM orders WHERE id = $1 FOR UPDATE`, id).Scan(&locked); err != nil {
		if notFound(err) {
			return ErrNotFound
		}
		return fmt.Errorf("lock order: %w", err)
	}
	rows, err := tx.Query(ctx, `
		SELECT si.ingredient_id, SUM(si.stock_deducted)
		FROM sale_ingredients si JOIN sales s ON s.id = si.sale_id
		WHERE s.order_id = $1 GROUP BY si.ingredient_id ORDER BY si.ingredient_id`, id)
	if err != nil {
		return fmt.Errorf("read what the order took: %w", err)
	}
	type back struct {
		ingredient string
		qty        float64
	}
	var giveBack []back
	for rows.Next() {
		var b back
		if err := rows.Scan(&b.ingredient, &b.qty); err != nil {
			rows.Close()
			return fmt.Errorf("scan: %w", err)
		}
		giveBack = append(giveBack, b)
	}
	rows.Close()
	for _, b := range giveBack {
		if _, err := tx.Exec(ctx, `UPDATE ingredients SET stock_quantity = stock_quantity + $1, updated_at = NOW() WHERE id = $2`, b.qty, b.ingredient); err != nil {
			return fmt.Errorf("give the stock back: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM orders WHERE id = $1`, id); err != nil {
		return fmt.Errorf("delete order: %w", err)
	}
	return tx.Commit(ctx)
}
