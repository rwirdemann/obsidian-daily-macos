package todo

import (
	"fmt"
	"os"
	"time"

	"github.com/ralf/obsidian-daily-macos/internal/config"
	"github.com/ralf/obsidian-daily-macos/internal/status"
	"github.com/ralf/obsidian-daily-macos/internal/undo"
)

// DeferTodo moves a todo to the next day's note.
func DeferTodo(
	vault *config.Vault,
	date time.Time,
	line int,
	expectText string,
	withChildren bool,
	heading string,
) (*status.Snapshot, *config.VaultError) {
	return DeferTodoTo(
		vault, date, line, expectText,
		withChildren, heading, "",
	)
}

// DeferTodoTo moves a todo to the next day's note, using an
// explicit undo path when set.
func DeferTodoTo(
	vault *config.Vault,
	date time.Time,
	line int,
	expectText string,
	withChildren bool,
	heading string,
	undoDest string,
) (*status.Snapshot, *config.VaultError) {
	if line == 0 {
		return nil, config.NewVaultError(
			"io", "line must be >= 1",
		)
	}
	nextDate := date.AddDate(0, 0, 1)
	cfg, err := vault.DailyNotesConfig()
	if err != nil {
		return nil, err
	}
	path, err := ResolvedNotePath(vault, cfg, date)
	if err != nil {
		return nil, err
	}
	if !fileExists(path) {
		return nil, config.NewVaultError(
			"io",
			fmt.Sprintf(
				"daily note does not exist: %s", path,
			),
		)
	}
	data, e := os.ReadFile(path)
	if e != nil {
		return nil, config.NewVaultError(
			"io",
			fmt.Sprintf(
				"failed to read %s: %v", path, e,
			),
		)
	}
	content := string(data)
	if err := expectLineText(
		content, line, expectText,
	); err != nil {
		return nil, err
	}
	todos := ParseTodos(content)
	var target *status.TodoItem
	for i := range todos {
		if todos[i].Line == line {
			target = &todos[i]
			break
		}
	}
	if target == nil {
		return nil, config.NewVaultError(
			"io",
			fmt.Sprintf(
				"line %d is not a todo", line,
			),
		)
	}
	drop := map[int]bool{line: true}
	if withChildren {
		parentDepth := target.Depth
		found := false
		for _, t := range todos {
			if t.Line == line {
				found = true
				continue
			}
			if !found {
				continue
			}
			if t.Depth <= parentDepth {
				break
			}
			drop[t.Line] = true
		}
	}
	var moving []checkboxItem
	for _, t := range todos {
		if drop[t.Line] {
			moving = append(moving, checkboxItem{
				depth:   t.Depth,
				checked: t.Checked,
				text:    t.Text,
			})
		}
	}
	baseDepth := moving[0].depth
	for i := range moving {
		moving[i].depth -= baseDepth
		if moving[i].depth < 0 {
			moving[i].depth = 0
		}
	}

	destPath, err := ResolvedNotePath(vault, cfg, nextDate)
	if err != nil {
		return nil, err
	}
	if destPath == path {
		return nil, config.NewVaultError(
			"io",
			"cannot defer onto the same note",
		)
	}
	destExisted := fileExists(destPath)
	var destBefore *string
	if destExisted {
		d, e := os.ReadFile(destPath)
		if e != nil {
			return nil, config.NewVaultError(
				"io",
				fmt.Sprintf(
					"failed to read %s: %v", destPath, e,
				),
			)
		}
		s := string(d)
		destBefore = &s
	}

	// Record both files' pre-mutation states.
	record := func(
		files []undo.FileState,
	) *config.VaultError {
		if undoDest != "" {
			return undo.RecordBeforeMultiTo(
				vault, date, files, undoDest,
			)
		}
		return undo.RecordBeforeMulti(
			vault, date, files,
		)
	}
	discardRecord := func() {
		if undoDest != "" {
			undo.DiscardAt(undoDest)
		} else {
			undo.Discard()
		}
	}

	if err := record([]undo.FileState{
		{Path: path, Before: &content},
		{Path: destPath, Before: destBefore},
	}); err != nil {
		return nil, err
	}

	// Create destination from template (no carry-over).
	if err := CreateNoteIfMissing(
		vault, cfg, destPath, nextDate,
	); err != nil {
		discardRecord()
		return nil, err
	}
	destData, e := os.ReadFile(destPath)
	if e != nil {
		discardRecord()
		return nil, config.NewVaultError(
			"io",
			fmt.Sprintf(
				"failed to read %s: %v", destPath, e,
			),
		)
	}
	destContent := string(destData)
	destNext := insertTodoLines(
		destContent,
		formatCheckboxLines(moving),
		inferTodoStyle(vault, cfg, nextDate, destContent),
		heading,
	)
	if err := WriteAtomic(
		vault.Root, destPath, destNext,
	); err != nil {
		// Roll back destination.
		if !destExisted {
			os.Remove(destPath)
		}
		discardRecord()
		return nil, err
	}

	next := withoutLines(content, drop)
	if err := WriteAtomic(
		vault.Root, path, next,
	); err != nil {
		// Roll back destination.
		if destBefore != nil {
			WriteAtomic(vault.Root, destPath, *destBefore)
		} else {
			os.Remove(destPath)
		}
		discardRecord()
		return nil, err
	}
	return ReadSnapshot(vault, date)
}
