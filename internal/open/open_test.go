package open

import (
	"os"
	"path/filepath"
	"testing"
)

func tmpVault(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(
		os.TempDir(), "open-test-"+name,
	)
	os.RemoveAll(dir)
	os.MkdirAll(
		filepath.Join(dir, ".obsidian"), 0o755,
	)
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func TestOpenURIVaultRelative(t *testing.T) {
	root := tmpVault(t, "vault-rel")
	note := filepath.Join(root, "daily", "2026-08-20.md")
	os.MkdirAll(filepath.Dir(note), 0o755)
	os.WriteFile(note, []byte(""), 0o644)

	uri := OpenURI(root, note)
	vaultName := filepath.Base(root)
	if !containsStr(uri, "vault=") {
		t.Fatalf("missing vault= in %q", uri)
	}
	if !containsStr(uri, vaultName) {
		t.Fatalf("missing vault name in %q", uri)
	}
	if !containsStr(uri, "file=") {
		t.Fatalf("missing file= in %q", uri)
	}
}

func TestOpenURIFallsBackToPath(t *testing.T) {
	note := "/some/random/path/note.md"
	uri := OpenURI("/nonexistent", note)
	if !containsStr(uri, "path=") {
		t.Fatalf("should fallback to path=: %q", uri)
	}
}

func TestPercentEncodePathSimple(t *testing.T) {
	got := percentEncodePath("hello/world")
	if got != "hello/world" {
		t.Fatalf("got %q", got)
	}
}

func TestPercentEncodePathSpaces(t *testing.T) {
	got := percentEncodePath("my vault/daily note.md")
	if !containsStr(got, "%20") {
		t.Fatalf(
			"spaces should be encoded: %q", got,
		)
	}
	if containsStr(got, " ") {
		t.Fatalf(
			"should not contain raw spaces: %q", got,
		)
	}
}

func TestPercentEncodePathPreservesSlash(t *testing.T) {
	got := percentEncodePath("a/b/c")
	if got != "a/b/c" {
		t.Fatalf("slashes should be preserved: %q", got)
	}
}

func TestPercentEncodePathSpecialChars(t *testing.T) {
	got := percentEncodePath("name@#$")
	if !containsStr(got, "%40") ||
		!containsStr(got, "%23") ||
		!containsStr(got, "%24") {
		t.Fatalf("special chars not encoded: %q", got)
	}
}

func TestPercentEncodePathUnreservedCharsPreserved(
	t *testing.T,
) {
	got := percentEncodePath("a-b_c.d~e")
	if got != "a-b_c.d~e" {
		t.Fatalf(
			"unreserved chars should be preserved: %q",
			got,
		)
	}
}

func TestVaultRelativePartsValid(t *testing.T) {
	root := tmpVault(t, "rel-parts")
	note := filepath.Join(root, "notes", "test.md")
	os.MkdirAll(filepath.Dir(note), 0o755)
	os.WriteFile(note, []byte(""), 0o644)

	vault, file, ok := vaultRelativeParts(root, note)
	if !ok {
		t.Fatal("should succeed")
	}
	if vault != filepath.Base(root) {
		t.Fatalf("vault = %q", vault)
	}
	if file != filepath.Join("notes", "test.md") {
		t.Fatalf("file = %q", file)
	}
}

func TestVaultRelativePartsNoObsidianDir(t *testing.T) {
	dir := filepath.Join(
		os.TempDir(), "open-test-no-obs",
	)
	os.RemoveAll(dir)
	os.MkdirAll(dir, 0o755)
	defer os.RemoveAll(dir)

	note := filepath.Join(dir, "test.md")
	os.WriteFile(note, []byte(""), 0o644)

	_, _, ok := vaultRelativeParts(dir, note)
	if ok {
		t.Fatal(
			"should fail without .obsidian dir",
		)
	}
}

func TestVaultRelativePartsOutside(t *testing.T) {
	root := tmpVault(t, "rel-outside")
	_, _, ok := vaultRelativeParts(
		root, "/completely/outside/note.md",
	)
	if ok {
		t.Fatal("should fail for outside path")
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
