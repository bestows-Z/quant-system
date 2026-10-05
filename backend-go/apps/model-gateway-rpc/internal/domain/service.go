package domain

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"net"
	"net/http"
	"net/url"
	"quant-system/backend-go/apps/model-gateway-rpc/modelgateway"
	"quant-system/backend-go/common/trace"
	"strings"
	"time"
)

type Cipher struct{ aead cipher.AEAD }

func NewCipher(secret string) (*Cipher, error) {
	if len(strings.TrimSpace(secret)) < 16 {
		return nil, errors.New("MODEL_ENCRYPTION_KEY must contain at least 16 characters")
	}
	sum := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Cipher{aead: aead}, nil
}
func (c *Cipher) Encrypt(value string) (string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := c.aead.Seal(nil, nonce, []byte(value), nil)
	return base64.RawStdEncoding.EncodeToString(append(nonce, sealed...)), nil
}
func (c *Cipher) Decrypt(value string) (string, error) {
	raw, err := base64.RawStdEncoding.DecodeString(value)
	if err != nil {
		return "", err
	}
	n := c.aead.NonceSize()
	if len(raw) < n {
		return "", errors.New("invalid encrypted value")
	}
	plain, err := c.aead.Open(nil, raw[:n], raw[n:], nil)
	return string(plain), err
}

type Service struct {
	db     *sql.DB
	crypto *Cipher
	http   *http.Client
}
type configRow struct {
	ID, UserID                                   uint64
	Provider, Name, BaseURL, Model, EncryptedKey string
	Temperature                                  float64
	Timeout, MaxTokens, ContextWindow            uint32
	Enabled, Default                             bool
	TestStatus, TestMessage                      string
	TestAt                                       sql.NullTime
	Created, Updated                             time.Time
}

func NewService(db *sql.DB, crypto *Cipher) *Service {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = safeDialContext
	return &Service{db: db, crypto: crypto, http: &http.Client{
		Transport: transport,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return errors.New("模型接口不允许 HTTP 重定向")
		},
	}}
}

func (s *Service) Providers(ctx context.Context) ([]*modelgateway.Provider, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT provider_code,display_name,default_base_url,api_style,enabled FROM model_providers ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*modelgateway.Provider, 0)
	for rows.Next() {
		p := &modelgateway.Provider{}
		if err = rows.Scan(&p.Code, &p.DisplayName, &p.DefaultBaseUrl, &p.ApiStyle, &p.Enabled); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
func (s *Service) Save(ctx context.Context, in *modelgateway.SaveConfigReq) (*modelgateway.ModelConfig, error) {
	if in.UserId == 0 || strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.ProviderCode) == "" || strings.TrimSpace(in.Model) == "" {
		return nil, status.Error(codes.InvalidArgument, "名称、Provider 和模型不能为空")
	}
	base := strings.TrimRight(strings.TrimSpace(in.BaseUrl), "/")
	if base == "" {
		_ = s.db.QueryRowContext(ctx, `SELECT default_base_url FROM model_providers WHERE provider_code=? AND enabled=1`, in.ProviderCode).Scan(&base)
	}
	if !strings.HasPrefix(base, "https://") && !strings.HasPrefix(base, "http://") {
		return nil, status.Error(codes.InvalidArgument, "Base URL 必须使用 http 或 https")
	}
	if err := validateBaseURL(base); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	timeout := in.TimeoutSeconds
	if timeout == 0 {
		timeout = 60
	}
	maxTokens := in.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}
	contextWindow := in.ContextWindowTokens
	if contextWindow == 0 {
		contextWindow = defaultContextWindow(in.ProviderCode)
	}
	temp := in.Temperature
	if temp < 0 || temp > 2 {
		return nil, status.Error(codes.InvalidArgument, "temperature 必须在 0 到 2 之间")
	}
	encrypted := ""
	if strings.TrimSpace(in.ApiKey) != "" {
		var err error
		encrypted, err = s.crypto.Encrypt(strings.TrimSpace(in.ApiKey))
		if err != nil {
			return nil, err
		}
	}
	if in.Id == 0 {
		if encrypted == "" {
			return nil, status.Error(codes.InvalidArgument, "API Key 不能为空")
		}
		res, err := s.db.ExecContext(ctx, `INSERT INTO model_configs(user_id,provider_code,name,base_url,model,encrypted_api_key,temperature,timeout_seconds,max_tokens,context_window_tokens,enabled) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, in.UserId, in.ProviderCode, strings.TrimSpace(in.Name), base, in.Model, encrypted, temp, timeout, maxTokens, contextWindow, in.Enabled)
		if err != nil {
			return nil, mapDuplicate(err)
		}
		id, _ := res.LastInsertId()
		return s.get(ctx, in.UserId, uint64(id))
	}
	existing, err := s.row(ctx, in.UserId, in.Id)
	if err != nil {
		return nil, err
	}
	if encrypted == "" {
		encrypted = existing.EncryptedKey
	}
	_, err = s.db.ExecContext(ctx, `UPDATE model_configs SET provider_code=?,name=?,base_url=?,model=?,encrypted_api_key=?,temperature=?,timeout_seconds=?,max_tokens=?,context_window_tokens=?,enabled=?,last_test_status='UNTESTED',last_test_message='' WHERE id=? AND user_id=? AND deleted_at IS NULL`, in.ProviderCode, strings.TrimSpace(in.Name), base, in.Model, encrypted, temp, timeout, maxTokens, contextWindow, in.Enabled, in.Id, in.UserId)
	if err != nil {
		return nil, mapDuplicate(err)
	}
	return s.get(ctx, in.UserId, in.Id)
}
func (s *Service) List(ctx context.Context, userID uint64) ([]*modelgateway.ModelConfig, error) {
	rows, err := s.db.QueryContext(ctx, selectConfig+` WHERE user_id=? AND deleted_at IS NULL ORDER BY is_default DESC,id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*modelgateway.ModelConfig, 0)
	for rows.Next() {
		r, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, toPB(r))
	}
	return out, rows.Err()
}
func (s *Service) Delete(ctx context.Context, userID, id uint64) error {
	res, err := s.db.ExecContext(ctx, `UPDATE model_configs SET deleted_at=CURRENT_TIMESTAMP(3),enabled=0,is_default=0 WHERE id=? AND user_id=? AND deleted_at IS NULL`, id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return status.Error(codes.NotFound, "模型配置不存在")
	}
	return nil
}
func (s *Service) Activate(ctx context.Context, userID, id uint64) (*modelgateway.ModelConfig, error) {
	r, err := s.row(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if !r.Enabled {
		return nil, status.Error(codes.FailedPrecondition, "模型配置未启用")
	}
	if r.TestStatus != "SUCCESS" {
		return nil, status.Error(codes.FailedPrecondition, "请先通过连接测试")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE model_configs SET is_default=0 WHERE user_id=? AND deleted_at IS NULL`, userID); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE model_configs SET is_default=1 WHERE id=? AND user_id=?`, id, userID); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return s.get(ctx, userID, id)
}
func (s *Service) Test(ctx context.Context, userID, id uint64) (*modelgateway.TestConfigResp, error) {
	r, err := s.row(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	started := time.Now()
	_, promptTokens, completionTokens, err := s.call(ctx, r, "你是连接测试助手。", "只回复 OK。", "")
	latency := time.Since(started).Milliseconds()
	state, msg := "SUCCESS", "连接正常"
	if err != nil {
		state = "FAILED"
		msg = truncate(err.Error(), 500)
	}
	_, _ = s.db.ExecContext(ctx, `UPDATE model_configs SET last_test_status=?,last_test_message=?,last_test_at=CURRENT_TIMESTAMP(3) WHERE id=? AND user_id=?`, state, msg, id, userID)
	promptTokens, completionTokens, tokenSource := normalizedUsage(promptTokens, completionTokens, "你是连接测试助手。", "只回复 OK。")
	_, _ = s.db.ExecContext(ctx, `INSERT INTO model_usage_records(user_id,config_id,provider_code,model,request_type,prompt_tokens,completion_tokens,total_tokens,token_source,latency_ms,status,error_message,trace_id) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`, userID, r.ID, r.Provider, r.Model, "CONNECTION_TEST", promptTokens, completionTokens, promptTokens+completionTokens, tokenSource, latency, state, msgIfFailed(state, msg), trace.FromContext(ctx))
	if err != nil {
		return &modelgateway.TestConfigResp{Success: false, Message: msg, LatencyMs: latency}, nil
	}
	return &modelgateway.TestConfigResp{Success: true, Message: msg, LatencyMs: latency}, nil
}
func (s *Service) Generate(ctx context.Context, in *modelgateway.GenerateReq) (*modelgateway.GenerateResp, error) {
	r, err := s.defaultRow(ctx, in.UserId)
	if err != nil {
		return nil, err
	}
	started := time.Now()
	content, promptTokens, completionTokens, callErr := s.call(ctx, r, in.SystemPrompt, in.Prompt, in.ResponseFormat)
	promptTokens, completionTokens, tokenSource := normalizedUsage(promptTokens, completionTokens, in.SystemPrompt, in.Prompt)
	latency := time.Since(started).Milliseconds()
	state, errMsg := "SUCCESS", ""
	if callErr != nil {
		state = "FAILED"
		errMsg = truncate(callErr.Error(), 500)
	}
	usageResult, usageErr := s.db.ExecContext(ctx, `INSERT INTO model_usage_records(user_id,config_id,provider_code,model,request_type,prompt_tokens,completion_tokens,total_tokens,token_source,latency_ms,status,error_message,trace_id) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`, in.UserId, r.ID, r.Provider, r.Model, defaultText(in.RequestType, "AGENT"), promptTokens, completionTokens, promptTokens+completionTokens, tokenSource, latency, state, errMsg, trace.FromContext(ctx))
	if callErr != nil {
		return nil, status.Error(codes.Unavailable, "模型调用失败："+errMsg)
	}
	if usageErr != nil {
		return nil, status.Error(codes.Internal, "模型调用记录保存失败")
	}
	usageID, err := usageResult.LastInsertId()
	if err != nil {
		return nil, status.Error(codes.Internal, "模型调用记录编号获取失败")
	}
	return &modelgateway.GenerateResp{Content: content, ProviderCode: r.Provider, Model: r.Model, LatencyMs: latency, PromptTokens: promptTokens, CompletionTokens: completionTokens, TotalTokens: promptTokens + completionTokens, ConfigId: r.ID, TokenSource: tokenSource, UsageRecordId: uint64(usageID)}, nil
}
func (s *Service) Usage(ctx context.Context, userID uint64, days uint32) (*modelgateway.UsageSummary, error) {
	if days == 0 {
		days = 30
	}
	var out modelgateway.UsageSummary
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*),COALESCE(SUM(status='SUCCESS'),0),COALESCE(SUM(status='FAILED'),0),COALESCE(SUM(prompt_tokens),0),COALESCE(SUM(completion_tokens),0),COALESCE(SUM(total_tokens),0),CAST(COALESCE(AVG(latency_ms),0) AS UNSIGNED) FROM model_usage_records WHERE user_id=? AND created_at>=DATE_SUB(NOW(),INTERVAL ? DAY)`, userID, days).Scan(&out.Requests, &out.SuccessRequests, &out.FailedRequests, &out.PromptTokens, &out.CompletionTokens, &out.TotalTokens, &out.AverageLatencyMs)
	if err != nil {
		return nil, err
	}
	if out.Requests > 0 {
		out.SuccessRate = float64(out.SuccessRequests) / float64(out.Requests)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT latency_ms FROM model_usage_records WHERE user_id=? AND created_at>=DATE_SUB(NOW(),INTERVAL ? DAY) ORDER BY latency_ms`, userID, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var latencies []int64
	for rows.Next() {
		var latency int64
		if err := rows.Scan(&latency); err != nil {
			return nil, err
		}
		latencies = append(latencies, latency)
	}
	if len(latencies) > 0 {
		out.P95LatencyMs = latencies[(len(latencies)*95+99)/100-1]
	}
	return &out, rows.Err()
}

func (s *Service) UsageRecords(ctx context.Context, in *modelgateway.ListUsageRecordsReq) ([]*modelgateway.UsageRecord, uint64, error) {
	days, limit, offset := in.GetDays(), in.GetLimit(), in.GetOffset()
	if days == 0 {
		days = 30
	}
	if limit == 0 || limit > 200 {
		limit = 50
	}
	where, args := `user_id=? AND created_at>=DATE_SUB(NOW(),INTERVAL ? DAY)`, []any{in.GetUserId(), days}
	if strings.TrimSpace(in.GetRequestType()) != "" {
		where += ` AND request_type=?`
		args = append(args, strings.TrimSpace(in.GetRequestType()))
	}
	var total uint64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM model_usage_records WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := s.db.QueryContext(ctx, `SELECT id,COALESCE(config_id,0),provider_code,model,request_type,prompt_tokens,completion_tokens,total_tokens,token_source,latency_ms,first_token_latency_ms,status,error_message,trace_id,COALESCE(report_id,0),created_at FROM model_usage_records WHERE `+where+` ORDER BY created_at DESC,id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]*modelgateway.UsageRecord, 0)
	for rows.Next() {
		item := &modelgateway.UsageRecord{}
		var created time.Time
		if err := rows.Scan(&item.Id, &item.ConfigId, &item.ProviderCode, &item.Model, &item.RequestType, &item.PromptTokens, &item.CompletionTokens, &item.TotalTokens, &item.TokenSource, &item.LatencyMs, &item.FirstTokenLatencyMs, &item.Status, &item.ErrorMessage, &item.TraceId, &item.ReportId, &created); err != nil {
			return nil, 0, err
		}
		item.CreatedAt = created.Format("2006-01-02 15:04:05")
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (s *Service) call(ctx context.Context, r configRow, systemPrompt, prompt, responseFormat string) (string, uint32, uint32, error) {
	key, err := s.crypto.Decrypt(r.EncryptedKey)
	if err != nil {
		return "", 0, 0, err
	}
	body := map[string]any{"model": r.Model, "temperature": r.Temperature, "max_tokens": r.MaxTokens, "messages": []map[string]string{{"role": "system", "content": systemPrompt}, {"role": "user", "content": prompt}}}
	if responseFormat == "json_object" {
		body["response_format"] = map[string]string{"type": "json_object"}
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(r.BaseURL, "/")+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return "", 0, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	client := *s.http
	client.Timeout = time.Duration(r.Timeout) * time.Second
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", 0, 0, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}
	var payload struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			Prompt     uint32 `json:"prompt_tokens"`
			Completion uint32 `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&payload); err != nil || len(payload.Choices) == 0 {
		return "", 0, 0, errors.New("模型响应格式无效")
	}
	return payload.Choices[0].Message.Content, payload.Usage.Prompt, payload.Usage.Completion, nil
}

const selectConfig = `SELECT id,user_id,provider_code,name,base_url,model,encrypted_api_key,temperature,timeout_seconds,max_tokens,context_window_tokens,enabled,is_default,last_test_status,last_test_message,last_test_at,created_at,updated_at FROM model_configs`

type scanner interface{ Scan(...any) error }

func scanRow(s scanner) (configRow, error) {
	var r configRow
	err := s.Scan(&r.ID, &r.UserID, &r.Provider, &r.Name, &r.BaseURL, &r.Model, &r.EncryptedKey, &r.Temperature, &r.Timeout, &r.MaxTokens, &r.ContextWindow, &r.Enabled, &r.Default, &r.TestStatus, &r.TestMessage, &r.TestAt, &r.Created, &r.Updated)
	return r, err
}
func (s *Service) row(ctx context.Context, userID, id uint64) (configRow, error) {
	r, err := scanRow(s.db.QueryRowContext(ctx, selectConfig+` WHERE id=? AND user_id=? AND deleted_at IS NULL`, id, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return r, status.Error(codes.NotFound, "模型配置不存在")
	}
	return r, err
}
func (s *Service) defaultRow(ctx context.Context, userID uint64) (configRow, error) {
	r, err := scanRow(s.db.QueryRowContext(ctx, selectConfig+` WHERE user_id=? AND is_default=1 AND enabled=1 AND deleted_at IS NULL LIMIT 1`, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return r, status.Error(codes.FailedPrecondition, "尚未配置并启用 AI 模型")
	}
	return r, err
}
func (s *Service) get(ctx context.Context, userID, id uint64) (*modelgateway.ModelConfig, error) {
	r, err := s.row(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	return toPB(r), nil
}
func toPB(r configRow) *modelgateway.ModelConfig {
	testAt := ""
	if r.TestAt.Valid {
		testAt = r.TestAt.Time.Format(time.RFC3339)
	}
	return &modelgateway.ModelConfig{Id: r.ID, ProviderCode: r.Provider, Name: r.Name, BaseUrl: r.BaseURL, Model: r.Model, ApiKeyMasked: maskKey(r.EncryptedKey), Temperature: r.Temperature, TimeoutSeconds: r.Timeout, MaxTokens: r.MaxTokens, ContextWindowTokens: r.ContextWindow, Enabled: r.Enabled, IsDefault: r.Default, LastTestStatus: r.TestStatus, LastTestMessage: r.TestMessage, LastTestAt: testAt, CreatedAt: r.Created.Format(time.RFC3339), UpdatedAt: r.Updated.Format(time.RFC3339)}
}

func normalizedUsage(promptTokens, completionTokens uint32, systemPrompt, prompt string) (uint32, uint32, string) {
	if promptTokens > 0 || completionTokens > 0 {
		return promptTokens, completionTokens, "PROVIDER"
	}
	return estimateTokens(systemPrompt + "\n" + prompt), 0, "ESTIMATED"
}

func estimateTokens(value string) uint32 {
	runes := []rune(value)
	if len(runes) == 0 {
		return 0
	}
	return uint32((len(runes) + 2) / 3)
}

func defaultContextWindow(provider string) uint32 {
	if strings.EqualFold(provider, "deepseek") {
		return 1000000
	}
	return 128000
}

func msgIfFailed(state, message string) string {
	if state == "FAILED" {
		return message
	}
	return ""
}
func maskKey(v string) string {
	if v == "" {
		return ""
	}
	return "********"
}
func mapDuplicate(err error) error {
	if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
		return status.Error(codes.AlreadyExists, "同名模型配置已存在")
	}
	return err
}
func truncate(v string, n int) string {
	if len(v) > n {
		return v[:n]
	}
	return v
}
func defaultText(v, d string) string {
	if strings.TrimSpace(v) == "" {
		return d
	}
	return v
}

func validateBaseURL(value string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Hostname() == "" || parsed.User != nil {
		return errors.New("Base URL 格式无效")
	}
	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if host == "localhost" || host == "metadata.google.internal" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") {
		return errors.New("Base URL 不允许访问本机、私网或云元数据地址")
	}
	if ip := net.ParseIP(host); ip != nil && !isPublicIP(ip) {
		return errors.New("Base URL 不允许访问本机、私网或云元数据地址")
	}
	return nil
}

func safeDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	addresses, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return nil, err
	}
	for _, ip := range addresses {
		if !isPublicIP(ip) {
			return nil, errors.New("模型接口解析到非公网地址，连接已阻止")
		}
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	return dialer.DialContext(ctx, network, net.JoinHostPort(addresses[0].String(), port))
}

func isPublicIP(ip net.IP) bool {
	return ip != nil && !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsUnspecified() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() && !ip.IsMulticast()
}
