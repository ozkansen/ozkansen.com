package middleware

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// serve, SecureHeaders'dan geçirilmiş bir yanıt döndürür.
func serve(t *testing.T, next http.Handler, r *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	SecureHeaders(next).ServeHTTP(rec, r)
	return rec
}

// okHandler, her zaman 200 döndüren sade bir handler.
func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestSecureHeadersPresent(t *testing.T) {
	rec := serve(t, okHandler(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.com/", http.NoBody))

	want := map[string]string{
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
		"Referrer-Policy":         "strict-origin-when-cross-origin",
		"Content-Security-Policy": cspPolicy,
		"Permissions-Policy":      permissionsPolicy,
	}

	for k, v := range want {
		if got := rec.Header().Get(k); got != v {
			t.Errorf("%s = %q, beklenen %q", k, got, v)
		}
	}
}

// TestHSTSSkippedOnPlainHTTP, düz HTTP isteklerinde HSTS gönderilmediğini
// doğrular. Bu güvenlik açısından kritiktir: başlık düz bağlantıda gönderilirse
// tarayıcı alan adını HTTP'ye karşı kalıcı olarak kilitler ve kullanıcı siteye
// erişemez olur. Karar tarayıcıda saklanır, sunucudan geri alınamaz.
func TestHSTSSkippedOnPlainHTTP(t *testing.T) {
	rec := serve(t, okHandler(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.com/", http.NoBody))

	if got := rec.Header().Get("Strict-Transport-Security"); got != "" {
		t.Errorf("düz HTTP'de HSTS gönderildi: %q", got)
	}
}

func TestHSTSSentOnTLS(t *testing.T) {
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.com/", http.NoBody)
	req.TLS = &tls.ConnectionState{}

	rec := serve(t, okHandler(), req)

	if got := rec.Header().Get("Strict-Transport-Security"); got != hstsPolicy {
		t.Errorf("HSTS = %q, beklenen %q", got, hstsPolicy)
	}
}

// TestSecureHeadersOnErrorResponses, hata yanıtlarının da korumalı olduğunu
// doğrular. SecureHeaders zincirin en dışında olduğu için 404/405 dahil her
// yanıt başlık almalıdır.
func TestSecureHeadersOnErrorResponses(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		status  int
	}{
		{"404", func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) }, http.StatusNotFound},
		{"500", func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "patlama", http.StatusInternalServerError)
		}, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.com/yok", http.NoBody)
			req.TLS = &tls.ConnectionState{}
			rec := serve(t, tt.handler, req)

			if rec.Code != tt.status {
				t.Fatalf("durum = %d, beklenen %d", rec.Code, tt.status)
			}
			for _, k := range []string{"X-Content-Type-Options", "Content-Security-Policy", "Permissions-Policy", "Strict-Transport-Security"} {
				if rec.Header().Get(k) == "" {
					t.Errorf("%s hata yanıtında eksik", k)
				}
			}
		})
	}
}

// TestPermissionsPolicyDeniesUnusedAPIs, kapatılan API'lerin gerçekten
// yasaklandığını doğrular. Sitenin kullandığı hiçbir özellik bu listede
// olmamalıdır.
func TestPermissionsPolicyDeniesUnusedAPIs(t *testing.T) {
	rec := serve(t, okHandler(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.com/", http.NoBody))
	policy := rec.Header().Get("Permissions-Policy")

	for _, feature := range []string{"camera", "microphone", "geolocation", "payment", "usb", "fullscreen"} {
		if want := feature + "=()"; !strings.Contains(policy, want) {
			t.Errorf("Permissions-Policy içinde %q yok: %s", want, policy)
		}
	}
}
