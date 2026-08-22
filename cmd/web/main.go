package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/a-h/templ"
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

	mux := http.NewServeMux()

	// 1. Statik Dosyalar
	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// 2. Sayfa ve API Route'ları
	mux.Handle("/", templ.Handler(pages.Home("Özkan")))

	mux.HandleFunc("/api/status", apiStatusHandler())

	// Middleware zincirini uygula
	loggingMiddleware := middleware.Logger(logger)
	handlerWithLogging := loggingMiddleware(mux)

	logger.Info("Sunucu başlatılıyor", slog.String("port", ":8080"))
	server := &http.Server{
		Addr:              ":8080",
		Handler:           handlerWithLogging,
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
// statusFragment HTML parçasını döndürür.
func apiStatusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		httputil.WriteHTML(w, http.StatusOK, statusFragment)
	}
}
