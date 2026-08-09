// Package handlers, HTTP uç noktalarını barındırır.
package handlers

import (
	"html"
	"log/slog"
	"net/http"
	"strings"

	"ozkansen.com/internal/i18n"
)

// maxFieldLen, form alanları için izin verilen maksimum uzunluktur.
const maxFieldLen = 1000

// Contact, /api/contact uç noktasını işler. Formu doğrular, mesajı loglar
// ve HTMX fragment'ı olarak kullanıcıya geri bildirim döndürür.
//
// Veritabanı altyapısı kurulana kadar mesajlar yalnızca loglanır.
func Contact(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseForm(); err != nil {
			logger.Error("Contact form parse edilemedi", slog.String("error", err.Error()))
			writeFeedback(w, i18n.DefaultLocale, "contact.form.error")
			return
		}

		locale := i18n.ParseLocale(r.FormValue("locale"))
		name := strings.TrimSpace(r.FormValue("name"))
		email := strings.TrimSpace(r.FormValue("email"))
		subject := strings.TrimSpace(r.FormValue("subject"))
		message := strings.TrimSpace(r.FormValue("message"))

		if !valid(name, email, subject, message) {
			logger.Warn("Contact form doğrulama hatası",
				slog.String("locale", string(locale)),
				slog.String("email", email),
			)
			writeFeedback(w, locale, "contact.form.error")
			return
		}

		logger.Info("Contact mesajı alındı",
			slog.String("locale", string(locale)),
			slog.String("name", name),
			slog.String("email", email),
			slog.String("subject", subject),
			slog.Int("message_len", len(message)),
		)

		writeFeedback(w, locale, "contact.form.success")
	}
}

// writeFeedback, HTMX tarafından #contact-feedback'e eklenen fragment'ı yazar.
// Mesaj metni, formdan gelen locale'e göre çevrilir.
func writeFeedback(w http.ResponseWriter, locale i18n.Locale, key string) {
	tr, err := i18n.Load(locale)
	if err != nil {
		tr, _ = i18n.Load(i18n.DefaultLocale)
	}
	msg := tr.T(key)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<div id="contact-feedback" class="text-sm">` + html.EscapeString(msg) + `</div>`))
}

// valid, form alanlarını basit kurallarla doğrular.
func valid(name, email, subject, message string) bool {
	if name == "" || len(name) > maxFieldLen {
		return false
	}
	if email == "" || len(email) > maxFieldLen || !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return false
	}
	if len(subject) > maxFieldLen {
		return false
	}
	if message == "" || len(message) > maxFieldLen {
		return false
	}
	return true
}
