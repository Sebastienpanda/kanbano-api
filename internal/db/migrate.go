package db

import (
	"log"
	"os/exec"
)

// RunMigrations applique les migrations au démarrage via `make migrate-up`.
func RunMigrations() {
	cmd := exec.Command("make", "migrate-up")
	output, err := cmd.CombinedOutput()
	log.Print(string(output))
	if err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}
}
