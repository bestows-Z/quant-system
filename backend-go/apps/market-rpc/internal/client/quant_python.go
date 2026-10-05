package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"quant-system/backend-go/common/xhttp"
)

type QuantPythonClient struct {
	http *xhttp.Client
}

func NewQuantPythonClient(baseURL string, timeout time.Duration) *QuantPythonClient {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &QuantPythonClient{http: xhttp.New(strings.TrimRight(baseURL, "/"), timeout)}
}

func (c *QuantPythonClient) Stocks(ctx context.Context, keyword string, page, pageSize int32) (string, error) {
	values := url.Values{}
	if keyword != "" {
		values.Set("keyword", keyword)
	}
	setInt(values, "page", page)
	setInt(values, "page_size", pageSize)
	return c.getRaw(ctx, withQuery("/api/stocks", values))
}

func (c *QuantPythonClient) SearchStocks(ctx context.Context, keyword string, page, pageSize int32) (string, error) {
	values := url.Values{}
	if keyword != "" {
		values.Set("keyword", keyword)
	}
	setInt(values, "page", page)
	setInt(values, "page_size", pageSize)
	return c.getRaw(ctx, withQuery("/api/stocks/search", values))
}

func (c *QuantPythonClient) Realtime(ctx context.Context, symbols []string, limit int32) (string, error) {
	values := url.Values{}
	if len(symbols) > 0 {
		values.Set("symbols", strings.Join(symbols, ","))
	}
	setInt(values, "limit", limit)
	return c.getRaw(ctx, withQuery("/api/market/realtime", values))
}

func (c *QuantPythonClient) RealtimeBySymbol(ctx context.Context, symbol string) (string, error) {
	return c.getRaw(ctx, "/api/market/realtime/"+url.PathEscape(symbol))
}

func (c *QuantPythonClient) Timeshare(ctx context.Context, symbol string) (string, error) {
	return c.getRaw(ctx, "/api/market/timeshare/"+url.PathEscape(symbol))
}

func (c *QuantPythonClient) OrderBook(ctx context.Context, symbol string) (string, error) {
	return c.getRaw(ctx, "/api/market/orderbook/"+url.PathEscape(symbol))
}

func (c *QuantPythonClient) Kline(ctx context.Context, symbol, period, startDate, endDate, adjust string) (string, error) {
	values := url.Values{}
	if period != "" {
		values.Set("period", period)
	}
	if startDate != "" {
		values.Set("start_date", startDate)
	}
	if endDate != "" {
		values.Set("end_date", endDate)
	}
	if adjust != "" {
		values.Set("adjust", adjust)
	}
	return c.getRaw(ctx, withQuery("/api/kline/"+url.PathEscape(symbol), values))
}

func (c *QuantPythonClient) Indicator(ctx context.Context, symbol, indicatorType, period, startDate, endDate, adjust string) (string, error) {
	values := url.Values{}
	if indicatorType != "" {
		values.Set("type", indicatorType)
	}
	if period != "" {
		values.Set("period", period)
	}
	if startDate != "" {
		values.Set("start_date", startDate)
	}
	if endDate != "" {
		values.Set("end_date", endDate)
	}
	if adjust != "" {
		values.Set("adjust", adjust)
	}
	return c.getRaw(ctx, withQuery("/api/indicator/"+url.PathEscape(symbol), values))
}

func (c *QuantPythonClient) FullIndicator(ctx context.Context, symbol, period, startDate, endDate, adjust string) (string, error) {
	values := url.Values{}
	if period != "" {
		values.Set("period", period)
	}
	if startDate != "" {
		values.Set("start_date", startDate)
	}
	if endDate != "" {
		values.Set("end_date", endDate)
	}
	if adjust != "" {
		values.Set("adjust", adjust)
	}
	return c.getRaw(ctx, withQuery("/api/indicator/full/"+url.PathEscape(symbol), values))
}

func (c *QuantPythonClient) SupportResistance(ctx context.Context, symbol, startDate, endDate string) (string, error) {
	values := url.Values{}
	if startDate != "" {
		values.Set("start_date", startDate)
	}
	if endDate != "" {
		values.Set("end_date", endDate)
	}
	return c.getRaw(ctx, withQuery("/api/analysis/support-resistance/"+url.PathEscape(symbol), values))
}

func (c *QuantPythonClient) MarketIndices(ctx context.Context) (string, error) {
	return c.getRaw(ctx, "/api/research/indices")
}

func (c *QuantPythonClient) MarketBreadth(ctx context.Context) (string, error) {
	return c.getRaw(ctx, "/api/research/market-breadth")
}

func (c *QuantPythonClient) CompanyProfile(ctx context.Context, symbol string) (string, error) {
	return c.getRaw(ctx, "/api/research/company/"+url.PathEscape(symbol))
}

func (c *QuantPythonClient) Announcements(ctx context.Context, symbol string, limit int32) (string, error) {
	values := url.Values{}
	setInt(values, "limit", limit)
	return c.getRaw(ctx, withQuery("/api/research/announcements/"+url.PathEscape(symbol), values))
}

func (c *QuantPythonClient) News(ctx context.Context, symbol, keyword string, limit int32) (string, error) {
	values := url.Values{}
	if keyword != "" {
		values.Set("keyword", keyword)
	}
	setInt(values, "limit", limit)
	return c.getRaw(ctx, withQuery("/api/research/news/"+url.PathEscape(symbol), values))
}

func (c *QuantPythonClient) MoneyFlow(ctx context.Context, symbol string, limit int32) (string, error) {
	values := url.Values{}
	setInt(values, "limit", limit)
	return c.getRaw(ctx, withQuery("/api/research/money-flow/"+url.PathEscape(symbol), values))
}

func (c *QuantPythonClient) DragonTiger(ctx context.Context, tradeDate string, limit int32) (string, error) {
	values := url.Values{}
	if tradeDate != "" {
		values.Set("trade_date", tradeDate)
	}
	setInt(values, "limit", limit)
	return c.getRaw(ctx, withQuery("/api/research/dragon-tiger", values))
}

func (c *QuantPythonClient) DragonTigerSeats(ctx context.Context, symbol, tradeDate string) (string, error) {
	values := url.Values{}
	values.Set("trade_date", tradeDate)
	return c.getRaw(ctx, withQuery("/api/research/dragon-tiger/"+url.PathEscape(symbol)+"/seats", values))
}

func (c *QuantPythonClient) getRaw(ctx context.Context, path string) (string, error) {
	var raw any
	if err := c.http.Get(ctx, path, &raw); err != nil {
		return "", err
	}
	return marshalRaw(raw)
}

func withQuery(path string, values url.Values) string {
	if encoded := values.Encode(); encoded != "" {
		return path + "?" + encoded
	}
	return path
}

func setInt(values url.Values, key string, value int32) {
	if value > 0 {
		values.Set(key, fmt.Sprintf("%d", value))
	}
}

func responseError(raw any) (int, string) {
	body, ok := raw.(map[string]any)
	if !ok {
		return 0, ""
	}
	codeValue, ok := body["code"]
	if !ok {
		return 0, ""
	}
	var code int
	switch value := codeValue.(type) {
	case float64:
		code = int(value)
	case int:
		code = value
	case json.Number:
		parsed, _ := value.Int64()
		code = int(parsed)
	default:
		return 0, ""
	}
	if code == 0 {
		return 0, ""
	}
	message, _ := body["message"].(string)
	if message == "" {
		message = "source_unavailable: quant-python 返回行情错误"
	}
	return code, message
}
