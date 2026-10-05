package domain

import (
	"context"
	"encoding/json"
	"fmt"

	"quant-system/backend-go/apps/notify-rpc/internal/repository"
	"quant-system/backend-go/apps/notify-rpc/notify"
	"quant-system/backend-go/common/constants"
	"quant-system/backend-go/common/idgen"
	"quant-system/backend-go/common/trace"
	"quant-system/backend-go/common/xmq"
)

type Service struct {
	notifications *repository.NotificationRepository
	events        *repository.EventRepository
	producer      xmq.Producer
}

func NewService(notifications *repository.NotificationRepository, events *repository.EventRepository, producer xmq.Producer) *Service {
	return &Service{notifications: notifications, events: events, producer: producer}
}

func (s *Service) CreateNotification(ctx context.Context, in *notify.CreateNotificationReq) (*notify.CreateNotificationResp, error) {
	item, err := s.notifications.Create(ctx, in)
	if err != nil {
		return nil, err
	}
	s.publishCreated(ctx, item)
	return &notify.CreateNotificationResp{Item: item}, nil
}

func (s *Service) ListNotifications(ctx context.Context, in *notify.ListNotificationsReq) (*notify.ListNotificationsResp, error) {
	items, err := s.notifications.List(ctx, in)
	if err != nil {
		return nil, err
	}
	return &notify.ListNotificationsResp{Items: items}, nil
}

func (s *Service) CountUnread(ctx context.Context, in *notify.CountUnreadReq) (*notify.CountUnreadResp, error) {
	count, err := s.notifications.CountUnread(ctx, in.GetUserId())
	if err != nil {
		return nil, err
	}
	return &notify.CountUnreadResp{Count: count}, nil
}

func (s *Service) MarkRead(ctx context.Context, in *notify.MarkReadReq) (*notify.MarkReadResp, error) {
	item, err := s.notifications.MarkRead(ctx, in.GetUserId(), in.GetId())
	if err != nil {
		return nil, err
	}
	return &notify.MarkReadResp{Item: item}, nil
}

func (s *Service) MarkAllRead(ctx context.Context, in *notify.MarkAllReadReq) (*notify.MarkAllReadResp, error) {
	affected, err := s.notifications.MarkAllRead(ctx, in.GetUserId())
	if err != nil {
		return nil, err
	}
	return &notify.MarkAllReadResp{Affected: affected}, nil
}

func (s *Service) DeleteNotification(ctx context.Context, in *notify.DeleteNotificationReq) (*notify.DeleteNotificationResp, error) {
	if in.GetUserId() == 0 {
		return nil, fmt.Errorf("用户信息无效")
	}
	var affected int64
	var err error
	if in.GetScope() == "read" || in.GetScope() == "all" {
		affected, err = s.notifications.DeleteScope(ctx, in.GetUserId(), in.GetScope())
	} else if in.GetId() > 0 {
		affected, err = s.notifications.Delete(ctx, in.GetUserId(), in.GetId())
	} else {
		return nil, fmt.Errorf("通知删除参数错误")
	}
	if err != nil {
		return nil, err
	}
	return &notify.DeleteNotificationResp{Ok: true, Affected: affected}, nil
}

func (s *Service) HandleEvent(ctx context.Context, event xmq.Event) error {
	body, _ := json.Marshal(event.Payload)
	created, err := s.events.SavePending(ctx, event.EventID, event.EventType, event.UserID, event.EventType, event.EventID, string(body))
	if err != nil || !created {
		return err
	}
	req, ok := BuildNotification(event)
	if !ok {
		return s.events.MarkHandled(ctx, event.EventID, 0)
	}
	resp, err := s.CreateNotification(ctx, req)
	if err != nil {
		s.events.MarkError(ctx, event.EventID, err.Error())
		return err
	}
	return s.events.MarkHandled(ctx, event.EventID, resp.Item.Id)
}

func (s *Service) publishCreated(ctx context.Context, item *notify.Notification) {
	_ = s.producer.Publish(ctx, xmq.Event{
		EventID:   idgen.NewEventID(),
		EventType: constants.EventNotifyCreated,
		TraceID:   trace.FromContext(ctx),
		UserID:    item.UserId,
		Payload: map[string]any{
			"notification_id": item.Id,
			"title":           item.Title,
			"notify_type":     item.NotifyType,
			"level":           item.Level,
		},
	})
}
