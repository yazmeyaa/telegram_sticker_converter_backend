package handlers

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func HandleStartCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
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

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   welcomeMsg,
	})
}
