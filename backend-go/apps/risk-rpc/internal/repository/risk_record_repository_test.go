package repository

import (
	"strings"
	"testing"
)

func TestRiskListProjectionExcludesRawEvidence(t *testing.T) {
	projection := strings.ToLower(riskListProjection)
	if strings.Contains(projection, "raw_data_json") || strings.Contains(projection, "raw_json") {
		t.Fatal("risk list projection must not load raw evidence")
	}
	if !strings.Contains(projection, "'{}'") {
		t.Fatal("risk list projection must preserve the response contract with empty evidence")
	}
}
