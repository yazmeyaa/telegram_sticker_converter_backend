package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/go-telegram/bot"
	"github.com/redis/go-redis/v9"
	"github.com/yazmeyaa/telegram_sticker_converter_backend/internal/handlers"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	rdb := redis.NewClient(&redis.Options{})
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("BOT_TOKEN not set")
	}

	opts := []bot.Option{
		bot.WithDefaultHandler(handlers.MessageHandler(rdb)),
		bot.WithCallbackQueryDataHandler("action=format_select", bot.MatchTypePrefix, handlers.FormatSelectHandler(rdb)),
		bot.WithCallbackQueryDataHandler("action=frame_select", bot.MatchTypePrefix, handlers.FrameSelectHandler(rdb)),
		bot.WithCallbackQueryDataHandler("action=size_select", bot.MatchTypePrefix, handlers.SizeSelectHandler(rdb)),
	}

	b, err := bot.New(token, opts...)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, handlers.HandleStartCommand)

	if err != nil {
		log.Fatal(err)
	}

	b.Start(ctx)
}
