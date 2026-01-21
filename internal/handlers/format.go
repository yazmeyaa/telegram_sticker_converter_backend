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

func FormatSelectHandler(rdb *redis.Client, logger Logger) th.Handler {
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

		logger.Info("Format selection callback received",
			"user_id", userID,
			"chat_id", chatID,
			"username", username,
			"callback_data", cb.Data,
			"action", "format_select",
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
				"action", "format_select",
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
			return err
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
			return err
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
			return err
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
			kb := tu.InlineKeyboard(
				tu.InlineKeyboardRow(
					tu.InlineKeyboardButton("🎬 First frame").WithCallbackData("action=frame_select&frame=first"),
					tu.InlineKeyboardButton("🎞️ All frames").WithCallbackData("action=frame_select&frame=all"),
				),
			)
			message := fmt.Sprintf("✅ Format selected: *%s*\n\n📥 Input: %s\n📤 Output: %s\n\n🎬 Choose which frames to extract:",
				escapeMarkdownV2(outputFormatName),
				escapeMarkdownV2(strings.ToUpper(inf)),
				escapeMarkdownV2(outputFormatName))

			_, err := bot.SendMessage(ctx, tu.Message(tu.ID(chatID), message).
				WithParseMode(telego.ModeMarkdownV2).
				WithReplyMarkup(kb))
			if err != nil {
				logger.Error("Failed to send frame selection message",
					"user_id", userID,
					"chat_id", chatID,
					"error", err,
					"action", "format_select",
				)
				return err
			}
			logger.Info("Frame selection message sent",
				"user_id", userID,
				"chat_id", chatID,
				"action", "format_select",
			)
		} else if outf == "lottie" {
			message := fmt.Sprintf("✅ Format selected: *%s*\n\n📥 Input: %s\n📤 Output: %s\n\n⏳ Processing your sticker\\.\\.\\.",
				escapeMarkdownV2(outputFormatName),
				escapeMarkdownV2(strings.ToUpper(inf)),
				escapeMarkdownV2(outputFormatName))

			_, err := bot.SendMessage(ctx, tu.Message(tu.ID(chatID), message).
				WithParseMode(telego.ModeMarkdownV2))
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
				bgCtx := context.Background()
				if err := ProcessSticker(bgCtx, bot, rdb, userID, chatID, logger); err != nil {
					logger.Error("Sticker processing failed",
						"user_id", userID,
						"chat_id", chatID,
						"error", err,
						"action", "process_sticker",
					)
					bot.SendMessage(bgCtx, tu.Message(tu.ID(chatID),
						fmt.Sprintf("❌ *Conversion failed*\n\n%s", escapeMarkdownV2(err.Error()))).
						WithParseMode(telego.ModeMarkdownV2))
				}
			}()
		} else {
			logger.Info("Sending size selection keyboard",
				"user_id", userID,
				"chat_id", chatID,
				"action", "format_select",
			)
			sendSizeKeyboard(ctx, bot, chatID, logger, inf, outputFormatName)
		}
		return nil
	}
}
