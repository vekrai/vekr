package models

import (
	"testing"
	"time"
)

func TestStreak(t *testing.T) {
	now := time.Date(2026, 11, 19, 15, 0, 0, 0, time.UTC)
	day := func(offset int) time.Time { return now.AddDate(0, 0, -offset) }

	cases := []struct {
		name     string
		activity []time.Time
		want     int
	}{
		{"none", nil, 0},
		{"today only", []time.Time{day(0)}, 1},
		{"today and yesterday twice", []time.Time{day(0), day(1), day(1)}, 2},
		{"yesterday keeps the streak alive", []time.Time{day(1), day(2)}, 2},
		{"gap breaks it", []time.Time{day(0), day(2), day(3)}, 1},
		{"two days ago is already broken", []time.Time{day(2)}, 0},
	}
	for _, c := range cases {
		if got := Streak(c.activity, now); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}

func TestHumanizeConcept(t *testing.T) {
	cases := map[string]string{
		"experience-vs-event": "Experience vs event",
		"if_clause_tense":     "If clause tense",
		"Big-O":               "Big O",
		"边界":                  "边界",
		"":                    "",
	}
	for in, want := range cases {
		if got := HumanizeConcept(in); got != want {
			t.Errorf("HumanizeConcept(%q) = %q, want %q", in, got, want)
		}
	}
}
