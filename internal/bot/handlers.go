package bot

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/logger"
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/renamer"
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/vcs"
)

type SessionState struct {
	ID              string
	UserID          int64
	Username        string
	Step            string
	RepoURL         string
	LocalDir        string
	OldName         string
	NewName         string
	ZipPath         string
	TargetRepo      string
	Token           string
	AuthorQuery     string
	DiscoveredLinks []renamer.DiscoveredLink
	SelectedLink    string
	LinkPage        int
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

type reactionItem struct {
	Type  string `json:"type"`
	Emoji string `json:"emoji"`
}

func (b *Bot) ReactToMessage(chatID int64, messageID int, emoji string) {
	reactions := []reactionItem{{Type: "emoji", Emoji: emoji}}
	data, err := json.Marshal(reactions)
	if err != nil {
		return
	}

	params := tgbotapi.Params{
		"chat_id":    strconv.FormatInt(chatID, 10),
		"message_id": strconv.Itoa(messageID),
		"reaction":   string(data),
	}
	_, _ = b.api.MakeRequest("setMessageReaction", params)
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
			b.ReactToMessage(msg.Chat.ID, msg.MessageID, "⚡")
			b.handleStart(msg)
			return
		case "help":
			b.sendHelpPage(msg.Chat.ID, 0, 1)
			return
		case "rename":
			b.promptRename(msg.Chat.ID, userID, msg.From.UserName)
			return
		case "author", "findname", "scan":
			args := strings.TrimSpace(msg.CommandArguments())
			b.promptAuthor(msg.Chat.ID, userID, msg.From.UserName, args)
			return
		case "links", "scanlinks", "urls":
			b.promptLinks(msg.Chat.ID, userID, msg.From.UserName)
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
	var userID int64
	var firstName string
	var userName string

	if msg.From != nil {
		userID = msg.From.ID
		firstName = msg.From.FirstName
		userName = msg.From.UserName
	} else {
		userID = msg.Chat.ID
		firstName = msg.Chat.FirstName
		userName = msg.Chat.UserName
	}

	logger.LogUserStart(userID, userName, firstName)

	isSudo := b.cfg.IsSudo(userID)

	header := renamer.ToSmallCaps("SUDEEPBOTS MODULE RENAMER")
	greeting := fmt.Sprintf(
		"<blockquote>⚡ <b>%s</b></blockquote>\n\n"+
			"<blockquote>👋 <b>ʜᴇʟʟᴏ %s!</b>\n"+
			"ᴡᴇʟᴄᴏᴍᴇ ᴛᴏ ᴛʜᴇ ᴇɴᴛᴇʀᴘʀɪsᴇ ᴄᴏᴅᴇʙᴀsᴇ ʀᴇғᴀᴄᴛᴏʀɪɴɢ & ʀᴇʙʀᴀɴᴅɪɴɢ ᴀᴜᴛᴏᴍᴀᴛɪᴏɴ ᴇɴɢɪɴᴇ.</blockquote>\n\n"+
			"<blockquote expandable><b>❏ sᴜᴘᴘᴏʀᴛᴇᴅ ᴍᴏᴅᴜʟᴇs & ʙᴏᴛs:</b>\n"+
			"❏ ᴍᴜsɪᴄ ʙᴏᴛs\n"+
			"❏ ᴀɪ & ᴄʜᴀᴛ ʙᴏᴛs\n"+
			"❏ ᴠᴏɪᴄᴇ ᴄʜᴀᴛ / ᴠᴄ ʙᴏᴛs\n"+
			"❏ ɢʀᴏᴜᴘ ᴍᴀɴᴀɢᴇᴍᴇɴᴛ ʙᴏᴛs\n"+
			"❏ sᴇᴄᴜʀɪᴛʏ & ᴀɴᴛɪ-ᴅᴅᴏs ʙᴏᴛs\n"+
			"❏ ᴠᴇʀɪғɪᴄᴀᴛɪᴏɴ & ᴀᴜᴛʜᴇɴᴛɪᴄᴀᴛɪᴏɴ ʙᴏᴛs\n"+
			"❏ ᴛɪᴄᴋᴇᴛ & sᴜᴘᴘᴏʀᴛ ʙᴏᴛs\n"+
			"❏ ʙʀᴏᴀᴅᴄᴀsᴛ & ᴀᴜᴛᴏ-ғᴏʀᴡᴀʀᴅ ʙᴏᴛs\n"+
			"❏ ᴀᴜᴛᴏᴍᴀᴛɪᴏɴ & ᴜᴛɪʟɪᴛʏ ʙᴏᴛs\n"+
			"❏ ғᴜɴ & ᴇɴᴛᴇʀᴛᴀɪɴᴍᴇɴᴛ ʙᴏᴛs\n"+
			"❏ ғɪʟᴇ & ᴍᴇᴅɪᴀ ʙᴏᴛs\n"+
			"❏ sᴇᴀʀᴄʜ & ᴅᴏᴡɴʟᴏᴀᴅᴇʀ ʙᴏᴛs\n"+
			"❏ ᴘᴀʏᴍᴇɴᴛ & sᴜʙsᴄʀɪᴘᴛɪᴏɴ ʙᴏᴛs\n"+
			"❏ ᴜsᴇʀʙᴏᴛs & ᴛᴇʟᴇɢʀᴀᴍ ᴛᴏᴏʟs\n"+
			"❏ ᴄʟᴏɴᴇ ʙᴏᴛs & ᴄᴜsᴛᴏᴍ sʏsᴛᴇᴍs\n"+
			"❏ ᴀɪ ᴀssɪsᴛᴀɴᴛs & ᴀᴘɪ ɪɴᴛᴇɢʀᴀᴛɪᴏɴ\n"+
			"❏ ʀᴇsᴛ ᴀᴘɪs & ʙᴀᴄᴋᴇɴᴅ sʏsᴛᴇᴍs\n"+
			"❏ ᴡᴇʙ ᴅᴀsʜʙᴏᴀʀᴅs & ᴀᴅᴍɪɴ ᴘᴀɴᴇʟs\n"+
			"❏ ᴄᴜsᴛᴏᴍ ᴀᴘᴋ / ᴀɴᴅʀᴏɪᴅ</blockquote>\n\n"+
			"<blockquote>👇 <i>ᴘᴀsᴛᴇ ᴀ ɢɪᴛʜᴜʙ ʀᴇᴘᴏsɪᴛᴏʀʏ ʟɪɴᴋ ᴏʀ ᴜᴘʟᴏᴀᴅ ᴀ .ᴢɪᴘ ᴀʀᴄʜɪᴠᴇ ᴛᴏ ʙᴇɢɪɴ!</i></blockquote>",
		header,
		firstName,
	)

	markup := MakeStartKeyboard(isSudo, b.cfg.OwnerUsername, b.cfg.RepoURL, b.cfg.SupportChat, b.cfg.FSubChannel)

	if b.cfg.StartImgURL != "" {
		photo := tgbotapi.NewPhoto(msg.Chat.ID, tgbotapi.FileURL(b.cfg.StartImgURL))
		photo.Caption = greeting
		photo.ParseMode = "HTML"
		photo.ReplyMarkup = markup
		_, err := b.api.Send(photo)
		if err == nil {
			return
		}
	}

	reply := tgbotapi.NewMessage(msg.Chat.ID, greeting)
	reply.ParseMode = "HTML"
	reply.ReplyMarkup = markup
	_, _ = b.api.Send(reply)
}

func (b *Bot) sendHelpPage(chatID int64, messageID int, page int) {
	var body string
	title := renamer.ToSmallCaps(fmt.Sprintf("Help & Documentation [Page %d/3]", page))

	switch page {
	case 1:
		body = fmt.Sprintf(
			"<blockquote>📖 <b>%s</b></blockquote>\n\n"+
				"<blockquote><b>❏ ɢᴇᴛᴛɪɴɢ sᴛᴀʀᴛᴇᴅ:</b>\n"+
				"❏ sᴇɴᴅ ᴀ ᴘᴜʙʟɪᴄ ɢɪᴛʜᴜʙ ᴜʀʟ ᴏʀ ᴜᴘʟᴏᴀᴅ ᴀ .ᴢɪᴘ ᴀʀᴄʜɪᴠᴇ.\n"+
				"❏ ᴇɴᴛᴇʀ ᴛʜᴇ ᴇxᴀᴄᴛ ᴏʟᴅ ᴍᴏᴅᴜʟᴇ ɴᴀᴍᴇ (ᴇ.ɢ. <code>ʏᴜᴋᴋɪ</code>).\n"+
				"❏ ᴇɴᴛᴇʀ ʏᴏᴜʀ ɴᴇᴡ ᴍᴏᴅᴜʟᴇ ɴᴀᴍᴇ (ᴇ.ɢ. <code>ᴘᴜʟsᴇ</code>).\n"+
				"❏ ᴜsᴇ <code>/author &lt;name&gt;</code> ᴛᴏ sᴄᴀɴ & ʀᴇᴘʟᴀᴄᴇ ᴀᴜᴛʜᴏʀ ɴᴀᴍᴇs.\n"+
				"❏ ᴜsᴇ <code>/links</code> ᴛᴏ ᴅɪsᴄᴏᴠᴇʀ & ʀᴇᴘʟᴀᴄᴇ ᴛᴇʟᴇɢʀᴀᴍ ʟɪɴᴋs.</blockquote>\n\n"+
				"<blockquote expandable><b>❏ sᴜᴘᴘᴏʀᴛᴇᴅ ʟᴀɴɢᴜᴀɢᴇs & ғɪʟᴇs:</b>\n"+
				"ᴘʏᴛʜᴏɴ (.ᴘʏ), ɢᴏʟᴀɴɢ (.ɢᴏ), ᴊᴀᴠᴀsᴄʀɪᴘᴛ / ᴛʏᴘᴇsᴄʀɪᴘᴛ (.ᴊs, .ᴛs), ʀᴜsᴛ (.ʀs), ᴄ / ᴄ++ (.ᴄ, .ᴄᴘᴘ, .ʜ), ᴊᴀᴠᴀ (.ᴊᴀᴠᴀ), sʜᴇʟʟ (.sʜ), ᴍᴀʀᴋᴅᴏᴡɴ (.ᴍᴅ), ᴊsᴏɴ, ʏᴀᴍʟ, ᴅᴏᴄᴋᴇʀғɪʟᴇ, ᴍᴀᴋᴇғɪʟᴇ, ᴀɴᴅ .ᴇɴᴠ.</blockquote>",
			title,
		)
	case 2:
		body = fmt.Sprintf(
			"<blockquote>🔡 <b>%s</b></blockquote>\n\n"+
				"<blockquote><b>❏ ᴜɴɪᴄᴏᴅᴇ ғᴏɴᴛ ʀᴇᴄᴏɢɴɪᴛɪᴏɴ:</b>\n"+
				"❏ ᴀᴜᴛᴏᴍᴀᴛɪᴄᴀʟʟʏ ᴅᴇᴄᴏᴅᴇs & ʀᴇɴᴀᴍᴇs sᴛʏʟɪᴢᴇᴅ ʟᴏᴏᴋᴀʟɪᴋᴇs (ʏᴜᴋᴋɪ, 𝐘𝐮𝐤𝐤𝐢, 𝒀𝒖𝒌𝒌𝒊).\n"+
				"❏ ɴᴏʀᴍᴀʟɪᴢᴇs 250+ ᴍᴀᴛʜᴇᴍᴀᴛɪᴄᴀʟ, sᴍᴀʟʟ-ᴄᴀᴘ, ᴀɴᴅ ʜᴏᴍᴏɢʟʏᴘʜ ʀᴜɴᴇs.</blockquote>\n\n"+
				"<blockquote expandable><b>❏ ᴅᴇʟɪᴠᴇʀʏ ᴘɪᴘᴇʟɪɴᴇs:</b>\n"+
				"❏ <b>ᴇxᴘᴏʀᴛ ᴀs ᴢɪᴘ:</b> ɪɴsᴛᴀɴᴛʟʏ ᴄᴏᴍᴘʀᴇssᴇs ᴀɴᴅ ᴜᴘʟᴏᴀᴅs ᴀ .ᴢɪᴘ ᴅᴏᴄᴜᴍᴇɴᴛ.\n"+
				"❏ <b>ᴘᴜsʜ ᴛᴏ ɢɪᴛʜᴜʙ:</b> ᴀᴜᴛʜᴇɴᴛɪᴄᴀᴛᴇs ᴀɴᴅ ᴘᴜsʜᴇs ᴅɪʀᴇᴄᴛʟʏ ᴛᴏ ʏᴏᴜʀ ɢɪᴛʜᴜʙ ʀᴇᴘᴏ.</blockquote>",
			title,
		)
	case 3:
		body = fmt.Sprintf(
			"<blockquote>👑 <b>%s</b></blockquote>\n\n"+
				"<blockquote><b>❏ sᴜᴅᴏ & ᴄʟᴏᴜᴅ ᴏᴘᴇʀᴀᴛɪᴏɴs:</b>\n"+
				"❏ <code>/stats</code> — ʀᴇᴀʟ-ᴛɪᴍᴇ ᴍᴇᴍᴏʀʏ, ɢᴏʀᴏᴜᴛɪɴᴇs, ʀᴇɴᴀᴍᴇs\n"+
				"❏ <code>/log on</code> / <code>/log off</code> — ᴛᴏɢɢʟᴇ ᴄʟᴇᴀɴ ᴇᴠᴇɴᴛ ʟᴏɢɢᴇʀ\n"+
				"❏ <code>/broadcast &lt;text&gt;</code> — sᴇɴᴅ ɢʟᴏʙᴀʟ ʙʀᴏᴀᴅᴄᴀsᴛ\n"+
				"❏ <code>/gban &lt;user_id&gt;</code> — ʙʟᴀᴄᴋʟɪsᴛ sᴘᴀᴍ ᴜsᴇʀ\n"+
				"❏ <code>/ungban &lt;user_id&gt;</code> — ʀᴇᴍᴏᴠᴇ ʙʟᴀᴄᴋʟɪsᴛ\n"+
				"❏ <code>/users</code> — ᴛᴏᴛᴀʟ ʀᴇɢɪsᴛᴇʀᴇᴅ ᴜsᴇʀ ᴄᴏᴜɴᴛ</blockquote>",
			title,
		)
	}

	markup := MakeHelpKeyboard(page)

	if messageID != 0 {
		editCaption := tgbotapi.NewEditMessageCaption(chatID, messageID, body)
		editCaption.ParseMode = "HTML"
		editCaption.ReplyMarkup = &markup
		_, err := b.api.Send(editCaption)
		if err != nil {
			editText := tgbotapi.NewEditMessageText(chatID, messageID, body)
			editText.ParseMode = "HTML"
			editText.ReplyMarkup = &markup
			_, _ = b.api.Send(editText)
		}
		return
	}

	if b.cfg.HelpImgURL != "" {
		photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileURL(b.cfg.HelpImgURL))
		photo.Caption = body
		photo.ParseMode = "HTML"
		photo.ReplyMarkup = markup
		_, err := b.api.Send(photo)
		if err == nil {
			return
		}
	}

	msg := tgbotapi.NewMessage(chatID, body)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = markup
	_, _ = b.api.Send(msg)
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
		renamer.ToSmallCaps("Start New Rebranding Task"),
	)

	reply := tgbotapi.NewMessage(chatID, text)
	reply.ParseMode = "HTML"
	_, _ = b.api.Send(reply)
}

func (b *Bot) promptAuthor(chatID int64, userID int64, username string, targetName string) {
	session := b.sm.Get(userID)
	if session == nil {
		sessionID := fmt.Sprintf("sess_%d_%d", userID, time.Now().Unix())
		session = &SessionState{
			ID:       sessionID,
			UserID:   userID,
			Username: username,
		}
		b.sm.Set(userID, session)
	}

	if targetName == "" {
		session.Step = "AWAITING_AUTHOR_TARGET"
		text := fmt.Sprintf(
			"<blockquote>🔍 <b>%s</b></blockquote>\n\n"+
				"<blockquote>Please reply with the <b>author name, handle, or identifier</b> to scan for in the codebase:\n"+
				"<i>Example:</i> <code>Rahul</code> or stylized <code>𝐑ᴀʜυʟ</code></blockquote>",
			renamer.ToSmallCaps("Scan Author / Custom Identifier"),
		)
		reply := tgbotapi.NewMessage(chatID, text)
		reply.ParseMode = "HTML"
		_, _ = b.api.Send(reply)
		return
	}

	session.AuthorQuery = targetName
	if session.LocalDir == "" {
		session.Step = "AWAITING_AUTHOR_SOURCE"
		text := fmt.Sprintf(
			"<blockquote>🔍 <b>Target Configured:</b> <code>%s</code></blockquote>\n\n"+
				"<blockquote>Please send the <b>GitHub Repository URL</b> or upload a <b>.zip</b> archive to scan for this identifier.</blockquote>",
			targetName,
		)
		reply := tgbotapi.NewMessage(chatID, text)
		reply.ParseMode = "HTML"
		_, _ = b.api.Send(reply)
		return
	}

	b.executeAuthorScan(chatID, session, targetName)
}

func (b *Bot) executeAuthorScan(chatID int64, session *SessionState, targetName string) {
	statusMsg := tgbotapi.NewMessage(chatID, fmt.Sprintf("<blockquote>🔍 <b>Scanning codebase for '%s' (including Unicode lookalikes)...</b></blockquote>", targetName))
	statusMsg.ParseMode = "HTML"
	sent, _ := b.api.Send(statusMsg)

	engine := renamer.NewEngine()
	report, err := engine.FindOccurrences(renamer.FindOptions{
		TargetDir:    session.LocalDir,
		SearchTerm:   targetName,
		IncludeFonts: true,
	})
	if err != nil {
		b.sendAIErrorBlock(chatID, sent.MessageID, "Author Codebase Scanning", err)
		return
	}

	if report.TotalHits == 0 {
		text := fmt.Sprintf(
			"<blockquote>🔍 <b>%s</b></blockquote>\n\n"+
				"<blockquote>• <b>Target Query:</b> <code>%s</code>\n"+
				"• <b>Matches Found:</b> <code>0</code> occurrences\n\n"+
				"No occurrences of this identifier were detected across the codebase.</blockquote>",
			renamer.ToSmallCaps("Scan Telemetry"),
			targetName,
		)
		edit := tgbotapi.NewEditMessageText(chatID, sent.MessageID, text)
		edit.ParseMode = "HTML"
		_, _ = b.api.Send(edit)
		return
	}

	var fileList []string
	maxShow := 8
	for i, h := range report.Hits {
		if i >= maxShow {
			fileList = append(fileList, fmt.Sprintf("<i>... and %d more files</i>", len(report.Hits)-maxShow))
			break
		}
		fileList = append(fileList, fmt.Sprintf("• <code>%s</code> (%d hits)", h.Path, h.Count))
	}

	text := fmt.Sprintf(
		"<blockquote>🔍 <b>%s</b></blockquote>\n\n"+
			"<blockquote><b>Scan Detection Telemetry:</b>\n"+
			"• <b>Target Query:</b> <code>%s</code>\n"+
			"• <b>Total Matches:</b> <code>%d</code> occurrences\n"+
			"• <b>Affected Files:</b> <code>%d</code> files</blockquote>\n\n"+
			"<blockquote expandable><b>File Locations:</b>\n%s</blockquote>\n\n"+
			"<blockquote>Do you wish to replace all occurrences across the codebase?</blockquote>",
		renamer.ToSmallCaps("Author Occurrences Detected"),
		targetName,
		report.TotalHits,
		report.FilesCount,
		strings.Join(fileList, "\n"),
	)

	edit := tgbotapi.NewEditMessageText(chatID, sent.MessageID, text)
	edit.ParseMode = "HTML"
	markup := MakeConfirmAuthorKeyboard(session.ID)
	edit.ReplyMarkup = &markup
	_, _ = b.api.Send(edit)
}

func (b *Bot) promptLinks(chatID int64, userID int64, username string) {
	session := b.sm.Get(userID)
	if session == nil {
		sessionID := fmt.Sprintf("sess_%d_%d", userID, time.Now().Unix())
		session = &SessionState{
			ID:       sessionID,
			UserID:   userID,
			Username: username,
		}
		b.sm.Set(userID, session)
	}

	if session.LocalDir == "" {
		session.Step = "AWAITING_LINKS_SOURCE"
		text := fmt.Sprintf(
			"<blockquote>🔗 <b>%s</b></blockquote>\n\n"+
				"<blockquote>Please send the <b>GitHub Repository URL</b> or upload a <b>.zip archive</b> first to discover and replace links across the codebase.</blockquote>",
			renamer.ToSmallCaps("Scan Repository Links"),
		)
		reply := tgbotapi.NewMessage(chatID, text)
		reply.ParseMode = "HTML"
		_, _ = b.api.Send(reply)
		return
	}

	b.executeLinkScan(chatID, session)
}

func (b *Bot) executeLinkScan(chatID int64, session *SessionState) {
	statusMsg := tgbotapi.NewMessage(chatID, "<blockquote>🔍 <b>Scanning repository files for links, channels, and endpoints...</b></blockquote>")
	statusMsg.ParseMode = "HTML"
	sent, _ := b.api.Send(statusMsg)

	engine := renamer.NewEngine()
	report, err := engine.ScanLinks(session.LocalDir)
	if err != nil {
		b.sendAIErrorBlock(chatID, sent.MessageID, "Repository Link Scanning", err)
		return
	}

	if len(report.Links) == 0 {
		text := fmt.Sprintf(
			"<blockquote>🔗 <b>%s</b></blockquote>\n\n"+
				"<blockquote>No channel links, Telegram URLs, or web endpoints were discovered in this repository.</blockquote>",
			renamer.ToSmallCaps("Link Scan Complete"),
		)
		edit := tgbotapi.NewEditMessageText(chatID, sent.MessageID, text)
		edit.ParseMode = "HTML"
		_, _ = b.api.Send(edit)
		return
	}

	session.DiscoveredLinks = report.Links
	session.LinkPage = 1
	totalPages := (len(report.Links) + 4) / 5

	text := fmt.Sprintf(
		"<blockquote>🔗 <b>%s</b></blockquote>\n\n"+
			"<blockquote><b>Discovered Telemetry:</b>\n"+
			"• <b>Unique Links Found:</b> <code>%d</code>\n"+
			"• <b>Total Link Occurrences:</b> <code>%d</code></blockquote>\n\n"+
			"<blockquote>👇 <i>Click any link in the interactive list below to replace it across the entire codebase:</i></blockquote>",
		renamer.ToSmallCaps("Repository Links Telemetry"),
		len(report.Links),
		report.TotalFound,
	)

	edit := tgbotapi.NewEditMessageText(chatID, sent.MessageID, text)
	edit.ParseMode = "HTML"
	markup := MakeLinksPaginationKeyboard(report.Links, 1, totalPages, session.ID)
	edit.ReplyMarkup = &markup
	_, _ = b.api.Send(edit)
}

func (b *Bot) handleRepoLink(chatID int64, userID int64, username, repoURL string) {
	existing := b.sm.Get(userID)
	isAuthorScan := existing != nil && (existing.Step == "AWAITING_AUTHOR_SOURCE" || existing.AuthorQuery != "")
	isLinksScan := existing != nil && existing.Step == "AWAITING_LINKS_SOURCE"
	authorQuery := ""
	if isAuthorScan && existing != nil {
		authorQuery = existing.AuthorQuery
	}

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

	session := &SessionState{
		ID:          sessionID,
		UserID:      userID,
		Username:    username,
		RepoURL:     repoURL,
		LocalDir:    workDir,
		AuthorQuery: authorQuery,
	}
	b.sm.Set(userID, session)

	if isAuthorScan && authorQuery != "" {
		_, _ = b.api.Request(tgbotapi.NewDeleteMessage(chatID, sent.MessageID))
		b.executeAuthorScan(chatID, session, authorQuery)
		return
	}

	if isLinksScan {
		_, _ = b.api.Request(tgbotapi.NewDeleteMessage(chatID, sent.MessageID))
		b.executeLinkScan(chatID, session)
		return
	}

	session.Step = "AWAITING_OLD_NAME"

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

	existing := b.sm.Get(userID)
	isAuthorScan := existing != nil && (existing.Step == "AWAITING_AUTHOR_SOURCE" || existing.AuthorQuery != "")
	isLinksScan := existing != nil && existing.Step == "AWAITING_LINKS_SOURCE"
	authorQuery := ""
	if isAuthorScan && existing != nil {
		authorQuery = existing.AuthorQuery
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

	session := &SessionState{
		ID:          sessionID,
		UserID:      userID,
		Username:    username,
		LocalDir:    extractedDir,
		AuthorQuery: authorQuery,
	}
	b.sm.Set(userID, session)

	if isAuthorScan && authorQuery != "" {
		_, _ = b.api.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, sent.MessageID))
		b.executeAuthorScan(msg.Chat.ID, session, authorQuery)
		return
	}

	if isLinksScan {
		_, _ = b.api.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, sent.MessageID))
		b.executeLinkScan(msg.Chat.ID, session)
		return
	}

	session.Step = "AWAITING_OLD_NAME"

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

	case "AWAITING_AUTHOR_TARGET":
		b.promptAuthor(chatID, session.UserID, session.Username, text)

	case "AWAITING_AUTHOR_SOURCE":
		if strings.HasPrefix(text, "http://") || strings.HasPrefix(text, "https://") {
			b.handleRepoLink(chatID, msg.From.ID, msg.From.UserName, text)
		} else {
			reply := tgbotapi.NewMessage(chatID, "<blockquote>⚠️ <i>Please provide a valid GitHub repository link or upload a .zip archive.</i></blockquote>")
			reply.ParseMode = "HTML"
			_, _ = b.api.Send(reply)
		}

	case "AWAITING_LINKS_SOURCE":
		if strings.HasPrefix(text, "http://") || strings.HasPrefix(text, "https://") {
			b.handleRepoLink(chatID, msg.From.ID, msg.From.UserName, text)
		} else {
			reply := tgbotapi.NewMessage(chatID, "<blockquote>⚠️ <i>Please provide a valid GitHub repository link or upload a .zip archive.</i></blockquote>")
			reply.ParseMode = "HTML"
			_, _ = b.api.Send(reply)
		}

	case "AWAITING_LINK_REPLACEMENT":
		newLink := text
		session.Step = "AWAITING_DELIVERY_CHOICE"

		statusMsg := tgbotapi.NewMessage(chatID, "<blockquote>⚙️ <b>Replacing link across codebase and validating syntax...</b></blockquote>")
		statusMsg.ParseMode = "HTML"
		sent, _ := b.api.Send(statusMsg)

		engine := renamer.NewEngine()
		filesMod, repCount, err := engine.ReplaceLink(session.LocalDir, session.SelectedLink, newLink)
		if err != nil {
			logger.LogRenameFailure(session.UserID, session.Username, err)
			b.sendAIErrorBlock(chatID, sent.MessageID, "Link Replacement Engine", err)
			return
		}

		syntaxRes := renamer.VerifyDirectorySyntax(session.LocalDir)
		var syntaxSummary string
		if len(syntaxRes.Warnings) == 0 {
			syntaxSummary = fmt.Sprintf("• <b>Syntax Integrity:</b> <code>100%% Valid (%d files checked)</code>", syntaxRes.Passed)
		} else {
			syntaxSummary = fmt.Sprintf("• <b>Syntax Integrity:</b> <code>%d passed, %d warnings</code>", syntaxRes.Passed, len(syntaxRes.Warnings))
		}

		_ = b.db.IncrementRenames()
		logger.LogRenameSuccess(session.UserID, session.Username, session.SelectedLink, newLink, len(session.DiscoveredLinks), filesMod, repCount, 0)

		resultText := fmt.Sprintf(
			"<blockquote>🎉 <b>%s</b></blockquote>\n\n"+
				"<blockquote><b>Link Replacement Telemetry:</b>\n"+
				"• <b>Original Link:</b> <code>%s</code>\n"+
				"• <b>New Link:</b> <code>%s</code>\n"+
				"• <b>Occurrences Replaced:</b> <code>%d</code>\n"+
				"• <b>Files Modified:</b> <code>%d</code>\n"+
				"%s</blockquote>\n\n"+
				"<blockquote>📦 <b>Select an action below:</b></blockquote>",
			renamer.ToSmallCaps("Link Replacement Completed"),
			session.SelectedLink,
			newLink,
			repCount,
			filesMod,
			syntaxSummary,
		)

		edit := tgbotapi.NewEditMessageText(chatID, sent.MessageID, resultText)
		edit.ParseMode = "HTML"
		markup := MakePostLinkReplaceKeyboard(session.ID)
		edit.ReplyMarkup = &markup
		_, _ = b.api.Send(edit)

	case "AWAITING_AUTHOR_REPLACEMENT":
		session.NewName = text
		session.Step = "AWAITING_DELIVERY_CHOICE"

		statusMsg := tgbotapi.NewMessage(chatID, "<blockquote>⚙️ <b>Replacing occurrences and validating syntax...</b></blockquote>")
		statusMsg.ParseMode = "HTML"
		sent, _ := b.api.Send(statusMsg)

		engine := renamer.NewEngine()
		report, err := engine.Execute(renamer.RenameOptions{
			TargetDir:    session.LocalDir,
			OldName:      session.AuthorQuery,
			NewName:      session.NewName,
			IncludeFonts: true,
		})

		if err != nil {
			logger.LogRenameFailure(session.UserID, session.Username, err)
			b.sendAIErrorBlock(chatID, sent.MessageID, "Author Replacement Engine", err)
			return
		}

		syntaxRes := renamer.VerifyDirectorySyntax(session.LocalDir)
		var syntaxSummary string
		if len(syntaxRes.Warnings) == 0 {
			syntaxSummary = fmt.Sprintf("• <b>Syntax Integrity:</b> <code>100%% Valid (%d files checked)</code>", syntaxRes.Passed)
		} else {
			syntaxSummary = fmt.Sprintf("• <b>Syntax Integrity:</b> <code>%d passed, %d warnings</code>", syntaxRes.Passed, len(syntaxRes.Warnings))
		}

		_ = b.db.IncrementRenames()
		logger.LogRenameSuccess(session.UserID, session.Username, session.AuthorQuery, session.NewName, report.FilesScanned, report.FilesModified, report.ReplacementsCount, report.DirectoriesRenamed+report.FilesRenamed)

		resultText := fmt.Sprintf(
			"<blockquote>🎉 <b>%s</b></blockquote>\n\n"+
				"<blockquote><b>Replacement Telemetry:</b>\n"+
				"• <b>Target Old:</b> <code>%s</code>\n"+
				"• <b>Target New:</b> <code>%s</code>\n"+
				"• <b>Occurrences Replaced:</b> <code>%d</code>\n"+
				"• <b>Files Modified:</b> <code>%d</code>\n"+
				"• <b>Paths Renamed:</b> <code>%d</code>\n"+
				"%s</blockquote>\n\n"+
				"<blockquote>📦 <b>Select your delivery method below:</b></blockquote>",
			renamer.ToSmallCaps("Author Replacement Completed"),
			session.AuthorQuery,
			session.NewName,
			report.ReplacementsCount,
			report.FilesModified,
			report.DirectoriesRenamed+report.FilesRenamed,
			syntaxSummary,
		)

		edit := tgbotapi.NewEditMessageText(chatID, sent.MessageID, resultText)
		edit.ParseMode = "HTML"
		markup := MakeDeliveryChoiceKeyboard(session.ID)
		edit.ReplyMarkup = &markup
		_, _ = b.api.Send(edit)

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

		syntaxRes := renamer.VerifyDirectorySyntax(session.LocalDir)
		var syntaxSummary string
		if len(syntaxRes.Warnings) == 0 {
			syntaxSummary = fmt.Sprintf("• <b>Syntax Integrity:</b> <code>100%% Valid (%d files checked)</code>", syntaxRes.Passed)
		} else {
			syntaxSummary = fmt.Sprintf("• <b>Syntax Integrity:</b> <code>%d passed, %d warnings</code>", syntaxRes.Passed, len(syntaxRes.Warnings))
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
				"• <b>Paths/Directories Renamed:</b> <code>%d</code>\n"+
				"%s</blockquote>\n\n"+
				"<blockquote>📦 <b>Select your delivery method below:</b></blockquote>",
			renamer.ToSmallCaps("Renaming Completed Successfully"),
			session.OldName,
			session.NewName,
			report.FilesScanned,
			report.FilesModified,
			report.ReplacementsCount,
			report.DirectoriesRenamed+report.FilesRenamed,
			syntaxSummary,
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

	oldLabel := session.OldName
	if oldLabel == "" {
		oldLabel = session.AuthorQuery
	}
	if oldLabel == "" {
		oldLabel = session.SelectedLink
	}
	commitMsg := fmt.Sprintf("feat: rebrand %s to %s via SUDEEPBOTS Module Renamer", oldLabel, session.NewName)
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
		renamer.ToSmallCaps("Pushed to GitHub Successfully"),
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
