package main

import (
	"fmt"
	"log"
)

func main() {
	// 1. Load Config
	cfg, err := LoadConfig("config.toml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. Initialize DB
	db, err := InitDB("stats.db")
	if err != nil {
		log.Fatalf("Failed to initialize DB: %v", err)
	}

	// 3. Start Monitor
	monitor := NewMonitor(db, cfg)
	monitor.Start()

	// 4. Start Web Server
	web := NewWebServer(db, cfg)
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Starting web server on %s", addr)
	if err := web.Start(addr); err != nil {
		log.Fatalf("Failed to start web server: %v", err)
	}
}
