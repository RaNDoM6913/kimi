package ui

import (
	"strings"
	"testing"

	"bot_moderator/internal/domain/model"
)

func TestRenderWorkStats(t *testing.T) {
	report := model.WorkStatsReport{
		Totals: model.WorkStatsTotals{
			Day:   3,
			Week:  12,
			Month: 30,
			All:   87,
		},
		Actors: []model.WorkStatsActor{
			{
				ActorTGID: 1001,
				ActorRole: "MODERATOR",
				Username:  "mod_one",
				Day:       2,
				Week:      9,
				Month:     20,
				All:       40,
			},
			{
				ActorTGID: 1002,
				ActorRole: "ADMIN",
				Username:  "",
				Day:       1,
				Week:      3,
				Month:     10,
				All:       47,
			},
		},
	}

	text := RenderWorkStats(report)
	if strings.TrimSpace(text) == "" {
		t.Fatal("expected non-empty stats text")
	}

	required := []string{
		"Work Stats",
		"Total day/week/month/all: 3 / 12 / 30 / 87",
		"@mod_one (MODERATOR) — day:2 | week:9 | month:20 | all:40",
		"1002 (ADMIN) — day:1 | week:3 | month:10 | all:47",
	}
	for _, token := range required {
		if !strings.Contains(text, token) {
			t.Fatalf("expected stats text to contain %q; got:\n%s", token, text)
		}
	}
}
