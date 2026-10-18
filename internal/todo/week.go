package todo

import (
	"time"

	"github.com/ralf/obsidian-daily-macos/internal/config"
	"github.com/ralf/obsidian-daily-macos/internal/status"
)

// WeekSummary returns Mon-Sun open/done counts.
func WeekSummary(
	vault *config.Vault, anchor time.Time,
) (*status.WeekSummary, *config.VaultError) {
	return WeekSummaryFiltered(vault, anchor, "")
}

// WeekSummaryFiltered returns Mon-Sun counts, optionally
// filtering each day by heading.
func WeekSummaryFiltered(
	vault *config.Vault,
	anchor time.Time,
	heading string,
) (*status.WeekSummary, *config.VaultError) {
	today := time.Now()
	// Find Monday of anchor's week.
	wd := int(anchor.Weekday())
	if wd == 0 {
		wd = 7 // Sunday = 7
	}
	daysFromMon := wd - 1
	monday := anchor.AddDate(0, 0, -daysFromMon)

	days := make([]status.DaySummary, 0, 7)
	for offset := 0; offset < 7; offset++ {
		d := monday.AddDate(0, 0, offset)
		snap, err := ReadSnapshotFiltered(
			vault, d, heading,
		)
		if err != nil {
			return nil, err
		}
		oc := 0
		if snap.OpenCount != nil {
			oc = *snap.OpenCount
		}
		dc := 0
		if snap.DoneCount != nil {
			dc = *snap.DoneCount
		}
		ex := false
		if snap.Exists != nil {
			ex = *snap.Exists
		}
		isT := d.Year() == today.Year() &&
			d.Month() == today.Month() &&
			d.Day() == today.Day()
		days = append(days, status.DaySummary{
			Date:      d.Format("2006-01-02"),
			OpenCount: oc,
			DoneCount: dc,
			Exists:    ex,
			IsToday:   isT,
		})
	}
	return &status.WeekSummary{
		St:   status.StateOk,
		Days: &days,
	}, nil
}
