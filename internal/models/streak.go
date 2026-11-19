package models

import "time"

func Streak(activity []time.Time, now time.Time) int {
	days := make(map[string]bool, len(activity))
	for _, at := range activity {
		days[at.In(now.Location()).Format(time.DateOnly)] = true
	}

	day := now
	if !days[day.Format(time.DateOnly)] {
		day = day.AddDate(0, 0, -1)
	}
	streak := 0
	for days[day.Format(time.DateOnly)] {
		streak++
		day = day.AddDate(0, 0, -1)
	}
	return streak
}
