---
layer: delivery
dependencies: [WebServer]
last_updated: 2026-08-22
---

# HTTPUtil

**Özet:** `internal/httputil` paketi, HTTP yanıtlarını standart ve güvenli biçimde yazmak için küçük yardımcı fonksiyonlar sunar. `WriteHTML`, `Content-Type: text/html; charset=utf-8` başlığını set eder, verilen durum koduyla yanıtı commit eder ve body'yi yazar. Yanıt commit edildikten sonra oluşan yazma hataları (bağlantı kopması vb.) durum kodunu değiştiremeyeceğinden `slog.Default()` ile loglanır.

**Kütüphaneler:** Go standard library (`net/http`, `log/slog`, `io`).

**Bağlantılar:** [[WebServer]] · [[LoggingMiddleware]] · [[Index]]

**Dosyalar:**
- `internal/httputil/response.go`

## Geniş açıklama

Paket, handler'ların "header set → status yaz → body yaz" kalıbını her seferinde elle tekrarlamasını engeller. [[WebServer]]'deki `/api/status` handler'ı, fragment yanıtını bu helper üzerinden üretir; böylece `Content-Type` ve hata loglama davranışı tüm HTML yanıtlarında tutarlı hale gelir.

### WriteHTML akışı

```mermaid
sequenceDiagram
    participant H as Handler (apiStatusHandler)
    participant W as WriteHTML
    participant RW as ResponseWriter (middleware sarmalayıcı)
    participant SL as slog.Default()

    H->>W: WriteHTML(w, 200, statusFragment)
    W->>RW: Header().Set("Content-Type", "text/html; charset=utf-8")
    W->>RW: WriteHeader(200)
    Note over RW: Yanıt commit edilir (durum kodu artık değişemez)
    W->>RW: io.WriteString(w, body)
    alt Yazma hatası (ör. istemci bağlantıyı kesti)
        RW-->>W: err
        W->>SL: Error("HTTP yanıtı yazılamadı", error)
    end
```

### Tasarım kararları

- **Log-only hata yönetimi:** Yanıt commit edildikten sonraki yazma hataları istemciye bildirilemez; tek yapılabilir şey gözlemlenebilirlik sağlamaktır. Bu yüzden hata `slog.Default()` ile `Error` seviyesinde kaydedilir ve fonksiyondan hata döndürülmez.
- **`io.WriteString` kullanımı:** Body string olarak yazılır; [[LoggingMiddleware]]'deki `responseWriter` sarmalayıcısı `WriteString`'i ileri taşıdığı için ekstra bir `[]byte` dönüşümü olmadan temel `ResponseWriter`'a ulaşır.
- **Durum kodu parametresi:** Status, çağıranın sorumluluğundadır; helper yalnızca mekanik işi merkezileştirir.

### Önemli noktalar

- Fonksiyon stateless'tır; test edilebilirlik için `slog.Default()` çağrısı çalışma zamanında çözülür (gerekirse `slog.SetDefault` ile değiştirilebilir).
- Paket diğer internal modüllere bağımlı değildir; yalnızca standard library kullanır. Bağımlılık yönü: [[WebServer]] → `httputil`.
