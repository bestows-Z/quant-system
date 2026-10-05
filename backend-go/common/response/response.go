package response

import (
	"encoding/json"
	"net/http"
	"reflect"
)

type Body struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
	TraceID string `json:"trace_id,omitempty"`
}

func Success(data any) Body {
	if data == nil {
		data = map[string]any{}
	} else {
		value := reflect.ValueOf(data)
		if value.Kind() == reflect.Slice && value.IsNil() {
			data = []any{}
		}
	}
	return Body{Code: 0, Message: "success", Data: data}
}

func SuccessWithTrace(data any, traceID string) Body {
	body := Success(data)
	body.TraceID = traceID
	return body
}

func Error(code int, message, traceID string) Body {
	return Body{Code: code, Message: message, Data: nil, TraceID: traceID}
}

func ErrorWithData(code int, message string, data any, traceID string) Body {
	if data == nil {
		data = map[string]any{}
	}
	return Body{Code: code, Message: message, Data: data, TraceID: traceID}
}

func JSON(w http.ResponseWriter, status int, body Body) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
