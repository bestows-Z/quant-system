package handler

import (
	"encoding/json"
	"net/http"
	"quant-system/backend-go/apps/gateway-api/internal/svc"
	"quant-system/backend-go/apps/gateway-api/internal/types"
	"quant-system/backend-go/apps/model-gateway-rpc/modelgatewayservice"
	"strconv"
)

func ListModelProvidersHandler(s *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reply, err := s.ModelGateway.ListProviders(r.Context(), &modelgatewayservice.ListProvidersReq{})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Items)
	}
}
func ListModelConfigsHandler(s *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid, ok := currentUserID(w, r)
		if !ok {
			return
		}
		reply, err := s.ModelGateway.ListConfigs(r.Context(), &modelgatewayservice.ListConfigsReq{UserId: uid})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Items)
	}
}
func SaveModelConfigHandler(s *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid, ok := currentUserID(w, r)
		if !ok {
			return
		}
		var req types.SaveModelConfigRequest
		if json.NewDecoder(r.Body).Decode(&req) != nil {
			writeError(w, r, 10001, "请求参数错误")
			return
		}
		reply, err := s.ModelGateway.SaveConfig(r.Context(), &modelgatewayservice.SaveConfigReq{UserId: uid, Id: req.ID, ProviderCode: req.ProviderCode, Name: req.Name, BaseUrl: req.BaseURL, Model: req.Model, ApiKey: req.APIKey, Temperature: req.Temperature, TimeoutSeconds: req.TimeoutSeconds, MaxTokens: req.MaxTokens, ContextWindowTokens: req.ContextWindowTokens, Enabled: req.Enabled})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Config)
	}
}

func ModelUsageRecordsHandler(s *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid, ok := currentUserID(w, r)
		if !ok {
			return
		}
		days, _ := strconv.ParseUint(r.URL.Query().Get("days"), 10, 32)
		limit, _ := strconv.ParseUint(r.URL.Query().Get("limit"), 10, 32)
		offset, _ := strconv.ParseUint(r.URL.Query().Get("offset"), 10, 32)
		reply, err := s.ModelGateway.ListUsageRecords(r.Context(), &modelgatewayservice.ListUsageRecordsReq{UserId: uid, Days: uint32(days), RequestType: r.URL.Query().Get("request_type"), Limit: uint32(limit), Offset: uint32(offset)})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, map[string]any{"items": reply.Items, "total": reply.Total})
	}
}
func DeleteModelConfigHandler(s *svc.ServiceContext) http.HandlerFunc {
	return modelConfigAction(s, "delete")
}
func ActivateModelConfigHandler(s *svc.ServiceContext) http.HandlerFunc {
	return modelConfigAction(s, "activate")
}
func TestModelConfigHandler(s *svc.ServiceContext) http.HandlerFunc {
	return modelConfigAction(s, "test")
}
func modelConfigAction(s *svc.ServiceContext, action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid, ok := currentUserID(w, r)
		if !ok {
			return
		}
		id, err := strconv.ParseUint(pathValue(r, "id"), 10, 64)
		if err != nil || id == 0 {
			writeError(w, r, 10001, "id 参数错误")
			return
		}
		req := &modelgatewayservice.ConfigReq{UserId: uid, Id: id}
		switch action {
		case "delete":
			reply, e := s.ModelGateway.DeleteConfig(r.Context(), req)
			err = e
			if err == nil {
				writeSuccess(w, r, reply)
			}
		case "activate":
			reply, e := s.ModelGateway.ActivateConfig(r.Context(), req)
			err = e
			if err == nil {
				writeSuccess(w, r, reply.Config)
			}
		case "test":
			reply, e := s.ModelGateway.TestConfig(r.Context(), req)
			err = e
			if err == nil {
				writeSuccess(w, r, reply)
			}
		}
		if err != nil {
			writeRPCError(w, r, err)
		}
	}
}
func ModelUsageHandler(s *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid, ok := currentUserID(w, r)
		if !ok {
			return
		}
		days, _ := strconv.ParseUint(r.URL.Query().Get("days"), 10, 32)
		reply, err := s.ModelGateway.Usage(r.Context(), &modelgatewayservice.UsageReq{UserId: uid, Days: uint32(days)})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply)
	}
}
