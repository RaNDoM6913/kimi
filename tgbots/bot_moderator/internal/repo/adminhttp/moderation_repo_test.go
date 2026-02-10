package adminhttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestModerationRepoAcquireCachesProfileAndMedia(t *testing.T) {
	t.Parallel()

	const actorTGID = int64(700001)
	now := time.Now().UTC().Truncate(time.Second)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/admin/bot/mod/queue/acquire" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("X-Actor-Tg-Id"); got != "700001" {
			t.Fatalf("unexpected actor header: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"moderation_item": {
				"id": 91,
				"user_id": 501,
				"status": "PENDING",
				"eta_bucket": "10m",
				"created_at": "` + now.Format(time.RFC3339) + `",
				"locked_at": "` + now.Format(time.RFC3339) + `"
			},
			"profile": {
				"user_id": 501,
				"tg_id": 998877,
				"username": "moderated_user",
				"display_name": "John Doe",
				"city_id": "Minsk",
				"gender": "M",
				"looking_for": "F",
				"goals": ["RELATIONSHIP"],
				"languages": ["ru","en"],
				"occupation": "Engineer",
				"education": "Bachelor"
			},
			"media": {
				"photos": [
					{"s3_key": "users/501/photo1.jpg"},
					{"url": "https://cdn.example.com/photo2.jpg"},
					{"presigned_url": "https://signed.example.com/photo3.jpg"}
				],
				"circle": {"presigned_url": "https://signed.example.com/circle.mp4"}
			}
		}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "bot-token", 2*time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	repo := NewModerationRepo(client, nil, false)
	ctx := WithActorTGID(context.Background(), actorTGID)

	item, err := repo.AcquireNextPending(ctx, actorTGID, 10*time.Minute)
	if err != nil {
		t.Fatalf("acquire next pending: %v", err)
	}
	if item.ID != 91 || item.UserID != 501 {
		t.Fatalf("unexpected item: %+v", item)
	}

	profile, err := repo.GetProfile(ctx, item.UserID)
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if profile.Username != "moderated_user" {
		t.Fatalf("unexpected profile username: %q", profile.Username)
	}

	photos, err := repo.ListPhotoKeys(ctx, item.UserID, 3)
	if err != nil {
		t.Fatalf("list photos: %v", err)
	}
	if len(photos) != 3 {
		t.Fatalf("unexpected photos len: %d (%v)", len(photos), photos)
	}
	if !strings.Contains(photos[0], "photo1.jpg") {
		t.Fatalf("unexpected first photo ref: %q", photos[0])
	}
	if !strings.Contains(photos[1], "photo2.jpg") {
		t.Fatalf("unexpected second photo ref: %q", photos[1])
	}
	if !strings.Contains(photos[2], "photo3.jpg") {
		t.Fatalf("unexpected third photo ref: %q", photos[2])
	}

	circle, err := repo.GetLatestCircleKey(ctx, item.UserID)
	if err != nil {
		t.Fatalf("get circle: %v", err)
	}
	if circle != "https://signed.example.com/circle.mp4" {
		t.Fatalf("unexpected circle ref: %q", circle)
	}
}

func TestShouldFallbackModeration(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		dual bool
		err  error
		want bool
	}{
		{
			name: "dual_404",
			dual: true,
			err: &RequestError{
				StatusCode:   http.StatusNotFound,
				Fallbackable: false,
			},
			want: true,
		},
		{
			name: "db_mode_404",
			dual: false,
			err: &RequestError{
				StatusCode:   http.StatusNotFound,
				Fallbackable: false,
			},
			want: false,
		},
		{
			name: "dual_401",
			dual: true,
			err: &RequestError{
				StatusCode:   http.StatusUnauthorized,
				Fallbackable: false,
			},
			want: false,
		},
		{
			name: "dual_5xx",
			dual: true,
			err: &RequestError{
				StatusCode:   http.StatusBadGateway,
				Fallbackable: true,
			},
			want: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := shouldFallbackModeration(tc.dual, tc.err)
			if got != tc.want {
				t.Fatalf("fallback mismatch: got=%v want=%v", got, tc.want)
			}
		})
	}
}
