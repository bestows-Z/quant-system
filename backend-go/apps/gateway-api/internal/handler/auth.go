package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"quant-system/backend-go/apps/gateway-api/internal/svc"
	"quant-system/backend-go/apps/gateway-api/internal/types"
	"quant-system/backend-go/apps/user-rpc/userclient"
	gwauth "quant-system/backend-go/common/auth"
	"quant-system/backend-go/common/response"
	"quant-system/backend-go/common/trace"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RegisterHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, 10001, "请求参数错误")
			return
		}
		reply, err := svcCtx.User.Register(r.Context(), &userclient.RegisterRequest{
			Username: req.Username,
			Email:    req.Email,
			Password: req.Password,
			Nickname: req.Nickname,
		})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, authPayload(reply))
	}
}

func LoginHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, 10001, "请求参数错误")
			return
		}
		reply, err := svcCtx.User.Login(r.Context(), &userclient.LoginRequest{
			Username: req.Username,
			Password: req.Password,
		})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, authPayload(reply))
	}
}

func ProfileHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := gwauth.ClaimsFromContext(r.Context())
		if !ok || claims.UserID == 0 {
			writeError(w, r, 10002, "invalid token")
			return
		}
		reply, err := svcCtx.User.Profile(r.Context(), &userclient.ProfileRequest{UserId: claims.UserID})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, userPayload(reply.User))
	}
}

func writeSuccess(w http.ResponseWriter, r *http.Request, data any) {
	response.JSON(w, http.StatusOK, response.SuccessWithTrace(data, trace.FromContext(r.Context())))
}

func writeError(w http.ResponseWriter, r *http.Request, code int, message string) {
	response.JSON(w, http.StatusOK, response.Error(code, message, trace.FromContext(r.Context())))
}

func writeErrorWithData(w http.ResponseWriter, r *http.Request, code int, message string, data any) {
	response.JSON(w, http.StatusOK, response.ErrorWithData(code, message, data, trace.FromContext(r.Context())))
}

func writeRPCError(w http.ResponseWriter, r *http.Request, err error) {
	code := 10099
	message := "internal server error"
	st, ok := status.FromError(err)
	if ok {
		message = st.Message()
		switch st.Code() {
		case codes.InvalidArgument:
			code = 10001
		case codes.Unauthenticated:
			code = 10002
		case codes.PermissionDenied:
			code = 10003
		case codes.NotFound:
			code = 10004
		case codes.AlreadyExists:
			code = 10006
		case codes.Unavailable:
			code = 503
		default:
			code = 10099
		}
	} else if !errors.Is(err, nil) {
		message = err.Error()
	}
	writeError(w, r, code, message)
}

func authPayload(reply *userclient.AuthReply) map[string]any {
	return map[string]any{
		"token":     reply.Token,
		"expire_at": reply.ExpireAt,
		"user":      userPayload(reply.User),
	}
}

func userPayload(info *userclient.UserInfo) map[string]any {
	if info == nil {
		return map[string]any{}
	}
	return map[string]any{
		"id":         info.Id,
		"username":   info.Username,
		"email":      info.Email,
		"nickname":   info.Nickname,
		"status":     info.Status,
		"roles":      info.Roles,
		"created_at": info.CreatedAt,
	}
}
