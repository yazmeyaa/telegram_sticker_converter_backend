package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// escapeMarkdownV2 экранирует специальные символы для Telegram MarkdownV2
// Специальные символы: _ * [ ] ( ) ~ ` > # + - = | { } . !
func escapeMarkdownV2(text string) string {
	// Список специальных символов, которые нужно экранировать
	specialChars := []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}

	result := text
	for _, char := range specialChars {
		result = strings.ReplaceAll(result, char, "\\"+char)
	}

	return result
}

func sendSizeKeyboard(ctx context.Context, b *bot.Bot, chatID int64, logger *slog.Logger, inputFormat, outputFormat string, frameOption ...string) {
	kb := [][]models.InlineKeyboardButton{
		{
			{Text: "📐 512×512", CallbackData: "action=size_select&size=512x512"},
			{Text: "📐 1024×1024", CallbackData: "action=size_select&size=1024x1024"},
		},
		{
			{Text: "🖼️ Original size", CallbackData: "action=size_select&size=original"},
		},
	}

	frameInfo := ""
	if len(frameOption) > 0 && frameOption[0] != "" {
		frameInfo = fmt.Sprintf("\n🎬 Frame: %s", escapeMarkdownV2(frameOption[0]))
	}

	message := fmt.Sprintf("✅ Format selected: *%s*\n\n📥 Input: %s%s\n\n📏 Choose output size:",
		escapeMarkdownV2(outputFormat),
		escapeMarkdownV2(strings.ToUpper(inputFormat)),
		frameInfo)
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      message,
		ParseMode: models.ParseModeMarkdown,
		ReplyMarkup: &models.InlineKeyboardMarkup{
			InlineKeyboard: kb,
		},
	})
	if err != nil {
		logger.Error("Failed to send size selection keyboard",
			"chat_id", chatID,
			"error", err,
			"action", "send_size_keyboard",
		)
	} else {
		logger.Info("Size selection keyboard sent",
			"chat_id", chatID,
			"action", "send_size_keyboard",
		)
	}
}
