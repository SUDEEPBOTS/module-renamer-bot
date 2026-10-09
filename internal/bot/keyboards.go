package bot

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/renamer"
)

// MakeStartKeyboard creates the main menu keyboard.
// Only includes Admin Panel if isSudo is true!
func MakeStartKeyboard(isSudo bool, supportLink, channelLink string) tgbotapi.InlineKeyboardMarkup {
	supportText := renamer.ToAestheticFancy("Support")
	channelText := renamer.ToAestheticFancy("Channel")
	helpText := renamer.ToBoldSerif("Help & Commands")

	var rows [][]tgbotapi.InlineKeyboardButton

	row1 := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonURL("📢 "+channelText, channelLink),
		tgbotapi.NewInlineKeyboardButtonURL("💬 "+supportText, supportLink),
	}
	rows = append(rows, row1)

	row2 := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("📖 "+helpText, "cmd_help"),
		tgbotapi.NewInlineKeyboardButtonData("⚡ 𝐑𝐞𝐧𝐚𝐦𝐞 𝐑𝐞𝐩𝐨", "cmd_rename_prompt"),
	}
	rows = append(rows, row2)

	// Admin Panel button is STRICTLY for sudo / admin users
	if isSudo {
		adminText := renamer.ToBoldSerif("Admin Panel")
		rowAdmin := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("👑 "+adminText, "cmd_admin_panel"),
		}
		rows = append(rows, rowAdmin)
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// MakeDeliveryChoiceKeyboard asks user whether to download as ZIP or Push to GitHub.
func MakeDeliveryChoiceKeyboard(sessionID string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📦 Download as ZIP", "export_zip:"+sessionID),
			tgbotapi.NewInlineKeyboardButtonData("🚀 Push to GitHub", "export_github:"+sessionID),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Cancel Operation", "cancel:"+sessionID),
		),
	)
}

// MakeFSubKeyboard generates Must-Join keyboard.
func MakeFSubKeyboard(channelUsername string) tgbotapi.InlineKeyboardMarkup {
	link := "https://t.me/" + channelUsername
	joinText := renamer.ToBoldSerif("Join Channel")

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("🔔 "+joinText, link),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Verify Membership", "verify_fsub"),
		),
	)
}

// MakeAdminPanelKeyboard builds the sudo admin control console.
func MakeAdminPanelKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 System Stats", "admin_stats"),
			tgbotapi.NewInlineKeyboardButtonData("👥 User Count", "admin_users"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📢 Broadcast Info", "admin_bcast_info"),
			tgbotapi.NewInlineKeyboardButtonData("🚫 GBan Guide", "admin_gban_info"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Main", "back_start"),
		),
	)
}
