package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/SUDEEPBOTS/module-renamer-bot/internal/bot"
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/config"
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/database"
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/renamer"
)

const Banner = `
========================================================================
  ⚡ SUDEEPBOTS MODULE RENAMER & CODE REFACTORING ENGINE (GOLANG) ⚡
========================================================================
`

func main() {
	fmt.Print(Banner)

	// Command-line flag overrides for standalone local execution
	cliDir := flag.String("dir", "", "Target directory to rebrand in CLI mode")
	cliOld := flag.String("old", "", "Old name/module to search for")
	cliNew := flag.String("new", "", "New name/module to replace with")
	flag.Parse()

	// If CLI flags provided, run standalone local renamer
	if *cliDir != "" && *cliOld != "" && *cliNew != "" {
		runStandaloneCLI(*cliDir, *cliOld, *cliNew)
		return
	}

	cfg := config.LoadConfig()

	// Initialize Database (MongoDB with automatic In-Memory fallback)
	db, err := database.NewDatabase(cfg.MongoURI, cfg.DatabaseName)
	if err != nil {
		log.Printf("⚠️ MongoDB connection failed: %v. Falling back to In-Memory DB.", err)
		db, _ = database.NewDatabase("", "")
	}
	defer db.Close()

	if cfg.MongoURI != "" {
		log.Println("🍃 Connected to MongoDB successfully!")
	} else {
		log.Println("💾 Running with In-Memory storage engine (MONGO_URI not provided).")
	}

	// Check if Telegram Bot Token is configured
	if cfg.BotToken == "" {
		fmt.Println("⚠️  Notice: BOT_TOKEN is not configured in environment or .env file.")
		fmt.Println("👉 To start the Telegram Bot, set BOT_TOKEN in your environment:")
		fmt.Println("   export BOT_TOKEN=\"your_bot_token_here\"")
		fmt.Println("   export MONGO_URI=\"mongodb+srv://...\" (optional)")
		fmt.Println("   export OWNER_ID=\"123456789\"")
		fmt.Println("")
		fmt.Println("💡 You can also use this tool as a standalone local CLI renamer:")
		fmt.Println("   go run cmd/bot/main.go -dir /path/to/project -old Yukki -new Pulse")
		fmt.Println("")
		fmt.Println("Waiting for configuration or OS interrupt...")

		// Wait for OS signal to keep container alive if running in docker
		waitForShutdown()
		return
	}

	// Initialize Telegram Bot
	telegramBot, err := bot.New(cfg, db)
	if err != nil {
		log.Fatalf("❌ Failed to start Telegram Bot: %v", err)
	}

	log.Println("🚀 Module Renamer Bot is active and polling updates...")

	go telegramBot.Start()

	waitForShutdown()
	log.Println("👋 Shutting down gracefully...")
}

func runStandaloneCLI(targetDir, oldName, newName string) {
	fmt.Printf("🔍 Scanning directory: %s\n", targetDir)
	fmt.Printf("🔄 Replacing '%s' -> '%s' (including stylized fonts)...\n", oldName, newName)

	engine := renamer.NewEngine()
	report, err := engine.Execute(renamer.RenameOptions{
		TargetDir:    targetDir,
		OldName:      oldName,
		NewName:      newName,
		IncludeFonts: true,
	})

	if err != nil {
		fmt.Printf("❌ Renaming failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("🎉 Renaming completed successfully!")
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
