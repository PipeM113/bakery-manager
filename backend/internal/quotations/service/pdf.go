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
	Description string
	Yield       float64
	YieldUnit   string
	PhotoURL    *string
	TotalCost   float64
	MarginPct   float64
	FinalPrice  float64
	Date        time.Time
}

// GenerateQuotePDF builds an A4 PDF cotización and returns its bytes.
// Uses absolute Y positioning to prevent section overlap.
func GenerateQuotePDF(data QuoteData) (*bytes.Buffer, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(20, 15, 20)
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddPage()

	tr := pdf.UnicodeTranslatorFromDescriptor("")

	const (
		leftMargin = 20.0
		pageW      = 170.0 // 210 - 20 left - 20 right
		topMargin  = 15.0
		sectionGap = 5.0
	)

	curY := topMargin

	// ── HEADER (10mm total) ───────────────────────────────────────────────────
	pdf.SetXY(leftMargin, curY)
	pdf.SetTextColor(180, 140, 60)
	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(pageW/2, 6, tr("Angeles'S Coffee & Bakery"), "", 0, "L", false, 0, "")
	pdf.SetTextColor(160, 155, 150)
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(pageW/2, 6, data.Date.Format("02/01/2006"), "", 0, "R", false, 0, "")
	curY += 6

	pdf.SetXY(leftMargin, curY)
	pdf.SetTextColor(160, 155, 150)
	pdf.SetFont("Helvetica", "", 8)
	pdf.CellFormat(pageW, 4, "COTIZACIÓN", "", 0, "L", false, 0, "")
	curY += 4
	// header total = 10mm ✓

	curY += sectionGap

	// Gold separator
	pdf.SetDrawColor(180, 140, 60)
	pdf.SetLineWidth(0.5)
	pdf.Line(leftMargin, curY, leftMargin+pageW, curY)
	curY += sectionGap

	// ── PHOTO (max 40mm height, centered) ────────────────────────────────────
	if data.PhotoURL != nil && *data.PhotoURL != "" {
		imgData, imgType, err := downloadImage(*data.PhotoURL)
		if err == nil {
			opts := gofpdf.ImageOptions{ImageType: imgType}
			pdf.RegisterImageOptionsReader("cake", opts, imgData)

			const maxImgH = 40.0
			const maxImgW = 80.0

			drawW, drawH := maxImgW, maxImgH
			if info := pdf.GetImageInfo("cake"); info != nil {
				pw, ph := info.Extent()
				if pw > 0 && ph > 0 {
					aspect := ph / pw
					// fit into maxImgW × maxImgH box
					if maxImgW*aspect <= maxImgH {
						drawW = maxImgW
						drawH = maxImgW * aspect
					} else {
						drawH = maxImgH
						drawW = maxImgH / aspect
					}
				}
			}

			imgX := leftMargin + (pageW-drawW)/2
			pdf.ImageOptions("cake", imgX, curY, drawW, drawH, false, gofpdf.ImageOptions{}, 0, "")
			curY += drawH + sectionGap
		}
		// if download fails: silently skip photo, no curY change
	}

	// ── RECIPE NAME (up to 2 lines ≈ 16mm, font 18pt) ────────────────────────
	pdf.SetTextColor(41, 37, 36)
	pdf.SetFont("Helvetica", "B", 18)
	pdf.SetXY(leftMargin, curY)

	name := tr(data.RecipeName)
	const lineH = 8.0
	const maxNameLines = 2
	pdf.MultiCell(pageW, lineH, name, "", "C", false)

	// Cap Y advance to maxNameLines * lineH to prevent long names from pushing content down
	nameEndY := pdf.GetY()
	maxNameEndY := curY + float64(maxNameLines)*lineH
	if nameEndY > maxNameEndY {
		curY = maxNameEndY
	} else {
		curY = nameEndY
	}
	curY += sectionGap

	// ── DESCRIPTION (up to 4 lines ≈ 24mm, font 11pt, max 20mm) ─────────────
	if data.Description != "" {
		pdf.SetTextColor(110, 105, 100)
		pdf.SetFont("Helvetica", "", 11)
		pdf.SetXY(leftMargin, curY)

		const descLineH = 5.0
		const maxDescLines = 4
		pdf.MultiCell(pageW, descLineH, tr(data.Description), "", "C", false)

		descEndY := pdf.GetY()
		maxDescEndY := curY + float64(maxDescLines)*descLineH
		if descEndY > maxDescEndY {
			curY = maxDescEndY
		} else {
			curY = descEndY
		}
		curY += sectionGap
	}

	// ── YIELD (8mm) ──────────────────────────────────────────────────────────
	pdf.SetXY(leftMargin, curY)
	pdf.SetTextColor(150, 145, 140)
	pdf.SetFont("Helvetica", "I", 10)
	yieldStr := fmt.Sprintf("Rendimiento: %.0f %s", data.Yield, data.YieldUnit)
	pdf.CellFormat(pageW, 6, tr(yieldStr), "", 0, "C", false, 0, "")
	curY += 8

	// ── SEPARATOR (2mm) ──────────────────────────────────────────────────────
	pdf.SetDrawColor(180, 140, 60)
	pdf.SetLineWidth(0.3)
	pdf.Line(leftMargin, curY, leftMargin+pageW, curY)
	curY += 2

	curY += sectionGap

	// ── PRICE (28pt bold, gold, 15mm) ─────────────────────────────────────────
	pdf.SetXY(leftMargin, curY)
	pdf.SetTextColor(184, 134, 11) // #B8860B
	pdf.SetFont("Helvetica", "B", 28)
	pdf.CellFormat(pageW, 12, formatCLP(data.FinalPrice), "", 0, "C", false, 0, "")
	curY += 14

	pdf.SetXY(leftMargin, curY)
	pdf.SetTextColor(150, 145, 140)
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(pageW, 5, tr("Precio (torta completa)"), "", 0, "C", false, 0, "")
	curY += 7

	curY += sectionGap

	// ── FOOTER SEPARATOR ─────────────────────────────────────────────────────
	pdf.SetDrawColor(180, 140, 60)
	pdf.SetLineWidth(0.3)
	pdf.Line(leftMargin, curY, leftMargin+pageW, curY)
	curY += 4

	// ── FOOTER (10mm, 8pt) ────────────────────────────────────────────────────
	pdf.SetXY(leftMargin, curY)
	pdf.SetTextColor(180, 175, 170)
	pdf.SetFont("Helvetica", "", 8)
	pdf.CellFormat(pageW, 5, tr("Cotización válida por 7 días a partir de la fecha de emisión."), "", 0, "C", false, 0, "")
	curY += 5
	pdf.SetXY(leftMargin, curY)
	pdf.CellFormat(pageW, 5, "Precios en Pesos Chilenos (CLP)", "", 0, "C", false, 0, "")

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
