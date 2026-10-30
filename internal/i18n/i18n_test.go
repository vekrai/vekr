package i18n

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"golang.org/x/text/language"
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
	if got := TCount(ctx, "notes.count", 1, nil); got != "1 item" {
		t.Errorf("TCount(notes.count, 1) = %q, want %q", got, "1 item")
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

	for _, kind := range []string{"learn", "practice"} {
		if !bundleHas(ctx, "queue.kind."+kind) {
			t.Errorf("queue.kind.%s is missing from locales/en.yaml", kind)
		}
	}
}

func bundleHas(ctx context.Context, id string) bool {
	return TCount(ctx, id, 2, nil) != id
}
