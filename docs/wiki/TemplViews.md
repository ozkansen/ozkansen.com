---
layer: delivery
dependencies: [WebServer, StaticAssets]
last_updated: 2026-07-31
---

# TemplViews

**Özet:** `internal/views` paketi, templ ile yazılmış sunucu taraflı view katmanıdır. `layout.Base` tüm sayfaların ortak HTML iskeletini (head, header/nav, main, footer) sağlar; `pages.Home` ise anasayfa içeriğini üretir. Home, istemci etkileşimi için Alpine.js (`x-data`) ve HTMX (`hx-get`) örneklerini barındırır.

**Kütüphaneler:** `github.com/a-h/templ` (type-safe HTML template engine), Tailwind CSS sınıfları.

**Bağlantılar:** [[WebServer]] · [[StaticAssets]] · [[Index]]

**Dosyalar:**
- `internal/views/layout/base.templ` (+ üretilen `base_templ.go`)
- `internal/views/pages/home.templ` (+ üretilen `home_templ.go`)

## Geniş açıklama

View katmanı iki pakete ayrılmıştır: `layout` (ortak iskelet) ve `pages` (sayfa içerikleri). `.templ` dosyaları kaynak koddur; `.templ.go` dosyaları `templ generate` ile üretilir ve **elle düzenlenmemelidir** (bkz. [[DevTooling]]).

### Component hiyerarşisi

```mermaid
flowchart TD
    Route["GET / → templ.Handler(pages.Home)"] --> Home[pages.Home]
    Home -->|"içerik / children"| Base[layout.Base]

    Base --> Head["<head>: meta, title, CSS, htmx.min.js, cdn.min.js (Alpine)"]
    Base --> Header["header: Personal Space logo + nav (Anasayfa / Projeler)"]
    Base --> Main["main: children slot"]
    Base --> Footer["footer: © 2026"]

    Home --> Hero[Hero: isim + tanıtım]
    Home --> AlpineCard["Alpine.js kartı: x-data open toggle"]
    Home --> HtmxCard["HTMX kartı: hx-get /api/status"]
    HtmxCard -->|"fragment"| API["/api/status (WebServer)"]
```

### layout.Base

- `Base(title string) templ.Component` → tek parametre alır, `<title>` etiketini besler.
- İçeriği `<main>` içindeki `children...` slot'una basar. Bu desen sayesinde her sayfa ortak iskeleti yeniden yazmaz.
- Footer ve header içinde sabit bağlantılar vardır: `/` (Anasayfa) ve `/projects` (Projeler). `/projects` için henüz route yok ([[WebServer]] notuna bakın).
- Statik bağımlılıklar: `/static/css/styles.css`, `/static/js/htmx.min.js`, `/static/js/cdn.min.js`.

### pages.Home

- `Home(name string) templ.Component` → hero bölümünde gösterilen adı alır.
- **Alpine.js örneği:** `x-data="{ open: false }"` ile tarayıcı içi durum tutulur; buton `@click` ile toggle eder, `x-show` + `x-transition` kutusu görünürlüğü yönetir. Sunucuya hiç istek atılmaz.
- **HTMX örneği:** Buton `hx-get="/api/status"`, `hx-target="#status-box"`, `hx-swap="outerHTML"` ile sunucudan sadece HTML fragmenti çeker; tam sayfa yenilenmez.

### Önemli noktalar

- Türkçe içerik ve `lang="tr"` ile site tamamen Türkçe.
- Tailwind sınıfları doğrudan template içinde yazılır; derleme sırasında [[StaticAssets]]'teki `input.css` `@source` direktifleri `.templ` dosyalarını tarar.
- Home, `@layout.Base(...)` çağrısı içinde kendi içeriğini bir closure component olarak ileterek render eder.
