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

type LoginLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *LoginLogic) Login(in *user.LoginRequest) (*user.AuthReply, error) {
	account := strings.TrimSpace(in.GetUsername())
	if account == "" || in.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "用户名和密码不能为空")
	}
	found, err := l.svcCtx.Users.FindByUsernameOrEmail(l.ctx, account)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.Unauthenticated, "用户名或密码错误")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "查询用户失败")
	}
	if found.Status != "ACTIVE" {
		return nil, status.Error(codes.PermissionDenied, "用户状态不可用")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(found.PasswordHash), []byte(in.GetPassword())); err != nil {
		return nil, status.Error(codes.Unauthenticated, "用户名或密码错误")
	}
	roles, err := l.svcCtx.Users.Roles(l.ctx, found.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, "查询角色失败")
	}
	expire := time.Duration(l.svcCtx.Config.JWT.ExpireHours) * time.Hour
	if expire <= 0 {
		expire = 72 * time.Hour
	}
	token, err := commonjwt.Generate(l.svcCtx.Config.JWT.Secret, found.ID, found.Username, roles, expire)
	if err != nil {
		return nil, status.Error(codes.Internal, "生成 token 失败")
	}

	return &user.AuthReply{
		User:     toUserInfo(found, roles),
		Token:    token,
		ExpireAt: time.Now().Add(expire).Unix(),
	}, nil
}
