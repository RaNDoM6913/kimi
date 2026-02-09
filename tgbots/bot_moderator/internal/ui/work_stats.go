package ui

import (
	"fmt"
	"strconv"
	"strings"

	"bot_moderator/internal/domain/model"
)

func RenderWorkStats(report model.WorkStatsReport) string {
	lines := []string{
		"Work Stats",
		fmt.Sprintf("Total day/week/month/all: %d / %d / %d / %d", report.Totals.Day, report.Totals.Week, report.Totals.Month, report.Totals.All),
	}
	if len(report.Actors) == 0 {
		lines = append(lines, "Сотрудники: —")
	} else {
		lines = append(lines, "Сотрудники:")
		for _, actor := range report.Actors {
			userLabel := renderWorkStatsUserLabel(actor.ActorTGID, actor.Username)
			role := strings.TrimSpace(actor.ActorRole)
			if role == "" {
				role = "-"
			}
			lines = append(lines, fmt.Sprintf(
				"%s (%s) — day:%d | week:%d | month:%d | all:%d",
				userLabel,
				role,
				actor.Day,
				actor.Week,
				actor.Month,
				actor.All,
			))
		}
	}
	return strings.Join(lines, "\n")
}

func renderWorkStatsUserLabel(tgID int64, username string) string {
	name := strings.TrimSpace(username)
	if name != "" {
		return "@" + name
	}
	return strconv.FormatInt(tgID, 10)
}
