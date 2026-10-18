package todo

import (
	"fmt"
	"os"
	"time"

	"github.com/ralf/obsidian-daily-macos/internal/config"
	"github.com/ralf/obsidian-daily-macos/internal/status"
	"github.com/ralf/obsidian-daily-macos/internal/undo"
)

// MoveOpenFromPrevious moves still-open todos from the most
// recent previous daily note into the target date.
func MoveOpenFromPrevious(
	vault *config.Vault,
	date time.Time,
	heading string,
) (int, *config.VaultError) {
	cfg, err := vault.DailyNotesConfig()
	if err != nil {
		return 0, err
	}
	path, err := ResolvedNotePath(vault, cfg, date)
	if err != nil {
		return 0, err
	}
	if e := CreateNoteIfMissing(
		vault, cfg, path, date,
	); e != nil {
		return 0, e
	}
	prev, _, found, err := carrySource(vault, cfg, date)
	if err != nil {
		return 0, err
	}
	if !found {
		return 0, nil
	}
	items, err := openTodoItems(vault, cfg, prev)
	if err != nil {
		return 0, err
	}
	if len(items) == 0 {
		return 0, nil
	}

	// Build existing set from target.
	existing := map[string]bool{}
	snap, err := ReadSnapshot(vault, date)
	if err != nil {
		return 0, err
	}
	if snap.Todos != nil {
		for _, t := range *snap.Todos {
			if !t.Checked {
				existing[t.Text] = true
			}
		}
	}

	var toMove []depthText
	for _, item := range items {
		if !existing[item.text] {
			toMove = append(toMove, depthText{
				depth: item.depth,
				text:  item.text,
			})
		}
	}

	// Read both notes before any mutations.
	prevPath, err := ResolvedNotePath(vault, cfg, prev)
	if err != nil {
		return 0, err
	}
	prevBefore, e := os.ReadFile(prevPath)
	if e != nil {
		return 0, config.NewVaultError(
			"io",
			fmt.Sprintf(
				"failed to read %s: %v", prevPath, e,
			),
		)
	}
	targetBefore, e := os.ReadFile(path)
	if e != nil {
		return 0, config.NewVaultError(
			"io",
			fmt.Sprintf(
				"failed to read %s: %v", path, e,
			),
		)
	}

	if len(toMove) > 0 {
		if e := addTodoLines(
			vault, date, toMove, heading,
		); e != nil {
			undo.Discard()
			return 0, e
		}
	}

	// Record both notes' pre-mutation states.
	pb := string(prevBefore)
	tb := string(targetBefore)
	if e := undo.RecordBeforeMulti(
		vault, date,
		[]undo.FileState{
			{Path: prevPath, Before: &pb},
			{Path: path, Before: &tb},
		},
	); e != nil {
		return 0, e
	}

	// Reconcile against target's current open todos.
	snap2, err := ReadSnapshot(vault, date)
	if err != nil {
		return 0, err
	}
	var targetOpen []string
	if snap2.Todos != nil {
		for _, t := range *snap2.Todos {
			if !t.Checked {
				targetOpen = append(targetOpen, t.Text)
			}
		}
	}

	budget := removalBudget(items, targetOpen)
	if e := removeTodos(vault, prev, budget); e != nil {
		return 0, e
	}
	return len(toMove), nil
}

func removalBudget(
	source []openTodo, target []string,
) map[string]int {
	available := map[string]int{}
	for _, item := range source {
		available[item.text]++
	}
	budget := map[string]int{}
	for _, text := range target {
		left := available[text]
		if left == 0 {
			continue
		}
		if budget[text] < left {
			budget[text]++
		}
	}
	return budget
}

func removeTodos(
	vault *config.Vault,
	date time.Time,
	budget map[string]int,
) *config.VaultError {
	cfg, err := vault.DailyNotesConfig()
	if err != nil {
		return err
	}
	path, err := ResolvedNotePath(vault, cfg, date)
	if err != nil {
		return err
	}
	if !fileExists(path) {
		return nil
	}
	data, e := os.ReadFile(path)
	if e != nil {
		return config.NewVaultError(
			"io",
			fmt.Sprintf(
				"failed to read %s: %v", path, e,
			),
		)
	}
	content := string(data)
	remaining := map[string]int{}
	for k, v := range budget {
		remaining[k] = v
	}
	// Walk from last to first.
	todos := ParseTodos(content)
	drop := map[int]bool{}
	for i := len(todos) - 1; i >= 0; i-- {
		t := todos[i]
		if t.Checked {
			continue
		}
		left := remaining[t.Text]
		if left > 0 {
			remaining[t.Text]--
			drop[t.Line] = true
		}
	}
	if len(drop) == 0 {
		return nil
	}
	next := withoutLines(content, drop)
	return WriteAtomic(vault.Root, path, next)
}

// CarryOver rolls open todos from the most recent previous
// daily note into the target date.
func CarryOver(
	vault *config.Vault,
	date time.Time,
	heading string,
) (*status.Snapshot, *config.VaultError) {
	if _, err := MoveOpenFromPrevious(
		vault, date, heading,
	); err != nil {
		return nil, err
	}
	return ReadSnapshot(vault, date)
}

// OpenInObsidian creates the daily note and opens it.
func OpenInObsidian(
	vault *config.Vault,
	date time.Time,
) (*status.Snapshot, *config.VaultError) {
	snap, err := prepareNoteForOpen(vault, date)
	if err != nil {
		return nil, err
	}
	if snap.ObsidianURI == nil {
		return nil, config.NewVaultError(
			"io", "missing obsidian URI",
		)
	}
	// Import open package.
	openPkg.Launch(*snap.ObsidianURI)
	return snap, nil
}

func prepareNoteForOpen(
	vault *config.Vault,
	date time.Time,
) (*status.Snapshot, *config.VaultError) {
	cfg, err := vault.DailyNotesConfig()
	if err != nil {
		return nil, err
	}
	path, err := ResolvedNotePath(vault, cfg, date)
	if err != nil {
		return nil, err
	}
	if e := CreateNoteIfMissing(
		vault, cfg, path, date,
	); e != nil {
		return nil, e
	}
	return ReadSnapshot(vault, date)
}

// openPkg is a reference to the open package to avoid
// import cycles. Set by init in the main package.
var openPkg openLauncher

type openLauncher interface {
	Launch(uri string) *config.VaultError
}

type defaultLauncher struct{}

func (defaultLauncher) Launch(
	uri string,
) *config.VaultError {
	// Will be overridden.
	return nil
}

func init() {
	openPkg = defaultLauncher{}
}

// SetLauncher sets the open launcher implementation.
func SetLauncher(l openLauncher) {
	openPkg = l
}

// OpenLauncher is the interface for launching URIs.
type OpenLauncher interface {
	Launch(uri string) *config.VaultError
}
