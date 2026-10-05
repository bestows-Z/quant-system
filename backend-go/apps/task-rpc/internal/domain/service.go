package domain

import (
	"context"
	"encoding/json"
	"fmt"

	"quant-system/backend-go/apps/task-rpc/internal/repository"
	"quant-system/backend-go/apps/task-rpc/task"
	"quant-system/backend-go/common/constants"
	"quant-system/backend-go/common/xmq"
)

type Service struct {
	tasks *repository.TaskRepository
	logs  *repository.TaskLogRepository
}

func NewService(tasks *repository.TaskRepository, logs *repository.TaskLogRepository) *Service {
	return &Service{tasks: tasks, logs: logs}
}

func (s *Service) CreateTask(ctx context.Context, in *task.CreateTaskReq) (*task.CreateTaskResp, error) {
	item, err := s.tasks.Create(ctx, in)
	if err != nil {
		return nil, err
	}
	_, _ = s.logs.Add(ctx, item.Id, item.TaskNo, "INFO", "任务已创建", in.GetPayloadJson())
	return &task.CreateTaskResp{Item: item}, nil
}

func (s *Service) ListTasks(ctx context.Context, in *task.ListTasksReq) (*task.ListTasksResp, error) {
	items, err := s.tasks.List(ctx, in)
	if err != nil {
		return nil, err
	}
	return &task.ListTasksResp{Items: items}, nil
}

func (s *Service) GetTask(ctx context.Context, in *task.GetTaskReq) (*task.GetTaskResp, error) {
	item, err := s.tasks.Get(ctx, in.GetUserId(), in.GetId())
	if err != nil {
		return nil, err
	}
	return &task.GetTaskResp{Item: item}, nil
}

func (s *Service) UpdateTaskStatus(ctx context.Context, in *task.UpdateTaskStatusReq) (*task.UpdateTaskStatusResp, error) {
	item, err := s.tasks.UpdateStatus(ctx, in.GetUserId(), in.GetId(), in.GetStatus(), in.GetProgress(), in.GetResultJson(), in.GetErrorMessage())
	if err != nil {
		return nil, err
	}
	_, _ = s.logs.Add(ctx, item.Id, item.TaskNo, "INFO", "任务状态更新为 "+item.Status, in.GetResultJson())
	return &task.UpdateTaskStatusResp{Item: item}, nil
}

func (s *Service) CancelTask(ctx context.Context, in *task.CancelTaskReq) (*task.CancelTaskResp, error) {
	item, err := s.tasks.Cancel(ctx, in.GetUserId(), in.GetId())
	if err != nil {
		return nil, err
	}
	_, _ = s.logs.Add(ctx, item.Id, item.TaskNo, "WARNING", "任务已取消", "{}")
	return &task.CancelTaskResp{Item: item}, nil
}

func (s *Service) AddTaskLog(ctx context.Context, in *task.AddTaskLogReq) (*task.AddTaskLogResp, error) {
	item, err := s.tasks.Get(ctx, in.GetUserId(), in.GetTaskId())
	if err != nil {
		return nil, err
	}
	log, err := s.logs.Add(ctx, item.Id, item.TaskNo, in.GetLevel(), in.GetMessage(), in.GetDetailJson())
	if err != nil {
		return nil, err
	}
	return &task.AddTaskLogResp{Item: log}, nil
}

func (s *Service) ListTaskLogs(ctx context.Context, in *task.ListTaskLogsReq) (*task.ListTaskLogsResp, error) {
	if _, err := s.tasks.Get(ctx, in.GetUserId(), in.GetTaskId()); err != nil {
		return nil, err
	}
	items, err := s.logs.List(ctx, in.GetTaskId(), in.GetLimit())
	if err != nil {
		return nil, err
	}
	return &task.ListTaskLogsResp{Items: items}, nil
}

func (s *Service) HandleEvent(ctx context.Context, event xmq.Event) error {
	taskType, statusText, title, sourceID := mapEvent(event)
	if taskType == "" {
		return nil
	}
	body, _ := json.Marshal(event)
	item, err := s.tasks.UpsertFromEvent(ctx, taskType, event.EventType, sourceID, title, statusText, event.UserID, string(body))
	if err != nil {
		return err
	}
	level := "INFO"
	if statusText == StatusFailed {
		level = "ERROR"
	}
	_, _ = s.logs.Add(ctx, item.Id, item.TaskNo, level, fmt.Sprintf("事件同步: %s", event.EventType), string(body))
	return nil
}

func mapEvent(event xmq.Event) (taskType, statusText, title, sourceID string) {
	sourceID = fmt.Sprint(event.Payload["task_id"])
	if sourceID == "<nil>" || sourceID == "" {
		sourceID = event.EventID
	}
	switch event.EventType {
	case constants.EventBacktestTaskCreated:
		return TypeBacktestRun, StatusRunning, "回测任务运行中", sourceID
	case constants.EventBacktestTaskCompleted:
		return TypeBacktestRun, StatusSuccess, "回测任务已完成", sourceID
	case constants.EventBacktestTaskFailed:
		return TypeBacktestRun, StatusFailed, "回测任务失败", sourceID
	case constants.EventStrategyRunCreated:
		return TypeStrategyRun, StatusRunning, "策略运行中", fmt.Sprint(event.Payload["run_id"])
	case constants.EventStrategyRunCompleted, constants.EventStrategySignalGenerated:
		return TypeStrategyRun, StatusSuccess, "策略运行完成", fmt.Sprint(event.Payload["run_id"])
	case constants.EventStrategyRunFailed:
		return TypeStrategyRun, StatusFailed, "策略运行失败", fmt.Sprint(event.Payload["run_id"])
	case constants.EventAgentTaskCreated:
		return TypeAgent, StatusRunning, "智能体分析中", sourceID
	case constants.EventAgentTaskCompleted:
		return TypeAgent, StatusSuccess, "智能体分析已完成", sourceID
	case constants.EventAgentTaskFailed:
		return TypeAgent, StatusFailed, "智能体分析失败", sourceID
	case constants.EventRiskWarningCreated:
		return TypeRisk, StatusSuccess, "风控分析产生预警", event.EventID
	case constants.EventRagDocumentUploaded, constants.EventRagDocumentChunked, constants.EventRagEmbeddingCompleted:
		return TypeRag, StatusSuccess, "RAG 文档处理事件", event.EventID
	default:
		return "", "", "", ""
	}
}
