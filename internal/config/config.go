// Package config resolves the Obsidian vault and daily-notes
// plugin settings.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ralf/obsidian-daily-macos/internal/format"
)

// VaultEnv is the environment variable for the vault root.
const VaultEnv = "OBSIDIAN_VAULT_ROOT"

// Vault represents an Obsidian vault root with optional
// archive folder pattern.
type Vault struct {
	Root    string
	Archive string // moment-style pattern, empty = none
}

// DailyNotesConfig holds the daily-notes.json settings.
type DailyNotesConfig struct {
	Folder   string // relative folder under vault root
	Format   string // moment-style date format
	Template string // optional template path (no .md)
	Archive  string // optional archive folder pattern
}

// DailyNotePaths holds the primary and resolved paths.
type DailyNotePaths struct {
	Primary  string // live path (creation target)
	Resolved string // primary if exists, else archive
}

// VaultError represents vault-related errors.
type VaultError struct {
	Code    string
	Message string
}

func (e *VaultError) Error() string { return e.Message }

// ErrorCode returns the machine-stable error code.
func (e *VaultError) ErrorCode() string { return e.Code }

// NewVaultError creates a VaultError with the given code.
func NewVaultError(code, message string) *VaultError {
	return &VaultError{Code: code, Message: message}
}

func errMissingEnv() *VaultError {
	return NewVaultError(
		"missing_vault",
		VaultEnv+" is not set (or pass --vault "+
			"/ set vaultPath in bar settings)",
	)
}

func errEmptyEnv() *VaultError {
	return NewVaultError("missing_vault", "vault path is empty")
}

func errNotADir(p string) *VaultError {
	return NewVaultError(
		"bad_vault",
		"vault root is not a directory: "+p,
	)
}

func errIo(msg string) *VaultError {
	return NewVaultError("io", msg)
}

func errBadFormat(msg string) *VaultError {
	return NewVaultError("bad_format", "daily note format: "+msg)
}

// Resolve creates a Vault from an optional CLI path, else
// the OBSIDIAN_VAULT_ROOT env var, plus an optional archive
// folder pattern.
func Resolve(
	cliVault, cliArchive string,
) (*Vault, *VaultError) {
	var vault *Vault
	var err *VaultError
	if cliVault != "" {
		vault, err = FromPath(cliVault)
	} else {
		vault, err = FromEnv()
	}
	if err != nil {
		return nil, err
	}
	if cliArchive != "" {
		trimmed := strings.TrimSpace(cliArchive)
		trimmed = strings.Trim(trimmed, "/")
		s := sanitizeRel(trimmed)
		if s != "" {
			vault.Archive = s
		}
	}
	return vault, nil
}

// FromPath creates a Vault from an explicit path.
func FromPath(path string) (*Vault, *VaultError) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return nil, errEmptyEnv()
	}
	info, e := os.Stat(trimmed)
	if e != nil || !info.IsDir() {
		return nil, errNotADir(trimmed)
	}
	return &Vault{Root: trimmed}, nil
}

// FromEnv creates a Vault from the OBSIDIAN_VAULT_ROOT env.
func FromEnv() (*Vault, *VaultError) {
	val, ok := os.LookupEnv(VaultEnv)
	if !ok {
		return nil, errMissingEnv()
	}
	return FromPath(val)
}

type dailyNotesJSON struct {
	Folder   *string `json:"folder"`
	Format   *string `json:"format"`
	Template *string `json:"template"`
}

// DailyNotesConfig parses .obsidian/daily-notes.json.
func (v *Vault) DailyNotesConfig() (
	*DailyNotesConfig, *VaultError,
) {
	cfg := &DailyNotesConfig{
		Format:  "YYYY-MM-DD",
		Archive: v.Archive,
	}
	p := filepath.Join(v.Root, ".obsidian", "daily-notes.json")
	data, e := os.ReadFile(p)
	if e != nil {
		if os.IsNotExist(e) {
			return cfg, nil
		}
		return nil, errIo(
			fmt.Sprintf("failed to read %s: %v", p, e),
		)
	}
	var parsed dailyNotesJSON
	if e := json.Unmarshal(data, &parsed); e != nil {
		return nil, errIo(
			fmt.Sprintf("failed to parse %s: %v", p, e),
		)
	}
	if parsed.Folder != nil {
		f := strings.TrimSpace(*parsed.Folder)
		f = strings.Trim(f, "/")
		if f != "" {
			cfg.Folder = sanitizeRel(f)
		}
	}
	if parsed.Format != nil {
		f := strings.TrimSpace(*parsed.Format)
		if f != "" {
			cfg.Format = f
		}
	}
	if parsed.Template != nil {
		t := strings.TrimSpace(*parsed.Template)
		t = strings.TrimSuffix(t, ".md")
		if t != "" {
			t = sanitizeRel(t)
			if t != "" {
				cfg.Template = t
			}
		}
	}
	return cfg, nil
}

// DailyNotePath returns the live note path for a date.
func (v *Vault) DailyNotePath(
	cfg *DailyNotesConfig, date time.Time,
) (string, *VaultError) {
	rel, e := format.FormatMoment(cfg.Format, date)
	if e != nil {
		return "", errBadFormat(e.Error())
	}
	path := v.Root
	if cfg.Folder != "" {
		for _, part := range strings.Split(cfg.Folder, "/") {
			var err *VaultError
			path, err = pushSafeComponent(path, part)
			if err != nil {
				return "", err
			}
		}
	}
	for _, part := range strings.Split(rel, "/") {
		var err *VaultError
		path, err = pushSafeComponent(path, part)
		if err != nil {
			return "", err
		}
	}
	path = withMdSuffix(path)
	if err := ensureUnderRoot(v.Root, path); err != nil {
		return "", err
	}
	return path, nil
}

// DailyNotePaths resolves primary and resolved paths.
func (v *Vault) DailyNotePaths(
	cfg *DailyNotesConfig, date time.Time,
) (*DailyNotePaths, *VaultError) {
	primary, err := v.DailyNotePath(cfg, date)
	if err != nil {
		return nil, err
	}
	resolved := primary
	if cfg.Archive != "" && !isFile(primary) {
		archive, err := v.ArchiveNotePath(cfg, date)
		if err == nil && isFile(archive) {
			resolved = archive
		}
	}
	return &DailyNotePaths{
		Primary:  primary,
		Resolved: resolved,
	}, nil
}

// ArchiveNotePath builds the archived note path.
func (v *Vault) ArchiveNotePath(
	cfg *DailyNotesConfig, date time.Time,
) (string, *VaultError) {
	if cfg.Archive == "" {
		return "", errBadFormat(
			"archive pattern is not configured",
		)
	}
	rel, e := format.FormatMoment(cfg.Archive, date)
	if e != nil {
		return "", errBadFormat(e.Error())
	}
	name, e := format.FormatMoment(cfg.Format, date)
	if e != nil {
		return "", errBadFormat(e.Error())
	}
	path := v.Root
	parts := append(
		strings.Split(rel, "/"), strings.Split(name, "/")...,
	)
	for _, part := range parts {
		var err *VaultError
		path, err = pushSafeComponent(path, part)
		if err != nil {
			return "", err
		}
	}
	path = withMdSuffix(path)
	if err := ensureUnderRoot(v.Root, path); err != nil {
		return "", err
	}
	return path, nil
}

// TemplatePath returns the template note path if configured.
func (v *Vault) TemplatePath(
	cfg *DailyNotesConfig,
) string {
	if cfg.Template == "" {
		return ""
	}
	path := v.Root
	for _, part := range strings.Split(cfg.Template, "/") {
		var err *VaultError
		path, err = pushSafeComponent(path, part)
		if err != nil {
			return ""
		}
	}
	path = withMdSuffix(path)
	if ensureUnderRoot(v.Root, path) != nil {
		return ""
	}
	return path
}

func sanitizeRel(value string) string {
	parts := strings.Split(value, "/")
	var out []string
	for _, p := range parts {
		if p != "" && p != "." && p != ".." {
			out = append(out, p)
		}
	}
	return strings.Join(out, "/")
}

func withMdSuffix(path string) string {
	ext := filepath.Ext(path)
	if strings.EqualFold(ext, ".md") {
		return path
	}
	return path + ".md"
}

func pushSafeComponent(
	path, part string,
) (string, *VaultError) {
	if part == "" || part == "." {
		return path, nil
	}
	if part == ".." || strings.ContainsRune(part, 0) {
		return "", errIo(
			fmt.Sprintf(
				"refusing path component that escapes "+
					"the vault: %q", part,
			),
		)
	}
	return filepath.Join(path, part), nil
}

func ensureUnderRoot(root, path string) *VaultError {
	// Use filepath.Rel to check if path is under root.
	rel, e := filepath.Rel(root, path)
	if e != nil || strings.HasPrefix(rel, "..") {
		return errIo(
			"resolved path escapes vault root: " + path,
		)
	}
	// Canonicalization check for existing paths.
	realPath, e1 := filepath.EvalSymlinks(path)
	realRoot, e2 := filepath.EvalSymlinks(root)
	if e1 == nil && e2 == nil {
		rel2, e := filepath.Rel(realRoot, realPath)
		if e != nil || strings.HasPrefix(rel2, "..") {
			return errIo(fmt.Sprintf(
				"resolved path escapes vault root: "+
					"%s resolves to %s",
				path, realPath,
			))
		}
	}
	return nil
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
