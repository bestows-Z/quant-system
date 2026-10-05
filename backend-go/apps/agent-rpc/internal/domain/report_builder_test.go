package domain

import (
	"strings"
	"testing"
)

func TestSanitizeReportRemovesDeterministicAdviceLanguage(t *testing.T) {
	report := sanitizeReport(AnalysisReport{
		Summary:      "推荐买入，必涨稳赚",
		RiskAnalysis: "已有持仓需注意止损，不应梭哈",
	})
	for _, blocked := range []string{"推荐买入", "必涨", "稳赚", "梭哈", "需注意止损"} {
		if strings.Contains(report.Summary+report.RiskAnalysis, blocked) {
			t.Fatalf("report still contains blocked wording %q", blocked)
		}
	}
	if !strings.Contains(report.RiskAnalysis, "风险限额与回撤约束") {
		t.Fatal("risk wording was not converted into a research-oriented warning")
	}
}

func TestResearchReferencesArchiveAnnouncementEvidence(t *testing.T) {
	data := map[string]any{"announcements": map[string]any{"data": map[string]any{"items": []any{map[string]any{"title": "年度报告", "url": "https://example.test/report", "source": "cninfo", "published_at": "2026-07-18"}}}}}
	refs := researchReferences(data)
	if len(refs) != 1 || refs[0]["url"] != "https://example.test/report" || refs[0]["evidence_type"] != "announcements" {
		t.Fatalf("unexpected archived references: %#v", refs)
	}
}
