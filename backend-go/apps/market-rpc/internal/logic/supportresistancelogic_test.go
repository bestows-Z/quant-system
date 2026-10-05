package logic

import "testing"

func TestParseSupportResistance(t *testing.T) {
	raw := `{"code":0,"data":{"symbol":"600176","period":"daily","quote_time":"2026-07-16","support_price":51.2,"pivot_price":53.1,"resistance_price":56.4,"support_strength":0.8,"resistance_strength":0.7,"methods":{"support":["swing_low"]},"algorithm_version":"support-resistance-v1","data_source":"tencent","valid_until":"2026-07-17T12:00:00Z"}}`
	item, ok := parseSupportResistance(raw)
	if !ok {
		t.Fatal("expected valid support resistance payload")
	}
	if item.Symbol != "600176" || item.SupportPrice != 51.2 || item.AlgorithmVersion != "support-resistance-v1" {
		t.Fatalf("unexpected item: %+v", item)
	}
}

func TestParseSupportResistanceRejectsErrorEnvelope(t *testing.T) {
	if _, ok := parseSupportResistance(`{"code":503,"data":null}`); ok {
		t.Fatal("expected error envelope rejection")
	}
}
