package service

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xuri/excelize/v2"
)

var validUnits = map[string]bool{
	"gr": true, "kg": true, "ml": true, "lt": true, "unidad": true, "und": true,
}

type IngredientImportRow struct {
	RowNum       int
	Name         string
	Brand        string
	PackageSize  float64
	PackagePrice float64
	DefaultUnit  string
	// ParseError is set when the size or the price could not be read; the row is rejected.
	ParseError string
}

type ImportRowError struct {
	Row    int    `json:"row"`
	Reason string `json:"reason"`
}

type ImportResult struct {
	Imported int              `json:"imported"`
	Updated  int              `json:"updated"`
	Skipped  int              `json:"skipped"`
	Errors   []ImportRowError `json:"errors"`
}

type IngredientImportService struct {
	db *pgxpool.Pool
}

func NewIngredientImportService(db *pgxpool.Pool) *IngredientImportService {
	return &IngredientImportService{db: db}
}

// ParseIngredientsFromExcel reads an .xlsx file and returns parsed rows.
// Expected columns: A=Nombre, B=Marca, C=Unidad, D=Tamaño Presentación, E=Precio Presentación
// Column F (Precio Unitario info) is ignored. Row 1 (header) is skipped.
func ParseIngredientsFromExcel(filePath string) ([]IngredientImportRow, int, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, 0, fmt.Errorf("abriendo archivo: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, 0, fmt.Errorf("el archivo no tiene hojas")
	}

	allRows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, 0, fmt.Errorf("leyendo filas: %w", err)
	}

	if len(allRows) < 2 {
		return nil, 0, nil
	}

	var rows []IngredientImportRow
	skipped := 0

	for i, row := range allRows[1:] { // row 0 = header
		excelRow := i + 2 // 1-indexed, +1 for header, +1 for 1-indexing

		// Ensure at least 5 columns (A–E); column F is optional/ignored
		for len(row) < 5 {
			row = append(row, "")
		}

		// A: Nombre (required)
		name := strings.TrimSpace(row[0])
		if name == "" {
			skipped++
			continue
		}

		// B: Marca (optional)
		brand := strings.TrimSpace(row[1])

		// C: Unidad (required)
		unit := strings.TrimSpace(strings.ToLower(row[2]))
		if unit == "unidad" {
			unit = "und"
		}

		// D: Tamaño Presentación (numeric, > 0) and E: Precio Presentación (numeric, > 0)
		var parseError string
		pkgSize, err := ParseCLDecimal(row[3])
		if err != nil || pkgSize <= 0 {
			parseError = fmt.Sprintf("el tamaño de la presentación debe ser un número mayor a 0 (se leyó '%s')", strings.TrimSpace(row[3]))
		}
		pkgPrice, err := ParseCLDecimal(row[4])
		if parseError == "" && (err != nil || pkgPrice <= 0) {
			parseError = fmt.Sprintf("el precio debe ser un número mayor a 0 (se leyó '%s')", strings.TrimSpace(row[4]))
		}

		rows = append(rows, IngredientImportRow{
			RowNum:       excelRow,
			Name:         name,
			Brand:        brand,
			PackageSize:  pkgSize,
			PackagePrice: pkgPrice,
			DefaultUnit:  unit,
			ParseError:   parseError,
		})
	}

	return rows, skipped, nil
}

// Import validates rows and upserts them in a single transaction.
// Existing ingredients (matched by name, case-insensitive) are updated.
// New ingredients are inserted. Stock and alert threshold are preserved on update.
func (s *IngredientImportService) Import(ctx context.Context, rows []IngredientImportRow, skipped int) (ImportResult, error) {
	result := ImportResult{Skipped: skipped, Errors: []ImportRowError{}}

	if len(rows) == 0 {
		return result, nil
	}

	// Validate row by row. A rejected row is reported with its reason; the rest are imported.
	seenNames := map[string]int{} // lowercase name -> first row number
	valid := make([]IngredientImportRow, 0, len(rows))

	for _, row := range rows {
		reject := func(reason string) {
			result.Errors = append(result.Errors, ImportRowError{Row: row.RowNum, Reason: reason})
		}
		norm := strings.ToLower(row.Name)

		if row.ParseError != "" {
			reject(row.ParseError)
			continue
		}
		if firstRow, seen := seenNames[norm]; seen {
			reject(fmt.Sprintf("nombre duplicado en el archivo (primera aparición: fila %d)", firstRow))
			continue
		}
		seenNames[norm] = row.RowNum

		if row.DefaultUnit == "" || !validUnits[row.DefaultUnit] {
			reject(fmt.Sprintf("unidad inválida '%s' (válidas: gr, kg, ml, lt, und)", row.DefaultUnit))
			continue
		}
		valid = append(valid, row)
	}
	rows = valid
	if len(rows) == 0 {
		return result, nil
	}

	// Upsert in a single transaction
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return result, fmt.Errorf("iniciando transacción: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, row := range rows {
		pricePerUnit := row.PackagePrice / row.PackageSize

		// Check if ingredient already exists (case-insensitive)
		var existingID string
		var oldPricePerUnit float64
		err := tx.QueryRow(ctx,
			`SELECT id, price_per_unit FROM ingredients WHERE LOWER(name) = LOWER($1)`,
			row.Name,
		).Scan(&existingID, &oldPricePerUnit)

		if err == nil {
			// Ingredient exists → update (preserve stock_quantity and alert_threshold)
			_, err = tx.Exec(ctx, `
				UPDATE ingredients
				SET name=$1, default_unit=$2, package_size=$3, package_price=$4,
				    price_per_unit=$5, brand=$6, updated_at=NOW()
				WHERE id=$7`,
				row.Name, row.DefaultUnit, row.PackageSize, row.PackagePrice,
				pricePerUnit, row.Brand, existingID,
			)
			if err != nil {
				return result, fmt.Errorf("actualizando fila %d: %w", row.RowNum, err)
			}

			// Record price history only if price changed
			if oldPricePerUnit != pricePerUnit {
				_, err = tx.Exec(ctx,
					`INSERT INTO ingredient_price_history (ingredient_id, price_per_unit, unit)
					 VALUES ($1, $2, $3)`,
					existingID, pricePerUnit, row.DefaultUnit,
				)
				if err != nil {
					return result, fmt.Errorf("historial de precio fila %d: %w", row.RowNum, err)
				}
			}

			result.Updated++
		} else {
			// Ingredient does not exist → insert
			var newID string
			err = tx.QueryRow(ctx, `
				INSERT INTO ingredients
				  (name, default_unit, package_size, package_price, price_per_unit,
				   stock_quantity, alert_threshold, brand)
				VALUES ($1,$2,$3,$4,$5,0,0,$6)
				RETURNING id`,
				row.Name, row.DefaultUnit, row.PackageSize, row.PackagePrice,
				pricePerUnit, row.Brand,
			).Scan(&newID)
			if err != nil {
				return result, fmt.Errorf("insertando fila %d: %w", row.RowNum, err)
			}

			_, err = tx.Exec(ctx,
				`INSERT INTO ingredient_price_history (ingredient_id, price_per_unit, unit)
				 VALUES ($1, $2, $3)`,
				newID, pricePerUnit, row.DefaultUnit,
			)
			if err != nil {
				return result, fmt.Errorf("creando historial fila %d: %w", row.RowNum, err)
			}

			result.Imported++
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return result, fmt.Errorf("confirmando transacción: %w", err)
	}

	return result, nil
}

var (
	thousandsGrouped = regexp.MustCompile(`^[1-9]\d{0,2}(\.\d{3})+$`) // 1.500  1.500.000
	plainNumber      = regexp.MustCompile(`^\d+(\.\d+)?$`)            // 1500  2.5  0.250
	commaDecimal     = regexp.MustCompile(`^(\d+|[1-9]\d{0,2}(\.\d{3})+),\d+$`)
)

// ParseCLDecimal reads a number the way it is written in Chile: "." groups thousands and ","
// marks the decimals ("1.500,50"), a leading "$" is ignored, and a plain "2.5" is 2.5.
// "1.500" is fifteen hundred; "0.250" is a quarter, because a group of thousands never
// starts with 0.
func ParseCLDecimal(s string) (float64, error) {
	t := strings.TrimSpace(s)
	t = strings.TrimSpace(strings.TrimPrefix(t, "$"))
	negative := strings.HasPrefix(t, "-")
	t = strings.TrimPrefix(t, "-")
	if t == "" {
		return 0, fmt.Errorf("número vacío")
	}

	switch {
	case commaDecimal.MatchString(t):
		t = strings.ReplaceAll(t, ".", "")
		t = strings.Replace(t, ",", ".", 1)
	case thousandsGrouped.MatchString(t):
		t = strings.ReplaceAll(t, ".", "")
	case plainNumber.MatchString(t):
	default:
		return 0, fmt.Errorf("'%s' no es un número", s)
	}
	v, err := strconv.ParseFloat(t, 64)
	if err != nil {
		return 0, fmt.Errorf("'%s' no es un número", s)
	}
	if negative {
		v = -v
	}
	return v, nil
}
