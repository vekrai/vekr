package assets

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	full := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPathChangesWhenContentChanges(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "js/app.js", "one")
	a := New(dir, "/static")

	first := a.Path("js/app.js")
	if !regexp.MustCompile(`^/static/js/app-[0-9a-f]{8}\.js$`).MatchString(first) {
		t.Fatalf("got %q, want a fingerprinted path", first)
	}
	if again := a.Path("js/app.js"); again != first {
		t.Errorf("got %q then %q, want a stable path for unchanged content", first, again)
	}

	writeFile(t, dir, "js/app.js", "two!")
	later := time.Now().Add(time.Second)
	os.Chtimes(filepath.Join(dir, "js/app.js"), later, later)
	if changed := a.Path("js/app.js"); changed == first {
		t.Errorf("got the same path %q after the content changed", changed)
	}
}

func TestPathFallsBackForMissingFile(t *testing.T) {
	a := New(t.TempDir(), "/static")
	if got := a.Path("css/missing.css"); got != "/static/css/missing.css" {
		t.Errorf("got %q, want the plain path", got)
	}
}

func TestHandlerServesFingerprintedFileImmutable(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "css/app.css", "body{}")
	a := New(dir, "/static")
	h := a.Handler()

	cases := []struct {
		url   string
		cache string
	}{
		{a.Path("css/app.css"), "public, max-age=31536000, immutable"},
		{"/static/css/app-00000000.css", "no-cache"},
		{"/static/css/app.css", "no-cache"},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, c.url, nil))
		if rec.Code != http.StatusOK || rec.Body.String() != "body{}" {
			t.Errorf("%s: got %d %q, want 200 with the file", c.url, rec.Code, rec.Body.String())
		}
		if got := rec.Header().Get("Cache-Control"); got != c.cache {
			t.Errorf("%s: got Cache-Control %q, want %q", c.url, got, c.cache)
		}
	}
}

func TestImportMapIncludesControllers(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "js/turbo.min.js", "t")
	writeFile(t, dir, "js/controllers/chat_controller.js", "c")
	a := New(dir, "/static")

	var parsed struct {
		Imports map[string]string `json:"imports"`
	}
	if err := json.Unmarshal([]byte(a.ImportMap(map[string]string{"@hotwired/turbo": "js/turbo.min.js"}, "js/controllers")), &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.Imports["@hotwired/turbo"] != a.Path("js/turbo.min.js") {
		t.Errorf("got turbo %q, want %q", parsed.Imports["@hotwired/turbo"], a.Path("js/turbo.min.js"))
	}
	if parsed.Imports["controllers/chat_controller.js"] != a.Path("js/controllers/chat_controller.js") {
		t.Errorf("got controller %q, want %q", parsed.Imports["controllers/chat_controller.js"], a.Path("js/controllers/chat_controller.js"))
	}
}
