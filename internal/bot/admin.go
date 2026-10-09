package bot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/renamer"
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/utils"
)

// HandleAdminCommand dispatches sudo/admin commands.
func (b *Bot) HandleAdminCommand(msg *tgbotapi.Message) {
	if !b.cfg.IsSudo(msg.From.ID) {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "⛔ <i>This command is restricted to Sudo Administrators.</i>")
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

func (b *Bot) sendAdminPanel(chatID int64) {
	title := renamer.ToBoldSerif("Admin Control Panel")
	text := fmt.Sprintf(
		"👑 <b><u>%s</u></b>\n\n"+
			"Welcome to the Sudo Management Deck.\n\n"+
			"<b>Available Commands:</b>\n"+
			"• <code>/stats</code> — Real-time RAM, CPU, Goroutines & Counters\n"+
			"• <code>/broadcast &lt;msg&gt;</code> — Send global broadcast to all users\n"+
			"• <code>/gban &lt;user_id&gt; [reason]</code> — Globally blacklist a user\n"+
			"• <code>/ungban &lt;user_id&gt;</code> — Remove global blacklist\n"+
			"• <code>/users</code> — Total active users counter\n",
		title,
	)

	reply := tgbotapi.NewMessage(chatID, text)
	reply.ParseMode = "HTML"
	reply.ReplyMarkup = MakeAdminPanelKeyboard()
	_, _ = b.api.Send(reply)
}

func (b *Bot) sendStats(chatID int64) {
	totalUsers, _ := b.db.CountUsers()
	totalRenames := b.db.GetTotalRenames()
	stats := utils.GetSystemStats()

	text := stats.FormatStatsMessage(totalUsers, totalRenames)
	reply := tgbotapi.NewMessage(chatID, text)
	reply.ParseMode = "HTML"
	_, _ = b.api.Send(reply)
}

func (b *Bot) handleGBan(msg *tgbotapi.Message, args string) {
	parts := strings.Fields(args)
	if len(parts) == 0 {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "⚠️ <b>Usage:</b> <code>/gban &lt;user_id&gt; [reason]</code>")
		reply.ParseMode = "HTML"
		_, _ = b.api.Send(reply)
		return
	}

	targetID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "❌ <i>Invalid User ID format.</i>")
		reply.ParseMode = "HTML"
		_, _ = b.api.Send(reply)
		return
	}

	reason := "Violation of bot policies"
	if len(parts) > 1 {
		reason = strings.Join(parts[1:], " ")
	}

	_ = b.db.BanUser(targetID, reason)

	reply := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("🚫 <b>User <code>%d</code> has been GBanned!</b>\n<b>Reason:</b> %s", targetID, reason))
	reply.ParseMode = "HTML"
	_, _ = b.api.Send(reply)
}

func (b *Bot) handleUnGBan(msg *tgbotapi.Message, args string) {
	targetID, err := strconv.ParseInt(strings.TrimSpace(args), 10, 64)
	if err != nil {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "⚠️ <b>Usage:</b> <code>/ungban &lt;user_id&gt;</code>")
		reply.ParseMode = "HTML"
		_, _ = b.api.Send(reply)
		return
	}

	_ = b.db.UnbanUser(targetID)

	reply := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("✅ <b>User <code>%d</code> has been unbanned!</b>", targetID))
	reply.ParseMode = "HTML"
	_, _ = b.api.Send(reply)
}

func (b *Bot) handleBroadcast(msg *tgbotapi.Message, text string) {
	if text == "" && msg.ReplyToMessage == nil {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "⚠️ <b>Usage:</b> Reply to a message with <code>/broadcast</code> or provide text.")
		reply.ParseMode = "HTML"
		_, _ = b.api.Send(reply)
		return
	}

	users, err := b.db.GetUsers()
	if err != nil || len(users) == 0 {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "❌ <i>No users found in database to broadcast to.</i>")
		reply.ParseMode = "HTML"
		_, _ = b.api.Send(reply)
		return
	}

	statusMsg := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("⏳ <i>Broadcasting to %d users in background...</i>", len(users)))
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
			time.Sleep(35 * time.Millisecond) // Respect Telegram rate limits (~30 msgs/sec)
		}

		edit := tgbotapi.NewEditMessageText(
			msg.Chat.ID,
			sentStatus.MessageID,
			fmt.Sprintf("📢 <b>Broadcast Completed!</b>\n\n✅ <b>Delivered:</b> %d\n❌ <b>Failed / Blocked:</b> %d", success, failed),
		)
		edit.ParseMode = "HTML"
		_, _ = b.api.Send(edit)
	}()
}

func (b *Bot) handleUserCount(chatID int64) {
	count, _ := b.db.CountUsers()
	reply := tgbotapi.NewMessage(chatID, fmt.Sprintf("👥 <b>Total Registered Users:</b> <code>%d</code>", count))
	reply.ParseMode = "HTML"
	_, _ = b.api.Send(reply)
}
