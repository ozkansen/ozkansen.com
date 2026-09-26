package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recoverer, handler paniklerini yakalar, paniği loglar ve istemciye
// 500 Internal Server Error döner.
//
// Neden gerekli: net/http paniği yakalar ama yalnızca kendi stderr'ine ham
// bir yığın izi yazar. Bu, üretimde JSON çıktı alan log toplama altyapısını
// atlar. Panikler ayrıca bağlantıyı yarım bırakır ve istek logu düşer.
//
// chi'nin middleware.Recoverer'ı stderr'e yazdığı ve çıktı hedefi
// değiştirilemediği için (recovererErrorWriter export edilmemiş) burada kendi
// sürümü kullanılır; böylece panik de Logger ile aynı yapılandırılmış loga
// yazılır.
//
// Zincir sırası önemlidir: Recoverer, Logger'ın İÇİNDE olmalıdır
// (Logger(Recoverer(handler))). Böylece panik yakalandığında Logger'ın
// next.ServeHTTP çağrısı normal döner ve istek logu da status=500 ile yazılır.
// Ters sırada (Recoverer(Logger(handler))) istek logu tamamen kaybolur.
func Recoverer(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				rvr := recover()
				if rvr == nil {
					return
				}
				// http.ErrAbortHandler, net/http'ye "bağlantıyı loglamadan
				// kapat" demektir; yakalamak sessizce yutulmasına yol açar.
				if err, ok := rvr.(error); ok && errors.Is(err, http.ErrAbortHandler) {
					panic(rvr)
				}

				logger.ErrorContext(r.Context(), "Handler paniği",
					slog.Any("panic", rvr),
					slog.String("stack", string(debug.Stack())),
				)

				// Yanıt henüz gönderilmediyse 500 döner. Gönderildiyse durum
				// kodu artık değiştirilemez; bu durumda bağlantı yarım kalır
				// ve istemci hatanın ayrıntısını alamaz.
				w.WriteHeader(http.StatusInternalServerError)
			}()

			next.ServeHTTP(w, r)
		})
	}
}
