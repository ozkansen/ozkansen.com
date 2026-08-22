---
layer: delivery
dependencies: [TemplViews, WebServer]
security_risk: [directory-listing-enabled]
tech_debt: []
last_updated: 2026-08-22
---

# StaticAssets

**Özet:** `static/` dizini tarayıcıya gönderilen statik varlıkları barındırır: Tailwind CSS ile derlenen stil dosyası ve iki istemci JS kütüphanesi (HTMX ve Alpine.js). CSS, Tailwind v4 `@source` direktifleriyle `.templ` ve `.go` kaynaklarını tarayarak sınıfları on-demand üretir; JS dosyaları CDN'den indirilmiş, **versiyonlu dosya adlarıyla** saklanan minified kopyalardır.

**Kütüphaneler:** Tailwind CSS v4, HTMX, Alpine.js.

**Bağlantılar:** [[TemplViews]] · [[WebServer]] · [[DevTooling]] · [[Index]]

**Dosyalar:**
- `static/css/input.css` — Tailwind girişi ve `@source` tarama direktifleri
- `static/css/styles.css` — üretilen (derlenmiş) çıktı
- `static/js/htmx_2.0.10.min.js` — HTMX 2.0.10 minified kopyası
- `static/js/alpinejs_3.16.2.min.js` — Alpine.js 3.16.2 minified kopyası

## Geniş açıklama

### CSS akışı (Tailwind v4)

```mermaid
flowchart LR
    Input["static/css/input.css"] -->|"npx tailwindcss"| Scan["@source taraması .templ + .go"]
    Scan --> Build[Kullanılan sınıfların derlenmesi]
    Build --> Output["static/css/styles.css (çıktı)"]
    Output -->|"<link>"| Base["layout.Base head (TemplViews)"]
```

- `input.css` satır 1'de `@import "tailwindcss";` ile framework'ü yükler.
- `@source "../../**/*.templ"` ve `@source "../../**/*.go"` direktifleri, kök dizindeki template ve Go dosyalarını tarar; böylece yalnızca kodda gerçekten kullanılan utility sınıfları çıktıya dahil edilir (on-demand derleme).
- `styles.css` üretilmiş dosyadır; `Makefile`'daki `build` hedefi `--minify` ile üretir (bkz. [[DevTooling]]).

### JS varlıkları

- `static/js/htmx_2.0.10.min.js` → [[TemplViews]]'deki HTMX kartı için `hx-*` davranışlarını sağlar.
- `static/js/alpinejs_3.16.2.min.js` → Alpine.js dosyası; `x-data`, `@click`, `x-show` direktiflerini çalıştırır.
- Her ikisi `defer` ile `layout.Base` head bölümünden yüklenir; ayrı CDN çağrısı yoktur (offline geliştirme için yerel kopya).
- **Versiyonlu adlandırma:** Dosya adları sürüm bilgisini taşır (`htmx_2.0.10`, `alpinejs_3.16.2`); sürüm yükseltmesi yeni bir dosyanın eklenip eskinin silinmesi ve [[TemplViews]]'deki `src` referansının güncellenmesiyle yapılır (eski `cdn.min.js`/`htmx.min.js` adları bu şekilde kaldırıldı). Cache-busting da bu sayede dosya adından çözülür.

### Servis

[[WebServer]] bu dizini `http.FileServer(http.Dir("./static"))` + `StripPrefix("/static/")` ile servis eder; tüm `href`/`src` referansları `/static/...` prefix'ini kullanır.

### Önemli noktalar

- `input.css` **kaynak**, `styles.css` **derlenmiş ürün**dür; sınıf eklemek için `.templ` dosyalarına yazmak yeterlidir (Tailwind taraması otomatik).
- Tailwind CLI sürümü `package.json`/`package-lock.json` ile sabitlenmiştir (reproducible install; bkz. [[DevTooling]]). Güncel derlemelerde çıktı başlığı `tailwindcss v4.3.3` olarak damgalanır.
- Statik dosyalar `.air.toml` içinde `exclude_dir` listesindedir; yani hot-reload tetiklemezler (bkz. [[DevTooling]]).
- Production build'de `static/` dizini `dist/static` altına kopyalanır ve `input.css` kopyadan çıkarılır (bkz. [[DevTooling]] → `make build`).
