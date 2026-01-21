package handlers

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/redis/go-redis/v9"
	"github.com/yazmeyaa/telegram_sticker_converter_backend/internal/storage"
)

func SizeSelectHandler(rdb *redis.Client, logger Logger) th.Handler {
	return func(c *th.Context, update telego.Update) error {
		cb := update.CallbackQuery
		if cb == nil {
			return nil
		}

		bot := c.Bot()
		ctx := c.Context()

		userID := cb.From.ID
		chatID := cb.Message.GetChat().ID
		msgID := cb.Message.GetMessageID()
		username := cb.From.Username

		logger.Info("Size selection callback received",
			"user_id", userID,
			"chat_id", chatID,
			"username", username,
			"callback_data", cb.Data,
			"action", "size_select",
		)

		// Немедленно отвечаем на callback и удаляем сообщение
		err := bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
			CallbackQueryID: cb.ID,
		})
		if err != nil {
			logger.Error("Failed to answer callback query",
				"user_id", userID,
				"callback_query_id", cb.ID,
				"error", err,
				"action", "size_select",
			)
		}

		err = bot.DeleteMessage(ctx, &telego.DeleteMessageParams{
			ChatID:    tu.ID(chatID),
			MessageID: msgID,
		})
		if err != nil {
			logger.Error("Failed to delete message",
				"user_id", userID,
				"chat_id", chatID,
				"message_id", msgID,
				"error", err,
				"action", "size_select",
			)
		}

		vals, err := url.ParseQuery(cb.Data)
		if err != nil {
			logger.Error("Failed to parse callback data",
				"user_id", userID,
				"chat_id", chatID,
				"callback_data", cb.Data,
				"error", err,
				"action", "size_select",
			)
			return err
		}
		size := vals.Get("size")
		if size == "" {
			logger.Warn("Size not specified in callback data",
				"user_id", userID,
				"chat_id", chatID,
				"callback_data", cb.Data,
				"action", "size_select",
			)
			return nil
		}

		state, err := storage.LoadState(ctx, rdb, userID)
		if err != nil || state == nil {
			logger.Warn("Cannot find user state, creating new",
				"user_id", userID,
				"chat_id", chatID,
				"error", err,
				"action", "size_select",
			)
			state = &storage.UserState{}
		}
		state.Stage = "size_select"
		state.Size = size

		if err := storage.SaveState(ctx, rdb, userID, *state); err != nil {
			logger.Error("Failed to save state",
				"user_id", userID,
				"chat_id", chatID,
				"error", err,
				"action", "size_select",
			)
			return err
		}

		state, err = storage.LoadState(ctx, rdb, userID)
		if err != nil || state == nil {
			logger.Warn("Cannot find user state",
				"user_id", userID,
				"chat_id", chatID,
				"error", err,
				"action", "size_select",
			)
		}

		formatDisplayName := map[string]string{
			"png":  "PNG",
			"jpeg": "JPEG",
			"webp": "WEBP",
			"gif":  "GIF",
			"webm": "WEBM",
			"mp4":  "MP4",
		}
		outputFormatName := formatDisplayName[state.OutputFormat]
		if outputFormatName == "" {
			outputFormatName = strings.ToUpper(state.OutputFormat)
		}

		sizeDisplayName := size
		if size == "original" {
			sizeDisplayName = "Original size"
		}

		frameInfo := ""
		if state.FrameOption == "all" {
			frameInfo = "\n🎞️ Frames: All frames"
		} else if state.FrameOption == "first" {
			frameInfo = "\n🎬 Frame: First frame"
		}

		// Определяем, будет ли создан большой zip файл
		isLargeZipFile := (state.FrameOption == "all" || state.FrameOption == "range") &&
			(state.OutputFormat == "png" || state.OutputFormat == "jpeg" || state.OutputFormat == "webp")

		logger.Info("Size selected",
			"user_id", userID,
			"chat_id", chatID,
			"size", size,
			"is_large_zip", isLargeZipFile,
			"action", "size_select",
		)

		processingNote := ""
		if isLargeZipFile {
			if state.Size == "original" {
				processingNote = "\n\n⚠️ *Note:* This will create a large ZIP file with all frames in original size\\. Processing may take several minutes\\. Please be patient\\."
			} else {
				processingNote = "\n\n⚠️ *Note:* This will create a ZIP file with all frames\\. Processing may take some time\\. Please be patient\\."
			}
		}

		message := fmt.Sprintf("✅ *All settings confirmed\\!*\n\n📥 Input: %s\n📤 Output: %s\n📏 Size: %s%s\n\n⏳ Processing your sticker\\.\\.\\.%s",
			escapeMarkdownV2(strings.ToUpper(state.InputFormat)),
			escapeMarkdownV2(outputFormatName),
			escapeMarkdownV2(sizeDisplayName),
			frameInfo,
			processingNote)

		_, err = bot.SendMessage(ctx, tu.Message(tu.ID(chatID), message).
			WithParseMode(telego.ModeMarkdownV2))
		if err != nil {
			logger.Error("Failed to send processing message",
				"user_id", userID,
				"chat_id", chatID,
				"error", err,
				"action", "size_select",
			)
		}

		logger.Info("Starting sticker processing",
			"user_id", userID,
			"chat_id", chatID,
			"input_format", state.InputFormat,
			"output_format", state.OutputFormat,
			"frame_option", state.FrameOption,
			"size", size,
			"action", "size_select",
		)

		go func() {
			bgCtx := context.Background()
			if err := ProcessSticker(bgCtx, bot, rdb, userID, chatID, logger); err != nil {
				logger.Error("Sticker processing failed",
					"user_id", userID,
					"chat_id", chatID,
					"error", err,
					"action", "process_sticker",
				)
				bot.SendMessage(bgCtx, tu.Message(tu.ID(chatID),
					fmt.Sprintf("❌ Conversion failed: %s", escapeMarkdownV2(err.Error()))).
					WithParseMode(telego.ModeMarkdownV2))
			}
		}()
		return nil
	}
}
