package logic

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"quant-system/backend-go/apps/user-rpc/internal/svc"
	"quant-system/backend-go/apps/user-rpc/user"
	commonjwt "quant-system/backend-go/common/jwt"

	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type RegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RegisterLogic) Register(in *user.RegisterRequest) (*user.AuthReply, error) {
	username := strings.TrimSpace(in.GetUsername())
	password := in.GetPassword()
	if len(username) < 3 {
		return nil, status.Error(codes.InvalidArgument, "用户名至少需要 3 个字符")
	}
	if len(password) < 6 {
		return nil, status.Error(codes.InvalidArgument, "密码至少需要 6 个字符")
	}
	if existing, err := l.svcCtx.Users.FindByUsernameOrEmail(l.ctx, username); err == nil && existing != nil {
		return nil, status.Error(codes.AlreadyExists, "用户名已存在")
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.Internal, "查询用户失败")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Error(codes.Internal, "密码加密失败")
	}
	created, err := l.svcCtx.Users.Create(l.ctx, username, strings.TrimSpace(in.GetEmail()), string(hash), strings.TrimSpace(in.GetNickname()))
	if err != nil {
		return nil, status.Error(codes.Internal, "创建用户失败")
	}
	roles, err := l.svcCtx.Users.Roles(l.ctx, created.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, "查询角色失败")
	}
	expire := time.Duration(l.svcCtx.Config.JWT.ExpireHours) * time.Hour
	if expire <= 0 {
		expire = 72 * time.Hour
	}
	token, err := commonjwt.Generate(l.svcCtx.Config.JWT.Secret, created.ID, created.Username, roles, expire)
	if err != nil {
		return nil, status.Error(codes.Internal, "生成 token 失败")
	}
	return &user.AuthReply{
		User:     toUserInfo(created, roles),
		Token:    token,
		ExpireAt: time.Now().Add(expire).Unix(),
	}, nil
}
