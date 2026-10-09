package bot

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/renamer"
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/utils"
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/vcs"
)

// HandleCallbackQuery dispatches inline keyboard button clicks.
func (b *Bot) HandleCallbackQuery(query *tgbotapi.CallbackQuery) {
	if query == nil || query.From == nil {
		return
	}

	userID := query.From.ID
	data := query.Data
	chatID := query.Message.Chat.ID
	messageID := query.Message.MessageID

	// Always acknowledge callback query to dismiss loading wheel
	ack := tgbotapi.NewCallback(query.ID, "")
	_, _ = b.api.Request(ack)

	// FSub verification
	if data == "verify_fsub" {
		joined, _ := b.CheckFSub(userID)
		if joined {
			_ = b.db.AddUser(userID, query.From.UserName, query.From.FirstName)
			edit := tgbotapi.NewEditMessageText(chatID, messageID, "✅ <b>Membership verified!</b> You may now use the bot. Type /start to begin.")
			edit.ParseMode = "HTML"
			_, _ = b.api.Send(edit)
		} else {
			alert := tgbotapi.NewCallbackWithAlert(query.ID, "❌ You haven't joined yet! Please join the channel first.")
			_, _ = b.api.Request(alert)
		}
		return
	}

	// Commands
	if data == "cmd_help" {
		b.handleHelp(chatID)
		return
	}

	if data == "cmd_rename_prompt" {
		b.promptRename(chatID, userID)
		return
	}

	// Admin console callbacks
	if strings.HasPrefix(data, "admin_") || data == "cmd_admin_panel" {
		if !b.cfg.IsSudo(userID) {
			alert := tgbotapi.NewCallbackWithAlert(query.ID, "⛔ Access Denied: Sudo Admin only.")
			_, _ = b.api.Request(alert)
			return
		}

		switch data {
		case "cmd_admin_panel":
			b.sendAdminPanel(chatID)
		case "admin_stats":
			totalUsers, _ := b.db.CountUsers()
			totalRenames := b.db.GetTotalRenames()
			stats := utils.GetSystemStats()
			edit := tgbotapi.NewEditMessageText(chatID, messageID, stats.FormatStatsMessage(totalUsers, totalRenames))
			edit.ParseMode = "HTML"
			markup := MakeAdminPanelKeyboard()
			edit.ReplyMarkup = &markup
			_, _ = b.api.Send(edit)
		case "admin_users":
			count, _ := b.db.CountUsers()
			edit := tgbotapi.NewEditMessageText(chatID, messageID, fmt.Sprintf("👥 <b>Total Registered Users:</b> <code>%d</code>", count))
			edit.ParseMode = "HTML"
			markup := MakeAdminPanelKeyboard()
			edit.ReplyMarkup = &markup
			_, _ = b.api.Send(edit)
		case "admin_bcast_info":
			edit := tgbotapi.NewEditMessageText(chatID, messageID, "📢 <b>Broadcast Guide:</b>\n\nUse <code>/broadcast &lt;message&gt;</code> or reply to a text/photo message with <code>/broadcast</code> to send to all bot users.")
			edit.ParseMode = "HTML"
			markup := MakeAdminPanelKeyboard()
			edit.ReplyMarkup = &markup
			_, _ = b.api.Send(edit)
		case "admin_gban_info":
			edit := tgbotapi.NewEditMessageText(chatID, messageID, "🚫 <b>GBan Guide:</b>\n\nUse <code>/gban &lt;user_id&gt; [reason]</code> to blacklist a user, and <code>/ungban &lt;user_id&gt;</code> to remove the blacklist.")
			edit.ParseMode = "HTML"
			markup := MakeAdminPanelKeyboard()
			edit.ReplyMarkup = &markup
			_, _ = b.api.Send(edit)
		}
		return
	}

	if data == "back_start" {
		b.handleStart(query.Message)
		return
	}

	// Export callbacks
	if strings.HasPrefix(data, "export_zip:") {
		b.handleExportZip(chatID, messageID, userID)
		return
	}

	if strings.HasPrefix(data, "export_github:") {
		b.handleExportGitHub(chatID, messageID, userID)
		return
	}

	if strings.HasPrefix(data, "cancel:") {
		b.sm.Clear(userID)
		edit := tgbotapi.NewEditMessageText(chatID, messageID, "❌ <i>Operation cancelled and temporary files discarded.</i>")
		edit.ParseMode = "HTML"
		_, _ = b.api.Send(edit)
		return
	}
}

func (b *Bot) handleExportZip(chatID int64, messageID int, userID int64) {
	session := b.sm.Get(userID)
	if session == nil {
		reply := tgbotapi.NewMessage(chatID, "⚠️ <i>Session expired or not found. Please start over with /rename.</i>")
		reply.ParseMode = "HTML"
		_, _ = b.api.Send(reply)
		return
	}

	edit := tgbotapi.NewEditMessageText(chatID, messageID, "📦 <i>Compressing rebranded project into ZIP archive...</i>")
	edit.ParseMode = "HTML"
	_, _ = b.api.Send(edit)

	zipName := fmt.Sprintf("%s_rebranded.zip", session.NewName)
	zipPath := filepath.Join(b.cfg.WorkDir, zipName)

	err := vcs.CreateZip(session.LocalDir, zipPath, renamer.DefaultIgnoredDirs)
	if err != nil {
		editErr := tgbotapi.NewEditMessageText(chatID, messageID, fmt.Sprintf("❌ <i>Failed to create ZIP archive: %v</i>", err))
		editErr.ParseMode = "HTML"
		_, _ = b.api.Send(editErr)
		return
	}

	// Send document to user
	doc := tgbotapi.NewDocument(chatID, tgbotapi.FilePath(zipPath))
	doc.Caption = fmt.Sprintf("✅ <b>Project Archive:</b> <code>%s</code>\nRebranded from <b>%s</b> to <b>%s</b> via SUDEEPBOTS Module Renamer ⚡", zipName, session.OldName, session.NewName)
	doc.ParseMode = "HTML"

	_, err = b.api.Send(doc)
	if err != nil {
		editErr := tgbotapi.NewEditMessageText(chatID, messageID, fmt.Sprintf("❌ <i>Telegram upload failed: %v</i>", err))
		editErr.ParseMode = "HTML"
		_, _ = b.api.Send(editErr)
		return
	}

	// Cleanup
	_ = os.Remove(zipPath)
	b.sm.Clear(userID)
}

func (b *Bot) handleExportGitHub(chatID int64, messageID int, userID int64) {
	session := b.sm.Get(userID)
	if session == nil {
		reply := tgbotapi.NewMessage(chatID, "⚠️ <i>Session expired. Please start over with /rename.</i>")
		reply.ParseMode = "HTML"
		_, _ = b.api.Send(reply)
		return
	}

	session.Step = "AWAITING_GITHUB_PUSH_URL"

	promptText := "🚀 <b><u>" + renamer.ToBoldSerif("Deploy to GitHub") + "</u></b>\n\n" +
		"Please reply with the <b>Target GitHub Repository URL</b>\n" +
		"<i>Example:</i> <code>https://github.com/SUDEEPBOTS/NewRebrandedRepo</code>"

	edit := tgbotapi.NewEditMessageText(chatID, messageID, promptText)
	edit.ParseMode = "HTML"
	_, _ = b.api.Send(edit)
}
