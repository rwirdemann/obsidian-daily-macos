package todo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ralf/obsidian-daily-macos/internal/config"
	"github.com/ralf/obsidian-daily-macos/internal/status"
	"github.com/ralf/obsidian-daily-macos/internal/undo"
)

// inboxName is the fixed file name for the inbox note.
const inboxName = "Inbox.md"

// inboxPath returns the absolute path to Inbox.md in the
// vault root.
func inboxPath(vault *config.Vault) string {
	return filepath.Join(vault.Root, inboxName)
}

// ReadInbox reads todos from Inbox.md and returns them as
// a snapshot with date set to "inbox".
func ReadInbox(
	vault *config.Vault,
) (*status.Snapshot, *config.VaultError) {
	path := inboxPath(vault)
	var todos []status.TodoItem
	exists := fileExists(path)
	if exists {
		data, e := os.ReadFile(path)
		if e != nil {
			return nil, config.NewVaultError(
				"io",
				fmt.Sprintf(
					"failed to read %s: %v", path, e,
				),
			)
		}
		todos = ParseTodos(string(data))
	}
	if todos == nil {
		todos = []status.TodoItem{}
	}
	snap := status.Ok("inbox", path, exists, todos)
	return &snap, nil
}

// InboxCount returns the number of open todos in Inbox.md.
func InboxCount(
	vault *config.Vault,
) (int, *config.VaultError) {
	path := inboxPath(vault)
	if !fileExists(path) {
		return 0, nil
	}
	data, e := os.ReadFile(path)
	if e != nil {
		return 0, config.NewVaultError(
			"io",
			fmt.Sprintf(
				"failed to read %s: %v", path, e,
			),
		)
	}
	count := 0
	for _, t := range ParseTodos(string(data)) {
		if !t.Checked {
			count++
		}
	}
	return count, nil
}

// AddInboxTodo appends an open checkbox todo to Inbox.md.
func AddInboxTodo(
	vault *config.Vault,
	text string,
) (*status.Snapshot, *config.VaultError) {
	text = trimSpace(text)
	if text == "" {
		return nil, config.NewVaultError(
			"io", "todo text is empty",
		)
	}
	if containsNewline(text) {
		return nil, config.NewVaultError(
			"io", "todo text must be a single line",
		)
	}

	path := inboxPath(vault)
	var content string
	if fileExists(path) {
		data, e := os.ReadFile(path)
		if e != nil {
			return nil, config.NewVaultError(
				"io",
				fmt.Sprintf(
					"failed to read %s: %v", path, e,
				),
			)
		}
		content = string(data)
	}

	before := content
	line := fmt.Sprintf("- [ ] %s", text)
	if content == "" {
		content = line + "\n"
	} else {
		if content[len(content)-1] != '\n' {
			content += "\n"
		}
		content += line + "\n"
	}

	if fileExists(path) {
		if e := undo.RecordBefore(
			vault, time.Now(), path, before,
		); e != nil {
			return nil, e
		}
	}

	if e := WriteAtomic(
		vault.Root, path, content,
	); e != nil {
		return nil, e
	}
	return ReadInbox(vault)
}

func trimSpace(s string) string {
	return strings.TrimSpace(s)
}

func containsNewline(s string) bool {
	return strings.ContainsAny(s, "\n\r")
}

// DeleteInboxTodo removes a todo from Inbox.md by line.
func DeleteInboxTodo(
	vault *config.Vault,
	line int,
	expectText string,
) (*status.Snapshot, *config.VaultError) {
	if line == 0 {
		return nil, config.NewVaultError(
			"io", "line must be >= 1",
		)
	}

	path := inboxPath(vault)
	if !fileExists(path) {
		return nil, config.NewVaultError(
			"io", "Inbox.md does not exist",
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
	found := false
	for _, t := range todos {
		if t.Line == line {
			found = true
			break
		}
	}
	if !found {
		return nil, config.NewVaultError(
			"io",
			fmt.Sprintf(
				"line %d is not a todo", line,
			),
		)
	}

	if err := undo.RecordBefore(
		vault, time.Now(), path, content,
	); err != nil {
		return nil, err
	}

	drop := map[int]bool{line: true}
	next := withoutLines(content, drop)
	if err := WriteAtomic(
		vault.Root, path, next,
	); err != nil {
		return nil, err
	}
	return ReadInbox(vault)
}

// ToggleInboxTodo toggles the checkbox on a 1-based line
// in Inbox.md.
func ToggleInboxTodo(
	vault *config.Vault,
	line int,
	expectText string,
) (*status.Snapshot, *config.VaultError) {
	if line == 0 {
		return nil, config.NewVaultError(
			"io", "line must be >= 1",
		)
	}

	path := inboxPath(vault)
	if !fileExists(path) {
		return nil, config.NewVaultError(
			"io", "Inbox.md does not exist",
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
	next, err := toggleLine(content, line)
	if err != nil {
		return nil, err
	}
	if err := writeAtomicWithUndo(
		vault, time.Now(), path, content, next,
	); err != nil {
		return nil, err
	}
	return ReadInbox(vault)
}

// MoveInboxTodo moves a single todo from Inbox.md into
// the daily note for the given date.
func MoveInboxTodo(
	vault *config.Vault,
	line int,
	expectText string,
	date time.Time,
	heading string,
) (*status.Snapshot, *config.VaultError) {
	if line == 0 {
		return nil, config.NewVaultError(
			"io", "line must be >= 1",
		)
	}

	ibPath := inboxPath(vault)
	if !fileExists(ibPath) {
		return nil, config.NewVaultError(
			"io", "Inbox.md does not exist",
		)
	}

	ibData, e := os.ReadFile(ibPath)
	if e != nil {
		return nil, config.NewVaultError(
			"io",
			fmt.Sprintf(
				"failed to read %s: %v", ibPath, e,
			),
		)
	}
	ibContent := string(ibData)

	if err := expectLineText(
		ibContent, line, expectText,
	); err != nil {
		return nil, err
	}

	todos := ParseTodos(ibContent)
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

	// Resolve daily note path and ensure it exists.
	cfg, err := vault.DailyNotesConfig()
	if err != nil {
		return nil, err
	}
	notePath, err := ResolvedNotePath(vault, cfg, date)
	if err != nil {
		return nil, err
	}
	if e := CreateNoteIfMissing(
		vault, cfg, notePath, date,
	); e != nil {
		return nil, e
	}

	// Read daily note before mutation.
	noteData, e := os.ReadFile(notePath)
	if e != nil {
		return nil, config.NewVaultError(
			"io",
			fmt.Sprintf(
				"failed to read %s: %v", notePath, e,
			),
		)
	}

	// Insert the todo into the daily note.
	items := []depthText{{
		depth: target.Depth,
		text:  target.Text,
	}}
	if e := addTodoLines(
		vault, date, items, heading,
	); e != nil {
		undo.Discard()
		return nil, e
	}

	// Record undo for both files.
	ib := ibContent
	nb := string(noteData)
	if e := undo.RecordBeforeMulti(
		vault, date,
		[]undo.FileState{
			{Path: ibPath, Before: &ib},
			{Path: notePath, Before: &nb},
		},
	); e != nil {
		return nil, e
	}

	// Remove the todo from Inbox.md.
	drop := map[int]bool{line: true}
	next := withoutLines(ibContent, drop)
	if e := WriteAtomic(
		vault.Root, ibPath, next,
	); e != nil {
		return nil, e
	}

	return ReadSnapshot(vault, date)
}
