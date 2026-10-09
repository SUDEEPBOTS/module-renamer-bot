package bot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/logger"
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/renamer"
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/utils"
)

func (b *Bot) HandleAdminCommand(msg *tgbotapi.Message) {
	if !b.cfg.IsSudo(msg.From.ID) {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "<blockquote>⛔ <i>This operation is restricted to Sudo Administrators.</i></blockquote>")
		reply.ParseMode = "HTML"
		_, _ = b.api.Send(reply)
		return
	}

	cmd := msg.Command()
	args := strings.TrimSpace(msg.CommandArguments())

	switch cmd {
	case "admin", "panel":
		b.sendAdminPanel(msg.Chat.ID)
	case "stats":
		b.sendStats(msg.Chat.ID)
	case "log", "logs":
		b.handleLogCommand(msg, args)
	case "gban":
		b.handleGBan(msg, args)
	case "ungban":
		b.handleUnGBan(msg, args)
	case "broadcast", "bcast":
		b.handleBroadcast(msg, args)
	case "users":
		b.handleUserCount(msg.Chat.ID)
	}
}

func (b *Bot) handleLogCommand(msg *tgbotapi.Message, args string) {
	arg := strings.ToLower(strings.TrimSpace(args))
	if arg == "on" || arg == "true" || arg == "enable" {
		logger.SetEnabled(true)
		reply := tgbotapi.NewMessage(msg.Chat.ID, "<blockquote>🟢 <b>Event Logging has been ENABLED.</b></blockquote>")
		reply.ParseMode = "HTML"
		_, _ = b.api.Send(reply)
		return
	}

	if arg == "off" || arg == "false" || arg == "disable" {
		logger.SetEnabled(false)
		reply := tgbotapi.NewMessage(msg.Chat.ID, "<blockquote>🔴 <b>Event Logging has been DISABLED.</b></blockquote>")
		reply.ParseMode = "HTML"
		_, _ = b.api.Send(reply)
		return
	}

	status := "ENABLED"
	if !logger.IsEnabled() {
		status = "DISABLED"
	}

	reply := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("<blockquote>📋 <b>Logger Status:</b> <code>%s</code>\n\nUsage: <code>/log on</code> or <code>/log off</code></blockquote>", status))
	reply.ParseMode = "HTML"
	_, _ = b.api.Send(reply)
}

func (b *Bot) sendAdminPanel(chatID int64) {
	title := renamer.ToSmallCaps("Admin Control Panel")
	text := fmt.Sprintf(
		"<blockquote>👑 <b>%s</b></blockquote>\n\n"+
			"<blockquote>Welcome to the Sudo Management Deck.</blockquote>\n\n"+
			"<blockquote expandable><b>Available Control Commands:</b>\n"+
			"• <code>/stats</code> — Real-time RAM, CPU, Goroutines & Counters\n"+
			"• <code>/log on</code> / <code>/log off</code> — Toggle clean event logger\n"+
			"• <code>/broadcast &lt;msg&gt;</code> — Send global broadcast to all users\n"+
			"• <code>/gban &lt;user_id&gt; [reason]</code> — Globally blacklist a user\n"+
			"• <code>/ungban &lt;user_id&gt;</code> — Remove global blacklist\n"+
			"• <code>/users</code> — Total active users counter</blockquote>",
		title,
	)

	reply := tgbotapi.NewMessage(chatID, text)
	reply.ParseMode = "HTML"
	reply.ReplyMarkup = MakeAdminPanelKeyboard(logger.IsEnabled())
	_, _ = b.api.Send(reply)
}

func (b *Bot) sendStats(chatID int64) {
	totalUsers, _ := b.db.CountUsers()
	totalRenames := b.db.GetTotalRenames()
	stats := utils.GetSystemStats()

	text := stats.FormatStatsMessage(totalUsers, totalRenames)

	if b.cfg.StatsImgURL != "" {
		photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileURL(b.cfg.StatsImgURL))
		photo.Caption = text
		photo.ParseMode = "HTML"
		_, err := b.api.Send(photo)
		if err == nil {
			return
		}
	}

	reply := tgbotapi.NewMessage(chatID, text)
	reply.ParseMode = "HTML"
	_, _ = b.api.Send(reply)
}

func (b *Bot) handleGBan(msg *tgbotapi.Message, args string) {
	parts := strings.Fields(args)
	if len(parts) == 0 {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "<blockquote>⚠️ <b>Usage:</b> <code>/gban &lt;user_id&gt; [reason]</code></blockquote>")
		reply.ParseMode = "HTML"
		_, _ = b.api.Send(reply)
		return
	}

	targetID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "<blockquote>❌ <i>Invalid User ID numeric format.</i></blockquote>")
		reply.ParseMode = "HTML"
		_, _ = b.api.Send(reply)
		return
	}

	reason := "Violation of bot policies"
	if len(parts) > 1 {
		reason = strings.Join(parts[1:], " ")
	}

	_ = b.db.BanUser(targetID, reason)

	reply := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("<blockquote>🚫 <b>User <code>%d</code> has been blacklisted!</b>\n<b>Reason:</b> %s</blockquote>", targetID, reason))
	reply.ParseMode = "HTML"
	_, _ = b.api.Send(reply)
}

func (b *Bot) handleUnGBan(msg *tgbotapi.Message, args string) {
	targetID, err := strconv.ParseInt(strings.TrimSpace(args), 10, 64)
	if err != nil {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "<blockquote>⚠️ <b>Usage:</b> <code>/ungban &lt;user_id&gt;</code></blockquote>")
		reply.ParseMode = "HTML"
		_, _ = b.api.Send(reply)
		return
	}

	_ = b.db.UnbanUser(targetID)

	reply := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("<blockquote>✅ <b>User <code>%d</code> has been unbanned!</b></blockquote>", targetID))
	reply.ParseMode = "HTML"
	_, _ = b.api.Send(reply)
}

func (b *Bot) handleBroadcast(msg *tgbotapi.Message, text string) {
	if text == "" && msg.ReplyToMessage == nil {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "<blockquote>⚠️ <b>Usage:</b> Reply to a message with <code>/broadcast</code> or provide text.</blockquote>")
		reply.ParseMode = "HTML"
		_, _ = b.api.Send(reply)
		return
	}

	users, err := b.db.GetUsers()
	if err != nil || len(users) == 0 {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "<blockquote>❌ <i>No users found in database to broadcast to.</i></blockquote>")
		reply.ParseMode = "HTML"
		_, _ = b.api.Send(reply)
		return
	}

	statusMsg := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("<blockquote>⏳ <i>Broadcasting to %d users in background...</i></blockquote>", len(users)))
	statusMsg.ParseMode = "HTML"
	sentStatus, _ := b.api.Send(statusMsg)

	go func() {
		success := 0
		failed := 0

		for _, userID := range users {
			var bMsg tgbotapi.MessageConfig
			if msg.ReplyToMessage != nil {
				bMsg = tgbotapi.NewMessage(userID, msg.ReplyToMessage.Text)
			} else {
				bMsg = tgbotapi.NewMessage(userID, text)
			}
			bMsg.ParseMode = "HTML"

			_, err := b.api.Send(bMsg)
			if err != nil {
				failed++
			} else {
				success++
			}
			time.Sleep(35 * time.Millisecond)
		}

		edit := tgbotapi.NewEditMessageText(
			msg.Chat.ID,
			sentStatus.MessageID,
			fmt.Sprintf("<blockquote>📢 <b>Broadcast Completed!</b>\n\n✅ <b>Delivered:</b> %d\n❌ <b>Failed / Blocked:</b> %d</blockquote>", success, failed),
		)
		edit.ParseMode = "HTML"
		_, _ = b.api.Send(edit)
	}()
}

func (b *Bot) handleUserCount(chatID int64) {
	count, _ := b.db.CountUsers()
	reply := tgbotapi.NewMessage(chatID, fmt.Sprintf("<blockquote>👥 <b>Total Registered Users:</b> <code>%d</code></blockquote>", count))
	reply.ParseMode = "HTML"
	_, _ = b.api.Send(reply)
}
