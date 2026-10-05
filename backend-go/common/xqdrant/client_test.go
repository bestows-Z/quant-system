package xqdrant

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEnsureUpsertAndSearch(t *testing.T) {
	collectionExists := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/collections/kb_1":
			if !collectionExists {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case r.Method == http.MethodPut && r.URL.Path == "/collections/kb_1":
			collectionExists = true
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case r.Method == http.MethodPut && r.URL.Path == "/collections/kb_1/points":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			points := body["points"].([]any)
			if len(points) != 1 {
				t.Fatal("expected one point")
			}
			point := points[0].(map[string]any)
			if point["id"] != float64(7) || point["vector"] == nil || point["payload"] == nil {
				t.Fatalf("invalid qdrant point contract: %#v", point)
			}
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/collections/kb_1/points/search":
			_, _ = w.Write([]byte(`{"result":[{"id":7,"score":0.91,"payload":{"content":"风险控制"}}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := New(server.URL, "default")
	ctx := context.Background()
	if err := client.EnsureCollection(ctx, "kb_1", 384); err != nil {
		t.Fatal(err)
	}
	if err := client.Upsert(ctx, "kb_1", Point{ID: 7, Vector: []float64{1, 0}, Payload: map[string]any{"content": "风险控制"}}); err != nil {
		t.Fatal(err)
	}
	hits, err := client.Search(ctx, "kb_1", []float64{1, 0}, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].Score != 0.91 {
		t.Fatalf("unexpected hits: %#v", hits)
	}
}
