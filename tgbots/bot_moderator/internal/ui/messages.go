package ui

import (
	"fmt"

	"bot_moderator/internal/domain/enums"
)

func StartMessage(role enums.Role) string {
	switch role {
	case enums.RoleNone:
		return "У вас нет доступа к этому боту"
	case enums.RoleModerator:
		return "Dating App: Приступить к модерации"
	default:
		return fmt.Sprintf("Dating App: роль %s", role)
	}
}
