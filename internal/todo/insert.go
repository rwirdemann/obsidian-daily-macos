package todo

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ralf/obsidian-daily-macos/internal/config"
	"github.com/ralf/obsidian-daily-macos/internal/status"
	"github.com/ralf/obsidian-daily-macos/internal/undo"
)

// AddTodo appends a new open todo.
func AddTodo(
	vault *config.Vault,
	date time.Time,
	text string,
	heading string,
) (*status.Snapshot, *config.VaultError) {
	return AddTodoUnder(vault, date, text, 0, heading)
}

// AddTodoUnder appends a new open todo, optionally nested
// under a parent line.
func AddTodoUnder(
	vault *config.Vault,
	date time.Time,
	text string,
	underLine int,
	heading string,
) (*status.Snapshot, *config.VaultError) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, config.NewVaultError(
			"io", "todo text is empty",
		)
	}
	if strings.ContainsAny(text, "\n\r") {
		return nil, config.NewVaultError(
			"io", "todo text must be a single line",
		)
	}
	if underLine == 0 {
		items := []depthText{{depth: 0, text: text}}
		if err := addTodoLines(
			vault, date, items, heading,
		); err != nil {
			return nil, err
		}
	} else {
		cfg, err := vault.DailyNotesConfig()
		if err != nil {
			return nil, err
		}
		path, err := ResolvedNotePath(vault, cfg, date)
		if err != nil {
			return nil, err
		}
		if e := EnsureNote(
			vault, cfg, path, date, heading,
		); e != nil {
			return nil, e
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
		var parent *status.TodoItem
		for i := range todos {
			if todos[i].Line == underLine {
				parent = &todos[i]
				break
			}
		}
		if parent == nil {
			return nil, config.NewVaultError(
				"io",
				fmt.Sprintf(
					"under-line %d is not a todo",
					underLine,
				),
			)
		}
		depth := parent.Depth + 1
		newLine := fmt.Sprintf(
			"%s- [ ] %s",
			strings.Repeat("  ", depth), text,
		)
		lines := splitLines(string(content))
		// Insert after parent line (0-based index = underLine-1,
		// insert at underLine).
		idx := underLine
		out := make([]string, 0, len(lines)+1)
		out = append(out, lines[:idx]...)
		out = append(out, newLine)
		out = append(out, lines[idx:]...)
		next := joinBody(string(content), out)
		if e := writeAtomicWithUndo(
			vault, date, path,
			string(content), next,
		); e != nil {
			return nil, e
		}
	}
	return ReadSnapshot(vault, date)
}

type depthText struct {
	depth int
	text  string
}

func addTodoLines(
	vault *config.Vault,
	date time.Time,
	items []depthText,
	heading string,
) *config.VaultError {
	formatted := make([]checkboxItem, len(items))
	for i, it := range items {
		formatted[i] = checkboxItem{
			depth:   it.depth,
			checked: false,
			text:    it.text,
		}
	}
	return addCheckboxLines(
		vault, date, formatted, true, heading,
	)
}

type checkboxItem struct {
	depth   int
	checked bool
	text    string
}

func formatCheckboxLines(items []checkboxItem) []string {
	var prevDepth *int
	lines := make([]string, len(items))
	for i, it := range items {
		d := it.depth
		if prevDepth == nil {
			d = 0
		} else if d > *prevDepth+1 {
			d = *prevDepth + 1
		}
		pd := d
		prevDepth = &pd
		mark := " "
		if it.checked {
			mark = "x"
		}
		lines[i] = fmt.Sprintf(
			"%s- [%s] %s",
			strings.Repeat("  ", d), mark, it.text,
		)
	}
	return lines
}

func addCheckboxLines(
	vault *config.Vault,
	date time.Time,
	items []checkboxItem,
	rollover bool,
	heading string,
) *config.VaultError {
	cfg, err := vault.DailyNotesConfig()
	if err != nil {
		return err
	}
	path, err := ResolvedNotePath(vault, cfg, date)
	if err != nil {
		return err
	}
	if rollover {
		if e := EnsureNote(
			vault, cfg, path, date, heading,
		); e != nil {
			return e
		}
	} else {
		if e := CreateNoteIfMissing(
			vault, cfg, path, date,
		); e != nil {
			return e
		}
	}
	content, e := os.ReadFile(path)
	if e != nil {
		return config.NewVaultError(
			"io",
			fmt.Sprintf(
				"failed to read %s: %v", path, e,
			),
		)
	}
	lines := formatCheckboxLines(items)
	style := inferTodoStyle(
		vault, cfg, date, string(content),
	)
	next := insertTodoLines(
		string(content), lines, style, heading,
	)
	if rollover {
		return writeAtomicWithUndo(
			vault, date, path, string(content), next,
		)
	}
	return WriteAtomic(vault.Root, path, next)
}

// CreateNoteIfMissing creates the note if it doesn't exist.
func CreateNoteIfMissing(
	vault *config.Vault,
	cfg *config.DailyNotesConfig,
	path string,
	date time.Time,
) *config.VaultError {
	if fileExists(path) {
		return nil
	}
	parent := pathDir(path)
	if parent != "" {
		if e := CreateDirAllInVault(
			vault.Root, parent,
		); e != nil {
			return e
		}
	}
	body := LoadTemplateBody(vault, cfg, date)
	return WriteAtomic(vault.Root, path, body)
}

// EnsureNote creates the note if missing and carries over
// previous open todos.
func EnsureNote(
	vault *config.Vault,
	cfg *config.DailyNotesConfig,
	path string,
	date time.Time,
	heading string,
) *config.VaultError {
	if fileExists(path) {
		return nil
	}
	if e := CreateNoteIfMissing(
		vault, cfg, path, date,
	); e != nil {
		return e
	}
	_, e := MoveOpenFromPrevious(vault, date, heading)
	return e
}

func writeAtomicWithUndo(
	vault *config.Vault,
	date time.Time,
	path, before, after string,
) *config.VaultError {
	if e := undo.RecordBefore(
		vault, date, path, before,
	); e != nil {
		return e
	}
	return WriteAtomic(vault.Root, path, after)
}

func pathDir(path string) string {
	dir := path[:strings.LastIndex(path, "/")]
	if dir == "" {
		return "."
	}
	return dir
}

// todoStyle describes how checkbox lists are laid out.
type todoStyle struct {
	compact          bool
	blankAfterHeading bool
}

var defaultTodoStyle = todoStyle{
	compact:          true,
	blankAfterHeading: true,
}

const todoStyleLookbackDays = 14

func inferTodoStyle(
	vault *config.Vault,
	cfg *config.DailyNotesConfig,
	date time.Time,
	current string,
) todoStyle {
	style := defaultTodoStyle
	compactKnown := false
	headingGapKnown := false
	if applyStyleFromContent(
		current, &style,
		&compactKnown, &headingGapKnown,
	) && compactKnown && headingGapKnown {
		return style
	}
	for offset := 1; offset <= todoStyleLookbackDays; offset++ {
		prev := date.AddDate(0, 0, -offset)
		body := noteBody(vault, cfg, prev)
		if body == "" {
			continue
		}
		if applyStyleFromContent(
			body, &style,
			&compactKnown, &headingGapKnown,
		) && compactKnown && headingGapKnown {
			break
		}
	}
	return style
}

func noteBody(
	vault *config.Vault,
	cfg *config.DailyNotesConfig,
	date time.Time,
) string {
	path, err := ResolvedNotePath(vault, cfg, date)
	if err != nil {
		return ""
	}
	if !fileExists(path) {
		return ""
	}
	data, e := os.ReadFile(path)
	if e != nil {
		return ""
	}
	return string(data)
}

func applyStyleFromContent(
	content string,
	style *todoStyle,
	compactKnown, headingGapKnown *bool,
) bool {
	lines := splitLines(content)
	fenced := FenceMask(lines)
	region := firstTodoRegion(lines, fenced, 0, len(lines))
	if region == nil {
		return false
	}
	if !*headingGapKnown {
		style.blankAfterHeading = region.start > 0 &&
			strings.TrimSpace(lines[region.start-1]) == ""
		*headingGapKnown = true
	}
	if !*compactKnown {
		if c, ok := compactBetween(
			lines, fenced, region.start, region.end,
		); ok {
			style.compact = c
			*compactKnown = true
		}
	}
	return *compactKnown && *headingGapKnown
}

type regionBounds struct {
	start, end int
}

func firstTodoRegion(
	lines []string, fenced []bool, from, to int,
) *regionBounds {
	start := -1
	for i := from; i < to; i++ {
		if !fenced[i] && checkboxRe.MatchString(lines[i]) {
			start = i
			break
		}
	}
	if start < 0 {
		return nil
	}
	last := start
	i := start + 1
	for i < to {
		if fenced[i] {
			break
		}
		if checkboxRe.MatchString(lines[i]) {
			last = i
			i++
			continue
		}
		if strings.TrimSpace(lines[i]) == "" {
			i++
			continue
		}
		break
	}
	return &regionBounds{start, last + 1}
}

func compactBetween(
	lines []string, fenced []bool, start, end int,
) (bool, bool) {
	var prev *int
	dense := false
	spaced := false
	count := 0
	for i := start; i < end; i++ {
		if fenced[i] || !checkboxRe.MatchString(lines[i]) {
			continue
		}
		count++
		if prev != nil {
			gap := false
			for j := *prev + 1; j < i; j++ {
				if strings.TrimSpace(lines[j]) == "" {
					gap = true
					break
				}
			}
			if gap {
				spaced = true
			} else {
				dense = true
			}
		}
		p := i
		prev = &p
	}
	if count < 2 {
		return false, false
	}
	return dense || !spaced, true
}

func isTitleHeading(line string) bool {
	t := strings.TrimSpace(line)
	hashes := 0
	for _, c := range t {
		if c == '#' {
			hashes++
		} else {
			break
		}
	}
	if hashes < 1 || hashes > 6 {
		return false
	}
	return len(t) > hashes &&
		(t[hashes] == ' ' || t[hashes] == '\t')
}

func skipFrontmatter(lines []string) int {
	if len(lines) == 0 ||
		strings.TrimSpace(lines[0]) != "---" {
		return 0
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return i + 1
		}
	}
	return 0
}

func insertTodoLines(
	content string,
	items []string,
	style todoStyle,
	heading string,
) string {
	lines := splitLines(content)
	fenced := FenceMask(lines)
	out := make([]string, len(lines))
	copy(out, lines)

	// Custom heading.
	want := strings.TrimSpace(heading)
	if want != "" {
		if bounds := customSectionBounds(
			lines, want,
		); bounds != nil {
			compact := style.compact
			if r := firstTodoRegion(
				lines, fenced,
				bounds.start, bounds.end,
			); r != nil {
				if c, ok := compactBetween(
					lines, fenced, r.start, r.end,
				); ok {
					compact = c
				}
			}
			at, needsBoundary := tasksSectionInsertAt(
				lines, bounds.start, bounds.end,
			)
			after := insertItems(
				&out, at, items, compact,
			)
			if needsBoundary {
				ensureBlankBeforeFollowing(&out, after)
			}
			return finishBody(out)
		}
	}

	// Default Tasks/Todos heading.
	if bounds := tasksSectionBounds(lines); bounds != nil {
		compact := style.compact
		if r := firstTodoRegion(
			lines, fenced, bounds.start, bounds.end,
		); r != nil {
			if c, ok := compactBetween(
				lines, fenced, r.start, r.end,
			); ok {
				compact = c
			}
		}
		at, needsBoundary := tasksSectionInsertAt(
			lines, bounds.start, bounds.end,
		)
		after := insertItems(&out, at, items, compact)
		if needsBoundary {
			ensureBlankBeforeFollowing(&out, after)
		}
		return finishBody(out)
	}

	// Existing todo list.
	if r := firstTodoRegion(
		lines, fenced, 0, len(lines),
	); r != nil {
		compact := style.compact
		if c, ok := compactBetween(
			lines, fenced, r.start, r.end,
		); ok {
			compact = c
		}
		after := insertItems(&out, r.end, items, compact)
		ensureBlankBeforeFollowing(&out, after)
		return finishBody(out)
	}

	// After frontmatter + title.
	at := skipFrontmatter(lines)
	for at < len(lines) &&
		strings.TrimSpace(lines[at]) == "" {
		at++
	}
	if at < len(lines) && isTitleHeading(lines[at]) {
		at++
	}
	afterTitle := at
	if style.blankAfterHeading {
		if afterTitle < len(lines) &&
			strings.TrimSpace(lines[afterTitle]) == "" {
			at = afterTitle + 1
		} else if afterTitle > 0 {
			out = insertAt(out, afterTitle, "")
			at = afterTitle + 1
		}
	}
	after := insertItems(&out, at, items, style.compact)
	ensureBlankBeforeFollowing(&out, after)
	return finishBody(out)
}

func tasksSectionBounds(
	lines []string,
) *regionBounds {
	f := &FenceScanner{}
	for i, line := range lines {
		if f.IsFenced(line) {
			continue
		}
		if tasksHeadingRe.MatchString(line) {
			return sectionRangeFrom(lines, i)
		}
	}
	return nil
}

func customSectionBounds(
	lines []string, want string,
) *regionBounds {
	wantL := strings.ToLower(strings.TrimSpace(want))
	if wantL == "" {
		return nil
	}
	f := &FenceScanner{}
	for i, line := range lines {
		if f.IsFenced(line) {
			continue
		}
		m := anyHeadingRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		title := strings.ToLower(strings.TrimSpace(m[2]))
		if title == wantL {
			return sectionRangeFrom(lines, i)
		}
	}
	return nil
}

func sectionRangeFrom(
	lines []string, idx int,
) *regionBounds {
	m := anyHeadingRe.FindStringSubmatch(lines[idx])
	if m == nil {
		return nil
	}
	level := len(m[1])
	secStart := idx + 1
	secEnd := secStart
	f := &FenceScanner{}
	for secEnd < len(lines) {
		if !f.IsFenced(lines[secEnd]) {
			nm := anyHeadingRe.FindStringSubmatch(
				lines[secEnd],
			)
			if nm != nil && len(nm[1]) <= level {
				break
			}
		}
		secEnd++
	}
	return &regionBounds{secStart, secEnd}
}

func tasksSectionInsertAt(
	lines []string, secStart, secEnd int,
) (int, bool) {
	at := secEnd
	for at > secStart &&
		strings.TrimSpace(lines[at-1]) == "" {
		at--
	}
	needsBoundary := false
	if at == secStart && secStart < secEnd &&
		strings.TrimSpace(lines[secStart]) == "" {
		at++
		if at == secEnd {
			needsBoundary = true
		}
	}
	return at, needsBoundary
}

func insertItems(
	out *[]string, at int, items []string, compact bool,
) int {
	pos := at
	if !compact && pos > 0 &&
		strings.TrimSpace((*out)[pos-1]) != "" {
		*out = insertAt(*out, pos, "")
		pos++
	}
	for i, item := range items {
		if !compact && i > 0 {
			*out = insertAt(*out, pos, "")
			pos++
		}
		*out = insertAt(*out, pos, item)
		pos++
	}
	return pos
}

func ensureBlankBeforeFollowing(out *[]string, after int) {
	if after < len(*out) &&
		strings.TrimSpace((*out)[after]) != "" {
		*out = insertAt(*out, after, "")
	}
}

func insertAt(s []string, idx int, val string) []string {
	s = append(s, "")
	copy(s[idx+1:], s[idx:])
	s[idx] = val
	return s
}

func splitLines(content string) []string {
	if content == "" {
		return nil
	}
	s := strings.TrimSuffix(content, "\n")
	return strings.Split(s, "\n")
}

func joinBody(content string, lines []string) string {
	next := strings.Join(lines, "\n")
	if strings.HasSuffix(content, "\n") && next != "" {
		next += "\n"
	}
	return next
}

func finishBody(out []string) string {
	body := strings.Join(out, "\n")
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	return body
}

func rewriteLine(
	content string, line int, newLine string,
) string {
	lines := splitLines(content)
	lines[line-1] = newLine
	return joinBody(content, lines)
}

func withoutLines(
	content string, drop map[int]bool,
) string {
	lines := splitLines(content)
	var out []string
	for i, line := range lines {
		if !drop[i+1] {
			out = append(out, line)
		}
	}
	return joinBody(content, out)
}

func rejectFencedLine(
	content string, line int,
) *config.VaultError {
	lines := splitLines(content)
	mask := FenceMask(lines)
	if line-1 < len(mask) && mask[line-1] {
		return config.NewVaultError(
			"io",
			fmt.Sprintf(
				"line %d is inside a fenced code block",
				line,
			),
		)
	}
	return nil
}

func expectLineText(
	content string, line int, expectText string,
) *config.VaultError {
	expect := strings.TrimSpace(expectText)
	if expect == "" {
		return nil
	}
	todos := ParseTodos(content)
	for _, t := range todos {
		if t.Line == line {
			if strings.TrimSpace(t.Text) == expect {
				return nil
			}
			return config.NewVaultError(
				"io",
				fmt.Sprintf(
					"stale view: line %d now holds %q "+
						"instead of %q; "+
						"refetch and retry",
					line,
					strings.TrimSpace(t.Text),
					expect,
				),
			)
		}
	}
	return config.NewVaultError(
		"io",
		fmt.Sprintf(
			"stale view: line %d is no longer a todo; "+
				"refetch and retry",
			line,
		),
	)
}
