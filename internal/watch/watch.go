// Package watch polls a daily note and emits JSON
// snapshots when it changes.
package watch

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/ralf/obsidian-daily-macos/internal/config"
	"github.com/ralf/obsidian-daily-macos/internal/status"
	"github.com/ralf/obsidian-daily-macos/internal/todo"
)

const pollInterval = time.Second

// Watch streams snapshots for date, or for today if date
// is nil: one immediately, then whenever path/mtime/content
// changes. It returns once the parent process exits, so no
// orphaned watcher outlives the app.
func Watch(
	cliVault, cliArchive, heading string,
	date *time.Time,
) {
	var lastKey string
	first := true
	parent := os.Getppid()
	for {
		if os.Getppid() != parent {
			return
		}
		snap := CurrentSnapshot(
			cliVault, cliArchive, heading, date,
		)
		key := snapshotKey(snap)
		if first || key != lastKey {
			if !emit(snap) {
				return
			}
			lastKey = key
			first = false
		}
		time.Sleep(pollInterval)
	}
}

// CurrentSnapshot gets the current snapshot for date, or
// for today if date is nil.
func CurrentSnapshot(
	cliVault, cliArchive, heading string,
	date *time.Time,
) *status.Snapshot {
	vault, err := config.Resolve(cliVault, cliArchive)
	if err != nil {
		snap := status.ErrorWithCode(
			err.Message, err.Code,
		)
		return &snap
	}
	d := time.Now()
	if date != nil {
		d = *date
	}
	snap, err := todo.ReadSnapshotFiltered(
		vault, d, heading,
	)
	if err != nil {
		s := status.ErrorWithCode(
			err.Message, err.Code,
		)
		return &s
	}
	return snap
}

func snapshotKey(snap *status.Snapshot) string {
	body, _ := json.Marshal(snap)
	if snap.Path != nil && snap.Exists != nil &&
		*snap.Exists {
		mtime := int64(0)
		length := int64(0)
		if info, err := os.Stat(*snap.Path); err == nil {
			mtime = info.ModTime().UnixMilli()
			length = info.Size()
		}
		return fmt.Sprintf(
			"ok:%s:%d:%d:%s",
			*snap.Path, mtime, length, body,
		)
	}
	return string(body)
}

func emit(snap *status.Snapshot) bool {
	data, _ := json.Marshal(snap)
	_, err := fmt.Fprintf(os.Stdout, "%s\n", data)
	return err == nil
}
