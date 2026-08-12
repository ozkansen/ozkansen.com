package content

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTestContent, verilen geçici dizinde test için içerik dosyaları oluşturur.
func writeTestContent(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	files := map[string]string{
		"settings.yaml":        "email: test@example.com\nstats:\n  - value: 34+\n    label_tr: Tamamlanan\n    label_en: Done\n",
		"about.yaml":           "paragraphs_tr:\n  - Merhaba\nparagraphs_en:\n  - Hello\nskills:\n  - Go\n",
		"services.yaml":        "services:\n  - icon: laptop\n    title_tr: Tasarım\n    title_en: Design\n    text_tr: Metin TR\n    text_en: Text EN\n",
		"testimonials.yaml":    "testimonials:\n  - name: Sarah\n    role_tr: CPO\n    role_en: CPO\n    img: /x.png\n    quote_tr: Alıntı TR\n    quote_en: Quote EN\n",
		"projects/novu/tr.md":  "---\ntitle: Novu TR\ntags: [SaaS]\nyear: 2025\nimage: /novu.png\n---\n## Açıklama\n\nİlk paragraf.\n",
		"projects/novu/en.md":  "---\ntitle: Novu EN\ntags: [SaaS]\nyear: 2025\nimage: /novu.png\n---\n## Desc\n\nFirst paragraph.\n",
		"projects/finlo/tr.md": "---\ntitle: Finlo TR\n---\nSadece TR var.\n",
		"blog/post1/tr.md":     "---\ntitle: Yazı TR\ntags: [Tasarım]\ndate: 8 Mart 2025\n---\nÖzet metin.\n",
		"blog/post1/en.md":     "---\ntitle: Post EN\ntags: [Design]\ndate: Mar 8, 2025\n---\nExcerpt text.\n",
	}

	for path, content := range files {
		full := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestLoad(t *testing.T) {
	dir := writeTestContent(t)
	s, err := Load(dir)
	if err != nil {
		t.Fatalf("Load hatası: %v", err)
	}

	if len(s.Projects) != 2 {
		t.Fatalf("beklenen 2 proje, %d geldi", len(s.Projects))
	}
	if len(s.Posts) != 1 {
		t.Fatalf("beklenen 1 yazı, %d geldi", len(s.Posts))
	}
	if len(s.Services) != 1 || len(s.Testimonials) != 1 {
		t.Fatalf("servis/yorum sayısı hatalı")
	}
	if s.Settings.Email != "test@example.com" {
		t.Fatalf("settings email hatalı: %q", s.Settings.Email)
	}
}

func TestYAMLFieldMapping(t *testing.T) {
	dir := writeTestContent(t)
	s, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	svc := s.Services[0]
	if svc.TitleTR != "Tasarım" || svc.TitleEN != "Design" {
		t.Errorf("servis title eşleşmedi: TR=%q EN=%q", svc.TitleTR, svc.TitleEN)
	}
	if svc.TextTR != "Metin TR" || svc.TextEN != "Text EN" {
		t.Errorf("servis text eşleşmedi: TR=%q EN=%q", svc.TextTR, svc.TextEN)
	}

	tm := s.Testimonials[0]
	if tm.RoleTR != "CPO" || tm.QuoteTR != "Alıntı TR" {
		t.Errorf("testimonial eşleşmedi: %+v", tm)
	}

	if len(s.About.ParagraphsTR) != 1 || s.About.ParagraphsTR[0] != "Merhaba" {
		t.Errorf("about paragraphs_tr eşleşmedi: %+v", s.About.ParagraphsTR)
	}
	if len(s.About.Skills) != 1 || s.About.Skills[0] != "Go" {
		t.Errorf("about skills eşleşmedi: %+v", s.About.Skills)
	}

	stats := s.Settings.Stats
	if len(stats) != 1 || stats[0].LabelTR != statLabelTR || stats[0].LabelEN != statLabelEN {
		t.Errorf("stat eşleşmedi: %+v", stats)
	}
}

func TestLocalization(t *testing.T) {
	dir := writeTestContent(t)
	s, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	novu, ok := findProject(s, "novu")
	if !ok {
		t.Fatal("novu projesi bulunamadı")
	}
	if got := novu.Title("tr"); got != "Novu TR" {
		t.Errorf("tr başlık: %q", got)
	}
	if got := novu.Title("en"); got != "Novu EN" {
		t.Errorf("en başlık: %q", got)
	}

	// markdown render edilmiş olmalı
	if !strings.Contains(string(novu.BodyHTML("tr")), "<h2>") {
		t.Errorf("markdown render edilmedi: %q", novu.BodyHTML("tr"))
	}

	// BodyText etiketsiz düz metin döndürmeli
	if got := novu.BodyText("tr"); strings.Contains(got, "<") {
		t.Errorf("BodyText HTML içermemeli: %q", got)
	}
}

func TestFallbackToEN(t *testing.T) {
	dir := writeTestContent(t)
	s, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	// finlo sadece tr.md içerir; en yoksa TR'ye düşmeli
	finlo, ok := findProject(s, "finlo")
	if !ok {
		t.Fatal("finlo projesi bulunamadı")
	}
	if got := finlo.Title("en"); got != "Finlo TR" {
		t.Errorf("fallback başarısız, beklenen Finlo TR, gelen %q", got)
	}
}

func findProject(s *Site, slug string) (*Project, bool) {
	for _, p := range s.Projects {
		if p.Slug == slug {
			return p, true
		}
	}
	return nil, false
}

const (
	statLabelTR = "Tamamlanan"
	statLabelEN = "Done"
)

func TestStatLabel(t *testing.T) {
	s := Stat{Value: "34+", LabelTR: statLabelTR, LabelEN: statLabelEN}
	if got := s.Label("tr"); got != statLabelTR {
		t.Errorf("tr: %q", got)
	}
	if got := s.Label("en"); got != statLabelEN {
		t.Errorf("en: %q", got)
	}
	// bilinmeyen locale TR'ye düşer
	if got := s.Label("fr"); got != statLabelTR {
		t.Errorf("fr fallback: %q", got)
	}
}
