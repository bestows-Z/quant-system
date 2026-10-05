package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"quant-system/backend-go/apps/gateway-api/internal/svc"
	"quant-system/backend-go/apps/notify-rpc/notifyclient"
	"quant-system/backend-go/apps/task-rpc/taskclient"
)

type createTaskReq struct {
	TaskType    string `json:"task_type"`
	SourceType  string `json:"source_type"`
	SourceID    string `json:"source_id"`
	Title       string `json:"title"`
	PayloadJSON string `json:"payload_json"`
}

type updateTaskStatusReq struct {
	Status       string `json:"status"`
	Progress     int64  `json:"progress"`
	ResultJSON   string `json:"result_json"`
	ErrorMessage string `json:"error_message"`
}

type addTaskLogReq struct {
	Level      string `json:"level"`
	Message    string `json:"message"`
	DetailJSON string `json:"detail_json"`
}

func CreateTaskHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		var req createTaskReq
		if json.NewDecoder(r.Body).Decode(&req) != nil {
			writeError(w, r, 10001, "请求参数错误")
			return
		}
		reply, err := svcCtx.Task.CreateTask(r.Context(), &taskclient.CreateTaskReq{UserId: userID, TaskType: req.TaskType, SourceType: req.SourceType, SourceId: req.SourceID, Title: req.Title, PayloadJson: req.PayloadJSON})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Item)
	}
}

func ListTasksHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		q := r.URL.Query()
		limit, _ := strconv.ParseInt(q.Get("limit"), 10, 64)
		reply, err := svcCtx.Task.ListTasks(r.Context(), &taskclient.ListTasksReq{UserId: userID, TaskType: q.Get("task_type"), Status: q.Get("status"), SourceType: q.Get("source_type"), SourceId: q.Get("source_id"), Limit: limit})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Items)
	}
}

func GetTaskCenterTaskHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := userAndPathID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Task.GetTask(r.Context(), &taskclient.GetTaskReq{UserId: userID, Id: id})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Item)
	}
}

func UpdateTaskStatusHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := userAndPathID(w, r)
		if !ok {
			return
		}
		var req updateTaskStatusReq
		if json.NewDecoder(r.Body).Decode(&req) != nil {
			writeError(w, r, 10001, "请求参数错误")
			return
		}
		reply, err := svcCtx.Task.UpdateTaskStatus(r.Context(), &taskclient.UpdateTaskStatusReq{UserId: userID, Id: id, Status: req.Status, Progress: req.Progress, ResultJson: req.ResultJSON, ErrorMessage: req.ErrorMessage})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Item)
	}
}

func CancelTaskHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := userAndPathID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Task.CancelTask(r.Context(), &taskclient.CancelTaskReq{UserId: userID, Id: id})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Item)
	}
}

func AddTaskLogHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := userAndPathID(w, r)
		if !ok {
			return
		}
		var req addTaskLogReq
		if json.NewDecoder(r.Body).Decode(&req) != nil {
			writeError(w, r, 10001, "请求参数错误")
			return
		}
		reply, err := svcCtx.Task.AddTaskLog(r.Context(), &taskclient.AddTaskLogReq{UserId: userID, TaskId: id, Level: req.Level, Message: req.Message, DetailJson: req.DetailJSON})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Item)
	}
}

func ListTaskLogsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := userAndPathID(w, r)
		if !ok {
			return
		}
		limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
		reply, err := svcCtx.Task.ListTaskLogs(r.Context(), &taskclient.ListTaskLogsReq{UserId: userID, TaskId: id, Limit: limit})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Items)
	}
}

func ListNotificationsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		q := r.URL.Query()
		limit, _ := strconv.ParseInt(q.Get("limit"), 10, 64)
		unreadOnly := q.Get("unread_only") == "true" || q.Get("unread_only") == "1"
		reply, err := svcCtx.Notify.ListNotifications(r.Context(), &notifyclient.ListNotificationsReq{UserId: userID, NotifyType: q.Get("notify_type"), Level: q.Get("level"), UnreadOnly: unreadOnly, Limit: limit})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Items)
	}
}

func CountUnreadNotificationsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Notify.CountUnread(r.Context(), &notifyclient.CountUnreadReq{UserId: userID})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, map[string]any{"count": reply.Count})
	}
}

func MarkNotificationReadHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := userAndPathID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Notify.MarkRead(r.Context(), &notifyclient.MarkReadReq{UserId: userID, Id: id})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Item)
	}
}

func MarkAllNotificationsReadHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Notify.MarkAllRead(r.Context(), &notifyclient.MarkAllReadReq{UserId: userID})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, map[string]any{"affected": reply.Affected})
	}
}

func DeleteNotificationHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := userAndPathID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Notify.DeleteNotification(r.Context(), &notifyclient.DeleteNotificationReq{UserId: userID, Id: id})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, map[string]any{"ok": reply.Ok, "affected": reply.Affected})
	}
}

func ClearNotificationsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		scope := r.URL.Query().Get("scope")
		if scope != "read" && scope != "all" {
			writeError(w, r, 10001, "清理范围必须是已读或全部")
			return
		}
		reply, err := svcCtx.Notify.DeleteNotification(r.Context(), &notifyclient.DeleteNotificationReq{UserId: userID, Scope: scope})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, map[string]any{"ok": reply.Ok, "affected": reply.Affected})
	}
}
