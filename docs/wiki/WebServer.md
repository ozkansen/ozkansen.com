---
layer: delivery
dependencies: [LoggingMiddleware, TemplViews, StaticAssets, HTTPUtil]
security_risk: [security-headers-missing, api-method-not-restricted]
tech_debt: [graceful-shutdown-missing, cwd-relative-static-path, apistatus-handler-indirection]
last_updated: 2026-08-22
---

# WebServer

**Özet:** `cmd/web/main.go` uygulamanın tek giriş noktasıdır (delivery katmanı). `slog` üzerine renkli text handler kurar, `http.ServeMux` ile route'ları tanımlar ve tüm istekleri logging middleware zincirinden geçirir. `:8080` portunda timeout'ları tanımlı bir `http.Server` ile bloklanarak HTTP sunucusunu ayağa kaldırır.

**Kütüphaneler:** Go standard library (`net/http`, `log/slog`, `os`, `time`), `github.com/lmittmann/tint` (renkli text handler), `github.com/a-h/templ` (templ component handler), `ozkansen.com/internal/httputil`.

**Bağlantılar:** [[LoggingMiddleware]] · [[TemplViews]] · [[StaticAssets]] · [[HTTPUtil]] · [[Linting]] · [[Index]]

**Dosyalar:**
- `cmd/web/main.go`

## Geniş açıklama

Uygulama, kişisel bir web sitesini sunan minimal bir Go web sunucusudur. Katman mantığı: giriş noktası → statik dosya sunucusu + route tanımları → middleware sarmalayıcı.

### Başlatma sırası

1. **Logger kurulumu:** `slog` default logger olarak `tint.NewTextHandler(os.Stdout, ...)` ile atanır. `LevelDebug` seviyesi ve `time.Kitchen` zaman formatı kullanılır; bu sayede geliştirme ortamında renkli, okunabilir loglar üretilir.
2. **Router:** `http.NewServeMux()` ile çoklayıcı oluşturulur.
3. **Statik dosyalar:** `./static` dizini `/static/` prefix'i altında servis edilir (`http.StripPrefix` ile prefix kırpılır).
4. **Sayfa route'u:** `/` → `templ.Handler(pages.Home("Özkan"))` ile anasayfa render edilir.
5. **API route'u:** `/api/status` → HTMX istekleri için sadece bir HTML parçası (fragment) döner; tam sayfa render edilmez. Yanıt `httputil.WriteHTML(w, http.StatusOK, statusFragment)` ile yazılır (bkz. [[HTTPUtil]]).
6. **Middleware zinciri:** `middleware.Logger(logger)` ile sarmalanan mux, `http.Server` yapısıyla `ListenAndServe` edilir.
7. **Sunucu timeout'ları:** `ReadHeaderTimeout: 10s`, `ReadTimeout: 30s`, `WriteTimeout: 30s`, `IdleTimeout: 60s` — `http.Server` struct'ı üzerinden tanımlanır (gosec G114 uyarısını giderir; bağlantı tabanlı saldırılara karşı koruma sağlar).

### İstek akışı

```mermaid
flowchart LR
    Client[İstemci / HTMX / Tarayıcı] -->|"GET /"| Mux
    Client -->|"GET /static/..."| Mux
    Client -->|"GET /api/status"| Mux

    subgraph LoggerMW [Logging Middleware]
        RW[responseWriter sarmalayıcı] --> Next[Sonraki Handler]
    end

    Mux --> LoggerMW
    Next --> RouteMux["ServeMux (ana router)"]

    RouteMux -->|"route /"| HomeHandler["templ.Handler(pages.Home)"]
    RouteMux -->|"route /api/status"| APIHandler["/api/status HandlerFunc"]
    RouteMux -->|"route /static/"| FileServer[FileServer + StripPrefix]

    HomeHandler --> Base["layout.Base (html shell)"]
    HomeHandler --> HomeComp["pages.Home (içerik)"]
    Base --> Static["./static/css/styles.css + JS"]
    APIHandler --> HTMLUtil["httputil.WriteHTML"]
```

### Önemli noktalar

- `/api/status` **fragment** döner (`div#status-box`), tam HTML değil. Bu, HTMX'in `hx-swap="outerHTML"` akışıyla uyumludur ve sayfa yenilenmeden güncelleme sağlar.
- Route tanımları tüm istekleri kapsamaz: `/projects` linki header'da bulunur (bkz. [[TemplViews]]) ancak henüz bir handler tanımlanmamıştır, bu yüzden 404 döner.
- Hata durumunda `logger.Error(...)` ile loglanır ve `os.Exit(1)` ile süreç sonlanır (eski `log.Fatal` yaklaşımı yerine default slog logger kullanılır).
- `Content-Type: text/html; charset=utf-8` header'ı API fragment için [[HTTPUtil]] içindeki `WriteHTML` tarafından set edilir.
- Yanıt yazma hataları handler'da değil, `WriteHTML` içinde merkezî olarak loglanır; bu, [[Linting]]'deki `errcheck` kuralının gereğini de karşılar.
- `http.Server` yapısı, `ReadHeaderTimeout` gibi alanlarla slowloris tarzı saldırılara karşı sınır koyar. Zaman aşımı değerleri `main.go` içinde sabittir.
