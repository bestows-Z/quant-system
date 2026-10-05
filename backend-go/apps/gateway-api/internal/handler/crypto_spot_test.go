package handler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type cryptoRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn cryptoRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return fn(r) }

func TestCryptoSpotMarketRejectsUnsupportedInputWithoutProxying(t *testing.T) {
	called := false
	client := &http.Client{Transport: cryptoRoundTripFunc(func(*http.Request) (*http.Response, error) {
		called = true
		return nil, context.Canceled
	})}
	handler := cryptoSpotMarketHandler("http://quant-python:8000", client)
	for _, query := range []string{"symbol=SOLUSDT&interval=1h", "symbol=BTCUSDT&interval=1m"} {
		recorder := httptest.NewRecorder()
		handler(recorder, httptest.NewRequest(http.MethodGet, "/api/crypto/spot/market?"+query, nil))
		if !strings.Contains(recorder.Body.String(), `"code":422`) {
			t.Fatalf("expected validation error, got %s", recorder.Body.String())
		}
	}
	if called {
		t.Fatal("invalid market input must not reach provider")
	}
}

func TestCryptoSpotMarketProxiesFixedEndpointAndPreservesFailure(t *testing.T) {
	var requested string
	client := &http.Client{Transport: cryptoRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		requested = r.URL.String()
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"code":503,"message":"币安公开行情不可用","data":{}}`))}, nil
	})}
	recorder := httptest.NewRecorder()
	cryptoSpotMarketHandler("http://quant-python:8000", client)(recorder, httptest.NewRequest(http.MethodGet, "/api/crypto/spot/market?symbol=ETHUSDT&interval=4h", nil))
	if requested != "http://quant-python:8000/api/crypto/spot/market?interval=4h&symbol=ETHUSDT" {
		t.Fatalf("unexpected upstream URL %s", requested)
	}
	if !strings.Contains(recorder.Body.String(), `"code":503`) {
		t.Fatalf("provider error must remain visible: %s", recorder.Body.String())
	}
}

func TestCryptoSpotMarketReturnsRealPayload(t *testing.T) {
	client := &http.Client{Transport: cryptoRoundTripFunc(func(*http.Request) (*http.Response, error) {
		body := `{"code":0,"message":"success","data":{"source":"binance","data_mode":"real","symbol":"BTCUSDT"}}`
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	recorder := httptest.NewRecorder()
	cryptoSpotMarketHandler("http://quant-python:8000", client)(recorder, httptest.NewRequest(http.MethodGet, "/api/crypto/spot/market", nil))
	if !strings.Contains(recorder.Body.String(), `"source":"binance"`) {
		t.Fatalf("unexpected payload %s", recorder.Body.String())
	}
}
