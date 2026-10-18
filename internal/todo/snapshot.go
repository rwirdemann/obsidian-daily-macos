package todo

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ralf/obsidian-daily-macos/internal/config"
	"github.com/ralf/obsidian-daily-macos/internal/open"
	"github.com/ralf/obsidian-daily-macos/internal/status"
)

// ReadSnapshot reads a daily note's todos into a snapshot.
func ReadSnapshot(
	vault *config.Vault, date time.Time,
) (*status.Snapshot, *config.VaultError) {
	return ReadSnapshotFiltered(vault, date, "")
}

// ReadSnapshotFiltered reads a daily note's todos, optionally
// filtering by heading.
func ReadSnapshotFiltered(
	vault *config.Vault, date time.Time, heading string,
) (*status.Snapshot, *config.VaultError) {
	cfg, err := vault.DailyNotesConfig()
	if err != nil {
		return nil, err
	}
	path, err := ResolvedNotePath(vault, cfg, date)
	if err != nil {
		return nil, err
	}
	dateStr := date.Format("2006-01-02")
	var todos []status.TodoItem
	exists := fileExists(path)
	if exists {
		content, e := os.ReadFile(path)
		if e != nil {
			return nil, config.NewVaultError(
				"io",
				fmt.Sprintf(
					"failed to read %s: %v", path, e,
				),
			)
		}
		all := ParseTodos(string(content))
		if heading != "" {
			todos = FilterTodosByHeading(
				string(content), all, heading,
			)
		} else {
			todos = all
		}
	}
	if todos == nil {
		todos = []status.TodoItem{}
	}
	snap := status.Ok(dateStr, path, exists, todos)
	if e := enrichSnapshot(
		vault, date, path, cfg, &snap,
	); e != nil {
		return nil, e
	}
	return &snap, nil
}

func enrichSnapshot(
	vault *config.Vault,
	date time.Time,
	path string,
	cfg *config.DailyNotesConfig,
	snap *status.Snapshot,
) *config.VaultError {
	uri := open.OpenURI(vault.Root, path)
	snap.ObsidianURI = &uri

	today := time.Now()
	isToday := date.Year() == today.Year() &&
		date.Month() == today.Month() &&
		date.Day() == today.Day()
	snap.IsToday = &isToday

	count, err := carrySourceCount(vault, cfg, date)
	if err != nil {
		return err
	}
	snap.CarryOverCount = &count

	if cfg.Template != "" {
		name := cfg.Template
		snap.TemplateName = &name
		tp := vault.TemplatePath(cfg)
		hasTemplate := tp != "" && fileExists(tp)
		created := hasTemplate && fileExists(path)
		snap.CreatedFromTemplate = &created
	}
	return nil
}

const carryOverLookbackDays = 30

func carrySource(
	vault *config.Vault,
	cfg *config.DailyNotesConfig,
	date time.Time,
) (time.Time, int, bool, *config.VaultError) {
	for offset := 1; offset <= carryOverLookbackDays; offset++ {
		prev := date.AddDate(0, 0, -offset)
		count, err := openTodoCount(vault, cfg, prev)
		if err != nil {
			return time.Time{}, 0, false, err
		}
		if count > 0 {
			return prev, count, true, nil
		}
	}
	return time.Time{}, 0, false, nil
}

func carrySourceCount(
	vault *config.Vault,
	cfg *config.DailyNotesConfig,
	date time.Time,
) (int, *config.VaultError) {
	_, count, found, err := carrySource(vault, cfg, date)
	if err != nil {
		return 0, err
	}
	if !found {
		return 0, nil
	}
	return count, nil
}

func openTodoCount(
	vault *config.Vault,
	cfg *config.DailyNotesConfig,
	date time.Time,
) (int, *config.VaultError) {
	path, err := ResolvedNotePath(vault, cfg, date)
	if err != nil {
		return 0, err
	}
	if !fileExists(path) {
		return 0, nil
	}
	content, e := os.ReadFile(path)
	if e != nil {
		return 0, config.NewVaultError(
			"io",
			fmt.Sprintf(
				"failed to read %s: %v", path, e,
			),
		)
	}
	todos := ParseTodos(string(content))
	count := 0
	for _, t := range todos {
		if !t.Checked {
			count++
		}
	}
	return count, nil
}

type openTodo struct {
	depth int
	text  string
}

func openTodoItems(
	vault *config.Vault,
	cfg *config.DailyNotesConfig,
	date time.Time,
) ([]openTodo, *config.VaultError) {
	path, err := ResolvedNotePath(vault, cfg, date)
	if err != nil {
		return nil, err
	}
	if !fileExists(path) {
		return nil, nil
	}
	content, e := os.ReadFile(path)
	if e != nil {
		return nil, config.NewVaultError(
			"io",
			fmt.Sprintf(
				"failed to read %s: %v", path, e,
			),
		)
	}
	todos := ParseTodos(string(content))
	var items []openTodo
	for _, t := range todos {
		if !t.Checked {
			items = append(items, openTodo{
				depth: t.Depth,
				text:  t.Text,
			})
		}
	}
	return items, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// LoadTemplateBody loads and expands a template for a date.
func LoadTemplateBody(
	vault *config.Vault,
	cfg *config.DailyNotesConfig,
	date time.Time,
) string {
	tp := vault.TemplatePath(cfg)
	if tp != "" {
		if raw, e := os.ReadFile(tp); e == nil {
			return expandTemplate(string(raw), date)
		}
	}
	return fmt.Sprintf(
		"# %s\n", date.Format("2006-01-02"),
	)
}

func expandTemplate(raw string, date time.Time) string {
	ds := date.Format("2006-01-02")
	s := strings.ReplaceAll(raw, "{{date}}", ds)
	s = strings.ReplaceAll(s, "{{date:YYYY-MM-DD}}", ds)
	s = strings.ReplaceAll(s, "{{time}}", "")
	return s
}
