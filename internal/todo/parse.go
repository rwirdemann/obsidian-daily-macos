// Package todo parses and mutates markdown checkbox todos in
// a daily note.
package todo

import (
	"regexp"
	"strings"

	"github.com/ralf/obsidian-daily-macos/internal/status"
)

var checkboxRe = regexp.MustCompile(
	`^(\s*)([-*+])\s+\[([ xX])\](\s+)(.*)$`,
)

var tasksHeadingRe = regexp.MustCompile(
	`(?i)^#{1,6}\s+(tasks|todos)\s*$`,
)

var anyHeadingRe = regexp.MustCompile(
	`^ {0,3}(#{1,6})[ \t]+(.*?)(?:[ \t]+#+)?[ \t]*$`,
)

// FenceScanner tracks whether lines are inside markdown
// fenced code blocks.
type FenceScanner struct {
	openMarker byte
	openLen    int
	isOpen     bool
}

// IsFenced returns true for fence markers or content inside
// a fenced code block.
func (f *FenceScanner) IsFenced(line string) bool {
	b := []byte(line)
	indent := 0
	for indent < len(b) && b[indent] == ' ' {
		indent++
	}
	if indent > 3 {
		return f.isOpen
	}
	if indent >= len(b) {
		return f.isOpen
	}
	marker := b[indent]
	if marker != '`' && marker != '~' {
		return f.isOpen
	}
	length := 0
	for i := indent; i < len(b) && b[i] == marker; i++ {
		length++
	}

	if f.isOpen {
		if marker == f.openMarker &&
			length >= f.openLen &&
			isAllWhitespace(b[indent+length:]) {
			f.isOpen = false
		}
		return true
	}

	if length < 3 {
		return false
	}
	if marker == '`' && containsByte(b[indent+length:], '`') {
		return false
	}
	f.openMarker = marker
	f.openLen = length
	f.isOpen = true
	return true
}

func isAllWhitespace(b []byte) bool {
	for _, c := range b {
		if c != ' ' && c != '\t' && c != '\r' && c != '\n' {
			return false
		}
	}
	return true
}

func containsByte(b []byte, c byte) bool {
	for _, x := range b {
		if x == c {
			return true
		}
	}
	return false
}

// FenceMask returns a bitmask of lines that are fence markers
// or inside fenced code blocks.
func FenceMask(lines []string) []bool {
	f := &FenceScanner{}
	mask := make([]bool, len(lines))
	for i, line := range lines {
		mask[i] = f.IsFenced(line)
	}
	return mask
}

// IndentLevel counts indentation levels: every tab is one
// level, every two spaces one level (Obsidian convention).
func IndentLevel(indent string) int {
	tabs := 0
	spaces := 0
	for _, c := range indent {
		if c == '\t' {
			tabs++
		} else if c == ' ' {
			spaces++
		}
	}
	return tabs + spaces/2
}

// ParseTodos extracts all checkbox todos from note content.
// Line numbers are 1-based. Nesting is normalized: each todo
// nests at most one level deeper than the previous one.
func ParseTodos(content string) []status.TodoItem {
	lines := strings.Split(
		strings.TrimSuffix(content, "\n"), "\n",
	)
	if content == "" {
		lines = nil
	}
	fenced := FenceMask(lines)
	var todos []status.TodoItem
	var stack []int // line numbers of ancestors by depth
	lastDepth := 0

	for idx, line := range lines {
		if fenced[idx] {
			continue
		}
		m := checkboxRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		lineNo := idx + 1
		checked := m[3] != " "
		text := m[5]
		raw := IndentLevel(m[1])

		depth := 0
		if len(stack) > 0 {
			depth = raw
			if depth > lastDepth+1 {
				depth = lastDepth + 1
			}
		}

		for len(stack) > depth {
			stack = stack[:len(stack)-1]
		}

		var parentLine *int
		if len(stack) > 0 {
			pl := stack[len(stack)-1]
			parentLine = &pl
		}
		stack = append(stack, lineNo)
		lastDepth = depth

		todos = append(todos, status.TodoItem{
			Line:       lineNo,
			Checked:    checked,
			Text:       text,
			Depth:      depth,
			ParentLine: parentLine,
		})
	}
	return todos
}

// FilterTodosByHeading filters todos to those under a matching
// markdown heading section.
func FilterTodosByHeading(
	content string,
	todos []status.TodoItem,
	heading string,
) []status.TodoItem {
	want := strings.TrimSpace(heading)
	if want == "" {
		return todos
	}
	wantL := strings.ToLower(want)
	lines := strings.Split(
		strings.TrimSuffix(content, "\n"), "\n",
	)
	if content == "" {
		lines = nil
	}
	fenced := FenceMask(lines)

	type lineRange struct{ start, end int }
	var ranges []lineRange

	for i := 0; i < len(lines); i++ {
		if fenced[i] {
			continue
		}
		m := anyHeadingRe.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		level := len(m[1])
		title := strings.ToLower(strings.TrimSpace(m[2]))
		if title != wantL {
			continue
		}
		start := i + 2 // 1-based: i+2
		j := i + 1
		for j < len(lines) {
			if !fenced[j] {
				nm := anyHeadingRe.FindStringSubmatch(lines[j])
				if nm != nil && len(nm[1]) <= level {
					break
				}
			}
			j++
		}
		ranges = append(
			ranges, lineRange{start, j + 1},
		)
	}

	if len(ranges) == 0 {
		return nil
	}

	var result []status.TodoItem
	for _, t := range todos {
		for _, r := range ranges {
			if t.Line >= r.start && t.Line < r.end {
				result = append(result, t)
				break
			}
		}
	}
	return result
}
