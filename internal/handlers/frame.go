package handlers

import (
	"context"
	"log/slog"
	"net/url"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/redis/go-redis/v9"
	"github.com/yazmeyaa/telegram_sticker_converter_backend/internal/storage"
)

func FrameSelectHandler(rdb *redis.Client, logger *slog.Logger) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		cb := update.CallbackQuery
		userID := cb.From.ID
		chatID := cb.Message.Message.Chat.ID
		username := cb.From.Username

		logger.Info("Frame selection callback received",
			"user_id", userID,
			"chat_id", chatID,
			"username", username,
			"callback_data", cb.Data,
			"action", "frame_select",
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
				"action", "frame_select",
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
				"action", "frame_select",
			)
		}

		vals, err := url.ParseQuery(cb.Data)
		if err != nil {
			logger.Error("Failed to parse callback data",
				"user_id", userID,
				"chat_id", chatID,
				"callback_data", cb.Data,
				"error", err,
				"action", "frame_select",
			)
			return
		}
		frame := vals.Get("frame")
		if frame == "" {
			logger.Warn("Frame not specified in callback data",
				"user_id", userID,
				"chat_id", chatID,
				"callback_data", cb.Data,
				"action", "frame_select",
			)
			return
		}

		state, err := storage.LoadState(ctx, rdb, userID)
		if err != nil || state == nil {
			logger.Warn("Cannot find user state, creating new",
				"user_id", userID,
				"chat_id", chatID,
				"error", err,
				"action", "frame_select",
			)
			state = &storage.UserState{}
		}
		state.Stage = "frame_select"
		state.FrameOption = frame

		if err := storage.SaveState(ctx, rdb, userID, *state); err != nil {
			logger.Error("Failed to save state",
				"user_id", userID,
				"chat_id", chatID,
				"error", err,
				"action", "frame_select",
			)
			return
		}

		state, err = storage.LoadState(ctx, rdb, userID)
		if err != nil || state == nil {
			logger.Warn("Cannot find user state, creating new",
				"user_id", userID,
				"chat_id", chatID,
				"error", err,
				"action", "frame_select",
			)
			state = &storage.UserState{}
		}

		formatDisplayName := map[string]string{
			"png":  "PNG",
			"jpeg": "JPEG",
			"webp": "WEBP",
		}
		outputFormatName := formatDisplayName[state.OutputFormat]
		if outputFormatName == "" {
			outputFormatName = strings.ToUpper(state.OutputFormat)
		}

		frameOptionName := "First frame"
		if frame == "all" {
			frameOptionName = "All frames"
		}

		logger.Info("Frame option selected",
			"user_id", userID,
			"chat_id", chatID,
			"frame_option", frame,
			"action", "frame_select",
		)

		sendSizeKeyboard(ctx, b, chatID, logger, state.InputFormat, outputFormatName, frameOptionName)
	}
}
