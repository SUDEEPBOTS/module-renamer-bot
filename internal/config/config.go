package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	BotToken      string
	MongoURI      string
	DatabaseName  string
	OwnerID       int64
	OwnerUsername string
	RepoURL       string
	SupportChat   string
	SudoUsers     map[int64]bool
	FSubChannel   string
	FSubChatID    int64
	GitHubToken   string
	WorkDir       string
	LogChannel    int64
	StartImgURL   string
	StatsImgURL   string
	HelpImgURL    string
}

func loadDotEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			v = strings.Trim(v, `"'`)
			if os.Getenv(k) == "" {
				_ = os.Setenv(k, v)
			}
		}
	}
}

func LoadConfig() *Config {
	loadDotEnv(".env")
	loadDotEnv("../.env")

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
		BotToken:      getEnv("BOT_TOKEN", ""),
		MongoURI:      getEnv("MONGO_URI", ""),
		DatabaseName:  getEnv("DATABASE_NAME", "renamer_bot"),
		OwnerID:       ownerID,
		OwnerUsername: getEnv("OWNER_USERNAME", "SUDEEPBOTS"),
		RepoURL:       getEnv("REPO_URL", "https://github.com/SUDEEPBOTS/module-renamer-bot"),
		SupportChat:   getEnv("SUPPORT_CHAT", "SUDEEPBOTS"),
		SudoUsers:     sudoMap,
		FSubChannel:   getEnv("FSUB_CHANNEL", "SUDEEPBOTS"),
		FSubChatID:    fsubChatID,
		GitHubToken:   getEnv("GITHUB_TOKEN", ""),
		WorkDir:       workDir,
		LogChannel:    logChannel,
		StartImgURL:   getEnv("START_IMG", "https://yukiapi.site/file/zNsU7hDf"),
		StatsImgURL:   getEnv("STATS_IMG", "https://yukiapi.site/file/3G26xQbs"),
		HelpImgURL:    getEnv("HELP_IMG", "https://yukiapi.site/file/at2NPZeU"),
	}
}

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
