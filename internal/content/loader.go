package content

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	"gopkg.in/yaml.v3"
)

// DefaultDir, içerik dosyalarının bulunduğu varsayılan dizindir.
const DefaultDir = "content"

// Load, verilen dizinden tüm site içeriğini okur ve işler.
func Load(dir string) (*Site, error) {
	if dir == "" {
		dir = DefaultDir
	}

	s := &Site{}
	if err := loadYAML(dir, "settings.yaml", &s.Settings); err != nil {
		return nil, err
	}
	if err := loadYAML(dir, "about.yaml", &s.About); err != nil {
		return nil, err
	}

	var svc struct {
		Services []*Service `yaml:"services"`
	}
	if err := loadYAML(dir, "services.yaml", &svc); err != nil {
		return nil, err
	}
	s.Services = svc.Services

	var tm struct {
		Testimonials []*Testimonial `yaml:"testimonials"`
	}
	if err := loadYAML(dir, "testimonials.yaml", &tm); err != nil {
		return nil, err
	}
	s.Testimonials = tm.Testimonials

	projects, err := loadSlugs(filepath.Join(dir, "projects"))
	if err != nil {
		return nil, fmt.Errorf("content: projeler yüklenemedi: %w", err)
	}
	s.Projects = toProjects(projects)

	posts, err := loadSlugs(filepath.Join(dir, "blog"))
	if err != nil {
		return nil, fmt.Errorf("content: blog yazıları yüklenemedi: %w", err)
	}
	s.Posts = toPosts(posts)

	return s, nil
}

// LoadDir, içerik dizinini döndürür; CONTENT_DIR env ile override edilebilir.
func LoadDir() string {
	if dir := os.Getenv("CONTENT_DIR"); dir != "" {
		return dir
	}
	return DefaultDir
}

// loadYAML, dizindeki tek bir YAML dosyasını hedefe ayrıştırır.
// İçerik dizini uygulama yapılandırmasından gelir (repo içi, güvenilir).
func loadYAML(dir, name string, dst any) error {
	data, err := os.ReadFile(filepath.Join(dir, name)) //nolint:gosec // içerik dizini güvenilir
	if err != nil {
		return fmt.Errorf("content: %s okunamadı: %w", name, err)
	}
	if err := yaml.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("content: %s ayrıştırılamadı: %w", name, err)
	}
	return nil
}

// loadSlugs, dizin altındaki her alt dizin için tr.md ve en.md dosyalarını
// okuyarak bir içerik listesi üretir.
func loadSlugs(dir string) ([]Item, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	items := make([]Item, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		slug := entry.Name()
		item, err := loadItem(filepath.Join(dir, slug), slug)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

// Item, proje ve blog yazılarının ortak okuma biçimidir.
type Item struct {
	Slug   string
	MetaTR itemMeta
	MetaEN itemMeta
	BodyTR string
	BodyEN string
}

// itemMeta, frontmatter'daki ortak meta alanlarıdır.
// Her dil dosyası (tr.md/en.md) kendi meta'sını taşır.
type itemMeta struct {
	Title    string   `yaml:"title"`
	Tags     []string `yaml:"tags"`
	Category string   `yaml:"category"`
	Year     string   `yaml:"year"`
	Image    string   `yaml:"image"`
	Date     string   `yaml:"date"`
}

// loadItem, bir slug dizinindeki tr.md ve en.md dosyalarını okur.
// Bir dil dosyası eksikse diğerine düşülür (fallback).
// Dosya yolları yalnızca slug dizinlerinden gelir (repo içi, güvenilir).
func loadItem(dir, slug string) (Item, error) {
	var item Item
	item.Slug = slug

	trData, err := os.ReadFile(filepath.Join(dir, "tr.md")) //nolint:gosec // içerik dizini güvenilir
	switch {
	case err == nil:
		item.MetaTR, item.BodyTR, err = parseMarkdown(trData)
		if err != nil {
			return item, fmt.Errorf("content: %s/tr.md: %w", slug, err)
		}
	case !os.IsNotExist(err):
		return item, fmt.Errorf("content: %s/tr.md okunamadı: %w", slug, err)
	}

	enData, err := os.ReadFile(filepath.Join(dir, "en.md")) //nolint:gosec // içerik dizini güvenilir
	switch {
	case err == nil:
		item.MetaEN, item.BodyEN, err = parseMarkdown(enData)
		if err != nil {
			return item, fmt.Errorf("content: %s/en.md: %w", slug, err)
		}
	case !os.IsNotExist(err):
		return item, fmt.Errorf("content: %s/en.md okunamadı: %w", slug, err)
	}

	return item, nil
}

// toProjects, okunan Item listesini Project tipine dönüştürür.
func toProjects(items []Item) []*Project {
	out := make([]*Project, 0, len(items))
	for i := range items {
		it := &items[i]
		out = append(out, &Project{
			Slug:     it.Slug,
			TitleTR:  it.MetaTR.Title,
			TitleEN:  it.MetaEN.Title,
			TagsTR:   it.MetaTR.Tags,
			TagsEN:   it.MetaEN.Tags,
			Category: firstNonEmpty(it.MetaTR.Category, it.MetaEN.Category),
			Year:     firstNonEmpty(it.MetaTR.Year, it.MetaEN.Year),
			Image:    firstNonEmpty(it.MetaTR.Image, it.MetaEN.Image),
			BodyTR:   it.BodyTR,
			BodyEN:   it.BodyEN,
		})
	}
	return out
}

// toPosts, okunan Item listesini Post tipine dönüştürür.
func toPosts(items []Item) []*Post {
	out := make([]*Post, 0, len(items))
	for i := range items {
		it := &items[i]
		out = append(out, &Post{
			Slug:    it.Slug,
			TitleTR: it.MetaTR.Title,
			TitleEN: it.MetaEN.Title,
			TagsTR:  it.MetaTR.Tags,
			TagsEN:  it.MetaEN.Tags,
			DateTR:  it.MetaTR.Date,
			DateEN:  it.MetaEN.Date,
			Image:   firstNonEmpty(it.MetaTR.Image, it.MetaEN.Image),
			BodyTR:  it.BodyTR,
			BodyEN:  it.BodyEN,
		})
	}
	return out
}

// firstNonEmpty, iki değerden boş olmayan ilkini döndürür.
func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// parseMarkdown, frontmatter'ı (--- ile ayrılmış YAML) ve markdown gövdeyi ayırır.
// Gövde goldmark ile HTML'e işlenir.
func parseMarkdown(data []byte) (itemMeta, string, error) {
	var meta itemMeta
	body := string(data)

	text := strings.TrimLeft(string(data), "\ufeff \t\r\n")
	rest, hasFrontmatter := strings.CutPrefix(text, "---")
	if hasFrontmatter {
		fm, bodyAfter, ok := strings.Cut(rest, "\n---")
		if !ok {
			return meta, "", errors.New("frontmatter kapanışı bulunamadı")
		}
		if err := yaml.Unmarshal([]byte(strings.TrimPrefix(fm, "\n")), &meta); err != nil {
			return meta, "", fmt.Errorf("frontmatter ayrıştırılamadı: %w", err)
		}
		body = bodyAfter
	}

	var buf bytes.Buffer
	if err := goldmark.New().Convert([]byte(body), &buf); err != nil {
		return meta, "", fmt.Errorf("markdown işlenemedi: %w", err)
	}
	return meta, buf.String(), nil
}

// stripTags, HTML etiketlerini kaldırır (kart özetleri için).
func stripTags(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	return b.String()
}
