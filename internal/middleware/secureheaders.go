package middleware

import "net/http"

// cspPolicy, tarayıcının bu sitede yükleyebileceği kaynakları beyaz listeler.
// script-src içindeki 'unsafe-eval' Alpine.js için zorunludur: Alpine,
// x-data/@click ifadelerini çalışma zamanında new Function() ile derler
// (eval eşdeğeri) ve CSP bunu yasaklarsa bileşenler sessizce bozulur.
const cspPolicy = "default-src 'self'; " +
	"script-src 'self' 'unsafe-eval'; " +
	"style-src 'self'; " +
	"img-src 'self' data:; " +
	"object-src 'none'; " +
	"base-uri 'self'; " +
	"frame-ancestors 'none'"

// SecureHeaders, tüm yanıtlara temel güvenlik başlıklarını ekler.
// Zincirin en dışına (ilk Use) konumlanmalıdır ki hata yanıtları dahil
// her yanıt korumalı olsun.
func SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		// X-Frame-Options eski tarayıcılar içindir; modern karşılığı
		// CSP frame-ancestors 'none' direktifidir. İkisi birlikte set edilir.
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Content-Security-Policy", cspPolicy)
		next.ServeHTTP(w, r)
	})
}
