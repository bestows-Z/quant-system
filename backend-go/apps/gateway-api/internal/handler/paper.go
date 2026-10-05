package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"quant-system/backend-go/apps/gateway-api/internal/svc"
	"quant-system/backend-go/apps/gateway-api/internal/types"
	"quant-system/backend-go/apps/paper-rpc/paperclient"
	gwauth "quant-system/backend-go/common/auth"
)

func CreatePaperAccountHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		var req types.CreatePaperAccountRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, 10001, "请求参数错误")
			return
		}
		reply, err := svcCtx.Paper.CreateAccount(r.Context(), &paperclient.CreateAccountReq{
			UserId:      userID,
			AccountName: req.AccountName,
			InitialCash: req.InitialCash.String(),
		})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Account)
	}
}

func ListPaperAccountsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Paper.ListAccounts(r.Context(), &paperclient.ListAccountsReq{UserId: userID})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Items)
	}
}

func paperPositionPayload(item *paperclient.Position) map[string]any {
	if item == nil {
		return map[string]any{}
	}
	return map[string]any{
		"id": item.GetId(), "account_id": item.GetAccountId(), "symbol": item.GetSymbol(), "name": item.GetName(),
		"volume": item.GetVolume(), "available_volume": item.GetAvailableVolume(), "frozen_volume": item.GetFrozenVolume(),
		"today_buy_volume": item.GetTodayBuyVolume(), "last_buy_date": item.GetLastBuyDate(), "avg_cost": item.GetAvgCost(),
		"market_price": item.GetMarketPrice(), "market_value": item.GetMarketValue(), "profit": item.GetProfit(), "profit_rate": item.GetProfitRate(),
	}
}

func paperOrderPayload(item *paperclient.Order) map[string]any {
	if item == nil {
		return map[string]any{}
	}
	return map[string]any{
		"id": item.GetId(), "order_no": item.GetOrderNo(), "client_order_id": item.GetClientOrderId(), "account_id": item.GetAccountId(), "user_id": item.GetUserId(),
		"symbol": item.GetSymbol(), "name": item.GetName(), "side": item.GetSide(), "order_type": item.GetOrderType(), "price": item.GetPrice(), "volume": item.GetVolume(),
		"filled_volume": item.GetFilledVolume(), "remaining_volume": item.GetRemainingVolume(), "avg_filled_price": item.GetAvgFilledPrice(), "status": item.GetStatus(),
		"amount": item.GetAmount(), "fee": item.GetFee(), "remark": item.GetRemark(), "created_at": item.GetCreatedAt(), "expire_at": item.GetExpireAt(), "quote_time": item.GetQuoteTime(), "data_mode": item.GetDataMode(), "disclaimer": item.GetDisclaimer(),
	}
}

func paperTradePayload(item *paperclient.Trade) map[string]any {
	if item == nil {
		return map[string]any{}
	}
	return map[string]any{"id": item.GetId(), "trade_no": item.GetTradeNo(), "order_id": item.GetOrderId(), "order_no": item.GetOrderNo(), "account_id": item.GetAccountId(), "user_id": item.GetUserId(), "symbol": item.GetSymbol(), "name": item.GetName(), "side": item.GetSide(), "price": item.GetPrice(), "volume": item.GetVolume(), "amount": item.GetAmount(), "fee": item.GetFee(), "stamp_duty": item.GetStampDuty(), "slippage": item.GetSlippage(), "trade_time": item.GetTradeTime(), "depth_level": item.GetDepthLevel(), "quote_time": item.GetQuoteTime(), "data_mode": item.GetDataMode()}
}

func GetPaperAccountHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, accountID, ok := userAndAccountID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Paper.GetAccount(r.Context(), &paperclient.GetAccountReq{UserId: userID, AccountId: accountID})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Account)
	}
}

func GetPaperAssetsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, accountID, ok := userAndAccountID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Paper.GetAccountAssets(r.Context(), &paperclient.GetAccountAssetsReq{UserId: userID, AccountId: accountID})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Account)
	}
}

func ListPaperPositionsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, accountID, ok := userAndAccountID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Paper.ListPositions(r.Context(), &paperclient.ListPositionsReq{UserId: userID, AccountId: accountID})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		items := make([]map[string]any, 0, len(reply.Items))
		for _, item := range reply.Items {
			items = append(items, paperPositionPayload(item))
		}
		writeSuccess(w, r, items)
	}
}

func CreatePaperOrderHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		var req types.CreatePaperOrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, 10001, "请求参数错误")
			return
		}
		reply, err := svcCtx.Paper.CreateOrder(r.Context(), &paperclient.CreateOrderReq{
			UserId:        userID,
			AccountId:     req.AccountID,
			ClientOrderId: req.ClientOrderID,
			Symbol:        req.Symbol,
			Name:          req.Name,
			Side:          req.Side,
			OrderType:     req.OrderType,
			Price:         req.Price.String(),
			Volume:        req.Volume,
		})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, paperOrderPayload(reply.Order))
	}
}

func CancelPaperOrderHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, orderID, ok := userAndOrderID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Paper.CancelOrder(r.Context(), &paperclient.CancelOrderReq{UserId: userID, OrderId: orderID})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, paperOrderPayload(reply.Order))
	}
}

func ListPaperOrdersHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		query := r.URL.Query()
		accountID, _ := strconv.ParseUint(query.Get("account_id"), 10, 64)
		reply, err := svcCtx.Paper.ListOrders(r.Context(), &paperclient.ListOrdersReq{
			UserId:    userID,
			AccountId: accountID,
			Symbol:    query.Get("symbol"),
			Status:    query.Get("status"),
		})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		items := make([]map[string]any, 0, len(reply.Items))
		for _, item := range reply.Items {
			items = append(items, paperOrderPayload(item))
		}
		writeSuccess(w, r, items)
	}
}

func GetPaperOrderHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, orderID, ok := userAndOrderID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Paper.GetOrder(r.Context(), &paperclient.GetOrderReq{UserId: userID, OrderId: orderID})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, paperOrderPayload(reply.Order))
	}
}

func ListPaperTradesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		query := r.URL.Query()
		accountID, _ := strconv.ParseUint(query.Get("account_id"), 10, 64)
		reply, err := svcCtx.Paper.ListTrades(r.Context(), &paperclient.ListTradesReq{
			UserId:    userID,
			AccountId: accountID,
			Symbol:    query.Get("symbol"),
		})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		items := make([]map[string]any, 0, len(reply.Items))
		for _, item := range reply.Items {
			items = append(items, paperTradePayload(item))
		}
		writeSuccess(w, r, items)
	}
}

func GetPaperEquityCurveHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		accountID, err := strconv.ParseUint(r.URL.Query().Get("account_id"), 10, 64)
		if err != nil || accountID == 0 {
			writeError(w, r, 10001, "account_id 参数错误")
			return
		}
		reply, err := svcCtx.Paper.GetEquityCurve(r.Context(), &paperclient.GetEquityCurveReq{UserId: userID, AccountId: accountID})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Items)
	}
}

func currentUserID(w http.ResponseWriter, r *http.Request) (uint64, bool) {
	claims, ok := gwauth.ClaimsFromContext(r.Context())
	if !ok || claims.UserID == 0 {
		writeError(w, r, 10002, "invalid token")
		return 0, false
	}
	return claims.UserID, true
}

func userAndAccountID(w http.ResponseWriter, r *http.Request) (uint64, uint64, bool) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return 0, 0, false
	}
	accountID, err := strconv.ParseUint(pathValue(r, "id"), 10, 64)
	if err != nil || accountID == 0 {
		writeError(w, r, 10001, "id 参数错误")
		return 0, 0, false
	}
	return userID, accountID, true
}

func userAndOrderID(w http.ResponseWriter, r *http.Request) (uint64, uint64, bool) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return 0, 0, false
	}
	orderID, err := strconv.ParseUint(pathValue(r, "id"), 10, 64)
	if err != nil || orderID == 0 {
		writeError(w, r, 10001, "id 参数错误")
		return 0, 0, false
	}
	return userID, orderID, true
}
