package bot

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/config"
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/database"
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/logger"
)

type Bot struct {
	api *tgbotapi.BotAPI
	cfg *config.Config
	db  database.Database
	sm  *SessionManager
}

func New(cfg *config.Config, db database.Database) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		return nil, err
	}

	api.Debug = false
	logger.InitLogger(api, cfg.LogChannel)
	logger.LogInfo("STARTUP", fmt.Sprintf("Authorized bot account: @%s (ID=%d)", api.Self.UserName, api.Self.ID))

	return &Bot{
		api: api,
		cfg: cfg,
		db:  db,
		sm:  NewSessionManager(),
	}, nil
}

func (b *Bot) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			go b.HandleMessage(update.Message)
		} else if update.CallbackQuery != nil {
			go b.HandleCallbackQuery(update.CallbackQuery)
		}
	}
}
