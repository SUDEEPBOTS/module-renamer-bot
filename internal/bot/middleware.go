package bot

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/renamer"
)

// CheckFSub verifies whether user is a member of the mandatory update channel.
func (b *Bot) CheckFSub(userID int64) (bool, error) {
	if b.cfg.FSubChannel == "" {
		return true, nil
	}

	channel := b.cfg.FSubChannel
	if !strings.HasPrefix(channel, "@") && b.cfg.FSubChatID == 0 {
		channel = "@" + channel
	}

	chatConfig := tgbotapi.GetChatMemberConfig{
		ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
			SuperGroupUsername: channel,
			UserID:             userID,
		},
	}

	if b.cfg.FSubChatID != 0 {
		chatConfig.ChatID = b.cfg.FSubChatID
		chatConfig.SuperGroupUsername = ""
	}

	member, err := b.api.GetChatMember(chatConfig)
	if err != nil {
		// If bot is not admin in channel, do not block user
		return true, nil
	}

	status := member.Status
	if status == "creator" || status == "administrator" || status == "member" || status == "restricted" {
		return true, nil
	}

	return false, nil
}

// SendFSubPrompt sends the mandatory channel join requirement prompt.
func (b *Bot) SendFSubPrompt(chatID int64) {
	fsubTitle := renamer.ToBoldSerif("Must Join Channel Required")
	cleanChannel := strings.TrimPrefix(b.cfg.FSubChannel, "@")

	text := "⚠️ <b><u>" + fsubTitle + "</u></b>\n\n" +
		"To use the <b>Module Renamer Engine</b>, you must first join our official updates channel!\n\n" +
		"👉 <b>Channel:</b> @" + cleanChannel + "\n\n" +
		"<i>Click the button below to join, then click 'Verify Membership' to start!</i>"

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = MakeFSubKeyboard(cleanChannel)
	_, _ = b.api.Send(msg)
}
