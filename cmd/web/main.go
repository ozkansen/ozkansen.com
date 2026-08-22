package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
	"github.com/lmittmann/tint"

	"ozkansen.com/internal/httputil"
	"ozkansen.com/internal/middleware"
	"ozkansen.com/internal/views/pages"
)

func main() {
	// Geliştirme ortamı için Text, Canlı (Prod) ortamı için JSON handler tercih edilebilir
	logger := slog.New(tint.NewTextHandler(os.Stdout, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: time.Kitchen,
		NoColor:    false,
	}))

	slog.SetDefault(logger)

	r := chi.NewRouter()

	// Middleware zinciri: stdlib uyumlu imza (func(http.Handler) http.Handler) doğrudan Use ile takılır.
	// Not: chi'de tüm Use çağrıları route kayıtlarından ÖNCE yapılmalıdır.
	r.Use(middleware.Logger(logger))

	// 1. Statik Dosyalar
	fs := http.FileServer(http.Dir("./static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fs))

	// 2. Sayfa ve API Route'ları
	// chi'de metot-kayıtlı route'lar diğer metotlara otomatik 405 + Allow döner;
	// bilinmeyen yollar NotFound handler'a (404) gider — catch-all önceliği tuzağı yoktur.
	r.Get("/", templ.Handler(pages.Home("Özkan")).ServeHTTP)
	r.Get("/api/status", apiStatusHandler)

	logger.Info("Sunucu başlatılıyor", slog.String("port", ":8080"))
	server := &http.Server{
		Addr:              ":8080",
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		logger.Error("Sunucu başlatılamadı", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

// statusFragment, /api/status uç noktasının HTMX fragment yanıtıdır.
const statusFragment = `<div id="status-box" class="p-4 bg-emerald-950/60 text-emerald-300 rounded-lg">🚀 Sunucu Aktif!</div>`

// apiStatusHandler, HTMX istekleri için tam sayfa yerine yalnızca
// statusFragment HTML parçasını döndürür. Route `r.Get` ile kayıtlıdır;
// diğer metotlara chi router otomatik olarak 405 Method Not Allowed + Allow döner.
func apiStatusHandler(w http.ResponseWriter, _ *http.Request) {
	httputil.WriteHTML(w, http.StatusOK, statusFragment)
}
