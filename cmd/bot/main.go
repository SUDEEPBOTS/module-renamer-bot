package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/SUDEEPBOTS/module-renamer-bot/internal/bot"
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/config"
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/database"
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/logger"
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/renamer"
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/server"
)

const Banner = `
========================================================================
  SUDEEPBOTS MODULE RENAMER & CODE REFACTORING ENGINE
  Copyright (C) 2026 SUDEEPBOTS <https://github.com/SUDEEPBOTS>
  Licensed under the MIT License

  Star the repository on GitHub if you find this project useful:
  https://github.com/SUDEEPBOTS/module-renamer-bot
========================================================================
`

func main() {
	fmt.Print(Banner)

	healthSrv := server.NewHealthServer()
	healthSrv.Start()
	defer healthSrv.Stop()

	cliDir := flag.String("dir", "", "Directory path to rebrand in CLI mode")
	cliOld := flag.String("old", "", "Old name/module to search for")
	cliNew := flag.String("new", "", "New name/module to replace with")
	flag.Parse()

	if *cliDir != "" && *cliOld != "" && *cliNew != "" {
		runStandaloneCLI(*cliDir, *cliOld, *cliNew)
		return
	}

	cfg := config.LoadConfig()

	db, err := database.NewDatabase(cfg.MongoURI, cfg.DatabaseName)
	if err != nil {
		logger.LogError("DATABASE", fmt.Errorf("MongoDB connection failed: %v. Using In-Memory fallback", err))
		db, _ = database.NewDatabase("", "")
	}
	defer db.Close()

	if cfg.MongoURI != "" {
		logger.LogInfo("DATABASE", "MongoDB connected successfully")
	} else {
		logger.LogInfo("DATABASE", "In-Memory storage engine initialized (MONGO_URI not provided)")
	}

	logger.LogInfo("SERVICE", "Service state: OPERATIONAL")
	logger.LogInfo("PROJECT", "Repository: https://github.com/SUDEEPBOTS/module-renamer-bot (Please star on GitHub)")

	if cfg.BotToken == "" {
		fmt.Println("[NOTICE] BOT_TOKEN is not configured in environment or .env file.")
		fmt.Println("[NOTICE] Health check endpoint operational on configured PORT for uptime monitoring.")
		fmt.Println("[NOTICE] Set BOT_TOKEN to activate Telegram Bot mode.")
		fmt.Println("[NOTICE] Waiting for configuration or OS interrupt...")

		waitForShutdown()
		return
	}

	telegramBot, err := bot.New(cfg, db)
	if err != nil {
		logger.LogError("STARTUP", fmt.Errorf("Telegram Bot initialization failed: %v", err))
		os.Exit(1)
	}

	logger.LogInfo("STARTUP", "Module Renamer Bot initialized and polling updates")

	go telegramBot.Start()

	waitForShutdown()
	logger.LogInfo("SHUTDOWN", "Application shutting down gracefully")
}

func runStandaloneCLI(targetDir, oldName, newName string) {
	fmt.Printf("[CLI] Scanning directory: %s\n", targetDir)
	fmt.Printf("[CLI] Replacing: %s -> %s\n", oldName, newName)

	engine := renamer.NewEngine()
	report, err := engine.Execute(renamer.RenameOptions{
		TargetDir:    targetDir,
		OldName:      oldName,
		NewName:      newName,
		IncludeFonts: true,
	})

	if err != nil {
		fmt.Printf("[ERROR] Renaming failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("[CLI] Renaming completed successfully.")
	fmt.Printf("• Files Scanned:       %d\n", report.FilesScanned)
	fmt.Printf("• Files Modified:      %d\n", report.FilesModified)
	fmt.Printf("• Occurrences Replaced:%d\n", report.ReplacementsCount)
	fmt.Printf("• Folders Renamed:     %d\n", report.DirectoriesRenamed)
	fmt.Printf("• Files Renamed:       %d\n", report.FilesRenamed)
}

func waitForShutdown() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
}
