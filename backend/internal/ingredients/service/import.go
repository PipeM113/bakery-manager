package service

import (
	"context"
	"fmt"
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

		// D: Tamaño Presentación (numeric)
		sizeStr := strings.TrimSpace(row[3])
		var pkgSize float64
		if sizeStr != "" {
			fmt.Sscanf(sizeStr, "%f", &pkgSize)
		}
		if pkgSize <= 0 {
			pkgSize = 1
		}

		// E: Precio Presentación (numeric)
		priceStr := strings.TrimSpace(row[4])
		var pkgPrice float64
		fmt.Sscanf(priceStr, "%f", &pkgPrice)

		rows = append(rows, IngredientImportRow{
			RowNum:       excelRow,
			Name:         name,
			Brand:        brand,
			PackageSize:  pkgSize,
			PackagePrice: pkgPrice,
			DefaultUnit:  unit,
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

	// Validate: duplicates within file, price > 0, valid unit
	seenNames := map[string]int{} // lowercase name -> first row number

	for _, row := range rows {
		norm := strings.ToLower(row.Name)

		if firstRow, seen := seenNames[norm]; seen {
			result.Errors = append(result.Errors, ImportRowError{
				Row:    row.RowNum,
				Reason: fmt.Sprintf("nombre duplicado en el archivo (primera aparición: fila %d)", firstRow),
			})
		} else {
			seenNames[norm] = row.RowNum
		}

		if row.PackagePrice <= 0 {
			result.Errors = append(result.Errors, ImportRowError{
				Row:    row.RowNum,
				Reason: "el precio debe ser mayor a 0",
			})
		}

		if row.DefaultUnit == "" || !validUnits[row.DefaultUnit] {
			result.Errors = append(result.Errors, ImportRowError{
				Row:    row.RowNum,
				Reason: fmt.Sprintf("unidad inválida '%s' (válidas: gr, kg, ml, lt, und)", row.DefaultUnit),
			})
		}
	}

	if len(result.Errors) > 0 {
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
