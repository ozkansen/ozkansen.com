// Package assets, statik varlıkların sunulmasını sağlar.
//
// http.FileServer'ın bu uygulama için üç istenmeyen varsayılanını düzeltir:
//
//   - Dizin listeleme kapatılır; aksi halde /static/js/ tüm varlıkları listeler.
//   - Sürüm numarası taşıyan varlıklar immutable cache'lenir; içerik ad zaten
//     sürümlendiği için yeniden doğrulamaya gerek yoktur.
//   - Varlık kökü çalışma dizinine bağlı olmaktan çıkar ve başlangıçta
//     doğrulanır; yanlış dizinden çalıştırmak artık sessiz 404'ler değildir.
package assets

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
)

// DefaultRoot, ortam değişkeni verilmediğinde kullanılan statik dizin.
const DefaultRoot = "./static"

const (
	// immutableCache, adında sürüm numarası taşıyan varlıklar içindir: içerik
	// dosya adı değişmeden değişmez, bu yüzden tarayıcı bir yıl boyunca
	// önbellekleyebilir.
	immutableCache = "public, max-age=31536000, immutable"

	// revalidateCache, sürüm taşımayan varlıklar içindir: her zaman taze
	// sunulur, Last-Modified/ETag üzerinden 304 dönebilir.
	revalidateCache = "public, max-age=0, must-revalidate"
)

// New, root dizinindeki statik varlıkları sunan handler'ı döndürür.
// root boşsa DefaultRoot kullanılır.
//
// root erişilemezse hata döner. Bu bilinçli bir tercihtir: varlıklar
// sunulamayacaksa bunu başlangıçta bildirmek, sunucuyu sağlıklı görünür
// başlatıp her sayfa yüklemesinde sessiz 404 üretmekten iyidir.
func New(root string) (http.Handler, error) {
	if root == "" {
		root = DefaultRoot
	}

	// os.DirFS kök dışına çıkan yolları da reddeder, dolayısıyla path traversal
	// koruması dosya sistemi seviyesinde gelir.
	fsys := os.DirFS(root)

	st, err := fs.Stat(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("statik dizin okunamadı (STATIC_DIR=%q): %w", root, err)
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("statik yol bir dizin değil (STATIC_DIR=%q)", root)
	}

	return newHandler(fsys), nil
}

// newHandler, verilen dosya sistemini sunar. Dosya sistemi parametre olarak
// alındığı için davranış gerçek dosya sistemine ihtiyaç duymadan test edilir.
func newHandler(fsys fs.FS) http.Handler {
	fileServer := http.FileServerFS(fsys)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := fsPath(r.URL.Path)
		st, err := fs.Stat(fsys, name)

		switch {
		case err != nil:
			// Dosya yok: FileServer'ın 404'ünü üretsin. net/http hata yolunda
			// Cache-Control'ı zaten siler.
		case st.IsDir():
			// Dizinler listelenmez. http.Error, net/http'in aksine
			// Cache-Control'ı silmediği için burada başlık hiç set edilmemeli:
			// aksi halde bir hata yanıtı bir yıl cache'lenir.
			http.NotFound(w, r)
			return
		default:
			w.Header().Set("Cache-Control", cacheControl(name))
		}

		fileServer.ServeHTTP(w, r)
	})
}

// fsPath, gelen URL yolunu fs.FS'in beklediği biçime çevirir. fs.FS yolları
// köksüzdür: "/css/styles.css" -> "css/styles.css", "/" -> ".".
func fsPath(urlPath string) string {
	p := strings.TrimPrefix(path.Clean("/"+urlPath), "/")
	if p == "" {
		return "."
	}
	return p
}

// cacheControl, varlığın fs.FS yoluna göre cache başlığını seçer. js/ altındaki
// dosyalar sürüm numarası taşır (htmx_4.0.0.min.js); CSS ve diğerleri taşımaz
// ve içerikleri güncellemeyle geldiği için yeniden doğrulanmalıdır.
func cacheControl(name string) string {
	if strings.HasPrefix(name, "js/") {
		return immutableCache
	}
	return revalidateCache
}
