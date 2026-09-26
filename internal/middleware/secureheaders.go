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

// permissionsPolicy, sayfanın kullanmasına ihtiyaç duymadığı tarayıcı
// API'lerini kapatır. Site yalnızca kendi kaynaklarını yükler; kamera,
// mikrofon, konum, ödeme gibi hiçbiri kullanılmaz. Kapatılan özelliklerin
// listesi bir tarayıcı API'si eklenmediği sürece güvenle genişletilebilir.
const permissionsPolicy = "accelerometer=(), autoplay=(), camera=(), display-capture=(), " +
	"encrypted-media=(), fullscreen=(), geolocation=(), gyroscope=(), magnetometer=(), " +
	"microphone=(), payment=(), picture-in-picture=(), screen-wake-lock=(), usb=(), " +
	"xr-spatial-tracking=()"

// hstsPolicy, tarayıcının bu alan adını yalnızca HTTPS üzerinden açmasını
// sağlar. includeSubDomains bilinçli olarak yoktur: tüm alt alan adlarını
// kapsar ve HTTPS'e geçmemiş başka bir alt alan adını de kilitler.
// max-age kademeli olarak artırılabilir; başlangıçta düşük tutulması,
// TLS kurulumu sorunluysa etkiyi sınırlar.
const hstsPolicy = "max-age=31536000"

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
		h.Set("Permissions-Policy", permissionsPolicy)

		// HSTS yalnızca TLS'li bağlantıda gönderilir. Düz HTTP üzerinden
		// göndermek ters etki yaratır: tarayıcı alan adını HTTP'ye karşı
		// kilitler ve kullanıcı siteye hiç erişemez hale gelir. Karar tarayıcı
		// tarafında kalıcıdır, geri alınamaz.
		//
		// Not: TLS bir reverse proxy'nin arkasında sonlandırılıyorsa r.TLS
		// nil olur ve bu başlık gönderilmez. O kurulumda X-Forwarded-Proto
		// desteği gerekir; güvenilir proxy listesi olmadan ilgili başlıklar
		// istemci tarafından sahte gelebileceği için bu bilinçli olarak
		// ertelenmiştir.
		if r.TLS != nil {
			h.Set("Strict-Transport-Security", hstsPolicy)
		}

		next.ServeHTTP(w, r)
	})
}
