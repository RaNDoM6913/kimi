package enums

type Role string

const (
	RoleOwner     Role = "OWNER"
	RoleAdmin     Role = "ADMIN"
	RoleModerator Role = "MODERATOR"
	RoleNone      Role = "NONE"
)
