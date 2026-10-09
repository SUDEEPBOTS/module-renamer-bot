package bot

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/renamer"
)

func MakeStartKeyboard(isSudo bool, ownerUsername, repoURL, supportChat, fsubChannel string) tgbotapi.InlineKeyboardMarkup {
	ownerBtnText := renamer.ToSmallCaps("Owner")
	repoBtnText := renamer.ToSmallCaps("Repo")
	supportBtnText := renamer.ToSmallCaps("Support")
	channelBtnText := renamer.ToSmallCaps("Channel")
	renameBtnText := renamer.ToSmallCaps("Rename Codebase")
	authorBtnText := renamer.ToSmallCaps("Scan Author")
	linksBtnText := renamer.ToSmallCaps("Scan Links")
	helpBtnText := renamer.ToSmallCaps("Help & Commands")

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
		adminBtnText := renamer.ToSmallCaps("Admin Panel")
		rowAdmin := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("🛡️ "+adminBtnText, "cmd_admin_panel"),
		}
		rows = append(rows, rowAdmin)
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func MakeConfirmAuthorKeyboard(sessionID string) tgbotapi.InlineKeyboardMarkup {
	replaceText := renamer.ToSmallCaps("Replace Name")
	cancelText := renamer.ToSmallCaps("Cancel")
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✍️ "+replaceText, "author_replace:"+sessionID),
			tgbotapi.NewInlineKeyboardButtonData("❌ "+cancelText, "cancel:"+sessionID),
		),
	)
}

func MakeHelpKeyboard(page int) tgbotapi.InlineKeyboardMarkup {
	backText := renamer.ToSmallCaps("Back")
	nextText := renamer.ToSmallCaps("Next")
	mainText := renamer.ToSmallCaps("Main Menu")

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
		backText := renamer.ToSmallCaps("Back")
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("⬅️ "+backText, fmt.Sprintf("link_page:%d", page-1)))
	}
	navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("📄 %d/%d", page, totalPages), "noop"))
	if page < totalPages {
		nextText := renamer.ToSmallCaps("Next")
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData(nextText+" ➡️", fmt.Sprintf("link_page:%d", page+1)))
	}
	rows = append(rows, navRow)

	cancelRow := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("❌ "+renamer.ToSmallCaps("Cancel"), "cancel:"+sessionID),
		tgbotapi.NewInlineKeyboardButtonData("🔙 "+renamer.ToSmallCaps("Main"), "back_start"),
	)
	rows = append(rows, cancelRow)

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func MakePostLinkReplaceKeyboard(sessionID string) tgbotapi.InlineKeyboardMarkup {
	zipText := renamer.ToSmallCaps("Export ZIP")
	gitText := renamer.ToSmallCaps("Push to GitHub")
	moreLinksText := renamer.ToSmallCaps("Scan More Links")
	mainText := renamer.ToSmallCaps("Main Menu")

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
	zipText := renamer.ToSmallCaps("Download as ZIP")
	gitText := renamer.ToSmallCaps("Push to GitHub")
	cancelText := renamer.ToSmallCaps("Cancel")

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📦 "+zipText, "export_zip:"+sessionID),
			tgbotapi.NewInlineKeyboardButtonData("🚀 "+gitText, "export_github:"+sessionID),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ "+cancelText, "cancel:"+sessionID),
		),
	)
}

func MakeFSubKeyboard(channelUsername string) tgbotapi.InlineKeyboardMarkup {
	link := "https://t.me/" + channelUsername
	joinText := renamer.ToSmallCaps("Join Channel")
	verifyText := renamer.ToSmallCaps("Verify Membership")

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("🔔 "+joinText, link),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 "+verifyText, "verify_fsub"),
		),
	)
}

func MakeAdminPanelKeyboard(loggerActive bool) tgbotapi.InlineKeyboardMarkup {
	logToggleText := "🔴 " + renamer.ToSmallCaps("Turn Logger OFF")
	if !loggerActive {
		logToggleText = "🟢 " + renamer.ToSmallCaps("Turn Logger ON")
	}

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 "+renamer.ToSmallCaps("System Stats"), "admin_stats"),
			tgbotapi.NewInlineKeyboardButtonData("👥 "+renamer.ToSmallCaps("User Count"), "admin_users"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(logToggleText, "admin_toggle_logger"),
			tgbotapi.NewInlineKeyboardButtonData("📢 "+renamer.ToSmallCaps("Broadcast"), "admin_bcast_info"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚫 "+renamer.ToSmallCaps("GBan Controls"), "admin_gban_info"),
			tgbotapi.NewInlineKeyboardButtonData("🔙 "+renamer.ToSmallCaps("Main Menu"), "back_start"),
		),
	)
}
