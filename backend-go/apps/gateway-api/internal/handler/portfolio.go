package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"quant-system/backend-go/apps/gateway-api/internal/svc"
	"quant-system/backend-go/apps/gateway-api/internal/types"
	"quant-system/backend-go/apps/portfolio-rpc/portfolioservice"
)

const maxPositionCSVSize = 2 << 20

func CreatePortfolioHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		var req types.CreatePortfolioRequest
		if json.NewDecoder(r.Body).Decode(&req) != nil {
			writeError(w, r, 10001, "请求参数错误")
			return
		}
		reply, err := svcCtx.Portfolio.CreatePortfolio(r.Context(), &portfolioservice.CreatePortfolioReq{UserId: userID, Name: req.Name, Description: req.Description})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Portfolio)
	}
}

func ListPortfoliosHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Portfolio.ListPortfolios(r.Context(), &portfolioservice.ListPortfoliosReq{UserId: userID})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Items)
	}
}

func GetPortfolioHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := userAndPortfolioID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Portfolio.GetPortfolio(r.Context(), &portfolioservice.GetPortfolioReq{UserId: userID, PortfolioId: id})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Portfolio)
	}
}

func DeletePortfolioHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := userAndPortfolioID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Portfolio.DeletePortfolio(r.Context(), &portfolioservice.DeletePortfolioReq{UserId: userID, PortfolioId: id})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply)
	}
}

func ListRealPositionsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := userAndPortfolioID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Portfolio.ListPositions(r.Context(), &portfolioservice.ListPositionsReq{UserId: userID, PortfolioId: id})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Items)
	}
}

func UpsertRealPositionHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, portfolioID, ok := userAndPortfolioID(w, r)
		if !ok {
			return
		}
		var req types.UpsertRealPositionRequest
		if json.NewDecoder(r.Body).Decode(&req) != nil {
			writeError(w, r, 10001, "请求参数错误")
			return
		}
		positionID := uint64(0)
		if raw := pathValue(r, "position_id"); raw != "" {
			positionID, _ = strconv.ParseUint(raw, 10, 64)
			if positionID == 0 {
				writeError(w, r, 10001, "position_id 参数错误")
				return
			}
		}
		reply, err := svcCtx.Portfolio.UpsertPosition(r.Context(), &portfolioservice.UpsertPositionReq{UserId: userID, PortfolioId: portfolioID, PositionId: positionID, Symbol: req.Symbol, Name: req.Name, Quantity: req.Quantity, AvailableQuantity: req.AvailableQuantity, AvgCost: req.AvgCost.String(), OpenedOn: req.OpenedOn, Note: req.Note, Version: req.Version})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Position)
	}
}

func DeleteRealPositionHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, portfolioID, ok := userAndPortfolioID(w, r)
		if !ok {
			return
		}
		positionID, err := strconv.ParseUint(pathValue(r, "position_id"), 10, 64)
		if err != nil || positionID == 0 {
			writeError(w, r, 10001, "position_id 参数错误")
			return
		}
		reply, err := svcCtx.Portfolio.DeletePosition(r.Context(), &portfolioservice.DeletePositionReq{UserId: userID, PortfolioId: portfolioID, PositionId: positionID})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply)
	}
}

func PreviewPositionImportHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, portfolioID, ok := userAndPortfolioID(w, r)
		if !ok {
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxPositionCSVSize+64*1024)
		if err := r.ParseMultipartForm(maxPositionCSVSize); err != nil {
			writeError(w, r, 10001, "CSV 文件过大或表单格式错误")
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			writeError(w, r, 10001, "请选择 CSV 文件")
			return
		}
		defer file.Close()
		if !strings.HasSuffix(strings.ToLower(header.Filename), ".csv") {
			writeError(w, r, 10001, "仅支持 CSV 文件")
			return
		}
		content, err := io.ReadAll(io.LimitReader(file, maxPositionCSVSize+1))
		if err != nil || len(content) > maxPositionCSVSize {
			writeError(w, r, 10001, "CSV 文件读取失败或超过 2MB")
			return
		}
		key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
		if key == "" {
			key = strings.TrimSpace(r.FormValue("idempotency_key"))
		}
		reply, err := svcCtx.Portfolio.PreviewImport(r.Context(), &portfolioservice.PreviewImportReq{UserId: userID, PortfolioId: portfolioID, FileName: header.Filename, CsvContent: content, IdempotencyKey: key})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply)
	}
}

func ConfirmPositionImportHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, portfolioID, ok := userAndPortfolioID(w, r)
		if !ok {
			return
		}
		var req types.ConfirmPositionImportRequest
		if json.NewDecoder(r.Body).Decode(&req) != nil || req.TaskID == 0 {
			writeError(w, r, 10001, "task_id 参数错误")
			return
		}
		reply, err := svcCtx.Portfolio.ConfirmImport(r.Context(), &portfolioservice.ConfirmImportReq{UserId: userID, PortfolioId: portfolioID, TaskId: req.TaskID})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply)
	}
}

func AnalyzePortfolioHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := userAndPortfolioID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Portfolio.AnalyzePortfolio(r.Context(), &portfolioservice.AnalyzePortfolioReq{UserId: userID, PortfolioId: id, Refresh: r.Method == http.MethodPost})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Analysis)
	}
}

func userAndPortfolioID(w http.ResponseWriter, r *http.Request) (uint64, uint64, bool) {
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
