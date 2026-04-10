package service

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
)

type QuoteData struct {
	RecipeName  string
	ClientName  string
	Description string
	Yield       float64
	YieldUnit   string
	PhotoURL    *string
	TotalCost   float64
	MarginPct   float64
	FinalPrice  float64
	Date        time.Time
}

// Business contact constants – edit here to update all PDFs.
const (
	bizName     = "Angeles'S Coffee & Bakery"
	bizPhone    = "+56 9 1234 5678"
	bizWhatsApp = "+56 9 1234 5678"
	bizInstagram = "@angelesscoffee"
	bizBanco    = "Banco Estado"
	bizCuenta   = "123-456789-0"
	bizTitular  = "Ángeles Martínez"
	bizRUT      = "12.345.678-9"
)

// GenerateQuotePDF builds an A4 PDF cotización and returns its bytes.
//
// Layout (top→bottom):
//  1. Header: biz name + date | "COTIZACIÓN" + client name
//  2. Gold separator
//  3. Two-column: image (left, large) | recipe name + rendimiento + precio (right)
//  4. Gold separator
//  5. Description paragraph
//  6. Gold separator
//  7. Transfer data (left) | Contact (right)
//  8. Notice box: 48h confirmation
//  9. Footer
func GenerateQuotePDF(data QuoteData) (*bytes.Buffer, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(20, 20, 20)
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddPage()

	tr := pdf.UnicodeTranslatorFromDescriptor("")

	const (
		lm       = 20.0  // left margin
		rm       = 20.0  // right margin (unused directly but symmetry)
		tm       = 20.0  // top margin
		pageW    = 170.0 // 210 - lm - rm
		gap      = 4.0
		smallGap = 2.0
	)

	y := tm

	// ── 1. HEADER ────────────────────────────────────────────────────────────
	// Row 1: biz name (gold, bold) | date (gray, right)
	pdf.SetXY(lm, y)
	pdf.SetTextColor(180, 140, 60)
	pdf.SetFont("Helvetica", "B", 13)
	pdf.CellFormat(pageW/2, 6, tr(bizName), "", 0, "L", false, 0, "")
	pdf.SetTextColor(160, 155, 150)
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(pageW/2, 6, data.Date.Format("02/01/2006"), "", 0, "R", false, 0, "")
	y += 7

	// Row 2: "COTIZACIÓN" (gray small caps) | "Cliente: X" (right)
	pdf.SetXY(lm, y)
	pdf.SetTextColor(160, 155, 150)
	pdf.SetFont("Helvetica", "", 8)
	pdf.CellFormat(pageW/2, 5, "COTIZACION", "", 0, "L", false, 0, "")
	if data.ClientName != "" {
		pdf.SetFont("Helvetica", "B", 8)
		pdf.CellFormat(pageW/2, 5, tr("Cliente: "+data.ClientName), "", 0, "R", false, 0, "")
	}
	y += 5

	y += smallGap

	// ── GOLD SEPARATOR ────────────────────────────────────────────────────────
	pdf.SetDrawColor(180, 140, 60)
	pdf.SetLineWidth(0.6)
	pdf.Line(lm, y, lm+pageW, y)
	y += gap

	// ── 2. TWO-COLUMN: IMAGE (left) | RECIPE INFO (right) ────────────────────
	const (
		imgColW  = 105.0 // ~62% of 170mm
		infoColW = 60.0  // remaining, with 5mm gap
		colGap   = 5.0
		maxImgH  = 80.0
	)
	infoColX := lm + imgColW + colGap
	blockStartY := y

	// Right column: recipe name, yield, price – drawn first to measure height
	rightColH := 0.0

	// Recipe name (up to 2 lines, 13pt bold)
	pdf.SetFont("Helvetica", "B", 13)
	pdf.SetTextColor(41, 37, 36)
	pdf.SetXY(infoColX, blockStartY)
	nameLineH := 6.5
	pdf.MultiCell(infoColW, nameLineH, tr(data.RecipeName), "", "L", false)
	nameEndY := pdf.GetY()
	maxNameEndY := blockStartY + 2*nameLineH
	if nameEndY > maxNameEndY {
		nameEndY = maxNameEndY
	}
	rightColH = nameEndY - blockStartY

	rightColH += gap

	// Yield
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(150, 145, 140)
	yieldY := blockStartY + rightColH
	pdf.SetXY(infoColX, yieldY)
	pdf.CellFormat(infoColW, 5, tr(fmt.Sprintf("Rendimiento: %.0f %s", data.Yield, data.YieldUnit)), "", 0, "L", false, 0, "")
	rightColH += 5 + gap

	// Price label
	priceY := blockStartY + rightColH
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(160, 155, 150)
	pdf.SetXY(infoColX, priceY)
	pdf.CellFormat(infoColW, 4, "PRECIO TOTAL", "", 0, "L", false, 0, "")
	rightColH += 4 + smallGap

	// Price value (gold, large)
	priceValY := blockStartY + rightColH
	pdf.SetFont("Helvetica", "B", 22)
	pdf.SetTextColor(180, 140, 60)
	pdf.SetXY(infoColX, priceValY)
	pdf.CellFormat(infoColW, 10, formatCLP(data.FinalPrice), "", 0, "L", false, 0, "")
	rightColH += 10 + smallGap

	// "porciones completas" label
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(160, 155, 150)
	pdf.SetXY(infoColX, blockStartY+rightColH)
	pdf.CellFormat(infoColW, 4, tr("porciones completas"), "", 0, "L", false, 0, "")
	rightColH += 4

	// Left column: image
	imgActualH := 0.0
	if data.PhotoURL != nil && *data.PhotoURL != "" {
		imgData, imgType, err := downloadImage(*data.PhotoURL)
		if err == nil {
			opts := gofpdf.ImageOptions{ImageType: imgType}
			pdf.RegisterImageOptionsReader("cake", opts, imgData)

			drawW := imgColW
			drawH := maxImgH
			if info := pdf.GetImageInfo("cake"); info != nil {
				pw, ph := info.Extent()
				if pw > 0 && ph > 0 {
					aspect := ph / pw
					if imgColW*aspect <= maxImgH {
						drawW = imgColW
						drawH = imgColW * aspect
					} else {
						drawH = maxImgH
						drawW = maxImgH / aspect
					}
				}
			}
			pdf.ImageOptions("cake", lm, blockStartY, drawW, drawH, false, gofpdf.ImageOptions{}, 0, "")
			imgActualH = drawH
		}
	}

	blockH := math.Max(imgActualH, rightColH)
	y = blockStartY + blockH + gap

	// ── GOLD SEPARATOR ────────────────────────────────────────────────────────
	pdf.SetDrawColor(180, 140, 60)
	pdf.SetLineWidth(0.3)
	pdf.Line(lm, y, lm+pageW, y)
	y += gap

	// ── 3. DESCRIPTION ───────────────────────────────────────────────────────
	desc := strings.TrimSpace(data.Description)
	if len([]rune(desc)) > 500 {
		runes := []rune(desc)
		desc = string(runes[:497]) + "..."
	}
	if desc != "" {
		pdf.SetFont("Helvetica", "", 8)
		pdf.SetTextColor(160, 155, 150)
		pdf.SetXY(lm, y)
		pdf.CellFormat(pageW, 4, "DESCRIPCION", "", 0, "L", false, 0, "")
		y += 4 + smallGap

		pdf.SetFont("Helvetica", "", 10)
		pdf.SetTextColor(80, 75, 70)
		pdf.SetXY(lm, y)

		const descLineH = 5.0
		const maxDescLines = 5
		pdf.MultiCell(pageW, descLineH, tr(desc), "", "L", false)
		descEndY := pdf.GetY()
		maxDescEndY := y + float64(maxDescLines)*descLineH
		if descEndY > maxDescEndY {
			y = maxDescEndY
		} else {
			y = descEndY
		}
		y += gap
	}

	// ── GOLD SEPARATOR ────────────────────────────────────────────────────────
	pdf.SetDrawColor(180, 140, 60)
	pdf.SetLineWidth(0.3)
	pdf.Line(lm, y, lm+pageW, y)
	y += gap

	// ── 4. TRANSFER + CONTACT (two columns) ──────────────────────────────────
	const (
		txColW   = 88.0
		ctColW   = 77.0
		txColGap = 5.0
	)
	ctColX := lm + txColW + txColGap
	sectionStartY := y

	// Left: Transfer data
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetTextColor(180, 140, 60)
	pdf.SetXY(lm, sectionStartY)
	pdf.CellFormat(txColW, 5, "DATOS DE TRANSFERENCIA", "", 0, "L", false, 0, "")

	txLines := []struct{ label, value string }{
		{"Banco", bizBanco},
		{"Nombre", bizTitular},
		{"Cuenta", bizCuenta},
		{"RUT", bizRUT},
	}
	txY := sectionStartY + 6
	for _, row := range txLines {
		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetTextColor(80, 75, 70)
		pdf.SetXY(lm, txY)
		pdf.CellFormat(22, 5, tr(row.label+":"), "", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 9)
		pdf.CellFormat(txColW-22, 5, tr(row.value), "", 0, "L", false, 0, "")
		txY += 5
	}
	txSectionH := txY - sectionStartY

	// Right: Contact data
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetTextColor(180, 140, 60)
	pdf.SetXY(ctColX, sectionStartY)
	pdf.CellFormat(ctColW, 5, "CONTACTO", "", 0, "L", false, 0, "")

	ctLines := []struct{ label, value string }{
		{"Telefono", bizPhone},
		{"WhatsApp", bizWhatsApp},
		{"Instagram", bizInstagram},
	}
	ctY := sectionStartY + 6
	for _, row := range ctLines {
		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetTextColor(80, 75, 70)
		pdf.SetXY(ctColX, ctY)
		pdf.CellFormat(22, 5, tr(row.label+":"), "", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 9)
		pdf.CellFormat(ctColW-22, 5, tr(row.value), "", 0, "L", false, 0, "")
		ctY += 5
	}
	ctSectionH := ctY - sectionStartY

	y = sectionStartY + math.Max(txSectionH, ctSectionH) + gap

	// ── 5. NOTICE BOX (48h) ───────────────────────────────────────────────────
	pdf.SetFillColor(250, 248, 243)
	pdf.SetDrawColor(180, 140, 60)
	pdf.SetLineWidth(0.3)
	pdf.Rect(lm, y, pageW, 22, "FD")

	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(120, 80, 20)
	pdf.SetXY(lm+3, y+3)
	pdf.CellFormat(pageW-6, 5, tr("! IMPORTANTE: confirmacion del pedido"), "", 0, "L", false, 0, "")

	noticeLines := []string{
		"Confirmar con 48 horas de anticipacion minimo.",
		"Transferir el monto total y enviar comprobante por WhatsApp.",
	}
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(80, 75, 70)
	lineY := y + 9
	for _, line := range noticeLines {
		pdf.SetXY(lm+3, lineY)
		pdf.CellFormat(pageW-6, 4.5, tr("- "+line), "", 0, "L", false, 0, "")
		lineY += 4.5
	}
	y += 22 + gap

	// ── 6. FOOTER ────────────────────────────────────────────────────────────
	pdf.SetDrawColor(180, 140, 60)
	pdf.SetLineWidth(0.3)
	pdf.Line(lm, y, lm+pageW, y)
	y += 3

	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(180, 175, 170)
	pdf.SetXY(lm, y)
	pdf.CellFormat(pageW, 4, tr("Cotizacion valida por 7 dias a partir de la fecha de emision. Precios en Pesos Chilenos (CLP)."), "", 0, "C", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("pdf output: %w", err)
	}
	return &buf, nil
}

func downloadImage(url string) (io.Reader, string, error) {
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("HTTP %d fetching image", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	imgType := "jpeg"
	lower := strings.ToLower(url)
	if strings.Contains(lower, ".png") {
		imgType = "png"
	} else if strings.Contains(lower, ".webp") {
		imgType = "jpeg" // gofpdf doesn't support webp; treat as jpeg fallback
	}

	return bytes.NewReader(data), imgType, nil
}

func formatCLP(n float64) string {
	rounded := int64(math.Round(n))
	s := strconv.FormatInt(rounded, 10)
	var result strings.Builder
	for i, ch := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result.WriteRune('.')
		}
		result.WriteRune(ch)
	}
	return "$" + result.String()
}
