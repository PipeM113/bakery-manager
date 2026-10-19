package service

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png" // registers PNG for image.Decode
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jung-kurt/gofpdf"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // registers WebP for image.Decode
)

//go:embed assets/fonts/*.ttf assets/logo-color.png
var assets embed.FS

// compress turns the PDF streams' compression on; the tests turn it off to read inside the file.
var compress = true

// photoTimeout is how long a photo may take to download; maxPhotoBytes how big it may be.
// A photo that exceeds either is left out: it never stops the PDF.
var photoTimeout = 5 * time.Second

const maxPhotoBytes = 5 << 20

// Brand colors (docs/diseno.md).
var (
	crema = [3]int{250, 246, 241}
	oro   = [3]int{115, 85, 35}
	negro = [3]int{28, 28, 28}
	lino  = [3]int{232, 221, 211}
)

const (
	fontTitle  = "playfair"
	fontLight  = "dmsans-light"
	fontNormal = "dmsans"
	fontMedium = "dmsans-medium"
)

// QuoteLine is one product of the quotation, as it was saved.
type QuoteLine struct {
	Name      string
	PhotoURL  *string
	Quantity  int
	SoldAs    string // "recipe" or "portion"
	UnitPrice int64
	LineTotal int64
}

// Business is what the PDF prints about the business: contact and transfer data.
type Business struct {
	Name, Phone, Instagram     string
	Bank, AccountType, Account string
	Holder, RUT, Email         string
}

type QuoteData struct {
	ClientName   string
	Date         time.Time
	Lines        []QuoteLine
	DeliveryCost int64
	Total        int64
	Business     Business
}

const (
	pageW, pageH = 210.0, 297.0
	lm           = 20.0
	contentW     = 170.0
	pageBottom   = 277.0
	rowPhotoH    = 16.0
	rowPlainH    = 10.0
	// The block after the lines (totals, transfer data, notice) must fit on one page.
	closingH = 100.0
)

// GenerateQuotePDF builds the A4 quotation: logo and client, one row per line, totals, transfer
// data, contact and the 48 hours notice. The rows continue on another page when they do not fit;
// the totals and the data always travel together on the last page.
func GenerateQuotePDF(data QuoteData) (*bytes.Buffer, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetCompression(compress)
	pdf.SetMargins(lm, 20, lm)
	pdf.SetAutoPageBreak(false, 0)
	pdf.SetTitle("Cotización "+data.ClientName, true)
	pdf.SetCreationDate(data.Date)
	for name, file := range map[string]string{
		fontTitle: "PlayfairDisplay-Regular.ttf", fontLight: "DMSans-Light.ttf",
		fontNormal: "DMSans-Regular.ttf", fontMedium: "DMSans-Medium.ttf",
	} {
		b, err := assets.ReadFile("assets/fonts/" + file)
		if err != nil {
			return nil, fmt.Errorf("font %s: %w", file, err)
		}
		pdf.AddUTF8FontFromBytes(name, "", b)
	}
	logo, err := assets.ReadFile("assets/logo-color.png")
	if err != nil {
		return nil, fmt.Errorf("logo: %w", err)
	}
	pdf.RegisterImageOptionsReader("logo", gofpdf.ImageOptions{ImageType: "png"}, bytes.NewReader(logo))

	pdf.SetHeaderFunc(func() { // every page has the cream ground
		pdf.SetFillColor(crema[0], crema[1], crema[2])
		pdf.Rect(0, 0, pageW, pageH, "F")
	})

	photos := fetchPhotos(data.Lines)
	registered := map[string]bool{}
	hasPhotos := false
	for _, p := range photos {
		if p != nil {
			hasPhotos = true
		}
	}

	text := func(font string, size float64, c [3]int) {
		pdf.SetFont(font, "", size)
		pdf.SetTextColor(c[0], c[1], c[2])
	}
	rule := func(y float64, c [3]int, width float64) {
		pdf.SetDrawColor(c[0], c[1], c[2])
		pdf.SetLineWidth(width)
		pdf.Line(lm, y, lm+contentW, y)
	}
	cell := func(x, y, w, h float64, s, align string) {
		pdf.SetXY(x, y)
		pdf.CellFormat(w, h, s, "", 0, align, false, 0, "")
	}

	// Columns of the lines table.
	const (
		photoW = 18.0
		qtyW   = 24.0
		unitW  = 30.0
		subW   = 30.0
	)
	nameX := lm
	if hasPhotos {
		nameX = lm + photoW
	}
	nameW := contentW - (nameX - lm) - qtyW - unitW - subW
	qtyX := nameX + nameW
	unitX := qtyX + qtyW
	subX := unitX + unitW

	pageHeader := func(first bool) float64 {
		y := 20.0
		if first {
			pdf.ImageOptions("logo", lm, y, 40, 40*414.0/1000.0, false, gofpdf.ImageOptions{ImageType: "png"}, 0, "")
			text(fontLight, 9, oro)
			cell(lm+contentW-70, y+1, 70, 5, "COTIZACIÓN", "R")
			text(fontLight, 9, negro)
			cell(lm+contentW-70, y+7, 70, 5, data.Date.Format("02/01/2006"), "R")
			text(fontMedium, 10, negro)
			cell(lm+contentW-90, y+13, 90, 6, "Cliente: "+data.ClientName, "R")
			y += 40*414.0/1000.0 + 6
		}
		rule(y, oro, 0.4)
		y += 4
		text(fontMedium, 7.5, oro)
		cell(nameX, y, nameW, 5, "PRODUCTO", "L")
		cell(qtyX, y, qtyW, 5, "CANTIDAD", "R")
		cell(unitX, y, unitW, 5, "PRECIO UNIT.", "R")
		cell(subX, y, subW, 5, "SUBTOTAL", "R")
		y += 6
		rule(y, lino, 0.2)
		return y + 2
	}

	pdf.AddPage()
	y := pageHeader(true)

	for i, l := range data.Lines {
		rowH := rowPlainH
		if hasPhotos {
			rowH = rowPhotoH
		}
		if y+rowH > pageBottom {
			pdf.AddPage()
			y = pageHeader(false)
		}
		if p := photos[i]; p != nil {
			name := photoName(*l.PhotoURL)
			if !registered[name] {
				pdf.RegisterImageOptionsReader(name, gofpdf.ImageOptions{ImageType: "jpeg"}, bytes.NewReader(p.jpeg))
				registered[name] = true
			}
			const box = 14.0
			w, h := box, box
			if p.w >= p.h {
				h = box * float64(p.h) / float64(p.w)
			} else {
				w = box * float64(p.w) / float64(p.h)
			}
			pdf.ImageOptions(name, lm+(box-w)/2, y+(rowH-h)/2-1, w, h, false, gofpdf.ImageOptions{ImageType: "jpeg"}, 0, "")
		}
		mid := y + (rowH-5)/2 - 1
		text(fontTitle, 11, negro)
		cell(nameX, mid, nameW-2, 5, fit(pdf, l.Name, nameW-2), "L")
		qty := strconv.Itoa(l.Quantity)
		if l.SoldAs == "portion" {
			qty += " porciones"
		}
		text(fontLight, 10, negro)
		cell(qtyX, mid, qtyW, 5, qty, "R")
		text(fontNormal, 10, negro)
		cell(unitX, mid, unitW, 5, clp(l.UnitPrice), "R")
		cell(subX, mid, subW, 5, clp(l.LineTotal), "R")
		y += rowH
		rule(y-1, lino, 0.15)
	}

	// Closing block: totals, transfer data, contact, notice.
	if y+closingH > pageBottom {
		pdf.AddPage()
		y = pageHeader(false)
	}
	y += 6
	labelX, valueX, valueW := lm+contentW-90, lm+contentW-40, 40.0
	if data.DeliveryCost > 0 {
		text(fontLight, 10, negro)
		cell(labelX, y, 50, 6, "Subtotal", "L")
		cell(valueX, y, valueW, 6, clp(data.Total-data.DeliveryCost), "R")
		y += 6
		cell(labelX, y, 50, 6, "Envío", "L")
		cell(valueX, y, valueW, 6, clp(data.DeliveryCost), "R")
		y += 8
	}
	pdf.SetDrawColor(oro[0], oro[1], oro[2])
	pdf.SetLineWidth(0.4)
	pdf.Line(labelX, y, lm+contentW, y)
	y += 3
	text(fontMedium, 8, oro)
	cell(labelX, y+3, 40, 6, "TOTAL", "L")
	text(fontTitle, 22, oro)
	cell(valueX-20, y, valueW+20, 12, clp(data.Total), "R")
	y += 22

	rule(y, oro, 0.4)
	y += 6
	half := contentW / 2
	text(fontMedium, 8, oro)
	cell(lm, y, half, 5, "DATOS DE TRANSFERENCIA", "L")
	cell(lm+half+5, y, half-5, 5, "CONTACTO", "L")
	y += 7
	b := data.Business
	text(fontLight, 10, negro)
	transfer := []string{b.Holder, "RUT: " + b.RUT, b.Bank, b.AccountType, "N° de cuenta: " + b.Account, b.Email}
	for i, s := range transfer {
		cell(lm, y+float64(i)*5.5, half, 5, s, "L")
	}
	for i, s := range []string{b.Name, b.Phone, b.Instagram} {
		cell(lm+half+5, y+float64(i)*5.5, half-5, 5, s, "L")
	}
	y += float64(len(transfer))*5.5 + 6

	pdf.SetFillColor(lino[0], lino[1], lino[2])
	pdf.Rect(lm, y, contentW, 12, "F")
	text(fontLight, 9, negro)
	cell(lm+4, y+3.5, contentW-8, 5, "Esta cotización debe confirmarse dentro de 48 horas para reservar tu pedido.", "L")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("write pdf: %w", err)
	}
	return &buf, nil
}

// clp writes whole pesos with thousands dots: $12.500.
func clp(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := strconv.FormatInt(n, 10)
	var out strings.Builder
	for i, ch := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out.WriteRune('.')
		}
		out.WriteRune(ch)
	}
	if neg {
		return "-$" + out.String()
	}
	return "$" + out.String()
}

// fit cuts a text with an ellipsis so it does not leave its column.
func fit(pdf *gofpdf.Fpdf, s string, width float64) string {
	if pdf.GetStringWidth(s) <= width {
		return s
	}
	r := []rune(s)
	for len(r) > 1 && pdf.GetStringWidth(string(r)+"...") > width {
		r = r[:len(r)-1]
	}
	return string(r) + "..."
}

// --- photos ---

type photo struct {
	jpeg []byte
	w, h int
}

func photoName(url string) string { return "photo-" + url }

// fetchPhotos downloads each distinct photo once, all at the same time. The result is aligned with
// the lines: nil where there is no photo or it could not be read.
func fetchPhotos(lines []QuoteLine) []*photo {
	out := make([]*photo, len(lines))
	unique := map[string]*photo{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, l := range lines {
		if l.PhotoURL == nil || *l.PhotoURL == "" {
			continue
		}
		url := *l.PhotoURL
		mu.Lock()
		_, seen := unique[url]
		unique[url] = nil
		mu.Unlock()
		if seen {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			p := downloadPhoto(url)
			mu.Lock()
			unique[url] = p
			mu.Unlock()
		}()
	}
	wg.Wait()
	for i, l := range lines {
		if l.PhotoURL != nil {
			out[i] = unique[*l.PhotoURL]
		}
	}
	return out
}

// downloadPhoto returns the photo as a JPEG on a white ground (whatever it was: the type comes from
// the bytes), or nil when it cannot be fetched or read.
func downloadPhoto(url string) *photo {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return nil
	}
	client := &http.Client{Timeout: photoTimeout}
	resp, err := client.Get(url) //nolint:gosec // the address was stored by the app itself
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxPhotoBytes+1))
	if err != nil || len(data) > maxPhotoBytes {
		return nil
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	b := src.Bounds()
	if b.Dx() == 0 || b.Dy() == 0 {
		return nil
	}
	w, h := b.Dx(), b.Dy()
	const maxSide = 480
	if w > maxSide || h > maxSide { // a thumbnail is all the row needs
		if w >= h {
			h, w = h*maxSide/w, maxSide
		} else {
			w, h = w*maxSide/h, maxSide
		}
		if w < 1 {
			w = 1
		}
		if h < 1 {
			h = 1
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, xdraw.Src)
	xdraw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, b, xdraw.Over, nil)
	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: 85}); err != nil {
		return nil
	}
	return &photo{jpeg: out.Bytes(), w: w, h: h}
}
