package middleware

import (
	"bufio"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"
)

// responseWriter, HTTP durum kodunu ve yazılan bayt miktarını yakalamak için sarmalayıcı yapı.
//
// status, WriteHeader hiç çağrılmamışsa net/http'in de kullanacağı varsayılan
// 200'den başlar. Böylece status her an gerçek durum kodunu taşır; logger tarafında
// "0, aslında 200 demek" gibi bir normalizasyona gerek kalmaz.
type responseWriter struct {
	http.ResponseWriter
	status       int
	wroteHeader  bool
	bytesWritten int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, status: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	// net/http yalnızca ilk çağrıyı uygular; log'un gerçek durum koduyla
	// uyumlu kalması için sonraki çağrılar yok sayılır.
	if rw.wroteHeader {
		return
	}
	rw.wroteHeader = true
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	// İlk yazma yanıtı 200'e sabitler; status zaten 200 olduğundan güncelleme
	// gerekmez, yalnızca sonraki WriteHeader çağrıları yoksayılsın.
	rw.wroteHeader = true
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}

// WriteString, io.StringWriter arayüzünü ileri taşır; böylece io.WriteString
// çağrıları []byte dönüşümü yapmadan temel ResponseWriter'a ulaşır.
func (rw *responseWriter) WriteString(s string) (int, error) {
	rw.wroteHeader = true
	n, err := io.WriteString(rw.ResponseWriter, s)
	rw.bytesWritten += n
	return n, err
}

// Flush, net/http Flusher arayüzünü ileri taşır (SSE/streaming yanıtlar için).
// Temel ResponseWriter Flusher desteklemiyorsa çağrı sessizce yok sayılır.
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack, WebSocket/uzun poll gibi senaryolar için temel bağlantıyı devralır.
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, errors.New("underlying ResponseWriter does not implement http.Hijacker")
}

// Push, HTTP/2 sunucu push için Pusher arayüzünü ileri taşır.
func (rw *responseWriter) Push(target string, opts *http.PushOptions) error {
	if p, ok := rw.ResponseWriter.(http.Pusher); ok {
		return p.Push(target, opts)
	}
	return http.ErrNotSupported
}

// Unwrap, http.ResponseController'ın sarmalayıcıyı aşarak temel
// ResponseWriter üzerindeki Flusher/Hijacker/Pusher yeteneklerini bulmasını sağlar.
func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

// stripControlChars, istemciden gelen bir değerden kontrol karakterlerini
// çıkarır. Görünür karakterler korunur, böylece log okunabilir kalır.
//
// Kapsam notu: slog'un TextHandler ve JSONHandler'ı bu karakterleri zaten
// kaçışlar (metin çıktısında \r\n iki karakter olarak yazılır, JSON'da \r\n
// olur), dolayısıyla çok satırlı log forging oluşmaz. Bu işlem ek güvenliktir:
// istemci başlıklarındaki kontrol karakterlerinin loga hiç girmesini garanti
// eder ve log çıktısı başka bir sink'e aktarılırsa ya da handler değişirse
// temizlenmemiş veriye güvenilmez.
func stripControlChars(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, s)
}

// remoteIP, "host:port" biçimindeki istemci adresinden yalnızca host kısmını
// döndürür. Port efemereldir ve her istekte değişir; loglanırsa aynı istemci
// gruplandırılamaz (log gürültüsü) ve gereksiz yere PII saklanır.
func remoteIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		// Beklenmeyen biçim (unix socket, eksik port): veri kaybına yol açmamak
		// için ham değeri tercih et.
		return remoteAddr
	}
	return host
}

// Logger gelen ve giden tüm HTTP isteklerini detaylı şekilde slog ile kaydeder.
func Logger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rw := newResponseWriter(w)

			// İsteği sonraki handler'a ilet
			next.ServeHTTP(rw, r)

			duration := time.Since(start)

			// Log seviyesi durum koduna göre ayarlanabilir (örneğin 5xx hatalar için Error)
			level := slog.LevelInfo
			if rw.status >= 500 {
				level = slog.LevelError
			} else if rw.status >= 400 {
				level = slog.LevelWarn
			}

			logger.Log(r.Context(), level, "HTTP Request",
				// method, path ve user_agent istemciden gelir. Temizlenerek
				// loglanır; ayrıntı için stripControlChars.
				slog.String("method", stripControlChars(r.Method)),
				slog.String("path", stripControlChars(r.URL.Path)),
				slog.Int("status", rw.status),
				slog.Int("bytes", rw.bytesWritten),
				slog.Duration("duration", duration),
				// remote_ip ağ katmanından gelir, istemci başlığı değildir.
				slog.String("remote_ip", remoteIP(r.RemoteAddr)),
				slog.String("user_agent", stripControlChars(r.UserAgent())),
			)
		})
	}
}
