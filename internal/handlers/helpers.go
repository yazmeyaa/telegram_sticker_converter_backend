package handlers

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func sendSizeKeyboard(ctx context.Context, b *bot.Bot, chatID int64) {
	kb := [][]models.InlineKeyboardButton{
		{
			{Text: "512x512", CallbackData: "action=size_select&size=512x512"},
			{Text: "1024x1024", CallbackData: "action=size_select&size=1024x1024"},
		},
		{
			{Text: "Original size", CallbackData: "action=size_select&size=original"},
		},
	}
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   "Select size",
		ReplyMarkup: &models.InlineKeyboardMarkup{
			InlineKeyboard: kb,
		},
	})
}
