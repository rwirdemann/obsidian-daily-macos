// Package open builds Obsidian URIs and launches the app.
package open

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ralf/obsidian-daily-macos/internal/config"
)

// OpenURI builds an Obsidian "open" URI for a note inside
// the vault root. Prefers vault-relative form to avoid
// leaking private paths.
func OpenURI(vaultRoot, note string) string {
	if vault, file, ok := vaultRelativeParts(
		vaultRoot, note,
	); ok {
		return fmt.Sprintf(
			"obsidian://open?vault=%s&file=%s",
			percentEncodePath(vault),
			percentEncodePath(file),
		)
	}
	return fmt.Sprintf(
		"obsidian://open?path=%s",
		percentEncodePath(note),
	)
}

func vaultRelativeParts(
	vaultRoot, note string,
) (string, string, bool) {
	root, err := filepath.EvalSymlinks(vaultRoot)
	if err != nil {
		return "", "", false
	}
	info, err := os.Stat(
		filepath.Join(root, ".obsidian"),
	)
	if err != nil || !info.IsDir() {
		return "", "", false
	}
	name := filepath.Base(root)
	if name == "" || name == "." || name == "/" {
		return "", "", false
	}
	resolved, err := filepath.EvalSymlinks(note)
	if err != nil {
		resolved = note
	}
	rel, err := filepath.Rel(root, resolved)
	if err != nil || strings.HasPrefix(rel, "..") ||
		rel == "." {
		return "", "", false
	}
	return name, rel, true
}

func percentEncodePath(value string) string {
	var out strings.Builder
	out.Grow(len(value) * 3)
	for i := 0; i < len(value); i++ {
		b := value[i]
		if isUnreserved(b) || b == '/' {
			out.WriteByte(b)
		} else {
			out.WriteByte('%')
			out.WriteByte(hexChar(b >> 4))
			out.WriteByte(hexChar(b & 0x0f))
		}
	}
	return out.String()
}

func isUnreserved(b byte) bool {
	return (b >= 'A' && b <= 'Z') ||
		(b >= 'a' && b <= 'z') ||
		(b >= '0' && b <= '9') ||
		b == '-' || b == '_' || b == '.' || b == '~'
}

func hexChar(n byte) byte {
	if n < 10 {
		return '0' + n
	}
	return 'A' + (n - 10)
}

// Launch opens a URI via macOS /usr/bin/open command.
func Launch(uri string) *config.VaultError {
	cmd := exec.Command("/usr/bin/open", uri)
	if err := cmd.Start(); err != nil {
		return config.NewVaultError(
			"io",
			fmt.Sprintf(
				"failed to spawn open: %v", err,
			),
		)
	}
	return nil
}
