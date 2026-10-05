package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"quant-system/backend-go/apps/gateway-api/internal/svc"
)

var cryptoSpotPairs = map[string]bool{"BTCUSDT": true, "ETHUSDT": true, "BNBUSDT": true}
var cryptoSpotIntervals = map[string]bool{"1h": true, "4h": true, "1d": true}

func CryptoSpotMarketHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return cryptoSpotMarketHandler(svcCtx.Config.QuantPython.BaseURL, &http.Client{Timeout: 12 * time.Second})
}

func cryptoSpotMarketHandler(baseURL string, client *http.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		symbol := r.URL.Query().Get("symbol")
		interval := r.URL.Query().Get("interval")
		if symbol == "" {
			symbol = "BTCUSDT"
		}
		if interval == "" {
			interval = "1h"
		}
		if !cryptoSpotPairs[symbol] || !cryptoSpotIntervals[interval] {
			writeError(w, r, 422, "仅支持 BTC/USDT、ETH/USDT、BNB/USDT 的 1h、4h、1d K 线")
			return
		}
		endpoint, err := url.Parse(strings.TrimRight(baseURL, "/") + "/api/crypto/spot/market")
		if err != nil || endpoint.Scheme != "http" && endpoint.Scheme != "https" {
			writeError(w, r, 503, "数字资产行情服务配置无效")
			return
		}
		query := endpoint.Query()
		query.Set("symbol", symbol)
		query.Set("interval", interval)
		endpoint.RawQuery = query.Encode()
		ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
		defer cancel()
		upstream, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
		if err != nil {
			writeError(w, r, 503, "数字资产行情请求失败")
			return
		}
		response, err := client.Do(upstream)
		if err != nil {
			writeError(w, r, 503, "数字资产行情服务暂不可用，请稍后重试")
			return
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			writeError(w, r, 503, "数字资产行情服务暂不可用，请稍后重试")
			return
		}
		var payload struct {
			Code    int             `json:"code"`
			Message string          `json:"message"`
			Data    json.RawMessage `json:"data"`
		}
		if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&payload); err != nil {
			writeError(w, r, 503, "数字资产行情响应无效")
			return
		}
		if payload.Code != 0 {
			if payload.Message == "" {
				payload.Message = "数字资产行情服务暂不可用"
			}
			writeError(w, r, payload.Code, payload.Message)
			return
		}
		if len(payload.Data) == 0 || string(payload.Data) == "null" {
			writeError(w, r, 503, "数字资产行情响应缺失")
			return
		}
		writeSuccess(w, r, payload.Data)
	}
}
