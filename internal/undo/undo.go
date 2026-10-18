// Package undo implements a file-based single-slot undo for
// the last note mutation.
package undo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ralf/obsidian-daily-macos/internal/config"
)

type undoRecord struct {
	Vault  string `json:"vault"`
	Date   string `json:"date"`
	Path   string `json:"path"`
	Before string `json:"before"`
}

type undoFile struct {
	Path   string  `json:"path"`
	Before *string `json:"before"`
}

type undoRecordMulti struct {
	Vault string     `json:"vault"`
	Date  string     `json:"date"`
	Files []undoFile `json:"files"`
}

func undoPath() (string, *config.VaultError) {
	if p := os.Getenv("OBSIDIAN_DAILY_QS_UNDO_PATH"); p != "" {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			return trimmed, nil
		}
	}
	dir, err := cacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(
		dir, "obsidian-daily-qs", "last-undo.json",
	), nil
}

func cacheDir() (string, *config.VaultError) {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		p := strings.TrimSpace(xdg)
		if p != "" && filepath.IsAbs(p) {
			return p, nil
		}
	}
	if home := os.Getenv("HOME"); home != "" {
		p := strings.TrimSpace(home)
		if p != "" && filepath.IsAbs(p) {
			return filepath.Join(p, ".cache"), nil
		}
	}
	return "", config.NewVaultError(
		"io",
		"cannot locate the undo cache: "+
			"set HOME or XDG_CACHE_HOME",
	)
}

func createPrivateDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.Chmod(dir, 0o700)
}

// RecordBefore records pre-mutation content of a single note.
func RecordBefore(
	vault *config.Vault,
	date time.Time,
	path, before string,
) *config.VaultError {
	b := before
	return RecordBeforeMulti(
		vault, date,
		[]FileState{{Path: path, Before: &b}},
	)
}

// FileState holds a file's path and pre-mutation content.
type FileState struct {
	Path   string
	Before *string // nil = file didn't exist
}

// RecordBeforeMulti records pre-mutation content of several
// notes for a single undo.
func RecordBeforeMulti(
	vault *config.Vault,
	date time.Time,
	files []FileState,
) *config.VaultError {
	dest, err := undoPath()
	if err != nil {
		return err
	}
	if parent := filepath.Dir(dest); parent != "" {
		if e := createPrivateDir(parent); e != nil {
			return config.NewVaultError(
				"io",
				fmt.Sprintf(
					"failed to secure undo dir %s: %v",
					parent, e,
				),
			)
		}
	}
	return RecordBeforeMultiTo(vault, date, files, dest)
}

// RecordBeforeMultiTo writes an undo record to an explicit
// path (for testing).
func RecordBeforeMultiTo(
	vault *config.Vault,
	date time.Time,
	files []FileState,
	dest string,
) *config.VaultError {
	rec := undoRecordMulti{
		Vault: vault.Root,
		Date:  date.Format("2006-01-02"),
	}
	for _, f := range files {
		uf := undoFile{Path: f.Path}
		if f.Before != nil {
			s := *f.Before
			uf.Before = &s
		}
		rec.Files = append(rec.Files, uf)
	}
	if parent := filepath.Dir(dest); parent != "" {
		if e := os.MkdirAll(parent, 0o700); e != nil {
			return config.NewVaultError(
				"io",
				fmt.Sprintf(
					"failed to create undo dir %s: %v",
					parent, e,
				),
			)
		}
	}
	data, e := json.MarshalIndent(rec, "", "  ")
	if e != nil {
		return config.NewVaultError(
			"io",
			fmt.Sprintf(
				"failed to serialize undo: %v", e,
			),
		)
	}
	if e := writeUndoFile(dest, data); e != nil {
		return config.NewVaultError(
			"io",
			fmt.Sprintf(
				"failed to write undo %s: %v", dest, e,
			),
		)
	}
	if e := os.Chmod(dest, 0o600); e != nil {
		os.Remove(dest)
		return config.NewVaultError(
			"io",
			fmt.Sprintf(
				"failed to secure undo %s: %v", dest, e,
			),
		)
	}
	return nil
}

func writeUndoFile(dest string, data []byte) error {
	// Remove symlinks before writing.
	if info, err := os.Lstat(dest); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			os.Remove(dest)
		}
	}
	f, err := os.OpenFile(
		dest,
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		0o600,
	)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

// Discard drops pending undo record.
func Discard() {
	if dest, err := undoPath(); err == nil {
		DiscardAt(dest)
	}
}

// DiscardAt drops the undo record at an explicit path.
func DiscardAt(dest string) {
	os.Remove(dest)
}

// WriteAtomicPublicFn is set by the todo package to avoid
// import cycles.
var WriteAtomicPublicFn func(
	vaultRoot, path, content string,
) *config.VaultError

// UndoLast undoes the last mutation.
func UndoLast(
	vault *config.Vault,
	readSnapshotFn func(
		*config.Vault, time.Time,
	) (interface{}, *config.VaultError),
) (interface{}, *config.VaultError) {
	dest, err := undoPath()
	if err != nil {
		return nil, err
	}
	return UndoLastFrom(vault, dest, readSnapshotFn)
}

// UndoLastFrom undoes from an explicit undo file.
func UndoLastFrom(
	vault *config.Vault,
	dest string,
	readSnapshotFn func(
		*config.Vault, time.Time,
	) (interface{}, *config.VaultError),
) (interface{}, *config.VaultError) {
	if _, e := os.Stat(dest); os.IsNotExist(e) {
		return nil, config.NewVaultError(
			"io", "nothing to undo",
		)
	}
	raw, e := os.ReadFile(dest)
	if e != nil {
		return nil, config.NewVaultError(
			"io",
			fmt.Sprintf(
				"failed to read undo %s: %v", dest, e,
			),
		)
	}
	// Try multi-file format first.
	var multi undoRecordMulti
	if json.Unmarshal(raw, &multi) == nil &&
		len(multi.Files) > 0 {
		if multi.Vault != vault.Root {
			return nil, config.NewVaultError(
				"io",
				"undo record is for a different vault; "+
					"refusing to restore",
			)
		}
		date, e := time.Parse("2006-01-02", multi.Date)
		if e != nil {
			return nil, config.NewVaultError(
				"io",
				fmt.Sprintf(
					"invalid undo date %s", multi.Date,
				),
			)
		}
		for _, f := range multi.Files {
			if f.Before != nil {
				if err := WriteAtomicPublicFn(
					vault.Root, f.Path, *f.Before,
				); err != nil {
					return nil, err
				}
			} else {
				if pathInsideVault(vault.Root, f.Path) {
					os.Remove(f.Path)
				}
			}
		}
		os.Remove(dest)
		return readSnapshotFn(vault, date)
	}
	// Legacy single-file format.
	var legacy undoRecord
	if e := json.Unmarshal(raw, &legacy); e != nil {
		return nil, config.NewVaultError(
			"io",
			fmt.Sprintf(
				"failed to parse undo: %v", e,
			),
		)
	}
	if legacy.Vault != vault.Root {
		return nil, config.NewVaultError(
			"io",
			"undo record is for a different vault; "+
				"refusing to restore",
		)
	}
	date, e := time.Parse("2006-01-02", legacy.Date)
	if e != nil {
		return nil, config.NewVaultError(
			"io",
			fmt.Sprintf(
				"invalid undo date %s", legacy.Date,
			),
		)
	}
	if err := WriteAtomicPublicFn(
		vault.Root, legacy.Path, legacy.Before,
	); err != nil {
		return nil, err
	}
	os.Remove(dest)
	return readSnapshotFn(vault, date)
}

func pathInsideVault(vaultRoot, path string) bool {
	root, err := filepath.EvalSymlinks(vaultRoot)
	if err != nil {
		return false
	}
	real, err := filepath.EvalSymlinks(path)
	if err == nil {
		rel, err := filepath.Rel(root, real)
		return err == nil && !strings.HasPrefix(rel, "..")
	}
	// Walk up to nearest existing ancestor.
	anchor := filepath.Dir(path)
	for {
		if _, err := os.Stat(anchor); err == nil {
			real, err := filepath.EvalSymlinks(anchor)
			if err != nil {
				return false
			}
			rel, err := filepath.Rel(root, real)
			return err == nil &&
				!strings.HasPrefix(rel, "..")
		}
		parent := filepath.Dir(anchor)
		if parent == anchor {
			return false
		}
		anchor = parent
	}
}
