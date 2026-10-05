package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"quant-system/backend-go/apps/gateway-api/internal/svc"

	_ "github.com/go-sql-driver/mysql"
)

type systemStatus struct {
	Name          string `json:"name"`
	Status        string `json:"status"`
	LatencyMS     int64  `json:"latency_ms"`
	Message       string `json:"message,omitempty"`
	CheckedAt     string `json:"checked_at"`
	Capability    string `json:"capability,omitempty"`
	LastSuccessAt string `json:"last_success_at,omitempty"`
	LastError     string `json:"last_error,omitempty"`
	Enabled       bool   `json:"enabled"`
}

type namedProbe struct {
	name string
	run  func(context.Context) error
}

func SystemHealthHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		probes := []namedProbe{
			{"gateway-api", func(context.Context) error { return nil }},
			{"mysql", mysqlProbe(svcCtx.Config.MySQL.DSN)},
			{"redis", tcpProbe(svcCtx.Config.CacheRedis.Host)},
			{"etcd", httpProbe("http://etcd:2379/health")},
			{"rabbitmq", tcpProbe(amqpAddress(svcCtx.Config.RabbitMQ.URL))},
			{"qdrant", httpProbe(strings.TrimRight(svcCtx.Config.Qdrant.Addr, "/") + "/healthz")},
			{"quant-python", httpProbe(strings.TrimRight(svcCtx.Config.QuantPython.BaseURL, "/") + "/health")},
			{"user-rpc", tcpProbe("user-rpc:9001")},
			{"market-rpc", tcpProbe("market-rpc:9002")},
			{"strategy-rpc", tcpProbe("strategy-rpc:9003")},
			{"backtest-rpc", tcpProbe("backtest-rpc:9004")},
			{"paper-rpc", tcpProbe("paper-rpc:9005")},
			{"risk-rpc", tcpProbe("risk-rpc:9006")},
			{"rag-rpc", tcpProbe("rag-rpc:9008")},
			{"agent-rpc", tcpProbe("agent-rpc:9009")},
			{"task-rpc", tcpProbe("task-rpc:9010")},
			{"notify-rpc", tcpProbe("notify-rpc:9011")},
			{"portfolio-rpc", tcpProbe("portfolio-rpc:9012")},
			{"model-gateway-rpc", tcpProbe("model-gateway-rpc:9013")},
			{"feature-rpc", tcpProbe("feature-rpc:9014")},
			{"auto-trader-rpc", tcpProbe("auto-trader-rpc:9016")},
			{"ws-api", tcpProbe("ws-api:8081")},
		}
		writeSuccess(w, r, runProbes(r.Context(), probes))
	}
}

func DataSourceStatusHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		base := strings.TrimRight(svcCtx.Config.QuantPython.BaseURL, "/")
		realtime := sourceProbe(r.Context(), base+"/api/market/realtime/000001", "realtime")
		history := sourceProbe(r.Context(), base+"/api/kline/000001?period=daily&start_date=2024-01-01&end_date=2024-01-31", "daily_kline")
		statuses := map[string]systemStatus{}
		for _, item := range []struct{ name, capability string }{
			{"tencent", "realtime"}, {"sina", "realtime"}, {"efinance", "stocks,realtime,kline"},
			{"baostock", "daily_kline"}, {"eastmoney", "stocks,realtime,kline"},
			{"akshare", "stocks,realtime,kline"}, {"local_csv", "stocks,daily_kline"},
		} {
			statuses[item.name] = systemStatus{Name: item.name, Capability: item.capability, Status: "UNKNOWN", Message: "本次探测未选中该 Provider", CheckedAt: time.Now().Format(time.RFC3339), Enabled: true}
		}
		for _, observed := range []systemStatus{realtime, history} {
			if observed.Name != "" {
				current := statuses[observed.Name]
				observed.Capability = current.Capability
				observed.Enabled = true
				statuses[observed.Name] = observed
			}
		}
		order := []string{"tencent", "sina", "efinance", "baostock", "eastmoney", "akshare", "local_csv"}
		items := make([]systemStatus, 0, len(order))
		for _, name := range order {
			items = append(items, statuses[name])
		}
		writeSuccess(w, r, items)
	}
}

func runProbes(parent context.Context, probes []namedProbe) []systemStatus {
	items := make([]systemStatus, len(probes))
	var wg sync.WaitGroup
	for index, probe := range probes {
		wg.Add(1)
		go func(index int, probe namedProbe) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(parent, 2*time.Second)
			defer cancel()
			started := time.Now()
			err := probe.run(ctx)
			item := systemStatus{Name: probe.name, Status: "UP", LatencyMS: time.Since(started).Milliseconds(), Message: "连接正常", CheckedAt: time.Now().Format(time.RFC3339), Enabled: true}
			if err != nil {
				item.Status = "DOWN"
				item.Message = err.Error()
				item.LastError = err.Error()
			} else {
				item.LastSuccessAt = item.CheckedAt
			}
			items[index] = item
		}(index, probe)
	}
	wg.Wait()
	return items
}

func sourceProbe(parent context.Context, endpoint, capability string) systemStatus {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	started := time.Now()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return systemStatus{Name: "unavailable", Status: "DOWN", LatencyMS: time.Since(started).Milliseconds(), Message: capability + " 探测失败", LastError: err.Error(), CheckedAt: time.Now().Format(time.RFC3339), Enabled: true}
	}
	defer response.Body.Close()
	var payload struct {
		Code int `json:"code"`
		Data struct {
			Source string `json:"source"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil || payload.Code != 0 || payload.Data.Source == "" {
		return systemStatus{Name: "unavailable", Status: "DOWN", LatencyMS: time.Since(started).Milliseconds(), Message: capability + " 返回不可用", LastError: fmt.Sprint(err), CheckedAt: time.Now().Format(time.RFC3339), Enabled: true}
	}
	now := time.Now().Format(time.RFC3339)
	return systemStatus{Name: payload.Data.Source, Status: "UP", LatencyMS: time.Since(started).Milliseconds(), Message: capability + " 实际响应成功", CheckedAt: now, LastSuccessAt: now, Enabled: true}
}

func mysqlProbe(dsn string) func(context.Context) error {
	return func(ctx context.Context) error {
		db, err := sql.Open("mysql", os.ExpandEnv(dsn))
		if err != nil {
			return err
		}
		defer db.Close()
		return db.PingContext(ctx)
	}
}

func tcpProbe(address string) func(context.Context) error {
	return func(ctx context.Context) error {
		if strings.TrimSpace(address) == "" {
			return fmt.Errorf("地址未配置")
		}
		dialer := net.Dialer{}
		conn, err := dialer.DialContext(ctx, "tcp", address)
		if err == nil {
			_ = conn.Close()
		}
		return err
	}
}

func httpProbe(endpoint string) func(context.Context) error {
	return func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return err
		}
		response, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer response.Body.Close()
		if response.StatusCode < 200 || response.StatusCode >= 400 {
			return fmt.Errorf("HTTP %d", response.StatusCode)
		}
		return nil
	}
}

func amqpAddress(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return "rabbitmq:5672"
	}
	if strings.Contains(parsed.Host, ":") {
		return parsed.Host
	}
	return parsed.Host + ":5672"
}
