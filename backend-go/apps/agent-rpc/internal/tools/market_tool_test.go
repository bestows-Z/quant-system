package tools

import "testing"

func TestDeduplicateResearchByURLAndTitleTime(t *testing.T) {
	value := map[string]any{"data": map[string]any{"items": []any{
		map[string]any{"title": "公告一", "url": "https://example.test/1", "published_at": "2026-07-18"},
		map[string]any{"title": "重复公告", "url": "https://example.test/1", "published_at": "2026-07-18"},
		map[string]any{"title": "公告二", "url": "", "published_at": "2026-07-19"},
		map[string]any{"title": "公告二", "url": "", "published_at": "2026-07-19"},
	}}}
	result := deduplicateResearch(value, 20).(map[string]any)["data"].(map[string]any)
	if result["count"] != 2 {
		t.Fatalf("expected two unique research items, got %#v", result)
	}
}
