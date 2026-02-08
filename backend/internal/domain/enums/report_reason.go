package enums

type ReportReason string

const (
	ReportReasonSpam    ReportReason = "spam"
	ReportReasonFake    ReportReason = "fake"
	ReportReasonAbusive ReportReason = "abusive"
	ReportReasonOther   ReportReason = "other"
)
