package i18n

import "testing"

func TestLoad(t *testing.T) {
	for _, l := range SupportedLocales {
		tr, err := Load(l)
		if err != nil {
			t.Fatalf("Load(%s): %v", l, err)
		}
		if tr.Locale() != l {
			t.Fatalf("Load(%s) returned locale %s", l, tr.Locale())
		}
		if tr.T("hero.title") == "" {
			t.Errorf("Load(%s): hero.title is empty", l)
		}
	}
}

func TestMissingKeyReturnsKey(t *testing.T) {
	tr, err := Load(DefaultLocale)
	if err != nil {
		t.Fatal(err)
	}
	if got := tr.T("no.such.key"); got != "no.such.key" {
		t.Fatalf("expected key echo, got %q", got)
	}
}

func TestParseLocale(t *testing.T) {
	if ParseLocale("en") != "en" {
		t.Fatal("expected en")
	}
	if ParseLocale("fr") != DefaultLocale {
		t.Fatal("expected fallback to default")
	}
}
