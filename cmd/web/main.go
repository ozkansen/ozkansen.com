package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"

	"ozkansen.com/internal/assets"
	"ozkansen.com/internal/httputil"
	"ozkansen.com/internal/middleware"
	"ozkansen.com/internal/views/pages"
)

// shutdownTimeout, kapanma sinyali alındıktan sonra uçuştaki isteklerin
// tamamlanması için tanınan süre. Docker/K8s'in varsayılan SIGTERM -> SIGKILL
// süresi 10 saniyedir; graceful kapanma şansı vermek için bu değer ondan
// küçük seçilmelidir, aksi halde süreç kapanmadan zorla öldürülür.
const shutdownTimeout = 10 * time.Second

func main() {
	// APP_ENV=production ile JSON + Info seviyesine geçilir; aksi halde
	// renkli metin + Debug seviyesi kullanılır.
	logger := newLogger(os.Getenv("APP_ENV"), os.Stdout)
	slog.SetDefault(logger)

	r := chi.NewRouter()

	// Middleware zinciri: stdlib uyumlu imza (func(http.Handler) http.Handler) doğrudan Use ile takılır.
	// Not: chi'de tüm Use çağrıları route kayıtlarından ÖNCE yapılmalıdır.
	r.Use(middleware.SecureHeaders) // en dış: hata yanıtları dahil her yanıt korumalı
	r.Use(middleware.Logger(logger))

	// 1. Statik Dosyalar
	// Varlık kökü STATIC_DIR ile verilebilir; verilmezse çalışma dizinindeki
	// ./static kullanılır. Erişilemiyorsa sunucu hiç başlamaz: sessiz 404'ler
	// üretmek, hatayı başlangıçta göstermekten daha kötüdür.
	staticHandler, err := assets.New(os.Getenv("STATIC_DIR"))
	if err != nil {
		logger.Error("Statik varlıklar yüklenemedi", slog.String("error", err.Error()))
		os.Exit(1)
	}
	r.Handle("/static/*", http.StripPrefix("/static/", staticHandler))

	// 2. Sayfa ve API Route'ları
	// chi'de metot-kayıtlı route'lar diğer metotlara otomatik 405 + Allow döner;
	// bilinmeyen yollar NotFound handler'a (404) gider — catch-all önceliği tuzağı yoktur.
	r.Get("/", templ.Handler(pages.Home("Özkan")).ServeHTTP)
	r.Get("/api/status", apiStatusHandler)
	r.NotFound(templ.Handler(pages.NotFound(), templ.WithStatus(http.StatusNotFound)).ServeHTTP)

	logger.Info("Sunucu başlatılıyor", slog.String("port", ":8080"))
	server := &http.Server{
		Addr:              ":8080",
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// SIGINT (Ctrl-C) ve SIGTERM (docker stop, K8s) için graceful shutdown.
	// stop() burada defer'lenmez: os.Exit(1) yolunda defer'ler çalışmaz, kaynaklar
	// her iki çıkış yolunda da açıkça bırakılır.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	// Shutdown, dinleyiciyi kapatıp ListenAndServe'i hemen serbest bırakır;
	// drain sonra sürer. Bu kanal olmadan "düzgün kapatıldı" logu drain bitmeden
	// kaybolur.
	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)

		<-ctx.Done()
		stop() // varsayılan sinyal davranışı: ikinci sinyal zorlamayla sonlandırır

		logger.Info("Kapanma sinyali alındı, uçuştaki istekler bekleniyor",
			slog.Duration("timeout", shutdownTimeout))

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("Kapanma hatası, bekleyen bağlantılar tamamlanmadı",
				slog.String("error", err.Error()))
			return
		}
		logger.Info("Sunucu düzgün kapatıldı")
	}()

	// Shutdown normal kapanmayı da ErrServerClosed ile bildirir; bu bir hata değil.
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("Sunucu başlatılamadı", slog.String("error", err.Error()))
		stop()
		os.Exit(1)
	}

	<-shutdownDone
}

// statusFragment, /api/status uç noktasının HTMX fragment yanıtıdır.
const statusFragment = `<div id="status-box" class="p-4 bg-emerald-950/60 text-emerald-300 rounded-lg">🚀 Sunucu Aktif!</div>`

// apiStatusHandler, HTMX istekleri için tam sayfa yerine yalnızca
// statusFragment HTML parçasını döndürür. Route `r.Get` ile kayıtlıdır;
// diğer metotlara chi router otomatik olarak 405 Method Not Allowed + Allow döner.
func apiStatusHandler(w http.ResponseWriter, _ *http.Request) {
	httputil.WriteHTML(w, http.StatusOK, statusFragment)
}
