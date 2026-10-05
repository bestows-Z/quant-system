package client

import (
	"encoding/json"
	"testing"
)

func TestDataQualityKeepsHistoricalEvidenceCounts(t *testing.T) {
	var quality DataQuality
	if err := json.Unmarshal([]byte(`{"status":"PARTIAL","valid_rows":66,"verified_status_rows":40,"verified_limit_rows":0}`), &quality); err != nil {
		t.Fatal(err)
	}
	if quality.ValidRows != 66 || quality.VerifiedStatusRows != 40 || quality.VerifiedLimitRows != 0 {
		t.Fatalf("quality evidence lost: %+v", quality)
	}
	encoded, err := json.Marshal(quality)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["verified_status_rows"] != float64(40) {
		t.Fatalf("status count missing from persisted JSON: %s", encoded)
	}
}
