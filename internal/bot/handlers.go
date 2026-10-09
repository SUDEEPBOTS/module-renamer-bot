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
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/logger"
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/renamer"
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/vcs"
)

type SessionState struct {
	ID         string
	UserID     int64
	Username   string
	Step       string
	RepoURL    string
	LocalDir   string
	OldName    string
	NewName    string
	ZipPath    string
	TargetRepo string
	Token      string
}

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

func (b *Bot) HandleMessage(msg *tgbotapi.Message) {
	if msg == nil || msg.From == nil {
		return
	}

	userID := msg.From.ID

	if b.db.IsBanned(userID) {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "<blockquote>🚫 <b>Access Prohibited:</b> You have been blacklisted from this bot.</blockquote>")
		reply.ParseMode = "HTML"
		_, _ = b.api.Send(reply)
		return
	}

	_ = b.db.AddUser(userID, msg.From.UserName, msg.From.FirstName)

	joined, _ := b.CheckFSub(userID)
	if !joined && !b.cfg.IsSudo(userID) {
		b.SendFSubPrompt(msg.Chat.ID)
		return
	}

	if msg.IsCommand() {
		switch msg.Command() {
		case "start":
			b.handleStart(msg)
			return
		case "help":
			b.sendHelpPage(msg.Chat.ID, 0, 1)
			return
		case "rename":
			b.promptRename(msg.Chat.ID, userID, msg.From.UserName)
			return
		case "cancel":
			b.sm.Clear(userID)
			reply := tgbotapi.NewMessage(msg.Chat.ID, "<blockquote>✅ <b>Active refactoring task cancelled and temporary workspace deleted.</b></blockquote>")
			reply.ParseMode = "HTML"
			_, _ = b.api.Send(reply)
			return
		case "admin", "panel", "stats", "gban", "ungban", "broadcast", "bcast", "users", "log", "logs":
			b.HandleAdminCommand(msg)
			return
		}
	}

	if msg.Document != nil {
		b.handleZipUpload(msg)
		return
	}

	session := b.sm.Get(userID)
	if session != nil {
		b.handleConversationStep(msg, session)
		return
	}

	text := strings.TrimSpace(msg.Text)
	if strings.HasPrefix(text, "http://") || strings.HasPrefix(text, "https://") {
		b.handleRepoLink(msg.Chat.ID, userID, msg.From.UserName, text)
		return
	}

	b.handleStart(msg)
}

func (b *Bot) handleStart(msg *tgbotapi.Message) {
	logger.LogUserStart(msg.From.ID, msg.From.UserName, msg.From.FirstName)

	isSudo := b.cfg.IsSudo(msg.From.ID)

	greeting := fmt.Sprintf(
		"<blockquote>⚡ <b>%s</b></blockquote>\n\n"+
			"<blockquote>👋 <b>Hello %s!</b>\n"+
			"Welcome to the enterprise codebase refactoring & rebranding automation engine built in <b>Golang</b>.</blockquote>\n\n"+
			"<blockquote expandable><b>Engine Specifications:</b>\n"+
			"• 🔄 <b>Universal Scanning:</b> Python, Go, JS, TS, Rust, C++, Java, configs\n"+
			"• 🔡 <b>Unicode Font Bypasser:</b> Decodes & renames stylized fonts (e.g. ʏᴜᴋᴋɪ, 𝐘𝐮𝐤𝐤𝐢)\n"+
			"• 📁 <b>Bottom-Up Restructuring:</b> Deepest-level file and directory path renames\n"+
			"• 📦 <b>Flexible Export:</b> Direct .ZIP document or automated GitHub push!</blockquote>\n\n"+
			"<blockquote>👇 <i>Paste a public GitHub link or send a .zip archive to begin!</i></blockquote>",
		renamer.ToBoldSerif("SUDEEPBOTS Module Renamer"),
		msg.From.FirstName,
	)

	reply := tgbotapi.NewMessage(msg.Chat.ID, greeting)
	reply.ParseMode = "HTML"
	reply.ReplyMarkup = MakeStartKeyboard(isSudo, b.cfg.OwnerUsername, b.cfg.RepoURL, b.cfg.SupportChat, b.cfg.FSubChannel)
	_, _ = b.api.Send(reply)
}

func (b *Bot) sendHelpPage(chatID int64, messageID int, page int) {
	var body string
	title := renamer.ToBoldSerif(fmt.Sprintf("Help & Documentation [Page %d/3]", page))

	switch page {
	case 1:
		body = fmt.Sprintf(
			"<blockquote>📖 <b>%s</b></blockquote>\n\n"+
				"<blockquote><b>1. Getting Started</b>\n"+
				"To rebrand an entire project:\n"+
				"• Send a public GitHub URL (e.g. <code>https://github.com/owner/repo</code>)\n"+
				"• OR upload a <code>.zip</code> source archive directly to this chat.\n"+
				"• Enter the exact <b>Old Module Name</b> (e.g. <code>Yukki</code>).\n"+
				"• Enter your desired <b>New Module Name</b> (e.g. <code>Pulse</code>).</blockquote>\n\n"+
				"<blockquote expandable><b>Supported Language Syntax:</b>\n"+
				"Python (.py), Golang (.go), JavaScript/TypeScript (.js, .ts), Rust (.rs), C/C++ (.c, .cpp, .h), Java (.java), Shell (.sh), Markdown (.md), JSON, YAML, Dockerfile, Makefile, and Environment (.env) files.</blockquote>",
			title,
		)
	case 2:
		body = fmt.Sprintf(
			"<blockquote>🔡 <b>%s</b></blockquote>\n\n"+
				"<blockquote><b>2. Unicode & Fancy Font Recognition</b>\n"+
				"Many repositories use aesthetic fonts in their README or code (e.g. ʏᴜᴋᴋɪ, 𝐘𝐮𝐤𝐤𝐢, 𝒀𝒖𝒌𝒌𝒊, 𝐒υᴘᴘσꝛᴛ).\n"+
				"Our engine maps all mathematical, small-cap, and stylized homoglyphs back to standard characters during matching.</blockquote>\n\n"+
				"<blockquote expandable><b>Delivery Options:</b>\n"+
				"• <b>📦 Export as ZIP:</b> Instantly compresses the cleaned codebase and uploads it as a Telegram document.\n"+
				"• <b>🚀 Push to GitHub:</b> Connects to your GitHub account and pushes directly to a fresh or existing repository!</blockquote>",
			title,
		)
	case 3:
		body = fmt.Sprintf(
			"<blockquote>👑 <b>%s</b></blockquote>\n\n"+
				"<blockquote><b>3. Sudo & Cloud Operations</b>\n"+
				"Administrators have access to real-time telemetry and management tools:</blockquote>\n\n"+
				"<blockquote expandable><b>Sudo Commands:</b>\n"+
				"• <code>/stats</code> — Real-time memory, goroutines, and renames\n"+
				"• <code>/log on</code> or <code>/log off</code> — Toggle console & channel logging\n"+
				"• <code>/broadcast &lt;text&gt;</code> — Send global broadcast to all users\n"+
				"• <code>/gban &lt;user_id&gt; [reason]</code> — Blacklist spam user\n"+
				"• <code>/ungban &lt;user_id&gt;</code> — Remove user blacklist\n"+
				"• <code>/users</code> — Total registered user count</blockquote>",
			title,
		)
	}

	markup := MakeHelpKeyboard(page)

	if messageID != 0 {
		edit := tgbotapi.NewEditMessageText(chatID, messageID, body)
		edit.ParseMode = "HTML"
		edit.ReplyMarkup = &markup
		_, _ = b.api.Send(edit)
	} else {
		msg := tgbotapi.NewMessage(chatID, body)
		msg.ParseMode = "HTML"
		msg.ReplyMarkup = markup
		_, _ = b.api.Send(msg)
	}
}

func (b *Bot) promptRename(chatID int64, userID int64, username string) {
	b.sm.Clear(userID)
	sessionID := fmt.Sprintf("sess_%d_%d", userID, time.Now().Unix())
	b.sm.Set(userID, &SessionState{
		ID:       sessionID,
		UserID:   userID,
		Username: username,
		Step:     "AWAITING_SOURCE",
	})

	text := fmt.Sprintf(
		"<blockquote>🚀 <b>%s</b></blockquote>\n\n"+
			"<blockquote>Please send the <b>GitHub Repository URL</b> (e.g. <code>https://github.com/group-66666/YukkiMusic-Go</code>) "+
			"or upload a <b>.zip</b> project archive to begin.</blockquote>",
		renamer.ToBoldSerif("Start New Rebranding Task"),
	)

	reply := tgbotapi.NewMessage(chatID, text)
	reply.ParseMode = "HTML"
	_, _ = b.api.Send(reply)
}

func (b *Bot) handleRepoLink(chatID int64, userID int64, username, repoURL string) {
	b.sm.Clear(userID)
	sessionID := fmt.Sprintf("sess_%d_%d", userID, time.Now().Unix())
	workDir := filepath.Join(b.cfg.WorkDir, sessionID)

	statusMsg := tgbotapi.NewMessage(chatID, "<blockquote>⏳ <b>Cloning repository from GitHub...</b></blockquote>")
	statusMsg.ParseMode = "HTML"
	sent, _ := b.api.Send(statusMsg)

	err := vcs.CloneRepo(repoURL, workDir)
	if err != nil {
		logger.LogRenameFailure(userID, username, err)
		b.sendAIErrorBlock(chatID, sent.MessageID, "Git Clone Failure", err)
		return
	}

	b.sm.Set(userID, &SessionState{
		ID:       sessionID,
		UserID:   userID,
		Username: username,
		Step:     "AWAITING_OLD_NAME",
		RepoURL:  repoURL,
		LocalDir: workDir,
	})

	text := "<blockquote>✅ <b>Repository cloned successfully!</b></blockquote>\n\n" +
		"<blockquote>🔍 <b>Step 1:</b> Enter the <b>OLD Name/Module</b> you wish to replace.\n" +
		"<i>Example:</i> <code>Yukki</code> or <code>YUKKIMUSIC</code></blockquote>"

	edit := tgbotapi.NewEditMessageText(chatID, sent.MessageID, text)
	edit.ParseMode = "HTML"
	_, _ = b.api.Send(edit)
}

func (b *Bot) handleZipUpload(msg *tgbotapi.Message) {
	userID := msg.From.ID
	username := msg.From.UserName
	doc := msg.Document

	if !strings.HasSuffix(strings.ToLower(doc.FileName), ".zip") {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "<blockquote>⚠️ <i>Please upload a valid .zip archive file!</i></blockquote>")
		reply.ParseMode = "HTML"
		_, _ = b.api.Send(reply)
		return
	}

	b.sm.Clear(userID)
	sessionID := fmt.Sprintf("sess_%d_%d", userID, time.Now().Unix())
	workDir := filepath.Join(b.cfg.WorkDir, sessionID)
	_ = os.MkdirAll(workDir, 0755)

	statusMsg := tgbotapi.NewMessage(msg.Chat.ID, "<blockquote>⏳ <b>Downloading and unpacking ZIP archive...</b></blockquote>")
	statusMsg.ParseMode = "HTML"
	sent, _ := b.api.Send(statusMsg)

	fileConfig := tgbotapi.FileConfig{FileID: doc.FileID}
	file, err := b.api.GetFile(fileConfig)
	if err != nil {
		b.sendAIErrorBlock(msg.Chat.ID, sent.MessageID, "Telegram File Download", err)
		return
	}

	downloadURL := file.Link(b.cfg.BotToken)
	zipDest := filepath.Join(workDir, "uploaded.zip")

	resp, err := http.Get(downloadURL)
	if err != nil {
		b.sendAIErrorBlock(msg.Chat.ID, sent.MessageID, "ZIP Payload Fetch", err)
		return
	}
	defer resp.Body.Close()

	out, err := os.Create(zipDest)
	if err != nil {
		b.sendAIErrorBlock(msg.Chat.ID, sent.MessageID, "Local Disk Write", err)
		return
	}
	_, _ = out.ReadFrom(resp.Body)
	out.Close()

	extractedDir := filepath.Join(workDir, "src")
	_ = os.MkdirAll(extractedDir, 0755)

	err = vcs.UnzipArchive(zipDest, extractedDir)
	if err != nil {
		b.sendAIErrorBlock(msg.Chat.ID, sent.MessageID, "ZIP Archive Extraction", err)
		return
	}

	b.sm.Set(userID, &SessionState{
		ID:       sessionID,
		UserID:   userID,
		Username: username,
		Step:     "AWAITING_OLD_NAME",
		LocalDir: extractedDir,
	})

	text := "<blockquote>✅ <b>Archive extracted successfully!</b></blockquote>\n\n" +
		"<blockquote>🔍 <b>Step 1:</b> Enter the <b>OLD Name/Module</b> you wish to replace.\n" +
		"<i>Example:</i> <code>Yukki</code></blockquote>"

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
			b.handleRepoLink(chatID, msg.From.ID, msg.From.UserName, text)
		} else {
			reply := tgbotapi.NewMessage(chatID, "<blockquote>⚠️ <i>Please provide a valid GitHub repository link.</i></blockquote>")
			reply.ParseMode = "HTML"
			_, _ = b.api.Send(reply)
		}

	case "AWAITING_OLD_NAME":
		session.OldName = text
		session.Step = "AWAITING_NEW_NAME"
		replyText := fmt.Sprintf(
			"<blockquote>✅ <b>Target Recorded:</b> <code>%s</code></blockquote>\n\n"+
				"<blockquote>✨ <b>Step 2:</b> Enter the <b>NEW Name/Module</b> to replace with.\n"+
				"<i>Example:</i> <code>Pulse</code></blockquote>",
			text,
		)
		reply := tgbotapi.NewMessage(chatID, replyText)
		reply.ParseMode = "HTML"
		_, _ = b.api.Send(reply)

	case "AWAITING_NEW_NAME":
		session.NewName = text
		session.Step = "AWAITING_DELIVERY_CHOICE"

		logger.LogRenameInitiated(session.UserID, session.Username, session.RepoURL, session.OldName, session.NewName)

		statusMsg := tgbotapi.NewMessage(chatID, "<blockquote>⚙️ <b>Refactoring codebase and evaluating Unicode font lookalikes...</b></blockquote>")
		statusMsg.ParseMode = "HTML"
		sent, _ := b.api.Send(statusMsg)

		engine := renamer.NewEngine()
		report, err := engine.Execute(renamer.RenameOptions{
			TargetDir:    session.LocalDir,
			OldName:      session.OldName,
			NewName:      session.NewName,
			IncludeFonts: true,
		})

		if err != nil {
			logger.LogRenameFailure(session.UserID, session.Username, err)
			b.sendAIErrorBlock(chatID, sent.MessageID, "Codebase Refactoring Engine", err)
			return
		}

		_ = b.db.IncrementRenames()
		logger.LogRenameSuccess(session.UserID, session.Username, session.OldName, session.NewName, report.FilesScanned, report.FilesModified, report.ReplacementsCount, report.DirectoriesRenamed+report.FilesRenamed)

		resultText := fmt.Sprintf(
			"<blockquote>🎉 <b>%s</b></blockquote>\n\n"+
				"<blockquote><b>Rebranding Telemetry:</b>\n"+
				"• <b>Target Old:</b> <code>%s</code>\n"+
				"• <b>Target New:</b> <code>%s</code>\n"+
				"• <b>Files Scanned:</b> <code>%d</code>\n"+
				"• <b>Files Modified:</b> <code>%d</code>\n"+
				"• <b>Occurrences Replaced:</b> <code>%d</code>\n"+
				"• <b>Paths/Directories Renamed:</b> <code>%d</code></blockquote>\n\n"+
				"<blockquote>📦 <b>Select your delivery method below:</b></blockquote>",
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
			b.executeGitHubPush(chatID, session, b.cfg.GitHubToken)
			return
		}

		replyText := "<blockquote>🔑 <b>Please provide your GitHub Personal Access Token (PAT):</b>\n" +
			"<i>(Requires 'repo' scope permissions to push changes)</i></blockquote>"
		reply := tgbotapi.NewMessage(chatID, replyText)
		reply.ParseMode = "HTML"
		_, _ = b.api.Send(reply)

	case "AWAITING_GITHUB_PUSH_TOKEN":
		token := text
		b.executeGitHubPush(chatID, session, token)
	}
}

func (b *Bot) executeGitHubPush(chatID int64, session *SessionState, token string) {
	statusMsg := tgbotapi.NewMessage(chatID, "<blockquote>🚀 <b>Initializing git branch and pushing to GitHub...</b></blockquote>")
	statusMsg.ParseMode = "HTML"
	sent, _ := b.api.Send(statusMsg)

	commitMsg := fmt.Sprintf("feat: rebrand %s to %s via SUDEEPBOTS Module Renamer", session.OldName, session.NewName)
	err := vcs.PushToGitHub(session.LocalDir, session.TargetRepo, token, commitMsg)

	if err != nil {
		logger.LogRenameFailure(session.UserID, session.Username, err)
		b.sendAIErrorBlock(chatID, sent.MessageID, "GitHub Remote Push", err)
		return
	}

	successText := fmt.Sprintf(
		"<blockquote>🚀 <b>%s</b></blockquote>\n\n"+
			"<blockquote>Your rebranded repository has been pushed cleanly to GitHub!\n\n"+
			"🔗 <b>Repository:</b> <a href=\"%s\">%s</a></blockquote>",
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

func (b *Bot) sendAIErrorBlock(chatID int64, messageID int, stage string, err error) {
	errText := fmt.Sprintf(
		"<blockquote>❌ <b>Execution Failure: %s</b>\n"+
			"An unexpected error occurred during execution. Copy the trace below and provide it to an AI assistant or developer to resolve:</blockquote>\n\n"+
			"<pre><code>[ERROR TRACE] Stage: %s\nTime: %s\nDetails: %v</code></pre>",
		stage, stage, time.Now().Format(time.RFC3339), err,
	)

	if messageID != 0 {
		edit := tgbotapi.NewEditMessageText(chatID, messageID, errText)
		edit.ParseMode = "HTML"
		_, _ = b.api.Send(edit)
	} else {
		msg := tgbotapi.NewMessage(chatID, errText)
		msg.ParseMode = "HTML"
		_, _ = b.api.Send(msg)
	}
}
