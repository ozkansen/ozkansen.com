package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"regexp"
	"strings"
	"testing"
	"time"
)

// logLine, logger'a bir satır yazdırıp çıktıyı döndürür.
func logLine(t *testing.T, env string, level slog.Level) string {
	t.Helper()

	var buf bytes.Buffer
	newLogger(env, &buf).Log(t.Context(), level, "test mesajı")
	return strings.TrimSpace(buf.String())
}

func TestNewLoggerProductionUsesJSON(t *testing.T) {
	out := logLine(t, appEnvProd, slog.LevelInfo)

	var record map[string]any
	if err := json.Unmarshal([]byte(out), &record); err != nil {
		t.Fatalf("çıktı JSON değil: %v\n%s", err, out)
	}
	if record["msg"] != "test mesajı" {
		t.Errorf("msg = %v", record["msg"])
	}
}

func TestNewLoggerDevelopmentUsesText(t *testing.T) {
	out := logLine(t, "development", slog.LevelInfo)

	if json.Valid([]byte(out)) {
		t.Errorf("geliştirme çıktısı JSON olmamalı: %s", out)
	}
	if !strings.Contains(out, "test mesajı") {
		t.Errorf("mesaj yok: %s", out)
	}
}

func TestLogLevelPerEnvironment(t *testing.T) {
	tests := []struct {
		env      string
		wantLogs bool
	}{
		{appEnvProd, false},   // prod: Debug bastırılır
		{"development", true}, // geliştirme: Debug görünür
		{"", true},            // tanımsız değer geliştirme sayılır
		{"PRODUCTION", false}, // büyük harf ayrımı yok
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			out := logLine(t, tt.env, slog.LevelDebug)

			if logged := out != ""; logged != tt.wantLogs {
				t.Errorf("APP_ENV=%q: Debug loglandı=%v, beklenen %v (%q)", tt.env, logged, tt.wantLogs, out)
			}
		})
	}
}

// TestLogTimestampIsUnambiguous, zaman damgasında yıl ve saat diliminin
// bulunduğunu doğrular. Bu olmadan loglar toplanamaz ve saat dilimleri
// arasında karşılaştırılamaz.
func TestLogTimestampIsUnambiguous(t *testing.T) {
	// Kesirli saniye: geliştirmede milisaniye, JSON'da nanosaniye.
	stamp := regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+(?:Z|[+-]\d{2}:\d{2})`)

	now := time.Now()
	year := now.Format("2006")

	for _, env := range []string{appEnvProd, "development"} {
		t.Run(env, func(t *testing.T) {
			out := logLine(t, env, slog.LevelInfo)

			if !stamp.MatchString(out) {
				t.Fatalf("zaman damgası yıl+saat dilimi içermiyor: %s", out)
			}
			if !strings.Contains(out, year) {
				t.Errorf("yıl %s bulunamadı: %s", year, out)
			}
		})
	}
}

func TestNonTerminalOutputHasNoColor(t *testing.T) {
	// bytes.Buffer terminal değildir; renk kodu (ESC) içermemeli.
	out := logLine(t, "development", slog.LevelInfo)

	if strings.ContainsRune(out, 0x1b) {
		t.Errorf("terminal olmayan çıktıda renk kodu var: %q", out)
	}
}

func TestIsTerminal(t *testing.T) {
	if isTerminal(&bytes.Buffer{}) {
		t.Error("bytes.Buffer terminal sayılmamalı")
	}
}

// TestProductionDurationIsReadable, JSON çıktısında sürenin okunur biçimde
// olduğunu doğrular. Varsayılan davranış çıplak int64 nanosaniye yazar
// ("duration":81009); tüketicinin birimi bilmesi gerekir.
func TestProductionDurationIsReadable(t *testing.T) {
	var buf bytes.Buffer
	newLogger(appEnvProd, &buf).Log(t.Context(), slog.LevelInfo, "test", slog.Duration("duration", 81*time.Microsecond))

	var record struct {
		Duration string `json:"duration"`
	}
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("JSON değil: %v\n%s", err, buf.String())
	}

	if record.Duration != "81µs" {
		t.Errorf("duration = %q, beklenen %q", record.Duration, "81µs")
	}
}

// TestNonDurationAttrsUnaffected, ReplaceAttr'ın yalnızca süreleri etkilediğini
// ve diğer alanlara dokunmadığını doğrular.
func TestNonDurationAttrsUnaffected(t *testing.T) {
	var buf bytes.Buffer
	newLogger(appEnvProd, &buf).Log(t.Context(), slog.LevelInfo, "test",
		slog.Int("status", 200),
		slog.String("path", "/"),
		slog.Int64("bytes", 42),
	)

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("JSON değil: %v", err)
	}
	if record["status"] != float64(200) {
		t.Errorf("status = %v, beklenen 200 (sayı olarak kalmalı)", record["status"])
	}
	if record["path"] != "/" {
		t.Errorf("path = %v", record["path"])
	}
	if record["bytes"] != float64(42) {
		t.Errorf("bytes = %v, beklenen 42", record["bytes"])
	}
}
