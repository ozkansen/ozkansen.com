package middleware

import (
	"bufio"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// responseWriter, HTTP durum kodunu ve yazılan bayt miktarını yakalamak için sarmalayıcı yapı.
type responseWriter struct {
	http.ResponseWriter
	status       int
	bytesWritten int
}

func (rw *responseWriter) WriteHeader(code int) {
	// Birden fazla çağrıyı yoksay: net/http ilk çağrıyı kullanır,
	// log'un gerçek durum koduyla uyumlu kalması için burada da ilki korunur.
	if rw.status != 0 {
		return
	}
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	// Eğer WriteHeader açıkça çağrılmadıysa Go varsayılan olarak 200 OK döner
	if rw.status == 0 {
		rw.status = http.StatusOK
	}
	n, err := rw.ResponseWriter.Write(b)
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

// Logger gelen ve giden tüm HTTP isteklerini detaylı şekilde slog ile kaydeder.
func Logger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rw := &responseWriter{
				ResponseWriter: w,
			}

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
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rw.status),
				slog.Int("bytes", rw.bytesWritten),
				slog.Duration("duration", duration),
				slog.String("remote_ip", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()),
			)
		})
	}
}
