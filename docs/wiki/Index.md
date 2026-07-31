---
last_updated: 2026-07-31
---

# Index

**Özet:** Bu dosya projenin Bilgi Grafiği ana haritasıdır (Knowledge Graph Index). Tüm wiki node'larını listeler, katman dağılımını gösterir ve bir isteğin istemciden yanıta kadar izlediği akışı özetler. Yeni bir modül/fikir araştırmadan önce buradan başlayın.

## Mimari Harita

```mermaid
flowchart LR
    subgraph Delivery
        WebServer
        LoggingMiddleware
        TemplViews
        StaticAssets
    end

    subgraph Tooling
        DevTooling
    end

    Client[İstemci] --> WebServer
    WebServer --> LoggingMiddleware
    WebServer --> TemplViews
    WebServer --> StaticAssets
    TemplViews --> StaticAssets
    DevTooling --> TemplViews
    DevTooling --> StaticAssets
    DevTooling --> WebServer
```

## Node Listesi

### Delivery Katmanı
- [[WebServer]] — giriş noktası, `ServeMux`, route tanımları, statik servis, middleware sarmalama
- [[LoggingMiddleware]] — istek/yanıt loglama, `responseWriter` sarmalayıcı, seviye mantığı
- [[TemplViews]] — templ view katmanı: `layout.Base` iskeleti + `pages.Home` sayfası
- [[StaticAssets]] — Tailwind CSS (v4 `@source`) + HTMX/Alpine.js statik kopyaları

### Araçlar / Süreç
- [[DevTooling]] — `Makefile` (dev/build/lint) + `.air.toml` hot-reload pipeline

## Bağımlılık Grafiği

```mermaid
graph TD
    WebServer --> LoggingMiddleware
    WebServer --> TemplViews
    WebServer --> StaticAssets
    TemplViews --> StaticAssets
    TemplViews --> WebServer
    StaticAssets --> TemplViews
```

## Uçtan Uca İstek Akışı

```mermaid
sequenceDiagram
    participant C as Tarayıcı (HTMX/Alpine)
    participant MW as LoggingMiddleware
    participant M as ServeMux
    participant V as TemplViews
    participant S as StaticAssets

    C->>MW: GET / (HTML isteği)
    MW->>M: istek loglama başlar
    M->>V: pages.Home + layout.Base render
    V->>S: /static/css/styles.css, htmx.min.js, cdn.min.js
    V-->>MW: HTML yanıtı
    MW-->>C: status, bytes, duration loglu yanıt

    C->>MW: GET /api/status (HTMX fragment)
    MW->>M: loglama
    M-->>MW: fragment div#status-box
    MW-->>C: yanıt (HTML fragment)
```

## İnceleme Rehberi

1. Uygulamanın nereden başladığını görmek için → [[WebServer]]
2. Her isteğin nasıl loglandığını anlamak için → [[LoggingMiddleware]]
3. Sayfa render ve component hiyerarşisi için → [[TemplViews]]
4. Stillerin nasıl derlendiği için → [[StaticAssets]]
5. Geliştirme/üretim pipeline'ı için → [[DevTooling]]
