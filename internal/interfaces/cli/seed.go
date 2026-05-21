package cli

import (
	"context"
	"log"
	"os"

	"github.com/josemontalban/quiniela-mundial/internal/application/commands"
)

// RunSeed executes the seed command. It wires dependencies manually
// (wire-based DI will replace this in later phases).
func RunSeed(seedCmd *commands.SeedTournament) {
	ctx := context.Background()
	if err := seedCmd.Execute(ctx); err != nil {
		log.Printf("[seed] ERROR: %v", err)
		os.Exit(1)
	}
	log.Println("[seed] completed successfully")
}
