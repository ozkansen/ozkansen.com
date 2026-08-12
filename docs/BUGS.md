# BUGS — Kod İnceleme Raporu

> Son güncelleme: 2026-08-03
> Kapsam: Tüm kaynak kod (`cmd/`, `internal/`, `Makefile`, `.air.toml`, `.golangci.yml`, `package.json`, templ dosyaları)

## 1. Özet

| Öncelik | Sayı | Başlık |
|---|---|---|
| 🔴 Yüksek | 3 | `package.json` git'te yok, `WriteHeader` state bozuyor, Flusher/Hijacker eksik |
| 🟠 Orta | 6 | Panic recovery yok, graceful shutdown yok, güvenlik header'ları yok, `log_path` query sızıyor, statik cache yok, Write hata yönetimi |
| 🟡 Düşük | 7 | Dead code, port tekrarı, remote_ip port, sürüm pinleme, fmt tutarsızlığı, README boş, templ import |

**Toplam: 16 madde** (4 hata, 3 optimizasyon, 9 iyileştirme)

---

## 2. Hatalar (Bugs)

### 🔴 BUG-1: `package.json` ve `package-lock.json` sürüm kontrolünde değil

**Konum:** `.gitignore:3-4`

```gitignore
node_modules/
package.json        # ← yanlış
package-lock.json   # ← yanlış
```

**Sorun:** `package.json` ignore edilmiş. Depoyu klonlayan biri `npm install` çalıştıramaz, Tailwind build edemez (`make install` ve `make dev` bozulur). `package-lock.json` da ignore edildiği için bağımlılık sürümleri tekrarlanabilir (reproducible) değil.

**Kanıt:**
```bash
$ git ls-files package.json package-lock.json
# (boş — track edilmiyor)

$ git check-ignore -v package.json
.gitignore:3:package.json	package.json
```

**Çözüm:** `.gitignore`'dan `package.json` ve `package-lock.json` satırlarını kaldır; `node_modules/` kalsın:

```gitignore
node_modules/
```

Sonra `git add package.json package-lock.json && git commit`.

---

### 🔴 BUG-2: `responseWriter.WriteHeader` çift çağrıda yanlış durum kodu logluyor

**Konum:** `internal/middleware/logging.go:16-19`

```go
func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code        // ← her çağrıda ezilir
	rw.ResponseWriter.WriteHeader(code)
}
```

**Sorun:** Bir handler `WriteHeader(200)` sonra koşullu olarak `WriteHeader(500)` çağırırsa, net/http ikinci çağrıyı **yok sayar** (gerçek yanıt 200 kalır) ama wrapper `rw.status`'u 500'e ezmiş olur. Log, gerçek istemciye giden durum koduyla uyuşmaz → yanlış log, yanlış metrik.

**Kanıt senaryosu:**
```
1. Handler: WriteHeader(200) → rw.status=200, underlying 200
2. Handler: WriteHeader(500) → rw.status=500, underlying IGNORED (net/http 200 kalır)
3. Logger: rw.status=500 loglanır → GERÇEK: 200, LOG: 500 ❌
```

**Çözüm:** İlk yazımı koruyan guard ekle:

```go
func (rw *responseWriter) WriteHeader(code int) {
	if rw.status != 0 {
		return
	}
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}
```

Bu düzeltme OPT-2'deki dead code sorununu da çözer (aşağıya bakın).

---

### 🔴 BUG-3: `responseWriter` `http.Flusher` / `http.Hijacker` / `http.Pusher` interface'lerini forward etmiyor

**Konum:** `internal/middleware/logging.go:10-29`

**Sorun:** Wrapper yalnızca `http.ResponseWriter` arayüzünü karşılıyor. net/http bağlantıyı yönetmek için `Flush()` çağırır; handler `http.Hijacker` ile soketi devralabilir; HTTP/2 için `Pusher` gerekebilir.

**Neden kritik:**
- `http.Flusher` eksikliği: net/http, yanıt gövdesini istemciye göndermek için `Flush()` çağırır. Wrapper Flusher interface'ini implement etmiyorsa, type assertion başarısız olur ve flush **sessizce başarısız** olur → veri tamponda kalır, bağlantı zaman aşımına uğrayabilir.
- `http.Hijacker` eksikliği: WebSocket upgrade veya long-polling handler'ları `Hijacker` ile soketi devralmak ister → panic riski.
- `http.Pusher` eksikliği: HTTP/2 server push mümkün değil.

**Çözüm:**

```go
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, fmt.Errorf("underlying ResponseWriter does not implement http.Hijacker")
}
```

---

### 🟠 BUG-4: `/api/status` Write hatası HTTP durum koduna yansımıyor

**Konum:** `cmd/web/main.go:36-41`

```go
mux.HandleFunc("/api/status", func(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	if _, err := w.Write([]byte(`...`)); err != nil {
		logger.Error("Status yanıtı yazılamadı", slog.String("error", err.Error()))
		// ← durum kodu hâlâ 200, ama Write başarısız oldu
	}
})
```

**Sorun:** `w.Write()` hatası genelde bağlantı koptuğunda oluşur. Ancak handler'ın mantığı: "önce yaz, sonra hata kontrol et" → Write başarılıysa zaten 200 commit edilmiş olur, başarısızsa hata fırlatılmıştır ama 200 dönüştür. Pratikte küçük bir sorundur ama **tutarlılık** açısından WriteHeader hatası gövde yazmadan önce kontrol edilmeli.

**İş akışı sorunu:**
```
1. w.Header().Set("Content-Type", "text/html") → header hâlâ ayarlanabilir
2. w.Write(body) → Go, header'ı otomatik flush eder (200 OK)
3. w.WriteHeader(500) → IGNORED (zaten commit edildi)
```

**Çözüm:**

```go
mux.HandleFunc("/api/status", func(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	if _, err := w.Write([]byte(`...`)); err != nil {
		logger.Error("Status yanıtı yazılamadı", slog.String("error", err.Error()))
	}
})
```

Not: Durum kodu zaten 200 olarak commit edildi; `WriteHeader(500)` eklemek anlamsızdır. Mantıklı çözüm: hata loglanır, istemci zaten bağlantıyı kaybetmiştir. Mevcut kod yeterlidir; buradaki sorun **davranışsal** değil, **tasarımsal** — handler'ın error handling stratejisi net değil.

---

## 3. Optimizasyonlar

### 🟠 OPT-1: Statik dosyalarda HTTP cache yok

**Konum:** `cmd/web/main.go:30-31`

**Sorun:** `styles.css`, `htmx.min.js`, `cdn.min.js` her istekte yeniden indirilir. İçerik sabit dosyalar olduğu için `Cache-Control` header'ı eklemek gereksiz bant genişliği ve gecikme önler.

**Çözüm:** Cache middleware'i:

```go
func cacheControl(maxAge int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))
			next.ServeHTTP(w, r)
		})
	}
}

// Kullanım:
mux.Handle("/static/",
	http.StripPrefix("/static/",
		cacheControl(86400)(http.FileServer(http.Dir("./static")))))
```

Not: `immutable` kullanmak için dosya adlarına content-hash eklenmeli (örn. `styles.abc123.css`). Şu an için `max-age=86400` (1 gün) yeterlidir.

---

### 🟡 OPT-2: `rw.status == 0` kontrolü ölü kod

**Konum:** `internal/middleware/logging.go:23-25`

```go
func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.status == 0 {           // ← hiç çalışmaz (status=200 ile başlatılıyor)
		rw.status = http.StatusOK
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}
```

**Sorun:** `responseWriter` struct literal'inde `status: http.StatusOK` verildiği için (`logging.go:39`) `Write` içindeki `if rw.status == 0` dalı hiç çalışmaz. Ölü kod.

**Çözüm:** BUG-2 düzeltmesiyle birlikte çözülür. `WriteHeader`'a guard eklendiğinde, `status` hiç 0 olmayacağı için bu satır gerçekten ölü kod olur. Seçenekler:

1. **Önerilen:** `Write`'taki guard'ı kaldır (zaten `WriteHeader` guard ile korunuyor):

```go
func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}
```

2. Alternatif: Initializer'dan `status`'u kaldır, `0` ile başlat ve `Write`'taki guard'a güven.

---

### 🟡 OPT-3: Port değeri iki kez hard-code edilmiş

**Konum:** `cmd/web/main.go:47,49`

```go
logger.Info("Sunucu başlatılıyor", slog.String("port", ":8080"))  // burada
server := &http.Server{
	Addr: ":8080",  // ve burada
```

**Sorun:** `":8080"` iki yerde yazılı; port değişince iki yer düzenlenmeli → unutma riski.

**Çözüm:**

```go
const listenAddr = ":8080"

logger.Info("Sunucu başlatılıyor", slog.String("addr", listenAddr))
server := &http.Server{
	Addr: listenAddr,
	...
}
```

---

## 4. İyileştirmeler

### 🟠 IMP-1: Graceful shutdown yok

**Konum:** `cmd/web/main.go:56-58`

```go
if err := server.ListenAndServe(); err != nil {
	log.Fatal(err)
}
```

**Sorun:** SIGINT/SIGTERM'de sunucu anında kapanır, sürmekte olan istekler kesilir. Ayrıca `ListenAndServe` normal kapanışta `http.ErrServerClosed` döner; `log.Fatal` bunu da hata olarak loglar.

**Çözüm:**

```go
import (
	"context"
	"os/signal"
	"syscall"
)

func main() {
	// ... mevcut kod ...

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	logger.Info("Sunucu kapatılıyor...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatal(err)
	}
	logger.Info("Sunucu başarıyla kapatıldı")
}
```

---

### 🟠 IMP-2: Panic recovery middleware yok

**Konum:** `cmd/web/main.go:43-45` (middleware zinciri)

**Sorun:** Bir handler panic fırlatırsa net/http bağlantıyı kapatır ama yapılandırılmış log üretilmez; stack trace yalnızca stderr'e gider. Üretimde 500 yanıtı ve `slog` logu beklenir.

**Çözüm:** Yeni `internal/middleware/recover.go`:

```go
package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
)

func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					stack := debug.Stack()
					logger.Error("panic recovered",
						slog.Any("panic", rec),
						slog.String("path", r.URL.Path),
						slog.String("method", r.Method),
						slog.String("stack", string(stack)),
					)
					http.Error(w,
						fmt.Sprintf("Internal Server Error: %v", rec),
						http.StatusInternalServerError,
					)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
```

Zincirde Logger'ın **içine** yerleştirilmeli:

```go
recoveryMiddleware := middleware.Recover(logger)
loggingMiddleware := middleware.Logger(logger)
handlerWithLogging := recoveryMiddleware(loggingMiddleware(mux))
```

---

### 🟠 IMP-3: Güvenlik header'ları eksik

**Konum:** `cmd/web/main.go` (hiçbir yerde yok)

**Sorun:** XSS/clickjacking/MIME-sniffing koruması için temel header'lar gönderilmiyor. HTMX + Alpine.js ekli bir site için bu eksiklik risk oluşturur.

**Çözüm:** Yeni `internal/middleware/security.go`:

```go
package middleware

import "net/http"

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("X-XSS-Protection", "0") // Modern tarayıcılar için devre dışı (artık önerilmiyor)
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'")
		next.ServeHTTP(w, r)
	})
}
```

---

### 🟠 IMP-4: Log path_raw olarak loglanıyor — hassas veri sızıntısı riski

**Konum:** `internal/middleware/logging.go:57`

```go
slog.String("path", r.URL.Path),
```

**Sorun:** `r.URL.Path` query string'leri içermiyor, ancak `r.URL.RequestURI()` tüm query string'i içerir. Şu an path query'siz loglanıyor — iyi. Ancak handler'da query parametreleri loglanmıyorsa, hata ayıklama zorlaşır. Daha da önemlisi, gelecekte birisi `r.URL.RequestURI()` eklerse hassas veri (token, email vb.) loga sızabilir.

**Çözüm:** Path'i loglamaya devam et, ama query string'i **asla** loglama. Gerekirse `r.URL.Query()`'den belirli anahtarları logla:

```go
slog.String("path", r.URL.Path),
// Asla: slog.String("query", r.URL.RawQuery)  ← hassas veri sızdırır
```

Bu bir best practice notudur; mevcut kod zaten doğru olanı yapıyor.

---

### 🟡 IMP-5: `remote_ip` alanı port içeriyor

**Konum:** `internal/middleware/logging.go:61`

```go
slog.String("remote_ip", r.RemoteAddr),
```

**Sorun:** `r.RemoteAddr` → `"127.0.0.1:50321"` formatındadır (IP:port). Log analizinde port gereksiz bilgi kirliliği yaratır. Ayrıca reverse-proxy (nginx, Caddy) arkasında gerçek istemci adresi `X-Forwarded-For` veya `X-Real-IP` header'ında bulunur.

**Çözüm:**

```go
import "net"

func extractIP(remoteAddr string) string {
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		return host
	}
	return remoteAddr
}

// Logger içinde:
slog.String("remote_ip", extractIP(r.RemoteAddr)),
```

Proxy varsa:

```go
func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return xff
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
```

---

### 🟡 IMP-6: `make install` sürüm pin'li değil

**Konum:** `Makefile:5-6`

```make
go install github.com/a-h/templ/cmd/templ@latest
go install github.com/air-verse/air@latest
```

**Sorun:** `go install ...@latest` ile templ ve air her kurulumda farklı sürüme yüklenebilir. `go.mod`'daki templ `v0.3.1020` pin'liyken CLI `@latest` kullanıyor.

**Çözüm:**

```make
go install github.com/a-h/templ/cmd/templ@v0.3.1020
go install github.com/air-verse/air@v1.61.5
```

---

### 🟡 IMP-7: `fmt` hedefi `gofumpt`/`gci` kapsamıyor

**Konum:** `Makefile:31-33`

```make
fmt:
	gofmt -w .
	templ fmt .
```

**Sorun:** `fmt` sadece `gofmt` + `templ fmt` çalıştırır; `.golangci.yml`'de tanımlı `gofumpt` ve `gci` formatter'ları yalnızca `lint-fix`'te devreye girer. İki formatlama davranışı tutarsız.

**Çözüm:**

```make
fmt:
	golangci-lint fmt .
	templ fmt .
```

---

### 🟡 IMP-8: Alpine.js dosyası belirsiz adla saklanıyor

**Konum:** `static/js/cdn.min.js`, `internal/views/layout/base.templ:13`

**Sorun:** `cdn.min.js` dosya adı Alpine.js olduğunu belli etmiyor; sürümü de belgelenmemiş. Bakımı zorlaştırır.

**Çözüm:** `alpine.min.js` olarak yeniden adlandır, `base.templ` referansını güncelle:

```html
<script src="/static/js/alpine.min.js" defer></script>
```

---

### 🟡 IMP-9: Test ve CI yok

**Sorun:** Projede hiç test dosyası (`_test.go`) ve CI pipeline'ı yok.

**Çözüm:**
- `internal/middleware/logging_test.go`: `httptest.NewRecorder` ile WriteHeader/Write/splitIP senaryoları.
- `internal/middleware/logging_test.go`:

```go
func TestLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := middleware.Logger(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rec.Code)
	}
}
```

- GitHub Actions `.github/workflows/ci.yml`:

```yaml
name: CI
on: [push, pull_request]
jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - uses: golangci/golangci-lint-action@v6
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: go test ./...
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: go build ./...
```

---

### 🟡 IMP-10: Statik yol göreceli (`./static`)

**Konum:** `cmd/web/main.go:30`

```go
fs := http.FileServer(http.Dir("./static"))
```

**Sorun:** `http.Dir("./static")` çalışma dizinine (cwd) bağlı; farklı dizinden başlatılırsa 404/panic riski.

**Çözüm:** `embed` kullanarak binary'ye gömün:

```go
//go:embed all:static
var staticFS embed.FS

// Kullanım:
fs := http.FileServer(http.FS(staticFS))
mux.Handle("/static/", http.StripPrefix("/static/", fs))
```

Alternatif olarak flag/env ile yapılandırma:

```go
staticDir := flag.String("static-dir", "./static", "Statik dosya dizini")
flag.Parse()
fs := http.FileServer(http.Dir(*staticDir))
```

---

### 🟡 IMP-11: `/favicon.ico` ve dizin listeleme

**Sorun:** Tarayıcı her yüklemede `/favicon.ico` ister → 404 log gürültüsü. `http.FileServer` dizinlerde index dosyası yoksa **dizin listeleme** döndürür (örn. `/static/js/` erişilirse).

**Çözüm:** Basit bir favicon ekleyin ve statik servisi sınırlayın:

```go
// favicon
mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./static/favicon.ico")
})

// dizin listeleme engelleme (isteğe bağlı)
mux.Handle("/static/",
	http.StripPrefix("/static/",
		http.FileServer(noDirList{http.Dir("./static")})))
```

---

### 🟡 IMP-12: README boş

**Konum:** `README.md`

**Sorun:** Depoyu klonlayan biri projeyi nasıl çalıştıracağını, bağımlılıkları ve yapısını bilemez.

**Çözüm:** Temel README:

```markdown
# ozkansen.com

Kişisel web sitesi — Go + Templ + HTMX + Tailwind CSS

## Hızlı Başlangıç

```bash
make install
make dev
# → http://localhost:8080
```

## Üretim

```bash
make build
./bin/main
```

## Teknoloji Yığını

- **Backend:** Go 1.26, `net/http`, `log/slog` + `tint`
- **Template:** [Templ](https://templ.guide/) (type-safe HTML)
- **Frontend:** HTMX + Alpine.js + Tailwind CSS v4
- **Lint:** golangci-lint v2

## Geliştirme

- `make dev` — templ, tailwind ve air izleyicilerini paralel başlatır
- `make lint` — golangci-lint taraması
- `make lint-fix` — otomatik düzeltme
- `make build` — production binary (`./bin/main`)
```

---

## 5. Önerilen Uygulama Sırası

1. **BUG-1** — `.gitignore` düzelt, `package.json` + `package-lock.json`'ı commit'le (proje klonlanabilirliği)
2. **BUG-2 + BUG-3** — `responseWriter` durum yönetimi + Flusher/Hijacker forward (gerçek runtime sorunları)
3. **IMP-1 + IMP-2** — graceful shutdown + panic recovery (üretim dayanıklılığı)
4. **IMP-3 + IMP-4** — güvenlik header'ları + log gizliliği
5. **OPT-1** — statik cache
6. Küçük temizlikler: OPT-2, OPT-3, IMP-5, IMP-6, IMP-7, IMP-8, IMP-9, IMP-10, IMP-11, IMP-12
7. **IMP-9** — testler + CI (sürekli kalite güvencesi)

---

## 6. Dosya Haritası

| Dosya | İlgili Maddeler |
|---|---|
| `cmd/web/main.go` | BUG-4, OPT-1, OPT-3, IMP-1, IMP-3, IMP-10, IMP-11 |
| `internal/middleware/logging.go` | BUG-2, BUG-3, OPT-2, IMP-4, IMP-5 |
| `.gitignore` | BUG-1 |
| `Makefile` | IMP-6, IMP-7 |
| `internal/views/layout/base.templ` | IMP-8 |
| `README.md` | IMP-12 |
| `package.json` | BUG-1 |
| `.golangci.yml` | — (güncel, sorun yok) |
| `.air.toml` | — (güncel, sorun yok) |
