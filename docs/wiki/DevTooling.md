---
layer: repository
dependencies: [TemplViews, StaticAssets, WebServer, Linting]
last_updated: 2026-08-22
---

# DevTooling

**Özet:** Geliştirme ve derleme iş akışını `Makefile` ve `.air.toml` üzerinden yönetir. `make dev` templ, Tailwind ve Air izleyicilerini paralel başlatır; `make build` templ üretimi + minified Tailwind + optimize edilmiş Go binary'sini (`dist/server`) ve statik varlıkları (`dist/static`) üretir. Air, `.templ` değişikliklerinde önce templ'i derleyip sonra Go binary'sini yeniden inşa ederek hot-reload sağlar.

**Kütüphaneler/Tool'lar:** GNU Make, `a-h/templ` CLI, Tailwind CSS CLI (`@tailwindcss/cli`), `air-verse/air`, `golangci-lint` (v2), `gofumpt`, `gci`, `gofmt`. Node tarafı sürümleri `package.json` + `package-lock.json` ile sabitlenmiştir.

**Bağlantılar:** [[TemplViews]] · [[StaticAssets]] · [[WebServer]] · [[Linting]] · [[Index]]

**Dosyalar:**
- `Makefile`
- `.air.toml`
- `.golangci.yml`

## Geniş açıklama

### Geliştirme akışı

```mermaid
flowchart TD
    Dev["make dev (paralel -j3)"] --> T[templ generate --watch]
    Dev --> TW[npx tailwindcss --watch]
    Dev --> A[air]

    T -->|"_templ.go üretir"| GoBuild["go build → tmp/main"]
    A -->|"değişim algıla (.go/.templ)"| GoBuild
    TW -->|"styles.css üretir"| Static["static/css/styles.css"]

    GoBuild --> Serve["tmp/main :8080"]
    Serve --> Logs[HTTP logları]
```

- **templ watcher:** `.templ` dosyası değişince `_templ.go` kodlarını yeniden üretir (bkz. [[TemplViews]]).
- **Tailwind watcher:** `input.css` kaynağını tarayıp `styles.css` üretir (bkz. [[StaticAssets]]).
- **Air watcher:** `.go` veya `.templ` değişiminde `templ generate && go build -o ./tmp/main ./cmd/web` çalıştırıp sunucuyu yeniden başlatır.

### Production build akışı (`make build`)

```mermaid
flowchart TD
    B["make build"] --> T["templ generate"]
    T --> TW["npx tailwindcss --minify<br/>(input.css → styles.css)"]
    TW --> GO["GOEXPERIMENT='simd,newinliner'<br/>go build -ldflags='-s -w' -trimpath<br/>-o ./dist/server ./cmd/web"]
    GO --> CP["cp -r static → dist/static"]
    CP --> RM["rm -rf dist/static/css/input.css"]
    RM --> OUT["dist/server + dist/static (dağıtıma hazır)"]
```

- **`GOEXPERIMENT="simd,newinliner"`:** Go 1.27 derleyici denemelerini açar (simd + yeni inliner); performans odaklı optimizasyon sağlar.
- **`-ldflags="-s -w" -trimpath`:** Sembol tablosu ve DWARF bilgisi atılır, dosya yolları kırpılır → küçük ve tekrarlanabilir (reproducible) binary.
- **`templ generate` ön koşulu:** Üretilen `_templ.go` dosyaları her zaman güncel kaynakla eşleşir.
- **`--minify` + `input.css` çıkarımı:** Dağıtım paketinde yalnızca minified CSS bulunur; kaynak dosya dışarıda bırakılır.

### Makefile hedefleri

| Hedef | Komut | Amaç |
|---|---|---|
| `install` | npm + go install | Geliştirme araçlarını kurar |
| `dev` | `make -j3 templ tailwind air` | Tüm izleyicileri paralel başlatır |
| `build` | `templ generate` + `tailwind --minify` + `go build` | Production binary üretir (`./dist/server` + `dist/static`) |
| `fmt` | `gofmt -w .` + `templ fmt .` | Formatlar |
| `lint` | `golangci-lint run ./...` | Kural ihlali taraması (bkz. [[Linting]]) |
| `lint-fix` | fmt + `golangci-lint run --fix` | Otomatik düzeltme (linter + gofumpt/gci formatter'ları) |

### Air yapılandırması (`.air.toml`)

- `cmd = "templ generate && go build -o ./tmp/main ./cmd/web"` → her derleme templ üretimiyle başlar.
- `exclude_dir`: `tmp`, `vendor`, `node_modules`, `static` (statik değişiklikler hot-reload tetiklemez).
- `include_ext`: `go`, `templ`, `html`; `exclude_regex`: `.*_templ.go` (üretilen dosyalar yeniden derleme başlatmaz).
- `stop_on_error = true` → derleme hatasında izleyici durur.
- Çıkışta `clean_on_exit` ile `tmp` temizlenir.

### Önemli noktalar

- Geliştirme binary'si `./tmp/main` içinde, production binary'si `./dist/server` içinde üretilir; bu dizinler gitignore'dadır.
- `package.json` ve `package-lock.json` takip edilir; npm bağımlılıkları (Tailwind CLI) lockfile ile reproducible kurulur. Runtime JS bağımlılığı yoktur ([[StaticAssets]] yerel kopyalar kullanır).
- Proje Go 1.27 ile derlenir (`go.mod`); `make build` ayrıca `GOEXPERIMENT` denemelerini etkinleştirir.
- Lint hiyerarşisi `fmt → lint → lint-fix` şeklinde önerilir; CI öncesi `lint` çalıştırılmalıdır.
- `.golangci.yml` ayrı bir node olarak incelenir: [[Linting]]. `make lint-fix` ayrıca `gofumpt` ve `gci` (import gruplama) formatter'larını da uygular.
