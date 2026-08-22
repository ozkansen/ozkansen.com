---
layer: repository
dependencies: [WebServer, LoggingMiddleware, HTTPUtil, StaticAssets]
last_updated: 2026-08-22
---

# Improvements

**Özet:** Bu node, [[Security_Audit]] denetiminden çıkan bulguların önceliklendirilmiş çözüm backlog'udur. Her madde, ilgili modül sayfasına köşeli parantez linkiyle bağlanır ve wiki seviyesinde somut uygulama önerisi içerir; kod, madde tek tek onaylanıp uygulandığında değiştirilir (AUDIT operasyonu kodu değiştirmez). Öncelikler P1 (önce) → P3 (sonra) şeklinde sıralanmıştır.

**Kütüphaneler/Standartlar:** Go standard library (`net/http`, `os/signal`, `context`, `embed`), OWASP secure headers rehberi.

**Bağlantılar:** [[Security_Audit]] · [[WebServer]] · [[LoggingMiddleware]] · [[StaticAssets]] · [[Index]]

**Dosyalar:**
- Etkilenecek dosyalar: `cmd/web/main.go`, `internal/middleware/logging.go`, `static/`

## Geniş açıklama

### Backlog görünümü

```mermaid
flowchart TD
    subgraph P1["P1 — Üretim Hazırlığı"]
        I1["I1: Graceful shutdown<br/>→ WebServer"]
    end
    subgraph P2["P2 — Sertleştirme"]
        I2["I2: Security headers middleware<br/>→ WebServer"]
        I3["I3: /api/status method guard<br/>→ WebServer"]
        I4["I4: go:embed static<br/>→ WebServer + StaticAssets"]
    end
    subgraph P3["P3 — Hijyen"]
        I5["I5: Log sanitization<br/>→ LoggingMiddleware"]
        I6["I6: .DS_Store ignore + artifact temizliği"]
        I7["I7: apiStatusHandler basitleştirme"]
    end
```

### I1 — Graceful shutdown `[HIGH]` → [[WebServer]]

`ListenAndServe` bloğu sinyal yönetimiyle sarılmalı:

```go
srv := &http.Server{ /* mevcut ayarlar */ }

ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

go func() {
    <-ctx.Done()
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    if err := srv.Shutdown(shutdownCtx); err != nil {
        logger.Error("Kapanma hatası", slog.String("error", err.Error()))
    }
}()

if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
    logger.Error("Sunucu başlatılamadı", slog.String("error", err.Error()))
    os.Exit(1)
}
logger.Info("Sunucu düzgün kapatıldı")
```

**Etki:** Deploy/restart sırasında in-flight istekler tamamlanır; Docker/K8s `SIGTERM` akışıyla uyumlu hale gelir.

### I2 — Security headers middleware `[MEDIUM]` → [[WebServer]]

Yeni bir küçük middleware ([[LoggingMiddleware]] kalıbında):

```go
func SecureHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        h := w.Header()
        h.Set("X-Content-Type-Options", "nosniff")
        h.Set("X-Frame-Options", "DENY")
        h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
        h.Set("Content-Security-Policy",
            "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:")
        next.ServeHTTP(w, r)
    })
}
```

Zincirleme: `SecureHeaders(middleware.Logger(logger)(mux))`. Not: CSP'de inline script kullanılmadığından (`layout.Base` yalnızca yerel dosya referansları içerir) `'unsafe-inline'` gerekmez; HTMX `hx-*` attribute'ları CSP ile çelişmez.

### I3 — `/api/status` metot guard'ı `[MEDIUM]` → [[WebServer]] — ✅ UYGULANDI (2026-08-22)

**Önemli ders:** İlk önerilen `mux.HandleFunc("GET /api/status", ...)` metot deseni tek başına **yetersizdi**: catch-all `/` route'u mevcutken ServeMux en spesifik *eşleşen* deseni seçer; POST istekleri `GET /api/status` ile eşleşmez ve `/`'e düşerek 200 dönerdi (405 yalnızca hiçbir desen eşleşmezken üretilir). Bu davranış httptest deneyiyle doğrulandı.

**Uygulanan nihai çözüm:** Düz path deseni (`"/api/status"`) + handler içinde metot guard'ı — catch-all'un varlığından bağımsız olarak çalışır:

```go
func apiStatusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		httputil.WriteHTML(w, http.StatusOK, statusFragment)
	}
}
```

Doğrulama: GET → 200 + fragment; POST/DELETE/OPTIONS → `405 Method Not Allowed` + `Allow: GET` header.

### I4 — Statik varlıkları gömme (`go:embed`) `[MEDIUM]` → [[StaticAssets]]

CWD bağımlılığını kökten çözer ve dağıtımı tek-binary yapar:

```go
//go:embed static
var staticFS embed.FS

sub, _ := fs.Sub(staticFS, "static")
mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(sub))))
```

Not: Embed yaklaşımı dizin listelemesini de doğal olarak kapatmaz; S3 için ayrıca özel bir handler veya listeleme kapatan sarmalayıcı gerekir. Alternatif (embed istenmezse): statik kök yolunu `-static` flag'i/env ile konfigüre et. Karar bu madde uygulandığında verilmelidir.

### I5 — Log sanitization `[LOW]` → [[LoggingMiddleware]]

User-Agent/path değerleri loglamadan önce kontrol karakterlerinden arındırılmalı:

```go
func sanitize(s string) string {
    return strings.Map(func(r rune) rune {
        if r < 0x20 || r == 0x7f { // kontrol karakterleri
            return -1
        }
        return r
    }, s)
}
```

Ayrıca S5 (IP/PII): kamuya açık dağıtımda IP'nin son octet'inin maskelenmesi ya da log retention süresinin belgelenmesi önerilir.

### I6 — Repo hijyeni `[LOW]` — ✅ KAPANDI (2026-08-22)

- [x] `.gitignore`'a `.DS_Store` eklendi.
- [x] Minified `styles.css` artifact'i karar sahibi tarafından commitlendi.

### I7 — `apiStatusHandler` basitleştirme `[LOW]` → [[WebServer]]

Fabrika fonksiyonu yerine doğrudan tanım yeterli (fragment sabit olduğundan state gerekmiyor):

```go
var apiStatusHandler = func(w http.ResponseWriter, _ *http.Request) {
    httputil.WriteHTML(w, http.StatusOK, statusFragment)
}
```

### Süreç notu

- Bu backlog'daki maddeler uygulandıkça ilgili modül sayfalarının `security_risk`/`tech_debt` alanları boşaltılır ve [[Security_Audit]] bulgu tabloları güncellenir.
- Yeni middleware eklenirse INGEST ile wiki'ye yeni node açılmalıdır.
