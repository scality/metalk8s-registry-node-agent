package main

import (
	"log"
	"os"
)

const (
	// The UID/GID 65532 should match the UID/GID used for Registry-Node-Agent
	NonRootUID = 65532
	NonRootGID = 65532
)

const (
	devicesLocation                 = "/devices"
	solutionArchivesLocationEnv     = "SOLUTION_ARCHIVES_LOCATION" // Must match Environment.SolutionArchivesLocation
	defaultSolutionArchivesLocation = "/archives"                  // Must match Environment.SolutionArchivesLocation
	solutionsLocationEnv            = "SOLUTIONS_LOCATION"         // Must match Environment.SolutionsLocation
	defaultSolutionsLocation        = "/solutions"                 // Must match Environment.SolutionsLocation
)

// envOrDefault returns the value of the environment variable key, or fallback when it is unset or empty.
func envOrDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

func main() {
	paths := []string{
		devicesLocation,
		envOrDefault(solutionArchivesLocationEnv, defaultSolutionArchivesLocation),
		envOrDefault(solutionsLocationEnv, defaultSolutionsLocation),
	}
	for _, path := range paths {
		if err := os.Chown(path, NonRootUID, NonRootGID); err != nil {
			log.Fatalf("Error setting up permissions for %s: %v", path, err)
		}
	}
}
