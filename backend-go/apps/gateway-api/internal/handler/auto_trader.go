package handler

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"quant-system/backend-go/apps/auto-trader-rpc/autotraderservice"
	"quant-system/backend-go/apps/gateway-api/internal/svc"
)

type createAutoAccountRequest struct {
	PaperAccountID        uint64  `json:"paper_account_id"`
	Name                  string  `json:"name"`
	MaxPositionRatio      float64 `json:"max_position_ratio"`
	MaxDailyLossRatio     float64 `json:"max_daily_loss_ratio"`
	MinSignalScore        float64 `json:"min_signal_score"`
	MaxSymbols            int32   `json:"max_symbols"`
	MaxTotalPositionRatio float64 `json:"max_total_position_ratio"`
}
type generateAutoStrategyRequest struct {
	Objective string `json:"objective"`
}

func autoAccountIDs(w http.ResponseWriter, r *http.Request) (uint64, uint64, bool) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return 0, 0, false
	}
	id, err := strconv.ParseUint(pathValue(r, "id"), 10, 64)
	if err != nil || id == 0 {
		writeError(w, r, 10001, "id 参数错误")
		return 0, 0, false
	}
	return userID, id, true
}
func writeAutoRaw(w http.ResponseWriter, r *http.Request, reply *autotraderservice.RawReply, err error) {
	if err != nil {
		writeRPCError(w, r, err)
		return
	}
	var payload map[string]any
	if json.Unmarshal([]byte(reply.GetJson()), &payload) != nil {
		writeError(w, r, 10099, "自动交易响应解析失败")
		return
	}
	if code, ok := numericCode(payload["code"]); ok && code != 0 {
		message, _ := payload["message"].(string)
		writeErrorWithData(w, r, code, message, payload["data"])
		return
	}
	writeSuccess(w, r, payload["data"])
}

func writeAutoItems(w http.ResponseWriter, r *http.Request, reply *autotraderservice.RawReply, err error) {
	if err != nil {
		writeRPCError(w, r, err)
		return
	}
	var payload map[string]any
	if json.Unmarshal([]byte(reply.GetJson()), &payload) != nil {
		writeError(w, r, 10099, "自动交易响应解析失败")
		return
	}
	data, _ := payload["data"].(map[string]any)
	items := data["items"]
	if items == nil {
		items = []any{}
	}
	writeSuccess(w, r, items)
}

func deliveryReq(r *http.Request, userID, accountID uint64, limit int32) *autotraderservice.DeliveryStatementReq {
	return &autotraderservice.DeliveryStatementReq{
		UserId: userID, AccountId: accountID, StartDate: r.URL.Query().Get("start_date"),
		EndDate: r.URL.Query().Get("end_date"), Symbol: r.URL.Query().Get("symbol"), Limit: limit,
	}
}

func autoRawItems(reply *autotraderservice.RawReply) ([]map[string]any, error) {
	var payload struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Items []map[string]any `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(reply.GetJson()), &payload); err != nil {
		return nil, err
	}
	if payload.Code != 0 {
		return nil, fmt.Errorf("%s", payload.Message)
	}
	return payload.Data.Items, nil
}

func csvText(value any) string {
	text := fmt.Sprint(value)
	trimmed := strings.TrimLeft(text, " \t\r\n")
	if trimmed != "" && strings.ContainsRune("=+-@", rune(trimmed[0])) {
		return "'" + text
	}
	return text
}
func ListAutoTraderAccountsHandler(s *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		reply, err := s.AutoTrader.ListAccounts(r.Context(), &autotraderservice.UserReq{UserId: userID})
		writeAutoItems(w, r, reply, err)
	}
}
func CreateAutoTraderAccountHandler(s *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		var req createAutoAccountRequest
		if json.NewDecoder(r.Body).Decode(&req) != nil {
			writeError(w, r, 10001, "请求参数错误")
			return
		}
		reply, err := s.AutoTrader.CreateAccount(r.Context(), &autotraderservice.CreateAccountReq{UserId: userID, PaperAccountId: req.PaperAccountID, Name: req.Name, MaxPositionRatio: req.MaxPositionRatio, MaxTotalPositionRatio: req.MaxTotalPositionRatio, MaxDailyLossRatio: req.MaxDailyLossRatio, MinSignalScore: req.MinSignalScore, MaxSymbols: req.MaxSymbols})
		writeAutoRaw(w, r, reply, err)
	}
}
func ListArchivedAutoTraderAccountsHandler(s *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		reply, err := s.AutoTrader.ListArchivedAccounts(r.Context(), &autotraderservice.UserReq{UserId: userID})
		writeAutoItems(w, r, reply, err)
	}
}
func UpdateAutoTraderAccountHandler(s *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := autoAccountIDs(w, r)
		if !ok {
			return
		}
		var req createAutoAccountRequest
		if json.NewDecoder(r.Body).Decode(&req) != nil {
			writeError(w, r, 10001, "请求参数错误")
			return
		}
		reply, err := s.AutoTrader.UpdateAccount(r.Context(), &autotraderservice.UpdateAccountReq{UserId: userID, AccountId: id, Name: req.Name, MaxPositionRatio: req.MaxPositionRatio, MaxTotalPositionRatio: req.MaxTotalPositionRatio, MaxDailyLossRatio: req.MaxDailyLossRatio, MinSignalScore: req.MinSignalScore, MaxSymbols: req.MaxSymbols})
		writeAutoRaw(w, r, reply, err)
	}
}
func ArchiveAutoTraderAccountHandler(s *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := autoAccountIDs(w, r)
		if !ok {
			return
		}
		reply, err := s.AutoTrader.ArchiveAccount(r.Context(), &autotraderservice.AccountReq{UserId: userID, AccountId: id})
		writeAutoRaw(w, r, reply, err)
	}
}
func EvaluateAutoStrategyHandler(s *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := autoAccountIDs(w, r)
		if !ok {
			return
		}
		reply, err := s.AutoTrader.EvaluateStrategy(r.Context(), &autotraderservice.StrategyReq{UserId: userID, StrategyId: id})
		writeAutoRaw(w, r, reply, err)
	}
}
func ListAutoStrategiesHandler(s *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := autoAccountIDs(w, r)
		if !ok {
			return
		}
		reply, err := s.AutoTrader.ListStrategies(r.Context(), &autotraderservice.AccountReq{UserId: userID, AccountId: id})
		writeAutoItems(w, r, reply, err)
	}
}
func GenerateAutoStrategyHandler(s *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := autoAccountIDs(w, r)
		if !ok {
			return
		}
		var req generateAutoStrategyRequest
		if json.NewDecoder(r.Body).Decode(&req) != nil || strings.TrimSpace(req.Objective) == "" {
			writeError(w, r, 10001, "请填写 AI 生成研究目标，或选择一个目标模板")
			return
		}
		reply, err := s.AutoTrader.GenerateStrategy(r.Context(), &autotraderservice.GenerateStrategyReq{UserId: userID, AccountId: id, Objective: req.Objective})
		writeAutoRaw(w, r, reply, err)
	}
}
func GetAutoStrategyHandler(s *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := autoAccountIDs(w, r)
		if !ok {
			return
		}
		reply, err := s.AutoTrader.GetStrategy(r.Context(), &autotraderservice.StrategyReq{UserId: userID, StrategyId: id})
		writeAutoRaw(w, r, reply, err)
	}
}
func PromoteAutoStrategyHandler(s *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := autoAccountIDs(w, r)
		if !ok {
			return
		}
		reply, err := s.AutoTrader.PromoteStrategy(r.Context(), &autotraderservice.StrategyReq{UserId: userID, StrategyId: id})
		writeAutoRaw(w, r, reply, err)
	}
}
func StartAutoTraderHandler(s *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := autoAccountIDs(w, r)
		if !ok {
			return
		}
		reply, err := s.AutoTrader.Start(r.Context(), &autotraderservice.AccountReq{UserId: userID, AccountId: id})
		writeAutoRaw(w, r, reply, err)
	}
}
func PauseAutoTraderHandler(s *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := autoAccountIDs(w, r)
		if !ok {
			return
		}
		reply, err := s.AutoTrader.Pause(r.Context(), &autotraderservice.AccountReq{UserId: userID, AccountId: id})
		writeAutoRaw(w, r, reply, err)
	}
}
func ListAutoTradeRunsHandler(s *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := autoAccountIDs(w, r)
		if !ok {
			return
		}
		reply, err := s.AutoTrader.ListRuns(r.Context(), &autotraderservice.AccountReq{UserId: userID, AccountId: id})
		writeAutoItems(w, r, reply, err)
	}
}
func ListAutoTradeDecisionsHandler(s *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := autoAccountIDs(w, r)
		if !ok {
			return
		}
		reply, err := s.AutoTrader.ListDecisions(r.Context(), &autotraderservice.AccountReq{UserId: userID, AccountId: id})
		writeAutoItems(w, r, reply, err)
	}
}

func ListAutoTradeDeliveriesHandler(s *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := autoAccountIDs(w, r)
		if !ok {
			return
		}
		reply, err := s.AutoTrader.ListDeliveryStatements(r.Context(), deliveryReq(r, userID, id, 500))
		writeAutoItems(w, r, reply, err)
	}
}

func ExportAutoTradeDeliveriesHandler(s *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := autoAccountIDs(w, r)
		if !ok {
			return
		}
		reply, err := s.AutoTrader.ListDeliveryStatements(r.Context(), deliveryReq(r, userID, id, 5000))
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		items, err := autoRawItems(reply)
		if err != nil {
			writeError(w, r, 10099, "交割单响应解析失败")
			return
		}
		filename := fmt.Sprintf("auto-trader-%d-deliveries-%s.csv", id, time.Now().Format("20060102-150405"))
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
		_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})
		writer := csv.NewWriter(w)
		_ = writer.Write([]string{"成交时间", "成交编号", "委托编号", "股票代码", "股票名称", "方向", "委托类型", "委托价", "成交价", "成交数量", "成交金额", "手续费", "印花税", "滑点", "盘口档位", "行情时间", "数据模式", "策略版本", "策略名称", "决策依据", "追踪编号"})
		for _, item := range items {
			side := map[string]string{"BUY": "买入", "SELL": "卖出"}[csvText(item["side"])]
			if side == "" {
				side = csvText(item["side"])
			}
			orderType := map[string]string{"MARKET": "市价", "LIMIT": "限价"}[csvText(item["order_type"])]
			if orderType == "" {
				orderType = csvText(item["order_type"])
			}
			_ = writer.Write([]string{csvText(item["trade_time"]), csvText(item["trade_no"]), csvText(item["order_no"]), csvText(item["symbol"]), csvText(item["name"]), side, orderType, csvText(item["order_price"]), csvText(item["fill_price"]), csvText(item["quantity"]), csvText(item["amount"]), csvText(item["fee"]), csvText(item["stamp_duty"]), csvText(item["slippage"]), csvText(item["depth_level"]), csvText(item["quote_time"]), csvText(item["data_mode"]), "v" + csvText(item["strategy_version"]), csvText(item["strategy_name"]), csvText(item["decision_reason"]), csvText(item["trace_id"])})
		}
		writer.Flush()
	}
}
