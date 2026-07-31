---
layer: delivery
dependencies: [LoggingMiddleware, TemplViews, StaticAssets]
last_updated: 2026-07-31
---

# WebServer

**Özet:** `cmd/web/main.go` uygulamanın tek giriş noktasıdır (delivery katmanı). `slog` üzerine renkli text handler kurar, `http.ServeMux` ile route'ları tanımlar ve tüm istekleri logging middleware zincirinden geçirir. `:8080` portunda bloklanarak HTTP sunucusunu ayağa kaldırır.

**Kütüphaneler:** Go standard library (`net/http`, `log/slog`, `log`), `github.com/lmittmann/tint` (renkli text handler), `github.com/a-h/templ` (templ component handler).

**Bağlantılar:** [[LoggingMiddleware]] · [[TemplViews]] · [[StaticAssets]] · [[Index]]

**Dosyalar:**
- `cmd/web/main.go`

## Geniş açıklama

Uygulama, kişisel bir web sitesini sunan minimal bir Go web sunucusudur. Katman mantığı: giriş noktası → statik dosya sunucusu + route tanımları → middleware sarmalayıcı.

### Başlatma sırası

1. **Logger kurulumu:** `slog` default logger olarak `tint.NewTextHandler(os.Stdout, ...)` ile atanır. `LevelDebug` seviyesi ve `time.Kitchen` zaman formatı kullanılır; bu sayede geliştirme ortamında renkli, okunabilir loglar üretilir.
2. **Router:** `http.NewServeMux()` ile çoklayıcı oluşturulur.
3. **Statik dosyalar:** `./static` dizini `/static/` prefix'i altında servis edilir (`http.StripPrefix` ile prefix kırpılır).
4. **Sayfa route'u:** `/` → `templ.Handler(pages.Home("Özkan"))` ile anasayfa render edilir.
5. **API route'u:** `/api/status` → HTMX istekleri için sadece bir HTML parçası (fragment) döner; tam sayfa render edilmez.
6. **Middleware zinciri:** `middleware.Logger(logger)` ile sarmalanan mux, `http.ListenAndServe(":8080", ...)` ile dinlenir.

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
```

### Önemli noktalar

- `/api/status` **fragment** döner (`div#status-box`), tam HTML değil. Bu, HTMX'in `hx-swap="outerHTML"` akışıyla uyumludur ve sayfa yenilenmeden güncelleme sağlar.
- Route tanımları tüm istekleri kapsamaz: `/projects` linki header'da bulunur (bkz. [[TemplViews]]) ancak henüz bir handler tanımlanmamıştır, bu yüzden 404 döner.
- Hata durumunda `log.Fatal(err)` ile süreç sonlanır.
- `Content-Type: text/html` header'ı API fragment için açıkça set edilir.
