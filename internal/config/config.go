package config

import (
	"os"
	"strconv"
	"strings"
)

// Config holds the application configuration parameters.
type Config struct {
	BotToken     string
	MongoURI     string
	DatabaseName string
	OwnerID      int64
	SudoUsers    map[int64]bool
	FSubChannel  string
	FSubChatID   int64
	GitHubToken  string
	WorkDir      string
	LogChannel   int64
}

// LoadConfig loads configuration from environment variables with safe defaults.
func LoadConfig() *Config {
	ownerID, _ := strconv.ParseInt(getEnv("OWNER_ID", "0"), 10, 64)
	fsubChatID, _ := strconv.ParseInt(getEnv("FSUB_CHAT_ID", "0"), 10, 64)
	logChannel, _ := strconv.ParseInt(getEnv("LOG_CHANNEL", "0"), 10, 64)

	sudoMap := make(map[int64]bool)
	if ownerID != 0 {
		sudoMap[ownerID] = true
	}

	sudoStr := getEnv("SUDO_USERS", "")
	if sudoStr != "" {
		for _, part := range strings.Split(sudoStr, ",") {
			part = strings.TrimSpace(part)
			if id, err := strconv.ParseInt(part, 10, 64); err == nil && id != 0 {
				sudoMap[id] = true
			}
		}
	}

	workDir := getEnv("WORK_DIR", "/tmp/renamer_work")
	_ = os.MkdirAll(workDir, 0755)

	return &Config{
		BotToken:     getEnv("BOT_TOKEN", ""),
		MongoURI:     getEnv("MONGO_URI", ""),
		DatabaseName: getEnv("DATABASE_NAME", "renamer_bot"),
		OwnerID:      ownerID,
		SudoUsers:    sudoMap,
		FSubChannel:  getEnv("FSUB_CHANNEL", "SUDEEPBOTS"),
		FSubChatID:   fsubChatID,
		GitHubToken:  getEnv("GITHUB_TOKEN", ""),
		WorkDir:      workDir,
		LogChannel:   logChannel,
	}
}

// IsSudo returns true if the user ID is the owner or in sudo users.
func (c *Config) IsSudo(userID int64) bool {
	if c.OwnerID != 0 && userID == c.OwnerID {
		return true
	}
	return c.SudoUsers[userID]
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return strings.TrimSpace(val)
	}
	return defaultVal
}
