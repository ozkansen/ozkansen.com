package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/a-h/templ"

	"ozkansen.com/internal/middleware"
	"ozkansen.com/internal/views/pages"
)

func main() {
	// Geliştirme ortamı için Text, Canlı (Prod) ortamı için JSON handler tercih edilebilir
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger) // Global logger olarak ayarla

	mux := http.NewServeMux()

	// 1. Statik Dosyalar
	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// 2. Sayfa ve API Route'ları
	mux.Handle("/", templ.Handler(pages.Home("Özkan")))

	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<div id="status-box" class="p-4 bg-emerald-950/60 text-emerald-300 rounded-lg">🚀 Sunucu Aktif!</div>`))
	})

	// Middleware zincirini uygula
	loggingMiddleware := middleware.Logger(logger)
	handlerWithLogging := loggingMiddleware(mux)

	logger.Info("Sunucu başlatılıyor", slog.String("port", ":8080"))
	if err := http.ListenAndServe(":8080", handlerWithLogging); err != nil {
		log.Fatal(err)
	}
}
