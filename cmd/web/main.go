package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/a-h/templ"
	"github.com/lmittmann/tint"

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

	mux.HandleFunc("/api/status", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		if _, err := w.Write([]byte(`<div id="status-box" class="p-4 bg-emerald-950/60 text-emerald-300 rounded-lg">🚀 Sunucu Aktif!</div>`)); err != nil {
			logger.Error("Status yanıtı yazılamadı", slog.String("error", err.Error()))
		}
	})

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
		log.Fatal(err)
	}
}
