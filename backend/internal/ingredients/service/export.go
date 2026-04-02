package service

import (
	"bytes"
	"fmt"
	"math"

	"github.com/PipeM113/bakery-manager/internal/ingredients/repository"
	"github.com/xuri/excelize/v2"
)

func formatCLP(n float64) string {
	rounded := int64(math.Round(n))
	if rounded == 0 {
		return "0"
	}
	negative := rounded < 0
	if negative {
		rounded = -rounded
	}
	s := fmt.Sprintf("%d", rounded)
	result := make([]byte, 0, len(s)+len(s)/3)
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, '.')
		}
		result = append(result, byte(c))
	}
	if negative {
		return "-" + string(result)
	}
	return string(result)
}

// ExportIngredientsToExcel generates an Excel file with the following columns:
// A=Nombre, B=Marca, C=Unidad, D=Tamaño Presentación, E=Precio Presentación, F=Precio Unitario (info)
// Columns A–E can be modified and re-imported. Column F is informational only.
func ExportIngredientsToExcel(ingredients []repository.Ingredient) (*bytes.Buffer, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Insumos"
	f.SetSheetName("Sheet1", sheet)

	// Header style: bold, gold background (#C9A84C), white text
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"C9A84C"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "bottom", Color: "A07830", Style: 2},
		},
	})
	if err != nil {
		return nil, err
	}

	// Info header style (column F): same gold but slightly muted
	infoHeaderStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"B89840"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "bottom", Color: "A07830", Style: 2},
		},
	})
	if err != nil {
		return nil, err
	}

	// Alternating row styles
	rowStyleLight, err := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"FDFAF5"}, Pattern: 1},
		Alignment: &excelize.Alignment{Vertical: "center"},
	})
	if err != nil {
		return nil, err
	}

	rowStyleDark, err := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"F5EFE0"}, Pattern: 1},
		Alignment: &excelize.Alignment{Vertical: "center"},
	})
	if err != nil {
		return nil, err
	}

	// Number cell styles (right-aligned, for D and E)
	numLightStyle, err := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"FDFAF5"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
	})
	if err != nil {
		return nil, err
	}

	numDarkStyle, err := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"F5EFE0"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
	})
	if err != nil {
		return nil, err
	}

	// Info cell styles for column F (slightly muted)
	infoLightStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: "999999", Italic: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"F8F4EC"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
	})
	if err != nil {
		return nil, err
	}

	infoDarkStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: "999999", Italic: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"EFE9D8"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
	})
	if err != nil {
		return nil, err
	}

	// Columns A–E are importable; F is informational
	headers := []string{"Nombre", "Marca", "Unidad", "Tamaño Presentación", "Precio Presentación", "Precio Unitario (info)"}
	cols := []string{"A", "B", "C", "D", "E", "F"}
	colWidths := []float64{30, 20, 12, 20, 22, 22}

	for i, h := range headers {
		cell := cols[i] + "1"
		f.SetCellValue(sheet, cell, h)
		style := headerStyle
		if i == 5 {
			style = infoHeaderStyle
		}
		f.SetCellStyle(sheet, cell, cell, style)
		f.SetColWidth(sheet, cols[i], cols[i], colWidths[i])
	}
	f.SetRowHeight(sheet, 1, 22)

	for idx, ing := range ingredients {
		row := idx + 2
		rowStr := fmt.Sprintf("%d", row)

		isEven := idx%2 == 1
		textStyle := rowStyleLight
		numStyle := numLightStyle
		infoStyle := infoLightStyle
		if isEven {
			textStyle = rowStyleDark
			numStyle = numDarkStyle
			infoStyle = infoDarkStyle
		}

		// A: Nombre
		f.SetCellValue(sheet, "A"+rowStr, ing.Name)
		f.SetCellStyle(sheet, "A"+rowStr, "A"+rowStr, textStyle)

		// B: Marca
		f.SetCellValue(sheet, "B"+rowStr, ing.Brand)
		f.SetCellStyle(sheet, "B"+rowStr, "B"+rowStr, textStyle)

		// C: Unidad
		f.SetCellValue(sheet, "C"+rowStr, ing.DefaultUnit)
		f.SetCellStyle(sheet, "C"+rowStr, "C"+rowStr, textStyle)

		// D: Tamaño Presentación (numeric, importable)
		f.SetCellValue(sheet, "D"+rowStr, ing.PackageSize)
		f.SetCellStyle(sheet, "D"+rowStr, "D"+rowStr, numStyle)

		// E: Precio Presentación (numeric, importable)
		f.SetCellValue(sheet, "E"+rowStr, ing.PackagePrice)
		f.SetCellStyle(sheet, "E"+rowStr, "E"+rowStr, numStyle)

		// F: Precio Unitario (CLP formatted, informational only)
		f.SetCellValue(sheet, "F"+rowStr, formatCLP(ing.PricePerUnit))
		f.SetCellStyle(sheet, "F"+rowStr, "F"+rowStr, infoStyle)

		f.SetRowHeight(sheet, row, 18)
	}

	// Freeze header row
	f.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf, nil
}
