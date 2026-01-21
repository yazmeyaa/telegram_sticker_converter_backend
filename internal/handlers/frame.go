package handlers

import (
	"net/url"
	"strings"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/redis/go-redis/v9"
	"github.com/yazmeyaa/telegram_sticker_converter_backend/internal/storage"
)

func FrameSelectHandler(rdb *redis.Client, logger Logger) th.Handler {
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

		logger.Info("Frame selection callback received",
			"user_id", userID,
			"chat_id", chatID,
			"username", username,
			"callback_data", cb.Data,
			"action", "frame_select",
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
				"action", "frame_select",
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
			return err
		}
		frame := vals.Get("frame")
		if frame == "" {
			logger.Warn("Frame not specified in callback data",
				"user_id", userID,
				"chat_id", chatID,
				"callback_data", cb.Data,
				"action", "frame_select",
			)
			return nil
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
			return err
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

		sendSizeKeyboard(ctx, bot, chatID, logger, state.InputFormat, outputFormatName, frameOptionName)
		return nil
	}
}
