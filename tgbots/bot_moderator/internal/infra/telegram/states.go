package telegram

type State string

const (
	StateIdle                State = "IDLE"
	StateWaitingRejectReason State = "WAITING_REJECT_REASON"
	StateWaitingBanReason    State = "WAITING_BAN_REASON"
)
