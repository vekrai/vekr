package assets

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const hashLength = 8

var fingerprinted = regexp.MustCompile(`^(.+)-([0-9a-f]{8})(\.[^./]+)$`)

type entry struct {
	modTime time.Time
	size    int64
	hash    string
}

type Assets struct {
	dir    string
	prefix string

	mu      sync.Mutex
	entries map[string]entry
}

func New(dir, prefix string) *Assets {
	return &Assets{dir: dir, prefix: strings.TrimSuffix(prefix, "/"), entries: make(map[string]entry)}
}

var Default = New("web/static", "/static")

func Path(name string) string { return Default.Path(name) }

func ImportMap(imports map[string]string, controllersDir string) string {
	return Default.ImportMap(imports, controllersDir)
}

func Handler() http.Handler { return Default.Handler() }

func (a *Assets) Path(name string) string {
	hash, err := a.hash(name)
	if err != nil {
		return a.prefix + "/" + name
	}
	ext := path.Ext(name)
	return a.prefix + "/" + strings.TrimSuffix(name, ext) + "-" + hash + ext
}

func (a *Assets) ImportMap(imports map[string]string, controllersDir string) string {
	resolved := make(map[string]string, len(imports))
	for specifier, name := range imports {
		resolved[specifier] = a.Path(name)
	}

	files, _ := os.ReadDir(filepath.Join(a.dir, filepath.FromSlash(controllersDir)))
	names := make([]string, 0, len(files))
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".js") {
			names = append(names, f.Name())
		}
	}
	sort.Strings(names)
	for _, n := range names {
		resolved["controllers/"+n] = a.Path(path.Join(controllersDir, n))
	}

	out, _ := json.Marshal(map[string]any{"imports": resolved})
	return string(out)
}

func (a *Assets) Handler() http.Handler {
	files := http.FileServer(http.Dir(a.dir))
	return http.StripPrefix(a.prefix+"/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Path
		dir, file := path.Split(name)
		if m := fingerprinted.FindStringSubmatch(file); m != nil {
			logical := dir + m[1] + m[3]
			if current, err := a.hash(logical); err == nil {
				if current == m[2] {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				} else {
					w.Header().Set("Cache-Control", "no-cache")
				}
				r2 := r.Clone(r.Context())
				r2.URL.Path = logical
				files.ServeHTTP(w, r2)
				return
			}
		}
		w.Header().Set("Cache-Control", "no-cache")
		files.ServeHTTP(w, r)
	}))
}

func (a *Assets) hash(name string) (string, error) {
	full := filepath.Join(a.dir, filepath.FromSlash(path.Clean("/"+name)))
	info, err := os.Stat(full)
	if err != nil {
		return "", err
	}

	a.mu.Lock()
	cached, ok := a.entries[name]
	a.mu.Unlock()
	if ok && cached.modTime.Equal(info.ModTime()) && cached.size == info.Size() {
		return cached.hash, nil
	}

	f, err := os.Open(full)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	sum := hex.EncodeToString(h.Sum(nil))[:hashLength]

	a.mu.Lock()
	a.entries[name] = entry{modTime: info.ModTime(), size: info.Size(), hash: sum}
	a.mu.Unlock()
	return sum, nil
}
