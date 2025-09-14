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

func MessageHandler(rdb *redis.Client) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.Message == nil || update.Message.Sticker == nil {
			return
		}
		sticker := update.Message.Sticker

		var inf string
		var kb [][]models.InlineKeyboardButton

		switch {
		case sticker.IsAnimated:
			inf = "tgs"
			kb = [][]models.InlineKeyboardButton{
				{
					{Text: "GIF", CallbackData: formatToKey(inf, "gif")},
					{Text: "WEBM", CallbackData: formatToKey(inf, "webm")},
					{Text: "MP4", CallbackData: formatToKey(inf, "mp4")},
				},
				{
					{Text: "PNG", CallbackData: formatToKey(inf, "png")},
					{Text: "JPEG", CallbackData: formatToKey(inf, "jpeg")},
					{Text: "WEBP", CallbackData: formatToKey(inf, "webp")},
				},
				{
					{Text: "Lottie.JSON", CallbackData: formatToKey(inf, "lottie")},
				},
			}
		case sticker.IsVideo:
			inf = "webm"
			kb = [][]models.InlineKeyboardButton{
				{
					{Text: "PNG", CallbackData: formatToKey(inf, "png")},
					{Text: "JPEG", CallbackData: formatToKey(inf, "jpeg")},
					{Text: "WEBP", CallbackData: formatToKey(inf, "webp")},
				},
				{
					{Text: "GIF", CallbackData: formatToKey(inf, "gif")},
					{Text: "WEBM", CallbackData: formatToKey(inf, "webm")},
				},
			}
		default:
			inf = "webp"
			kb = [][]models.InlineKeyboardButton{
				{
					{Text: "PNG", CallbackData: formatToKey(inf, "png")},
					{Text: "JPEG", CallbackData: formatToKey(inf, "jpeg")},
				},
			}
		}

		rdb.Del(ctx, fmt.Sprintf("user_state:%d", update.Message.From.ID))

		state := storage.UserState{
			Stage:  "init",
			FileID: sticker.FileID,
		}

		_ = storage.SaveState(ctx, rdb, update.Message.From.ID, state)

		message := fmt.Sprintf("Got a sticker!\n\nInput format: %s\nSelect output format...", inf)

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   message,
			ReplyMarkup: &models.InlineKeyboardMarkup{
				InlineKeyboard: kb,
			},
		})
	}
}

func formatToKey(inf, outf string) string {
	v := url.Values{}
	v.Set("action", "format_select")
	v.Set("in", inf)
	v.Set("out", outf)
	return v.Encode()
}
