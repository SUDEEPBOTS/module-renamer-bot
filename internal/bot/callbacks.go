package bot

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/logger"
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/renamer"
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/utils"
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/vcs"
)

func (b *Bot) HandleCallbackQuery(query *tgbotapi.CallbackQuery) {
	if query == nil || query.From == nil {
		return
	}

	userID := query.From.ID
	data := query.Data
	chatID := query.Message.Chat.ID
	messageID := query.Message.MessageID

	ack := tgbotapi.NewCallback(query.ID, "")
	_, _ = b.api.Request(ack)

	if data == "verify_fsub" {
		joined, _ := b.CheckFSub(userID)
		if joined {
			_ = b.db.AddUser(userID, query.From.UserName, query.From.FirstName)
			edit := tgbotapi.NewEditMessageText(chatID, messageID, "<blockquote>✅ <b>Channel membership verified!</b> You can now use the bot. Type /start to open the main menu.</blockquote>")
			edit.ParseMode = "HTML"
			_, _ = b.api.Send(edit)
		} else {
			alert := tgbotapi.NewCallbackWithAlert(query.ID, "❌ You haven't joined yet! Please join the channel first.")
			_, _ = b.api.Request(alert)
		}
		return
	}

	if strings.HasPrefix(data, "help_page_") {
		pageNum, _ := strconv.Atoi(strings.TrimPrefix(data, "help_page_"))
		if pageNum < 1 {
			pageNum = 1
		}
		b.sendHelpPage(chatID, messageID, pageNum)
		return
	}

	if data == "cmd_rename_prompt" {
		b.promptRename(chatID, userID, query.From.UserName)
		return
	}

	if strings.HasPrefix(data, "admin_") || data == "cmd_admin_panel" {
		if !b.cfg.IsSudo(userID) {
			alert := tgbotapi.NewCallbackWithAlert(query.ID, "⛔ Access Denied: Sudo Administrators only.")
			_, _ = b.api.Request(alert)
			return
		}

		switch data {
		case "cmd_admin_panel":
			b.sendAdminPanel(chatID)
		case "admin_toggle_logger":
			newState := !logger.IsEnabled()
			logger.SetEnabled(newState)
			stateStr := "ENABLED"
			if !newState {
				stateStr = "DISABLED"
			}
			alert := tgbotapi.NewCallbackWithAlert(query.ID, fmt.Sprintf("Logger is now %s", stateStr))
			_, _ = b.api.Request(alert)

			markup := MakeAdminPanelKeyboard(logger.IsEnabled())
			editMarkup := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, markup)
			_, _ = b.api.Send(editMarkup)

		case "admin_stats":
			totalUsers, _ := b.db.CountUsers()
			totalRenames := b.db.GetTotalRenames()
			stats := utils.GetSystemStats()
			edit := tgbotapi.NewEditMessageText(chatID, messageID, stats.FormatStatsMessage(totalUsers, totalRenames))
			edit.ParseMode = "HTML"
			markup := MakeAdminPanelKeyboard(logger.IsEnabled())
			edit.ReplyMarkup = &markup
			_, _ = b.api.Send(edit)

		case "admin_users":
			count, _ := b.db.CountUsers()
			edit := tgbotapi.NewEditMessageText(chatID, messageID, fmt.Sprintf("<blockquote>👥 <b>Total Registered Users:</b> <code>%d</code></blockquote>", count))
			edit.ParseMode = "HTML"
			markup := MakeAdminPanelKeyboard(logger.IsEnabled())
			edit.ReplyMarkup = &markup
			_, _ = b.api.Send(edit)

		case "admin_bcast_info":
			edit := tgbotapi.NewEditMessageText(chatID, messageID, "<blockquote>📢 <b>Broadcast Instructions:</b>\n\nUse <code>/broadcast &lt;message&gt;</code> or reply to a text/photo message with <code>/broadcast</code> to send to all registered bot users.</blockquote>")
			edit.ParseMode = "HTML"
			markup := MakeAdminPanelKeyboard(logger.IsEnabled())
			edit.ReplyMarkup = &markup
			_, _ = b.api.Send(edit)

		case "admin_gban_info":
			edit := tgbotapi.NewEditMessageText(chatID, messageID, "<blockquote>🚫 <b>GBan Instructions:</b>\n\nUse <code>/gban &lt;user_id&gt; [reason]</code> to blacklist a spammer, and <code>/ungban &lt;user_id&gt;</code> to remove the blacklist.</blockquote>")
			edit.ParseMode = "HTML"
			markup := MakeAdminPanelKeyboard(logger.IsEnabled())
			edit.ReplyMarkup = &markup
			_, _ = b.api.Send(edit)
		}
		return
	}

	if data == "back_start" {
		b.handleStart(query.Message)
		return
	}

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
		edit := tgbotapi.NewEditMessageText(chatID, messageID, "<blockquote>❌ <b>Operation cancelled and temporary files discarded.</b></blockquote>")
		edit.ParseMode = "HTML"
		_, _ = b.api.Send(edit)
		return
	}
}

func (b *Bot) handleExportZip(chatID int64, messageID int, userID int64) {
	session := b.sm.Get(userID)
	if session == nil {
		reply := tgbotapi.NewMessage(chatID, "<blockquote>⚠️ <i>Session expired or not found. Please start over with /rename.</i></blockquote>")
		reply.ParseMode = "HTML"
		_, _ = b.api.Send(reply)
		return
	}

	edit := tgbotapi.NewEditMessageText(chatID, messageID, "<blockquote>📦 <b>Compressing rebranded project into ZIP archive...</b></blockquote>")
	edit.ParseMode = "HTML"
	_, _ = b.api.Send(edit)

	zipName := fmt.Sprintf("%s_rebranded.zip", session.NewName)
	zipPath := filepath.Join(b.cfg.WorkDir, zipName)

	err := vcs.CreateZip(session.LocalDir, zipPath, renamer.DefaultIgnoredDirs)
	if err != nil {
		logger.LogRenameFailure(userID, session.Username, err)
		b.sendAIErrorBlock(chatID, messageID, "ZIP Archive Generation", err)
		return
	}

	doc := tgbotapi.NewDocument(chatID, tgbotapi.FilePath(zipPath))
	doc.Caption = fmt.Sprintf("<blockquote>✅ <b>Rebranded Project:</b> <code>%s</code>\nRebranded from <b>%s</b> to <b>%s</b> via SUDEEPBOTS Module Renamer ⚡</blockquote>", zipName, session.OldName, session.NewName)
	doc.ParseMode = "HTML"

	_, err = b.api.Send(doc)
	if err != nil {
		logger.LogRenameFailure(userID, session.Username, err)
		b.sendAIErrorBlock(chatID, messageID, "Telegram Document Dispatch", err)
		return
	}

	_ = os.Remove(zipPath)
	b.sm.Clear(userID)
}

func (b *Bot) handleExportGitHub(chatID int64, messageID int, userID int64) {
	session := b.sm.Get(userID)
	if session == nil {
		reply := tgbotapi.NewMessage(chatID, "<blockquote>⚠️ <i>Session expired. Please start over with /rename.</i></blockquote>")
		reply.ParseMode = "HTML"
		_, _ = b.api.Send(reply)
		return
	}

	session.Step = "AWAITING_GITHUB_PUSH_URL"

	promptText := fmt.Sprintf(
		"<blockquote>🚀 <b>%s</b></blockquote>\n\n"+
			"<blockquote>Please reply with the <b>Target GitHub Repository URL</b>\n"+
			"<i>Example:</i> <code>https://github.com/SUDEEPBOTS/NewRebrandedRepo</code></blockquote>",
		renamer.ToBoldSerif("Deploy to GitHub"),
	)

	edit := tgbotapi.NewEditMessageText(chatID, messageID, promptText)
	edit.ParseMode = "HTML"
	_, _ = b.api.Send(edit)
}
