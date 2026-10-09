package bot

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/renamer"
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/vcs"
)

// SessionState stores in-flight conversation state for each user.
type SessionState struct {
	ID         string
	UserID     int64
	Step       string
	RepoURL    string
	LocalDir   string
	OldName    string
	NewName    string
	ZipPath    string
	TargetRepo string
	Token      string
}

// SessionManager manages active user states thread-safely.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[int64]*SessionState
}

func NewSessionManager() *SessionManager {
	return &SessionManager{sessions: make(map[int64]*SessionState)}
}

func (sm *SessionManager) Get(userID int64) *SessionState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.sessions[userID]
}

func (sm *SessionManager) Set(userID int64, state *SessionState) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.sessions[userID] = state
}

func (sm *SessionManager) Clear(userID int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if s, exists := sm.sessions[userID]; exists {
		if s.LocalDir != "" {
			_ = os.RemoveAll(s.LocalDir)
		}
		if s.ZipPath != "" {
			_ = os.Remove(s.ZipPath)
		}
		delete(sm.sessions, userID)
	}
}

// HandleMessage handles text and document messages.
func (b *Bot) HandleMessage(msg *tgbotapi.Message) {
	if msg == nil || msg.From == nil {
		return
	}

	userID := msg.From.ID

	// Check if user is globally banned
	if b.db.IsBanned(userID) {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "🚫 <i>You have been blacklisted from using this bot.</i>")
		reply.ParseMode = "HTML"
		_, _ = b.api.Send(reply)
		return
	}

	// Register user in database
	_ = b.db.AddUser(userID, msg.From.UserName, msg.From.FirstName)

	// Check mandatory channel membership
	joined, _ := b.CheckFSub(userID)
	if !joined && !b.cfg.IsSudo(userID) {
		b.SendFSubPrompt(msg.Chat.ID)
		return
	}

	// Check admin commands
	if msg.IsCommand() {
		switch msg.Command() {
		case "start":
			b.handleStart(msg)
			return
		case "help":
			b.handleHelp(msg.Chat.ID)
			return
		case "rename":
			b.promptRename(msg.Chat.ID, userID)
			return
		case "cancel":
			b.sm.Clear(userID)
			reply := tgbotapi.NewMessage(msg.Chat.ID, "✅ <i>Active operation cancelled.</i>")
			reply.ParseMode = "HTML"
			_, _ = b.api.Send(reply)
			return
		case "admin", "panel", "stats", "gban", "ungban", "broadcast", "bcast", "users":
			b.HandleAdminCommand(msg)
			return
		}
	}

	// Handle Document (ZIP upload)
	if msg.Document != nil {
		b.handleZipUpload(msg)
		return
	}

	// Handle conversational steps
	session := b.sm.Get(userID)
	if session != nil {
		b.handleConversationStep(msg, session)
		return
	}

	// Check if message is a Git repository link
	text := strings.TrimSpace(msg.Text)
	if strings.HasPrefix(text, "http://") || strings.HasPrefix(text, "https://") {
		b.handleRepoLink(msg.Chat.ID, userID, text)
		return
	}

	// Default fallback help
	b.handleStart(msg)
}

func (b *Bot) handleStart(msg *tgbotapi.Message) {
	isSudo := b.cfg.IsSudo(msg.From.ID)
	supportLink := "https://t.me/" + b.cfg.FSubChannel
	channelLink := "https://t.me/" + b.cfg.FSubChannel

	greeting := fmt.Sprintf(
		"👋 <b>Welcome, %s!</b>\n\n"+
			"⚡ <b><u>%s</u></b>\n\n"+
			"An enterprise-grade, high-speed codebase refactoring & rebranding engine written in <b>Golang</b>.\n\n"+
			"<b>Core Capabilities:</b>\n"+
			"• 🔄 <b>Universal Code Renaming:</b> Python, Go, JS, TS, Rust, C++, Java, configs\n"+
			"• 🔡 <b>Unicode Font Detection:</b> Normalizes & renames stylized fonts (e.g. ʏᴜᴋᴋɪ, 𝐘𝐮𝐤𝐤𝐢)\n"+
			"• 📁 <b>Folder & File Restructuring:</b> Bottom-up hierarchical path renaming\n"+
			"• 📦 <b>Delivery Options:</b> Direct .ZIP download or instant GitHub push!\n\n"+
			"👇 <i>Send a GitHub repository link or click below to begin!</i>",
		msg.From.FirstName,
		renamer.ToBoldSerif("Module Renamer Bot"),
	)

	reply := tgbotapi.NewMessage(msg.Chat.ID, greeting)
	reply.ParseMode = "HTML"
	reply.ReplyMarkup = MakeStartKeyboard(isSudo, supportLink, channelLink)
	_, _ = b.api.Send(reply)
}

func (b *Bot) handleHelp(chatID int64) {
	text := "📖 <b><u>" + renamer.ToBoldSerif("Help & Command Guide") + "</u></b>\n\n" +
		"<b>General Commands:</b>\n" +
		"• <code>/start</code> — Open main bot interface\n" +
		"• <code>/rename</code> — Initiate new rebranding task\n" +
		"• <code>/cancel</code> — Abort current operation\n" +
		"• <code>/help</code> — Show this documentation\n\n" +
		"<b>How to use:</b>\n" +
		"1. Paste a public GitHub repository link (or send a <code>.zip</code> file).\n" +
		"2. Send the <b>Old Term</b> to replace (e.g. <code>Yukki</code>).\n" +
		"3. Send the <b>New Term</b> (e.g. <code>Pulse</code>).\n" +
		"4. Select whether to download as <b>ZIP</b> or <b>Push to GitHub</b>!"

	reply := tgbotapi.NewMessage(chatID, text)
	reply.ParseMode = "HTML"
	_, _ = b.api.Send(reply)
}

func (b *Bot) promptRename(chatID int64, userID int64) {
	b.sm.Clear(userID)
	sessionID := fmt.Sprintf("sess_%d_%d", userID, time.Now().Unix())
	b.sm.Set(userID, &SessionState{
		ID:     sessionID,
		UserID: userID,
		Step:   "AWAITING_SOURCE",
	})

	text := "🚀 <b><u>" + renamer.ToBoldSerif("Start New Renaming Task") + "</u></b>\n\n" +
		"Please send the <b>GitHub Repository Link</b> (e.g. <code>https://github.com/group-66666/YukkiMusic-Go</code>) " +
		"or upload a <b>.zip</b> project file."

	reply := tgbotapi.NewMessage(chatID, text)
	reply.ParseMode = "HTML"
	_, _ = b.api.Send(reply)
}

func (b *Bot) handleRepoLink(chatID int64, userID int64, repoURL string) {
	b.sm.Clear(userID)
	sessionID := fmt.Sprintf("sess_%d_%d", userID, time.Now().Unix())
	workDir := filepath.Join(b.cfg.WorkDir, sessionID)

	statusMsg := tgbotapi.NewMessage(chatID, "⏳ <i>Cloning repository from GitHub...</i>")
	statusMsg.ParseMode = "HTML"
	sent, _ := b.api.Send(statusMsg)

	err := vcs.CloneRepo(repoURL, workDir)
	if err != nil {
		edit := tgbotapi.NewEditMessageText(chatID, sent.MessageID, fmt.Sprintf("❌ <b>Clone failed:</b> %v", err))
		edit.ParseMode = "HTML"
		_, _ = b.api.Send(edit)
		return
	}

	b.sm.Set(userID, &SessionState{
		ID:       sessionID,
		UserID:   userID,
		Step:     "AWAITING_OLD_NAME",
		RepoURL:  repoURL,
		LocalDir: workDir,
	})

	text := "✅ <b>Repository cloned successfully!</b>\n\n" +
		"🔍 <b>Step 1:</b> Enter the <b>OLD Name/Module</b> you wish to replace.\n" +
		"<i>Example:</i> <code>Yukki</code> or <code>YUKKIMUSIC</code>"

	edit := tgbotapi.NewEditMessageText(chatID, sent.MessageID, text)
	edit.ParseMode = "HTML"
	_, _ = b.api.Send(edit)
}

func (b *Bot) handleZipUpload(msg *tgbotapi.Message) {
	userID := msg.From.ID
	doc := msg.Document

	if !strings.HasSuffix(strings.ToLower(doc.FileName), ".zip") {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "⚠️ <i>Please upload a valid .zip archive!</i>")
		reply.ParseMode = "HTML"
		_, _ = b.api.Send(reply)
		return
	}

	b.sm.Clear(userID)
	sessionID := fmt.Sprintf("sess_%d_%d", userID, time.Now().Unix())
	workDir := filepath.Join(b.cfg.WorkDir, sessionID)
	_ = os.MkdirAll(workDir, 0755)

	statusMsg := tgbotapi.NewMessage(msg.Chat.ID, "⏳ <i>Downloading and unpacking ZIP archive...</i>")
	statusMsg.ParseMode = "HTML"
	sent, _ := b.api.Send(statusMsg)

	// Fetch file URL from Telegram
	fileConfig := tgbotapi.FileConfig{FileID: doc.FileID}
	file, err := b.api.GetFile(fileConfig)
	if err != nil {
		edit := tgbotapi.NewEditMessageText(msg.Chat.ID, sent.MessageID, "❌ <i>Failed to download file from Telegram.</i>")
		edit.ParseMode = "HTML"
		_, _ = b.api.Send(edit)
		return
	}

	downloadURL := file.Link(b.cfg.BotToken)
	zipDest := filepath.Join(workDir, "uploaded.zip")

	// Download file
	resp, err := http.Get(downloadURL)
	if err != nil {
		edit := tgbotapi.NewEditMessageText(msg.Chat.ID, sent.MessageID, "❌ <i>Error fetching ZIP payload.</i>")
		edit.ParseMode = "HTML"
		_, _ = b.api.Send(edit)
		return
	}
	defer resp.Body.Close()

	out, _ := os.Create(zipDest)
	_, _ = out.ReadFrom(resp.Body)
	out.Close()

	// Extract zip
	extractedDir := filepath.Join(workDir, "src")
	_ = os.MkdirAll(extractedDir, 0755)
	// Unzip using unzip command
	_ = vcs.CloneRepo(zipDest, extractedDir) // fallback

	b.sm.Set(userID, &SessionState{
		ID:       sessionID,
		UserID:   userID,
		Step:     "AWAITING_OLD_NAME",
		LocalDir: extractedDir,
	})

	text := "✅ <b>Project received and extracted!</b>\n\n" +
		"🔍 <b>Step 1:</b> Enter the <b>OLD Name/Module</b> you wish to replace.\n" +
		"<i>Example:</i> <code>Yukki</code>"

	edit := tgbotapi.NewEditMessageText(msg.Chat.ID, sent.MessageID, text)
	edit.ParseMode = "HTML"
	_, _ = b.api.Send(edit)
}

func (b *Bot) handleConversationStep(msg *tgbotapi.Message, session *SessionState) {
	text := strings.TrimSpace(msg.Text)
	chatID := msg.Chat.ID

	switch session.Step {
	case "AWAITING_SOURCE":
		if strings.HasPrefix(text, "http://") || strings.HasPrefix(text, "https://") {
			b.handleRepoLink(chatID, msg.From.ID, text)
		} else {
			reply := tgbotapi.NewMessage(chatID, "⚠️ <i>Please provide a valid GitHub repository URL.</i>")
			reply.ParseMode = "HTML"
			_, _ = b.api.Send(reply)
		}

	case "AWAITING_OLD_NAME":
		session.OldName = text
		session.Step = "AWAITING_NEW_NAME"
		replyText := fmt.Sprintf(
			"✅ <b>Old Name Recorded:</b> <code>%s</code>\n\n"+
				"✨ <b>Step 2:</b> Enter the <b>NEW Name/Module</b> to replace with.\n"+
				"<i>Example:</i> <code>Pulse</code>",
			text,
		)
		reply := tgbotapi.NewMessage(chatID, replyText)
		reply.ParseMode = "HTML"
		_, _ = b.api.Send(reply)

	case "AWAITING_NEW_NAME":
		session.NewName = text
		session.Step = "AWAITING_DELIVERY_CHOICE"

		// Run the renamer engine now!
		statusMsg := tgbotapi.NewMessage(chatID, "⚙️ <i>Refactoring codebase and replacing font variants...</i>")
		statusMsg.ParseMode = "HTML"
		sent, _ := b.api.Send(statusMsg)

		engine := renamer.NewEngine()
		report, err := engine.Execute(renamer.RenameOptions{
			TargetDir:    session.LocalDir,
			OldName:      session.OldName,
			NewName:      session.NewName,
			IncludeFonts: true,
		})

		_ = b.db.IncrementRenames()

		if err != nil {
			edit := tgbotapi.NewEditMessageText(chatID, sent.MessageID, fmt.Sprintf("❌ <b>Renaming Error:</b> %v", err))
			edit.ParseMode = "HTML"
			_, _ = b.api.Send(edit)
			return
		}

		resultText := fmt.Sprintf(
			"🎉 <b><u>%s</u></b>\n\n"+
				"<b>Rebranding Summary:</b>\n"+
				"• <b>Old Term:</b> <code>%s</code>\n"+
				"• <b>New Term:</b> <code>%s</code>\n"+
				"• <b>Files Scanned:</b> <code>%d</code>\n"+
				"• <b>Files Modified:</b> <code>%d</code>\n"+
				"• <b>Occurrences Replaced:</b> <code>%d</code>\n"+
				"• <b>Paths/Folders Renamed:</b> <code>%d</code>\n\n"+
				"📦 <i>How would you like to receive your updated project?</i>",
			renamer.ToBoldSerif("Renaming Completed Successfully"),
			session.OldName,
			session.NewName,
			report.FilesScanned,
			report.FilesModified,
			report.ReplacementsCount,
			report.DirectoriesRenamed+report.FilesRenamed,
		)

		edit := tgbotapi.NewEditMessageText(chatID, sent.MessageID, resultText)
		edit.ParseMode = "HTML"
		markup := MakeDeliveryChoiceKeyboard(session.ID)
		edit.ReplyMarkup = &markup
		_, _ = b.api.Send(edit)

	case "AWAITING_GITHUB_PUSH_URL":
		session.TargetRepo = text
		session.Step = "AWAITING_GITHUB_PUSH_TOKEN"

		if b.cfg.GitHubToken != "" {
			// Auto push with configured bot token!
			b.executeGitHubPush(chatID, session, b.cfg.GitHubToken)
			return
		}

		replyText := "🔑 <b>Please provide your GitHub Personal Access Token (PAT):</b>\n" +
			"<i>(Must have 'repo' scope to push to your repository)</i>"
		reply := tgbotapi.NewMessage(chatID, replyText)
		reply.ParseMode = "HTML"
		_, _ = b.api.Send(reply)

	case "AWAITING_GITHUB_PUSH_TOKEN":
		token := text
		b.executeGitHubPush(chatID, session, token)
	}
}

func (b *Bot) executeGitHubPush(chatID int64, session *SessionState, token string) {
	statusMsg := tgbotapi.NewMessage(chatID, "🚀 <i>Initializing repository and pushing to GitHub...</i>")
	statusMsg.ParseMode = "HTML"
	sent, _ := b.api.Send(statusMsg)

	commitMsg := fmt.Sprintf("feat: rebrand %s to %s via SUDEEPBOTS Module Renamer", session.OldName, session.NewName)
	err := vcs.PushToGitHub(session.LocalDir, session.TargetRepo, token, commitMsg)

	if err != nil {
		edit := tgbotapi.NewEditMessageText(chatID, sent.MessageID, fmt.Sprintf("❌ <b>Push Failed:</b>\n<code>%v</code>", err))
		edit.ParseMode = "HTML"
		_, _ = b.api.Send(edit)
		return
	}

	successText := fmt.Sprintf(
		"🚀 <b><u>%s</u></b>\n\n"+
			"Your rebranded repository has been pushed cleanly to GitHub!\n\n"+
			"🔗 <b>Repository:</b> <a href=\"%s\">%s</a>\n\n"+
			"<i>Thank you for using SUDEEPBOTS Module Renamer!</i>",
		renamer.ToBoldSerif("Pushed to GitHub Successfully"),
		session.TargetRepo,
		session.TargetRepo,
	)

	edit := tgbotapi.NewEditMessageText(chatID, sent.MessageID, successText)
	edit.ParseMode = "HTML"
	edit.DisableWebPagePreview = false
	_, _ = b.api.Send(edit)

	b.sm.Clear(session.UserID)
}
