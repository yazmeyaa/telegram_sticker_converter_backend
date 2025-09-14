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

func FrameSelectHandler(rdb *redis.Client) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		cb := update.CallbackQuery
		userID := cb.From.ID
		chatID := cb.Message.Message.Chat.ID

		go func() {
			b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: cb.ID,
			})
			b.DeleteMessage(ctx, &bot.DeleteMessageParams{
				ChatID:    chatID,
				MessageID: cb.Message.Message.ID,
			})
		}()

		vals, err := url.ParseQuery(cb.Data)
		if err != nil {
			fmt.Println("failed to parse callback data:", err)
			return
		}
		frame := vals.Get("frame")
		if frame == "" {
			fmt.Println("frame not specified")
			return
		}

		state, _ := storage.LoadState(ctx, rdb, userID)
		if state == nil {
			state = &storage.UserState{}
		}
		state.Stage = "frame_select"
		state.FrameOption = frame

		if err := storage.SaveState(ctx, rdb, userID, *state); err != nil {
			fmt.Println("failed to save state:", err)
			return
		}

		go sendSizeKeyboard(ctx, b, chatID)
	}
}
