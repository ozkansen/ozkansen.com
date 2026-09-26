package main

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// testStaticRoot, testler için geçerli bir statik kök hazırlar.
func testStaticRoot(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "styles.css"), []byte("body{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

// newTestRouter, gerçek route zincirini test logger'ı ile kurar.
func newTestRouter(t *testing.T) (*chi.Mux, *bytes.Buffer) {
	t.Helper()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	r, err := newRouter(logger, testStaticRoot(t))
	if err != nil {
		t.Fatalf("newRouter: %v", err)
	}
	return r, &buf
}

// TestRouterRecoversPanicAndLogsRequest, gerçek route zincirinin bir paniği
// 500'e çevirdiğini VE isteği logladığını doğrular.
//
// Bu testin asıl işlevi zincir SIRASINI korumaktır: Recoverer, Logger'ın
// içinde değilse panik Logger'ın çağrısını atlar ve istek logu sessizce
// kaybolur. Yanlış sırayı kuran bir düzenleme burada kırmızıya döner.
func TestRouterRecoversPanicAndLogsRequest(t *testing.T) {
	r, logs := newTestRouter(t)

	// Panik fırlatan bir route ekle: chi'de Use'tan sonra route eklenebilir.
	r.Get("/test/patlar", func(http.ResponseWriter, *http.Request) {
		panic("bilerek fırlatıldı")
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test/patlar", http.NoBody))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("durum = %d, beklenen %d", rec.Code, http.StatusInternalServerError)
	}

	out := logs.String()
	if !strings.Contains(out, "bilerek fırlatıldı") {
		t.Errorf("panik loglanmamış: %s", out)
	}
	if !strings.Contains(out, "HTTP Request") {
		t.Errorf("istek logu kaybolmuş (muhtemelen zincir sırası bozuk): %s", out)
	}
	if !strings.Contains(out, "status=500") {
		t.Errorf("istek logu status=500 içermiyor: %s", out)
	}
	if !strings.Contains(out, "path=/test/patlar") {
		t.Errorf("istek yolu loglanmamış: %s", out)
	}
}

func TestRouterServesHome(t *testing.T) {
	r, _ := newTestRouter(t)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody))

	if rec.Code != http.StatusOK {
		t.Fatalf("durum = %d, beklenen %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "Merhaba") {
		t.Errorf("anasayfa gövdesi beklenen içeriği taşımıyor")
	}
	// Güvenlik başlıkları tüm zincir boyunca korunmalı.
	if rec.Header().Get("Content-Security-Policy") == "" {
		t.Error("CSP başlığı eksik")
	}
}

func TestRouterNotFoundUsesPage(t *testing.T) {
	r, _ := newTestRouter(t)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/yok-boyle-sayfa", http.NoBody))

	if rec.Code != http.StatusNotFound {
		t.Errorf("durum = %d, beklenen %d", rec.Code, http.StatusNotFound)
	}
	if !strings.Contains(rec.Body.String(), "Sayfa bulunamadı") {
		t.Errorf("404 sayfası render edilmemiş")
	}
}

func TestRouterMethodNotAllowed(t *testing.T) {
	r, _ := newTestRouter(t)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/status", http.NoBody))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("durum = %d, beklenen %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if allow := rec.Header().Get("Allow"); !strings.Contains(allow, "GET") {
		t.Errorf("Allow başlığı GET içermiyor: %q", allow)
	}
}

func TestNewRouterFailsWithoutStaticDir(t *testing.T) {
	if _, err := newRouter(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)), "/var/tmp/ozkansen-yok-boyle-bir-dizin"); err == nil {
		t.Fatal("statik dizin yokken newRouter hata vermeli")
	}
}
