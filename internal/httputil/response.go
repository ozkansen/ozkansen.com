package httputil

import (
	"io"
	"log/slog"
	"net/http"
)

// WriteHTML, w'ye text/html başlığı ve verilen durum koduyla body'yi yazar.
// Yanıt commit edildikten sonraki yazma hataları (genelde bağlantı kopması)
// durum kodunu değiştiremeyeceğinden yalnızca loglanabilir; bu nedenle hata
// slog.Default() ile kaydedilir.
func WriteHTML(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if _, err := io.WriteString(w, body); err != nil {
		slog.Default().Error("HTTP yanıtı yazılamadı", slog.String("error", err.Error()))
	}
}
