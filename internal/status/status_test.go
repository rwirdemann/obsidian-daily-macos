package status

import (
	"encoding/json"
	"testing"
)

func TestSerializesOkSnapshot(t *testing.T) {
	snap := Ok(
		"2026-08-20", "/vault/2026-08-20.md", true,
		[]TodoItem{{
			Line: 3, Checked: false,
			Text: "Ship", Depth: 0,
		}},
	)
	data, err := json.Marshal(snap)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	json.Unmarshal(data, &m)
	if m["state"] != "ok" {
		t.Fatalf("state = %v", m["state"])
	}
	if m["openCount"].(float64) != 1 {
		t.Fatalf("openCount = %v", m["openCount"])
	}
	if m["doneCount"].(float64) != 0 {
		t.Fatalf("doneCount = %v", m["doneCount"])
	}
	todos := m["todos"].([]interface{})
	first := todos[0].(map[string]interface{})
	if first["line"].(float64) != 3 {
		t.Fatalf("line = %v", first["line"])
	}
}

func TestSerializesNestedTodos(t *testing.T) {
	pl := 3
	snap := Ok(
		"2026-08-20", "/vault/2026-08-20.md", true,
		[]TodoItem{
			{Line: 3, Checked: false, Text: "Parent",
				Depth: 0},
			{Line: 4, Checked: false, Text: "Child",
				Depth: 1, ParentLine: &pl},
		},
	)
	data, _ := json.Marshal(snap)
	var m map[string]interface{}
	json.Unmarshal(data, &m)
	todos := m["todos"].([]interface{})
	first := todos[0].(map[string]interface{})
	if _, ok := first["parentLine"]; ok {
		t.Fatal("parentLine should be omitted for depth 0")
	}
	second := todos[1].(map[string]interface{})
	if second["parentLine"].(float64) != 3 {
		t.Fatalf("parentLine = %v", second["parentLine"])
	}
	if second["depth"].(float64) != 1 {
		t.Fatalf("depth = %v", second["depth"])
	}
}

func TestSerializesErrorSnapshot(t *testing.T) {
	snap := ErrorWithCode("missing vault", "io")
	data, _ := json.Marshal(snap)
	s := string(data)
	if !contains(s, `"state":"error"`) {
		t.Fatalf("missing state:error in %s", s)
	}
	if !contains(s, "missing vault") {
		t.Fatalf("missing message in %s", s)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) &&
		containsStr(s, sub)
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
