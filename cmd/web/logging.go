package main

import (
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/lmittmann/tint"
)

// appEnvProd, makine tarafından okunabilir log üretilen ortamdır. Bu değer
// dışındaki her şey geliştirme ortamı sayılır.
const appEnvProd = "production"

// timeFormat, geliştirme loglarındaki zaman damgası biçimidir. Yıl ve saat
// dilimi içerir; bu olmadan loglar toplanamaz (hangi yıl olduğu bilinmez) ve
// farklı saat dilimleri arasında karşılaştırılamaz. Milisaniye çözünürlüğü,
// istek sürelerinin mikrosaniye mertebesinde olması nedeniyle korunur.
const timeFormat = "2006-01-02T15:04:05.000Z07:00"

// newLogger, ortama uygun slog handler'ı kurar.
//
//	production : JSON çıktı, Info seviyesi.
//	diğer      : renkli metin, Debug seviyesi.
//
// JSON handler'ın zaman damgası biçimi slog'un varsayılanıdır (RFC3339 +
// nanosaniye) ve ayrıca ayarlanmaz.
func newLogger(env string, w io.Writer) *slog.Logger {
	// EqualFold: APP_ENV=Production yazıldığında sessizce geliştirme moduna
	// düşmektense büyük/küçük harf farkı önemsiz olsun.
	if strings.EqualFold(env, appEnvProd) {
		// Prod'da Debug seviyesi hem gürültü hem maliyettir; Info yeterlidir.
		return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	return slog.New(tint.NewTextHandler(w, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: timeFormat,
		NoColor:    !isTerminal(w),
	}))
}

// isTerminal, w hedefinin bir terminal olup olmadığını bildirir. Terminal
// olmayan hedeflere (dosya, pipe, docker logs) renk kodu yazmak gürültüdür ve
// log satırlarını okunamaz hale getirir.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}
