package undo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ralf/obsidian-daily-macos/internal/config"
)

func tmpDir(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(
		os.TempDir(), "undo-test-"+name,
	)
	os.RemoveAll(dir)
	os.MkdirAll(dir, 0o755)
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func tmpVault(t *testing.T, name string) string {
	t.Helper()
	dir := tmpDir(t, name)
	os.MkdirAll(
		filepath.Join(dir, ".obsidian"), 0o755,
	)
	return dir
}

func date(y, m, d int) time.Time {
	return time.Date(
		y, time.Month(m), d, 0, 0, 0, 0, time.UTC,
	)
}

func TestRecordAndReadMulti(t *testing.T) {
	root := tmpVault(t, "record-multi")
	vault := &config.Vault{Root: root}
	dest := filepath.Join(root, "undo.json")

	before := "old content"
	err := RecordBeforeMultiTo(
		vault, date(2026, 8, 20),
		[]FileState{
			{
				Path:   filepath.Join(root, "note.md"),
				Before: &before,
			},
		},
		dest,
	)
	if err != nil {
		t.Fatal(err)
	}
	data, e := os.ReadFile(dest)
	if e != nil {
		t.Fatal(e)
	}
	var rec undoRecordMulti
	if e := json.Unmarshal(data, &rec); e != nil {
		t.Fatal(e)
	}
	if rec.Vault != root {
		t.Fatalf("vault = %q", rec.Vault)
	}
	if rec.Date != "2026-08-20" {
		t.Fatalf("date = %q", rec.Date)
	}
	if len(rec.Files) != 1 {
		t.Fatalf("files = %d", len(rec.Files))
	}
	if rec.Files[0].Before == nil ||
		*rec.Files[0].Before != "old content" {
		t.Fatalf("before = %v", rec.Files[0].Before)
	}
}

func TestRecordNilBefore(t *testing.T) {
	root := tmpVault(t, "record-nil")
	vault := &config.Vault{Root: root}
	dest := filepath.Join(root, "undo.json")

	err := RecordBeforeMultiTo(
		vault, date(2026, 8, 20),
		[]FileState{
			{
				Path:   filepath.Join(root, "new.md"),
				Before: nil,
			},
		},
		dest,
	)
	if err != nil {
		t.Fatal(err)
	}
	data, e := os.ReadFile(dest)
	if e != nil {
		t.Fatal(e)
	}
	var rec undoRecordMulti
	json.Unmarshal(data, &rec)
	if rec.Files[0].Before != nil {
		t.Fatal("before should be nil for new file")
	}
}

func TestDiscardAt(t *testing.T) {
	root := tmpDir(t, "discard")
	dest := filepath.Join(root, "undo.json")
	os.WriteFile(dest, []byte("{}"), 0o600)
	DiscardAt(dest)
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatal("undo file should be deleted")
	}
}

func TestUndoLastFromRestores(t *testing.T) {
	root := tmpVault(t, "undo-restore")
	vault := &config.Vault{Root: root}
	note := filepath.Join(root, "note.md")

	// Set up WriteAtomicPublicFn.
	WriteAtomicPublicFn = func(
		vaultRoot, path, content string,
	) *config.VaultError {
		if e := os.WriteFile(
			path, []byte(content), 0o644,
		); e != nil {
			return config.NewVaultError("io", e.Error())
		}
		return nil
	}
	defer func() { WriteAtomicPublicFn = nil }()

	// Write mutated content.
	os.WriteFile(
		note, []byte("mutated"), 0o644,
	)

	// Record undo.
	dest := filepath.Join(root, "undo.json")
	before := "original"
	RecordBeforeMultiTo(
		vault, date(2026, 8, 20),
		[]FileState{
			{Path: note, Before: &before},
		},
		dest,
	)

	// Undo.
	called := false
	readFn := func(
		v *config.Vault, d time.Time,
	) (interface{}, *config.VaultError) {
		called = true
		return "snapshot", nil
	}
	result, err := UndoLastFrom(vault, dest, readFn)
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("readSnapshotFn not called")
	}
	if result != "snapshot" {
		t.Fatalf("result = %v", result)
	}

	// File restored.
	data, e := os.ReadFile(note)
	if e != nil {
		t.Fatal(e)
	}
	if string(data) != "original" {
		t.Fatalf("content = %q", string(data))
	}

	// Undo file removed.
	if _, e := os.Stat(dest); !os.IsNotExist(e) {
		t.Fatal("undo file should be removed")
	}
}

func TestUndoLastFromNothingToUndo(t *testing.T) {
	root := tmpVault(t, "undo-nothing")
	vault := &config.Vault{Root: root}
	dest := filepath.Join(root, "nonexistent.json")

	readFn := func(
		v *config.Vault, d time.Time,
	) (interface{}, *config.VaultError) {
		return nil, nil
	}
	_, err := UndoLastFrom(vault, dest, readFn)
	if err == nil {
		t.Fatal("expected 'nothing to undo' error")
	}
}

func TestUndoLastFromWrongVault(t *testing.T) {
	root := tmpVault(t, "undo-wrong-vault")
	vault := &config.Vault{Root: root}
	dest := filepath.Join(root, "undo.json")

	// Write undo for a different vault.
	rec := undoRecordMulti{
		Vault: "/some/other/vault",
		Date:  "2026-08-20",
		Files: []undoFile{
			{Path: "/some/other/vault/note.md"},
		},
	}
	data, _ := json.Marshal(rec)
	os.WriteFile(dest, data, 0o600)

	readFn := func(
		v *config.Vault, d time.Time,
	) (interface{}, *config.VaultError) {
		return nil, nil
	}
	_, err := UndoLastFrom(vault, dest, readFn)
	if err == nil {
		t.Fatal("expected vault mismatch error")
	}
}

func TestUndoLastFromDeletesNewFile(t *testing.T) {
	root := tmpVault(t, "undo-delete")
	vault := &config.Vault{Root: root}
	note := filepath.Join(root, "new-note.md")
	dest := filepath.Join(root, "undo.json")

	WriteAtomicPublicFn = func(
		vaultRoot, path, content string,
	) *config.VaultError {
		if e := os.WriteFile(
			path, []byte(content), 0o644,
		); e != nil {
			return config.NewVaultError("io", e.Error())
		}
		return nil
	}

	// Create the file that didn't exist before.
	os.WriteFile(note, []byte("new"), 0o644)

	// Record undo with nil before (file was new).
	rec := undoRecordMulti{
		Vault: root,
		Date:  "2026-08-20",
		Files: []undoFile{
			{Path: note, Before: nil},
		},
	}
	data, _ := json.Marshal(rec)
	os.WriteFile(dest, data, 0o600)

	readFn := func(
		v *config.Vault, d time.Time,
	) (interface{}, *config.VaultError) {
		return nil, nil
	}
	_, err := UndoLastFrom(vault, dest, readFn)
	if err != nil {
		t.Fatal(err)
	}
	if _, e := os.Stat(note); !os.IsNotExist(e) {
		t.Fatal("new file should be removed on undo")
	}
}

func TestUndoFilePermissions(t *testing.T) {
	root := tmpVault(t, "undo-perms")
	vault := &config.Vault{Root: root}
	dest := filepath.Join(root, "undo.json")

	before := "content"
	RecordBeforeMultiTo(
		vault, date(2026, 8, 20),
		[]FileState{
			{
				Path:   filepath.Join(root, "note.md"),
				Before: &before,
			},
		},
		dest,
	)

	info, e := os.Stat(dest)
	if e != nil {
		t.Fatal(e)
	}
	perm := info.Mode().Perm()
	if perm != 0o600 {
		t.Fatalf("perm = %o, want 600", perm)
	}
}
