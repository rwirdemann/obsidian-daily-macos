package format

import (
	"testing"
	"time"
)

func date(y, m, d int) time.Time {
	return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
}

func TestFormatsDefaultISO(t *testing.T) {
	got, err := FormatMoment("YYYY-MM-DD", date(2026, 8, 20))
	if err != nil {
		t.Fatal(err)
	}
	if got != "2026-08-20" {
		t.Fatalf("got %q, want 2026-08-20", got)
	}
}

func TestFormatsNestedFolders(t *testing.T) {
	got, err := FormatMoment(
		"YYYY/MMMM/YYYY-MMM-DD", date(2026, 1, 5),
	)
	if err != nil {
		t.Fatal(err)
	}
	want := "2026/January/2026-Jan-05"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestFormatsWeekday(t *testing.T) {
	d := date(2026, 8, 20) // Thursday
	got, err := FormatMoment("dddd", d)
	if err != nil {
		t.Fatal(err)
	}
	if got != "Thursday" {
		t.Fatalf("got %q, want Thursday", got)
	}
	got, err = FormatMoment("ddd", d)
	if err != nil {
		t.Fatal(err)
	}
	if got != "Thu" {
		t.Fatalf("got %q, want Thu", got)
	}
}

func TestEmptyPatternErrors(t *testing.T) {
	_, err := FormatMoment("", date(2026, 1, 1))
	if err == nil {
		t.Fatal("expected error for empty pattern")
	}
}

func TestLiteralTextPreserved(t *testing.T) {
	got, err := FormatMoment(
		"note-YYYY-MM-DD", date(2026, 3, 15),
	)
	if err != nil {
		t.Fatal(err)
	}
	if got != "note-2026-03-15" {
		t.Fatalf("got %q", got)
	}
}

func TestYYToken(t *testing.T) {
	got, err := FormatMoment("YY", date(2026, 1, 1))
	if err != nil {
		t.Fatal(err)
	}
	if got != "26" {
		t.Fatalf("got %q, want 26", got)
	}
}

func TestUnpaddedTokens(t *testing.T) {
	got, err := FormatMoment("M-D", date(2026, 3, 5))
	if err != nil {
		t.Fatal(err)
	}
	if got != "3-5" {
		t.Fatalf("got %q, want 3-5", got)
	}
}
