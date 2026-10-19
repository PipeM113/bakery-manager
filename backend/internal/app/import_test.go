package app_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xuri/excelize/v2"
)

// Sprint 2a, rule P8: a bad price or size rejects its row with the reason; it no longer
// enters as 0 or 1. Good rows in the same file are still imported.
func TestP8_ImportRejectsBadNumbersAndKeepsTheGoodRows(t *testing.T) {
	api, _ := newPricingAPI(t)

	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	rows := [][]string{
		{"Nombre", "Marca", "Unidad", "Tamaño", "Precio"},
		{"Harina", "", "gr", "1000", "2000"},   // row 2: fine
		{"Azucar", "", "kg", "1", "1.500"},     // row 3: 1.500 is one thousand five hundred
		{"Sal", "", "kg", "1", "abc"},          // row 4: not a number
		{"Aceite", "", "lt", "", "1500"},       // row 5: empty size
		{"Leche", "", "lt", "1", "-200"},       // row 6: negative price
		{"Cacao", "", "gr", "250", "1.500,50"}, // row 7: fine, 1500.50
		{"Regalo", "", "gr", "1000", "0"},      // row 8: price 0
		{"Levadura", "", "gr", "-5", "800"},    // row 9: negative size
	}
	for i, r := range rows {
		for j, v := range r {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+1)
			if err := f.SetCellStr(sheet, cell, v); err != nil {
				t.Fatal(err)
			}
		}
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, _ := mw.CreateFormFile("file", "insumos.xlsx")
	part.Write(buf.Bytes()) //nolint:errcheck
	mw.Close()
	req := httptest.NewRequest("POST", "/ingredients/import", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+api.token)
	rec := httptest.NewRecorder()
	api.h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}

	var result struct {
		Imported int `json:"imported"`
		Errors   []struct {
			Row    int    `json:"row"`
			Reason string `json:"reason"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Imported != 3 {
		t.Errorf("imported = %d, want 3 (Harina, Azucar, Cacao)", result.Imported)
	}
	gotRows := map[int]string{}
	for _, e := range result.Errors {
		gotRows[e.Row] = e.Reason
		if e.Reason == "" {
			t.Errorf("row %d has no reason", e.Row)
		}
	}
	for _, row := range []int{4, 5, 6, 8, 9} {
		if _, ok := gotRows[row]; !ok {
			t.Errorf("row %d should have been rejected; errors = %v", row, result.Errors)
		}
	}
	if len(result.Errors) != 5 {
		t.Errorf("errors = %v, want exactly rows 4, 5, 6, 8 and 9", result.Errors)
	}

	status, raw := api.do("GET", "/ingredients", nil)
	if status != http.StatusOK {
		t.Fatalf("list: status = %d", status)
	}
	var list []map[string]any
	if err := json.Unmarshal(raw, &list); err != nil {
		t.Fatal(err)
	}
	prices := map[string]float64{}
	for _, ing := range list {
		prices[ing["name"].(string)] = ing["package_price"].(float64)
	}
	if len(prices) != 3 {
		t.Errorf("stored ingredients = %v, want only Harina, Azucar and Cacao", prices)
	}
	if prices["Azucar"] != 1500 {
		t.Errorf("Azucar package_price = %v, want 1500", prices["Azucar"])
	}
	if prices["Cacao"] != 1500.5 {
		t.Errorf("Cacao package_price = %v, want 1500.5", prices["Cacao"])
	}
}
