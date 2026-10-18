// Package status defines JSON snapshot types shared by the CLI
// and QML frontend.
package status

import "encoding/json"

// State is the result state of an operation.
type State string

const (
	StateOk    State = "ok"
	StateError State = "error"
)

// TodoItem represents a single checkbox todo in a daily note.
type TodoItem struct {
	Line       int    `json:"line"`
	Checked    bool   `json:"checked"`
	Text       string `json:"text"`
	Depth      int    `json:"depth"`
	ParentLine *int   `json:"parentLine,omitempty"`
}

// Snapshot is the main status response.
type Snapshot struct {
	St                  State       `json:"state"`
	Date                *string     `json:"date,omitempty"`
	Path                *string     `json:"path,omitempty"`
	Exists              *bool       `json:"exists,omitempty"`
	OpenCount           *int        `json:"openCount,omitempty"`
	DoneCount           *int        `json:"doneCount,omitempty"`
	Todos               *[]TodoItem `json:"todos,omitempty"`
	ObsidianURI         *string     `json:"obsidianUri,omitempty"`
	CarryOverCount      *int        `json:"carryOverCount,omitempty"`
	IsToday             *bool       `json:"isToday,omitempty"`
	TemplateName        *string     `json:"templateName,omitempty"`
	CreatedFromTemplate *bool       `json:"createdFromTemplate,omitempty"`
	ErrorCode           *string     `json:"errorCode,omitempty"`
	Error               *string     `json:"error,omitempty"`
}

// DaySummary holds counts for a single day.
type DaySummary struct {
	Date      string `json:"date"`
	OpenCount int    `json:"openCount"`
	DoneCount int    `json:"doneCount"`
	Exists    bool   `json:"exists"`
	IsToday   bool   `json:"isToday"`
}

// WeekSummary holds the week view response.
type WeekSummary struct {
	St        State        `json:"state"`
	Days      *[]DaySummary `json:"days,omitempty"`
	ErrorCode *string      `json:"errorCode,omitempty"`
	Error     *string      `json:"error,omitempty"`
}

func ptr[T any](v T) *T { return &v }

// ErrorWithCode creates an error snapshot.
func ErrorWithCode(message, code string) Snapshot {
	return Snapshot{
		St:        StateError,
		ErrorCode: &code,
		Error:     &message,
	}
}

// Ok creates a success snapshot from parsed todos.
func Ok(
	date, path string, exists bool, todos []TodoItem,
) Snapshot {
	open := 0
	done := 0
	for _, t := range todos {
		if t.Checked {
			done++
		} else {
			open++
		}
	}
	return Snapshot{
		St:        StateOk,
		Date:      &date,
		Path:      &path,
		Exists:    &exists,
		OpenCount: &open,
		DoneCount: &done,
		Todos:     &todos,
	}
}

// WeekError creates an error WeekSummary.
func WeekError(message, code string) WeekSummary {
	return WeekSummary{
		St:        StateError,
		ErrorCode: &code,
		Error:     &message,
	}
}

// JSON serializes a value to JSON.
func JSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
