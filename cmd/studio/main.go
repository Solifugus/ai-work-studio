// Package main provides the main entry point for AI Work Studio GUI application.
//
// This is the graphical user interface version of AI Work Studio, built with
// Fyne framework for cross-platform native GUI experience.
//
// Usage:
//   ./studio [options]
//
// Options:
//   -config string    Configuration file path
//   -data string      Data directory path (overrides config)
//   -verbose          Enable verbose logging
//   -version          Show version information
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/yourusername/ai-work-studio/internal/config"
	"github.com/yourusername/ai-work-studio/pkg/ui"
)

// Version information
const (
	Version   = "1.0.0"
	BuildDate = "2026-01-18"
	AppName   = "AI Work Studio"
)

func main() {
	// Parse command line arguments
	var (
		configPath = flag.String("config", "", "Configuration file path")
		dataDir    = flag.String("data", "", "Data directory path (overrides config)")
		verbose    = flag.Bool("verbose", false, "Enable verbose logging")
		version    = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()

	// Show version information if requested
	if *version {
		fmt.Printf("%s v%s (built %s)\n", AppName, Version, BuildDate)
		fmt.Println("A goal-directed autonomous agent system with native GUI")
		os.Exit(0)
	}

	// Setup logging
	if *verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		log.Printf("Starting %s v%s", AppName, Version)
	} else {
		log.SetFlags(log.LstdFlags)
	}

	// Get default config path if not specified
	if *configPath == "" {
		var err error
		*configPath, err = config.GetConfigPath()
		if err != nil {
			log.Fatalf("Error getting config path: %v", err)
		}
	}

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	// Override data directory if specified
	if *dataDir != "" {
		cfg.DataDir = *dataDir
		log.Printf("Using data directory: %s", cfg.DataDir)
	}

	// Override verbose setting if specified
	if *verbose {
		cfg.Preferences.VerboseOutput = true
	}

	if cfg.Preferences.VerboseOutput {
		log.Printf("Configuration loaded from: %s", *configPath)
		log.Printf("Data directory: %s", cfg.DataDir)
		log.Printf("User ID: %s", cfg.Session.UserID)
	}

	// Ensure data directory exists
	if err := cfg.EnsureDataDir(); err != nil {
		log.Fatalf("Error setting up data directory: %v", err)
	}

	// Create and run the application
	app, err := ui.NewApp(cfg, *configPath)
	if err != nil {
		log.Fatalf("Error creating application: %v", err)
	}

	log.Printf("%s started successfully", AppName)

	// Run the application (blocks until window is closed)
	if err := app.Run(); err != nil {
		log.Fatalf("Error running application: %v", err)
	}

	log.Printf("%s exited cleanly", AppName)
}