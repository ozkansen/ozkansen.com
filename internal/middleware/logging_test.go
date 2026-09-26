package middleware

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

// statusCapture, slog kayıtlarındaki "status" alanını yakalar.
type statusCapture struct{ status int }

func (c *statusCapture) Enabled(context.Context, slog.Level) bool { return true }

func (c *statusCapture) Handle(_ context.Context, r slog.Record) error { //nolint:gocritic // slog.Handler arayüzü imzayı zorunlu kılar
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "status" {
			c.status = int(a.Value.Int64())
		}
		return true
	})
	return nil
}

func (c *statusCapture) WithAttrs([]slog.Attr) slog.Handler { return c }

func (c *statusCapture) WithGroup(string) slog.Handler { return c }

// TestLoggerLogsActualStatus, loglanan durum kodunun istemciye giden gerçek
// durum koduna eşit olduğunu doğrular. Regresyon: handler hiçbir şey yazmadığında
// logger status=0 kaydediyordu.
func TestLoggerLogsActualStatus(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"handler hiçbir şey yazmıyor", func(http.ResponseWriter, *http.Request) {}},
		{"Write çağrılıyor", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("ok"))
		}},
		{"WriteString çağrılıyor", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, "ok")
		}},
		{"WriteHeader çağrılıyor", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusTeapot)
		}},
		{"gövdesiz 204", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}},
		{"Write sonrası WriteHeader yok sayılıyor", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("ok"))
			w.WriteHeader(http.StatusTeapot)
		}},
		{"WriteHeader sonrası tekrar yok sayılıyor", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.WriteHeader(http.StatusOK)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capture := &statusCapture{}
			rec := httptest.NewRecorder()

			Logger(slog.New(capture))(tt.handler).
				ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody))

			if capture.status == 0 {
				t.Fatal("loglanan durum kodu 0")
			}
			if capture.status != rec.Code {
				t.Errorf("loglanan durum %d, gerçek durum %d", capture.status, rec.Code)
			}
		})
	}
}

func TestLoggerCountsBytes(t *testing.T) {
	body := "çok satırlı gövde"

	Logger(slog.New(&statusCapture{}))(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, body)
	})).ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody))

	rw := newResponseWriter(httptest.NewRecorder())
	_, _ = rw.WriteString(body)

	if rw.bytesWritten != len(body) {
		t.Errorf("bytesWritten = %d, beklenen %d", rw.bytesWritten, len(body))
	}
}
