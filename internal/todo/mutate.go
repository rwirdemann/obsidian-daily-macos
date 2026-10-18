package todo

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ralf/obsidian-daily-macos/internal/config"
	"github.com/ralf/obsidian-daily-macos/internal/status"
)

// ToggleTodo toggles the checkbox on a 1-based line.
func ToggleTodo(
	vault *config.Vault,
	date time.Time,
	line int,
	expectText string,
) (*status.Snapshot, *config.VaultError) {
	if line == 0 {
		return nil, config.NewVaultError(
			"io", "line must be >= 1",
		)
	}
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
	next, err := toggleLine(content, line)
	if err != nil {
		return nil, err
	}
	if err := writeAtomicWithUndo(
		vault, date, path, content, next,
	); err != nil {
		return nil, err
	}
	return ReadSnapshot(vault, date)
}

func toggleLine(
	content string, line int,
) (string, *config.VaultError) {
	lines := splitLines(content)
	idx := line - 1
	if idx >= len(lines) {
		return "", config.NewVaultError(
			"io",
			fmt.Sprintf(
				"line %d is past end of file (%d lines)",
				line, len(lines),
			),
		)
	}
	if err := rejectFencedLine(content, line); err != nil {
		return "", err
	}
	m := checkboxRe.FindStringSubmatch(lines[idx])
	if m == nil {
		return "", config.NewVaultError(
			"io",
			fmt.Sprintf(
				"line %d is not a checkbox todo", line,
			),
		)
	}
	checked := m[3] != " "
	mark := "x"
	if checked {
		mark = " "
	}
	newLine := fmt.Sprintf(
		"%s%s [%s]%s%s", m[1], m[2], mark, m[4], m[5],
	)
	return rewriteLine(content, line, newLine), nil
}

// EditTodo rewrites the text of a todo on a given line.
func EditTodo(
	vault *config.Vault,
	date time.Time,
	line int,
	expectText string,
	newText string,
) (*status.Snapshot, *config.VaultError) {
	if line == 0 {
		return nil, config.NewVaultError(
			"io", "line must be >= 1",
		)
	}
	newText = strings.TrimSpace(newText)
	if newText == "" {
		return nil, config.NewVaultError(
			"io", "todo text is empty",
		)
	}
	if strings.ContainsAny(newText, "\n\r") {
		return nil, config.NewVaultError(
			"io", "todo text must be a single line",
		)
	}
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
	lines := splitLines(content)
	idx := line - 1
	if idx >= len(lines) {
		return nil, config.NewVaultError(
			"io",
			fmt.Sprintf(
				"line %d is past end of file", line,
			),
		)
	}
	if err := rejectFencedLine(content, line); err != nil {
		return nil, err
	}
	m := checkboxRe.FindStringSubmatch(lines[idx])
	if m == nil {
		return nil, config.NewVaultError(
			"io",
			fmt.Sprintf(
				"line %d is not a checkbox todo", line,
			),
		)
	}
	newLine := fmt.Sprintf(
		"%s%s [%s]%s%s", m[1], m[2], m[3], m[4], newText,
	)
	next := rewriteLine(content, line, newLine)
	if err := writeAtomicWithUndo(
		vault, date, path, content, next,
	); err != nil {
		return nil, err
	}
	return ReadSnapshot(vault, date)
}

// DeleteTodo deletes a todo, optionally with nested children.
func DeleteTodo(
	vault *config.Vault,
	date time.Time,
	line int,
	expectText string,
	withChildren bool,
) (*status.Snapshot, *config.VaultError) {
	if line == 0 {
		return nil, config.NewVaultError(
			"io", "line must be >= 1",
		)
	}
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
	next := withoutLines(content, drop)
	if err := writeAtomicWithUndo(
		vault, date, path, content, next,
	); err != nil {
		return nil, err
	}
	return ReadSnapshot(vault, date)
}

// SetIndent adjusts a todo's indentation by delta.
func SetIndent(
	vault *config.Vault,
	date time.Time,
	line int,
	expectText string,
	delta int,
) (*status.Snapshot, *config.VaultError) {
	if line == 0 {
		return nil, config.NewVaultError(
			"io", "line must be >= 1",
		)
	}
	if delta == 0 {
		return ReadSnapshot(vault, date)
	}
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
	lines := splitLines(content)
	idx := line - 1
	if idx >= len(lines) {
		return nil, config.NewVaultError(
			"io",
			fmt.Sprintf(
				"line %d is past end of file", line,
			),
		)
	}
	if err := rejectFencedLine(content, line); err != nil {
		return nil, err
	}
	m := checkboxRe.FindStringSubmatch(lines[idx])
	if m == nil {
		return nil, config.NewVaultError(
			"io",
			fmt.Sprintf(
				"line %d is not a checkbox todo", line,
			),
		)
	}
	level := IndentLevel(m[1]) + delta
	if level < 0 {
		level = 0
	}
	indent := strings.Repeat("  ", level)
	newLine := fmt.Sprintf(
		"%s%s [%s]%s%s",
		indent, m[2], m[3], m[4], m[5],
	)
	next := rewriteLine(content, line, newLine)
	if err := writeAtomicWithUndo(
		vault, date, path, content, next,
	); err != nil {
		return nil, err
	}
	return ReadSnapshot(vault, date)
}
