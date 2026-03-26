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

func main() {
	err := os.Chown("/devices", NonRootUID, NonRootGID)
	if err != nil {
		log.Fatalf("Error setting up permissions for /devices: %v", err)
	}
	err = os.Chown("/archives", NonRootUID, NonRootGID)
	if err != nil {
		log.Fatalf("Error setting up permissions for /archives: %v", err)
	}
	err = os.Chown("/solutions", NonRootUID, NonRootGID)
	if err != nil {
		log.Fatalf("Error setting up permissions for /solutions: %v", err)
	}
}
