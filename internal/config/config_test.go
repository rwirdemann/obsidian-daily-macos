package config

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

var tempSeq uint64

func tmpVault(t *testing.T, name string) string {
	t.Helper()
	n := atomic.AddUint64(&tempSeq, 1)
	dir := filepath.Join(
		os.TempDir(),
		"obsidian-daily-go-"+name+
			"-"+time.Now().Format("20060102150405")+
			"-"+itoa(n),
	)
	os.RemoveAll(dir)
	os.MkdirAll(filepath.Join(dir, ".obsidian"), 0o755)
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func itoa(n uint64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func date(y, m, d int) time.Time {
	return time.Date(
		y, time.Month(m), d, 0, 0, 0, 0, time.UTC,
	)
}

func TestDefaultsWhenSettingsMissing(t *testing.T) {
	root := tmpVault(t, "defaults")
	vault := &Vault{Root: root}
	cfg, err := vault.DailyNotesConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Folder != "" {
		t.Fatalf("folder = %q", cfg.Folder)
	}
	if cfg.Format != "YYYY-MM-DD" {
		t.Fatalf("format = %q", cfg.Format)
	}
	if cfg.Template != "" {
		t.Fatalf("template = %q", cfg.Template)
	}
	path, verr := vault.DailyNotePath(cfg, date(2026, 8, 20))
	if verr != nil {
		t.Fatal(verr)
	}
	want := filepath.Join(root, "2026-08-20.md")
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
}

func TestDottedDateFormatGetsmdSuffix(t *testing.T) {
	root := tmpVault(t, "dotted")
	os.WriteFile(
		filepath.Join(root, ".obsidian", "daily-notes.json"),
		[]byte(`{"folder":"Daily","format":"DD.MM.YYYY"}`),
		0o644,
	)
	vault := &Vault{Root: root}
	cfg, _ := vault.DailyNotesConfig()
	path, err := vault.DailyNotePath(cfg, date(2026, 8, 20))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "Daily", "20.08.2026.md")
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
}

func TestYearFirstDottedFormatGetsMdSuffix(t *testing.T) {
	root := tmpVault(t, "dotted-year")
	os.WriteFile(
		filepath.Join(root, ".obsidian", "daily-notes.json"),
		[]byte(`{"format":"YYYY.MM.DD"}`),
		0o644,
	)
	vault := &Vault{Root: root}
	cfg, _ := vault.DailyNotesConfig()
	path, err := vault.DailyNotePath(cfg, date(2026, 8, 20))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "2026.08.20.md")
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
}

func TestExistingMdSuffixNotDoubled(t *testing.T) {
	root := tmpVault(t, "already-md")
	os.WriteFile(
		filepath.Join(root, ".obsidian", "daily-notes.json"),
		[]byte(
			`{"format":"YYYY-MM-DD.md",`+
				`"template":"Templates/Daily.md"}`,
		),
		0o644,
	)
	vault := &Vault{Root: root}
	cfg, _ := vault.DailyNotesConfig()
	path, err := vault.DailyNotePath(cfg, date(2026, 8, 20))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "2026-08-20.md")
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
}

func TestReadsFolderFormatAndNestedDate(t *testing.T) {
	root := tmpVault(t, "nested")
	os.WriteFile(
		filepath.Join(root, ".obsidian", "daily-notes.json"),
		[]byte(
			`{"folder":"Daily",`+
				`"format":"YYYY/MM/YYYY-MM-DD",`+
				`"template":"Templates/Daily"}`,
		),
		0o644,
	)
	vault := &Vault{Root: root}
	cfg, _ := vault.DailyNotesConfig()
	if cfg.Folder != "Daily" {
		t.Fatalf("folder = %q", cfg.Folder)
	}
	if cfg.Format != "YYYY/MM/YYYY-MM-DD" {
		t.Fatalf("format = %q", cfg.Format)
	}
	path, err := vault.DailyNotePath(cfg, date(2026, 8, 20))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(
		root, "Daily", "2026", "08", "2026-08-20.md",
	)
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
}

func TestStripsParentDirSegmentsFromFolder(t *testing.T) {
	root := tmpVault(t, "escape")
	os.WriteFile(
		filepath.Join(root, ".obsidian", "daily-notes.json"),
		[]byte(
			`{"folder":"../outside","format":"YYYY-MM-DD"}`,
		),
		0o644,
	)
	vault := &Vault{Root: root}
	cfg, _ := vault.DailyNotesConfig()
	if cfg.Folder != "outside" {
		t.Fatalf("folder = %q", cfg.Folder)
	}
	path, err := vault.DailyNotePath(cfg, date(2026, 8, 20))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "outside", "2026-08-20.md")
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
}

func TestResolveSanitizesArchivePattern(t *testing.T) {
	root := tmpVault(t, "archive-escape")
	vault, err := Resolve(root, "../outside/YYYY")
	if err != nil {
		t.Fatal(err)
	}
	if vault.Archive != "outside/YYYY" {
		t.Fatalf("archive = %q", vault.Archive)
	}
}

func TestResolveIgnoresBlankArchive(t *testing.T) {
	root := tmpVault(t, "archive-blank")
	vault, err := Resolve(root, "   ")
	if err != nil {
		t.Fatal(err)
	}
	if vault.Archive != "" {
		t.Fatalf("archive = %q", vault.Archive)
	}
}

func TestResolvesArchivePathWhenPrimaryMissing(t *testing.T) {
	root := tmpVault(t, "archive-resolve")
	os.WriteFile(
		filepath.Join(root, ".obsidian", "daily-notes.json"),
		[]byte(
			`{"folder":"dailies","format":"YYYY-MM-DD"}`,
		),
		0o644,
	)
	vault := &Vault{
		Root:    root,
		Archive: "dailies/_archive/YYYY",
	}
	cfg, _ := vault.DailyNotesConfig()
	d := date(2026, 8, 20)

	// Nothing exists: resolved falls back to primary.
	paths, err := vault.DailyNotePaths(cfg, d)
	if err != nil {
		t.Fatal(err)
	}
	primary := filepath.Join(
		root, "dailies", "2026-08-20.md",
	)
	if paths.Primary != primary {
		t.Fatalf("primary = %q", paths.Primary)
	}
	if paths.Resolved != paths.Primary {
		t.Fatalf(
			"resolved = %q, want primary", paths.Resolved,
		)
	}

	// Archive note exists: resolved points there.
	archived := filepath.Join(
		root, "dailies", "_archive", "2026", "2026-08-20.md",
	)
	os.MkdirAll(filepath.Dir(archived), 0o755)
	os.WriteFile(archived, []byte("- [ ] old\n"), 0o644)
	paths, _ = vault.DailyNotePaths(cfg, d)
	if paths.Resolved != archived {
		t.Fatalf("resolved = %q, want %q",
			paths.Resolved, archived)
	}
}

func TestPrimaryPathWinsOverArchive(t *testing.T) {
	root := tmpVault(t, "archive-primary-wins")
	os.WriteFile(
		filepath.Join(root, ".obsidian", "daily-notes.json"),
		[]byte(
			`{"folder":"dailies","format":"YYYY-MM-DD"}`,
		),
		0o644,
	)
	vault := &Vault{
		Root:    root,
		Archive: "dailies/_archive/YYYY",
	}
	cfg, _ := vault.DailyNotesConfig()
	d := date(2026, 8, 20)

	live := filepath.Join(root, "dailies", "2026-08-20.md")
	os.MkdirAll(filepath.Dir(live), 0o755)
	os.WriteFile(live, []byte("- [ ] live\n"), 0o644)

	archived := filepath.Join(
		root, "dailies", "_archive", "2026", "2026-08-20.md",
	)
	os.MkdirAll(filepath.Dir(archived), 0o755)
	os.WriteFile(
		archived, []byte("- [ ] archived\n"), 0o644,
	)

	paths, _ := vault.DailyNotePaths(cfg, d)
	if paths.Resolved != live {
		t.Fatalf("resolved = %q, want %q",
			paths.Resolved, live)
	}
}
