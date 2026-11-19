package i18n

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

func TestResolveFallsBackToEnglish(t *testing.T) {
	cases := []struct {
		session, accept string
	}{
		{"", ""},
		{"", "fr-FR,fr;q=0.9"},
		{"xx-not-a-tag", "de"},
		{"", "en-GB,en;q=0.8"},
	}
	for _, c := range cases {
		if got := Resolve(c.session, c.accept); got != language.English {
			t.Errorf("Resolve(%q, %q) = %v, want en", c.session, c.accept, got)
		}
	}
}

func TestTranslateLooksUpCatalog(t *testing.T) {
	ctx := WithLanguage(context.Background(), language.English)

	if got := T(ctx, "chat.send"); got != "Send" {
		t.Errorf("T(chat.send) = %q, want %q", got, "Send")
	}
	if got := TData(ctx, "practice.title", map[string]any{"Title": "Recursion"}); got != "Practice: Recursion" {
		t.Errorf("TData(practice.title) = %q", got)
	}
	if got := TCount(ctx, "notes.count", 1, nil); got != "1 note" {
		t.Errorf("TCount(notes.count, 1) = %q, want %q", got, "1 note")
	}
	if got := TCount(ctx, "dashboard.weak.missed", 3, map[string]any{"Misses": 2}); got != "missed 2 of 3 attempts" {
		t.Errorf("TCount(dashboard.weak.missed, 3) = %q", got)
	}
}

func TestTranslateWithoutMiddlewareUsesDefault(t *testing.T) {
	if got := T(context.Background(), "chat.send"); got != "Send" {
		t.Errorf("T without localizer in ctx = %q, want %q", got, "Send")
	}
	if got := Lang(context.Background()); got != "en" {
		t.Errorf("Lang without localizer in ctx = %q, want en", got)
	}
}

func TestMissingKeyReturnsID(t *testing.T) {
	if got := T(context.Background(), "no.such.key"); got != "no.such.key" {
		t.Errorf("T(missing) = %q, want the id back", got)
	}
}

func TestEveryTemplateKeyExistsInCatalog(t *testing.T) {
	files, err := filepath.Glob("../../web/templates/*.templ")
	if err != nil || len(files) == 0 {
		t.Fatalf("glob templates: %v (found %d)", err, len(files))
	}

	call := regexp.MustCompile(`\bt(?:Data|Count)?\(ctx, "([^"]+)"\s*[,)]`)
	ctx := context.Background()

	for _, file := range files {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		for _, m := range call.FindAllStringSubmatch(string(src), -1) {
			if !bundleHas(ctx, m[1]) {
				t.Errorf("%s uses %q, which is missing from locales/en.yaml", filepath.Base(file), m[1])
			}
		}
	}

	dynamic := []string{"queue.kind.learn", "queue.kind.practice"}
	for _, status := range []string{"not_started", "learning", "practicing", "mastered"} {
		dynamic = append(dynamic, "topic.status."+status, "topic.status_hint."+status)
	}
	for _, itemType := range []string{"mcq", "flashcard", "open", "code"} {
		dynamic = append(dynamic, "practice.type."+itemType)
	}
	for _, verdict := range []string{"passed", "passed_with_note", "not_yet"} {
		dynamic = append(dynamic, "practice.verdict."+verdict)
	}
	for _, relation := range []string{"prerequisite_of", "related", "source_of"} {
		dynamic = append(dynamic, "note.relation."+relation+".in", "note.relation."+relation+".out")
	}
	for _, tool := range []string{"websearch", "webfetch", "search_notes", "read_note", "edit_note", "generate_curriculum_preview", "commit_curriculum_preview", "generate_practice_items"} {
		dynamic = append(dynamic, "chat.tool."+tool)
	}
	for _, n := range []string{"1", "2", "3"} {
		dynamic = append(dynamic, "chat.suggest."+n, "chat.suggest_note."+n)
	}
	for _, id := range dynamic {
		if !bundleHas(ctx, id) {
			t.Errorf("%s is missing from locales/en.yaml", id)
		}
	}
}

func TestLocalesHaveTheSameKeys(t *testing.T) {
	keys := func(name string) map[string]bool {
		src, err := os.ReadFile(filepath.Join("../../locales", name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		var catalog map[string]any
		if err := yaml.Unmarshal(src, &catalog); err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		out := make(map[string]bool, len(catalog))
		for k := range catalog {
			out[k] = true
		}
		return out
	}

	en, zh := keys("en.yaml"), keys("zh.yaml")
	for k := range en {
		if !zh[k] {
			t.Errorf("%s is in en.yaml but missing from zh.yaml", k)
		}
	}
	for k := range zh {
		if !en[k] {
			t.Errorf("%s is in zh.yaml but missing from en.yaml", k)
		}
	}
}

func bundleHas(ctx context.Context, id string) bool {
	return TCount(ctx, id, 2, nil) != id
}
