package service

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

// Sprint 18 (P1-P6): the PDF for a person shows only final prices; the one for a company shows the detail
// and the conditions of the order. The final prices are the same in both: only what is shown changes.

func forPerson(lines ...QuoteLine) QuoteData {
	d := sample(lines...)
	d.Kind = KindPerson
	return d
}

func forCompany(lines ...QuoteLine) QuoteData {
	d := sample(lines...)
	d.Kind = KindCompany
	d.ValidDays = 7
	return d
}

func text(t *testing.T, d QuoteData) string { return pdftotext(t, generate(t, d), "-layout") }

func TestP1_APersonWithOneProductSeesOnlyTheTotalAndTheDelivery(t *testing.T) {
	d := forPerson(line("Torta de milhojas", 2, "recipe", 25000, 50000))
	d.DeliveryCost, d.Total = 3000, 53000
	out := text(t, d)
	for _, gone := range []string{"PRECIO UNIT", "CANTIDAD", "SUBTOTAL", "Subtotal", "PRODUCTO"} {
		if strings.Contains(out, gone) {
			t.Errorf("a one-product quotation for a person still shows %q:\n%s", gone, out)
		}
	}
	for _, want := range []string{"TOTAL", "$53.000", "Envío", "$3.000"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "$25.000") {
		t.Errorf("the unit price must not be shown to a person:\n%s", out)
	}
}

func TestP2_ADeliveryThatCostsNothingSaysSo(t *testing.T) {
	out := text(t, forPerson(line("Torta", 1, "recipe", 25000, 25000)))
	if !regexp.MustCompile(`Envío\s+Sin costo`).MatchString(out) {
		t.Errorf("a free delivery should read \"Envío  Sin costo\":\n%s", out)
	}
}

func TestP3_APersonWithSeveralProductsSeesEachFinalPriceAndNoBreakdown(t *testing.T) {
	d := forPerson(line("Torta de milhojas", 2, "recipe", 25000, 50000), line("Pie de limón", 1, "recipe", 18500, 18500))
	d.DeliveryCost, d.Total = 2500, 71000
	out := text(t, d)
	for _, want := range []string{"Torta de milhojas", "$50.000", "Pie de limón", "$18.500", "Envío", "$2.500", "TOTAL", "$71.000"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	for _, gone := range []string{"PRECIO UNIT", "SUBTOTAL", "Subtotal", "$25.000"} {
		if strings.Contains(out, gone) {
			t.Errorf("the list for a person shows %q (it counts the same thing twice):\n%s", gone, out)
		}
	}
	if !regexp.MustCompile(`Torta de milhojas\s*.*2`).MatchString(out) {
		t.Errorf("the quantity of the line must be visible:\n%s", out)
	}
}

func TestP4_ACompanySeesTheDetailAsBefore(t *testing.T) {
	d := forCompany(line("Torta de milhojas", 2, "recipe", 25000, 50000))
	d.DeliveryCost, d.Total = 3000, 53000
	out := text(t, d)
	for _, want := range []string{"PRODUCTO", "CANTIDAD", "PRECIO UNIT", "SUBTOTAL", "$25.000", "$50.000", "Subtotal", "Envío", "$3.000", "TOTAL", "$53.000"} {
		if !strings.Contains(out, want) {
			t.Errorf("the PDF of a company misses %q:\n%s", want, out)
		}
	}
}

func TestP5_ACompanySeesTheConditionsOfTheOrder(t *testing.T) {
	d := forCompany(line("Torta", 1, "recipe", 25000, 25000))
	date := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)
	d.DeliveryDate, d.PaymentTerms, d.ValidDays = &date, "Transferencia a 30 días", 10
	out := text(t, d)
	for _, want := range []string{"CONDICIONES", "Fecha de entrega", "15/04/2026", "Forma de pago", "Transferencia a 30 días", "Validez", "10 días", "10/04/2026"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "48 horas") {
		t.Errorf("a company quotation uses its own validity, not the 48 hours of the people:\n%s", out)
	}
}

func TestP6_ACompanyWithNoConditionsStillSaysHowLongItIsValid(t *testing.T) {
	out := text(t, forCompany(line("Torta", 1, "recipe", 25000, 25000)))
	if !strings.Contains(out, "Validez") || !strings.Contains(out, "7 días") {
		t.Errorf("the validity is missing:\n%s", out)
	}
	for _, gone := range []string{"Fecha de entrega", "Forma de pago"} {
		if strings.Contains(out, gone) {
			t.Errorf("%q has no value and must not appear:\n%s", gone, out)
		}
	}
}

func TestP7_ThePersonKeepsTheNoticeOf48HoursAndBothKeepTheTransferData(t *testing.T) {
	for name, d := range map[string]QuoteData{
		"person":  forPerson(line("Torta", 1, "recipe", 25000, 25000)),
		"company": forCompany(line("Torta", 1, "recipe", 25000, 25000)),
	} {
		out := text(t, d)
		for _, want := range []string{"DATOS DE TRANSFERENCIA", "Banco Prueba", "CONTACTO", "pagos@prueba.test"} {
			if !strings.Contains(out, want) {
				t.Errorf("%s: missing %q:\n%s", name, want, out)
			}
		}
		if name == "person" && !strings.Contains(out, "48 horas") {
			t.Errorf("the notice of 48 hours is gone from the quotation of a person:\n%s", out)
		}
	}
}

func TestP8_TheKindThatIsNotPersonKeepsTheFullDetail(t *testing.T) {
	// The quotations made before the kinds existed (and the tests of sprint 12) have no kind: they keep the detail.
	out := text(t, sample(line("Torta", 2, "recipe", 25000, 50000)))
	if !strings.Contains(out, "PRECIO UNIT") || !strings.Contains(out, "SUBTOTAL") {
		t.Errorf("a quotation with no kind must keep the detail:\n%s", out)
	}
}
