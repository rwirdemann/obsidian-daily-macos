package todo

import (
	"strings"
	"testing"
)

func TestToggleLineUncheckedToChecked(t *testing.T) {
	content := "- [ ] Buy milk\n"
	result, err := toggleLine(content, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, "- [x] Buy milk") {
		t.Fatalf("result = %q", result)
	}
}

func TestToggleLineCheckedToUnchecked(t *testing.T) {
	content := "- [x] Done\n"
	result, err := toggleLine(content, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, "- [ ] Done") {
		t.Fatalf("result = %q", result)
	}
}

func TestToggleLineNotCheckbox(t *testing.T) {
	content := "Some text\n"
	_, err := toggleLine(content, 1)
	if err == nil {
		t.Fatal("expected error for non-checkbox line")
	}
}

func TestToggleLinePastEnd(t *testing.T) {
	content := "- [ ] A\n"
	_, err := toggleLine(content, 5)
	if err == nil {
		t.Fatal("expected error for line past end")
	}
}

func TestToggleLinePreservesIndent(t *testing.T) {
	content := "  - [ ] Nested\n"
	result, err := toggleLine(content, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(
		strings.TrimSuffix(result, "\n"),
		"  - [x]",
	) {
		t.Fatalf(
			"should preserve indent: %q", result,
		)
	}
}

func TestToggleLineMultiple(t *testing.T) {
	content := strings.Join([]string{
		"- [ ] First",
		"- [x] Second",
		"- [ ] Third",
	}, "\n") + "\n"
	result, err := toggleLine(content, 2)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(
		strings.TrimSuffix(result, "\n"), "\n",
	)
	if lines[0] != "- [ ] First" {
		t.Fatalf("line 1 changed: %q", lines[0])
	}
	if lines[1] != "- [ ] Second" {
		t.Fatalf("line 2 = %q", lines[1])
	}
	if lines[2] != "- [ ] Third" {
		t.Fatalf("line 3 changed: %q", lines[2])
	}
}
