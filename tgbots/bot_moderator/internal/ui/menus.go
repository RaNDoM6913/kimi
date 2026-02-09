package ui

import "bot_moderator/internal/domain/enums"

func MenuByRole(role enums.Role) [][]string {
	switch role {
	case enums.RoleOwner:
		return [][]string{
			{"Dating App"},
			{"Приступить к модерации"},
			{"Access", "Find user"},
			{"History", "System"},
			{"Work Stats"},
		}
	case enums.RoleAdmin:
		return [][]string{
			{"Dating App"},
			{"Приступить к модерации"},
			{"Access", "Find user"},
		}
	case enums.RoleModerator:
		return [][]string{
			{"Dating App"},
			{"Приступить к модерации"},
		}
	default:
		return [][]string{}
	}
}
