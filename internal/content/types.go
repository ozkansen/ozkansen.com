// Package content, sitedeki dinamik içeriği (projeler, blog, hizmetler,
// yorumlar, ayarlar) repo içindeki markdown/YAML dosyalarından yükler.
package content

import (
	"html"
	"html/template"
	"strings"
)

// Site, dosya tabanlı tüm site içeriğini bir arada tutar.
type Site struct {
	Settings     Settings
	Services     []*Service
	About        About
	Testimonials []*Testimonial
	Projects     []*Project
	Posts        []*Post
}

// Settings, sitenin global değerlerini tutar.
type Settings struct {
	Email    string
	Linkedin string
	Github   string
	Stats    []Stat
}

// Stat, hero bölümündeki tek bir istatistik değeridir.
type Stat struct {
	Value   string `yaml:"value"`
	LabelTR string `yaml:"label_tr"`
	LabelEN string `yaml:"label_en"`
}

// Label, locale'e göre istatistik etiketini döndürür; TR yoksa EN'ye düşer.
func (s Stat) Label(locale string) string {
	if locale == "en" && s.LabelEN != "" {
		return s.LabelEN
	}
	if s.LabelTR != "" {
		return s.LabelTR
	}
	return s.LabelEN
}

// Service, bir hizmet kartıdır.
type Service struct {
	Icon    string `yaml:"icon"`
	TitleTR string `yaml:"title_tr"`
	TitleEN string `yaml:"title_en"`
	TextTR  string `yaml:"text_tr"`
	TextEN  string `yaml:"text_en"`
}

// Title, locale'e göre hizmet başlığını döndürür.
func (s *Service) Title(locale string) string {
	if locale == "en" && s.TitleEN != "" {
		return s.TitleEN
	}
	if s.TitleTR != "" {
		return s.TitleTR
	}
	return s.TitleEN
}

// Text, locale'e göre hizmet açıklamasını döndürür.
func (s *Service) Text(locale string) string {
	if locale == "en" && s.TextEN != "" {
		return s.TextEN
	}
	if s.TextTR != "" {
		return s.TextTR
	}
	return s.TextEN
}

// About, hakkında bölümünün içeriğidir.
type About struct {
	ParagraphsTR []string `yaml:"paragraphs_tr"`
	ParagraphsEN []string `yaml:"paragraphs_en"`
	Skills       []string `yaml:"skills"`
}

// Paragraphs, locale'e göre hakkında paragraflarını döndürür.
func (a About) Paragraphs(locale string) []string {
	if locale == "en" && len(a.ParagraphsEN) > 0 {
		return a.ParagraphsEN
	}
	if len(a.ParagraphsTR) > 0 {
		return a.ParagraphsTR
	}
	return a.ParagraphsEN
}

// Testimonial, tek bir müşteri yorumudur.
type Testimonial struct {
	Name    string `yaml:"name"`
	RoleTR  string `yaml:"role_tr"`
	RoleEN  string `yaml:"role_en"`
	Img     string `yaml:"img"`
	QuoteTR string `yaml:"quote_tr"`
	QuoteEN string `yaml:"quote_en"`
}

// Role, locale'e göre yorum sahibinin rolünü döndürür.
func (t *Testimonial) Role(locale string) string {
	if locale == "en" && t.RoleEN != "" {
		return t.RoleEN
	}
	if t.RoleTR != "" {
		return t.RoleTR
	}
	return t.RoleEN
}

// Quote, locale'e göre yorum metnini döndürür.
func (t *Testimonial) Quote(locale string) string {
	if locale == "en" && t.QuoteEN != "" {
		return t.QuoteEN
	}
	if t.QuoteTR != "" {
		return t.QuoteTR
	}
	return t.QuoteEN
}

// Project, slug'ı ile bir projeyi temsil eder.
type Project struct {
	Slug     string
	TitleTR  string
	TitleEN  string
	TagsTR   []string
	TagsEN   []string
	Category string
	Year     string
	Image    string
	BodyTR   string
	BodyEN   string
}

// Title, locale'e göre proje başlığını döndürür.
func (p *Project) Title(locale string) string {
	if locale == "en" && p.TitleEN != "" {
		return p.TitleEN
	}
	if p.TitleTR != "" {
		return p.TitleTR
	}
	return p.TitleEN
}

// Tags, locale'e göre proje etiketlerini döndürür.
func (p *Project) Tags(locale string) []string {
	if locale == "en" && len(p.TagsEN) > 0 {
		return p.TagsEN
	}
	if len(p.TagsTR) > 0 {
		return p.TagsTR
	}
	return p.TagsEN
}

// BodyHTML, locale'e göre proje gövdesinin işlenmiş (güvenli) HTML'ini döndürür.
// İçerik repo'dan (güvenilir) geldiği için auto-escape devre dışıdır.
func (p *Project) BodyHTML(locale string) template.HTML {
	body := p.BodyTR
	if locale == "en" && p.BodyEN != "" {
		body = p.BodyEN
	}
	return template.HTML(body) //nolint:gosec // içerik repo içi, güvenilir
}

// BodyText, locale'e göre proje gövdesinin düz metin halini döndürür
// (kartlarda kullanılır).
func (p *Project) BodyText(locale string) string {
	body := p.BodyTR
	if locale == "en" && p.BodyEN != "" {
		body = p.BodyEN
	}
	return strings.TrimSpace(html.UnescapeString(stripTags(body)))
}

// Post, slug'ı ile bir blog yazısını temsil eder.
type Post struct {
	Slug    string
	TitleTR string
	TitleEN string
	TagsTR  []string
	TagsEN  []string
	DateTR  string
	DateEN  string
	Image   string
	BodyTR  string
	BodyEN  string
}

// Title, locale'e göre yazı başlığını döndürür.
func (p *Post) Title(locale string) string {
	if locale == "en" && p.TitleEN != "" {
		return p.TitleEN
	}
	if p.TitleTR != "" {
		return p.TitleTR
	}
	return p.TitleEN
}

// Tags, locale'e göre yazı etiketlerini döndürür.
func (p *Post) Tags(locale string) []string {
	if locale == "en" && len(p.TagsEN) > 0 {
		return p.TagsEN
	}
	if len(p.TagsTR) > 0 {
		return p.TagsTR
	}
	return p.TagsEN
}

// Date, locale'e göre yazı tarihini döndürür.
func (p *Post) Date(locale string) string {
	if locale == "en" && p.DateEN != "" {
		return p.DateEN
	}
	if p.DateTR != "" {
		return p.DateTR
	}
	return p.DateEN
}

// BodyText, locale'e göre yazı gövdesinin düz metin özetini döndürür
// (kartlarda kullanılır).
func (p *Post) BodyText(locale string) string {
	body := p.BodyTR
	if locale == "en" && p.BodyEN != "" {
		body = p.BodyEN
	}
	return strings.TrimSpace(stripTags(body))
}
