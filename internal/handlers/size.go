package handlers

import (
	"context"
	"fmt"
	"net/url"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/redis/go-redis/v9"
	"github.com/yazmeyaa/telegram_sticker_converter_backend/internal/storage"
)

func SizeSelectHandler(rdb *redis.Client) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		cb := update.CallbackQuery
		userID := cb.From.ID
		chatID := cb.Message.Message.Chat.ID
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: cb.ID,
		})

		vals, err := url.ParseQuery(cb.Data)
		if err != nil {
			fmt.Println("failed to parse callback data:", err)
			return
		}
		size := vals.Get("size")
		if size == "" {
			fmt.Println("size not specified")
			return
		}

		state, _ := storage.LoadState(ctx, rdb, userID)
		if state == nil {
			state = &storage.UserState{}
		}
		state.Stage = "size_select"
		state.Size = size

		if err := storage.SaveState(ctx, rdb, userID, *state); err != nil {
			fmt.Println("failed to save state:", err)
			return
		}

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "Processing...",
		})

		b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    chatID,
			MessageID: cb.Message.Message.ID,
		})

		go func() {
			if err := ProcessSticker(ctx, b, rdb, userID, chatID); err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: chatID,
					Text:   fmt.Sprintf("❌ Conversion failed: %s", err),
				})
			}
		}()
	}
}
