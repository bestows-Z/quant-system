package domain

import "testing"

func TestParseCSV(t *testing.T) {
	content := []byte("股票代码,股票名称,持仓数量,可用数量,成本价,建仓日期,备注\n600176,中国巨石,1000,800,46.80,2026-07-01,核心仓\n000001,平安银行,200,200,10.20,,观察\n")
	rows, errs := ParseCSV(content)
	if len(errs) != 0 || len(rows) != 2 {
		t.Fatalf("rows=%d errors=%v", len(rows), errs)
	}
	if rows[0].Symbol != "600176" || rows[0].AvgCost != "46.800000" {
		t.Fatalf("unexpected first row: %+v", rows[0])
	}
}

func TestParseCSVRejectsInvalidAndDuplicateRows(t *testing.T) {
	content := []byte("symbol,quantity,available_quantity,avg_cost\n600176,100,101,46.8\n600176,100,100,46.8\n600176,200,200,47.1\n")
	rows, errs := ParseCSV(content)
	if len(rows) != 1 || len(errs) != 2 {
		t.Fatalf("rows=%d errors=%d", len(rows), len(errs))
	}
}

func TestParseCSVRequiresColumns(t *testing.T) {
	_, errs := ParseCSV([]byte("symbol,quantity\n600176,100\n"))
	if len(errs) != 1 || errs[0].Field != "avg_cost" {
		t.Fatalf("unexpected errors: %+v", errs)
	}
}

func TestParseQuoteRawReadsItemsEnvelope(t *testing.T) {
	item, err := parseQuoteRaw(`{"code":0,"data":{"items":[{"symbol":"600176","name":"中国巨石","price":47.71,"source":"tencent","data_mode":"real","quote_time":"2026-07-16 15:00:00"}]}}`)
	if err != nil {
		t.Fatal(err)
	}
	if item.Price.StringFixed(2) != "47.71" || item.Source != "tencent" || item.DataMode != "real" {
		t.Fatalf("unexpected quote: %+v", item)
	}
}
