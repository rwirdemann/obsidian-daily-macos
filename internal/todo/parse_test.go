package todo

import (
	"strings"
	"testing"
)

func TestParseSingleTodo(t *testing.T) {
	todos := ParseTodos("- [ ] Buy milk\n")
	if len(todos) != 1 {
		t.Fatalf("got %d todos", len(todos))
	}
	if todos[0].Line != 1 {
		t.Fatalf("line = %d", todos[0].Line)
	}
	if todos[0].Checked {
		t.Fatal("should be unchecked")
	}
	if todos[0].Text != "Buy milk" {
		t.Fatalf("text = %q", todos[0].Text)
	}
	if todos[0].Depth != 0 {
		t.Fatalf("depth = %d", todos[0].Depth)
	}
}

func TestParseCheckedTodo(t *testing.T) {
	todos := ParseTodos("- [x] Done\n")
	if len(todos) != 1 {
		t.Fatalf("got %d todos", len(todos))
	}
	if !todos[0].Checked {
		t.Fatal("should be checked")
	}
}

func TestParseUppercaseX(t *testing.T) {
	todos := ParseTodos("- [X] Done\n")
	if len(todos) != 1 || !todos[0].Checked {
		t.Fatal("uppercase X should be checked")
	}
}

func TestParseNestedTodos(t *testing.T) {
	content := strings.Join([]string{
		"- [ ] Parent",
		"  - [ ] Child",
		"    - [ ] Grandchild",
	}, "\n") + "\n"
	todos := ParseTodos(content)
	if len(todos) != 3 {
		t.Fatalf("got %d todos", len(todos))
	}
	if todos[0].Depth != 0 {
		t.Fatalf("parent depth = %d", todos[0].Depth)
	}
	if todos[1].Depth != 1 {
		t.Fatalf("child depth = %d", todos[1].Depth)
	}
	if todos[2].Depth != 2 {
		t.Fatalf("grandchild depth = %d", todos[2].Depth)
	}
	if todos[1].ParentLine == nil ||
		*todos[1].ParentLine != 1 {
		t.Fatalf(
			"child parentLine = %v", todos[1].ParentLine,
		)
	}
	if todos[2].ParentLine == nil ||
		*todos[2].ParentLine != 2 {
		t.Fatalf(
			"grandchild parentLine = %v",
			todos[2].ParentLine,
		)
	}
}

func TestParseNormalizesDepthJump(t *testing.T) {
	content := strings.Join([]string{
		"- [ ] A",
		"        - [ ] B",
	}, "\n") + "\n"
	todos := ParseTodos(content)
	if len(todos) != 2 {
		t.Fatalf("got %d todos", len(todos))
	}
	if todos[1].Depth != 1 {
		t.Fatalf(
			"normalized depth = %d, want 1",
			todos[1].Depth,
		)
	}
}

func TestParseEmptyContent(t *testing.T) {
	todos := ParseTodos("")
	if len(todos) != 0 {
		t.Fatalf("got %d todos for empty", len(todos))
	}
}

func TestParseSkipsNonCheckboxLines(t *testing.T) {
	content := strings.Join([]string{
		"# Title",
		"Some text",
		"- [ ] Real todo",
		"- plain list item",
	}, "\n") + "\n"
	todos := ParseTodos(content)
	if len(todos) != 1 {
		t.Fatalf("got %d todos", len(todos))
	}
	if todos[0].Text != "Real todo" {
		t.Fatalf("text = %q", todos[0].Text)
	}
}

func TestParseMultipleMarkerTypes(t *testing.T) {
	content := strings.Join([]string{
		"- [ ] Dash",
		"* [ ] Star",
		"+ [ ] Plus",
	}, "\n") + "\n"
	todos := ParseTodos(content)
	if len(todos) != 3 {
		t.Fatalf("got %d todos", len(todos))
	}
}

func TestFenceScannerSkipsFencedCheckbox(t *testing.T) {
	content := strings.Join([]string{
		"- [ ] Real",
		"```",
		"- [ ] Inside fence",
		"```",
		"- [ ] Also real",
	}, "\n") + "\n"
	todos := ParseTodos(content)
	if len(todos) != 2 {
		t.Fatalf("got %d todos, want 2", len(todos))
	}
	if todos[0].Text != "Real" {
		t.Fatalf("first = %q", todos[0].Text)
	}
	if todos[1].Text != "Also real" {
		t.Fatalf("second = %q", todos[1].Text)
	}
}

func TestFenceScannerTildeFence(t *testing.T) {
	content := strings.Join([]string{
		"~~~",
		"- [ ] Inside",
		"~~~",
		"- [ ] Outside",
	}, "\n") + "\n"
	todos := ParseTodos(content)
	if len(todos) != 1 {
		t.Fatalf("got %d todos", len(todos))
	}
	if todos[0].Text != "Outside" {
		t.Fatalf("text = %q", todos[0].Text)
	}
}

func TestFenceMask(t *testing.T) {
	lines := []string{
		"normal",
		"```",
		"code",
		"```",
		"normal again",
	}
	mask := FenceMask(lines)
	if mask[0] {
		t.Fatal("line 0 should not be fenced")
	}
	if !mask[1] {
		t.Fatal("line 1 should be fenced (open)")
	}
	if !mask[2] {
		t.Fatal("line 2 should be fenced")
	}
	if !mask[3] {
		t.Fatal("line 3 should be fenced (close)")
	}
	if mask[4] {
		t.Fatal("line 4 should not be fenced")
	}
}

func TestIndentLevel(t *testing.T) {
	tests := []struct {
		indent string
		want   int
	}{
		{"", 0},
		{"  ", 1},
		{"    ", 2},
		{"\t", 1},
		{"\t\t", 2},
		{" ", 0},
		{"   ", 1},
	}
	for _, tt := range tests {
		got := IndentLevel(tt.indent)
		if got != tt.want {
			t.Errorf(
				"IndentLevel(%q) = %d, want %d",
				tt.indent, got, tt.want,
			)
		}
	}
}

func TestFilterTodosByHeading(t *testing.T) {
	content := strings.Join([]string{
		"# Title",
		"- [ ] Top-level",
		"## Tasks",
		"- [ ] Under tasks",
		"## Notes",
		"- [ ] Under notes",
	}, "\n") + "\n"
	all := ParseTodos(content)
	if len(all) != 3 {
		t.Fatalf("total = %d", len(all))
	}
	filtered := FilterTodosByHeading(
		content, all, "Tasks",
	)
	if len(filtered) != 1 {
		t.Fatalf("filtered = %d", len(filtered))
	}
	if filtered[0].Text != "Under tasks" {
		t.Fatalf("text = %q", filtered[0].Text)
	}
}

func TestFilterTodosByHeadingEmpty(t *testing.T) {
	content := "- [ ] A\n"
	all := ParseTodos(content)
	filtered := FilterTodosByHeading(content, all, "")
	if len(filtered) != len(all) {
		t.Fatal("empty heading should return all")
	}
}

func TestFilterTodosByHeadingCaseInsensitive(t *testing.T) {
	content := strings.Join([]string{
		"## TASKS",
		"- [ ] Item",
	}, "\n") + "\n"
	all := ParseTodos(content)
	filtered := FilterTodosByHeading(
		content, all, "tasks",
	)
	if len(filtered) != 1 {
		t.Fatal("should match case-insensitively")
	}
}

func TestFilterTodosByHeadingNoMatch(t *testing.T) {
	content := strings.Join([]string{
		"## Notes",
		"- [ ] Item",
	}, "\n") + "\n"
	all := ParseTodos(content)
	filtered := FilterTodosByHeading(
		content, all, "Tasks",
	)
	if len(filtered) != 0 {
		t.Fatalf("should be empty, got %d", len(filtered))
	}
}
