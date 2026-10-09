package bot

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/config"
	"github.com/SUDEEPBOTS/module-renamer-bot/internal/database"
)

// Bot represents the core Telegram Bot instance.
type Bot struct {
	api *tgbotapi.BotAPI
	cfg *config.Config
	db  database.Database
	sm  *SessionManager
}

// New creates and configures a new Bot.
func New(cfg *config.Config, db database.Database) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		return nil, err
	}

	api.Debug = false
	log.Printf("🤖 Authorized bot account: @%s", api.Self.UserName)

	return &Bot{
		api: api,
		cfg: cfg,
		db:  db,
		sm:  NewSessionManager(),
	}, nil
}

// Start begins the long polling update loop.
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
