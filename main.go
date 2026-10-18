// Omarchy Quattro backend for Obsidian daily note todos.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ralf/obsidian-daily-macos/internal/config"
	"github.com/ralf/obsidian-daily-macos/internal/open"
	"github.com/ralf/obsidian-daily-macos/internal/status"
	"github.com/ralf/obsidian-daily-macos/internal/todo"
	"github.com/ralf/obsidian-daily-macos/internal/undo"
	"github.com/ralf/obsidian-daily-macos/internal/watch"
)

// Global flags.
var (
	flagVault         string
	flagArchiveFolder string
	flagStdin         bool
)

// stdinFields holds private fields from stdin JSON.
type stdinFields struct {
	Text          *string `json:"text"`
	ExpectText    *string `json:"expectText"`
	Vault         *string `json:"vault"`
	ArchiveFolder *string `json:"archiveFolder"`
}

func (s *stdinFields) vaultPath() string {
	if s.Vault != nil {
		t := strings.TrimSpace(*s.Vault)
		if t != "" {
			return t
		}
	}
	return ""
}

func (s *stdinFields) archive() string {
	if s.ArchiveFolder != nil {
		t := strings.TrimSpace(*s.ArchiveFolder)
		if t != "" {
			return t
		}
	}
	return ""
}

func init() {
	// Wire up the undo → todo dependency.
	undo.WriteAtomicPublicFn = todo.WriteAtomic

	// Wire up the todo → open dependency.
	todo.SetLauncher(&openLauncherImpl{})
}

type openLauncherImpl struct{}

func (o *openLauncherImpl) Launch(
	uri string,
) *config.VaultError {
	return open.Launch(uri)
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "obsidian-daily-macos",
		Short: "Backend for the Omarchy Obsidian Daily widget",
	}

	rootCmd.PersistentFlags().StringVar(
		&flagVault, "vault", "",
		"Absolute path to the Obsidian vault",
	)
	rootCmd.PersistentFlags().StringVar(
		&flagArchiveFolder, "archive-folder", "",
		"Archive folder pattern relative to vault root",
	)
	rootCmd.PersistentFlags().BoolVar(
		&flagStdin, "stdin", false,
		"Read private fields as JSON from stdin",
	)

	rootCmd.AddCommand(
		statusCmd(), watchCmd(), addCmd(),
		toggleCmd(), editCmd(), deleteCmd(),
		deferCmd(), indentCmd(), outdentCmd(),
		undoCmd(), weekCmd(), carryOverCmd(),
		openCmd(), inboxCmd(), inboxAddCmd(),
		inboxDeleteCmd(), inboxMoveCmd(),
		inboxToggleCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func readStdinFields() (*stdinFields, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to read stdin: %v", err,
		)
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil, fmt.Errorf(
			"expected JSON fields on stdin",
		)
	}
	// Two-stage parsing: ensure it's an object.
	var raw json.RawMessage
	if err := json.Unmarshal(
		[]byte(trimmed), &raw,
	); err != nil {
		return nil, fmt.Errorf("%s", stdinShapeError(0, 0))
	}
	if trimmed[0] != '{' {
		return nil, fmt.Errorf("%s", stdinShapeError(0, 0))
	}
	// Strict parsing with DisallowUnknownFields.
	dec := json.NewDecoder(strings.NewReader(trimmed))
	dec.DisallowUnknownFields()
	var fields stdinFields
	if err := dec.Decode(&fields); err != nil {
		return nil, fmt.Errorf(
			"%s", stdinShapeError(0, 0),
		)
	}
	return &fields, nil
}

func stdinShapeError(line, col int) string {
	where := ""
	if line > 0 {
		where = fmt.Sprintf(
			" (line %d, column %d)", line, col,
		)
	}
	return fmt.Sprintf(
		"invalid stdin JSON: expected an object with "+
			"`vault`/`archiveFolder`/`text`/"+
			"`expectText`%s",
		where,
	)
}

func resolveArgs(
	isWeek bool,
) (*stdinFields, string, string, error) {
	var fields stdinFields
	if flagStdin {
		f, err := readStdinFields()
		if err != nil {
			return nil, "", "", err
		}
		fields = *f
	}
	vaultArg := flagVault
	if vaultArg == "" {
		vaultArg = fields.vaultPath()
	}
	archiveArg := flagArchiveFolder
	if archiveArg == "" {
		archiveArg = fields.archive()
	}
	return &fields, vaultArg, archiveArg, nil
}

func resolveText(
	flagText string, fields *stdinFields,
) (string, error) {
	if flagStdin && fields != nil {
		return nonEmptyField(fields.Text, "text")
	}
	s := flagText
	return nonEmptyStr(s, "--text")
}

func resolveExpectText(
	flagExpect string, fields *stdinFields,
) string {
	if flagStdin && fields != nil {
		if fields.ExpectText != nil {
			t := strings.TrimSpace(*fields.ExpectText)
			if t != "" {
				return t
			}
		}
		return ""
	}
	return strings.TrimSpace(flagExpect)
}

func nonEmptyField(
	val *string, name string,
) (string, error) {
	if val != nil {
		s := strings.TrimSpace(*val)
		if s != "" {
			return s, nil
		}
	}
	return "", fmt.Errorf("missing %s", name)
}

func nonEmptyStr(val, name string) (string, error) {
	s := strings.TrimSpace(val)
	if s != "" {
		return s, nil
	}
	return "", fmt.Errorf("missing %s", name)
}

func parseDate(dateStr string) (time.Time, error) {
	if dateStr == "" {
		return time.Now(), nil
	}
	t, err := time.Parse(
		"2006-01-02", strings.TrimSpace(dateStr),
	)
	if err != nil {
		return time.Time{}, fmt.Errorf(
			"invalid --date %q; expected YYYY-MM-DD",
			dateStr,
		)
	}
	return t, nil
}

func emitSnapshot(snap *status.Snapshot) {
	data, _ := json.Marshal(snap)
	fmt.Println(string(data))
	if snap.St == status.StateError {
		os.Exit(1)
	}
}

func emitWeek(ws *status.WeekSummary) {
	data, _ := json.Marshal(ws)
	fmt.Println(string(data))
	if ws.St == status.StateError {
		os.Exit(1)
	}
}

func errorSnap(msg string) {
	snap := status.ErrorWithCode(msg, "io")
	emitSnapshot(&snap)
}

func errorWeek(msg string) {
	ws := status.WeekError(msg, "io")
	emitWeek(&ws)
}

func run(
	vaultArg, archiveArg, dateStr string,
	fn func(
		*config.Vault, time.Time,
	) (*status.Snapshot, *config.VaultError),
) {
	vault, err := config.Resolve(vaultArg, archiveArg)
	if err != nil {
		snap := status.ErrorWithCode(
			err.Message, err.Code,
		)
		emitSnapshot(&snap)
		return
	}
	date, e := parseDate(dateStr)
	if e != nil {
		errorSnap(e.Error())
		return
	}
	snap, err := fn(vault, date)
	if err != nil {
		s := status.ErrorWithCode(
			err.Message, err.Code,
		)
		emitSnapshot(&s)
		return
	}
	emitSnapshot(snap)
}

// Command definitions.

func statusCmd() *cobra.Command {
	var dateStr, heading string
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Print one status snapshot as JSON",
		Run: func(cmd *cobra.Command, args []string) {
			_, vaultArg, archiveArg, err := resolveArgs(
				false,
			)
			if err != nil {
				errorSnap(err.Error())
				return
			}
			run(vaultArg, archiveArg, dateStr,
				func(
					v *config.Vault, d time.Time,
				) (*status.Snapshot, *config.VaultError) {
					return todo.ReadSnapshotFiltered(
						v, d, heading,
					)
				},
			)
		},
	}
	cmd.Flags().StringVar(
		&dateStr, "date", "", "Target date",
	)
	cmd.Flags().StringVar(
		&heading, "heading", "",
		"Filter todos under this heading",
	)
	return cmd
}

func watchCmd() *cobra.Command {
	var dateStr, heading string
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Stream status snapshots as JSON lines",
		Run: func(cmd *cobra.Command, args []string) {
			_, vaultArg, archiveArg, err := resolveArgs(
				false,
			)
			if err != nil {
				errorSnap(err.Error())
				return
			}
			var date *time.Time
			if dateStr != "" {
				d, e := parseDate(dateStr)
				if e != nil {
					errorSnap(e.Error())
					return
				}
				date = &d
			}
			watch.Watch(
				vaultArg, archiveArg, heading, date,
			)
		},
	}
	cmd.Flags().StringVar(
		&dateStr, "date", "",
		"Target date (default: today)",
	)
	cmd.Flags().StringVar(
		&heading, "heading", "",
		"Filter by heading",
	)
	return cmd
}

func addCmd() *cobra.Command {
	var text, dateStr, heading string
	var underLine int
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add an open checkbox todo",
		Run: func(cmd *cobra.Command, args []string) {
			fields, vaultArg, archiveArg, err := resolveArgs(
				false,
			)
			if err != nil {
				errorSnap(err.Error())
				return
			}
			resolvedText, e := resolveText(text, fields)
			if e != nil {
				errorSnap(e.Error())
				return
			}
			run(vaultArg, archiveArg, dateStr,
				func(
					v *config.Vault, d time.Time,
				) (*status.Snapshot, *config.VaultError) {
					return todo.AddTodoUnder(
						v, d, resolvedText,
						underLine, heading,
					)
				},
			)
		},
	}
	cmd.Flags().StringVar(&text, "text", "", "Todo text")
	cmd.Flags().StringVar(&dateStr, "date", "", "Target date")
	cmd.Flags().IntVar(
		&underLine, "under-line", 0,
		"Nest under this todo line",
	)
	cmd.Flags().StringVar(
		&heading, "heading", "",
		"Insert under this heading",
	)
	return cmd
}

func toggleCmd() *cobra.Command {
	var line int
	var expectText, dateStr string
	cmd := &cobra.Command{
		Use:   "toggle",
		Short: "Toggle a checkbox",
		Run: func(cmd *cobra.Command, args []string) {
			fields, vaultArg, archiveArg, err := resolveArgs(
				false,
			)
			if err != nil {
				errorSnap(err.Error())
				return
			}
			et := resolveExpectText(expectText, fields)
			run(vaultArg, archiveArg, dateStr,
				func(
					v *config.Vault, d time.Time,
				) (*status.Snapshot, *config.VaultError) {
					return todo.ToggleTodo(v, d, line, et)
				},
			)
		},
	}
	cmd.Flags().IntVar(&line, "line", 0, "1-based line")
	cmd.MarkFlagRequired("line")
	cmd.Flags().StringVar(
		&expectText, "expect-text", "",
		"Expected text for stale-view check",
	)
	cmd.Flags().StringVar(&dateStr, "date", "", "Target date")
	return cmd
}

func editCmd() *cobra.Command {
	var line int
	var text, expectText, dateStr string
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Rewrite todo text",
		Run: func(cmd *cobra.Command, args []string) {
			fields, vaultArg, archiveArg, err := resolveArgs(
				false,
			)
			if err != nil {
				errorSnap(err.Error())
				return
			}
			resolvedText, e := resolveText(text, fields)
			if e != nil {
				errorSnap(e.Error())
				return
			}
			et := resolveExpectText(expectText, fields)
			run(vaultArg, archiveArg, dateStr,
				func(
					v *config.Vault, d time.Time,
				) (*status.Snapshot, *config.VaultError) {
					return todo.EditTodo(
						v, d, line, et, resolvedText,
					)
				},
			)
		},
	}
	cmd.Flags().IntVar(&line, "line", 0, "1-based line")
	cmd.MarkFlagRequired("line")
	cmd.Flags().StringVar(&text, "text", "", "New todo text")
	cmd.Flags().StringVar(
		&expectText, "expect-text", "",
		"Expected text",
	)
	cmd.Flags().StringVar(&dateStr, "date", "", "Target date")
	return cmd
}

func deleteCmd() *cobra.Command {
	var line int
	var expectText, dateStr string
	var withChildren bool
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a todo",
		Run: func(cmd *cobra.Command, args []string) {
			fields, vaultArg, archiveArg, err := resolveArgs(
				false,
			)
			if err != nil {
				errorSnap(err.Error())
				return
			}
			et := resolveExpectText(expectText, fields)
			run(vaultArg, archiveArg, dateStr,
				func(
					v *config.Vault, d time.Time,
				) (*status.Snapshot, *config.VaultError) {
					return todo.DeleteTodo(
						v, d, line, et, withChildren,
					)
				},
			)
		},
	}
	cmd.Flags().IntVar(&line, "line", 0, "1-based line")
	cmd.MarkFlagRequired("line")
	cmd.Flags().StringVar(
		&expectText, "expect-text", "",
		"Expected text",
	)
	cmd.Flags().BoolVar(
		&withChildren, "with-children", false,
		"Delete nested children",
	)
	cmd.Flags().StringVar(&dateStr, "date", "", "Target date")
	return cmd
}

func deferCmd() *cobra.Command {
	var line int
	var expectText, dateStr, heading string
	var withChildren bool
	// Use "defer" as the command name.
	cmd := &cobra.Command{
		Use:   "defer",
		Short: "Move a todo to the next day",
		Run: func(cmd *cobra.Command, args []string) {
			fields, vaultArg, archiveArg, err := resolveArgs(
				false,
			)
			if err != nil {
				errorSnap(err.Error())
				return
			}
			et := resolveExpectText(expectText, fields)
			run(vaultArg, archiveArg, dateStr,
				func(
					v *config.Vault, d time.Time,
				) (*status.Snapshot, *config.VaultError) {
					return todo.DeferTodo(
						v, d, line, et,
						withChildren, heading,
					)
				},
			)
		},
	}
	cmd.Flags().IntVar(&line, "line", 0, "1-based line")
	cmd.MarkFlagRequired("line")
	cmd.Flags().StringVar(
		&expectText, "expect-text", "",
		"Expected text",
	)
	cmd.Flags().BoolVar(
		&withChildren, "with-children", false,
		"Move nested children",
	)
	cmd.Flags().StringVar(&dateStr, "date", "", "Target date")
	cmd.Flags().StringVar(
		&heading, "heading", "",
		"Insert under heading in destination",
	)
	return cmd
}

func indentCmd() *cobra.Command {
	var line int
	var expectText, dateStr string
	cmd := &cobra.Command{
		Use:   "indent",
		Short: "Indent a todo one level",
		Run: func(cmd *cobra.Command, args []string) {
			fields, vaultArg, archiveArg, err := resolveArgs(
				false,
			)
			if err != nil {
				errorSnap(err.Error())
				return
			}
			et := resolveExpectText(expectText, fields)
			run(vaultArg, archiveArg, dateStr,
				func(
					v *config.Vault, d time.Time,
				) (*status.Snapshot, *config.VaultError) {
					return todo.SetIndent(
						v, d, line, et, 1,
					)
				},
			)
		},
	}
	cmd.Flags().IntVar(&line, "line", 0, "1-based line")
	cmd.MarkFlagRequired("line")
	cmd.Flags().StringVar(
		&expectText, "expect-text", "",
		"Expected text",
	)
	cmd.Flags().StringVar(&dateStr, "date", "", "Target date")
	return cmd
}

func outdentCmd() *cobra.Command {
	var line int
	var expectText, dateStr string
	cmd := &cobra.Command{
		Use:   "outdent",
		Short: "Outdent a todo one level",
		Run: func(cmd *cobra.Command, args []string) {
			fields, vaultArg, archiveArg, err := resolveArgs(
				false,
			)
			if err != nil {
				errorSnap(err.Error())
				return
			}
			et := resolveExpectText(expectText, fields)
			run(vaultArg, archiveArg, dateStr,
				func(
					v *config.Vault, d time.Time,
				) (*status.Snapshot, *config.VaultError) {
					return todo.SetIndent(
						v, d, line, et, -1,
					)
				},
			)
		},
	}
	cmd.Flags().IntVar(&line, "line", 0, "1-based line")
	cmd.MarkFlagRequired("line")
	cmd.Flags().StringVar(
		&expectText, "expect-text", "",
		"Expected text",
	)
	cmd.Flags().StringVar(&dateStr, "date", "", "Target date")
	return cmd
}

func undoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "undo",
		Short: "Restore previous note contents",
		Run: func(cmd *cobra.Command, args []string) {
			_, vaultArg, archiveArg, err := resolveArgs(
				false,
			)
			if err != nil {
				errorSnap(err.Error())
				return
			}
			vault, verr := config.Resolve(
				vaultArg, archiveArg,
			)
			if verr != nil {
				snap := status.ErrorWithCode(
					verr.Message, verr.Code,
				)
				emitSnapshot(&snap)
				return
			}
			result, verr := undo.UndoLast(
				vault,
				func(
					v *config.Vault, d time.Time,
				) (interface{}, *config.VaultError) {
					return todo.ReadSnapshot(v, d)
				},
			)
			if verr != nil {
				snap := status.ErrorWithCode(
					verr.Message, verr.Code,
				)
				emitSnapshot(&snap)
				return
			}
			if snap, ok := result.(*status.Snapshot); ok {
				emitSnapshot(snap)
			}
		},
	}
}

func weekCmd() *cobra.Command {
	var dateStr, heading string
	cmd := &cobra.Command{
		Use:   "week",
		Short: "Print Mon-Sun open/done counts",
		Run: func(cmd *cobra.Command, args []string) {
			_, vaultArg, archiveArg, err := resolveArgs(true)
			if err != nil {
				errorWeek(err.Error())
				return
			}
			vault, verr := config.Resolve(
				vaultArg, archiveArg,
			)
			if verr != nil {
				ws := status.WeekError(
					verr.Message, verr.Code,
				)
				emitWeek(&ws)
				return
			}
			date, e := parseDate(dateStr)
			if e != nil {
				errorWeek(e.Error())
				return
			}
			ws, verr := todo.WeekSummaryFiltered(
				vault, date, heading,
			)
			if verr != nil {
				w := status.WeekError(
					verr.Message, verr.Code,
				)
				emitWeek(&w)
				return
			}
			emitWeek(ws)
		},
	}
	cmd.Flags().StringVar(&dateStr, "date", "", "Target date")
	cmd.Flags().StringVar(
		&heading, "heading", "",
		"Filter by heading",
	)
	return cmd
}

func carryOverCmd() *cobra.Command {
	var dateStr, heading string
	cmd := &cobra.Command{
		Use:   "carry-over",
		Short: "Move previous open todos into target day",
		Run: func(cmd *cobra.Command, args []string) {
			_, vaultArg, archiveArg, err := resolveArgs(
				false,
			)
			if err != nil {
				errorSnap(err.Error())
				return
			}
			run(vaultArg, archiveArg, dateStr,
				func(
					v *config.Vault, d time.Time,
				) (*status.Snapshot, *config.VaultError) {
					return todo.CarryOver(v, d, heading)
				},
			)
		},
	}
	cmd.Flags().StringVar(&dateStr, "date", "", "Target date")
	cmd.Flags().StringVar(
		&heading, "heading", "",
		"Insert under heading",
	)
	return cmd
}

func openCmd() *cobra.Command {
	var dateStr string
	cmd := &cobra.Command{
		Use:   "open",
		Short: "Create daily note and open in Obsidian",
		Run: func(cmd *cobra.Command, args []string) {
			_, vaultArg, archiveArg, err := resolveArgs(
				false,
			)
			if err != nil {
				errorSnap(err.Error())
				return
			}
			run(vaultArg, archiveArg, dateStr,
				func(
					v *config.Vault, d time.Time,
				) (*status.Snapshot, *config.VaultError) {
					return todo.OpenInObsidian(v, d)
				},
			)
		},
	}
	cmd.Flags().StringVar(
		&dateStr, "date", "", "Target date",
	)
	return cmd
}

func inboxCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "inbox",
		Short: "Print Inbox.md todos as JSON",
		Run: func(cmd *cobra.Command, args []string) {
			_, vaultArg, archiveArg, err := resolveArgs(
				false,
			)
			if err != nil {
				errorSnap(err.Error())
				return
			}
			vault, verr := config.Resolve(
				vaultArg, archiveArg,
			)
			if verr != nil {
				snap := status.ErrorWithCode(
					verr.Message, verr.Code,
				)
				emitSnapshot(&snap)
				return
			}
			snap, verr := todo.ReadInbox(vault)
			if verr != nil {
				s := status.ErrorWithCode(
					verr.Message, verr.Code,
				)
				emitSnapshot(&s)
				return
			}
			emitSnapshot(snap)
		},
	}
}

func inboxAddCmd() *cobra.Command {
	var text string
	cmd := &cobra.Command{
		Use:   "inbox-add",
		Short: "Add a todo to Inbox.md",
		Run: func(cmd *cobra.Command, args []string) {
			fields, vaultArg, archiveArg, err :=
				resolveArgs(false)
			if err != nil {
				errorSnap(err.Error())
				return
			}
			resolvedText, e := resolveText(
				text, fields,
			)
			if e != nil {
				errorSnap(e.Error())
				return
			}
			vault, verr := config.Resolve(
				vaultArg, archiveArg,
			)
			if verr != nil {
				snap := status.ErrorWithCode(
					verr.Message, verr.Code,
				)
				emitSnapshot(&snap)
				return
			}
			snap, verr := todo.AddInboxTodo(
				vault, resolvedText,
			)
			if verr != nil {
				s := status.ErrorWithCode(
					verr.Message, verr.Code,
				)
				emitSnapshot(&s)
				return
			}
			emitSnapshot(snap)
		},
	}
	cmd.Flags().StringVar(
		&text, "text", "", "Todo text",
	)
	return cmd
}

func inboxDeleteCmd() *cobra.Command {
	var line int
	var expectText string
	cmd := &cobra.Command{
		Use:   "inbox-delete",
		Short: "Delete a todo from Inbox.md",
		Run: func(cmd *cobra.Command, args []string) {
			fields, vaultArg, archiveArg, err :=
				resolveArgs(false)
			if err != nil {
				errorSnap(err.Error())
				return
			}
			et := resolveExpectText(expectText, fields)
			vault, verr := config.Resolve(
				vaultArg, archiveArg,
			)
			if verr != nil {
				snap := status.ErrorWithCode(
					verr.Message, verr.Code,
				)
				emitSnapshot(&snap)
				return
			}
			snap, verr := todo.DeleteInboxTodo(
				vault, line, et,
			)
			if verr != nil {
				s := status.ErrorWithCode(
					verr.Message, verr.Code,
				)
				emitSnapshot(&s)
				return
			}
			emitSnapshot(snap)
		},
	}
	cmd.Flags().IntVar(
		&line, "line", 0,
		"1-based line in Inbox.md",
	)
	cmd.MarkFlagRequired("line")
	cmd.Flags().StringVar(
		&expectText, "expect-text", "",
		"Expected text for stale-view check",
	)
	return cmd
}

func inboxToggleCmd() *cobra.Command {
	var line int
	var expectText string
	cmd := &cobra.Command{
		Use:   "inbox-toggle",
		Short: "Toggle a todo checkbox in Inbox.md",
		Run: func(cmd *cobra.Command, args []string) {
			fields, vaultArg, archiveArg, err :=
				resolveArgs(false)
			if err != nil {
				errorSnap(err.Error())
				return
			}
			et := resolveExpectText(expectText, fields)
			vault, verr := config.Resolve(
				vaultArg, archiveArg,
			)
			if verr != nil {
				snap := status.ErrorWithCode(
					verr.Message, verr.Code,
				)
				emitSnapshot(&snap)
				return
			}
			snap, verr := todo.ToggleInboxTodo(
				vault, line, et,
			)
			if verr != nil {
				s := status.ErrorWithCode(
					verr.Message, verr.Code,
				)
				emitSnapshot(&s)
				return
			}
			emitSnapshot(snap)
		},
	}
	cmd.Flags().IntVar(
		&line, "line", 0,
		"1-based line in Inbox.md",
	)
	cmd.MarkFlagRequired("line")
	cmd.Flags().StringVar(
		&expectText, "expect-text", "",
		"Expected text for stale-view check",
	)
	return cmd
}

func inboxMoveCmd() *cobra.Command {
	var line int
	var expectText, dateStr, heading string
	cmd := &cobra.Command{
		Use:   "inbox-move",
		Short: "Move an inbox todo into a daily note",
		Run: func(cmd *cobra.Command, args []string) {
			fields, vaultArg, archiveArg, err :=
				resolveArgs(false)
			if err != nil {
				errorSnap(err.Error())
				return
			}
			et := resolveExpectText(expectText, fields)
			run(vaultArg, archiveArg, dateStr,
				func(
					v *config.Vault, d time.Time,
				) (*status.Snapshot, *config.VaultError) {
					return todo.MoveInboxTodo(
						v, line, et, d, heading,
					)
				},
			)
		},
	}
	cmd.Flags().IntVar(
		&line, "line", 0, "1-based line in Inbox.md",
	)
	cmd.MarkFlagRequired("line")
	cmd.Flags().StringVar(
		&expectText, "expect-text", "",
		"Expected text for stale-view check",
	)
	cmd.Flags().StringVar(
		&dateStr, "date", "", "Target date",
	)
	cmd.Flags().StringVar(
		&heading, "heading", "",
		"Insert under heading in daily note",
	)
	return cmd
}
