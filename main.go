package main

import (
	"log"

	"github.com/tsuru/gcp-state-metrics/app"
)

func main() {
	err := app.Start()
	if err != nil {
		log.Fatalf("unable to start server: %v", err)
	}
}
