package handlers

import (
	"strings"
	"testing"

	"vekr/internal/models"
)

func TestInterleaveByTopicAlternatesTopics(t *testing.T) {
	items := []models.PracticeItem{
		{ID: 1, TopicID: 10},
		{ID: 2, TopicID: 10},
		{ID: 3, TopicID: 10},
		{ID: 4, TopicID: 20},
		{ID: 5, TopicID: 30},
		{ID: 6, TopicID: 30},
	}

	got := interleaveByTopic(items)

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

func TestWriteEventPrefixesEveryLine(t *testing.T) {
	var sb strings.Builder
	if err := writeEvent(&sb, "<turbo-stream>\n<template>hi</template>\n</turbo-stream>"); err != nil {
		t.Fatalf("writeEvent: %v", err)
	}
	want := "data: <turbo-stream>\ndata: <template>hi</template>\ndata: </turbo-stream>\n\n"
	if sb.String() != want {
		t.Errorf("got %q, want %q", sb.String(), want)
	}
}
