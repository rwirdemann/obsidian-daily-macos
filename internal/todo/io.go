package todo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ralf/obsidian-daily-macos/internal/config"
)

// WriteAtomic writes content to path atomically, verifying
// it stays inside the vault root.
func WriteAtomic(
	vaultRoot, path, content string,
) *config.VaultError {
	target, err := resolveNoteTarget(vaultRoot, path)
	if err != nil {
		return err
	}
	tmp := tempPathFor(target)
	writeErr := func() *config.VaultError {
		mode := atomicTempMode(target)
		f, e := os.OpenFile(
			tmp,
			os.O_WRONLY|os.O_CREATE|os.O_EXCL,
			mode,
		)
		if e != nil {
			return config.NewVaultError(
				"io",
				fmt.Sprintf(
					"failed to create %s: %v", tmp, e,
				),
			)
		}
		_, e = f.WriteString(content)
		if e != nil {
			f.Close()
			return config.NewVaultError(
				"io",
				fmt.Sprintf(
					"failed to write %s: %v", tmp, e,
				),
			)
		}
		if e = f.Sync(); e != nil {
			f.Close()
			return config.NewVaultError(
				"io",
				fmt.Sprintf(
					"failed to sync %s: %v", tmp, e,
				),
			)
		}
		f.Close()
		return nil
	}()
	if writeErr != nil {
		os.Remove(tmp)
		return writeErr
	}
	// Re-apply existing note permissions.
	if info, e := os.Stat(target); e == nil {
		os.Chmod(tmp, info.Mode().Perm())
	}
	if e := os.Rename(tmp, target); e != nil {
		os.Remove(tmp)
		return config.NewVaultError(
			"io",
			fmt.Sprintf(
				"failed to replace %s: %v", target, e,
			),
		)
	}
	return nil
}

func resolveNoteTarget(
	vaultRoot, path string,
) (string, *config.VaultError) {
	root, e := filepath.EvalSymlinks(vaultRoot)
	if e != nil {
		return "", config.NewVaultError(
			"io",
			fmt.Sprintf(
				"failed to resolve vault root %s: %v",
				vaultRoot, e,
			),
		)
	}
	parent := filepath.Dir(path)
	realParent, e := filepath.EvalSymlinks(parent)
	if e != nil {
		return "", config.NewVaultError(
			"io",
			fmt.Sprintf(
				"failed to resolve %s: %v", parent, e,
			),
		)
	}
	if !isUnder(root, realParent) {
		return "", config.NewVaultError(
			"io",
			fmt.Sprintf(
				"refusing to write outside the vault: "+
					"%s resolves to %s",
				parent, realParent,
			),
		)
	}
	name := filepath.Base(path)
	resolved := filepath.Join(realParent, name)
	realResolved, e := filepath.EvalSymlinks(resolved)
	if e == nil {
		if isUnder(root, realResolved) {
			return realResolved, nil
		}
		return "", config.NewVaultError(
			"io",
			fmt.Sprintf(
				"refusing to write outside the vault: "+
					"%s resolves to %s",
				path, realResolved,
			),
		)
	}
	return resolved, nil
}

func tempPathFor(target string) string {
	nonce := time.Now().UnixNano()
	base := filepath.Base(target)
	name := fmt.Sprintf(
		"%s.tmp-obsidian-daily-qs-%d-%d",
		base, os.Getpid(), nonce,
	)
	return filepath.Join(filepath.Dir(target), name)
}

func atomicTempMode(target string) os.FileMode {
	if info, e := os.Stat(target); e == nil {
		return info.Mode().Perm()
	}
	return 0o600
}

// CreateDirAllInVault creates directories after verifying
// they stay inside the vault.
func CreateDirAllInVault(
	vaultRoot, dir string,
) *config.VaultError {
	root, e := filepath.EvalSymlinks(vaultRoot)
	if e != nil {
		return config.NewVaultError(
			"io",
			fmt.Sprintf(
				"failed to resolve vault root %s: %v",
				vaultRoot, e,
			),
		)
	}
	// Find nearest existing ancestor.
	anchor := dir
	for {
		if info, err := os.Stat(anchor); err == nil &&
			info.IsDir() {
			break
		}
		parent := filepath.Dir(anchor)
		if parent == anchor {
			return config.NewVaultError(
				"io",
				fmt.Sprintf(
					"no existing parent of %s", dir,
				),
			)
		}
		anchor = parent
	}
	realAnchor, e := filepath.EvalSymlinks(anchor)
	if e != nil {
		return config.NewVaultError(
			"io",
			fmt.Sprintf(
				"failed to resolve %s: %v", anchor, e,
			),
		)
	}
	if !isUnder(root, realAnchor) {
		return config.NewVaultError(
			"io",
			fmt.Sprintf(
				"refusing to create directories outside "+
					"the vault: %s resolves to %s",
				anchor, realAnchor,
			),
		)
	}
	if e := os.MkdirAll(dir, 0o755); e != nil {
		return config.NewVaultError(
			"io",
			fmt.Sprintf(
				"failed to create %s: %v", dir, e,
			),
		)
	}
	return nil
}

func isUnder(root, path string) bool {
	// Use filepath.Rel to check containment.
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..")
}

// ResolvedNotePath returns the actual path of a daily note,
// checking archive fallback.
func ResolvedNotePath(
	vault *config.Vault,
	cfg *config.DailyNotesConfig,
	date time.Time,
) (string, *config.VaultError) {
	paths, err := vault.DailyNotePaths(cfg, date)
	if err != nil {
		return "", err
	}
	return paths.Resolved, nil
}
