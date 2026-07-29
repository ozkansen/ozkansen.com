package main

import (
	"log"
	"net/http"

	"ozkansen.com/internal/views/pages"

	"github.com/a-h/templ"
)

func main() {
	// 1. Statik dosyaların sunulması (CSS, resimler)
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// 2. Anasayfa Routing (Templ Bileşeni)
	http.Handle("/", templ.Handler(pages.Home("Özkan")))

	// 3. htmx Partial HTML Endpoint'i
	http.HandleFunc("/api/status", func(w http.ResponseWriter, r *r.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
			<div id="status-box" class="p-4 bg-emerald-950/60 border border-emerald-800 text-emerald-300 rounded-lg text-sm font-medium">
				🚀 Sunucu Aktif! HTMX isteği başarıyla işlendi ve partial HTML güncellendi.
			</div>
		`))
	})

	log.Println("Sunucu 8080 portunda çalışıyor: http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
