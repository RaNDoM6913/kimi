package enums

type AuditAction string

const (
	AuditActionBotStart          AuditAction = "BOT_START"
	AuditActionRoleGranted       AuditAction = "ROLE_GRANTED"
	AuditActionRoleRevoked       AuditAction = "ROLE_REVOKED"
	AuditActionModerationApprove AuditAction = "MODERATION_APPROVE"
	AuditActionModerationReject  AuditAction = "MODERATION_REJECT"
	AuditActionLookupUser        AuditAction = "LOOKUP_USER"
	AuditActionBanUser           AuditAction = "BAN_USER"
	AuditActionUnbanUser         AuditAction = "UNBAN_USER"
	AuditActionForceReview       AuditAction = "FORCE_REVIEW"
	AuditActionSystemToggleReg   AuditAction = "SYSTEM_TOGGLE_REGISTRATION"
	AuditActionSystemViewUsers   AuditAction = "SYSTEM_VIEW_USERS_COUNT"
	AuditActionSystemViewWork    AuditAction = "SYSTEM_VIEW_WORK_STATS"
)
