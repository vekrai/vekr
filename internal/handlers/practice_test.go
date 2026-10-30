package handlers

import (
	"testing"

	"vekr/internal/models"
)

func TestInterleaveByNodeAlternatesTopics(t *testing.T) {
	items := []models.PracticeItem{
		{ID: 1, NodeID: 10},
		{ID: 2, NodeID: 10},
		{ID: 3, NodeID: 10},
		{ID: 4, NodeID: 20},
		{ID: 5, NodeID: 30},
		{ID: 6, NodeID: 30},
	}

	got := interleaveByNode(items)

	want := []uint{1, 4, 5, 2, 6, 3}
	if len(got) != len(want) {
		t.Fatalf("got %d items, want %d", len(got), len(want))
	}
	for i, id := range want {
		if got[i].ID != id {
			t.Errorf("position %d: got item %d, want %d", i, got[i].ID, id)
		}
	}
}

func TestIsLocalPath(t *testing.T) {
	cases := map[string]bool{
		"/drill?concept_id=1": true,
		"/topics/3/practice":  true,
		"":                    false,
		"https://evil.test":   false,
		"//evil.test":         false,
		"/\\evil.test":        false,
	}
	for path, want := range cases {
		if got := isLocalPath(path); got != want {
			t.Errorf("isLocalPath(%q) = %v, want %v", path, got, want)
		}
	}
}
