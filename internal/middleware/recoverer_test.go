package middleware

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// panicHandler, panik fırlatan bir handler.
func panicHandler(value any) http.Handler {
	return http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic(value) })
}

// newTestLogger, çıktıyı yakalayan bir logger kurar.
func newTestLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestRecovererConvertsPanicTo500(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/patlar", http.NoBody)

	Recoverer(newTestLogger(&bytes.Buffer{}))(panicHandler("patlama")).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("durum = %d, beklenen %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestRecovererLogsPanicWithStack(t *testing.T) {
	var buf bytes.Buffer
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/patlar", http.NoBody)

	Recoverer(newTestLogger(&buf))(panicHandler("patlama metni")).
		ServeHTTP(rec, req)

	out := buf.String()
	for _, want := range []string{"Handler paniği", "patlama metni", "stack", "middleware"} {
		if !strings.Contains(out, want) {
			t.Errorf("logda %q yok: %s", want, out)
		}
	}
}

func TestRecovererPassesThroughWithoutPanic(t *testing.T) {
	var buf bytes.Buffer
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/ok", http.NoBody)

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("normal"))
	})
	Recoverer(newTestLogger(&buf))(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusTeapot {
		t.Errorf("durum = %d, beklenen %d (panik yoksa dokunulmamalı)", rec.Code, http.StatusTeapot)
	}
	if rec.Body.String() != "normal" {
		t.Errorf("gövde = %q", rec.Body.String())
	}
	if buf.Len() != 0 {
		t.Errorf("panik yokken log yazıldı: %s", buf.String())
	}
}

// TestRecovererRepanicsErrAbortHandler, http.ErrAbortHandler'ın yeniden
// fırlatıldığını doğrular. Bu değer net/http'ye "bağlantıyı loglama, kapat"
// demektir; yakalamak sessizce yutulmasına yol açar.
func TestRecovererRepanicsErrAbortHandler(t *testing.T) {
	defer func() {
		rvr := recover()
		if err, ok := rvr.(error); !ok || !errors.Is(err, http.ErrAbortHandler) {
			t.Errorf("recover() = %v, beklenen %v", rvr, http.ErrAbortHandler)
		}
	}()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/iptal", http.NoBody)
	Recoverer(newTestLogger(&bytes.Buffer{}))(panicHandler(http.ErrAbortHandler)).
		ServeHTTP(httptest.NewRecorder(), req)
}

// TestRecovererRepanicsWrappedErrAbortHandler, sarmalanmış ErrAbortHandler'ın da
// yeniden fırlatıldığını doğrular; aksi halde sessizce yutulurdu.
func TestRecovererRepanicsWrappedErrAbortHandler(t *testing.T) {
	wrapped := fmt.Errorf("sarmalandı: %w", http.ErrAbortHandler)

	defer func() {
		if rvr := recover(); rvr == nil {
			t.Error("sarmalanmış ErrAbortHandler yutuldu")
		}
	}()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/iptal", http.NoBody)
	Recoverer(newTestLogger(&bytes.Buffer{}))(panicHandler(wrapped)).
		ServeHTTP(httptest.NewRecorder(), req)
}

// TestPanicStillLogsRequest, doğru zincir sırasının istek logunu koruduğunu
// doğrular. Recoverer Logger'ın dışında olsaydı, panik Logger'ın çağrısını
// atlar ve istek logu hiç yazılmazdı.
func TestPanicStillLogsRequest(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf)

	// Logger(Recoverer(handler)) — main.go'daki sıra.
	handler := Logger(logger)(Recoverer(logger)(panicHandler("patlama")))
	handler.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/patlar", http.NoBody))

	out := buf.String()
	if !strings.Contains(out, "Handler paniği") {
		t.Errorf("panik logu yok: %s", out)
	}
	if !strings.Contains(out, `"msg=\"HTTP Request\"`) && !strings.Contains(out, "HTTP Request") {
		t.Errorf("istek logu yok: %s", out)
	}
	if !strings.Contains(out, "status=500") {
		t.Errorf("istek logu status=500 içermiyor: %s", out)
	}
}

// TestWrongOrderLosesRequestLog, ters sıralamanın istek logunu düşürdüğünü
// belgeler. main.go'da bu sıra yanlışlıkla kurulursa test kırmızıya döner.
func TestWrongOrderLosesRequestLog(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf)

	// Recoverer(Logger(handler)) — YANLIŞ sıra.
	handler := Recoverer(logger)(Logger(logger)(panicHandler("patlama")))
	handler.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/patlar", http.NoBody))

	out := buf.String()
	if !strings.Contains(out, "Handler paniği") {
		t.Errorf("panik logu yok: %s", out)
	}
	if strings.Contains(out, "HTTP Request") {
		t.Errorf("ters sırada istek logu YAZILMAMALI (bulundu): %s", out)
	}
}

func TestRecovererHandlesNonStringPanic(t *testing.T) {
	var buf bytes.Buffer
	rec := httptest.NewRecorder()

	Recoverer(newTestLogger(&buf))(panicHandler(errors.New("hata nesnesi"))).
		ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/x", http.NoBody))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("durum = %d, beklenen %d", rec.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(buf.String(), "hata nesnesi") {
		t.Errorf("hata değeri logda yok: %s", buf.String())
	}
}
