package todo

import (
	"strings"
	"testing"
)

func TestFormatCheckboxLines(t *testing.T) {
	items := []checkboxItem{
		{depth: 0, checked: false, text: "Buy milk"},
		{depth: 1, checked: true, text: "Whole"},
	}
	lines := formatCheckboxLines(items)
	if lines[0] != "- [ ] Buy milk" {
		t.Fatalf("line 0 = %q", lines[0])
	}
	if lines[1] != "  - [x] Whole" {
		t.Fatalf("line 1 = %q", lines[1])
	}
}

func TestFormatCheckboxNormalizesDepth(t *testing.T) {
	items := []checkboxItem{
		{depth: 3, checked: false, text: "Deep"},
	}
	lines := formatCheckboxLines(items)
	if lines[0] != "- [ ] Deep" {
		t.Fatalf(
			"first item should be depth 0: %q", lines[0],
		)
	}
}

func TestInsertTodoLinesAppendToExisting(t *testing.T) {
	content := strings.Join([]string{
		"# Title",
		"",
		"- [ ] First",
	}, "\n") + "\n"
	items := []string{"- [ ] Second"}
	style := todoStyle{compact: true, blankAfterHeading: true}
	result := insertTodoLines(content, items, style, "")
	if !strings.Contains(result, "- [ ] First") ||
		!strings.Contains(result, "- [ ] Second") {
		t.Fatalf("result = %q", result)
	}
}

func TestInsertTodoLinesUnderTasksHeading(t *testing.T) {
	content := strings.Join([]string{
		"# Title",
		"",
		"## Tasks",
		"",
		"- [ ] Existing",
		"",
		"## Notes",
		"Some text",
	}, "\n") + "\n"
	items := []string{"- [ ] New"}
	style := todoStyle{compact: true, blankAfterHeading: true}
	result := insertTodoLines(content, items, style, "")
	lines := strings.Split(
		strings.TrimSuffix(result, "\n"), "\n",
	)
	newIdx := -1
	notesIdx := -1
	for i, l := range lines {
		if l == "- [ ] New" {
			newIdx = i
		}
		if l == "## Notes" {
			notesIdx = i
		}
	}
	if newIdx == -1 {
		t.Fatal("new todo not found")
	}
	if notesIdx == -1 {
		t.Fatal("Notes heading not found")
	}
	if newIdx >= notesIdx {
		t.Fatalf(
			"new (%d) should be before Notes (%d)",
			newIdx, notesIdx,
		)
	}
}

func TestInsertTodoLinesCustomHeading(t *testing.T) {
	content := strings.Join([]string{
		"# Title",
		"",
		"## My Tasks",
		"",
		"## Notes",
	}, "\n") + "\n"
	items := []string{"- [ ] New"}
	style := todoStyle{compact: true, blankAfterHeading: true}
	result := insertTodoLines(
		content, items, style, "My Tasks",
	)
	lines := strings.Split(
		strings.TrimSuffix(result, "\n"), "\n",
	)
	headIdx := -1
	newIdx := -1
	notesIdx := -1
	for i, l := range lines {
		if l == "## My Tasks" {
			headIdx = i
		}
		if l == "- [ ] New" {
			newIdx = i
		}
		if l == "## Notes" {
			notesIdx = i
		}
	}
	if newIdx < headIdx || newIdx >= notesIdx {
		t.Fatalf(
			"new (%d) should be between heading (%d) "+
				"and Notes (%d)",
			newIdx, headIdx, notesIdx,
		)
	}
}

func TestInsertTodoLinesAfterTitle(t *testing.T) {
	content := "# Title\n"
	items := []string{"- [ ] New"}
	style := todoStyle{compact: true, blankAfterHeading: true}
	result := insertTodoLines(content, items, style, "")
	if !strings.Contains(result, "# Title") {
		t.Fatal("title lost")
	}
	if !strings.Contains(result, "- [ ] New") {
		t.Fatal("todo missing")
	}
}

func TestInsertTodoLinesSpacedStyle(t *testing.T) {
	content := "# Title\n\n"
	items := []string{"- [ ] A", "- [ ] B"}
	style := todoStyle{
		compact:           false,
		blankAfterHeading: true,
	}
	result := insertTodoLines(content, items, style, "")
	lines := strings.Split(
		strings.TrimSuffix(result, "\n"), "\n",
	)
	idxA := -1
	idxB := -1
	for i, l := range lines {
		if l == "- [ ] A" {
			idxA = i
		}
		if l == "- [ ] B" {
			idxB = i
		}
	}
	if idxA == -1 || idxB == -1 {
		t.Fatalf("items not found in %q", result)
	}
	if idxB-idxA < 2 {
		t.Fatal("spaced style should have gap between items")
	}
}

func TestSplitLinesEmpty(t *testing.T) {
	if splitLines("") != nil {
		t.Fatal("empty should return nil")
	}
}

func TestSplitLinesTrailingNewline(t *testing.T) {
	lines := splitLines("a\nb\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines", len(lines))
	}
}

func TestJoinBodyPreservesTrailingNewline(t *testing.T) {
	original := "a\nb\n"
	lines := []string{"a", "b", "c"}
	result := joinBody(original, lines)
	if !strings.HasSuffix(result, "\n") {
		t.Fatal("should preserve trailing newline")
	}
}

func TestFinishBodyAddsTrailingNewline(t *testing.T) {
	out := []string{"a", "b"}
	result := finishBody(out)
	if !strings.HasSuffix(result, "\n") {
		t.Fatal("should ensure trailing newline")
	}
}

func TestRewriteLine(t *testing.T) {
	content := "line1\nline2\nline3\n"
	result := rewriteLine(content, 2, "replaced")
	if !strings.Contains(result, "replaced") {
		t.Fatalf("result = %q", result)
	}
	if strings.Contains(result, "line2") {
		t.Fatal("old line2 should be gone")
	}
}

func TestWithoutLines(t *testing.T) {
	content := "a\nb\nc\n"
	drop := map[int]bool{2: true}
	result := withoutLines(content, drop)
	if strings.Contains(result, "b") {
		t.Fatalf("line 2 should be removed: %q", result)
	}
	if !strings.Contains(result, "a") ||
		!strings.Contains(result, "c") {
		t.Fatalf("other lines lost: %q", result)
	}
}

func TestExpectLineTextMatchesOk(t *testing.T) {
	content := "- [ ] Buy milk\n"
	err := expectLineText(content, 1, "Buy milk")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExpectLineTextMismatch(t *testing.T) {
	content := "- [ ] Buy milk\n"
	err := expectLineText(content, 1, "Sell milk")
	if err == nil {
		t.Fatal("expected stale view error")
	}
}

func TestExpectLineTextEmptySkips(t *testing.T) {
	content := "- [ ] Buy milk\n"
	err := expectLineText(content, 1, "")
	if err != nil {
		t.Fatal("empty expectText should skip check")
	}
}

func TestRejectFencedLine(t *testing.T) {
	content := strings.Join([]string{
		"```",
		"- [ ] Inside fence",
		"```",
	}, "\n") + "\n"
	err := rejectFencedLine(content, 2)
	if err == nil {
		t.Fatal("should reject fenced line")
	}
}

func TestRejectFencedLineOutside(t *testing.T) {
	content := "- [ ] Outside\n"
	err := rejectFencedLine(content, 1)
	if err != nil {
		t.Fatal("should not reject normal line")
	}
}

func TestSkipFrontmatter(t *testing.T) {
	lines := []string{
		"---", "title: Test", "---", "# Title",
	}
	idx := skipFrontmatter(lines)
	if idx != 3 {
		t.Fatalf("skip = %d, want 3", idx)
	}
}

func TestSkipFrontmatterNone(t *testing.T) {
	lines := []string{"# Title", "content"}
	idx := skipFrontmatter(lines)
	if idx != 0 {
		t.Fatalf("skip = %d, want 0", idx)
	}
}
