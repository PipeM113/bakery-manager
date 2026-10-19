package service

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Sprint 12 (Q8, Q9, Q10): the quotation PDF carries the brand, one row per line, and a photo that
// cannot be read never stops it.

var business = Business{
	Name: "Panadería de Prueba", Phone: "+56 9 1111 2222", Instagram: "@prueba",
	Bank: "Banco Prueba", Account: "000-111222-3", Holder: "Titular Prueba", RUT: "11.111.111-1",
}

func line(name string, qty int, soldAs string, unit, total int64) QuoteLine {
	return QuoteLine{Name: name, Quantity: qty, SoldAs: soldAs, UnitPrice: unit, LineTotal: total}
}

func sample(lines ...QuoteLine) QuoteData {
	var sum int64
	for _, l := range lines {
		sum += l.LineTotal
	}
	return QuoteData{
		ClientName: "Ana Pérez", Date: time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC),
		Lines: lines, Total: sum, Business: business,
	}
}

func generate(t *testing.T, data QuoteData) []byte {
	t.Helper()
	buf, err := GenerateQuotePDF(data)
	if err != nil {
		t.Fatalf("GenerateQuotePDF: %v", err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF")) {
		t.Fatalf("not a PDF: %.20q", buf.Bytes())
	}
	return buf.Bytes()
}

// raw makes the PDF readable (no compressed streams) for the tests that look inside it.
func raw(t *testing.T, data QuoteData) string {
	t.Helper()
	was := compress
	compress = false
	t.Cleanup(func() { compress = was })
	return string(generate(t, data))
}

func pdftotext(t *testing.T, pdf []byte, args ...string) string {
	t.Helper()
	path, err := exec.LookPath("pdftotext")
	if err != nil {
		t.Skip("pdftotext (poppler-utils) is not installed")
	}
	cmd := exec.Command(path, append(args, "-", "-")...)
	cmd.Stdin = bytes.NewReader(pdf)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("pdftotext: %v", err)
	}
	return string(out)
}

var pageRE = regexp.MustCompile(`/Type /Page[^s]`)

func pages(raw string) int { return len(pageRE.FindAllString(raw, -1)) }

func images(raw string) int { return strings.Count(raw, "/Subtype /Image") }

// --- Q8: the brand ---

func TestQ8_UsesTheBrandColorsAndNotTheOldOnes(t *testing.T) {
	out := raw(t, sample(line("Torta", 1, "recipe", 1500, 1500)))
	// A grey (equal red, green and blue) is written by the PDF library as a single value: "0.110 g".
	for name, c := range map[string]string{
		"crema (#FAF6F1)": `0\.980 0\.965 0\.945 (rg|RG)`,
		"oro (#735523)":   `0\.451 0\.333 0\.137 (rg|RG)`,
		"negro (#1C1C1C)": `0\.110 (g|G)\b`,
	} {
		if !regexp.MustCompile(c).MatchString(out) {
			t.Errorf("the PDF does not use %s", name)
		}
	}
	for name, c := range map[string]string{
		"the old gold":  "0.706 0.549 0.235",
		"the old brown": "0.161 0.145 0.141",
	} {
		if strings.Contains(out, c) {
			t.Errorf("the PDF still uses %s (%s)", name, c)
		}
	}
}

func TestQ8_EmbedsPlayfairAndDMSansAndNoHelvetica(t *testing.T) {
	out := raw(t, sample(line("Torta", 1, "recipe", 1500, 1500)))
	if !regexp.MustCompile(`(?i)Playfair`).MatchString(out) {
		t.Error("Playfair Display is not in the PDF")
	}
	if !regexp.MustCompile(`(?i)DM ?Sans`).MatchString(out) {
		t.Error("DM Sans is not in the PDF")
	}
	if strings.Contains(out, "Helvetica") {
		t.Error("the PDF still uses Helvetica")
	}
}

func TestQ8_TheLogoIsInThePDF(t *testing.T) {
	if images(raw(t, sample(line("Torta", 1, "recipe", 1500, 1500)))) < 1 {
		t.Error("no logo image in the PDF")
	}
}

func TestQ8_FontsLogoAndTheirSourcesAreInTheRepository(t *testing.T) {
	for _, f := range []string{"PlayfairDisplay-Regular.ttf", "DMSans-Light.ttf", "DMSans-Regular.ttf", "DMSans-Medium.ttf"} {
		b, err := os.ReadFile(filepath.Join("assets", "fonts", f))
		if err != nil {
			t.Errorf("%s: %v", f, err)
			continue
		}
		if len(b) < 20_000 || !bytes.HasPrefix(b, []byte{0, 1, 0, 0}) {
			t.Errorf("%s is not a TrueType font (%d bytes)", f, len(b))
		}
	}
	if _, err := os.Stat(filepath.Join("assets", "fonts", "OFL.txt")); err != nil {
		t.Errorf("the fonts' licence (OFL) is not included: %v", err)
	}
	f, err := os.Open(filepath.Join("assets", "logo-color.png"))
	if err != nil {
		t.Fatalf("logo: %v", err)
	}
	defer f.Close()
	cfg, err := png.DecodeConfig(f)
	if err != nil || cfg.Width < 400 {
		t.Errorf("logo-color.png is not a PNG of at least 400 px wide (%v, %+v)", err, cfg)
	}
}

func TestQ8_TheSourcesAreWrittenDown(t *testing.T) {
	b, err := os.ReadFile("../../../../docs/fuentes.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Playfair Display", "DM Sans", "OFL"} {
		if !strings.Contains(string(b), want) {
			t.Errorf("docs/fuentes.md does not mention %q", want)
		}
	}
	d, err := os.ReadFile("../../../../docs/diseno.md")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(d), "El PDF de las cotizaciones aún no usa") {
		t.Error("docs/diseno.md still says the PDF has no brand")
	}
}

// --- Q9: what the PDF says ---

func TestQ9_OneRowPerLineWithTheNumbers(t *testing.T) {
	text := pdftotext(t, generate(t, sample(
		line("Torta de limón", 2, "recipe", 15000, 30000),
		line("Ñandú de chocolate", 12, "portion", 2500, 30000),
	)))
	for _, want := range []string{"Ana Pérez", "Torta de limón", "Ñandú de chocolate", "$15.000", "$30.000", "$60.000", "31/03/2026"} {
		if !strings.Contains(text, want) {
			t.Errorf("the PDF text does not contain %q:\n%s", want, text)
		}
	}
}

func TestQ9_DeliveryShowsOnlyWhenThereIsOne(t *testing.T) {
	with := sample(line("Torta", 1, "recipe", 10000, 10000))
	with.DeliveryCost = 2500
	with.Total = 12500
	text := strings.ToLower(pdftotext(t, generate(t, with)))
	for _, want := range []string{"envío", "$2.500", "$12.500"} {
		if !strings.Contains(text, strings.ToLower(want)) {
			t.Errorf("with delivery, the PDF does not contain %q", want)
		}
	}
	text = strings.ToLower(pdftotext(t, generate(t, sample(line("Torta", 1, "recipe", 10000, 10000)))))
	if strings.Contains(text, "envío") {
		t.Error("without delivery, the PDF still has the delivery row")
	}
}

func TestQ9_TransferDataContactAndTheNotice(t *testing.T) {
	text := pdftotext(t, generate(t, sample(line("Torta", 1, "recipe", 1500, 1500))))
	for _, want := range []string{business.Name, business.Phone, business.Instagram, business.Bank, business.Account, business.Holder, business.RUT, "48 horas"} {
		if !strings.Contains(text, want) {
			t.Errorf("the PDF does not contain %q", want)
		}
	}
}

func TestQ9_ManyLinesContinueOnAnotherPageAndTheTotalIsAtTheEnd(t *testing.T) {
	few := raw(t, sample(line("Torta", 1, "recipe", 1500, 1500), line("Pie", 2, "recipe", 1000, 2000)))
	if n := pages(few); n != 1 {
		t.Errorf("two lines take %d pages, want 1", n)
	}
	var lines []QuoteLine
	for i := 1; i <= 30; i++ {
		lines = append(lines, line("Producto "+strconv.Itoa(i), 1, "recipe", 1000, 1000))
	}
	data := sample(lines...)
	pdf := generate(t, data)
	was := compress
	compress = false
	many := string(generate(t, data))
	compress = was
	n := pages(many)
	if n < 2 {
		t.Fatalf("30 lines take %d page(s), want at least 2", n)
	}
	last := strings.ToLower(pdftotext(t, pdf, "-f", strconv.Itoa(n), "-l", strconv.Itoa(n)))
	for _, want := range []string{"total", "$30.000", strings.ToLower(business.Account)} {
		if !strings.Contains(last, want) {
			t.Errorf("the last page does not contain %q", want)
		}
	}
	all := pdftotext(t, pdf)
	if !strings.Contains(all, "Producto 1") || !strings.Contains(all, "Producto 30") {
		t.Error("a line is missing from the PDF")
	}
}

// --- Q10: photos ---

func photoServer(t *testing.T) *httptest.Server {
	t.Helper()
	webp, _ := base64.StdEncoding.DecodeString("UklGRh4AAABXRUJQVlA4TBEAAAAvJ0AHAAdQvCKXp/+BiOh/AAA=")
	img := image.NewRGBA(image.Rect(0, 0, 60, 40))
	for x := 0; x < 60; x++ {
		for y := 0; y < 40; y++ {
			img.Set(x, y, color.RGBA{200, 120, 60, 255})
		}
	}
	var jpg, pn bytes.Buffer
	_ = jpeg.Encode(&jpg, img, nil)
	_ = png.Encode(&pn, img)

	mux := http.NewServeMux()
	// The names lie on purpose: the type is read from the bytes, not from the address.
	mux.HandleFunc("/jpeg-named.png", func(w http.ResponseWriter, _ *http.Request) { w.Write(jpg.Bytes()) })
	mux.HandleFunc("/png-named.jpg", func(w http.ResponseWriter, _ *http.Request) { w.Write(pn.Bytes()) })
	mux.HandleFunc("/webp-named.jpeg", func(w http.ResponseWriter, _ *http.Request) { w.Write(webp) })
	mux.HandleFunc("/garbage.jpg", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("this is not an image")) })
	mux.HandleFunc("/huge.jpg", func(w http.ResponseWriter, _ *http.Request) { w.Write(make([]byte, 6<<20)) })
	mux.HandleFunc("/slow.jpg", func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.Write(jpg.Bytes())
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func withPhoto(l QuoteLine, url string) QuoteLine { l.PhotoURL = &url; return l }

func TestQ10_JPEGPNGAndWebPAreDrawnWhateverTheirAddressSays(t *testing.T) {
	srv := photoServer(t)
	base := images(raw(t, sample(line("Torta", 1, "recipe", 1500, 1500))))
	for _, path := range []string{"/jpeg-named.png", "/png-named.jpg", "/webp-named.jpeg"} {
		got := images(raw(t, sample(withPhoto(line("Torta", 1, "recipe", 1500, 1500), srv.URL+path))))
		if got != base+1 {
			t.Errorf("%s: %d images, want %d (the logo and the photo)", path, got, base+1)
		}
	}
}

func TestQ10_APhotoThatCannotBeReadDoesNotStopThePDF(t *testing.T) {
	srv := photoServer(t)
	was := photoTimeout
	photoTimeout = 100 * time.Millisecond
	t.Cleanup(func() { photoTimeout = was })

	base := images(raw(t, sample(line("Torta", 1, "recipe", 1500, 1500))))
	for _, path := range []string{"/garbage.jpg", "/huge.jpg", "/slow.jpg", "/missing.jpg"} {
		data := sample(withPhoto(line("Torta", 1, "recipe", 1500, 1500), srv.URL+path), line("Pie", 1, "recipe", 1000, 1000))
		out := raw(t, data)
		if got := images(out); got != base {
			t.Errorf("%s: %d images, want %d (the line goes without photo)", path, got, base)
		}
	}
	data := sample(withPhoto(line("Torta", 1, "recipe", 1500, 1500), "http://127.0.0.1:1/unreachable.jpg"))
	if text := pdftotext(t, generate(t, data)); !strings.Contains(text, "Torta") {
		t.Error("the line of the photo that is unreachable is missing")
	}
}

func TestQ10_TheSamePhotoTwiceIsEmbeddedOnce(t *testing.T) {
	srv := photoServer(t)
	base := images(raw(t, sample(line("Torta", 1, "recipe", 1500, 1500))))
	url := srv.URL + "/jpeg-named.png"
	got := images(raw(t, sample(
		withPhoto(line("Torta", 1, "recipe", 1500, 1500), url),
		withPhoto(line("Torta grande", 2, "recipe", 1500, 3000), url),
	)))
	if got != base+1 {
		t.Errorf("%d images, want %d", got, base+1)
	}
}
