package assets

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

// Test verilerinde tekrar eden yollar; goconst kuralı gereği sabitleştirilir.
const (
	fsCSS  = "css/styles.css"  // fs.FS biçimi
	urlCSS = "/css/styles.css" // istek yolu
	jsDir  = "/js"             // dizin isteği
)

// testFS, handler davranışını diskte dosya oluşturmadan sınamak için kullanılır.
var testFS = fstest.MapFS{
	fsCSS:                  {Data: []byte("body{}")},
	"js/htmx_4.0.0.min.js": {Data: []byte("/* htmx */")},
	"img/logo.png":         {Data: []byte("\x89PNG")},
}

func get(t *testing.T, h http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, http.NoBody))
	return rec
}

func TestHandlerServesFile(t *testing.T) {
	rec := get(t, newHandler(testFS), urlCSS)

	if rec.Code != http.StatusOK {
		t.Fatalf("durum = %d, beklenen %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); got != "body{}" {
		t.Errorf("gövde = %q, beklenen %q", got, "body{}")
	}
}

func TestHandlerHidesDirectoryListing(t *testing.T) {
	for _, dir := range []string{"/", "/css", jsDir, "/css/"} {
		t.Run(dir, func(t *testing.T) {
			rec := get(t, newHandler(testFS), dir)

			if rec.Code != http.StatusNotFound {
				t.Errorf("dizin isteği %d döndü, beklenen %d", rec.Code, http.StatusNotFound)
			}
			// Listede dosya adları sızmamalı.
			if body := rec.Body.String(); strings.Contains(body, "styles.css") || strings.Contains(body, "htmx_4.0.0.min.js") {
				t.Errorf("dizin içeriği sızdı: %q", body)
			}
		})
	}
}

// TestErrorResponseIsNotCacheable, hata yanıtlarının asla immutable
// cache'lenmediğini doğrular. Bu gerçek bir risktir: net/http'in FileServer
// hata yolunda Cache-Control'ı kendisi siler, ancak dizin 404'leri
// http.NotFound üzerinden döner ve http.Error onu temizlemez. Başlık yanlış
// yere konursa istemci bir yıl boyunca 404 görür.
func TestErrorResponseIsNotCacheable(t *testing.T) {
	paths := []string{
		"/js/yok_1.0.0.min.js", // eksik dosya
		"/js/",                 // dizin (immutable'a en yakın aday)
		jsDir,                  // dizin
		"/",                    // kök
	}

	for _, p := range paths {
		t.Run(p, func(t *testing.T) {
			rec := get(t, newHandler(testFS), p)

			if rec.Code < 400 {
				t.Fatalf("durum = %d, beklenen hata", rec.Code)
			}
			if cc := rec.Header().Get("Cache-Control"); cc == immutableCache {
				t.Errorf("%s → hata yanıtı immutable cache'lendi: %q", p, cc)
			}
		})
	}
}

func TestCacheControlByPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"sürümlü JS", "js/htmx_4.0.0.min.js", immutableCache},
		{"sürümsüz CSS", "css/styles.css", revalidateCache},
		{"diğer varlık", "img/logo.png", revalidateCache},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cacheControl(tt.path); got != tt.want {
				t.Errorf("cacheControl(%q) = %q, beklenen %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestFSPath(t *testing.T) {
	tests := []struct{ in, want string }{
		{urlCSS, fsCSS},
		{"/js", "js"},
		{"/js/", "js"},
		{"/", "."},
		{"", "."},
		{"/../../etc/passwd", "etc/passwd"},
		{"/css/../js/a.js", "js/a.js"},
	}

	for _, tt := range tests {
		if got := fsPath(tt.in); got != tt.want {
			t.Errorf("fsPath(%q) = %q, beklenen %q", tt.in, got, tt.want)
		}
	}
}

func TestHandlerSetsCacheControl(t *testing.T) {
	js := get(t, newHandler(testFS), "/js/htmx_4.0.0.min.js")
	if got := js.Header().Get("Cache-Control"); got != immutableCache {
		t.Errorf("JS cache = %q, beklenen %q", got, immutableCache)
	}

	css := get(t, newHandler(testFS), urlCSS)
	if got := css.Header().Get("Cache-Control"); got != revalidateCache {
		t.Errorf("CSS cache = %q, beklenen %q", got, revalidateCache)
	}
}

func TestNewRejectsMissingRoot(t *testing.T) {
	if _, err := New(filepath.Join(t.TempDir(), "yok")); err == nil {
		t.Fatal("olmayan dizin için hata bekleniyordu")
	}
}

func TestNewRejectsNonDirectory(t *testing.T) {
	file := filepath.Join(t.TempDir(), "static")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := New(file); err == nil {
		t.Fatal("dosya yolu için hata bekleniyordu")
	}
}

func TestNewUsesDefaultRoot(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "static"), 0o750); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	if _, err := New(""); err != nil {
		t.Fatalf("New(\"\") hata verdi: %v", err)
	}
}

func TestNewServesFromGivenRoot(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "css"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "css", "styles.css"), []byte("p{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	h, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	rec := get(t, h, urlCSS)
	if rec.Code != http.StatusOK || rec.Body.String() != "p{}" {
		t.Errorf("durum = %d, gövde = %q", rec.Code, rec.Body.String())
	}
}
