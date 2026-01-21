package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

// Logger interface для логирования
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	Warn(msg string, args ...any)
}

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

func sendSizeKeyboard(ctx context.Context, bot *telego.Bot, chatID int64, logger Logger, inputFormat, outputFormat string, frameOption ...string) {
	kb := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("📐 512×512").WithCallbackData("action=size_select&size=512x512"),
			tu.InlineKeyboardButton("📐 1024×1024").WithCallbackData("action=size_select&size=1024x1024"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("🖼️ Original size").WithCallbackData("action=size_select&size=original"),
		),
	)

	frameInfo := ""
	if len(frameOption) > 0 && frameOption[0] != "" {
		frameInfo = fmt.Sprintf("\n🎬 Frame: %s", escapeMarkdownV2(frameOption[0]))
	}

	message := fmt.Sprintf("✅ Format selected: *%s*\n\n📥 Input: %s%s\n\n📏 Choose output size:",
		escapeMarkdownV2(outputFormat),
		escapeMarkdownV2(strings.ToUpper(inputFormat)),
		frameInfo)

	_, err := bot.SendMessage(ctx, tu.Message(tu.ID(chatID), message).
		WithParseMode(telego.ModeMarkdownV2).
		WithReplyMarkup(kb))
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

// Ensure *slog.Logger implements Logger interface
var _ Logger = (*slog.Logger)(nil)
