package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/redis/go-redis/v9"
	"github.com/yazmeyaa/telegram_sticker_converter_backend/internal/storage"
)

func FormatSelectHandler(rdb *redis.Client, logger *slog.Logger) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		cb := update.CallbackQuery
		userID := cb.From.ID
		chatID := cb.Message.Message.Chat.ID
		username := cb.From.Username

		logger.Info("Format selection callback received",
			"user_id", userID,
			"chat_id", chatID,
			"username", username,
			"callback_data", cb.Data,
			"action", "format_select",
		)

		// Немедленно отвечаем на callback и удаляем сообщение
		_, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: cb.ID,
		})
		if err != nil {
			logger.Error("Failed to answer callback query",
				"user_id", userID,
				"callback_query_id", cb.ID,
				"error", err,
				"action", "format_select",
			)
		}

		_, err = b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    chatID,
			MessageID: cb.Message.Message.ID,
		})
		if err != nil {
			logger.Error("Failed to delete message",
				"user_id", userID,
				"chat_id", chatID,
				"message_id", cb.Message.Message.ID,
				"error", err,
				"action", "format_select",
			)
		}

		vals, err := url.ParseQuery(cb.Data)
		if err != nil {
			logger.Error("Failed to parse callback data",
				"user_id", userID,
				"chat_id", chatID,
				"callback_data", cb.Data,
				"error", err,
				"action", "format_select",
			)
			return
		}

		inf := vals.Get("in")
		outf := vals.Get("out")

		state, err := storage.LoadState(ctx, rdb, userID)
		if err != nil || state == nil {
			logger.Error("Cannot find user state",
				"user_id", userID,
				"chat_id", chatID,
				"error", err,
				"action", "format_select",
			)
			return
		}
		state.Stage = "format_select"
		state.InputFormat = inf
		state.OutputFormat = outf

		if err := storage.SaveState(ctx, rdb, userID, *state); err != nil {
			logger.Error("Failed to save state",
				"user_id", userID,
				"chat_id", chatID,
				"error", err,
				"action", "format_select",
			)
			return
		}

		logger.Info("Format selected",
			"user_id", userID,
			"chat_id", chatID,
			"input_format", inf,
			"output_format", outf,
			"action", "format_select",
		)

		isAnimated := inf == "tgs" || inf == "webm"
		isImageOut := outf == "png" || outf == "webp" || outf == "jpeg"

		formatDisplayName := map[string]string{
			"png":    "PNG",
			"jpeg":   "JPEG",
			"webp":   "WEBP",
			"gif":    "GIF",
			"webm":   "WEBM",
			"mp4":    "MP4",
			"lottie": "Lottie JSON",
		}

		outputFormatName := formatDisplayName[outf]
		if outputFormatName == "" {
			outputFormatName = strings.ToUpper(outf)
		}

		if isAnimated && isImageOut {
			kb := [][]models.InlineKeyboardButton{
				{
					{Text: "🎬 First frame", CallbackData: "action=frame_select&frame=first"},
					{Text: "🎞️ All frames", CallbackData: "action=frame_select&frame=all"},
				},
			}
			message := fmt.Sprintf("✅ Format selected: *%s*\n\n📥 Input: %s\n📤 Output: %s\n\n🎬 Choose which frames to extract:",
				escapeMarkdownV2(outputFormatName),
				escapeMarkdownV2(strings.ToUpper(inf)),
				escapeMarkdownV2(outputFormatName))
			_, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:      chatID,
				Text:        message,
				ParseMode:   models.ParseModeMarkdown,
				ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: kb},
			})
			if err != nil {
				logger.Error("Failed to send frame selection message",
					"user_id", userID,
					"chat_id", chatID,
					"error", err,
					"action", "format_select",
				)
			} else {
				logger.Info("Frame selection message sent",
					"user_id", userID,
					"chat_id", chatID,
					"action", "format_select",
				)
			}
		} else if outf == "lottie" {
			message := fmt.Sprintf("✅ Format selected: *%s*\n\n📥 Input: %s\n📤 Output: %s\n\n⏳ Processing your sticker\\.\\.\\.",
				escapeMarkdownV2(outputFormatName),
				escapeMarkdownV2(strings.ToUpper(inf)),
				escapeMarkdownV2(outputFormatName))
			_, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:    chatID,
				Text:      message,
				ParseMode: models.ParseModeMarkdown,
			})
			if err != nil {
				logger.Error("Failed to send processing message",
					"user_id", userID,
					"chat_id", chatID,
					"error", err,
					"action", "format_select",
				)
			}

			logger.Info("Starting sticker processing (lottie format)",
				"user_id", userID,
				"chat_id", chatID,
				"input_format", inf,
				"output_format", outf,
				"action", "format_select",
			)
			go func() {
				if err := ProcessSticker(ctx, b, rdb, userID, chatID, logger); err != nil {
					logger.Error("Sticker processing failed",
						"user_id", userID,
						"chat_id", chatID,
						"error", err,
						"action", "process_sticker",
					)
					b.SendMessage(ctx, &bot.SendMessageParams{
						ChatID:    chatID,
						Text:      fmt.Sprintf("❌ *Conversion failed*\n\n%s", escapeMarkdownV2(err.Error())),
						ParseMode: models.ParseModeMarkdown,
					})
				}
			}()
		} else {
			logger.Info("Sending size selection keyboard",
				"user_id", userID,
				"chat_id", chatID,
				"action", "format_select",
			)
			sendSizeKeyboard(ctx, b, chatID, logger, inf, outputFormatName)
		}

	}
}
