package logic

import (
	"quant-system/backend-go/apps/user-rpc/internal/repository"
	"quant-system/backend-go/apps/user-rpc/user"
)

func toUserInfo(item *repository.User, roles []string) *user.UserInfo {
	email := ""
	if item.Email.Valid {
		email = item.Email.String
	}
	nickname := ""
	if item.Nickname.Valid {
		nickname = item.Nickname.String
	}
	return &user.UserInfo{
		Id:        item.ID,
		Username:  item.Username,
		Email:     email,
		Nickname:  nickname,
		Status:    item.Status,
		Roles:     roles,
		CreatedAt: item.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}
