package middleware

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

// statusCapture, slog kayıtlarındaki "status" ve "remote_ip" alanlarını yakalar.
type statusCapture struct {
	status   int
	remoteIP string
}

func (c *statusCapture) Enabled(context.Context, slog.Level) bool { return true }

func (c *statusCapture) Handle(_ context.Context, r slog.Record) error { //nolint:gocritic // slog.Handler arayüzü imzayı zorunlu kılar
	r.Attrs(func(a slog.Attr) bool {
		switch a.Key {
		case "status":
			c.status = int(a.Value.Int64())
		case "remote_ip":
			c.remoteIP = a.Value.String()
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

// loopback, IPv4 test adresi; tabloda tekrar tekrar yazmamak için.
const loopback = "127.0.0.1"

func TestRemoteIP(t *testing.T) {
	tests := []struct {
		name string
		addr string
		want string
	}{
		{"IPv4 ve port", loopback + ":48994", loopback},
		{"IPv6 ve port", "[::1]:51506", "::1"},
		{"IPv4 port içermiyor", "192.0.2.1", "192.0.2.1"},
		{"host boş (unix socket)", ":8080", ""},
		{"bozuk biçim", loopback, loopback},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := remoteIP(tt.addr); got != tt.want {
				t.Errorf("remoteIP(%q) = %q, beklenen %q", tt.addr, got, tt.want)
			}
		})
	}
}

// TestRemoteIPIsStable, aynı istemcinin portu değişse bile loglanan IP'nin
// sabit kaldığını doğrular; aksi halde loglar istemci bazında gruplanamaz.
func TestRemoteIPIsStable(t *testing.T) {
	if a, b := remoteIP("203.0.113.7:40001"), remoteIP("203.0.113.7:62110"); a != b {
		t.Errorf("farklı portlar farklı IP üretti: %q != %q", a, b)
	}
}

// TestLoggerLogsRemoteIPWithoutPort, Logger'ın logladığı remote_ip alanında port
// bulunmadığını uçtan uca doğrular. Regresyon: alan r.RemoteAddr olduğunda
// efemerel port her istekte değişiyordu.
func TestLoggerLogsRemoteIPWithoutPort(t *testing.T) {
	capture := &statusCapture{}
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
	req.RemoteAddr = "203.0.113.9:62341"

	Logger(slog.New(capture))(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
	).ServeHTTP(httptest.NewRecorder(), req)

	if got, want := capture.remoteIP, "203.0.113.9"; got != want {
		t.Errorf("loglanan remote_ip = %q, beklenen %q (port içermemeli)", got, want)
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
