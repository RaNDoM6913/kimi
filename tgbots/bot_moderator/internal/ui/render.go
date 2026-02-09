package ui

import "bot_moderator/internal/domain/enums"

func RenderStart(role enums.Role) (string, [][]string) {
	return StartMessage(role), MenuByRole(role)
}
