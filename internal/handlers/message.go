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

func MessageHandler(rdb *redis.Client, logger *slog.Logger) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.Message == nil || (update.Message.Sticker == nil && len(update.Message.Entities) == 0) {
			return
		}

		userID := update.Message.From.ID
		chatID := update.Message.Chat.ID
		username := update.Message.From.Username

		var sticker *models.Sticker
		if update.Message.Sticker != nil {
			sticker = update.Message.Sticker
			logger.Info("Received sticker",
				"user_id", userID,
				"chat_id", chatID,
				"username", username,
				"sticker_file_id", sticker.FileID,
				"sticker_is_animated", sticker.IsAnimated,
				"sticker_is_video", sticker.IsVideo,
				"action", "sticker_received",
			)
		} else {
			for _, e := range update.Message.Entities {
				if e.Type != models.MessageEntityTypeCustomEmoji {
					continue
				}

				logger.Info("Received custom emoji",
					"user_id", userID,
					"chat_id", chatID,
					"username", username,
					"custom_emoji_id", e.CustomEmojiID,
					"action", "custom_emoji_received",
				)

				s, err := b.GetCustomEmojiStickers(ctx, &bot.GetCustomEmojiStickersParams{
					CustomEmojiIDs: []string{e.CustomEmojiID},
				})
				if err != nil {
					logger.Error("Failed to get custom emoji stickers",
						"user_id", userID,
						"chat_id", chatID,
						"custom_emoji_id", e.CustomEmojiID,
						"error", err,
						"action", "custom_emoji_received",
					)
					return
				}
				if len(s) < 1 {
					logger.Warn("No stickers returned for custom emoji",
						"user_id", userID,
						"chat_id", chatID,
						"custom_emoji_id", e.CustomEmojiID,
						"action", "custom_emoji_received",
					)
					return
				}
				sticker = s[0]
				logger.Info("Custom emoji sticker retrieved",
					"user_id", userID,
					"chat_id", chatID,
					"sticker_file_id", sticker.FileID,
					"sticker_is_animated", sticker.IsAnimated,
					"sticker_is_video", sticker.IsVideo,
					"action", "custom_emoji_received",
				)
			}
		}

		var inf string
		var kb [][]models.InlineKeyboardButton

		switch {
		case sticker.IsAnimated:
			inf = "tgs"
			kb = [][]models.InlineKeyboardButton{
				{
					{Text: "GIF", CallbackData: formatToKey(inf, "gif")},
					{Text: "WEBM", CallbackData: formatToKey(inf, "webm")},
					{Text: "MP4", CallbackData: formatToKey(inf, "mp4")},
				},
				{
					{Text: "PNG", CallbackData: formatToKey(inf, "png")},
					{Text: "JPEG", CallbackData: formatToKey(inf, "jpeg")},
					{Text: "WEBP", CallbackData: formatToKey(inf, "webp")},
				},
				{
					{Text: "Lottie.JSON", CallbackData: formatToKey(inf, "lottie")},
				},
			}
		case sticker.IsVideo:
			inf = "webm"
			kb = [][]models.InlineKeyboardButton{
				{
					{Text: "PNG", CallbackData: formatToKey(inf, "png")},
					{Text: "JPEG", CallbackData: formatToKey(inf, "jpeg")},
					{Text: "WEBP", CallbackData: formatToKey(inf, "webp")},
				},
				{
					{Text: "GIF", CallbackData: formatToKey(inf, "gif")},
				},
			}
		default:
			inf = "webp"
			kb = [][]models.InlineKeyboardButton{
				{
					{Text: "PNG", CallbackData: formatToKey(inf, "png")},
					{Text: "JPEG", CallbackData: formatToKey(inf, "jpeg")},
				},
			}
		}

		rdb.Del(ctx, fmt.Sprintf("user_state:%d", userID))

		state := storage.UserState{
			Stage:  "init",
			FileID: sticker.FileID,
		}

		if err := storage.SaveState(ctx, rdb, userID, state); err != nil {
			logger.Error("Failed to save user state",
				"user_id", userID,
				"chat_id", chatID,
				"error", err,
				"action", "sticker_received",
			)
		}

		logger.Info("User state initialized",
			"user_id", userID,
			"chat_id", chatID,
			"input_format", inf,
			"sticker_file_id", sticker.FileID,
			"action", "sticker_received",
		)

		formatDisplayName := map[string]string{
			"tgs":  "TGS (Animated)",
			"webm": "WEBM (Video)",
			"webp": "WEBP (Static)",
		}
		inputFormatName := formatDisplayName[inf]
		if inputFormatName == "" {
			inputFormatName = strings.ToUpper(inf)
		}

		message := fmt.Sprintf("🎨 *Sticker received\\!*\n\n📥 Input format: *%s*\n\n📤 Select output format:",
			escapeMarkdownV2(inputFormatName))

		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    chatID,
			Text:      message,
			ParseMode: models.ParseModeMarkdown,
			ReplyMarkup: &models.InlineKeyboardMarkup{
				InlineKeyboard: kb,
			},
		})
		if err != nil {
			logger.Error("Failed to send format selection message",
				"user_id", userID,
				"chat_id", chatID,
				"error", err,
				"action", "sticker_received",
			)
		} else {
			logger.Info("Format selection message sent",
				"user_id", userID,
				"chat_id", chatID,
				"input_format", inf,
				"action", "sticker_received",
			)
		}
	}
}

func formatToKey(inf, outf string) string {
	v := url.Values{}
	v.Set("action", "format_select")
	v.Set("in", inf)
	v.Set("out", outf)
	return v.Encode()
}
