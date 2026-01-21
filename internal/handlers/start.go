package handlers

import (
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
)

func HandleStartCommand(logger Logger) th.Handler {
	return func(c *th.Context, update telego.Update) error {
		msg := update.Message
		if msg == nil {
			return nil
		}

		bot := c.Bot()
		ctx := c.Context()

		userID := msg.From.ID
		chatID := msg.Chat.ID
		username := msg.From.Username

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

		_, err := bot.SendMessage(ctx, tu.Message(tu.ID(chatID), welcomeMsg))
		if err != nil {
			logger.Error("Failed to send welcome message",
				"user_id", userID,
				"chat_id", chatID,
				"error", err,
				"action", "start_command",
			)
			return err
		}
		logger.Info("Welcome message sent",
			"user_id", userID,
			"chat_id", chatID,
			"action", "start_command",
		)
		return nil
	}
}
