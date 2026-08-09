// Package i18n, site içeriğini locale bazlı çevirmek için
// embed edilmiş JSON dosyalarını kullanan basit bir çeviri mekanizması sağlar.
package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"slices"
)

//go:embed locales/*.json
var localesFS embed.FS

// Locale, desteklenen bir dil kodunu temsil eder (ör. "tr", "en").
type Locale string

// SupportedLocales, sitede aktif olarak desteklenen dillerin listesidir.
var SupportedLocales = []Locale{"tr", "en"}

// DefaultLocale, kök / adresinden yönlendirilecek varsayılan dildir.
const DefaultLocale = Locale("tr")

// Translator, tek bir locale'e ait çeviri sözlüğünü tutar.
type Translator struct {
	locale Locale
	data   map[string]string
}

// String, Translator'ı Locale tipine dönüştürür.
func (t *Translator) String() string {
	return string(t.locale)
}

// Locale, aktif dili döndürür.
func (t *Translator) Locale() Locale {
	return t.locale
}

// T, verilen anahtar için çeviriyi döndürür. Anahtar bulunamazsa
// anahtarın kendisini döndürür; böylece eksik çeviriler gözden kaçmaz.
func (t *Translator) T(key string) string {
	if v, ok := t.data[key]; ok {
		return v
	}
	return key
}

// IsSupported, verilen değerin desteklenen bir locale olup olmadığını söyler.
func IsSupported(locale Locale) bool {
	return slices.Contains(SupportedLocales, locale)
}

// ParseLocale, bir URL öneki veya string değerden Locale üretir.
// Desteklenmeyen değerler için DefaultLocale döner.
func ParseLocale(s string) Locale {
	locale := Locale(s)
	if IsSupported(locale) {
		return locale
	}
	return DefaultLocale
}

// Load, verilen locale için çeviri sözlüğünü embed'li JSON'dan yükler.
func Load(locale Locale) (*Translator, error) {
	if !IsSupported(locale) {
		locale = DefaultLocale
	}

	path := "locales/" + string(locale) + ".json"
	data, err := localesFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("i18n: locale dosyası okunamadı %q: %w", path, err)
	}

	var entries map[string]string
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("i18n: locale dosyası çözümlenemedi %q: %w", path, err)
	}

	return &Translator{locale: locale, data: entries}, nil
}
