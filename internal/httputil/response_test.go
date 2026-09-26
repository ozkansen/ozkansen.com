package httputil

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// contentType, WriteHTML'in set etmesi beklenen Content-Type değeridir.
const contentType = "text/html; charset=utf-8"

func TestWriteHTML(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		wantType   string
		wantStatus int
	}{
		{"200", http.StatusOK, "<p>merhaba</p>", contentType, http.StatusOK},
		{"404", http.StatusNotFound, "yok", contentType, http.StatusNotFound},
		{"500", http.StatusInternalServerError, "hata", contentType, http.StatusInternalServerError},
		{"bos govde", http.StatusOK, "", contentType, http.StatusOK},
		{"bos govde, 204", http.StatusNoContent, "", contentType, http.StatusNoContent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			WriteHTML(rec, tt.status, tt.body)

			if rec.Code != tt.wantStatus {
				t.Errorf("durum = %d, beklenen %d", rec.Code, tt.wantStatus)
			}
			if got := rec.Header().Get("Content-Type"); got != tt.wantType {
				t.Errorf("Content-Type = %q, beklenen %q", got, tt.wantType)
			}
			if got := rec.Body.String(); got != tt.body {
				t.Errorf("gövde = %q, beklenen %q", got, tt.body)
			}
		})
	}
}

// TestWriteHTMLDoesNotDoubleSetType, Content-Type'ın iki kez set edilmediğini ve
// gövdenin başına eklenmediğini doğrular.
func TestWriteHTMLSetsTypeExactlyOnce(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteHTML(rec, http.StatusOK, "<div>x</div>")

	values := rec.Header().Values("Content-Type")
	if len(values) != 1 {
		t.Errorf("Content-Type %d kez set edildi: %q", len(values), values)
	}
	if !strings.HasPrefix(rec.Body.String(), "<div>") {
		t.Errorf("gövde beklenmeyen bir şeyle başlıyor: %q", rec.Body.String())
	}
}

// errWrite, testte yazma hatası üretmek için kullanılır.
var errWrite = errors.New("bağlantı koptu")

// failingWriter, yazma işleminde hata döndüren bir ResponseWriter'dır.
// io.StringWriter'ı da uygular: production'da middleware'ın responseWriter
// sarmalayıcısı bu arayüzü sağlar, böylece io.WriteString Write yerine
// WriteString yolunu kullanır. Test her iki yolu da kapsar.
type failingWriter struct {
	header http.Header
	code   int
}

func (w *failingWriter) Header() http.Header {
	if w.header == nil {
		w.header = http.Header{}
	}
	return w.header
}

func (w *failingWriter) Write([]byte) (int, error)       { return 0, errWrite }
func (w *failingWriter) WriteString(string) (int, error) { return 0, errWrite }
func (w *failingWriter) WriteHeader(code int)            { w.code = code }

// TestWriteHTMLLogsWriteError, commit sonrası yazma hatasının panik vermeden
// loglandığını doğrular. Yanıt commit edildikten sonra durum kodu değiştirilemez;
// bu nedenle tek yapılabilen gözlemlenebilirlik sağlamaktır.
func TestWriteHTMLLogsWriteError(t *testing.T) {
	var logs bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(old) })

	w := &failingWriter{}
	WriteHTML(w, http.StatusOK, "gövde")

	if w.code != http.StatusOK {
		t.Errorf("durum = %d, beklenen %d", w.code, http.StatusOK)
	}
	if !strings.Contains(logs.String(), "HTTP yanıtı yazılamadı") {
		t.Errorf("yazma hatası loglanmadı: %q", logs.String())
	}
	if !strings.Contains(logs.String(), errWrite.Error()) {
		t.Errorf("hata metni logda yok: %q", logs.String())
	}
}

// TestWriteHTMLUsesCurrentDefaultLogger, slog.Default() çalışma zamanında
// çözüldüğü için testte değiştirilen logger'ın kullanıldığını doğrular.
func TestWriteHTMLNoErrorStaysSilent(t *testing.T) {
	var logs bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(old) })

	WriteHTML(httptest.NewRecorder(), http.StatusOK, "sorunsuz")

	if logs.Len() != 0 {
		t.Errorf("hata yokken log yazıldı: %q", logs.String())
	}
}
