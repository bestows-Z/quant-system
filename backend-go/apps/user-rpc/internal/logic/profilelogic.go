package logic

import (
	"context"
	"database/sql"
	"errors"

	"quant-system/backend-go/apps/user-rpc/internal/svc"
	"quant-system/backend-go/apps/user-rpc/user"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ProfileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewProfileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ProfileLogic {
	return &ProfileLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ProfileLogic) Profile(in *user.ProfileRequest) (*user.ProfileReply, error) {
	if in.GetUserId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id 不能为空")
	}
	found, err := l.svcCtx.Users.FindByID(l.ctx, in.GetUserId())
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "用户不存在")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "查询用户失败")
	}
	roles, err := l.svcCtx.Users.Roles(l.ctx, found.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, "查询角色失败")
	}

	return &user.ProfileReply{User: toUserInfo(found, roles)}, nil
}
