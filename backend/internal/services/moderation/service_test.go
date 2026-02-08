package moderation

import "testing"

func TestETABucketFromQueueSize(t *testing.T) {
	tests := []struct {
		queueSize int
		want      string
	}{
		{queueSize: 0, want: "up_to_10"},
		{queueSize: 5, want: "up_to_10"},
		{queueSize: 10, want: "up_to_10"},
		{queueSize: 11, want: "up_to_20"},
		{queueSize: 20, want: "up_to_20"},
		{queueSize: 21, want: "up_to_30"},
		{queueSize: 30, want: "up_to_30"},
		{queueSize: 31, want: "up_to_40"},
		{queueSize: 40, want: "up_to_40"},
		{queueSize: 41, want: "up_to_50"},
		{queueSize: 49, want: "up_to_50"},
		{queueSize: 50, want: "more_than_hour"},
		{queueSize: 200, want: "more_than_hour"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := ETABucketFromQueueSize(tt.queueSize)
			if got != tt.want {
				t.Fatalf("unexpected bucket for queue=%d: got %s want %s", tt.queueSize, got, tt.want)
			}
		})
	}
}
