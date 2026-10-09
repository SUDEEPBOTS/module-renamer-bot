package bot

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/renamer"
)

func MakeStartKeyboard(isSudo bool, ownerUsername, repoURL, supportChat, fsubChannel string) tgbotapi.InlineKeyboardMarkup {
	ownerBtnText := renamer.ToAestheticFancy("Owner")
	repoBtnText := renamer.ToAestheticFancy("Repo")
	supportBtnText := renamer.ToAestheticFancy("Support")
	channelBtnText := renamer.ToAestheticFancy("Channel")
	helpBtnText := renamer.ToBoldSerif("Help & Commands")
	renameBtnText := renamer.ToBoldSerif("Rename Codebase")
	authorBtnText := renamer.ToAestheticFancy("Scan Author")
	linksBtnText := renamer.ToAestheticFancy("Scan Links")

	ownerLink := "https://t.me/" + strings.TrimPrefix(ownerUsername, "@")
	supportLink := "https://t.me/" + strings.TrimPrefix(supportChat, "@")
	channelLink := "https://t.me/" + strings.TrimPrefix(fsubChannel, "@")

	var rows [][]tgbotapi.InlineKeyboardButton

	row1 := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonURL("👑 "+ownerBtnText, ownerLink),
		tgbotapi.NewInlineKeyboardButtonURL("🔗 "+repoBtnText, repoURL),
	}
	rows = append(rows, row1)

	row2 := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonURL("💬 "+supportBtnText, supportLink),
		tgbotapi.NewInlineKeyboardButtonURL("📢 "+channelBtnText, channelLink),
	}
	rows = append(rows, row2)

	row3 := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("⚡ "+renameBtnText, "cmd_rename_prompt"),
		tgbotapi.NewInlineKeyboardButtonData("🔍 "+authorBtnText, "cmd_author_prompt"),
	}
	rows = append(rows, row3)

	row4 := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🔗 "+linksBtnText, "cmd_links_scan"),
		tgbotapi.NewInlineKeyboardButtonData("📖 "+helpBtnText, "help_page_1"),
	}
	rows = append(rows, row4)

	if isSudo {
		adminBtnText := renamer.ToBoldSerif("Admin Panel")
		rowAdmin := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("🛡️ "+adminBtnText, "cmd_admin_panel"),
		}
		rows = append(rows, rowAdmin)
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func MakeConfirmAuthorKeyboard(sessionID string) tgbotapi.InlineKeyboardMarkup {
	replaceText := renamer.ToBoldSerif("Replace Name")
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✍️ "+replaceText, "author_replace:"+sessionID),
			tgbotapi.NewInlineKeyboardButtonData("❌ Cancel", "cancel:"+sessionID),
		),
	)
}

func MakeHelpKeyboard(page int) tgbotapi.InlineKeyboardMarkup {
	backText := renamer.ToAestheticFancy("Back")
	nextText := renamer.ToAestheticFancy("Next")
	mainText := renamer.ToBoldSerif("Main Menu")

	var navRow []tgbotapi.InlineKeyboardButton

	switch page {
	case 1:
		navRow = append(navRow,
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s ➡️", nextText), "help_page_2"),
		)
	case 2:
		navRow = append(navRow,
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("⬅️ %s", backText), "help_page_1"),
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s ➡️", nextText), "help_page_3"),
		)
	case 3:
		navRow = append(navRow,
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("⬅️ %s", backText), "help_page_2"),
		)
	}

	closeRow := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🔙 "+mainText, "back_start"),
	}

	return tgbotapi.NewInlineKeyboardMarkup(navRow, closeRow)
}

func MakeLinksPaginationKeyboard(links []renamer.DiscoveredLink, page int, totalPages int, sessionID string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	perPage := 5
	startIdx := (page - 1) * perPage
	endIdx := startIdx + perPage
	if startIdx < 0 {
		startIdx = 0
	}
	if startIdx > len(links) {
		startIdx = len(links)
	}
	if endIdx > len(links) {
		endIdx = len(links)
	}

	for i := startIdx; i < endIdx; i++ {
		link := links[i]
		display := link.URL
		if len(display) > 28 {
			display = display[:25] + "..."
		}
		btnText := fmt.Sprintf("🔗 %s (%d)", display, link.Count)
		btnData := fmt.Sprintf("link_sel:%d", link.Index)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(btnText, btnData),
		))
	}

	var navRow []tgbotapi.InlineKeyboardButton
	if page > 1 {
		backText := renamer.ToAestheticFancy("Back")
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("⬅️ "+backText, fmt.Sprintf("link_page:%d", page-1)))
	}
	navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("📄 %d/%d", page, totalPages), "noop"))
	if page < totalPages {
		nextText := renamer.ToAestheticFancy("Next")
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData(nextText+" ➡️", fmt.Sprintf("link_page:%d", page+1)))
	}
	rows = append(rows, navRow)

	cancelRow := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("❌ Cancel", "cancel:"+sessionID),
		tgbotapi.NewInlineKeyboardButtonData("🔙 "+renamer.ToBoldSerif("Main"), "back_start"),
	)
	rows = append(rows, cancelRow)

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func MakePostLinkReplaceKeyboard(sessionID string) tgbotapi.InlineKeyboardMarkup {
	zipText := renamer.ToBoldSerif("Export ZIP")
	gitText := renamer.ToBoldSerif("Push to GitHub")
	moreLinksText := renamer.ToAestheticFancy("Scan More Links")
	mainText := renamer.ToBoldSerif("Main Menu")

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📦 "+zipText, "export_zip:"+sessionID),
			tgbotapi.NewInlineKeyboardButtonData("🚀 "+gitText, "export_github:"+sessionID),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔗 "+moreLinksText, "cmd_links_scan"),
			tgbotapi.NewInlineKeyboardButtonData("🔙 "+mainText, "back_start"),
		),
	)
}

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

func MakeAdminPanelKeyboard(loggerActive bool) tgbotapi.InlineKeyboardMarkup {
	logToggleText := "🔴 Turn Logger OFF"
	if !loggerActive {
		logToggleText = "🟢 Turn Logger ON"
	}

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 System Stats", "admin_stats"),
			tgbotapi.NewInlineKeyboardButtonData("👥 User Count", "admin_users"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(logToggleText, "admin_toggle_logger"),
			tgbotapi.NewInlineKeyboardButtonData("📢 Broadcast", "admin_bcast_info"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚫 GBan Controls", "admin_gban_info"),
			tgbotapi.NewInlineKeyboardButtonData("🔙 Main Menu", "back_start"),
		),
	)
}
