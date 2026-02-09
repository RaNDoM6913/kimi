package enums

type RejectReason string

const (
	RejectReasonFakePhoto       RejectReason = "FAKE_PHOTO"
	RejectReasonLowQuality      RejectReason = "LOW_QUALITY"
	RejectReasonPolicyViolation RejectReason = "POLICY_VIOLATION"
)
