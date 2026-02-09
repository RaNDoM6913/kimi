package model

import "bot_moderator/internal/domain/enums"

type BotRoleAssignment struct {
	TgID      int64
	Role      enums.Role
	Username  string
	FirstName string
	LastName  string
}
