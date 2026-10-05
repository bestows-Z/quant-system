package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"quant-system/backend-go/apps/backtest-rpc/backtestclient"
	"quant-system/backend-go/apps/gateway-api/internal/svc"
	"quant-system/backend-go/apps/gateway-api/internal/types"
	"quant-system/backend-go/apps/strategy-rpc/strategyclient"
)

func ListStrategiesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Strategy.ListStrategies(r.Context(), &strategyclient.ListStrategiesReq{UserId: userID})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Items)
	}
}
func GetStrategyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := userAndPathID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Strategy.GetStrategy(r.Context(), &strategyclient.GetStrategyReq{UserId: userID, Id: id})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Item)
	}
}

func backtestResultPayload(item *backtestclient.BacktestResult) map[string]any {
	if item == nil {
		return map[string]any{}
	}
	return map[string]any{
		"id": item.GetId(), "task_id": item.GetTaskId(), "symbol": item.GetSymbol(),
		"strategy_code": item.GetStrategyCode(), "total_return": item.GetTotalReturn(),
		"annual_return": item.GetAnnualReturn(), "max_drawdown": item.GetMaxDrawdown(),
		"win_rate": item.GetWinRate(), "profit_loss_ratio": item.GetProfitLossRatio(),
		"total_trades": item.GetTotalTrades(), "winning_trades": item.GetWinningTrades(),
		"losing_trades": item.GetLosingTrades(), "final_asset": item.GetFinalAsset(),
		"raw_result_json": item.GetRawResultJson(), "disclaimer": item.GetDisclaimer(),
		"data_source": item.GetDataSource(), "data_mode": item.GetDataMode(), "params_json": item.GetParamsJson(),
		"engine_version": item.GetEngineVersion(), "execution_model": item.GetExecutionModel(),
		"adjust": item.GetAdjust(), "quality_status": item.GetQualityStatus(),
		"config_json": item.GetConfigJson(), "quality_json": item.GetQualityJson(),
		"realized_pnl": item.GetRealizedPnl(), "unrealized_pnl": item.GetUnrealizedPnl(),
		"total_fees": item.GetTotalFees(), "turnover_rate": item.GetTurnoverRate(),
		"exposure_rate": item.GetExposureRate(),
	}
}
func CreateStrategyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		var req types.CreateStrategyRequest
		if json.NewDecoder(r.Body).Decode(&req) != nil {
			writeError(w, r, 10001, "请求参数错误")
			return
		}
		reply, err := svcCtx.Strategy.CreateStrategy(r.Context(), &strategyclient.CreateStrategyReq{UserId: userID, Name: req.Name, Code: req.Code, Description: req.Description, ParamsJson: req.ParamsJSON})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Item)
	}
}
func UpdateStrategyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := userAndPathID(w, r)
		if !ok {
			return
		}
		var req types.UpdateStrategyRequest
		if json.NewDecoder(r.Body).Decode(&req) != nil {
			writeError(w, r, 10001, "请求参数错误")
			return
		}
		reply, err := svcCtx.Strategy.UpdateStrategy(r.Context(), &strategyclient.UpdateStrategyReq{UserId: userID, Id: id, Name: req.Name, Description: req.Description, ParamsJson: req.ParamsJSON, Status: req.Status})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Item)
	}
}
func DeleteStrategyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := userAndPathID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Strategy.DeleteStrategies(r.Context(), &strategyclient.DeleteStrategiesReq{UserId: userID, Ids: []uint64{id}})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		if reply.GetDeletedCount() == 0 {
			writeError(w, r, 10004, "只能删除自己创建且尚未删除的策略")
			return
		}
		writeSuccess(w, r, map[string]any{"deleted_count": reply.GetDeletedCount()})
	}
}
func RunStrategyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := userAndPathID(w, r)
		if !ok {
			return
		}
		var req types.RunStrategyRequest
		if json.NewDecoder(r.Body).Decode(&req) != nil {
			writeError(w, r, 10001, "请求参数错误")
			return
		}
		reply, err := svcCtx.Strategy.RunStrategy(r.Context(), &strategyclient.RunStrategyReq{UserId: userID, StrategyId: id, StrategyCode: req.StrategyCode, Symbol: req.Symbol, ParamsJson: req.ParamsJSON})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply)
	}
}
func GetSignalsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
		reply, err := svcCtx.Strategy.GetSignals(r.Context(), &strategyclient.GetSignalsReq{UserId: userID, StrategyCode: r.URL.Query().Get("strategy_code"), SignalType: r.URL.Query().Get("signal_type"), Limit: limit})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Items)
	}
}
func GetSignalsBySymbolHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
		reply, err := svcCtx.Strategy.GetSignalsBySymbol(r.Context(), &strategyclient.GetSignalsBySymbolReq{UserId: userID, Symbol: pathValue(r, "symbol"), Limit: limit})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Items)
	}
}
func GetLatestSignalHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Strategy.GetLatestSignal(r.Context(), &strategyclient.GetLatestSignalReq{UserId: userID, Symbol: pathValue(r, "symbol")})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Item)
	}
}

func RunBacktestHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		var req types.RunBacktestRequest
		if json.NewDecoder(r.Body).Decode(&req) != nil {
			writeError(w, r, 10001, "请求参数错误")
			return
		}
		reply, err := svcCtx.Backtest.RunBacktest(r.Context(), &backtestclient.RunBacktestReq{
			UserId: userID, Symbol: req.Symbol, StrategyCode: req.StrategyCode, StartDate: req.StartDate, EndDate: req.EndDate,
			InitialCash: req.InitialCash.String(), FeeRate: req.FeeRate.String(), SlippageRate: req.SlippageRate.String(),
			ParamsJson: req.ParamsJSON, Adjust: req.Adjust, MinCommission: req.MinCommission.String(),
			StampDutyRate: req.StampDutyRate.String(), TransferFeeRate: req.TransferFeeRate.String(),
			MaxVolumeRatio: req.MaxVolumeRatio.String(), TargetPositionRatio: req.TargetPositionRatio.String(),
			LotSize: req.LotSize, TickSize: req.TickSize.String(),
		})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply)
	}
}
func ListBacktestTasksHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
		reply, err := svcCtx.Backtest.ListBacktestTasks(r.Context(), &backtestclient.ListBacktestTasksReq{UserId: userID, Status: r.URL.Query().Get("status"), Limit: limit})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Items)
	}
}
func GetBacktestTaskHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := userAndPathID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Backtest.GetBacktestTask(r.Context(), &backtestclient.GetBacktestTaskReq{UserId: userID, TaskId: id})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Item)
	}
}
func GetBacktestResultHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := userAndPathID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Backtest.GetBacktestResult(r.Context(), &backtestclient.GetBacktestResultReq{UserId: userID, TaskId: id})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, backtestResultPayload(reply.Item))
	}
}
func GetBacktestTradesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := userAndPathID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Backtest.GetBacktestTrades(r.Context(), &backtestclient.GetBacktestTradesReq{UserId: userID, TaskId: id})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Items)
	}
}
func GetBacktestOrdersHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := userAndPathID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Backtest.GetBacktestOrders(r.Context(), &backtestclient.GetBacktestOrdersReq{UserId: userID, TaskId: id})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Items)
	}
}
func GetBacktestEquityCurveHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := userAndPathID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Backtest.GetBacktestEquityCurve(r.Context(), &backtestclient.GetBacktestEquityCurveReq{UserId: userID, TaskId: id})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Items)
	}
}

func DeleteBacktestHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := userAndPathID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Backtest.DeleteBacktest(r.Context(), &backtestclient.DeleteBacktestReq{UserId: userID, TaskId: id})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, map[string]any{"affected": reply.GetAffected(), "already_deleted": reply.GetAlreadyDeleted()})
	}
}

func DeleteBacktestsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		var req struct {
			TaskIDs []uint64 `json:"task_ids"`
			Scope   string   `json:"scope"`
		}
		if r.ContentLength > 0 && json.NewDecoder(r.Body).Decode(&req) != nil {
			writeError(w, r, 10001, "请求参数错误")
			return
		}
		if req.Scope == "" {
			req.Scope = r.URL.Query().Get("scope")
		}
		reply, err := svcCtx.Backtest.DeleteBacktests(r.Context(), &backtestclient.DeleteBacktestsReq{UserId: userID, TaskIds: req.TaskIDs, Scope: req.Scope})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, map[string]any{"affected": reply.GetAffected(), "already_deleted": reply.GetAlreadyDeleted()})
	}
}
