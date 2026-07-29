package middleware

import (
	"log/slog"
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

// Logger gelen ve giden tüm HTTP isteklerini detaylı şekilde slog ile kaydeder.
func Logger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rw := &responseWriter{
				ResponseWriter: w,
				status:         http.StatusOK, // Varsayılan değer
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
