package stream

import (
	"testing"
	"time"

	"quant-system/backend-go/apps/ws-api/internal/config"
	"quant-system/backend-go/common/markettime"
)

func TestDecodeQuotePayload(t *testing.T) {
	raw := `{"code":0,"message":"success","data":{"items":[{"symbol":"600176","price":21.5,"quote_time":"2026-07-17 14:59:59","source":"tencent","data_mode":"real"}],"source":"tencent","data_mode":"real"}}`
	payload, meta, err := decodePayload(raw, true)
	if err != nil {
		t.Fatal(err)
	}
	if payload["symbol"] != "600176" || meta.Source != "tencent" || meta.DataMode != "real" || meta.QuoteTime == "" {
		t.Fatalf("unexpected payload=%#v meta=%#v", payload, meta)
	}
}

func TestDecodeRejectsUnavailableUpstream(t *testing.T) {
	_, _, err := decodePayload(`{"code":503,"message":"source unavailable","data":{}}`, false)
	if err == nil {
		t.Fatal("expected upstream error")
	}
}

func TestShouldPollPreventsOverlappingSymbolRequests(t *testing.T) {
	streamer := NewMarketStreamer(nil, nil, nil, nil, config.StreamConf{})
	now := time.Date(2026, time.July, 20, 10, 0, 0, 0, markettime.Location())
	if !streamer.shouldPoll("600176", now) {
		t.Fatal("first poll should be admitted")
	}
	if streamer.shouldPoll("600176", now.Add(time.Second)) {
		t.Fatal("overlapping poll should be rejected")
	}
	streamer.finishPoll("600176")
	if !streamer.shouldPoll("600176", now.Add(2*time.Second)) {
		t.Fatal("poll should resume after prior request finishes")
	}
}
