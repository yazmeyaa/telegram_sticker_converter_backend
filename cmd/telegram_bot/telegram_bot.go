package main

import (
	"context"
	"io"
	"log"
	"os"
	"os/signal"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/redis/go-redis/v9"
	"github.com/yazmeyaa/telegram_sticker_converter_backend/internal/handlers"
	"github.com/yazmeyaa/telegram_sticker_converter_backend/internal/logger"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Инициализация логгера
	appLogger := logger.New("telegram_bot", []io.Writer{os.Stdout})

	rdb := redis.NewClient(&redis.Options{})
	if err := rdb.Ping(ctx).Err(); err != nil {
		appLogger.Error("Redis connection failed", "error", err)
		log.Fatal("Redis connection failed:", err)
	}

	appLogger.Info("Redis connection established")

	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		appLogger.Error("BOT_TOKEN not set")
		log.Fatal("BOT_TOKEN not set")
	}

	// Определение URL Telegram API
	telegramAPIURL := os.Getenv("TELEGRAM_API_URL")

	var botOptions []telego.BotOption
	if telegramAPIURL != "" {
		appLogger.Info("Using custom Telegram API URL", "url", telegramAPIURL)
		botOptions = append(botOptions, telego.WithAPIServer(telegramAPIURL))
	}

	bot, err := telego.NewBot(token, botOptions...)
	if err != nil {
		appLogger.Error("Failed to create bot", "error", err)
		log.Fatal(err)
	}

	me, err := bot.GetMe(ctx)
	if err != nil {
		appLogger.Error("Failed to get bot info", "error", err)
		log.Fatal(err)
	}

	appLogger.Info("Telegram bot authorized", "username", me.Username)

	updates, err := bot.UpdatesViaLongPolling(ctx, nil)
	if err != nil {
		appLogger.Error("Failed to start long polling", "error", err)
		log.Fatal(err)
	}

	bh, err := th.NewBotHandler(bot, updates)
	if err != nil {
		appLogger.Error("Failed to create bot handler", "error", err)
		log.Fatal(err)
	}
	defer bh.Stop()

	// Команда /start
	bh.Handle(handlers.HandleStartCommand(appLogger), th.CommandEqual("start"))

	// Callback query handlers
	bh.Handle(handlers.FormatSelectHandler(rdb, appLogger), th.CallbackDataContains("action=format_select"))
	bh.Handle(handlers.FrameSelectHandler(rdb, appLogger), th.CallbackDataContains("action=frame_select"))
	bh.Handle(handlers.SizeSelectHandler(rdb, appLogger), th.CallbackDataContains("action=size_select"))

	// Стикеры
	bh.Handle(handlers.MessageHandler(rdb, appLogger), th.AnyMessage(), stickerOrCustomEmojiPredicate())

	appLogger.Info("Telegram bot started", "telegram_api_url", telegramAPIURL)

	go bh.Start()

	<-ctx.Done()
	appLogger.Info("Shutting down bot")
}

// stickerOrCustomEmojiPredicate возвращает предикат для фильтрации стикеров и custom emoji
func stickerOrCustomEmojiPredicate() th.Predicate {
	return func(ctx context.Context, update telego.Update) bool {
		if update.Message == nil {
			return false
		}
		// Стикер
		if update.Message.Sticker != nil {
			return true
		}
		// Custom emoji
		if update.Message.Entities != nil {
			for _, e := range update.Message.Entities {
				if e.Type == telego.EntityTypeCustomEmoji {
					return true
				}
			}
		}
		return false
	}
}

// Для совместимости
var _ = tu.ID
