package domain

import (
	"testing"

	"quant-system/backend-go/apps/rag-rpc/rag"
)

func TestMergeSearchResultsPreservesInputs(t *testing.T) {
	vector := &rag.SearchResult{ChunkId: 1, Score: 0.8, Content: "vector"}
	keyword := &rag.SearchResult{ChunkId: 1, Score: 0.6, Content: "keyword"}
	items := mergeSearchResults([]*rag.SearchResult{vector}, []*rag.SearchResult{keyword}, 10)
	if len(items) != 1 || items[0].Score < 0.7399 || items[0].Score > 0.7401 {
		t.Fatalf("unexpected merged score: %+v", items)
	}
	if items[0] == vector || items[0] == keyword || vector.Score != 0.8 || keyword.Score != 0.6 {
		t.Fatal("merge must not modify or reuse the protobuf inputs")
	}
}
