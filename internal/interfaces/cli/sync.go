package cli

import (
	"log"
	"time"

	"github.com/josemontalban/quiniela-mundial/internal/application/commands"
)

// SyncInput holds the date range for syncing results.
type SyncInput struct {
	From string // "2026-06-11"
	To   string // "2026-06-11"
}

// RunSync executes the sync command.
func RunSync(cmd *commands.SyncResults, input SyncInput) error {
	from, err := time.Parse("2006-01-02", input.From)
	if err != nil {
		from = time.Now().AddDate(0, 0, -1) // default: yesterday
	}
	to, err := time.Parse("2006-01-02", input.To)
	if err != nil {
		to = time.Now().AddDate(0, 0, 7) // default: next week
	}

	results, err := cmd.Execute(nil, from, to)
	if err != nil {
		return err
	}

	ok, skipped, errors := 0, 0, 0
	for _, r := range results {
		if r.Error != "" {
			errors++
		} else if r.Skipped {
			skipped++
		} else {
			ok++
		}
	}
	log.Printf("[sync] done: %d synced, %d skipped, %d errors, %d total pts awarded",
		ok, skipped, errors, sumPoints(results))
	return nil
}

func sumPoints(results []commands.SyncResult) int {
	total := 0
	for _, r := range results {
		total += r.TotalPoints
	}
	return total
}
