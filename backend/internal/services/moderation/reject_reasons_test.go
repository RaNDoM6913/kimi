package moderation

import "testing"

func TestListRejectReasonsCoversAllowedCodes(t *testing.T) {
	svc := NewService(nil, nil, nil, nil)
	items := svc.ListRejectReasons()

	if len(items) != len(allowedRejectReasonCodes) {
		t.Fatalf("unexpected reject reasons count: got=%d want=%d", len(items), len(allowedRejectReasonCodes))
	}

	byCode := make(map[string]RejectReasonItem, len(items))
	for _, item := range items {
		if _, exists := byCode[item.ReasonCode]; exists {
			t.Fatalf("duplicate reason code: %s", item.ReasonCode)
		}
		byCode[item.ReasonCode] = item
	}

	for code := range allowedRejectReasonCodes {
		item, ok := byCode[code]
		if !ok {
			t.Fatalf("missing reason code: %s", code)
		}
		if item.Label == "" {
			t.Fatalf("empty label for reason code: %s", code)
		}
		if item.ReasonText == "" {
			t.Fatalf("empty reason_text for reason code: %s", code)
		}
		if item.RequiredFixStep == "" {
			t.Fatalf("empty required_fix_step for reason code: %s", code)
		}
	}
}
