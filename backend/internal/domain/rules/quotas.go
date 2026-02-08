package rules

import "time"

const (
	FreeLikesPerDay = 35
)

func UnlimitedLikesForPlus(isPlus bool) bool {
	return isPlus
}

func DayKey(now time.Time, loc *time.Location) string {
	if loc == nil {
		loc = time.UTC
	}
	return now.In(loc).Format("2006-01-02")
}

func NextResetAt(now time.Time, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	local := now.In(loc)
	next := time.Date(local.Year(), local.Month(), local.Day()+1, 0, 0, 0, 0, loc)
	return next.UTC()
}
