package handlers

import (
	"context"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func HandleStartCommand(logger *slog.Logger) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		userID := update.Message.From.ID
		chatID := update.Message.Chat.ID
		username := update.Message.From.Username

		logger.Info("User started bot",
			"user_id", userID,
			"chat_id", chatID,
			"username", username,
			"action", "start_command",
		)

		const welcomeMsg = `🦆 Duck Sticker Helper here.  
I help you convert Telegram stickers into different formats — simple as that.  

Supported input: TGS, WEBP, WEBM  
Possible output: PNG, JPEG, WEBP, GIF, MP4, WEBM, and even individual frames.  

Usage is easy:  
1. Send me a sticker  
2. Pick the format  
3. Get your file back  

Duckcoin channel 👉 @duckcoin 
Quack-Quack.`

		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   welcomeMsg,
		})
		if err != nil {
			logger.Error("Failed to send welcome message",
				"user_id", userID,
				"chat_id", chatID,
				"error", err,
				"action", "start_command",
			)
		} else {
			logger.Info("Welcome message sent",
				"user_id", userID,
				"chat_id", chatID,
				"action", "start_command",
			)
		}
	}
}
