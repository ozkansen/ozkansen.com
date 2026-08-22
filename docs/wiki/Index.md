---
last_updated: 2026-08-22
---

# Index

**Özet:** Bu dosya projenin Bilgi Grafiği ana haritasıdır (Knowledge Graph Index). Tüm wiki node'larını listeler, katman dağılımını gösterir ve bir isteğin istemciden yanıta kadar izlediği akışı özetler. Yeni bir modül/fikir araştırmadan önce buradan başlayın.

## Mimari Harita

```mermaid
flowchart LR
    subgraph Delivery
        WebServer
        SecureHeaders
        LoggingMiddleware
        HTTPUtil
        TemplViews
        StaticAssets
    end

    subgraph Tooling
        DevTooling
        Linting
        Security_Audit
        Improvements
    end

    Client[İstemci] --> WebServer
    WebServer --> SecureHeaders
    WebServer --> LoggingMiddleware
    WebServer --> HTTPUtil
    WebServer --> TemplViews
    WebServer --> StaticAssets
    TemplViews --> StaticAssets
    DevTooling --> TemplViews
    DevTooling --> StaticAssets
    DevTooling --> WebServer
    DevTooling --> Linting
    Linting --> WebServer
    Linting --> TemplViews
    Security_Audit --> WebServer
    Security_Audit --> LoggingMiddleware
    Security_Audit --> StaticAssets
    Security_Audit --> Improvements
```

## Node Listesi

### Delivery Katmanı
- [[WebServer]] — giriş noktası, chi v5 router (`r.Get` metot kayıtları → otomatik 405), `http.Server` timeout'ları, `r.Use` middleware
- [[SecureHeaders]] — tüm yanıtlara güvenlik başlıkları (CSP + nosniff + DENY + Referrer-Policy); S1 çözümü
- [[LoggingMiddleware]] — istek/yanıt loglama, `responseWriter` sarmalayıcı (WriteHeader/WriteString/Flush/Hijack/Push/Unwrap ileri taşıma), seviye mantığı
- [[HTTPUtil]] — `httputil.WriteHTML`: Content-Type + durum kodu ile HTML yanıtı yazımı, commit sonrası hata loglama
- [[TemplViews]] — templ view katmanı: `layout.Base` iskeleti + `pages.Home` sayfası
- [[StaticAssets]] — Tailwind CSS (v4 `@source`) + versiyonlu HTMX/Alpine.js statik kopyaları

### Araçlar / Süreç
- [[DevTooling]] — `Makefile` (dev/build/lint) + `.air.toml` hot-reload pipeline
- [[Linting]] — golangci-lint v2 konfigürasyonu: linter seti, ayarlar, exclusions, gofumpt/gci formatter'ları
- [[Security_Audit]] — OWASP + performans denetim raporu (2026-08-22): bulgular seviyelere göre listeli, temiz alanlar belgeli
- [[Improvements]] — denetimden çıkan önceliklendirilmiş çözüm backlog'u (P1 graceful shutdown → P3 hijyen)

## Bağımlılık Grafiği

```mermaid
graph TD
    WebServer --> LoggingMiddleware
    WebServer --> HTTPUtil
    WebServer --> TemplViews
    WebServer --> StaticAssets
    TemplViews --> StaticAssets
    TemplViews --> WebServer
    StaticAssets --> TemplViews
    DevTooling --> Linting
    Linting --> WebServer
```

## Uçtan Uca İstek Akışı

```mermaid
sequenceDiagram
    participant C as Tarayıcı (HTMX/Alpine)
    participant MW as LoggingMiddleware
    participant M as ServeMux
    participant V as TemplViews
    participant U as HTTPUtil
    participant S as StaticAssets

    C->>MW: GET / (HTML isteği)
    MW->>M: istek loglama başlar
    M->>V: pages.Home + layout.Base render
    V->>S: /static/css/styles.css, htmx_2.0.10.min.js, alpinejs_3.16.2.min.js
    V-->>MW: HTML yanıtı
    MW-->>C: status, bytes, duration loglu yanıt

    C->>MW: GET /api/status (HTMX fragment)
    MW->>M: loglama
    M->>U: WriteHTML(w, 200, fragment div#status-box)
    U-->>MW: text/html#59; charset=utf-8 yanıtı
    MW-->>C: yanıt (HTML fragment)
```

## İnceleme Rehberi

1. Uygulamanın nereden başladığını görmek için → [[WebServer]]
2. Yanıt güvenlik başlıkları için → [[SecureHeaders]]
3. Her isteğin nasıl loglandığını anlamak için → [[LoggingMiddleware]]
4. HTML yanıtlarının nasıl yazıldığı için → [[HTTPUtil]]
5. Sayfa render ve component hiyerarşisi için → [[TemplViews]]
6. Stillerin nasıl derlendiği için → [[StaticAssets]]
7. Geliştirme/üretim pipeline'ı için → [[DevTooling]]
8. Lint kural ve istisnaları için → [[Linting]]
9. Güvenlik/performans riskleri ve temiz alanlar için → [[Security_Audit]]
10. Risk giderme planı ve refactor sırası için → [[Improvements]]
