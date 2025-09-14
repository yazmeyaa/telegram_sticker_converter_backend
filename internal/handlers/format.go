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

func FormatSelectHandler(rdb *redis.Client) bot.HandlerFunc {
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

		inf := vals.Get("in")
		outf := vals.Get("out")

		state, _ := storage.LoadState(ctx, rdb, userID)
		if state == nil {
			state = &storage.UserState{}
		}
		state.Stage = "format_select"
		state.InputFormat = inf
		state.OutputFormat = outf

		if err := storage.SaveState(ctx, rdb, userID, *state); err != nil {
			fmt.Println("failed to save state:", err)
			return
		}

		isAnimated := inf == "tgs" || inf == "webm"
		isImageOut := outf == "png" || outf == "webp" || outf == "jpeg"

		if isAnimated && isImageOut {
			kb := [][]models.InlineKeyboardButton{
				{
					{Text: "First frame", CallbackData: "action=frame_select&frame=first"},
					// {Text: "Nth frame", CallbackData: "action=frame_select&frame=nth"},
					{Text: "All frames", CallbackData: "action=frame_select&frame=all"},
				},
			}
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:      chatID,
				Text:        "Select frame option",
				ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: kb},
			})
		} else {
			sendSizeKeyboard(ctx, b, chatID)
		}

		b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    chatID,
			MessageID: cb.Message.Message.ID,
		})
	}
}
