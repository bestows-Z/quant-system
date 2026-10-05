package middleware

import (
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"quant-system/backend-go/common/auth"
	qjwt "quant-system/backend-go/common/jwt"
	"quant-system/backend-go/common/response"
	"quant-system/backend-go/common/trace"
)

type Middleware func(http.Handler) http.Handler

func Chain(handler http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get("X-Trace-ID")
		if traceID == "" {
			traceID = trace.NewTraceID()
		}
		w.Header().Set("X-Trace-ID", traceID)
		next.ServeHTTP(w, r.WithContext(trace.WithTraceID(r.Context(), traceID)))
	})
}

func AccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("method=%s path=%s latency=%s ip=%s ua=%q trace_id=%s",
			r.Method, r.URL.Path, time.Since(start), r.RemoteAddr, r.UserAgent(), trace.FromContext(r.Context()))
	})
}

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic=%v stack=%s trace_id=%s", rec, debug.Stack(), trace.FromContext(r.Context()))
				response.JSON(w, http.StatusOK, response.Error(10099, "internal server error", trace.FromContext(r.Context())))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Trace-ID, Idempotency-Key")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func JWTAuth(secret string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			tokenText := strings.TrimPrefix(header, "Bearer ")
			if tokenText == "" {
				response.JSON(w, http.StatusOK, response.Error(10002, "missing token", trace.FromContext(r.Context())))
				return
			}
			claims, err := qjwt.Parse(secret, tokenText)
			if err != nil {
				response.JSON(w, http.StatusOK, response.Error(10002, "invalid token", trace.FromContext(r.Context())))
				return
			}
			next.ServeHTTP(w, r.WithContext(auth.WithClaims(r.Context(), claims)))
		})
	}
}

func RBAC(_ ...string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
}

func RateLimit(_ string, _ int) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
}

func Idempotency() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
}
